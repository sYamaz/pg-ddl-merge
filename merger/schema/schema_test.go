package schema

import (
	"testing"

	"github.com/shunyamazaki/pg-ddl-merge/merger/parser"
)

// ---- helpers ----------------------------------------------------------------

func newSchema() *Schema { return New() }

func applyAll(t *testing.T, s *Schema, stmts ...parser.Statement) {
	t.Helper()
	for _, stmt := range stmts {
		if err := s.Apply(stmt); err != nil {
			t.Fatalf("Apply(%T): %v", stmt, err)
		}
	}
}

func ptr(s string) *string { return &s }

// ---- CREATE TABLE -----------------------------------------------------------

func TestApplyCreateTable(t *testing.T) {
	s := newSchema()
	applyAll(t, s, parser.CreateTableStmt{
		TableName: "users",
		Columns: []parser.ColumnDef{
			{Name: "id", DataType: "bigint"},
		},
	})
	if len(s.Tables) != 1 || s.Tables[0].Name != "users" {
		t.Fatalf("Tables: %v", s.Tables)
	}
}

func TestApplyCreateTable_Duplicate(t *testing.T) {
	s := newSchema()
	stmt := parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}}
	applyAll(t, s, stmt)
	if err := s.Apply(stmt); err == nil {
		t.Error("expected error for duplicate CREATE TABLE")
	}
}

// ---- DROP TABLE -------------------------------------------------------------

func TestApplyDropTable(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.DropTableStmt{TableName: "t"},
	)
	if len(s.Tables) != 0 {
		t.Errorf("expected empty Tables, got %d", len(s.Tables))
	}
}

func TestApplyDropTable_NotFound(t *testing.T) {
	s := newSchema()
	if err := s.Apply(parser.DropTableStmt{TableName: "no_such"}); err == nil {
		t.Error("expected error")
	}
}

func TestApplyDropTable_IfExists_NoError(t *testing.T) {
	s := newSchema()
	if err := s.Apply(parser.DropTableStmt{TableName: "no_such", IfExists: true}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyDropTable_RemovesAssociatedIndexes(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"},
		parser.DropTableStmt{TableName: "t"},
	)
	if len(s.Indexes) != 0 {
		t.Errorf("expected no indexes after DROP TABLE, got %d", len(s.Indexes))
	}
}

// ---- ALTER TABLE: columns ---------------------------------------------------

func TestApplyAddColumn(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionAddColumn, Column: "email", ColDef: &parser.ColumnDef{Name: "email", DataType: "text"}},
		}},
	)
	if len(s.Tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(s.Tables[0].Columns))
	}
}

func TestApplyAddColumn_Duplicate(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
	)
	err := s.Apply(parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
		{Kind: parser.ActionAddColumn, Column: "id", ColDef: &parser.ColumnDef{Name: "id", DataType: "int"}},
	}})
	if err == nil {
		t.Error("expected error for duplicate ADD COLUMN")
	}
}

func TestApplyDropColumn(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{
			{Name: "id", DataType: "int"},
			{Name: "tmp", DataType: "text"},
		}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionDropColumn, Column: "tmp"},
		}},
	)
	if len(s.Tables[0].Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(s.Tables[0].Columns))
	}
}

func TestApplyDropColumn_NotFound(t *testing.T) {
	s := newSchema()
	applyAll(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})
	err := s.Apply(parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
		{Kind: parser.ActionDropColumn, Column: "no_col"},
	}})
	if err == nil {
		t.Error("expected error")
	}
}

func TestApplyAlterColumnType(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "price", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionAlterColumnType, Column: "price", DataType: "numeric(10,2)"},
		}},
	)
	if s.Tables[0].Columns[0].DataType != "numeric(10,2)" {
		t.Errorf("DataType: %q", s.Tables[0].Columns[0].DataType)
	}
}

func TestApplySetDefault(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "status", DataType: "text"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionSetDefault, Column: "status", Default: "'active'"},
		}},
	)
	col := s.Tables[0].Columns[0]
	if col.Default == nil || *col.Default != "'active'" {
		t.Errorf("Default: %v", col.Default)
	}
}

func TestApplyDropDefault(t *testing.T) {
	defVal := "'active'"
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "status", DataType: "text", Default: &defVal}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionDropDefault, Column: "status"},
		}},
	)
	if s.Tables[0].Columns[0].Default != nil {
		t.Error("expected Default=nil after DROP DEFAULT")
	}
}

func TestApplySetNotNull(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "n", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionSetNotNull, Column: "n"},
		}},
	)
	if !s.Tables[0].Columns[0].NotNull {
		t.Error("expected NotNull=true")
	}
}

func TestApplyDropNotNull(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "n", DataType: "int", NotNull: true}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionDropNotNull, Column: "n"},
		}},
	)
	if s.Tables[0].Columns[0].NotNull {
		t.Error("expected NotNull=false")
	}
}

func TestApplyRenameColumn(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "old", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionRenameColumn, Column: "old", NewName: "new"},
		}},
	)
	if s.Tables[0].Columns[0].Name != "new" {
		t.Errorf("Name: %q", s.Tables[0].Columns[0].Name)
	}
}

