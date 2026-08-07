package repository

// assistant_deployment_scan.go
//
// 本文件集中定义教学智能体部署和不可变版本的数据库扫描协议、
// 仓储错误以及不涉及数据库的基础持久化校验。
//
// 安全边界：
//   - 完整提示词和上下文快照只进入后端内部模型；
//   - 公开编号查询使用独立的最小运行字段集合；
//   - JSONB字段统一读取为文本，由后续Service按明确结构解析；
//   - nullable助手ID和插槽ID使用临时字符串转换，避免空UUID扫描歧义；
//   - 本文件不执行权限推断、不调用AI、不返回浏览器响应。

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/models"
)

const (
	// PostgreSQL INTEGER上限。部署版本号存储在INTEGER字段中。
	assistantDeploymentMaxVersion = int(^uint32(0) >> 1)
)

var (
	// ErrAssistantDeploymentNotFound 表示部署不存在或不属于指定边界。
	ErrAssistantDeploymentNotFound = errors.New(
		"课件教学智能体部署不存在",
	)

	// ErrAssistantDeploymentVersionNotFound 表示指定不可变版本不存在。
	ErrAssistantDeploymentVersionNotFound = errors.New(
		"课件教学智能体部署版本不存在",
	)

	// ErrAssistantDeploymentPageAlreadyLive 表示页面已有active或paused部署。
	ErrAssistantDeploymentPageAlreadyLive = errors.New(
		"当前课件页面已经存在未撤销部署",
	)

	// ErrAssistantDeploymentPublicIDConflict 表示随机公开编号发生唯一冲突。
	ErrAssistantDeploymentPublicIDConflict = errors.New(
		"课件教学智能体公开部署编号冲突",
	)

	// ErrAssistantDeploymentRevoked 表示部署已经进入不可恢复的撤销状态。
	ErrAssistantDeploymentRevoked = errors.New(
		"课件教学智能体部署已经撤销",
	)

	// ErrAssistantDeploymentStateConflict 表示当前状态不允许执行目标操作。
	ErrAssistantDeploymentStateConflict = errors.New(
		"课件教学智能体部署状态冲突",
	)

	// ErrAssistantDeploymentInvalidRecord 表示仓储写入参数不完整。
	ErrAssistantDeploymentInvalidRecord = errors.New(
		"课件教学智能体部署记录无效",
	)
)

// assistantDeploymentSelectColumns 是完整内部部署扫描字段。
//
// 调用SQL必须使用assistant_deployments别名d。
const assistantDeploymentSelectColumns = `
	d.id::text,
	d.public_id,
	COALESCE(d.slot_id::text, ''),
	d.courseware_id::text,
	d.page_id::text,
	d.owner_user_id::text,
	d.school_id::text,
	d.education_domain,
	d.current_version,
	d.access_mode,
	d.status,
	d.daily_call_limit,
	d.per_session_turn_limit,
	COALESCE(d.allowed_origins_json::text, '[]'),
	d.valid_from,
	d.valid_until,
	d.created_at,
	d.updated_at`

// assistantDeploymentRuntimeSelectColumns 是public_id运行查询所需最小字段。
//
// 运行时不需要编辑态slot_id，因此第三列固定为空字符串以复用统一扫描器。
const assistantDeploymentRuntimeSelectColumns = `
	d.id::text,
	d.public_id,
	''::text,
	d.courseware_id::text,
	d.page_id::text,
	d.owner_user_id::text,
	d.school_id::text,
	d.education_domain,
	d.current_version,
	d.access_mode,
	d.status,
	d.daily_call_limit,
	d.per_session_turn_limit,
	COALESCE(d.allowed_origins_json::text, '[]'),
	d.valid_from,
	d.valid_until,
	d.created_at,
	d.updated_at`

