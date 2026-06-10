package services

// 教案管理业务逻辑层
// 负责教案CRUD、状态流转、AI评审、Fork、评审管理、提示词模板管理
//
// v104改动：SubmitForReview 提交评审前自动将所有 pending 批注归档，
//           防止新旧轮次批注混显
// v121改动（AI辅助修改 Bug 修复 · 方案C）：
//   ReviewLessonPlan 的 revision 分支：教研员退回教案时自动恢复最新一轮的
//   archived 批注为 pending。业务含义是"退回即继续工作"——作者需要用 AI
//   辅助修改继续处理上一轮未彻底解决的批注，不能让它们因为已归档而消失按钮。
//
// v168改动（第二批治本·功能A·正文产出硬门控 切片A）：
//   新增 ErrLPContentEmpty 错误，并在三个状态流转入口加教案正文非空硬校验：
//     - SubmitForReview  提交评审：正文为空时拒绝（堵住"无正文教案进入评审流程"）
//     - PublishPersonal  个人发布：正文为空时拒绝
//     - PublishShared    共享发布：正文为空时拒绝
//   这是把"无正文不流转"从前端补丁升级为后端强制校验的一部分。
//   门控判据极简：strings.TrimSpace(lp.ContentMarkdown) == ""，
//   与阶段完成度系统（checkWriteStage 的篇幅/关键词软提醒）完全解耦，
//   避免误伤合法的短教案。审核端（L1/L2 approved）的正文校验在 review_v2_service.go。
//
// 迭代一 Phase 4 改动（数据隔离收口）：
//   ListLessonPlans 新增 scope *DataScope 入参，从 DataScope 中拆出 UserIDs/IsAdmin
//   传给 repository.ListLessonPlans，实现"教案列表只返回请求者可见范围内的教案"。
//   可见性规则（A 本人 ∪ C 管辖成员 ∪ B 共享可见）的 SQL 拼接在 repo 层；
//   本层只负责把 DataScope 转成 repo 能接受的原始参数（避免 repo 反向依赖 services）。
//   scope 为 nil 时按 fail-closed 退化为"非 admin + 空白名单"（只看共享教案），
//   绝不退化为 admin 全量——任何不确定都收窄，不放大。
//
// 迭代一 Phase 4 收尾改动（共享发布越权封堵）：
//   新增 ErrLPNotPublisher 错误。PublishShared 原先只校验"状态必须 approved"，
//   未校验调用者身份——任何登录用户拿到一个 approved 教案 ID 都能调它翻状态，
//   是一个越权口子。本次补上"仅作者本人 OR admin 可共享发布"的校验：
//     - 作者本人：callerID == lp.AuthorID（教师对自己已通过的教案做收尾确认）。
//     - admin：通过 FindUserByID 反查角色判定（超级管理员兜底，与 ReviewL1 的
//       admin 判定惯例一致，无需给 PublishShared 加 role 参数、不动 handler 签名）。
//     - 其他所有人（含审核员 senior_operator/教研组长/其他教师）：拒绝。
//   语义依据：审核员的"公开权力"体现在审核通过（approved 即对全平台可见），
//   不需要也不应该再经手"共享发布"这一步，避免权限链条出现重复与责任模糊。

