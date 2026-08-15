package ai

import (
	"strings"

	"tedna/internal/models"
)

// lessonPlanMathOutputPolicy 是教案生成场景统一追加的系统级数学符号规范。
//
// 该策略只约束“数学表达式如何呈现”，不改变教案结构、事实来源、权限、阶段流程或业务提示词。
// 放在统一AI客户端边界，可覆盖普通教案对话、正式整稿、Evidence Harness修复、流式生成和
// 同场景下的非流式修订链，避免把同一规则复制到多个阶段Prompt中后发生版本漂移。
const lessonPlanMathOutputPolicy = `【数学符号输出规范 · 强制】
本规范优先于本轮其他提示中关于数学公式书写格式的要求。

所有面向教师或学生的数学表达式必须直接使用 Unicode 字符书写，禁止使用 LaTeX/TeX 数学语法。
禁止使用 $...$、$$...$$、\(...\)、\[...\]、\frac、\sqrt、\theta、\alpha、\mu、\Delta、\pi、
\rho、\omega、\times、\cdot、\le、\ge、\approx、_{}、^{} 等数学源码形式。

常用替换：
\theta→θ  \alpha→α  \mu→μ  \Delta→Δ  \pi→π  \rho→ρ  \omega→ω
\times→×  \cdot→·  \le→≤  \ge→≥  \approx→≈  \sqrt{2}→√2
下标：F_1→F₁  v_0→v₀
上标：v^2→v²  m/s^2→m/s²
分式：\frac{G}{\cos\theta}→G/cosθ；复杂分式使用括号，例如 (mv²)/(2r)。

正确示例：
A. F₁ = G·tanθ，F₂ = G/cosθ
B. F₁ = G·sinθ，F₂ = G·cosθ

错误示例（严禁作为数学表达式输出）：
A. $F_1 = G\tan\theta$，$F_2 = \frac{G}{\cos\theta}$

输出前必须自查：教师或学生可见正文中的数学表达式如果仍含 $ 数学定界符或反斜杠数学命令，
必须先改写为 Unicode 再输出。代码、路径、正则或转义字符中的非数学用途 $ 和 \ 不属于数学表达式，
不得为了满足本规范破坏其原有语义。`

// coursewareMathOutputPolicy 是课件生成场景统一追加的系统级数学符号规范。
//
// 课件AI经常直接返回HTML/CSS/JavaScript，因此不能粗暴禁止源码层出现“$”或“\”。
// 本策略只禁止最终渲染给教师或学生看的数学文本使用LaTeX，同时明确保护程序源码中的合法字符。
const coursewareMathOutputPolicy = `【数学符号输出规范 · 强制】
本规范优先于本轮其他提示中关于数学公式书写格式的要求。

课件最终渲染给教师或学生看到的标题、正文、题目、选项、按钮文字、标签、图注和提示文字中，
所有数学表达式必须直接使用 Unicode 字符书写，禁止显示 LaTeX/TeX 数学源码，也不得依赖
MathJax、KaTeX 或其他运行时公式渲染器把 LaTeX 转成可见公式。

禁止在可见数学文本中使用 $...$、$$...$$、\(...\)、\[...\]、\frac、\sqrt、\theta、\alpha、
\mu、\Delta、\pi、\rho、\omega、\times、\cdot、\le、\ge、\approx、_{}、^{} 等数学源码形式。

常用替换：
\theta→θ  \alpha→α  \mu→μ  \Delta→Δ  \pi→π  \rho→ρ  \omega→ω
\times→×  \cdot→·  \le→≤  \ge→≥  \approx→≈  \sqrt{2}→√2
下标：F_1→F₁  v_0→v₀
上标：v^2→v²  m/s^2→m/s²
分式：\frac{G}{\cos\theta}→G/cosθ；复杂分式使用括号，例如 (mv²)/(2r)。

正确示例：
A. F₁ = G·tanθ，F₂ = G/cosθ
B. F₁ = G·sinθ，F₂ = G·cosθ

错误示例（严禁出现在课件可见数学文本中）：
A. $F_1 = G\tan\theta$，$F_2 = \frac{G}{\cos\theta}$

输出前必须自查最终可见文字：若数学内容仍含 $ 数学定界符或反斜杠数学命令，必须先改写为 Unicode。
HTML、CSS、JavaScript 源码因程序语义需要出现的 $、\、转义、正则或字符串内容允许保留；
不得为了消除这些字符破坏页面结构、脚本、样式、JSON 或交互逻辑。`

// applyMathOutputPolicy 根据可信的运行场景给消息追加系统级数学输出规范。
//
// 场景优先读取TraceContext，因为它描述本次真实业务调用；TraceContext缺失时才回退到
// GetEffectiveConfig固化的SceneCode。函数始终复制消息切片，不修改调用方持有的数据，
// 从而避免并发复用历史消息时发生跨请求污染。
func applyMathOutputPolicy(
	cfg *EffectiveConfig,
	messages []ChatMessage,
	traceCtx *TraceContext,
) []ChatMessage {
	policy := mathOutputPolicyForScene(resolveMathOutputPolicyScene(cfg, traceCtx))
	if policy == "" {
		return cloneChatMessages(messages)
	}

	result := cloneChatMessages(messages)
	lastSystemIndex := -1
	for i := range result {
		if strings.TrimSpace(result[i].Role) == "system" {
			lastSystemIndex = i
		}
	}

	if lastSystemIndex >= 0 {
		if strings.Contains(result[lastSystemIndex].Content, policy) {
			return result
		}

		base := strings.TrimSpace(result[lastSystemIndex].Content)
		if base == "" {
			result[lastSystemIndex].Content = policy
		} else {
			result[lastSystemIndex].Content = base + "\n\n" + policy
		}
		return result
	}

	return append([]ChatMessage{{Role: "system", Content: policy}}, result...)
}

// resolveMathOutputPolicyScene 返回本次调用用于系统输出策略判定的场景码。
func resolveMathOutputPolicyScene(
	cfg *EffectiveConfig,
	traceCtx *TraceContext,
) string {
	if traceCtx != nil && strings.TrimSpace(traceCtx.SceneCode) != "" {
		return strings.TrimSpace(traceCtx.SceneCode)
	}
	if cfg != nil {
		return strings.TrimSpace(cfg.SceneCode)
	}
	return ""
}

// mathOutputPolicyForScene 只对白名单中的教案/课件生成与微调场景生效，避免影响评估、压缩、
// 图片提示词、TTS等并不直接产出教师/学生可见数学正文的AI任务。
func mathOutputPolicyForScene(sceneCode string) string {
	switch sceneCode {
	case models.SceneLessonPlan,
		models.SceneLessonPlanHarness:
		return lessonPlanMathOutputPolicy

	case models.SceneCWIndex,
		models.SceneCWScheme,
		models.SceneCWGenerate,
		models.SceneCWNavRefine,
		models.SceneCWPageRefine,
		models.SceneCWTopicDirect,
		models.SceneCW3DSingle:
		return coursewareMathOutputPolicy

	default:
		return ""
	}
}

// cloneChatMessages 复制消息切片，字符串字段为不可变值，无需额外深拷贝。
func cloneChatMessages(messages []ChatMessage) []ChatMessage {
	if len(messages) == 0 {
		return nil
	}
	return append([]ChatMessage(nil), messages...)
}
