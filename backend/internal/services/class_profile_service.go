package services

// class_profile_service.go — 班级学情（差异化教学）业务逻辑（老师私有资料，独立模块）
//
// 批次1 范围：班级学情卡 CRUD（手写入口）。
// 批次2a 范围：学生个体档案 CRUD（接出已备好的 repository 学生方法）。
//   成绩单导入、AI 总结学情、教案挂载注入 → 批次2b/2c/3。
//
// 鉴权口径：纯个人。班级是老师自己带的，一律按 created_by == userID 校验，
// 不需要 unit_plans 那套 group/school/system 可见性，比单元方案简单得多。
//
// 学生档案的归属判断走"班级卡归属"——先确认这个班是你的（GetClassProfileByID 比对 CreatedBy），
// 你才能操作它下面的学生。学生实体自身的 owner_id 只是冗余，真正的权限闸门在班级卡。
//
// 合规红线复述：班级学情卡只承载"群体结论"（匿名、无个人身份信息），它才是注入 AI 的内容；
// 学生个体明细在 class_students（本地，永不注入 AI），由后续批次的"AI 总结"在后端就地
// 聚合成匿名统计量后才喂 AI，个体明细（含学号代号）绝不进注入链路。

import (
        "context"
        "errors"
        "strings"

        "github.com/jackc/pgx/v5/pgconn"
        "tedna/internal/logger"
        "tedna/internal/models"
        "tedna/internal/repository"
)

var classProfileLog = logger.WithModule("services.class_profile")

// 业务错误（sentinel，供 handler 用 errors.Is 映射 HTTP 码）
var (
        ErrClassProfileFieldRequired = errors.New("学科、年级、班级名为必填")
        ErrClassProfileNotOwner      = errors.New("只能操作自己创建的班级学情卡")
        // 批次2a 新增：学生档案相关
        ErrStudentCodeRequired = errors.New("学号代号不能为空")
        ErrStudentTierInvalid  = errors.New("分层只能是 A / B / C 或留空")
        ErrStudentCodeDup      = errors.New("该班已存在相同的学号代号")
)

// ClassProfileService 班级学情服务（v1 无需 cfg：批次1/2a 不调 AI；AI 总结在 2c 接）
type ClassProfileService struct{}

// NewClassProfileService 构造
func NewClassProfileService() *ClassProfileService {
        return &ClassProfileService{}
}

// ========================================================================
// 班级学情卡 CRUD（批次1，保持不变）
// ========================================================================

// ListProfiles 列出当前老师自己的全部班级学情卡
func (s *ClassProfileService) ListProfiles(ctx context.Context, userID string) ([]*models.ClassProfileListItem, error) {
        return repository.ListClassProfiles(ctx, userID)
}

// GetProfile 取单张班级学情卡详情（含四大段正文），仅创建者本人可看
func (s *ClassProfileService) GetProfile(ctx context.Context, userID, id string) (*models.ClassProfile, error) {
        p, err := repository.GetClassProfileByID(ctx, id)
        if err != nil {
                return nil, err
        }
        if p.CreatedBy != userID {
                return nil, ErrClassProfileNotOwner
        }
        return p, nil
}

// CreateProfile 新建一张班级学情卡（手写入口；四大段可为空，老师后续慢慢补）
func (s *ClassProfileService) CreateProfile(ctx context.Context, userID string, req *models.CreateClassProfileRequest) (*models.ClassProfile, error) {
        subject := strings.TrimSpace(req.Subject)
        grade := strings.TrimSpace(req.Grade)
        className := strings.TrimSpace(req.ClassName)
        if subject == "" || grade == "" || className == "" {
                return nil, ErrClassProfileFieldRequired
        }

        studentCount := req.StudentCount
        if studentCount < 0 {
                studentCount = 0
        }

        p := &models.ClassProfile{
                Subject:        subject,
                Grade:          grade,
                ClassName:      className,
                Term:           strings.TrimSpace(req.Term),
                StudentCount:   studentCount,
                OverallProfile: strings.TrimSpace(req.OverallProfile),
                TierStructure:  strings.TrimSpace(req.TierStructure),
                WeakPoints:     strings.TrimSpace(req.WeakPoints),
                TeachingAdvice: strings.TrimSpace(req.TeachingAdvice),
                // 手写入口建卡：来源标记 manual（无论这次有没有填四大段）
                LastAnalyzedFrom: models.ClassAnalyzedFromManual,
                CreatedBy:        userID,
        }
        if err := repository.CreateClassProfile(ctx, p); err != nil {
                return nil, err
        }
        classProfileLog.Info("新建班级学情卡", "id", p.ID, "class", p.ClassName, "owner", userID)
        return p, nil
}

