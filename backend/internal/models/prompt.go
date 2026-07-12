package models

import (
	"time"
)

// ==================== 提示词模型 ====================
//
// v2 改造（提示词管理页整合治理）：
//   原设计只硬编码 9 个 Pipeline 时代的 key（prompt_a~g/dict/ability_table），
//   但 prompts 表已随课件工坊、知识库压缩等模块膨胀到 28 个 key。
//   旧的 IsValidPromptKey 写死 9 个白名单，导致课件/知识库那 19 个 key
//   在管理页「看得到但一保存就被 ErrInvalidPromptKey 拦死」（半残状态）。
//
//   本次治理方向：
//     1. key 校验改为「DB 里存在即合法」——校验逻辑下沉到 service 层调
//        repository.PromptKeyExists（本文件不再维护 ValidPromptKeys 白名单）。
//        以后任何模块往 prompts 表插新 key，管理页自动纳管，无需改代码。
//     2. 每个 key 附带「危险分档」PromptCategory（high/mid/kb），前端据此
//        用红/橙/绿三色标记 + 差异化二次确认文案，防止误改课件重型提示词
//        导致批量生成崩溃。未登记分档的新 key 默认归「mid（中危）」保守兜底。
//     3. 中文名 / 用途描述随响应下发（PromptResponse.Category / Description），
//        前端不再硬编码这些文案，新增 key 也能显示合理默认名。

// Prompt 对应数据库 prompts 表
type Prompt struct {
	ID        string    `json:"id"`         // UUID 主键（数据库 gen_random_uuid）
	PromptKey string    `json:"prompt_key"` // 提示词标识（如 prompt_a / prompt_courseware_generate）
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
	PromptName  string    `json:"prompt_name"`  // 提示词中文名（PromptNameMap 取，缺省回退 key）
	Category    string    `json:"category"`     // 危险分档：high/mid/kb（前端红/橙/绿 + 二次确认文案）
	Description string    `json:"description"`  // 用途说明（PromptDescriptionMap 取，缺省回退通用文案）
	Content     string    `json:"content"`      // 提示词内容
	Version     int       `json:"version"`      // 当前版本号
	ContentLen  int       `json:"content_len"`  // 内容长度（字符数）
	IsCurrent   bool      `json:"is_current"`   // 是否为当前版本
	CreatedBy   *string   `json:"created_by"`   // 创建者ID
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
}

// PromptListResponse 提示词列表响应（当前生效版本全集）
type PromptListResponse struct {
	Prompts []PromptResponse `json:"prompts"` // 提示词列表（当前生效版本）
	Total   int              `json:"total"`   // 总数
}

