package services

// courseware_comic_project_views.go — 知识点漫画浏览器安全视图构建
//
// 本文件只负责把内部项目和分格实体转换成浏览器可读取的安全视图：
//   - 解析教材、知识点、人物、对白、关系与覆盖层JSON；
//   - 生成资源公开URL所需的稳定视图字段；
//   - 按格号排序详情中的漫画分格；
//   - 不执行授权判断，不修改数据库，也不暴露生成提示词以外的新字段。

import (
	"encoding/json"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// BuildCoursewareComicDetailView 构建浏览器安全详情。
func BuildCoursewareComicDetailView(
	project *models.CoursewareComicProject,
	panels []*models.CoursewareComicPanel,
) (*models.CoursewareComicProjectDetailView, error) {
	projectView, err :=
		buildCoursewareComicProjectView(
			project,
		)
	if err != nil {
		return nil, err
	}

	panelViews := make(
		[]*models.CoursewareComicPanelView,
		0,
		len(panels),
	)

	for _, panel := range panels {
		view, viewErr :=
			buildCoursewareComicPanelView(
				panel,
			)
		if viewErr != nil {
			return nil, viewErr
		}

		panelViews = append(
			panelViews,
			view,
		)
	}

	sort.Slice(
		panelViews,
		func(left int, right int) bool {
			return panelViews[left].PanelNo <
				panelViews[right].PanelNo
		},
	)

	return &models.CoursewareComicProjectDetailView{
		Project: projectView,
		Panels:  panelViews,
	}, nil
}

func buildCoursewareComicProjectView(
	project *models.CoursewareComicProject,
) (*models.CoursewareComicProjectView, error) {
	if project == nil {
		return nil,
			repository.ErrCoursewareComicProjectNotFound
	}

	var unit models.CoursewareComicTextbookUnitSnapshot
	var kps []models.CoursewareComicKnowledgePointSnapshot
	var bible models.CoursewareComicCharacterBible

	if err := json.Unmarshal(
		[]byte(project.TextbookUnitSnapshotJSON),
		&unit,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(
		[]byte(project.KnowledgePointsJSON),
		&kps,
	); err != nil {
		return nil, err
	}

	bibleJSON :=
		strings.TrimSpace(
			project.CharacterBibleJSON,
		)

	if bibleJSON == "" ||
		bibleJSON == "{}" {
		bible =
			models.CoursewareComicCharacterBible{
				Version:    1,
				Characters: []models.CoursewareComicCharacter{},
			}
	} else if err := json.Unmarshal(
		[]byte(bibleJSON),
		&bible,
	); err != nil {
		return nil, err
	}

	return &models.CoursewareComicProjectView{
		ID:               project.ID,
		CoursewareID:     project.CoursewareID,
		EducationDomain:  project.EducationDomain,
		Title:            project.Title,
		Subject:          project.Subject,
		Grade:            project.Grade,
		Publisher:        project.PublisherSnapshot,
		Semester:         project.SemesterSnapshot,
		TextbookUnit:     unit,
		KnowledgePoints:  kps,
		KnowledgeContent: project.KnowledgeContentSnapshot,
		TeacherFocus:     project.TeacherFocus,
		AssistantID:      project.AssistantID,
		NarrativeMode:    project.NarrativeMode,
		VisualStyle:      project.VisualStyle,
		PanelCount:       project.PanelCount,
		LayoutMode:       project.LayoutMode,
		PageLayout: json.RawMessage(
			project.PageLayoutJSON,
		),
		InteractionConfig: json.RawMessage(
			project.InteractionConfigJSON,
		),
		StyleAOCIText:  project.StyleAOCIText,
		CharacterBible: bible,
		ContinuityLedger: json.RawMessage(
			project.ContinuityLedgerJSON,
		),
		CharacterSheetAssetID:      project.CharacterSheetAssetID,
		Status:                     project.Status,
		InsertedPageID:             project.InsertedPageID,
		InsertedPageNumberSnapshot: project.InsertedPageNumberSnapshot,
		Version:                    project.Version,
		LastError:                  project.LastError,
		CreatedAt:                  project.CreatedAt,
		UpdatedAt:                  project.UpdatedAt,
	}, nil
}

func buildCoursewareComicPanelView(
	panel *models.CoursewareComicPanel,
) (*models.CoursewareComicPanelView, error) {
	if panel == nil {
		return nil,
			repository.ErrCoursewareComicPanelNotFound
	}

	var characterIDs []string
	var dialogues []models.CoursewareComicDialogue
	var relations []models.CoursewareComicPanelRelation
	var overlay models.CoursewareComicOverlayDocument

	if err := json.Unmarshal(
		[]byte(panel.CharacterIDsJSON),
		&characterIDs,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(
		[]byte(panel.DialoguesJSON),
		&dialogues,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(
		[]byte(panel.RelationsJSON),
		&relations,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(
		[]byte(panel.OverlayDocumentJSON),
		&overlay,
	); err != nil {
		return nil, err
	}

	return &models.CoursewareComicPanelView{
		ID:                    panel.ID,
		ProjectID:             panel.ProjectID,
		PanelNo:               panel.PanelNo,
		ImageKey:              panel.ImageKey,
		StoryPurpose:          panel.StoryPurpose,
		KnowledgeClaim:        panel.KnowledgeClaim,
		SceneText:             panel.SceneText,
		CharacterIDs:          characterIDs,
		ActionText:            panel.ActionText,
		CameraText:            panel.CameraText,
		NarrationText:         panel.NarrationText,
		Dialogues:             dialogues,
		KnowledgePresentation: panel.KnowledgePresentation,
		VisualPrompt:          panel.VisualPrompt,
		NegativePrompt:        panel.NegativePrompt,
		AOCIText:              panel.AOCIText,
		Relations:             relations,
		OverlayDocument:       overlay,
		OverlayVersion:        panel.OverlayVersion,
		Status:                panel.Status,
		CurrentAssetID:        panel.CurrentAssetID,
		Version:               panel.Version,
		LastError:             panel.LastError,
		CreatedAt:             panel.CreatedAt,
		UpdatedAt:             panel.UpdatedAt,
	}, nil
}
