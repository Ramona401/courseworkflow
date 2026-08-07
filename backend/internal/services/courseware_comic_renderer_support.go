package services

// courseware_comic_renderer_support.go — 漫画渲染公共安全辅助、样式与脚本
//
// 本文件集中维护颜色对比、背景透明度、稳定片段替换、布局类、
// DOM ID、画布范围、安全CSS和离线交互脚本。
// 覆盖层严格使用教师保存的固定宽高与字号，并使用紧凑留白垂直居中。

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"tedna/internal/models"
)

var (
	coursewareComicDOMIDUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	coursewareComicHexColorPattern    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

func replaceCoursewareComicPanelFragment(
	pageHTML string,
	projectID string,
	panelID string,
	newPanelHTML string,
) (string, error) {
	if err := validateCoursewareComicMarkerID(
		projectID,
	); err != nil {
		return "", err
	}

	if err := validateCoursewareComicMarkerID(
		panelID,
	); err != nil {
		return "", err
	}

	startMarker :=
		coursewareComicPanelStartMarker(
			projectID,
			panelID,
		)

	endMarker :=
		coursewareComicPanelEndMarker(
			projectID,
			panelID,
		)

	if strings.Count(
		pageHTML,
		startMarker,
	) != 1 ||
		strings.Count(
			pageHTML,
			endMarker,
		) != 1 {
		return "",
			fmt.Errorf(
				"漫画格稳定标记缺失或重复",
			)
	}

	startIndex :=
		strings.Index(
			pageHTML,
			startMarker,
		)

	endIndex :=
		strings.Index(
			pageHTML,
			endMarker,
		)

	if startIndex < 0 ||
		endIndex <= startIndex {
		return "",
			fmt.Errorf(
				"漫画格稳定标记顺序异常",
			)
	}

	endIndex +=
		len(endMarker)

	return pageHTML[:startIndex] +
			newPanelHTML +
			pageHTML[endIndex:],
		nil
}

func resolveCoursewareComicLayoutClass(
	panelCount int,
) string {
	switch panelCount {
	case 4:
		return "tedna-comic-layout--4"

	case 5:
		return "tedna-comic-layout--5"

	case 6:
		return "tedna-comic-layout--6"

	default:
		return "tedna-comic-layout--stepper"
	}
}

func sanitizeCoursewareComicElementType(
	value string,
) string {
	value =
		strings.TrimSpace(value)

	if models.IsValidCWComicElementType(
		value,
	) {
		return value
	}

	return models.CWComicElementCaption
}

func isCoursewareComicBubbleType(
	value string,
) bool {
	return value ==
		models.CWComicElementSpeechBubble ||
		value ==
			models.CWComicElementThoughtBubble
}

func resolveCoursewareComicRenderPalette(
	styleID string,
	elementType string,
) coursewareComicRenderPalette {
	value :=
		strings.ToLower(
			strings.TrimSpace(
				styleID,
			),
		)

	switch value {
	case "speech_soft":
		return coursewareComicRenderPalette{
			Fill:   "#FFFFFF",
			Stroke: "#8B5CF6",
			Text:   "#312E81",
		}

	case "speech_outline":
		return coursewareComicRenderPalette{
			Fill:   "#FFFFFF",
			Stroke: "#0F172A",
			Text:   "#111827",
		}

	case "speech_capsule":
		return coursewareComicRenderPalette{
			Fill:   "#EFF6FF",
			Stroke: "#2563EB",
			Text:   "#172554",
		}

	case "speech_pop":
		return coursewareComicRenderPalette{
			Fill:   "#FFF7ED",
			Stroke: "#EA580C",
			Text:   "#7C2D12",
		}

	case "thought_cloud",
		"thought_soft",
		"thought_outline":
		return coursewareComicRenderPalette{
			Fill:   "#F8FAFC",
			Stroke: "#475569",
			Text:   "#1E293B",
		}

	case "question_blue":
		return coursewareComicRenderPalette{
			Fill:   "#EFF6FF",
			Stroke: "#2563EB",
			Text:   "#172554",
		}

	case "question_orange":
		return coursewareComicRenderPalette{
			Fill:   "#FFF7ED",
			Stroke: "#EA580C",
			Text:   "#7C2D12",
		}

	case "card_light":
		return coursewareComicRenderPalette{
			Fill:   "#FFFFFF",
			Stroke: "#94A3B8",
			Text:   "#111827",
		}

	case "card_accent":
		return coursewareComicRenderPalette{
			Fill:   "#4C1D95",
			Stroke: "#A78BFA",
			Text:   "#FFFFFF",
		}
	}

	switch {
	case strings.Contains(
		value,
		"purple",
	) ||
		elementType ==
			models.CWComicElementQuestionCard:
		return coursewareComicRenderPalette{
			Fill:   "#F3E8FF",
			Stroke: "#7E22CE",
			Text:   "#3B0764",
		}

	case strings.Contains(
		value,
		"warning",
	) ||
		elementType ==
			models.CWComicElementWarningCard:
		return coursewareComicRenderPalette{
			Fill:   "#FFF7ED",
			Stroke: "#EA580C",
			Text:   "#7C2D12",
		}

	case elementType ==
		models.CWComicElementKnowledgeCard ||
		elementType ==
			models.CWComicElementAnswerCard:
		return coursewareComicRenderPalette{
			Fill:   "#ECFDF5",
			Stroke: "#059669",
			Text:   "#064E3B",
		}

	case elementType ==
		models.CWComicElementNarration ||
		elementType ==
			models.CWComicElementCaption:
		return coursewareComicRenderPalette{
			Fill:   "#FFFBEB",
			Stroke: "#D97706",
			Text:   "#78350F",
		}

	case elementType ==
		models.CWComicElementThoughtBubble:
		return coursewareComicRenderPalette{
			Fill:   "#F8FAFC",
			Stroke: "#475569",
			Text:   "#1E293B",
		}

	default:
		return coursewareComicRenderPalette{
			Fill:   "#FFFFFF",
			Stroke: "#1E293B",
			Text:   "#111827",
		}
	}
}

func sanitizeCoursewareComicColor(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if coursewareComicHexColorPattern.MatchString(value) {
		return strings.ToUpper(value)
	}
	return fallback
}

// normalizeCoursewareComicBackgroundOpacity 规范背景透明度乘数。
//
// 历史文档的零值表示尚未设置，必须按1处理而不是全透明。
func normalizeCoursewareComicBackgroundOpacity(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 1
	}
	if value < 0.2 {
		return 0.2
	}
	if value > 1 {
		return 1
	}
	return value
}

