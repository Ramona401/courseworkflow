package services

// courseware_comic_page_renderer.go — 知识点漫画页面与分格确定性渲染主流程
//
// 本文件只负责页面骨架、导航、分格布局和稳定标记。
// 覆盖层视觉、气泡尾巴、颜色与文字渲染拆分到
// courseware_comic_overlay_renderer.go；公共样式、脚本和安全辅助拆分到
// courseware_comic_renderer_support.go，避免单文件职责和行数继续膨胀。

import (
	"encoding/json"
	"fmt"
	htmlstd "html"
	"regexp"
	"strconv"
	"strings"

	"tedna/internal/models"
)

// coursewareComicPanelRenderData 是渲染器接收的单格数据。
type coursewareComicPanelRenderData struct {
	Panel    *models.CoursewareComicPanel
	ImageURL string
}

var coursewareComicMarkerIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func truncateCoursewareComicRunes(
	value string,
	maxLength int,
) string {
	value = strings.TrimSpace(value)

	if maxLength <= 0 {
		return ""
	}

	runes := []rune(value)

	if len(runes) <= maxLength {
		return value
	}

	return string(runes[:maxLength])
}

func coursewareComicProjectStartMarker(
	projectID string,
) string {
	return "<!-- TEDNA_COMIC_START:" +
		projectID +
		" -->"
}

func coursewareComicProjectEndMarker(
	projectID string,
) string {
	return "<!-- TEDNA_COMIC_END:" +
		projectID +
		" -->"
}

func coursewareComicPanelStartMarker(
	projectID string,
	panelID string,
) string {
	return "<!-- TEDNA_COMIC_PANEL_START:" +
		projectID +
		":" +
		panelID +
		" -->"
}

func coursewareComicPanelEndMarker(
	projectID string,
	panelID string,
) string {
	return "<!-- TEDNA_COMIC_PANEL_END:" +
		projectID +
		":" +
		panelID +
		" -->"
}

func validateCoursewareComicMarkerID(
	value string,
) error {
	value = strings.TrimSpace(value)

	if value == "" ||
		!coursewareComicMarkerIDPattern.MatchString(value) {
		return fmt.Errorf(
			"漫画稳定标记ID不合法",
		)
	}

	return nil
}

// renderCoursewareComicPageHTML 渲染完整课件页面。
func renderCoursewareComicPageHTML(
	courseware *models.Courseware,
	project *models.CoursewareComicProject,
	panels []coursewareComicPanelRenderData,
	pageNumber int,
	totalPages int,
) (string, error) {
	if courseware == nil ||
		project == nil ||
		pageNumber < 1 ||
		totalPages < 1 ||
		len(panels) < 4 ||
		len(panels) > 8 {
		return "",
			fmt.Errorf(
				"漫画页面渲染上下文无效",
			)
	}

	if err := validateCoursewareComicMarkerID(
		project.ID,
	); err != nil {
		return "", err
	}

	nav := buildSafeNavBlock(
		courseware.NavTemplateHTML,
		pageNumber,
		totalPages,
	)

	if strings.TrimSpace(nav) == "" {
		nav = fmt.Sprintf(
			`<!-- NAV_START --><div style="height:80px;padding:0 56px;display:flex;align-items:center;justify-content:space-between;background:#FFFFFF;border-bottom:1px solid #E5E7EB;color:#111827;"><strong>%s</strong><span>%d / %d</span></div><!-- NAV_END -->`,
			htmlstd.EscapeString(
				courseware.Title,
			),
			pageNumber,
			totalPages,
		)
	}

	var builder strings.Builder

	builder.WriteString(
		`<div class="cw-page tedna-comic-page"`,
	)
	builder.WriteString(
		` data-tedna-comic-project-id="`,
	)
	builder.WriteString(
		htmlstd.EscapeString(project.ID),
	)
	builder.WriteString(
		`" style="width:1920px;height:1080px;overflow:hidden;position:relative;background:var(--cw-bg,#F8FAFC);color:var(--cw-text,#111827);font-family:'Noto Sans SC','Microsoft YaHei',sans-serif;">`,
	)
	builder.WriteString("\n")
	builder.WriteString(nav)
	builder.WriteString("\n")
	builder.WriteString(
		coursewareComicProjectStartMarker(
			project.ID,
		),
	)
	builder.WriteString("\n")
	builder.WriteString(
		renderCoursewareComicPageStyle(),
	)
	builder.WriteString("\n")
	builder.WriteString(
		`<main class="tedna-comic-content">`,
	)
	builder.WriteString(
		renderCoursewareComicHeading(
			project,
		),
	)

	if len(panels) >= 7 {
		stepperHTML, err :=
			renderCoursewareComicStepper(
				project,
				panels,
			)
		if err != nil {
			return "", err
		}

		builder.WriteString(stepperHTML)
	} else {
		builder.WriteString(
			`<section class="tedna-comic-layout `,
		)
		builder.WriteString(
			resolveCoursewareComicLayoutClass(
				len(panels),
			),
		)
		builder.WriteString(`">`)

		for _, panelData := range panels {
			panelHTML, err :=
				renderCoursewareComicPanelHTML(
					project,
					panelData,
					false,
				)
			if err != nil {
				return "", err
			}

			builder.WriteString(panelHTML)
		}

		builder.WriteString(`</section>`)
	}

	builder.WriteString(`</main>`)
	builder.WriteString("\n")
	builder.WriteString(
		renderCoursewareComicPageScript(),
	)
	builder.WriteString("\n")
	builder.WriteString(
		coursewareComicProjectEndMarker(
			project.ID,
		),
	)
	builder.WriteString("\n</div>")

	return builder.String(), nil
}

