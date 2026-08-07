/**
 * CWReviewCarryoverPanel.tsx
 *
 * 审核员复查上一轮整改问题。
 *
 * 本组件只收集审核员的复审判断，真正保存解决状态发生在提交正式审核决定时。
 *
 * 展示原则：
 *   - 先显示复审进度和阻断数量，再按稳定页面分组；
 *   - 默认隐藏本轮已经勾选“确认解决”的问题，减少长列表滚动；
 *   - 每条问题默认只展示紧凑摘要，原整改要求和状态解释按需展开；
 *   - 只有 applied 问题可以勾选确认解决，其他状态必须显示不可操作原因；
 *   - 页面定位继续以稳定 page_id 为准，页码仅用于展示和跳转。
 */

import { useEffect, useMemo, useState, type ChangeEvent, type CSSProperties } from "react";

import type { CWReviewCarryoverItem } from "@/api/coursewares";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  purple: "#7C3AED",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

interface PageReference { id: string; page_number: number; }

interface CarryoverGroup {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWReviewCarryoverItem[];
}

interface VisibleCarryoverGroup { group: CarryoverGroup; visibleItems: CWReviewCarryoverItem[]; }

export interface CWReviewCarryoverPanelProps {
  items: CWReviewCarryoverItem[];
  pages: PageReference[];
  pendingReviewRound: number;
  resolvedItemIds: string[];
  onResolvedChange: (itemId: string, resolved: boolean) => void;
  onSelectPage: (pageNumber: number) => void;
}

const SEVERITY_CONFIG: Record<
  string,
  { label: string; color: string; background: string; order: number }
> = {
  critical: { label: "必须处理", color: "#B91C1C", background: "#FEE2E2", order: 1 },
  high: { label: "优先处理", color: "#C2410C", background: "#FFEDD5", order: 2 },
  medium: { label: "建议处理", color: "#A16207", background: "#FEF9C3", order: 3 },
  low: { label: "可以优化", color: "#0369A1", background: "#E0F2FE", order: 4 },
  info: { label: "供参考", color: C.textSec, background: "#F1F5F9", order: 5 },
};

const STATUS_CONFIG: Record<
  string,
  { label: string; help: string; color: string; background: string }
> = {
  applied: {
    label: "作者已完成修改",
    help: "系统已经记录作者完成页面修改。请打开当前页面检查实际结果，再决定是否确认解决。",
    color: C.primary,
    background: "#EEF2FF",
  },
  applying: {
    label: "作者仍在修改",
    help: "作者尚未记录完成修改，本轮不能确认解决。",
    color: C.warning,
    background: "#FFFBEB",
  },
  confirmed: {
    label: "未记录完成修改",
    help: "作者尚未记录完成修改，本轮不能确认解决。",
    color: C.warning,
    background: "#FFFBEB",
  },
  detected: {
    label: "整改要求不完整",
    help: "历史问题没有形成完整执行记录，本轮不能确认解决。",
    color: C.warning,
    background: "#FFFBEB",
  },
  discussing: {
    label: "整改要求仍在讨论",
    help: "历史问题仍处于讨论阶段，本轮不能确认解决。",
    color: C.purple,
    background: "#F5F3FF",
  },
  stale: {
    label: "页面后来发生变化",
    help: "页面在作者登记完成后又发生变化，需要作者重新检查并登记完成。",
    color: C.danger,
    background: "#FEF2F2",
  },
  orphaned: {
    label: "原页面已经删除",
    help: "原问题页面已经删除，本轮不能直接确认解决，需要继续退回或检查整课结构。",
    color: C.danger,
    background: "#FEF2F2",
  },
};

function formatDateTime(raw: string | null): string {
  if (!raw) {
    return "";
  }

  try {
    return new Date(raw).toLocaleString("zh-CN");
  } catch {
    return raw;
  }
}

function resolveCurrentPage(item: CWReviewCarryoverItem, pages: PageReference[]): PageReference | undefined {
  const pageID = item.page_id?.trim() || "";
  return pageID ? pages.find((page) => page.id === pageID) : undefined;
}

