package models

// ai_assistant_courseware_review_scene.go
//
// 课件相关AI分析使用两个独立场景：
//
//   courseware_review
//     正式L1/L2审核工作台使用。
//     提示词面向审核员，强调证据、风险和人工审核意见。
//
//   courseware_self_review
//     课件作者在工坊最终确认页使用。
//     提示词面向作者，强调修改动作、问题定位和再次自检。
//
// 两个场景共用同一套HTML/CSS/JavaScript静态分析引擎，
// 但助手提示词和权限不能混用。
//
// 两个场景都执行四维精准匹配：
//   - education_domain精确；
//   - subject精确；
//   - grade_range严格同义匹配；
//   - scenes必须明确包含当前场景。

const (
	// CWAIReviewLevelSelf 是作者课件自审的会话用途保留值。
	//
	// 正式审核继续使用ReviewLevelL1=1和ReviewLevelL2=2。
	CWAIReviewLevelSelf = 0

	// SceneCoursewareReview 是正式课件审核专用助手场景。
	SceneCoursewareReview = "courseware_review"

	// SceneCoursewareSelfReview 是课件作者自审专用助手场景。
	SceneCoursewareSelfReview = "courseware_self_review"
)

func init() {
	registerCoursewareAssistantScene(
		SceneCoursewareReview,
	)
	registerCoursewareAssistantScene(
		SceneCoursewareSelfReview,
	)
}

// registerCoursewareAssistantScene 幂等加入助手场景白名单。
func registerCoursewareAssistantScene(
	scene string,
) {
	for _, existing :=
		range ValidAssistantScenes {
		if existing == scene {
			return
		}
	}

	ValidAssistantScenes = append(
		ValidAssistantScenes,
		scene,
	)
}