import (
        "context"
        "encoding/json"
        "errors"
        "fmt"
        "strings"

        "tedna/internal/logger"
        "tedna/internal/models"
        "tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
        ErrLPTitleRequired      = errors.New("教案标题不能为空")
        ErrLPSubjectRequired    = errors.New("学科不能为空")
        ErrLPGradeRequired      = errors.New("年级不能为空")
        ErrLPTopicRequired      = errors.New("课题不能为空")
        ErrLPNotFound           = errors.New("教案不存在")
        ErrLPNotAuthor          = errors.New("只有作者可以操作此教案")
        ErrLPCannotEdit         = errors.New("当前状态不允许编辑")
        ErrLPCannotSubmit       = errors.New("当前状态不允许提交评审")
        ErrLPCannotDevelop      = errors.New("当前状态不允许进入课件开发")
        ErrLPAlreadyDeveloping  = errors.New("教案已在课件开发中")
        ErrLPGroupRequired      = errors.New("提交评审需要指定教研组")
        // v168新增：教案正文为空时禁止提交评审/发布（功能A硬门控）
        ErrLPContentEmpty       = errors.New("教案正文为空，请先在备课工坊生成完整教案正文后再操作")
        // Phase 4 收尾新增：共享发布越权封堵——仅作者本人或 admin 可共享发布
        ErrLPNotPublisher       = errors.New("只有作者本人或系统管理员可以共享发布此教案")
        ErrTemplateNotFound     = errors.New("提示词模板不存在")
        ErrTemplateLevelInvalid = errors.New("无效的模板层级")
        ErrTemplateNameRequired = errors.New("模板名称不能为空")
)

// StartDevelopmentResult Phase6：进入课件开发的返回结果
type StartDevelopmentResult struct {
        PipelineID string `json:"pipeline_id"` // 新创建的Pipeline ID
        Message    string `json:"message"`
}

// LessonPlanService 教案管理服务
type LessonPlanService struct {
        compService *ComponentService
}

var lpLog = logger.WithModule("lesson_plan")

// strPtr 字符串指针辅助函数（空字符串返回nil）
func strPtr(s string) *string {
        if s == "" {
                return nil
        }
        return &s
}

// NewLessonPlanService 创建教案服务实例
func NewLessonPlanService(compService *ComponentService) *LessonPlanService {
        return &LessonPlanService{compService: compService}
}

// ==================== 教案 CRUD ====================

// CreateLessonPlan 创建教案
func (s *LessonPlanService) CreateLessonPlan(ctx context.Context, req *models.CreateLessonPlanRequest, authorID string) (*models.LessonPlan, error) {
        req.Title = strings.TrimSpace(req.Title)
        req.Subject = strings.TrimSpace(req.Subject)
        req.Grade = strings.TrimSpace(req.Grade)
        req.Topic = strings.TrimSpace(req.Topic)

        if req.Title == "" {
                return nil, ErrLPTitleRequired
        }
        if req.Subject == "" {
                return nil, ErrLPSubjectRequired
        }
        if req.Grade == "" {
                return nil, ErrLPGradeRequired
        }
        if req.Topic == "" {
                return nil, ErrLPTopicRequired
        }

        dur := req.DurationMinutes
        if dur <= 0 {
                dur = 45
        }

        lp := &models.LessonPlan{
                Title:           req.Title,
                Subject:         req.Subject,
                Grade:           req.Grade,
                Topic:           req.Topic,
                DurationMinutes: dur,
                Status:          models.LPStatusDraft,
                Visibility:      models.LPVisibilityPersonal,
                AuthorID:        authorID,
        }

        if err := repository.CreateLessonPlan(ctx, lp); err != nil {
                lpLog.Error("创建教案失败", "title", req.Title, "author", authorID, "error", err)
                return nil, err
        }

        lpLog.Info("创建教案成功", "plan_id", lp.ID, "title", lp.Title, "author", authorID)
        return lp, nil
}