func TestApplyRenameTo(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionRenameTo, NewName: "t2"},
		}},
	)
	if s.Tables[0].Name != "t2" {
		t.Errorf("Name: %q", s.Tables[0].Name)
	}
	// original key must be gone, new key must work
	if _, ok := s.tableIndex["t"]; ok {
		t.Error("old key still in tableIndex")
	}
	if _, ok := s.tableIndex["t2"]; !ok {
		t.Error("new key not in tableIndex")
	}
}

// ---- ALTER TABLE: constraints -----------------------------------------------

func TestApplyAddConstraint(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionAddConstraint, Constraint: parser.TableConstraint{Name: "pk", Definition: "PRIMARY KEY (id)"}},
		}},
	)
	if len(s.Tables[0].Constraints) != 1 {
		t.Errorf("expected 1 constraint")
	}
}

func TestApplyAddConstraint_Duplicate(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{
			TableName: "t",
			Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
			Constraints: []parser.TableConstraint{{Name: "pk", Definition: "PRIMARY KEY (id)"}},
		},
	)
	err := s.Apply(parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
		{Kind: parser.ActionAddConstraint, Constraint: parser.TableConstraint{Name: "pk", Definition: "PRIMARY KEY (id)"}},
	}})
	if err == nil {
		t.Error("expected error for duplicate constraint")
	}
}

func TestApplyDropConstraint(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{
			TableName:   "t",
			Columns:     []parser.ColumnDef{{Name: "id", DataType: "int"}},
			Constraints: []parser.TableConstraint{{Name: "pk", Definition: "PRIMARY KEY (id)"}},
		},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionDropConstraint, Constraint: parser.TableConstraint{Name: "pk"}},
		}},
	)
	if len(s.Tables[0].Constraints) != 0 {
		t.Error("expected 0 constraints after DROP CONSTRAINT")
	}
}

// ---- CREATE / DROP INDEX ----------------------------------------------------

func TestApplyCreateIndex(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Unique: true, Body: "(id)"},
	)
	if len(s.Indexes) != 1 || !s.Indexes[0].Unique {
		t.Errorf("Indexes: %+v", s.Indexes)
	}
}

func TestApplyCreateIndex_Duplicate(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"},
	)
	if err := s.Apply(parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"}); err == nil {
		t.Error("expected error")
	}
}

func TestApplyDropIndex(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"},
		parser.DropIndexStmt{IndexName: "idx"},
	)
	if len(s.Indexes) != 0 {
		t.Errorf("expected empty Indexes, got %d", len(s.Indexes))
	}
}

func TestApplyDropIndex_NotFound(t *testing.T) {
	s := newSchema()
	if err := s.Apply(parser.DropIndexStmt{IndexName: "no_idx"}); err == nil {
		t.Error("expected error")
	}
}

func TestApplyDropIndex_IfExists_NoError(t *testing.T) {
	s := newSchema()
	if err := s.Apply(parser.DropIndexStmt{IndexName: "no_idx", IfExists: true}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- CREATE / DROP SEQUENCE -------------------------------------------------

func TestApplyCreateSequence(t *testing.T) {
	s := newSchema()
	applyAll(t, s, parser.CreateSequenceStmt{SeqName: "seq", Body: "START 1"})
	if len(s.Sequences) != 1 || s.Sequences[0].Name != "seq" {
		t.Errorf("Sequences: %+v", s.Sequences)
	}
}

func TestApplyDropSequence(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateSequenceStmt{SeqName: "seq"},
		parser.DropSequenceStmt{SeqName: "seq"},
	)
	if len(s.Sequences) != 0 {
		t.Error("expected empty Sequences")
	}
}

func TestApplyDropSequence_NotFound(t *testing.T) {
	s := newSchema()
	if err := s.Apply(parser.DropSequenceStmt{SeqName: "no_seq"}); err == nil {
		t.Error("expected error")
	}
}

// ---- CREATE TYPE (enum) -----------------------------------------------------

func TestApplyCreateType(t *testing.T) {
	s := newSchema()
	applyAll(t, s, parser.CreateTypeStmt{TypeName: "status", Labels: []string{"active", "inactive"}})
	if len(s.Types) != 1 || s.Types[0].Name != "status" {
		t.Errorf("Types: %+v", s.Types)
	}
}

func TestApplyCreateType_Duplicate(t *testing.T) {
	s := newSchema()
	stmt := parser.CreateTypeStmt{TypeName: "status", Labels: []string{"a"}}
	applyAll(t, s, stmt)
	if err := s.Apply(stmt); err == nil {
		t.Error("expected error for duplicate CREATE TYPE")
	}
}

// ---- Unknown ----------------------------------------------------------------

func TestApplyUnknown(t *testing.T) {
	s := newSchema()
	applyAll(t, s, parser.UnknownStmt{Raw: "SET search_path = public"})
	if len(s.Unknowns) != 1 {
		t.Errorf("Unknowns: %v", s.Unknowns)
	}
}

// ---- RenameTo updates indexes -----------------------------------------------

func TestApplyRenameTo_UpdatesIndexes(t *testing.T) {
	s := newSchema()
	applyAll(t, s,
		parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}},
		parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"},
		parser.AlterTableStmt{TableName: "t", Actions: []parser.AlterAction{
			{Kind: parser.ActionRenameTo, NewName: "t2"},
		}},
	)
	if s.Indexes[0].TableName != "t2" {
		t.Errorf("index TableName: %q", s.Indexes[0].TableName)
	}
}
