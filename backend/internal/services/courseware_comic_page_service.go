package services

// courseware_comic_page_service.go — 漫画页面插入和稳定分格同步
//
// 提供两条确定性写入链：
//
// 1. InsertOrRefreshPage
//    - 首次调用：原子插入新页面；
//    - 插页后严格校准全部导航页码；
//    - 记录project.inserted_page_id；
//    - 重复调用：不创建重复页，只刷新原漫画页。
//
// 2. SyncPanelToInsertedPage
//    - 单格重新生成完成后调用；
//    - 只替换该格的TEDNA_COMIC_PANEL_START/END区间；
//    - 页面稳定标记缺失、重复或页面已并发修改时返回409；
//    - 覆盖前必须成功保存页面完整版本快照。
//
// 幂等恢复：
//   插页已经成功但项目状态尚未来得及记录时，服务会通过项目稳定标记
//   找到已存在页面，不会再次创建重复漫画页。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

type CoursewareComicPageResult struct {
	CoursewareID string `json:"courseware_id"`
	ProjectID    string `json:"project_id"`
	PageID       string `json:"page_id"`
	PageNumber   int    `json:"page_number"`
	Status       string `json:"status"`
	Created      bool   `json:"created"`
	Updated      bool   `json:"updated"`
}

// CoursewareComicPageService 漫画页面业务服务。
type CoursewareComicPageService struct {
	coursewareService *CoursewareService
}

// NewCoursewareComicPageService 创建漫画页面服务。
func NewCoursewareComicPageService(
	coursewareService *CoursewareService,
) *CoursewareComicPageService {
	if coursewareService == nil {
		coursewareService =
			NewCoursewareService()
	}

	return &CoursewareComicPageService{
		coursewareService:
			coursewareService,
	}
}

