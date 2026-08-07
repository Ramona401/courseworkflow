package handlers

// courseware_add_page_discussion_handler_test.go — 新增页讨论特殊路径测试。
//
// 本测试不启动HTTP服务、不连接数据库。
// 只验证/pages/new/rebuild-discussion不会与数字页讨论路径混淆。

import "testing"

func TestExtractCWAddPageDiscussionPathValid(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/coursewares/cw-123/pages/new/rebuild-discussion",
                )

        if got != "cw-123" {
                t.Fatalf(
                        "课件ID解析错误: %q",
                        got,
                )
        }
}

func TestExtractCWAddPageDiscussionPathAcceptsTrailingSlash(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/coursewares/cw-456/pages/new/rebuild-discussion/",
                )

        if got != "cw-456" {
                t.Fatalf(
                        "带尾斜杠路径解析错误: %q",
                        got,
                )
        }
}

func TestExtractCWAddPageDiscussionPathRejectsNumericPagePath(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/coursewares/cw-123/pages/3/rebuild-discussion",
                )

        if got != "" {
                t.Fatalf(
                        "数字页面讨论路径不应被识别为新增页路径: %q",
                        got,
                )
        }
}

func TestExtractCWAddPageDiscussionPathRejectsNestedCoursewareID(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/coursewares/group/cw-123/pages/new/rebuild-discussion",
                )

        if got != "" {
                t.Fatalf(
                        "包含额外路径段的课件ID应被拒绝: %q",
                        got,
                )
        }
}

func TestExtractCWAddPageDiscussionPathRejectsEmptyCoursewareID(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/coursewares//pages/new/rebuild-discussion",
                )

        if got != "" {
                t.Fatalf(
                        "空课件ID应被拒绝: %q",
                        got,
                )
        }
}

func TestExtractCWAddPageDiscussionPathRejectsWrongPrefix(
        t *testing.T,
) {
        got :=
                extractCWAddPageDiscussionPath(
                        "/api/v1/other/cw-123/pages/new/rebuild-discussion",
                )

        if got != "" {
                t.Fatalf(
                        "错误前缀路径应被拒绝: %q",
                        got,
                )
        }
}