// UpdateProfile 更新班级学情卡（编辑定位字段 + 四大段群体学情内容）
//
// 手写编辑场景：来源标记 manual，且不更新 last_analyzed_at（手写不算"一次分析"）。
// 仅创建者本人可编辑。
func (s *ClassProfileService) UpdateProfile(ctx context.Context, userID, id string, req *models.UpdateClassProfileRequest) error {
        p, err := repository.GetClassProfileByID(ctx, id)
        if err != nil {
                return err
        }
        if p.CreatedBy != userID {
                return ErrClassProfileNotOwner
        }

        req.Subject = strings.TrimSpace(req.Subject)
        req.Grade = strings.TrimSpace(req.Grade)
        req.ClassName = strings.TrimSpace(req.ClassName)
        if req.Subject == "" || req.Grade == "" || req.ClassName == "" {
                return ErrClassProfileFieldRequired
        }
        req.Term = strings.TrimSpace(req.Term)
        req.OverallProfile = strings.TrimSpace(req.OverallProfile)
        req.TierStructure = strings.TrimSpace(req.TierStructure)
        req.WeakPoints = strings.TrimSpace(req.WeakPoints)
        req.TeachingAdvice = strings.TrimSpace(req.TeachingAdvice)
        if req.StudentCount < 0 {
                req.StudentCount = 0
        }

        // setAnalyzedNow=false：手写编辑不刷新"最近分析时间"
        return repository.UpdateClassProfile(ctx, id, req, models.ClassAnalyzedFromManual, false)
}

// DeleteProfile 软删除班级学情卡，仅创建者本人可删
func (s *ClassProfileService) DeleteProfile(ctx context.Context, userID, id string) error {
        p, err := repository.GetClassProfileByID(ctx, id)
        if err != nil {
                return err
        }
        if p.CreatedBy != userID {
                return ErrClassProfileNotOwner
        }
        return repository.DeleteClassProfile(ctx, id)
}

// ========================================================================
// 学生个体档案 CRUD（批次2a 新增）
//
// 全部经 ensureProfileOwned 做"班级卡归属"校验：先确认这个班是当前老师的，
// 才允许操作它下面的学生。单个学生的越权由这道闸门统一兜住。
// 合规红线：本层只存学号代号、分层、薄弱点、备注；成绩走批次2b 导入由后端归并。
// ========================================================================

// ensureProfileOwned 校验班级卡存在且归属当前老师，返回班级卡（供后续复用）
func (s *ClassProfileService) ensureProfileOwned(ctx context.Context, userID, classProfileID string) (*models.ClassProfile, error) {
        p, err := repository.GetClassProfileByID(ctx, classProfileID)
        if err != nil {
                return nil, err
        }
        if p.CreatedBy != userID {
                return nil, ErrClassProfileNotOwner
        }
        return p, nil
}

// syncStudentCount 把班级卡的 student_count 刷成实际学生数（best-effort，失败仅记日志不阻断）
//
// 这是增删学生后的贴心副作用：让卡片上展示的人数自动等于实际录入的学生数。
// 注意：用 repo 的 UpdateClassProfile 会重写四大段，故这里直接用专门的轻量更新——
// 但 repo 暂无"只更 student_count"的方法，为不扩散改动，本批次先用整卡回写的方式：
// 读出当前卡 → 用其现值构造 Update 请求（四大段原样）→ 仅 student_count 改为实际数。
// setAnalyzedNow=false 不刷新分析时间，来源沿用卡片原有 last_analyzed_from。
func (s *ClassProfileService) syncStudentCount(ctx context.Context, p *models.ClassProfile) {
        cnt, err := repository.CountClassStudents(ctx, p.ID)
        if err != nil {
                classProfileLog.Warn("统计学生数失败，跳过同步人数", "profile", p.ID, "err", err.Error())
                return
        }
        if cnt == p.StudentCount {
                return // 人数未变，无需写库
        }
        req := &models.UpdateClassProfileRequest{
                Subject:        p.Subject,
                Grade:          p.Grade,
                ClassName:      p.ClassName,
                Term:           p.Term,
                StudentCount:   cnt,
                OverallProfile: p.OverallProfile,
                TierStructure:  p.TierStructure,
                WeakPoints:     p.WeakPoints,
                TeachingAdvice: p.TeachingAdvice,
        }
        // 来源沿用卡片原有标记（默认 manual），不刷新分析时间
        from := p.LastAnalyzedFrom
        if from == "" {
                from = models.ClassAnalyzedFromManual
        }
        if err := repository.UpdateClassProfile(ctx, p.ID, req, from, false); err != nil {
                classProfileLog.Warn("同步学生数到班级卡失败", "profile", p.ID, "err", err.Error())
        }
}

// ListStudents 列出某班级的全部学生档案（已转为前端展示结构，scores 解析为数组）
func (s *ClassProfileService) ListStudents(ctx context.Context, userID, classProfileID string) ([]models.ClassStudentView, error) {
        if _, err := s.ensureProfileOwned(ctx, userID, classProfileID); err != nil {
                return nil, err
        }
        students, err := repository.ListClassStudents(ctx, classProfileID)
        if err != nil {
                return nil, err
        }
        views := make([]models.ClassStudentView, 0, len(students))
        for _, st := range students {
                views = append(views, st.ToClassStudentView())
        }
        return views, nil
}

