/**
 * RefinePanelControls.tsx
 *
 * 页面微调面板的纯交互展示层。
 *
 * 负责：
 *   - 保留结构/全页重构模式切换；
 *   - 直接微调输入与执行按钮；
 *   - 先讨论后重构入口；
 *   - 截图、参考代码、模板页和前页连续性参考；
 *   - 就地文字编辑、页面历史和按方案重生入口。
 *
 * 本组件不调用页面AI接口，不决定整改项状态。
 */

import { useState } from "react";
import type {
  ClipboardEvent as ReactClipboardEvent,
  KeyboardEvent as ReactKeyboardEvent,
  ReactNode,
  RefObject,
} from "react";

import type { CWRefineMode } from "@/api/coursewares";

import CoursewareContinuityReferencePicker from "./CoursewareContinuityReferencePicker";
import type {
  CoursewareContinuityReferenceSelection,
} from "./CoursewareContinuityReferencePicker";
import InlineTextEditor from "./InlineTextEditor";
import RebuildDiscussionPanel from "./RebuildDiscussionPanel";
import SnippetInjectPicker from "./SnippetInjectPicker";
import type { InjectedSnippet } from "./SnippetInjectPicker";
import TemplatePageReferencePicker from "./TemplatePageReferencePicker";
import type {
  TemplatePageReferenceSelection,
} from "./TemplatePageReferencePicker";
import { C } from "./workshopConstants";

export interface RefinePanelControlsProps {
  coursewareId: string;
  pageNum: number;
  refineMode: CWRefineMode;
  rebuildWorkflow: "discussion" | "direct";
  discussionActive: boolean;
  refineInput: string;
  refineInputRef: RefObject<HTMLTextAreaElement | null>;
  refineRunning: boolean;
  regenRunning: boolean;
  voiceActive: boolean;
  voiceStatusText: string;
  voiceControl: ReactNode;
  refineImage: string;
  injectedSnippet: InjectedSnippet | null;
  templatePageReference: TemplatePageReferenceSelection | null;
  continuityPageReferences: CoursewareContinuityReferenceSelection | null;
  rebuildReferenceContext: string;
  historyControl: ReactNode;

  onRefineModeChange: (mode: CWRefineMode) => void;
  onRebuildWorkflowChange: (workflow: "discussion" | "direct") => void;
  onDiscussionActiveChange: (active: boolean) => void;
  onRefineInputChange: (value: string) => void;
  onRefineKeyDown: (
    event: ReactKeyboardEvent<HTMLTextAreaElement>,
  ) => void;
  onRefinePaste: (
    event: ReactClipboardEvent<HTMLTextAreaElement>,
  ) => void;
  onRefine: () => void;
  onClearRefineImage: () => void;
  onImageFile: (file: File, fromPaste?: boolean) => void;
  onSnippetInject: (snippet: InjectedSnippet) => void;
  onSnippetRemove: () => void;
  onTemplateSelect: (
    selection: TemplatePageReferenceSelection,
  ) => void;
  onTemplateRemove: () => void;
  onContinuitySelect: (
    selection: CoursewareContinuityReferenceSelection,
  ) => void;
  onContinuityRemove: () => void;
  onInlineEditorSaved: (pageNumber: number, html: string) => void;
  onRegenerate: () => void;
  onDiscussionPageUpdated: (pageNumber: number, html: string) => void;
  onDiscussionMessageSent: () => void;
}

