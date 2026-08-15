package services

import (
	"errors"
	"testing"
)

func TestNormalizeCoursewareAssistantTTSCharacter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:     "empty defaults to female",
			input:    "",
			expected: coursewareAssistantTTSCharacterFemale,
		},
		{
			name:     "female stays female",
			input:    "female",
			expected: coursewareAssistantTTSCharacterFemale,
		},
		{
			name:     "male ignores spaces and case",
			input:    " MALE ",
			expected: coursewareAssistantTTSCharacterMale,
		},
		{
			name:      "unknown character rejected",
			input:     "robot",
			expectErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := normalizeCoursewareAssistantTTSCharacter(
				testCase.input,
			)

			if testCase.expectErr {
				if !errors.Is(err, ErrCoursewareAssistantTTSInvalidRequest) {
					t.Fatalf(
						"expected invalid request error, got %v",
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if actual != testCase.expected {
				t.Fatalf(
					"expected %q, got %q",
					testCase.expected,
					actual,
				)
			}
		})
	}
}

func TestSelectCoursewareAssistantTTSVoiceByCharacter(t *testing.T) {
	tests := []struct {
		name             string
		text             string
		character        string
		expectedVoice    string
		expectedLanguage string
	}{
		{
			name:             "female chinese uses vivi",
			text:             "我们来认识月相变化。",
			character:        coursewareAssistantTTSCharacterFemale,
			expectedVoice:    coursewareAssistantTTSFemaleChineseVoice,
			expectedLanguage: "zh-CN",
		},
		{
			name:             "male chinese uses yunzhou",
			text:             "我们来认识月相变化。",
			character:        coursewareAssistantTTSCharacterMale,
			expectedVoice:    coursewareAssistantTTSMaleChineseVoice,
			expectedLanguage: "zh-CN",
		},
		{
			name:             "male english keeps tim",
			text:             "Let us learn the phases of the Moon.",
			character:        coursewareAssistantTTSCharacterMale,
			expectedVoice:    coursewareAssistantTTSEnglishVoice,
			expectedLanguage: "en-US",
		},
		{
			name:             "female english keeps tim",
			text:             "A B C D E F G",
			character:        coursewareAssistantTTSCharacterFemale,
			expectedVoice:    coursewareAssistantTTSEnglishVoice,
			expectedLanguage: "en-US",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			voice, language := selectCoursewareAssistantTTSVoice(
				testCase.text,
				testCase.character,
			)

			if voice != testCase.expectedVoice {
				t.Fatalf(
					"expected voice %q, got %q",
					testCase.expectedVoice,
					voice,
				)
			}

			if language != testCase.expectedLanguage {
				t.Fatalf(
					"expected language %q, got %q",
					testCase.expectedLanguage,
					language,
				)
			}
		})
	}
}
