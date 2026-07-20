package handlers

// courseware_owner_control_helper.go — 课件作者私有控制面Handler授权辅助
//
// 使用范围：
//   - 课件标题、删除、步骤确认和回退；
//   - 风格、导航栏和Logo；
//   - 页面增删改排序；
//   - 发布、代码开放范围和集体备课管理；
//   - 背景、页级背景和字体。
//
// Handler预检不能替代Service授权。预检的主要作用是：
//   1. 在解析JSON、multipart和打开上传文件前尽早拒绝非法请求；
//   2. 为后续Service传递可信Actor；
//   3. Service仍会重新加载正式数据库课件并执行二次校验。

import (
	"context"
	"errors"
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// authorizeCoursewareOwnerRuntimeForHandler 构造可信Actor并执行作者运行域预检。
func authorizeCoursewareOwnerRuntimeForHandler(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor := services.BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)

	_, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// authorizeCoursewareViewForHandler 执行课件统一查看权预检。
//
// 适用于背景、页级背景和字体的GET端点，防止通过猜测课件ID读取
// 私有课件的选择配置。
func authorizeCoursewareViewForHandler(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) error {
	actor := services.BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)

	_, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForView(
				ctx,
				coursewareID,
				actor,
			)

	return err
}

// writeCoursewareControlError 映射核心控制面错误。
func writeCoursewareControlError(
	w http.ResponseWriter,
	err error,
) {
	if errors.Is(
		err,
		services.ErrCoursewareControlMutationLocked,
	) {
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)
		return
	}

	writeCoursewareOwnerRuntimeError(
		w,
		err,
	)
}
