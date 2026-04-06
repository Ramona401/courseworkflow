package repository

// pipeline_index_repo.go — Pipeline定稿索引数据访问层
//
// 职责：
//   - 存取 pipeline_indexes 表
//   - verify步骤完成后写入定稿索引
//   - 2审Evaluator/Meta读取定稿索引替代 course_indexes
//
// 设计说明：
//   - course_indexes 永远保存原始OSS索引，本文件不涉及
//   - pipeline_indexes 是平台内部流转的定稿版，每个pipeline只有一条记录
//   - UpsertPipelineIndex 使用 ON CONFLICT 幂等写入，重复verify时覆盖旧记录

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"tedna/internal/database"
)

// PipelineIndex pipeline定稿索引记录
type PipelineIndex struct {
	ID           string     `json:"id"`
	PipelineID   string     `json:"pipeline_id"`
	IndexContent string     `json:"index_content"`
	IndexHash    string     `json:"index_hash"`
	PageCount    int        `json:"page_count"`
	TotalLength  int        `json:"total_length"`
	ReviewRound  int        `json:"review_round"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// UpsertPipelineIndex 写入或更新Pipeline定稿索引
// verify步骤成功后调用，每个pipeline只保留一条最新记录
// 使用 ON CONFLICT(pipeline_id) DO UPDATE 幂等写入
func UpsertPipelineIndex(pipelineID string, indexContent string, reviewRound int, pageCount int) error {
	ctx := context.Background()

	// 计算内容哈希，用于后续完整性校验
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(indexContent)))
	totalLength := len(indexContent)

	_, err := database.DB.Exec(ctx,
		`INSERT INTO pipeline_indexes
		    (pipeline_id, index_content, index_hash, page_count, total_length, review_round, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (pipeline_id) DO UPDATE SET
		    index_content = EXCLUDED.index_content,
		    index_hash    = EXCLUDED.index_hash,
		    page_count    = EXCLUDED.page_count,
		    total_length  = EXCLUDED.total_length,
		    review_round  = EXCLUDED.review_round,
		    updated_at    = NOW()`,
		pipelineID, indexContent, hash, pageCount, totalLength, reviewRound,
	)
	if err != nil {
		return fmt.Errorf("写入pipeline定稿索引失败(pipeline=%s): %w", pipelineID, err)
	}
	return nil
}

// GetPipelineIndex 读取Pipeline定稿索引
// 2审Evaluator/Meta调用，替代 GetCourseIndex
// 若不存在返回 ErrPipelineIndexNotFound
func GetPipelineIndex(pipelineID string) (*PipelineIndex, error) {
	ctx := context.Background()

	idx := &PipelineIndex{}
	var indexHash *string
	var pageCount, totalLength *int

	err := database.DB.QueryRow(ctx,
		`SELECT id, pipeline_id, index_content, index_hash,
		        page_count, total_length, review_round, created_at, updated_at
		 FROM pipeline_indexes
		 WHERE pipeline_id = $1`,
		pipelineID,
	).Scan(
		&idx.ID, &idx.PipelineID, &idx.IndexContent, &indexHash,
		&pageCount, &totalLength, &idx.ReviewRound,
		&idx.CreatedAt, &idx.UpdatedAt,
	)
	if err != nil {
		return nil, ErrPipelineIndexNotFound
	}

	if indexHash != nil {
		idx.IndexHash = *indexHash
	}
	if pageCount != nil {
		idx.PageCount = *pageCount
	}
	if totalLength != nil {
		idx.TotalLength = *totalLength
	}

	return idx, nil
}

// ErrPipelineIndexNotFound pipeline定稿索引不存在错误
// 发生场景：2审时verify还未成功写入，或pipeline_id错误
var ErrPipelineIndexNotFound = fmt.Errorf("pipeline定稿索引不存在，请确认一轮verify已成功完成")