// assistantDeploymentVersionSelectColumns 是完整内部版本扫描字段。
//
// 调用SQL必须使用assistant_deployment_versions别名v。
const assistantDeploymentVersionSelectColumns = `
	v.deployment_id::text,
	v.version,
	COALESCE(v.assistant_id::text, ''),
	v.assistant_prompt_snapshot,
	v.assistant_prompt_hash,
	COALESCE(v.teaching_plan_json::text, '{}'),
	COALESCE(v.context_snapshot_json::text, '{}'),
	v.context_snapshot_hash,
	v.page_html_hash,
	COALESCE(v.courseware_snapshot_json::text, '{}'),
	v.created_by::text,
	v.created_at`

// scanAssistantDeployment 扫描完整内部部署记录。
func scanAssistantDeployment(
	row interface {
		Scan(dest ...interface{}) error
	},
) (
	*models.AssistantDeployment,
	error,
) {
	deployment := &models.AssistantDeployment{}
	slotID := ""

	err := row.Scan(
		&deployment.ID,
		&deployment.PublicID,
		&slotID,
		&deployment.CoursewareID,
		&deployment.PageID,
		&deployment.OwnerUserID,
		&deployment.SchoolID,
		&deployment.EducationDomain,
		&deployment.CurrentVersion,
		&deployment.AccessMode,
		&deployment.Status,
		&deployment.DailyCallLimit,
		&deployment.PerSessionTurnLimit,
		&deployment.AllowedOriginsJSON,
		&deployment.ValidFrom,
		&deployment.ValidUntil,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(slotID) != "" {
		deployment.SlotID = &slotID
	}

	deployment.PublicID =
		strings.TrimSpace(deployment.PublicID)
	deployment.EducationDomain =
		strings.ToLower(
			strings.TrimSpace(
				deployment.EducationDomain,
			),
		)
	deployment.AccessMode =
		strings.TrimSpace(
			deployment.AccessMode,
		)
	deployment.Status =
		strings.TrimSpace(
			deployment.Status,
		)
	deployment.AllowedOriginsJSON =
		normalizeAssistantDeploymentAllowedOriginsJSON(
			deployment.AllowedOriginsJSON,
		)

	return deployment, nil
}

// scanAssistantDeploymentVersion 扫描完整后端版本快照。
func scanAssistantDeploymentVersion(
	row interface {
		Scan(dest ...interface{}) error
	},
) (
	*models.AssistantDeploymentVersion,
	error,
) {
	version := &models.AssistantDeploymentVersion{}
	assistantID := ""

	err := row.Scan(
		&version.DeploymentID,
		&version.Version,
		&assistantID,
		&version.AssistantPromptSnapshot,
		&version.AssistantPromptHash,
		&version.TeachingPlanJSON,
		&version.ContextSnapshotJSON,
		&version.ContextSnapshotHash,
		&version.PageHTMLHash,
		&version.CoursewareSnapshotJSON,
		&version.CreatedBy,
		&version.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(assistantID) != "" {
		version.AssistantID = &assistantID
	}

	return version, nil
}

// validateAssistantDeploymentCreateRecord 校验首个部署事务的必要字段。
func validateAssistantDeploymentCreateRecord(
	deployment *models.AssistantDeployment,
	version *models.AssistantDeploymentVersion,
) error {
	if deployment == nil ||
		version == nil {
		return ErrAssistantDeploymentInvalidRecord
	}

	if deployment.SlotID == nil ||
		strings.TrimSpace(*deployment.SlotID) == "" ||
		strings.TrimSpace(deployment.CoursewareID) == "" ||
		strings.TrimSpace(deployment.PageID) == "" ||
		strings.TrimSpace(deployment.OwnerUserID) == "" ||
		strings.TrimSpace(deployment.SchoolID) == "" {
		return ErrAssistantDeploymentInvalidRecord
	}

	if !models.IsTeachingEducationDomain(
		strings.ToLower(
			strings.TrimSpace(
				deployment.EducationDomain,
			),
		),
	) {
		return ErrAssistantDeploymentInvalidRecord
	}

	if strings.TrimSpace(
		deployment.AccessMode,
	) !=
		models.AssistantDeploymentAccessOriginAllowlist ||
		strings.TrimSpace(
			deployment.Status,
		) !=
			models.AssistantDeploymentStatusActive {
		return ErrAssistantDeploymentInvalidRecord
	}

	if deployment.DailyCallLimit < 1 ||
		deployment.DailyCallLimit > 100000 ||
		deployment.PerSessionTurnLimit < 1 ||
		deployment.PerSessionTurnLimit > 100 {
		return ErrAssistantDeploymentInvalidRecord
	}

	return validateAssistantDeploymentVersionRecord(
		version,
		false,
	)
}

// validateAssistantDeploymentVersionRecord 校验版本快照必要字段。
//
// requireIdentity=false用于首个版本创建，部署ID和版本号由事务内部填写。
func validateAssistantDeploymentVersionRecord(
	version *models.AssistantDeploymentVersion,
	requireIdentity bool,
) error {
	if version == nil {
		return ErrAssistantDeploymentInvalidRecord
	}

	if requireIdentity &&
		(strings.TrimSpace(version.DeploymentID) == "" ||
			version.Version <= 0 ||
			version.Version > assistantDeploymentMaxVersion) {
		return ErrAssistantDeploymentInvalidRecord
	}

	if strings.TrimSpace(
		version.AssistantPromptSnapshot,
	) == "" ||
		len(strings.TrimSpace(
			version.AssistantPromptHash,
		)) != 64 ||
		len(strings.TrimSpace(
			version.ContextSnapshotHash,
		)) != 64 ||
		len(strings.TrimSpace(
			version.PageHTMLHash,
		)) != 64 ||
		strings.TrimSpace(
			version.TeachingPlanJSON,
		) == "" ||
		strings.TrimSpace(
			version.ContextSnapshotJSON,
		) == "" ||
		strings.TrimSpace(
			version.CoursewareSnapshotJSON,
		) == "" ||
		strings.TrimSpace(
			version.CreatedBy,
		) == "" {
		return ErrAssistantDeploymentInvalidRecord
	}

	return nil
}

// normalizeAssistantDeploymentAllowedOriginsJSON 统一空来源列表。
func normalizeAssistantDeploymentAllowedOriginsJSON(
	raw string,
) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]"
	}

	return raw
}

