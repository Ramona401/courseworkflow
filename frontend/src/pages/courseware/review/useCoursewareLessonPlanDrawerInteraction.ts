/**
 * useCoursewareLessonPlanDrawerInteraction.ts
 *
 * 来源教案对照抽屉的交互状态 Hook。
 *
 * 负责：
 *   - 响应式宽度、拖拽、键盘调宽和本地宽度记忆；
 *   - 打开后的焦点进入、移动端焦点环、Escape 关闭与焦点恢复；
 *   - 相关章节自动滚动和教师主动重新定位；
 *   - 拖拽期间 document.body 全局样式的可靠恢复。
 *
 * 本模块不读取来源教案接口，也不修改教案、课件或整改项状态。
 */

import {
  type KeyboardEvent,
  type PointerEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import type {
  LessonDocumentSection,
} from "@/pages/lesson-plans/workshop/components/lessonDocumentStructure";

interface UseCoursewareLessonPlanDrawerInteractionOptions {
  open: boolean;
  storageKey: string;
  openerElement?: HTMLElement | null;
  onClose: () => void;
  hasContent: boolean;
  matchedSection?: LessonDocumentSection;
  focusContextKey: string;
}

interface ManualSectionFocus {
  contextKey: string;
  sectionID: string;
}

const DEFAULT_WIDTH = 620;
const MINIMUM_WIDTH = 420;
const MAXIMUM_WIDTH = 960;
const MINIMUM_PRIMARY_WIDTH = 520;
const DIVIDER_WIDTH = 10;
const KEYBOARD_STEP = 24;
const KEYBOARD_LARGE_STEP = 64;
const COMPACT_BREAKPOINT = 1024;
const SECTION_SCROLL_DELAY_MS = 30;

export function useCoursewareLessonPlanDrawerInteraction({
  open,
  storageKey,
  openerElement,
  onClose,
  hasContent,
  matchedSection,
  focusContextKey,
}: UseCoursewareLessonPlanDrawerInteractionOptions) {
  const [fullScreen, setFullScreen] = useState(false);
  const [windowWidth, setWindowWidth] = useState(readWindowWidth);
  const [panelWidth, setPanelWidth] = useState(() => readStoredWidth(storageKey));
  const [dragging, setDragging] = useState(false);
  const [hoveringDivider, setHoveringDivider] = useState(false);
  const [manualFocus, setManualFocus] = useState<ManualSectionFocus>({
    contextKey: "",
    sectionID: "",
  });

  const panelRef = useRef<HTMLElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const openerRef = useRef<HTMLElement | null>(null);
  const sectionRefs = useRef<Map<string, HTMLElement>>(new Map());
  const startXRef = useRef(0);
  const startWidthRef = useRef(panelWidth);
  const bodyCursorRef = useRef("");
  const bodyUserSelectRef = useRef("");
  const scrollTimerRef = useRef<number | null>(null);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  const compact = windowWidth < COMPACT_BREAKPOINT;
  const maximumWidth = Math.max(
    MINIMUM_WIDTH,
    Math.min(
      MAXIMUM_WIDTH,
      windowWidth - MINIMUM_PRIMARY_WIDTH - DIVIDER_WIDTH,
    ),
  );
  const normalizedPanelWidth = clamp(
    panelWidth,
    MINIMUM_WIDTH,
    maximumWidth,
  );
  const matchedSectionID = matchedSection?.id || "";
  const focusedSectionID =
    manualFocus.contextKey === focusContextKey
      ? manualFocus.sectionID
      : matchedSectionID;

  const restoreBodyInteraction = useCallback(() => {
    document.body.style.cursor = bodyCursorRef.current;
    document.body.style.userSelect = bodyUserSelectRef.current;
  }, []);

  const closeDrawer = useCallback(() => {
    setFullScreen(false);
    setDragging(false);
    restoreBodyInteraction();
    onCloseRef.current();

    window.setTimeout(() => {
      openerRef.current?.focus();
    }, 0);
  }, [restoreBodyInteraction]);

  useEffect(() => {
    if (!open) return;

    openerRef.current =
      openerElement ||
      (document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null);

    const focusTimer = window.setTimeout(() => {
      closeButtonRef.current?.focus();
    }, 0);

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        closeDrawer();
        return;
      }

      if (event.key !== "Tab" || (!compact && !fullScreen)) return;

      const focusable = panelRef.current?.querySelectorAll<HTMLElement>(
        "button:not([disabled]),[href],[tabindex]:not([tabindex='-1'])",
      );

      if (!focusable || focusable.length === 0) return;

      const first = focusable[0];
      const last = focusable[focusable.length - 1];

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      window.clearTimeout(focusTimer);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [closeDrawer, compact, fullScreen, open, openerElement]);

  useEffect(() => {
    const handleResize = () => {
      setWindowWidth(window.innerWidth);
    };

    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      restoreBodyInteraction();
    };
  }, [restoreBodyInteraction]);

  useEffect(() => {
    if (compact || fullScreen) return;

    try {
      localStorage.setItem(storageKey, String(Math.round(normalizedPanelWidth)));
    } catch {
      // 浏览器禁用本地存储时，不影响当前抽屉的阅读与调宽。
    }
  }, [compact, fullScreen, normalizedPanelWidth, storageKey]);

  const updateWidth = useCallback(
    (candidate: number) => {
      setPanelWidth(clamp(candidate, MINIMUM_WIDTH, maximumWidth));
    },
    [maximumWidth],
  );

  const beginResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      if (compact || fullScreen) return;

      event.preventDefault();
      event.currentTarget.setPointerCapture(event.pointerId);
      startXRef.current = event.clientX;
      startWidthRef.current = normalizedPanelWidth;
      bodyCursorRef.current = document.body.style.cursor;
      bodyUserSelectRef.current = document.body.style.userSelect;
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      setDragging(true);
    },
    [compact, fullScreen, normalizedPanelWidth],
  );

  const continueResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      if (!dragging || compact || fullScreen) return;

      event.preventDefault();
      updateWidth(
        startWidthRef.current + startXRef.current - event.clientX,
      );
    },
    [compact, dragging, fullScreen, updateWidth],
  );

  const finishResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      if (!dragging) return;

      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId);
      }

      setDragging(false);
      restoreBodyInteraction();
    },
    [dragging, restoreBodyInteraction],
  );

  const handleDividerKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      if (compact || fullScreen) return;

      const step = event.shiftKey ? KEYBOARD_LARGE_STEP : KEYBOARD_STEP;

      if (event.key === "ArrowLeft") {
        event.preventDefault();
        updateWidth(normalizedPanelWidth + step);
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        updateWidth(normalizedPanelWidth - step);
      } else if (event.key === "Home" || event.key === "Enter") {
        event.preventDefault();
        updateWidth(DEFAULT_WIDTH);
      }
    },
    [compact, fullScreen, normalizedPanelWidth, updateWidth],
  );

  const scrollSectionIntoView = useCallback((sectionID: string) => {
    sectionRefs.current.get(sectionID)?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, []);

  const scheduleSectionScroll = useCallback(
    (sectionID: string) => {
      if (!sectionID) return;

      if (scrollTimerRef.current !== null) {
        window.clearTimeout(scrollTimerRef.current);
      }

      scrollTimerRef.current = window.setTimeout(() => {
        scrollSectionIntoView(sectionID);
        scrollTimerRef.current = null;
      }, SECTION_SCROLL_DELAY_MS);
    },
    [scrollSectionIntoView],
  );

  useEffect(() => {
    if (!open || !hasContent || !matchedSectionID) return;
    scheduleSectionScroll(matchedSectionID);
  }, [hasContent, matchedSectionID, open, scheduleSectionScroll]);

  useEffect(
    () => () => {
      if (scrollTimerRef.current !== null) {
        window.clearTimeout(scrollTimerRef.current);
      }
    },
    [],
  );

  const scrollToSection = useCallback(
    (section: LessonDocumentSection | undefined) => {
      if (!section) return;

      setManualFocus({
        contextKey: focusContextKey,
        sectionID: section.id,
      });
      scheduleSectionScroll(section.id);
    },
    [focusContextKey, scheduleSectionScroll],
  );

  return {
    fullScreen,
    setFullScreen,
    compact,
    panelWidth: normalizedPanelWidth,
    maximumWidth,
    updateWidth,
    dragging,
    hoveringDivider,
    setHoveringDivider,
    panelRef,
    closeButtonRef,
    sectionRefs,
    focusedSectionID,
    scrollToSection,
    closeDrawer,
    beginResize,
    continueResize,
    finishResize,
    handleDividerKeyDown,
    restoreBodyInteraction,
    constants: {
      defaultWidth: DEFAULT_WIDTH,
      minimumWidth: MINIMUM_WIDTH,
      minimumPrimaryWidth: MINIMUM_PRIMARY_WIDTH,
      dividerWidth: DIVIDER_WIDTH,
    },
  };
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}

function readWindowWidth(): number {
  return typeof window === "undefined" ? COMPACT_BREAKPOINT : window.innerWidth;
}

function readStoredWidth(storageKey: string): number {
  if (typeof window === "undefined") return DEFAULT_WIDTH;

  try {
    const raw = localStorage.getItem(storageKey);
    const parsed = raw ? Number.parseInt(raw, 10) : Number.NaN;

    if (Number.isFinite(parsed)) {
      return clamp(parsed, MINIMUM_WIDTH, MAXIMUM_WIDTH);
    }
  } catch {
    // 浏览器禁用本地存储时使用默认宽度。
  }

  return DEFAULT_WIDTH;
}
