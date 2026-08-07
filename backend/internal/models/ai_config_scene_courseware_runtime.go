package models

// ai_config_scene_courseware_runtime.go
//
// 本文件集中登记已经进入生产数据库和正式运行链、
// 但历史基础场景字典尚未收录的课件AI运行场景。
//
// 背景：
//   ai_scene_configs读取接口会返回数据库中的全部场景，
//   但保存接口会先调用IsValidSceneCode校验场景代码。
//   若数据库已有记录而ValidSceneCodes没有登记，管理后台虽然能够展示，
//   保存时却会返回“无效的场景代码”，形成“可见但不可配置”的不一致状态。
//
// 设计原则：
//   1. 采用幂等init登记，不直接扩大ai_config.go基础文件；
//   2. 同时维护场景代码、中文名称和前端分组；
//   3. 重复登记不产生重复代码；
//   4. 场景登记只解决配置合法性，不改变AI调用权限和模型分流规则。

const (
	// SceneCWLessonNormalize 用于把完整教案规整为课件逐页生成可复用的结构化正文。
	SceneCWLessonNormalize = "courseware_lesson_normalize"

	// SceneCWMediaPrompt 用于课件图片提示词、视频分镜和图片IAOCI规划。
	SceneCWMediaPrompt = "courseware_media_prompt"
)

func init() {
	registerCoursewareRuntimeAIConfigScene(
		SceneCWLessonNormalize,
		"课件来源教案规整",
		"courseware",
	)

	registerCoursewareRuntimeAIConfigScene(
		SceneCWMediaPrompt,
		"课件媒体提示词规划",
		"courseware",
	)
}

// registerCoursewareRuntimeAIConfigScene 幂等登记课件运行场景。
//
// ValidSceneCodes是保存接口的合法性白名单；
// SceneNameMap和SceneGroupMap分别决定管理后台中文名称和业务分组。
// 已存在的名称或分组不覆盖，避免未来其它模块提供更精确的正式定义。
func registerCoursewareRuntimeAIConfigScene(
	code string,
	name string,
	group string,
) {
	if !IsValidSceneCode(code) {
		ValidSceneCodes = append(
			ValidSceneCodes,
			code,
		)
	}

	if _, exists := SceneNameMap[code]; !exists {
		SceneNameMap[code] = name
	}

	if _, exists := SceneGroupMap[code]; !exists {
		SceneGroupMap[code] = group
	}
}
