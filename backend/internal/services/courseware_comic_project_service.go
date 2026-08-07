package services

// courseware_comic_project_service.go — 知识点漫画项目创建服务
//
// 创建入口同时兼容两种来源：
//
// 一键自由输入模式：
//   - 浏览器只提交knowledge_text；
//   - 教材、单元和课标知识点均不必选择；
//   - 服务端从正式课件读取学科、年级、教育域和视觉锚点；
//   - 服务端自动补齐标题、叙事模式、视觉风格、格数和布局；
//   - 教师原文被固化为稳定知识快照，随后交给统一AI规划。
//
// 旧教材模式：
//   - 仍支持publisher、textbook_unit_id和kp_codes；
//   - 后端继续重新查询教材和课标知识点，浏览器不能伪造快照。
//
// 项目列表、详情和单格编辑方法位于：
// courseware_comic_project_edit_service.go。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareComicMaxKnowledgePoints   = 12
	coursewareComicMaxTeacherFocusRunes = 8000
	coursewareComicMaxTitleRunes        = 200
)

var (
	ErrCoursewareComicProjectServiceUnavailable = errors.New(
		"知识点漫画项目服务不可用",
	)

	ErrCoursewareComicProjectInvalidRequest = errors.New(
		"知识点漫画项目请求无效",
	)

	ErrCoursewareComicProjectK12Required = errors.New(
		"知识点漫画第一版仅支持K12课件",
	)

	ErrCoursewareComicProjectGradeInvalid = errors.New(
		"课件年级无法匹配教材年级",
	)

	ErrCoursewareComicProjectUnitNotFound = errors.New(
		"选择的教材单元不存在或与课件不匹配",
	)

	ErrCoursewareComicProjectKnowledgePointInvalid = errors.New(
		"选择的知识点不存在或与课件不匹配",
	)

	ErrCoursewareComicProjectKnowledgePointOutsideUnit = errors.New(
		"选择的知识点不属于当前教材单元",
	)

	ErrCoursewareComicProjectTeacherFocusTooLong = errors.New(
		"漫画补充要求长度超过上限",
	)

	ErrCoursewareComicOverlayInvalid = errors.New(
		"漫画文字与气泡文档无效",
	)

	ErrCoursewareComicPromptInvalid = errors.New(
		"漫画格图片提示词或IAOCI无效",
	)
)

// CoursewareComicProjectService 是漫画项目业务服务。
type CoursewareComicProjectService struct {
	coursewareService *CoursewareService
}

// NewCoursewareComicProjectService 创建默认服务。
func NewCoursewareComicProjectService() *CoursewareComicProjectService {
	return &CoursewareComicProjectService{
		coursewareService: NewCoursewareService(),
	}
}

// NewCoursewareComicProjectServiceWithDependencies 创建可注入依赖的服务。
func NewCoursewareComicProjectServiceWithDependencies(
	coursewareService *CoursewareService,
) *CoursewareComicProjectService {
	return &CoursewareComicProjectService{
		coursewareService: coursewareService,
	}
}

