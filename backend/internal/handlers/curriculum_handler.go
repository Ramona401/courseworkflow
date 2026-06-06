package handlers

// curriculum_handler.go — 课程知识库公共只读查询处理器
//
// 平台级公共接口（/api/v1/curriculum/*），学科无关、场景无关：
//   课件工坊「从主题创建」查它做难度适配；
//   备课工坊「教案撰写」将来同样复用同一批接口取知识点。
// 因此本 handler 不依赖任何 courseware/lesson_plan 服务，只读 curriculum_repo。
//
// 提供接口：
//   GET /api/v1/curriculum/knowledge-points?subject=数学&grade=3   — 按学科+年级查知识点清单（供勾选+难度适配）
//   GET /api/v1/curriculum/textbook-units?subject=数学&publisher=人教版&grade=3&semester=上册 — 查教材单元
//   GET /api/v1/curriculum/publishers?subject=数学&grade=3         — 查某学科某年级有哪些教材版本

import (
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// CurriculumHandler 课程知识库只读查询处理器
type CurriculumHandler struct{}

// NewCurriculumHandler 创建课程知识库处理器
func NewCurriculumHandler() *CurriculumHandler {
	return &CurriculumHandler{}
}

// ListKnowledgePoints GET /api/v1/curriculum/knowledge-points
// 按学科+年级查询课标知识点清单。
// query: subject(必填) grade(年级数字1-9,可选;不传则返回该学科全部年级)
// 返回按 domain 分组排序的知识点数组，供前端勾选 + 后续难度适配。
func (h *CurriculumHandler) ListKnowledgePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	// 登录即可访问（公共数据，但仍要求登录态，避免裸暴露）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := r.URL.Query().Get("subject")
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	gradeNum, _ := strconv.Atoi(r.URL.Query().Get("grade"))

	kps, err := repository.ListCurriculumKPsBySubjectGrade(r.Context(), subject, gradeNum)
	if err != nil {
		utils.InternalError(w, "查询知识点失败: "+err.Error())
		return
	}
	if kps == nil {
		kps = nil // 保持 nil → JSON 序列化为 null；前端按空处理
	}
	utils.Success(w, map[string]interface{}{
		"subject":          subject,
		"grade":            gradeNum,
		"knowledge_points": kps,
		"total":            len(kps),
	})
}

// ListTextbookUnits GET /api/v1/curriculum/textbook-units
// 按学科+版本+年级+册查询教材单元。
// query: subject(必填) publisher(可选) grade(可选) semester(可选)
func (h *CurriculumHandler) ListTextbookUnits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := r.URL.Query().Get("subject")
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	publisher := r.URL.Query().Get("publisher")
	semester := r.URL.Query().Get("semester")
	gradeNum, _ := strconv.Atoi(r.URL.Query().Get("grade"))

	units, err := repository.ListTextbookUnits(r.Context(), subject, publisher, gradeNum, semester)
	if err != nil {
		utils.InternalError(w, "查询教材单元失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"subject":   subject,
		"publisher": publisher,
		"grade":     gradeNum,
		"semester":  semester,
		"units":     units,
		"total":     len(units),
	})
}

// ListPublishers GET /api/v1/curriculum/publishers
// 查询某学科某年级下有哪些教材版本（供前端版本下拉）。
// query: subject(必填) grade(必填)
func (h *CurriculumHandler) ListPublishers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := r.URL.Query().Get("subject")
	gradeNum, _ := strconv.Atoi(r.URL.Query().Get("grade"))
	if subject == "" || gradeNum <= 0 {
		utils.BadRequest(w, "缺少 subject 或 grade 参数")
		return
	}

	publishers, err := repository.ListTextbookPublishers(r.Context(), subject, gradeNum)
	if err != nil {
		utils.InternalError(w, "查询教材版本失败: "+err.Error())
		return
	}
	if publishers == nil {
		publishers = []string{}
	}
	utils.Success(w, map[string]interface{}{
		"subject":    subject,
		"grade":      gradeNum,
		"publishers": publishers,
	})
}
