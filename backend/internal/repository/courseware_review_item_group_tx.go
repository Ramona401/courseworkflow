package repository

// courseware_review_item_group_tx.go
//
// R-06 正式问题组仓储的事务内部辅助模块。
//
// 本文件负责稳定锁顺序、组与成员CAS更新、成员稳定身份恢复以及追加式事件写入。
// 跨组移动、合并和拆分的公开事务入口位于courseware_review_item_group_structural_repo.go。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func lockCWReviewItemGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	groupID string,
	sessionID string,
	actorID string,
) (*models.CoursewareReviewItemGroup, error) {
	group, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewItemGroupSelectColumns+`
			 FROM courseware_review_item_groups
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3
			 FOR UPDATE`,
			strings.TrimSpace(groupID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemGroupNotFound
		}
		return nil, fmt.Errorf("锁定课件审核问题组失败: %w", err)
	}
	return group, nil
}

func lockCWReviewItemGroupPairTx(
	ctx context.Context,
	tx pgx.Tx,
	firstGroupID string,
	secondGroupID string,
	sessionID string,
	actorID string,
) (map[string]*models.CoursewareReviewItemGroup, error) {
	groupIDs := []string{
		strings.TrimSpace(firstGroupID),
		strings.TrimSpace(secondGroupID),
	}
	if groupIDs[0] == "" ||
		groupIDs[1] == "" ||
		groupIDs[0] == groupIDs[1] {
		return nil, ErrCoursewareReviewItemGroupConflict
	}

	sort.Strings(groupIDs)

	groups := make(
		map[string]*models.CoursewareReviewItemGroup,
		2,
	)
	for _, groupID := range groupIDs {
		group, err := lockCWReviewItemGroupTx(
			ctx,
			tx,
			groupID,
			sessionID,
			actorID,
		)
		if err != nil {
			return nil, err
		}
		groups[group.ID] = group
	}

	return groups, nil
}

func ensureCWReviewItemGroupExpectedVersion(
	group *models.CoursewareReviewItemGroup,
	expectedVersion int,
) error {
	if group == nil ||
		group.Status != models.CWReviewItemGroupStatusActive ||
		expectedVersion < 1 ||
		group.Version != expectedVersion {
		return ErrCoursewareReviewItemGroupConflict
	}

	return nil
}

func lockCWReviewItemGroupMembersTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
) ([]*models.CoursewareReviewItemGroupMember, error) {
	if group == nil {
		return nil, ErrCoursewareReviewItemGroupConflict
	}

	rows, err := tx.Query(
		ctx,
		`SELECT `+cwReviewItemGroupMemberSelectColumns+`
		 FROM courseware_review_item_group_members
		 WHERE group_id = $1
		   AND courseware_id = $2
		   AND source_session_id = $3
		   AND created_by = $4
		   AND status = 'active'
		 ORDER BY item_id::text ASC
		 FOR UPDATE`,
		group.ID,
		group.CoursewareID,
		group.SourceSessionID,
		group.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"锁定课件审核问题组成员失败: %w",
			err,
		)
	}
	defer rows.Close()

	members := make(
		[]*models.CoursewareReviewItemGroupMember,
		0,
	)

	for rows.Next() {
		member, scanErr :=
			scanCoursewareReviewItemGroupMember(
				rows,
			)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描锁定问题组成员失败: %w",
				scanErr,
			)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历锁定问题组成员失败: %w",
			err,
		)
	}

	return members, nil
}

func lockCWReviewItemGroupGovernableItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
) error {
	members, err := lockCWReviewItemGroupMembersTx(
		ctx,
		tx,
		group,
	)
	if err != nil {
		return err
	}

	itemIDs := make(
		[]string,
		0,
		len(members),
	)

	for _, member := range members {
		itemIDs = append(
			itemIDs,
			member.ItemID,
		)
	}

	return lockCWReviewItemGroupItemIDsTx(
		ctx,
		tx,
		itemIDs,
		group.CoursewareID,
		group.SourceSessionID,
		group.CreatedBy,
	)
}

