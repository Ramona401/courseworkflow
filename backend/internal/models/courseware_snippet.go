package models

// courseware_snippet.go — 【代码收藏库·批次C新增】课件页面代码收藏模型
//
// 场景：老师把自己调整满意的课件页HTML"打星收藏"进个人代码库（表 courseware_code_snippets），
//   之后在任何课件的单页微调时可从收藏库选一条注入为参考代码，让AI按范本的
//   布局骨架/交互方式/视觉手法微调当前页。
//
// 设计要点：
//   - 纯个人归属（user_id），无共享维度；共享诉求走既有的共享课件库+code_share_scope体系。
//   - html_content 存收藏时的完整快照——源页之后被修改/删除不影响收藏（快照语义），
//     故 source_courseware_id/source_page_number 仅作溯源展示、不设外键。
//   - 列表场景不回传 html_content 全文（单条可达几十KB），用 CoursewareCodeSnippetListItem
//     附 html_len 字节数供前端展示体量；注入/预览时按 id 单独取全文。

import "time"

// CoursewareCodeSnippet 代码收藏完整实体（含 HTML 全文，单条查询用）
type CoursewareCodeSnippet struct {
        ID                 string    `json:"id"`                   // 收藏ID（uuid）
        UserID             string    `json:"user_id"`              // 归属用户
        Title              string    `json:"title"`                // 收藏名称（老师自己起）
        Note               string    `json:"note"`                 // 可选备注
        HTMLContent        string    `json:"html_content"`         // 完整页面HTML快照
        SourceCoursewareID string    `json:"source_courseware_id"` // 溯源：来源课件ID（可能已删除）
        SourcePageNumber   int       `json:"source_page_number"`   // 溯源：来源页码
        CreatedAt          time.Time `json:"created_at"`           // 收藏时间
}

// CoursewareCodeSnippetListItem 收藏列表项（轻量，不含 html_content 全文，附字节数）
type CoursewareCodeSnippetListItem struct {
        ID                 string    `json:"id"`                   // 收藏ID
        Title              string    `json:"title"`                // 收藏名称
        Note               string    `json:"note"`                 // 备注
        HTMLLen            int       `json:"html_len"`             // HTML快照字节数（前端展示体量用）
        SourceCoursewareID string    `json:"source_courseware_id"` // 溯源课件ID
        SourcePageNumber   int       `json:"source_page_number"`   // 溯源页码
        CreatedAt          time.Time `json:"created_at"`           // 收藏时间
}
