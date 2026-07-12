package services

// class_profile_match.go - Class learning-profile injection context builder (differentiated teaching, independent module)
//
// Responsibility: turn a single mounted, status=active class profile (one class_profiles row,
// owned by the lesson-plan author) into a hard-instruction context block for the lesson-planning AI.
// It is injected by workshop_stage_service.go's LoadStagePromptContextV2 in three stages:
// analyze / design / write (NOT review / revise -- a class profile guides design, not scoring).
//
// Difference from the unit-plan layer (unit_plan_match.go):
//   - Unit plan: the syllabus of the big unit this lesson belongs to; five-stage全程; preempts course outline.
//   - Class profile: who the students in front of the teacher are; three-stage (analyze/design/write);
//     dimension-orthogonal to the unit plan, injected INDEPENDENTLY and NEVER yields
//     (a lesson can carry both a unit plan and a class profile at the same time -- Yuhan decision).
//
// Hard-instruction style follows the lessons learned in BuildCourseOutlinesContext: explicitly tell
// the AI to claim this material, never say it cannot see it, never guess from stale memory, and
// actively design differentiated instruction for the A/B/C tiers and the listed weak points.
//
// COMPLIANCE RED LINE (do not violate): this builder assembles ONLY the four group-conclusion段
// of the class card (overall_profile / tier_structure / weak_points / teaching_advice). These are
// anonymous group descriptions with no individual student identity. It NEVER touches class_students
// (individual detail), which never enters the injection chain.
//
// IMPORTANT: this file uses plain ASCII punctuation only (no full-width Chinese quotes) inside Go
// string literals, to avoid any source-encoding compile surprises. Chinese text content is fine;
// only quote/paren punctuation around literals stays ASCII.
//
// This file does no matching/scoring (an explicit mount needs no matching). Whether to inject is
// decided by the caller (fetch by ID via repository.GetClassProfileByID, verify status==active and
// owner==author), this function only builds the block.

import (
	"strings"

	"tedna/internal/models"
)

// classProfileContextMaxRunes caps the total runes of the class-profile injection block.
//
// The four段 are teacher-authored group conclusions and are normally short, but a teacher could paste
// a long passage. Cap by rune (Chinese counted per char, never splitting a half character), consistent
// with the unit-plan / textbook / outline injection caps. 4000 runes is plenty for four group-level段.
const classProfileContextMaxRunes = 4000

// truncateRunesForClassProfile safely truncates by rune, appending an omission marker when overlong.
func truncateRunesForClassProfile(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "\n......(班级学情内容较长, 此处已截断, 核心分层与薄弱点见上文)"
}

// BuildClassProfileContext assembles one active, owner-verified class profile into an injectable
// hard-instruction context block.
//
// The caller must have already verified status==active and owner==author; this function does not
// re-check status/ownership. Returns the full block (with leading/trailing markers and hard
// instructions); if cp is nil or all four段 are empty, returns "" so the caller injects nothing
// (never a hollow shell block).
func BuildClassProfileContext(cp *models.ClassProfile) string {
	if cp == nil {
		return ""
	}
	// All four group-conclusion段 empty -> nothing substantive to inject.
	if !cp.HasProfileContent() {
		return ""
	}

	overall := strings.TrimSpace(cp.OverallProfile)
	tier := strings.TrimSpace(cp.TierStructure)
	weak := strings.TrimSpace(cp.WeakPoints)
	advice := strings.TrimSpace(cp.TeachingAdvice)

	// Locating fields (anonymous, no individual identity): class name / grade / subject / headcount.
	className := strings.TrimSpace(cp.ClassName)
	grade := strings.TrimSpace(cp.Grade)
	subject := strings.TrimSpace(cp.Subject)

	var b strings.Builder
	b.WriteString("\n\n【本班真实学情 (老师已显式挂载, 必须据此做差异化教学设计)】\n")
	b.WriteString("以下是老师为这个班亲自建立的班级学情, 描述的是这节课要面对的真实学生群体。\n")
	b.WriteString("这是匿名的群体学情结论 (不含任何学生个人身份信息), 是老师口中的 班级学情 / 学情资料。\n")
	b.WriteString("重要要求 (务必遵守):\n")
	b.WriteString("  1. 这是老师亲自挂载的真实资料, 请直接阅读并采用, 不要说 看不到资料, 也不要凭旧记忆编造学情。\n")
	b.WriteString("  2. 教学分析/教学设计/教案撰写都必须贴着下面的真实学情来做差异化设计: 针对 A/B/C 三个层次分别设计, 并针对列出的薄弱点做专门处理。\n")
	b.WriteString("  3. 不要做无差别的大锅饭设计; 凡涉及活动难度/提问梯度/分层任务/作业分层时, 应明确对应到 A 层 (拔尖)、B 层 (中等)、C 层 (学困) 的不同需求。\n")
	b.WriteString("  4. 若学情与课文/课题存在张力 (如薄弱点正是本课难点), 应优先在设计中正面化解。\n")

	// Identity line.
	idParts := make([]string, 0, 3)
	if className != "" {
		idParts = append(idParts, className)
	}
	if grade != "" {
		idParts = append(idParts, grade)
	}
	if subject != "" {
		idParts = append(idParts, subject)
	}
	if len(idParts) > 0 {
		b.WriteString("\n---- 班级: " + strings.Join(idParts, " / ") + " ----\n")
	} else {
		b.WriteString("\n---- 本班学情 ----\n")
	}
	if cp.StudentCount > 0 {
		b.WriteString("班级人数: " + itoaClassProfile(cp.StudentCount) + " 人\n")
	}

	// Body: the four group-conclusion段, only non-empty ones, merged then rune-capped as a whole.
	var body strings.Builder
	if overall != "" {
		body.WriteString("\n【整体画像】\n")
		body.WriteString(overall)
	}
	if tier != "" {
		body.WriteString("\n\n【分层结构 (A/B/C 三层群体描述)】\n")
		body.WriteString(tier)
	}
	if weak != "" {
		body.WriteString("\n\n【学科薄弱点】\n")
		body.WriteString(weak)
	}
	if advice != "" {
		body.WriteString("\n\n【老师的分层教学建议】\n")
		body.WriteString(advice)
	}
	b.WriteString(truncateRunesForClassProfile(body.String(), classProfileContextMaxRunes))

	b.WriteString("\n---- 本班学情结束 ----\n")
	return b.String()
}

// itoaClassProfile is a tiny local int->string helper (avoids importing strconv just for one call,
// keeping this file's import set minimal and matching the lightweight style of sibling match files).
func itoaClassProfile(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
