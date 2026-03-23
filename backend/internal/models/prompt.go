package models

import (
	"time"
)

// ==================== 提示词模型 ====================

// Prompt 对应数据库 prompts 表
type Prompt struct {
	ID        string    `json:"id"`         // UUID 主键（数据库 gen_random_uuid）
	PromptKey string    `json:"prompt_key"` // 提示词标识（prompt_a~f / dict / ability_table）
	Content   string    `json:"content"`    // 提示词完整内容
	Version   int       `json:"version"`    // 版本号（从1开始递增）
	IsCurrent bool      `json:"is_current"` // 是否为当前生效版本
	CreatedBy *string   `json:"created_by"` // 创建者用户ID
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// PromptResponse 返回给前端的提示词信息（单条）
type PromptResponse struct {
	ID          string    `json:"id"`           // UUID
	PromptKey   string    `json:"prompt_key"`   // 提示词标识
	PromptName  string    `json:"prompt_name"`  // 提示词中文名
	Content     string    `json:"content"`      // 提示词内容
	Version     int       `json:"version"`      // 当前版本号
	ContentLen  int       `json:"content_len"`  // 内容长度（字符数）
	IsCurrent   bool      `json:"is_current"`   // 是否为当前版本
	CreatedBy   *string   `json:"created_by"`   // 创建者ID
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
}

// PromptListResponse 提示词列表响应（8个槽位）
type PromptListResponse struct {
	Prompts []PromptResponse `json:"prompts"` // 提示词列表（当前生效版本）
	Total   int              `json:"total"`   // 总数
}

// PromptVersionResponse 单条版本历史记录
type PromptVersionResponse struct {
	ID        string    `json:"id"`         // UUID
	Version   int       `json:"version"`    // 版本号
	Content   string    `json:"content"`    // 该版本的内容
	ContentLen int      `json:"content_len"` // 内容长度
	IsCurrent bool      `json:"is_current"` // 是否为当前生效版本
	CreatedBy *string   `json:"created_by"` // 创建者ID
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// PromptVersionListResponse 版本历史列表响应
type PromptVersionListResponse struct {
	PromptKey  string                  `json:"prompt_key"`  // 提示词标识
	PromptName string                  `json:"prompt_name"` // 提示词中文名
	Versions   []PromptVersionResponse `json:"versions"`    // 版本列表（按版本号倒序）
	Total      int                     `json:"total"`       // 总版本数
}

// UpdatePromptRequest 更新提示词请求体
type UpdatePromptRequest struct {
	Content string `json:"content"` // 新的提示词内容（完整内容）
}

// ==================== 提示词槽位常量与映射 ====================

// 提示词标识常量（8个槽位）
const (
	PromptKeyA            = "prompt_a"      // Scanner 扫描定位提示词
	PromptKeyB            = "prompt_b"      // Evaluator 评估打分提示词
	PromptKeyC            = "prompt_c"      // Translator 翻译转换提示词
	PromptKeyD            = "prompt_d"      // Reviewer 审核检查提示词
	PromptKeyE            = "prompt_e"      // Meta 元评估仲裁提示词
	PromptKeyF            = "prompt_f"      // Generator 页面生成提示词
	PromptKeyDict         = "dict"          // TE-DNA 解压缩字典
	PromptKeyAbilityTable = "ability_table"
	PromptKeyG            = "prompt_g" // 能力定位表
)

// ValidPromptKeys 有效提示词标识列表
var ValidPromptKeys = []string{
	PromptKeyA, PromptKeyB, PromptKeyC, PromptKeyD,
	PromptKeyE, PromptKeyF, PromptKeyDict, PromptKeyAbilityTable, PromptKeyG,
}

// PromptNameMap 提示词标识→中文名映射
var PromptNameMap = map[string]string{
	PromptKeyA:            "Prompt A — Scanner 扫描定位",
	PromptKeyB:            "Prompt B — Evaluator 评估打分",
	PromptKeyC:            "Prompt C — Translator 翻译转换",
	PromptKeyD:            "Prompt D — Reviewer 审核检查",
	PromptKeyE:            "Prompt E — Meta 元评估仲裁",
	PromptKeyF:            "Prompt F — Generator 页面生成",
	PromptKeyDict:         "TE-DNA 解压缩字典",
	PromptKeyAbilityTable: "能力定位表",
	PromptKeyG:            "Prompt G — IndexGen 索引生成器",
}

// PromptDescriptionMap 提示词标识→用途说明映射
var PromptDescriptionMap = map[string]string{
	PromptKeyA:            "K12课程定位：156门课程体系 + 能力定位表 + 学段标准（约15K字）",
	PromptKeyB:            "4维度评估：E1难度适配 + E2时间节奏 + E3互动评估 + E4课程设计（约12K字）",
	PromptKeyC:            "索引差异→逐页修改指令，零编码泄露（约8K字）",
	PromptKeyD:            "一致性 + 质量双层检查（约8K字）",
	PromptKeyE:            "N轮交叉比对 + 修改方案 + 优化索引（约12K字）",
	PromptKeyF:            "HTML最小侵入修改 + 5种op分流（约2K字）",
	PromptKeyDict:         "TE-DNA编码格式解压缩速查表（约3K字）",
	PromptKeyAbilityTable: "156门课×能力等级对照表，Tab分隔（约20K字）",
	PromptKeyG:            "验收用索引压缩：HTML→课程页面索引+模块索引（约17K字）",
}

// IsValidPromptKey 检查提示词标识是否有效
func IsValidPromptKey(key string) bool {
	for _, k := range ValidPromptKeys {
		if k == key {
			return true
		}
	}
	return false
}
