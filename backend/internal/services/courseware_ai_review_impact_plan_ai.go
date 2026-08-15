package services

// courseware_ai_review_impact_plan_ai.go
//
// R-07影响方案AI候选生成器。
//
// 关键安全边界：
//   1. 输入只来自服务端重读的可信assistant消息和当前治理快照；
//   2. AI只能返回operation_type、教师可读summary和payload；
//   3. AI不能生成operation_id；
//   4. AI不能生成preconditions；
//   5. AI输出仍必须经过freeze层逐类型校验并绑定可信业务事实；
//   6. 本文件不执行任何业务修改。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

type cwAIReviewImpactPlanAIOperation struct {
	OperationType string          `json:"operation_type"`
	Summary       string          `json:"summary"`
	Payload       json.RawMessage `json:"payload"`
}

type cwAIReviewImpactPlanAIResponse struct {
	Summary    string                            `json:"summary"`
	Operations []cwAIReviewImpactPlanAIOperation `json:"operations"`
}

func (s *CoursewareAIReviewRunner) generateCWAIReviewImpactPlan(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	pageDigests []models.CWAIReviewPageDigest,
	items []*models.CoursewareReviewItem,
	sourceMessage *models.CoursewareAIReviewMessage,
	sourceMeta *cwAIReviewGlobalDiscussionMeta,
	groups []*repository.CoursewareReviewImpactGroupSnapshot,
	relations []*repository.CoursewareReviewImpactRelationSnapshot,
	userID string,
) (*cwAIReviewImpactPlanAIResponse, *ai.CallResult, error) {
	if s == nil || s.cfg == nil {
		return nil, nil, errors.New(
			"课件AI审核影响方案模型配置未初始化",
		)
	}

	if session == nil ||
		courseware == nil ||
		sourceMessage == nil ||
		sourceMeta == nil {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	contextJSON, err := json.MarshalIndent(
		map[string]interface{}{
			"trusted_global_message": map[string]interface{}{
				"id":                sourceMessage.ID,
				"content":           sourceMessage.Content,
				"summary":           sourceMeta.Summary,
				"relations":         sourceMeta.Relations,
				"proposals":         sourceMeta.Proposals,
				"selected_item_ids": sourceMeta.SelectedItemIDs,
			},
			"selected_items":       items,
			"current_groups":       groups,
			"current_relations":    relations,
			"current_page_digests": pageDigests,
		},
		"",
		"  ",
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"序列化课件审核影响方案上下文失败: %w",
			err,
		)
	}

	systemPrompt := `你是“课件审核影响方案规划器”。

你的任务不是执行任何修改，而是把已经可信保存的全局讨论结论整理成“教师确认前的候选影响方案”。

【绝对安全规则】
1. 不得声称已经修改页面、整改项、问题组、关系或审核结果。
2. 不得自动确认修改要求。
3. 不得自动提交人工审核决定。
4. 不得生成operation_id，operation_id由服务端生成。
5. 不得生成preconditions，前置条件由服务端从当前数据库状态冻结。
6. 只能引用输入中实际存在的item_id、group_id、member_id、relation_id和page_id。
7. 对已有整改项的操作只能涉及trusted_global_message.selected_item_ids中的整改项。
8. update_candidate_suggestion中的candidate_instruction必须逐字使用可信全局消息proposals里的suggested_instruction，不能自行改写。
9. dismiss_item只能用于可信proposal明确为consider_dismiss的整改项。
10. create_relation只能使用可信全局消息relations中已有的关系类型、方向和说明，不能发明新的pairwise relation。
11. create_group、move_group_member、merge_groups、split_group只是组织问题，不得改变整改项本身的severity、页面、确认指令或整改状态。
12. merge_groups会移动来源组全部active成员，因此只有确有必要时才提出。
13. create_item表示建议新增一条独立问题，不能借此替换或覆盖现有问题。
14. cancel_relation只能取消当前已经存在且active的关系。
15. candidate_instruction只作为候选建议，永远不等于已确认修改要求。
16. 每个operation必须是教师可以独立取消勾选的最小业务动作。
17. 不输出隐藏思维链。

operation_type只允许：
- create_group
- move_group_member
- merge_groups
- split_group
- create_relation
- cancel_relation
- create_item
- dismiss_item
- update_candidate_suggestion

各payload严格格式如下：

create_group:
{
  "name": "教学主题或改进目标",
  "item_ids": ["整改项ID"],
  "primary_item_id": "可为空的主问题ID",
  "reason": "原因"
}

move_group_member:
{
  "member_id": "稳定成员ID",
  "target_group_id": "目标组ID",
  "reason": "原因"
}

merge_groups:
{
  "source_group_id": "被合并组ID",
  "target_group_id": "保留组ID",
  "reason": "原因"
}

split_group:
{
  "source_group_id": "来源组ID",
  "name": "新组名称",
  "item_ids": ["需要移入新组的整改项ID"],
  "primary_item_id": "可为空的新组主问题ID",
  "reason": "原因"
}

create_relation:
{
  "relation_type": "duplicate|conflict|merge|dependency|possibly_resolved",
  "source_item_id": "源整改项ID",
  "target_item_id": "目标整改项ID",
  "explanation": "必须逐字使用可信消息中的关系说明"
}

cancel_relation:
{
  "relation_id": "当前active关系ID",
  "reason": "取消原因"
}

create_item:
{
  "page_id": "页级问题填写page_id，整课问题为空字符串",
  "severity": "critical|high|medium|low",
  "dimension": "审核维度",
  "title": "独立问题标题",
  "description": "问题描述",
  "candidate_instruction": "候选修改建议，可为空"
}

dismiss_item:
{
  "item_id": "整改项ID",
  "reason": "暂不处理原因"
}

update_candidate_suggestion:
{
  "item_id": "整改项ID",
  "candidate_instruction": "必须逐字使用可信proposal的suggested_instruction"
}

只输出JSON对象，不要代码围栏：
{
  "summary": "教师看到的整份方案摘要",
  "operations": [
    {
      "operation_type": "update_candidate_suggestion",
      "summary": "将更新建议：……",
      "payload": {}
    }
  ]
}`

	userPrompt := fmt.Sprintf(
		`## 使用场景
这是课件AI审核全局讨论后的“教师确认前影响方案”。

## 课件
标题：%s
学科：%s
学习层级：%s
审核级别：%d

## 服务端可信上下文
%s

请仅根据上述可信上下文生成可取消勾选的候选操作。
若一项动作无法用现有ID和可信消息支撑，则不要生成该操作。`,
		courseware.Title,
		courseware.Subject,
		courseware.Grade,
		session.ReviewLevel,
		string(contextJSON),
	)

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_ai_review",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"获取课件审核影响方案模型配置失败: %w",
			err,
		)
	}

	sceneCode := models.SceneCoursewareReview
	if session.ReviewLevel == models.CWAIReviewLevelSelf {
		sceneCode = models.SceneCoursewareSelfReview
	}

	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		userID,
	)

	traceContext := &ai.TraceContext{
		SceneCode:    sceneCode,
		UserID:       &userID,
		SchoolID:     schoolIDPtr(schoolID),
		LessonPlanID: session.LessonPlanID,
	}

	callResult, err := ai.CallAI(
		aiConfig,
		systemPrompt,
		userPrompt,
		traceContext,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"生成课件审核影响方案失败: %w",
			err,
		)
	}

	response, err := parseCWAIReviewImpactPlanAIResponse(
		callResult.Content,
	)
	if err != nil {
		return nil, nil, err
	}

	return response, callResult, nil
}

