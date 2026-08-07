package services

// lesson_plan_import_service_test.go
//
// 本测试验证上下文12新增的导入创建纯规则：
//   - paste、docx、pdf三种来源白名单；
//   - k12、vocational、adult三个具体域解析；
//   - mixed、空域、冲突、无有效归属和数据库错误均fail-closed；
//   - 显式INSERT收到Service解析域；
//   - 数据库返回快照不一致时拒绝；
//   - review之前阶段统一为skipped，review为in_progress。
//
// 测试通过依赖注入脱离数据库，不修改全局函数变量。

import (
	"context"
	"errors"
	"testing"

	"strings"
	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestNormalizeLessonPlanImportSourceType(
	t *testing.T,
) {
	validSources := []string{
		"paste",
		"docx",
		"pdf",
		" PDF ",
	}

	for _, source := range validSources {
		source := source
		t.Run(source, func(t *testing.T) {
			normalized, err :=
				normalizeLessonPlanImportSourceType(
					source,
				)
			if err != nil {
				t.Fatalf(
					"合法来源被拒绝: %v",
					err,
				)
			}

			if normalized !=
				stringsForImportTest(source) {
				t.Fatalf(
					"来源规范化错误: got=%s",
					normalized,
				)
			}
		})
	}

	_, err := normalizeLessonPlanImportSourceType(
		"word",
	)
	if err == nil ||
		!errors.Is(
			err,
			ErrLPGenImportSourceInvalid,
		) {
		t.Fatalf(
			"非法来源未被正确拒绝: %v",
			err,
		)
	}
}

func TestResolveImportedLessonPlanCreationDomain(
	t *testing.T,
) {
	domains := []string{
		models.EducationDomainK12,
		models.EducationDomainVocational,
		models.EducationDomainAdult,
	}

	for _, domain := range domains {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			deps := lessonPlanImportCreationDeps{
				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					if userID != "user-1" {
						t.Fatalf(
							"读取了错误用户: %s",
							userID,
						)
					}
					return &models.User{
						Role: models.RoleOperator,
					}, nil
				},
				resolveEducationDomain: func(
					ctx context.Context,
					userID string,
					role string,
				) (string, error) {
					if role != models.RoleOperator {
						t.Fatalf(
							"未使用数据库实时角色: %s",
							role,
						)
					}
					return domain, nil
				},
			}

			resolved, err :=
				resolveImportedLessonPlanCreationDomain(
					context.Background(),
					"user-1",
					deps,
				)
			if err != nil {
				t.Fatalf(
					"解析失败: %v",
					err,
				)
			}
			if resolved != domain {
				t.Fatalf(
					"解析域错误: got=%s want=%s",
					resolved,
					domain,
				)
			}
		})
	}
}

func TestResolveImportedLessonPlanCreationDomainRejectsInvalidStates(
	t *testing.T,
) {
	tests := []struct {
		name        string
		resolved    string
		resolveErr  error
		expectedErr error
	}{
		{
			name: "无有效教学组织",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
			expectedErr: ErrLPCreationEducationDomainRequired,
		},
		{
			name: "跨具体教育域冲突",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainConflict,
			expectedErr: ErrLPCreationEducationDomainConflict,
		},
		{
			name: "数据库异常",
			resolveErr: errors.New(
				"database unavailable",
			),
			expectedErr: ErrLPCreationEducationDomainResolveFailed,
		},
		{
			name:        "mixed",
			resolved:    models.EducationDomainMixed,
			expectedErr: ErrLPCreationEducationDomainResolveFailed,
		},
		{
			name:        "common",
			resolved:    models.EducationDomainCommon,
			expectedErr: ErrLPCreationEducationDomainResolveFailed,
		},
		{
			name:        "空教育域",
			resolved:    "",
			expectedErr: ErrLPCreationEducationDomainResolveFailed,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			deps := lessonPlanImportCreationDeps{
				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					return &models.User{
						Role: models.RoleOperator,
					}, nil
				},
				resolveEducationDomain: func(
					ctx context.Context,
					userID string,
					role string,
				) (string, error) {
					return testCase.resolved,
						testCase.resolveErr
				},
			}

			_, err :=
				resolveImportedLessonPlanCreationDomain(
					context.Background(),
					"user-1",
					deps,
				)
			if err == nil {
				t.Fatal(
					"预期失败，实际成功",
				)
			}
			if !errors.Is(
				err,
				testCase.expectedErr,
			) {
				t.Fatalf(
					"错误类型不符: got=%v want=%v",
					err,
					testCase.expectedErr,
				)
			}
		})
	}
}

func TestCreateImportedLessonPlanWithEducationDomain(
	t *testing.T,
) {
	domains := []string{
		models.EducationDomainK12,
		models.EducationDomainVocational,
		models.EducationDomainAdult,
	}

	for _, domain := range domains {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			createCalled := false

			deps := lessonPlanImportCreationDeps{
				createWithEducationDomain: func(
					ctx context.Context,
					lessonPlan *models.LessonPlan,
					educationDomain string,
				) error {
					createCalled = true

					if educationDomain != domain {
						t.Fatalf(
							"显式写域错误: got=%s want=%s",
							educationDomain,
							domain,
						)
					}

					lessonPlan.ID = "plan-1"
					lessonPlan.EducationDomain =
						domain
					return nil
				},
			}

			lessonPlan := &models.LessonPlan{
				AuthorID: "user-1",
			}

			err :=
				createImportedLessonPlanWithEducationDomain(
					context.Background(),
					lessonPlan,
					domain,
					deps,
				)
			if err != nil {
				t.Fatalf(
					"显式创建失败: %v",
					err,
				)
			}
			if !createCalled {
				t.Fatal(
					"未调用显式写域Repository",
				)
			}
			if lessonPlan.EducationDomain != domain {
				t.Fatalf(
					"数据库快照错误: %s",
					lessonPlan.EducationDomain,
				)
			}
		})
	}
}

