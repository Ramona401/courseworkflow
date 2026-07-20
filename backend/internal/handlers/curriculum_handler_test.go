package handlers

// curriculum_handler_test.go
//
// 覆盖上下文14的K12基础查询HTTP限制：
//   - K12调用真实查询依赖；
//   - vocational、adult、mixed、common、空值和非法值返回成功空数组；
//   - 无归属、区域任命未就绪和跨域冲突返回成功空数组；
//   - 用户、教育域和K12数据查询错误返回500；
//   - 三个响应字段始终保持JSON数组类型。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
)

func TestCurriculumKnowledgePointsEducationDomains(
	t *testing.T,
) {
	tests := []struct {
		name          string
		domain        string
		expectedTotal int
		expectQuery   bool
	}{
		{
			name:          "K12正常查询",
			domain:        models.EducationDomainK12,
			expectedTotal: 1,
			expectQuery:   true,
		},
		{
			name:   "职业教育返回空数组",
			domain: models.EducationDomainVocational,
		},
		{
			name:   "成人教育返回空数组",
			domain: models.EducationDomainAdult,
		},
		{
			name:   "mixed返回空数组",
			domain: models.EducationDomainMixed,
		},
		{
			name:   "common返回空数组",
			domain: models.EducationDomainCommon,
		},
		{
			name: "空域返回空数组",
		},
		{
			name:   "非法域返回空数组",
			domain: "invalid-domain",
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			queryCalled := false

			handler := &CurriculumHandler{
				deps: curriculumHandlerDeps{
					findUser: curriculumTestFindUser,

					resolveEducationDomain: func(
						ctx context.Context,
						userID string,
						role string,
					) (string, error) {
						return testCase.domain, nil
					},

					listKnowledgePoints: func(
						ctx context.Context,
						educationDomain string,
						subject string,
						gradeNum int,
					) ([]*models.CurriculumKP, error) {
						queryCalled = true

						if educationDomain !=
							models.EducationDomainK12 {
							t.Fatalf(
								"非K12域进入数据库查询: %s",
								educationDomain,
							)
						}

						return []*models.CurriculumKP{
							{
								ID:       "kp-1",
								Subject:  subject,
								GradeNum: gradeNum,
								KPCode:   "MATH-3-1",
								KPName:   "测试知识点",
							},
						}, nil
					},
				},
			}

			recorder := httptest.NewRecorder()
			request := curriculumTestRequest(
				"/api/v1/curriculum/knowledge-points?subject=数学&grade=3",
			)

			handler.ListKnowledgePoints(
				recorder,
				request,
			)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"HTTP状态错误: got=%d body=%s",
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if queryCalled != testCase.expectQuery {
				t.Fatalf(
					"查询调用状态错误: got=%v want=%v",
					queryCalled,
					testCase.expectQuery,
				)
			}

			data := curriculumTestResponseData(
				t,
				recorder,
			)
			points := curriculumTestArrayField(
				t,
				data,
				"knowledge_points",
			)

			if len(points) != testCase.expectedTotal {
				t.Fatalf(
					"知识点数量错误: got=%d want=%d",
					len(points),
					testCase.expectedTotal,
				)
			}
		})
	}
}

func TestCurriculumUnavailableAndConflictReturnEmpty(
	t *testing.T,
) {
	tests := []struct {
		name       string
		resolveErr error
	}{
		{
			name: "没有确定教学教育域",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "存在跨教育域归属冲突",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainConflict,
		},
		{
			name: "区域管理员任命未就绪",
			resolveErr: repository.
				ErrRegionAdminEducationDomainNotReady,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			handler := &CurriculumHandler{
				deps: curriculumHandlerDeps{
					findUser: curriculumTestFindUser,

					resolveEducationDomain: func(
						ctx context.Context,
						userID string,
						role string,
					) (string, error) {
						return "", testCase.resolveErr
					},

					listKnowledgePoints: func(
						ctx context.Context,
						educationDomain string,
						subject string,
						gradeNum int,
					) ([]*models.CurriculumKP, error) {
						t.Fatal(
							"无有效教育域时不应查询数据库",
						)
						return nil, nil
					},
				},
			}

			recorder := httptest.NewRecorder()
			request := curriculumTestRequest(
				"/api/v1/curriculum/knowledge-points?subject=数学&grade=3",
			)

			handler.ListKnowledgePoints(
				recorder,
				request,
			)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"HTTP状态错误: got=%d body=%s",
					recorder.Code,
					recorder.Body.String(),
				)
			}

			data := curriculumTestResponseData(
				t,
				recorder,
			)
			points := curriculumTestArrayField(
				t,
				data,
				"knowledge_points",
			)

			if len(points) != 0 {
				t.Fatalf(
					"预期空数组，实际数量=%d",
					len(points),
				)
			}
		})
	}
}

