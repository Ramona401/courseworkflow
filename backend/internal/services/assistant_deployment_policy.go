package services

// assistant_deployment_policy.go
//
// 本文件定义课件教学智能体部署发布前的确定性策略校验与浏览器安全转换。
//
// 核心边界：
//   - 只有达到preview或confirmed生产状态的正式课件可以发布；
//   - submitted和in_pipeline继续复用课件核心控制面写锁；
//   - 插槽必须处于active状态，并保持MVP固定的右下角悬浮模式；
//   - 页面教学方案本身即可发布，已有AI助手只作为可选教学风格增强；
//   - 允许来源必须是精确Origin，禁止通配符、路径、查询参数和用户凭据；
//   - 外部来源只允许HTTPS，HTTP仅允许本机开发地址；
//   - 教师响应只包含部署元数据和哈希，不包含提示词或上下文正文。

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"tedna/internal/models"
)

const (
	assistantDeploymentMaxAllowedOrigins = 20
	assistantDeploymentMaxOriginRunes    = 512
)

var (
	ErrAssistantDeploymentActorRequired = errors.New(
		"缺少可信课件教学智能体部署操作者",
	)
	ErrAssistantDeploymentCoursewareNotPublishable = errors.New(
		"课件当前状态不允许发布教学智能体",
	)
	ErrAssistantDeploymentSlotRequired = errors.New(
		"当前页面缺少可发布的教学智能体插槽",
	)
	ErrAssistantDeploymentSlotInactive = errors.New(
		"当前页面教学智能体插槽已停用",
	)

	// ErrAssistantDeploymentAssistantRequired 保留用于兼容历史错误识别。
	//
	// 当前发布流程不再要求插槽必须绑定已有AI助手。
	ErrAssistantDeploymentAssistantRequired = errors.New(
		"发布教学智能体前必须选择可用AI助手",
	)

	// ErrAssistantDeploymentAssistantPromptRequired 仅表示教师主动选择的
	// 已有助手当前缺少可发布提示词，不表示页面方案本身不能发布。
	ErrAssistantDeploymentAssistantPromptRequired = errors.New(
		"选择的AI助手缺少可发布提示词",
	)

	ErrAssistantDeploymentSchoolRequired = errors.New(
		"当前教师未绑定可用学校，不能发布教学智能体",
	)
	ErrAssistantDeploymentPolicyInvalid = errors.New(
		"课件教学智能体部署策略无效",
	)
	ErrAssistantDeploymentOriginInvalid = errors.New(
		"课件教学智能体允许来源格式无效",
	)
	ErrAssistantDeploymentSnapshotInvalid = errors.New(
		"课件教学智能体发布快照无效",
	)
	ErrAssistantDeploymentStoredPolicyInvalid = errors.New(
		"课件教学智能体已保存运行策略无效",
	)
	ErrAssistantDeploymentSlotChanged = errors.New(
		"部署关联的教学智能体插槽已删除或替换",
	)
)

// assistantDeploymentNormalizedPolicy 是写仓储前的规范化策略。
type assistantDeploymentNormalizedPolicy struct {
	DailyCallLimit      int
	PerSessionTurnLimit int
	AllowedOrigins      []string
	AllowedOriginsJSON  string
	ValidUntil          *time.Time
}

// assistantDeploymentPolicySnapshot 是版本审计快照中的发布策略。
type assistantDeploymentPolicySnapshot struct {
	AccessMode       string     `json:"access_mode"`
	DailyCallLimit   int        `json:"daily_call_limit"`
	SessionTurnLimit int        `json:"per_session_turn_limit"`
	AllowedOrigins   []string   `json:"allowed_origins"`
	ValidUntil       *time.Time `json:"valid_until"`
}

// validateAssistantDeploymentActor 校验服务调用者身份。
func validateAssistantDeploymentActor(
	actor *CoursewareActorContext,
) error {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return ErrAssistantDeploymentActorRequired
	}

	return nil
}

// validateAssistantDeploymentPublishableCourseware 校验课件生产状态。
func validateAssistantDeploymentPublishableCourseware(
	courseware *models.Courseware,
) error {
	if courseware == nil ||
		!models.IsTeachingEducationDomain(
			strings.ToLower(
				strings.TrimSpace(courseware.EducationDomain),
			),
		) {
		return ErrCoursewareEducationDomainInvalid
	}

	if err := validateCoursewareControlMutationState(
		courseware,
	); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrAssistantDeploymentCoursewareNotPublishable,
			err,
		)
	}

	switch strings.TrimSpace(courseware.Status) {
	case "preview", "confirmed":
		return nil
	default:
		return ErrAssistantDeploymentCoursewareNotPublishable
	}
}

