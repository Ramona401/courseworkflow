package services

// unit_plan_service.go — 单元方案（大单元备课）业务逻辑（独立模块，不碰课时备课主线）
//
// 大单元挂载（前端入口）新增：
//   ListMountableUnitPlans —— 供「教案挂载单元方案选择器」列出可挂载的单元方案。
//   只返回 active（调 repository.ListActiveUnitPlansForMount），口径与注入层焊死一致。
//   可见范围解析（admin 全部 / 非 admin 全局∪本组∪本校）复用本文件已有的
//   教研组查询 + resolveUserSchoolIDs，与 ListUnitPlans 完全同源。
//
// v233 变更（课程大纲教材版本绑定，对齐备课工坊）：
//   1) StartSession 接收并规范化 req.CourseOutlinePublisher（三态：nil=不关联 / ""=通用版 /
//      具名=该版本），落到草稿实体随 CreateUnitPlanDraft 一次性入库——会话建立时定版；
//   2) buildSystemPrompt 的大纲注入从旧「MatchBestOutline 自动学段打分取一份」改为三态分流：
//        publisher == nil → 静默不注入（存量老数据均为 NULL，零回归）；
//        publisher != nil → MatchOutlinesByPublisher 按选定版本精确匹配（零跨版本兜底），
//                           命中多份用 BuildCourseOutlinesContext 全部注入（与备课工坊同口径）；
//      并返回 injected 标记，供 StartSession 按"是否真的注入了大纲"生成不同开场白，
//      避免未注入时仍让 AI"从已注入的大纲中定位本单元"的误导话术；
//   3) Chat 每轮从方案实体重读 CourseOutlinePublisher 传入 buildSystemPrompt，天然生效。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var unitPlanLog = logger.WithModule("services.unit_plan")

// 复用备课文本场景（已配模型+走境内外分流），避免新场景回落到'豆包'致503
const unitDesignSceneCode = "lesson_plan"
const unitDesignPromptKey = "prompt_unit_design"

var (
	ErrUnitPlanFieldRequired = errors.New("学科、年级、单元为必填")
	ErrUnitPlanScopeInvalid  = errors.New("归属类型非法")
	ErrUnitPlanNoPermission  = errors.New("您没有权限产出该归属的单元方案")
	ErrUnitPlanNotOwner      = errors.New("只能操作自己创建的单元方案草稿")
	ErrUnitPlanSaveEmpty     = errors.New("方案正文为空，无法保存")
)

// UnitPlanService 单元方案服务（持 cfg 供 AI 调用取密钥与兜底模型）
type UnitPlanService struct {
	cfg *config.Config
}

func NewUnitPlanService(cfg *config.Config) *UnitPlanService {
	return &UnitPlanService{cfg: cfg}
}

// ---------- 列表/详情 ----------

func (s *UnitPlanService) ListUnitPlans(ctx context.Context, role, userID string) ([]*models.UnitPlanListItem, error) {
	if role == models.RoleAdmin {
		return repository.ListUnitPlans(ctx, true, nil, nil, userID)
	}
	groups, gErr := repository.GetUserTeachingGroups(ctx, userID)
	if gErr != nil {
		unitPlanLog.Warn("查询用户教研组失败", "user", userID, "error", gErr)
	}
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	schoolIDs := s.resolveUserSchoolIDs(ctx, role, userID)
	return repository.ListUnitPlans(ctx, false, groupIDs, schoolIDs, userID)
}

// ListMountableUnitPlans 列出「可被教案挂载」的单元方案（挂载选择器专用）
//
// 与 ListUnitPlans 的区别：只返回 active（不含任何草稿），口径与注入层焊死一致——
// 老师在选择器里看到的每一项都是挂上即生效的（注入层只注入 active）。
// 可见范围解析与 ListUnitPlans 完全同源（admin 全部 active / 非 admin 全局∪本组∪本校 active）。
//
// subject 选填：传非空则只列该学科的单元方案（挂载选择器通常按当前教案学科收窄，减少噪音）；
// 传空串则不按学科过滤，列出全部可见 active。
func (s *UnitPlanService) ListMountableUnitPlans(ctx context.Context, role, userID, subject string) ([]*models.UnitPlanListItem, error) {
	if role == models.RoleAdmin {
		return repository.ListActiveUnitPlansForMount(ctx, true, nil, nil, strings.TrimSpace(subject))
	}
	groups, gErr := repository.GetUserTeachingGroups(ctx, userID)
	if gErr != nil {
		unitPlanLog.Warn("查询用户教研组失败（挂载选择器）", "user", userID, "error", gErr)
	}
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	schoolIDs := s.resolveUserSchoolIDs(ctx, role, userID)
	return repository.ListActiveUnitPlansForMount(ctx, false, groupIDs, schoolIDs, strings.TrimSpace(subject))
}