// InsertOrRefreshPage 首次插入或刷新已有漫画页面。
func (s *CoursewareComicPageService) InsertOrRefreshPage(
	ctx context.Context,
	coursewareID string,
	projectID string,
	expectedVersion int,
	insertAt int,
	actor *CoursewareActorContext,
) (*CoursewareComicPageResult, error) {
	if s == nil ||
		expectedVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	coursewareID =
		strings.TrimSpace(coursewareID)
	projectID =
		strings.TrimSpace(projectID)

	courseware, scopedActor, err :=
		s.coursewareService.
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	if err :=
		validateCoursewarePageMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if project.Version !=
		expectedVersion {
		return nil,
			repository.ErrCoursewareComicProjectConflict
	}

	if project.Status !=
		models.CWComicProjectStatusReady &&
		project.Status !=
			models.CWComicProjectStatusInserted {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	panels, err :=
		repository.ListCoursewareComicPanels(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	renderData, err :=
		s.loadCoursewareComicPanelRenderData(
			ctx,
			courseware.ID,
			project,
			panels,
		)
	if err != nil {
		return nil, err
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, err
	}

	existingPage, err :=
		findCoursewareComicInsertedPage(
			pages,
			project,
		)
	if err != nil {
		return nil, err
	}

	created := false
	updated := false

	var targetPage *models.CoursewarePage

	if existingPage == nil {
		if insertAt <= 0 {
			insertAt =
				len(pages) + 1
		}

		if insertAt < 1 ||
			insertAt >
				len(pages)+1 {
			return nil,
				ErrCoursewareComicProjectInvalidRequest
		}

		pageHTML, renderErr :=
			renderCoursewareComicPageHTML(
				courseware,
				project,
				renderData,
				insertAt,
				len(pages)+1,
			)
		if renderErr != nil {
			return nil, renderErr
		}

		pageHTML, renderErr =
			applyCoursewareComicTemplate(
				ctx,
				courseware,
				insertAt,
				pageHTML,
			)
		if renderErr != nil {
			return nil, renderErr
		}

		placeholderMap :=
			buildCoursewareComicPlaceholderMap(
				project,
				panels,
			)

		targetPage =
			&models.CoursewarePage{
				CoursewareID:
					courseware.ID,
				PageNumber:
					insertAt,
				Title:
					project.Title,
				Purpose:
					"通过连续漫画情境呈现知识点、形成性问题与解释",
				ContentSummary:
					truncateCoursewareComicRunes(
						project.KnowledgeContentSnapshot,
						1000,
					),
				InteractionType:
					"click",
				VisualFormat:
					"knowledge_comic",
				MediaRequirements:
					"comic_panels",
				EstimatedComplexity:
					4,
				PageIndex:
					buildCoursewareComicPageIndex(
						project,
						panels,
					),
				IdxCognitiveLevel:
					3,
				IdxInteractionLevel:
					1,
				IdxVisualFormat:
					"CM",
				HTMLContent:
					pageHTML,
				PlaceholderMap:
					placeholderMap,
				MatchedComponentIDs:
					"[]",
				Status:
					models.CWPageStatusGenerated,
			}

		if err :=
			repository.InsertCoursewarePageAtPosition(
				ctx,
				targetPage,
				insertAt,
			); err != nil {
			return nil, err
		}

		created = true

		// 页数变化后必须严格刷新全部页面导航栏。
		if err :=
			s.coursewareService.
				ResyncCWPageNumbers(
					ctx,
					courseware.ID,
				); err != nil {
			return nil,
				fmt.Errorf(
					"%w: 漫画页已插入，但导航页码校准失败: %v",
					repository.ErrCoursewareComicProjectConflict,
					err,
				)
		}

		pages, err =
			repository.ListCoursewarePages(
				ctx,
				courseware.ID,
			)
		if err != nil {
			return nil, err
		}

		targetPage =
			findCoursewarePageByID(
				pages,
				targetPage.ID,
			)
		if targetPage == nil {
			return nil,
				repository.ErrCoursewareComicProjectConflict
		}
	} else {
		targetPage = existingPage

		pagePosition :=
			findCoursewarePagePosition(
				pages,
				targetPage.ID,
			)
		if pagePosition < 1 {
			return nil,
				repository.ErrCoursewareComicProjectConflict
		}

		if strings.Count(
			targetPage.HTMLContent,
			coursewareComicProjectStartMarker(
				project.ID,
			),
		) != 1 {
			return nil,
				fmt.Errorf(
					"%w: 已插入页面的漫画项目标记缺失或重复",
					repository.ErrCoursewareComicProjectConflict,
				)
		}

		pageHTML, renderErr :=
			renderCoursewareComicPageHTML(
				courseware,
				project,
				renderData,
				pagePosition,
				len(pages),
			)
		if renderErr != nil {
			return nil, renderErr
		}

		pageHTML, renderErr =
			applyCoursewareComicTemplate(
				ctx,
				courseware,
				pagePosition,
				pageHTML,
			)
		if renderErr != nil {
			return nil, renderErr
		}

		if err :=
			s.saveAndReplaceCoursewareComicPageHTML(
				ctx,
				courseware,
				pages,
				targetPage,
				pageHTML,
				"知识点漫画整页刷新前",
			); err != nil {
			return nil, err
		}

		updated = true
		targetPage.HTMLContent =
			pageHTML
		targetPage.PageNumber =
			pagePosition
	}

	latestProject, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if latestProject.Status ==
		models.CWComicProjectStatusReady {
		latestProject, err =
			repository.MarkCoursewareComicProjectInserted(
				ctx,
				courseware.ID,
				project.ID,
				scopedActor.UserID,
				targetPage.ID,
				targetPage.PageNumber,
				latestProject.Version,
			)
		if err != nil {
			return nil, err
		}
	}

	return &CoursewareComicPageResult{
		CoursewareID:
			courseware.ID,
		ProjectID:
			project.ID,
		PageID:
			targetPage.ID,
		PageNumber:
			targetPage.PageNumber,
		Status:
			latestProject.Status,
		Created:
			created,
		Updated:
			updated,
	}, nil
}

// SyncPanelToInsertedPage 只替换已插入漫画页中的一个稳定漫画格。
func (s *CoursewareComicPageService) SyncPanelToInsertedPage(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	expectedPanelVersion int,
	actor *CoursewareActorContext,
) (*CoursewareComicPageResult, error) {
	if s == nil ||
		expectedPanelVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	courseware, scopedActor, err :=
		s.coursewareService.
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

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	if err :=
		validateCoursewarePageMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			strings.TrimSpace(
				projectID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if project.Status !=
		models.CWComicProjectStatusReady &&
		project.Status !=
			models.CWComicProjectStatusInserted {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			courseware.ID,
			project.ID,
			strings.TrimSpace(
				panelID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		expectedPanelVersion ||
		panel.Status !=
			models.CWComicPanelStatusGenerated ||
		panel.CurrentAssetID == nil {
		return nil,
			repository.ErrCoursewareComicPanelConflict
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			*panel.CurrentAssetID,
		)
	if err != nil ||
		asset == nil ||
		asset.CoursewareID !=
			courseware.ID ||
		asset.AssetType !=
			models.CWAssetTypeImage {
		return nil,
			repository.ErrCoursewareComicAssetInvalid
	}

	imageURL :=
		resolveAssetPublicURL(asset)
	if imageURL == "" {
		return nil,
			repository.ErrCoursewareComicAssetInvalid
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, err
	}

	targetPage, err :=
		findCoursewareComicInsertedPage(
			pages,
			project,
		)
	if err != nil {
		return nil, err
	}
	if targetPage == nil {
		return nil,
			fmt.Errorf(
				"%w: 漫画尚未插入课件页面",
				repository.ErrCoursewareComicProjectConflict,
			)
	}

	if strings.Count(
		targetPage.HTMLContent,
		coursewareComicProjectStartMarker(
			project.ID,
		),
	) != 1 {
		return nil,
			fmt.Errorf(
				"%w: 漫画页面项目标记缺失或重复",
				repository.ErrCoursewareComicProjectConflict,
			)
	}

	panelHTML, err :=
		renderCoursewareComicPanelHTML(
			project,
			coursewareComicPanelRenderData{
				Panel:    panel,
				ImageURL: imageURL,
			},
			project.PanelCount >= 7,
		)
	if err != nil {
		return nil, err
	}

	newHTML, err :=
		replaceCoursewareComicPanelFragment(
			targetPage.HTMLContent,
			project.ID,
			panel.ID,
			panelHTML,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				repository.ErrCoursewareComicProjectConflict,
				err,
			)
	}

	if err :=
		s.saveAndReplaceCoursewareComicPageHTML(
			ctx,
			courseware,
			pages,
			targetPage,
			newHTML,
			fmt.Sprintf(
				"知识点漫画第%d格图片更新前",
				panel.PanelNo,
			),
		); err != nil {
		return nil, err
	}

	latestProject, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if latestProject.Status ==
		models.CWComicProjectStatusReady {
		latestProject, err =
			repository.MarkCoursewareComicProjectInserted(
				ctx,
				courseware.ID,
				project.ID,
				scopedActor.UserID,
				targetPage.ID,
				targetPage.PageNumber,
				latestProject.Version,
			)
		if err != nil {
			return nil, err
		}
	}

	return &CoursewareComicPageResult{
		CoursewareID:
			courseware.ID,
		ProjectID:
			project.ID,
		PageID:
			targetPage.ID,
		PageNumber:
			targetPage.PageNumber,
		Status:
			latestProject.Status,
		Created:
			false,
		Updated:
			true,
	}, nil
}

func (s *CoursewareComicPageService) loadCoursewareComicPanelRenderData(
	ctx context.Context,
	coursewareID string,
	project *models.CoursewareComicProject,
	panels []*models.CoursewareComicPanel,
) ([]coursewareComicPanelRenderData, error) {
	if project == nil ||
		len(panels) !=
			project.PanelCount {
		return nil,
			fmt.Errorf(
				"漫画分格数量与项目规划不一致",
			)
	}

	result := make(
		[]coursewareComicPanelRenderData,
		0,
		len(panels),
	)

	for _, panel := range panels {
		if panel == nil ||
			panel.Status !=
				models.CWComicPanelStatusGenerated ||
			panel.CurrentAssetID == nil {
			return nil,
				repository.ErrCoursewareComicPanelNotGeneratable
		}

		asset, err :=
			repository.GetCWAssetByID(
				ctx,
				*panel.CurrentAssetID,
			)
		if err != nil ||
			asset == nil ||
			asset.CoursewareID !=
				coursewareID ||
			asset.AssetType !=
				models.CWAssetTypeImage {
			return nil,
				repository.ErrCoursewareComicAssetInvalid
		}

		imageURL :=
			resolveAssetPublicURL(asset)
		if imageURL == "" {
			return nil,
				repository.ErrCoursewareComicAssetInvalid
		}

		result = append(
			result,
			coursewareComicPanelRenderData{
				Panel:    panel,
				ImageURL: imageURL,
			},
		)
	}

	return result, nil
}

func (s *CoursewareComicPageService) saveAndReplaceCoursewareComicPageHTML(
	ctx context.Context,
	courseware *models.Courseware,
	allPages []*models.CoursewarePage,
	targetPage *models.CoursewarePage,
	newHTML string,
	versionNote string,
) error {
	if courseware == nil ||
		targetPage == nil ||
		targetPage.UpdatedAt == nil {
		return repository.
			ErrCoursewareComicProjectConflict
	}

	if err :=
		validateCoursewarePageHTMLPayload(
			newHTML,
		); err != nil {
		return err
	}

	genService :=
		&CoursewareGenService{}

	if err :=
		genService.
			SavePageVersionBeforeOverwriteStrict(
				ctx,
				targetPage.ID,
				courseware.ID,
				targetPage.HTMLContent,
				models.CWPageVersionSourceManual,
				versionNote,
			); err != nil {
		return err
	}

	orderedPageIDs := make(
		[]string,
		0,
		len(allPages),
	)

	foundTarget := false

	for _, page := range allPages {
		if page == nil {
			continue
		}

		orderedPageIDs = append(
			orderedPageIDs,
			page.ID,
		)

		if page.ID ==
			targetPage.ID {
			foundTarget = true
		}
	}

	if !foundTarget ||
		len(orderedPageIDs) !=
			len(allPages) {
		return repository.
			ErrCoursewareComicProjectConflict
	}

	err :=
		repository.ApplyCoursewarePageCalibration(
			ctx,
			courseware.ID,
			orderedPageIDs,
			map[string]repository.
				CoursewarePageCalibrationHTML{
				targetPage.ID: {
					HTMLContent:
						newHTML,
					ExpectedUpdatedAt:
						*targetPage.UpdatedAt,
				},
			},
		)
	if err != nil {
		return fmt.Errorf(
			"%w: 页面已被其他操作修改，请刷新后重试: %v",
			repository.ErrCoursewareComicProjectConflict,
			err,
		)
	}

	return nil
}

func applyCoursewareComicTemplate(
	ctx context.Context,
	courseware *models.Courseware,
	pageNumber int,
	pageHTML string,
) (string, error) {
	genService :=
		&CoursewareGenService{}

	styleConfig :=
		genService.parseStyleConfig(
			courseware.StyleConfig,
		)

	templateInfo, err :=
		genService.loadTemplateInfo(
			ctx,
			styleConfig.TemplateID,
		)
	if err != nil ||
		templateInfo == nil {
		templateInfo =
			genService.defaultTemplateInfo()
	}

	genService.attachUserBackground(
		ctx,
		courseware,
		templateInfo,
	)

	return genService.applyTemplateBackground(
		pageHTML,
		templateInfo,
		pageNumber,
	), nil
}

func findCoursewareComicInsertedPage(
	pages []*models.CoursewarePage,
	project *models.CoursewareComicProject,
) (*models.CoursewarePage, error) {
	if project == nil {
		return nil,
			repository.ErrCoursewareComicProjectNotFound
	}

	if project.InsertedPageID != nil &&
		strings.TrimSpace(
			*project.InsertedPageID,
		) != "" {
		page :=
			findCoursewarePageByID(
				pages,
				*project.InsertedPageID,
			)

		if page == nil {
			return nil,
				fmt.Errorf(
					"%w: 项目记录的漫画页面已经不存在",
					repository.ErrCoursewareComicProjectConflict,
				)
		}

		return page, nil
	}

	marker :=
		coursewareComicProjectStartMarker(
			project.ID,
		)

	var matchedPage *models.CoursewarePage

	for _, page := range pages {
		if page == nil ||
			!strings.Contains(
				page.HTMLContent,
				marker,
			) {
			continue
		}

		if matchedPage != nil {
			return nil,
				fmt.Errorf(
					"%w: 同一漫画项目出现多个页面",
					repository.ErrCoursewareComicProjectConflict,
				)
		}

		matchedPage = page
	}

	return matchedPage, nil
}

func findCoursewarePageByID(
	pages []*models.CoursewarePage,
	pageID string,
) *models.CoursewarePage {
	for _, page := range pages {
		if page != nil &&
			page.ID ==
				pageID {
			return page
		}
	}

	return nil
}

func findCoursewarePagePosition(
	pages []*models.CoursewarePage,
	pageID string,
) int {
	for index, page := range pages {
		if page != nil &&
			page.ID ==
				pageID {
			return index + 1
		}
	}

	return 0
}

func buildCoursewareComicPlaceholderMap(
	project *models.CoursewareComicProject,
	panels []*models.CoursewareComicPanel,
) string {
	panelIDs := make(
		[]string,
		0,
		len(panels),
	)

	for _, panel := range panels {
		if panel != nil {
			panelIDs = append(
				panelIDs,
				panel.ID,
			)
		}
	}

	encoded, err :=
		json.Marshal(
			map[string]interface{}{
				"comic_project_id":
					project.ID,
				"comic_panel_ids":
					panelIDs,
			},
		)
	if err != nil {
		return "{}"
	}

	return string(encoded)
}

func buildCoursewareComicPageIndex(
	project *models.CoursewareComicProject,
	panels []*models.CoursewareComicPanel,
) string {
	var builder strings.Builder

	builder.WriteString(
		"COMIC_PROJECT=",
	)
	builder.WriteString(project.ID)

	for _, panel := range panels {
		if panel == nil {
			continue
		}

		builder.WriteString(
			fmt.Sprintf(
				"\nCOMIC_PANEL_%02d=%s",
				panel.PanelNo,
				panel.ImageKey,
			),
		)
	}

	return builder.String()
}