// applyCoursewareComicBackgroundOpacity 把透明度乘数应用到安全预设背景。
func applyCoursewareComicBackgroundOpacity(color string, opacity float64) string {
	red, green, blue, ok := parseCoursewareComicHexColor(color)
	if !ok {
		return color
	}

	return fmt.Sprintf(
		"rgba(%d,%d,%d,%s)",
		red,
		green,
		blue,
		strconv.FormatFloat(opacity, 'f', 3, 64),
	)
}

// resolveCoursewareComicRenderTextColor 统一自动和手动文字颜色。
//
// 空模式与auto均按背景明暗自动选择，只有manual才使用教师颜色。
// 这样历史黑色文字遇到深色卡片时也会自动恢复成白色。
func resolveCoursewareComicRenderTextColor(
	style models.CoursewareComicTextStyle,
	background string,
	fallback string,
) string {
	mode := strings.ToLower(strings.TrimSpace(style.ColorMode))
	if mode == models.CWComicTextColorModeManual {
		return sanitizeCoursewareComicColor(style.Color, fallback)
	}

	if coursewareComicRelativeLuminance(background) < 0.46 {
		return "#FFFFFF"
	}
	return "#111827"
}

func parseCoursewareComicHexColor(value string) (int, int, int, bool) {
	value = strings.TrimSpace(value)
	if !coursewareComicHexColorPattern.MatchString(value) {
		return 0, 0, 0, false
	}

	parsed, err := strconv.ParseUint(value[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}

	return int((parsed >> 16) & 0xFF), int((parsed >> 8) & 0xFF), int(parsed & 0xFF), true
}

func coursewareComicRelativeLuminance(value string) float64 {
	red, green, blue, ok := parseCoursewareComicHexColor(value)
	if !ok {
		return 1
	}

	channel := func(component int) float64 {
		normalized := float64(component) / 255
		if normalized <= 0.03928 {
			return normalized / 12.92
		}
		return math.Pow((normalized+0.055)/1.055, 2.4)
	}

	return 0.2126*channel(red) + 0.7152*channel(green) + 0.0722*channel(blue)
}

func sanitizeCoursewareComicTextAlign(
	value string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "left":
		return "left"

	case "right":
		return "right"

	default:
		return "center"
	}
}