func TestCreateImportedLessonPlanDetectsStoredDomainMismatch(
	t *testing.T,
) {
	deps := lessonPlanImportCreationDeps{
		createWithEducationDomain: func(
			ctx context.Context,
			lessonPlan *models.LessonPlan,
			educationDomain string,
		) error {
			lessonPlan.ID = "plan-mismatch"
			lessonPlan.EducationDomain =
				models.EducationDomainAdult
			return nil
		},
	}

	lessonPlan := &models.LessonPlan{
		AuthorID: "user-1",
	}

	err := createImportedLessonPlanWithEducationDomain(
		context.Background(),
		lessonPlan,
		models.EducationDomainK12,
		deps,
	)
	if err == nil {
		t.Fatal(
			"快照不一致时预期失败",
		)
	}
	if !errors.Is(
		err,
		ErrLPCreationEducationDomainResolveFailed,
	) {
		t.Fatalf(
			"错误类型不符: %v",
			err,
		)
	}
}

func TestBuildImportedLessonPlanStageOutputs(
	t *testing.T,
) {
	snapshots := []models.StageConfigSnapshot{
		{
			StageCode:  "analyze",
			StageOrder: 1,
		},
		{
			StageCode:  "design",
			StageOrder: 2,
		},
		{
			StageCode:  "write",
			StageOrder: 3,
		},
		{
			StageCode:  "review",
			StageOrder: 4,
		},
		{
			StageCode:  "revise",
			StageOrder: 5,
		},
	}

	outputs, skipped, err :=
		buildImportedLessonPlanStageOutputs(
			snapshots,
		)
	if err != nil {
		t.Fatalf(
			"构建阶段记录失败: %v",
			err,
		)
	}

	if len(outputs) != 4 {
		t.Fatalf(
			"阶段记录数量错误: got=%d want=4",
			len(outputs),
		)
	}
	if len(skipped) != 3 {
		t.Fatalf(
			"跳过阶段数量错误: got=%d want=3",
			len(skipped),
		)
	}

	for index := 0; index < 3; index++ {
		if outputs[index].Status !=
			models.StageOutputSkipped {
			t.Fatalf(
				"前置阶段未标记skipped: index=%d status=%s",
				index,
				outputs[index].Status,
			)
		}
	}

	if outputs[3].StageCode != "review" ||
		outputs[3].Status !=
			models.StageOutputInProgress {
		t.Fatalf(
			"review阶段状态错误: code=%s status=%s",
			outputs[3].StageCode,
			outputs[3].Status,
		)
	}
}

func TestBuildImportedLessonPlanStageOutputsRequiresReview(
	t *testing.T,
) {
	_, _, err :=
		buildImportedLessonPlanStageOutputs(
			[]models.StageConfigSnapshot{
				{
					StageCode:  "analyze",
					StageOrder: 1,
				},
				{
					StageCode:  "write",
					StageOrder: 2,
				},
			},
		)
	if err == nil ||
		!errors.Is(
			err,
			ErrLPGenImportReviewStageRequired,
		) {
		t.Fatalf(
			"缺少review阶段未被拒绝: %v",
			err,
		)
	}
}

func TestBuildImportOpeningMessageEncouragesImmediateConversation(
	t *testing.T,
) {
	request :=
		&models.ImportExistingPlanRequest{
			Subject: "化学",
			Grade:   "九年级",
			Topic:   "金属钠",
			ContentMarkdown: strings.Repeat(
				"教学内容",
				20,
			),
			SourceType: "docx_fidelity",
		}

	message :=
		buildImportOpeningMessage(
			request,
			[]string{
				"analyze",
				"design",
				"write",
			},
		)

	if message == nil {
		t.Fatal(
			"开场消息不应为空",
		)
	}

	requiredTexts := []string{
		"保留原格式Word文档",
		"教学分析 → 教学设计 → 教案撰写",
		"现在可以直接开始",
		"直接开始聊天评审",
		"后台质量检查会独立进行",
		"不影响您继续聊天",
		"原始Word文档和版式内容仍会保留",
	}

	for _, requiredText := range requiredTexts {
		if !strings.Contains(
			message.Content,
			requiredText,
		) {
			t.Fatalf(
				"开场消息缺少必要文案%q：%s",
				requiredText,
				message.Content,
			)
		}
	}

	forbiddenTexts := []string{
		"请稍候",
		"等待右侧出现",
		"右侧面板显示详细评分",
	}

	for _, forbiddenText := range forbiddenTexts {
		if strings.Contains(
			message.Content,
			forbiddenText,
		) {
			t.Fatalf(
				"开场消息仍包含阻塞性文案%q：%s",
				forbiddenText,
				message.Content,
			)
		}
	}
}

// stringsForImportTest 是测试内最小规范化辅助，
// 避免仅为一个断言引入额外生产依赖。
func stringsForImportTest(
	value string,
) string {
	switch value {
	case " PDF ":
		return "pdf"
	default:
		return value
	}
}
