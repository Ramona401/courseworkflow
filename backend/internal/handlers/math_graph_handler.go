package handlers

// math_graph_handler.go — 数学图形/学科实验/力学场景 AI 生成处理器
//
// 端点：POST /api/v1/math-graph/generate
//
// target:
//   - 空 / math_graph：数学 JSXGraph 构造代码
//   - physics_lab：物理实验 HTML 组件
//   - chem_experiment：化学实验 HTML 组件
//   - biology_lab：生命科学互动 HTML 组件
//   - geography_lab：地理互动探究 HTML 组件
//   - physics_scene：Matter.js 力学场景 setup 构造代码

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/services"
	"tedna/internal/utils"
)

var mathGraphHandlerLog = logger.WithModule("math_graph_handler")

type MathGraphHandler struct {
	svc *services.MathGraphAIService
}

func NewMathGraphHandler(svc *services.MathGraphAIService) *MathGraphHandler {
	return &MathGraphHandler{svc: svc}
}

type mathGraphGenerateRequest struct {
	Target       string `json:"target"`
	Mode         string `json:"mode"`
	Description  string `json:"description"`
	BaseCode     string `json:"base_code"`
	TemplateName string `json:"template_name"`
	BoundingBox  string `json:"bounding_box"`
	Image        string `json:"image"`
}

type mathGraphGenerateResponse struct {
	Code string `json:"code"`
}

func (h *MathGraphHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req mathGraphGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		target = "math_graph"
	}

	if strings.TrimSpace(req.Description) == "" &&
		strings.TrimSpace(req.Image) == "" {
		switch target {
		case "math_graph":
			utils.BadRequest(w, "请输入图形描述或上传题目图片")
		case "physics_scene":
			utils.BadRequest(w, "请输入力学场景描述或上传题目图片")
		case "geography_lab":
			utils.BadRequest(w, "请输入地理探究描述或上传地图、图表或教材图片")
		default:
			utils.BadRequest(w, "请输入实验描述或上传题目/装置图片")
		}
		return
	}

	var (
		code string
		err  error
	)

	switch target {
	case "math_graph":
		code, err = h.svc.Generate(
			r.Context(),
			userID,
			&services.MathGraphGenInput{
				Mode:         req.Mode,
				Description:  req.Description,
				BaseCode:     req.BaseCode,
				TemplateName: req.TemplateName,
				BoundingBox:  req.BoundingBox,
				Image:        req.Image,
			},
		)

	case "physics_lab", "chem_experiment", "biology_lab":
		code, err = h.svc.GenerateSubjectExperiment(
			r.Context(),
			userID,
			&services.SubjectExperimentGenInput{
				Target:       target,
				Mode:         req.Mode,
				Description:  req.Description,
				BaseCode:     req.BaseCode,
				TemplateName: req.TemplateName,
				Image:        req.Image,
			},
		)

	case "geography_lab":
		code, err = h.svc.GenerateGeographyExperiment(
			r.Context(),
			userID,
			&services.SubjectExperimentGenInput{
				Target:       target,
				Mode:         req.Mode,
				Description:  req.Description,
				BaseCode:     req.BaseCode,
				TemplateName: req.TemplateName,
				Image:        req.Image,
			},
		)

	case "physics_scene":
		code, err = h.svc.GeneratePhysicsScene(
			r.Context(),
			userID,
			&services.PhysicsSceneGenInput{
				Mode:         req.Mode,
				Description:  req.Description,
				BaseCode:     req.BaseCode,
				TemplateName: req.TemplateName,
				Image:        req.Image,
			},
		)

	default:
		utils.BadRequest(w, "不支持的生成目标 target: "+target)
		return
	}

	if err != nil {
		mathGraphHandlerLog.Error(
			"AI组件生成失败",
			"user", userID,
			"target", target,
			"mode", req.Mode,
			"error", err,
		)
		utils.InternalError(w, "AI组件生成失败: "+err.Error())
		return
	}

	utils.Success(w, &mathGraphGenerateResponse{Code: code})
}
