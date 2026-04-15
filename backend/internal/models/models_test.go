// Package models 数据模型单元测试（扩展）
// 覆盖范围：ai_config / user / course / lesson_plan / ai_trace / organization / lesson_plan_component / prompt / external_data / teaching_profile
package models

import (
	"encoding/json"
	"testing"
)

// ==================== ai_config.go 测试 ====================

// TestParseFallbackModels_ValidJSON 测试有效JSON解析降级模型列表
func TestParseFallbackModels_ValidJSON(t *testing.T) {
	raw := []byte(`["model-a","model-b","model-c"]`)
	result := ParseFallbackModels(raw)
	if len(result) != 3 {
		t.Fatalf("期望3个模型, 实际%d", len(result))
	}
	if result[0] != "model-a" {
		t.Errorf("第1个模型期望model-a, 实际%s", result[0])
	}
	if result[2] != "model-c" {
		t.Errorf("第3个模型期望model-c, 实际%s", result[2])
	}
}

// TestParseFallbackModels_EmptyInput 测试空输入返回nil
func TestParseFallbackModels_EmptyInput(t *testing.T) {
	if result := ParseFallbackModels(nil); result != nil {
		t.Errorf("nil输入应返回nil, 实际%v", result)
	}
	if result := ParseFallbackModels([]byte{}); result != nil {
		t.Errorf("空切片应返回nil, 实际%v", result)
	}
}

// TestParseFallbackModels_InvalidJSON 测试无效JSON返回nil
func TestParseFallbackModels_InvalidJSON(t *testing.T) {
	if result := ParseFallbackModels([]byte("{bad json}")); result != nil {
		t.Errorf("无效JSON应返回nil, 实际%v", result)
	}
}

// TestParseFallbackModels_EmptyArray 测试空数组
func TestParseFallbackModels_EmptyArray(t *testing.T) {
	result := ParseFallbackModels([]byte("[]"))
	if result == nil {
		t.Fatal("空数组不应返回nil")
	}
	if len(result) != 0 {
		t.Errorf("空数组长度应为0, 实际%d", len(result))
	}
}

// TestIsValidSceneCode 测试场景代码校验
func TestIsValidSceneCode(t *testing.T) {
	// 全部有效场景代码
	for _, code := range ValidSceneCodes {
		if !IsValidSceneCode(code) {
			t.Errorf("场景代码 %q 应该有效", code)
		}
	}
	// 无效场景代码
	invalids := []string{"", "invalid", "SCANNER", "scanner ", " scanner", "test"}
	for _, code := range invalids {
		if IsValidSceneCode(code) {
			t.Errorf("场景代码 %q 不应该有效", code)
		}
	}
}

// TestSceneNameMap_Completeness 测试场景名称映射完整性
func TestSceneNameMap_Completeness(t *testing.T) {
	for _, code := range ValidSceneCodes {
		name, ok := SceneNameMap[code]
		if !ok || name == "" {
			t.Errorf("场景 %q 在SceneNameMap中无中文名称", code)
		}
	}
	if len(SceneNameMap) != len(ValidSceneCodes) {
		t.Errorf("SceneNameMap(%d项)与ValidSceneCodes(%d项)数量不一致", len(SceneNameMap), len(ValidSceneCodes))
	}
}

// TestSceneGroupMap_Completeness 测试场景分组映射完整性
func TestSceneGroupMap_Completeness(t *testing.T) {
	for _, code := range ValidSceneCodes {
		group, ok := SceneGroupMap[code]
		if !ok || group == "" {
			t.Errorf("场景 %q 在SceneGroupMap中无分组", code)
		}
		if group != "pipeline" && group != "lesson_plan" {
			t.Errorf("场景 %q 分组 %q 不在合法范围(pipeline/lesson_plan)", code, group)
		}
	}
}

// TestConfigKeyDescriptions_Completeness 测试全局配置键描述完整性
func TestConfigKeyDescriptions_Completeness(t *testing.T) {
	keys := []string{ConfigKeyAPIBaseURL, ConfigKeyAPIKeyEnc, ConfigKeyDefaultModel, ConfigKeyTemperature, ConfigKeyMaxTokens}
	for _, k := range keys {
		desc, ok := ConfigKeyDescriptions[k]
		if !ok || desc == "" {
			t.Errorf("配置键 %q 无描述", k)
		}
	}
}

