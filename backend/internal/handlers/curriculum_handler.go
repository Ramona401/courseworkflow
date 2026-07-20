package handlers

// curriculum_handler.go — K12课程知识库只读查询处理器
//
// 课程知识点、教材单元和出版社数据目前都是K12专属基础数据。
//
// 安全规则：
//   1. JWT只提供用户ID，不作为教育域真相源；
//   2. 每次请求实时读取users.role；
//   3. 使用教案创建严格解析器取得唯一具体教学域；
//   4. 只有K12继续查询数据库；
//   5. vocational、adult、mixed、空值、非法值和归属冲突返回成功空数组；
//   6. 数据库或基础设施错误返回5xx，不伪装成空数据。
//
// Repository还会再次校验显式educationDomain参数，
// 防止其它内部调用绕过Handler。

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// curriculumHandlerDeps 是课程知识库查询的最小依赖集合。
type curriculumHandlerDeps struct {
	findUser func(
		ctx context.Context,
		userID string,
	) (*models.User, error)

	resolveEducationDomain func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error)

	listKnowledgePoints func(
		ctx context.Context,
		educationDomain string,
		subject string,
		gradeNum int,
	) ([]*models.CurriculumKP, error)

	listTextbookUnits func(
		ctx context.Context,
		educationDomain string,
		subject string,
		publisher string,
		gradeNum int,
		semester string,
	) ([]*models.TextbookUnit, error)

	listPublishers func(
		ctx context.Context,
		educationDomain string,
		subject string,
		gradeNum int,
	) ([]string, error)
}

// CurriculumHandler 课程知识库只读查询处理器。
type CurriculumHandler struct {
	deps curriculumHandlerDeps
}

// NewCurriculumHandler 创建课程知识库处理器。
func NewCurriculumHandler() *CurriculumHandler {
	return &CurriculumHandler{
		deps: curriculumHandlerDeps{
			findUser: repository.FindUserByID,
			resolveEducationDomain: repository.
				ResolveLessonPlanCreationEducationDomain,
			listKnowledgePoints: repository.
				ListCurriculumKPsBySubjectGrade,
			listTextbookUnits: repository.
				ListTextbookUnits,
			listPublishers: repository.
				ListTextbookPublishers,
		},
	}
}

// resolveCurriculumEducationDomain 实时解析当前请求的可信教育域。
//
// 无有效教学域、域冲突或区域管理员任命未就绪属于安全空结果；
// 数据库和其它基础设施错误必须向上传递。
func (h *CurriculumHandler) resolveCurriculumEducationDomain(
	ctx context.Context,
	userID string,
) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", nil
	}

	user, err := h.deps.findUser(ctx, userID)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserNotFound,
		) {
			return "", nil
		}

		return "", fmt.Errorf(
			"读取课程知识库查询用户失败: %w",
			err,
		)
	}
	if user == nil ||
		strings.TrimSpace(user.Role) == "" {
		return "", nil
	}

	domain, err := h.deps.resolveEducationDomain(
		ctx,
		userID,
		user.Role,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.ErrLessonPlanCreationEducationDomainConflict,
			),
			errors.Is(
				err,
				repository.ErrRegionAdminEducationDomainNotReady,
			):
			return "", nil

		default:
			return "", fmt.Errorf(
				"解析课程知识库查询教育域失败: %w",
				err,
			)
		}
	}

	domain = strings.ToLower(
		strings.TrimSpace(domain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", nil
	}

	return domain, nil
}

// ListKnowledgePoints GET /api/v1/curriculum/knowledge-points
func (h *CurriculumHandler) ListKnowledgePoints(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := strings.TrimSpace(
		r.URL.Query().Get("subject"),
	)
	if subject == "" {
		utils.BadRequest(
			w,
			"缺少 subject 参数",
		)
		return
	}

	gradeNum, _ := strconv.Atoi(
		r.URL.Query().Get("grade"),
	)

	domain, err := h.resolveCurriculumEducationDomain(
		r.Context(),
		claims.UserID,
	)
	if err != nil {
		utils.InternalError(
			w,
			"查询知识点失败",
		)
		return
	}

	knowledgePoints := []*models.CurriculumKP{}

	if domain == models.EducationDomainK12 {
		knowledgePoints, err =
			h.deps.listKnowledgePoints(
				r.Context(),
				domain,
				subject,
				gradeNum,
			)
		if err != nil {
			utils.InternalError(
				w,
				"查询知识点失败",
			)
			return
		}
		if knowledgePoints == nil {
			knowledgePoints =
				[]*models.CurriculumKP{}
		}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"subject":          subject,
			"grade":            gradeNum,
			"knowledge_points": knowledgePoints,
			"total":            len(knowledgePoints),
		},
	)
}

// ListTextbookUnits GET /api/v1/curriculum/textbook-units
func (h *CurriculumHandler) ListTextbookUnits(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := strings.TrimSpace(
		r.URL.Query().Get("subject"),
	)
	if subject == "" {
		utils.BadRequest(
			w,
			"缺少 subject 参数",
		)
		return
	}

	publisher := strings.TrimSpace(
		r.URL.Query().Get("publisher"),
	)
	semester := strings.TrimSpace(
		r.URL.Query().Get("semester"),
	)
	gradeNum, _ := strconv.Atoi(
		r.URL.Query().Get("grade"),
	)

	domain, err := h.resolveCurriculumEducationDomain(
		r.Context(),
		claims.UserID,
	)
	if err != nil {
		utils.InternalError(
			w,
			"查询教材单元失败",
		)
		return
	}

	units := []*models.TextbookUnit{}

	if domain == models.EducationDomainK12 {
		units, err = h.deps.listTextbookUnits(
			r.Context(),
			domain,
			subject,
			publisher,
			gradeNum,
			semester,
		)
		if err != nil {
			utils.InternalError(
				w,
				"查询教材单元失败",
			)
			return
		}
		if units == nil {
			units = []*models.TextbookUnit{}
		}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"subject":   subject,
			"publisher": publisher,
			"grade":     gradeNum,
			"semester":  semester,
			"units":     units,
			"total":     len(units),
		},
	)
}

// ListPublishers GET /api/v1/curriculum/publishers
func (h *CurriculumHandler) ListPublishers(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := strings.TrimSpace(
		r.URL.Query().Get("subject"),
	)
	gradeNum, _ := strconv.Atoi(
		r.URL.Query().Get("grade"),
	)
	if subject == "" || gradeNum <= 0 {
		utils.BadRequest(
			w,
			"缺少 subject 或 grade 参数",
		)
		return
	}

	domain, err := h.resolveCurriculumEducationDomain(
		r.Context(),
		claims.UserID,
	)
	if err != nil {
		utils.InternalError(
			w,
			"查询教材版本失败",
		)
		return
	}

	publishers := []string{}

	if domain == models.EducationDomainK12 {
		publishers, err =
			h.deps.listPublishers(
				r.Context(),
				domain,
				subject,
				gradeNum,
			)
		if err != nil {
			utils.InternalError(
				w,
				"查询教材版本失败",
			)
			return
		}
		if publishers == nil {
			publishers = []string{}
		}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"subject":    subject,
			"grade":      gradeNum,
			"publishers": publishers,
		},
	)
}