// GetLessonPlan 获取教案详情
func (s *LessonPlanService) GetLessonPlan(ctx context.Context, id string) (*models.LessonPlanDetailResponse, error) {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanNotFound) {
                        return nil, ErrLPNotFound
                }
                return nil, err
        }

        _ = repository.IncrementLessonPlanView(ctx, id)

        authorName := ""
        if author, err := repository.FindUserByID(ctx, lp.AuthorID); err == nil {
                authorName = author.DisplayName
        }

        groupName := ""
        if lp.GroupID != nil {
                if group, err := repository.GetTeachingGroupByID(ctx, *lp.GroupID); err == nil {
                        groupName = group.Name
                }
        }

        reviews, err := repository.ListLessonPlanReviews(ctx, id)
        if err != nil {
                reviews = []*models.LessonPlanReviewItem{}
        }

        // Phase 7A：查询关联配方名称
        recipeName := ""
        if lp.RecipeID != nil && *lp.RecipeID != "" {
                if recipe, err := repository.GetRecipeByID(ctx, *lp.RecipeID); err == nil {
                        recipeName = recipe.Name
                }
        }

        // Phase6：查询关联Pipeline（如有）
        var linkedPipelineID *string
        linkedPipeline, err := repository.GetPipelineByLessonPlanID(id)
        if err == nil && linkedPipeline != nil {
                linkedPipelineID = &linkedPipeline.ID
        }

        // v125新增：查询教案互动数据（点赞/收藏计数+当前用户状态）
        var interactionCounts *models.InteractionCounts
        interactionCounts, _ = repository.GetInteractionCounts(ctx, id, lp.AuthorID)
        // 注意：这里传的是 lp.AuthorID 作为 currentUserID 的占位
        // 实际应该传调用者ID，但 GetLessonPlan 签名只有 id 没有 callerID
        // 互动数据在详情页由前端单独调用 /interactions 接口获取用户态
        // 这里只填充计数（is_liked/is_favorited 在详情接口中不保证准确）
        likeCount := 0
        favoriteCount := 0
        if interactionCounts != nil {
                likeCount = interactionCounts.LikeCount
                favoriteCount = interactionCounts.FavoriteCount
        }

        return &models.LessonPlanDetailResponse{
                ID:                lp.ID,
                Title:             lp.Title,
                Subject:           lp.Subject,
                Grade:             lp.Grade,
                Topic:             lp.Topic,
                DurationMinutes:   lp.DurationMinutes,
                ContentMarkdown:   lp.ContentMarkdown,
                ContentStructured: lp.ContentStructured,
                GenerationConfig:  lp.GenerationConfig,
                MatchedComponents: lp.MatchedComponents,
                AIReviewScore:     lp.AIReviewScore,
                AIReviewResult:    lp.AIReviewResult,
                AIReviewHistory:   lp.AIReviewHistory,
                Status:            lp.Status,
                StatusName:        models.LPStatusNameMap[lp.Status],
                Visibility:        lp.Visibility,
                AuthorID:          lp.AuthorID,
                AuthorName:        authorName,
                GroupID:           lp.GroupID,
                GroupName:         groupName,
                SchoolID:          lp.SchoolID,
                ForkedFrom:        lp.ForkedFrom,
                ForkCount:         lp.ForkCount,
                ViewCount:         lp.ViewCount,
                UseCount:          lp.UseCount,
                Version:           lp.Version,
                RecipeID:          lp.RecipeID,
                RecipeName:        recipeName,
                LessonIndex:       lp.LessonIndex,
                IdxCognitiveLevel: lp.IdxCognitiveLevel,
                IdxPedagogyIntensity: lp.IdxPedagogyIntensity,
                IdxStructureType:  lp.IdxStructureType,
                IdxQualityLevel:   lp.IdxQualityLevel,
                CurrentStage:     lp.CurrentStage,
                StageConfig:      lp.StageConfig,
                LikeCount:     likeCount,
                FavoriteCount: favoriteCount,
                Reviews:           reviews,
                LinkedPipelineID:  linkedPipelineID,
                CreatedAt:         lp.CreatedAt,
                UpdatedAt:         lp.UpdatedAt,
        }, nil
}