// ==================== user.go 测试 ====================

// TestToUserInfo_Basic 测试User转UserInfo基本字段
func TestToUserInfo_Basic(t *testing.T) {
	u := &User{
		ID: "u-001", Username: "alice", DisplayName: "Alice",
		Role: RoleAdmin, Status: StatusActive, LoginCount: 5,
	}
	info := u.ToUserInfo()
	if info.ID != "u-001" {
		t.Errorf("ID期望u-001, 实际%s", info.ID)
	}
	if info.Username != "alice" {
		t.Errorf("Username期望alice, 实际%s", info.Username)
	}
	if info.Role != RoleAdmin {
		t.Errorf("Role期望admin, 实际%s", info.Role)
	}
	if info.LoginCount != 5 {
		t.Errorf("LoginCount期望5, 实际%d", info.LoginCount)
	}
	if info.HasTeachingProfile {
		t.Error("未设置TeachingProfileJSON时HasTeachingProfile应为false")
	}
}

// TestToUserInfo_WithTeachingProfile 测试有教学画像时HasTeachingProfile为true
func TestToUserInfo_WithTeachingProfile(t *testing.T) {
	profileJSON := `{"assessment_version":1}`
	u := &User{
		ID: "u-002", Username: "bob", DisplayName: "Bob",
		Role: RoleOperator, Status: StatusActive,
		TeachingProfileJSON: &profileJSON,
	}
	info := u.ToUserInfo()
	if !info.HasTeachingProfile {
		t.Error("设置TeachingProfileJSON后HasTeachingProfile应为true")
	}
}

// TestToUserInfo_PasswordHidden 测试密码哈希不泄露到UserInfo
func TestToUserInfo_PasswordHidden(t *testing.T) {
	u := &User{
		ID: "u-003", Username: "charlie", DisplayName: "Charlie",
		PasswordHash: "hashed_secret", Role: RoleViewer, Status: StatusActive,
	}
	info := u.ToUserInfo()
	// UserInfo结构体中没有PasswordHash字段，验证JSON序列化后不包含
	data, _ := json.Marshal(info)
	if str := string(data); contains(str, "hashed_secret") {
		t.Error("UserInfo JSON不应包含密码哈希")
	}
}

// TestGetTeachingProfile_Nil 测试未设置时返回nil
func TestGetTeachingProfile_Nil(t *testing.T) {
	u := &User{ID: "u-004"}
	if p := u.GetTeachingProfile(); p != nil {
		t.Error("TeachingProfileJSON为nil时GetTeachingProfile应返回nil")
	}
}

// TestGetTeachingProfile_ValidJSON 测试有效JSON解析
func TestGetTeachingProfile_ValidJSON(t *testing.T) {
	profileJSON := `{"assessment_version":1,"teaching_style":"mature","ai_collaboration":"tool","experience_years":10}`
	u := &User{ID: "u-005", TeachingProfileJSON: &profileJSON}
	p := u.GetTeachingProfile()
	if p == nil {
		t.Fatal("有效JSON不应返回nil")
	}
	if p.AssessmentVersion != 1 {
		t.Errorf("AssessmentVersion期望1, 实际%d", p.AssessmentVersion)
	}
	if p.TeachingStyle != "mature" {
		t.Errorf("TeachingStyle期望mature, 实际%s", p.TeachingStyle)
	}
	if p.ExperienceYears != 10 {
		t.Errorf("ExperienceYears期望10, 实际%d", p.ExperienceYears)
	}
}

// TestGetTeachingProfile_InvalidJSON 测试无效JSON返回nil
func TestGetTeachingProfile_InvalidJSON(t *testing.T) {
	bad := `{invalid}`
	u := &User{ID: "u-006", TeachingProfileJSON: &bad}
	if p := u.GetTeachingProfile(); p != nil {
		t.Error("无效JSON应返回nil")
	}
}

// TestIsValidRole 测试角色校验
func TestIsValidRole(t *testing.T) {
	for _, r := range ValidRoles {
		if !IsValidRole(r) {
			t.Errorf("角色 %q 应该有效", r)
		}
	}
	invalids := []string{"", "superadmin", "Admin", "root", "test"}
	for _, r := range invalids {
		if IsValidRole(r) {
			t.Errorf("角色 %q 不应该有效", r)
		}
	}
}

