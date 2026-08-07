package repository

// assistant_deployment_version_repo.go
//
// 本文件实现部署版本追加和读取。
//
// 版本追加事务严格按顺序执行：
//   1. 使用deployment_id、courseware_id、page_id和owner_user_id定位部署；
//   2. SELECT ... FOR UPDATE锁定部署行；
//   3. 从锁定行current_version计算下一版本号；
//   4. INSERT新的不可变版本快照；
//   5. UPDATE部署current_version；
//   6. 提交事务。
//
// 同一部署的并发发布会串行等待同一部署行锁，因此不会计算出重复版本号。
// 本文件不提供版本UPDATE或DELETE方法，数据库触发器也拒绝版本UPDATE。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// insertAssistantDeploymentVersionTx 在已有事务中插入不可变版本。
func insertAssistantDeploymentVersionTx(
	ctx context.Context,
	tx pgx.Tx,
	version *models.AssistantDeploymentVersion,
) error {
	if tx == nil {
		return ErrAssistantDeploymentInvalidRecord
	}

	if err :=
		validateAssistantDeploymentVersionRecord(
			version,
			true,
		); err != nil {
		return err
	}

	err :=
		tx.QueryRow(
			ctx,
			`
			INSERT INTO assistant_deployment_versions (
				deployment_id,
				version,
				assistant_id,
				assistant_prompt_snapshot,
				assistant_prompt_hash,
				teaching_plan_json,
				context_snapshot_json,
				context_snapshot_hash,
				page_html_hash,
				courseware_snapshot_json,
				created_by,
				created_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6::jsonb,
				$7::jsonb,
				$8,
				$9,
				$10::jsonb,
				$11,
				NOW()
			)
			RETURNING created_at`,
			version.DeploymentID,
			version.Version,
			version.AssistantID,
			version.AssistantPromptSnapshot,
			strings.TrimSpace(
				version.AssistantPromptHash,
			),
			version.TeachingPlanJSON,
			version.ContextSnapshotJSON,
			strings.TrimSpace(
				version.ContextSnapshotHash,
			),
			strings.TrimSpace(
				version.PageHTMLHash,
			),
			version.CoursewareSnapshotJSON,
			version.CreatedBy,
		).Scan(
			&version.CreatedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"插入教学智能体不可变部署版本失败: %w",
			err,
		)
	}

	return nil
}

// AppendAssistantDeploymentVersion 原子追加新版本并更新current_version。
//
// paused部署允许发布新版本，但仍保持paused；
// revoked部署拒绝任何新版本。
func AppendAssistantDeploymentVersion(
	ctx context.Context,
	deploymentID string,
	coursewareID string,
	pageID string,
	ownerUserID string,
	version *models.AssistantDeploymentVersion,
) (
	*models.AssistantDeploymentVersion,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	coursewareID =
		strings.TrimSpace(coursewareID)
	pageID =
		strings.TrimSpace(pageID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)

	if deploymentID == "" ||
		coursewareID == "" ||
		pageID == "" ||
		ownerUserID == "" ||
		version == nil {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	tx, err :=
		database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体版本发布事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentVersion int
		currentStatus  string
	)

	err =
		tx.QueryRow(
			ctx,
			`
			SELECT
				current_version,
				status
			FROM assistant_deployments
			WHERE id = $1
			  AND courseware_id = $2
			  AND page_id = $3
			  AND owner_user_id = $4
			FOR UPDATE`,
			deploymentID,
			coursewareID,
			pageID,
			ownerUserID,
		).Scan(
			&currentVersion,
			&currentStatus,
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
				"锁定教学智能体部署版本序列失败: %w",
				err,
			)
	}

	if strings.TrimSpace(
		currentStatus,
	) ==
		models.AssistantDeploymentStatusRevoked {
		return nil,
			ErrAssistantDeploymentRevoked
	}

	nextVersion, err :=
		assistantDeploymentNextVersion(
			currentVersion,
		)
	if err != nil {
		return nil, err
	}

	version.DeploymentID =
		deploymentID
	version.Version =
		nextVersion

	if err :=
		insertAssistantDeploymentVersionTx(
			ctx,
			tx,
			version,
		); err != nil {
		return nil, err
	}

	result, err :=
		tx.Exec(
			ctx,
			`
			UPDATE assistant_deployments
			SET
				current_version = $2,
				updated_at = NOW()
			WHERE id = $1
			  AND current_version = $3`,
			deploymentID,
			nextVersion,
			currentVersion,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"更新教学智能体部署当前版本失败: %w",
				err,
			)
	}
	if result.RowsAffected() != 1 {
		return nil,
			ErrAssistantDeploymentStateConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体版本发布事务失败: %w",
				err,
			)
	}

	return version, nil
}

// GetAssistantDeploymentVersion 按部署ID和版本号读取内部快照。
func GetAssistantDeploymentVersion(
	ctx context.Context,
	deploymentID string,
	versionNumber int,
) (
	*models.AssistantDeploymentVersion,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	if deploymentID == "" ||
		versionNumber <= 0 {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	version, err :=
		scanAssistantDeploymentVersion(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentVersionSelectColumns+
					`
				 FROM assistant_deployment_versions v
				 WHERE v.deployment_id = $1
				   AND v.version = $2`,
				deploymentID,
				versionNumber,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantDeploymentVersionNotFound
		}

		return nil,
			fmt.Errorf(
				"查询教学智能体部署版本失败: %w",
				err,
			)
	}

	return version, nil
}

// GetCurrentAssistantDeploymentVersion 读取部署当前不可变版本。
func GetCurrentAssistantDeploymentVersion(
	ctx context.Context,
	deploymentID string,
) (
	*models.AssistantDeploymentVersion,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	version, err :=
		scanAssistantDeploymentVersion(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentVersionSelectColumns+
					`
				 FROM assistant_deployment_versions v
				 JOIN assistant_deployments d
				   ON d.id = v.deployment_id
				  AND d.current_version = v.version
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
				ErrAssistantDeploymentVersionNotFound
		}

		return nil,
			fmt.Errorf(
				"查询教学智能体当前部署版本失败: %w",
				err,
			)
	}

	return version, nil
}

// ListAssistantDeploymentVersionsForOwner 返回教师部署版本历史。
//
// 仓储返回内部记录；Service必须转换为只含版本号和哈希的安全响应，
// 不得把AssistantPromptSnapshot或ContextSnapshotJSON直接返回浏览器。
func ListAssistantDeploymentVersionsForOwner(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
) (
	[]*models.AssistantDeploymentVersion,
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

	rows, err :=
		database.DB.Query(
			ctx,
			`SELECT `+
				assistantDeploymentVersionSelectColumns+
				`
			 FROM assistant_deployment_versions v
			 JOIN assistant_deployments d
			   ON d.id = v.deployment_id
			 WHERE v.deployment_id = $1
			   AND d.owner_user_id = $2
			 ORDER BY v.version DESC`,
			deploymentID,
			ownerUserID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询教学智能体部署版本列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	versions := make(
		[]*models.AssistantDeploymentVersion,
		0,
	)

	for rows.Next() {
		version, scanErr :=
			scanAssistantDeploymentVersion(rows)
		if scanErr != nil {
			return nil,
				fmt.Errorf(
					"扫描教学智能体部署版本失败: %w",
					scanErr,
				)
		}

		versions = append(
			versions,
			version,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历教学智能体部署版本失败: %w",
				err,
			)
	}

	return versions, nil
}
