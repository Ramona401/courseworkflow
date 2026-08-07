package services

// courseware_comic_free_knowledge.go — 教师自由输入知识点兼容层
//
// 本文件把教师输入转换为现有漫画项目能够稳定保存的知识快照：
//   - 不要求教材版本、册次、单元或课标知识点编码；
//   - 不修改数据库表结构；
//   - 使用项目级随机UUID作为虚拟单元ID，不引用教材表；
//   - 生成一个稳定的教师输入知识点快照；
//   - 自动选择标题、叙事模式、视觉风格、格数和布局；
//   - 旧教材创建流程不受影响。

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"tedna/internal/models"
)

const (
	coursewareComicCustomPublisher =
		"教师自定义知识点"

	coursewareComicCustomSemester =
		"自由输入"

	coursewareComicCustomKPCode =
		"TEACHER-INPUT"

	coursewareComicKnowledgeTextMaxRunes =
		8000
)

type coursewareComicFreeKnowledgeSource struct {
	Publisher string
	Semester  string
	UnitID    string

	UnitSnapshotJSON      string
	KnowledgePointsJSON   string
	KnowledgeContent      string
	KnowledgeDisplayTitle string
}

// applyCoursewareComicAutomaticDefaults 自动补齐老师无需选择的漫画参数。
func applyCoursewareComicAutomaticDefaults(
	request *models.CreateCoursewareComicProjectRequest,
	courseware *models.Courseware,
) {
	if request == nil {
		return
	}

	knowledgeText :=
		strings.TrimSpace(
			request.KnowledgeText,
		)

	subject := ""
	grade := ""
	coursewareTitle := ""

	if courseware != nil {
		subject =
			strings.TrimSpace(
				courseware.Subject,
			)

		grade =
			strings.TrimSpace(
				courseware.Grade,
			)

		coursewareTitle =
			strings.TrimSpace(
				courseware.Title,
			)
	}

	if strings.TrimSpace(
		request.Title,
	) == "" {
		request.Title =
			buildCoursewareComicAutomaticTitle(
				knowledgeText,
				coursewareTitle,
			)
	}

	if strings.TrimSpace(
		request.NarrativeMode,
	) == "" {
		request.NarrativeMode =
			resolveCoursewareComicAutomaticNarrative(
				subject,
				knowledgeText,
			)
	}

	if strings.TrimSpace(
		request.VisualStyle,
	) == "" {
		request.VisualStyle =
			resolveCoursewareComicAutomaticVisualStyle(
				subject,
				grade,
				knowledgeText,
			)
	}

	if request.PanelCount == 0 {
		request.PanelCount =
			resolveCoursewareComicAutomaticPanelCount(
				knowledgeText,
			)
	}

	if strings.TrimSpace(
		request.LayoutMode,
	) == "" {
		request.LayoutMode =
			resolveCoursewareComicAutomaticLayout(
				request.PanelCount,
			)
	}
}

// buildCoursewareComicFreeKnowledgeSource 把自由输入转换为兼容旧流程的快照。
func buildCoursewareComicFreeKnowledgeSource(
	courseware *models.Courseware,
	knowledgeText string,
) (*coursewareComicFreeKnowledgeSource, error) {
	knowledgeText =
		strings.TrimSpace(
			knowledgeText,
		)

	if courseware == nil ||
		knowledgeText == "" ||
		utf8.RuneCountInString(
			knowledgeText,
		) > coursewareComicKnowledgeTextMaxRunes {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	displayTitle :=
		firstCoursewareComicKnowledgeLine(
			knowledgeText,
			60,
		)

	if displayTitle == "" {
		displayTitle =
			"教师输入知识点"
	}

	unitID :=
		uuid.NewString()

	unitSnapshot :=
		models.CoursewareComicTextbookUnitSnapshot{
			ID:
				unitID,
			Publisher:
				coursewareComicCustomPublisher,
			GradeNum:
				parseCoursewareComicGradeNum(
					courseware.Grade,
				),
			Semester:
				coursewareComicCustomSemester,
			UnitNumber:
				0,
			UnitTitle:
				"教师自定义知识点",
			LessonTitle:
				displayTitle,
			ContentSummary:
				truncateCoursewareComicFreeKnowledgeRunes(
					knowledgeText,
					4000,
				),
			KPCodes:
				[]string{
					coursewareComicCustomKPCode,
				},
		}

	knowledgePoints :=
		[]models.CoursewareComicKnowledgePointSnapshot{
			{
				KPCode:
					coursewareComicCustomKPCode,
				KPName:
					displayTitle,
				ContentRequirement:
					knowledgeText,
				AcademicRequirement:
					"",
				TeachingHint:
					"结合当前课件学科、年级和已有内容，准确规划故事、场景、知识结论与形成性评价。",
				DepthLevel:
					0,
				SourceRef:
					"teacher_input",
			},
		}

	unitJSON, err :=
		json.Marshal(
			unitSnapshot,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化教师知识点单元快照失败: %w",
				err,
			)
	}

	knowledgeJSON, err :=
		json.Marshal(
			knowledgePoints,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化教师知识点快照失败: %w",
				err,
			)
	}

	return &coursewareComicFreeKnowledgeSource{
		Publisher:
			coursewareComicCustomPublisher,
		Semester:
			coursewareComicCustomSemester,
		UnitID:
			unitID,
		UnitSnapshotJSON:
			string(unitJSON),
		KnowledgePointsJSON:
			string(knowledgeJSON),
		KnowledgeContent:
			"来源：教师自由输入\n知识点与教学要求：\n" +
				knowledgeText,
		KnowledgeDisplayTitle:
			displayTitle,
	}, nil
}

