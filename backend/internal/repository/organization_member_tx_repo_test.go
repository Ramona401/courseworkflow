package repository

import (
	"strings"
	"testing"
)

// TestNormalizeSchoolMemberSourceLegacy 验证历史跨校批量来源值会被稳定归一化，
// 防止滚动部署期间旧Handler继续把31字符来源写入varchar(30)字段。
func TestNormalizeSchoolMemberSourceLegacy(t *testing.T) {
	got, err := NormalizeSchoolMemberSource("admin_multi_school_batch_create")
	if err != nil {
		t.Fatalf("历史来源值归一化不应失败: %v", err)
	}
	if got != SchoolMemberSourceAdminMultiSchoolBatch {
		t.Fatalf(
			"历史来源值归一化结果错误: got=%q want=%q",
			got,
			SchoolMemberSourceAdminMultiSchoolBatch,
		)
	}
}

// TestNormalizeSchoolMemberSourceEmpty 验证空来源保持既有兼容语义，统一回退manual。
func TestNormalizeSchoolMemberSourceEmpty(t *testing.T) {
	got, err := NormalizeSchoolMemberSource("   ")
	if err != nil {
		t.Fatalf("空来源归一化不应失败: %v", err)
	}
	if got != SchoolMemberSourceManual {
		t.Fatalf(
			"空来源回退错误: got=%q want=%q",
			got,
			SchoolMemberSourceManual,
		)
	}
}

// TestNormalizeSchoolMemberSourceRejectsTooLong 验证未知超长来源在进入数据库前明确失败，
// 禁止再次退化成逐行SQLSTATE 22001和笼统的“用户名已被占用或创建失败”。
func TestNormalizeSchoolMemberSourceRejectsTooLong(t *testing.T) {
	_, err := NormalizeSchoolMemberSource(
		strings.Repeat("x", SchoolMemberSourceMaxLength+1),
	)
	if err == nil {
		t.Fatal("超长学校成员来源标记应被拒绝")
	}
}