func TestCurriculumTextbookResponsesStayArraysForNonK12(
	t *testing.T,
) {
	handler := &CurriculumHandler{
		deps: curriculumHandlerDeps{
			findUser: curriculumTestFindUser,

			resolveEducationDomain: func(
				ctx context.Context,
				userID string,
				role string,
			) (string, error) {
				return models.EducationDomainAdult, nil
			},

			listTextbookUnits: func(
				ctx context.Context,
				educationDomain string,
				subject string,
				publisher string,
				gradeNum int,
				semester string,
			) ([]*models.TextbookUnit, error) {
				t.Fatal(
					"成人教育不应查询K12教材单元",
				)
				return nil, nil
			},

			listPublishers: func(
				ctx context.Context,
				educationDomain string,
				subject string,
				gradeNum int,
			) ([]string, error) {
				t.Fatal(
					"成人教育不应查询K12出版社",
				)
				return nil, nil
			},
		},
	}

	unitRecorder := httptest.NewRecorder()
	unitRequest := curriculumTestRequest(
		"/api/v1/curriculum/textbook-units?subject=数学&grade=3",
	)

	handler.ListTextbookUnits(
		unitRecorder,
		unitRequest,
	)

	if unitRecorder.Code != http.StatusOK {
		t.Fatalf(
			"教材单元HTTP状态错误: %d",
			unitRecorder.Code,
		)
	}

	unitData := curriculumTestResponseData(
		t,
		unitRecorder,
	)
	units := curriculumTestArrayField(
		t,
		unitData,
		"units",
	)

	if len(units) != 0 {
		t.Fatalf(
			"成人教育教材单元不是空数组: %v",
			units,
		)
	}

	publisherRecorder := httptest.NewRecorder()
	publisherRequest := curriculumTestRequest(
		"/api/v1/curriculum/publishers?subject=数学&grade=3",
	)

	handler.ListPublishers(
		publisherRecorder,
		publisherRequest,
	)

	if publisherRecorder.Code != http.StatusOK {
		t.Fatalf(
			"出版社HTTP状态错误: %d",
			publisherRecorder.Code,
		)
	}

	publisherData := curriculumTestResponseData(
		t,
		publisherRecorder,
	)
	publishers := curriculumTestArrayField(
		t,
		publisherData,
		"publishers",
	)

	if len(publishers) != 0 {
		t.Fatalf(
			"成人教育出版社不是空数组: %v",
			publishers,
		)
	}
}

func TestCurriculumDatabaseErrorsReturn500(
	t *testing.T,
) {
	tests := []struct {
		name string
		deps curriculumHandlerDeps
	}{
		{
			name: "读取用户错误",
			deps: curriculumHandlerDeps{
				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					return nil, errors.New(
						"database unavailable",
					)
				},
			},
		},
		{
			name: "解析教育域错误",
			deps: curriculumHandlerDeps{
				findUser: curriculumTestFindUser,

				resolveEducationDomain: func(
					ctx context.Context,
					userID string,
					role string,
				) (string, error) {
					return "", errors.New(
						"database unavailable",
					)
				},
			},
		},
		{
			name: "K12知识点查询错误",
			deps: curriculumHandlerDeps{
				findUser: curriculumTestFindUser,

				resolveEducationDomain: func(
					ctx context.Context,
					userID string,
					role string,
				) (string, error) {
					return models.EducationDomainK12,
						nil
				},

				listKnowledgePoints: func(
					ctx context.Context,
					educationDomain string,
					subject string,
					gradeNum int,
				) ([]*models.CurriculumKP, error) {
					return nil, errors.New(
						"database unavailable",
					)
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			handler := &CurriculumHandler{
				deps: testCase.deps,
			}

			recorder := httptest.NewRecorder()
			request := curriculumTestRequest(
				"/api/v1/curriculum/knowledge-points?subject=数学&grade=3",
			)

			handler.ListKnowledgePoints(
				recorder,
				request,
			)

			if recorder.Code !=
				http.StatusInternalServerError {
				t.Fatalf(
					"数据库错误未返回500: got=%d body=%s",
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func curriculumTestFindUser(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	return &models.User{
		ID:   userID,
		Role: models.RoleOperator,
	}, nil
}

func curriculumTestRequest(
	target string,
) *http.Request {
	request := httptest.NewRequest(
		http.MethodGet,
		target,
		nil,
	)

	claims := &services.JWTClaims{
		UserID: "user-1",
		Role:   models.RoleOperator,
	}

	ctx := context.WithValue(
		request.Context(),
		middleware.ClaimsKey,
		claims,
	)

	return request.WithContext(ctx)
}

func curriculumTestResponseData(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) map[string]interface{} {
	t.Helper()

	var payload map[string]interface{}
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&payload,
	); err != nil {
		t.Fatalf(
			"解析响应JSON失败: %v body=%s",
			err,
			recorder.Body.String(),
		)
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf(
			"响应缺少data对象: %v",
			payload,
		)
	}

	return data
}

func curriculumTestArrayField(
	t *testing.T,
	data map[string]interface{},
	field string,
) []interface{} {
	t.Helper()

	value, exists := data[field]
	if !exists {
		t.Fatalf(
			"响应缺少字段%s: %v",
			field,
			data,
		)
	}

	array, ok := value.([]interface{})
	if !ok {
		t.Fatalf(
			"字段%s不是数组: type=%T value=%v",
			field,
			value,
			value,
		)
	}

	return array
}
