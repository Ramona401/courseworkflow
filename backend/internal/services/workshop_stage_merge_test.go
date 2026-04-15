package services

// workshop_stage_merge_test.go — 阶段合并辅助函数单元测试
//
// 测试范围（纯函数，无数据库依赖）：
//   - buildDefaultSnapshots：从系统阶段定义构建快照
//   - detectStagesConfigFormat：检测配置格式（new/legacy/unknown）
//   - findStageIndex：查找阶段索引
//   - removeStage：移除指定阶段
//   - mergeStagesLegacyFormat（WorkshopStageService方法，但纯逻辑）

import (
"encoding/json"
"testing"

"tedna/internal/models"
)

// ==================== buildDefaultSnapshots 测试 ====================

// TestBuildDefaultSnapshots_Basic 基本快照构建
func TestBuildDefaultSnapshots_Basic(t *testing.T) {
defaults := []*models.WorkshopStage{
{StageCode: "analyze", StageName: "教学分析", StageOrder: 1, AIRole: "分析师", GateMode: "suggest", Skippable: true},
{StageCode: "write", StageName: "教案撰写", StageOrder: 3, AIRole: "撰写者", GateMode: "force", Skippable: false},
}
snapshots := buildDefaultSnapshots(defaults)
if len(snapshots) != 2 {
t.Fatalf("应有2个快照，实际%d", len(snapshots))
}
if snapshots[0].StageCode != "analyze" {
t.Errorf("第一个应为analyze，实际%s", snapshots[0].StageCode)
}
if snapshots[0].StageName != "教学分析" {
t.Errorf("StageName应为教学分析，实际%s", snapshots[0].StageName)
}
if snapshots[0].StageOrder != 1 {
t.Errorf("StageOrder应为1，实际%d", snapshots[0].StageOrder)
}
if snapshots[0].AIRole != "分析师" {
t.Errorf("AIRole应为分析师，实际%s", snapshots[0].AIRole)
}
if snapshots[0].GateMode != "suggest" {
t.Errorf("GateMode应为suggest，实际%s", snapshots[0].GateMode)
}
if !snapshots[0].Skippable {
t.Error("Skippable应为true")
}
if snapshots[1].Skippable {
t.Error("write的Skippable应为false")
}
}

// TestBuildDefaultSnapshots_Empty 空输入返回空切片
func TestBuildDefaultSnapshots_Empty(t *testing.T) {
snapshots := buildDefaultSnapshots(nil)
if len(snapshots) != 0 {
t.Errorf("nil输入应返回空切片，实际%d个", len(snapshots))
}
snapshots2 := buildDefaultSnapshots([]*models.WorkshopStage{})
if len(snapshots2) != 0 {
t.Errorf("空切片输入应返回空切片，实际%d个", len(snapshots2))
}
}

// ==================== detectStagesConfigFormat 测试 ====================

// TestDetectStagesConfigFormat_NewFormat 新格式（含enabled字段）
func TestDetectStagesConfigFormat_NewFormat(t *testing.T) {
config := `[{"stage_code":"analyze","enabled":true,"order":1}]`
format := detectStagesConfigFormat(config)
if format != "new" {
t.Errorf("含enabled字段应识别为new，实际%s", format)
}
}

// TestDetectStagesConfigFormat_LegacyFormat 旧格式（含action字段）
func TestDetectStagesConfigFormat_LegacyFormat(t *testing.T) {
config := `[{"stage_code":"analyze","action":"skip"}]`
format := detectStagesConfigFormat(config)
if format != "legacy" {
t.Errorf("含action字段应识别为legacy，实际%s", format)
}
}

// TestDetectStagesConfigFormat_Unknown 未知格式
func TestDetectStagesConfigFormat_Unknown(t *testing.T) {
tests := []struct {
name   string
input  string
expect string
}{
{"空对象数组", `[{"stage_code":"analyze"}]`, "unknown"},
{"空数组", `[]`, "unknown"},
{"无效JSON", `not json`, "unknown"},
{"空字符串", ``, "unknown"},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
format := detectStagesConfigFormat(tc.input)
if format != tc.expect {
t.Errorf("输入%q应返回%s，实际%s", tc.input, tc.expect, format)
}
})
}
}

// ==================== findStageIndex 测试 ====================

// TestFindStageIndex_Found 找到指定阶段
func TestFindStageIndex_Found(t *testing.T) {
snapshots := []models.StageConfigSnapshot{
{StageCode: "analyze"},
{StageCode: "design"},
{StageCode: "write"},
}
idx := findStageIndex(snapshots, "design")
if idx != 1 {
t.Errorf("design应在索引1，实际%d", idx)
}
}

// TestFindStageIndex_NotFound 未找到返回-1
func TestFindStageIndex_NotFound(t *testing.T) {
snapshots := []models.StageConfigSnapshot{
{StageCode: "analyze"},
}
idx := findStageIndex(snapshots, "non_existent")
if idx != -1 {
t.Errorf("未找到应返回-1，实际%d", idx)
}
}

// TestFindStageIndex_EmptyList 空列表返回-1
func TestFindStageIndex_EmptyList(t *testing.T) {
idx := findStageIndex([]models.StageConfigSnapshot{}, "analyze")
if idx != -1 {
t.Errorf("空列表应返回-1，实际%d", idx)
}
}

// ==================== removeStage 测试 ====================

// TestRemoveStage_Exists 移除存在的阶段
func TestRemoveStage_Exists(t *testing.T) {
snapshots := []models.StageConfigSnapshot{
{StageCode: "analyze"},
{StageCode: "design"},
{StageCode: "write"},
}
result := removeStage(snapshots, "design")
if len(result) != 2 {
t.Fatalf("移除后应有2个阶段，实际%d", len(result))
}
if result[0].StageCode != "analyze" || result[1].StageCode != "write" {
t.Errorf("移除design后应为[analyze,write]，实际[%s,%s]", result[0].StageCode, result[1].StageCode)
}
}