func safeCoursewareComicDOMID(
	value string,
) string {
	value =
		coursewareComicDOMIDUnsafePattern.
			ReplaceAllString(
				value,
				"-",
			)

	value =
		strings.Trim(
			value,
			"-",
		)

	if value == "" {
		return "tedna-comic-element"
	}

	return value
}

func clampCoursewareComicUnit(
	value float64,
) float64 {
	if value < 0 {
		return 0
	}

	if value > 1 {
		return 1
	}

	return value
}

func clampCoursewareComicSize(
	value float64,
) float64 {
	if value < 0.04 {
		return 0.04
	}

	if value > 1 {
		return 1
	}

	return value
}

func formatCoursewareComicPercent(
	value float64,
) string {
	return strconv.FormatFloat(
		value*100,
		'f',
		3,
		64,
	) + "%"
}

func renderCoursewareComicPageStyle() string {
	return `<style>
.tedna-comic-content{position:absolute;top:80px;left:0;right:0;bottom:0;padding:24px 38px 30px;display:flex;flex-direction:column;gap:18px;box-sizing:border-box}
.tedna-comic-heading{height:94px;display:flex;align-items:center;justify-content:space-between;gap:28px;padding:0 18px}
.tedna-comic-heading h1{margin:4px 0 0;font-size:42px;line-height:1.12;font-weight:900;letter-spacing:.02em}
.tedna-comic-kicker{font-size:18px;font-weight:800;color:#2563EB;letter-spacing:.16em}
.tedna-comic-source{text-align:right;font-size:19px;line-height:1.5;color:#475569;white-space:nowrap}
.tedna-comic-source strong{font-size:21px;color:#0F172A}
.tedna-comic-layout{flex:1;min-height:0;display:grid;gap:18px}
.tedna-comic-layout--4{grid-template-columns:repeat(2,minmax(0,1fr));grid-template-rows:repeat(2,minmax(0,1fr))}
.tedna-comic-layout--5{grid-template-columns:1.35fr 1fr 1fr;grid-template-rows:repeat(2,minmax(0,1fr))}
.tedna-comic-layout--5 .tedna-comic-panel--first{grid-row:1 / span 2}
.tedna-comic-layout--6{grid-template-columns:repeat(3,minmax(0,1fr));grid-template-rows:repeat(2,minmax(0,1fr))}
.tedna-comic-panel{position:relative;min-width:0;min-height:0;overflow:hidden;border:4px solid #0F172A;border-radius:22px;background:#CBD5E1;box-shadow:0 14px 28px rgba(15,23,42,.16)}
.tedna-comic-panel-image{position:absolute;inset:0;width:100%;height:100%;object-fit:cover;display:block}
.tedna-comic-panel-number{position:absolute;left:14px;top:12px;z-index:80;width:38px;height:38px;border-radius:50%;display:flex;align-items:center;justify-content:center;background:#0F172A;color:#FFF;font-size:21px;font-weight:900}
.tedna-comic-overlay-layer{position:absolute;inset:0;z-index:10;pointer-events:none}
.tedna-comic-overlay{position:absolute;box-sizing:border-box;display:flex;flex-direction:column;align-items:stretch;min-height:0;max-height:none;overflow:visible;filter:drop-shadow(0 4px 8px rgba(15,23,42,.18));pointer-events:none}
.tedna-comic-bubble-shape{position:absolute;inset:0;width:100%;height:100%;max-height:none;overflow:visible}
.tedna-comic-tail-layer{position:absolute;inset:0;width:100%;height:100%;pointer-events:none}
.tedna-comic-overlay-text{position:relative;z-index:2;width:100%;height:100%;min-height:0;max-height:none;display:flex;flex-direction:column;justify-content:center;box-sizing:border-box;padding:9px 16px;overflow:hidden;word-break:normal;overflow-wrap:anywhere;white-space:pre-wrap}
.tedna-comic-overlay--speech_bubble .tedna-comic-overlay-text,.tedna-comic-overlay--thought_bubble .tedna-comic-overlay-text{padding:10px 18px}
.tedna-comic-overlay--narration .tedna-comic-overlay-text,.tedna-comic-overlay--caption .tedna-comic-overlay-text{padding:9px 16px}
.tedna-comic-overlay--knowledge_card .tedna-comic-overlay-text,.tedna-comic-overlay--warning_card .tedna-comic-overlay-text,.tedna-comic-overlay--question_card .tedna-comic-overlay-text,.tedna-comic-overlay--answer_card .tedna-comic-overlay-text{padding:14px 22px;justify-content:flex-start}
.tedna-comic-question-label{font-size:.82em;font-weight:900;letter-spacing:.08em;margin-bottom:4px}
.tedna-comic-question-title{font-weight:900;margin-bottom:5px}
.tedna-comic-options{margin:0 0 5px;padding-left:1.45em;font-size:.82em;line-height:1.38}
.tedna-comic-answer-button{pointer-events:auto;align-self:flex-start;border:0;border-radius:999px;padding:6px 13px;background:#7E22CE;color:#FFF;font:inherit;font-size:.72em;font-weight:800;cursor:pointer}
.tedna-comic-answer{margin-top:5px;padding-top:5px;border-top:1px dashed currentColor;font-size:.74em;line-height:1.35}
.tedna-comic-stepper{flex:1;min-height:0;display:grid;grid-template-columns:minmax(0,1fr) 210px;gap:18px}
.tedna-comic-stepper-stage{position:relative;min-width:0;min-height:0}
.tedna-comic-panel--step{position:absolute;inset:0;display:none}
.tedna-comic-panel--step.is-active{display:block}
.tedna-comic-stepper-controls{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));align-content:start;gap:12px;padding:4px}
.tedna-comic-step-button{height:98px;border:3px solid transparent;border-radius:17px;background:#E2E8F0;color:#334155;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:3px}
.tedna-comic-step-button span{width:44px;height:44px;border-radius:50%;display:flex;align-items:center;justify-content:center;background:#0F172A;color:#FFF;font-size:23px;font-weight:900}
.tedna-comic-step-button strong{font-size:16px}
.tedna-comic-step-button.is-active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 0 0 4px rgba(37,99,235,.16)}
</style>`
}