func buildCoursewareComicAutomaticTitle(
	knowledgeText string,
	coursewareTitle string,
) string {
	seed :=
		firstCoursewareComicKnowledgeLine(
			knowledgeText,
			48,
		)

	if seed == "" {
		seed =
			firstCoursewareComicKnowledgeLine(
				coursewareTitle,
				48,
			)
	}

	if seed == "" {
		return "知识点漫画"
	}

	if strings.Contains(
		seed,
		"漫画",
	) {
		return truncateCoursewareComicFreeKnowledgeRunes(
			seed,
			200,
		)
	}

	return truncateCoursewareComicFreeKnowledgeRunes(
		seed+"·知识点漫画",
		200,
	)
}

func resolveCoursewareComicAutomaticNarrative(
	subject string,
	knowledgeText string,
) string {
	contextText :=
		strings.ToLower(
			strings.TrimSpace(
				subject+" "+knowledgeText,
			),
		)

	switch {
	case containsAnyCoursewareComicText(
		contextText,
		"道德",
		"法治",
		"政治",
		"公民",
		"权利",
		"义务",
		"规则",
	):
		return "civic_case"

	case containsAnyCoursewareComicText(
		contextText,
		"旅行",
		"路线",
		"城市",
		"景点",
		"地理",
		"连云港",
	):
		return "travel_adventure"

	case containsAnyCoursewareComicText(
		contextText,
		"实验",
		"探究",
		"观察",
		"验证",
		"为什么",
		"如何判断",
	):
		return "inquiry_mystery"

	default:
		return "knowledge_story"
	}
}

func resolveCoursewareComicAutomaticVisualStyle(
	subject string,
	grade string,
	knowledgeText string,
) string {
	contextText :=
		strings.ToLower(
			strings.TrimSpace(
				subject+" "+knowledgeText,
			),
		)

	gradeNum :=
		parseCoursewareComicGradeNum(
			grade,
		)

	switch {
	case containsAnyCoursewareComicText(
		contextText,
		"历史",
		"地理",
		"城市",
		"旅行",
		"遗址",
		"文物",
	):
		return "realistic_illustration"

	case containsAnyCoursewareComicText(
		contextText,
		"语文",
		"故事",
		"童话",
		"道德",
		"法治",
	):
		return "warm_storybook"

	case gradeNum > 0 &&
		gradeNum <= 3:
		return "warm_storybook"

	default:
		return "science_encyclopedia"
	}
}

func resolveCoursewareComicAutomaticPanelCount(
	knowledgeText string,
) int {
	length :=
		utf8.RuneCountInString(
			strings.TrimSpace(
				knowledgeText,
			),
		)

	switch {
	case length > 260:
		return 6

	case length > 100:
		return 5

	default:
		return 4
	}
}

func resolveCoursewareComicAutomaticLayout(
	panelCount int,
) string {
	switch {
	case panelCount >= 7:
		return models.CWComicLayoutCarousel

	case panelCount == 5:
		return models.CWComicLayoutSpotlight

	default:
		return models.CWComicLayoutGrid
	}
}

func firstCoursewareComicKnowledgeLine(
	value string,
	limit int,
) string {
	for _, line :=
		range strings.Split(
			strings.TrimSpace(
				value,
			),
			"\n",
		) {
		line =
			strings.TrimSpace(
				line,
			)

		if line != "" {
			return truncateCoursewareComicFreeKnowledgeRunes(
				line,
				limit,
			)
		}
	}

	return ""
}

func truncateCoursewareComicFreeKnowledgeRunes(
	value string,
	limit int,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if limit <= 0 {
		return ""
	}

	runes :=
		[]rune(value)

	if len(runes) <= limit {
		return value
	}

	return string(
		runes[:limit],
	)
}

func containsAnyCoursewareComicText(
	value string,
	needles ...string,
) bool {
	for _, needle :=
		range needles {
		if strings.Contains(
			value,
			strings.ToLower(
				strings.TrimSpace(
					needle,
				),
			),
		) {
			return true
		}
	}

	return false
}
