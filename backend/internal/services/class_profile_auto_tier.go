package services

// class_profile_auto_tier.go — 班级学情·按分数线自动分层（批次2d）
//
// 一句话职责：老师设两条分数线（A 线 / C 线），系统按每个学生的最新成绩(latest_score)
// 自动归入 ABC 三层并批量写回，省去逐个学生手动分层的繁琐。
//
// 分层规则：
//   latest_score >= aLine            → A 层（拔尖）
//   cLine <= latest_score < aLine    → B 层（中等）
//   latest_score <  cLine            → C 层（学困）
//   latest_score 为空（从未导入成绩）→ 跳过，tier 原样不动（分数线无从判断）
//
// 产品决策（已与 Yuhan 敲定）：
//   - 按分数线【重算全部有成绩的学生】，覆盖其现有分层（简单可预期）。
//   - 无成绩的学生整个跳过、tier 不动（不清也不改）——分数线只动它能判断的学生。
//   - 后端批量端点一次请求完成（前端不循环调单个更新）。
//
// 实现取舍：复用现成 repository.UpdateClassStudent 逐个写回（N 个学生 N 次 UPDATE），
// 班级规模通常几十人，量小可接受，不为此写专门批量 SQL（改动面最小、最稳）。
// 分层只动 Tier 字段，其余（学号/成绩/薄弱点/备注）原样保留。

import (
	"context"
	"errors"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ---------- 业务错误（sentinel）----------

var (
	// ErrAutoTierLineInvalid 分数线非法（A 线必须 > C 线，且都 >= 0）
	ErrAutoTierLineInvalid = errors.New("分数线设置不合法：A 线必须高于 C 线，且均不能为负")
)

// ---------- 请求/响应 DTO（本文件内定义，纯传输结构不污染 model 文件）----------

// AutoTierRequest 自动分层请求（两条分数线）
type AutoTierRequest struct {
	ALine float64 `json:"a_line"` // A 层下限：>= 此分 → A
	CLine float64 `json:"c_line"` // C 层上限：< 此分 → C；介于 [CLine, ALine) → B
}

// AutoTierResult 自动分层结果（各层人数 + 跳过统计）
type AutoTierResult struct {
	TotalStudents int `json:"total_students"` // 班级总学生数
	TierA         int `json:"tier_a"`         // 归入 A 层人数
	TierB         int `json:"tier_b"`         // 归入 B 层人数
	TierC         int `json:"tier_c"`         // 归入 C 层人数
	SkippedNoScore int `json:"skipped_no_score"` // 因无成绩被跳过、tier 未动的人数
	Updated       int `json:"updated"`        // 实际发生 tier 变更并写库的人数
}

// ========================================================================
// 对外主方法：AutoTierStudents —— 按分数线批量自动分层
// ========================================================================

// AutoTierStudents 按分数线把本班有成绩的学生自动归入 ABC 三层并写回。
//
// 流程：
//   1. ensureProfileOwned 校验班级卡归属当前老师（复用同包闸门）。
//   2. 校验分数线合法（A 线 > C 线，均 >= 0）。
//   3. 拉本班全部学生，逐个按 latest_score 归层；无成绩则跳过不动。
//   4. 仅当 tier 实际发生变化时才写库（减少无谓 UPDATE）。
//   5. 返回各层人数 + 跳过统计。
func (s *ClassProfileService) AutoTierStudents(
	ctx context.Context, userID, classProfileID string, req *AutoTierRequest,
) (*AutoTierResult, error) {

	// 1) 归属校验
	if _, err := s.ensureProfileOwned(ctx, userID, classProfileID); err != nil {
		return nil, err
	}

	// 2) 分数线合法性
	if req.ALine < 0 || req.CLine < 0 || req.ALine <= req.CLine {
		return nil, ErrAutoTierLineInvalid
	}

	// 3) 拉学生
	students, err := repository.ListClassStudents(ctx, classProfileID)
	if err != nil {
		return nil, err
	}

	result := &AutoTierResult{TotalStudents: len(students)}

	for _, st := range students {
		// 无成绩 → 跳过，tier 不动
		if st.LatestScore == nil {
			result.SkippedNoScore++
			continue
		}
		score := *st.LatestScore

		// 按分数线归层
		var newTier string
		switch {
		case score >= req.ALine:
			newTier = models.StudentTierA
			result.TierA++
		case score < req.CLine:
			newTier = models.StudentTierC
			result.TierC++
		default:
			newTier = models.StudentTierB
			result.TierB++
		}

		// 仅当 tier 实际变化时才写库
		if st.Tier == newTier {
			continue
		}
		st.Tier = newTier
		if err := repository.UpdateClassStudent(ctx, st); err != nil {
			// 单个写库失败不整体中断：记入日志，继续处理其余学生
			classProfileLog.Warn("自动分层写回单个学生失败",
				"profile", classProfileID, "student", st.ID, "err", err.Error())
			continue
		}
		result.Updated++
	}

	classProfileLog.Info("按分数线自动分层完成",
		"profile", classProfileID, "owner", userID,
		"aLine", req.ALine, "cLine", req.CLine,
		"A", result.TierA, "B", result.TierB, "C", result.TierC,
		"skipped", result.SkippedNoScore, "updated", result.Updated)

	return result, nil
}
