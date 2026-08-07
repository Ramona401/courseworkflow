package services

// courseware_add_page_discussion_prompt_test.go — 新增页讨论提示词解析测试。
//
// 本文件验证：
//   - 标准JSON和Markdown围栏JSON解析；
//   - 合法的部分plan只更新模型明确返回的字段；
//   - 真正破损的JSON进入宽容降级并保留当前方案；
//   - 空字符串指针可以明确清空旧需求。

import "testing"

func cwAddPageTestStringPointer(
        value string,
) *string {
        return &value
}

func cwAddPageTestIntPointer(
        value int,
) *int {
        return &value
}

func TestParseCWAddPageDiscussionAIResponseFromCodeFence(
        t *testing.T,
) {
        source := "```json\n" +
                `{
  "reply": "我建议把这一页做成前后概念对比页。",
  "summary": "对比两个核心概念",
  "ready_for_confirmation": true,
  "plan": {
    "title": "概念对比",
    "purpose": "帮助学生辨析差异",
    "content_summary": "左侧展示概念A，右侧展示概念B，中间总结共同点与差异。",
    "interaction_type": "click",
    "visual_format": "comparison",
    "media_requirements": "",
    "estimated_complexity": 4
  }
}` +
                "\n```"

        response, err :=
                parseCWAddPageDiscussionAIResponse(
                        source,
                        CoursewareAddPagePlan{},
                )
        if err != nil {
                t.Fatalf(
                        "围栏JSON解析失败: %v",
                        err,
                )
        }

        if response.Reply !=
                "我建议把这一页做成前后概念对比页。" {
                t.Fatalf(
                        "回复内容不符: %s",
                        response.Reply,
                )
        }

        if !response.ReadyForConfirmation {
                t.Fatal(
                        "期望ready_for_confirmation=true",
                )
        }

        if response.Plan.VisualFormat == nil ||
                *response.Plan.VisualFormat !=
                        "comparison" {
                t.Fatalf(
                        "视觉形式解析错误: %#v",
                        response.Plan.VisualFormat,
                )
        }
}

func TestParseCWAddPageDiscussionAIResponseAllowsPartialPlan(
        t *testing.T,
) {
        currentPlan := CoursewareAddPagePlan{
                Title:
                        "实验观察",
                Purpose:
                        "识别实验现象",
                ContentSummary:
                        "展示实验步骤和观察结果",
                InteractionType:
                        "static",
                VisualFormat:
                        "cards",
                MediaRequirements:
                        "实验器材示意图",
                EstimatedComplexity:
                        3,
        }

        source := `{
  "reply": "标题调整为实验现象归纳，其余要求保持不变。",
  "summary": "只调整页面标题",
  "ready_for_confirmation": true,
  "plan": {
    "title": "实验现象归纳"
  }
}`

        response, err :=
                parseCWAddPageDiscussionAIResponse(
                        source,
                        currentPlan,
                )
        if err != nil {
                t.Fatalf(
                        "部分plan解析失败: %v",
                        err,
                )
        }

        if !response.ReadyForConfirmation {
                t.Fatal(
                        "合法JSON中的确认状态应被保留",
                )
        }

        merged := normalizeCWAddPagePlan(
                mergeCWAddPagePlans(
                        currentPlan,
                        response.Plan,
                ),
        )

        if merged.Title !=
                "实验现象归纳" {
                t.Fatalf(
                        "标题没有更新: %s",
                        merged.Title,
                )
        }

        if merged.ContentSummary !=
                currentPlan.ContentSummary {
                t.Fatal(
                        "模型未返回的内容概要必须保留",
                )
        }

        if merged.MediaRequirements !=
                currentPlan.MediaRequirements {
                t.Fatal(
                        "模型未返回的媒体需求必须保留",
                )
        }
}

func TestParseCWAddPageDiscussionAIResponseKeepsPlanOnMalformedJSON(
        t *testing.T,
) {
        currentPlan := CoursewareAddPagePlan{
                Title:
                        "实验观察",
                Purpose:
                        "识别实验现象",
                ContentSummary:
                        "展示实验步骤和观察结果",
                InteractionType:
                        "static",
                VisualFormat:
                        "cards",
                MediaRequirements:
                        "实验器材示意图",
                EstimatedComplexity:
                        3,
        }

        // reply字段中故意使用未转义的英文双引号，
        // 使整个JSON真实失效，触发宽容字段提取降级。
        source := `{
  "reply": "我建议保留当前结构，但把"注意事项"放到右侧。",
  "summary": "保留原方案并强化注意事项",
  "ready_for_confirmation": true,
  "plan": {
    "title": "实验观察"
  }
}`

        response, err :=
                parseCWAddPageDiscussionAIResponse(
                        source,
                        currentPlan,
                )
        if err != nil {
                t.Fatalf(
                        "破损JSON宽容解析失败: %v",
                        err,
                )
        }

        if response.ReadyForConfirmation {
                t.Fatal(
                        "格式异常降级时不得声称方案已可确认",
                )
        }

        if response.Plan.ContentSummary == nil ||
                *response.Plan.ContentSummary !=
                        currentPlan.ContentSummary {
                t.Fatal(
                        "格式异常时必须完整保留当前页面方案",
                )
        }

        if response.Plan.MediaRequirements == nil ||
                *response.Plan.MediaRequirements !=
                        currentPlan.MediaRequirements {
                t.Fatal(
                        "格式异常时必须保留当前媒体需求",
                )
        }
}

func TestMergeCWAddPagePlansSupportsExplicitClear(
        t *testing.T,
) {
        current := CoursewareAddPagePlan{
                Title:
                        "实验观察",
                Purpose:
                        "观察实验现象",
                ContentSummary:
                        "展示实验过程",
                InteractionType:
                        "click",
                VisualFormat:
                        "cards",
                MediaRequirements:
                        "实验视频",
                EstimatedComplexity:
                        4,
        }

        next := cwAddPageDiscussionAIPlan{
                Title:
                        cwAddPageTestStringPointer(
                                "实验现象归纳",
                        ),
                MediaRequirements:
                        cwAddPageTestStringPointer(
                                "",
                        ),
                EstimatedComplexity:
                        cwAddPageTestIntPointer(
                                2,
                        ),
        }

        merged := mergeCWAddPagePlans(
                current,
                next,
        )

        if merged.Title !=
                "实验现象归纳" {
                t.Fatalf(
                        "标题合并失败: %s",
                        merged.Title,
                )
        }

        if merged.MediaRequirements != "" {
                t.Fatalf(
                        "明确清空媒体需求失败: %q",
                        merged.MediaRequirements,
                )
        }

        if merged.Purpose !=
                current.Purpose {
                t.Fatal(
                        "模型未返回的字段必须保留",
                )
        }

        if merged.EstimatedComplexity != 2 {
                t.Fatalf(
                        "复杂度合并失败: %d",
                        merged.EstimatedComplexity,
                )
        }
}