// ListLessonPlans 获取教案列表
//
// 迭代一 Phase 4（数据隔离收口）：新增 scope *DataScope 入参。
//   从 scope 中拆出 UserIDs（本人 ∪ 管辖成员白名单）与 IsAdmin（是否看全部），
//   传给 repository.ListLessonPlans。可见性 OR 子句（本人/管辖 ∪ 共享可见）在 repo 层拼接。
//   scope 为 nil 的防御性处理：退化为"非 admin + 空白名单"——只能看共享教案，
//   绝不退化为 admin 全量（fail-closed）。正常调用路径由 handler 保证 scope 非 nil。
func (s *LessonPlanService) ListLessonPlans(ctx context.Context, authorID string, groupID string, status string, subject string, grade string, limit int, offset int, qualityLevel int, structureType int, cognitiveLevel int, pedagogyIntensity int, scope *DataScope) (*models.LessonPlanListResponse, error) {
        // 从 DataScope 拆出 repo 层需要的原始参数（repo 不能反向依赖 services 包）
        var scopeUserIDs []string
        scopeIsAdmin := false
        if scope != nil {
                scopeIsAdmin = scope.IsAdmin
                scopeUserIDs = scope.UserIDs
        } else {
                // 防御性 fail-closed：scope 缺失时按非 admin + 空白名单处理，
                // 仅靠 repo 层 OR 子句的"共享可见"分支放行，绝不放大为全量。
                scopeUserIDs = []string{}
        }

        items, total, err := repository.ListLessonPlans(ctx, authorID, groupID, status, subject, grade, limit, offset, qualityLevel, structureType, cognitiveLevel, pedagogyIntensity, scopeUserIDs, scopeIsAdmin)
        if err != nil {
                return nil, err
        }
        if items == nil {
                items = []*models.LessonPlanListItem{}
        }
        return &models.LessonPlanListResponse{LessonPlans: items, Total: total}, nil
}

// UpdateLessonPlan 更新教案内容
func (s *LessonPlanService) UpdateLessonPlan(ctx context.Context, id string, callerID string, req *models.UpdateLessonPlanRequest) error {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanNotFound) {
                        return ErrLPNotFound
                }
                return err
        }

        if lp.AuthorID != callerID {
                return ErrLPNotAuthor
        }
        // v125改动：允许 approved/published_shared 状态编辑（老师持续改进）
        // 仅 submitted（评审中锁定）和 developing/completed（课件关联锁定）不可编辑
        editableStatuses := map[string]bool{
                models.LPStatusDraft:             true,
                models.LPStatusPublishedPersonal: true,
                models.LPStatusRevision:          true,
                models.LPStatusApproved:          true,
                models.LPStatusPublishedShared:   true,
        }
        if !editableStatuses[lp.Status] {
                return ErrLPCannotEdit
        }

        title := strings.TrimSpace(req.Title)
        if title == "" {
                title = lp.Title
        }
        dur := req.DurationMinutes
        if dur <= 0 {
                dur = lp.DurationMinutes
        }

        if err := repository.UpdateLessonPlanContent(ctx, id, title, req.ContentMarkdown, "", dur); err != nil {
                lpLog.Error("更新教案失败", "plan_id", id, "error", err)
                return err
        }
        return nil
}

// DeleteLessonPlan 删除教案
func (s *LessonPlanService) DeleteLessonPlan(ctx context.Context, id string, callerID string) error {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanNotFound) {
                        return ErrLPNotFound
                }
                return err
        }

        if lp.AuthorID != callerID {
                return ErrLPNotAuthor
        }

        if err := repository.DeleteLessonPlan(ctx, id); err != nil {
                lpLog.Error("删除教案失败", "plan_id", id, "error", err)
                return err
        }
        lpLog.Info("删除教案成功", "plan_id", id)
        return nil
}

// ==================== 教案状态流转 ====================

// PublishPersonal 个人发布
//
// v168新增：发布前校验教案正文非空（功能A硬门控）。
// 无正文教案不应进入"已发布"状态——发布意味着教案可被生成课件、被他人查看，
// 必须保证有实质正文内容。
func (s *LessonPlanService) PublishPersonal(ctx context.Context, id string, callerID string) error {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                return s.mapNotFoundErr(err)
        }
        if lp.AuthorID != callerID {
                return ErrLPNotAuthor
        }
        if lp.Status != models.LPStatusDraft && lp.Status != models.LPStatusRevision {
                return errors.New("只有草稿或退回状态的教案可以个人发布")
        }
        // v168：正文非空硬校验
        if strings.TrimSpace(lp.ContentMarkdown) == "" {
                lpLog.Info("个人发布被拦截：教案正文为空", "plan_id", id)
                return ErrLPContentEmpty
        }
        return repository.UpdateLessonPlanStatus(ctx, id, models.LPStatusPublishedPersonal)
}

