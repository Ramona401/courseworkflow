package models

// education_domain.go — 教育域与教育画像统一模型
//
// 教育域用于隔离不同教育场景下的课程、页面语义、知识依据和共享资源：
//   k12          中小学
//   vocational   职业教育
//   adult        成人教育
//   mixed        区域、教育局、系统管理员等跨域管理上下文
//
// 组织教育域与资源教育域是两套不同的合法值集合：
//   - 组织教育域：k12 / vocational / adult / mixed；
//   - 资源教育域：k12 / vocational / adult / common。
//
// mixed不是普通教学资源所属域，只用于跨域管理组织和管理账号；
// common只用于跨具体教学域共享的公共资源，不能写入organizations.education_domain。

import "strings"

const (
	EducationDomainK12        = "k12"
	EducationDomainVocational = "vocational"
	EducationDomainAdult      = "adult"
	EducationDomainMixed      = "mixed"

	// EducationDomainCommon 是教学资源专用公共域。
	//
	// common资源可被k12、vocational和adult三个具体教学域使用，
	// 但它不是组织教育域，也不能作为普通用户的当前教学教育域。
	EducationDomainCommon = "common"
)

// ValidEducationDomains organizations.education_domain允许值。
var ValidEducationDomains = []string{
	EducationDomainK12,
	EducationDomainVocational,
	EducationDomainAdult,
	EducationDomainMixed,
}

// ValidResourceEducationDomains 教案、课件、配方、AI助手等教学资源允许值。
//
// mixed只表示跨域管理上下文，绝不能写入教学资源；
// common表示可被三个具体教学域共同使用的公共资源。
var ValidResourceEducationDomains = []string{
	EducationDomainK12,
	EducationDomainVocational,
	EducationDomainAdult,
	EducationDomainCommon,
}

// IsValidEducationDomain 判断是否为合法组织教育域。
func IsValidEducationDomain(domain string) bool {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	for _, item := range ValidEducationDomains {
		if item == normalized {
			return true
		}
	}
	return false
}

// IsTeachingEducationDomain 判断是否为可以承载具体教学资源的教育域。
func IsTeachingEducationDomain(domain string) bool {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	return normalized == EducationDomainK12 ||
		normalized == EducationDomainVocational ||
		normalized == EducationDomainAdult
}

// IsResourceEducationDomain 判断是否为合法教学资源教育域。
//
// 与IsValidEducationDomain刻意分开：
//   - mixed只属于组织与管理上下文；
//   - common只属于教学资源。
//
// 任何资源写入、读取授权和运行时匹配都应使用本函数，不得复用组织域校验。
func IsResourceEducationDomain(domain string) bool {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	for _, item := range ValidResourceEducationDomains {
		if item == normalized {
			return true
		}
	}
	return false
}

// NormalizeEducationDomain 规范化教育域。
// 非法值默认回退k12，保证存量用户和异常组织不会因教育域缺失而无法使用平台。
func NormalizeEducationDomain(domain string) string {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	if IsValidEducationDomain(normalized) {
		return normalized
	}
	return EducationDomainK12
}

// NormalizeResourceEducationDomain 规范化教学资源教育域。
//
// 合法值原样返回；非法值回退k12，仅用于兼容存量数据与非授权型展示。
// 权限判断不得依赖该回退行为，应直接调用IsResourceEducationDomain或
// ResourceEducationDomainMatches，以免把非法值静默当成k12放行。
func NormalizeResourceEducationDomain(domain string) string {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	if IsResourceEducationDomain(normalized) {
		return normalized
	}
	return EducationDomainK12
}

// ResourceEducationDomainMatches 判断一个教学资源是否可在当前教育域中使用。
//
// 匹配规则：
//   - 具体教学域(k12/vocational/adult)：只允许同域资源或common资源；
//   - mixed管理上下文：允许跨域查看与管理全部合法资源；
//   - 非法资源域、非法当前域、current=common：一律拒绝。
//
// 重要：进入具体教案运行时，调用方必须先用lesson_plan.education_domain快照
// 覆盖mixed管理员Actor中的EducationDomain，再调用本函数。这样管理员虽然能跨域管理，
// 但在某一份具体教案里仍只能使用该教案所属域或common资源。
func ResourceEducationDomainMatches(resourceDomain string, currentDomain string) bool {
	resource := strings.ToLower(strings.TrimSpace(resourceDomain))
	current := strings.ToLower(strings.TrimSpace(currentDomain))

	if !IsResourceEducationDomain(resource) {
		return false
	}

	if current == EducationDomainMixed {
		return true
	}

	if !IsTeachingEducationDomain(current) {
		return false
	}

	return resource == EducationDomainCommon || resource == current
}