function isMarkedResolved(
  item: CWReviewCarryoverItem,
  resolvedIDSet: ReadonlySet<string>,
): boolean {
  return item.status === "applied" && resolvedIDSet.has(item.id);
}

function buildCarryoverGroups(items: CWReviewCarryoverItem[]): CarryoverGroup[] {
  const groupMap = new Map<string, CarryoverGroup>();

  for (const item of items) {
    const isGlobalIssue = item.page_number_snapshot <= 0;
    const stablePageID = item.page_id?.trim() || "";
    const key = isGlobalIssue
      ? "global"
      : stablePageID || ["snapshot", item.page_number_snapshot].join(":");

    const current = groupMap.get(key) || {
      key,
      pageNumber: isGlobalIssue ? 0 : item.page_number_snapshot,
      pageTitle: isGlobalIssue ? "整课问题" : `第${item.page_number_snapshot}页`,
      items: [],
    };

    current.items.push(item);
    groupMap.set(key, current);
  }

  const groups = Array.from(groupMap.values());

  for (const group of groups) {
    group.items.sort((left, right) => {
      const leftOrder = SEVERITY_CONFIG[left.severity]?.order || 99;
      const rightOrder = SEVERITY_CONFIG[right.severity]?.order || 99;
      return leftOrder !== rightOrder ? leftOrder - rightOrder : left.id.localeCompare(right.id);
    });
  }

  return groups.sort((left, right) => {
    if (left.pageNumber === 0) {
      return -1;
    }

    if (right.pageNumber === 0) {
      return 1;
    }

    return left.pageNumber - right.pageNumber;
  });
}