// TestIsValidStatus 测试用户状态校验
func TestIsValidStatus(t *testing.T) {
	if !IsValidStatus(StatusActive) {
		t.Error("active应该有效")
	}
	if !IsValidStatus(StatusDisabled) {
		t.Error("disabled应该有效")
	}
	invalids := []string{"", "deleted", "Active", "banned"}
	for _, s := range invalids {
		if IsValidStatus(s) {
			t.Errorf("状态 %q 不应该有效", s)
		}
	}
}

// TestValidRoles_Count 测试角色常量数量=4
func TestValidRoles_Count(t *testing.T) {
	if len(ValidRoles) != 4 {
		t.Errorf("ValidRoles应有4个角色, 实际%d", len(ValidRoles))
	}
}

// ==================== course.go 测试 ====================

// TestIsValidCourseStatus 测试课程状态校验
func TestIsValidCourseStatus(t *testing.T) {
	if !IsValidCourseStatus(CourseStatusActive) {
		t.Error("active应该有效")
	}
	if !IsValidCourseStatus(CourseStatusArchived) {
		t.Error("archived应该有效")
	}
	invalids := []string{"", "deleted", "Active", "pending"}
	for _, s := range invalids {
		if IsValidCourseStatus(s) {
			t.Errorf("课程状态 %q 不应该有效", s)
		}
	}
}

// TestValidStages 测试学段常量
func TestValidStages(t *testing.T) {
	expected := map[string]bool{"primary": true, "middle": true, "high": true, "": true}
	for _, s := range ValidStages {
		if !expected[s] {
			t.Errorf("学段 %q 不在期望列表中", s)
		}
	}
}

// ==================== lesson_plan.go 测试 ====================

// TestLPStatusNameMap_Completeness 测试教案状态映射完整性
func TestLPStatusNameMap_Completeness(t *testing.T) {
	statuses := []string{
		LPStatusDraft, LPStatusPublishedPersonal, LPStatusSubmitted,
		LPStatusRevision, LPStatusApproved, LPStatusPublishedShared,
		LPStatusDeveloping, LPStatusCompleted,
	}
	for _, s := range statuses {
		name, ok := LPStatusNameMap[s]
		if !ok || name == "" {
			t.Errorf("教案状态 %q 在LPStatusNameMap中无中文名称", s)
		}
	}
	if len(LPStatusNameMap) != len(statuses) {
		t.Errorf("LPStatusNameMap(%d项)与状态常量(%d项)数量不一致", len(LPStatusNameMap), len(statuses))
	}
}

// TestIsValidTemplateLevel 测试模板层级校验
func TestIsValidTemplateLevel(t *testing.T) {
	for _, l := range ValidTemplateLevels {
		if !IsValidTemplateLevel(l) {
			t.Errorf("模板层级 %q 应该有效", l)
		}
	}
	invalids := []string{"", "global", "national", "Region"}
	for _, l := range invalids {
		if IsValidTemplateLevel(l) {
			t.Errorf("模板层级 %q 不应该有效", l)
		}
	}
}

// ==================== ai_trace.go 测试 ====================

// TestEstimateCost_KnownModel 测试已知模型的成本估算
func TestEstimateCost_KnownModel(t *testing.T) {
	// Sonnet 4.6: 输入$3/M, 输出$15/M
	cost := EstimateCost("anthropic/claude-sonnet-4.6", 1000000, 1000000)
	expected := 3.0 + 15.0 // $18
	if cost != expected {
		t.Errorf("Sonnet 4.6 1M/1M 成本期望$%.2f, 实际$%.2f", expected, cost)
	}
}

// TestEstimateCost_OpusModel 测试Opus模型定价
func TestEstimateCost_OpusModel(t *testing.T) {
	// Opus 4.6: 输入$15/M, 输出$75/M
	cost := EstimateCost("anthropic/claude-opus-4.6", 100000, 50000)
	expected := 0.1*15.0 + 0.05*75.0 // $1.5 + $3.75 = $5.25
	if cost != expected {
		t.Errorf("Opus 4.6 100K/50K 成本期望$%.2f, 实际$%.2f", expected, cost)
	}
}

