package services

// courseware_comic_generation_task_test.go
//
// 验证漫画图片后台任务登记结果的稳定错误映射。
// 不连接数据库、不启动goroutine、不调用图片模型。

import (
	"errors"
	"testing"

	"tedna/internal/repository"
)

func TestCoursewareComicGenerationTaskStartMapping(
	t *testing.T,
) {
	cases := []struct {
		name     string
		result   BackgroundStartResult
		expected error
	}{
		{
			name:
				"正常启动",
			result:
				BackgroundStarted,
			expected:
				nil,
		},
		{
			name:
				"任务重复",
			result:
				BackgroundAlreadyRunning,
			expected:
				repository.
					ErrCoursewareComicProjectConflict,
		},
		{
			name:
				"服务排空",
			result:
				BackgroundRejectedDraining,
			expected:
				ErrCoursewareComicProjectServiceUnavailable,
		},
		{
			name:
				"任务参数无效",
			result:
				BackgroundInvalid,
			expected:
				ErrCoursewareComicProjectInvalidRequest,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				actual :=
					mapCoursewareComicTaskStartResult(
						item.result,
					)

				if item.expected == nil {
					if actual != nil {
						t.Fatalf(
							"预期成功，实际错误: %v",
							actual,
						)
					}

					return
				}

				if !errors.Is(
					actual,
					item.expected,
				) {
					t.Fatalf(
						"错误映射不一致: result=%s actual=%v expected=%v",
						item.result,
						actual,
						item.expected,
					)
				}
			},
		)
	}
}
