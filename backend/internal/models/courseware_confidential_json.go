package models

// courseware_confidential_json.go — 课件图片索引JSON保密出口
//
// 设计目标：
//   - IAOCI及课程锚点索引只保存在后端和数据库；
//   - 前端只获得自然语言风格摘要、是否已经形成风格、图片和状态；
//   - 即使调用者直接序列化领域模型，也不能把内部索引带入JSON；
//   - 不改变数据库扫描、内部服务调用、图片生成和确认事务。
//
// 当前保护字段：
//   - CoursewareStyleSession.style_aoci_text；
//   - CoursewareStyleMessage.style_aoci_text；
//   - CoursewareStylePreview.generation_prompt；
//   - Courseware.style_anchor_vaoci；
//   - CoursewareDetailResponse.style_anchor_vaoci。
//
// style_ready只表示后台是否已经形成可用风格规则，
// 不包含索引正文，也不能反推出内部编码。

import (
	"encoding/json"
	"strings"
)

// filterConfidentialJSON 从已经使用无方法别名序列化的JSON中删除保密字段，
// 并按需加入安全的派生状态。
func filterConfidentialJSON(
	raw []byte,
	removeFields []string,
	addFields map[string]json.RawMessage,
) ([]byte, error) {
	payload := make(
		map[string]json.RawMessage,
	)

	if err := json.Unmarshal(
		raw,
		&payload,
	); err != nil {
		return nil, err
	}

	for _, field := range removeFields {
		delete(payload, field)
	}

	for field, value := range addFields {
		payload[field] = value
	}

	return json.Marshal(payload)
}

// MarshalJSON 序列化风格会话时隐藏完整IAOCI，
// 仅返回style_ready供界面判断能否生成验证图。
func (item CoursewareStyleSession) MarshalJSON() ([]byte, error) {
	type safeAlias CoursewareStyleSession

	raw, err := json.Marshal(
		safeAlias(item),
	)
	if err != nil {
		return nil, err
	}

	styleReady :=
		strings.TrimSpace(
			item.StyleAOCIText,
		) != ""

	readyJSON := json.RawMessage("false")
	if styleReady {
		readyJSON =
			json.RawMessage("true")
	}

	return filterConfidentialJSON(
		raw,
		[]string{
			"style_aoci_text",
		},
		map[string]json.RawMessage{
			"style_ready": readyJSON,
		},
	)
}

// MarshalJSON 序列化聊天消息时隐藏每轮IAOCI快照。
func (item CoursewareStyleMessage) MarshalJSON() ([]byte, error) {
	type safeAlias CoursewareStyleMessage

	raw, err := json.Marshal(
		safeAlias(item),
	)
	if err != nil {
		return nil, err
	}

	return filterConfidentialJSON(
		raw,
		[]string{
			"style_aoci_text",
		},
		nil,
	)
}

// MarshalJSON 序列化验证图时隐藏后台实际生成提示词。
//
// 前端只需要预览类型、状态和asset_id。
func (item CoursewareStylePreview) MarshalJSON() ([]byte, error) {
	type safeAlias CoursewareStylePreview

	raw, err := json.Marshal(
		safeAlias(item),
	)
	if err != nil {
		return nil, err
	}

	return filterConfidentialJSON(
		raw,
		[]string{
			"generation_prompt",
		},
		nil,
	)
}

// MarshalJSON 序列化课件内部模型时隐藏课程锚点索引正文。
func (item Courseware) MarshalJSON() ([]byte, error) {
	type safeAlias Courseware

	raw, err := json.Marshal(
		safeAlias(item),
	)
	if err != nil {
		return nil, err
	}

	return filterConfidentialJSON(
		raw,
		[]string{
			"style_anchor_vaoci",
		},
		nil,
	)
}

// MarshalJSON 序列化课件详情响应时隐藏课程锚点索引正文。
//
// 前端仍可读取style_anchor_asset_id和style_anchor_url，
// 从而判断是否已有统一画风并显示缩略图。
func (item CoursewareDetailResponse) MarshalJSON() ([]byte, error) {
	type safeAlias CoursewareDetailResponse

	raw, err := json.Marshal(
		safeAlias(item),
	)
	if err != nil {
		return nil, err
	}

	return filterConfidentialJSON(
		raw,
		[]string{
			"style_anchor_vaoci",
		},
		nil,
	)
}