func renderCoursewareComicPageScript() string {
	return `<script>
(function(){
  var script=document.currentScript;
  var root=script&&script.closest('.tedna-comic-page');
  if(!root){return;}

  root.querySelectorAll('[data-tedna-comic-target]').forEach(function(button){
    button.addEventListener('click',function(){
      var target=button.getAttribute('data-tedna-comic-target');

      root.querySelectorAll('.tedna-comic-panel--step').forEach(function(panel){
        panel.classList.toggle(
          'is-active',
          panel.getAttribute('data-tedna-comic-panel-id')===target
        );
      });

      root.querySelectorAll('[data-tedna-comic-target]').forEach(function(item){
        item.classList.toggle('is-active',item===button);
      });
    });
  });

  root.querySelectorAll('[data-tedna-answer-target]').forEach(function(button){
    button.addEventListener('click',function(){
      var target=button.getAttribute('data-tedna-answer-target');
      var answer=root.querySelector('[data-tedna-answer-id="'+target+'"]');

      if(!answer){return;}

      var opening=answer.hasAttribute('hidden');

      if(opening){
        answer.removeAttribute('hidden');
        button.textContent='收起答案';
      }else{
        answer.setAttribute('hidden','hidden');
        button.textContent='查看答案';
      }
    });
  });
})();
</script>`
}