func (s *UnitPlanService) GetUnitPlan(ctx context.Context, role, userID, id string) (*models.UnitPlan, []models.UnitPlanMessage, error) {
	p, err := repository.GetUnitPlanByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if p.Status == models.UnitPlanStatusDraft && p.CreatedBy != userID && role != models.RoleAdmin {
		return nil, nil, ErrUnitPlanNoPermission
	}
	return p, models.ParseUnitPlanLog(p.ConversationLog), nil
}

// ---------- 开始会话（建草稿 + 出第一步） ----------

func (s *UnitPlanService) StartSession(ctx context.Context, role, userID string, req *models.StartUnitPlanRequest) (*models.UnitPlan, string, error) {
	if !models.IsValidUnitPlanScope(req.Scope) {
		return nil, "", ErrUnitPlanScopeInvalid
	}
	if req.Scope == models.UnitPlanScopeSystem {
		req.ScopeTargetID = models.UnitPlanSystemTargetID
	}
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Grade) == "" ||
		strings.TrimSpace(req.Unit) == "" || strings.TrimSpace(req.ScopeTargetID) == "" {
		return nil, "", ErrUnitPlanFieldRequired
	}
	if !s.canManageScope(ctx, role, userID, req.Scope, req.ScopeTargetID) {
		return nil, "", ErrUnitPlanNoPermission
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(req.Grade) + strings.TrimSpace(req.Volume) + strings.TrimSpace(req.Unit) + " 单元方案"
	}

	// v233：规范化教材版本三态（与教案侧 UpdateLessonPlanCourseOutlinePublisher 同口径）：
	//   nil 保持 nil（不关联大纲，含老客户端不传该字段的情况）；
	//   非 nil 去空白后保留——注意空串是有效版本值（通用/不限版本），不可归一为 nil。
	if req.CourseOutlinePublisher != nil {
		trimmed := strings.TrimSpace(*req.CourseOutlinePublisher)
		req.CourseOutlinePublisher = &trimmed
	}

	draft := &models.UnitPlan{
		Scope:                  req.Scope,
		ScopeTargetID:          req.ScopeTargetID,
		Subject:                strings.TrimSpace(req.Subject),
		Grade:                  strings.TrimSpace(req.Grade),
		Volume:                 strings.TrimSpace(req.Volume),
		Unit:                   strings.TrimSpace(req.Unit),
		Title:                  title,
		SourceType:             models.UnitPlanSourceGenerated,
		CreatedBy:              userID,
		CourseOutlinePublisher: req.CourseOutlinePublisher,
	}
	if err := repository.CreateUnitPlanDraft(ctx, draft); err != nil {
		return nil, "", err
	}

	systemPrompt, outlineInjected := s.buildSystemPrompt(ctx, draft.Subject, draft.Grade, draft.CourseOutlinePublisher)

	// 开场白按"是否真的注入了大纲"分流：
	//   已注入 → 让 AI 从已注入大纲中定位本单元并回显篇目课时（原有话术）；
	//   未注入（未关联/选定版本下无大纲/大纲正文为空）→ 请老师自行提供篇目课时，
	//   绝不能让 AI 去"已注入的大纲"里找（根本没注入，会诱发幻觉或"读不到资料"话术）。
	var kickoff string
	if outlineInjected {
		kickoff = fmt.Sprintf(
			"现在开始为以下单元做大单元整体教学设计：\n学科【%s】 年级【%s】 册次【%s】 单元【%s】。\n"+
				"请执行步骤1（确认基础信息）：从已注入的课程大纲中定位本单元，回显学科/年级/册次/单元名/本单元课文篇目（含略读带*）与大致课时，请老师确认或补充。严格只做这一步，然后停下等确认。",
			draft.Subject, draft.Grade, upDash(draft.Volume), draft.Unit,
		)
	} else {
		kickoff = fmt.Sprintf(
			"现在开始为以下单元做大单元整体教学设计：\n学科【%s】 年级【%s】 册次【%s】 单元【%s】。\n"+
				"请执行步骤1（确认基础信息）：本次备课未注入课程大纲，请回显学科/年级/册次/单元名，并请老师提供本单元的课文篇目（含略读带*）与大致课时；不要凭记忆猜测篇目。严格只做这一步，然后停下等确认。",
			draft.Subject, draft.Grade, upDash(draft.Volume), draft.Unit,
		)
	}

	reply, err := s.callUnitAI(ctx, userID, systemPrompt, kickoff)
	if err != nil {
		return draft, "", err
	}
	if e := repository.AppendUnitPlanMessage(ctx, draft.ID, "assistant", reply); e != nil {
		unitPlanLog.Warn("追加开场消息失败", "id", draft.ID, "error", e)
	}
	return draft, reply, nil
}

