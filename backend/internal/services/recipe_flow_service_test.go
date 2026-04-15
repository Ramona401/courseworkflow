package services

// recipe_flow_service_test.go — 备课配方流程校验与预设模板单元测试
//
// 测试范围：
//   - ValidateStageFlow：9条校验规则全覆盖（5阻断+3警告+1信息）
//   - GetFlowPresets：4个预设模板正确性
//
// 测试策略：
//   ValidateStageFlow 是 RecipeService 的方法，但不使用任何内部状态，
//   因此可以用空的 RecipeService{} 作为接收者直接调用。

import (
"testing"

"tedna/internal/models"
)

// ==================== 辅助函数 ====================

// newRecipeService 创建空的RecipeService用于测试纯逻辑方法
func newRecipeService() *RecipeService {
return &RecipeService{}
}

// fullFlow 返回标准5阶段全开流程（基准配置）
func fullFlow() []models.StageFlowItem {
return []models.StageFlowItem{
{StageCode: "analyze", Enabled: true, Order: 1},
{StageCode: "design", Enabled: true, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "review", Enabled: true, Order: 4},
{StageCode: "revise", Enabled: true, Order: 5},
}
}

// minimalFlow 返回最小合法流程（仅write+revise）
func minimalFlow() []models.StageFlowItem {
return []models.StageFlowItem{
{StageCode: "analyze", Enabled: false, Order: 1},
{StageCode: "design", Enabled: false, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "review", Enabled: false, Order: 4},
{StageCode: "revise", Enabled: true, Order: 5},
}
}

// hasMessageCode 检查校验结果中是否包含指定code的消息
func hasMessageCode(result *models.FlowValidationResult, code string) bool {
for _, msg := range result.Messages {
if msg.Code == code {
return true
}
}
return false
}

// hasMessageLevel 检查校验结果中是否包含指定级别的消息
func hasMessageLevel(result *models.FlowValidationResult, level string) bool {
for _, msg := range result.Messages {
if msg.Level == level {
return true
}
}
return false
}

// countMessagesByLevel 统计指定级别的消息数量
func countMessagesByLevel(result *models.FlowValidationResult, level string) int {
count := 0
for _, msg := range result.Messages {
if msg.Level == level {
count++
}
}
return count
}

// ==================== ValidateStageFlow 测试 ====================

// TestValidateStageFlow_FullFlow 测试标准5阶段全开——应通过且无错误
func TestValidateStageFlow_FullFlow(t *testing.T) {
svc := newRecipeService()
result := svc.ValidateStageFlow(fullFlow())

if !result.Valid {
t.Errorf("完整流程应通过校验，但Valid=false，消息: %+v", result.Messages)
}
if hasMessageLevel(result, models.FlowMsgError) {
t.Errorf("完整流程不应有error级消息，但存在: %+v", result.Messages)
}
}

// TestValidateStageFlow_Rule1_EmptyFlow 规则1：流程不能为空（全部禁用）
func TestValidateStageFlow_Rule1_EmptyFlow(t *testing.T) {
svc := newRecipeService()
// 场景1：空切片
result := svc.ValidateStageFlow([]models.StageFlowItem{})
if result.Valid {
t.Error("空流程应阻断，但Valid=true")
}
if !hasMessageCode(result, "flow_empty") {
t.Error("空流程应包含flow_empty消息")
}

// 场景2：全部disabled
allDisabled := []models.StageFlowItem{
{StageCode: "write", Enabled: false, Order: 1},
{StageCode: "revise", Enabled: false, Order: 2},
}
result2 := svc.ValidateStageFlow(allDisabled)
if result2.Valid {
t.Error("全部禁用应阻断，但Valid=true")
}
if !hasMessageCode(result2, "flow_empty") {
t.Error("全部禁用应包含flow_empty消息")
}
}

