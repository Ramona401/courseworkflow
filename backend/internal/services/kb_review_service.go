package services

// kb_review_service.go — 知识库课标审核服务（解码人话队列 + 三选一 + commit + 蓝绿切换）
//
// 职责（PRD §3.4 专利/商业秘密保护核心）：
//   审核员所见永远是「人话」——索引原文(KP ... | SJ:M|DP:2 [K]...)绝不进任何返回结构，
//   一律经 DecodeCurriculumIndex 解码为中文卡片(数学·理解应用层·学业要求:...)。
//
// 四类操作：
//   GetReviewQueue  取某 job 待审单元，多轮草稿逐轮解码成人话并排，低置信高亮 conflicts
//   ReviewAction    三选一：confirm(采纳仲裁选中版) / select(选指定轮) / reject(退回)
//   CommitBatch     把 approved/auto_passed 单元的 final_line 解码拆列，灌 curriculum_standards
//                   （带 batch_tag、status=候选态 staged，待整批 SwitchBatch 才转 active）
//   SwitchBatch     蓝绿切换：旧 active→archived、新批→active（调 repo 单事务）
//
// commit 落库状态约定：
//   灌入时 status='staged'（非 active 候选态），消费端只读 active 看不到；
//   SwitchBatch 把该 batch_tag 整体转 active、旧批转 archived，实现瞬时蓝绿切换。

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// KBStagedStatus commit 灌入时的候选态（非 active，消费端不可见，待蓝绿切换转正）
const KBStagedStatus = "staged"

// ==================== 服务定义 ====================

// KBReviewService 知识库课标审核服务
type KBReviewService struct{}

// NewKBReviewService 创建审核服务（无外部依赖，纯编排 repo + 解码服务）
func NewKBReviewService() *KBReviewService {
	return &KBReviewService{}
}

// ==================== GetReviewQueue 审核队列（解码人话） ====================

// GetReviewQueue 取某 job 的待人工审核单元（review_status=need_review），
// 把每个单元的多轮草稿逐轮解码成人话卡片并排返回，附仲裁冲突点供前端高亮。
// 索引原文不进返回结构，只返回解码后的中文。
func (s *KBReviewService) GetReviewQueue(ctx context.Context, jobID string) ([]*models.KBReviewItemView, error) {
	items, err := repository.ListKBItemsByJobAndReviewStatus(ctx, jobID, models.KBReviewStatusNeedReview)
	if err != nil {
		return nil, fmt.Errorf("读取待审单元失败: %w", err)
	}

	views := make([]*models.KBReviewItemView, 0, len(items))
	for _, it := range items {
		view := s.buildReviewItemView(it)
		views = append(views, view)
	}
	return views, nil
}

// GetItemView 取单个单元的审核视图（人话），供审核员查看任意单元（含已通过的复核）
func (s *KBReviewService) GetItemView(ctx context.Context, itemID string) (*models.KBReviewItemView, error) {
	it, err := repository.GetKBItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("单元不存在: %w", err)
	}
	return s.buildReviewItemView(it), nil
}

// buildReviewItemView 把一个 item 的多轮草稿解码成人话视图
func (s *KBReviewService) buildReviewItemView(it *models.KBCompressItem) *models.KBReviewItemView {
	view := &models.KBReviewItemView{
		ItemID:        it.ID,
		Seq:           it.Seq,
		Confidence:    it.Confidence,
		ReviewStatus:  it.ReviewStatus,
		SourceExcerpt: it.SourceExcerpt,
		PageLabel:     it.PageLabel,
		Rounds:        []models.KBRoundView{},
	}

	// 解析仲裁结论，取冲突点与选中轮
	if arb := models.ParseArbitration(it.Arbitration); arb != nil {
		view.Conflicts = arb.Conflicts
		view.ChosenRound = arb.ChosenRound
	}
	if view.Conflicts == nil {
		view.Conflicts = []string{}
	}

	// 逐轮草稿解码成人话
	rounds := models.ParseDraftRounds(it.DraftRounds)
	for _, r := range rounds {
		rv := models.KBRoundView{
			Round: r.Round,
			Model: r.Model,
			Error: r.Error,
		}
		if r.Error == "" && strings.TrimSpace(r.Line) != "" {
			rv.Decoded = DecodeCurriculumIndex(r.Line)
		}
		view.Rounds = append(view.Rounds, rv)
	}
	return view
}

// ==================== ReviewAction 三选一 ====================

