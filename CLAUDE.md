# pg-ddl-merge

PostgreSQL の DDL マイグレーションファイル群（`01_xxx.sql`, `02_xxx.sql`...）を
セマンティックに解析し、最終スキーマ状態を表すクリーンな DDL ファイルに統合する Go 製 CLI ツール。

## コマンド

```bash
go build ./...          # ビルド
go test ./...           # 全テスト実行
go test ./... -count=1 -coverprofile=coverage.out  # カバレッジ付き
go test ./merger -run TestRun_Golden -update        # ゴールデンファイル更新
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
- `ALTER TABLE`: `ADD COLUMN`, `DROP COLUMN`, `ALTER COLUMN TYPE`, `SET/DROP DEFAULT`, `SET/DROP NOT NULL`, `RENAME COLUMN`, `RENAME TO`, `ADD/DROP CONSTRAINT`
- `CREATE [UNIQUE] INDEX` / `DROP INDEX`
- `CREATE SEQUENCE` / `DROP SEQUENCE`
- `CREATE TYPE ... AS ENUM`
- 未認識ステートメント → 末尾に verbatim pass-through

## 注意事項

- 外部依存なし（stdlib のみ）
- FK 依存のトポロジカルソートは未対応。テーブルは最初の `CREATE TABLE` 登場順で出力される
- `RENAME COLUMN` 後、テーブル制約定義の文字列は自動更新されない（stderr に警告を出力）
- ドル引用符（`$$...$$`）内の `;` はステートメント区切りとして扱われない
