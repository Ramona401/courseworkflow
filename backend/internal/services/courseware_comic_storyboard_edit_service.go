package services

// courseware_comic_storyboard_edit_service.go — 教师端单格分镜安全编辑服务
//
// 本文件负责：
//   - 重新执行课件作者、课件写锁和项目归属校验；
//   - 只允许第二步、尚未确认且未开始图片生产的分镜修改；
//   - 校验教师可见教学字段的长度和必填关系；
//   - 根据教学字段确定性重建后端图片内容提示；
//   - 使用 panel.version CAS 保存，避免多标签页静默覆盖。
//
// 对白、旁白、题目与气泡继续由覆盖层编辑器维护。
// IAOCI、负面提示词、内部图片键和跨格关系不接受浏览器输入。

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareComicStoryboardPurposeMaxRunes      = 500
	coursewareComicStoryboardKnowledgeMaxRunes    = 2000
	coursewareComicStoryboardSceneMaxRunes        = 4000
	coursewareComicStoryboardActionMaxRunes       = 3000
	coursewareComicStoryboardCameraMaxRunes       = 1000
	coursewareComicStoryboardPresentationMaxRunes = 3000
)

type normalizedCoursewareComicStoryboardPanel struct {
	StoryPurpose          string
	KnowledgeClaim        string
	SceneText             string
	ActionText            string
	CameraText            string
	KnowledgePresentation string
}

func normalizeCoursewareComicStoryboardPanel(
	request *models.UpdateCoursewareComicStoryboardPanelRequest,
) (*normalizedCoursewareComicStoryboardPanel, error) {
	if request == nil || request.ExpectedVersion < 1 {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	normalized := &normalizedCoursewareComicStoryboardPanel{
		StoryPurpose:          strings.TrimSpace(request.StoryPurpose),
		KnowledgeClaim:        strings.TrimSpace(request.KnowledgeClaim),
		SceneText:             strings.TrimSpace(request.SceneText),
		ActionText:            strings.TrimSpace(request.ActionText),
		CameraText:            strings.TrimSpace(request.CameraText),
		KnowledgePresentation: strings.TrimSpace(request.KnowledgePresentation),
	}

	if normalized.StoryPurpose == "" ||
		normalized.KnowledgeClaim == "" ||
		(normalized.SceneText == "" && normalized.ActionText == "") {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	if utf8.RuneCountInString(normalized.StoryPurpose) >
		coursewareComicStoryboardPurposeMaxRunes ||
		utf8.RuneCountInString(normalized.KnowledgeClaim) >
			coursewareComicStoryboardKnowledgeMaxRunes ||
		utf8.RuneCountInString(normalized.SceneText) >
			coursewareComicStoryboardSceneMaxRunes ||
		utf8.RuneCountInString(normalized.ActionText) >
			coursewareComicStoryboardActionMaxRunes ||
		utf8.RuneCountInString(normalized.CameraText) >
			coursewareComicStoryboardCameraMaxRunes ||
		utf8.RuneCountInString(normalized.KnowledgePresentation) >
			coursewareComicStoryboardPresentationMaxRunes {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	return normalized, nil
}

// buildCoursewareComicStoryboardVisualPrompt
// 根据教师可见业务字段重建图片内容事实。
//
// 该文本只描述教学内容、场景、动作和镜头，不接受模型、供应商、密钥、
// 画风或IAOCI。实际画风、比例、清晰度和人物连续性仍由后端确认流程附加。
func buildCoursewareComicStoryboardVisualPrompt(
	panel *models.CoursewareComicPanel,
	normalized *normalizedCoursewareComicStoryboardPanel,
) string {
	if panel == nil || normalized == nil {
		return ""
	}

	return strings.Join(
		[]string{
			"【教师确认的最终分镜内容】",
			"以下教学分镜字段由教师在第二步修改，生成图片时优先于旧规划中的同类描述。",
			fmt.Sprintf("漫画格编号：%d", panel.PanelNo),
			"故事职责：" + normalized.StoryPurpose,
			"知识结论：" + normalized.KnowledgeClaim,
			"场景：" + normalized.SceneText,
			"人物动作：" + normalized.ActionText,
			"镜头：" + normalized.CameraText,
			"知识呈现：" + normalized.KnowledgePresentation,
			"只生成无文字画面；不得生成对白、旁白、题目、公式、标签、Logo、水印或伪字符。",
		},
		"\n",
	)
}

// UpdatePanelStoryboard 保存第二步中的单格分镜修改。
func (s *CoursewareComicProjectService) UpdatePanelStoryboard(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	actor *CoursewareActorContext,
	request *models.UpdateCoursewareComicStoryboardPanelRequest,
) (*models.CoursewareComicPanelView, error) {
	if s == nil || actor == nil {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	coursewareID = strings.TrimSpace(coursewareID)
	projectID = strings.TrimSpace(projectID)
	panelID = strings.TrimSpace(panelID)

	if coursewareID == "" || projectID == "" || panelID == "" {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	normalized, err := normalizeCoursewareComicStoryboardPanel(request)
	if err != nil {
		return nil, err
	}

	courseware, scopedActor, err := s.resolveCoursewareService().
		LoadCoursewareForOwnerRuntime(ctx, coursewareID, actor)
	if err != nil {
		return nil, err
	}

	if err := validateCoursewareControlMutationState(courseware); err != nil {
		return nil, err
	}

	project, err := repository.GetCoursewareComicProjectByIDForUser(
		ctx,
		courseware.ID,
		projectID,
		scopedActor.UserID,
	)
	if err != nil {
		return nil, err
	}

	if project.Status != models.CWComicProjectStatusPlanned {
		return nil, repository.ErrCoursewareComicProjectNotEditable
	}

	workflow, err := repository.GetCoursewareComicWorkflowState(
		ctx,
		courseware.ID,
		project.ID,
		scopedActor.UserID,
	)
	if err != nil {
		return nil, err
	}

	if workflow.Stage != models.CWComicWorkflowStoryboard ||
		workflow.StoryboardConfirmedAt != nil {
		return nil, repository.ErrCoursewareComicProjectNotEditable
	}

	panel, err := repository.GetCoursewareComicPanelByIDForProject(
		ctx,
		courseware.ID,
		project.ID,
		panelID,
		scopedActor.UserID,
	)
	if err != nil {
		return nil, err
	}

	if panel.Version != request.ExpectedVersion {
		return nil, repository.ErrCoursewareComicPanelConflict
	}

	if panel.CurrentAssetID != nil ||
		panel.Status == models.CWComicPanelStatusGenerating ||
		panel.Status == models.CWComicPanelStatusGenerated {
		return nil, repository.ErrCoursewareComicProjectNotEditable
	}

	visualPrompt := buildCoursewareComicStoryboardVisualPrompt(
		panel,
		normalized,
	)
	if strings.TrimSpace(visualPrompt) == "" {
		return nil, ErrCoursewareComicProjectInvalidRequest
	}

	updated, err := repository.UpdateCoursewareComicStoryboardPanelIfUnchanged(
		ctx,
		courseware.ID,
		project.ID,
		panel.ID,
		scopedActor.UserID,
		request.ExpectedVersion,
		normalized.StoryPurpose,
		normalized.KnowledgeClaim,
		normalized.SceneText,
		normalized.ActionText,
		normalized.CameraText,
		normalized.KnowledgePresentation,
		visualPrompt,
	)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicPanelView(updated)
}
