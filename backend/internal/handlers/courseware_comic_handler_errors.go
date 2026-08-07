package handlers

// courseware_comic_handler_errors.go — 知识点漫画错误到HTTP状态码映射
//
// 映射原则：
//   - 400：请求正文、选项、教材、知识点或编辑文档不合法；
//   - 402：积分不足；
//   - 403：不是课件作者或教育域不允许；
//   - 404：课件、漫画项目或漫画格不存在；
//   - 409：审核锁、版本冲突、状态冲突或需要重新规划；
//   - 422：漫画助手学科或场景不适用；
//   - 502：上游AI失败或结构化输出失败；
//   - 503：AI配置或服务未就绪；
//   - 500：数据库、参考资料或服务端快照异常。

import (
	"errors"
	"net/http"

	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

func writeCoursewareComicHandlerError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		utils.InternalError(
			w,
			"知识点漫画请求处理失败",
		)

	// ==================== 400 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicProjectInvalidRequest,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicProjectGradeInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicProjectUnitNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicProjectKnowledgePointInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicProjectTeacherFocusTooLong,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicOverlayInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicPromptInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanInvalidRequest,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanInstructionTooLong,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicWorkflowInvalidRequest,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicStyleInstructionTooLong,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	// ==================== 402 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanCreditsInsufficient,
	):
		utils.Fail(
			w,
			http.StatusPaymentRequired,
			"积分余额不足，无法生成知识点漫画规划",
		)

	// ==================== 403 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicProjectK12Required,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanK12Required,
		),
		errors.Is(
			err,
			services.ErrCoursewareActorRequired,
		),
		errors.Is(
			err,
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		),
		errors.Is(
			err,
			repository.ErrCoursewareComicEducationDomainUnsupported,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权执行此知识点漫画操作",
		)

	// ==================== 404 ====================

	case errors.Is(
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
		repository.ErrCoursewareComicProjectNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"知识点漫画项目不存在",
		)

	case errors.Is(
		err,
		repository.ErrCoursewareComicPanelNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"知识点漫画分格不存在",
		)

	// ==================== 409 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanMutationLocked,
	),
		errors.Is(
			err,
			services.ErrCoursewareControlMutationLocked,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"课件正在审核或生产流程中，暂不允许修改知识点漫画",
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicNarrativeReplanRequired,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"叙事方式已经改变，请重新生成分镜后再确认",
		)

	case errors.Is(
		err,
		repository.ErrCoursewareComicProjectConflict,
	),
		errors.Is(
			err,
			repository.ErrCoursewareComicPanelConflict,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"漫画内容已经发生变化，请刷新后重试",
		)

	case errors.Is(
		err,
		repository.ErrCoursewareComicProjectNotEditable,
	),
		errors.Is(
			err,
			repository.ErrCoursewareComicPanelNotGeneratable,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"漫画项目当前状态不允许执行此操作",
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanSchoolRequired,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"当前教师未绑定可用学校，不能生成知识点漫画规划",
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanAssistantUnavailable,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"选择的漫画助手当前不可用",
		)

	// ==================== 422 ====================

	case errors.Is(
		err,
		services.ErrCoursewareAssistantAssistantScopeMismatch,
	):
		utils.Fail(
			w,
			http.StatusUnprocessableEntity,
			"选择的漫画助手不适用于当前学科或知识点漫画场景",
		)

	// ==================== 502 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanAICallFailed,
	):
		utils.Fail(
			w,
			http.StatusBadGateway,
			"知识点漫画规划生成失败，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanInvalidOutput,
	):
		utils.Fail(
			w,
			http.StatusBadGateway,
			"AI未返回完整有效的漫画规划，请重新生成",
		)

	// ==================== 503 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicProjectServiceUnavailable,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanServiceUnavailable,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanAIConfigUnavailable,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"知识点漫画服务暂不可用",
		)

	// ==================== 500 ====================

	case errors.Is(
		err,
		services.ErrCoursewareComicReferenceReadFailed,
	):
		utils.InternalError(
			w,
			"读取知识点漫画参考资料失败，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicPlanContextInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicPlanSchoolResolveFailed,
	),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
	):
		utils.InternalError(
			w,
			"知识点漫画服务数据异常，请稍后重试",
		)

	default:
		utils.InternalError(
			w,
			"知识点漫画请求处理失败",
		)
	}
}
