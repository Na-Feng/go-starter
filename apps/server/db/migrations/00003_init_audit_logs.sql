-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id          TEXT PRIMARY KEY,
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    detail      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip          TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs (actor_id);

INSERT INTO permissions (id, code, name) VALUES
    ('perm_iam_audit_read', 'iam:audit:read', '查看操作审计日志')
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
