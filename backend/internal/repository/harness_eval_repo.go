package repository

// harness_eval_repo.go — Harness 输出采集表数据访问层
//
// 对应数据库表:harness_eval_samples
//
// 用途:
//   采集备课工坊每轮 AI 对话的「输入画像 + 输出全文」,落库供后台 judge 分析与 admin 人工标注,
//   驱动 harness 的「体检 + 进化」。仅 admin 后台使用,数据死锁后台、永不外泄。
//
// 本期(采集阶段)只实现【写入】一个函数 InsertHarnessEvalSample。
// judge 端(捞 pending 样本 / 回填 judge_result)与 admin 端(裁决 / 清理全文)的查询函数,
// 待 judge 与看板阶段再按届时确定的需求补入本文件,现在不预写以免返工。
//
// 设计要点:
//   1. 软关联:lesson_plan_id / user_id / school_id 均为字符串、表上无外键,
//      被引用对象(教案/用户/学校)删数据时本表不连带、不报错(镜像 teacher_assistant_prefs 风格)。
//   2. 采集失败【绝不阻塞主对话流】:本函数只负责把一条记录写进库并如实返回 error,
//      是否容忍失败由调用方决定 —— 调用方(processChatStageAsync)会用独立 goroutine 调用本函数,
//      失败仅记 Warn 日志,不影响老师正在进行的备课。
//   3. 写库范式对齐 teacher_assistant_pref_repo.go:database.DB.Exec + fmt.Errorf 包错。

import (
        "context"
        "fmt"

        "tedna/internal/database"
)

// HarnessEvalSampleInput 采集一条 harness 评估样本的入参。
//
// 用结构体而非一长串位置参数,是为了让调用侧每个字段带名字传入,
// 避免多个 string 字段顺序传错导致的「编译能过、运行时数据错位」隐患。
//
// 字段分三组(与表结构一致),judge_status 由 DB 默认 'pending',
// judge_result / admin_verdict / admin_note 采集时一律留空,后续由 judge / admin 回填,
// 故本入参【不含】结论组字段。
type HarnessEvalSampleInput struct {
        // ---------- 定位组 ----------
        LessonPlanID string // 所属教案 ID(软关联)
        StageCode    string // 阶段码 analyze/design/write/review/revise
        UserID       string // 教案作者 ID(软关联,空串可)
        SchoolID     string // 作者所属学校 ID(软关联,空串=查不到)

        // ---------- 画像组(永久,轻量)----------
        ModelUsed     string // generator 本轮实际用的模型(分析以此为准)
        IsDowngraded  bool   // 采集时算的「是否因学校未授权而降级走境内模型」
        AssistantID   string // 本轮挂载的助手 ID(空串=纯骨架)
        AssistantName string // 本轮挂载的助手名(空串=纯骨架)

        // ---------- 原料组(易失,分析完可清空)----------
        SystemPrompt string // 本轮拼好的完整系统提示词(判「是否被助手带跑」的证据)
        AIOutput     string // 本轮 AI 完整输出
}

// InsertHarnessEvalSample 写入一条 harness 评估采集样本。
//
// id / created_at / judge_status 走表默认值(gen_random_uuid / now / 'pending'),不在此显式传。
// judge_result / admin_verdict / admin_note 采集时留空(走默认),由后续 judge / admin 回填。
//
// 返回 error 仅反映「这条是否写进了库」;是否因此阻塞业务由调用方决定
// (采集设计为 best-effort,调用方应在独立 goroutine 内调用并仅记 Warn,不阻塞主对话流)。
func InsertHarnessEvalSample(ctx context.Context, in HarnessEvalSampleInput) error {
        query := `
                INSERT INTO harness_eval_samples
                        (lesson_plan_id, stage_code, user_id, school_id,
                         model_used, is_downgraded, assistant_id, assistant_name,
                         system_prompt, ai_output)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
        `
        _, err := database.DB.Exec(ctx, query,
                in.LessonPlanID, in.StageCode, in.UserID, in.SchoolID,
                in.ModelUsed, in.IsDowngraded, in.AssistantID, in.AssistantName,
                in.SystemPrompt, in.AIOutput,
        )
        if err != nil {
                return fmt.Errorf("写入 harness 评估采集样本失败: %w", err)
        }
        return nil
}