func parseCWAIReviewImpactPlanAIResponse(
	content string,
) (*cwAIReviewImpactPlanAIResponse, error) {
	jsonText, ok := ai.ExtractJSON(content)
	if !ok || strings.TrimSpace(jsonText) == "" {
		jsonText = strings.TrimSpace(content)
	}

	decoder := json.NewDecoder(
		bytes.NewReader([]byte(jsonText)),
	)
	decoder.DisallowUnknownFields()

	var response cwAIReviewImpactPlanAIResponse

	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf(
			"解析课件审核影响方案结果失败: %w",
			err,
		)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	response.Summary = strings.TrimSpace(
		response.Summary,
	)

	if response.Summary == "" ||
		len(response.Operations) == 0 ||
		len(response.Operations) > cwAIReviewImpactPlanMaxOperations {
		return nil, ErrCWAIReviewImpactPlanNoOperations
	}

	for index := range response.Operations {
		operation := &response.Operations[index]

		operation.OperationType = strings.TrimSpace(
			operation.OperationType,
		)
		operation.Summary = strings.TrimSpace(
			operation.Summary,
		)

		if !models.IsCWReviewImpactOperationType(
			operation.OperationType,
		) ||
			operation.Summary == "" ||
			len(operation.Payload) == 0 {
			return nil, ErrCWAIReviewImpactPlanInvalid
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(
			operation.Payload,
			&payload,
		); err != nil || payload == nil {
			return nil, ErrCWAIReviewImpactPlanInvalid
		}
	}

	return &response, nil
}
