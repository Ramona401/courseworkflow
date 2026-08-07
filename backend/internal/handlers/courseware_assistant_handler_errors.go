package handlers

// courseware_assistant_handler_errors.go
//
// 本文件把课件教学智能体Service错误稳定映射为HTTP状态码。
//
// 映射原则：
//   - 400：请求协议、正文长度或字段值错误；
//   - 402：教师积分不足；
//   - 403：课件、助手或教育域使用权限不足；
//   - 404：课件、页面或插槽不存在；
//   - 409：审核锁、重复插槽或账号绑定状态冲突；
//   - 422：业务范围不匹配；
//   - 502：上游AI调用或自动纠正后的结构化输出失败；
//   - 503：AI配置或服务依赖未就绪；
//   - 500：数据库、资源快照或服务端教育域异常。
//
// 默认错误不把数据库错误、SQL、提示词、模型供应商或AI原始响应返回浏览器。

import (
	"errors"
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// writeCoursewareAssistantHandlerError 统一返回安全错误响应。
func writeCoursewareAssistantHandlerError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		utils.InternalError(
			w,
			"课件教学智能体请求处理失败",
		)

	// ==================== 400 请求错误 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantInvalidRequest,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantPlanInstructionTooLong,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	// ==================== 402 积分不足 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPlanCreditsInsufficient,
	):
		utils.Fail(
			w,
			http.StatusPaymentRequired,
			"积分余额不足，无法生成课件教学智能体方案",
		)

	// ==================== 403 权限拒绝 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantReadDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareAssistantWriteDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareAssistantAssistantUseDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareAssistantAssistantDomainMismatch,
		),
		errors.Is(
			err,
			services.ErrCoursewareActorRequired,
		),
		errors.Is(
			err,
			services.ErrCoursewareViewDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权执行此课件教学智能体操作",
		)

	// ==================== 404 资源不存在 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantCoursewareNotFound,
	),
		errors.Is(
			err,
			services.ErrCoursewareAccessNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件不存在",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPageNotFound,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantContextPageNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件页面不存在",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantSlotNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件教学智能体方案不存在",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantAssistantNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"历史教学风格助手不存在，系统将使用页面默认教学风格",
		)

	// ==================== 409 状态冲突 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantMutationLocked,
	),
		errors.Is(
			err,
			services.ErrCoursewareControlMutationLocked,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"课件正在审核流程中，暂不允许修改教学智能体方案",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantSlotAlreadyExists,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"当前页面已经存在教学智能体方案",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantAssistantInactive,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantPlanAssistantUnavailable,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"历史教学风格助手当前不可用，系统将使用页面默认教学风格",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantContextNoPages,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"课件没有可用于教学智能体的页面",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantContextLessonMissing,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"课件来源教案当前不可用",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPlanSchoolRequired,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"当前教师未绑定可用学校，不能生成教学智能体方案",
		)

	// ==================== 422 业务范围不匹配 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantAssistantScopeMismatch,
	):
		utils.Fail(
			w,
			http.StatusUnprocessableEntity,
			"历史教学风格助手不适用于当前课件，系统将使用页面默认教学风格",
		)

	// ==================== 502 上游AI错误 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPlanAICallFailed,
	):
		utils.Fail(
			w,
			http.StatusBadGateway,
			"教学智能体方案生成失败，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPlanInvalidOutput,
	):
		utils.Fail(
			w,
			http.StatusBadGateway,
			coursewareAssistantPlanInvalidOutputMessage(
				err,
			),
		)

	// ==================== 503 服务未就绪 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantPlanServiceUnavailable,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantPlanAIConfigUnavailable,
		):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"课件教学智能体服务暂不可用",
		)

	// ==================== 500 服务端数据异常 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantCoursewareDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		),
		errors.Is(
			err,
			services.ErrCoursewareAssistantContextLessonDomainMismatch,
		):
		utils.InternalError(
			w,
			"课件教学教育域异常，请联系管理员处理",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantCreatorNotFound,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantPlanSchoolResolveFailed,
		),
		errors.Is(
			err,
			services.ErrCoursewareAssistantContextBuildFailed,
		):
		utils.InternalError(
			w,
			"课件教学智能体服务数据异常，请稍后重试",
		)

	default:
		utils.InternalError(
			w,
			"课件教学智能体请求处理失败",
		)
	}
}

// coursewareAssistantPlanInvalidOutputMessage 返回教师可理解且不泄露内部数据的原因。
func coursewareAssistantPlanInvalidOutputMessage(
	err error,
) string {
	message := ""

	if err != nil {
		message = err.Error()
	}

	switch {
	case strings.Contains(
		message,
		"JSON",
	),
		strings.Contains(
			message,
			"完整合法",
		):
		return "AI返回的方案格式不完整，系统已自动纠正一次但仍未成功，请重新生成。"

	case strings.Contains(
		message,
		"教学方式",
	):
		return "AI生成结果与所选学习方式不一致，系统已自动纠正一次，请重新生成。"

	case strings.Contains(
		message,
		"互动步骤",
	):
		return "AI生成的互动步骤数量或结构不符合课堂要求，系统已自动纠正一次，请重新生成。"

	case strings.Contains(
		message,
		"提示",
	):
		return "AI生成的提示层级不符合课堂安全要求，系统已自动纠正一次，请重新生成。"

	case strings.Contains(
		message,
		"学习困难",
	),
		strings.Contains(
			message,
			"分支",
		):
		return "AI生成的学习困难应对方案引用不完整，系统已自动纠正一次，请重新生成。"

	case strings.Contains(
		message,
		"学生先参与",
	),
		strings.Contains(
			message,
			"答案",
		):
		return "AI生成结果没有通过学生参与和答案保护校验，系统已自动纠正一次，请重新生成。"

	default:
		return "AI生成结果没有通过课堂结构与答案保护校验，系统已自动纠正一次，请重新生成。"
	}
}