export default function RefinePanelControls({
  coursewareId,
  pageNum,
  refineMode,
  rebuildWorkflow,
  discussionActive,
  refineInput,
  refineInputRef,
  refineRunning,
  regenRunning,
  voiceActive,
  voiceStatusText,
  voiceControl,
  refineImage,
  injectedSnippet,
  templatePageReference,
  continuityPageReferences,
  rebuildReferenceContext,
  historyControl,
  onRefineModeChange,
  onRebuildWorkflowChange,
  onDiscussionActiveChange,
  onRefineInputChange,
  onRefineKeyDown,
  onRefinePaste,
  onRefine,
  onClearRefineImage,
  onImageFile,
  onSnippetInject,
  onSnippetRemove,
  onTemplateSelect,
  onTemplateRemove,
  onContinuitySelect,
  onContinuityRemove,
  onInlineEditorSaved,
  onRegenerate,
  onDiscussionPageUpdated,
  onDiscussionMessageSent,
}: RefinePanelControlsProps) {
  const [showInlineEditor, setShowInlineEditor] = useState(false);

  const directInputVisible =
    refineMode !== "rebuild" || rebuildWorkflow === "direct";

  const disabled = refineRunning || regenRunning;

  const openImagePicker = (): void => {
    const input = document.createElement("input");

    input.type = "file";
    input.accept = "image/*";

    input.onchange = (event) => {
      const file =
        (event.target as HTMLInputElement).files?.[0];

      if (file) {
        onImageFile(file);
      }
    };

    input.click();
  };

  return (
    <>
      <div style={modeRowStyle}>
        <button
          type="button"
          onClick={() => onRefineModeChange("preserve")}
          disabled={disabled || discussionActive || voiceActive}
          style={modeButtonStyle(
            refineMode === "preserve",
            "#7C3AED",
            disabled || discussionActive,
          )}
        >
          🛠 保留结构微调
        </button>

        <button
          type="button"
          onClick={() => onRefineModeChange("rebuild")}
          disabled={disabled || discussionActive || voiceActive}
          style={modeButtonStyle(
            refineMode === "rebuild",
            "#EA580C",
            disabled || discussionActive,
          )}
        >
          🧱 全页重构
        </button>

        <span style={modeHintStyle}>
          {refineMode === "rebuild"
            ? "允许重建内容区布局、ID、函数与交互；当前版本会自动保存"
            : "适合改文字、颜色、大小、位置或删除明确指定的局部元素"}
        </span>
      </div>

      {refineMode === "rebuild" && (
        <div style={rebuildWorkflowStyle}>
          <button
            type="button"
            onClick={() => onRebuildWorkflowChange("discussion")}
            disabled={disabled || discussionActive || voiceActive}
            style={workflowButtonStyle(
              rebuildWorkflow === "discussion",
            )}
          >
            💬 先讨论后重构
          </button>

          <button
            type="button"
            onClick={() => onRebuildWorkflowChange("direct")}
            disabled={disabled || discussionActive || voiceActive}
            style={workflowButtonStyle(
              rebuildWorkflow === "direct",
            )}
          >
            ⚡ 直接重构
          </button>

          <span style={rebuildHintStyle}>
            {discussionActive
              ? "当前讨论已建立，请先完成或取消讨论"
              : rebuildWorkflow === "discussion"
                ? "AI先澄清并形成执行方案，老师确认后才生成代码"
                : "提交后立即生成并写回页面"}
          </span>
        </div>
      )}

      {directInputVisible ? (
        <>
          <div style={directInputRowStyle}>
            <span style={currentPageBadgeStyle}>
              当前：第 {pageNum || "—"} 页
            </span>

            <textarea
              ref={refineInputRef}
              value={refineInput}
              onChange={(event) => onRefineInputChange(event.target.value)}
              placeholder={
                refineMode === "rebuild"
                  ? "例如：延续所选前页的人物、卡片体系和逐步点击逻辑，把本页开发为下一阶段任务..."
                  : "例如：标题字号再大一些、删除右上角人物、调整卡片位置...（回车提交，Shift+回车换行）"
              }
              onKeyDown={onRefineKeyDown}
              onPaste={onRefinePaste}
              rows={2}
              style={refineTextareaStyle}
              disabled={disabled || voiceActive}
            />

            {voiceControl}

            <button
              type="button"
              onClick={onRefine}
              disabled={
                disabled ||
                voiceActive ||
                pageNum <= 0 ||
                !refineInput.trim()
              }
              style={refineButtonStyle(
                refineMode,
                pageNum > 0 &&
                  !!refineInput.trim() &&
                  !disabled &&
                  !voiceActive,
              )}
            >
              {refineRunning
                ? refineMode === "rebuild"
                  ? "⏳ 重构中..."
                  : "⏳ 微调中..."
                : refineMode === "rebuild"
                  ? "🧱 重构本页"
                  : "🎨 AI微调"}
            </button>
          </div>

          <div style={draftHintStyle}>
            {voiceStatusText || (
              <>
                当前课件和页码的输入已自动保存 · 点击麦克风可语音输入 ·
                讨论或微调失败不会清除 · Ctrl/Command+Z恢复误删 ·
                截图不会进入文字草稿
              </>
            )}
          </div>
        </>
      ) : (
        <RebuildDiscussionPanel
          coursewareId={coursewareId}
          pageNum={pageNum}
          content={refineInput}
          onContentChange={onRefineInputChange}
          referenceContext={rebuildReferenceContext}
          image={refineImage || undefined}
          disabled={disabled}
          onPageUpdated={onDiscussionPageUpdated}
          onMessageSent={onDiscussionMessageSent}
          onActiveChange={onDiscussionActiveChange}
        />
      )}

      <div style={toolbarStyle}>
        {refineImage ? (
          <div style={imageSummaryStyle}>
            <img
              src={refineImage}
              alt="参考截图"
              style={imagePreviewStyle}
            />

            <span style={imageLabelStyle}>
              已附截图（AI修改将参考）
            </span>

            <button
              type="button"
              onClick={onClearRefineImage}
              disabled={disabled || voiceActive}
              style={removeImageButtonStyle}
            >
              移除
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={openImagePicker}
            disabled={disabled || voiceActive}
            style={imagePickerButtonStyle}
          >
            📷 附参考截图（或在输入框 Ctrl+V 粘贴）
          </button>
        )}

        <div style={{ flex: 1 }} />

        {refineMode === "rebuild" && (
          <CoursewareContinuityReferencePicker
            coursewareId={coursewareId}
            currentPageNumber={pageNum}
            selected={continuityPageReferences}
            onSelect={onContinuitySelect}
            onRemove={onContinuityRemove}
            disabled={disabled || discussionActive || pageNum <= 1}
          />
        )}

        {refineMode === "rebuild" && (
          <TemplatePageReferencePicker
            selected={templatePageReference}
            onSelect={onTemplateSelect}
            onRemove={onTemplateRemove}
            disabled={disabled || discussionActive || pageNum <= 0}
          />
        )}

        <SnippetInjectPicker
          injected={injectedSnippet}
          onInject={onSnippetInject}
          onRemove={onSnippetRemove}
          disabled={disabled || discussionActive || pageNum <= 0}
        />

        <button
          type="button"
          onClick={() => setShowInlineEditor(true)}
          disabled={
            pageNum <= 0 ||
            disabled ||
            discussionActive ||
            voiceActive
          }
          title={
            pageNum <= 0
              ? "请先在上方预览区选中页"
              : "点选文字改内容、字号、颜色、加粗和字体，点选图片可替换"
          }
          style={toolbarButtonStyle(
            "#0EA5E9",
            "#0284C7",
            pageNum > 0 && !disabled,
          )}
        >
          ✏️ 就地编辑
        </button>

        {historyControl}

        <button
          type="button"
          onClick={onRegenerate}
          disabled={pageNum <= 0 || disabled || discussionActive}
          title={
            pageNum <= 0
              ? "请先在上方预览区选中页"
              : "忽略当前页面内容，按原页面方案从零生成"
          }
          style={regenerateButtonStyle(pageNum > 0 && !disabled)}
        >
          {regenRunning ? "⏳ 重生中..." : "🔄 按方案重生"}
        </button>
      </div>

      {showInlineEditor && (
        <InlineTextEditor
          coursewareId={coursewareId}
          pageNum={pageNum}
          onPageUpdated={onInlineEditorSaved}
          onClose={() => setShowInlineEditor(false)}
        />
      )}
    </>
  );
}

