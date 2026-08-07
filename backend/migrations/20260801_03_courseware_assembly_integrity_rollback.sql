BEGIN;

ALTER TABLE coursewares
    DROP CONSTRAINT IF EXISTS chk_coursewares_assembly_active_run_consistency;

ALTER TABLE courseware_assembly_runs
    DROP CONSTRAINT IF EXISTS chk_courseware_assembly_runs_finish_consistency;

ALTER TABLE coursewares
    DROP CONSTRAINT IF EXISTS fk_coursewares_active_assembly_run;

ALTER TABLE coursewares
    ADD CONSTRAINT fk_coursewares_active_assembly_run
    FOREIGN KEY (active_assembly_run_id)
    REFERENCES courseware_assembly_runs(id)
    ON DELETE SET NULL;

ALTER TABLE courseware_assembly_runs
    DROP CONSTRAINT IF EXISTS uq_courseware_assembly_runs_id_courseware;

COMMIT;
