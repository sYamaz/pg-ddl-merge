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
| `ALTER TABLE` | 🔶 | ADD/DROP COLUMN・ALTER COLUMN TYPE/DEFAULT/NOT NULL・RENAME COLUMN/TO・ADD/DROP CONSTRAINT に対応。未認識アクションは警告を出してスキップ（エラーにならない） |
| `DROP TABLE` | ✅ | IF EXISTS・複数テーブル名（`a, b, c`）・CASCADE/RESTRICT（無視）に対応 |
| `TRUNCATE` | ✅ | パース・同一テーブルセットで dedup（後勝ち）・RESTART IDENTITY/CASCADE 保持 |

### インデックス

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE INDEX` | 🔶 | UNIQUE・CONCURRENTLY・IF NOT EXISTS に対応。USING method・WHERE（部分インデックス）・INCLUDE・WITH（storage）・TABLESPACE は Body に verbatim 保持のみ |
| `ALTER INDEX` | 🔶 | RENAME TO に対応。他のアクションは UnknownStmt でパススルー |
| `DROP INDEX` | ✅ | IF EXISTS・CONCURRENTLY 対応 |
| `REINDEX` | ❌ | 対応対象外でも可（パースより pass-through が現実的） |

### シーケンス

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE SEQUENCE` | ✅ | 名前抽出・本体オプションを個別フィールドとして保持（多行対応） |
| `ALTER SEQUENCE` | ✅ | RENAME TO・INCREMENT BY/MINVALUE/MAXVALUE/START WITH/CACHE/CYCLE/OWNED BY/AS/SET LOGGED に対応。RESTART はパース済み（runtime 状態のため body 更新なし） |
| `DROP SEQUENCE` | ✅ | IF EXISTS 対応 |

