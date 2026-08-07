/**
 * 教学智能体发布策略的数据模型与转换函数。
 *
 * 本文件不渲染界面，集中负责：
 *   - 普通教师看到的轮数、学生人数和有效期预设；
 *   - 标准课堂默认策略；
 *   - 从正式部署恢复教师可读策略；
 *   - 自动计算带重试余量的每日调用额度；
 *   - 将教师策略转换为后端发布请求。
 */

import type {
  CoursewareAssistantDeploymentView,
  PublishCoursewareAssistantDeploymentRequest,
} from "@/api/coursewares";

export type CoursewareAssistantTurnPreset =
  | "4"
  | "8"
  | "12"
  | "custom";

export type CoursewareAssistantStudentPreset =
  | "10"
  | "50"
  | "100"
  | "custom";

export type CoursewareAssistantValidityPreset =
  | "class"
  | "7d"
  | "30d"
  | "long"
  | "custom";

export interface CoursewareAssistantDeploymentChoiceOption<
  Value extends string,
> {
  value: Value;
  title: string;
  description: string;
}

export interface CoursewareAssistantDeploymentPolicyDraft {
  dailyCallLimit: string;
  perSessionTurnLimit: string;
  expectedStudents: string;
  turnPreset: CoursewareAssistantTurnPreset;
  studentPreset: CoursewareAssistantStudentPreset;
  validityPreset: CoursewareAssistantValidityPreset;
  validUntil: string;
  externalEnabled: boolean;
  externalOrigins: string;
}

export const COURSEWARE_ASSISTANT_TURN_PRESETS:
  readonly CoursewareAssistantDeploymentChoiceOption<CoursewareAssistantTurnPreset>[] = [
    {
      value: "4",
      title: "快速检查",
      description: "最多4轮",
    },
    {
      value: "8",
      title: "标准互动",
      description: "最多8轮 · 推荐",
    },
    {
      value: "12",
      title: "深入讨论",
      description: "最多12轮",
    },
    {
      value: "custom",
      title: "自定义",
      description: "输入其他轮数",
    },
  ];

export const COURSEWARE_ASSISTANT_STUDENT_PRESETS:
  readonly CoursewareAssistantDeploymentChoiceOption<CoursewareAssistantStudentPreset>[] = [
    {
      value: "10",
      title: "小组",
      description: "约10人",
    },
    {
      value: "50",
      title: "一个班",
      description: "约50人 · 推荐",
    },
    {
      value: "100",
      title: "多个班",
      description: "约100人",
    },
    {
      value: "custom",
      title: "自定义",
      description: "输入其他人数",
    },
  ];

export const COURSEWARE_ASSISTANT_VALIDITY_PRESETS:
  readonly CoursewareAssistantDeploymentChoiceOption<CoursewareAssistantValidityPreset>[] = [
    {
      value: "class",
      title: "本节课",
      description: "约3小时后结束",
    },
    {
      value: "7d",
      title: "7天",
      description: "标准课堂 · 推荐",
    },
    {
      value: "30d",
      title: "30天",
      description: "单元学习",
    },
    {
      value: "long",
      title: "长期有效",
      description: "由老师手动暂停",
    },
    {
      value: "custom",
      title: "自定义时间",
      description: "选择具体日期",
    },
  ];

const THREE_HOURS_MS =
  3 * 60 * 60 * 1000;

const SEVEN_DAYS_MS =
  7 * 24 * 60 * 60 * 1000;

const THIRTY_DAYS_MS =
  30 * 24 * 60 * 60 * 1000;

/**
 * 生成适用于datetime-local输入框的本地时间字符串。
 */
function futureDateTimeLocal(
  milliseconds: number,
): string {
  const date =
    new Date(
      Date.now() +
        milliseconds,
    );

  const offset =
    date.getTimezoneOffset() *
    60000;

  return new Date(
    date.getTime() -
      offset,
  )
    .toISOString()
    .slice(0, 16);
}

/**
 * 把ISO时间转换为datetime-local输入格式。
 */
function toDateTimeLocal(
  value: string | null,
): string {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

  if (
    !Number.isFinite(
      date.getTime(),
    )
  ) {
    return "";
  }

  const offset =
    date.getTimezoneOffset() *
    60000;

  return new Date(
    date.getTime() -
      offset,
  )
    .toISOString()
    .slice(0, 16);
}

/**
 * 按学生人数和每人轮数预留20%重试空间，并向上取整到50。
 */
function calculateDailyCallLimit(
  students: number,
  turns: number,
): number {
  if (
    !Number.isFinite(
      students,
    ) ||
    !Number.isFinite(
      turns,
    ) ||
    students < 1 ||
    turns < 1
  ) {
    return 50;
  }

  const withReserve =
    students *
    turns *
    1.2;

  const rounded =
    Math.ceil(
      withReserve / 50,
    ) * 50;

  return Math.min(
    100000,
    Math.max(
      50,
      rounded,
    ),
  );
}

