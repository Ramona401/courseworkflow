-- =====================================================================
-- 课件↔教案对齐报告 功能数据库迁移（修正版）
-- 修正：ai_scene_configs 无 created_at 列；prompts 无 updated_at 列
-- 内容：
--   1. 新建 courseware_alignment_reports 表
--   2. ai_scene_configs 插入 courseware_alignment 场景（opus，未授权校自动降级qwen）
--   3. prompts 插入 prompt_courseware_alignment 对齐提示词（is_current=true）
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1. 对齐报告表
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS courseware_alignment_reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    courseware_id UUID NOT NULL REFERENCES coursewares(id) ON DELETE CASCADE,
    lesson_plan_id UUID,
    overall       VARCHAR(20) NOT NULL DEFAULT 'aligned',
    summary       TEXT NOT NULL DEFAULT '',
    report_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
    status        VARCHAR(20) NOT NULL DEFAULT 'generating',
    error_message TEXT NOT NULL DEFAULT '',
    model_used    VARCHAR(100) NOT NULL DEFAULT '',
    tokens_used   INT NOT NULL DEFAULT 0,
    page_count    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cw_alignment_courseware
    ON courseware_alignment_reports(courseware_id);

CREATE INDEX IF NOT EXISTS idx_cw_alignment_status
    ON courseware_alignment_reports(status);

-- ---------------------------------------------------------------------
-- 2. AI 场景配置：courseware_alignment（无 created_at 列）
-- ---------------------------------------------------------------------
INSERT INTO ai_scene_configs (id, scene_code, model, temperature, max_tokens, fallback_models, is_active, updated_at)
VALUES (
    gen_random_uuid(),
    'courseware_alignment',
    'anthropic/claude-opus-4.8',
    0.30,
    8000,
    '[]'::jsonb,
    true,
    now()
)
ON CONFLICT (scene_code) DO UPDATE
    SET model = EXCLUDED.model,
        temperature = EXCLUDED.temperature,
        max_tokens = EXCLUDED.max_tokens,
        updated_at = now();

-- ---------------------------------------------------------------------
-- 3. 对齐提示词：prompt_courseware_alignment（无 updated_at 列）
-- ---------------------------------------------------------------------
UPDATE prompts SET is_current = false WHERE prompt_key = 'prompt_courseware_alignment';

INSERT INTO prompts (id, prompt_key, content, version, is_current, created_at)
VALUES (
    gen_random_uuid(),
    'prompt_courseware_alignment',
    $PROMPT$你是教学设计审查专家。下面给你一份"原始教案"和一份据此生成的"课件逐页方案"。请你严格比对：这份课件方案是否忠实还原了教案的教学意图，有没有遗漏、新增、或教学目标的偏移。

你的任务是给一线老师提供"课件做得对不对、要不要注意"的明确信号，而不是泛泛的好评。要敢于指出问题。

## 比对维度

1. **覆盖度(coverage)**：教案里规划的每个核心教学环节（如导入、新知讲解、例题、练习、活动、小结、作业等），课件方案有没有对应的页覆盖？
   - covered：有对应页且内容相符
   - partial：有对应页但内容明显不足/简略
   - missing：教案有这个环节但课件方案完全没有对应页（最需要老师注意）

2. **新增内容(additions)**：课件方案里出现了教案中没有的内容（AI自由发挥的部分）。老师需要确认这些新增是否合适。

3. **教学意图偏移(intent_shifts)**：某页的教学目的(purpose)与教案对应环节的教学目标相比，有没有方向性的偏移或拔高/降低。

## 输出格式

只输出一个JSON对象，不要任何额外说明、不要markdown代码围栏。结构如下：

{
  "overall": "aligned | minor | major",
  "summary": "一句话总览，30字以内，说明整体忠实程度",
  "coverage": [
    {"plan_segment": "教案里的环节名", "status": "covered|partial|missing", "page_nums": [对应课件页码数组，missing时为空数组], "note": "简短说明，20字内"}
  ],
  "additions": [
    {"page_num": 页码, "desc": "这一页新增了什么教案没有的内容，20字内"}
  ],
  "intent_shifts": [
    {"page_num": 页码, "plan_intent": "教案对应环节的目标", "scheme_purpose": "课件这一页的目的", "note": "偏移点说明，20字内"}
  ]
}

## overall 判定标准

- aligned：核心环节全部covered，无明显遗漏，无方向性偏移（可有少量无害新增）
- minor：有1-2处partial或无害新增，但主线完整
- major：有任一核心环节missing，或有方向性的intent_shift

## 硬性要求

- coverage 必须列出教案的全部核心环节，不能只挑覆盖好的列。missing 和 partial 是这份报告最有价值的部分，务必如实指出。
- 如果某类信号没有（比如没有任何新增），对应数组返回空数组 []，不要编造。
- page_nums / page_num 必须是课件方案里真实存在的页码。
- summary 要客观，有问题就说有问题，不要无脑夸"高度契合"。
$PROMPT$,
    1,
    true,
    now()
);
