package repository

// assistant_deployment_repo.go
//
// 本文件实现教学智能体部署的首发创建和查询。
//
// 首次发布采用单一事务：
//   1. 使用crypto/rand生成不可预测public_id；
//   2. 插入assistant_deployments并直接设置current_version=1；
//   3. 插入第一条不可变assistant_deployment_versions快照；
//   4. 任一步失败时整笔事务回滚，不留下current_version=0的半部署。
//
// 查询边界：
//   - 内部按ID查询返回后端完整部署记录；
//   - 教师管理查询绑定owner_user_id；
//   - 页面活动部署查询同时绑定courseware、page和owner；
//   - public_id查询只读取运行会话所需最小字段；
//   - 不通过public_id返回提示词或上下文快照。
//
// 本文件不执行教师权限推断，课件作者权限由Service负责。

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

const (
	// 32个随机字节经RawURLBase64编码后得到43个URL安全字符。
	assistantDeploymentPublicIDRandomBytes = 32

	// 极低概率随机碰撞最多重试5次。
	assistantDeploymentPublicIDMaxAttempts = 5
)

// generateAssistantDeploymentPublicID 生成不可预测的URL安全公开编号。
func generateAssistantDeploymentPublicID() (
	string,
	error,
) {
	randomBytes := make(
		[]byte,
		assistantDeploymentPublicIDRandomBytes,
	)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {
		return "",
			fmt.Errorf(
				"生成教学智能体公开编号随机数失败: %w",
				err,
			)
	}

	return base64.RawURLEncoding.
			EncodeToString(
				randomBytes,
			),
		nil
}

// CreateAssistantDeploymentWithFirstVersion 原子创建部署和版本1。
//
// PublicID始终由本仓储覆盖生成，不信任调用方提供值。
// 成功后会回填deployment.ID、PublicID、CurrentVersion和时间字段，
// 同时回填version.DeploymentID、Version和CreatedAt。
func CreateAssistantDeploymentWithFirstVersion(
	ctx context.Context,
	deployment *models.AssistantDeployment,
	version *models.AssistantDeploymentVersion,
) error {
	if deployment == nil ||
		version == nil {
		return ErrAssistantDeploymentInvalidRecord
	}

	deployment.AccessMode =
		strings.TrimSpace(
			deployment.AccessMode,
		)
	deployment.Status =
		strings.TrimSpace(
			deployment.Status,
		)

	if deployment.AccessMode == "" {
		deployment.AccessMode =
			models.AssistantDeploymentAccessOriginAllowlist
	}

	if deployment.Status == "" {
		deployment.Status =
			models.AssistantDeploymentStatusActive
	}

	deployment.EducationDomain =
		strings.ToLower(
			strings.TrimSpace(
				deployment.EducationDomain,
			),
		)
	deployment.AllowedOriginsJSON =
		normalizeAssistantDeploymentAllowedOriginsJSON(
			deployment.AllowedOriginsJSON,
		)

	if err :=
		validateAssistantDeploymentCreateRecord(
			deployment,
			version,
		); err != nil {
		return err
	}

	for attempt := 0; attempt <
		assistantDeploymentPublicIDMaxAttempts; attempt++ {
		publicID, err :=
			generateAssistantDeploymentPublicID()
		if err != nil {
			return err
		}

		deployment.PublicID =
			publicID

		err =
			createAssistantDeploymentWithFirstVersionOnce(
				ctx,
				deployment,
				version,
			)
		if err == nil {
			return nil
		}

		if assistantDeploymentConstraintName(
			err,
		) ==
			"uq_assistant_deployments_public_id" {
			continue
		}

		return wrapAssistantDeploymentWriteError(
			"创建课件教学智能体部署",
			err,
		)
	}

	return ErrAssistantDeploymentPublicIDConflict
}

// createAssistantDeploymentWithFirstVersionOnce 执行一次首发事务。
func createAssistantDeploymentWithFirstVersionOnce(
	ctx context.Context,
	deployment *models.AssistantDeployment,
	version *models.AssistantDeploymentVersion,
) error {
	tx, err :=
		database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启教学智能体首发事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err =
		tx.QueryRow(
			ctx,
			`
			INSERT INTO assistant_deployments (
				public_id,
				slot_id,
				courseware_id,
				page_id,
				owner_user_id,
				school_id,
				education_domain,
				current_version,
				access_mode,
				status,
				daily_call_limit,
				per_session_turn_limit,
				allowed_origins_json,
				valid_from,
				valid_until,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				1,
				$8,
				'active',
				$9,
				$10,
				$11::jsonb,
				COALESCE($12, NOW()),
				$13,
				NOW(),
				NOW()
			)
			RETURNING
				id::text,
				current_version,
				access_mode,
				status,
				valid_from,
				valid_until,
				created_at,
				updated_at`,
			deployment.PublicID,
			deployment.SlotID,
			deployment.CoursewareID,
			deployment.PageID,
			deployment.OwnerUserID,
			deployment.SchoolID,
			deployment.EducationDomain,
			deployment.AccessMode,
			deployment.DailyCallLimit,
			deployment.PerSessionTurnLimit,
			deployment.AllowedOriginsJSON,
			deployment.ValidFrom,
			deployment.ValidUntil,
		).Scan(
			&deployment.ID,
			&deployment.CurrentVersion,
			&deployment.AccessMode,
			&deployment.Status,
			&deployment.ValidFrom,
			&deployment.ValidUntil,
			&deployment.CreatedAt,
			&deployment.UpdatedAt,
		)
	if err != nil {
		return err
	}

	version.DeploymentID =
		deployment.ID
	version.Version = 1

	if err :=
		insertAssistantDeploymentVersionTx(
			ctx,
			tx,
			version,
		); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交教学智能体首发事务失败: %w",
			err,
		)
	}

	return nil
}

