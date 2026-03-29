# 実装優先度バックログ

`postgres-command-coverage.md` で部分対応（🔶）になっている項目を、
アプリケーションマイグレーションへの影響度で優先度付けしたもの。

---

## 優先度：高

### 1. `ALTER EXTENSION` の `UPDATE TO` 対応

**問題**

`ALTER EXTENSION name UPDATE [TO new_version]` は実際のマイグレーションで頻出する
（例: `ALTER EXTENSION postgis UPDATE TO '3.4.0'`）。
現状は RENAME TO 以外が UnknownStmt でパススルーされ、バージョン追跡ができない。

**期待する挙動**

- `ALTER EXTENSION name UPDATE TO 'version'` を解析し、`GenericObject.Body` を更新
- `ADD MEMBER` / `DROP MEMBER` はスコープ外（DBA 操作）として引き続きパススルーで良い

---

### 2. `ALTER DOMAIN` の内容変更対応

**問題**

ドメインに制約・デフォルト値変更が含まれるマイグレーションで、内容変更が追跡されない。
現状 RENAME TO 以外は UnknownStmt でパススルー。

**期待する挙動**

- `ADD CONSTRAINT name CHECK (expr)` → ドメイン定義に制約を追記
- `DROP CONSTRAINT [IF EXISTS] name` → ドメイン定義から制約を除去
- `SET DEFAULT expr` / `DROP DEFAULT` → ドメイン定義のデフォルト値を更新
- `SET NOT NULL` / `DROP NOT NULL` → ドメイン定義の NOT NULL を更新
- `OWNER TO` / `SET SCHEMA` は警告を出してスキップ（スキーマモデルに反映なし）

---

### 3. `ALTER POLICY` の内容変更対応

**問題**

Row Level Security ポリシーの条件変更（`USING` / `WITH CHECK`）はアプリレイヤーのマイグレーションに含まれることがある。
現状 RENAME TO 以外が UnknownStmt でパススルーされ、前の `CREATE POLICY` と後の `ALTER POLICY` が分離して出力される。

**期待する挙動**

- `ALTER POLICY name ON table [TO roles] [USING (expr)] [WITH CHECK (expr)]` を解析し、
  `GenericObject.Body`（CREATE POLICY の verbatim 定義）を上書き更新
- ポリシーが schema 内に存在しない場合は末尾にパススルー

---

## 優先度：中

### 4. `ALTER VIEW` の内容変更対応

**問題**

ビューのスキーマ変更やオーナー変更がマイグレーションに含まれる場合、RENAME TO 以外は追跡されない。

**期待する挙動**

- `ALTER VIEW name ALTER COLUMN col SET DEFAULT expr` → ビュー定義内のカラムデフォルトを更新（verbatim 保持）
- `OWNER TO` → 警告を出してスキップ（ビューの verbatim には反映しない）
- `SET SCHEMA` → 警告を出してスキップ
- `ALTER COLUMN col DROP DEFAULT` → 同上

---

### 5. `ALTER TABLE` の `RENAME TO` / `RENAME COLUMN` 後の inline 制約追跡

**問題**

テーブルまたはカラムを RENAME した後、`RENAME COLUMN` 後のテーブル制約定義文字列は自動更新されない
（`CLAUDE.md` にも警告出力の旨が記載されている）。
カラムインライン定義（InlineConstraints）の `REFERENCES`・`CHECK` 等も同様に文字列が古いまま。

**期待する挙動**

- `RENAME COLUMN old TO new` 時に `Constraints` の定義文字列内の旧カラム名を置換
- `RENAME TO` 時に inline FK（`REFERENCES` 節）の参照テーブル名が自己参照の場合は更新
- 完全解決が困難な場合でも、現状の警告出力を維持しつつ追跡精度を上げる

---

### 6. `ALTER TRIGGER` の `RENAME TO` 以外への対応

**問題**

`ALTER TRIGGER name ON table DEPENDS ON EXTENSION ext` など、RENAME TO 以外は UnknownStmt でパススルー。
アプリマイグレーションでの頻度は低いが、トリガーの依存関係変更で問題になる可能性がある。

**期待する挙動**

- `DEPENDS ON EXTENSION` / `NO DEPENDS ON EXTENSION` はそのままパススルーで良い（DBA 操作に近い）
- 将来的に `ALTER TRIGGER ... ENABLE/DISABLE` はトリガー定義側に反映できると望ましい

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
