package parser

import (
	"strings"
	"testing"
)

// ---- helpers ----------------------------------------------------------------

func mustParse(t *testing.T, sql string) Statement {
	t.Helper()
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", sql, err)
	}
	return stmt
}

func ptr(s string) *string { return &s }

// ---- CREATE TABLE -----------------------------------------------------------

func TestParseCreateTable_Simple(t *testing.T) {
	stmt := mustParse(t, "CREATE TABLE users (id bigint, name text)")
	ct, ok := stmt.(CreateTableStmt)
	if !ok {
		t.Fatalf("want CreateTableStmt, got %T", stmt)
	}
	if ct.TableName != "users" {
		t.Errorf("TableName: got %q, want %q", ct.TableName, "users")
	}
	if len(ct.Columns) != 2 {
		t.Fatalf("Columns len: got %d, want 2", len(ct.Columns))
	}
	if ct.Columns[0].Name != "id" || ct.Columns[0].DataType != "bigint" {
		t.Errorf("col0: %+v", ct.Columns[0])
	}
}

func TestParseCreateTable_IfNotExists(t *testing.T) {
	stmt := mustParse(t, "CREATE TABLE IF NOT EXISTS t (id int)")
	ct := stmt.(CreateTableStmt)
	if ct.TableName != "t" {
		t.Errorf("TableName: got %q, want %q", ct.TableName, "t")
	}
}

func TestParseCreateTable_NotNullAndDefault(t *testing.T) {
	stmt := mustParse(t, "CREATE TABLE t (n int NOT NULL DEFAULT 0)")
	ct := stmt.(CreateTableStmt)
	col := ct.Columns[0]
	if !col.NotNull {
		t.Error("expected NotNull=true")
	}
	if col.Default == nil || *col.Default != "0" {
		t.Errorf("Default: got %v", col.Default)
	}
}

func TestParseCreateTable_WithConstraint(t *testing.T) {
	stmt := mustParse(t, "CREATE TABLE t (id int, CONSTRAINT pk PRIMARY KEY (id))")
	ct := stmt.(CreateTableStmt)
	if len(ct.Constraints) != 1 {
		t.Fatalf("Constraints len: got %d, want 1", len(ct.Constraints))
	}
	if ct.Constraints[0].Name != "pk" {
		t.Errorf("constraint name: got %q", ct.Constraints[0].Name)
	}
}

// ---- ALTER TABLE ------------------------------------------------------------

func TestParseAlterTable_AddColumn(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ADD COLUMN email text")
	at := stmt.(AlterTableStmt)
	if at.TableName != "t" {
		t.Errorf("TableName: got %q", at.TableName)
	}
	if len(at.Actions) != 1 {
		t.Fatalf("Actions len: got %d", len(at.Actions))
	}
	a := at.Actions[0]
	if a.Kind != ActionAddColumn {
		t.Errorf("Kind: got %v", a.Kind)
	}
	if a.ColDef == nil || a.ColDef.DataType != "text" {
		t.Errorf("ColDef: %+v", a.ColDef)
	}
}

func TestParseAlterTable_DropColumn(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t DROP COLUMN old_col")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionDropColumn || a.Column != "old_col" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_AlterColumnType(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ALTER COLUMN price TYPE numeric(10,2)")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionAlterColumnType {
		t.Errorf("Kind: got %v", a.Kind)
	}
	if a.DataType != "numeric(10,2)" {
		t.Errorf("DataType: got %q", a.DataType)
	}
}

func TestParseAlterTable_SetDefault(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ALTER COLUMN status SET DEFAULT 'active'")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionSetDefault || a.Default != "'active'" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_DropDefault(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ALTER COLUMN status DROP DEFAULT")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionDropDefault {
		t.Errorf("Kind: got %v", a.Kind)
	}
}

func TestParseAlterTable_SetNotNull(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ALTER COLUMN name SET NOT NULL")
	at := stmt.(AlterTableStmt)
	if at.Actions[0].Kind != ActionSetNotNull {
		t.Errorf("Kind: got %v", at.Actions[0].Kind)
	}
}

func TestParseAlterTable_DropNotNull(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ALTER COLUMN name DROP NOT NULL")
	at := stmt.(AlterTableStmt)
	if at.Actions[0].Kind != ActionDropNotNull {
		t.Errorf("Kind: got %v", at.Actions[0].Kind)
	}
}

func TestParseAlterTable_RenameColumn(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t RENAME COLUMN old TO new")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionRenameColumn || a.Column != "old" || a.NewName != "new" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_RenameTo(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t RENAME TO t2")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionRenameTo || a.NewName != "t2" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_AddConstraint(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ADD CONSTRAINT fk FOREIGN KEY (uid) REFERENCES users(id)")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionAddConstraint || a.Constraint.Name != "fk" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_DropConstraint(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t DROP CONSTRAINT fk")
	at := stmt.(AlterTableStmt)
	a := at.Actions[0]
	if a.Kind != ActionDropConstraint || a.Constraint.Name != "fk" {
		t.Errorf("action: %+v", a)
	}
}

