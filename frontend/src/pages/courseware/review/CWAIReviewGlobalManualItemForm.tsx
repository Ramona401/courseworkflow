/**
 * CWAIReviewGlobalManualItemForm.tsx
 *
 * 全局讨论中的人工新增整改项表单。
 *
 * 安全边界：
 *   1. 只有已经保存的全局assistant消息存在时才允许提交；
 *   2. 页面列表从当前课件重新读取，不接受用户手工填写page_id；
 *   3. 空页面选择明确表示整课问题；
 *   4. 候选修改指令只保存为original_suggestion，不能在此处确认；
 *   5. 创建成功后不修改页面、不提交审核决定；
 *   6. 所有文本与页面数量在提交前按后端边界再次复核。
 */

import {
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  createCWAIReviewGlobalManualItems,
  getCourseware,
  type CWAIReviewItem,
  type CWAIReviewSeverity,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  cwGlobalPageButtonStyle,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES,
  CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES,
  CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES,
  CW_GLOBAL_MANUAL_ITEM_MAX_PAGES,
  CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";

interface CWGlobalPageOption {
  id: string;
  pageNumber: number;
  title: string;
}

export interface CWAIReviewGlobalManualItemFormProps {
  sessionId: string;
  messageId: string;
  items: CWAIReviewItem[];

  onSelectPage: (pageNumber: number) => void;
  onCreated: (items: CWAIReviewItem[]) => void;
}

function buildFallbackPageOptions(
  items: CWAIReviewItem[],
): CWGlobalPageOption[] {
  const pageMap = new Map<string, CWGlobalPageOption>();

  for (const item of items) {
    const pageID = item.page_id?.trim() || "";

    if (!pageID || item.page_number_snapshot <= 0) {
      continue;
    }

    pageMap.set(pageID, {
      id: pageID,
      pageNumber: item.page_number_snapshot,
      title:
        item.page_title_snapshot.trim() ||
        `第${item.page_number_snapshot}页`,
    });
  }

  return Array.from(pageMap.values()).sort(
    (left, right) => left.pageNumber - right.pageNumber,
  );
}

function normalizeCoursewarePages(
  value: unknown,
): CWGlobalPageOption[] {
  if (
    !value ||
    typeof value !== "object" ||
    !Array.isArray(
      (value as { pages?: unknown }).pages,
    )
  ) {
    return [];
  }

  const result: CWGlobalPageOption[] = [];

  for (const raw of (
    value as {
      pages: unknown[];
    }
  ).pages) {
    if (!raw || typeof raw !== "object") {
      continue;
    }

    const candidate = raw as Record<string, unknown>;
    const id =
      typeof candidate.id === "string"
        ? candidate.id.trim()
        : "";
    const pageNumber =
      typeof candidate.page_number === "number"
        ? candidate.page_number
        : 0;
    const title =
      typeof candidate.title === "string"
        ? candidate.title.trim()
        : "";

    if (!id || pageNumber <= 0) {
      continue;
    }

    result.push({
      id,
      pageNumber,
      title: title || `第${pageNumber}页`,
    });
  }

  return result.sort(
    (left, right) => left.pageNumber - right.pageNumber,
  );
}

export default function CWAIReviewGlobalManualItemForm({
  sessionId,
  messageId,
  items,
  onSelectPage,
  onCreated,
}: CWAIReviewGlobalManualItemFormProps) {
  const [open, setOpen] = useState(false);
  const [scope, setScope] =
    useState<"courseware" | "pages">("courseware");
  const [pageOptions, setPageOptions] =
    useState<CWGlobalPageOption[]>([]);
  const [selectedPageIds, setSelectedPageIds] =
    useState<string[]>([]);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [candidateInstruction, setCandidateInstruction] =
    useState("");
  const [severity, setSeverity] =
    useState<CWAIReviewSeverity>("medium");
  const [dimension, setDimension] = useState("人工补充");

  const [loadingPages, setLoadingPages] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const coursewareID =
    items[0]?.courseware_id?.trim() || "";

  const fallbackPages = useMemo(
    () => buildFallbackPageOptions(items),
    [items],
  );

  const titleLength =
    countCWGlobalRunes(title.trim());
  const descriptionLength =
    countCWGlobalRunes(description.trim());
  const candidateInstructionLength =
    countCWGlobalRunes(candidateInstruction.trim());
  const dimensionLength =
    countCWGlobalRunes(dimension.trim());

  const textFieldsValid =
    titleLength > 0 &&
    titleLength <= CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES &&
    descriptionLength > 0 &&
    descriptionLength <=
      CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES &&
    candidateInstructionLength > 0 &&
    candidateInstructionLength <=
      CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES &&
    dimensionLength <=
      CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES;

  const pageSelectionValid =
    scope === "courseware" ||
    (
      selectedPageIds.length > 0 &&
      selectedPageIds.length <= CW_GLOBAL_MANUAL_ITEM_MAX_PAGES
    );

  const canSubmit =
    !submitting &&
    !!messageId &&
    textFieldsValid &&
    pageSelectionValid;

  useEffect(() => {
    setPageOptions(fallbackPages);

    if (!open || !coursewareID) {
      return;
    }

    let active = true;
    setLoadingPages(true);

    void getCourseware(coursewareID)
      .then((courseware) => {
        if (!active) {
          return;
        }

        const currentPages =
          normalizeCoursewarePages(courseware);

        if (currentPages.length > 0) {
          setPageOptions(currentPages);
        }
      })
      .catch((cause) => {
        if (!active) {
          return;
        }

        if (fallbackPages.length === 0) {
          setError(
            cause instanceof Error
              ? cause.message
              : "读取课件页面失败",
          );
        }
      })
      .finally(() => {
        if (active) {
          setLoadingPages(false);
        }
      });

    return () => {
      active = false;
    };
  }, [
    coursewareID,
    fallbackPages,
    open,
  ]);

  useEffect(() => {
    const availableIDs = new Set(
      pageOptions.map((page) => page.id),
    );

    setSelectedPageIds((previous) =>
      previous.filter((pageID) =>
        availableIDs.has(pageID),
      ),
    );
  }, [pageOptions]);

  const handleTogglePage = (
    pageID: string,
    checked: boolean,
  ) => {
    if (!checked) {
      setSelectedPageIds((previous) =>
        previous.filter((value) => value !== pageID),
      );
      setError("");
      return;
    }

    if (
      selectedPageIds.length >=
      CW_GLOBAL_MANUAL_ITEM_MAX_PAGES
    ) {
      setError(
        `一次最多关联${CW_GLOBAL_MANUAL_ITEM_MAX_PAGES}个页面`,
      );
      return;
    }

    setSelectedPageIds((previous) =>
      Array.from(new Set([...previous, pageID])),
    );
    setError("");
  };

  const handleSubmit = async () => {
    const normalizedTitle = title.trim();
    const normalizedDescription = description.trim();
    const normalizedInstruction =
      candidateInstruction.trim();
    const normalizedDimension =
      dimension.trim() || "人工补充";

    if (!messageId) {
      setError("请先完成至少一轮全局讨论，再人工新增问题");
      return;
    }

    if (
      !normalizedTitle ||
      !normalizedDescription ||
      !normalizedInstruction
    ) {
      setError("请完整填写标题、问题描述和候选修改指令");
      return;
    }

    if (
      countCWGlobalRunes(normalizedTitle) >
      CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES
    ) {
      setError(
        `问题标题不能超过${CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES}字`,
      );
      return;
    }

    if (
      countCWGlobalRunes(normalizedDescription) >
      CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES
    ) {
      setError(
        `问题描述不能超过${CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES}字`,
      );
      return;
    }

    if (
      countCWGlobalRunes(normalizedInstruction) >
      CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES
    ) {
      setError(
        `候选修改指令不能超过${CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES}字`,
      );
      return;
    }

    if (
      countCWGlobalRunes(normalizedDimension) >
      CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES
    ) {
      setError(
        `问题维度不能超过${CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES}字`,
      );
      return;
    }

    if (
      selectedPageIds.length >
      CW_GLOBAL_MANUAL_ITEM_MAX_PAGES
    ) {
      setError(
        `一次最多关联${CW_GLOBAL_MANUAL_ITEM_MAX_PAGES}个页面`,
      );
      return;
    }

    if (scope === "pages" && selectedPageIds.length === 0) {
      setError("请选择至少一个关联页面，或改为整课问题");
      return;
    }

    if (submitting) {
      return;
    }

    setSubmitting(true);
    setError("");
    setMessage("");

    try {
      const result =
        await createCWAIReviewGlobalManualItems(
          sessionId,
          {
            message_id: messageId,
            title: normalizedTitle,
            description: normalizedDescription,
            candidate_instruction: normalizedInstruction,
            severity,
            dimension: normalizedDimension,
            page_ids:
              scope === "pages"
                ? selectedPageIds
                : [],
          },
        );

      const createdItems = result.items || [];

      if (createdItems.length === 0) {
        throw new Error("后端未返回已创建的整改项");
      }

      onCreated(createdItems);

      setTitle("");
      setDescription("");
      setCandidateInstruction("");
      setSeverity("medium");
      setDimension("人工补充");
      setScope("courseware");
      setSelectedPageIds([]);
      setMessage(
        result.message ||
          `已创建${createdItems.length}条整改项，仍需逐条独立确认。`,
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "人工新增整改项失败",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      style={{
        marginTop: "10px",
        padding: "10px",
        borderRadius: "8px",
        border: `1px solid ${C.border}`,
        background: C.card,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "8px",
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              color: C.text,
              fontSize: "11px",
              fontWeight: 700,
            }}
          >
            人工补充新问题
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            可创建整课问题，或按选中页面拆分为多条页级问题。
            候选指令创建后仍需逐条独立确认。
          </div>
        </div>

        <button
          type="button"
          onClick={() => {
            setOpen((previous) => !previous);
            setError("");
            setMessage("");
          }}
          style={{
            padding: "5px 9px",
            borderRadius: "6px",
            border: `1px solid ${C.primary}`,
            background: C.primarySoft,
            color: C.primary,
            fontSize: "10px",
            fontWeight: 700,
            cursor: "pointer",
          }}
        >
          {open ? "收起表单" : "新增问题"}
        </button>
      </div>

      {open && (
        <div
          style={{
            marginTop: "9px",
            paddingTop: "9px",
            borderTop: `1px solid ${C.border}`,
          }}
        >
          {!messageId && (
            <div
              style={{
                padding: "7px 8px",
                borderRadius: "7px",
                background: C.warningSoft,
                color: C.warning,
                fontSize: "9px",
                lineHeight: 1.5,
              }}
            >
              人工新增必须绑定可信全局讨论来源。请先选择至少两条问题完成一轮讨论。
            </div>
          )}

          <input
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            maxLength={CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES}
            disabled={submitting}
            placeholder="问题标题"
            style={inputStyle}
          />
          <InputLimitHint
            current={titleLength}
            max={CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES}
          />

          <textarea
            value={description}
            onChange={(event) =>
              setDescription(event.target.value)
            }
            rows={3}
            maxLength={
              CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES
            }
            disabled={submitting}
            placeholder="描述用户实际发现的问题、影响范围和人工判断依据"
            style={textareaStyle}
          />
          <InputLimitHint
            current={descriptionLength}
            max={CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES}
          />

          <textarea
            value={candidateInstruction}
            onChange={(event) =>
              setCandidateInstruction(event.target.value)
            }
            rows={3}
            maxLength={
              CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES
            }
            disabled={submitting}
            placeholder="候选修改指令：明确修改对象、目标、保留内容和验收标准"
            style={textareaStyle}
          />
          <InputLimitHint
            current={candidateInstructionLength}
            max={
              CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES
            }
          />

          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 1fr",
              gap: "7px",
              marginTop: "7px",
            }}
          >
            <select
              value={severity}
              onChange={(event) =>
                setSeverity(
                  event.target.value as CWAIReviewSeverity,
                )
              }
              disabled={submitting}
              style={inputStyle}
            >
              <option value="critical">严重</option>
              <option value="high">高风险</option>
              <option value="medium">中风险</option>
              <option value="low">低风险</option>
              <option value="info">提示</option>
            </select>

            <div>
              <input
                value={dimension}
                onChange={(event) =>
                  setDimension(event.target.value)
                }
                maxLength={
                  CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES
                }
                disabled={submitting}
                placeholder="问题维度"
                style={inputStyle}
              />
              <InputLimitHint
                current={dimensionLength}
                max={
                  CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES
                }
              />
            </div>
          </div>

          <div
            style={{
              display: "flex",
              gap: "10px",
              marginTop: "8px",
              color: C.textSec,
              fontSize: "10px",
            }}
          >
            <label style={{ cursor: "pointer" }}>
              <input
                type="radio"
                checked={scope === "courseware"}
                onChange={() => setScope("courseware")}
              />{" "}
              整课问题
            </label>

            <label style={{ cursor: "pointer" }}>
              <input
                type="radio"
                checked={scope === "pages"}
                onChange={() => setScope("pages")}
              />{" "}
              关联页面
            </label>
          </div>

          {scope === "pages" && (
            <div
              style={{
                maxHeight: "190px",
                overflowY: "auto",
                marginTop: "7px",
                padding: "6px",
                borderRadius: "7px",
                border: `1px solid ${C.border}`,
                background: C.bg,
              }}
            >
              <div
                style={{
                  marginBottom: "6px",
                  color:
                    selectedPageIds.length >
                    CW_GLOBAL_MANUAL_ITEM_MAX_PAGES
                      ? C.danger
                      : C.textMuted,
                  fontSize: "9px",
                  fontWeight: 600,
                }}
              >
                已选择 {selectedPageIds.length}/
                {CW_GLOBAL_MANUAL_ITEM_MAX_PAGES} 个页面
              </div>

              {loadingPages && (
                <div
                  style={{
                    color: C.textMuted,
                    fontSize: "9px",
                  }}
                >
                  正在读取当前课件页面…
                </div>
              )}

              {!loadingPages && pageOptions.length === 0 && (
                <div
                  style={{
                    color: C.warning,
                    fontSize: "9px",
                    lineHeight: 1.5,
                  }}
                >
                  暂未读取到可关联页面，请刷新后重试或创建整课问题。
                </div>
              )}

              {pageOptions.map((page) => {
                const selected =
                  selectedPageIds.includes(page.id);
                const pageSelectionFull =
                  selectedPageIds.length >=
                  CW_GLOBAL_MANUAL_ITEM_MAX_PAGES;

                return (
                  <label
                    key={page.id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "7px",
                      marginBottom: "5px",
                      padding: "6px",
                      borderRadius: "6px",
                      background: C.card,
                      cursor:
                        pageSelectionFull && !selected
                          ? "not-allowed"
                          : "pointer",
                      opacity:
                        pageSelectionFull && !selected
                          ? 0.65
                          : 1,
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={selected}
                      disabled={pageSelectionFull && !selected}
                      onChange={(event) =>
                        handleTogglePage(
                          page.id,
                          event.target.checked,
                        )
                      }
                    />

                    <button
                      type="button"
                      onClick={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        onSelectPage(page.pageNumber);
                      }}
                      style={{
                        ...cwGlobalPageButtonStyle,
                        color: C.primary,
                      }}
                    >
                      P{page.pageNumber}
                    </button>

                    <span
                      style={{
                        minWidth: 0,
                        flex: 1,
                        color: C.textSec,
                        fontSize: "9px",
                      }}
                    >
                      {page.title}
                    </span>
                  </label>
                );
              })}
            </div>
          )}

          <button
            type="button"
            onClick={() => void handleSubmit()}
            disabled={!canSubmit}
            style={{
              width: "100%",
              marginTop: "8px",
              padding: "7px",
              borderRadius: "7px",
              border: "none",
              background: canSubmit ? C.primary : "#CBD5E1",
              color: "#fff",
              fontSize: "10px",
              fontWeight: 700,
              cursor: canSubmit ? "pointer" : "not-allowed",
            }}
          >
            {submitting
              ? "正在创建整改项…"
              : "确认人工新增"}
          </button>

          {error && (
            <Feedback
              type="error"
              content={error}
            />
          )}

          {message && (
            <Feedback
              type="success"
              content={message}
            />
          )}
        </div>
      )}
    </div>
  );
}

function InputLimitHint({
  current,
  max,
}: {
  current: number;
  max: number;
}) {
  const exceeded = current > max;

  return (
    <div
      style={{
        marginTop: "3px",
        color: exceeded ? C.danger : C.textMuted,
        fontSize: "8px",
        lineHeight: 1.4,
        textAlign: "right",
      }}
    >
      {current}/{max}
    </div>
  );
}

function Feedback({
  type,
  content,
}: {
  type: "success" | "error";
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "7px",
        padding: "7px 8px",
        borderRadius: "7px",
        background:
          type === "success"
            ? C.successSoft
            : C.dangerSoft,
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "9px",
        lineHeight: 1.5,
      }}
    >
      {content}
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  width: "100%",
  boxSizing: "border-box",
  marginTop: "7px",
  padding: "8px 9px",
  borderRadius: "7px",
  border: `1px solid ${C.border}`,
  background: "#fff",
  color: C.text,
  fontFamily: "inherit",
  fontSize: "10px",
  outline: "none",
};

const textareaStyle: React.CSSProperties = {
  ...inputStyle,
  resize: "vertical",
  lineHeight: 1.6,
};
