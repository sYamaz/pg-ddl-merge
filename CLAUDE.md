# pg-ddl-merge

PostgreSQL の DDL マイグレーションファイル群（`01_xxx.sql`, `02_xxx.sql`...）を
セマンティックに解析し、最終スキーマ状態を表すクリーンな DDL ファイルに統合する Go 製 CLI ツール。

## コマンド

```bash
go build ./...          # ビルド
go test ./...           # 全テスト実行
go test ./... -count=1 -coverprofile=coverage.out  # カバレッジ付き
go test ./merger -run TestRun_Golden -update        # ゴールデンファイル更新
go test -tags integration ./integration             # PostgreSQL 統合テスト（要 Docker）
go run . -input ./ddl -output ./merged.sql          # 実行
```

## フラグ

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-input` / `-i` | `.` | 入力ディレクトリ |
| `-output` / `-o` | `./merged.sql` | 出力ファイルパス |

## アーキテクチャ

```
main.go                     # CLI エントリーポイント（flag パッケージ）
merger/
  merger.go                 # Run() オーケストレーション
  sorter.go                 # ファイルを数値プレフィックス順にソート
  parser/
    splitter.go             # SQL 文字列 → []string（ステートメント分割）
    ast.go                  # AST 型定義
    parser.go               # string → Statement（DDL パーサー）
  schema/
    model.go                # Schema / Table / ColumnDef 等の構造体
    schema.go               # Apply() — DDL をスキーマモデルに適用
  emitter/
    emitter.go              # Schema → クリーンな SQL 文字列
docs/
  postgres-command-coverage.md  # 対応コマンドの網羅状況ドキュメント
testdata/
  integration/                  # ゴールデンテスト用入出力ディレクトリ（各シナリオごとにサブディレクトリ）
```

## 入力ファイルの命名規則

`{数値}_{名前}.sql` 形式のみ対象（例: `01_users.sql`, `10_indexes.sql`）。
数値プレフィックスの昇順で処理される。同プレフィックスの場合はファイル名で二次ソート。

## 対応DDL操作

- `CREATE TABLE` / `DROP TABLE`
- `CREATE TABLE ... PARTITION OF`
- `CREATE TEMPORARY TABLE` / `CREATE UNLOGGED TABLE`
- `ALTER TABLE`: `ADD COLUMN`, `DROP COLUMN`, `ALTER COLUMN TYPE`, `SET/DROP DEFAULT`, `SET/DROP NOT NULL`, `RENAME COLUMN`, `RENAME TO`, `ADD/DROP CONSTRAINT`, `RENAME CONSTRAINT`, `ALTER COLUMN SET GENERATED { ALWAYS | BY DEFAULT }`
- `CREATE [UNIQUE] INDEX [CONCURRENTLY] [IF NOT EXISTS]` / `DROP INDEX`
  - 同名インデックスの再定義は後勝ち（`IF NOT EXISTS` の場合はスキップ）
- `CREATE SEQUENCE` / `DROP SEQUENCE`
- `ALTER SEQUENCE`（オプション変更）
- `CREATE TYPE ... AS ENUM` / `AS (composite)` / `AS RANGE`
- `DROP TYPE` / `ALTER TYPE`（`ADD VALUE`, `RENAME VALUE`, `RENAME TO`）
- `TRUNCATE TABLE`
- `CREATE [OR REPLACE] VIEW` / `DROP VIEW`
- `CREATE [OR REPLACE] MATERIALIZED VIEW` / `DROP MATERIALIZED VIEW`
- `CREATE [OR REPLACE] FUNCTION` / `DROP FUNCTION`
- `CREATE [OR REPLACE] PROCEDURE` / `DROP PROCEDURE`
- `ALTER FUNCTION` / `ALTER PROCEDURE`: `RENAME TO` + `OWNER TO` / `SECURITY DEFINER/INVOKER` などの非 RENAME アクションに対応。対象関数が schema 内に存在する場合は CREATE の直後に出力。存在しない場合は末尾にパススルー
- `CREATE [CONSTRAINT] [OR REPLACE] TRIGGER` / `DROP TRIGGER`
- `CREATE DOMAIN` / `DROP DOMAIN`
- `CREATE EXTENSION [IF NOT EXISTS]` / `DROP EXTENSION`
- `CREATE SCHEMA` / `DROP SCHEMA`
- `CREATE POLICY` / `DROP POLICY`
- `CREATE [OR REPLACE] RULE` / `DROP RULE`
- `ALTER INDEX/VIEW/…`: `RENAME TO` に対応。それ以外は UnknownStmt でパススルー
- 未認識ステートメント → 末尾に verbatim pass-through

## 出力順序

Emitter は以下の順で出力する（依存関係の都合で変更不可）：
`SCHEMA` → `EXTENSION` → `SEQUENCE` → `TYPE(ENUM/composite/range)` → `DOMAIN` → `TABLE` → `INDEX` → `FUNCTION/PROCEDURE` → `VIEW/MATERIALIZED VIEW` → `TRIGGER` → `POLICY/RULE` → `TRUNCATE` → `UNKNOWN(pass-through)`

## テスト方針

3 層構成。統合テストが最重要。

| 層 | コマンド | 目的 |
|----|---------|------|
| ユニット | `go test ./merger/...` | パーサー・スキーマ・エミッターの設計通りの挙動 |
| ゴールデン | `go test ./merger -run TestRun_Golden` | マージ出力の文字列フォーマット回帰 |
| **PostgreSQL 統合** | `go test -tags integration ./integration` | **最重要**：実 DB で sequential 適用と merged 適用が同一スキーマになることを `pg_dump` で比較 |

新機能・バグ修正時は統合テストシナリオを追加すること。統合テストを通過しない変更はマージしない。

### 統合テストシナリオの追加方法

```
testdata/integration/NN_scenario_name/
  input/           # 01_xxx.sql, 02_xxx.sql ... （順次適用される）
  want.sql         # ゴールデンテスト用の期待出力
  setup.sql        # （省略可）sequential/merged 両 DB に事前適用する SQL
```

## 注意事項

- `merger` パッケージ本体は外部依存なし（stdlib のみ）。`integration` パッケージは testcontainers・lib/pq を使用
- FK 依存のトポロジカルソートは未対応。テーブルは最初の `CREATE TABLE` 登場順で出力される
- `RENAME COLUMN` 後、テーブル制約定義の文字列は自動更新されない（stderr に警告を出力）
- ドル引用符（`$$...$$`）内の `;` はステートメント区切りとして扱われない
