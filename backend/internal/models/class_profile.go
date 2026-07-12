package models

// class_profile.go — 班级学情数据模型（差异化教学·老师私有资料，独立模块）
//
// 班级学情 = 老师为自己带的某个班建立的「学情卡」，落在「我的备课资料 → 班级学情」Tab。
// 它是一份"料"（备课资料）：备课时老师显式挂载某个班，引擎全阶段注入这张卡的群体学情内容，
// 帮助 AI 做分层教学设计。
//
// 三层数据结构（核心设计，务必理解）：
//   1. ClassStudent  学生个体档案（本地明细，永不注入 AI）—— 学号代号+成绩+分层+备注，是"原料库"
//   2. AI 定期总结   —— 把学生明细在后端就地汇总成统计量后喂 AI（个体明细绝不进注入链路）
//   3. ClassProfile  班级学情卡（群体结论，注入 AI）—— 整体画像+分层结构+薄弱点+教学建议，是"成品"
//
// 合规红线（贯穿设计，不可妥协）：
//   - 学生个体档案只存「学号代号」（如 01 / S001），绝不存真名。
//   - 个体明细永不注入 AI；注入 AI 的永远只有班级卡的"群体结论"（匿名、无个人身份信息）。
//
// 归属：纯个人（v1 决策）。鉴权一律按 created_by/owner_id == 当前用户，
//       不走 unit_plans 那套 group/school/system 可见性。表里保留 scope/scope_target_id
//       两列只是为了将来若要做共享时零迁移扩展，v1 固定 personal + 全零占位。
//
// ---------- 批次2a 变更说明 ----------
// 批次1 只承载了班级卡（第三层）的请求体。批次2a 补齐"学生个体档案"（第一层）的请求体与
// 学号代号自动编号辅助，供 service/handler 把已备好的 repository 学生 CRUD 接出前端。
// 成绩字段（scores/latest_score）在手动录入路径里只读、不接收前端手填——成绩走批次2b 的
// 成绩单导入，由后端归并写入。故学生请求体里不含成绩字段。
//
// ---------- 批次2b 变更说明 ----------
// 补齐"成绩单导入"的请求体/响应体与成绩归并辅助函数。导入是成绩进入系统的唯一路径。
//
// ---------- 批次2b-2 变更说明 ----------
// 贴合老师真实用法（"一次考试一个批次、对着成绩挨个学生填一行"）做两点调整：
//   1. 考试名称 + 考试日期从"每行自带"上提为"整批统一"——一次导入就是一次考试，
//      老师在弹窗里填一次考试名和日期，Excel 表里不再含这两列，每行只填逐生不同的内容。
//   2. 导入行从"只带成绩"扩为"带成绩 + 薄弱点 + 备注"——老师对着成绩顺手把这次观察到的
//      薄弱点和备注一起填，免得导完成绩再回平台一个个补。
//      归并语义：成绩追加进 scores 数组（看趋势）；薄弱点/备注非空则覆盖学生档案当前值
//      （取老师最新判断），留空则不动（避免误清旧值）。
//   合规红线同样覆盖薄弱点/备注：不写真名与隐私（界面文案兜底）。

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ---------- 常量 ----------

// scope（v1 固定 personal，其余值预留）
const (
	ClassProfileScopePersonal = "personal"
)

// 班级学情卡的占位归属ID（全零UUID，与其它资料表同一套占位规范）
const ClassProfileSystemTargetID = "00000000-0000-0000-0000-000000000000"

// 状态
const (
	ClassProfileStatusActive   = "active"   // 正常
	ClassProfileStatusArchived = "archived" // 软删除
)

// 学情卡最近一次更新来源（last_analyzed_from，仅留痕展示用）
const (
	ClassAnalyzedFromManual      = "manual"       // 老师手写
	ClassAnalyzedFromAIChat      = "ai_chat"      // AI 对话引导生成
	ClassAnalyzedFromScoreImport = "score_import" // 成绩单导入生成
	ClassAnalyzedFromAISummary   = "ai_summary"   // 由学生档案 AI 总结生成
)

// 学生分层标签（v1 固定 ABC 三层）
const (
	StudentTierNone = "" // 未分层
	StudentTierA    = "A" // 拔尖
	StudentTierB    = "B" // 中等
	StudentTierC    = "C" // 学困
)

// ---------- 实体：班级学情卡（群体结论，注入 AI）----------

