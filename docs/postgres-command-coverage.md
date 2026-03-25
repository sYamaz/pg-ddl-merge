# PostgreSQL 16 コマンド対応状況

PostgreSQL 16 の全 SQL コマンドを「DDL（対応対象）」「スコープ外 DDL」「非 DDL」に分類し、
現在の実装状況を記録する。

---

## 凡例

| 記号 | 意味 |
|------|------|
| ✅   | 対応済み（テストあり） |
| 🔶   | 部分対応（一部の構文のみ） |
| ❌   | 未対応（UnknownStmt でパススルー） |
| —    | 対応対象外 |

---

## 1. スコープ内 DDL（アプリケーション開発のマイグレーションで頻出）

これらは pg-ddl-merge が意味的にマージすべきコマンド群。

### テーブル

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE TABLE` | 🔶 | 基本カラム定義・NOT NULL・DEFAULT・制約（名前付き/無名）に対応。TEMP/UNLOGGED/INHERITS/LIKE/PARTITION BY/WITH/TABLESPACE/ON COMMIT 未対応 |
| `ALTER TABLE` | 🔶 | ADD/DROP COLUMN・ALTER COLUMN TYPE/DEFAULT/NOT NULL・RENAME COLUMN/TO・ADD/DROP CONSTRAINT に対応。SET SCHEMA・ATTACH/DETACH PARTITION・SET (storage)・CLUSTER ON 等未対応 |
| `DROP TABLE` | ✅ | IF EXISTS 対応。CASCADE/RESTRICT は無視（verbatim ではない点に注意） |
| `TRUNCATE` | ❌ | マイグレーションで稀だが対応対象 |

### インデックス

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE INDEX` | 🔶 | UNIQUE・CONCURRENTLY・IF NOT EXISTS に対応。USING method・WHERE（部分インデックス）・INCLUDE・WITH（storage）・TABLESPACE は Body に verbatim 保持のみ |
| `ALTER INDEX` | ❌ | RENAME・SET/RESET storage parameter 等 |
| `DROP INDEX` | ✅ | IF EXISTS・CONCURRENTLY 対応 |
| `REINDEX` | ❌ | 対応対象外でも可（パースより pass-through が現実的） |

### シーケンス

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE SEQUENCE` | 🔶 | 名前抽出のみ。本体は verbatim 保持 |
| `ALTER SEQUENCE` | ❌ | RESTART / INCREMENT / OWNED BY 等 |
| `DROP SEQUENCE` | ✅ | IF EXISTS 対応 |

### 型

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE TYPE` | 🔶 | `AS ENUM` のみ対応。COMPOSITE / RANGE / DOMAIN 形式は UnknownStmt |
| `ALTER TYPE` | ❌ | ADD VALUE（ENUM）・RENAME VALUE 等 |
| `DROP TYPE` | ❌ | |

### ビュー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE VIEW` | ❌ | |
| `CREATE OR REPLACE VIEW` | ❌ | |
| `ALTER VIEW` | ❌ | |
| `DROP VIEW` | ❌ | |

### マテリアライズドビュー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE MATERIALIZED VIEW` | ❌ | |
| `ALTER MATERIALIZED VIEW` | ❌ | |
| `DROP MATERIALIZED VIEW` | ❌ | |
| `REFRESH MATERIALIZED VIEW` | ❌ | |

### スキーマ

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE SCHEMA` | ❌ | |
| `ALTER SCHEMA` | ❌ | |
| `DROP SCHEMA` | ❌ | |

### 関数・プロシージャ

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE FUNCTION` | ❌ | `$$...$$` のパーススルーは splitter で対応済み |
| `CREATE OR REPLACE FUNCTION` | ❌ | |
| `ALTER FUNCTION` | ❌ | |
| `DROP FUNCTION` | ❌ | |
| `CREATE PROCEDURE` | ❌ | |
| `ALTER PROCEDURE` | ❌ | |
| `DROP PROCEDURE` | ❌ | |

### トリガー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE TRIGGER` | ❌ | |
| `ALTER TRIGGER` | ❌ | |
| `DROP TRIGGER` | ❌ | |

### 拡張

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE EXTENSION` | ❌ | |
| `ALTER EXTENSION` | ❌ | |
| `DROP EXTENSION` | ❌ | |

### ドメイン

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE DOMAIN` | ❌ | |
| `ALTER DOMAIN` | ❌ | |
| `DROP DOMAIN` | ❌ | |

