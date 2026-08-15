/**
 * useExternalRefineInstruction.ts
 *
 * 将正式审核或作者自审中已经独立确认的修改要求，
 * 写入当前页面的受保护微调草稿。
 *
 * 安全边界：
 *   1. 只填充草稿，不调用页面微调接口；
 *   2. 保留老师当前尚未提交的草稿；
 *   3. 相同注入ID只消费一次；
 *   4. 注入后切换到安全默认的保留结构微调；
 *   5. 页面真正修改成功后，父组件再解除一次性注入信号。
 */

import {
  useEffect,
  useRef,
} from "react";

import type {
  Dispatch,
  SetStateAction,
} from "react";

import type {
  CWRefineMode,
} from "@/api/coursewares";

import type {
  ProtectedDraftSetter,
} from "@/hooks/useProtectedDraft";

/** 父组件传入的一次性整改要求。 */
export interface RefinePanelExternalInstruction {
  /** 每次注入使用唯一ID，防止重复追加草稿。 */
  id: string;

  /** 整改项所属课件。 */
  coursewareId: string;

  /** 后端整改项ID，执行微调时必须原样提交。 */
  reviewItemId: string;

  /** 稳定页面解析后的当前页码。 */
  targetPageNumber: number;

  /** 用户已经独立确认的最终修改要求或修改方案。 */
  content: string;
}

export interface UseExternalRefineInstructionOptions {
  externalInstruction?:
    | RefinePanelExternalInstruction
    | null;

  coursewareId: string;
  pageNum: number;

  updateRefineInput:
    ProtectedDraftSetter;

  setRefineMode:
    Dispatch<
      SetStateAction<CWRefineMode>
    >;

  setMessage:
    Dispatch<
      SetStateAction<string>
    >;
}

/**
 * 消费外部确认要求并写入受保护草稿。
 */
export function useExternalRefineInstruction({
  externalInstruction,
  coursewareId,
  pageNum,
  updateRefineInput,
  setRefineMode,
  setMessage,
}: UseExternalRefineInstructionOptions): void {
  /**
   * React严格模式可能重复执行挂载Effect；
   * 使用Ref保证同一ID不会重复追加。
   */
  const lastConsumedIDRef =
    useRef("");

  useEffect(() => {
    if (
      !externalInstruction ||
      externalInstruction
        .coursewareId !==
        coursewareId ||
      externalInstruction
        .targetPageNumber !==
        pageNum
    ) {
      return;
    }

    const instructionID =
      externalInstruction
        .id.trim();

    if (
      !instructionID ||
      lastConsumedIDRef
        .current ===
        instructionID
    ) {
      return;
    }

    lastConsumedIDRef.current =
      instructionID;

    const content =
      externalInstruction
        .content.trim();

    if (!content) {
      return;
    }

    updateRefineInput(
      (previous) => {
        const current =
          previous.trim();

        if (
          current.includes(
            content,
          )
        ) {
          return previous;
        }

        return current
          ? `${current}\n\n${content}`
          : content;
      },
    );

    setRefineMode(
      "preserve",
    );

    setMessage(
      "✅ 已把当前确认的修改要求带入本页草稿，请检查后手动点击“AI微调”",
    );
  }, [
    externalInstruction,
    coursewareId,
    pageNum,
    setMessage,
    setRefineMode,
    updateRefineInput,
  ]);
}
