# pg-ddl-merge

PostgreSQL のマイグレーション DDL ファイル群（`01_xxx.sql`, `02_xxx.sql`...）をセマンティックに解析し、最終スキーマ状態を表すクリーンな DDL ファイルに統合する Go 製 CLI ツール。

## 概要

マイグレーション管理ツール（Flyway, golang-migrate 等）で蓄積された大量の DDL ファイルを、適用後の最終状態を表す 1 つの SQL ファイルに集約します。単純な文字列結合ではなく、DDL をセマンティックに解析・適用するため、途中の `ALTER TABLE`・`DROP TABLE` 等が吸収された、クリーンな出力が得られます。

### 例

```
01_create_users.sql  ── CREATE TABLE users (id SERIAL, name TEXT);
02_add_email.sql     ── ALTER TABLE users ADD COLUMN email TEXT NOT NULL;
03_rename.sql        ── ALTER TABLE users RENAME COLUMN name TO username;
```

↓ `pg-ddl-merge` で統合

```sql
CREATE TABLE users (
    id   SERIAL,
    username TEXT,
    email TEXT NOT NULL
);
```

## インストール

```bash
go install github.com/sYamaz/pg-ddl-merge@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/sYamaz/pg-ddl-merge
cd pg-ddl-merge
go build -o pg-ddl-merge .
```

## 使い方

```bash
pg-ddl-merge -input ./migrations -output ./schema.sql
```

### フラグ

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-input` / `-i` | `.` | 入力ディレクトリ（DDL ファイルを含むディレクトリ） |
| `-output` / `-o` | `./merged.sql` | 出力ファイルパス |

## 入力ファイルの命名規則

`{数値}_{名前}.sql` 形式のファイルのみ処理対象です（例: `01_users.sql`, `10_indexes.sql`）。
数値プレフィックスの昇順で処理され、同じプレフィックスの場合はファイル名でソートされます。

## 対応 DDL 操作

| カテゴリ | 操作 |
|---------|------|
| テーブル | `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE` |
| インデックス | `CREATE [UNIQUE] INDEX`, `ALTER INDEX`, `DROP INDEX` |
| シーケンス | `CREATE SEQUENCE`, `ALTER SEQUENCE`, `DROP SEQUENCE` |
| 型 | `CREATE TYPE AS ENUM`, `ALTER TYPE`, `DROP TYPE` |
| ビュー | `CREATE [OR REPLACE] VIEW`, `ALTER VIEW`, `DROP VIEW` |
| マテリアライズドビュー | `CREATE MATERIALIZED VIEW`, `ALTER MATERIALIZED VIEW`, `DROP MATERIALIZED VIEW` |
| スキーマ | `CREATE SCHEMA`, `ALTER SCHEMA`, `DROP SCHEMA` |
| 関数 / プロシージャ | `CREATE [OR REPLACE] FUNCTION`, `CREATE PROCEDURE`, `ALTER FUNCTION`, `DROP FUNCTION`, `DROP PROCEDURE` |
| トリガー | `CREATE [OR REPLACE] TRIGGER`, `ALTER TRIGGER`, `DROP TRIGGER` |
| 拡張 | `CREATE EXTENSION`, `ALTER EXTENSION`, `DROP EXTENSION` |
| ドメイン | `CREATE DOMAIN`, `ALTER DOMAIN`, `DROP DOMAIN` |
| ポリシー | `CREATE POLICY`, `ALTER POLICY`, `DROP POLICY` |
| ルール | `CREATE [OR REPLACE] RULE`, `ALTER RULE`, `DROP RULE` |
| その他 | 未認識ステートメント → 末尾に verbatim pass-through |

詳細は [docs/postgres-command-coverage.md](docs/postgres-command-coverage.md) を参照してください。

## アーキテクチャ

```
main.go                      # CLI エントリーポイント
merger/
  merger.go                  # Run() オーケストレーション
  sorter.go                  # ファイルを数値プレフィックス順にソート
  parser/
    splitter.go              # SQL 文字列 → []string（ステートメント分割）
    ast.go                   # AST 型定義
    parser.go                # string → Statement（DDL パーサー）
  schema/
    model.go                 # Schema / Table / ColumnDef 等の構造体
    schema.go                # Apply() — DDL をスキーマモデルに適用
  emitter/
    emitter.go               # Schema → クリーンな SQL 文字列
docs/
  postgres-command-coverage.md  # 対応コマンドの網羅状況ドキュメント
testdata/
  integration/               # ゴールデンテスト用入出力（シナリオごとにサブディレクトリ）
```

処理の流れ: `SQL files` → `parser.Split` → `parser.Parse` → `schema.Apply` → `emitter.Emit` → `merged.sql`

## 注意事項

- **外部依存なし** — stdlib のみ使用
- FK 依存のトポロジカルソートは未対応。テーブルは最初の `CREATE TABLE` 登場順で出力される
- `RENAME COLUMN` 後、テーブル制約定義の文字列は自動更新されない（stderr に警告を出力）
- ドル引用符（`$$...$$`）内の `;` はステートメント区切りとして扱われない

## 開発

```bash
go build ./...                                              # ビルド
go test ./...                                               # 全テスト実行
go test ./... -count=1 -coverprofile=coverage.out           # カバレッジ付き
go test ./merger -run TestRun_Golden -update                # ゴールデンファイル更新
```

## ライセンス

MIT
