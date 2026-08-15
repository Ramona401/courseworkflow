/**
 * CWReviewIssueFilters.tsx
 *
 * 课件评审问题搜索与筛选栏。
 *
 * 支持关键词、严重程度、状态、页面和来源筛选，并生成只包含问题ID与枚举条件的分享链接。
 * “/”用于聚焦搜索框；输入控件获得焦点时不会抢占其他快捷键。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type CSSProperties,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from "react";

import type {
  CWAIReviewItem,
} from "@/api/coursewares";

import {
  createDefaultCWReviewFilters,
  hasActiveCWReviewFilters,
  isCWReviewTextEntryTarget,
  isNearestVisibleCWReviewElement,
  type CWReviewIssueFilterState,
  type CWReviewPageFilter,
  type CWReviewSeverityFilter,
  type CWReviewSourceFilter,
  type CWReviewStatusFilter,
} from "./coursewareReviewWorkspaceState";

import {
  countCWReviewActiveFilters,
  recordCoursewareReviewUsage,
  resolveCWReviewFilterPageNumber,
  resolveCWReviewUsageModeFromElement,
} from "./coursewareReviewUsage";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

export interface CWReviewIssueFiltersProps {
  items: CWAIReviewItem[];
  value: CWReviewIssueFilterState;
  resultCount: number;
  shareURL: string;
  onChange: (
    value: CWReviewIssueFilterState,
  ) => void;
}

interface PageOption {
  value: CWReviewPageFilter;
  label: string;
  order: number;
}

function buildPageOptions(
  items: CWAIReviewItem[],
): PageOption[] {
  const options =
    new Map<string, PageOption>();

  for (const item of items) {
    if (item.page_number_snapshot <= 0) {
      options.set("global", {
        value: "global",
        label: "整课问题",
        order: 0,
      });
      continue;
    }

    const value =
      String(
        item.page_number_snapshot,
      );

    const title =
      item.page_title_snapshot
        .trim();

    options.set(value, {
      value,
      label: title
        ? `P${value} · ${title}`
        : `P${value}`,
      order:
        item.page_number_snapshot,
    });
  }

  return Array.from(
    options.values(),
  ).sort(
    (left, right) =>
      left.order - right.order,
  );
}

async function copyShareURL(
  shareURL: string,
): Promise<void> {
  if (
    navigator.clipboard &&
    window.isSecureContext
  ) {
    await navigator.clipboard.writeText(
      shareURL,
    );
    return;
  }

  const textarea =
    document.createElement(
      "textarea",
    );

  textarea.value = shareURL;
  textarea.style.position =
    "fixed";
  textarea.style.left =
    "-9999px";
  textarea.style.opacity =
    "0";

  document.body.appendChild(
    textarea,
  );

  textarea.focus();
  textarea.select();

  const copied =
    document.execCommand(
      "copy",
    );

  document.body.removeChild(
    textarea,
  );

  if (!copied) {
    throw new Error(
      "浏览器拒绝复制",
    );
  }
}

export default function CWReviewIssueFilters({
  items,
  value,
  resultCount,
  shareURL,
  onChange,
}: CWReviewIssueFiltersProps) {
  const rootRef =
    useRef<HTMLElement | null>(null);

  const searchInputRef =
    useRef<HTMLInputElement | null>(null);

  const lastTrackedFilterSignatureRef =
    useRef("");

  const [
    copyMessage,
    setCopyMessage,
  ] = useState("");

  const pageOptions =
    useMemo(
      () =>
        buildPageOptions(
          items,
        ),
      [items],
    );

  const active =
    hasActiveCWReviewFilters(
      value,
    );

  const activeFilterCount =
    countCWReviewActiveFilters(
      value,
    );

  const resolveUsageMode = () =>
    resolveCWReviewUsageModeFromElement(
      rootRef.current,
    );

  useEffect(() => {
    const handleKeyDown = (
      event: KeyboardEvent,
    ) => {
      if (
        event.key !== "/" ||
        event.ctrlKey ||
        event.metaKey ||
        event.altKey ||
        isCWReviewTextEntryTarget(
          event.target,
        )
      ) {
        return;
      }

      if (
        !isNearestVisibleCWReviewElement(
          searchInputRef.current,
          '[data-cw-review-search-input="true"]',
        )
      ) {
        return;
      }

      event.preventDefault();

      searchInputRef.current
        ?.focus();

      const mode =
        resolveUsageMode();

      if (mode) {
        recordCoursewareReviewUsage({
          event: "keyboard_shortcut",
          mode,
          shortcut: "focus_search",
          totalCount: items.length,
          visibleCount: resultCount,
          activeFilterCount,
          pageNumber:
            resolveCWReviewFilterPageNumber(
              value,
            ),
        });
      }
    };

    window.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      window.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, [
    activeFilterCount,
    items.length,
    resultCount,
    value,
  ]);

  useEffect(() => {
    const signature = [
      value.query.trim()
        ? "query"
        : "",
      value.severity,
      value.status,
      value.page,
      value.source,
      value.showCompleted
        ? "completed"
        : "",
    ].join("|");

    if (!active) {
      lastTrackedFilterSignatureRef.current =
        signature;
      return;
    }

    if (
      lastTrackedFilterSignatureRef.current ===
      signature
    ) {
      return;
    }

    const timer =
      window.setTimeout(() => {
        const mode =
          resolveUsageMode();

        if (!mode) {
          return;
        }

        lastTrackedFilterSignatureRef.current =
          signature;

        recordCoursewareReviewUsage({
          event: "filter_applied",
          mode,
          totalCount: items.length,
          visibleCount: resultCount,
          activeFilterCount,
          pageNumber:
            resolveCWReviewFilterPageNumber(
              value,
            ),
        });
      }, 650);

    return () => {
      window.clearTimeout(timer);
    };
  }, [
    active,
    activeFilterCount,
    items.length,
    resultCount,
    value,
  ]);

  const update = <
    Key extends
      keyof CWReviewIssueFilterState,
  >(
    key: Key,
    nextValue:
      CWReviewIssueFilterState[Key],
  ) => {
    const next = {
      ...value,
      [key]: nextValue,
    };

    if (
      key === "status" &&
      nextValue === "completed"
    ) {
      next.showCompleted = true;
    }

    onChange(next);
  };

  const clearFilters = () => {
    onChange(
      createDefaultCWReviewFilters(),
    );

    window.setTimeout(() => {
      searchInputRef.current
        ?.focus();
    }, 0);
  };

  const copyLink = () => {
    setCopyMessage("");

    void copyShareURL(
      shareURL,
    )
      .then(() => {
        setCopyMessage(
          "链接已复制",
        );

        const mode =
          resolveUsageMode();

        if (mode) {
          recordCoursewareReviewUsage({
            event: "link_copied",
            mode,
            totalCount: items.length,
            visibleCount: resultCount,
            activeFilterCount,
            pageNumber:
              resolveCWReviewFilterPageNumber(
                value,
              ),
          });
        }
      })
      .catch(() => {
        setCopyMessage(
          "复制失败，请从地址栏复制",
        );
      });
  };

  return (
    <section
      ref={rootRef}
      aria-label="问题搜索与筛选"
      style={{
        marginTop: "14px",
        padding: "12px",
        borderRadius: "10px",
        border: `1px solid ${C.border}`,
        background: C.card,
      }}
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "minmax(220px, 2fr) repeat(4, minmax(130px, 1fr))",
          gap: "8px",
          alignItems: "end",
        }}
      >
        <FilterField
          label="搜索问题"
          hint="按 / 快速聚焦"
        >
          <input
            ref={searchInputRef}
            data-cw-review-search-input="true"
            type="search"
            value={value.query}
            maxLength={120}
            placeholder="标题、说明、整改要求或页码"
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              update(
                "query",
                event.target.value,
              )
            }
            onKeyDown={(event: ReactKeyboardEvent<HTMLInputElement>) => {
              if (
                event.key === "Escape" &&
                value.query
              ) {
                event.preventDefault();
                update(
                  "query",
                  "",
                );
              }
            }}
            style={inputStyle}
          />
        </FilterField>

        <FilterField label="严重程度">
          <select
            value={value.severity}
            onChange={(event: ChangeEvent<HTMLSelectElement>) =>
              update(
                "severity",
                event.target
                  .value as
                  CWReviewSeverityFilter,
              )
            }
            style={inputStyle}
          >
            <option value="all">
              全部级别
            </option>
            <option value="critical">
              必须处理
            </option>
            <option value="high">
              高优先级
            </option>
            <option value="medium">
              中优先级
            </option>
            <option value="low">
              低优先级
            </option>
            <option value="info">
              供参考
            </option>
          </select>
        </FilterField>

        <FilterField label="处理状态">
          <select
            value={value.status}
            onChange={(event: ChangeEvent<HTMLSelectElement>) =>
              update(
                "status",
                event.target
                  .value as
                  CWReviewStatusFilter,
              )
            }
            style={inputStyle}
          >
            <option value="all">
              全部状态
            </option>
            <option value="pending">
              待处理
            </option>
            <option value="applying">
              修改中
            </option>
            <option value="waiting_confirm">
              待确认
            </option>
            <option value="stale">
              需要重新检查
            </option>
            <option value="completed">
              已完成
            </option>
          </select>
        </FilterField>

        <FilterField label="页面范围">
          <select
            value={value.page}
            onChange={(event: ChangeEvent<HTMLSelectElement>) =>
              update(
                "page",
                event.target
                  .value as
                  CWReviewPageFilter,
              )
            }
            style={inputStyle}
          >
            <option value="all">
              全部页面
            </option>

            {pageOptions.map(
              (option) => (
                <option
                  key={option.value}
                  value={option.value}
                >
                  {option.label}
                </option>
              ),
            )}
          </select>
        </FilterField>

        <FilterField label="问题来源">
          <select
            value={value.source}
            onChange={(event: ChangeEvent<HTMLSelectElement>) =>
              update(
                "source",
                event.target
                  .value as
                  CWReviewSourceFilter,
              )
            }
            style={inputStyle}
          >
            <option value="all">
              全部来源
            </option>
            <option value="formal">
              正式审核
            </option>
            <option value="self">
              作者自审
            </option>
          </select>
        </FilterField>
      </div>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "8px",
          flexWrap: "wrap",
          marginTop: "10px",
          paddingTop: "10px",
          borderTop:
            `1px solid ${C.border}`,
        }}
      >
        <label
          style={{
            minHeight: "36px",
            display: "inline-flex",
            alignItems: "center",
            gap: "7px",
            padding: "7px 10px",
            borderRadius: "8px",
            border:
              `1px solid ${
                value.showCompleted
                  ? C.success
                  : C.border
              }`,
            background:
              value.showCompleted
                ? "#ECFDF5"
                : "#FFFFFF",
            color:
              value.showCompleted
                ? C.success
                : C.textSec,
            fontSize: "13px",
            fontWeight: 700,
            cursor: "pointer",
            boxSizing: "border-box",
          }}
        >
          <input
            type="checkbox"
            checked={
              value.showCompleted
            }
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              update(
                "showCompleted",
                event.target.checked,
              )
            }
          />
          包含已完成
        </label>

        <span
          style={{
            color: active
              ? C.primary
              : C.textSec,
            fontSize: "13px",
            fontWeight: 700,
          }}
        >
          当前显示 {resultCount} / {items.length} 条
        </span>

        <span
          style={{
            flex: 1,
            minWidth: "12px",
          }}
        />

        {active && (
          <button
            type="button"
            onClick={clearFilters}
            style={secondaryButtonStyle}
          >
            清除筛选
          </button>
        )}

        <button
          type="button"
          onClick={copyLink}
          style={{
            ...secondaryButtonStyle,
            color: C.primary,
            border:
              `1px solid ${C.primary}`,
          }}
        >
          复制当前视图链接
        </button>

        {copyMessage && (
          <span
            role="status"
            style={{
              color:
                copyMessage ===
                "链接已复制"
                  ? C.success
                  : C.warning,
              fontSize: "12px",
              lineHeight: 1.5,
            }}
          >
            {copyMessage}
          </span>
        )}
      </div>

      <div
        style={{
          marginTop: "7px",
          color: C.textMuted,
          fontSize: "12px",
          lineHeight: 1.5,
        }}
      >
        快捷键：/ 搜索，Alt+↓ 进入下一条任务；单问题工作区中使用 Esc 返回、Alt+←/→ 切换问题、Alt+L 查看相关教案。
      </div>
    </section>
  );
}

function FilterField({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <label
      style={{
        minWidth: 0,
        display: "block",
      }}
    >
      <span
        style={{
          display: "flex",
          alignItems: "center",
          gap: "5px",
          marginBottom: "4px",
          color: C.text,
          fontSize: "12px",
          fontWeight: 700,
        }}
      >
        {label}

        {hint && (
          <span
            style={{
              color: C.textMuted,
              fontSize: "11px",
              fontWeight: 500,
            }}
          >
            {hint}
          </span>
        )}
      </span>

      {children}
    </label>
  );
}

const inputStyle: CSSProperties = {
  width: "100%",
  minHeight: "38px",
  padding: "8px 10px",
  borderRadius: "8px",
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  color: C.text,
  fontSize: "13px",
  fontFamily: "inherit",
  boxSizing: "border-box",
  outline: "none",
};

const secondaryButtonStyle: CSSProperties = {
  minHeight: "36px",
  padding: "8px 11px",
  borderRadius: "8px",
  border: `1px solid ${C.border}`,
  background: C.bg,
  color: C.textSec,
  fontSize: "12px",
  fontWeight: 700,
  cursor: "pointer",
};
