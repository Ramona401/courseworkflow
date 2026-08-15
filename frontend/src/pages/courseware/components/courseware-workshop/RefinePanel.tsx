/**
 * RefinePanel.tsx — 页面微调主编排。
 *
 * 本文件只负责：
 *   - 当前页受保护草稿与语音输入；
 *   - AI微调/重构请求编排；
 *   - 已确认整改要求的一次性注入；
 *   - 页面修改完成后的整改状态反馈；
 *   - 截图与参考资源状态。
 *
 * 展示职责：
 *   - RefinePanelControls：微调、重构、截图、参考资源和就地编辑控件；
 *   - RefinePanelVersionHistory：页面历史、回退和版本对比。
 *
 * R-01.1：
 *   - 带整改项的AI页面修改成功后不自动表示问题解决；
 *   - 后端返回applied时，只用短时Toast提示“已完成修改，等待检查”；
 *   - 错误和需要人工重新检查的情况仍保留在当前面板附近；
 *   - 外部整改要求只写入草稿，用户仍需手动执行AI修改。
 */

import { useEffect, useMemo, useRef, useState } from "react";

import { refinePage, regenerateCWPage } from "@/api/coursewares";
import type { CWRefineMode } from "@/api/coursewares";
import VoiceInputButton from "@/components/voice/VoiceInputButton";
import { useProtectedDraft } from "@/hooks/useProtectedDraft";
import { useVoiceDraftInput } from "@/hooks/useVoiceDraftInput";
import CWReviewToast from "@/pages/courseware/review/CWReviewToast";
import { useAuth } from "@/store/auth";

import {
  buildCoursewareContinuityReferenceMarker,
} from "./CoursewareContinuityReferencePicker";
import type {
  CoursewareContinuityReferenceSelection,
} from "./CoursewareContinuityReferencePicker";
import RefinePanelControls from "./RefinePanelControls";
import RefinePanelVersionHistory from "./RefinePanelVersionHistory";
import { buildTemplatePageReferenceMarker } from "./TemplatePageReferencePicker";
import type {
  TemplatePageReferenceSelection,
} from "./TemplatePageReferencePicker";
import type { InjectedSnippet } from "./SnippetInjectPicker";
import { useExternalRefineInstruction } from "./useExternalRefineInstruction";
import type {
  RefinePanelExternalInstruction,
} from "./useExternalRefineInstruction";
import { C } from "./workshopConstants";

interface Props {
  coursewareId: string;
  pageNum: number;
  onPageUpdated: (pageNum: number, html: string) => void;
  externalInstruction?: RefinePanelExternalInstruction | null;
  onExternalInstructionResolved?: (instructionID: string) => void;
}

const LEGACY_REFINE_DRAFT_PREFIX = "tedna_cw_refine_draft_";

function legacyRefineDraftKey(coursewareId: string, pageNum: number): string {
  return `${LEGACY_REFINE_DRAFT_PREFIX}${coursewareId}_${pageNum}`;
}

function readLegacyRefineDraft(coursewareId: string, pageNum: number): string {
  if (!coursewareId || pageNum <= 0) {
    return "";
  }

  try {
    return sessionStorage.getItem(
      legacyRefineDraftKey(coursewareId, pageNum),
    ) || "";
  } catch {
    return "";
  }
}

function removeLegacyRefineDraft(coursewareId: string, pageNum: number): void {
  if (!coursewareId || pageNum <= 0) {
    return;
  }

  try {
    sessionStorage.removeItem(
      legacyRefineDraftKey(coursewareId, pageNum),
    );
  } catch {
    // 旧键删除失败不影响微调主流程。
  }
}