func TestParseAlterTable_MultipleActions(t *testing.T) {
	stmt := mustParse(t, "ALTER TABLE t ADD COLUMN a int, ADD COLUMN b text")
	at := stmt.(AlterTableStmt)
	if len(at.Actions) != 2 {
		t.Fatalf("Actions len: got %d, want 2", len(at.Actions))
	}
}

// ---- DROP TABLE -------------------------------------------------------------

func TestParseDropTable(t *testing.T) {
	stmt := mustParse(t, "DROP TABLE users")
	dt := stmt.(DropTableStmt)
	if dt.TableName != "users" || dt.IfExists {
		t.Errorf("%+v", dt)
	}
}

func TestParseDropTable_IfExists(t *testing.T) {
	stmt := mustParse(t, "DROP TABLE IF EXISTS users")
	dt := stmt.(DropTableStmt)
	if !dt.IfExists {
		t.Error("expected IfExists=true")
	}
}

// ---- CREATE INDEX -----------------------------------------------------------

func TestParseCreateIndex(t *testing.T) {
	stmt := mustParse(t, "CREATE INDEX idx_email ON users (email)")
	ci := stmt.(CreateIndexStmt)
	if ci.IndexName != "idx_email" || ci.TableName != "users" || ci.Unique {
		t.Errorf("%+v", ci)
	}
}

func TestParseCreateUniqueIndex(t *testing.T) {
	stmt := mustParse(t, "CREATE UNIQUE INDEX idx_email ON users (email)")
	ci := stmt.(CreateIndexStmt)
	if !ci.Unique {
		t.Error("expected Unique=true")
	}
}

func TestParseCreateIndex_Concurrently(t *testing.T) {
	stmt := mustParse(t, "CREATE INDEX CONCURRENTLY idx_name ON t (name)")
	ci := stmt.(CreateIndexStmt)
	if ci.IndexName != "idx_name" {
		t.Errorf("IndexName: %q", ci.IndexName)
	}
}

// ---- DROP INDEX -------------------------------------------------------------

func TestParseDropIndex(t *testing.T) {
	stmt := mustParse(t, "DROP INDEX idx_email")
	di := stmt.(DropIndexStmt)
	if di.IndexName != "idx_email" || di.IfExists {
		t.Errorf("%+v", di)
	}
}

func TestParseDropIndex_IfExists(t *testing.T) {
	stmt := mustParse(t, "DROP INDEX IF EXISTS idx_email")
	di := stmt.(DropIndexStmt)
	if !di.IfExists {
		t.Error("expected IfExists=true")
	}
}

// ---- CREATE SEQUENCE --------------------------------------------------------

func TestParseCreateSequence(t *testing.T) {
	stmt := mustParse(t, "CREATE SEQUENCE users_id_seq START 1")
	cs := stmt.(CreateSequenceStmt)
	if cs.SeqName != "users_id_seq" {
		t.Errorf("SeqName: %q", cs.SeqName)
	}
	if !strings.Contains(cs.Body, "START 1") {
		t.Errorf("Body: %q", cs.Body)
	}
}

// ---- DROP SEQUENCE ----------------------------------------------------------

func TestParseDropSequence(t *testing.T) {
	stmt := mustParse(t, "DROP SEQUENCE users_id_seq")
	ds := stmt.(DropSequenceStmt)
	if ds.SeqName != "users_id_seq" || ds.IfExists {
		t.Errorf("%+v", ds)
	}
}

func TestParseDropSequence_IfExists(t *testing.T) {
	stmt := mustParse(t, "DROP SEQUENCE IF EXISTS users_id_seq")
	ds := stmt.(DropSequenceStmt)
	if !ds.IfExists {
		t.Error("expected IfExists=true")
	}
}

// ---- CREATE TYPE ------------------------------------------------------------

func TestParseCreateTypeEnum(t *testing.T) {
	stmt := mustParse(t, "CREATE TYPE status AS ENUM ('active', 'inactive', 'pending')")
	ct := stmt.(CreateTypeStmt)
	if ct.TypeName != "status" {
		t.Errorf("TypeName: %q", ct.TypeName)
	}
	if len(ct.Labels) != 3 {
		t.Fatalf("Labels len: got %d, want 3", len(ct.Labels))
	}
	if ct.Labels[0] != "active" {
		t.Errorf("Labels[0]: %q", ct.Labels[0])
	}
}

func TestParseCreateTypeNonEnum_FallsBackToUnknown(t *testing.T) {
	stmt := mustParse(t, "CREATE TYPE address AS (street text, city text)")
	if _, ok := stmt.(UnknownStmt); !ok {
		t.Errorf("want UnknownStmt, got %T", stmt)
	}
}

// ---- Unknown ----------------------------------------------------------------

func TestParseUnknown(t *testing.T) {
	stmt := mustParse(t, "SET search_path = public")
	u, ok := stmt.(UnknownStmt)
	if !ok {
		t.Fatalf("want UnknownStmt, got %T", stmt)
	}
	if u.Raw != "SET search_path = public" {
		t.Errorf("Raw: %q", u.Raw)
	}
}