// TestRemoveStage_NotExists 移除不存在的阶段（原样返回）
func TestRemoveStage_NotExists(t *testing.T) {
snapshots := []models.StageConfigSnapshot{
{StageCode: "analyze"},
{StageCode: "write"},
}
result := removeStage(snapshots, "non_existent")
if len(result) != 2 {
t.Errorf("移除不存在的阶段不应改变长度，实际%d", len(result))
}
}

// TestRemoveStage_EmptyList 空列表移除不panic
func TestRemoveStage_EmptyList(t *testing.T) {
result := removeStage([]models.StageConfigSnapshot{}, "analyze")
if len(result) != 0 {
t.Errorf("空列表移除应返回空，实际%d", len(result))
}
}

// ==================== mergeStagesLegacyFormat 测试 ====================

// TestMergeStagesLegacyFormat_Skip 旧格式skip操作
func TestMergeStagesLegacyFormat_Skip(t *testing.T) {
svc := &WorkshopStageService{}
defaults := []models.StageConfigSnapshot{
{StageCode: "analyze", StageName: "教学分析", StageOrder: 1},
{StageCode: "design", StageName: "教学设计", StageOrder: 2},
{StageCode: "write", StageName: "教案撰写", StageOrder: 3},
}
config := `[{"stage_code":"design","action":"skip"}]`
result, err := svc.mergeStagesLegacyFormat(defaults, config)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if len(result) != 2 {
t.Fatalf("skip后应有2个阶段，实际%d", len(result))
}
// 应重新编号
if result[0].StageOrder != 1 || result[1].StageOrder != 2 {
t.Errorf("重新编号后应为1,2，实际%d,%d", result[0].StageOrder, result[1].StageOrder)
}
}

// TestMergeStagesLegacyFormat_Override 旧格式override操作
func TestMergeStagesLegacyFormat_Override(t *testing.T) {
svc := &WorkshopStageService{}
defaults := []models.StageConfigSnapshot{
{StageCode: "analyze", StageName: "教学分析", AIRole: "原角色", GateMode: "suggest"},
}
config := `[{"stage_code":"analyze","action":"override","ai_role":"新角色","gate_mode":"force"}]`
result, err := svc.mergeStagesLegacyFormat(defaults, config)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if result[0].AIRole != "新角色" {
t.Errorf("AIRole应被覆盖为'新角色'，实际%s", result[0].AIRole)
}
if result[0].GateMode != "force" {
t.Errorf("GateMode应被覆盖为force，实际%s", result[0].GateMode)
}
if result[0].StageName != "教学分析" {
t.Errorf("未覆盖的字段应保持原值，StageName实际%s", result[0].StageName)
}
}

// TestMergeStagesLegacyFormat_InsertAfter 旧格式insert_after操作
func TestMergeStagesLegacyFormat_InsertAfter(t *testing.T) {
svc := &WorkshopStageService{}
defaults := []models.StageConfigSnapshot{
{StageCode: "analyze", StageName: "教学分析", StageOrder: 1},
{StageCode: "write", StageName: "教案撰写", StageOrder: 2},
}
config := `[{"stage_code":"custom_stage","action":"insert_after","insert_after":"analyze","stage_name":"自定义阶段","ai_role":"自定义角色"}]`
result, err := svc.mergeStagesLegacyFormat(defaults, config)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if len(result) != 3 {
t.Fatalf("插入后应有3个阶段，实际%d", len(result))
}
if result[1].StageCode != "custom_stage" {
t.Errorf("第二个应为custom_stage，实际%s", result[1].StageCode)
}
if result[1].StageName != "自定义阶段" {
t.Errorf("插入阶段名应为'自定义阶段'，实际%s", result[1].StageName)
}
// 插入阶段无gate_mode时应默认为suggest
if result[1].GateMode != models.StageGateSuggest {
t.Errorf("默认GateMode应为suggest，实际%s", result[1].GateMode)
}
// 重新编号
for i, s := range result {
if s.StageOrder != i+1 {
t.Errorf("阶段%d的StageOrder应为%d，实际%d", i, i+1, s.StageOrder)
}
}
}

// TestMergeStagesLegacyFormat_InvalidJSON 无效JSON使用默认值
func TestMergeStagesLegacyFormat_InvalidJSON(t *testing.T) {
svc := &WorkshopStageService{}
defaults := []models.StageConfigSnapshot{
{StageCode: "write", StageName: "教案撰写", StageOrder: 1},
}
result, err := svc.mergeStagesLegacyFormat(defaults, "not valid json")
if err != nil {
t.Fatalf("无效JSON不应报错: %v", err)
}
if len(result) != 1 {
t.Errorf("无效JSON应返回原始默认阶段，实际%d个", len(result))
}
}

// TestMergeStagesLegacyFormat_OverrideSkippable 覆盖Skippable字段
func TestMergeStagesLegacyFormat_OverrideSkippable(t *testing.T) {
svc := &WorkshopStageService{}
defaults := []models.StageConfigSnapshot{
{StageCode: "analyze", Skippable: true},
}
// Skippable是指针类型，需要在JSON中明确设置
skippableFalse := false
overrides := []models.StageOverride{
{StageCode: "analyze", Action: "override", Skippable: &skippableFalse},
}
configJSON, _ := json.Marshal(overrides)
result, err := svc.mergeStagesLegacyFormat(defaults, string(configJSON))
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if result[0].Skippable != false {
t.Error("Skippable应被覆盖为false")
}
}
