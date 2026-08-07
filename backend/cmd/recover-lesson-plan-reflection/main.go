package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
)

const (
	targetLessonPlanID = "fe788215-5150-45ea-8221-d4d19667c04e"

	expectedPlanVersion  = 19
	expectedWordVersion  = 19
	expectedImageCount   = 8
	expectedMessageCount = 14

	expectedWordSHA256 = "f6182ea15889aba6a247aab71e074868447aa6e17267760588040f21f5b00f72"
)

type wordMetrics struct {
	ImageCount int `json:"image_count"`
}

func main() {
	os.Exit(run())
}

func run() int {
	apply := flag.Bool(
		"apply",
		false,
		"执行确定性恢复并确认个人发布",
	)
	flag.Parse()

	cfg := config.Load()
	database.Init(cfg)
	defer database.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Minute,
	)
	defer cancel()

	plan, wordDocument, messageCount, err :=
		loadAndValidateRecoveryBaseline(ctx)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"恢复基线校验失败:",
			err,
		)
		return 1
	}

	nextContent, err :=
		buildReflectionRecoveryMarkdown(
			plan.ContentMarkdown,
		)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"生成确定性恢复正文失败:",
			err,
		)
		return 1
	}

	fmt.Println(
		"确定性恢复预检通过",
	)
	fmt.Println(
		"教案ID:",
		plan.ID,
	)
	fmt.Println(
		"当前正文版本:",
		plan.Version,
	)
	fmt.Println(
		"当前Word版本:",
		wordDocument.Version,
	)
	fmt.Println(
		"当前图片数量:",
		expectedImageCount,
	)
	fmt.Println(
		"当前聊天消息数量:",
		messageCount,
	)
	fmt.Println(
		"恢复后换行数量:",
		strings.Count(nextContent, "\n"),
	)
	fmt.Println(
		"恢复内容:",
		"仅教后反思三个既有位置",
	)
	fmt.Println(
		"排除内容:",
		"补充建议及聊天稿中的其它全文改写",
	)

	if !*apply {
		fmt.Println(
			"当前为干跑模式，未修改数据库或Word文件",
		)
		return 0
	}

	result, err :=
		services.
			UpdateLessonPlanContentPreservingWord(
				ctx,
				services.
					LessonPlanContentMutationInput{
					PlanID:            plan.ID,
					CallerID:          plan.AuthorID,
					Title:             plan.Title,
					ContentMarkdown:   nextContent,
					ContentStructured: plan.ContentStructured,
					DurationMinutes:   plan.DurationMinutes,
					ExpectedVersion:   plan.Version,
					ExpectedContent:   plan.ContentMarkdown,
					ChangeSource: models.
						LessonPlanWordChangeSourceManual,
					ChangeSummary: "确定性恢复老师已确认的教后反思三段",
				},
			)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"正文与Word原子恢复失败:",
			err,
		)
		return 1
	}
	if !result.Changed ||
		result.CurrentVersion !=
			expectedPlanVersion+1 {
		fmt.Fprintln(
			os.Stderr,
			"恢复结果版本异常:",
			result.CurrentVersion,
		)
		return 1
	}

	if err :=
		repository.
			CommitLessonPlanPersonalPublishAtVersion(
				ctx,
				plan.ID,
				plan.AuthorID,
				result.CurrentVersion,
			); err != nil {
		fmt.Fprintln(
			os.Stderr,
			"恢复后发布确认失败:",
			err,
		)
		return 1
	}

	if err :=
		verifyAppliedRecovery(
			ctx,
			wordDocument.CurrentFileSHA256,
			messageCount,
		); err != nil {
		fmt.Fprintln(
			os.Stderr,
			"恢复后完整性验证失败:",
			err,
		)
		return 1
	}

	fmt.Println(
		"✅ 教后反思确定性恢复完成",
	)
	fmt.Println(
		"✅ 正文版本: v20",
	)
	fmt.Println(
		"✅ Word版本: v20",
	)
	fmt.Println(
		"✅ Word状态: active",
	)
	fmt.Println(
		"✅ 保留图片: 8",
	)
	fmt.Println(
		"✅ 发布状态: published_personal",
	)
	fmt.Println(
		"✅ 已删除Logo未恢复",
	)
	fmt.Println(
		"✅ 补充建议未写入正文",
	)

	return 0
}