// ReviewAction 审核员对单个单元执行三选一动作。
//
//	confirm：采纳仲裁选中的轮次作 final_line，状态置 approved
//	select ：采纳审核员指定的轮次(req.ChosenRound)作 final_line，状态置 approved
//	reject ：退回，状态置 rejected（不入库；如需重压由后续重跑任务处理）
func (s *KBReviewService) ReviewAction(ctx context.Context, itemID string, req *models.KBReviewActionRequest, reviewerID string) error {
	it, err := repository.GetKBItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("单元不存在: %w", err)
	}

	switch req.Action {
	case models.KBReviewActionReject:
		return repository.UpdateKBItemReview(ctx, itemID,
			models.KBReviewStatusRejected, it.FinalLine, reviewerID, req.ReviewNote)

	case models.KBReviewActionConfirm:
		// 采纳仲裁选中轮（若 final_line 已是高置信定稿则直接用）
		finalLine := it.FinalLine
		if strings.TrimSpace(finalLine) == "" {
			arb := models.ParseArbitration(it.Arbitration)
			chosen := 0
			if arb != nil {
				chosen = arb.ChosenRound
			}
			finalLine = pickRoundLine(models.ParseDraftRounds(it.DraftRounds), chosen)
		}
		if strings.TrimSpace(finalLine) == "" {
			return fmt.Errorf("无可采纳的有效索引，请改用选版或退回")
		}
		return repository.UpdateKBItemReview(ctx, itemID,
			models.KBReviewStatusApproved, finalLine, reviewerID, req.ReviewNote)

	case models.KBReviewActionSelect:
		// 采纳审核员指定的轮次
		if req.ChosenRound <= 0 {
			return fmt.Errorf("请指定要采纳的轮次")
		}
		finalLine := pickRoundLine(models.ParseDraftRounds(it.DraftRounds), req.ChosenRound)
		if strings.TrimSpace(finalLine) == "" {
			return fmt.Errorf("指定轮次无有效索引")
		}
		return repository.UpdateKBItemReview(ctx, itemID,
			models.KBReviewStatusApproved, finalLine, reviewerID, req.ReviewNote)

	default:
		return fmt.Errorf("无效的审核动作: %s", req.Action)
	}
}

// ==================== CommitBatch 灌入目标表（候选态） ====================

// CommitBatchResult commit 结果摘要
type CommitBatchResult struct {
	Committed int      `json:"committed"` // 成功灌入条数
	Skipped   int      `json:"skipped"`   // 跳过条数（无 final_line/解码失败/非通过态）
	Errors    []string `json:"errors"`    // 各条错误（截断展示）
}

// CommitBatch 把某 job 下 approved + auto_passed 的单元灌入 curriculum_standards。
// 用 DecodeCurriculumIndex 把 final_line 解码拆成结构化各列，status 置 staged 候选态。
// 灌入后回填 kb_compress_items.committed_ref。整批灌完不自动切换，需再调 SwitchBatch。
func (s *KBReviewService) CommitBatch(ctx context.Context, jobID string, batchTag string) (*CommitBatchResult, error) {
	if strings.TrimSpace(batchTag) == "" {
		return nil, fmt.Errorf("批次标识不能为空")
	}

	approved, err := repository.ListKBItemsByJobAndReviewStatus(ctx, jobID, models.KBReviewStatusApproved)
	if err != nil {
		return nil, fmt.Errorf("读取已通过单元失败: %w", err)
	}
	autoPassed, err := repository.ListKBItemsByJobAndReviewStatus(ctx, jobID, models.KBReviewStatusAutoPassed)
	if err != nil {
		return nil, fmt.Errorf("读取自动通过单元失败: %w", err)
	}
	all := append(approved, autoPassed...)

	result := &CommitBatchResult{Errors: []string{}}
	for _, it := range all {
		if it.Committed {
			result.Skipped++
			continue
		}
		if strings.TrimSpace(it.FinalLine) == "" {
			result.Skipped++
			continue
		}
		row := s.decodeToInsertRow(it.FinalLine, batchTag, it.ReviewStatus)
		if row == nil || row.KPCode == "" || row.KPName == "" {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("seq=%d 解码缺关键字段(kp_code/kp_name)，跳过", it.Seq))
			continue
		}
		newID, insErr := repository.InsertCurriculumStandard(ctx, row)
		if insErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("seq=%d 灌入失败: %s", it.Seq, truncateKB(insErr.Error(), 120)))
			continue
		}
		_ = repository.MarkKBItemCommitted(ctx, it.ID, newID)
		result.Committed++
	}
	return result, nil
}