// TestEstimateCost_UnknownModel 测试未知模型回退到Sonnet定价
func TestEstimateCost_UnknownModel(t *testing.T) {
	costUnknown := EstimateCost("unknown-model", 1000000, 1000000)
	costSonnet := EstimateCost("anthropic/claude-sonnet-4.6", 1000000, 1000000)
	if costUnknown != costSonnet {
		t.Errorf("未知模型应回退到Sonnet定价: unknown=$%.2f, sonnet=$%.2f", costUnknown, costSonnet)
	}
}

// TestEstimateCost_ZeroTokens 测试零token成本为0
func TestEstimateCost_ZeroTokens(t *testing.T) {
	cost := EstimateCost("anthropic/claude-sonnet-4.6", 0, 0)
	if cost != 0 {
		t.Errorf("零token成本应为0, 实际$%.6f", cost)
	}
}

// TestModelPricingMap_NonEmpty 测试定价表非空
func TestModelPricingMap_NonEmpty(t *testing.T) {
	if len(ModelPricingMap) == 0 {
		t.Fatal("ModelPricingMap不应为空")
	}
	for model, pricing := range ModelPricingMap {
		if pricing.PromptPer1M <= 0 {
			t.Errorf("模型 %q 输入定价应>0, 实际%.2f", model, pricing.PromptPer1M)
		}
		if pricing.CompletionPer1M <= 0 {
			t.Errorf("模型 %q 输出定价应>0, 实际%.2f", model, pricing.CompletionPer1M)
		}
	}
}

// ==================== organization.go 测试 ====================

// TestIsValidOrgType 测试组织类型校验
func TestIsValidOrgType(t *testing.T) {
	if !IsValidOrgType(OrgTypeRegion) {
		t.Error("region应该有效")
	}
	if !IsValidOrgType(OrgTypeSchool) {
		t.Error("school应该有效")
	}
	invalids := []string{"", "company", "Region", "district"}
	for _, typ := range invalids {
		if IsValidOrgType(typ) {
			t.Errorf("组织类型 %q 不应该有效", typ)
		}
	}
}

// TestIsValidGroupMemberRole 测试教研组成员角色校验
func TestIsValidGroupMemberRole(t *testing.T) {
	if !IsValidGroupMemberRole(GroupMemberRoleMember) {
		t.Error("member应该有效")
	}
	if !IsValidGroupMemberRole(GroupMemberRoleBackbone) {
		t.Error("backbone应该有效")
	}
	invalids := []string{"", "leader", "admin", "Member"}
	for _, r := range invalids {
		if IsValidGroupMemberRole(r) {
			t.Errorf("教研组成员角色 %q 不应该有效", r)
		}
	}
}

// ==================== lesson_plan_component.go 测试 ====================

// TestIsValidLibraryType 测试组件库类型校验
func TestIsValidLibraryType(t *testing.T) {
	// 全部13种类型
	for _, lt := range ValidLibraryTypes {
		if !IsValidLibraryType(lt) {
			t.Errorf("组件库类型 %q 应该有效", lt)
		}
	}
	if len(ValidLibraryTypes) != 13 {
		t.Errorf("ValidLibraryTypes应有13种, 实际%d", len(ValidLibraryTypes))
	}
	// 无效类型
	if IsValidLibraryType("") || IsValidLibraryType("unknown") {
		t.Error("空字符串和unknown不应该有效")
	}
}

// TestLibraryTypeNameMap_Completeness 测试组件库类型名称映射完整性
func TestLibraryTypeNameMap_Completeness(t *testing.T) {
	for _, lt := range ValidLibraryTypes {
		name, ok := LibraryTypeNameMap[lt]
		if !ok || name == "" {
			t.Errorf("组件库类型 %q 在LibraryTypeNameMap中无中文名称", lt)
		}
	}
}

// TestIsValidInjectionMode 测试注入模式校验
func TestIsValidInjectionMode(t *testing.T) {
	for _, m := range ValidInjectionModes {
		if !IsValidInjectionMode(m) {
			t.Errorf("注入模式 %q 应该有效", m)
		}
	}
	if IsValidInjectionMode("") || IsValidInjectionMode("force") {
		t.Error("空字符串和force不应是有效注入模式")
	}
}

