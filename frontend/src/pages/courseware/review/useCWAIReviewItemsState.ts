/**
 * useCWAIReviewItemsState.ts
 *
 * 课件AI审核整改项加载、采纳、更新和正式交付选择状态。
 *
 * 交付选择规则：
 *   - 已确认且存在确认指令的正式整改项可以交付；
 *   - 首次加载时可交付项默认选中；
 *   - 新确认或恢复成confirmed时自动选中；
 *   - 忽略、失效或进入其他不可交付状态时自动移除；
 *   - 作者自审项不进入正式审核交付选择。
 */

import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";

import {
  getCWAIReviewSessionItems,
  materializeCWAIReviewItems,
  type CWAIReviewItem,
  type CWAIReviewSession,
} from "@/api/coursewares";

import {
  isDeliverableFormalReviewItem,
  mergeCWAIReviewItems,
} from "./cwAIReviewControllerHelpers";

export interface UseCWAIReviewItemsStateOptions {
  session: CWAIReviewSession | null;
  isSelfReview: boolean;

  mountedRef: MutableRefObject<boolean>;

  setError: Dispatch<
    SetStateAction<string>
  >;
}

export function useCWAIReviewItemsState({
  session,
  isSelfReview,
  mountedRef,
  setError,
}: UseCWAIReviewItemsStateOptions) {
  const [
    items,
    setItems,
  ] = useState<CWAIReviewItem[]>([]);

  const [
    selectedItemIds,
    setSelectedItemIds,
  ] = useState<string[]>([]);

  const [
    materializingFindingIds,
    setMaterializingFindingIds,
  ] = useState<string[]>([]);

  const resetReviewItems =
    useCallback(() => {
      setItems([]);
      setSelectedItemIds([]);
      setMaterializingFindingIds([]);
    }, []);

  const handleRemoveSelectedItem =
    useCallback(
      (
        itemId: string,
      ) => {
        setSelectedItemIds(
          (previous) =>
            previous.filter(
              (value) =>
                value !== itemId,
            ),
        );
      },
      [],
    );

  const loadSessionItems =
    useCallback(
      async (
        sessionId: string,
      ) => {
        const result =
          await getCWAIReviewSessionItems(
            sessionId,
          );

        if (!mountedRef.current) {
          return;
        }

        const nextItems =
          result.items || [];

        setItems(nextItems);

        if (isSelfReview) {
          setSelectedItemIds([]);
          return;
        }

        setSelectedItemIds(
          nextItems
            .filter(
              isDeliverableFormalReviewItem,
            )
            .map(
              (item) => item.id,
            ),
        );
      },
      [
        isSelfReview,
        mountedRef,
      ],
    );

  useEffect(() => {
    if (
      !session ||
      session.status !== "done"
    ) {
      return;
    }

    void loadSessionItems(
      session.id,
    ).catch((cause) => {
      if (!mountedRef.current) {
        return;
      }

      setError(
        cause instanceof Error
          ? cause.message
          : "加载整改项失败",
      );
    });
  }, [
    loadSessionItems,
    mountedRef,
    session?.id,
    session?.status,
    setError,
  ]);

  const handleAdoptFinding =
    useCallback(
      async (
        findingId: string,
      ) => {
        if (
          !session ||
          session.status !== "done" ||
          materializingFindingIds
            .includes(findingId)
        ) {
          return;
        }

        setMaterializingFindingIds(
          (previous) => [
            ...previous,
            findingId,
          ],
        );

        setError("");

        try {
          const result =
            await materializeCWAIReviewItems(
              session.id,
              [findingId],
            );

          if (!mountedRef.current) {
            return;
          }

          setItems(
            (previous) =>
              mergeCWAIReviewItems(
                previous,
                result.items || [],
              ),
          );
        } catch (cause) {
          if (!mountedRef.current) {
            return;
          }

          setError(
            cause instanceof Error
              ? cause.message
              : "创建整改项失败",
          );
        } finally {
          if (mountedRef.current) {
            setMaterializingFindingIds(
              (previous) =>
                previous.filter(
                  (id) =>
                    id !== findingId,
                ),
            );
          }
        }
      },
      [
        materializingFindingIds,
        mountedRef,
        session,
        setError,
      ],
    );

  const handleToggleItemSelection =
    useCallback(
      (
        itemId: string,
        selected: boolean,
      ) => {
        const targetItem =
          items.find(
            (item) =>
              item.id === itemId,
          );

        if (
          selected &&
          !isDeliverableFormalReviewItem(
            targetItem,
          )
        ) {
          return;
        }

        if (!selected) {
          handleRemoveSelectedItem(
            itemId,
          );
          return;
        }

        setSelectedItemIds(
          (previous) =>
            Array.from(
              new Set([
                ...previous,
                itemId,
              ]),
            ),
        );
      },
      [
        handleRemoveSelectedItem,
        items,
      ],
    );

  const handleItemChanged =
    useCallback(
      (
        item: CWAIReviewItem,
      ) => {
        const previousItem =
          items.find(
            (currentItem) =>
              currentItem.id ===
              item.id,
          );

        const wasDeliverable =
          isDeliverableFormalReviewItem(
            previousItem,
          );

        const isDeliverable =
          isDeliverableFormalReviewItem(
            item,
          );

        setItems(
          (previous) =>
            mergeCWAIReviewItems(
              previous,
              [item],
            ),
        );

        if (
          isSelfReview ||
          item.source_type !==
            "formal"
        ) {
          return;
        }

        setSelectedItemIds(
          (previous) => {
            if (!isDeliverable) {
              return previous.filter(
                (itemID) =>
                  itemID !== item.id,
              );
            }

            if (!wasDeliverable) {
              return Array.from(
                new Set([
                  ...previous,
                  item.id,
                ]),
              );
            }

            return previous;
          },
        );
      },
      [
        isSelfReview,
        items,
      ],
    );

  return {
    items,
    selectedItemIds,
    materializingFindingIds,

    resetReviewItems,
    handleRemoveSelectedItem,
    handleAdoptFinding,
    handleToggleItemSelection,
    handleItemChanged,
  };
}

export type CWAIReviewItemsState =
  ReturnType<
    typeof useCWAIReviewItemsState
  >;