function presetForTurns(
  turns: number,
): CoursewareAssistantTurnPreset {
  if (turns === 4) {
    return "4";
  }

  if (turns === 8) {
    return "8";
  }

  if (turns === 12) {
    return "12";
  }

  return "custom";
}

function presetForStudents(
  students: number,
): CoursewareAssistantStudentPreset {
  if (students === 10) {
    return "10";
  }

  if (students === 50) {
    return "50";
  }

  if (students === 100) {
    return "100";
  }

  return "custom";
}

/**
 * 根据剩余有效期恢复最接近的教师卡片。
 *
 * 已运行一段时间的7天或30天部署会有自然时间损耗，
 * 因此采用合理区间，而不是要求精确等于完整天数。
 */
function inferValidityPreset(
  value: string | null,
): CoursewareAssistantValidityPreset {
  if (!value) {
    return "long";
  }

  const date =
    new Date(value);

  const remainingHours =
    (
      date.getTime() -
      Date.now()
    ) /
    3600000;

  if (
    remainingHours > 0 &&
    remainingHours <= 6
  ) {
    return "class";
  }

  if (
    remainingHours >=
      5 * 24 &&
    remainingHours <=
      8 * 24
  ) {
    return "7d";
  }

  if (
    remainingHours >=
      25 * 24 &&
    remainingHours <=
      35 * 24
  ) {
    return "30d";
  }

  return "custom";
}

function parseOriginLines(
  value: string,
): string[] {
  const seen =
    new Set<string>();

  const result: string[] =
    [];

  value
    .split(/[\n,，]+/)
    .map(
      (item) =>
        item.trim(),
    )
    .filter(Boolean)
    .forEach(
      (item) => {
        if (
          !seen.has(item)
        ) {
          seen.add(item);
          result.push(item);
        }
      },
    );

  return result;
}

export function createDefaultCoursewareAssistantDeploymentPolicy():
  CoursewareAssistantDeploymentPolicyDraft {
  const students = 50;
  const turns = 8;

  return {
    dailyCallLimit:
      String(
        calculateDailyCallLimit(
          students,
          turns,
        ),
      ),
    perSessionTurnLimit:
      String(turns),
    expectedStudents:
      String(students),
    turnPreset: "8",
    studentPreset: "50",
    validityPreset: "7d",
    validUntil:
      futureDateTimeLocal(
        SEVEN_DAYS_MS,
      ),
    externalEnabled: false,
    externalOrigins: "",
  };
}

export function coursewareAssistantDeploymentPolicyFromLive(
  deployment:
    CoursewareAssistantDeploymentView,
  internalOrigin: string,
): CoursewareAssistantDeploymentPolicyDraft {
  const turns =
    deployment
      .per_session_turn_limit;

  const estimatedStudents =
    Math.max(
      1,
      Math.round(
        deployment
          .daily_call_limit /
          Math.max(
            1,
            turns,
          ) /
          1.2,
      ),
    );

  const externalOrigins =
    deployment.allowed_origins
      .filter(
        (origin) =>
          origin !==
          internalOrigin,
      );

  return {
    dailyCallLimit:
      String(
        deployment
          .daily_call_limit,
      ),
    perSessionTurnLimit:
      String(turns),
    expectedStudents:
      String(
        estimatedStudents,
      ),
    turnPreset:
      presetForTurns(
        turns,
      ),
    studentPreset:
      presetForStudents(
        estimatedStudents,
      ),
    validityPreset:
      inferValidityPreset(
        deployment.valid_until,
      ),
    validUntil:
      toDateTimeLocal(
        deployment.valid_until,
      ),
    externalEnabled:
      externalOrigins.length >
      0,
    externalOrigins:
      externalOrigins.join(
        "\n",
      ),
  };
}

/**
 * 学生人数或轮数变化时同步自动额度。
 */
export function withAutomaticCoursewareAssistantDailyLimit(
  policy:
    CoursewareAssistantDeploymentPolicyDraft,
): CoursewareAssistantDeploymentPolicyDraft {
  const students =
    Number(
      policy.expectedStudents,
    );

  const turns =
    Number(
      policy
        .perSessionTurnLimit,
    );

  if (
    !Number.isInteger(
      students,
    ) ||
    !Number.isInteger(
      turns,
    ) ||
    students < 1 ||
    turns < 1
  ) {
    return policy;
  }

  return {
    ...policy,
    dailyCallLimit:
      String(
        calculateDailyCallLimit(
          students,
          turns,
        ),
      ),
  };
}

export function selectCoursewareAssistantTurnPreset(
  previous:
    CoursewareAssistantDeploymentPolicyDraft,
  preset:
    CoursewareAssistantTurnPreset,
): CoursewareAssistantDeploymentPolicyDraft {
  if (preset === "custom") {
    return {
      ...previous,
      turnPreset: preset,
    };
  }

  return withAutomaticCoursewareAssistantDailyLimit({
    ...previous,
    turnPreset: preset,
    perSessionTurnLimit:
      preset,
  });
}

