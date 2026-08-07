package models

// ai_config_scene_alignment.go — 课件教案对齐场景注册
//
// courseware_alignment 已存在于 ai_scene_configs 数据库中，
// 但历史代码没有把它登记进 ValidSceneCodes，导致管理后台读取正常、
// 保存时却被 AIConfigService.UpdateSceneConfig 判定为无效场景。
//
// 本文件采用与其它扩展场景一致的幂等 init 注册方式：
//   - 不改动 ai_config.go 的稳定基础字典；
//   - 重复注册不会产生重复场景；
//   - 补齐中文名和课件分组；
//   - 后续可在统一 AI 管理中心正常修改模型、温度、Token和Fallback。

const (
	// SceneCWAlignment 对比课件方案与来源教案教学意图。
	SceneCWAlignment = "courseware_alignment"
)

func init() {
	if !IsValidSceneCode(SceneCWAlignment) {
		ValidSceneCodes = append(
			ValidSceneCodes,
			SceneCWAlignment,
		)
	}

	if _, exists := SceneNameMap[SceneCWAlignment]; !exists {
		SceneNameMap[SceneCWAlignment] =
			"课件与教案对齐分析"
	}

	if _, exists := SceneGroupMap[SceneCWAlignment]; !exists {
		SceneGroupMap[SceneCWAlignment] =
			"courseware"
	}
}
