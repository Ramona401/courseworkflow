-- v110: 学校管理员功能
-- 目标：支持学校管理员查看本校全部教师（含未分配教研组教师）
-- 方案A：在 users 表新增 school_id 字段（可空，向后兼容）

ALTER TABLE users
ADD COLUMN IF NOT EXISTS school_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_school_id ON users(school_id);

COMMENT ON COLUMN users.school_id IS '用户所属学校ID（学校管理员创建教师时写入）';