// ---------- 对话一轮 ----------

func (s *UnitPlanService) Chat(ctx context.Context, role, userID, id, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errors.New("消息不能为空")
	}
	p, err := repository.GetUnitPlanByID(ctx, id)
	if err != nil {
		return "", err
	}
	if p.CreatedBy != userID {
		return "", ErrUnitPlanNotOwner
	}
	history := models.ParseUnitPlanLog(p.ConversationLog)
	// v233：每轮从方案实体重读版本选择做三态注入（会话建立时已定版落库）
	systemPrompt, _ := s.buildSystemPrompt(ctx, p.Subject, p.Grade, p.CourseOutlinePublisher)
	userPrompt := buildUnitChatUserPrompt(history, message)

	reply, err := s.callUnitAI(ctx, userID, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	if e := repository.AppendUnitPlanMessage(ctx, id, "user", message); e != nil {
		unitPlanLog.Warn("追加用户消息失败", "id", id, "error", e)
	}
	if e := repository.AppendUnitPlanMessage(ctx, id, "assistant", reply); e != nil {
		unitPlanLog.Warn("追加助手消息失败", "id", id, "error", e)
	}
	return reply, nil
}

// ---------- 定稿保存 / 删除 ----------

func (s *UnitPlanService) Save(ctx context.Context, role, userID, id string, req *models.SaveUnitPlanRequest) error {
	p, err := repository.GetUnitPlanByID(ctx, id)
	if err != nil {
		return err
	}
	if p.CreatedBy != userID && role != models.RoleAdmin {
		return ErrUnitPlanNotOwner
	}
	if strings.TrimSpace(req.Content) == "" {
		return ErrUnitPlanSaveEmpty
	}
	if strings.TrimSpace(req.Title) == "" {
		req.Title = p.Title
	} else {
		req.Title = strings.TrimSpace(req.Title)
	}
	req.UnitTheme = strings.TrimSpace(req.UnitTheme)
	return repository.FinalizeUnitPlan(ctx, id, req)
}

func (s *UnitPlanService) Delete(ctx context.Context, role, userID, id string) error {
	p, err := repository.GetUnitPlanByID(ctx, id)
	if err != nil {
		return err
	}
	if p.CreatedBy != userID && !s.canManageScope(ctx, role, userID, p.Scope, p.ScopeTargetID) {
		return ErrUnitPlanNoPermission
	}
	return repository.DeleteUnitPlan(ctx, id)
}

// ---------- 内部：AI 调用 ----------

func (s *UnitPlanService) callUnitAI(ctx context.Context, userID, systemPrompt, userPrompt string) (string, error) {
	cfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		unitDesignSceneCode,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("AI配置获取失败: %w", err)
	}
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	uid := userID
	traceCtx := &ai.TraceContext{
		SceneCode: unitDesignSceneCode,
		UserID:    &uid,
		SchoolID:  schoolIDPtr(schoolID),
	}
	result, err := ai.CallAI(cfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Content), nil
}