// TestValidateStageFlow_Rule2_RequiredStageMissing 规则2：不可移除阶段缺失
func TestValidateStageFlow_Rule2_RequiredStageMissing(t *testing.T) {
svc := newRecipeService()

// 场景1：缺少write（write不可移除）
noWrite := []models.StageFlowItem{
{StageCode: "analyze", Enabled: true, Order: 1},
{StageCode: "revise", Enabled: true, Order: 2},
}
result := svc.ValidateStageFlow(noWrite)
if result.Valid {
t.Error("缺少write应阻断")
}
if !hasMessageCode(result, "required_stage_missing") {
t.Error("缺少write应包含required_stage_missing消息")
}

// 场景2：缺少revise（revise不可移除）
noRevise := []models.StageFlowItem{
{StageCode: "write", Enabled: true, Order: 1},
{StageCode: "review", Enabled: true, Order: 2},
}
result2 := svc.ValidateStageFlow(noRevise)
if result2.Valid {
t.Error("缺少revise应阻断")
}
if !hasMessageCode(result2, "required_stage_missing") {
t.Error("缺少revise应包含required_stage_missing消息")
}

// 场景3：同时缺少write和revise
noBoth := []models.StageFlowItem{
{StageCode: "analyze", Enabled: true, Order: 1},
{StageCode: "design", Enabled: true, Order: 2},
}
result3 := svc.ValidateStageFlow(noBoth)
if result3.Valid {
t.Error("缺少write+revise应阻断")
}
// 应该有两条required_stage_missing
count := 0
for _, msg := range result3.Messages {
if msg.Code == "required_stage_missing" {
count++
}
}
if count < 2 {
t.Errorf("缺少write+revise应至少有2条required_stage_missing消息，实际%d条", count)
}
}

// TestValidateStageFlow_Rule3_ReviseNotLast 规则3：修订定稿必须在最后
func TestValidateStageFlow_Rule3_ReviseNotLast(t *testing.T) {
svc := newRecipeService()
// revise在write之前
stages := []models.StageFlowItem{
{StageCode: "revise", Enabled: true, Order: 1},
{StageCode: "write", Enabled: true, Order: 2},
}
result := svc.ValidateStageFlow(stages)
if result.Valid {
t.Error("revise不在最后应阻断")
}
if !hasMessageCode(result, "revise_not_last") {
t.Error("应包含revise_not_last消息")
}
}

// TestValidateStageFlow_Rule4_ReviewBeforeWrite 规则4：review必须在write之后
func TestValidateStageFlow_Rule4_ReviewBeforeWrite(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "review", Enabled: true, Order: 1},
{StageCode: "write", Enabled: true, Order: 2},
{StageCode: "revise", Enabled: true, Order: 3},
}
result := svc.ValidateStageFlow(stages)
if result.Valid {
t.Error("review在write之前应阻断")
}
if !hasMessageCode(result, "review_before_write") {
t.Error("应包含review_before_write消息")
}
}

// TestValidateStageFlow_Rule4_ReviewAfterWrite_OK 规则4反向：review在write之后应通过
func TestValidateStageFlow_Rule4_ReviewAfterWrite_OK(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "write", Enabled: true, Order: 1},
{StageCode: "review", Enabled: true, Order: 2},
{StageCode: "revise", Enabled: true, Order: 3},
}
result := svc.ValidateStageFlow(stages)
if !result.Valid {
t.Errorf("review在write之后应通过，消息: %+v", result.Messages)
}
if hasMessageCode(result, "review_before_write") {
t.Error("不应包含review_before_write消息")
}
}

// TestValidateStageFlow_Rule5_Duplicate 规则5：阶段不能重复
func TestValidateStageFlow_Rule5_Duplicate(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "write", Enabled: true, Order: 1},
{StageCode: "write", Enabled: true, Order: 2},
{StageCode: "revise", Enabled: true, Order: 3},
}
result := svc.ValidateStageFlow(stages)
if result.Valid {
t.Error("重复阶段应阻断")
}
if !hasMessageCode(result, "stage_duplicate") {
t.Error("应包含stage_duplicate消息")
}
}