// validateAssistantDeploymentSlot 校验编辑态插槽仍可发布。
//
// AssistantID允许为空。页面教学方案、答案保护规则和确定性页面上下文
// 已经构成完整可运行智能体；教师主动选择的已有助手只增强教学风格。
func validateAssistantDeploymentSlot(
	slot *models.CoursewareAssistantSlotView,
	coursewareID string,
	pageID string,
) error {
	if slot == nil ||
		strings.TrimSpace(slot.ID) == "" ||
		strings.TrimSpace(slot.CoursewareID) !=
			strings.TrimSpace(coursewareID) ||
		strings.TrimSpace(slot.PageID) !=
			strings.TrimSpace(pageID) {
		return ErrAssistantDeploymentSlotRequired
	}

	if strings.TrimSpace(slot.Status) !=
		models.CoursewareAssistantSlotStatusActive {
		return ErrAssistantDeploymentSlotInactive
	}

	if !models.IsValidCoursewareAssistantDisplayMode(
		slot.DisplayMode,
	) ||
		!models.IsValidCoursewareAssistantPosition(
			slot.DisplayPosition,
		) {
		return ErrAssistantDeploymentSlotRequired
	}

	if strings.TrimSpace(slot.Title) == "" ||
		strings.TrimSpace(slot.WelcomeMessage) == "" ||
		strings.TrimSpace(slot.TeachingRole) == "" ||
		strings.TrimSpace(slot.LearningObjective) == "" ||
		slot.GuidancePlan.AnswerLeakPolicy.DirectAnswerAllowed {
		return ErrAssistantDeploymentSlotRequired
	}

	return nil
}

// normalizeAssistantDeploymentPolicy 校验并规范额度、来源和有效期。
func normalizeAssistantDeploymentPolicy(
	dailyCallLimit int,
	perSessionTurnLimit int,
	allowedOrigins []string,
	validUntil *time.Time,
	now time.Time,
) (
	*assistantDeploymentNormalizedPolicy,
	error,
) {
	if dailyCallLimit < 1 ||
		dailyCallLimit > 100000 ||
		perSessionTurnLimit < 1 ||
		perSessionTurnLimit > 100 {
		return nil,
			ErrAssistantDeploymentPolicyInvalid
	}

	if validUntil != nil &&
		!validUntil.After(now) {
		return nil,
			ErrAssistantDeploymentPolicyInvalid
	}

	normalizedOrigins, err :=
		normalizeAssistantDeploymentAllowedOrigins(
			allowedOrigins,
		)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(
		normalizedOrigins,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: 序列化允许来源失败: %v",
				ErrAssistantDeploymentPolicyInvalid,
				err,
			)
	}

	return &assistantDeploymentNormalizedPolicy{
		DailyCallLimit:      dailyCallLimit,
		PerSessionTurnLimit: perSessionTurnLimit,
		AllowedOrigins:      normalizedOrigins,
		AllowedOriginsJSON:  string(encoded),
		ValidUntil:          validUntil,
	}, nil
}

// normalizeAssistantDeploymentAllowedOrigins 生成稳定去重的精确Origin列表。
func normalizeAssistantDeploymentAllowedOrigins(
	origins []string,
) (
	[]string,
	error,
) {
	if len(origins) == 0 ||
		len(origins) > assistantDeploymentMaxAllowedOrigins {
		return nil,
			ErrAssistantDeploymentOriginInvalid
	}

	seen := make(
		map[string]struct{},
		len(origins),
	)
	result := make(
		[]string,
		0,
		len(origins),
	)

	for _, raw := range origins {
		normalized, err :=
			normalizeAssistantDeploymentOrigin(
				raw,
			)
		if err != nil {
			return nil, err
		}

		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		result = append(
			result,
			normalized,
		)
	}

	if len(result) == 0 {
		return nil,
			ErrAssistantDeploymentOriginInvalid
	}

	sort.Strings(result)

	return result, nil
}