// GetAssistantDeploymentByID 按内部ID读取完整部署记录。
func GetAssistantDeploymentByID(
	ctx context.Context,
	deploymentID string,
) (
	*models.AssistantDeployment,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	deployment, err :=
		scanAssistantDeployment(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.id = $1`,
				deploymentID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantDeploymentNotFound
		}

		return nil,
			fmt.Errorf(
				"查询课件教学智能体部署失败: %w",
				err,
			)
	}

	return deployment, nil
}

// GetAssistantDeploymentForOwner 按部署和付费创建者双重边界读取。
func GetAssistantDeploymentForOwner(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
) (
	*models.AssistantDeployment,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)

	if deploymentID == "" ||
		ownerUserID == "" {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	deployment, err :=
		scanAssistantDeployment(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.id = $1
				   AND d.owner_user_id = $2`,
				deploymentID,
				ownerUserID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantDeploymentNotFound
		}

		return nil,
			fmt.Errorf(
				"查询教师课件教学智能体部署失败: %w",
				err,
			)
	}

	return deployment, nil
}

// GetLiveAssistantDeploymentByPageForOwner 查询页面当前active或paused部署。
func GetLiveAssistantDeploymentByPageForOwner(
	ctx context.Context,
	coursewareID string,
	pageID string,
	ownerUserID string,
) (
	*models.AssistantDeployment,
	error,
) {
	coursewareID =
		strings.TrimSpace(coursewareID)
	pageID =
		strings.TrimSpace(pageID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)

	if coursewareID == "" ||
		pageID == "" ||
		ownerUserID == "" {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	deployment, err :=
		scanAssistantDeployment(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.courseware_id = $1
				   AND d.page_id = $2
				   AND d.owner_user_id = $3
				   AND d.status IN ('active', 'paused')
				 LIMIT 1`,
				coursewareID,
				pageID,
				ownerUserID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantDeploymentNotFound
		}

		return nil,
			fmt.Errorf(
				"查询页面当前教学智能体部署失败: %w",
				err,
			)
	}

	return deployment, nil
}

// ListAssistantDeploymentsByCoursewareForOwner 返回课件全部部署历史。
//
// 包含revoked记录，便于教师查看历史，但不返回任何版本提示词正文。
func ListAssistantDeploymentsByCoursewareForOwner(
	ctx context.Context,
	coursewareID string,
	ownerUserID string,
) (
	[]*models.AssistantDeployment,
	error,
) {
	coursewareID =
		strings.TrimSpace(coursewareID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)

	if coursewareID == "" ||
		ownerUserID == "" {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	rows, err :=
		database.DB.Query(
			ctx,
			`SELECT `+
				assistantDeploymentSelectColumns+
				`
			 FROM assistant_deployments d
			 WHERE d.courseware_id = $1
			   AND d.owner_user_id = $2
			 ORDER BY d.updated_at DESC, d.created_at DESC`,
			coursewareID,
			ownerUserID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询课件教学智能体部署列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	deployments := make(
		[]*models.AssistantDeployment,
		0,
	)

	for rows.Next() {
		deployment, scanErr :=
			scanAssistantDeployment(rows)
		if scanErr != nil {
			return nil,
				fmt.Errorf(
					"扫描课件教学智能体部署失败: %w",
					scanErr,
				)
		}

		deployments = append(
			deployments,
			deployment,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历课件教学智能体部署失败: %w",
				err,
			)
	}

	return deployments, nil
}

// GetAssistantDeploymentRuntimeByPublicID 按公开编号读取最小运行字段。
//
// 本查询不读取slot_id、助手提示词、教学方案或上下文快照。
// 运行Service必须继续校验状态、有效期、来源域名和current_version。
func GetAssistantDeploymentRuntimeByPublicID(
	ctx context.Context,
	publicID string,
) (
	*models.AssistantDeployment,
	error,
) {
	publicID =
		strings.TrimSpace(publicID)
	if publicID == "" {
		return nil,
			ErrAssistantDeploymentNotFound
	}

	deployment, err :=
		scanAssistantDeployment(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentRuntimeSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.public_id = $1`,
				publicID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantDeploymentNotFound
		}

		return nil,
			fmt.Errorf(
				"查询公开课件教学智能体部署失败: %w",
				err,
			)
	}

	return deployment, nil
}