export default function CWReviewCarryoverPanel({
  items,
  pages,
  pendingReviewRound,
  resolvedItemIds,
  onResolvedChange,
  onSelectPage,
}: CWReviewCarryoverPanelProps) {
  const resolvedIDSet = useMemo(() => new Set(resolvedItemIds), [resolvedItemIds]);
  const groups = useMemo(() => buildCarryoverGroups(items), [items]);
  const [showResolved, setShowResolved] = useState(false);
  const [expandedGroupKeys, setExpandedGroupKeys] = useState<Set<string>>(new Set());
  const [expandedItemIDs, setExpandedItemIDs] = useState<Set<string>>(new Set());

  const appliedCount = items.filter((item) => item.status === "applied").length;
  const unfinishedCount = items.length - appliedCount;
  const resolvedCount = items.filter((item) => isMarkedResolved(item, resolvedIDSet)).length;
  const waitingReviewCount = appliedCount - resolvedCount;
  const changedPageCount = items.filter(
    (item) => item.status === "stale" || item.status === "orphaned",
  ).length;

  const pendingAppliedItems =
    groups
      .flatMap((group) => group.items)
      .filter(
        (item) =>
          item.status === "applied" &&
          !resolvedIDSet.has(item.id),
      );

  const pendingAppliedCount =
    pendingAppliedItems.length;

  const nextReviewItem =
    pendingAppliedItems[0];

  const nextReviewGroup =
    nextReviewItem
      ? groups.find((group) =>
          group.items.some(
            (item) =>
              item.id ===
              nextReviewItem.id,
          ),
        )
      : undefined;

  const visibleGroups = useMemo<VisibleCarryoverGroup[]>(
    () =>
      groups
        .map((group) => ({
          group,
          visibleItems: showResolved
            ? group.items
            : group.items.filter((item) => !isMarkedResolved(item, resolvedIDSet)),
        }))
        .filter(({ visibleItems }) => visibleItems.length > 0),
    [groups, resolvedIDSet, showResolved],
  );

  useEffect(() => {
    setExpandedGroupKeys((previous) => {
      const validKeys = new Set(groups.map((group) => group.key));
      const next = new Set(Array.from(previous).filter((key) => validKeys.has(key)));

      if (next.size === 0) {
        const firstUnresolvedGroup =
          groups.find((group) =>
            group.items.some((item) => !isMarkedResolved(item, resolvedIDSet)),
          ) || groups[0];

        if (firstUnresolvedGroup) {
          next.add(firstUnresolvedGroup.key);
        }
      }

      return next;
    });
  }, [groups, resolvedIDSet]);

  const markItems = (
    targets: CWReviewCarryoverItem[],
    resolved: boolean,
  ) => {
    for (const item of targets) {
      const currentlyResolved =
        isMarkedResolved(
          item,
          resolvedIDSet,
        );

      if (
        resolved &&
        item.status === "applied" &&
        !currentlyResolved
      ) {
        onResolvedChange(
          item.id,
          true,
        );
      }

      if (
        !resolved &&
        currentlyResolved
      ) {
        onResolvedChange(
          item.id,
          false,
        );
      }
    }
  };

  const markAll = (
    resolved: boolean,
  ) => {
    markItems(
      items,
      resolved,
    );
  };

  const openNextReview = () => {
    if (!nextReviewItem) {
      return;
    }

    if (nextReviewGroup) {
      setExpandedGroupKeys(
        (previous) =>
          new Set(previous).add(
            nextReviewGroup.key,
          ),
      );
    }

    setExpandedItemIDs(
      (previous) =>
        new Set(previous).add(
          nextReviewItem.id,
        ),
    );

    const currentPage =
      resolveCurrentPage(
        nextReviewItem,
        pages,
      );

    if (currentPage) {
      onSelectPage(
        currentPage.page_number,
      );
    }
  };

  const toggleGroup = (groupKey: string) => {
    setExpandedGroupKeys((previous) => {
      const next = new Set(previous);
      next.has(groupKey) ? next.delete(groupKey) : next.add(groupKey);
      return next;
    });
  };

  const toggleItemDetails = (itemID: string) => {
    setExpandedItemIDs((previous) => {
      const next = new Set(previous);
      next.has(itemID) ? next.delete(itemID) : next.add(itemID);
      return next;
    });
  };

  if (items.length === 0) {
    return (
      <div style={emptyStateStyle}>
        <div style={{ marginBottom: "8px", fontSize: "28px" }}>✅</div>
        本轮没有需要复查的历史整改问题。
      </div>
    );
  }

  return (
    <section>
      <div style={overviewPanelStyle}>
        <div style={overviewTitleStyle}>第 {pendingReviewRound} 轮复审 · 上轮整改</div>
        <div style={overviewDescriptionStyle}>
          只有作者已记录完成修改的 applied 问题可以确认解决。请逐条打开当前页面检查，
          不要仅根据状态批量通过。
        </div>

        <div style={metricGridStyle}>
          <OverviewMetric label="全部问题" value={items.length} tone="primary" />
          <OverviewMetric label="待复查" value={waitingReviewCount} tone="primary" />
          <OverviewMetric label="未完成" value={unfinishedCount} tone="warning" />
          <OverviewMetric label="已确认解决" value={resolvedCount} tone="success" />
          <OverviewMetric label="页面变化" value={changedPageCount} tone="danger" />
        </div>

        {unfinishedCount > 0 && (
          <div style={blockingNoticeStyle}>
            暂时不能审核通过：还有 {unfinishedCount} 条上一轮问题尚未达到可复查状态。
          </div>
        )}

        <div style={actionRowStyle}>
          <button
            type="button"
            onClick={() => markAll(true)}
            disabled={
              pendingAppliedCount === 0
            }
            style={actionButtonStyle(
              pendingAppliedCount > 0,
              "success",
            )}
          >
            确认全部待复查问题
            {" "}
            {pendingAppliedCount}
          </button>

          {nextReviewItem && (
            <button
              type="button"
              onClick={
                openNextReview
              }
              style={actionButtonStyle(
                true,
                "primary",
              )}
            >
              打开下一条待复查
            </button>
          )}

          {resolvedCount > 0 && (
            <button
              type="button"
              onClick={() => markAll(false)}
              style={actionButtonStyle(true, "secondary")}
            >
              清除全部确认
            </button>
          )}
        </div>
      </div>

      <div style={{ ...actionRowStyle, marginTop: "12px" }}>
        <button
          type="button"
          onClick={() => setShowResolved(false)}
          aria-pressed={!showResolved}
          style={filterButtonStyle(!showResolved, "primary")}
        >
          仅看未确认解决
        </button>

        <button
          type="button"
          onClick={() => setShowResolved(true)}
          aria-pressed={showResolved}
          style={filterButtonStyle(showResolved, "success")}
        >
          查看已确认解决
        </button>
      </div>

      {visibleGroups.length === 0 ? (
        <div style={{ ...emptyStateStyle, marginTop: "12px", background: C.card }}>
          当前没有待复查的问题。可以查看已确认解决的记录，或继续提交本轮审核决定。
        </div>
      ) : (
        visibleGroups.map(({ group, visibleItems }) => {
          const expanded = expandedGroupKeys.has(group.key);
          const resolvedInGroup = group.items.filter((item) =>
            isMarkedResolved(item, resolvedIDSet),
          ).length;
          const appliedInGroup = group.items.filter((item) => item.status === "applied").length;
          const pendingAppliedInGroup = group.items.filter(
            (item) =>
              item.status === "applied" &&
              !resolvedIDSet.has(item.id),
          ).length;
          const unfinishedInGroup = group.items.length - appliedInGroup;
          const changedInGroup = group.items.filter(
            (item) => item.status === "stale" || item.status === "orphaned",
          ).length;
          const firstItem = group.items[0];
          const currentPage = firstItem ? resolveCurrentPage(firstItem, pages) : undefined;

          return (
            <div key={group.key} style={groupContainerStyle}>
              <div
                style={{
                  ...groupHeaderStyle,
                  background: unfinishedInGroup > 0 ? "#FFFFFF" : C.bg,
                }}
              >
                <button
                  type="button"
                  onClick={() => toggleGroup(group.key)}
                  aria-expanded={expanded}
                  style={groupToggleStyle}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      ...arrowStyle,
                      color: group.pageNumber > 0 ? C.primary : C.purple,
                      transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
                    }}
                  >
                    ▶
                  </span>

                  <span style={{ minWidth: 0, flex: 1 }}>
                    <span style={groupTitleStyle}>
                      {group.pageNumber > 0 ? `P${group.pageNumber} ` : "整课 "}
                      {group.pageTitle || "历史整改问题"}
                    </span>
                    <span style={groupSummaryStyle}>
                      {group.items.length} 条问题
                      {unfinishedInGroup > 0 ? ` · ${unfinishedInGroup} 条未完成` : ""}
                      {appliedInGroup > 0 ? ` · ${appliedInGroup} 条可复查` : ""}
                      {resolvedInGroup > 0 ? ` · ${resolvedInGroup} 条已确认解决` : ""}
                      {changedInGroup > 0 ? ` · ${changedInGroup} 条页面变化` : ""}
                    </span>
                  </span>
                </button>

                {pendingAppliedInGroup >
                  0 && (
                  <button
                    type="button"
                    onClick={() =>
                      markItems(
                        group.items,
                        true,
                      )
                    }
                    style={actionButtonStyle(
                      true,
                      "success",
                    )}
                  >
                    确认
                    {group.pageNumber > 0
                      ? "本页"
                      : "本组"}
                    {" "}
                    {pendingAppliedInGroup}
                    {" "}
                    条
                  </button>
                )}

                {currentPage && (
                  <button
                    type="button"
                    onClick={() => onSelectPage(currentPage.page_number)}
                    style={openPageButtonStyle}
                  >
                    打开页面
                  </button>
                )}
              </div>

              {expanded && (
                <div style={groupBodyStyle}>
                  {visibleItems.map((item) => (
                    <CarryoverIssueCard
                      key={item.id}
                      item={item}
                      pages={pages}
                      selected={isMarkedResolved(item, resolvedIDSet)}
                      detailsOpen={expandedItemIDs.has(item.id)}
                      onToggleDetails={() => toggleItemDetails(item.id)}
                      onResolvedChange={onResolvedChange}
                      onSelectPage={onSelectPage}
                    />
                  ))}
                </div>
              )}
            </div>
          );
        })
      )}
    </section>
  );
}

