package services

// courseware_asset_confidential_json.go — 普通风格锚点响应保密
//
// SetStyleAnchorResult内部仍保存VAOCI，供后端业务、日志和测试使用。
// JSON响应只返回：
//   - asset_id：正式锚点图片；
//   - anchor_url：供前端展示的图片地址。
//
// 完整VAOCI不再通过普通设锚点接口进入浏览器。

import "encoding/json"

// MarshalJSON 隐藏SetStyleAnchorResult.VAOCI。
func (item SetStyleAnchorResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			AssetID   string `json:"asset_id"`
			AnchorURL string `json:"anchor_url"`
		}{
			AssetID:   item.AssetID,
			AnchorURL: item.AnchorURL,
		},
	)
}