export function selectCoursewareAssistantStudentPreset(
  previous:
    CoursewareAssistantDeploymentPolicyDraft,
  preset:
    CoursewareAssistantStudentPreset,
): CoursewareAssistantDeploymentPolicyDraft {
  if (preset === "custom") {
    return {
      ...previous,
      studentPreset: preset,
    };
  }

  return withAutomaticCoursewareAssistantDailyLimit({
    ...previous,
    studentPreset: preset,
    expectedStudents:
      preset,
  });
}

export function selectCoursewareAssistantValidityPreset(
  previous:
    CoursewareAssistantDeploymentPolicyDraft,
  preset:
    CoursewareAssistantValidityPreset,
): CoursewareAssistantDeploymentPolicyDraft {
  switch (preset) {
  case "class":
    return {
      ...previous,
      validityPreset: preset,
      validUntil:
        futureDateTimeLocal(
          THREE_HOURS_MS,
        ),
    };

  case "7d":
    return {
      ...previous,
      validityPreset: preset,
      validUntil:
        futureDateTimeLocal(
          SEVEN_DAYS_MS,
        ),
    };

  case "30d":
    return {
      ...previous,
      validityPreset: preset,
      validUntil:
        futureDateTimeLocal(
          THIRTY_DAYS_MS,
        ),
    };

  case "long":
    return {
      ...previous,
      validityPreset: preset,
      validUntil: "",
    };

  default:
    return {
      ...previous,
      validityPreset:
        "custom",
      validUntil:
        previous.validUntil ||
        futureDateTimeLocal(
          SEVEN_DAYS_MS,
        ),
    };
  }
}

export function buildCoursewareAssistantDeploymentRequest(
  policy:
    CoursewareAssistantDeploymentPolicyDraft,
  internalOrigin: string,
): {
  request:
    PublishCoursewareAssistantDeploymentRequest | null;
  error: string;
} {
  const dailyCallLimit =
    Number(
      policy.dailyCallLimit,
    );

  const perSessionTurnLimit =
    Number(
      policy
        .perSessionTurnLimit,
    );

  const expectedStudents =
    Number(
      policy.expectedStudents,
    );

  if (
    !Number.isInteger(
      expectedStudents,
    ) ||
    expectedStudents < 1 ||
    expectedStudents > 5000
  ) {
    return {
      request: null,
      error:
        "预计学生人数必须是1至5000之间的整数。",
    };
  }

  if (
    !Number.isInteger(
      dailyCallLimit,
    ) ||
    dailyCallLimit < 1 ||
    dailyCallLimit > 100000
  ) {
    return {
      request: null,
      error:
        "每日调用额度必须是1至100000之间的整数。",
    };
  }

  if (
    !Number.isInteger(
      perSessionTurnLimit,
    ) ||
    perSessionTurnLimit < 1 ||
    perSessionTurnLimit > 100
  ) {
    return {
      request: null,
      error:
        "每位学生的最大互动轮数必须是1至100之间的整数。",
    };
  }

  if (!internalOrigin) {
    return {
      request: null,
      error:
        "浏览器无法识别当前TE-DNA站点，不能安全发布。",
    };
  }

  const extraOrigins =
    policy.externalEnabled
      ? parseOriginLines(
          policy.externalOrigins,
        )
      : [];

  if (
    policy.externalEnabled &&
    extraOrigins.length === 0
  ) {
    return {
      request: null,
      error:
        "开启外部平台后，至少填写一个精确HTTPS来源。",
    };
  }

  const allowedOrigins =
    Array.from(
      new Set([
        internalOrigin,
        ...extraOrigins,
      ]),
    );

  if (
    allowedOrigins.length > 20
  ) {
    return {
      request: null,
      error:
        "允许来源最多20个，当前TE-DNA站点固定占用1个。",
    };
  }

  let validUntil:
    string | null = null;

  if (
    policy.validityPreset !==
      "long"
  ) {
    if (!policy.validUntil) {
      return {
        request: null,
        error:
          "请选择有效的使用结束时间。",
      };
    }

    const date =
      new Date(
        policy.validUntil,
      );

    if (
      !Number.isFinite(
        date.getTime(),
      ) ||
      date.getTime() <=
        Date.now()
    ) {
      return {
        request: null,
        error:
          "使用结束时间必须晚于当前时间。",
      };
    }

    validUntil =
      date.toISOString();
  }

  return {
    request: {
      daily_call_limit:
        dailyCallLimit,
      per_session_turn_limit:
        perSessionTurnLimit,
      allowed_origins:
        allowedOrigins,
      valid_until:
        validUntil,
    },
    error: "",
  };
}
