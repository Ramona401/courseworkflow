package repository

// courseware_review_impact_plan_snapshot_repo.go
//
// R-07影响方案草稿生成阶段使用的只读治理快照。
//
// 本文件不执行任何业务修改。
// AI生成候选计划之前，Service会读取当前问题组、成员和关系状态；
// 这些状态随后会被冻结为operation preconditions。
// 最终Apply仍必须在独立事务中重新读取并锁定，不能信任本阶段快照。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
)

// CoursewareReviewImpactGroupMemberSnapshot 保存问题组成员的CAS事实。
type CoursewareReviewImpactGroupMemberSnapshot struct {
	ID      string `json:"id"`
	GroupID string `json:"group_id"`
	ItemID  string `json:"item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`
}

// CoursewareReviewImpactGroupSnapshot 保存问题组当前治理事实。
type CoursewareReviewImpactGroupSnapshot struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	PrimaryItemID     *string `json:"primary_item_id,omitempty"`
	Status            string  `json:"status"`
	Version           int     `json:"version"`
	MergedIntoGroupID *string `json:"merged_into_group_id,omitempty"`

	Members []CoursewareReviewImpactGroupMemberSnapshot `json:"members"`
}

// CoursewareReviewImpactRelationSnapshot 保存pairwise relation当前事实。
type CoursewareReviewImpactRelationSnapshot struct {
	ID string `json:"id"`

	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
	Explanation  string `json:"explanation"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	SourceGlobalMessageID *string `json:"source_global_message_id,omitempty"`
}

// LoadCoursewareReviewImpactGovernanceSnapshot
// 读取当前审核者会话内的问题组、成员和关系快照。
func LoadCoursewareReviewImpactGovernanceSnapshot(
	ctx context.Context,
	sessionID string,
	actorID string,
) (
	[]*CoursewareReviewImpactGroupSnapshot,
	[]*CoursewareReviewImpactRelationSnapshot,
	error,
) {
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)

	if sessionID == "" || actorID == "" {
		return nil, nil, ErrCoursewareReviewImpactPlanConflict
	}

	groupRows, err := database.DB.Query(
		ctx,
		`SELECT
			id,
			name,
			COALESCE(primary_item_id::text, ''),
			status,
			version,
			COALESCE(merged_into_group_id::text, '')
		 FROM courseware_review_item_groups
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY created_at ASC, id ASC`,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"读取影响方案问题组快照失败: %w",
			err,
		)
	}

	groups := make(
		[]*CoursewareReviewImpactGroupSnapshot,
		0,
	)
	groupMap := make(
		map[string]*CoursewareReviewImpactGroupSnapshot,
	)

	for groupRows.Next() {
		group := &CoursewareReviewImpactGroupSnapshot{
			Members: make(
				[]CoursewareReviewImpactGroupMemberSnapshot,
				0,
			),
		}

		var primaryItemID string
		var mergedIntoGroupID string

		if err := groupRows.Scan(
			&group.ID,
			&group.Name,
			&primaryItemID,
			&group.Status,
			&group.Version,
			&mergedIntoGroupID,
		); err != nil {
			groupRows.Close()

			return nil, nil, fmt.Errorf(
				"扫描影响方案问题组快照失败: %w",
				err,
			)
		}

		if primaryItemID != "" {
			group.PrimaryItemID = &primaryItemID
		}

		if mergedIntoGroupID != "" {
			group.MergedIntoGroupID = &mergedIntoGroupID
		}

		groups = append(groups, group)
		groupMap[group.ID] = group
	}

	if err := groupRows.Err(); err != nil {
		groupRows.Close()

		return nil, nil, fmt.Errorf(
			"遍历影响方案问题组快照失败: %w",
			err,
		)
	}
	groupRows.Close()

	memberRows, err := database.DB.Query(
		ctx,
		`SELECT
			id,
			group_id,
			item_id,
			status,
			version
		 FROM courseware_review_item_group_members
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY group_id ASC, item_id ASC, id ASC`,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"读取影响方案问题组成员快照失败: %w",
			err,
		)
	}

	for memberRows.Next() {
		var member CoursewareReviewImpactGroupMemberSnapshot

		if err := memberRows.Scan(
			&member.ID,
			&member.GroupID,
			&member.ItemID,
			&member.Status,
			&member.Version,
		); err != nil {
			memberRows.Close()

			return nil, nil, fmt.Errorf(
				"扫描影响方案问题组成员快照失败: %w",
				err,
			)
		}

		group, exists := groupMap[member.GroupID]
		if !exists {
			memberRows.Close()

			return nil, nil, ErrCoursewareReviewImpactPlanConflict
		}

		group.Members = append(
			group.Members,
			member,
		)
	}

	if err := memberRows.Err(); err != nil {
		memberRows.Close()

		return nil, nil, fmt.Errorf(
			"遍历影响方案问题组成员快照失败: %w",
			err,
		)
	}
	memberRows.Close()

	relationRows, err := database.DB.Query(
		ctx,
		`SELECT
			id,
			relation_type,
			source_item_id,
			target_item_id,
			explanation,
			status,
			version,
			COALESCE(source_global_message_id::text, '')
		 FROM courseware_review_item_relations
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY created_at ASC, id ASC`,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"读取影响方案关系快照失败: %w",
			err,
		)
	}
	defer relationRows.Close()

	relations := make(
		[]*CoursewareReviewImpactRelationSnapshot,
		0,
	)

	for relationRows.Next() {
		relation := &CoursewareReviewImpactRelationSnapshot{}
		var sourceGlobalMessageID string

		if err := relationRows.Scan(
			&relation.ID,
			&relation.RelationType,
			&relation.SourceItemID,
			&relation.TargetItemID,
			&relation.Explanation,
			&relation.Status,
			&relation.Version,
			&sourceGlobalMessageID,
		); err != nil {
			return nil, nil, fmt.Errorf(
				"扫描影响方案关系快照失败: %w",
				err,
			)
		}

		if sourceGlobalMessageID != "" {
			relation.SourceGlobalMessageID =
				&sourceGlobalMessageID
		}

		relations = append(
			relations,
			relation,
		)
	}

	if err := relationRows.Err(); err != nil {
		return nil, nil, fmt.Errorf(
			"遍历影响方案关系快照失败: %w",
			err,
		)
	}

	return groups, relations, nil
}
