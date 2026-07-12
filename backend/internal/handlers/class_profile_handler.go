package handlers

// class_profile_handler.go — 班级学情 HTTP 处理器（差异化教学·老师私有资料，独立模块）
//
// 批次1 路由（班级卡，全部纯个人，登录即可，归属校验在 service 层）：
//   GET    /api/v1/class-profiles        列出我的班级学情卡 → {profiles,total}
//   POST   /api/v1/class-profiles        新建一张卡（手写入口）→ {profile}
//   GET    /api/v1/class-profiles/{id}   卡片详情（含四大段）→ {profile}
//   PUT    /api/v1/class-profiles/{id}   更新卡片 → {updated:true}
//   DELETE /api/v1/class-profiles/{id}   软删除 → {deleted:true}
//
// 批次2a 路由（学生个体档案，挂在班级卡 {id} 下，归属判断走"班级卡归属"）：
//   GET    /api/v1/class-profiles/{id}/students         列出该班学生 → {students,total}
//   POST   /api/v1/class-profiles/{id}/students         新建学生（学号留空自动编号）→ {student}
//   PUT    /api/v1/class-profiles/{id}/students/{sid}   更新学生 → {student}
//   DELETE /api/v1/class-profiles/{id}/students/{sid}   删除学生 → {deleted:true}
//
// 批次2b 路由（成绩单导入，挂在学生集合下的固定子路径 import）：
//   POST   /api/v1/class-profiles/{id}/students/import  导入成绩长表 → {result}
//
// 批次2c 路由（AI 总结学情，挂在学生集合下的固定子路径 summarize）：
//   POST   /api/v1/class-profiles/{id}/students/summarize  让 AI 基于匿名统计量生成四大段 → {result}
//
// 批次2d 路由（按分数线自动分层，挂在学生集合下的固定子路径 auto-tier）：
//   POST   /api/v1/class-profiles/{id}/students/auto-tier  按两条分数线把有成绩学生归 ABC 层 → {result}
//     注意：import / summarize / auto-tier 同为固定关键字，路径形态与 .../students/{sid} 都是三段，
//     故在三段分支里必须"先判固定关键字"再落到 {sid} 的 PUT/DELETE，否则会被错当成 studentID。
//     用精确字符串相等判断，UUID 不会恰好等于这些关键字，故不冲突。
//
// 路由分发说明：因为 routes 里 /api/v1/class-profiles/ 这条已吃掉所有带后缀的路径，
// 学生档案/导入/总结/分层端点必须在 HandleItem 内按路径形态分发——
// 把 class-profiles/ 后的 rest 按 "/" 切段：
//   ["{id}"]                          → 班级卡详情/更新/删除（原有）
//   ["{id}","students"]               → 学生列表 / 新建
//   ["{id}","students","{sid}"]       → 学生更新 / 删除
//   ["{id}","students","import"]      → 成绩单导入（批次2b）
//   ["{id}","students","summarize"]   → AI 总结学情（批次2c）
//   ["{id}","students","auto-tier"]   → 按分数线自动分层（批次2d）
//
// 合规红线：学生档案只承载学号代号/分层/薄弱点/备注；成绩走批次2b 导入，由后端归并。
// AI 总结只把"匿名群体统计量"喂 AI，学生个体明细（含学号代号）永不进 AI 链路。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

type ClassProfileHandler struct {
	svc *services.ClassProfileService
	cfg *config.Config // 批次2c 新增：AI 总结需要取 AES 密钥 + 兜底模型/网关
}

// NewClassProfileHandler 构造（批次2c 起持 cfg，供 SummarizeClassProfile 调 AI）
func NewClassProfileHandler(svc *services.ClassProfileService, cfg *config.Config) *ClassProfileHandler {
	return &ClassProfileHandler{svc: svc, cfg: cfg}
}

const classProfilePathPrefix = "/api/v1/class-profiles"

// HandleCollection 处理 /api/v1/class-profiles（GET 列表 / POST 新建）
func (h *ClassProfileHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := h.svc.ListProfiles(r.Context(), claims.UserID)
		if err != nil {
			utils.InternalError(w, err.Error())
			return
		}
		utils.Success(w, map[string]interface{}{"profiles": items, "total": len(items)})
	case http.MethodPost:
		var req models.CreateClassProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "请求体解析失败")
			return
		}
		p, err := h.svc.CreateProfile(r.Context(), claims.UserID, &req)
		if err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"profile": p})
	default:
		utils.BadRequest(w, "不支持的方法")
	}
}

