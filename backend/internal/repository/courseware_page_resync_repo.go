package repository

// courseware_page_resync_repo.go — 课件页号安全重排（② 导航栏分母烧死专题配套）
//
// 背景：courseware_pages 有 UNIQUE(courseware_id, page_number) 约束。
//   直接逐页 UPDATE page_number 重排会中途撞号（例如把某页改成 5 时表里仍有另一页是 5）。
//   本文件提供两阶段避撞重排，置于单个事务内：
//     阶段一：把本课件所有页 page_number 偏移到高位（+cwPageNumOffset），挪出 1..N 撞号区；
//     阶段二：按给定的目标顺序 orderedPageIDs，逐个落 1..N。
//   事务保证要么全成、要么回滚，绝不留下半重排的脏状态。
//
// 复用方：
//   - ReorderCoursewarePages（拖拽重排，修正其原有的逐页裸 UPDATE 撞号隐患）；
//   - service 层 ResyncCWPageNumbers（增删页后按升序补洞重排）。

import (
        "context"
        "fmt"
        "time"

        "tedna/internal/database"
)

// cwPageNumOffset 阶段一偏移量：须大于任何课件实际页数，确保偏移后不与目标 1..N 区间相撞。
const cwPageNumOffset = 100000

// ResequenceCoursewarePagesByIDs 在单个事务内按 orderedPageIDs 顺序把页号安全重排为 1..N。
// orderedPageIDs 为目标顺序的页ID列表（下标 0 → page_number 1，以此类推）。
// 两阶段避撞：先整体偏移到高位，再逐个落目标值，规避 UNIQUE(courseware_id,page_number) 撞号。
func ResequenceCoursewarePagesByIDs(ctx context.Context, coursewareID string, orderedPageIDs []string) error {
        if len(orderedPageIDs) == 0 {
                return nil
        }
        tx, err := database.DB.Begin(ctx)
        if err != nil {
                return fmt.Errorf("开启事务失败: %w", err)
        }
        defer func() { _ = tx.Rollback(ctx) }()

        now := time.Now()

        // 阶段一：本课件所有页整体偏移到高位，挪出 1..N 撞号区
        // （用 page_number = page_number + offset，对全课件页生效，含不在 orderedPageIDs 里的页也一并挪开）
        if _, err = tx.Exec(ctx,
                `UPDATE courseware_pages SET page_number = page_number + $1, updated_at = $2
WHERE courseware_id = $3`,
                cwPageNumOffset, now, coursewareID); err != nil {
                return fmt.Errorf("页号重排阶段一(偏移)失败: %w", err)
        }

        // 阶段二：按目标顺序逐个落 1..N
        for i, pid := range orderedPageIDs {
                if _, err = tx.Exec(ctx,
                        `UPDATE courseware_pages SET page_number = $1, updated_at = $2
WHERE id = $3 AND courseware_id = $4`,
                        i+1, now, pid, coursewareID); err != nil {
                        return fmt.Errorf("页号重排阶段二(落位 id=%s)失败: %w", pid, err)
                }
        }

        if err = tx.Commit(ctx); err != nil {
                return fmt.Errorf("页号重排提交事务失败: %w", err)
        }
        return nil
}