// decodeToInsertRow 把 final_line 解码并映射为目标表插入行（status=staged 候选态）
// 复用 DecodeCurriculumIndex 拆出人话各字段，再映射回结构化列。
func (s *KBReviewService) decodeToInsertRow(finalLine string, batchTag string, reviewStatus string) *repository.CurriculumInsertRow {
	decoded := DecodeCurriculumIndex(finalLine)
	if decoded == nil {
		return nil
	}

	// 置信度按审核路径分层：人工确认(approved)高于高置信自动通过(auto_passed)
	confidence := 80
	if reviewStatus == models.KBReviewStatusApproved {
		confidence = 90 // 人工逐条确认，可信度最高
	}
	row := &repository.CurriculumInsertRow{
		KPCode:     decoded.KPCode,
		BatchTag:   batchTag,
		Status:     KBStagedStatus,
		Confidence: confidence,
	}

	// 学科：解码出的是中文名（如"数学"），直接用
	row.Subject = decoded.SubjectName
	// 学段
	row.Stage = decoded.StageName
	// 年级：从解码的中文年级名反推数字（解码服务输出"三年级"等，这里取数字）
	row.GradeNum = gradeNameToNum(decoded.GradeName)
	// 深度档：从"理解应用层"等反推 1-3
	row.DepthLevel = depthNameToNum(decoded.DepthName)

	// 语义字段：从解码的 Fields 里按标签名取
	for _, f := range decoded.Fields {
		switch f.Label {
		case "知识点名称":
			row.KPName = f.Content
		case "学业要求":
			row.AcademicRequirement = f.Content
		case "内容边界":
			row.ContentRequirement = f.Content
		case "教学提示":
			row.TeachingHint = f.Content
		case "所属领域":
			row.Domain = f.Content
		case "核心素养":
			row.CoreCompetency = f.Content
		}
	}
	row.SourceRef = "AI压缩入库·" + batchTag
	return row
}

// ==================== SwitchBatch 蓝绿切换 ====================

// SwitchBatchResult 切换结果
type SwitchBatchResult struct {
	Archived  int `json:"archived"`  // 归档的旧 active 条数
	Activated int `json:"activated"` // 激活的新批条数
}

// SwitchBatch 蓝绿切换：把指定 batchTag 转 active、其余旧 active 转 archived。
// 切换前先核对该批已灌入条数，0 条则拒绝切换（避免切到空批导致消费端无数据）。
func (s *KBReviewService) SwitchBatch(ctx context.Context, batchTag string) (*SwitchBatchResult, error) {
	if strings.TrimSpace(batchTag) == "" {
		return nil, fmt.Errorf("批次标识不能为空")
	}
	cnt, err := repository.CountCurriculumByBatch(ctx, batchTag)
	if err != nil {
		return nil, fmt.Errorf("核对批次条数失败: %w", err)
	}
	if cnt == 0 {
		return nil, fmt.Errorf("批次 %s 无任何已灌入数据，拒绝切换（请先 commit）", batchTag)
	}

	archived, activated, err := repository.SwitchCurriculumBatch(ctx, batchTag)
	if err != nil {
		return nil, fmt.Errorf("蓝绿切换失败: %w", err)
	}
	return &SwitchBatchResult{Archived: archived, Activated: activated}, nil
}

// ==================== 辅助函数 ====================

// gradeNameToNum 把解码的中文年级名反推为数字（1-12，学段级=0）
func gradeNameToNum(name string) int {
	m := map[string]int{
		"一年级": 1, "二年级": 2, "三年级": 3, "四年级": 4, "五年级": 5, "六年级": 6,
		"七年级(初一)": 7, "八年级(初二)": 8, "九年级(初三)": 9,
		"高一": 10, "高二": 11, "高三": 12,
		"学段级(不绑定具体年级)": 0,
	}
	if n, ok := m[name]; ok {
		return n
	}
	// 兜底：尝试从名字里抠数字
	for cn, n := range map[string]int{"一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6, "七": 7, "八": 8, "九": 9} {
		if strings.Contains(name, cn+"年级") {
			return n
		}
	}
	return 0
}

// depthNameToNum 把解码的深度档中文名反推为 1-3
func depthNameToNum(name string) int {
	switch name {
	case "体验感知层":
		return 1
	case "理解应用层":
		return 2
	case "分析迁移层":
		return 3
	default:
		return 2 // 兜底取中间档
	}
}

// truncateKB 按 rune 安全截断（错误信息展示用）
func truncateKB(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}

// 让 strconv 被使用（保留以备序号/数字解析扩展）
var _ = strconv.Atoi
