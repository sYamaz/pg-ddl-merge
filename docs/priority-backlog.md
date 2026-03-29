# 実装優先度バックログ

`postgres-command-coverage.md` で部分対応（🔶）になっている項目を、
アプリケーションマイグレーションへの影響度で優先度付けしたもの。

> **注:** `ALTER VIEW` の内容変更対応（ALTER COLUMN SET/DROP DEFAULT・SET/RESET options）は
> d16441f にて実装済み。coverage doc の 🔶 表記は更新要。

---

## 優先度：中

### 1. カラムインライン制約の `RENAME COLUMN` 追跡

`PRIMARY KEY`・`UNIQUE`・`REFERENCES reftable`・`CHECK (expr)` のカラムインライン記述は
現在 `InlineConstraints` に verbatim 保持されており、`RENAME COLUMN old TO new` を実行しても
その中のカラム名は更新されない。

テーブルレベル制約（`Constraints`）のカラム名更新（ワード境界置換）は実装済みだが、
インライン制約は対象外のため、以下のようなケースでマージ結果が sequential 適用と乖離する可能性がある。

```sql
-- 例: インライン CHECK に旧カラム名が残る
ALTER TABLE t RENAME COLUMN amount TO price;
-- → CHECK (amount > 0) が CHECK (price > 0) に更新されない
```

**対象構文：**
| 項目 | InlineConstraints の例 |
|------|----------------------|
| `CHECK (expr)` カラムインライン | `CHECK (col > 0)` |
| `REFERENCES reftable [(col)]` | `REFERENCES orders (col_id)` |
| `PRIMARY KEY` カラムインライン | `PRIMARY KEY` ← カラム名は列名そのものなので影響なし |
| `UNIQUE` カラムインライン | `UNIQUE` ← 同上 |

実質的に影響が出るのは `CHECK` と `REFERENCES ... (col)` のみ。`PRIMARY KEY`・`UNIQUE` インラインは
カラム名を式中に持たないため自動更新不要。

---

## 優先度：低

以下はいずれも DBA 操作・使用頻度が低く、アプリマイグレーションに含まれることは稀。
現状の UnknownStmt パススルーで実用上問題ない。

| 項目 | 理由 |
|------|------|
| `ALTER INDEX` RENAME TO 以外 | `SET TABLESPACE`・`CLUSTER ON` 等は DBA 操作 |
| `ALTER MATERIALIZED VIEW` RENAME TO 以外 | 同上（`SET TABLESPACE` 等） |
| `ALTER SCHEMA` RENAME TO 以外 | `OWNER TO` 等はロール管理寄り |
| `ALTER TRIGGER` RENAME TO 以外 | `DEPENDS ON EXTENSION` は稀。`ENABLE/DISABLE` は ALTER TABLE 経由で対応済み |
| `ALTER RULE` RENAME TO 以外 | ルール自体の使用頻度が低い |
| `ALTER TABLE`（概要 🔶）| 詳細カバレッジは全項目 ✅。概要の 🔶 は summary 表示の都合 |
