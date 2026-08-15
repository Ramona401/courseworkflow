package services

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestBuildCWReviewGoalDriftItem(t *testing.T) {
	source := &models.CoursewareReviewItem{
		ID:              "source-item-1",
		CoursewareID:    "courseware-1",
		SourceSessionID: "session-1",
		SourceFindingID: "finding-1",
		OriginType:      models.CWReviewItemOriginAIFinding,
		SourceType:      models.CWReviewItemSourceSelf,
		ReviewLevel:     0,
		ReviewRound:     0,
		CreatedBy:       "user-1",
		OwnerID:         "user-1",
		Severity:        models.CWReviewSeverityMedium,
		Dimension:       "teaching_logic",
		Title:           "原问题",
		Description:     "原问题描述",
		EvidenceJSON:    "{}",
		Status:          models.CWReviewItemStatusConfirmed,
	}

	page := &repository.CoursewareReviewPageSnapshot{
		ID:          "page-1",
		PageNumber:  2,
		Title:       "课堂练习",
		HTMLContent: "<div>课堂练习</div>",
	}

	content := "课堂提问完成后，学生还需要看到更明确的反馈。"

	item, err := buildCWReviewGoalDriftItem(
		source,
		page,
		content,
		"user-1",
		"goal_drift_test",
	)
	if err != nil {
		t.Fatalf("构造目标漂移独立改进项失败: %v", err)
	}

	if item.OriginType != models.CWReviewItemOriginGoalDriftManual {
		t.Fatalf("来源类型错误: %s", item.OriginType)
	}

	if !models.IsCWReviewItemOriginType(item.OriginType) {
		t.Fatalf("模型层未接受目标漂移来源: %s", item.OriginType)
	}

	if item.SourceGlobalMessageID != nil {
		t.Fatal("目标漂移人工拆项不能绑定全局讨论消息")
	}

	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil ||
		item.DeliveredInstructionVersionID != nil {
		t.Fatal("新问题不能直接进入正式交付历史")
	}

	if item.Status != models.CWReviewItemStatusDetected {
		t.Fatalf("新问题初始状态应为detected，实际为: %s", item.Status)
	}

	if item.PageID == nil || *item.PageID != page.ID {
		t.Fatalf("新问题没有继承当前稳定页面: %#v", item.PageID)
	}

	if item.PageNumberSnapshot != page.PageNumber {
		t.Fatalf(
			"页面快照页码错误: got=%d want=%d",
			item.PageNumberSnapshot,
			page.PageNumber,
		)
	}

	if item.PageHTMLHash != cwAIReviewHash(page.HTMLContent) {
		t.Fatal("新问题必须冻结创建时的当前页面快照")
	}

	teacherView := BuildCWReviewItemTeacherView(item)

	if teacherView.ImprovementGoal != content {
		t.Fatalf(
			"新的独立问题没有保留教师输入: got=%q want=%q",
			teacherView.ImprovementGoal,
			content,
		)
	}

	if !teacherView.ManualCheckRequired {
		t.Fatal("目标漂移人工拆项必须要求教师实际检查")
	}

	if len(teacherView.AcceptanceChecks) < 2 {
		t.Fatalf(
			"新的独立问题缺少教师检查项: %#v",
			teacherView.AcceptanceChecks,
		)
	}

	var evidence map[string]interface{}
	if err := json.Unmarshal(
		[]byte(item.EvidenceJSON),
		&evidence,
	); err != nil {
		t.Fatalf("解析目标漂移内部证据失败: %v", err)
	}

	if evidence["origin_type"] != models.CWReviewItemOriginGoalDriftManual {
		t.Fatalf("内部证据来源类型错误: %#v", evidence)
	}

	if evidence["source_goal_drift_item_id"] != source.ID {
		t.Fatalf("内部证据没有保留来源问题ID: %#v", evidence)
	}

	if _, ok := evidence["teacher_view_snapshot"]; !ok {
		t.Fatal("目标漂移新问题必须固化教师视图快照")
	}
}

func TestCWGoalDriftTeacherTitle(t *testing.T) {
	short := "让学生完成选择后看到清楚反馈"
	if got := cwGoalDriftTeacherTitle(short); got != short {
		t.Fatalf("短标题不应被改写: %q", got)
	}

	long := strings.Repeat("改", cwReviewGoalDriftTitleMaxRunes+20)
	got := cwGoalDriftTeacherTitle(long)

	if utf8.RuneCountInString(got) !=
		cwReviewGoalDriftTitleMaxRunes+1 {
		t.Fatalf(
			"长标题截断长度错误: got=%d",
			utf8.RuneCountInString(got),
		)
	}

	if !strings.HasSuffix(got, "…") {
		t.Fatalf("截断标题应带省略号: %q", got)
	}
}
