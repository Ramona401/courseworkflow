package handlers

// subject_handler.go — 统一课程定义与教育域课程目录处理器
//
// 公开接口：
//   GET /api/v1/subjects
//   根据当前登录用户的教育域和教学组织，返回：
//     - 当前教育域公共课程；
//     - 当前学校专属课程。
//   vocational和adult目录为空时绝不回退到K12。
//
// 管理接口：
//   GET    /api/v1/admin/subjects
//   POST   /api/v1/admin/subjects
//   PUT    /api/v1/admin/subjects/{id}
//   DELETE /api/v1/admin/subjects/{id}
//
// 管理端新增课程时，必须同时提交至少一条课程目录配置。
// 后端在同一事务中写入subjects和subject_catalog_entries，
// 防止再次产生“课程定义已创建，但教师下拉不可见”的孤立数据。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

/* ==================== 处理器构造 ==================== */

type SubjectHandler struct{}

func NewSubjectHandler() *SubjectHandler {
	return &SubjectHandler{}
}

/* ==================== 公开课程目录 ==================== */

// ListPublic 按当前用户教育域和教学组织返回启用课程。
func (h *SubjectHandler) ListPublic(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	educationContext, err :=
		repository.ResolveUserEducationContext(
			r.Context(),
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		utils.InternalError(
			w,
			"解析教育域失败: "+err.Error(),
		)
		return
	}

	items, err :=
		repository.ListActiveSubjectsForEducationContext(
			r.Context(),
			educationContext.EducationDomain,
			educationContext.OrganizationID,
		)
	if err != nil {
		utils.InternalError(
			w,
			"查询课程列表失败: "+err.Error(),
		)
		return
	}

	if items == nil {
		items = []*models.Subject{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"subjects":         items,
			"total":            len(items),
			"education_domain": educationContext.EducationDomain,
			"education_org_id": educationContext.OrganizationID,
		},
	)
}

/* ==================== 后台管理列表 ==================== */

// ListAdmin 返回全部课程定义及其教育域、适用学校目录配置。
func (h *SubjectHandler) ListAdmin(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	items, err :=
		repository.ListAllSubjects(
			r.Context(),
		)
	if err != nil {
		utils.InternalError(
			w,
			"查询课程列表失败: "+err.Error(),
		)
		return
	}

	if items == nil {
		items =
			[]*models.SubjectAdminItem{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"subjects": items,
			"total":    len(items),
		},
	)
}

/* ==================== 新建课程 ==================== */

// Create 在一个事务中创建课程定义和至少一条课程目录配置。
func (h *SubjectHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var req models.CreateSubjectRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Note = strings.TrimSpace(req.Note)

	if req.Name == "" {
		utils.BadRequest(
			w,
			"缺少课程名称",
		)
		return
	}

	if len(req.CatalogEntries) == 0 {
		utils.BadRequest(
			w,
			"请至少配置一个教育域或适用学校",
		)
		return
	}

	item, err := repository.CreateSubject(
		r.Context(),
		&req,
		claims.UserID,
	)
	if err != nil {
		writeSubjectRepositoryError(
			w,
			err,
			"新建课程失败",
		)
		return
	}

	utils.Success(w, item)
}

/* ==================== 编辑课程 ==================== */

// Update 编辑课程定义，并在提交catalog_entries时完整替换目录配置。
//
// 行内启停只提交is_active，不携带catalog_entries，
// 因此不会误删当前教育域和学校归属。
func (h *SubjectHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	id := extractSubjectID(r.URL.Path)
	if id == "" {
		utils.BadRequest(
			w,
			"缺少课程ID",
		)
		return
	}

	var req models.UpdateSubjectRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if req.Name != nil {
		trimmed :=
			strings.TrimSpace(*req.Name)
		if trimmed == "" {
			utils.BadRequest(
				w,
				"课程名称不能为空",
			)
			return
		}
		req.Name = &trimmed
	}

	if req.Code != nil {
		trimmed :=
			strings.TrimSpace(*req.Code)
		req.Code = &trimmed
	}

	if req.Note != nil {
		trimmed :=
			strings.TrimSpace(*req.Note)
		req.Note = &trimmed
	}

	item, err := repository.UpdateSubject(
		r.Context(),
		id,
		&req,
		claims.UserID,
	)
	if err != nil {
		writeSubjectRepositoryError(
			w,
			err,
			"编辑课程失败",
		)
		return
	}

	utils.Success(w, item)
}

/* ==================== 删除课程 ==================== */

// Delete 删除非内置课程定义。
//
// 数据库外键会级联清理该课程的全部目录配置。
// 历史教案和课件使用课程名称快照，不受删除影响。
func (h *SubjectHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	id := extractSubjectID(r.URL.Path)
	if id == "" {
		utils.BadRequest(
			w,
			"缺少课程ID",
		)
		return
	}

	if err := repository.DeleteSubject(
		r.Context(),
		id,
	); err != nil {
		writeSubjectRepositoryError(
			w,
			err,
			"删除课程失败",
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "已删除",
		},
	)
}

/* ==================== 错误映射 ==================== */

// writeSubjectRepositoryError 将课程仓储的确定性业务错误转换为HTTP响应。
//
// 参数错误使用400；资源不存在使用404；
// 未识别的数据库或系统错误统一使用500。
func writeSubjectRepositoryError(
	w http.ResponseWriter,
	err error,
	action string,
) {
	switch err {
	case repository.ErrSubjectNotFound:
		utils.Fail(
			w,
			http.StatusNotFound,
			"课程不存在",
		)

	case repository.ErrSubjectNameExists:
		utils.BadRequest(
			w,
			"课程名称已存在",
		)

	case repository.ErrSubjectSystemGuard:
		utils.BadRequest(
			w,
			"内置课程不可删除，如需隐藏请改为停用",
		)

	case repository.ErrSubjectCatalogRequired:
		utils.BadRequest(
			w,
			"请至少配置一个教育域或适用学校",
		)

	case repository.ErrSubjectCatalogInvalidDomain:
		utils.BadRequest(
			w,
			"课程目录教育域无效",
		)

	case repository.ErrSubjectCatalogDuplicate:
		utils.BadRequest(
			w,
			"同一教育域或学校存在重复课程配置",
		)

	case repository.ErrSubjectCatalogOrganizationNotFound:
		utils.BadRequest(
			w,
			"指定学校不存在或已经停用",
		)

	case repository.ErrSubjectCatalogOrganizationMismatch:
		utils.BadRequest(
			w,
			"指定学校的教育域与课程目录教育域不一致",
		)

	default:
		utils.InternalError(
			w,
			action+": "+err.Error(),
		)
	}
}

/* ==================== 路径解析 ==================== */

// extractSubjectID 从管理路由末尾提取课程ID。
func extractSubjectID(
	path string,
) string {
	normalized :=
		strings.TrimRight(path, "/")

	index :=
		strings.LastIndex(
			normalized,
			"/",
		)

	if index < 0 ||
		index == len(normalized)-1 {
		return ""
	}

	return normalized[index+1:]
}
