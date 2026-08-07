package services

// lesson_plan_outline_guard.go
// 历史文件名保留的正式Harness共享辅助。
//
// 旧课程大纲专用阻塞Guard已经退出运行链。
// 本文件只保留多证据Harness仍在使用的无业务判定辅助：
//   - 平台质量调用TraceContext；
//   - 合格正文的缓冲SSE播放；
//   - Judge字符串列表规范化与截断。
//
// 这里不读取课程大纲，不注入课程大纲全文，也不执行课程大纲专用Judge。

import (
        "strings"

        aiClient "tedna/internal/ai"
        "tedna/internal/models"
)

const bufferedOutlineReplyChunkRunes = 240

// buildOutlineGuardTraceContext 构造平台质量调用的追踪上下文。
//
// 历史函数名为兼容现有多证据Harness调用保留；
// 故意不复制UserID，避免Judge和局部修复重复消耗教师个人积分。
func buildOutlineGuardTraceContext(
        source *aiClient.TraceContext,
) *aiClient.TraceContext {
        trace := &aiClient.TraceContext{
                SceneCode: models.SceneLessonPlanHarness,
        }

        if source == nil {
                return trace
        }

        trace.PipelineID = source.PipelineID
        trace.LessonPlanID = source.LessonPlanID
        trace.SchoolID = source.SchoolID

        return trace
}

// broadcastBufferedCourseOutlineReply 播放已经通过正式Harness的完整正文。
//
// 历史函数名为兼容调用保留；当前不再表示课程大纲专用Guard。
func broadcastBufferedCourseOutlineReply(
        planID string,
        turnID string,
        content string,
) int {
        runes := []rune(content)
        if len(runes) == 0 {
                return 0
        }

        chunkCount := 0

        for start := 0;
                start < len(runes);
                start += bufferedOutlineReplyChunkRunes {
                end := start +
                        bufferedOutlineReplyChunkRunes
                if end > len(runes) {
                        end = len(runes)
                }

                GlobalLPSSEHub.Broadcast(
                        planID,
                        models.LPSSEEvent{
                                EventType:    models.LPSSEChunk,
                                PlanID:       planID,
                                ClientTurnID: turnID,
                                Chunk:        string(runes[start:end]),
                        },
                )
                chunkCount++
        }

        return chunkCount
}

// normalizeOutlineGuardList 规范化Judge返回的字符串列表。
//
// 历史函数名为兼容多证据Harness保留。
func normalizeOutlineGuardList(
        values []string,
) []string {
        output := make(
                []string,
                0,
                len(values),
        )

        for _, value := range values {
                value = strings.TrimSpace(value)
                if value == "" {
                        continue
                }

                exists := false
                for _, current := range output {
                        if current == value {
                                exists = true
                                break
                        }
                }

                if !exists {
                        output = append(
                                output,
                                value,
                        )
                }
        }

        return output
}

// limitOutlineGuardList 限制Judge摘要展示数量。
//
// 历史函数名为兼容多证据Harness保留。
func limitOutlineGuardList(
        values []string,
        limit int,
) []string {
        if len(values) <= limit {
                return values
        }

        return values[:limit]
}