// ClassProfile 班级学情卡实体（对应 class_profiles 表）
//
// overall_profile / tier_structure / weak_points / teaching_advice 四个字段
// 是"会注入 AI 的群体学情内容"，均为匿名群体描述，不含任何学生个人身份信息。
type ClassProfile struct {
	ID            string `json:"id"`
	Scope         string `json:"scope"`           // v1 固定 personal
	ScopeTargetID string `json:"scope_target_id"` // v1 固定全零占位

	// 班级定位字段
	Subject      string `json:"subject"`       // 学科
	Grade        string `json:"grade"`         // 年级或学段文字
	ClassName    string `json:"class_name"`    // 班级名，如"初二3班"
	Term         string `json:"term"`          // 学期/学年
	StudentCount int    `json:"student_count"` // 班级人数

	// ===== 注入 AI 的群体学情内容（成品）=====
	OverallProfile string `json:"overall_profile"` // 整体画像（基础水平/学习风格/班风）
	TierStructure  string `json:"tier_structure"`  // 分层结构（A/B/C 群体描述，匿名）
	WeakPoints     string `json:"weak_points"`     // 学科薄弱点
	TeachingAdvice string `json:"teaching_advice"` // 分层教学建议

	// 分析留痕
	LastAnalyzedAt   *time.Time `json:"last_analyzed_at"`   // 最近一次总结时间，nil=从未总结
	LastAnalyzedFrom string     `json:"last_analyzed_from"` // 来源（manual/ai_chat/score_import/ai_summary）

	// 通用字段
	CreatedBy string    `json:"created_by"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClassProfileListItem 列表项（不含正文四大段，列表轻量）
type ClassProfileListItem struct {
	ID               string     `json:"id"`
	Subject          string     `json:"subject"`
	Grade            string     `json:"grade"`
	ClassName        string     `json:"class_name"`
	Term             string     `json:"term"`
	StudentCount     int        `json:"student_count"`
	HasProfile       bool       `json:"has_profile"`        // 学情卡四大段是否已有内容（前端显示"已填/待完善"）
	LastAnalyzedAt   *time.Time `json:"last_analyzed_at"`   // 最近总结时间
	LastAnalyzedFrom string     `json:"last_analyzed_from"` // 来源
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ---------- 实体：学生个体档案（本地明细，永不注入 AI）----------

// ClassStudentScore 学生历次成绩的一条记录（存 scores jsonb 数组元素）
//
// Max 为满分（导入模板未含满分列时为 0，表示"未提供满分"，展示时只显得分）。
// At 为考试日期标记（导入路径下统一为 YYYY-MM-DD，便于按日期取最新）。
type ClassStudentScore struct {
	Name  string  `json:"name"`  // 考试/作业名，如"期中"
	Score float64 `json:"score"` // 得分
	Max   float64 `json:"max"`   // 满分（0=未提供）
	At    string  `json:"at"`    // 时间标记，如"2026-01-15"
}

// ClassStudent 学生个体档案实体（对应 class_students 表）
//
// ⚠ 合规红线：永不存真名（只存 student_code 学号代号），本实体永不注入 AI。
type ClassStudent struct {
	ID             string    `json:"id"`
	ClassProfileID string    `json:"class_profile_id"` // 归属班级
	OwnerID        string    `json:"owner_id"`         // 创建人（冗余，便于直接鉴权）
	StudentCode    string    `json:"student_code"`     // 学号代号，如"01"/"S001"
	Tier           string    `json:"tier"`             // A/B/C，空=未分层
	Scores         string    `json:"-"`                // jsonb 原始文本，service 层按需 Parse
	LatestScore    *float64  `json:"latest_score"`     // 最近一次成绩（冗余，便于排序），nil=无
	WeakTopics     string    `json:"weak_topics"`      // 易错点/薄弱知识点
	Note           string    `json:"note"`             // 老师私人备注
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ClassStudentView 学生档案对前端的展示结构（Scores 已解析为数组）
type ClassStudentView struct {
	ID          string              `json:"id"`
	StudentCode string              `json:"student_code"`
	Tier        string              `json:"tier"`
	Scores      []ClassStudentScore `json:"scores"`
	LatestScore *float64            `json:"latest_score"`
	WeakTopics  string              `json:"weak_topics"`
	Note        string              `json:"note"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// ParseClassStudentScores 解析 scores jsonb 文本为成绩数组（空/非法返空切片）
func ParseClassStudentScores(raw string) []ClassStudentScore {
	if raw == "" || raw == "[]" {
		return []ClassStudentScore{}
	}
	var arr []ClassStudentScore
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return []ClassStudentScore{}
	}
	return arr
}