// normalizeAssistantDeploymentOrigin 校验单个Origin。
func normalizeAssistantDeploymentOrigin(
	raw string,
) (
	string,
	error,
) {
	raw = strings.TrimSpace(raw)

	if raw == "" ||
		len([]rune(raw)) > assistantDeploymentMaxOriginRunes ||
		strings.Contains(raw, "*") {
		return "",
			ErrAssistantDeploymentOriginInvalid
	}

	parsed, err := url.Parse(raw)
	if err != nil ||
		parsed == nil ||
		parsed.Opaque != "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawPath != "" {
		return "",
			ErrAssistantDeploymentOriginInvalid
	}

	scheme := strings.ToLower(
		strings.TrimSpace(
			parsed.Scheme,
		),
	)
	hostname := strings.ToLower(
		strings.TrimSpace(
			parsed.Hostname(),
		),
	)
	port := strings.TrimSpace(
		parsed.Port(),
	)

	if hostname == "" {
		return "",
			ErrAssistantDeploymentOriginInvalid
	}

	if scheme != "https" &&
		!(scheme == "http" &&
			isAssistantDeploymentLocalHostname(
				hostname,
			)) {
		return "",
			ErrAssistantDeploymentOriginInvalid
	}

	if (scheme == "https" && port == "443") ||
		(scheme == "http" && port == "80") {
		port = ""
	}

	host := hostname

	if strings.Contains(
		hostname,
		":",
	) {
		host = "[" + hostname + "]"
	}

	if port != "" {
		host = net.JoinHostPort(
			hostname,
			port,
		)
	}

	return scheme + "://" + host,
		nil
}

// isAssistantDeploymentLocalHostname 判断HTTP开发来源是否仅限本机。
func isAssistantDeploymentLocalHostname(
	hostname string,
) bool {
	hostname = strings.ToLower(
		strings.TrimSpace(hostname),
	)

	if hostname == "localhost" {
		return true
	}

	ip := net.ParseIP(hostname)

	return ip != nil &&
		ip.IsLoopback()
}

// assistantDeploymentAllowedOriginsFromJSON 严格解析已保存来源。
func assistantDeploymentAllowedOriginsFromJSON(
	raw string,
) (
	[]string,
	error,
) {
	var origins []string

	if err := json.Unmarshal(
		[]byte(
			strings.TrimSpace(raw),
		),
		&origins,
	); err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrAssistantDeploymentStoredPolicyInvalid,
				err,
			)
	}

	if origins == nil {
		origins = []string{}
	}

	return origins, nil
}

// assistantDeploymentViewFromRecord 转换教师端安全部署元数据。
func assistantDeploymentViewFromRecord(
	deployment *models.AssistantDeployment,
) (
	*models.AssistantDeploymentView,
	error,
) {
	if deployment == nil {
		return nil,
			ErrAssistantDeploymentStoredPolicyInvalid
	}

	origins, err :=
		assistantDeploymentAllowedOriginsFromJSON(
			deployment.AllowedOriginsJSON,
		)
	if err != nil {
		return nil, err
	}

	return &models.AssistantDeploymentView{
		ID:                  deployment.ID,
		PublicID:            deployment.PublicID,
		SlotID:              deployment.SlotID,
		CoursewareID:        deployment.CoursewareID,
		PageID:              deployment.PageID,
		EducationDomain:     deployment.EducationDomain,
		CurrentVersion:      deployment.CurrentVersion,
		AccessMode:          deployment.AccessMode,
		Status:              deployment.Status,
		DailyCallLimit:      deployment.DailyCallLimit,
		PerSessionTurnLimit: deployment.PerSessionTurnLimit,
		AllowedOrigins:      origins,
		ValidFrom:           deployment.ValidFrom,
		ValidUntil:          deployment.ValidUntil,
		CreatedAt:           deployment.CreatedAt,
		UpdatedAt:           deployment.UpdatedAt,
	}, nil
}

// assistantDeploymentVersionViewFromRecord 转换版本哈希元数据。
func assistantDeploymentVersionViewFromRecord(
	version *models.AssistantDeploymentVersion,
) *models.AssistantDeploymentVersionView {
	if version == nil {
		return nil
	}

	return &models.AssistantDeploymentVersionView{
		Version:             version.Version,
		AssistantID:         version.AssistantID,
		AssistantPromptHash: version.AssistantPromptHash,
		ContextSnapshotHash: version.ContextSnapshotHash,
		PageHTMLHash:        version.PageHTMLHash,
		CreatedAt:           version.CreatedAt,
	}
}
