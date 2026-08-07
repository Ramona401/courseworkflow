package services

// courseware_assistant_context_hash.go
//
// 本文件负责课件教学智能体上下文快照的稳定序列化和哈希。
//
// 哈希规则：
//   - 使用Go结构体固定字段顺序进行JSON序列化；
//   - 页面和教案内容在进入本模块前已经确定性规范化；
//   - GeneratedAt只表示装配时间，不参与稳定内容哈希；
//   - 页面HTML哈希基于去除首尾空白后的完整HTML；
//   - 使用SHA-256小写十六进制字符串。
//
// 因此相同教学内容即使在不同时间重新装配，也会得到相同上下文哈希。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// marshalCoursewareAssistantContextSnapshot 序列化完整持久化快照。
func marshalCoursewareAssistantContextSnapshot(
	snapshot models.AssistantDeploymentContextSnapshot,
) (
	string,
	error,
) {
	encoded, err :=
		json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件教学智能体上下文快照失败: %w",
			err,
		)
	}

	return string(encoded), nil
}

// hashCoursewareAssistantContextSnapshot 计算稳定教学内容哈希。
//
// GeneratedAt在副本中置空，不改变实际持久化快照。
func hashCoursewareAssistantContextSnapshot(
	snapshot models.AssistantDeploymentContextSnapshot,
) (
	string,
	error,
) {
	canonical :=
		snapshot
	canonical.GeneratedAt =
		nil

	encoded, err :=
		json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件教学智能体稳定哈希输入失败: %w",
			err,
		)
	}

	return coursewareAssistantSHA256Bytes(
		encoded,
	), nil
}

// coursewareAssistantPageHTMLHash 计算完整页面HTML哈希。
func coursewareAssistantPageHTMLHash(
	htmlContent string,
) string {
	return coursewareAssistantSHA256String(
		strings.TrimSpace(htmlContent),
	)
}

// coursewareAssistantSHA256String 计算字符串SHA-256。
func coursewareAssistantSHA256String(
	value string,
) string {
	return coursewareAssistantSHA256Bytes(
		[]byte(value),
	)
}

// coursewareAssistantSHA256Bytes 计算字节SHA-256。
func coursewareAssistantSHA256Bytes(
	value []byte,
) string {
	sum :=
		sha256.Sum256(value)

	return hex.EncodeToString(
		sum[:],
	)
}