// CreateProject 创建漫画项目并固化可信知识快照。
//
// KnowledgeText非空时优先使用自由输入模式。
// KnowledgeText为空时继续走旧教材和课标知识点模式。
func (s *CoursewareComicProjectService) CreateProject(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	request *models.CreateCoursewareComicProjectRequest,
) (*models.CoursewareComicProjectView, error) {
	if s == nil {
		return nil,
			ErrCoursewareComicProjectServiceUnavailable
	}

	if request == nil {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	if coursewareID == "" ||
		actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	normalizeCoursewareComicCreateRequest(
		request,
	)

	if utf8.RuneCountInString(
		request.TeacherFocus,
	) > coursewareComicMaxTeacherFocusRunes {
		return nil,
			ErrCoursewareComicProjectTeacherFocusTooLong
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
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

	if strings.ToLower(
		strings.TrimSpace(
			courseware.EducationDomain,
		),
	) != models.EducationDomainK12 {
		return nil,
			ErrCoursewareComicProjectK12Required
	}

	// 标题、叙事、风格、格数和布局均由服务端自动补齐。
	// 旧客户端显式提交合法值时仍保留其选择。
	applyCoursewareComicAutomaticDefaults(
		request,
		courseware,
	)

	normalizeCoursewareComicCreateRequest(
		request,
	)

	if request.Title == "" ||
		utf8.RuneCountInString(
			request.Title,
		) > coursewareComicMaxTitleRunes {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	var (
		publisherSnapshot        string
		semesterSnapshot         string
		textbookUnitID           *string
		textbookUnitSnapshotJSON string
		knowledgePointsJSON      string
		knowledgeContentSnapshot string
	)

	if request.KnowledgeText != "" {
		freeSource, freeErr :=
			buildCoursewareComicFreeKnowledgeSource(
				courseware,
				request.KnowledgeText,
			)
		if freeErr != nil {
			return nil, freeErr
		}

		publisherSnapshot =
			freeSource.Publisher

		semesterSnapshot =
			freeSource.Semester

		unitID :=
			freeSource.UnitID

		textbookUnitID =
			&unitID

		textbookUnitSnapshotJSON =
			freeSource.UnitSnapshotJSON

		knowledgePointsJSON =
			freeSource.KnowledgePointsJSON

		knowledgeContentSnapshot =
			freeSource.KnowledgeContent
	} else {
		legacySource, legacyErr :=
			buildCoursewareComicLegacyKnowledgeSource(
				ctx,
				courseware,
				request,
			)
		if legacyErr != nil {
			return nil, legacyErr
		}

		publisherSnapshot =
			legacySource.Publisher

		semesterSnapshot =
			legacySource.Semester

		textbookUnitID =
			legacySource.UnitID

		textbookUnitSnapshotJSON =
			legacySource.UnitSnapshotJSON

		knowledgePointsJSON =
			legacySource.KnowledgePointsJSON

		knowledgeContentSnapshot =
			legacySource.KnowledgeContent
	}

	pageLayoutJSON,
		interactionJSON,
		err :=
		buildCoursewareComicDefaultConfigs(
			request.PanelCount,
			request.LayoutMode,
		)
	if err != nil {
		return nil, err
	}

	project :=
		&models.CoursewareComicProject{
			CoursewareID:    courseware.ID,
			CreatedBy:       scopedActor.UserID,
			EducationDomain: models.EducationDomainK12,

			Title: request.Title,
			Subject: strings.TrimSpace(
				courseware.Subject,
			),
			Grade: strings.TrimSpace(
				courseware.Grade,
			),

			PublisherSnapshot:        publisherSnapshot,
			SemesterSnapshot:         semesterSnapshot,
			TextbookUnitID:           textbookUnitID,
			TextbookUnitSnapshotJSON: textbookUnitSnapshotJSON,

			KnowledgePointsJSON:      knowledgePointsJSON,
			KnowledgeContentSnapshot: knowledgeContentSnapshot,
			TeacherFocus:             request.TeacherFocus,

			AssistantID: normalizeCoursewareComicOptionalID(
				request.AssistantID,
			),
			NarrativeMode: request.NarrativeMode,
			VisualStyle:   request.VisualStyle,
			PanelCount:    request.PanelCount,
			LayoutMode:    request.LayoutMode,

			PageLayoutJSON:        pageLayoutJSON,
			InteractionConfigJSON: interactionJSON,

			StyleAOCIText:        "",
			CharacterBibleJSON:   "{}",
			ContinuityLedgerJSON: "{}",

			Status:  models.CWComicProjectStatusDraft,
			Version: 1,
		}

	if err :=
		repository.CreateCoursewareComicProject(
			ctx,
			project,
		); err != nil {
		return nil, err
	}

	return buildCoursewareComicProjectView(
		project,
	)
}

type coursewareComicLegacyKnowledgeSource struct {
	Publisher string
	Semester  string
	UnitID    *string

	UnitSnapshotJSON    string
	KnowledgePointsJSON string
	KnowledgeContent    string
}

// buildCoursewareComicLegacyKnowledgeSource 保留旧教材创建协议。
func buildCoursewareComicLegacyKnowledgeSource(
	ctx context.Context,
	courseware *models.Courseware,
	request *models.CreateCoursewareComicProjectRequest,
) (*coursewareComicLegacyKnowledgeSource, error) {
	if courseware == nil ||
		request == nil ||
		request.Publisher == "" ||
		request.TextbookUnitID == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	gradeNum :=
		parseCoursewareComicGradeNum(
			courseware.Grade,
		)
	if gradeNum <= 0 {
		return nil,
			ErrCoursewareComicProjectGradeInvalid
	}

	unit, err :=
		loadCoursewareComicTextbookUnit(
			ctx,
			courseware,
			request,
			gradeNum,
		)
	if err != nil {
		return nil, err
	}

	kpCodes, err :=
		normalizeCoursewareComicKPCodes(
			request.KPCodes,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareComicKPCodesBelongToUnit(
			unit,
			kpCodes,
		); err != nil {
		return nil, err
	}

	knowledgePoints, err :=
		loadCoursewareComicKnowledgePoints(
			ctx,
			courseware,
			gradeNum,
			kpCodes,
		)
	if err != nil {
		return nil, err
	}

	unitSnapshot, err :=
		buildCoursewareComicUnitSnapshot(
			unit,
		)
	if err != nil {
		return nil, err
	}

	kpSnapshots :=
		buildCoursewareComicKPSnapshots(
			knowledgePoints,
		)

	unitJSON, err :=
		json.Marshal(
			unitSnapshot,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化教材单元快照失败: %w",
				err,
			)
	}

	kpJSON, err :=
		json.Marshal(
			kpSnapshots,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化知识点快照失败: %w",
				err,
			)
	}

	unitID :=
		unit.ID

	return &coursewareComicLegacyKnowledgeSource{
		Publisher:           unit.Publisher,
		Semester:            unit.Semester,
		UnitID:              &unitID,
		UnitSnapshotJSON:    string(unitJSON),
		KnowledgePointsJSON: string(kpJSON),
		KnowledgeContent: buildCoursewareComicKnowledgeContent(
			unit,
			knowledgePoints,
		),
	}, nil
}

func normalizeCoursewareComicCreateRequest(
	request *models.CreateCoursewareComicProjectRequest,
) {
	if request == nil {
		return
	}

	request.KnowledgeText =
		strings.TrimSpace(
			request.KnowledgeText,
		)

	request.Title =
		strings.TrimSpace(
			request.Title,
		)

	request.Publisher =
		strings.TrimSpace(
			request.Publisher,
		)

	request.Semester =
		strings.TrimSpace(
			request.Semester,
		)

	request.TextbookUnitID =
		strings.TrimSpace(
			request.TextbookUnitID,
		)

	request.NarrativeMode =
		strings.TrimSpace(
			request.NarrativeMode,
		)

	request.VisualStyle =
		strings.TrimSpace(
			request.VisualStyle,
		)

	request.LayoutMode =
		strings.TrimSpace(
			request.LayoutMode,
		)

	request.TeacherFocus =
		strings.TrimSpace(
			request.TeacherFocus,
		)
}

func (s *CoursewareComicProjectService) resolveCoursewareService() *CoursewareService {
	if s != nil &&
		s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}
