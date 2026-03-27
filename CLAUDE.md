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
- `ALTER TABLE`: `ADD COLUMN`, `DROP COLUMN`, `ALTER COLUMN TYPE`, `SET/DROP DEFAULT`, `SET/DROP NOT NULL`, `RENAME COLUMN`, `RENAME TO`, `ADD/DROP CONSTRAINT`
- `CREATE [UNIQUE] INDEX` / `DROP INDEX`
- `CREATE SEQUENCE` / `DROP SEQUENCE`
- `ALTER SEQUENCE`（オプション変更）
- `CREATE TYPE ... AS ENUM` / `AS (composite)` / `AS RANGE`
- `DROP TYPE` / `ALTER TYPE`（`ADD VALUE`, `RENAME VALUE`, `RENAME TO`）
- `TRUNCATE TABLE`
- 未認識ステートメント → 末尾に verbatim pass-through

## テスト方針

このプロジェクトにはテストの役割が異なる 3 層がある。

### 1. ユニットテスト（`merger/parser`, `merger/schema`, `merger/emitter`）
パーサー・スキーマモデル・エミッターの設計通りの挙動をテストする。
モックや in-memory 構造体で完結するため、高速に実行できる。

### 2. ゴールデンテスト（`go test ./merger -run TestRun_Golden`）
`testdata/integration/*/input/` の SQL を実際に処理し、`want.sql` と一致するかを検証する。
出力フォーマットの回帰テストとして機能する。

### 3. PostgreSQL 統合テスト（`go test -tags integration ./integration`）
**このツールの品質を最終的に保証するテスト**。
実際の PostgreSQL 16 コンテナを使い、以下を検証する：
1. 入力 SQL ファイルを順次適用したスキーマ（sequential DB）
2. merger が生成したマージ済み SQL を適用したスキーマ（merged DB）
の 2 つを `pg_dump` で比較し、**意味的に等価**であることを確認する。

**統合テストが最重要**：merger の出力が PostgreSQL に正しく適用できること、
かつ sequential 適用と同一スキーマになることを実際の DB で保証するため。
新機能追加・バグ修正時は必ず対応する統合テストシナリオを `testdata/integration/` に追加すること。
統合テストを通過しない変更はマージしない。

## 注意事項

- 外部依存なし（stdlib のみ）
- FK 依存のトポロジカルソートは未対応。テーブルは最初の `CREATE TABLE` 登場順で出力される
- `RENAME COLUMN` 後、テーブル制約定義の文字列は自動更新されない（stderr に警告を出力）
- ドル引用符（`$$...$$`）内の `;` はステートメント区切りとして扱われない