// SubmitForReview 提交评审
//
// v104改动：提交前自动将所有 pending 批注归档（archived），
// 防止老师重新提交后，评审员新建的批注和上一轮未处理的批注混在一起显示。
// 只归档 pending 状态；已 resolved 的批注保持不变，作为历史记录保留。
//
// v168改动：状态校验通过后、归档批注之前，增加教案正文非空硬校验（功能A硬门控）。
// 这是堵住"无正文教案进入评审流程"根洞的关键——历史上《春》《英语》等坏数据
// 正是因为提交评审与审核两端都没有正文校验，才得以走到 approved。
func (s *LessonPlanService) SubmitForReview(ctx context.Context, id string, callerID string, groupID string) error {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                return s.mapNotFoundErr(err)
        }
        if lp.AuthorID != callerID {
                return ErrLPNotAuthor
        }
        if groupID == "" {
                return ErrLPGroupRequired
        }
        if lp.Status != models.LPStatusDraft &&
                lp.Status != models.LPStatusPublishedPersonal &&
                lp.Status != models.LPStatusRevision {
                return ErrLPCannotSubmit
        }
        // v168：正文非空硬校验（在归档批注/改可见性等副作用之前拦截，避免半途状态）
        if strings.TrimSpace(lp.ContentMarkdown) == "" {
                lpLog.Info("提交评审被拦截：教案正文为空", "plan_id", id)
                return ErrLPContentEmpty
        }

        // v104：自动归档本轮所有 pending 批注，避免新旧轮次混显
        // 归档失败不阻断提交流程，仅记录日志
        if archiveErr := repository.ArchiveAnnotationsByPlanID(ctx, id); archiveErr != nil {
                lpLog.Error("归档批注失败（不影响提交）", "plan_id", id, "error", archiveErr)
        } else {
                lpLog.Info("已归档本轮待处理批注", "plan_id", id)
        }

        if err := repository.UpdateLessonPlanVisibility(ctx, id, models.LPVisibilityGroup, &groupID); err != nil {
                return err
        }
        // v127新增：自动设置 review_school_id（从教研组反查学校）
        schoolID := ""
        if groupID != "" {
                if group, gErr := repository.GetTeachingGroupByID(ctx, groupID); gErr == nil {
                        schoolID = group.SchoolID
                }
        }
        if schoolID != "" {
                _ = repository.UpdateLessonPlanReviewLevel(ctx, id, 0, &schoolID)
        }

        return repository.UpdateLessonPlanStatus(ctx, id, models.LPStatusSubmitted)
}