func renderCoursewareComicHeading(
	project *models.CoursewareComicProject,
) string {
	var builder strings.Builder

	builder.WriteString(
		`<header class="tedna-comic-heading">`,
	)
	builder.WriteString(
		`<div><div class="tedna-comic-kicker">知识点漫画</div><h1>`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			project.Title,
		),
	)
	builder.WriteString(
		`</h1></div>`,
	)

	builder.WriteString(
		`<div class="tedna-comic-source"><strong>`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			project.PublisherSnapshot,
		),
	)
	builder.WriteString(`</strong>`)

	if strings.TrimSpace(
		project.SemesterSnapshot,
	) != "" {
		builder.WriteString(" · ")
		builder.WriteString(
			htmlstd.EscapeString(
				project.SemesterSnapshot,
			),
		)
	}

	builder.WriteString(`<br><span>`)
	builder.WriteString(
		htmlstd.EscapeString(
			project.Subject,
		),
	)
	builder.WriteString(" · ")
	builder.WriteString(
		htmlstd.EscapeString(
			project.Grade,
		),
	)
	builder.WriteString(
		`</span></div></header>`,
	)

	return builder.String()
}

// renderCoursewareComicStepper 渲染7至8格主舞台。
//
// 步进按钮只显示格号，不复制缩略图图片。
// 因此单格局部更新时只替换漫画格标记区间即可，不会留下旧缩略图。
func renderCoursewareComicStepper(
	project *models.CoursewareComicProject,
	panels []coursewareComicPanelRenderData,
) (string, error) {
	var builder strings.Builder

	builder.WriteString(
		`<section class="tedna-comic-stepper">`,
	)
	builder.WriteString(
		`<div class="tedna-comic-stepper-stage">`,
	)

	for _, panelData := range panels {
		panelHTML, err :=
			renderCoursewareComicPanelHTML(
				project,
				panelData,
				true,
			)
		if err != nil {
			return "", err
		}

		builder.WriteString(panelHTML)
	}

	builder.WriteString(`</div>`)
	builder.WriteString(
		`<div class="tedna-comic-stepper-controls" role="tablist" aria-label="漫画分镜">`,
	)

	for _, panelData := range panels {
		panel := panelData.Panel

		if panel == nil {
			continue
		}

		activeClass := ""

		if panel.PanelNo == 1 {
			activeClass = " is-active"
		}

		builder.WriteString(
			`<button type="button" class="tedna-comic-step-button`,
		)
		builder.WriteString(activeClass)
		builder.WriteString(
			`" data-tedna-comic-target="`,
		)
		builder.WriteString(
			htmlstd.EscapeString(
				panel.ID,
			),
		)
		builder.WriteString(
			`" role="tab"><span>`,
		)
		builder.WriteString(
			strconv.Itoa(
				panel.PanelNo,
			),
		)
		builder.WriteString(
			`</span><strong>第`,
		)
		builder.WriteString(
			strconv.Itoa(
				panel.PanelNo,
			),
		)
		builder.WriteString(
			`格</strong></button>`,
		)
	}

	builder.WriteString(`</div></section>`)

	return builder.String(), nil
}

