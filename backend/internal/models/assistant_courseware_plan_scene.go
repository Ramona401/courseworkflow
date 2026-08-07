package models

// assistant_courseware_plan_scene.go
//
// 把课件教学智能体方案生成注册为可独立配置的AI场景。
// 使用独立文件和幂等init，避免继续扩大ai_config.go。

// SceneCoursewareAssistantPlan 是课件教学智能体方案生成场景。
const SceneCoursewareAssistantPlan = "courseware_assistant_plan"

func init() {
	registered := false
	for _, code := range ValidSceneCodes {
		if code == SceneCoursewareAssistantPlan {
			registered = true
			break
		}
	}

	if !registered {
		ValidSceneCodes = append(
			ValidSceneCodes,
			SceneCoursewareAssistantPlan,
		)
	}

	SceneNameMap[SceneCoursewareAssistantPlan] =
		"课件教学智能体方案"
	SceneGroupMap[SceneCoursewareAssistantPlan] =
		"courseware"
}