// TestIsValidScope 测试可见范围校验
func TestIsValidScope(t *testing.T) {
	for _, s := range ValidScopes {
		if !IsValidScope(s) {
			t.Errorf("范围 %q 应该有效", s)
		}
	}
	if len(ValidScopes) != 5 {
		t.Errorf("ValidScopes应有5种, 实际%d", len(ValidScopes))
	}
	if IsValidScope("") || IsValidScope("public_all") {
		t.Error("空字符串和public_all不应该有效")
	}
}

// ==================== prompt.go 测试 ====================

// TestIsValidPromptKey 测试提示词标识校验
func TestIsValidPromptKey(t *testing.T) {
	for _, k := range ValidPromptKeys {
		if !IsValidPromptKey(k) {
			t.Errorf("提示词标识 %q 应该有效", k)
		}
	}
	if IsValidPromptKey("") || IsValidPromptKey("prompt_z") || IsValidPromptKey("Prompt_A") {
		t.Error("空字符串/prompt_z/Prompt_A不应该有效")
	}
}

// TestPromptNameMap_Completeness 测试提示词名称映射完整性
func TestPromptNameMap_Completeness(t *testing.T) {
	for _, k := range ValidPromptKeys {
		name, ok := PromptNameMap[k]
		if !ok || name == "" {
			t.Errorf("提示词 %q 在PromptNameMap中无中文名称", k)
		}
	}
}

// TestPromptDescriptionMap_Completeness 测试提示词描述映射完整性
func TestPromptDescriptionMap_Completeness(t *testing.T) {
	for _, k := range ValidPromptKeys {
		desc, ok := PromptDescriptionMap[k]
		if !ok || desc == "" {
			t.Errorf("提示词 %q 在PromptDescriptionMap中无描述", k)
		}
	}
}

// ==================== external_data.go 测试 ====================

// TestIsSensitiveEDKey 测试敏感配置键判断
func TestIsSensitiveEDKey(t *testing.T) {
	if !IsSensitiveEDKey(EDKeyOSSAccessKeyEnc) {
		t.Error("oss_access_key_enc应该是敏感键")
	}
	if !IsSensitiveEDKey(EDKeyPushAPIToken) {
		t.Error("push_api_token应该是敏感键")
	}
	// 非敏感键
	nonSensitive := []string{EDKeyOSSEndpoint, EDKeyOSSBucket, EDKeyOSSAccessKeyID, EDKeyPushAPIURL, ""}
	for _, k := range nonSensitive {
		if IsSensitiveEDKey(k) {
			t.Errorf("配置键 %q 不应该是敏感键", k)
		}
	}
}

// TestEDKeyDescriptions_Completeness 测试外部数据配置描述完整性
func TestEDKeyDescriptions_Completeness(t *testing.T) {
	allKeys := append(EDKeyGroupOSS, EDKeyGroupPush...)
	for _, k := range allKeys {
		desc, ok := EDKeyDescriptions[k]
		if !ok || desc == "" {
			t.Errorf("配置键 %q 在EDKeyDescriptions中无描述", k)
		}
	}
}

// ==================== teaching_profile.go 测试 ====================

// TestStyleModeMap_Completeness 测试风格→模式映射完整性
func TestStyleModeMap_Completeness(t *testing.T) {
	styles := []string{StyleMature, StyleGrowing, StyleBeginner}
	for _, s := range styles {
		mode, ok := StyleModeMap[s]
		if !ok || mode == "" {
			t.Errorf("教学风格 %q 在StyleModeMap中无映射", s)
		}
		if mode != "guided" && mode != "efficient" {
			t.Errorf("教学风格 %q 映射模式 %q 不合法", s, mode)
		}
	}
}

// TestStyleStagesMap_Completeness 测试风格→阶段列表映射完整性
func TestStyleStagesMap_Completeness(t *testing.T) {
	styles := []string{StyleMature, StyleGrowing, StyleBeginner}
	for _, s := range styles {
		stages, ok := StyleStagesMap[s]
		if !ok || len(stages) == 0 {
			t.Errorf("教学风格 %q 在StyleStagesMap中无映射或为空", s)
		}
	}
	// 新手和成长型应该有5个完整阶段
	if len(StyleStagesMap[StyleGrowing]) != 5 {
		t.Errorf("成长型应有5个阶段, 实际%d", len(StyleStagesMap[StyleGrowing]))
	}
	if len(StyleStagesMap[StyleBeginner]) != 5 {
		t.Errorf("新手型应有5个阶段, 实际%d", len(StyleStagesMap[StyleBeginner]))
	}
	// 成熟型应少于5个
	if len(StyleStagesMap[StyleMature]) >= 5 {
		t.Errorf("成熟型阶段应少于5个, 实际%d", len(StyleStagesMap[StyleMature]))
	}
}

