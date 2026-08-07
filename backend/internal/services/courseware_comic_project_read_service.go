package services

// courseware_comic_project_read_service.go — 知识点漫画项目读取服务
//
// 本文件负责：
//   - 返回作者在指定课件中的全部漫画项目；
//   - 返回单个项目及全部漫画格；
//   - 所有读取继续经过课件作者运行通道；
//   - 浏览器只获得显式安全视图，不暴露内部JSON字符串或提示词配置。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ListProjects 返回作者在课件中的全部漫画项目。
func (s *CoursewareComicProjectService) ListProjects(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicProjectListView, error) {
	_, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				actor,
			)
	if err != nil {
		return nil, err
	}

	projects, err :=
		repository.ListCoursewareComicProjectsByCourseware(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	views := make(
		[]*models.CoursewareComicProjectView,
		0,
		len(projects),
	)

	for _, project := range projects {
		view, viewErr :=
			buildCoursewareComicProjectView(
				project,
			)
		if viewErr != nil {
			return nil, viewErr
		}

		views = append(
			views,
			view,
		)
	}

	return &models.CoursewareComicProjectListView{
		Projects: views,
		Total:    len(views),
	}, nil
}

// GetProjectDetail 返回项目和全部漫画格。
func (s *CoursewareComicProjectService) GetProjectDetail(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicProjectDetailView, error) {
	_, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				actor,
			)
	if err != nil {
		return nil, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	panels, err :=
		repository.ListCoursewareComicPanels(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return BuildCoursewareComicDetailView(
		project,
		panels,
	)
}