// ScoresToJSON 把成绩数组序列化为 jsonb 文本（nil/空数组返 "[]"）
func ScoresToJSON(arr []ClassStudentScore) string {
	if len(arr) == 0 {
		return "[]"
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ToClassStudentView 把档案实体转为前端展示结构（解析 scores）
func (s *ClassStudent) ToClassStudentView() ClassStudentView {
	return ClassStudentView{
		ID:          s.ID,
		StudentCode: s.StudentCode,
		Tier:        s.Tier,
		Scores:      ParseClassStudentScores(s.Scores),
		LatestScore: s.LatestScore,
		WeakTopics:  s.WeakTopics,
		Note:        s.Note,
		UpdatedAt:   s.UpdatedAt,
	}
}

// ---------- 请求体：班级学情卡 ----------

// CreateClassProfileRequest 新建班级学情卡（手写入口：直接填或留空慢慢补）
type CreateClassProfileRequest struct {
	Subject        string `json:"subject"`         // 必填
	Grade          string `json:"grade"`           // 必填
	ClassName      string `json:"class_name"`      // 必填（班级名）
	Term           string `json:"term"`            // 可选（学期）
	StudentCount   int    `json:"student_count"`   // 可选（人数）
	OverallProfile string `json:"overall_profile"` // 可选（整体画像）
	TierStructure  string `json:"tier_structure"`  // 可选（分层结构）
	WeakPoints     string `json:"weak_points"`     // 可选（薄弱点）
	TeachingAdvice string `json:"teaching_advice"` // 可选（教学建议）
}

// UpdateClassProfileRequest 更新班级学情卡（编辑卡片四大段 + 定位字段）
type UpdateClassProfileRequest struct {
	Subject        string `json:"subject"`
	Grade          string `json:"grade"`
	ClassName      string `json:"class_name"`
	Term           string `json:"term"`
	StudentCount   int    `json:"student_count"`
	OverallProfile string `json:"overall_profile"`
	TierStructure  string `json:"tier_structure"`
	WeakPoints     string `json:"weak_points"`
	TeachingAdvice string `json:"teaching_advice"`
}

// ---------- 请求体：学生个体档案（批次2a 新增）----------
//
// 手动录入路径只接收"定性判断"字段（学号代号/分层/薄弱点/备注），
// 不接收成绩——成绩走批次2b 的成绩单导入由后端归并写入。
// 故请求体里刻意不含 scores/latest_score，防止前端误传成绩走手动通道。

// CreateClassStudentRequest 新建一条学生档案
//
// StudentCode 留空时由 service 层自动编号（取该班现有学号最大纯数字 +1，补零两位）。
type CreateClassStudentRequest struct {
	StudentCode string `json:"student_code"` // 学号代号，留空=自动编号
	Tier        string `json:"tier"`         // A/B/C，空=未分层
	WeakTopics  string `json:"weak_topics"`  // 易错点/薄弱知识点（可选）
	Note        string `json:"note"`         // 老师私人备注（可选）
}

// UpdateClassStudentRequest 更新一条学生档案（同样不含成绩字段）
type UpdateClassStudentRequest struct {
	StudentCode string `json:"student_code"` // 学号代号（编辑时必填，不允许改空）
	Tier        string `json:"tier"`         // A/B/C，空=未分层
	WeakTopics  string `json:"weak_topics"`
	Note        string `json:"note"`
}

// ---------- 请求体/响应体：成绩单导入（批次2b，2b-2 调整）----------
//
// 一次导入 = 一次考试一个批次。考试名称、考试日期整批统一（在请求体顶层），
// 每行只带逐生不同的内容（学号代号 / 分数 / 薄弱点 / 备注）。
// 后端按学号代号归并：成绩追加进 scores 数组；薄弱点/备注非空则覆盖学生档案当前值。

// ImportScoreRow 成绩单的一行（2b-2：去掉考试名/日期，加薄弱点/备注）
//
// 考试名/日期已上提到 ImportScoresRequest 顶层（整批统一），故本行不再含。
// WeakTopics/Note 为老师对着这次成绩顺手填的观察，非空则覆盖该生当前值，留空不动。
type ImportScoreRow struct {
	StudentCode string  `json:"student_code"` // 学号代号（必填；不存在则自动建生）
	Score       float64 `json:"score"`        // 分数（必填）
	WeakTopics  string  `json:"weak_topics"`  // 薄弱知识点（可选，非空覆盖）
	Note        string  `json:"note"`         // 备注（可选，非空覆盖）
}

// ImportScoresRequest 成绩单导入请求（2b-2：考试名/日期上提为整批统一）
type ImportScoresRequest struct {
	ExamName string           `json:"exam_name"` // 本批考试名称（必填，整批统一，如"3月月考"）
	ExamDate string           `json:"exam_date"` // 本批考试日期（必填，整批统一，YYYY-MM-DD）
	Rows     []ImportScoreRow `json:"rows"`
}

// ImportScoresResult 成绩单导入结果（供前端展示"追加 X 人 Y 条 / 新建 Z 人 / 更新画像 W 人"）
type ImportScoresResult struct {
	TotalRows         int      `json:"total_rows"`          // 收到的行数（不含前端已滤掉的空行）
	AppendedScores    int      `json:"appended_scores"`     // 实际写入/更新的成绩条数
	AffectedStudents  int      `json:"affected_students"`   // 被追加成绩的学生数（含新建的）
	CreatedStudents   int      `json:"created_students"`    // 因学号不存在而自动新建的学生数
	ProfileUpdated    int      `json:"profile_updated"`     // 因薄弱点/备注非空而更新了画像的学生数（2b-2 新增）
	SkippedRows       int      `json:"skipped_rows"`        // 因数据非法被跳过的行数
	Errors            []string `json:"errors"`              // 被跳过行的原因（截断展示，最多若干条）
}

// ---------- 校验辅助 ----------

// IsValidStudentTier 校验分层标签（空串=未分层，也算合法）
func IsValidStudentTier(t string) bool {
	return t == StudentTierNone || t == StudentTierA || t == StudentTierB || t == StudentTierC
}

// HasProfileContent 判断学情卡四大段是否已有任何内容（供列表显示"已填/待完善"）
func (p *ClassProfile) HasProfileContent() bool {
	return p.OverallProfile != "" || p.TierStructure != "" ||
		p.WeakPoints != "" || p.TeachingAdvice != ""
}

// ---------- 学号代号自动编号辅助（批次2a 新增）----------

// NextStudentCode 根据该班已有学号代号列表，生成下一个自动编号（两位补零："01"/"02"…）。
//
// 规则：扫描所有已有学号代号，提取其中"纯数字"的那些（如 "01"/"7"/"012"），
// 取其最大值 +1，补零到至少两位返回。若现有学号都不是纯数字（如 "S001"），
// 则从 1 开始（返回 "01"）。
//
// 这样设计的原因：老师可能手填带前缀的学号（S001），自动编号只负责"纯数字序列"这一种，
// 与手填学号互不干扰；重复由 repo 层 UNIQUE(class_profile_id,student_code) 兜底。
func NextStudentCode(existing []string) string {
	maxN := 0
	for _, code := range existing {
		c := strings.TrimSpace(code)
		if c == "" {
			continue
		}
		// 仅当整个学号都是数字时才纳入序列计算
		if n, err := strconv.Atoi(c); err == nil && n > maxN {
			maxN = n
		}
	}
	next := maxN + 1
	// 补零到至少两位
	return fmt.Sprintf("%02d", next)
}

// ---------- 成绩归并辅助（批次2b 新增）----------

// MergeScoreInto 把一条新成绩归并进现有成绩数组，返回归并后的数组。
//
// 去重键 = 考试名称 + 考试日期（name+at）：
//   - 命中已有条目：更新该条的分数与满分（覆盖，便于改错分）。
//   - 未命中：追加为新条目。
// 不在此处排序——数组顺序保持"录入先后"，latest_score 由 LatestScoreOf 单独按日期算。
func MergeScoreInto(existing []ClassStudentScore, incoming ClassStudentScore) []ClassStudentScore {
	for i := range existing {
		if existing[i].Name == incoming.Name && existing[i].At == incoming.At {
			existing[i].Score = incoming.Score
			existing[i].Max = incoming.Max
			return existing
		}
	}
	return append(existing, incoming)
}

// LatestScoreOf 从成绩数组里取"考试日期最新"那一条的分数（用于回填 latest_score）。
//
// 日期为 YYYY-MM-DD 字符串，字典序即时间序，故直接字符串比较取最大 At 的那条。
// 空数组返回 nil。若多条同为最新日期，取数组中靠后出现的一条（通常为最后录入）。
func LatestScoreOf(arr []ClassStudentScore) *float64 {
	if len(arr) == 0 {
		return nil
	}
	bestIdx := 0
	for i := 1; i < len(arr); i++ {
		// >= 使同日期时取靠后的一条
		if arr[i].At >= arr[bestIdx].At {
			bestIdx = i
		}
	}
	v := arr[bestIdx].Score
	return &v
}