function modeButtonStyle(
  active: boolean,
  accent: string,
  disabled: boolean,
) {
  return {
    padding: "8px 15px",
    borderRadius: 7,
    border: active
      ? `1px solid ${accent}`
      : "1px solid transparent",
    background: active ? "#FFFFFF" : "transparent",
    color: active ? accent : "#6B7280",
    boxShadow: active ? `0 1px 4px ${accent}2E` : "none",
    fontSize: 12,
    fontWeight: 700,
    cursor: disabled ? "default" : "pointer",
  } as const;
}

function workflowButtonStyle(active: boolean) {
  return {
    padding: "8px 14px",
    borderRadius: 7,
    border: active
      ? "1px solid #EA580C"
      : "1px solid transparent",
    background: active ? "#FFFFFF" : "transparent",
    color: active ? "#C2410C" : "#6B7280",
    fontSize: 12,
    fontWeight: 700,
    cursor: "pointer",
  } as const;
}

function refineButtonStyle(mode: CWRefineMode, active: boolean) {
  return {
    padding: "10px 20px",
    borderRadius: 8,
    border: "none",
    background: active
      ? mode === "rebuild"
        ? "#EA580C"
        : "#7C3AED"
      : "#E5E7EB",
    color: active ? "#FFFFFF" : "#9CA3AF",
    fontSize: 14,
    fontWeight: 600,
    cursor: active ? "pointer" : "default",
    whiteSpace: "nowrap",
    marginTop: 2,
  } as const;
}

