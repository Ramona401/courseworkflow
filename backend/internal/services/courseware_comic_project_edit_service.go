package services

// courseware_comic_project_edit_service.go — 漫画单格编辑服务
//
// 本文件负责：
//   - 保存教师调整后的旁白、对白、题目和覆盖层布局；
//   - 保存单格图片视觉提示词、负面提示词和完整IAOCI；
//   - 使用panel.version进行CAS并发保护；
//   - 保留AI原始文字，防止浏览器覆盖恢复事实源；
//   - 强制补回无文字、无Logo、无水印和人物连续性约束。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// UpdatePanelOverlay 保存教师调整后的文字和气泡排版。
func (s *CoursewareComicProjectService) UpdatePanelOverlay(
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

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				panelID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		request.ExpectedVersion {
		return nil,
			repository.ErrCoursewareComicPanelConflict
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
		repository.UpdateCoursewareComicPanelOverlayIfUnchanged(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				panelID,
			),
			scopedActor.UserID,
			request.ExpectedVersion,
			strings.TrimSpace(
				request.NarrationText,
			),
			string(dialoguesJSON),
			string(overlayJSON),
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicPanelView(
		updated,
	)
}

// UpdatePanelPrompt 保存教师调整后的单格提示词和IAOCI。
func (s *CoursewareComicProjectService) UpdatePanelPrompt(
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

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				panelID,
			),
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		request.ExpectedVersion {
		return nil,
			repository.ErrCoursewareComicPanelConflict
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
		aoci.ImageKey != panel.ImageKey ||
		aoci.IndexType !=
			models.CWImageIndexTypeImage {
		return nil,
			ErrCoursewareComicPromptInvalid
	}

	projectPanels, err :=
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

	for _, relation := range aoci.Relations {
		relations = append(
			relations,
			models.CoursewareComicPanelRelation{
				TargetImageKey: relation.TargetImageKey,
				RelationCode:   relation.RelationCode,
				InheritMask:    relation.InheritMask,
				SemanticNote:   relation.SemanticNote,
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
		repository.UpdateCoursewareComicPanelPromptIfUnchanged(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				panelID,
			),
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

	return buildCoursewareComicPanelView(
		updated,
	)
}
