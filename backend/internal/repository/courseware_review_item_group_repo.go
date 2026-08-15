package repository

// courseware_review_item_group_repo.go
//
// R-06 正式问题组的查询、扫描和治理历史读取仓储。
//
// 设计边界：
//   1. 读取始终同时限定审核会话和治理人；
//   2. 不在本文件执行写操作；
//   3. 组、成员和事件使用独立扫描函数，避免浏览器或Service直接依赖数据库行结构；
//   4. 组基础写操作位于courseware_review_item_group_mutation_repo.go；
//   5. 成员加入和移除位于courseware_review_item_group_member_repo.go；
//   6. 跨组移动、合并和拆分位于courseware_review_item_group_structural_repo.go。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrCoursewareReviewItemGroupNotFound 合并不存在和越过治理边界两种情况。
	ErrCoursewareReviewItemGroupNotFound = errors.New(
		"课件审核问题组不存在",
	)

	// ErrCoursewareReviewItemGroupMemberNotFound 合并成员不存在和越权两种情况。
	ErrCoursewareReviewItemGroupMemberNotFound = errors.New(
		"课件审核问题组成员不存在",
	)

	// ErrCoursewareReviewItemGroupConflict 表示组、成员版本或业务状态已经变化。
	ErrCoursewareReviewItemGroupConflict = errors.New(
		"课件审核问题组状态已变化，请刷新后重试",
	)
)

const cwReviewItemGroupSelectColumns = `
	id,
	courseware_id,
	source_session_id,
	name,
	COALESCE(primary_item_id::text, ''),
	status,
	version,
	COALESCE(merged_into_group_id::text, ''),
	created_by,
	created_at,
	updated_at`

const cwReviewItemGroupMemberSelectColumns = `
	id,
	group_id,
	courseware_id,
	source_session_id,
	item_id,
	status,
	version,
	created_by,
	COALESCE(removed_by::text, ''),
	removed_at,
	created_at,
	updated_at`

func scanCoursewareReviewItemGroup(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewItemGroup, error) {
	group := &models.CoursewareReviewItemGroup{}

	var (
		primaryItemID     string
		mergedIntoGroupID string
	)

	err := row.Scan(
		&group.ID,
		&group.CoursewareID,
		&group.SourceSessionID,
		&group.Name,
		&primaryItemID,
		&group.Status,
		&group.Version,
		&mergedIntoGroupID,
		&group.CreatedBy,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if primaryItemID != "" {
		group.PrimaryItemID = &primaryItemID
	}

	if mergedIntoGroupID != "" {
		group.MergedIntoGroupID = &mergedIntoGroupID
	}

	return group, nil
}

func scanCoursewareReviewItemGroupMember(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewItemGroupMember, error) {
	member := &models.CoursewareReviewItemGroupMember{}
	var removedBy string

	err := row.Scan(
		&member.ID,
		&member.GroupID,
		&member.CoursewareID,
		&member.SourceSessionID,
		&member.ItemID,
		&member.Status,
		&member.Version,
		&member.CreatedBy,
		&removedBy,
		&member.RemovedAt,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if removedBy != "" {
		member.RemovedBy = &removedBy
	}

	return member, nil
}

// GetCoursewareReviewItemGroupByID 读取当前治理人的一条问题组。
func GetCoursewareReviewItemGroupByID(
	ctx context.Context,
	groupID string,
	sessionID string,
	creatorID string,
) (*models.CoursewareReviewItemGroup, error) {
	group, err := scanCoursewareReviewItemGroup(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewItemGroupSelectColumns+`
			 FROM courseware_review_item_groups
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3`,
			strings.TrimSpace(groupID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(creatorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemGroupNotFound
		}

		return nil, fmt.Errorf(
			"查询课件审核问题组失败: %w",
			err,
		)
	}

	return group, nil
}

// ListCoursewareReviewItemGroupsBySession 返回当前治理人的全部问题组。
func ListCoursewareReviewItemGroupsBySession(
	ctx context.Context,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItemGroup, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewItemGroupSelectColumns+`
		 FROM courseware_review_item_groups
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY
			CASE status
				WHEN 'active' THEN 1
				ELSE 2
			END,
			updated_at DESC,
			id ASC`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核问题组列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	groups := make(
		[]*models.CoursewareReviewItemGroup,
		0,
	)

	for rows.Next() {
		group, scanErr := scanCoursewareReviewItemGroup(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件审核问题组失败: %w",
				scanErr,
			)
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核问题组失败: %w",
			err,
		)
	}

	return groups, nil
}

// ListCoursewareReviewItemGroupMembers 返回一条问题组的全部稳定成员身份。
//
// member自身已经保存created_by，并且数据库复合外键保证它与所属组created_by一致，
// 因此这里不额外JOIN问题组，避免重复表连接和列名歧义。
func ListCoursewareReviewItemGroupMembers(
	ctx context.Context,
	groupID string,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItemGroupMember, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewItemGroupMemberSelectColumns+`
		 FROM courseware_review_item_group_members
		 WHERE group_id = $1
		   AND source_session_id = $2
		   AND created_by = $3
		 ORDER BY
			CASE status
				WHEN 'active' THEN 1
				ELSE 2
			END,
			created_at ASC,
			id ASC`,
		strings.TrimSpace(groupID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核问题组成员失败: %w",
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
			scanCoursewareReviewItemGroupMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件审核问题组成员失败: %w",
				scanErr,
			)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核问题组成员失败: %w",
			err,
		)
	}

	return members, nil
}

// ListCoursewareReviewItemGroupEvents 返回一条问题组的追加式治理历史。
func ListCoursewareReviewItemGroupEvents(
	ctx context.Context,
	groupID string,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItemGroupEvent, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT
			event.id,
			event.group_id,
			event.courseware_id,
			event.source_session_id,
			event.group_version,
			event.event_type,
			event.actor_id,
			COALESCE(event.member_id::text, ''),
			event.member_version,
			COALESCE(event.related_group_id::text, ''),
			event.reason,
			COALESCE(event.metadata_json::text, '{}'),
			event.created_at
		FROM courseware_review_item_group_events AS event
		INNER JOIN courseware_review_item_groups AS review_group
			ON review_group.id = event.group_id
			AND review_group.source_session_id =
				event.source_session_id
		WHERE event.group_id = $1
		  AND event.source_session_id = $2
		  AND review_group.created_by = $3
		ORDER BY event.group_version ASC`,
		strings.TrimSpace(groupID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核问题组事件失败: %w",
			err,
		)
	}
	defer rows.Close()

	events := make(
		[]*models.CoursewareReviewItemGroupEvent,
		0,
	)

	for rows.Next() {
		event := &models.CoursewareReviewItemGroupEvent{}

		var (
			memberID       string
			memberVersion  *int
			relatedGroupID string
		)

		if err := rows.Scan(
			&event.ID,
			&event.GroupID,
			&event.CoursewareID,
			&event.SourceSessionID,
			&event.GroupVersion,
			&event.EventType,
			&event.ActorID,
			&memberID,
			&memberVersion,
			&relatedGroupID,
			&event.Reason,
			&event.MetadataJSON,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课件审核问题组事件失败: %w",
				err,
			)
		}

		if memberID != "" {
			event.MemberID = &memberID
		}

		event.MemberVersion = memberVersion

		if relatedGroupID != "" {
			event.RelatedGroupID = &relatedGroupID
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核问题组事件失败: %w",
			err,
		)
	}

	return events, nil
}
