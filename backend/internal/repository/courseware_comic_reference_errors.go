package repository

// courseware_comic_reference_errors.go — 知识点漫画参考资源错误定义
//
// 集中维护业务错误和PostgreSQL约束错误转换，
// 供参考资源仓储与上层服务复用。

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCoursewareComicReferenceNotFound = errors.New(
		"知识点漫画参考资源不存在",
	)

	ErrCoursewareComicReferenceLimitReached = errors.New(
		"每个知识点漫画项目最多绑定8项参考资源",
	)

	ErrCoursewareComicReferenceConflict = errors.New(
		"该参考资源已经绑定到当前漫画项目",
	)

	ErrCoursewareComicReferenceInvalid = errors.New(
		"知识点漫画参考资源记录无效",
	)
)

func wrapCoursewareComicReferenceWriteError(
	err error,
) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError

	if errors.As(
		err,
		&pgErr,
	) {
		switch pgErr.Code {
		case "23505":
			return ErrCoursewareComicReferenceConflict

		case "23503",
			"23514",
			"22P02":
			return ErrCoursewareComicReferenceInvalid
		}
	}

	return fmt.Errorf(
		"创建知识点漫画参考资源失败: %w",
		err,
	)
}
