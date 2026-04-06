package models

// teaching_profile.go — 教学风格前测数据模型
//
// 迭代3新增：
//   - TeachingProfile：前测结果结构体，存入 users.teaching_profile JSONB
//   - 前测对话请求/响应类型
//   - 教学风格/AI协作偏好 枚举常量
//   - 判定结果结构体
//   - 自动配方生成请求/响应

import "time"

// ==================== 前测结果（存入 users.teaching_profile JSONB）====================

// TeachingProfile 教学风格前测结果
// 对应 PRD §八 users.teaching_profile JSONB 格式
type TeachingProfile struct {
	// 版本信息
	AssessmentVersion int        `json:"assessment_version"` // 前测版本号（当前=1）
	AssessedAt        *time.Time `json:"assessed_at"`        // 前测完成时间

	// 教师基本信息（Q1采集）
	ExperienceYears int    `json:"experience_years"` // 教龄
	SubjectPrimary  string `json:"subject_primary"`  // 主要学科
	GradePrimary    string `json:"grade_primary"`    // 主要年级

	// 风格判定结果
	TeachingStyle   string `json:"teaching_style"`   // 教学风格：mature/growing/beginner
	AICollaboration string `json:"ai_collaboration"` // AI协作偏好：tool/collaborative/guided

	// 细节偏好（Q5采集）
	Priorities []string `json:"priorities"` // 质量关注点（多选）

	// 系统推荐
	RecommendedMode   string   `json:"recommended_mode"`   // 推荐备课模式：guided/efficient
	RecommendedStages []string `json:"recommended_stages"` // 推荐阶段列表

	// 教案结构偏好（Q6采集，自然语言）
	LessonStructureDesc string `json:"lesson_structure_desc"`

	// 原始对话记录（用于审计和改进）
	ConversationLog []AssessmentMessage `json:"conversation_log"`
}

// AssessmentMessage 前测对话消息
type AssessmentMessage struct {
	Role      string `json:"role"`       // user / assistant
	Content   string `json:"content"`    // 消息内容
	Timestamp string `json:"timestamp"`  // ISO时间戳
	StepCode  string `json:"step_code"`  // 对应问题编号：q1/q2/q3/q4/q5/q6/summary
}

// ==================== 教学风格常量 ====================

// 教学经验与风格（维度A）
const (
	StyleMature  = "mature"  // 成熟型（8年以上）
	StyleGrowing = "growing" // 成长型（3-8年）
	StyleBeginner = "beginner" // 新手型（3年以下）
)

// AI协作偏好（维度B）
const (
	CollabTool         = "tool"          // 工具型（"你帮我做完，我来改"）
	CollabCollaborative = "collaborative" // 协作型（"我们一起想"）
	CollabGuided       = "guided"        // 引导型（"你带着我走"）
)

// 质量关注点选项
var AssessmentPriorityOptions = []string{
	"activity_detail",    // 活动设计细节
	"student_response",   // 学生预期反应
	"time_allocation",    // 时间分配合理性
	"knowledge_accuracy", // 知识点准确性
	"differentiation",    // 分层教学设计
	"assessment_design",  // 评估方式设计
	"resource_alignment", // 资源与目标对齐
	"innovation",         // 教学创新性
}

// ==================== 前测API请求/响应 ====================

// AssessmentStartRequest 开始前测请求
type AssessmentStartRequest struct {
	// 无需参数，后端根据当前用户初始化
}

// AssessmentStartResponse 开始前测响应
type AssessmentStartResponse struct {
	SessionID      string              `json:"session_id"`      // 会话ID（用教案ID复用）
	OpeningMessage AssessmentMessage   `json:"opening_message"` // AI开场白
	TotalSteps     int                 `json:"total_steps"`     // 总问题数（6）
	CurrentStep    string              `json:"current_step"`    // 当前步骤：q1
}

// AssessmentChatRequest 前测聊天请求
type AssessmentChatRequest struct {
	Message string `json:"message"` // 用户回复内容
}

// AssessmentChatResponse 前测聊天响应
type AssessmentChatResponse struct {
	AIMessage   AssessmentMessage `json:"ai_message"`    // AI回复
	CurrentStep string            `json:"current_step"`  // 当前步骤
	IsComplete  bool              `json:"is_complete"`   // 是否已完成所有问题
	Progress    int               `json:"progress"`      // 进度百分比（0-100）
}

// AssessmentSubmitRequest 提交前测结果请求
// AI输出 <assessment_result> JSON后，前端解析并提交
type AssessmentSubmitRequest struct {
	// AI判定的原始结果
	ExperienceYears     int      `json:"experience_years"`
	SubjectPrimary      string   `json:"subject_primary"`
	GradePrimary        string   `json:"grade_primary"`
	TeachingStyle       string   `json:"teaching_style"`
	AICollaboration     string   `json:"ai_collaboration"`
	Priorities          []string `json:"priorities"`
	LessonStructureDesc string   `json:"lesson_structure_desc"`
}

// AssessmentSubmitResponse 提交前测结果响应
type AssessmentSubmitResponse struct {
	Profile  *TeachingProfile `json:"profile"`   // 最终存储的画像
	RecipeID string           `json:"recipe_id"` // 自动生成的配方ID（如果生成了）
}

// AssessmentResultResponse 获取前测结果响应
type AssessmentResultResponse struct {
	HasProfile bool             `json:"has_profile"` // 是否已完成前测
	Profile    *TeachingProfile `json:"profile"`     // 前测结果（可能为nil）
}

// AutoRecipeRequest 自动生成配方请求
type AutoRecipeRequest struct {
	// 无需额外参数，从当前用户的teaching_profile读取
}

// AutoRecipeResponse 自动生成配方响应
type AutoRecipeResponse struct {
	RecipeID   string `json:"recipe_id"`   // 生成的配方ID
	RecipeName string `json:"recipe_name"` // 生成的配方名称
}

// ==================== 风格→推荐 映射规则 ====================

// StyleModeMap 教学风格→推荐备课模式映射
// 规则兜底：新手不能被推荐为efficient
var StyleModeMap = map[string]string{
	StyleMature:   "efficient",
	StyleGrowing:  "guided",
	StyleBeginner: "guided",
}

// CollabModeOverride AI协作偏好对推荐模式的覆盖
// tool型无论经验都倾向efficient，guided型无论经验都倾向guided
var CollabModeOverride = map[string]string{
	CollabTool:   "efficient",
	CollabGuided: "guided",
	// collaborative：不覆盖，使用StyleModeMap的结果
}

// StyleStagesMap 教学风格→推荐阶段列表映射
var StyleStagesMap = map[string][]string{
	StyleMature:   {"analyze", "write", "revise"},                        // 老手：跳过设计和评审
	StyleGrowing:  {"analyze", "design", "write", "review", "revise"},    // 成长：完整5步
	StyleBeginner: {"analyze", "design", "write", "review", "revise"},    // 新手：完整5步
}
