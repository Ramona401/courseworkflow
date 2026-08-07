package services

// courseware_review_submission_errors.go
//
// 课件重新提交审核的跨层业务错误。
//
// 仓储返回的具体错误会携带：
//
//   - 尚未完成修改的问题数量；
//   - 页面内容已变化、需要重新检查的问题数量；
//   - 原问题页面已经删除的问题数量。
//
// 本别名供HTTP处理器使用errors.Is识别统一业务场景。
// 仓储错误继续原样向上传递，因此不会丢失面向作者的数量说明。

import "tedna/internal/repository"

// ErrCWSubmitRemediationIncomplete 表示正式整改尚未达到重新提交条件。
var ErrCWSubmitRemediationIncomplete = repository.ErrCWReviewSubmissionRemediationIncomplete
