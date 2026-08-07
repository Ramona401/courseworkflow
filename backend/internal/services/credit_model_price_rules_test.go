package services

import (
	"math"
	"testing"
	"time"
)

func TestResolveActualModelPrice(
	t *testing.T,
) {
	currentTime := time.Date(
		2026,
		time.July,
		29,
		20,
		0,
		0,
		0,
		time.FixedZone(
			"UTC+8",
			8*60*60,
		),
	)

	testCases := []struct {
		name           string
		model          string
		inputTokens    int
		now            time.Time
		expectedInput  float64
		expectedOutput float64
	}{
		{
			name:           "Claude Opus 4.8版本模型",
			model:          "anthropic/claude-4.8-opus-20260528",
			now:            currentTime,
			expectedInput:  0.005,
			expectedOutput: 0.025,
		},
		{
			name:           "Claude Sonnet 5推广价",
			model:          "anthropic/claude-sonnet-5-20260630",
			now:            currentTime,
			expectedInput:  0.002,
			expectedOutput: 0.010,
		},
		{
			name:  "Claude Sonnet 5推广结束后",
			model: "anthropic/claude-sonnet-5",
			now: time.Date(
				2026,
				time.September,
				1,
				0,
				0,
				0,
				0,
				time.FixedZone(
					"UTC+8",
					8*60*60,
				),
			),
			expectedInput:  0.003,
			expectedOutput: 0.015,
		},
		{
			name:           "Claude Haiku 4.5",
			model:          "anthropic/claude-4.5-haiku-20251001",
			now:            currentTime,
			expectedInput:  0.001,
			expectedOutput: 0.005,
		},
		{
			name:           "Gemini 3.5 Flash",
			model:          "google/gemini-3.5-flash-20260519",
			now:            currentTime,
			expectedInput:  0.0015,
			expectedOutput: 0.009,
		},
		{
			name:           "Gemini 3.1 Pro基础档",
			model:          "google/gemini-3.1-pro-preview-20260219",
			inputTokens:    200000,
			now:            currentTime,
			expectedInput:  0.002,
			expectedOutput: 0.012,
		},
		{
			name:           "Gemini 3.1 Pro高阶档",
			model:          "google/gemini-3.1-pro-preview-20260219",
			inputTokens:    200001,
			now:            currentTime,
			expectedInput:  0.004,
			expectedOutput: 0.018,
		},
		{
			name:           "Qwen 3.7 Max当前五折价",
			model:          "qwen3.7-max",
			now:            currentTime,
			expectedInput:  (6.0 / 7.2) / 1000.0,
			expectedOutput: (18.0 / 7.2) / 1000.0,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				price :=
					resolveActualModelPrice(
						testCase.model,
						testCase.inputTokens,
						testCase.now,
						nil,
					)

				if price == nil {
					t.Fatal(
						"价格规则返回nil",
					)
				}

				if math.Abs(
					price.CostPer1kInput-
						testCase.expectedInput,
				) > 0.000000001 {
					t.Fatalf(
						"输入价不一致：得到%.12f，期望%.12f",
						price.CostPer1kInput,
						testCase.expectedInput,
					)
				}

				if math.Abs(
					price.CostPer1kOutput-
						testCase.expectedOutput,
				) > 0.000000001 {
					t.Fatalf(
						"输出价不一致：得到%.12f，期望%.12f",
						price.CostPer1kOutput,
						testCase.expectedOutput,
					)
				}
			},
		)
	}
}