function CarryoverIssueCard({
  item,
  pages,
  selected,
  detailsOpen,
  onToggleDetails,
  onResolvedChange,
  onSelectPage,
}: {
  item: CWReviewCarryoverItem;
  pages: PageReference[];
  selected: boolean;
  detailsOpen: boolean;
  onToggleDetails: () => void;
  onResolvedChange: (itemID: string, resolved: boolean) => void;
  onSelectPage: (pageNumber: number) => void;
}) {
  const canResolve = item.status === "applied";
  const severity = SEVERITY_CONFIG[item.severity] || SEVERITY_CONFIG.info;
  const status = STATUS_CONFIG[item.status] || {
    label: "等待人工检查",
    help: "请结合当前页面和原整改要求作出判断。",
    color: C.textSec,
    background: C.bg,
  };
  const currentPage = resolveCurrentPage(item, pages);
  const isGlobal = item.page_number_snapshot <= 0;
  const originalPageDeleted = !isGlobal && !currentPage;
  const locationLabel = isGlobal
    ? "整课"
    : currentPage
      ? `P${currentPage.page_number}`
      : `原P${item.page_number_snapshot}`;
  const summaryTitle = item.title.trim() || item.description.trim() || "未填写问题标题";
  const summaryDescription = item.title.trim() ? item.description.trim() : "";

  return (
    <article
      style={{
        ...issueCardStyle,
        border: selected ? "2px solid #86EFAC" : `1px solid ${C.border}`,
        background: selected ? "#F0FDF4" : C.card,
      }}
    >
      <div style={tagRowStyle}>
        <span style={tagStyle(severity.color, severity.background)}>{severity.label}</span>
        <span style={tagStyle(status.color, status.background)}>{status.label}</span>
        <span style={metaTextStyle}>
          原L{item.original_review_level}第{item.original_review_round}轮 · {locationLabel}
          {originalPageDeleted ? " · 页面已删除" : ""}
        </span>
      </div>

      <div style={issueTitleStyle}>{summaryTitle}</div>

      {summaryDescription && <div style={issueDescriptionStyle}>{summaryDescription}</div>}

      <div style={actionRowStyle}>
        {currentPage && (
          <button
            type="button"
            onClick={() => onSelectPage(currentPage.page_number)}
            style={actionButtonStyle(true, "primary")}
          >
            打开当前页面
          </button>
        )}

        <button
          type="button"
          onClick={onToggleDetails}
          aria-expanded={detailsOpen}
          style={actionButtonStyle(true, "secondary")}
        >
          {detailsOpen ? "收起要求" : "查看整改要求"}
        </button>
      </div>

      <label
        style={{
          ...resolutionLabelStyle,
          border: selected ? "1px solid #86EFAC" : `1px solid ${C.border}`,
          background: !canResolve ? C.bg : selected ? "#DCFCE7" : "#FFFFFF",
          color: !canResolve ? C.textMuted : selected ? "#166534" : C.textSec,
          cursor: canResolve ? "pointer" : "not-allowed",
        }}
      >
        <input
          type="checkbox"
          checked={selected}
          disabled={!canResolve}
          onChange={(event: ChangeEvent<HTMLInputElement>) => {
            if (canResolve) {
              onResolvedChange(item.id, event.target.checked);
            }
          }}
          style={{ marginTop: "3px" }}
        />
        <span>
          {!canResolve
            ? "作者尚未记录完成修改，本轮不能确认解决"
            : selected
              ? "本轮确认：这个问题已经解决"
              : "检查当前课件后，确认这个问题是否已经解决"}
        </span>
      </label>

      {detailsOpen && (
        <div style={detailsContainerStyle}>
          {summaryDescription && <DetailBlock title="问题说明" content={item.description} />}
          <DetailBlock title="上轮确认的整改要求" content={item.confirmed_instruction} />
          <div style={{ ...statusHelpStyle, background: status.background, color: status.color }}>
            {status.help}
            {item.applied_at && <> 作者完成时间：{formatDateTime(item.applied_at)}。</>}
          </div>
        </div>
      )}
    </article>
  );
}