// HandleItem 处理 /api/v1/class-profiles/{id} 及其下的学生档案子路径
func (h *ClassProfileHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, classProfilePathPrefix+"/"), "/")
	if rest == "" {
		utils.BadRequest(w, "缺少ID")
		return
	}
	// 按 "/" 切段决定走班级卡还是学生档案
	segs := strings.Split(rest, "/")

	// ["{id}","students",...] → 学生档案/导入/总结/分层分发
	if len(segs) >= 2 && segs[1] == "students" {
		h.handleStudents(w, r, claims.UserID, segs)
		return
	}

	// 否则按原班级卡 {id} 处理（只接受单段，多余段视为非法路径）
	if len(segs) != 1 {
		utils.BadRequest(w, "路径不合法")
		return
	}
	id := segs[0]

	switch r.Method {
	case http.MethodGet:
		p, err := h.svc.GetProfile(r.Context(), claims.UserID, id)
		if err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"profile": p})
	case http.MethodPut:
		var req models.UpdateClassProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "请求体解析失败")
			return
		}
		if err := h.svc.UpdateProfile(r.Context(), claims.UserID, id, &req); err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"updated": true})
	case http.MethodDelete:
		if err := h.svc.DeleteProfile(r.Context(), claims.UserID, id); err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"deleted": true})
	default:
		utils.BadRequest(w, "不支持的方法")
	}
}

// handleStudents 处理学生档案/导入/总结/分层子路径
//
// segs 形态：
//   ["{id}","students"]               → 学生列表(GET) / 新建(POST)
//   ["{id}","students","import"]      → 成绩单导入(POST)（批次2b，必须先于 {sid} 判定）
//   ["{id}","students","summarize"]   → AI 总结学情(POST)（批次2c，必须先于 {sid} 判定）
//   ["{id}","students","auto-tier"]   → 按分数线自动分层(POST)（批次2d，必须先于 {sid} 判定）
//   ["{id}","students","{sid}"]       → 学生更新(PUT) / 删除(DELETE)
func (h *ClassProfileHandler) handleStudents(w http.ResponseWriter, r *http.Request, userID string, segs []string) {
	classProfileID := segs[0]
	if classProfileID == "" {
		utils.BadRequest(w, "缺少班级ID")
		return
	}

	// 集合路径 .../students（无第三段）：GET 列表 / POST 新建
	if len(segs) == 2 {
		switch r.Method {
		case http.MethodGet:
			students, err := h.svc.ListStudents(r.Context(), userID, classProfileID)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"students": students, "total": len(students)})
		case http.MethodPost:
			var req models.CreateClassStudentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				utils.BadRequest(w, "请求体解析失败")
				return
			}
			view, err := h.svc.CreateStudent(r.Context(), userID, classProfileID, &req)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"student": view})
		default:
			utils.BadRequest(w, "不支持的方法")
		}
		return
	}

	// 三段路径：先判固定关键字 import / summarize / auto-tier，再落到 {sid}
	if len(segs) == 3 {
		third := segs[2]

		// .../students/import → 成绩单导入（批次2b）
		if third == "import" {
			if r.Method != http.MethodPost {
				utils.BadRequest(w, "不支持的方法")
				return
			}
			var req models.ImportScoresRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				utils.BadRequest(w, "请求体解析失败")
				return
			}
			result, err := h.svc.ImportScores(r.Context(), userID, classProfileID, &req)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"result": result})
			return
		}

		// .../students/summarize → AI 总结学情（批次2c，只生成不落库）
		if third == "summarize" {
			if r.Method != http.MethodPost {
				utils.BadRequest(w, "不支持的方法")
				return
			}
			result, err := h.svc.SummarizeClassProfile(r.Context(), h.cfg, userID, classProfileID)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"result": result})
			return
		}

		// .../students/auto-tier → 按分数线自动分层（批次2d）
		if third == "auto-tier" {
			if r.Method != http.MethodPost {
				utils.BadRequest(w, "不支持的方法")
				return
			}
			var req services.AutoTierRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				utils.BadRequest(w, "请求体解析失败")
				return
			}
			result, err := h.svc.AutoTierStudents(r.Context(), userID, classProfileID, &req)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"result": result})
			return
		}

		// .../students/{sid} → 学生更新 / 删除
		studentID := third
		if studentID == "" {
			utils.BadRequest(w, "缺少学生ID")
			return
		}
		switch r.Method {
		case http.MethodPut:
			var req models.UpdateClassStudentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				utils.BadRequest(w, "请求体解析失败")
				return
			}
			view, err := h.svc.UpdateStudent(r.Context(), userID, classProfileID, studentID, &req)
			if err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"student": view})
		case http.MethodDelete:
			if err := h.svc.DeleteStudent(r.Context(), userID, classProfileID, studentID); err != nil {
				h.mapError(w, err)
				return
			}
			utils.Success(w, map[string]interface{}{"deleted": true})
		default:
			utils.BadRequest(w, "不支持的方法")
		}
		return
	}

	utils.BadRequest(w, "路径不合法")
}

// mapError 把 service/repository 的 sentinel 错误映射为 HTTP 状态码
func (h *ClassProfileHandler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrClassProfileNotFound),
		errors.Is(err, repository.ErrClassStudentNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrClassProfileFieldRequired),
		errors.Is(err, services.ErrStudentCodeRequired),
		errors.Is(err, services.ErrStudentTierInvalid),
		errors.Is(err, services.ErrStudentCodeDup),
		errors.Is(err, services.ErrClassSummaryNoData),
		errors.Is(err, services.ErrAutoTierLineInvalid): // 批次2d：分数线非法 → 400
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrClassProfileNotOwner):
		utils.Forbidden(w, err.Error())
	case errors.Is(err, services.ErrClassSummaryAIFailed):
		utils.InternalError(w, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}
