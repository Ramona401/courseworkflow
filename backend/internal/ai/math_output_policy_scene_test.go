package ai

import (
	"testing"

	"tedna/internal/models"
)

func TestMathOutputPolicyForSceneCoverage(t *testing.T) {
	tests := []struct {
		name       string
		sceneCode  string
		wantPolicy string
	}{
		{
			name:       "lesson_plan",
			sceneCode:  models.SceneLessonPlan,
			wantPolicy: lessonPlanMathOutputPolicy,
		},
		{
			name:       "lesson_plan_harness",
			sceneCode:  models.SceneLessonPlanHarness,
			wantPolicy: lessonPlanMathOutputPolicy,
		},
		{
			name:       "courseware_index",
			sceneCode:  models.SceneCWIndex,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_scheme",
			sceneCode:  models.SceneCWScheme,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_generate",
			sceneCode:  models.SceneCWGenerate,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_nav_refine",
			sceneCode:  models.SceneCWNavRefine,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_page_refine",
			sceneCode:  models.SceneCWPageRefine,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_topic_direct",
			sceneCode:  models.SceneCWTopicDirect,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "courseware_3d_single",
			sceneCode:  models.SceneCW3DSingle,
			wantPolicy: coursewareMathOutputPolicy,
		},
		{
			name:       "pipeline_scanner_excluded",
			sceneCode:  models.SceneScanner,
			wantPolicy: "",
		},
		{
			name:       "courseware_image_generation_excluded",
			sceneCode:  models.SceneCWImageGen,
			wantPolicy: "",
		},
		{
			name:       "courseware_video_generation_excluded",
			sceneCode:  models.SceneCWVideoGen,
			wantPolicy: "",
		},
		{
			name:       "courseware_tts_excluded",
			sceneCode:  models.SceneCWSubtitleTTS,
			wantPolicy: "",
		},
		{
			name:       "stage_coach_excluded",
			sceneCode:  models.SceneStageCoach,
			wantPolicy: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mathOutputPolicyForScene(test.sceneCode)
			if got != test.wantPolicy {
				t.Fatalf(
					"场景策略不符合预期：scene=%s got_policy=%t want_policy=%t",
					test.sceneCode,
					got != "",
					test.wantPolicy != "",
				)
			}
		})
	}
}

func TestResolveMathOutputPolicyScene(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneCWGenerate}

	if got := resolveMathOutputPolicyScene(cfg, nil); got != models.SceneCWGenerate {
		t.Fatalf("缺少TraceContext时未回退配置场景：got=%s", got)
	}

	traceCtx := &TraceContext{SceneCode: models.SceneLessonPlanHarness}
	if got := resolveMathOutputPolicyScene(cfg, traceCtx); got != models.SceneLessonPlanHarness {
		t.Fatalf("TraceContext场景未取得最高优先级：got=%s", got)
	}
}