func lockCWReviewItemGroupItemIDsTx(
	ctx context.Context,
	tx pgx.Tx,
	itemIDs []string,
	coursewareID string,
	sessionID string,
	actorID string,
) error {
	unique := make(
		map[string]struct{},
		len(itemIDs),
	)
	normalized := make(
		[]string,
		0,
		len(itemIDs),
	)

	for _, raw := range itemIDs {
		itemID := strings.TrimSpace(raw)
		if itemID == "" {
			return ErrCoursewareReviewItemGroupConflict
		}

		if _, exists := unique[itemID]; exists {
			continue
		}

		unique[itemID] = struct{}{}
		normalized = append(
			normalized,
			itemID,
		)
	}

	sort.Strings(normalized)

	for _, itemID := range normalized {
		if err := lockGovernableCWReviewItemTx(
			ctx,
			tx,
			itemID,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(actorID),
			true,
		); err != nil {
			return err
		}
	}

	return nil
}

func lockCWReviewItemGroupMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	sessionID string,
	actorID string,
) (*models.CoursewareReviewItemGroupMember, error) {
	member, err := scanCoursewareReviewItemGroupMember(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewItemGroupMemberSelectColumns+`
			 FROM courseware_review_item_group_members
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3
			 FOR UPDATE`,
			strings.TrimSpace(memberID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupMemberNotFound
		}

		return nil, fmt.Errorf(
			"锁定课件审核问题组成员失败: %w",
			err,
		)
	}

	return member, nil
}

func lockCWReviewItemGroupMemberByItemTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	coursewareID string,
	sessionID string,
	actorID string,
) (*models.CoursewareReviewItemGroupMember, error) {
	member, err := scanCoursewareReviewItemGroupMember(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewItemGroupMemberSelectColumns+`
			 FROM courseware_review_item_group_members
			 WHERE item_id = $1
			   AND courseware_id = $2
			   AND source_session_id = $3
			   AND created_by = $4
			 FOR UPDATE`,
			strings.TrimSpace(itemID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"按整改项锁定问题组成员失败: %w",
			err,
		)
	}

	return member, nil
}

func insertCWReviewItemGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	sessionID string,
	name string,
	actorID string,
) (*models.CoursewareReviewItemGroup, error) {
	group, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`INSERT INTO courseware_review_item_groups (
				courseware_id,
				source_session_id,
				name,
				status,
				version,
				created_by,
				created_at,
				updated_at
			 )
			 VALUES (
				$1,
				$2,
				$3,
				'active',
				1,
				$4,
				NOW(),
				NOW()
			 )
			 RETURNING `+cwReviewItemGroupSelectColumns,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(name),
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"创建课件审核问题组失败: %w",
			err,
		)
	}

	return group, nil
}

func bumpCWReviewItemGroupVersionTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
) (*models.CoursewareReviewItemGroup, error) {
	next, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_groups
			 SET
				version = version + 1,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND status = 'active'
			   AND version = $2
			 RETURNING `+cwReviewItemGroupSelectColumns,
			group.ID,
			group.Version,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"递增课件审核问题组版本失败: %w",
			err,
		)
	}

	return next, nil
}

func renameCWReviewItemGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
	name string,
) (*models.CoursewareReviewItemGroup, error) {
	next, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_groups
			 SET
				name = $3,
				version = version + 1,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND status = 'active'
			   AND version = $2
			 RETURNING `+cwReviewItemGroupSelectColumns,
			group.ID,
			group.Version,
			strings.TrimSpace(name),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"重命名课件审核问题组失败: %w",
			err,
		)
	}

	return next, nil
}

func setCWReviewItemGroupPrimaryTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
	primaryItemID *string,
) (*models.CoursewareReviewItemGroup, error) {
	next, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_groups
			 SET
				primary_item_id = $3,
				version = version + 1,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND status = 'active'
			   AND version = $2
			 RETURNING `+cwReviewItemGroupSelectColumns,
			group.ID,
			group.Version,
			primaryItemID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"设置课件审核问题组主问题失败: %w",
			err,
		)
	}

	return next, nil
}

func mergeCWReviewItemGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
	targetGroupID string,
) (*models.CoursewareReviewItemGroup, error) {
	next, err := scanCoursewareReviewItemGroup(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_groups
			 SET
				status = 'merged',
				merged_into_group_id = $3,
				primary_item_id = NULL,
				version = version + 1,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND status = 'active'
			   AND version = $2
			 RETURNING `+cwReviewItemGroupSelectColumns,
			group.ID,
			group.Version,
			strings.TrimSpace(targetGroupID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"合并课件审核问题组失败: %w",
			err,
		)
	}

	return next, nil
}

func attachCWReviewItemGroupMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
	itemID string,
	actorID string,
) (*models.CoursewareReviewItemGroupMember, error) {
	existing, err :=
		lockCWReviewItemGroupMemberByItemTx(
			ctx,
			tx,
			itemID,
			group.CoursewareID,
			group.SourceSessionID,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		member, insertErr :=
			scanCoursewareReviewItemGroupMember(
				tx.QueryRow(
					ctx,
					`INSERT INTO
						courseware_review_item_group_members (
							group_id,
							courseware_id,
							source_session_id,
							item_id,
							status,
							version,
							created_by,
							created_at,
							updated_at
						)
					 VALUES (
						 $1,
						 $2,
						 $3,
						 $4,
						 'active',
						 1,
						 $5,
						 NOW(),
						 NOW()
					 )
					 RETURNING `+
						cwReviewItemGroupMemberSelectColumns,
					group.ID,
					group.CoursewareID,
					group.SourceSessionID,
					strings.TrimSpace(itemID),
					strings.TrimSpace(actorID),
				),
			)
		if insertErr != nil {
			return nil, fmt.Errorf(
				"创建课件审核问题组成员失败: %w",
				insertErr,
			)
		}

		return member, nil
	}

	if existing.Status !=
		models.CWReviewItemGroupMemberStatusRemoved {
		return nil,
			ErrCoursewareReviewItemGroupConflict
	}

	member, err := scanCoursewareReviewItemGroupMember(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_group_members
			 SET
				group_id = $3,
				status = 'active',
				version = version + 1,
				removed_by = NULL,
				removed_at = NULL,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND version = $2
			   AND status = 'removed'
			 RETURNING `+
				cwReviewItemGroupMemberSelectColumns,
			existing.ID,
			existing.Version,
			group.ID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"恢复课件审核问题组成员失败: %w",
			err,
		)
	}

	return member, nil
}

func removeCWReviewItemGroupMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	member *models.CoursewareReviewItemGroupMember,
	actorID string,
) (*models.CoursewareReviewItemGroupMember, error) {
	next, err := scanCoursewareReviewItemGroupMember(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_group_members
			 SET
				status = 'removed',
				version = version + 1,
				removed_by = $3,
				removed_at = clock_timestamp(),
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND version = $2
			   AND status = 'active'
			 RETURNING `+
				cwReviewItemGroupMemberSelectColumns,
			member.ID,
			member.Version,
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"移除课件审核问题组成员失败: %w",
			err,
		)
	}

	return next, nil
}

func moveCWReviewItemGroupMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	member *models.CoursewareReviewItemGroupMember,
	targetGroupID string,
) (*models.CoursewareReviewItemGroupMember, error) {
	next, err := scanCoursewareReviewItemGroupMember(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_item_group_members
			 SET
				group_id = $3,
				version = version + 1,
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND version = $2
			   AND status = 'active'
			 RETURNING `+
				cwReviewItemGroupMemberSelectColumns,
			member.ID,
			member.Version,
			strings.TrimSpace(targetGroupID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		return nil, fmt.Errorf(
			"移动课件审核问题组成员失败: %w",
			err,
		)
	}

	return next, nil
}

func insertCWReviewItemGroupEventTx(
	ctx context.Context,
	tx pgx.Tx,
	group *models.CoursewareReviewItemGroup,
	eventType string,
	actorID string,
	reason string,
	memberID *string,
	memberVersion *int,
	relatedGroupID *string,
	metadata map[string]interface{},
) error {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf(
			"序列化课件审核问题组事件失败: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO courseware_review_item_group_events (
			group_id,
			courseware_id,
			source_session_id,
			group_version,
			event_type,
			actor_id,
			member_id,
			member_version,
			related_group_id,
			reason,
			metadata_json,
			created_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11::jsonb,
			clock_timestamp()
		)`,
		group.ID,
		group.CoursewareID,
		group.SourceSessionID,
		group.Version,
		strings.TrimSpace(eventType),
		strings.TrimSpace(actorID),
		memberID,
		memberVersion,
		relatedGroupID,
		strings.TrimSpace(reason),
		string(metadataJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录课件审核问题组事件失败: %w",
			err,
		)
	}

	return nil
}

func sameOptionalString(
	left *string,
	right *string,
) bool {
	return optionalStringValue(left) ==
		optionalStringValue(right)
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