// assistantDeploymentNextVersion 计算下一不可变版本号。
func assistantDeploymentNextVersion(
	current int,
) (
	int,
	error,
) {
	if current < 0 ||
		current >= assistantDeploymentMaxVersion {
		return 0,
			ErrAssistantDeploymentStateConflict
	}

	return current + 1, nil
}

// assistantDeploymentStatusTransitionAllowed 固定不可恢复状态机。
func assistantDeploymentStatusTransitionAllowed(
	current string,
	target string,
) bool {
	current =
		strings.TrimSpace(current)
	target =
		strings.TrimSpace(target)

	switch current {
	case models.AssistantDeploymentStatusActive:
		return target ==
			models.AssistantDeploymentStatusPaused ||
			target ==
				models.AssistantDeploymentStatusRevoked

	case models.AssistantDeploymentStatusPaused:
		return target ==
			models.AssistantDeploymentStatusActive ||
			target ==
				models.AssistantDeploymentStatusRevoked

	case models.AssistantDeploymentStatusRevoked:
		return false

	default:
		return false
	}
}

// assistantDeploymentConstraintName 提取PostgreSQL唯一约束名。
func assistantDeploymentConstraintName(
	err error,
) string {
	var pgError *pgconn.PgError

	if errors.As(
		err,
		&pgError,
	) {
		return strings.TrimSpace(
			pgError.ConstraintName,
		)
	}

	return ""
}

// wrapAssistantDeploymentWriteError 把已知数据库约束转为稳定仓储错误。
func wrapAssistantDeploymentWriteError(
	action string,
	err error,
) error {
	switch assistantDeploymentConstraintName(
		err,
	) {
	case "uq_assistant_deployments_public_id":
		return fmt.Errorf(
			"%w: %v",
			ErrAssistantDeploymentPublicIDConflict,
			err,
		)

	case "uq_assistant_deployments_page_live":
		return fmt.Errorf(
			"%w: %v",
			ErrAssistantDeploymentPageAlreadyLive,
			err,
		)

	default:
		return fmt.Errorf(
			"%s失败: %w",
			action,
			err,
		)
	}
}