function OverviewMetric({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "primary" | "success" | "warning" | "danger";
}) {
  const config = {
    primary: { color: C.primary, background: "#EEF2FF" },
    success: { color: C.success, background: "#ECFDF5" },
    warning: { color: C.warning, background: "#FFF7ED" },
    danger: { color: C.danger, background: "#FEF2F2" },
  }[tone];
  const active = value > 0 || label === "全部问题";

  return (
    <div
      style={{
        minWidth: "82px",
        padding: "8px 10px",
        borderRadius: "9px",
        background: active ? config.background : C.bg,
        color: active ? config.color : C.textMuted,
      }}
    >
      <div style={{ fontSize: "20px", fontWeight: 700, lineHeight: 1.2 }}>{value}</div>
      <div style={{ marginTop: "3px", fontSize: "12px", fontWeight: 600, lineHeight: 1.4 }}>
        {label}
      </div>
    </div>
  );
}

function DetailBlock({ title, content }: { title: string; content: string }) {
  return (
    <div style={detailBlockStyle}>
      <div style={{ color: C.text, fontSize: "13px", fontWeight: 700 }}>{title}</div>
      <div style={detailContentStyle}>{content || "暂无内容"}</div>
    </div>
  );
}

function tagStyle(color: string, background: string): CSSProperties {
  return { padding: "3px 7px", borderRadius: "6px", background, color, fontSize: "12px", fontWeight: 700 };
}

