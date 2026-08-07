/**
 * MyAssistantsPage.tsx — 我的AI助手与提示词工坊主页面
 *
 * 页面是一块对话画布，老师可以：
 *   1. 和AI对话创建助手；
 *   2. 参考已有助手；
 *   3. 粘贴已有提示词；
 *   4. 从历史教案生成教学风格画像。
 *
 * 教育域适配：
 *   - 学科来自当前组织课程目录；
 *   - 学习层级来自education-domain/options.ts统一定义；
 *   - K12显示一年级至高三、小学/初中/高中；
 *   - 职业教育显示职一、职二、职三和中职不限年级；
 *   - 成人教育显示入门、进阶、高级、管理者和不限层级；
 *   - 页面显示值与数据库规范值分离，例如显示“职一”，
 *     实际传递和保存“中职Ⅰ年级”。
 *
 * 职责边界：
 *   本页只负责创建和管理助手；
 *   助手的实际使用入口在备课工坊。
 */

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useSubjects } from "@/hooks/useSubjects";
import { useEducationProfile } from "@/hooks/useEducationProfile";
import {
  getAssistantLevelOptions,
  normalizeEducationLevelValue,
} from "@/education-domain/options";
import { useAuth } from "@/store/auth";
import {
  listAssistants,
  getAssistant,
  forkAssistant,
  deleteAssistant,
  ASSISTANT_SOURCE_LABELS,
  ASSISTANT_SOURCE_EMOJI,
  ASSISTANT_SCENE_LABELS,
  type AIAssistantListItem,
  type AssistantScene,
  type AssistantSource,
} from "@/api/ai-assistants";
import AssistantDesignerPanel from "@/components/ai-assistants/AssistantDesignerPanel";
import SaveAssistantModal from "@/components/ai-assistants/SaveAssistantModal";
import PastePromptModal from "@/components/ai-assistants/PastePromptModal";
import AssistantEditModal from "@/components/ai-assistants/AssistantEditModal";
import AssistantStyleProfileModal from "./AssistantStyleProfileModal";
import DrawerAssistantCard, { miniBtn } from "./DrawerAssistantCard";

/* ==================== 页面样式 ==================== */

const C = {
  primary: "#4F7BE8",
  primaryLight: "rgba(79,123,232,0.08)",
  accent: "#F59E0B",
  success: "#10B981",
  danger: "#EF4444",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  bg: "#FAFBFC",
  card: "#FFFFFF",
  border: "#F3F4F6",
  borderMid: "#E5E7EB",
  groupAccent: "#F59E0B",
};

/** 记住老师当前选择的课程和学习层级。 */
const LS_KEY = "tedna_my_assistants_subject";

const LS_GRADE_KEY = "tedna_my_assistants_grade";

/** 对话式创作默认覆盖备课工坊全部阶段。 */
const WORKSHOP_SCENES: AssistantScene[] = [
  "workshop_analyze",
  "workshop_design",
  "workshop_write",
  "workshop_review",
  "workshop_revise",
];

/** 按角色返回顶部身份提示。 */
function roleHint(role: string | undefined): string {
  if (role === "admin") {
    return "和AI聊一聊就能造助手。你可以存为自己用，也可以发布为本校推荐或全平台系统助手。";
  }

  if (role === "senior_operator") {
    return "和AI聊一聊就能造助手。你可以只给自己用，也可以发布为本校推荐。";
  }

  return "和AI聊一聊，就能造出懂你的备课助手。想参考现成助手，可点右上角“现成助手”。";
}

/* ==================== 主页面 ==================== */

