/**
 * 教学智能体方案编辑器状态Hook。
 *
 * 职责：
 *   - 按用户、课件和稳定page_id隔离浏览器草稿；
 *   - 从数据库恢复当前页面插槽；
 *   - 加载安全上下文回执；
 *   - 按教师选择的教学方式调用AI生成可编辑方案草稿；
 *   - 创建、更新和删除数据库插槽；
 *   - 提供撤销、重做、未保存状态和离开页面提醒。
 *
 * 安全边界：
 *   - 不读取助手完整提示词；
 *   - 不读取页面完整HTML或教案全文；
 *   - AI生成或保存失败时绝不清空当前草稿；
 *   - 旧页面异步响应不能覆盖新页面草稿。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  Dispatch,
  SetStateAction,
} from "react";

import {
  createCoursewareAssistantSlot,
  deleteCoursewareAssistantSlot,
  generateCoursewareAssistantPlan,
  getCourseware,
  getCoursewareAssistantContextPreview,
  listCoursewareAssistantSlots,
  updateCoursewareAssistantSlot,
  type CoursewareAssistantContextPreview,
  type CoursewareAssistantSlotView,
} from "@/api/coursewares";

import {
  useProtectedDraft,
} from "@/hooks/useProtectedDraft";

import type {
  CoursewareAssistantSelectedPage,
} from "./coursewareAssistantSelection";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  applyGeneratedCoursewareAssistantPlan,
  createEmptyCoursewareAssistantDraft,
  draftFromCoursewareAssistantSlot,
  parseCoursewareAssistantDraft,
  serializeCoursewareAssistantDraft,
  toCreateCoursewareAssistantRequest,
  toUpdateCoursewareAssistantRequest,
  validateCoursewareAssistantDraft,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

export interface CoursewareAssistantCourseMeta {
  subject: string;
  grade: string;
  lessonPlanId: string | null;
}

export interface CoursewareAssistantEditorNotice {
  kind: "success" | "info" | "error";
  text: string;
}

interface UseCoursewareAssistantEditorOptions {
  coursewareId: string;
  selectedPage:
    CoursewareAssistantSelectedPage | null;
  userId?: string | null;
}

export function useCoursewareAssistantEditor({
  coursewareId,
  selectedPage,
  userId,
}: UseCoursewareAssistantEditorOptions) {
  /**
   * 只使用稳定page_id参与请求和Hook依赖。
   *
   * 父页面会在每次渲染时重新构造selectedPage展示对象；
   * 若直接依赖整个对象，内部状态更新也可能让加载回调反复重建，
   * 进而重复请求课件、插槽和上下文。
   */
  const selectedPageID =
    selectedPage?.pageId.trim() || "";

  const resourceKey =
    selectedPageID
      ? `${coursewareId}:${selectedPageID}`
      : `${coursewareId}:missing-page`;

  const activeResourceRef =
    useRef(resourceKey);

  activeResourceRef.current =
    resourceKey;

  const loadRequestRef =
    useRef(0);

  const operationRequestRef =
    useRef(0);

  const [
    serverSlot,
    setServerSlot,
  ] = useState<
    CoursewareAssistantSlotView | null
  >(null);

  const [
    courseMeta,
    setCourseMeta,
  ] = useState<CoursewareAssistantCourseMeta>({
    subject: "",
    grade: "",
    lessonPlanId: null,
  });

  const [
    serverInitialDraft,
    setServerInitialDraft,
  ] = useState<CoursewareAssistantEditorDraft>(
    () =>
      createEmptyCoursewareAssistantDraft(),
  );

  const [
    contextPreview,
    setContextPreview,
  ] = useState<
    CoursewareAssistantContextPreview | null
  >(null);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    contextLoading,
    setContextLoading,
  ] = useState(false);

  const [
    generating,
    setGenerating,
  ] = useState(false);

  const [
    saving,
    setSaving,
  ] = useState(false);

  const [
    deleting,
    setDeleting,
  ] = useState(false);

  const [
    loadError,
    setLoadError,
  ] = useState("");

  const [
    contextError,
    setContextError,
  ] = useState("");

  const [
    notice,
    setNotice,
  ] = useState<
    CoursewareAssistantEditorNotice | null
  >(null);

  const [
    showValidation,
    setShowValidation,
  ] = useState(false);

  const initialValue =
    useMemo(
      () =>
        serializeCoursewareAssistantDraft(
          serverInitialDraft,
        ),
      [
        serverInitialDraft,
      ],
    );

  const protectedDraft =
    useProtectedDraft({
      userId,
      scope:
        "courseware-assistant-editor",
      resourceId:
        resourceKey,
      field: "structured-plan",
      initialValue,
      enabled:
        Boolean(selectedPageID),
      maxHistory: 30,
      coalesceMs: 800,
    });

  /**
   * 单独提取稳定的草稿写入函数。
   *
   * 避免setDraft回调依赖整个protectedDraft结果对象，
   * 同时让Hooks依赖检查准确追踪实际使用的函数。
   */
  const protectedDraftSetValue =
    protectedDraft.setValue;

  const draft =
    useMemo(
      () =>
        parseCoursewareAssistantDraft(
          protectedDraft.value,
          serverInitialDraft,
        ),
      [
        protectedDraft.value,
        serverInitialDraft,
      ],
    );

  const serializedDraft =
    useMemo(
      () =>
        serializeCoursewareAssistantDraft(
          draft,
        ),
      [
        draft,
      ],
    );

  const isDirty =
    serializedDraft !==
    initialValue;

  const validationErrors =
    useMemo(
      () =>
        showValidation
          ? validateCoursewareAssistantDraft(
              draft,
            )
          : [],
      [
        draft,
        showValidation,
      ],
    );

  const setDraft: Dispatch<
    SetStateAction<
      CoursewareAssistantEditorDraft
    >
  > = useCallback(
    (next) => {
      protectedDraftSetValue(
        (previousText) => {
          const previous =
            parseCoursewareAssistantDraft(
              previousText,
              serverInitialDraft,
            );

          const resolved =
            typeof next ===
              "function"
              ? next(previous)
              : next;

          return serializeCoursewareAssistantDraft(
            resolved,
          );
        },
      );
    },
    [
      protectedDraftSetValue,
      serverInitialDraft,
    ],
  );

  const loadContext =
    useCallback(async () => {
      if (!selectedPageID) {
        setContextPreview(null);
        setContextError("");
        setContextLoading(false);
        return;
      }

      const capturedResource =
        resourceKey;

      setContextLoading(true);
      setContextError("");

      try {
        const result =
          await getCoursewareAssistantContextPreview(
            coursewareId,
            selectedPageID,
          );

        if (
          activeResourceRef.current !==
          capturedResource
        ) {
          return;
        }

        setContextPreview(
          result,
        );
      } catch (cause) {
        if (
          activeResourceRef.current !==
          capturedResource
        ) {
          return;
        }

        setContextPreview(null);
        setContextError(
          cause instanceof Error
            ? cause.message
            : "读取安全上下文回执失败",
        );
      } finally {
        if (
          activeResourceRef.current ===
          capturedResource
        ) {
          setContextLoading(false);
        }
      }
    }, [
      coursewareId,
      resourceKey,
      selectedPageID,
    ]);

  const loadEditor =
    useCallback(async () => {
      const requestID =
        loadRequestRef.current + 1;

      loadRequestRef.current =
        requestID;

      operationRequestRef.current += 1;

      setGenerating(false);
      setSaving(false);
      setDeleting(false);
      setNotice(null);
      setShowValidation(false);
      setLoadError("");
      setContextError("");
      setContextPreview(null);

      if (!selectedPageID) {
        setLoading(false);
        setContextLoading(false);
        setServerSlot(null);
        setServerInitialDraft(
          createEmptyCoursewareAssistantDraft(),
        );
        return;
      }

      const capturedResource =
        resourceKey;

      setLoading(true);

      try {
        const [
          courseware,
          slotResult,
        ] = await Promise.all([
          getCourseware(
            coursewareId,
          ),
          listCoursewareAssistantSlots(
            coursewareId,
          ),
        ]);

        if (
          loadRequestRef.current !==
            requestID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        const slot =
          (
            slotResult.slots ||
            []
          ).find(
            (item) =>
              item.page_id ===
              selectedPageID,
          ) || null;

        setCourseMeta({
          subject:
            courseware.subject || "",
          grade:
            courseware.grade || "",
          lessonPlanId:
            courseware.lesson_plan_id ||
            null,
        });

        setServerSlot(
          slot,
        );

        setServerInitialDraft(
          slot
            ? draftFromCoursewareAssistantSlot(
                slot,
              )
            : createEmptyCoursewareAssistantDraft(),
        );

        setLoading(false);

        void loadContext();
      } catch (cause) {
        if (
          loadRequestRef.current !==
            requestID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        setLoading(false);
        setLoadError(
          cause instanceof Error
            ? cause.message
            : "读取教学智能体编辑数据失败",
        );
      }
    }, [
      coursewareId,
      loadContext,
      resourceKey,
      selectedPageID,
    ]);

  useEffect(() => {
    void loadEditor();

    return () => {
      loadRequestRef.current += 1;
      operationRequestRef.current += 1;
    };
  }, [
    loadEditor,
  ]);

  useEffect(() => {
    if (!isDirty) {
      return;
    }

    const handleBeforeUnload =
      (
        event: BeforeUnloadEvent,
      ) => {
        event.preventDefault();
        event.returnValue = "";
      };

    window.addEventListener(
      "beforeunload",
      handleBeforeUnload,
    );

    return () =>
      window.removeEventListener(
        "beforeunload",
        handleBeforeUnload,
      );
  }, [
    isDirty,
  ]);

  const generatePlan =
    useCallback(async () => {
      if (
        !selectedPageID ||
        generating ||
        saving ||
        deleting
      ) {
        return;
      }

      if (
        Array.from(
          draft.teacherInstruction,
        ).length >
        COURSEWARE_ASSISTANT_LIMITS
          .teacherInstruction
      ) {
        setNotice({
          kind: "error",
          text:
            `给AI的补充要求不能超过${
              COURSEWARE_ASSISTANT_LIMITS
                .teacherInstruction
            }个字符`,
        });
        return;
      }

      const capturedResource =
        resourceKey;

      const operationID =
        operationRequestRef.current + 1;

      operationRequestRef.current =
        operationID;

      setGenerating(true);
      setNotice(null);

      try {
        const result =
          await generateCoursewareAssistantPlan(
            coursewareId,
            selectedPageID,
            {
              assistant_id:
                draft.assistantId,
              teaching_mode:
                draft.guidancePlan
                  .teaching_mode,
              teacher_instruction:
                draft.teacherInstruction,
            },
          );

        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        setDraft(
          (previous) =>
            applyGeneratedCoursewareAssistantPlan(
              previous,
              result,
            ),
        );

        setShowValidation(false);
        setNotice({
          kind: "info",
          text:
            "AI已按所选学习方式生成方案，并写入当前浏览器草稿。尚未保存到数据库，请检查后点击保存。",
        });
      } catch (cause) {
        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        setNotice({
          kind: "error",
          text:
            cause instanceof Error
              ? cause.message
              : "AI方案生成失败，当前草稿已保留",
        });
      } finally {
        if (
          operationRequestRef.current ===
            operationID &&
          activeResourceRef.current ===
            capturedResource
        ) {
          setGenerating(false);
        }
      }
    }, [
      coursewareId,
      deleting,
      draft,
      generating,
      resourceKey,
      saving,
      selectedPageID,
      setDraft,
    ]);

  const save =
    useCallback(async () => {
      if (
        !selectedPageID ||
        loading ||
        generating ||
        saving ||
        deleting
      ) {
        return;
      }

      setShowValidation(true);

      const errors =
        validateCoursewareAssistantDraft(
          draft,
        );

      if (errors.length > 0) {
        setNotice({
          kind: "error",
          text:
            "当前方案仍有未完成或不符合协议的字段，请根据校验提示修改。",
        });
        return;
      }

      const capturedResource =
        resourceKey;

      const operationID =
        operationRequestRef.current + 1;

      operationRequestRef.current =
        operationID;

      setSaving(true);
      setNotice(null);

      try {
        let saved:
          CoursewareAssistantSlotView;

        if (serverSlot) {
          saved =
            await updateCoursewareAssistantSlot(
              coursewareId,
              serverSlot.id,
              toUpdateCoursewareAssistantRequest(
                draft,
              ),
            );
        } else {
          /**
           * 创建接口按后端契约默认生成active插槽，
           * Create请求本身不接受status字段。
           *
           * 若教师首次保存前已经选择disabled，
           * 创建成功后立即使用更新接口同步最终状态。
           */
          const created =
            await createCoursewareAssistantSlot(
              coursewareId,
              selectedPageID,
              toCreateCoursewareAssistantRequest(
                draft,
              ),
            );

          if (
            created.status ===
            draft.status
          ) {
            saved = created;
          } else {
            try {
              saved =
                await updateCoursewareAssistantSlot(
                  coursewareId,
                  created.id,
                  toUpdateCoursewareAssistantRequest(
                    draft,
                  ),
                );
            } catch (statusCause) {
              /**
               * 创建已经成功、状态同步失败时不能继续把它当作新插槽。
               *
               * 先接管数据库中已经存在的插槽作为服务器基线，
               * 但不清除浏览器草稿。教师选择的disabled仍保留为
               * 未保存修改，下一次点击保存会走更新而不是重复创建。
               */
              if (
                operationRequestRef.current ===
                  operationID &&
                activeResourceRef.current ===
                  capturedResource
              ) {
                setServerSlot(
                  created,
                );

                setServerInitialDraft(
                  draftFromCoursewareAssistantSlot(
                    created,
                  ),
                );

                setShowValidation(
                  false,
                );

                setNotice({
                  kind: "error",
                  text:
                    statusCause instanceof
                    Error
                      ? `插槽已创建，但状态同步失败：${statusCause.message}。停用草稿已保留，请再次保存。`
                      : "插槽已创建，但状态同步失败。停用草稿已保留，请再次保存。",
                });
              }

              return;
            }
          }
        }

        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        const savedDraft =
          draftFromCoursewareAssistantSlot(
            saved,
          );

        protectedDraft.clear();
        setServerSlot(saved);
        setServerInitialDraft(
          savedDraft,
        );
        setShowValidation(false);
        setNotice({
          kind: "success",
          text:
            "教学智能体方案已保存到数据库。",
        });

        void loadContext();
      } catch (cause) {
        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        setNotice({
          kind: "error",
          text:
            cause instanceof Error
              ? cause.message
              : "保存失败，当前草稿已保留",
        });
      } finally {
        if (
          operationRequestRef.current ===
            operationID &&
          activeResourceRef.current ===
            capturedResource
        ) {
          setSaving(false);
        }
      }
    }, [
      coursewareId,
      deleting,
      draft,
      generating,
      loadContext,
      loading,
      protectedDraft,
      resourceKey,
      saving,
      selectedPageID,
      serverSlot,
    ]);

  const remove =
    useCallback(async () => {
      if (
        !serverSlot ||
        deleting ||
        saving ||
        generating
      ) {
        return;
      }

      const capturedResource =
        resourceKey;

      const operationID =
        operationRequestRef.current + 1;

      operationRequestRef.current =
        operationID;

      setDeleting(true);
      setNotice(null);

      try {
        await deleteCoursewareAssistantSlot(
          coursewareId,
          serverSlot.id,
        );

        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        protectedDraft.clear();
        setServerSlot(null);
        setServerInitialDraft(
          createEmptyCoursewareAssistantDraft(),
        );
        setShowValidation(false);
        setNotice({
          kind: "success",
          text:
            "当前页面的教学智能体插槽已删除。历史部署版本不会被删除。",
        });
      } catch (cause) {
        if (
          operationRequestRef.current !==
            operationID ||
          activeResourceRef.current !==
            capturedResource
        ) {
          return;
        }

        setNotice({
          kind: "error",
          text:
            cause instanceof Error
              ? cause.message
              : "删除失败，当前草稿已保留",
        });
      } finally {
        if (
          operationRequestRef.current ===
            operationID &&
          activeResourceRef.current ===
            capturedResource
        ) {
          setDeleting(false);
        }
      }
    }, [
      coursewareId,
      deleting,
      generating,
      protectedDraft,
      resourceKey,
      saving,
      serverSlot,
    ]);

  return {
    draft,
    setDraft,
    serverSlot,
    courseMeta,
    contextPreview,
    loading,
    contextLoading,
    generating,
    saving,
    deleting,
    loadError,
    contextError,
    notice,
    isDirty,
    validationErrors,
    canUndo:
      protectedDraft.canUndo,
    canRedo:
      protectedDraft.canRedo,
    undo:
      protectedDraft.undo,
    redo:
      protectedDraft.redo,
    handleKeyDown:
      protectedDraft.handleKeyDown,
    refresh:
      loadEditor,
    refreshContext:
      loadContext,
    generatePlan,
    save,
    remove,
  };
}
