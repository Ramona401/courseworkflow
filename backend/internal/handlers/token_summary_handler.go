package handlers

// token_summary_handler.go — Token积分消费汇总报告HTTP处理器
//
// 登录用户按TokenScope查看自身范围内积分数据。
// 只有RoleAdmin且IsSuper=true的超级管理员可以：
//   - 查看model汇总维度；
//   - 获得美元成本字段。
// 其它角色获得独立公共DTO，响应中不序列化成本字段。

import (
	"net/http"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GetConsumptionSummary 获取消费汇总报告。
func (h *TokenHandler) GetConsumptionSummary(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持GET请求",
			nil,
		)
		return
	}

	query :=
		r.URL.Query()

	dimension :=
		query.Get("dimension")

	switch dimension {
	case models.SummaryDimRegion,
		models.SummaryDimSchool,
		models.SummaryDimUser,
		models.SummaryDimModel,
		models.SummaryDimScene,
		models.SummaryDimTime:

	default:
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"无效的汇总维度(region/school/user/model/scene/time)",
			nil,
		)
		return
	}

	canViewInternalCosts :=
		tokenCanViewInternalCostDetails(r)

	if dimension ==
		models.SummaryDimModel &&
		!canViewInternalCosts {
		utils.JSON(
			w,
			http.StatusForbidden,
			-1,
			"模型维度仅超级管理员可查看",
			nil,
		)
		return
	}

	var (
		fromTime time.Time
		toTime   time.Time
	)

	if value :=
		query.Get("from");
		value != "" {
		parsed, err :=
			time.ParseInLocation(
				"2006-01-02",
				value,
				time.Local,
			)

		if err == nil {
			fromTime =
				parsed
		}
	}

	if value :=
		query.Get("to");
		value != "" {
		parsed, err :=
			time.ParseInLocation(
				"2006-01-02",
				value,
				time.Local,
			)

		if err == nil {
			toTime =
				parsed.AddDate(
					0,
					0,
					1,
				)
		}
	}

	scope :=
		h.resolveScope(r)

	input :=
		&services.ConsumptionSummaryInput{
			Dimension: dimension,
			From:      fromTime,
			To:        toTime,
		}

	if userFilter :=
		query.Get("user_filter");
		userFilter != "" {
		if userIDAllowedInScope(
			scope,
			userFilter,
		) {
			input.UserFilter =
				userFilter
		} else {
			input.SchoolMember =
				[]string{}
		}
	}

	if schoolFilter :=
		query.Get("school_filter");
		schoolFilter != "" &&
		input.UserFilter == "" {
		memberIDs, err :=
			repository.ListSchoolMemberIDs(
				r.Context(),
				schoolFilter,
			)
		if err != nil {
			utils.JSON(
				w,
				http.StatusInternalServerError,
				-1,
				"查询学校成员失败",
				nil,
			)
			return
		}

		input.SchoolMember =
			intersectWithScopeUsers(
				scope,
				memberIDs,
			)
	}

	response, err :=
		h.tokenService.GetConsumptionSummary(
			r.Context(),
			input,
			scope,
		)
	if err != nil {
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"获取汇总报告失败",
			nil,
		)
		return
	}

	if canViewInternalCosts {
		utils.JSON(
			w,
			http.StatusOK,
			0,
			"",
			response,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		buildTokenPublicConsumptionSummary(
			response,
		),
	)
}

func userIDAllowedInScope(
	scope *services.TokenScope,
	userID string,
) bool {
	if scope == nil {
		return false
	}

	if scope.UserIDs == nil {
		return true
	}

	for _, id :=
		range scope.UserIDs {
		if id == userID {
			return true
		}
	}

	return false
}

func intersectWithScopeUsers(
	scope *services.TokenScope,
	memberIDs []string,
) []string {
	if scope == nil {
		return []string{}
	}

	if scope.UserIDs == nil {
		return memberIDs
	}

	scopeSet :=
		make(
			map[string]struct{},
			len(scope.UserIDs),
		)

	for _, id :=
		range scope.UserIDs {
		scopeSet[id] =
			struct{}{}
	}

	result :=
		make(
			[]string,
			0,
			len(memberIDs),
		)

	for _, id :=
		range memberIDs {
		if _, exists :=
			scopeSet[id];
			exists {
			result =
				append(
					result,
					id,
				)
		}
	}

	return result
}
