# 実装優先度バックログ

`postgres-command-coverage.md` で部分対応（🔶）になっている項目を、
アプリケーションマイグレーションへの影響度で優先度付けしたもの。

---

## ✅ 実装済み（旧優先度：高・中）

### 1. `ALTER EXTENSION` の `UPDATE TO` 対応

`ALTER EXTENSION name UPDATE [TO version]` を `AlterObjectOptsStmt` としてパースし、
CREATE EXTENSION の直後に PostAlter として出力。`ADD MEMBER` / `DROP MEMBER` は DBA 操作のためスコープ外・パススルーのまま。

### 2. `ALTER DOMAIN` の内容変更対応

`ADD/DROP CONSTRAINT`, `SET/DROP DEFAULT`, `SET/DROP NOT NULL` などを PostAlter として CREATE DOMAIN の直後に出力。`OWNER TO` / `SET SCHEMA` は警告を出してスキップ。

### 3. `ALTER POLICY` の内容変更対応

`TO roles` / `USING (expr)` / `WITH CHECK (expr)` などの内容変更アクションを PostAlter として CREATE POLICY の直後に出力。ポリシーが schema 内に存在しない場合は末尾にパススルー。

---

## ✅ 実装済み（旧優先度：中）

### 4. `ALTER VIEW` の内容変更対応

`ALTER VIEW name ALTER COLUMN col SET DEFAULT expr` などの内容変更を `AlterObjectOptsStmt` としてパースし、
CREATE VIEW の直後に PostAlter として出力。`OWNER TO` / `SET SCHEMA` は警告を出してスキップ。

### 5. `ALTER TABLE` の `RENAME TO` / `RENAME COLUMN` 後の inline 制約追跡

- `RENAME COLUMN old TO new` 時に `Constraints` の定義文字列内の旧カラム名をワード境界マッチで置換
- `RENAME TO` 時に自己参照 `FOREIGN KEY ... REFERENCES tablename` の旧テーブル名を更新

### 6. `ALTER TRIGGER` の `RENAME TO` 以外

`DEPENDS ON EXTENSION` / `NO DEPENDS ON EXTENSION` はそのままパススルー（UnknownStmt → 末尾出力）。
`ENABLE/DISABLE` は PostgreSQL では `ALTER TABLE ... ENABLE/DISABLE TRIGGER` 構文であり、
ALTER TABLE ActionSkip として処理される。ALTER TRIGGER 固有の追加対応は不要と判断。

---

## 優先度：低

| 項目 | 理由 |
|------|------|
| `ALTER INDEX` RENAME TO 以外 | `SET TABLESPACE` 等は DBA 操作で app マイグレーションには稀 |
| `ALTER MATERIALIZED VIEW` RENAME TO 以外 | 同上 |
| `ALTER SCHEMA` RENAME TO 以外 | 同上 |
| `ALTER RULE` RENAME TO 以外 | ルール自体の使用頻度が低い |
| `REINDEX` | 運用コマンド、pass-through で十分 |
| `REFRESH MATERIALIZED VIEW` | データ操作のためパススルーで十分 |
| inline 制約の構造追跡強化（PRIMARY KEY・UNIQUE・CHECK） | verbatim 保持で現状は動作するため、DROP CONSTRAINT 対応（完了）で十分な場合が多い |
