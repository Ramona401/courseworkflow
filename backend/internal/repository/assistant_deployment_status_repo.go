package repository

// assistant_deployment_status_repo.go
//
// 本文件实现部署暂停、恢复、撤销以及可变运行策略更新。
//
// 状态机固定为：
//   active  -> paused
//   paused  -> active
//   active  -> revoked
//   paused  -> revoked
//   revoked -> 无任何后续状态
//
// 每次状态变化都在事务中锁定部署行，避免并发暂停、恢复和撤销互相覆盖。
// 撤销不可恢复，也不提供删除部署记录的方法。
//
// 运行策略只允许在active或paused状态更新；revoked部署保留历史原值。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// PauseAssistantDeployment 把active部署切换为paused。
func PauseAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
) (
	*models.AssistantDeployment,
	error,
) {
	return transitionAssistantDeploymentStatus(
		ctx,
		deploymentID,
		ownerUserID,
		models.AssistantDeploymentStatusPaused,
	)
}

// ResumeAssistantDeployment 把paused部署恢复为active。
//
// revoked部署不能通过本方法恢复。
func ResumeAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
) (
	*models.AssistantDeployment,
	error,
) {
	return transitionAssistantDeploymentStatus(
		ctx,
		deploymentID,
		ownerUserID,
		models.AssistantDeploymentStatusActive,
	)
}

// RevokeAssistantDeployment 永久撤销active或paused部署。
func RevokeAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
) (
	*models.AssistantDeployment,
	error,
) {
	return transitionAssistantDeploymentStatus(
		ctx,
		deploymentID,
		ownerUserID,
		models.AssistantDeploymentStatusRevoked,
	)
}

// transitionAssistantDeploymentStatus 在行锁事务中执行状态变化。
func transitionAssistantDeploymentStatus(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
	targetStatus string,
) (
	*models.AssistantDeployment,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)
	targetStatus =
		strings.TrimSpace(targetStatus)

	if deploymentID == "" ||
		ownerUserID == "" ||
		!models.IsValidAssistantDeploymentStatus(
			targetStatus,
		) {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	tx, err :=
		database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体部署状态事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	deployment, err :=
		scanAssistantDeployment(
			tx.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.id = $1
				   AND d.owner_user_id = $2
				 FOR UPDATE`,
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
				"锁定教学智能体部署状态失败: %w",
				err,
			)
	}

	if deployment.Status ==
		models.AssistantDeploymentStatusRevoked {
		return nil,
			ErrAssistantDeploymentRevoked
	}

	if !assistantDeploymentStatusTransitionAllowed(
		deployment.Status,
		targetStatus,
	) {
		return nil,
			ErrAssistantDeploymentStateConflict
	}

	err =
		tx.QueryRow(
			ctx,
			`
			UPDATE assistant_deployments
			SET
				status = $2,
				updated_at = NOW()
			WHERE id = $1
			RETURNING
				status,
				updated_at`,
			deployment.ID,
			targetStatus,
		).Scan(
			&deployment.Status,
			&deployment.UpdatedAt,
		)
	if err != nil {
		return nil,
			wrapAssistantDeploymentWriteError(
				"更新教学智能体部署状态",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体部署状态事务失败: %w",
				err,
			)
	}

	return deployment, nil
}

// UpdateAssistantDeploymentPolicy 更新额度、轮数、来源列表和有效期。
//
// 本函数不改变public_id、课件、页面、创建者、学校、教育域和current_version。
func UpdateAssistantDeploymentPolicy(
	ctx context.Context,
	deploymentID string,
	ownerUserID string,
	dailyCallLimit int,
	perSessionTurnLimit int,
	allowedOriginsJSON string,
	validUntil *time.Time,
) (
	*models.AssistantDeployment,
	error,
) {
	deploymentID =
		strings.TrimSpace(deploymentID)
	ownerUserID =
		strings.TrimSpace(ownerUserID)
	allowedOriginsJSON =
		normalizeAssistantDeploymentAllowedOriginsJSON(
			allowedOriginsJSON,
		)

	if deploymentID == "" ||
		ownerUserID == "" ||
		dailyCallLimit < 1 ||
		dailyCallLimit > 100000 ||
		perSessionTurnLimit < 1 ||
		perSessionTurnLimit > 100 {
		return nil,
			ErrAssistantDeploymentInvalidRecord
	}

	tx, err :=
		database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体部署策略事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	deployment, err :=
		scanAssistantDeployment(
			tx.QueryRow(
				ctx,
				`SELECT `+
					assistantDeploymentSelectColumns+
					`
				 FROM assistant_deployments d
				 WHERE d.id = $1
				   AND d.owner_user_id = $2
				 FOR UPDATE`,
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
				"锁定教学智能体部署策略失败: %w",
				err,
			)
	}

	if deployment.Status ==
		models.AssistantDeploymentStatusRevoked {
		return nil,
			ErrAssistantDeploymentRevoked
	}

	err =
		tx.QueryRow(
			ctx,
			`
			UPDATE assistant_deployments
			SET
				daily_call_limit = $2,
				per_session_turn_limit = $3,
				allowed_origins_json = $4::jsonb,
				valid_until = $5,
				updated_at = NOW()
			WHERE id = $1
			RETURNING
				daily_call_limit,
				per_session_turn_limit,
				allowed_origins_json::text,
				valid_until,
				updated_at`,
			deployment.ID,
			dailyCallLimit,
			perSessionTurnLimit,
			allowedOriginsJSON,
			validUntil,
		).Scan(
			&deployment.DailyCallLimit,
			&deployment.PerSessionTurnLimit,
			&deployment.AllowedOriginsJSON,
			&deployment.ValidUntil,
			&deployment.UpdatedAt,
		)
	if err != nil {
		return nil,
			wrapAssistantDeploymentWriteError(
				"更新教学智能体部署运行策略",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体部署策略事务失败: %w",
				err,
			)
	}

	return deployment, nil
}
