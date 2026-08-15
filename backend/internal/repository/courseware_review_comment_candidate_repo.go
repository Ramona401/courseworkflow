package repository

// courseware_review_comment_candidate_repo.go
//
// R-08 正式课件审核意见不可变候选仓储。
//
// 仓储职责严格限定为：
//
//   1. 创建一条服务端已经冻结完成的候选记录；
//   2. 按candidate + courseware + session + creator完整作用域读取；
//   3. 不提供UPDATE或DELETE入口。
//
// stale判断、当前事实重读、AI生成和替换/追加语义全部属于Service职责。
// 浏览器不能直接构造本仓储输入。

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
	// ErrCoursewareReviewCommentCandidateNotFound 同时承担不存在和作用域不匹配的安全返回。
	ErrCoursewareReviewCommentCandidateNotFound = errors.New(
		"课件审核意见候选不存在或不属于当前审核",
	)
)

const cwReviewCommentCandidateSelectColumns = `
		candidate.id::text,
		candidate.courseware_id::text,
		candidate.source_session_id::text,
		candidate.created_by::text,
		candidate.review_level,
		candidate.candidate_schema_version,
		candidate.candidate_text,
		candidate.original_comment_snapshot,
		candidate.original_comment_hash,
		candidate.selected_item_ids_json::text,
		candidate.input_snapshot_schema_version,
		candidate.input_snapshot_json::text,
		candidate.input_hash,
		candidate.diff_schema_version,
		candidate.diff_json::text,
		candidate.model_used,
		candidate.tokens_used,
		candidate.created_at`

// CreateCoursewareReviewCommentCandidateInput 是Service冻结后的可信持久化输入。
type CreateCoursewareReviewCommentCandidateInput struct {
	CoursewareID    string
	SourceSessionID string
	CreatedBy       string
	ReviewLevel     int

	CandidateSchemaVersion int
	CandidateText          string

	OriginalCommentSnapshot string
	OriginalCommentHash     string

	SelectedItemIDsJSON string

	InputSnapshotSchemaVersion int
	InputSnapshotJSON          string
	InputHash                  string

	DiffSchemaVersion int
	DiffJSON          string

	ModelUsed  string
	TokensUsed int
}

// scanCoursewareReviewCommentCandidate 统一扫描候选记录。
func scanCoursewareReviewCommentCandidate(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewCommentCandidate, error) {
	candidate := &models.CoursewareReviewCommentCandidate{}

	err := row.Scan(
		&candidate.ID,
		&candidate.CoursewareID,
		&candidate.SourceSessionID,
		&candidate.CreatedBy,
		&candidate.ReviewLevel,
		&candidate.CandidateSchemaVersion,
		&candidate.CandidateText,
		&candidate.OriginalCommentSnapshot,
		&candidate.OriginalCommentHash,
		&candidate.SelectedItemIDsJSON,
		&candidate.InputSnapshotSchemaVersion,
		&candidate.InputSnapshotJSON,
		&candidate.InputHash,
		&candidate.DiffSchemaVersion,
		&candidate.DiffJSON,
		&candidate.ModelUsed,
		&candidate.TokensUsed,
		&candidate.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return candidate, nil
}

// CreateCoursewareReviewCommentCandidate 创建一条新的不可变审核意见候选。
//
// 单次INSERT无需额外事务；数据库CHECK和不可变trigger作为最终完整性防线。
func CreateCoursewareReviewCommentCandidate(
	ctx context.Context,
	input *CreateCoursewareReviewCommentCandidateInput,
) (*models.CoursewareReviewCommentCandidate, error) {
	if input == nil {
		return nil, errors.New("课件审核意见候选创建输入不能为空")
	}

	coursewareID := strings.TrimSpace(input.CoursewareID)
	sessionID := strings.TrimSpace(input.SourceSessionID)
	createdBy := strings.TrimSpace(input.CreatedBy)
	candidateText := strings.TrimSpace(input.CandidateText)
	originalCommentHash := strings.TrimSpace(input.OriginalCommentHash)
	inputHash := strings.TrimSpace(input.InputHash)

	if coursewareID == "" ||
		sessionID == "" ||
		createdBy == "" ||
		candidateText == "" ||
		originalCommentHash == "" ||
		inputHash == "" {
		return nil, errors.New("课件审核意见候选缺少必要可信字段")
	}

	candidate, err := scanCoursewareReviewCommentCandidate(
		database.DB.QueryRow(
			ctx,
			`INSERT INTO courseware_review_comment_candidates AS candidate (
				id,
				courseware_id,
				source_session_id,
				created_by,
				review_level,
				candidate_schema_version,
				candidate_text,
				original_comment_snapshot,
				original_comment_hash,
				selected_item_ids_json,
				input_snapshot_schema_version,
				input_snapshot_json,
				input_hash,
				diff_schema_version,
				diff_json,
				model_used,
				tokens_used
			)
			VALUES (
				gen_random_uuid(),
				$1::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9::jsonb,
				$10,
				$11::jsonb,
				$12,
				$13,
				$14::jsonb,
				$15,
				$16
			)
			RETURNING `+cwReviewCommentCandidateSelectColumns,
			coursewareID,
			sessionID,
			createdBy,
			input.ReviewLevel,
			input.CandidateSchemaVersion,
			candidateText,
			input.OriginalCommentSnapshot,
			originalCommentHash,
			input.SelectedItemIDsJSON,
			input.InputSnapshotSchemaVersion,
			input.InputSnapshotJSON,
			inputHash,
			input.DiffSchemaVersion,
			input.DiffJSON,
			strings.TrimSpace(input.ModelUsed),
			input.TokensUsed,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("创建课件审核意见候选失败: %w", err)
	}

	return candidate, nil
}

// GetCoursewareReviewCommentCandidate 按完整审核作用域读取一条不可变候选。
//
// 不允许只凭candidate ID查询，防止把UUID本身当成授权凭据。
//
// candidateID属于URL不可信输入，因此这里故意使用id::text比较，
// 避免非法UUID字符串在SQL的::uuid转换阶段抛出22P02并被错误升级成500。
// 其他三个作用域ID全部来自已经授权的服务端Session/Actor，可以安全按UUID比较。
func GetCoursewareReviewCommentCandidate(
	ctx context.Context,
	candidateID string,
	coursewareID string,
	sessionID string,
	createdBy string,
) (*models.CoursewareReviewCommentCandidate, error) {
	candidateID = strings.TrimSpace(candidateID)
	coursewareID = strings.TrimSpace(coursewareID)
	sessionID = strings.TrimSpace(sessionID)
	createdBy = strings.TrimSpace(createdBy)

	if candidateID == "" ||
		coursewareID == "" ||
		sessionID == "" ||
		createdBy == "" {
		return nil, ErrCoursewareReviewCommentCandidateNotFound
	}

	candidate, err := scanCoursewareReviewCommentCandidate(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewCommentCandidateSelectColumns+`
			 FROM courseware_review_comment_candidates AS candidate
			 WHERE candidate.id::text = $1
			   AND candidate.courseware_id = $2::uuid
			   AND candidate.source_session_id = $3::uuid
			   AND candidate.created_by = $4::uuid`,
			candidateID,
			coursewareID,
			sessionID,
			createdBy,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewCommentCandidateNotFound
		}

		return nil, fmt.Errorf("查询课件审核意见候选失败: %w", err)
	}

	return candidate, nil
}
