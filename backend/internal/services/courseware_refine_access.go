package services

// courseware_refine_access.go — 课件教研微调专属可信Actor授权
//
// 教研微调权限与普通完整编辑权限不同：
//
//   - 作者本人可以微调；
//   - 进行中的集体备课参与者可以微调；
//   - admin不因平台角色自动获得微调权；
//   - in_pipeline和submitted状态不能微调；
//   - 非作者参与者必须与课件教育域匹配；
//   - 作者换校后继续使用课件历史education_domain快照。
//
// 授权成功后返回收敛到课件快照域的Actor，供AI配置、资源加载和
// 写库前最终授权继续使用。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// LoadCoursewareForRefine 按ID加载课件并验证教研微调权限。
func (s *CoursewareService) LoadCoursewareForRefine(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareAccessNotFound,
				err,
			)
	}

	allowed, err :=
		s.CanRefineLoadedCourseware(
			ctx,
			courseware,
			actor,
		)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil,
			nil,
			ErrCoursewareEditDenied
	}

	return courseware,
		scopeAuthorizedCoursewareActor(
			actor,
			courseware,
		),
		nil
}
