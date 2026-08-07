package repository

// courseware_assembly_context.go — 自动装配写回context身份
//
// 该文件建立服务层与仓储层之间的轻量身份传递，不修改原AutoAssemble函数签名。
//
// 版本包装入口把CoursewareID、Version和RunID写入context；
// 现有主编排、IAOCI逐槽位、呈现保护和视频建议调用继续传递同一个context；
// 仓储写入函数据此选择普通写入或版本化写入。
//
// context只承载数据库运行身份，不承载可变业务数据，也不替代权限校验。

import (
	"context"
	"strings"
)

// CoursewareAssemblyWriteContext 是自动装配写回在context中的数据库身份。
type CoursewareAssemblyWriteContext struct {
	CoursewareID string
	Version      int64
	RunID        string
}

type coursewareAssemblyWriteContextKey struct{}

// WithCoursewareAssemblyWriteContext 把当前数据库装配运行身份写入context。
func WithCoursewareAssemblyWriteContext(
	ctx context.Context,
	input CoursewareAssemblyWriteContext,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	input.CoursewareID = strings.TrimSpace(
		input.CoursewareID,
	)
	input.RunID = strings.TrimSpace(
		input.RunID,
	)

	return context.WithValue(
		ctx,
		coursewareAssemblyWriteContextKey{},
		input,
	)
}

// coursewareAssemblyWriteContextFrom 读取并校验装配写回身份。
func coursewareAssemblyWriteContextFrom(
	ctx context.Context,
) (
	CoursewareAssemblyWriteContext,
	bool,
) {
	if ctx == nil {
		return CoursewareAssemblyWriteContext{},
			false
	}

	input, ok := ctx.Value(
		coursewareAssemblyWriteContextKey{},
	).(CoursewareAssemblyWriteContext)
	if !ok {
		return CoursewareAssemblyWriteContext{},
			false
	}

	input.CoursewareID = strings.TrimSpace(
		input.CoursewareID,
	)
	input.RunID = strings.TrimSpace(
		input.RunID,
	)

	if input.CoursewareID == "" ||
		input.Version <= 0 ||
		input.RunID == "" {
		return CoursewareAssemblyWriteContext{},
			false
	}

	return input, true
}