// ReviewLessonPlan 评审教案
// Phase5：approved且ai_review_score>=8.5时异步触发通道二自动萃取
//
// v121改动（AI辅助修改 Bug 修复 · 方案C）：
//   revision 分支新增：教研员退回教案时,自动把最新一轮已归档(archived)的批注
//   恢复为 pending。业务含义是"退回即继续工作"——作者需要用 AI 辅助修改继续
//   处理上一轮未彻底解决的批注,不能让它们因为已归档而隐藏所有操作按钮。
//   恢复失败不阻断退回流程,仅记录日志。
func (s *LessonPlanService) ReviewLessonPlan(ctx context.Context, planID string, reviewerID string, req *models.CreateLessonPlanReviewRequest) error {
        lp, err := repository.GetLessonPlanByID(ctx, planID)
        if err != nil {
                return s.mapNotFoundErr(err)
        }

        if lp.Status != models.LPStatusSubmitted {
                return errors.New("只有已提交评审的教案可以评审")
        }

        if req.Decision != "approved" && req.Decision != "revision" && req.Decision != "rejected" {
                return errors.New("评审决策无效，可选值：approved/revision/rejected")
        }

        existingReviews, _ := repository.ListLessonPlanReviews(ctx, planID)
        round := len(existingReviews) + 1

        review := &models.LessonPlanReview{
                LessonPlanID: planID,
                ReviewerID:   reviewerID,
                Decision:     req.Decision,
                Score:        req.Score,
                Dimensions:   req.Dimensions,
                Comments:     req.Comments,
                Suggestions:  req.Suggestions,
                Round:        round,
        }
        if err := repository.CreateLessonPlanReview(ctx, review); err != nil {
                lpLog.Error("创建评审记录失败", "plan_id", planID, "error", err)
                return err
        }

        switch req.Decision {
        case "approved":
                _ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusApproved)
        case "revision":
                _ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusRevision)
                // v121新增：退回时恢复最新一轮 archived 批注为 pending
                // 让作者能看到"🤖 AI辅助修改"按钮,继续处理上一轮未彻底解决的批注
                // 恢复失败不影响退回主流程,只记录日志
                if restoreErr := repository.RestoreArchivedAnnotationsForLatestRound(ctx, planID); restoreErr != nil {
                        lpLog.Error("恢复归档批注失败（不影响退回）", "plan_id", planID, "error", restoreErr)
                } else {
                        lpLog.Info("已恢复最新一轮归档批注为待处理", "plan_id", planID)
                }
        case "rejected":
                _ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusDraft)
                _ = repository.UpdateLessonPlanVisibility(ctx, planID, models.LPVisibilityPersonal, nil)
        }

        lpLog.Info("教案评审完成", "plan_id", planID, "decision", req.Decision, "round", round)

        // Phase5通道二：approved+评分>=8.5异步萃取
        if req.Decision == "approved" && lp.AIReviewScore != nil && *lp.AIReviewScore >= 8.5 {
                planContent := lp.ContentMarkdown
                subject := lp.Subject
                grade := lp.Grade
                go func() {
                        bgCtx := context.Background()
                        lpLog.Info("触发通道二自动萃取", "plan_id", planID, "ai_score", *lp.AIReviewScore)
                        if err := s.compService.AutoExtractFromLessonPlan(
                                bgCtx, planID, planContent, subject, grade, reviewerID,
                        ); err != nil {
                                lpLog.Error("通道二自动萃取失败", "plan_id", planID, "error", err)
                        }
                }()
        }

        return nil
}

// PublishShared 共享发布
//
// v168改动：共享发布前校验教案正文非空（功能A硬门控）。
// 共享发布是教案对外可见的最终状态，绝不允许无正文教案进入。
// 正常情况下能走到 approved 的教案已经过审核端校验，这里是双保险。
//
// 迭代一 Phase 4 收尾改动（越权封堵）：
//   新增"仅作者本人 OR admin 可共享发布"的身份校验。原先此方法只校验状态，
//   导致任何登录用户拿到一个 approved 教案 ID 就能调它翻状态（越权）。
//   - 作者本人（callerID == lp.AuthorID）：放行。教师对自己已通过的教案做收尾确认。
//   - admin：放行（超级管理员兜底）。通过 FindUserByID 反查角色判定，
//     与 review_v2_service.go 的 ReviewL1 admin 判定惯例一致，无需给本方法加 role 参数。
//   - 其他所有人（含审核员 senior_operator/教研组长/其他教师）：返回 ErrLPNotPublisher。
//     审核员的"公开权力"已在审核通过（approved 即全平台可见）时行使完毕，
//     不应再经手共享发布，避免权限链重复与责任模糊。
func (s *LessonPlanService) PublishShared(ctx context.Context, id string, callerID string) error {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                return s.mapNotFoundErr(err)
        }

        // Phase 4 收尾：身份校验——仅作者本人 OR admin 可共享发布
        isAuthor := lp.AuthorID == callerID
        isAdmin := false
        if !isAuthor {
                // 非作者时才需反查角色判断是否 admin（作者本人直接放行，省一次查询）
                if u, uErr := repository.FindUserByID(ctx, callerID); uErr == nil && u.Role == models.RoleAdmin {
                        isAdmin = true
                }
        }
        if !isAuthor && !isAdmin {
                lpLog.Info("共享发布被拦截：调用者非作者且非管理员", "plan_id", id, "caller", callerID)
                return ErrLPNotPublisher
        }

        if lp.Status != models.LPStatusApproved {
                return errors.New("只有评审通过的教案可以共享发布")
        }
        // v168：正文非空硬校验（双保险）
        if strings.TrimSpace(lp.ContentMarkdown) == "" {
                lpLog.Info("共享发布被拦截：教案正文为空", "plan_id", id)
                return ErrLPContentEmpty
        }
        return repository.UpdateLessonPlanStatus(ctx, id, models.LPStatusPublishedShared)
}