function filterButtonStyle(active: boolean, tone: "primary" | "success"): CSSProperties {
  const config = {
    primary: { color: C.primary, background: "#EEF2FF" },
    success: { color: C.success, background: "#ECFDF5" },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border: `1px solid ${active ? config.color : C.border}`,
    background: active ? config.background : "#FFFFFF",
    color: active ? config.color : C.textSec,
    fontSize: "13px",
    fontWeight: 700,
    cursor: "pointer",
  };
}

function actionButtonStyle(
  enabled: boolean,
  tone: "primary" | "success" | "secondary",
): CSSProperties {
  const config = {
    primary: { color: "#FFFFFF", background: C.primary, border: C.primary },
    success: { color: "#FFFFFF", background: C.success, border: C.success },
    secondary: { color: C.textSec, background: "#FFFFFF", border: C.border },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 10px",
    borderRadius: "8px",
    border: `1px solid ${enabled ? config.border : "#CBD5E1"}`,
    background: enabled ? config.background : "#F1F5F9",
    color: enabled ? config.color : C.textMuted,
    fontSize: "12px",
    fontWeight: 700,
    cursor: enabled ? "pointer" : "not-allowed",
  };
}

const emptyStateStyle: CSSProperties = {
  padding: "28px 16px", border: `1px dashed ${C.border}`, borderRadius: "10px", color: C.textSec,
  fontSize: "14px", lineHeight: 1.7, textAlign: "center",
};

const overviewPanelStyle: CSSProperties = {
  padding: "14px", border: `1px solid ${C.primary}30`, borderRadius: "10px", background: "#EEF2FF",
};

const overviewTitleStyle: CSSProperties = {
  color: C.text, fontSize: "18px", fontWeight: 700, lineHeight: 1.4,
};

const overviewDescriptionStyle: CSSProperties = {
  marginTop: "5px", maxWidth: "75ch", color: C.textSec, fontSize: "14px", lineHeight: 1.6,
};