func loadAndValidateRecoveryBaseline(
	ctx context.Context,
) (
	*models.LessonPlan,
	*models.LessonPlanWordDocument,
	int,
	error,
) {
	plan, err :=
		repository.GetLessonPlanByID(
			ctx,
			targetLessonPlanID,
		)
	if err != nil {
		return nil, nil, 0, err
	}

	if plan.Version != expectedPlanVersion {
		return nil, nil, 0, fmt.Errorf(
			"正文版本不是v%d，而是v%d",
			expectedPlanVersion,
			plan.Version,
		)
	}
	if plan.Status !=
		models.LPStatusPublishedPersonal {
		return nil, nil, 0, fmt.Errorf(
			"教案状态不是published_personal: %s",
			plan.Status,
		)
	}

	wordDocument, err :=
		repository.
			GetLessonPlanWordDocumentForOwner(
				ctx,
				plan.ID,
				plan.AuthorID,
			)
	if err != nil {
		return nil, nil, 0, err
	}

	if wordDocument.Version !=
		expectedWordVersion {
		return nil, nil, 0, fmt.Errorf(
			"Word版本不是v%d，而是v%d",
			expectedWordVersion,
			wordDocument.Version,
		)
	}
	if wordDocument.Status !=
		models.
			LessonPlanWordDocumentStatusActive {
		return nil, nil, 0, fmt.Errorf(
			"Word状态不是active: %s",
			wordDocument.Status,
		)
	}
	if !strings.EqualFold(
		wordDocument.CurrentFileSHA256,
		expectedWordSHA256,
	) {
		return nil, nil, 0, fmt.Errorf(
			"Word文件哈希已变化: %s",
			wordDocument.CurrentFileSHA256,
		)
	}
	if wordDocument.SemanticMarkdown !=
		plan.ContentMarkdown {
		return nil, nil, 0, fmt.Errorf(
			"Word语义正文与平台正文不同步",
		)
	}

	var metrics wordMetrics
	if err := json.Unmarshal(
		[]byte(wordDocument.MetricsJSON),
		&metrics,
	); err != nil {
		return nil, nil, 0, fmt.Errorf(
			"解析Word指标失败: %w",
			err,
		)
	}
	if metrics.ImageCount !=
		expectedImageCount {
		return nil, nil, 0, fmt.Errorf(
			"图片数量不是%d，而是%d",
			expectedImageCount,
			metrics.ImageCount,
		)
	}

	messageCount, err :=
		loadConversationMessageCount(
			ctx,
			plan.ID,
		)
	if err != nil {
		return nil, nil, 0, err
	}
	if messageCount !=
		expectedMessageCount {
		return nil, nil, 0, fmt.Errorf(
			"聊天消息数量不是%d，而是%d",
			expectedMessageCount,
			messageCount,
		)
	}

	return plan, wordDocument, messageCount, nil
}

func loadConversationMessageCount(
	ctx context.Context,
	planID string,
) (int, error) {
	var count int
	err := database.DB.QueryRow(
		ctx,
		`
SELECT JSONB_ARRAY_LENGTH(
	COALESCE(conversation_log, '[]'::jsonb)
)
FROM lesson_plans
WHERE id = $1
  AND deleted_at IS NULL
`,
		planID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf(
			"读取聊天消息数量失败: %w",
			err,
		)
	}
	return count, nil
}

func verifyAppliedRecovery(
	ctx context.Context,
	previousFileSHA256 string,
	expectedMessages int,
) error {
	plan, err :=
		repository.GetLessonPlanByID(
			ctx,
			targetLessonPlanID,
		)
	if err != nil {
		return err
	}

	if plan.Version !=
		expectedPlanVersion+1 {
		return fmt.Errorf(
			"正文版本不是v20: v%d",
			plan.Version,
		)
	}
	if plan.Status !=
		models.LPStatusPublishedPersonal {
		return fmt.Errorf(
			"发布状态异常: %s",
			plan.Status,
		)
	}
	if err :=
		validateRecoveredReflection(
			plan.ContentMarkdown,
		); err != nil {
		return err
	}

	wordDocument, err :=
		repository.
			GetLessonPlanWordDocumentForOwner(
				ctx,
				plan.ID,
				plan.AuthorID,
			)
	if err != nil {
		return err
	}

	if wordDocument.Version !=
		expectedWordVersion+1 {
		return fmt.Errorf(
			"Word版本不是v20: v%d",
			wordDocument.Version,
		)
	}
	if wordDocument.Status !=
		models.
			LessonPlanWordDocumentStatusActive {
		return fmt.Errorf(
			"Word状态异常: %s",
			wordDocument.Status,
		)
	}
	if wordDocument.SemanticMarkdown !=
		plan.ContentMarkdown {
		return fmt.Errorf(
			"恢复后Word与平台正文不同步",
		)
	}
	if strings.EqualFold(
		wordDocument.CurrentFileSHA256,
		previousFileSHA256,
	) {
		return fmt.Errorf(
			"恢复后Word文件哈希没有变化",
		)
	}

	var metrics wordMetrics
	if err := json.Unmarshal(
		[]byte(wordDocument.MetricsJSON),
		&metrics,
	); err != nil {
		return err
	}
	if metrics.ImageCount !=
		expectedImageCount {
		return fmt.Errorf(
			"恢复后图片数量异常: %d",
			metrics.ImageCount,
		)
	}

	messageCount, err :=
		loadConversationMessageCount(
			ctx,
			plan.ID,
		)
	if err != nil {
		return err
	}
	if messageCount != expectedMessages {
		return fmt.Errorf(
			"恢复过程改变了聊天消息数量: %d",
			messageCount,
		)
	}

	var snapshotCount int
	err = database.DB.QueryRow(
		ctx,
		`
SELECT COUNT(*)
FROM lesson_plan_word_document_versions
WHERE lesson_plan_id = $1
  AND version = $2
  AND file_sha256 = $3
  AND semantic_markdown = $4
`,
		plan.ID,
		wordDocument.Version,
		wordDocument.CurrentFileSHA256,
		wordDocument.SemanticMarkdown,
	).Scan(&snapshotCount)
	if err != nil {
		return fmt.Errorf(
			"验证Word不可变快照失败: %w",
			err,
		)
	}
	if snapshotCount != 1 {
		return fmt.Errorf(
			"Word v20不可变快照数量异常: %d",
			snapshotCount,
		)
	}

	return nil
}
