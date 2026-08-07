package repository

// assistant_deployment_repo_test.go
//
// 本测试验证不连接数据库的仓储核心不变量：
//   - public_id使用URL安全密码学随机格式；
//   - 批量生成的公开编号不重复；
//   - revoked状态没有恢复路径；
//   - active和paused只允许既定状态变化；
//   - 版本号严格递增并拒绝非法当前版本；
//   - 空来源列表稳定为[]。
//
// 数据库并发串行由AppendAssistantDeploymentVersion中的
// SELECT ... FOR UPDATE和同事务current_version更新保证。

import (
	"regexp"
	"testing"

	"tedna/internal/models"
)

// TestAssistantDeploymentRepoPublicIDFormat 验证公开编号格式和随机唯一性。
func TestAssistantDeploymentRepoPublicIDFormat(
	t *testing.T,
) {
	format :=
		regexp.MustCompile(
			`^[A-Za-z0-9_-]{43}$`,
		)

	seen := make(
		map[string]bool,
		256,
	)

	for index := 0; index < 256; index++ {
		publicID, err :=
			generateAssistantDeploymentPublicID()
		if err != nil {
			t.Fatalf(
				"生成第%d个公开编号失败: %v",
				index+1,
				err,
			)
		}

		if !format.MatchString(
			publicID,
		) {
			t.Fatalf(
				"公开编号不是43位URL安全格式: %s",
				publicID,
			)
		}

		if seen[publicID] {
			t.Fatalf(
				"批量生成出现重复公开编号: %s",
				publicID,
			)
		}

		seen[publicID] = true
	}
}

// TestAssistantDeploymentRepoStatusTransitions 验证固定状态机。
func TestAssistantDeploymentRepoStatusTransitions(
	t *testing.T,
) {
	type transitionCase struct {
		current string
		target  string
		allowed bool
	}

	cases :=
		[]transitionCase{
			{
				current: models.AssistantDeploymentStatusActive,
				target:  models.AssistantDeploymentStatusPaused,
				allowed: true,
			},
			{
				current: models.AssistantDeploymentStatusActive,
				target:  models.AssistantDeploymentStatusRevoked,
				allowed: true,
			},
			{
				current: models.AssistantDeploymentStatusPaused,
				target:  models.AssistantDeploymentStatusActive,
				allowed: true,
			},
			{
				current: models.AssistantDeploymentStatusPaused,
				target:  models.AssistantDeploymentStatusRevoked,
				allowed: true,
			},
			{
				current: models.AssistantDeploymentStatusActive,
				target:  models.AssistantDeploymentStatusActive,
				allowed: false,
			},
			{
				current: models.AssistantDeploymentStatusPaused,
				target:  models.AssistantDeploymentStatusPaused,
				allowed: false,
			},
			{
				current: models.AssistantDeploymentStatusRevoked,
				target:  models.AssistantDeploymentStatusActive,
				allowed: false,
			},
			{
				current: models.AssistantDeploymentStatusRevoked,
				target:  models.AssistantDeploymentStatusPaused,
				allowed: false,
			},
			{
				current: models.AssistantDeploymentStatusRevoked,
				target:  models.AssistantDeploymentStatusRevoked,
				allowed: false,
			},
		}

	for _, item := range cases {
		actual :=
			assistantDeploymentStatusTransitionAllowed(
				item.current,
				item.target,
			)

		if actual != item.allowed {
			t.Fatalf(
				"状态变化结果错误: current=%s target=%s expected=%t actual=%t",
				item.current,
				item.target,
				item.allowed,
				actual,
			)
		}
	}
}

// TestAssistantDeploymentRepoNextVersion 验证版本号严格递增。
func TestAssistantDeploymentRepoNextVersion(
	t *testing.T,
) {
	first, err :=
		assistantDeploymentNextVersion(0)
	if err != nil ||
		first != 1 {
		t.Fatalf(
			"首个版本号错误: version=%d error=%v",
			first,
			err,
		)
	}

	next, err :=
		assistantDeploymentNextVersion(7)
	if err != nil ||
		next != 8 {
		t.Fatalf(
			"追加版本号错误: version=%d error=%v",
			next,
			err,
		)
	}

	if _, err :=
		assistantDeploymentNextVersion(-1); err == nil {
		t.Fatal(
			"负数current_version应当被拒绝",
		)
	}

	if _, err :=
		assistantDeploymentNextVersion(
			assistantDeploymentMaxVersion,
		); err == nil {
		t.Fatal(
			"PostgreSQL INTEGER上限current_version应当被拒绝",
		)
	}
}

// TestAssistantDeploymentRepoAllowedOriginsDefault 验证空来源列表归一化。
func TestAssistantDeploymentRepoAllowedOriginsDefault(
	t *testing.T,
) {
	if actual :=
		normalizeAssistantDeploymentAllowedOriginsJSON(
			"",
		); actual != "[]" {
		t.Fatalf(
			"空来源列表默认值错误: %s",
			actual,
		)
	}

	if actual :=
		normalizeAssistantDeploymentAllowedOriginsJSON(
			`  ["https://course.example"]  `,
		); actual != `["https://course.example"]` {
		t.Fatalf(
			"来源列表规范化错误: %s",
			actual,
		)
	}
}