### 型

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE TYPE` | ✅ | `AS ENUM`・`AS (composite)`・`AS RANGE` に対応。verbatim 保持・RENAME TO・DROP TYPE 対応。Base type は UnknownStmt |
| `ALTER TYPE` | ✅ | ADD VALUE（IF NOT EXISTS・BEFORE/AFTER）・RENAME VALUE・RENAME TO に対応 |
| `DROP TYPE` | ✅ | IF EXISTS 対応 |

### ビュー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE VIEW` | ✅ | verbatim 保持・dedup（同名は後勝ち） |
| `CREATE OR REPLACE VIEW` | ✅ | OR REPLACE で上書き |
| `ALTER VIEW` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP VIEW` | ✅ | IF EXISTS 対応 |

### マテリアライズドビュー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE MATERIALIZED VIEW` | ✅ | verbatim 保持 |
| `ALTER MATERIALIZED VIEW` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP MATERIALIZED VIEW` | ✅ | IF EXISTS 対応 |
| `REFRESH MATERIALIZED VIEW` | ❌ | データ操作のためパススルーで十分 |

### スキーマ

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE SCHEMA` | ✅ | verbatim 保持 |
| `ALTER SCHEMA` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP SCHEMA` | ✅ | IF EXISTS 対応 |

### 関数・プロシージャ

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE FUNCTION` | ✅ | verbatim 保持（`$$...$$` 内の `;` は splitter で対処済み） |
| `CREATE OR REPLACE FUNCTION` | ✅ | OR REPLACE で上書き |
| `ALTER FUNCTION` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP FUNCTION` | ✅ | IF EXISTS・名前抽出（`(` 前まで）に対応 |
| `CREATE PROCEDURE` | ✅ | verbatim 保持 |
| `ALTER PROCEDURE` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP PROCEDURE` | ✅ | IF EXISTS 対応 |

### トリガー

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE TRIGGER` | ✅ | verbatim 保持。キーは `triggername_on_tablename` |
| `CREATE CONSTRAINT TRIGGER` | ✅ | 同上 |
| `CREATE OR REPLACE TRIGGER` | ✅ | OR REPLACE で上書き |
| `ALTER TRIGGER` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP TRIGGER` | ✅ | IF EXISTS・ON tablename に対応 |

### 拡張

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE EXTENSION` | ✅ | verbatim 保持・IF NOT EXISTS 対応 |
| `ALTER EXTENSION` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP EXTENSION` | ✅ | IF EXISTS 対応 |

### ドメイン

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE DOMAIN` | ✅ | verbatim 保持 |
| `ALTER DOMAIN` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP DOMAIN` | ✅ | IF EXISTS 対応 |

### ポリシー（Row Level Security）

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE POLICY` | ✅ | verbatim 保持。キーは `policyname_on_tablename` |
| `ALTER POLICY` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP POLICY` | ✅ | IF EXISTS・ON tablename に対応 |

### ルール

| コマンド | 状況 | 備考 |
|---------|------|------|
| `CREATE RULE` | ✅ | verbatim 保持。キーは `rulename_on_tablename` |
| `CREATE OR REPLACE RULE` | ✅ | OR REPLACE で上書き |
| `ALTER RULE` | 🔶 | RENAME TO のみ対応。その他は UnknownStmt でパススルー |
| `DROP RULE` | ✅ | IF EXISTS・ON tablename に対応 |

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
| `PARTITION OF parent_table` | ✅ | ObjPartition として名前キー管理・verbatim 保持。DROP TABLE で削除可能。親テーブルの後に emit |
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
| `ALTER COLUMN col SET STATISTICS n` | ✅ | 警告を出してスキップ（スキーマモデルに影響なし） |
| `ALTER COLUMN col SET STORAGE type` | ✅ | 同上 |
| `ALTER COLUMN col SET COMPRESSION method` | ✅ | 同上 |
| `ALTER COLUMN col SET (option)` / `RESET (option)` | ✅ | 同上 |
| `ALTER COLUMN col ADD GENERATED ...` | ✅ | 同上 |
| `ALTER COLUMN col SET GENERATED ...` | ✅ | 同上 |
| `ALTER COLUMN col DROP IDENTITY` | ✅ | 同上 |
| `ALTER COLUMN TYPE ... USING expr`（型変換） | ✅ | USING 節は除去して型名のみ保持 |
| `SET SCHEMA new_schema` | ✅ | 警告を出してスキップ |
| `SET TABLESPACE new_tablespace` | ✅ | 同上 |
| `SET (storage_parameter)` | ✅ | 同上 |
| `RESET (storage_parameter)` | ✅ | 同上 |
| `CLUSTER ON index_name` | ✅ | 同上 |
| `SET WITHOUT CLUSTER` | ✅ | 同上 |
| `SET ACCESS METHOD method` | ✅ | 同上 |
| `ENABLE/DISABLE TRIGGER` | ✅ | 同上 |
| `ENABLE/DISABLE RULE` | ✅ | 同上 |
| `ENABLE/DISABLE ROW LEVEL SECURITY` | ✅ | 同上 |
| `FORCE/NO FORCE ROW LEVEL SECURITY` | ✅ | 同上 |
| `ATTACH PARTITION` | ✅ | 同上 |
| `DETACH PARTITION` | ✅ | 同上 |
| `VALIDATE CONSTRAINT name` | ✅ | 同上 |
| `INHERIT / NO INHERIT parent` | ✅ | 同上 |
| `OF type_name` / `NOT OF` | ✅ | 同上 |
| `OWNER TO new_owner` | ✅ | 同上 |

---

## DROP TABLE 構文の詳細カバレッジ

| 構文 | 状況 | 備考 |
|------|------|------|
| `DROP TABLE name` | ✅ | |
| `DROP TABLE IF EXISTS name` | ✅ | |
| 複数テーブル（`DROP TABLE a, b`） | ✅ | 全テーブルを順番に削除 |
| `CASCADE` | ✅ | パース時に無視（シーケンス等の依存物は別途処理） |
| `RESTRICT` | ✅ | パース時に無視 |

---

## TRUNCATE 構文の詳細カバレッジ

| 構文 | 状況 | 備考 |
|------|------|------|
| `TRUNCATE TABLE name` | ✅ | パース・dedup（同一テーブルセットは後勝ち） |
| `TRUNCATE TABLE name RESTART IDENTITY` | ✅ | |
| `TRUNCATE TABLE name CASCADE` | ✅ | |
