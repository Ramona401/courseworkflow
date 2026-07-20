package handlers

// courseware_access_handler.go — 课件统一访问错误到HTTP状态码映射
//
// 课件详情、页面、原文、对齐报告、集体备课状态等按ID读取入口统一使用：
//   - 课件不存在：404；
//   - Actor缺失、无查看权或教育域不匹配：403；
//   - 课件教育域为空、非法或不能作为具体运行域：500；
//   - 其它数据库或内部错误：500。
//
// 不把具体跨域信息返回给前端，避免通过错误文案探测其它教育域资源。

import (
	"errors"
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

func handleCoursewareAccessError(
	w http.ResponseWriter,
	err error,
	fallbackMessage string,
) {
	switch {
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
		services.ErrCoursewareActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareViewDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权查看此课件",
		)

	case errors.Is(
		err,
		services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		utils.InternalError(
			w,
			"课件教育域异常，请联系管理员处理",
		)

	default:
		if fallbackMessage == "" {
			fallbackMessage = "课件访问失败"
		}
		utils.InternalError(
			w,
			fallbackMessage+": "+err.Error(),
		)
	}
}
