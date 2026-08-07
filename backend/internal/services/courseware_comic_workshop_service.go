package services

// courseware_comic_workshop_service.go
//
// 提供漫画工坊持续编辑能力：
//   - 项目详情补充服务端校验后的安全图片URL；
//   - 项目详情补充不含正文和摘要的参考资源安全元数据；
//   - 已插入项目继续保存文字、题目和气泡；
//   - 已插入项目继续保存后端内部图片提示词与IAOCI；
//   - 更新响应继续返回当前漫画格安全图片URL。
//
// 图片URL只从正式绑定资产解析，资产必须属于当前课件且类型为image。
// 教师端公共HTTP路由不开放图片提示词和IAOCI直接编辑入口。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// GetProjectDetailForBrowser
// 返回带安全图片URL和参考资源元数据的漫画项目详情。
func (s *CoursewareComicProjectService) GetProjectDetailForBrowser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicProjectDetailView, error) {
	detail, err :=
		s.GetProjectDetail(
			ctx,
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if detail == nil ||
		detail.Project == nil {
		return nil,
			repository.
				ErrCoursewareComicProjectNotFound
	}

	detail.Project.CharacterSheetURL =
		loadCoursewareComicBrowserAssetURL(
			ctx,
			detail.Project.CoursewareID,
			detail.Project.CharacterSheetAssetID,
		)

	for _, panel := range detail.Panels {
		if panel == nil {
			continue
		}

		panel.CurrentAssetURL =
			loadCoursewareComicBrowserAssetURL(
				ctx,
				detail.Project.CoursewareID,
				panel.CurrentAssetID,
			)
	}

	referenceList, err :=
		s.ListProjectReferencesForBrowser(
			ctx,
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if referenceList == nil ||
		referenceList.References == nil {
		detail.References =
			[]*models.CoursewareComicReferenceResourceView{}
	} else {
		detail.References =
			referenceList.References
	}

	return detail, nil
}

// UpdatePanelOverlayForWorkshop
// 保存文字、题目和气泡覆盖层。
func (s *CoursewareComicProjectService) UpdatePanelOverlayForWorkshop(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	actor *CoursewareActorContext,
	request *models.UpdateCoursewareComicPanelOverlayRequest,
) (*models.CoursewareComicPanelView, error) {
	if request == nil ||
		request.ExpectedVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	courseware, scopedActor, err :=
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

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	projectID =
		strings.TrimSpace(
			projectID,
		)

	panelID =
		strings.TrimSpace(
			panelID,
		)

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			courseware.ID,
			projectID,
			panelID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		request.ExpectedVersion {
		return nil,
			repository.
				ErrCoursewareComicPanelConflict
	}

	if err :=
		preserveCoursewareComicOriginalOverlay(
			panel.OverlayDocumentJSON,
			&request.OverlayDocument,
		); err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareComicOverlayDocument(
			&request.OverlayDocument,
		); err != nil {
		return nil, err
	}

	dialogues :=
		deriveCoursewareComicDialogues(
			request.OverlayDocument.Elements,
		)

	dialoguesJSON, err :=
		json.Marshal(
			dialogues,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化漫画对白失败: %w",
				err,
			)
	}

	overlayJSON, err :=
		json.Marshal(
			request.OverlayDocument,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化漫画覆盖层失败: %w",
				err,
			)
	}

	updated, err :=
		repository.
			UpdateCoursewareComicPanelOverlayForWorkshopIfUnchanged(
				ctx,
				courseware.ID,
				projectID,
				panelID,
				scopedActor.UserID,
				request.ExpectedVersion,
				request.NarrationText,
				string(dialoguesJSON),
				string(overlayJSON),
			)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicPanelBrowserView(
		ctx,
		courseware.ID,
		updated,
	)
}

