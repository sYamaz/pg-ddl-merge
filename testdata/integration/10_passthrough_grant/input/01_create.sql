-- このテーブルはユーザー情報を管理する
CREATE TABLE users (
    id INT,
    name TEXT NOT NULL
);

/* ロール権限付与 */
GRANT SELECT ON users TO readonly_role;
GRANT INSERT, UPDATE ON users TO app_role;

-- インデックス
CREATE INDEX idx_users_name ON users (name);
