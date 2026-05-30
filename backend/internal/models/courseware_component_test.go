package models

// courseware_component_test.go — v139 课件组件/模板模型验证函数单元测试
//
// 测试范围：
//   - IsValidCWTemplateScope: 4级 scope 验证
//   - IsValidCWStyleCategory: 6种风格类别验证
//   - IsValidCWComponentType: 7种组件类型验证
//   - CWTemplateScopeNameMap: scope 中文名映射完整性

import (
	"testing"
)

// ==================== IsValidCWTemplateScope 测试 ====================

func TestIsValidCWTemplateScope_AllValid(t *testing.T) {
	validScopes := []string{
		CWTemplateScopeSystem,
		CWTemplateScopeSchool,
		CWTemplateScopeGroup,
		CWTemplateScopePersonal,
	}
	for _, s := range validScopes {
		if !IsValidCWTemplateScope(s) {
			t.Errorf("'%s' 应为有效 scope", s)
		}
	}
}

func TestIsValidCWTemplateScope_Invalid(t *testing.T) {
	invalidScopes := []string{"", "global", "district", "region", "public", "SYSTEM", "Personal"}
	for _, s := range invalidScopes {
		if IsValidCWTemplateScope(s) {
			t.Errorf("'%s' 不应为有效 scope", s)
		}
	}
}

// ==================== IsValidCWStyleCategory 测试 ====================

func TestIsValidCWStyleCategory_AllValid(t *testing.T) {
	validCategories := []string{
		CWStyleMinimalist, CWStylePlayful, CWStyleTech,
		CWStyleAcademic, CWStyleOrganic, CWStyleImmersive,
	}
	for _, c := range validCategories {
		if !IsValidCWStyleCategory(c) {
			t.Errorf("'%s' 应为有效风格类别", c)
		}
	}
}

func TestIsValidCWStyleCategory_Invalid(t *testing.T) {
	invalidCategories := []string{"", "modern", "classic", "MINIMALIST", "Playful", "3d"}
	for _, c := range invalidCategories {
		if IsValidCWStyleCategory(c) {
			t.Errorf("'%s' 不应为有效风格类别", c)
		}
	}
}

// ==================== IsValidCWComponentType 测试 ====================

func TestIsValidCWComponentType_AllValid(t *testing.T) {
	validTypes := []string{
		"layout", "interaction", "3d", "animation",
		"data_viz", "multimedia", "style",
	}
	for _, ct := range validTypes {
		if !IsValidCWComponentType(ct) {
			t.Errorf("'%s' 应为有效组件类型", ct)
		}
	}
}

func TestIsValidCWComponentType_Invalid(t *testing.T) {
	invalidTypes := []string{"", "template", "widget", "Layout", "3D"}
	for _, ct := range invalidTypes {
		if IsValidCWComponentType(ct) {
			t.Errorf("'%s' 不应为有效组件类型", ct)
		}
	}
}

// ==================== CWTemplateScopeNameMap 完整性 ====================

func TestCWTemplateScopeNameMap_AllScopesCovered(t *testing.T) {
	requiredScopes := []string{
		CWTemplateScopeSystem,
		CWTemplateScopeSchool,
		CWTemplateScopeGroup,
		CWTemplateScopePersonal,
	}
	for _, s := range requiredScopes {
		name, ok := CWTemplateScopeNameMap[s]
		if !ok {
			t.Errorf("CWTemplateScopeNameMap 缺少 '%s' 的中文名", s)
		}
		if name == "" {
			t.Errorf("CWTemplateScopeNameMap['%s'] 不应为空字符串", s)
		}
	}
}