// UpdatePanelPromptForWorkshop
// 保存后端内部图片提示词和IAOCI。
//
// 教师端公共HTTP路由不挂载此方法。
func (s *CoursewareComicProjectService) UpdatePanelPromptForWorkshop(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	actor *CoursewareActorContext,
	request *models.UpdateCoursewareComicPanelPromptRequest,
) (*models.CoursewareComicPanelView, error) {
	if request == nil ||
		request.ExpectedVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	courseware, scopedActor, err :=
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

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	projectID =
		strings.TrimSpace(
			projectID,
		)

	panelID =
		strings.TrimSpace(
			panelID,
		)

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			courseware.ID,
			projectID,
			panelID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		request.ExpectedVersion {
		return nil,
			repository.
				ErrCoursewareComicPanelConflict
	}

	visualPrompt :=
		strings.TrimSpace(
			request.VisualPrompt,
		)

	if visualPrompt == "" {
		return nil,
			ErrCoursewareComicPromptInvalid
	}

	aoci, err :=
		utils.ParseImageAOCI(
			request.AOCIText,
		)
	if err != nil ||
		aoci.ImageKey !=
			panel.ImageKey ||
		aoci.IndexType !=
			models.CWImageIndexTypeImage {
		return nil,
			ErrCoursewareComicPromptInvalid
	}

	projectPanels, err :=
		repository.ListCoursewareComicPanels(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareComicEditedRelations(
			panel,
			projectPanels,
			aoci.Relations,
		); err != nil {
		return nil, err
	}

	aoci.ContinuityLevel = 3

	aoci.UsageRole =
		models.CWImageUsageStory

	aoci.ExportText =
		appendCoursewareComicHardConstraint(
			aoci.ExportText,
			"图片内不生成文字，对话和教学文字由HTML与SVG覆盖层渲染",
		)

	aoci.NegativeText =
		appendCoursewareComicHardConstraint(
			aoci.NegativeText,
			"禁止可读文字、字幕、公式、标签、Logo、水印和伪字符",
		)

	formattedAOCI, err :=
		utils.FormatImageAOCI(
			aoci,
		)
	if err != nil {
		return nil,
			ErrCoursewareComicPromptInvalid
	}

	relations := make(
		[]models.CoursewareComicPanelRelation,
		0,
		len(aoci.Relations),
	)

	for _, relation :=
		range aoci.Relations {
		relations = append(
			relations,
			models.CoursewareComicPanelRelation{
				TargetImageKey:
					relation.TargetImageKey,
				RelationCode:
					relation.RelationCode,
				InheritMask:
					relation.InheritMask,
				SemanticNote:
					relation.SemanticNote,
			},
		)
	}

	relationsJSON, err :=
		json.Marshal(
			relations,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化漫画关系失败: %w",
				err,
			)
	}

	visualPrompt =
		appendCoursewareComicHardConstraint(
			visualPrompt,
			"画面中不得出现任何文字、字幕、公式、标签、Logo、水印或伪字符",
		)

	negativePrompt :=
		appendCoursewareComicHardConstraint(
			request.NegativePrompt,
			"禁止文字、字幕、公式、标签、Logo、水印、伪字符和人物身份漂移",
		)

	updated, err :=
		repository.
			UpdateCoursewareComicPanelPromptForWorkshopIfUnchanged(
				ctx,
				courseware.ID,
				projectID,
				panelID,
				scopedActor.UserID,
				request.ExpectedVersion,
				visualPrompt,
				negativePrompt,
				formattedAOCI,
				string(relationsJSON),
			)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicPanelBrowserView(
		ctx,
		courseware.ID,
		updated,
	)
}

func buildCoursewareComicPanelBrowserView(
	ctx context.Context,
	coursewareID string,
	panel *models.CoursewareComicPanel,
) (*models.CoursewareComicPanelView, error) {
	view, err :=
		buildCoursewareComicPanelView(
			panel,
		)
	if err != nil {
		return nil, err
	}

	view.CurrentAssetURL =
		loadCoursewareComicBrowserAssetURL(
			ctx,
			coursewareID,
			view.CurrentAssetID,
		)

	return view, nil
}

func loadCoursewareComicBrowserAssetURL(
	ctx context.Context,
	coursewareID string,
	assetID *string,
) string {
	if assetID == nil ||
		strings.TrimSpace(
			*assetID,
		) == "" {
		return ""
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			strings.TrimSpace(
				*assetID,
			),
		)

	if err != nil ||
		asset == nil ||
		asset.CoursewareID !=
			strings.TrimSpace(
				coursewareID,
			) ||
		asset.AssetType !=
			models.CWAssetTypeImage {
		return ""
	}

	return resolveAssetPublicURL(
		asset,
	)
}