// TestValidateStageFlow_Rule6_SkipAnalyze 规则6：跳过分析阶段产生警告
func TestValidateStageFlow_Rule6_SkipAnalyze(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "analyze", Enabled: false, Order: 1},
{StageCode: "design", Enabled: true, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "review", Enabled: true, Order: 4},
{StageCode: "revise", Enabled: true, Order: 5},
}
result := svc.ValidateStageFlow(stages)
// 跳过analyze只是warning，不阻断
if !result.Valid {
t.Error("跳过analyze不应阻断")
}
if !hasMessageCode(result, "skip_analyze") {
t.Error("应包含skip_analyze警告")
}
}

// TestValidateStageFlow_Rule7_SkipDesign 规则7：跳过设计阶段产生警告
func TestValidateStageFlow_Rule7_SkipDesign(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "analyze", Enabled: true, Order: 1},
{StageCode: "design", Enabled: false, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "review", Enabled: true, Order: 4},
{StageCode: "revise", Enabled: true, Order: 5},
}
result := svc.ValidateStageFlow(stages)
if !result.Valid {
t.Error("跳过design不应阻断")
}
if !hasMessageCode(result, "skip_design") {
t.Error("应包含skip_design警告")
}
}

// TestValidateStageFlow_Rule8_SkipReview 规则8：跳过评审阶段产生警告
func TestValidateStageFlow_Rule8_SkipReview(t *testing.T) {
svc := newRecipeService()
stages := []models.StageFlowItem{
{StageCode: "analyze", Enabled: true, Order: 1},
{StageCode: "design", Enabled: true, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "review", Enabled: false, Order: 4},
{StageCode: "revise", Enabled: true, Order: 5},
}
result := svc.ValidateStageFlow(stages)
if !result.Valid {
t.Error("跳过review不应阻断")
}
if !hasMessageCode(result, "skip_review") {
t.Error("应包含skip_review警告")
}
}

// TestValidateStageFlow_Rule9_MinimalFlow 规则9：极简模式（≤2阶段）信息提示
func TestValidateStageFlow_Rule9_MinimalFlow(t *testing.T) {
svc := newRecipeService()
result := svc.ValidateStageFlow(minimalFlow())
// 极简流程（write+revise=2阶段）应通过但有info提示
if !result.Valid {
t.Errorf("最小合法流程应通过，消息: %+v", result.Messages)
}
if !hasMessageCode(result, "minimal_flow") {
t.Error("应包含minimal_flow信息提示")
}
// 检查info级别
found := false
for _, msg := range result.Messages {
if msg.Code == "minimal_flow" && msg.Level == models.FlowMsgInfo {
found = true
break
}
}
if !found {
t.Error("minimal_flow消息应为info级别")
}
}

// TestValidateStageFlow_MinimalFlowWarnings 最小流程同时触发多条警告
func TestValidateStageFlow_MinimalFlowWarnings(t *testing.T) {
svc := newRecipeService()
result := svc.ValidateStageFlow(minimalFlow())
// 跳过了analyze+design+review，应有3条warning
warningCount := countMessagesByLevel(result, models.FlowMsgWarning)
if warningCount != 3 {
t.Errorf("最小流程应有3条warning（skip_analyze+skip_design+skip_review），实际%d条", warningCount)
}
}

// TestValidateStageFlow_MultipleErrors 多条阻断错误同时触发
func TestValidateStageFlow_MultipleErrors(t *testing.T) {
svc := newRecipeService()
// review在write之前 + revise不在最后 + write重复
stages := []models.StageFlowItem{
{StageCode: "review", Enabled: true, Order: 1},
{StageCode: "revise", Enabled: true, Order: 2},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "write", Enabled: true, Order: 4},
}
result := svc.ValidateStageFlow(stages)
if result.Valid {
t.Error("多条错误应阻断")
}
errorCount := countMessagesByLevel(result, models.FlowMsgError)
if errorCount < 2 {
t.Errorf("应至少有2条error消息，实际%d条，消息: %+v", errorCount, result.Messages)
}
}