### ポリシー（Row Level Security）

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE POLICY` | ❌ | |
| `ALTER POLICY` | ❌ | |
| `DROP POLICY` | ❌ | |

### ルール

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE RULE` | ❌ | |
| `ALTER RULE` | ❌ | |
| `DROP RULE` | ❌ | |

---

## 2. スコープ外 DDL（DBA レベル・インフラ管理）

これらはアプリケーションのマイグレーションに含まれることが稀なため、
pg-ddl-merge の対応対象外とし、UnknownStmt として末尾に pass-through する。

| コマンド群 | 理由 |
|-----------|------|
| `CREATE/ALTER/DROP DATABASE` | DB インスタンス管理 |
| `CREATE/ALTER/DROP TABLESPACE` | ストレージ管理 |
| `CREATE/ALTER/DROP ROLE`, `CREATE/ALTER/DROP USER`, `CREATE/ALTER/DROP GROUP` | ロール管理 |
| `GRANT`, `REVOKE` | 権限管理（pass-through で十分） |
| `CREATE/ALTER/DROP SERVER`, `CREATE/ALTER/DROP FOREIGN DATA WRAPPER`, `ALTER FOREIGN TABLE`, `CREATE/DROP FOREIGN TABLE`, `IMPORT FOREIGN SCHEMA`, `CREATE/ALTER/DROP USER MAPPING` | FDW / 外部サーバー管理 |
| `CREATE/ALTER/DROP COLLATION` | コレーション管理 |
| `CREATE/ALTER/DROP CONVERSION` | エンコーディング変換管理 |
| `CREATE/ALTER/DROP LANGUAGE` | 手続き言語管理 |
| `CREATE/ALTER/DROP AGGREGATE`, `CREATE/ALTER/DROP OPERATOR`, `CREATE/ALTER/DROP OPERATOR CLASS`, `CREATE/ALTER/DROP OPERATOR FAMILY`, `CREATE/ALTER/DROP CAST`, `CREATE/ALTER/DROP TRANSFORM`, `CREATE/DROP ACCESS METHOD` | 型システム拡張 |
| `CREATE/ALTER/DROP TEXT SEARCH CONFIGURATION/DICTIONARY/PARSER/TEMPLATE` | テキスト検索管理 |
| `CREATE/ALTER/DROP PUBLICATION`, `CREATE/ALTER/DROP SUBSCRIPTION` | 論理レプリケーション管理 |
| `CREATE/ALTER/DROP EVENT TRIGGER` | イベントトリガー管理 |
| `CREATE/ALTER/DROP STATISTICS` | 統計オブジェクト管理 |
| `CREATE TABLE AS` | AS SELECT 形式は DDL+DML のハイブリッド |
| `SELECT INTO` | 同上 |
| `SECURITY LABEL` | セキュリティラベル管理 |
| `COMMENT` | メタデータのみ |
| `CLUSTER` | 物理並び替え（運用コマンド） |
| `REINDEX` | 運用コマンド |

---

## 3. 非 DDL（対応不要）

スキーマ構造の定義・変更を行わないコマンド。
pg-ddl-merge の入力には含まれない想定のため対応不要。