// renderCoursewareComicPanelHTML 渲染一个稳定漫画格区间。
func renderCoursewareComicPanelHTML(
	project *models.CoursewareComicProject,
	panelData coursewareComicPanelRenderData,
	stepper bool,
) (string, error) {
	panel := panelData.Panel

	if project == nil ||
		panel == nil ||
		strings.TrimSpace(
			panelData.ImageURL,
		) == "" {
		return "",
			fmt.Errorf(
				"漫画格渲染数据不完整",
			)
	}

	if err := validateCoursewareComicMarkerID(
		project.ID,
	); err != nil {
		return "", err
	}

	if err := validateCoursewareComicMarkerID(
		panel.ID,
	); err != nil {
		return "", err
	}

	var overlayDocument models.CoursewareComicOverlayDocument

	if err := json.Unmarshal(
		[]byte(
			panel.OverlayDocumentJSON,
		),
		&overlayDocument,
	); err != nil {
		return "",
			fmt.Errorf(
				"解析漫画格覆盖层失败: %w",
				err,
			)
	}

	panelClass := "tedna-comic-panel"

	if panel.PanelNo == 1 {
		panelClass +=
			" tedna-comic-panel--first"
	}

	if stepper {
		panelClass +=
			" tedna-comic-panel--step"

		if panel.PanelNo == 1 {
			panelClass += " is-active"
		}
	}

	var builder strings.Builder

	builder.WriteString(
		coursewareComicPanelStartMarker(
			project.ID,
			panel.ID,
		),
	)
	builder.WriteString("\n")
	builder.WriteString(`<article class="`)
	builder.WriteString(panelClass)
	builder.WriteString(
		`" data-tedna-comic-panel-id="`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			panel.ID,
		),
	)
	builder.WriteString(
		`" data-tedna-comic-panel-no="`,
	)
	builder.WriteString(
		strconv.Itoa(
			panel.PanelNo,
		),
	)
	builder.WriteString(`">`)

	builder.WriteString(
		`<img class="tedna-comic-panel-image" src="`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			panelData.ImageURL,
		),
	)
	builder.WriteString(
		`" alt="第`,
	)
	builder.WriteString(
		strconv.Itoa(
			panel.PanelNo,
		),
	)
	builder.WriteString(`格：`)
	builder.WriteString(
		htmlstd.EscapeString(
			panel.KnowledgeClaim,
		),
	)
	builder.WriteString(`">`)

	builder.WriteString(
		`<div class="tedna-comic-panel-number">`,
	)
	builder.WriteString(
		strconv.Itoa(
			panel.PanelNo,
		),
	)
	builder.WriteString(`</div>`)

	builder.WriteString(
		`<div class="tedna-comic-overlay-layer">`,
	)

	for _, element := range overlayDocument.Elements {
		builder.WriteString(
			renderCoursewareComicOverlayElement(
				project.ID,
				panel.ID,
				element,
			),
		)
	}

	builder.WriteString(`</div></article>`)
	builder.WriteString("\n")
	builder.WriteString(
		coursewareComicPanelEndMarker(
			project.ID,
			panel.ID,
		),
	)

	return builder.String(), nil
}