// TestBeginnerNotEfficient 测试新手不会被推荐为efficient模式
func TestBeginnerNotEfficient(t *testing.T) {
	mode := StyleModeMap[StyleBeginner]
	if mode == "efficient" {
		t.Error("新手型不应被推荐为efficient模式")
	}
}

// ==================== workshop_stage.go 测试 ====================

// TestSystemStageFlowRules_Completeness 测试系统阶段流程规则完整性
func TestSystemStageFlowRules_Completeness(t *testing.T) {
	expectedCodes := []string{"analyze", "design", "write", "review", "revise"}
	for _, code := range expectedCodes {
		rule, ok := SystemStageFlowRules[code]
		if !ok {
			t.Errorf("阶段 %q 在SystemStageFlowRules中不存在", code)
			continue
		}
		if rule.StageCode != code {
			t.Errorf("阶段规则StageCode不匹配: 期望%q, 实际%q", code, rule.StageCode)
		}
		if rule.StageName == "" {
			t.Errorf("阶段 %q 的StageName不应为空", code)
		}
	}
}

// TestReviseStage_MustBeLast 测试revise阶段必须在最后
func TestReviseStage_MustBeLast(t *testing.T) {
	rule := SystemStageFlowRules["revise"]
	if !rule.MustBeLast {
		t.Error("revise阶段MustBeLast应为true")
	}
	if rule.Removable {
		t.Error("revise阶段不可移除（Removable应为false）")
	}
	if rule.Reorderable {
		t.Error("revise阶段不可调序（Reorderable应为false）")
	}
}

// TestWriteStage_NotRemovable 测试write阶段不可移除
func TestWriteStage_NotRemovable(t *testing.T) {
	rule := SystemStageFlowRules["write"]
	if rule.Removable {
		t.Error("write阶段不可移除（Removable应为false）")
	}
}

// TestReviewStage_MustAfterWrite 测试review阶段必须在write之后
func TestReviewStage_MustAfterWrite(t *testing.T) {
	rule := SystemStageFlowRules["review"]
	if rule.MustAfter != "write" {
		t.Errorf("review阶段MustAfter期望write, 实际%q", rule.MustAfter)
	}
}

// ==================== role.go 测试 ====================

// TestValidResources_NonEmpty 测试权限资源枚举非空
func TestValidResources_NonEmpty(t *testing.T) {
	if len(ValidResources) == 0 {
		t.Fatal("ValidResources不应为空")
	}
	expected := []string{"pipeline", "user", "course", "ai_config", "prompt", "system", "lesson_plan"}
	for _, r := range expected {
		if !ValidResources[r] {
			t.Errorf("资源 %q 应在ValidResources中", r)
		}
	}
}

// TestValidActions_NonEmpty 测试权限动作枚举非空
func TestValidActions_NonEmpty(t *testing.T) {
	if len(ValidActions) == 0 {
		t.Fatal("ValidActions不应为空")
	}
	expected := []string{"create", "read", "update", "delete"}
	for _, a := range expected {
		if !ValidActions[a] {
			t.Errorf("动作 %q 应在ValidActions中", a)
		}
	}
}

// TestValidBaseRoles 测试基础角色枚举
func TestValidBaseRoles(t *testing.T) {
	if len(ValidBaseRoles) != 4 {
		t.Errorf("ValidBaseRoles应有4个, 实际%d", len(ValidBaseRoles))
	}
	for _, code := range []string{RoleCodeAdmin, RoleCodeSeniorOperator, RoleCodeOperator, RoleCodeViewer} {
		if !ValidBaseRoles[code] {
			t.Errorf("角色码 %q 应在ValidBaseRoles中", code)
		}
	}
}

// ==================== 辅助函数 ====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
