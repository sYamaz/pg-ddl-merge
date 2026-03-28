# 実装優先度バックログ

部分対応（🔶）・未対応（❌）の項目について、実際のマイグレーションへの影響度で優先度を整理したもの。

---

## 優先度：高

### 1. ~~inline 制約と `DROP CONSTRAINT` の不整合~~ ✅ 対応済み

**問題（解決済み）**

`CREATE TABLE` でカラムインラインに定義した PRIMARY KEY / UNIQUE / REFERENCES は `col.InlineConstraints` に格納される。
その後 `ALTER TABLE ... DROP CONSTRAINT` で PostgreSQL が自動付与した名前（`table_pkey` 等）を指定すると
`t.Constraints`（named constraint リスト）に見つけられずエラーになっていた。

**対応内容**（2026-03-28）

- `merger/parser/parser.go`: `DROP CONSTRAINT IF EXISTS` の `IfExists` フラグを `AlterAction` に正しく伝搬
- `merger/schema/schema.go`: named constraint に見つからない場合、PostgreSQL 自動命名規則でカラムの inline 制約を検索・除去する `dropInlineConstraintByAutoName()` を追加
  - `{table}_pkey` → inline `PRIMARY KEY`
  - `{table}_{col}_key` → inline `UNIQUE`
  - `{table}_{col}_fkey` → inline `REFERENCES`
  - `{table}_{col}_check` → inline `CHECK`
- 統合テストシナリオ `22_inline_constraint_drop` を追加

---

### 2. ~~`ALTER COLUMN TYPE` で inline `REFERENCES` が型不一致になっても検知されない~~ ✅ 警告追加済み

**問題（部分対応済み）**

カラムに inline FK (`REFERENCES`) がある状態で型変更しても `InlineConstraints` はそのまま保持され、
エミット結果は valid に見えるが参照先テーブルの型と不一致なら PostgreSQL は適用を拒否する。

**対応内容**（2026-03-28）

- `merger/schema/schema.go`: `ActionAlterColumnType` で `InlineConstraints` に `REFERENCES` が含まれる場合、
  stderr に警告を出力するようにした

> **Note**: 出力 SQL の修正（REFERENCES 句の除去など）は行っていない。
> PostgreSQL 側でも型変更＋FK は失敗するケースが多く、マイグレーション設計の問題として扱う。

---

## 優先度：中

### 3. `ALTER FUNCTION` / `ALTER PROCEDURE` の `OWNER TO` / `SECURITY` 変更

アプリのマイグレーションで関数の実行権限変更（`OWNER TO`、`SECURITY DEFINER`/`INVOKER`）が含まれる場合がある。
現状は UnknownStmt でパススルーされるが、後続の DROP/CREATE と組み合わさると出力順序が意図と異なる可能性がある。

---

## 優先度：低

| 項目 | 理由 |
|------|------|
| `REINDEX` ❌ | スコープ外扱いが妥当、pass-through で十分 |
| `REFRESH MATERIALIZED VIEW` ❌ | データ操作、DDL マージ対象外 |
| `ALTER INDEX` RENAME TO 以外 | `SET TABLESPACE` 等は DBA 操作で app マイグレーションには稀 |
| `ALTER VIEW` / `ALTER SCHEMA` / `ALTER MATERIALIZED VIEW` RENAME TO 以外 | 同上 |
