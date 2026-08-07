package services

// courseware_nav_markers.go — 课件导航栏边界标记的包级统一定义。
//
// 多条课件生成与修改链路都需要识别导航栏边界，包括：
//   - 页面装配与媒体填充；
//   - 导航栏完整性保护；
//   - 导航栏模板替换；
//   - 增删页、排序和一键页码校准。
//
// 这些链路必须共享同一套标记，不能在各文件中分别硬编码，
// 否则某次局部重构删除定义后会造成整个services包无法编译。
//
// 文本常量用于生成、查找和替换正式平台HTML。
// 正则模式用于兼容历史页面中标记大小写不同或空格略有差异的情况。

const (
	// cwNavStartMarker是平台正式导航栏起始标记。
	cwNavStartMarker = "<!-- NAV_START -->"

	// cwNavEndMarker是平台正式导航栏结束标记。
	cwNavEndMarker = "<!-- NAV_END -->"

	// cwNavStartMarkerPattern是导航栏起始标记的兼容正则片段。
	//
	// 可匹配：
	//   <!-- NAV_START -->
	//   <!--NAV_START-->
	//   <!-- nav_start -->
	cwNavStartMarkerPattern = `<!--\s*NAV_START\s*-->`

	// cwNavEndMarkerPattern是导航栏结束标记的兼容正则片段。
	//
	// 可匹配：
	//   <!-- NAV_END -->
	//   <!--NAV_END-->
	//   <!-- nav_end -->
	cwNavEndMarkerPattern = `<!--\s*NAV_END\s*-->`
)
