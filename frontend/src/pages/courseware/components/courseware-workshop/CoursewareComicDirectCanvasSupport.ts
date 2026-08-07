/**
 * CoursewareComicDirectCanvasSupport.ts
 *
 * 漫画直接编辑画布公共支持模块：
 *   - 统一声明画布、元素和问题卡组件参数；
 *   - 定义拖动、缩放和气泡尾巴控制点交互状态；
 *   - 计算缩放后的归一化坐标；
 *   - 把项目画幅转换为CSS aspect-ratio；
 *   - 提供画布基础展示样式。
 *
 * 本文件不渲染JSX，也不修改覆盖层文档。
 */
import type { CSSProperties, KeyboardEvent, PointerEvent, } from 'react';
import type { CoursewareComicAspectRatio, CoursewareComicOverlayDocument, CoursewareComicOverlayElement, CoursewareComicPanel, CoursewareComicTextStyle, } from '@/api/coursewares';
import type { CoursewareComicTailLayoutPatch, } from './coursewareComicTailEditing';
export type ResizeCorner = 'nw' | 'ne' | 'sw' | 'se';
export type PointerInteractionMode = 'drag' | 'resize' | 'tail_origin' | 'tail_target';
export interface LayoutPatch extends CoursewareComicTailLayoutPatch {
    x?: number;
    y?: number;
    width?: number;
    height?: number;
}
export interface PointerInteraction {
    elementID: string;
    mode: PointerInteractionMode;
    corner?: ResizeCorner;
    pointerID: number;
    startClientX: number;
    startClientY: number;
    startX: number;
    startY: number;
    startWidth: number;
    startHeight: number;
}
export interface CoursewareComicDirectCanvasProps {
    panel: CoursewareComicPanel;
    aspectRatio: CoursewareComicAspectRatio;
    overlayDocument: CoursewareComicOverlayDocument;
    disabled: boolean;
    selectedElementID: string;
    editingElementID: string;
    onSelectElement: (elementID: string) => void;
    onBeginEditing: (elementID: string) => void;
    onEndEditing: () => void;
    onContentChange: (elementID: string, value: string) => void;
    onQuestionTextChange: (elementID: string, field: 'question' | 'explanation', value: string) => void;
    onQuestionOptionsChange: (elementID: string, value: string) => void;
    onQuestionAnswerChange: (elementID: string, value: number) => void;
    onLayoutChange: (elementID: string, patch: LayoutPatch) => void;
    onTextStyleChange: (elementID: string, patch: Partial<CoursewareComicTextStyle>) => void;
    onCycleStyle: (elementID: string, styleID?: string) => void;
    onAutoFit: (elementID: string) => void;
    onDuplicate: (elementID: string) => void;
    onDelete: (elementID: string) => void;
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => boolean;
}
export interface CoursewareComicCanvasElementProps {
    element: CoursewareComicOverlayElement;
    selected: boolean;
    editing: boolean;
    disabled: boolean;
    canDelete: boolean;
    onSelectElement: (elementID: string) => void;
    onBeginEditing: (elementID: string) => void;
    onEndEditing: () => void;
    onContentChange: (elementID: string, value: string) => void;
    onLayoutChange: (elementID: string, patch: LayoutPatch) => void;
    onTextStyleChange: (elementID: string, patch: Partial<CoursewareComicTextStyle>) => void;
    onCycleStyle: (elementID: string, styleID?: string) => void;
    onAutoFit: (elementID: string) => void;
    onDuplicate: (elementID: string) => void;
    onDelete: (elementID: string) => void;
    onBeginPointerInteraction: (event: PointerEvent<HTMLDivElement>, element: CoursewareComicOverlayElement, mode: PointerInteractionMode, corner?: ResizeCorner) => void;
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => boolean;
}
export interface CoursewareComicQuestionPopoverProps {
    element: CoursewareComicOverlayElement;
    disabled: boolean;
    onClose: () => void;
    onTextChange: (elementID: string, field: 'question' | 'explanation', value: string) => void;
    onOptionsChange: (elementID: string, value: string) => void;
    onAnswerChange: (elementID: string, value: number) => void;
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => boolean;
}
export function clampCanvasValue(value: number, minimum: number, maximum: number): number {
    return Math.min(maximum, Math.max(minimum, value));
}
/**
 * resolveResizePatch
 *
 * 根据四角手柄和指针移动量计算新位置与尺寸。
 */
export function resolveResizePatch(interaction: PointerInteraction, deltaX: number, deltaY: number): LayoutPatch {
    const minimumWidth = 0.07;
    const minimumHeight = 0.03;
    let x = interaction.startX;
    let y = interaction.startY;
    let width = interaction.startWidth;
    let height = interaction.startHeight;
    if (interaction.corner ===
        'se' ||
        interaction.corner ===
            'ne') {
        width =
            clampCanvasValue(interaction.startWidth +
                deltaX, minimumWidth, 1 -
                interaction.startX);
    }
    if (interaction.corner ===
        'sw' ||
        interaction.corner ===
            'nw') {
        x =
            clampCanvasValue(interaction.startX +
                deltaX, 0, interaction.startX +
                interaction.startWidth -
                minimumWidth);
        width =
            interaction.startWidth +
                (interaction.startX -
                    x);
    }
    if (interaction.corner ===
        'se' ||
        interaction.corner ===
            'sw') {
        height =
            clampCanvasValue(interaction.startHeight +
                deltaY, minimumHeight, 1 -
                interaction.startY);
    }
    if (interaction.corner ===
        'ne' ||
        interaction.corner ===
            'nw') {
        y =
            clampCanvasValue(interaction.startY +
                deltaY, 0, interaction.startY +
                interaction.startHeight -
                minimumHeight);
        height =
            interaction.startHeight +
                (interaction.startY -
                    y);
    }
    return {
        x,
        y,
        width,
        height,
    };
}
export function resolveCoursewareComicCanvasAspectRatio(value: CoursewareComicAspectRatio): string {
    switch (value) {
        case '4:3':
            return '4 / 3';
        case '1:1':
            return '1 / 1';
        case '3:4':
            return '3 / 4';
        case '9:16':
            return '9 / 16';
        default:
            return '16 / 9';
    }
}
export const directCanvasWorkspaceStyle: CSSProperties = {
    width: '100%',
};
export const directCanvasInstructionStyle: CSSProperties = {
    marginBottom: 6,
    color: '#64748B',
    fontSize: 9,
    textAlign: 'center',
};
export const directCanvasStyle: CSSProperties = {
    position: 'relative',
    width: '100%',
    overflow: 'hidden',
    borderRadius: 10,
    border: '1px solid #CBD5E1',
    background: 'linear-gradient(135deg,#EEF2FF,#F8FAFC)',
    boxShadow: '0 10px 28px rgba(15,23,42,0.10)',
    touchAction: 'none',
    userSelect: 'none',
};
export const directCanvasImageStyle: CSSProperties = {
    position: 'absolute',
    inset: 0,
    width: '100%',
    height: '100%',
    objectFit: 'cover',
    display: 'block',
    pointerEvents: 'none',
};
export const directCanvasEmptyStyle: CSSProperties = {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#94A3B8',
    textAlign: 'center',
    fontSize: 11,
    pointerEvents: 'none',
};
export const directCanvasEmptyIconStyle: CSSProperties = {
    marginBottom: 5,
    fontSize: 26,
};
export const directCanvasSelectedHintStyle: CSSProperties = {
    marginTop: 6,
    color: '#64748B',
    fontSize: 9,
    textAlign: 'center',
};