// StartDevelopment 进入课件开发
func (s *LessonPlanService) StartDevelopment(ctx context.Context, id string, callerID string) (*StartDevelopmentResult, error) {
        lp, err := repository.GetLessonPlanByID(ctx, id)
        if err != nil {
                return nil, s.mapNotFoundErr(err)
        }
        if lp.AuthorID != callerID {
                return nil, ErrLPNotAuthor
        }
        if lp.Status != models.LPStatusPublishedPersonal &&
                lp.Status != models.LPStatusApproved &&
                lp.Status != models.LPStatusPublishedShared {
                return nil, ErrLPCannotDevelop
        }
        if lp.Status == models.LPStatusDeveloping {
                return nil, ErrLPAlreadyDeveloping
        }

        // 检查是否已有关联Pipeline（防止重复创建）
        existingPipeline, err := repository.GetPipelineByLessonPlanID(id)
        if err == nil && existingPipeline != nil {
                lpLog.Info("教案已有关联Pipeline，跳过创建", "plan_id", id, "pipeline_id", existingPipeline.ID)
                _ = repository.UpdateLessonPlanStatus(ctx, id, models.LPStatusDeveloping)
                return &StartDevelopmentResult{
                        PipelineID: existingPipeline.ID,
                        Message:    "教案已关联课件开发任务",
                }, nil
        }

        courseCode := fmt.Sprintf("LP-%s-%s", lp.Subject, lp.Grade)
        courseName := fmt.Sprintf("%s %s — %s", lp.Grade, lp.Subject, lp.Topic)
        lessonPlanID := id

        callerIDPtr := callerID
        pipeline := &models.Pipeline{
                CourseCode:   courseCode,
                CourseName:   courseName,
                StartedBy:    &callerIDPtr,
                CurrentStep:  models.StepDbCheck,
                Status:       models.PipelineStatusPending,
                AutoMode:     true,
                ReviewRound:  1,
                LessonPlanID: &lessonPlanID,
        }
        defaultCfg := models.DefaultPipelineConfig()
        cfgBytes, _ := jsonMarshal(defaultCfg)
        pipeline.Config = string(cfgBytes)

        if err := repository.CreatePipeline(pipeline); err != nil {
                lpLog.Error("创建课件开发Pipeline失败", "plan_id", id, "error", err)
                return nil, fmt.Errorf("创建课件开发任务失败: %w", err)
        }

        if err := repository.UpdateLessonPlanStatus(ctx, id, models.LPStatusDeveloping); err != nil {
                lpLog.Error("更新教案状态失败", "plan_id", id, "error", err)
                return nil, err
        }

        lpLog.Info("教案进入课件开发",
                "plan_id", id,
                "pipeline_id", pipeline.ID,
                "course_code", courseCode,
        )

        return &StartDevelopmentResult{
                PipelineID: pipeline.ID,
                Message:    "已创建课件开发任务，请在课件审核系统继续操作",
        }, nil
}

// ForkLessonPlan Fork教案
func (s *LessonPlanService) ForkLessonPlan(ctx context.Context, sourceID string, callerID string) (*models.LessonPlan, error) {
        source, err := repository.GetLessonPlanByID(ctx, sourceID)
        if err != nil {
                return nil, s.mapNotFoundErr(err)
        }

        if source.Status != models.LPStatusPublishedShared &&
                source.Status != models.LPStatusApproved {
                return nil, errors.New("只能fork已发布或评审通过的教案")
        }

        newLP, err := repository.ForkLessonPlan(ctx, sourceID, callerID)
        if err != nil {
                lpLog.Error("Fork教案失败", "source_id", sourceID, "error", err)
                return nil, err
        }
        lpLog.Info("Fork教案成功", "source_id", sourceID, "new_id", newLP.ID, "author", callerID)
        return newLP, nil
}