export default function MyAssistantsPage() {
  const { user } = useAuth();
  const role = user?.role;
  const [searchParams] = useSearchParams();

  const requestedSceneValue = (searchParams.get("scene") || "").trim();

  const requestedScene: AssistantScene | undefined =
    requestedSceneValue in ASSISTANT_SCENE_LABELS
      ? (requestedSceneValue as AssistantScene)
      : undefined;

  const requestedSceneLabel = requestedScene
    ? ASSISTANT_SCENE_LABELS[requestedScene]
    : "";

  const requestedSubject = (searchParams.get("subject") || "").trim();

  const requestedGrade = (searchParams.get("grade") || "").trim();

  const requestedAssistantID = (searchParams.get("assistant_id") || "").trim();

  const shouldOpenDrawer =
    searchParams.get("drawer") === "1" || Boolean(requestedScene);

  const { domain, profile } = useEducationProfile();

  const {
    subjects: subjectOptions,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects({
    withAny: true,
  });

  /**
   * 当前教育域可使用的助手层级。
   *
   * mixed管理上下文由统一选项层兼容为K12；
   * 进入具体教案后，后端仍会按教案教育域快照收敛。
   */
  const assistantLevelOptions = useMemo(
    () => getAssistantLevelOptions(domain),
    [domain],
  );

  const assistantLevelValues = useMemo(
    () => assistantLevelOptions.map((option) => option.value),
    [assistantLevelOptions],
  );

  /* ==================== 课程状态 ==================== */

  const [subject, setSubject] = useState<string>(() => {
    try {
      return requestedSubject || localStorage.getItem(LS_KEY) || "";
    } catch {
      return "";
    }
  });

  useEffect(() => {
    try {
      localStorage.setItem(LS_KEY, subject);
    } catch {
      // 浏览器禁用本地存储时静默忽略。
    }
  }, [subject]);

  /**
   * 切换账号或教育域后，清理旧课程值。
   * 空字符串只表示助手列表中的“全部课程”筛选。
   */
  useEffect(() => {
    if (subjectsLoading) return;

    if (!subjectOptions.includes(subject)) {
      setSubject("");
    }
  }, [subjectsLoading, subjectOptions, subject]);

  /* ==================== 学习层级状态 ==================== */

  const [grade, setGrade] = useState<string>(() => {
    try {
      return requestedGrade || localStorage.getItem(LS_GRADE_KEY) || "";
    } catch {
      return "";
    }
  });

  useEffect(() => {
    try {
      localStorage.setItem(LS_GRADE_KEY, grade);
    } catch {
      // 浏览器禁用本地存储时静默忽略。
    }
  }, [grade]);

  /**
   * 切换教育域后，把历史别名规范化为当前域正式值。
   *
   * 例如：
   *   职一 → 中职Ⅰ年级；
   *   中职二年级 → 中职Ⅱ年级；
   *   进阶 → 成人进阶。
   *
   * 跨域旧值无法规范化时回退到当前域第一个具体层级。
   */
  useEffect(() => {
    if (assistantLevelOptions.length === 0) {
      if (grade) setGrade("");
      return;
    }

    const raw = grade.trim();

    if (raw === "" && assistantLevelValues.includes("")) {
      return;
    }

    const normalized = normalizeEducationLevelValue(domain, raw);

    if (normalized && assistantLevelValues.includes(normalized)) {
      if (normalized !== grade) {
        setGrade(normalized);
      }
      return;
    }

    setGrade(assistantLevelOptions[0].value);
  }, [domain, assistantLevelOptions, assistantLevelValues, grade]);

  /* ==================== 现成助手列表 ==================== */

  const [related, setRelated] = useState<AIAssistantListItem[]>([]);

  const [loading, setLoading] = useState(false);

  const [listErr, setListErr] = useState<string | null>(null);

  const [forkingId, setForkingId] = useState<string | null>(null);

  const [analyzingId, setAnalyzingId] = useState<string | null>(null);

  const [deletingId, setDeletingId] = useState<string | null>(null);

  /* ==================== 页面弹层与画布状态 ==================== */

  const [drawerOpen, setDrawerOpen] = useState(false);

  const [injected, setInjected] = useState("");

  const [saveOpen, setSaveOpen] = useState(false);

  const [draftToSave, setDraftToSave] = useState("");

  const [pasteOpen, setPasteOpen] = useState(false);

  const [styleProfileOpen, setStyleProfileOpen] = useState(false);

  const [editId, setEditId] = useState<string | null>(null);

  const [createContextAssistantOpen, setCreateContextAssistantOpen] =
    useState(false);

  const deepLinkHandledRef = useRef("");

  const [banner, setBanner] = useState<string | null>(null);

  /* ==================== 加载现成助手 ==================== */

  const loadRelated = useCallback(async () => {
    if (subjectsLoading) return;

    if (subjectsEmpty) {
      setRelated([]);
      setListErr(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    setListErr(null);

    try {
      const response = await listAssistants({
        scene: requestedScene,
        subject: subject || undefined,
        grade: requestedScene && grade ? grade : undefined,
      });

      const items = response.assistants || [];

      setRelated(items);

      const deepLinkKey = [
        requestedScene || "",
        requestedAssistantID,
        subject,
        grade,
      ].join("|");

      if (shouldOpenDrawer && deepLinkHandledRef.current !== deepLinkKey) {
        setDrawerOpen(true);

        if (requestedAssistantID) {
          const selected = items.find(
            (item) => item.id === requestedAssistantID,
          );

          if (selected?.can_edit) {
            setEditId(selected.id);
          } else if (selected) {
            setBanner(
              `ℹ️ “${selected.name}”可以在课件审核中使用，但当前账号不能编辑它。`,
            );
          } else {
            setBanner(
              "ℹ️ 当前精准条件下未找到刚才选择的助手，请检查其学科、具体年级和课件审核场景。",
            );
          }
        }

        deepLinkHandledRef.current = deepLinkKey;
      }
    } catch (error) {
      setListErr(error instanceof Error ? error.message : "加载助手列表失败");
      setRelated([]);
    } finally {
      setLoading(false);
    }
  }, [
    subject,
    grade,
    requestedScene,
    requestedAssistantID,
    shouldOpenDrawer,
    subjectsLoading,
    subjectsEmpty,
  ]);

  useEffect(() => {
    void loadRelated();
  }, [loadRelated]);

  useEffect(() => {
    if (!banner) return;

    const timer = setTimeout(() => setBanner(null), 5000);

    return () => clearTimeout(timer);
  }, [banner]);

  /* ==================== 对话画布辅助 ==================== */

  const injectToCanvas = useCallback((text: string) => {
    setInjected("");

    setTimeout(() => setInjected(text), 30);
  }, []);

  const handleUseStyleProfile = useCallback(
    (profileText: string) => {
      const guide =
        `下面是根据我的历史教案和教研材料形成、并由我确认后的《教学风格与成长画像》。\n\n` +
        `${profileText}\n\n` +
        `请不要机械复刻任何一份旧教案。请先帮我确认哪些内容应该成为长期助手规则，` +
        `哪些只适合作为提醒；然后结合组件知识库，和我一起生成一位既尊重我的教学优势、` +
        `又会主动指出问题并推动我持续提升的助手。`;

      setStyleProfileOpen(false);
      injectToCanvas(guide);
    },
    [injectToCanvas],
  );

  const handleApplyDraft = useCallback((draft: string) => {
    setDraftToSave(draft);
    setSaveOpen(true);
  }, []);

  /* ==================== 保存与管理操作 ==================== */

  const handleSaved = useCallback(
    (_id: string, source: AssistantSource) => {
      setSaveOpen(false);

      if (source === "group") {
        setBanner("✅ 已发布为共享助手，授权范围内的老师备课时可以选用。");
      } else if (source === "system") {
        setBanner("✅ 已发布为系统助手，全平台老师可以选用。");
      } else {
        setBanner("✅ 助手已保存，备课时可在工坊里选用。");
      }

      void loadRelated();
    },
    [loadRelated],
  );

  const handleFork = useCallback(
    async (item: AIAssistantListItem) => {
      if (forkingId) return;

      setForkingId(item.id);

      try {
        const forked = await forkAssistant(item.id);

        setBanner(
          `✅ 已复制为“${forked.name}”，可在工坊选用，也可继续让AI修改。`,
        );

        await loadRelated();
      } catch (error) {
        setBanner(
          `⚠️ 复制失败：${error instanceof Error ? error.message : "请重试"}`,
        );
      } finally {
        setForkingId(null);
      }
    },
    [forkingId, loadRelated],
  );

  const handleAnalyze = useCallback(
    async (item: AIAssistantListItem) => {
      if (analyzingId) return;

      setAnalyzingId(item.id);

      try {
        const full = await getAssistant(item.id);

        if (full.prompt_protected) {
          setBanner(
            `⚠️ 作者未开放“${item.name}”的提示词原文，无法交给AI分析，但仍可在备课工坊直接使用。`,
          );
          return;
        }

        const prompt = full.full_prompt || "(这个助手没有可读的设定内容)";

        const guide =
          `我想参考“${item.name}”这个助手，造一个类似的。\n\n` +
          `它的完整设定是：\n${prompt}\n\n` +
          `请帮我分析它的设计思路，并和我讨论：基于我的需求，我可以从哪些角度补充或改进？`;

        setDrawerOpen(false);
        injectToCanvas(guide);
      } catch (error) {
        setBanner(
          `⚠️ 读取助手失败：${
            error instanceof Error ? error.message : "请重试"
          }`,
        );
      } finally {
        setAnalyzingId(null);
      }
    },
    [analyzingId, injectToCanvas],
  );

  const handleEdit = useCallback((item: AIAssistantListItem) => {
    setEditId(item.id);
  }, []);

  const handleEditSaved = useCallback(() => {
    setEditId(null);
    setBanner("✅ 助手已更新");
    void loadRelated();
  }, [loadRelated]);

  const handleDelete = useCallback(
    async (item: AIAssistantListItem) => {
      if (deletingId) return;

      const confirmed = window.confirm(
        `确定删除助手“${item.name}”吗？\n此操作不可恢复。`,
      );

      if (!confirmed) return;

      setDeletingId(item.id);

      try {
        await deleteAssistant(item.id);

        setBanner(`✅ 已删除“${item.name}”`);

        await loadRelated();
      } catch (error) {
        setBanner(
          `⚠️ 删除失败：${error instanceof Error ? error.message : "请重试"}`,
        );
      } finally {
        setDeletingId(null);
      }
    },
    [deletingId, loadRelated],
  );

  const handlePasteSaveDirect = useCallback((text: string) => {
    setPasteOpen(false);
    setDraftToSave(text);
    setSaveOpen(true);
  }, []);

  const handlePasteSendToAI = useCallback(
    (text: string) => {
      setPasteOpen(false);

      const guide =
        `这是我已经写好的一段提示词，请帮我审阅并润色，让它更清晰、更适合做备课助手：\n\n` +
        `${text}\n\n` +
        `请指出可以改进的地方，我们一起把它打磨好。`;

      injectToCanvas(guide);
    },
    [injectToCanvas],
  );

  /* ==================== 助手分组 ==================== */

  const grouped: Record<AssistantSource, AIAssistantListItem[]> = {
    system: [],
    group: [],
    personal: [],
  };

  for (const assistant of related) {
    grouped[assistant.source].push(assistant);
  }

  const groupCount = grouped.group.length;

  const totalCount = related.length;

  const groupAnalyzable = grouped.group.some(
    (assistant) => assistant.can_view_prompt,
  );

  const groupForkable = grouped.group.some((assistant) => assistant.can_fork);

  const displayOrder: AssistantSource[] = ["group", "personal", "system"];

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: "12px",
        height: "100%",
        position: "relative",
      }}
    >
      {/* ==================== 顶栏 ==================== */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "14px",
          flexWrap: "wrap",
          padding: "12px 16px",
          background: C.card,
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          flexShrink: 0,
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "8px",
          }}
        >
          <span
            style={{
              fontSize: "13px",
              fontWeight: 600,
              color: C.text,
            }}
          >
            我主要教
          </span>

          <select
            value={subject}
            disabled={subjectsLoading || subjectsEmpty}
            onChange={(event) => setSubject(event.target.value)}
            style={{
              padding: "6px 11px",
              borderRadius: "8px",
              border: `1px solid ${C.borderMid}`,
              fontSize: "14px",
              fontWeight: 600,
              color: C.primary,
              background: C.primaryLight,
              cursor: "pointer",
              outline: "none",
            }}
          >
            {subjectOptions.map((option) => (
              <option key={option} value={option}>
                {option || "(全部课程)"}
              </option>
            ))}
          </select>

          {subjectsEmpty && !subjectsLoading && (
            <span
              style={{
                fontSize: "11px",
                color: C.danger,
                whiteSpace: "nowrap",
              }}
            >
              当前组织未配置课程
            </span>
          )}
        </div>

        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "8px",
          }}
        >
          <span
            style={{
              fontSize: "13px",
              fontWeight: 600,
              color: C.text,
            }}
          >
            {profile.grade_label}
          </span>

          <select
            value={grade}
            onChange={(event) => setGrade(event.target.value)}
            style={{
              padding: "6px 11px",
              borderRadius: "8px",
              border: `1px solid ${C.borderMid}`,
              fontSize: "14px",
              fontWeight: 600,
              color: C.primary,
              background: C.primaryLight,
              cursor: "pointer",
              outline: "none",
            }}
          >
            {assistantLevelOptions.map((option) => (
              <option key={option.value || "__empty__"} value={option.value}>
                {option.label}
                {option.automatic ? "" : "（仅手动）"}
              </option>
            ))}
          </select>
        </div>

        <div
          style={{
            flex: 1,
            minWidth: "160px",
            fontSize: "12px",
            color: C.textSec,
            lineHeight: 1.6,
          }}
        >
          {roleHint(role)}
        </div>

        <button
          type="button"
          onClick={() => setStyleProfileOpen(true)}
          style={{
            display: "flex",
            alignItems: "center",
            gap: "6px",
            padding: "8px 14px",
            borderRadius: "9px",
            border: "1px solid rgba(16,185,129,0.45)",
            background: "rgba(16,185,129,0.08)",
            color: "#047857",
            fontSize: "13px",
            fontWeight: 650,
            cursor: "pointer",
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
          title="选择自己的历史教案或上传教研材料，提取教学风格与成长方向"
        >
          📚 从我的教案生成
        </button>

        <button
          type="button"
          onClick={() => setPasteOpen(true)}
          style={{
            display: "flex",
            alignItems: "center",
            gap: "6px",
            padding: "8px 14px",
            borderRadius: "9px",
            border: `1px solid ${C.borderMid}`,
            background: "#fff",
            color: C.textSec,
            fontSize: "13px",
            fontWeight: 600,
            cursor: "pointer",
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
          title="已有现成提示词时，可直接粘贴保存或交给AI润色"
        >
          ✍️ 粘贴提示词
        </button>

        <button
          type="button"
          onClick={() => setDrawerOpen(true)}
          style={{
            display: "flex",
            alignItems: "center",
            gap: "6px",
            padding: "8px 14px",
            borderRadius: "9px",
            border: `1px solid ${C.primary}`,
            background: C.primaryLight,
            color: C.primary,
            fontSize: "13px",
            fontWeight: 600,
            cursor: "pointer",
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
          title="查看现成助手，可参考、复制或交给AI分析"
        >
          📚 现成助手
          {totalCount > 0 ? ` (${totalCount})` : ""}
        </button>
      </div>

      {requestedScene && (
        <div
          style={{
            padding: "10px 14px",
            borderRadius: "10px",
            background: "rgba(79,123,232,0.08)",
            border: "1px solid rgba(79,123,232,0.24)",
            color: "#334155",
            fontSize: "12px",
            lineHeight: 1.6,
            display: "flex",
            alignItems: "center",
            gap: "10px",
            flexWrap: "wrap",
            flexShrink: 0,
          }}
        >
          <span
            style={{
              flex: 1,
              minWidth: "220px",
            }}
          >
            🎯 正在管理
            <b>“{requestedSceneLabel}”</b>
            助手。列表只显示与
            <b>{subject || "当前学科"}</b>、<b>{grade || "当前具体层级"}</b>
            精准匹配的候选。
          </span>

          <button
            type="button"
            onClick={() => setCreateContextAssistantOpen(true)}
            style={{
              padding: "7px 12px",
              borderRadius: "8px",
              border: `1px solid ${C.primary}`,
              background: C.primaryLight,
              color: C.primary,
              fontSize: "12px",
              fontWeight: 650,
              cursor: "pointer",
              whiteSpace: "nowrap",
            }}
          >
            + 新建{requestedSceneLabel}助手
          </button>
        </div>
      )}

      {groupCount > 0 && role !== "senior_operator" && role !== "admin" && (
        <div
          style={{
            padding: "9px 14px",
            borderRadius: "10px",
            background: "rgba(245,158,11,0.08)",
            border: "1px solid rgba(245,158,11,0.25)",
            color: "#92400E",
            fontSize: "12.5px",
            lineHeight: 1.6,
            flexShrink: 0,
          }}
        >
          {groupForkable || groupAnalyzable ? (
            <>
              💡 教研组或学校已沉淀 <b>{groupCount}</b> 个共享助手。
              {groupForkable && "可直接使用或复制到我的，"}
              {groupAnalyzable && "也可交给AI分析设计思路。"}
            </>
          ) : (
            <>
              💡 教研组或学校已准备 <b>{groupCount}</b>{" "}
              个共享助手，备课时可直接选用。
            </>
          )}
        </div>
      )}

      {banner && (
        <div
          style={{
            padding: "9px 14px",
            borderRadius: "10px",
            background: banner.startsWith("⚠️")
              ? "rgba(239,68,68,0.06)"
              : "rgba(16,185,129,0.08)",
            border: `1px solid ${
              banner.startsWith("⚠️")
                ? "rgba(239,68,68,0.2)"
                : "rgba(16,185,129,0.25)"
            }`,
            color: banner.startsWith("⚠️") ? C.danger : "#047857",
            fontSize: "13px",
            fontWeight: 500,
            flexShrink: 0,
          }}
        >
          {banner}
        </div>
      )}

      {/* ==================== 对话画布 ==================== */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          background: C.card,
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          padding: "14px",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <div
          style={{
            flex: 1,
            minHeight: 0,
          }}
        >
          <AssistantDesignerPanel
            subject={subject}
            grade={grade}
            scenes={WORKSHOP_SCENES}
            initialDraft=""
            onApplyDraft={handleApplyDraft}
            applyButtonLabel="✓ 存为我的助手"
            applyHintText="保存后可在工坊选用，也会出现在现成助手中"
            injectedInput={injected}
            fillHeight
            collapsibleDraft
          />
        </div>
      </div>

      {/* ==================== 侧滑抽屉 ==================== */}
      <div
        onClick={() => setDrawerOpen(false)}
        style={{
          position: "fixed",
          inset: 0,
          background: "rgba(17,24,39,0.35)",
          opacity: drawerOpen ? 1 : 0,
          pointerEvents: drawerOpen ? "auto" : "none",
          transition: "opacity 200ms ease",
          zIndex: 9000,
        }}
      />

      <div
        style={{
          position: "fixed",
          top: 0,
          right: 0,
          bottom: 0,
          width: "380px",
          maxWidth: "90vw",
          background: C.card,
          boxShadow: "-8px 0 32px rgba(0,0,0,0.12)",
          transform: drawerOpen ? "translateX(0)" : "translateX(100%)",
          transition: "transform 260ms cubic-bezier(0.4,0,0.2,1)",
          zIndex: 9001,
          display: "flex",
          flexDirection: "column",
        }}
      >
        <div
          style={{
            padding: "16px 18px",
            borderBottom: `1px solid ${C.border}`,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexShrink: 0,
          }}
        >
          <span
            style={{
              fontSize: "14px",
              fontWeight: 700,
              color: C.text,
            }}
          >
            📚 现成助手
            {subject ? `（${subject}）` : ""}
          </span>

          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "10px",
            }}
          >
            <button
              type="button"
              onClick={() => void loadRelated()}
              title="刷新"
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                fontSize: "13px",
                color: C.primary,
              }}
            >
              🔄
            </button>

            <button
              type="button"
              onClick={() => setDrawerOpen(false)}
              title="关闭"
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                fontSize: "20px",
                color: C.textMuted,
                lineHeight: 1,
              }}
            >
              ×
            </button>
          </div>
        </div>

        <div
          style={{
            padding: "10px 18px",
            fontSize: "11px",
            color: C.textSec,
            lineHeight: 1.6,
            background: C.bg,
            borderBottom: `1px solid ${C.border}`,
            flexShrink: 0,
          }}
        >
          这里展示当前教育域和组织范围内可见的助手。 备课时也可以直接选用。
        </div>

        <div
          style={{
            flex: 1,
            overflow: "auto",
            padding: "12px 14px",
          }}
        >
          {loading && (
            <div
              style={{
                padding: "30px 0",
                textAlign: "center",
                color: C.textMuted,
                fontSize: "12px",
              }}
            >
              加载中…
            </div>
          )}

          {listErr && !loading && (
            <div
              style={{
                padding: "12px",
                borderRadius: "8px",
                textAlign: "center",
                background: "rgba(239,68,68,0.06)",
                border: "1px solid rgba(239,68,68,0.15)",
                color: C.danger,
                fontSize: "12px",
              }}
            >
              ⚠️ {listErr}
              <br />
              <button
                type="button"
                onClick={() => void loadRelated()}
                style={{
                  marginTop: "6px",
                  ...miniBtn(C.danger),
                }}
              >
                重试
              </button>
            </div>
          )}

          {!loading && !listErr && totalCount === 0 && (
            <div
              style={{
                padding: "36px 16px",
                textAlign: "center",
                color: C.textMuted,
                fontSize: "12px",
                lineHeight: 1.7,
              }}
            >
              <div
                style={{
                  fontSize: "30px",
                  marginBottom: "8px",
                }}
              >
                🗂️
              </div>
              {subject ? `“${subject}”暂无现成助手` : "暂无现成助手"}
              <br />
              关闭抽屉后可在对话画布中创建。
            </div>
          )}

          {!loading && !listErr && totalCount > 0 && (
            <>
              {displayOrder.map((source) => {
                const items = grouped[source];

                if (items.length === 0) {
                  return null;
                }

                const isBenchmark = source === "group";

                return (
                  <div
                    key={source}
                    style={{
                      marginBottom: "14px",
                    }}
                  >
                    <div
                      style={{
                        padding: "2px 2px 6px",
                        fontSize: "11px",
                        fontWeight: 700,
                        color: isBenchmark ? C.groupAccent : C.textSec,
                      }}
                    >
                      {ASSISTANT_SOURCE_EMOJI[source]}{" "}
                      {ASSISTANT_SOURCE_LABELS[source]}
                      <span
                        style={{
                          color: C.textMuted,
                          fontWeight: 400,
                        }}
                      >
                        {" "}
                        ({items.length})
                      </span>
                    </div>

                    {items.map((item) => (
                      <DrawerAssistantCard
                        key={item.id}
                        item={item}
                        forking={forkingId === item.id}
                        analyzing={analyzingId === item.id}
                        deleting={deletingId === item.id}
                        highlight={isBenchmark}
                        onAnalyze={() => void handleAnalyze(item)}
                        onFork={() => void handleFork(item)}
                        onEdit={() => handleEdit(item)}
                        onDelete={() => void handleDelete(item)}
                      />
                    ))}
                  </div>
                );
              })}
            </>
          )}
        </div>
      </div>

      {/* ==================== 弹窗 ==================== */}
      <SaveAssistantModal
        open={saveOpen}
        draft={draftToSave}
        userRole={role}
        defaultSubject={subject}
        defaultGrade={grade}
        onClose={() => setSaveOpen(false)}
        onSaved={handleSaved}
      />

      <AssistantEditModal
        open={createContextAssistantOpen}
        mode="create-personal"
        defaultScene={requestedScene}
        defaultSubject={subject}
        defaultGrade={grade}
        onClose={() => setCreateContextAssistantOpen(false)}
        onSaved={() => {
          setCreateContextAssistantOpen(false);
          setBanner(`✅ ${requestedSceneLabel || "AI"}助手已创建`);
          void loadRelated();
        }}
      />

      <AssistantEditModal
        open={editId !== null}
        mode="edit"
        assistantId={editId || undefined}
        onClose={() => setEditId(null)}
        onSaved={handleEditSaved}
      />

      <AssistantStyleProfileModal
        open={styleProfileOpen}
        subject={subject}
        grade={grade}
        currentUserID={user?.id || ""}
        onClose={() => setStyleProfileOpen(false)}
        onUseProfile={handleUseStyleProfile}
      />

      <PastePromptModal
        open={pasteOpen}
        onClose={() => setPasteOpen(false)}
        onSaveDirect={handlePasteSaveDirect}
        onSendToAI={handlePasteSendToAI}
      />
    </div>
  );
}
