package repository

// school_model_policy_repo.go — 学校模型策略数据访问层
//
// 对应表：school_model_policies（school_id 主键 + overseas_enabled + note + granted_by）
// 核心查询：IsSchoolOverseasEnabled —— 某学校是否被授权使用境外模型。
// fail-closed 原则：空 schoolID / 无记录 / 查询出错，一律返回 false（按境内处理）。
//
// 风格对齐 kb_authorized_repo.go：database.DB、ctx、fmt.Errorf 包错、LEFT JOIN 填名。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// IsSchoolOverseasEnabled 判断某学校是否被授权使用境外模型。
// 返回 (true, nil) 仅当该学校有记录且 overseas_enabled=true。
// schoolID 为空 → 直接返回 (false, nil)，不查库（无归属用户一律境内）。
func IsSchoolOverseasEnabled(ctx context.Context, schoolID string) (bool, error) {
	if schoolID == "" {
		return false, nil
	}
	var enabled bool
	err := database.DB.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM school_model_policies
			WHERE school_id = $1 AND overseas_enabled = true
		)`,
		schoolID,
	).Scan(&enabled)
	if err != nil {
		return false, fmt.Errorf("查询学校模型策略失败: %w", err)
	}
	return enabled, nil
}

// GetSchoolModelPolicy 获取某学校的策略记录（不存在返回 nil, nil）
func GetSchoolModelPolicy(ctx context.Context, schoolID string) (*models.SchoolModelPolicy, error) {
	if schoolID == "" {
		return nil, nil
	}
	p := &models.SchoolModelPolicy{}
	err := database.DB.QueryRow(ctx, `
		SELECT school_id, overseas_enabled, COALESCE(note,''), granted_by, created_at, updated_at
		FROM school_model_policies
		WHERE school_id = $1
	`, schoolID).Scan(
		&p.SchoolID, &p.OverseasEnabled, &p.Note, &p.GrantedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		// pgx 在无行时返回 ErrNoRows，这里统一当作"无策略"处理，不作为错误上抛
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("查询学校模型策略记录失败: %w", err)
	}
	return p, nil
}

// ListSchoolModelPolicies 列出全部学校模型策略（JOIN organizations 填学校名，LEFT JOIN users 填授权人名）
// 仅列出 school_model_policies 表中已有记录的学校（未登记的学校默认境内，不在此列表）。
func ListSchoolModelPolicies(ctx context.Context) ([]*models.SchoolModelPolicyItem, error) {
	query := `
		SELECT smp.school_id, COALESCE(o.name,''), smp.overseas_enabled,
		       COALESCE(smp.note,''),
		       COALESCE(g.display_name, COALESCE(g.username,''), ''),
		       smp.created_at, smp.updated_at
		FROM school_model_policies smp
		LEFT JOIN organizations o ON o.id = smp.school_id
		LEFT JOIN users g ON g.id = smp.granted_by
		ORDER BY smp.updated_at DESC
	`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询学校模型策略列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.SchoolModelPolicyItem{}
	for rows.Next() {
		item := &models.SchoolModelPolicyItem{}
		if err := rows.Scan(
			&item.SchoolID, &item.SchoolName, &item.OverseasEnabled,
			&item.Note, &item.GrantedByName, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描学校模型策略行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpsertSchoolModelPolicy 新增或更新学校模型策略（admin 授权/取消授权入口）
// grantedBy 可空（空串写 NULL）。ON CONFLICT 按 school_id 更新。
func UpsertSchoolModelPolicy(ctx context.Context, schoolID string, overseasEnabled bool, note string, grantedBy string) error {
	if schoolID == "" {
		return fmt.Errorf("schoolID 为空")
	}
	var grantedByArg interface{}
	if grantedBy == "" {
		grantedByArg = nil
	} else {
		grantedByArg = grantedBy
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO school_model_policies (school_id, overseas_enabled, note, granted_by, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (school_id) DO UPDATE
		SET overseas_enabled = EXCLUDED.overseas_enabled,
		    note             = EXCLUDED.note,
		    granted_by       = EXCLUDED.granted_by,
		    updated_at       = now()
	`, schoolID, overseasEnabled, note, grantedByArg)
	if err != nil {
		return fmt.Errorf("保存学校模型策略失败: %w", err)
	}
	return nil
}

// DeleteSchoolModelPolicy 删除学校模型策略记录（删除=回到默认境内）
func DeleteSchoolModelPolicy(ctx context.Context, schoolID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM school_model_policies WHERE school_id = $1`,
		schoolID,
	)
	if err != nil {
		return fmt.Errorf("删除学校模型策略失败: %w", err)
	}
	return nil
}