export default function RefinePanel({
  coursewareId,
  pageNum,
  onPageUpdated,
  externalInstruction,
  onExternalInstructionResolved,
}: Props) {
  const { user } = useAuth();

  const legacyInitialDraft = useMemo(
    () => readLegacyRefineDraft(coursewareId, pageNum),
    [coursewareId, pageNum],
  );

  const refineDraft = useProtectedDraft({
    userId: user?.id,
    scope: "courseware-page-refine",
    resourceId: [coursewareId, `page-${pageNum}`].join("|"),
    field: "instruction",
    initialValue: legacyInitialDraft,
    maxHistory: 40,
  });

  const refineInput = refineDraft.value;
  const updateRefineInput = refineDraft.setValue;

  const [refineRunning, setRefineRunning] = useState(false);
  const [refineMode, setRefineMode] = useState<CWRefineMode>("preserve");
  const [rebuildWorkflow, setRebuildWorkflow] =
    useState<"discussion" | "direct">("discussion");
  const [discussionActive, setDiscussionActive] = useState(false);
  const [refineImage, setRefineImage] = useState("");
  const [regenRunning, setRegenRunning] = useState(false);
  const [message, setMessage] = useState("");
  const [reviewToastMessage, setReviewToastMessage] = useState("");
  const [versionRefreshKey, setVersionRefreshKey] = useState(0);
  const [injectedSnippet, setInjectedSnippet] =
    useState<InjectedSnippet | null>(null);
  const [templatePageReference, setTemplatePageReference] =
    useState<TemplatePageReferenceSelection | null>(null);
  const [continuityPageReferences, setContinuityPageReferences] =
    useState<CoursewareContinuityReferenceSelection | null>(null);

  const refineInputRef = useRef<HTMLTextAreaElement>(null);

  const directRefineInputVisible =
    refineMode !== "rebuild" || rebuildWorkflow === "direct";

  const refineVoice = useVoiceDraftInput({
    value: refineInput,
    setValue: updateRefineInput,
    disabled:
      !directRefineInputVisible ||
      refineRunning ||
      regenRunning ||
      discussionActive ||
      pageNum <= 0,
    maxDurationSeconds: 120,
    onFinalFocus: (finalValue) => {
      const element = refineInputRef.current;

      if (!element) {
        return;
      }

      element.focus();
      element.setSelectionRange(finalValue.length, finalValue.length);
    },
    onError: (voiceError) => {
      setMessage(`❌ 语音输入失败: ${voiceError}`);
    },
  });

  useExternalRefineInstruction({
    externalInstruction,
    coursewareId,
    pageNum,
    updateRefineInput,
    setRefineMode,
    setMessage,
  });

  const activeExternalInstruction =
    externalInstruction?.coursewareId === coursewareId &&
    externalInstruction.targetPageNumber === pageNum
      ? externalInstruction
      : null;

  const activeReviewItemID =
    activeExternalInstruction?.reviewItemId.trim() || "";

  const activeInstructionSignalID =
    activeExternalInstruction?.id.trim() || "";

  useEffect(() => {
    refineVoice.cancel();
    setRefineMode("preserve");
    setRebuildWorkflow("discussion");
    setDiscussionActive(false);
    setRefineImage("");
    setInjectedSnippet(null);
    setTemplatePageReference(null);
    setContinuityPageReferences(null);
    setMessage("");
    setReviewToastMessage("");

    // 切页身份才是本Effect的边界。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [coursewareId, pageNum]);

  const buildRebuildDiscussionReferenceContext = (): string => {
    let referenceContext = "";

    if (injectedSnippet) {
      const maxReferenceLength = 24000;
      const referenceHTML =
        injectedSnippet.html.length > maxReferenceLength
          ? `${injectedSnippet.html.slice(
              0,
              maxReferenceLength,
            )}\n<!-- 参考代码过长，已截断 -->`
          : injectedSnippet.html;

      referenceContext +=
        `【参考代码范本·开始】（收藏名：${injectedSnippet.title}）\n` +
        "参照以下范本的布局骨架、交互方式与视觉手法；" +
        "教学内容仍以当前页为准，不得照抄范本文字。\n" +
        `\`\`\`html\n${referenceHTML}\n\`\`\`\n` +
        "【参考代码范本·结束】";
    }

    if (continuityPageReferences) {
      referenceContext +=
        (referenceContext ? "\n\n" : "") +
        buildCoursewareContinuityReferenceMarker(
          continuityPageReferences,
        );
    }

    if (templatePageReference) {
      referenceContext +=
        (referenceContext ? "\n\n" : "") +
        buildTemplatePageReferenceMarker(templatePageReference);
    }

    return referenceContext.trim();
  };

  const buildFinalInstruction = (): string => {
    let finalInstruction = refineInput.trim();

    if (injectedSnippet) {
      const maxReferenceLength = 24000;
      const referenceHTML =
        injectedSnippet.html.length > maxReferenceLength
          ? `${injectedSnippet.html.slice(
              0,
              maxReferenceLength,
            )}\n<!-- 参考代码过长，已截断 -->`
          : injectedSnippet.html;

      finalInstruction +=
        `\n\n【参考代码范本·开始】（收藏名：${injectedSnippet.title}）\n` +
        "以下是我指定的参考代码。请在落实上面修改意见时，" +
        "参照这段范本的布局骨架、交互方式与视觉手法；" +
        "但教学内容（文字/数据/图片）仍以当前页为准，" +
        "不要照抄范本里的文字内容。\n" +
        `\`\`\`html\n${referenceHTML}\n\`\`\`\n` +
        "【参考代码范本·结束】";
    }

    if (continuityPageReferences) {
      finalInstruction +=
        "\n\n" +
        buildCoursewareContinuityReferenceMarker(
          continuityPageReferences,
        );
    }

    if (templatePageReference) {
      finalInstruction +=
        "\n\n" +
        buildTemplatePageReferenceMarker(templatePageReference);
    }

    return finalInstruction;
  };

  const clearConsumedReferences = (): void => {
    setRefineImage("");
    setInjectedSnippet(null);
    setTemplatePageReference(null);
    setContinuityPageReferences(null);
  };

  const handleRefinePage = async (): Promise<void> => {
    if (
      !coursewareId ||
      pageNum <= 0 ||
      !refineInput.trim() ||
      refineRunning ||
      regenRunning ||
      refineVoice.isActive
    ) {
      return;
    }

    setRefineRunning(true);
    setReviewToastMessage("");

    try {
      const result = await refinePage(
        coursewareId,
        pageNum,
        buildFinalInstruction(),
        refineImage || undefined,
        refineMode,
        activeReviewItemID || undefined,
      );

      if (result.html_content) {
        onPageUpdated(pageNum, result.html_content);
      }

      refineDraft.commit();
      removeLegacyRefineDraft(coursewareId, pageNum);
      clearConsumedReferences();

      if (activeReviewItemID) {
        if (result.review_item_warning) {
          setMessage(
            "⚠️ 页面已完成修改，但这条问题需要重新检查：" +
              result.review_item_warning,
          );
        } else if (result.review_item_status === "applied") {
          setMessage("");
          setReviewToastMessage("已完成修改，等待检查");
        } else {
          setMessage(
            "⚠️ 页面已完成修改，请刷新整改中心确认这条问题的当前处理情况。",
          );
        }
      } else {
        setMessage(`✅ ${result.message}`);
      }

      if (activeReviewItemID && activeInstructionSignalID) {
        onExternalInstructionResolved?.(activeInstructionSignalID);
      }

      setVersionRefreshKey((previous) => previous + 1);
    } catch (cause) {
      const actionName = refineMode === "rebuild" ? "全页重构" : "微调";

      setMessage(
        `❌ ${actionName}失败: ${
          cause instanceof Error ? cause.message : "未知错误"
        }`,
      );
    } finally {
      setRefineRunning(false);
    }
  };

  const handleRegeneratePage = async (): Promise<void> => {
    if (
      !coursewareId ||
      pageNum <= 0 ||
      regenRunning ||
      refineRunning ||
      refineVoice.isActive
    ) {
      return;
    }

    if (
      !window.confirm(
        `⚠️ 重生第 ${pageNum} 页将按方案从零重画整页，` +
          "会清空本页已插入的图片（图片资产仍在多媒体库，可重新插入）。" +
          "确定重生？",
      )
    ) {
      return;
    }

    setRegenRunning(true);
    setMessage(`🔄 正在重生第 ${pageNum} 页，请稍候...`);

    try {
      const result = await regenerateCWPage(coursewareId, pageNum);

      if (result.html_content) {
        onPageUpdated(pageNum, result.html_content);
      }

      setMessage(`✅ ${result.message}`);
      setVersionRefreshKey((previous) => previous + 1);
    } catch (cause) {
      setMessage(
        `❌ 重生失败: ${
          cause instanceof Error ? cause.message : "未知错误"
        }`,
      );
    } finally {
      setRegenRunning(false);
    }
  };

  const handleInlineEditorSaved = (
    updatedPageNumber: number,
    html: string,
  ): void => {
    onPageUpdated(updatedPageNumber, html);
    setMessage(`✅ 第 ${updatedPageNumber} 页文字修改已保存`);
    setVersionRefreshKey((previous) => previous + 1);
  };

  const handleDiscussionPageUpdated = (
    updatedPageNumber: number,
    updatedHTML: string,
  ): void => {
    onPageUpdated(updatedPageNumber, updatedHTML);
    clearConsumedReferences();
    setMessage(`✅ 第${updatedPageNumber}页已按确认方案完成全页重构`);
    setVersionRefreshKey((previous) => previous + 1);
  };

  const loadRefineImageFile = (
    file: File,
    fromPaste = false,
  ): void => {
    if (file.size > 8 * 1024 * 1024) {
      setMessage("❌ 截图不能超过8MB");
      return;
    }

    const reader = new FileReader();

    reader.onload = () => {
      setRefineImage(
        typeof reader.result === "string" ? reader.result : "",
      );

      if (fromPaste) {
        setMessage("✅ 已从剪贴板粘贴截图，微调将参考该图");
      }
    };

    reader.onerror = () => {
      setMessage("❌ 截图读取失败");
    };

    reader.readAsDataURL(file);
  };

  const handleRefinePaste = (
    event: React.ClipboardEvent<HTMLTextAreaElement>,
  ): void => {
    const items = event.clipboardData?.items;

    if (!items) {
      return;
    }

    for (let index = 0; index < items.length; index += 1) {
      const item = items[index];

      if (!item.type || !item.type.startsWith("image/")) {
        continue;
      }

      const file = item.getAsFile();

      if (!file) {
        continue;
      }

      event.preventDefault();
      loadRefineImageFile(file, true);
      return;
    }
  };

  const handleRefineKeyDown = (
    event: React.KeyboardEvent<HTMLTextAreaElement>,
  ): void => {
    if (refineDraft.handleKeyDown(event)) {
      return;
    }

    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();

      if (
        !refineRunning &&
        !regenRunning &&
        !refineVoice.isActive &&
        pageNum > 0 &&
        refineInput.trim()
      ) {
        void handleRefinePage();
      }
    }
  };

  const handleRefineModeChange = (nextMode: CWRefineMode): void => {
    setRefineMode(nextMode);

    if (nextMode === "preserve") {
      setTemplatePageReference(null);
      setContinuityPageReferences(null);
    } else {
      setRebuildWorkflow("discussion");
    }
  };

  const voiceControl = (
    <VoiceInputButton
      status={refineVoice.status}
      isSupported={refineVoice.isSupported}
      elapsedSeconds={refineVoice.elapsedSeconds}
      disabled={
        refineRunning ||
        regenRunning ||
        discussionActive ||
        pageNum <= 0
      }
      error={refineVoice.error}
      onStart={refineVoice.begin}
      onStop={refineVoice.stop}
      onCancel={refineVoice.cancel}
    />
  );

  return (
    <div
      style={{
        marginTop: 16,
        padding: "16px",
        borderRadius: 10,
        border: `1px solid ${C.border}`,
        background: "#FAFAFA",
      }}
    >
      <div
        style={{
          fontSize: 13,
          fontWeight: 600,
          color: C.textPrimary,
          marginBottom: 10,
        }}
      >
        {refineMode === "rebuild"
          ? "🧱 全页重构：允许重新设计本页内容区，导航栏与模板风格保持不变"
          : "🎨 保留结构微调：只修改指定部分，保留当前布局、ID、函数和交互"}
      </div>

      <RefinePanelControls
        coursewareId={coursewareId}
        pageNum={pageNum}
        refineMode={refineMode}
        rebuildWorkflow={rebuildWorkflow}
        discussionActive={discussionActive}
        refineInput={refineInput}
        refineInputRef={refineInputRef}
        refineRunning={refineRunning}
        regenRunning={regenRunning}
        voiceActive={refineVoice.isActive}
        voiceStatusText={refineVoice.statusText}
        voiceControl={voiceControl}
        refineImage={refineImage}
        injectedSnippet={injectedSnippet}
        templatePageReference={templatePageReference}
        continuityPageReferences={continuityPageReferences}
        rebuildReferenceContext={buildRebuildDiscussionReferenceContext()}
        historyControl={
          <RefinePanelVersionHistory
            coursewareId={coursewareId}
            pageNum={pageNum}
            refreshKey={versionRefreshKey}
            discussionActive={discussionActive}
            voiceActive={refineVoice.isActive}
            onPageUpdated={onPageUpdated}
            onMessage={setMessage}
          />
        }
        onRefineModeChange={handleRefineModeChange}
        onRebuildWorkflowChange={setRebuildWorkflow}
        onDiscussionActiveChange={setDiscussionActive}
        onRefineInputChange={updateRefineInput}
        onRefineKeyDown={handleRefineKeyDown}
        onRefinePaste={handleRefinePaste}
        onRefine={() => void handleRefinePage()}
        onClearRefineImage={() => setRefineImage("")}
        onImageFile={loadRefineImageFile}
        onSnippetInject={(snippet) => {
          setInjectedSnippet(snippet);
          setTemplatePageReference(null);
        }}
        onSnippetRemove={() => setInjectedSnippet(null)}
        onTemplateSelect={(selection) => {
          setTemplatePageReference(selection);
          setInjectedSnippet(null);
        }}
        onTemplateRemove={() => setTemplatePageReference(null)}
        onContinuitySelect={setContinuityPageReferences}
        onContinuityRemove={() => setContinuityPageReferences(null)}
        onInlineEditorSaved={handleInlineEditorSaved}
        onRegenerate={() => void handleRegeneratePage()}
        onDiscussionPageUpdated={handleDiscussionPageUpdated}
        onDiscussionMessageSent={() => setRefineImage("")}
      />

      <div
        style={{
          marginTop: 6,
          fontSize: 11,
          color: "#9CA3AF",
        }}
      >
        💡 保留结构微调＝只改指定局部并保留当前布局、ID和交互；
        全页重构＝根据你的要求重新设计本页内容区，但保留导航栏与模板风格；
        按方案重生＝忽略当前页面，依据原页面方案从零生成。
        三种AI修改前系统都会自动保存当前版本，可在「📜 历史版本」中对比或回退。
        就地编辑仍适合改错字、字号、颜色和替换图片，不调用AI。
      </div>

      {message && (
        <div
          role="status"
          style={{
            marginTop: 12,
            padding: "10px 14px",
            borderRadius: 8,
            fontSize: 13,
            background: message.startsWith("❌")
              ? "#FEE2E2"
              : message.startsWith("✅")
                ? "#D1FAE5"
                : message.startsWith("⚠️")
                  ? "#FFF7ED"
                  : "#EFF6FF",
            color: message.startsWith("❌")
              ? "#DC2626"
              : message.startsWith("✅")
                ? "#059669"
                : message.startsWith("⚠️")
                  ? "#9A3412"
                  : "#2563EB",
          }}
        >
          {message}
        </div>
      )}

      <CWReviewToast
        message={reviewToastMessage}
        onClose={() => setReviewToastMessage("")}
      />
    </div>
  );
}
