/**
 * 页面教学智能体教师工作台。
 *
 * 三个主Tab：
 *   1. 学生怎么学；
 *   2. 互动内容；
 *   3. 试用与发布。
 *
 * 创建、保存、发布和页面更新统一放入固定浮动操作台。
 * 发布管理器始终挂载，仅按Tab控制完整表单显隐。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  useAuth,
} from "@/store/auth";

import type {
  CoursewareAssistantSelectedPage,
} from "./coursewareAssistantSelection";

import CoursewareAssistantTeachingModeSelector from "./CoursewareAssistantTeachingModeSelector";
import CoursewareAssistantCoreEditor from "./CoursewareAssistantCoreEditor";
import CoursewareAssistantPolicyContextEditor from "./CoursewareAssistantPolicyContextEditor";
import CoursewareAssistantQuestionChainEditor from "./CoursewareAssistantQuestionChainEditor";
import CoursewareAssistantBranchEditor from "./CoursewareAssistantBranchEditor";
import CoursewareAssistantContextReceipt from "./CoursewareAssistantContextReceipt";
import CoursewareAssistantActionDock from "./CoursewareAssistantActionDock";
import CoursewareAssistantDeploymentManager from "./CoursewareAssistantDeploymentManager";
import CoursewareAssistantPreview from "./CoursewareAssistantPreview";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  coursewareAssistantRuneLength,
} from "./coursewareAssistantDraft";

import type {
  CoursewareAssistantDeploymentController,
  CoursewareAssistantDeploymentDockState,
} from "./coursewareAssistantDeploymentDock";

import {
  EMPTY_COURSEWARE_ASSISTANT_DEPLOYMENT_DOCK_STATE,
} from "./coursewareAssistantDeploymentDock";

import {
  COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_EVENT,
  COURSEWARE_ASSISTANT_VALIDITY_SECTION_ID,
  consumeCoursewareAssistantValiditySettingsRequest,
} from "./coursewareAssistantValidityNavigation";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
} from "./CoursewareAssistantEditorShared";

import {
  CoursewareAssistantAdvancedToolbar,
  CoursewareAssistantCurrentPageCard,
  CoursewareAssistantMessageBox,
  CoursewareAssistantNextStepCard,
  CoursewareAssistantPanelHeader,
  CoursewareAssistantPlanSummary,
  CoursewareAssistantTabBar,
  CoursewareAssistantValidationBox,
  type CoursewareAssistantTabOption,
} from "./CoursewareAssistantPanelSections";

import {
  useCoursewareAssistantEditor,
} from "./useCoursewareAssistantEditor";

interface CoursewareAssistantPanelProps {
  coursewareId: string;
  selectedPage: CoursewareAssistantSelectedPage | null;
}

type MainTab =
  | "learning"
  | "content"
  | "delivery";

type ContentTab =
  | "basics"
  | "steps"
  | "support"
  | "context";

const DELIVERY_SECTION_ID =
  "courseware-assistant-delivery-section";

const MAIN_TABS:
  readonly CoursewareAssistantTabOption<MainTab>[] = [
    {
      value: "learning",
      label: "1. 学生怎么学",
      description: "选方式并创建",
    },
    {
      value: "content",
      label: "2. 互动内容",
      description: "查看和调整方案",
    },
    {
      value: "delivery",
      label: "3. 试用与发布",
      description: "设置课堂使用方式",
    },
  ];

const CONTENT_TABS:
  readonly CoursewareAssistantTabOption<ContentTab>[] = [
    {
      value: "basics",
      label: "名称与开场",
      description: "学生最先看到什么",
    },
    {
      value: "steps",
      label: "学生互动",
      description: "实际提问和学习动作",
    },
    {
      value: "support",
      label: "提示与困难",
      description: "答不出来时怎样帮助",
    },
    {
      value: "context",
      label: "参考内容",
      description: "AI可以参考哪些材料",
    },
  ];

export default function CoursewareAssistantPanel({
  coursewareId,
  selectedPage,
}: CoursewareAssistantPanelProps) {
  const { user } = useAuth();

  const editor = useCoursewareAssistantEditor({
    coursewareId,
    selectedPage,
    userId: user?.id,
  });

  const deploymentControllerRef =
    useRef<CoursewareAssistantDeploymentController | null>(
      null,
    );

  const [deploymentState, setDeploymentState] =
    useState<CoursewareAssistantDeploymentDockState>(
      EMPTY_COURSEWARE_ASSISTANT_DEPLOYMENT_DOCK_STATE,
    );

  const [activeMainTab, setActiveMainTab] =
    useState<MainTab>("learning");

  const [activeContentTab, setActiveContentTab] =
    useState<ContentTab>("basics");

  const [validityFocusSequence, setValidityFocusSequence] =
    useState(0);

  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const editorBusy =
    editor.loading ||
    editor.generating ||
    editor.saving ||
    editor.deleting;

  const hasGeneratedContent = Boolean(
    editor.draft.title.trim() &&
    editor.draft.welcomeMessage.trim() &&
    editor.draft.guidancePlan.question_chain.some(
      (step) => step.prompt.trim(),
    ),
  );

  const instructionLength = coursewareAssistantRuneLength(
    editor.draft.teacherInstruction,
  );

  const generateBlocker = editor.loadError
    ? "当前页面教学智能体读取失败，请先点击更新页面或重新进入课件"
    : instructionLength >
        COURSEWARE_ASSISTANT_LIMITS.teacherInstruction
      ? `补充要求不能超过${COURSEWARE_ASSISTANT_LIMITS.teacherInstruction}个字符`
      : "";

  const hasSavedSlot = Boolean(editor.serverSlot);
  const selectedPageID = selectedPage?.pageId || "";

  useEffect(() => {
    setActiveMainTab("learning");
    setActiveContentTab("basics");
    setDeploymentState({
      ...EMPTY_COURSEWARE_ASSISTANT_DEPLOYMENT_DOCK_STATE,
    });
    setValidityFocusSequence(0);
  }, [selectedPageID]);

  const openDelivery = useCallback(() => {
    setActiveMainTab("delivery");

    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        document.getElementById(DELIVERY_SECTION_ID)?.scrollIntoView({
          behavior: "smooth",
          block: "start",
        });
      });
    });
  }, []);

  const openValiditySettings = useCallback(() => {
    setActiveMainTab("delivery");
    setValidityFocusSequence((previous) => previous + 1);
  }, []);

  useEffect(() => {
    if (!validityFocusSequence || activeMainTab !== "delivery") return;

    let cancelled = false;
    let timer: number | null = null;
    let attempts = 0;

    const focusValiditySection = () => {
      if (cancelled) return;

      const section = document.getElementById(
        COURSEWARE_ASSISTANT_VALIDITY_SECTION_ID,
      );

      if (section && section.getClientRects().length > 0) {
        section.scrollIntoView({
          behavior: "smooth",
          block: "center",
        });

        const focusTarget = section.querySelector<HTMLElement>(
          "button:not(:disabled), input:not(:disabled)",
        );

        window.setTimeout(() => {
          focusTarget?.focus({ preventScroll: true });
        }, 220);

        if (typeof section.animate === "function") {
          section.animate(
            [
              { boxShadow: "0 0 0 0 rgba(79,123,232,0)" },
              { boxShadow: "0 0 0 4px rgba(79,123,232,0.20)" },
              { boxShadow: "0 0 0 0 rgba(79,123,232,0)" },
            ],
            {
              duration: 1400,
              easing: "ease-out",
            },
          );
        }

        setValidityFocusSequence(0);
        return;
      }

      attempts += 1;
      if (attempts < 30) {
        timer = window.setTimeout(
          focusValiditySection,
          50,
        );
      }
    };

    const frame = window.requestAnimationFrame(
      focusValiditySection,
    );

    return () => {
      cancelled = true;
      window.cancelAnimationFrame(frame);
      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [
    activeMainTab,
    validityFocusSequence,
  ]);

  useEffect(() => {
    if (!selectedPageID) return;

    const consumeRequest = () => {
      if (
        consumeCoursewareAssistantValiditySettingsRequest(
          coursewareId,
          selectedPageID,
        )
      ) {
        openValiditySettings();
      }
    };

    consumeRequest();
    window.addEventListener(
      COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_EVENT,
      consumeRequest,
    );

    return () => {
      window.removeEventListener(
        COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_EVENT,
        consumeRequest,
      );
    };
  }, [
    coursewareId,
    openValiditySettings,
    selectedPageID,
  ]);

  const handleDelete = () => {
    if (!editor.serverSlot) {
      return;
    }

    if (
      !window.confirm(
        "确定删除当前页面的教学智能体配置吗？已发布的历史版本不会被删除。",
      )
    ) {
      return;
    }

    void editor.remove();
  };

  const handleGenerate = () => {
    setActiveMainTab("learning");

    if (generateBlocker) {
      return;
    }

    void editor.generatePlan();
  };

  const handleSave = () => {
    void editor.save();
  };

  const handlePublish = async () => {
    const controller =
      deploymentControllerRef.current;

    if (!controller) {
      openDelivery();
      return;
    }

    const published =
      await controller.publishCurrent();

    if (!published) {
      openDelivery();
    }
  };

  return (
    <>
      <section
        style={{
          marginTop: 16,
          padding: 16,
          paddingBottom: 280,
          borderRadius: 12,
          border: `1px solid ${C.border}`,
          background: C.background,
        }}
      >
        <CoursewareAssistantPanelHeader
          selectedPage={selectedPage}
          hasSlot={hasSavedSlot}
          isDirty={editor.isDirty}
        />

        {!selectedPage && (
          <CoursewareAssistantMessageBox
            kind="warning"
            text="当前页面缺少稳定page_id。系统不会使用可变化的页码保存教学智能体，请刷新课件后重新选择页面。"
          />
        )}

        {selectedPage && (
          <CoursewareAssistantCurrentPageCard
            selectedPage={selectedPage}
          />
        )}

        {editor.notice && (
          <CoursewareAssistantMessageBox
            kind={editor.notice.kind}
            text={editor.notice.text}
          />
        )}

        {editor.loadError && (
          <CoursewareAssistantMessageBox
            kind="error"
            text={editor.loadError}
          />
        )}

        {editor.validationErrors.length > 0 && (
          <CoursewareAssistantValidationBox
            errors={editor.validationErrors}
          />
        )}

        {selectedPage && editor.loading && (
          <div
            style={{
              padding: "28px 12px",
              textAlign: "center",
              color: C.textMuted,
              fontSize: 12,
            }}
          >
            正在恢复当前页面的教学智能体…
          </div>
        )}

        {selectedPage &&
          !editor.loading &&
          !editor.loadError && (
            <>
              <CoursewareAssistantTabBar
                options={MAIN_TABS}
                value={activeMainTab}
                onChange={setActiveMainTab}
                disabled={editorBusy}
              />

              {activeMainTab === "learning" && (
                <>
                  <CoursewareAssistantTeachingModeSelector
                    draft={editor.draft}
                    onChange={editor.setDraft}
                    subject={editor.courseMeta.subject}
                    grade={editor.courseMeta.grade}
                    pageTitle={selectedPage.pageTitle}
                    pageSummary={
                      editor.contextPreview?.current_page
                        .content_summary || ""
                    }
                    interactionType={
                      editor.contextPreview?.interaction
                        .declared_type || ""
                    }
                    disabled={editorBusy}
                  />

                  <CoursewareAssistantCoreEditor
                    draft={editor.draft}
                    onChange={editor.setDraft}
                    subject={editor.courseMeta.subject}
                    grade={editor.courseMeta.grade}
                    lessonPlanId={
                      editor.courseMeta.lessonPlanId
                    }
                    disabled={editorBusy}
                    generating={editor.generating}
                    mode="generation"
                    onKeyDown={editor.handleKeyDown}
                  />

                  {hasGeneratedContent && (
                    <CoursewareAssistantNextStepCard
                      title="学生活动方案已经创建"
                      description="下一步查看开场、互动步骤、提示和完成标准。保存与发布按钮始终位于右下角操作台。"
                      buttonLabel="查看互动内容"
                      disabled={editorBusy}
                      onClick={() => {
                        setActiveMainTab("content");
                      }}
                    />
                  )}
                </>
              )}

              {activeMainTab === "content" && (
                <>
                  <CoursewareAssistantPlanSummary
                    draft={editor.draft}
                    generating={editor.generating}
                  />

                  <CoursewareAssistantTabBar
                    options={CONTENT_TABS}
                    value={activeContentTab}
                    onChange={setActiveContentTab}
                    disabled={editorBusy}
                    compact
                  />

                  <CoursewareAssistantAdvancedToolbar
                    hasSlot={hasSavedSlot}
                    busy={editorBusy}
                    deleting={editor.deleting}
                    canUndo={editor.canUndo}
                    canRedo={editor.canRedo}
                    onUndo={editor.undo}
                    onRedo={editor.redo}
                    onRefresh={() => {
                      void editor.refresh();
                    }}
                    onDelete={handleDelete}
                  />

                  {activeContentTab === "basics" && (
                    <CoursewareAssistantCoreEditor
                      draft={editor.draft}
                      onChange={editor.setDraft}
                      subject={editor.courseMeta.subject}
                      grade={editor.courseMeta.grade}
                      lessonPlanId={
                        editor.courseMeta.lessonPlanId
                      }
                      disabled={editorBusy}
                      generating={editor.generating}
                      mode="advanced"
                      onKeyDown={editor.handleKeyDown}
                    />
                  )}

                  {activeContentTab === "steps" && (
                    <CoursewareAssistantQuestionChainEditor
                      draft={editor.draft}
                      onChange={editor.setDraft}
                      disabled={editorBusy}
                      onKeyDown={editor.handleKeyDown}
                    />
                  )}

                  {activeContentTab === "support" && (
                    <>
                      <CoursewareAssistantPolicyContextEditor
                        draft={editor.draft}
                        onChange={editor.setDraft}
                        disabled={editorBusy}
                        mode="teaching"
                        onKeyDown={editor.handleKeyDown}
                      />

                      <CoursewareAssistantBranchEditor
                        draft={editor.draft}
                        onChange={editor.setDraft}
                        disabled={editorBusy}
                        onKeyDown={editor.handleKeyDown}
                      />
                    </>
                  )}

                  {activeContentTab === "context" && (
                    <>
                      <CoursewareAssistantPolicyContextEditor
                        draft={editor.draft}
                        onChange={editor.setDraft}
                        disabled={editorBusy}
                        mode="context"
                        onKeyDown={editor.handleKeyDown}
                      />

                      <CoursewareAssistantContextReceipt
                        preview={editor.contextPreview}
                        loading={editor.contextLoading}
                        error={editor.contextError}
                        onRefresh={() => {
                          void editor.refreshContext();
                        }}
                      />
                    </>
                  )}
                </>
              )}

              {activeMainTab === "delivery" &&
                editor.isDirty && (
                  <CoursewareAssistantMessageBox
                    kind="warning"
                    text="当前还有未保存修改。发布只会读取数据库中已经保存的方案，请使用右下角操作台的“保存方案”。"
                  />
                )}

              <div id={DELIVERY_SECTION_ID}>
                <CoursewareAssistantDeploymentManager
                  ref={deploymentControllerRef}
                  coursewareId={coursewareId}
                  pageId={selectedPage.pageId}
                  pageTitle={selectedPage.pageTitle}
                  hasSavedSlot={hasSavedSlot}
                  hasUnsavedChanges={editor.isDirty}
                  slotStatus={editor.serverSlot?.status}
                  slotUpdatedAt={
                    editor.serverSlot?.updated_at
                  }
                  disabled={editorBusy}
                  visible={
                    activeMainTab === "delivery"
                  }
                  onStateChange={setDeploymentState}
                />

                {activeMainTab === "delivery" && (
                  <CoursewareAssistantPreview
                    coursewareId={coursewareId}
                    pageId={selectedPage.pageId}
                    pageTitle={selectedPage.pageTitle}
                    hasSavedSlot={hasSavedSlot}
                    hasUnsavedChanges={editor.isDirty}
                    disabled={editorBusy}
                  />
                )}
              </div>
            </>
          )}
      </section>

      <CoursewareAssistantActionDock
        selectedPage={selectedPage}
        hasGeneratedContent={hasGeneratedContent}
        hasSavedSlot={hasSavedSlot}
        isDirty={editor.isDirty}
        generateBlocker={generateBlocker}
        loading={editor.loading}
        generating={editor.generating}
        saving={editor.saving}
        deleting={editor.deleting}
        deployment={deploymentState}
        onGenerate={handleGenerate}
        onSave={handleSave}
        onPublish={handlePublish}
        onOpenDelivery={openDelivery}
      />
    </>
  );
}