### DML
- `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `VALUES`

### トランザクション制御 (TCL)
- `BEGIN`, `COMMIT`, `ROLLBACK`, `SAVEPOINT`, `RELEASE SAVEPOINT`, `ROLLBACK TO SAVEPOINT`
- `START TRANSACTION`, `END`
- `PREPARE TRANSACTION`, `COMMIT PREPARED`, `ROLLBACK PREPARED`
- `SET CONSTRAINTS`

### カーソル
- `DECLARE`, `FETCH`, `MOVE`, `CLOSE`

### 動的 SQL / プリペアドステートメント
- `PREPARE`, `EXECUTE`, `DEALLOCATE`

### セッション管理
- `SET`, `RESET`, `SHOW`, `DISCARD`
- `SET ROLE`, `SET SESSION AUTHORIZATION`, `SET TRANSACTION`

### 通知
- `LISTEN`, `NOTIFY`, `UNLISTEN`

### メンテナンス
- `ANALYZE`, `VACUUM`, `EXPLAIN`, `CHECKPOINT`, `LOAD`

### その他
- `ABORT`, `CALL`, `COPY`, `DO`, `LOCK`, `REASSIGN OWNED`, `DROP OWNED`

---

## CREATE TABLE 構文の詳細カバレッジ

PostgreSQL 16 `CREATE TABLE` の構文要素ごとの対応状況。

### テーブル修飾子

| 構文 | 状況 | 備考 |
|------|------|------|
| `CREATE TABLE name (...)` | ✅ | |
| `CREATE TABLE IF NOT EXISTS` | ✅ | |
| `CREATE TEMPORARY TABLE` / `CREATE TEMP TABLE` | ✅ | `Table.Temporary=true` として追跡・出力 |
| `CREATE UNLOGGED TABLE` | ✅ | `Table.Unlogged=true` として追跡・出力 |

### カラム定義

| 構文 | 状況 | 備考 |
|------|------|------|
| `name data_type` | ✅ | |
| `NOT NULL` | ✅ | |
| `NULL`（明示） | ✅ | 除去して NotNull=false 扱い |
| `DEFAULT expr` | ✅ | |
| `PRIMARY KEY`（カラムインライン） | 🔶 | InlineConstraints に verbatim 保持 |
| `UNIQUE`（カラムインライン） | 🔶 | InlineConstraints に verbatim 保持 |
| `REFERENCES reftable`（カラムインライン FK） | 🔶 | InlineConstraints に verbatim 保持（ON DELETE/UPDATE 等含む） |
| `CHECK (expr)`（カラムインライン） | 🔶 | InlineConstraints に verbatim 保持 |
| `GENERATED ALWAYS AS (expr) STORED` | ✅ | InlineConstraints に verbatim 保持・テストあり |
| `GENERATED { ALWAYS \| BY DEFAULT } AS IDENTITY` | ✅ | InlineConstraints に verbatim 保持・テストあり |
| `COLLATE collation` | ✅ | `ColumnDef.Collation` フィールドとして個別保持・出力 |
| `COMPRESSION method` | ✅ | InlineConstraints に verbatim 保持・テストあり |
| `STORAGE { PLAIN \| EXTERNAL \| EXTENDED \| MAIN }` | ✅ | InlineConstraints に verbatim 保持・テストあり |

### テーブルレベル制約

| 構文 | 状況 | 備考 |
|------|------|------|
| `CONSTRAINT name PRIMARY KEY (cols)` | ✅ | |
| `CONSTRAINT name UNIQUE (cols)` | ✅ | |
| `CONSTRAINT name FOREIGN KEY (cols) REFERENCES ...` | ✅ | 定義全体は verbatim 保持 |
| `CONSTRAINT name CHECK (expr)` | ✅ | |
| `PRIMARY KEY (cols)`（無名） | ✅ | |
| `UNIQUE (cols)`（無名） | ✅ | |
| `FOREIGN KEY (cols) REFERENCES ...`（無名） | ✅ | |
| `CHECK (expr)`（無名） | ✅ | |
| `UNIQUE NULLS [ NOT ] DISTINCT` | ✅ | TableConstraint.Definition に verbatim 保持・テストあり |
| `CHECK (expr) NO INHERIT` | ✅ | InlineConstraints に verbatim 保持・テストあり |
| `EXCLUDE [USING method] (...)` | ✅ | TableConstraint.Definition に verbatim 保持・テストあり |
| FK: `MATCH FULL \| PARTIAL \| SIMPLE` | ✅ | verbatim 保持・テストあり |
| FK: `ON DELETE action` | ✅ | verbatim 保持・テストあり |
| FK: `ON UPDATE action` | ✅ | verbatim 保持・テストあり |
| 制約: `DEFERRABLE \| NOT DEFERRABLE` | ✅ | verbatim 保持・テストあり |
| 制約: `INITIALLY IMMEDIATE \| DEFERRED` | ✅ | verbatim 保持・テストあり |

### テーブル構造オプション

| 構文 | 状況 | 備考 |
|------|------|------|
| `INHERITS (parent_table)` | ✅ | 閉じ括弧後の節は無視（スキーマ追跡には影響なし）・テストあり |
| `LIKE source_table [INCLUDING/EXCLUDING ...]` | ✅ | TableConstraint.Definition に verbatim 保持・テストあり |
| `PARTITION BY { RANGE \| LIST \| HASH }` | ✅ | 閉じ括弧後の節は無視・テストあり |
| `PARTITION OF parent_table` | ❌ | `PARTITION OF` を含む CREATE TABLE 形式は未対応（switch にマッチしない） |
| `WITH (storage_parameter)` | ✅ | 閉じ括弧後の節は無視・テストあり |
| `WITHOUT OIDS` | ✅ | 同上（PostgreSQL 12 以降実質的に no-op） |
| `ON COMMIT { PRESERVE ROWS \| DELETE ROWS \| DROP }` | ✅ | 閉じ括弧後の節は無視・TEMPORARY TABLE と組み合わせ使用 |
| `TABLESPACE tablespace_name` | ✅ | 閉じ括弧後の節は無視・テストあり |
| `USING method` | ✅ | 同上 |

---

## ALTER TABLE 構文の詳細カバレッジ

| 構文 | 状況 | 備考 |
|------|------|------|
| `ADD COLUMN [IF NOT EXISTS] col_def` | ✅ | |
| `DROP COLUMN [IF EXISTS] col` | ✅ | |
| `ALTER COLUMN col TYPE type` | ✅ | |
| `ALTER COLUMN col SET DATA TYPE type` | ✅ | |
| `ALTER COLUMN col SET DEFAULT expr` | ✅ | |
| `ALTER COLUMN col DROP DEFAULT` | ✅ | |
| `ALTER COLUMN col SET NOT NULL` | ✅ | |
| `ALTER COLUMN col DROP NOT NULL` | ✅ | |
| `RENAME COLUMN old TO new` | ✅ | |
| `RENAME TO new_name` | ✅ | |
| `ADD CONSTRAINT name def` | ✅ | |
| `DROP CONSTRAINT [IF EXISTS] name` | ✅ | |
| 複数アクション（カンマ区切り） | ✅ | |
| `ONLY` 修飾子 | ✅ | パース時に無視 |
| `ALTER COLUMN col SET STATISTICS n` | ❌ | |
| `ALTER COLUMN col SET STORAGE type` | ❌ | |
| `ALTER COLUMN col SET COMPRESSION method` | ❌ | |
| `ALTER COLUMN col SET (option)` / `RESET (option)` | ❌ | |
| `ALTER COLUMN col ADD GENERATED ...` | ❌ | |
| `ALTER COLUMN col SET GENERATED ...` | ❌ | |
| `ALTER COLUMN col DROP IDENTITY` | ❌ | |
| `ADD COLUMN ... USING expr`（型変換） | ❌ | |
| `SET SCHEMA new_schema` | ❌ | |
| `SET TABLESPACE new_tablespace` | ❌ | |
| `SET (storage_parameter)` | ❌ | |
| `RESET (storage_parameter)` | ❌ | |
| `CLUSTER ON index_name` | ❌ | |
| `SET WITHOUT CLUSTER` | ❌ | |
| `SET ACCESS METHOD method` | ❌ | |
| `ENABLE/DISABLE TRIGGER` | ❌ | |
| `ENABLE/DISABLE RULE` | ❌ | |
| `ENABLE/DISABLE ROW LEVEL SECURITY` | ❌ | |
| `FORCE/NO FORCE ROW LEVEL SECURITY` | ❌ | |
| `ATTACH PARTITION` | ❌ | |
| `DETACH PARTITION` | ❌ | |
| `VALIDATE CONSTRAINT name` | ❌ | |
| `INHERIT / NO INHERIT parent` | ❌ | |
| `OF type_name` / `NOT OF` | ❌ | |
| `OWNER TO new_owner` | ❌ | |

---

## DROP TABLE 構文の詳細カバレッジ

| 構文 | 状況 | 備考 |
|------|------|------|
| `DROP TABLE name` | ✅ | |
| `DROP TABLE IF EXISTS name` | ✅ | |
| 複数テーブル（`DROP TABLE a, b`） | ❌ | 最初のテーブル名のみ取得される |
| `CASCADE` | ❌ | パース時に無視（シーケンス等の依存物は別途処理） |
| `RESTRICT` | ❌ | パース時に無視 |

---

## TRUNCATE 構文の詳細カバレッジ

| 構文 | 状況 | 備考 |
|------|------|------|
| `TRUNCATE TABLE name` | ❌ | UnknownStmt で pass-through |
| `TRUNCATE TABLE name RESTART IDENTITY` | ❌ | |
| `TRUNCATE TABLE name CASCADE` | ❌ | |
