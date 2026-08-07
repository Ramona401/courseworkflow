package services

// courseware_export_assistant.go — 离线导出中的历史助手痕迹清理
//
// 当前产品规则：
//   - 离线ZIP完全不支持教学智能体；
//   - 导出过程不查询部署，不映射page_id，不写public_id；
//   - 不注入悬浮入口、embed脚本、运行接口地址或联网提示按钮；
//   - 教学智能体只存在于TE-DNA平台登录态预览，以及独立HTTPS在线发布链路。
//
// 本文件保留的唯一职责是防御性清理历史导出HTML：
//   早期版本会在页面中写入TEDNA-ASSISTANT-EXPORT标记块。正常情况下该片段
//   只存在于旧ZIP，不会保存回数据库；但老师可能重新导入或复制旧HTML，因此新导出
//   必须先清除完整标记块，并在最终写入ZIP前检查是否仍有残留。
//
// 安全策略：
//   - 完整BEGIN/END块可以确定性删除；
//   - 不完整标记、孤立DOM编号、data-public-id或/embed/assistant/地址不会猜测修复，
//     而是由containsCoursewareExportAssistantArtifacts报告，编排服务随即停止导出；
//   - 宁可明确拒绝一个异常页面，也不生成携带在线助手标识的离线ZIP。

import (
	"regexp"
	"strings"
)

const cwExportAssistantMarker = "TEDNA-ASSISTANT-EXPORT"

var cwExportAssistantMarkedBlockPattern = regexp.MustCompile(
	`(?is)\s*<!--\s*TEDNA-ASSISTANT-EXPORT:BEGIN\s*-->.*?<!--\s*TEDNA-ASSISTANT-EXPORT:END\s*-->\s*`,
)

var cwExportAssistantResidualPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)TEDNA-ASSISTANT-EXPORT`),
	regexp.MustCompile(`(?i)tedna-assistant-export`),
	regexp.MustCompile(`(?i)data-public-id\s*=`),
	regexp.MustCompile(`(?i)data-base-url\s*=`),
	regexp.MustCompile(`(?i)/embed/assistant/`),
	regexp.MustCompile(`(?i)/api/v1/assistant-runtime/`),
	regexp.MustCompile(`(?i)runtime[_-]?token\s*[:=]`),
}

// stripCoursewareExportAssistantArtifacts 删除完整的历史助手注入块。
//
// 没有完整BEGIN/END边界时保持原文，由残留检查函数识别并阻断导出，
// 避免正则在异常HTML中误删正常课件正文或结束标签。
func stripCoursewareExportAssistantArtifacts(document string) string {
	if strings.TrimSpace(document) == "" {
		return document
	}

	return cwExportAssistantMarkedBlockPattern.ReplaceAllString(
		document,
		"",
	)
}

// containsCoursewareExportAssistantArtifacts 检查页面是否仍包含在线助手痕迹。
//
// 此函数用于清理后和页面包装后的双重检查，覆盖旧稳定标记、旧DOM命名空间、
// 公开编号和运行站点数据属性、正式embed路径、运行API路径及短时令牌字段。
func containsCoursewareExportAssistantArtifacts(document string) bool {
	for _, pattern := range cwExportAssistantResidualPatterns {
		if pattern.MatchString(document) {
			return true
		}
	}

	return false
}