const metricGridStyle: CSSProperties = {
  display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(92px, 1fr))", gap: "8px",
  marginTop: "12px",
};

const blockingNoticeStyle: CSSProperties = {
  marginTop: "12px", padding: "10px 12px", border: "1px solid #FECACA", borderRadius: "8px",
  background: "#FEF2F2", color: C.danger, fontSize: "14px", fontWeight: 600, lineHeight: 1.6,
};

const actionRowStyle: CSSProperties = {
  display: "flex", alignItems: "center", gap: "8px", flexWrap: "wrap", marginTop: "10px",
};

const groupContainerStyle: CSSProperties = {
  marginTop: "12px", overflow: "hidden", border: `1px solid ${C.border}`, borderRadius: "10px",
  background: C.card,
};

const groupHeaderStyle: CSSProperties = {
  display: "flex", alignItems: "stretch", gap: "8px", padding: "10px 12px",
};

const groupToggleStyle: CSSProperties = {
  minWidth: 0, flex: 1, display: "flex", alignItems: "center", gap: "10px", padding: 0,
  border: "none", background: "transparent", textAlign: "left", cursor: "pointer",
};

const arrowStyle: CSSProperties = {
  flexShrink: 0, fontSize: "16px", transition: "transform 160ms ease",
};

const groupTitleStyle: CSSProperties = {
  display: "block", overflow: "hidden", color: C.text, fontSize: "15px", fontWeight: 700,
  lineHeight: 1.5, textOverflow: "ellipsis", whiteSpace: "nowrap",
};

const groupSummaryStyle: CSSProperties = {
  display: "block", marginTop: "3px", color: C.textMuted, fontSize: "12px", lineHeight: 1.5,
};

const openPageButtonStyle: CSSProperties = {
  flexShrink: 0, minHeight: "36px", padding: "8px 10px", border: `1px solid ${C.primary}`,
  borderRadius: "8px", background: "#EEF2FF", color: C.primary, fontSize: "12px",
  fontWeight: 700, cursor: "pointer",
};

const groupBodyStyle: CSSProperties = {
  padding: "2px 12px 12px", borderTop: `1px solid ${C.border}`,
};

const issueCardStyle: CSSProperties = {
  marginTop: "10px", padding: "12px", borderRadius: "9px",
};

const tagRowStyle: CSSProperties = {
  display: "flex", alignItems: "center", gap: "6px", flexWrap: "wrap",
};

const metaTextStyle: CSSProperties = {
  color: C.textMuted, fontSize: "12px", lineHeight: 1.5,
};

const issueTitleStyle: CSSProperties = {
  marginTop: "8px", color: C.text, fontSize: "15px", fontWeight: 700, lineHeight: 1.55,
};

const issueDescriptionStyle: CSSProperties = {
  display: "-webkit-box", marginTop: "4px", overflow: "hidden", color: C.textSec,
  fontSize: "13px", lineHeight: 1.6, WebkitLineClamp: 2, WebkitBoxOrient: "vertical",
};

const resolutionLabelStyle: CSSProperties = {
  display: "flex", alignItems: "flex-start", gap: "8px", marginTop: "10px", padding: "10px",
  borderRadius: "8px", fontSize: "13px", fontWeight: 700, lineHeight: 1.55,
};

const detailsContainerStyle: CSSProperties = {
  marginTop: "10px", paddingTop: "10px", borderTop: `1px solid ${C.border}`,
};

const detailBlockStyle: CSSProperties = {
  marginTop: "8px", padding: "9px 10px", borderRadius: "8px", background: C.bg,
};

const detailContentStyle: CSSProperties = {
  marginTop: "4px", color: C.textSec, fontSize: "14px", lineHeight: 1.65,
  whiteSpace: "pre-wrap", wordBreak: "break-word",
};

const statusHelpStyle: CSSProperties = {
  marginTop: "8px", padding: "9px 10px", borderRadius: "8px", fontSize: "13px", lineHeight: 1.6,
};