// TestValidateStageFlow_DisabledStagesIgnored 禁用阶段不参与校验
func TestValidateStageFlow_DisabledStagesIgnored(t *testing.T) {
svc := newRecipeService()
// review禁用，不应触发review_before_write
stages := []models.StageFlowItem{
{StageCode: "review", Enabled: false, Order: 1},
{StageCode: "write", Enabled: true, Order: 3},
{StageCode: "revise", Enabled: true, Order: 5},
}
result := svc.ValidateStageFlow(stages)
if hasMessageCode(result, "review_before_write") {
t.Error("禁用的review不应触发review_before_write")
}
}

// ==================== GetFlowPresets 测试 ====================

// TestGetFlowPresets_ReturnsFourPresets 预设模板数量为4
func TestGetFlowPresets_ReturnsFourPresets(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
if len(presets) != 4 {
t.Fatalf("预设模板应有4个，实际%d个", len(presets))
}
}

// TestGetFlowPresets_UniqueKeys 预设模板Key不重复
func TestGetFlowPresets_UniqueKeys(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
keySet := make(map[string]bool)
for _, p := range presets {
if keySet[p.Key] {
t.Errorf("预设模板Key重复: %s", p.Key)
}
keySet[p.Key] = true
}
}

// TestGetFlowPresets_AllPresetsValid 所有预设模板通过流程校验
func TestGetFlowPresets_AllPresetsValid(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
for _, p := range presets {
result := svc.ValidateStageFlow(p.Stages)
if !result.Valid {
t.Errorf("预设模板 %q 未通过流程校验，消息: %+v", p.Key, result.Messages)
}
}
}

// TestGetFlowPresets_FullGuidedHasAllEnabled 完整引导模板应5阶段全开
func TestGetFlowPresets_FullGuidedHasAllEnabled(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
var fullGuided *models.FlowPreset
for _, p := range presets {
if p.Key == "full_guided" {
fullGuided = p
break
}
}
if fullGuided == nil {
t.Fatal("未找到full_guided预设")
}
enabledCount := 0
for _, s := range fullGuided.Stages {
if s.Enabled {
enabledCount++
}
}
if enabledCount != 5 {
t.Errorf("full_guided应有5个启用阶段，实际%d个", enabledCount)
}
if fullGuided.PromptMode != models.PromptModeGuided {
t.Errorf("full_guided的PromptMode应为guided，实际%s", fullGuided.PromptMode)
}
}

// TestGetFlowPresets_ExpressHasTwoEnabled 极速模式应只有write+revise
func TestGetFlowPresets_ExpressHasTwoEnabled(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
var express *models.FlowPreset
for _, p := range presets {
if p.Key == "express" {
express = p
break
}
}
if express == nil {
t.Fatal("未找到express预设")
}
enabledCount := 0
enabledCodes := []string{}
for _, s := range express.Stages {
if s.Enabled {
enabledCount++
enabledCodes = append(enabledCodes, s.StageCode)
}
}
if enabledCount != 2 {
t.Errorf("express应有2个启用阶段，实际%d个: %v", enabledCount, enabledCodes)
}
if express.PromptMode != models.PromptModeEfficient {
t.Errorf("express的PromptMode应为efficient，实际%s", express.PromptMode)
}
}

// TestGetFlowPresets_AllHaveRequiredFields 所有预设模板字段非空
func TestGetFlowPresets_AllHaveRequiredFields(t *testing.T) {
svc := newRecipeService()
presets := svc.GetFlowPresets()
for _, p := range presets {
if p.Key == "" {
t.Error("预设模板Key不应为空")
}
if p.Name == "" {
t.Errorf("预设模板 %q Name不应为空", p.Key)
}
if p.Description == "" {
t.Errorf("预设模板 %q Description不应为空", p.Key)
}
if p.Duration == "" {
t.Errorf("预设模板 %q Duration不应为空", p.Key)
}
if p.Icon == "" {
t.Errorf("预设模板 %q Icon不应为空", p.Key)
}
if p.PromptMode == "" {
t.Errorf("预设模板 %q PromptMode不应为空", p.Key)
}
if len(p.Stages) == 0 {
t.Errorf("预设模板 %q Stages不应为空", p.Key)
}
}
}