// PromptVersionResponse 单条版本历史记录
type PromptVersionResponse struct {
	ID         string    `json:"id"`          // UUID
	Version    int       `json:"version"`     // 版本号
	Content    string    `json:"content"`     // 该版本的内容
	ContentLen int       `json:"content_len"` // 内容长度
	IsCurrent  bool      `json:"is_current"`  // 是否为当前生效版本
	CreatedBy  *string   `json:"created_by"`  // 创建者ID
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
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

// ==================== 危险分档常量 ====================
//
// 三档语义：
//   high — 高危：改错直接导致线上业务崩溃（课件生成/渲染类，含画布契约、
//          字号硬约束、API 占位符等精密约定）。前端红色警示 + 最严厉二次确认。
//   kb   — 知识库类：相对独立的课标/教材压缩入库提示词，改动影响面局限在
//          知识库压缩子系统。前端绿色 + 普通二次确认。
//   mid  — 中危：其余全部（Pipeline 八步、索引字典、各业务提示词）。改动影响
//          质量但不至于直接崩。也是「未登记新 key」的默认兜底档。前端橙色 + 普通二次确认。
const (
	PromptCategoryHigh = "high" // 高危：课件生成/渲染类，改错崩业务
	PromptCategoryMid  = "mid"  // 中危：Pipeline/索引/业务类（也是新 key 默认档）
	PromptCategoryKB   = "kb"   // 知识库类：课标/教材压缩，影响面局限
)

// PromptCategoryMap 提示词标识 → 危险分档映射。
// 未在本表登记的 key（例如未来新增的提示词）由 GetPromptCategory 兜底为 mid（中危）。
var PromptCategoryMap = map[string]string{
	// ---------- 🔴 高危：课件生成/渲染类（改错直接崩课件工坊） ----------
	"prompt_courseware_generate":         PromptCategoryHigh, // 课件 HTML 逐页生成（画布契约/字号硬约束/API占位符，最精密）
	"prompt_courseware_3d_single":        PromptCategoryHigh, // 3D 互动单页生成（Three.js 完整文档）
	"prompt_courseware_scheme":           PromptCategoryHigh, // 层2 方案翻译（8字段 JSON，喂给逐页生成）
	"prompt_courseware_index":            PromptCategoryHigh, // 课件页级索引字典（脉络+逐页 AOCI）
	"prompt_courseware_template_extract": PromptCategoryHigh, // 风格模板 AI 提取（5色9变量样例页契约）
	"prompt_courseware_template_refine":  PromptCategoryHigh, // 风格模板 AI 微调
	"prompt_courseware_image_prompt":     PromptCategoryHigh, // 配图提示词改写（占位驱动+风格锚点联动）
	"prompt_courseware_video_prompt":     PromptCategoryHigh, // 视频分镜数组策划（严格 JSON 数组）

	// ---------- 🟢 知识库类：课标/教材压缩入库（影响面局限在知识库子系统） ----------
	"prompt_curriculum_index": PromptCategoryKB, // 课标压缩一行制索引
	"prompt_textbook_index":   PromptCategoryKB, // 教材单元压缩索引
	"dict_curriculum":         PromptCategoryKB, // 课标索引解码字典
	"dict_textbook":           PromptCategoryKB, // 教材索引解码字典

	// ---------- 🟡 中危：Pipeline 八步 + 索引字典 + 各业务提示词 ----------
	"prompt_a":                           PromptCategoryMid, // Scanner 扫描定位
	"prompt_b":                           PromptCategoryMid, // Evaluator 评估打分
	"prompt_c":                           PromptCategoryMid, // Translator 翻译转换
	"prompt_d":                           PromptCategoryMid, // Reviewer 审核检查
	"prompt_e":                           PromptCategoryMid, // Meta 元评估仲裁
	"prompt_f":                           PromptCategoryMid, // Generator 页面生成
	"prompt_g":                           PromptCategoryMid, // IndexGen 索引生成器（验收用）
	"dict":                               PromptCategoryMid, // TE-DNA 通用解压缩字典
	"ability_table":                      PromptCategoryMid, // 能力定位表
	"prompt_component_index":             PromptCategoryMid, // 教案组件索引字典
	"prompt_lesson_index":                PromptCategoryMid, // 教案 AOCI 索引字典
	"prompt_stage_coach":                 PromptCategoryMid, // 阶段教练评估
	"prompt_courseware_alignment":        PromptCategoryMid, // 课件↔教案对齐校验
	"prompt_courseware_lesson_normalize": PromptCategoryMid, // 教案预处理规整
	"prompt_courseware_vaoci_extract":    PromptCategoryMid, // 课件配图风格锚点提取
	"prompt_unit_design":                 PromptCategoryMid, // 单元方案逐步设计
}

// GetPromptCategory 返回指定 key 的危险分档。
// 未登记的 key 一律兜底为 mid（中危）——保证新增提示词至少有二次确认保护，
// 符合「DB 有就纳管、未登记默认中危」的产品决策（选项 A）。
func GetPromptCategory(key string) string {
	if c, ok := PromptCategoryMap[key]; ok {
		return c
	}
	return PromptCategoryMid
}

// ==================== 中文名与描述映射 ====================

// PromptNameMap 提示词标识 → 中文名。缺省由 GetPromptName 回退为 key 本身。
var PromptNameMap = map[string]string{
	// Pipeline 八步
	"prompt_a":      "Prompt A — Scanner 扫描定位",
	"prompt_b":      "Prompt B — Evaluator 评估打分",
	"prompt_c":      "Prompt C — Translator 翻译转换",
	"prompt_d":      "Prompt D — Reviewer 审核检查",
	"prompt_e":      "Prompt E — Meta 元评估仲裁",
	"prompt_f":      "Prompt F — Generator 页面生成",
	"prompt_g":      "Prompt G — IndexGen 索引生成器",
	"dict":          "TE-DNA 通用解压缩字典",
	"ability_table": "能力定位表",
	// 教案 / 组件 / 阶段索引字典
	"prompt_component_index": "教案组件索引字典",
	"prompt_lesson_index":    "教案 AOCI 索引字典",
	"prompt_stage_coach":     "阶段教练评估",
	"prompt_unit_design":     "单元方案逐步设计",
	// 课件工坊系列
	"prompt_courseware_generate":         "课件生成 — 逐页 HTML 生成",
	"prompt_courseware_3d_single":        "课件生成 — 3D 互动单页",
	"prompt_courseware_scheme":           "课件方案 — 层2 方案翻译",
	"prompt_courseware_index":            "课件索引 — 页级 AOCI 字典",
	"prompt_courseware_template_extract": "课件模板 — AI 风格提取",
	"prompt_courseware_template_refine":  "课件模板 — AI 风格微调",
	"prompt_courseware_image_prompt":     "课件配图 — 提示词改写",
	"prompt_courseware_video_prompt":     "课件视频 — 分镜数组策划",
	"prompt_courseware_alignment":        "课件校验 — 方案↔教案对齐",
	"prompt_courseware_lesson_normalize": "课件预处理 — 教案规整",
	"prompt_courseware_vaoci_extract":    "课件配图 — 风格锚点提取",
	// 知识库压缩系列
	"prompt_curriculum_index": "知识库 — 课标压缩索引",
	"prompt_textbook_index":   "知识库 — 教材压缩索引",
	"dict_curriculum":         "知识库 — 课标解码字典",
	"dict_textbook":           "知识库 — 教材解码字典",
}

// PromptDescriptionMap 提示词标识 → 用途说明。缺省由 GetPromptDescription 回退为通用文案。
var PromptDescriptionMap = map[string]string{
	// Pipeline 八步
	"prompt_a":      "K12 课程定位：课程体系 + 能力定位表 + 学段标准",
	"prompt_b":      "4 维度评估：E1 难度 + E2 时间 + E3 互动 + E4 课程设计",
	"prompt_c":      "索引差异 → 逐页修改指令，零编码泄露",
	"prompt_d":      "一致性 + 质量双层检查",
	"prompt_e":      "N 轮交叉比对 + 修改方案 + 优化索引",
	"prompt_f":      "HTML 最小侵入修改 + 多种 op 分流",
	"prompt_g":      "验收用索引压缩：HTML → 课程页面索引 + 模块索引",
	"dict":          "TE-DNA 编码格式解压缩速查表",
	"ability_table": "课程 × 能力等级对照表",
	// 教案 / 组件 / 阶段
	"prompt_component_index": "教案组件 6 维编码 + 5 语义标签解码字典",
	"prompt_lesson_index":    "教案 6 维编码 + 5 语义标签解码字典",
	"prompt_stage_coach":     "备课 5 阶段检查规则 + 教练评分",
	"prompt_unit_design":     "大单元备课逐步引导对话，产出整单元教学设计",
	// 课件工坊
	"prompt_courseware_generate":         "⚠ 课件核心：1920×1080 画布契约 + 字号硬约束 + edu 平台 API 占位符。改错直接导致批量生成崩溃",
	"prompt_courseware_3d_single":        "⚠ Three.js 完整 HTML 文档生成，含技术约束与步骤系统。改错致 3D 单页无法渲染",
	"prompt_courseware_scheme":           "⚠ 层2 方案翻译，输出逐页 8 字段 JSON 喂给生成引擎。改错致方案解析失败",
	"prompt_courseware_index":            "⚠ 课件页级索引字典，输出脉络概述 + 逐页 AOCI。改错致索引生成异常",
	"prompt_courseware_template_extract": "⚠ 从 HTML 抽象提取风格模板（5 色 + 9 CSS 变量 + 样例页契约）",
	"prompt_courseware_template_refine":  "⚠ 按自然语言指令微调风格模板，保持画布与变量绑定",
	"prompt_courseware_image_prompt":     "⚠ 配图提示词改写，占位驱动 + 风格锚点联动。改错致配图跑偏",
	"prompt_courseware_video_prompt":     "⚠ 视频分镜策划，输出严格 JSON 数组。改错致分镜解析失败",
	"prompt_courseware_alignment":        "课件方案与教案的对齐校验报告",
	"prompt_courseware_lesson_normalize": "教案 / 文档原文 AI 规整为去噪保核的干净文本",
	"prompt_courseware_vaoci_extract":    "多模态读图，提取风格 DNA + 人物固定形象",
	// 知识库压缩
	"prompt_curriculum_index": "课标知识点压缩为一行制多维索引",
	"prompt_textbook_index":   "教材单元压缩为一行制多维索引",
	"dict_curriculum":         "课标一行制索引 → 人话解码映射表",
	"dict_textbook":           "教材一行制索引 → 人话解码映射表",
}

// GetPromptName 返回中文名，未登记则回退 key 本身（保证新增 key 也有可读名）
func GetPromptName(key string) string {
	if n, ok := PromptNameMap[key]; ok {
		return n
	}
	return key
}

// GetPromptDescription 返回用途说明，未登记则回退通用文案
func GetPromptDescription(key string) string {
	if d, ok := PromptDescriptionMap[key]; ok {
		return d
	}
	return "（未登记说明的提示词，编辑前请确认其用途与影响范围）"
}