// ==================== 提示词模板管理 ====================

// CreatePromptTemplate 创建提示词模板
func (s *LessonPlanService) CreatePromptTemplate(ctx context.Context, req *models.CreatePromptTemplateRequest, createdBy string) (*models.PromptTemplate, error) {
        req.Name = strings.TrimSpace(req.Name)
        if req.Name == "" {
                return nil, ErrTemplateNameRequired
        }
        if !models.IsValidTemplateLevel(req.Level) {
                return nil, ErrTemplateLevelInvalid
        }

        pt := &models.PromptTemplate{
                Name:               req.Name,
                Description:        strPtr(req.Description),
                Level:              req.Level,
                OwnerID:            req.OwnerID,
                ParentTemplateID:   req.ParentTemplateID,
                SystemPrompt:       strPtr(req.SystemPrompt),
                ContextRules:       strPtr(req.ContextRules),
                GenerationRules:    strPtr(req.GenerationRules),
                ReviewRules:        strPtr(req.ReviewRules),
                OutputFormat:       strPtr(req.OutputFormat),
                CustomInstructions: strPtr(req.CustomInstructions),
                Subject:            req.Subject,
                GradeRange:         req.GradeRange,
                IsDefault:          req.IsDefault,
                CreatedBy:          &createdBy,
        }

        if err := repository.CreatePromptTemplate(ctx, pt); err != nil {
                lpLog.Error("创建模板失败", "name", req.Name, "error", err)
                return nil, err
        }
        lpLog.Info("创建模板成功", "template_id", pt.ID, "name", pt.Name)
        return pt, nil
}

// ListPromptTemplates 获取模板列表
func (s *LessonPlanService) ListPromptTemplates(ctx context.Context, level string, ownerID string) (*models.PromptTemplateListResponse, error) {
        items, err := repository.ListPromptTemplates(ctx, level, ownerID)
        if err != nil {
                return nil, err
        }
        if items == nil {
                items = []*models.PromptTemplateListItem{}
        }
        return &models.PromptTemplateListResponse{Templates: items, Total: len(items)}, nil
}

// GetPromptTemplate 获取模板详情
func (s *LessonPlanService) GetPromptTemplate(ctx context.Context, id string) (*models.PromptTemplate, error) {
        pt, err := repository.GetPromptTemplateByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrTemplateNotFound) {
                        return nil, ErrTemplateNotFound
                }
                return nil, err
        }
        return pt, nil
}

// UpdatePromptTemplate 更新模板
func (s *LessonPlanService) UpdatePromptTemplate(ctx context.Context, id string, req *models.UpdatePromptTemplateRequest) error {
        req.Name = strings.TrimSpace(req.Name)
        if req.Name == "" {
                return ErrTemplateNameRequired
        }
        if err := repository.UpdatePromptTemplate(ctx, id, req); err != nil {
                if errors.Is(err, repository.ErrTemplateNotFound) {
                        return ErrTemplateNotFound
                }
                return err
        }
        return nil
}

// ResolvePromptTemplate 解析继承链
func (s *LessonPlanService) ResolvePromptTemplate(ctx context.Context, templateID string) (*models.ResolvedPromptTemplate, error) {
        return repository.ResolvePromptTemplateChain(ctx, templateID)
}

// ==================== 辅助方法 ====================

func (s *LessonPlanService) mapNotFoundErr(err error) error {
        if errors.Is(err, repository.ErrLessonPlanNotFound) {
                return ErrLPNotFound
        }
        return err
}

// jsonMarshal 序列化为JSON字节（供StartDevelopment序列化Pipeline配置使用）
func jsonMarshal(v interface{}) ([]byte, error) {
        return json.Marshal(v)
}