// EducationProfile 教育域前端语义与能力画像。
//
// 第一阶段只下发稳定的语义标签和能力开关，前端据此集中适配页面。
// 不在每个页面散落domain判断，也不在此处写死复杂业务表单。
type EducationProfile struct {
	Code                     string `json:"code"`
	Name                     string `json:"name"`
	SubjectLabel             string `json:"subject_label"`
	GradeLabel               string `json:"grade_label"`
	TopicLabel               string `json:"topic_label"`
	LessonPlanLabel          string `json:"lesson_plan_label"`
	UnitPlanLabel            string `json:"unit_plan_label"`
	LearnerProfileLabel      string `json:"learner_profile_label"`
	CourseOutlineLabel       string `json:"course_outline_label"`
	CurriculumEnabled        bool   `json:"curriculum_enabled"`
	PublisherEnabled         bool   `json:"publisher_enabled"`
	MajorEnabled             bool   `json:"major_enabled"`
	PracticalTrainingEnabled bool   `json:"practical_training_enabled"`
}

// EducationProfileForDomain 返回指定教育域的统一教育画像。
func EducationProfileForDomain(domain string) EducationProfile {
	switch NormalizeEducationDomain(domain) {
	case EducationDomainVocational:
		return EducationProfile{
			Code:                     EducationDomainVocational,
			Name:                     "职业教育",
			SubjectLabel:             "课程",
			GradeLabel:               "年级或学期",
			TopicLabel:               "教学主题或工作任务",
			LessonPlanLabel:          "教学设计",
			UnitPlanLabel:            "课程模块方案",
			LearnerProfileLabel:      "学习者情况",
			CourseOutlineLabel:       "教学依据",
			CurriculumEnabled:        false,
			PublisherEnabled:         false,
			MajorEnabled:             true,
			PracticalTrainingEnabled: true,
		}

	case EducationDomainAdult:
		return EducationProfile{
			Code:                     EducationDomainAdult,
			Name:                     "成人教育",
			SubjectLabel:             "培训类别",
			GradeLabel:               "学习基础",
			TopicLabel:               "培训主题",
			LessonPlanLabel:          "培训方案",
			UnitPlanLabel:            "培训项目方案",
			LearnerProfileLabel:      "学习者画像",
			CourseOutlineLabel:       "培训依据",
			CurriculumEnabled:        false,
			PublisherEnabled:         false,
			MajorEnabled:             false,
			PracticalTrainingEnabled: false,
		}

	case EducationDomainMixed:
		return EducationProfile{
			Code:                     EducationDomainMixed,
			Name:                     "跨域管理",
			SubjectLabel:             "课程",
			GradeLabel:               "学习层级",
			TopicLabel:               "教学主题",
			LessonPlanLabel:          "教学设计",
			UnitPlanLabel:            "教学方案",
			LearnerProfileLabel:      "学习者情况",
			CourseOutlineLabel:       "教学依据",
			CurriculumEnabled:        true,
			PublisherEnabled:         true,
			MajorEnabled:             true,
			PracticalTrainingEnabled: true,
		}

	default:
		return EducationProfile{
			Code:                     EducationDomainK12,
			Name:                     "中小学",
			SubjectLabel:             "学科",
			GradeLabel:               "年级",
			TopicLabel:               "课题",
			LessonPlanLabel:          "教案",
			UnitPlanLabel:            "大单元方案",
			LearnerProfileLabel:      "班级学情",
			CourseOutlineLabel:       "课程大纲",
			CurriculumEnabled:        true,
			PublisherEnabled:         true,
			MajorEnabled:             false,
			PracticalTrainingEnabled: false,
		}
	}
}

// UserEducationContext 当前用户确定性的教学组织和教育域上下文。
//
// 多校归属规则：
//  1. mixed管理组织不覆盖k12/vocational/adult具体教学学校；
//  2. 多个具体学校若属于同一教育域，允许存在并选择确定性的首个组织；
//  3. 若同时存在不同具体教育域，DomainConflict=true，由日志和后续治理处理；
//  4. OrganizationID用于学校私有课程目录和后续资源教育域快照。
type UserEducationContext struct {
	OrganizationID   string          `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	OrganizationLogo string          `json:"organization_logo"`
	EducationDomain  string          `json:"education_domain"`
	PortalModules    map[string]bool `json:"portal_modules"`
	DomainConflict   bool            `json:"domain_conflict"`
}
