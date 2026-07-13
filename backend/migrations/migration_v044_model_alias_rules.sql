-- migration_v044_model_alias_rules.sql
-- 批三-2：模型别名映射表
--
-- 用途：把真实厂商模型名（如 anthropic/claude-sonnet-4-5、qwen-max）映射为业务可读别名
--       （如「智学大模型·标准版」），供老师侧渲染时替换，避免暴露真实厂商/模型。
--
-- 匹配规则（批三-3 老师侧将据此查询）：
--   1) 先找 match_type='exact' 且 pattern 完全等于模型名的启用规则；
--   2) 再找 match_type='prefix' 且模型名以 pattern 开头的启用规则，
--      按 priority 降序、pattern 长度降序取第一条（最长最优先前缀）；
--   3) 都没命中 → 用 ai_configs 的 model_alias_fallback 兜底别名（默认「智学大模型」）。
--
-- 注：本表只存映射规则；兜底别名存在 ai_configs(model_alias_fallback)，admin 可改。

CREATE TABLE IF NOT EXISTS model_alias_rules (
    id          uuid                     NOT NULL DEFAULT gen_random_uuid(),
    match_type  varchar(16)              NOT NULL DEFAULT 'prefix',  -- exact / prefix
    pattern     varchar(256)             NOT NULL,                   -- 模型名或前缀
    alias       varchar(128)             NOT NULL,                   -- 业务别名
    priority    integer                  NOT NULL DEFAULT 0,         -- 同时命中时大者优先
    enabled     boolean                  NOT NULL DEFAULT true,
    note        text                     NOT NULL DEFAULT '',        -- 备注（可选）
    created_by  uuid,                                                -- 创建人（可空）
    created_at  timestamp with time zone NOT NULL DEFAULT now(),
    updated_at  timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT model_alias_rules_pkey PRIMARY KEY (id),
    CONSTRAINT model_alias_rules_match_type_chk CHECK (match_type IN ('exact','prefix')),
    CONSTRAINT model_alias_rules_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- 唯一约束：同一 match_type + pattern 不重复（避免重复规则）
CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_alias_type_pattern
    ON model_alias_rules (match_type, pattern);

-- 查询索引：老师侧高频按 enabled + match_type + priority 检索
CREATE INDEX IF NOT EXISTS idx_model_alias_lookup
    ON model_alias_rules (enabled, match_type, priority DESC)
    WHERE enabled = true;

COMMENT ON TABLE model_alias_rules IS '批三-2：真实模型名→业务别名映射规则（exact精确/prefix前缀，精确优先）';