// CreateStudent 新建一条学生档案（学号留空则自动编号），返回展示结构
func (s *ClassProfileService) CreateStudent(ctx context.Context, userID, classProfileID string, req *models.CreateClassStudentRequest) (*models.ClassStudentView, error) {
        profile, err := s.ensureProfileOwned(ctx, userID, classProfileID)
        if err != nil {
                return nil, err
        }

        tier := strings.TrimSpace(req.Tier)
        if !models.IsValidStudentTier(tier) {
                return nil, ErrStudentTierInvalid
        }

        code := strings.TrimSpace(req.StudentCode)
        if code == "" {
                // 学号留空：取该班现有学号列表，自动编号（纯数字序列 +1，补零两位）
                existing, err := repository.ListClassStudents(ctx, classProfileID)
                if err != nil {
                        return nil, err
                }
                codes := make([]string, 0, len(existing))
                for _, e := range existing {
                        codes = append(codes, e.StudentCode)
                }
                code = models.NextStudentCode(codes)
        }

        st := &models.ClassStudent{
                ClassProfileID: classProfileID,
                OwnerID:        userID,
                StudentCode:    code,
                Tier:           tier,
                Scores:         "[]", // 新建学生无成绩，成绩走批次2b 导入
                LatestScore:    nil,
                WeakTopics:     strings.TrimSpace(req.WeakTopics),
                Note:           strings.TrimSpace(req.Note),
        }
        if err := repository.CreateClassStudent(ctx, st); err != nil {
                if isUniqueViolation(err) {
                        return nil, ErrStudentCodeDup
                }
                return nil, err
        }

        // 同步班级卡人数（best-effort）
        s.syncStudentCount(ctx, profile)

        classProfileLog.Info("新建学生档案", "profile", classProfileID, "code", code, "owner", userID)
        view := st.ToClassStudentView()
        return &view, nil
}

// UpdateStudent 更新一条学生档案（学号/分层/薄弱点/备注；成绩字段不经此路径）
func (s *ClassProfileService) UpdateStudent(ctx context.Context, userID, classProfileID, studentID string, req *models.UpdateClassStudentRequest) (*models.ClassStudentView, error) {
        if _, err := s.ensureProfileOwned(ctx, userID, classProfileID); err != nil {
                return nil, err
        }

        // 取出现有学生，校验它确实属于这个班（防跨班越权改）
        st, err := repository.GetClassStudentByID(ctx, studentID)
        if err != nil {
                return nil, err
        }
        if st.ClassProfileID != classProfileID {
                return nil, repository.ErrClassStudentNotFound
        }

        code := strings.TrimSpace(req.StudentCode)
        if code == "" {
                return nil, ErrStudentCodeRequired
        }
        tier := strings.TrimSpace(req.Tier)
        if !models.IsValidStudentTier(tier) {
                return nil, ErrStudentTierInvalid
        }

        // 只改"定性字段"，成绩相关（Scores/LatestScore）原样保留不动
        st.StudentCode = code
        st.Tier = tier
        st.WeakTopics = strings.TrimSpace(req.WeakTopics)
        st.Note = strings.TrimSpace(req.Note)

        if err := repository.UpdateClassStudent(ctx, st); err != nil {
                if isUniqueViolation(err) {
                        return nil, ErrStudentCodeDup
                }
                return nil, err
        }

        classProfileLog.Info("更新学生档案", "profile", classProfileID, "student", studentID, "owner", userID)
        view := st.ToClassStudentView()
        return &view, nil
}

// DeleteStudent 删除一条学生档案（硬删），并同步班级卡人数
func (s *ClassProfileService) DeleteStudent(ctx context.Context, userID, classProfileID, studentID string) error {
        profile, err := s.ensureProfileOwned(ctx, userID, classProfileID)
        if err != nil {
                return err
        }

        // 校验学生确属本班（防跨班越权删）
        st, err := repository.GetClassStudentByID(ctx, studentID)
        if err != nil {
                return err
        }
        if st.ClassProfileID != classProfileID {
                return repository.ErrClassStudentNotFound
        }

        if err := repository.DeleteClassStudent(ctx, studentID); err != nil {
                return err
        }

        // 同步班级卡人数（best-effort）
        s.syncStudentCount(ctx, profile)

        classProfileLog.Info("删除学生档案", "profile", classProfileID, "student", studentID, "owner", userID)
        return nil
}

// ---------- 内部辅助 ----------

// isUniqueViolation 判定是否为 PostgreSQL 唯一约束冲突（23505）
//
// 学生表有 UNIQUE(class_profile_id,student_code)，同班重复学号会触发此错误，
// service 层把它翻译成友好的 ErrStudentCodeDup。
func isUniqueViolation(err error) bool {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) {
                return pgErr.Code == "23505"
        }
        return false
}