// buildSystemPrompt 取 prompt_unit_design + 按选定教材版本三态注入课程大纲
//
// v233（对齐备课工坊，替代旧 MatchBestOutline 自动打分注入）：
//   publisher == nil        → 不注入任何大纲（未关联；存量老数据均为此态，零回归），返回 (base, false)；
//   publisher 指向 ""        → 通用/不限版本：只注入 publisher 为空串的大纲；
//   publisher 指向 "人教版"  → 只注入该版本大纲（MatchOutlinesByPublisher 零跨版本兜底）。
// 命中多份（相邻年级/不同册次）用 BuildCourseOutlinesContext 全部注入，与备课工坊同口径。
// 第二返回值 injected：是否真的注入了大纲内容（供开场白分流，宁缺不错——匹配不到就是 false）。
func (s *UnitPlanService) buildSystemPrompt(ctx context.Context, subject, grade string, publisher *string) (string, bool) {
	base := ""
	if p, err := repository.GetCurrentPromptByKey(unitDesignPromptKey); err == nil && p != nil {
		base = p.Content
	} else {
		unitPlanLog.Warn("取单元设计提示词失败，使用空骨架", "error", err)
	}

	// 三态分流第一层：未关联大纲 → 静默不注入
	if publisher == nil {
		return base, false
	}

	candidates, err := repository.ListActiveOutlinesBySubject(ctx, subject)
	if err != nil {
		unitPlanLog.Warn("查课程大纲候选失败", "subject", subject, "error", err)
		return base, false
	}

	// 版本精确匹配（学段相交 + publisher 严格相等，零跨版本兜底；宁缺不错）
	hits := MatchOutlinesByPublisher(grade, *publisher, candidates)
	if len(hits) == 0 {
		unitPlanLog.Info("选定版本下无匹配课程大纲，跳过注入",
			"subject", subject, "grade", grade, "publisher", *publisher)
		return base, false
	}

	outlineCtx := BuildCourseOutlinesContext(hits)
	if outlineCtx == "" {
		// 命中大纲但正文全为空，同样视为未注入
		return base, false
	}
	base += outlineCtx
	unitPlanLog.Info("单元备课已按选定版本注入课程大纲",
		"subject", subject, "grade", grade, "publisher", *publisher, "count", len(hits))
	return base, true
}

// ---------- 内部：权限（镜像 course_outline_service.canManageScope）----------

func (s *UnitPlanService) canManageScope(ctx context.Context, role, userID, scope, targetID string) bool {
	if role == models.RoleAdmin {
		return true
	}
	switch scope {
	case models.UnitPlanScopeSystem:
		return false
	case models.UnitPlanScopeSchool:
		if role != models.RoleSeniorOperator {
			return false
		}
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err != nil || school == nil {
			return false
		}
		return school.ID == targetID
	case models.UnitPlanScopeGroup:
		ok, err := repository.IsGroupLeadOrBackbone(ctx, targetID, userID)
		if err != nil {
			unitPlanLog.Warn("校验组长/骨干失败", "group", targetID, "user", userID, "error", err)
			return false
		}
		return ok
	}
	return false
}

func (s *UnitPlanService) resolveUserSchoolIDs(ctx context.Context, role, userID string) []string {
	if role == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err == nil && school != nil && school.ID != "" {
			return []string{school.ID}
		}
	}
	return []string{}
}

// ---------- 内部：纯函数辅助 ----------

func upDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（未填）"
	}
	return s
}

func buildUnitChatUserPrompt(history []models.UnitPlanMessage, newMsg string) string {
	var b strings.Builder
	b.WriteString("【到目前为止的对话记录】\n")
	if len(history) == 0 {
		b.WriteString("（无）\n")
	} else {
		for _, m := range history {
			who := "老师"
			if m.Role == "assistant" {
				who = "你（大单元架构师）"
			}
			b.WriteString(who + "：" + m.Content + "\n\n")
		}
	}
	b.WriteString("【老师本轮输入】\n")
	b.WriteString(newMsg)
	b.WriteString("\n\n请依据你的工作纪律，只推进当前这一步（或按老师意见修改当前步），然后停下等确认。")
	return b.String()
}