function toolbarButtonStyle(
  border: string,
  color: string,
  active: boolean,
) {
  return {
    padding: "8px 16px",
    borderRadius: 8,
    border: `1px solid ${active ? border : C.border}`,
    background: active ? "#F0F9FF" : "#FFFFFF",
    color: active ? color : "#9CA3AF",
    fontSize: 13,
    fontWeight: 600,
    cursor: active ? "pointer" : "default",
    whiteSpace: "nowrap",
  } as const;
}

function regenerateButtonStyle(active: boolean) {
  return {
    padding: "8px 18px",
    borderRadius: 8,
    border: "none",
    background: active
      ? "linear-gradient(135deg, #F59E0B, #EF4444)"
      : "#E5E7EB",
    color: active ? "#FFFFFF" : "#9CA3AF",
    fontSize: 13,
    fontWeight: 600,
    cursor: active ? "pointer" : "default",
    whiteSpace: "nowrap",
  } as const;
}

const modeRowStyle = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  marginBottom: 12,
  padding: 4,
  borderRadius: 9,
  background: "#F3F4F6",
  width: "fit-content",
  maxWidth: "100%",
  flexWrap: "wrap",
} as const;

const modeHintStyle = {
  padding: "0 8px",
  fontSize: 11,
  color: "#9CA3AF",
  lineHeight: 1.5,
} as const;

const rebuildWorkflowStyle = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  marginBottom: 12,
  padding: 4,
  borderRadius: 9,
  background: "#FFF7ED",
  width: "fit-content",
  maxWidth: "100%",
  flexWrap: "wrap",
} as const;

const rebuildHintStyle = {
  padding: "0 8px",
  fontSize: 11,
  color: "#9A3412",
  lineHeight: 1.5,
} as const;

const directInputRowStyle = {
  display: "flex",
  gap: 10,
  flexWrap: "wrap",
  alignItems: "flex-start",
} as const;

const currentPageBadgeStyle = {
  padding: "8px 12px",
  borderRadius: 8,
  background: C.primaryBg,
  color: C.primary,
  fontSize: 13,
  fontWeight: 600,
  whiteSpace: "nowrap",
  marginTop: 2,
} as const;

const refineTextareaStyle = {
  flex: 1,
  padding: "10px 14px",
  borderRadius: 8,
  border: `1px solid ${C.border}`,
  fontSize: 14,
  outline: "none",
  minWidth: 200,
  resize: "vertical",
  fontFamily: "inherit",
  lineHeight: 1.6,
  boxSizing: "border-box",
} as const;

const draftHintStyle = {
  marginTop: 5,
  fontSize: 10,
  color: "#9CA3AF",
  lineHeight: 1.5,
} as const;

const toolbarStyle = {
  display: "flex",
  gap: 10,
  marginTop: 10,
  flexWrap: "wrap",
  alignItems: "center",
} as const;

const imageSummaryStyle = {
  display: "flex",
  alignItems: "center",
  gap: 6,
} as const;

const imagePreviewStyle = {
  width: 40,
  height: 40,
  objectFit: "cover",
  borderRadius: 6,
  border: "2px solid #7C3AED",
} as const;

const imageLabelStyle = {
  fontSize: 11,
  color: "#7C3AED",
} as const;

const removeImageButtonStyle = {
  padding: "2px 8px",
  borderRadius: 4,
  border: "1px solid #EF4444",
  background: "transparent",
  color: "#EF4444",
  fontSize: 11,
  cursor: "pointer",
} as const;

const imagePickerButtonStyle = {
  padding: "8px 14px",
  borderRadius: 8,
  border: "1px dashed #7C3AED",
  background: "rgba(124,58,237,0.04)",
  color: "#7C3AED",
  fontSize: 13,
  cursor: "pointer",
} as const;
