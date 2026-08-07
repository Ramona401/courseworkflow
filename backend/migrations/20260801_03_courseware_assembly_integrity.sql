BEGIN;

-- ============================================================================
-- 课件自动装配运行归属与生命周期完整性加固
-- ============================================================================
--
-- 第一批迁移已经建立：
--   - coursewares当前装配版本和活动运行指针；
--   - courseware_assembly_runs不可变运行记录；
--   - 页面装配版本和布局验收状态。
--
-- 本迁移补齐数据库自身必须保证的两个事实：
--
--   1. coursewares.active_assembly_run_id所引用的运行必须属于本课件；
--      不能只保证“存在这个运行ID”，还必须保证run.courseware_id=coursewares.id。
--
--   2. 装配状态、活动运行指针和finished_at必须一致；
--      禁止出现running却没有活动运行、终态却仍指向活动运行、
--      running却已经finished或终态没有finished_at等矛盾数据。
--
-- 这些约束是后续旧任务防覆盖和浏览器质量验收门能够可信运行的前提。
-- ============================================================================

-- PostgreSQL复合外键要求被引用列组具有唯一约束。
-- id本身虽已是主键，但仍显式声明(id, courseware_id)唯一，
-- 使下方复合外键可以同时验证运行ID和课件归属。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_courseware_assembly_runs_id_courseware'
          AND conrelid = 'courseware_assembly_runs'::regclass
    ) THEN
        ALTER TABLE courseware_assembly_runs
            ADD CONSTRAINT uq_courseware_assembly_runs_id_courseware
            UNIQUE (id, courseware_id);
    END IF;
END
$$;

-- 替换第一批中的单列外键。
--
-- 新外键：
--   (coursewares.active_assembly_run_id, coursewares.id)
--       →
--   (courseware_assembly_runs.id, courseware_assembly_runs.courseware_id)
--
-- 约束设为延迟检查：
-- 删除课件时，courseware_assembly_runs会因原有courseware_id外键级联删除；
-- 到事务结束时课件本身和运行记录均已删除，不形成循环删除阻塞。
ALTER TABLE coursewares
    DROP CONSTRAINT IF EXISTS fk_coursewares_active_assembly_run;

ALTER TABLE coursewares
    ADD CONSTRAINT fk_coursewares_active_assembly_run
    FOREIGN KEY (active_assembly_run_id, id)
    REFERENCES courseware_assembly_runs(id, courseware_id)
    DEFERRABLE INITIALLY DEFERRED;

-- 课件当前状态与活动运行指针必须一致。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_coursewares_assembly_active_run_consistency'
          AND conrelid = 'coursewares'::regclass
    ) THEN
        ALTER TABLE coursewares
            ADD CONSTRAINT chk_coursewares_assembly_active_run_consistency
            CHECK (
                (
                    assembly_status IN (
                        'running',
                        'cancel_requested'
                    )
                    AND active_assembly_run_id IS NOT NULL
                )
                OR
                (
                    assembly_status NOT IN (
                        'running',
                        'cancel_requested'
                    )
                    AND active_assembly_run_id IS NULL
                )
            );
    END IF;
END
$$;

-- 单次运行状态与finished_at必须一致。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_courseware_assembly_runs_finish_consistency'
          AND conrelid = 'courseware_assembly_runs'::regclass
    ) THEN
        ALTER TABLE courseware_assembly_runs
            ADD CONSTRAINT chk_courseware_assembly_runs_finish_consistency
            CHECK (
                (
                    status IN (
                        'running',
                        'cancel_requested'
                    )
                    AND finished_at IS NULL
                )
                OR
                (
                    status IN (
                        'completed',
                        'cancelled',
                        'failed',
                        'interrupted'
                    )
                    AND finished_at IS NOT NULL
                )
            );
    END IF;
END
$$;

COMMENT ON CONSTRAINT fk_coursewares_active_assembly_run
    ON coursewares IS
    '活动装配运行必须真实属于本课件，禁止跨课件引用运行ID。';

COMMENT ON CONSTRAINT chk_coursewares_assembly_active_run_consistency
    ON coursewares IS
    'running/cancel_requested必须有活动运行；其它状态必须清空活动运行。';

COMMENT ON CONSTRAINT chk_courseware_assembly_runs_finish_consistency
    ON courseware_assembly_runs IS
    '活动运行finished_at必须为空；终态运行finished_at必须非空。';

COMMIT;
