package emitter

import (
	"strings"
	"testing"

	"github.com/sYamaz/pg-ddl-merge/merger/parser"
	"github.com/sYamaz/pg-ddl-merge/merger/schema"
)

func newSchema() *schema.Schema { return schema.New() }

func mustApply(t *testing.T, s *schema.Schema, stmt parser.Statement) {
	t.Helper()
	if err := s.Apply(stmt); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func ptr(v string) *string { return &v }

// ---- CREATE TABLE -----------------------------------------------------------

func TestEmit_SimpleTable(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "users",
		Columns: []parser.ColumnDef{
			{Name: "id", DataType: "bigint"},
			{Name: "name", DataType: "text"},
		},
	})

	got := Emit(s)
	if !strings.Contains(got, "CREATE TABLE users (") {
		t.Errorf("missing CREATE TABLE: %s", got)
	}
	if !strings.Contains(got, "id bigint") {
		t.Errorf("missing column id: %s", got)
	}
	if !strings.Contains(got, "name text") {
		t.Errorf("missing column name: %s", got)
	}
}

func TestEmit_TemporaryTable(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "tmp",
		Temporary: true,
		Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE TEMPORARY TABLE tmp (") {
		t.Errorf("expected TEMPORARY TABLE: %s", got)
	}
}

func TestEmit_UnloggedTable(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "logs",
		Unlogged:  true,
		Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE UNLOGGED TABLE logs (") {
		t.Errorf("expected UNLOGGED TABLE: %s", got)
	}
}

func TestEmit_ColumnNotNullDefault(t *testing.T) {
	s := newSchema()
	defVal := "0"
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "t",
		Columns: []parser.ColumnDef{
			{Name: "n", DataType: "int", NotNull: true, Default: &defVal},
		},
	})

	got := Emit(s)
	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("missing NOT NULL: %s", got)
	}
	if !strings.Contains(got, "DEFAULT 0") {
		t.Errorf("missing DEFAULT: %s", got)
	}
}

func TestEmit_TableWithConstraint(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "t",
		Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
		Constraints: []parser.TableConstraint{
			{Name: "pk", Definition: "PRIMARY KEY (id)"},
		},
	})

	got := Emit(s)
	if !strings.Contains(got, "CONSTRAINT pk PRIMARY KEY (id)") {
		t.Errorf("missing constraint: %s", got)
	}
}

func TestEmit_TableWithAnonConstraint(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "t",
		Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
		Constraints: []parser.TableConstraint{
			{Definition: "PRIMARY KEY (id)"},
		},
	})

	got := Emit(s)
	if !strings.Contains(got, "    PRIMARY KEY (id)") {
		t.Errorf("missing anon constraint: %s", got)
	}
}

func TestEmit_LastColumnNoTrailingComma(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{
		TableName: "t",
		Columns:   []parser.ColumnDef{{Name: "id", DataType: "int"}},
	})

	got := Emit(s)
	// "id int," would be wrong for a single-column table
	if strings.Contains(got, "id int,") {
		t.Errorf("last column should not have trailing comma: %s", got)
	}
}

// ---- Sequences --------------------------------------------------------------

func TestEmit_Sequence(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateSequenceStmt{SeqName: "seq", Body: "START 1"})

	got := Emit(s)
	if !strings.Contains(got, "CREATE SEQUENCE seq START 1;") {
		t.Errorf("missing sequence: %s", got)
	}
}

func TestEmit_SequenceNoBody(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateSequenceStmt{SeqName: "seq"})

	got := Emit(s)
	if !strings.Contains(got, "CREATE SEQUENCE seq;") {
		t.Errorf("missing sequence: %s", got)
	}
}

// ---- Enum types -------------------------------------------------------------

func TestEmit_EnumType(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTypeStmt{TypeName: "status", Labels: []string{"active", "inactive"}})

	got := Emit(s)
	if !strings.Contains(got, "CREATE TYPE status AS ENUM (") {
		t.Errorf("missing CREATE TYPE: %s", got)
	}
	if !strings.Contains(got, "'active'") {
		t.Errorf("missing label: %s", got)
	}
}

// ---- Indexes ----------------------------------------------------------------

func TestEmit_Index(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})
	mustApply(t, s, parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"})

	got := Emit(s)
	if !strings.Contains(got, "CREATE INDEX idx ON t (id);") {
		t.Errorf("missing index: %s", got)
	}
}

func TestEmit_UniqueIndex(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})
	mustApply(t, s, parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Unique: true, Body: "(id)"})

	got := Emit(s)
	if !strings.Contains(got, "CREATE UNIQUE INDEX") {
		t.Errorf("missing UNIQUE: %s", got)
	}
}

// ---- Unknowns ---------------------------------------------------------------

func TestEmit_Unknown(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.UnknownStmt{Raw: "SET search_path = public"})

	got := Emit(s)
	if !strings.Contains(got, "SET search_path = public;") {
		t.Errorf("missing unknown stmt: %s", got)
	}
}

// ---- Output ends with newline -----------------------------------------------

func TestEmit_EndsWithNewline(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})

	got := Emit(s)
	if !strings.HasSuffix(got, "\n") {
		t.Error("output should end with newline")
	}
}

// ---- Ordering: sequences → types → tables → indexes → unknowns --------------

func TestEmit_Ordering(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})
	mustApply(t, s, parser.CreateSequenceStmt{SeqName: "seq"})
	mustApply(t, s, parser.CreateTypeStmt{TypeName: "status", Labels: []string{"a"}})
	mustApply(t, s, parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"})

	got := Emit(s)
	seqPos := strings.Index(got, "CREATE SEQUENCE")
	typePos := strings.Index(got, "CREATE TYPE")
	tablePos := strings.Index(got, "CREATE TABLE")
	idxPos := strings.Index(got, "CREATE INDEX")

	if !(seqPos < typePos && typePos < tablePos && tablePos < idxPos) {
		t.Errorf("wrong ordering: seq=%d type=%d table=%d idx=%d\n%s", seqPos, typePos, tablePos, idxPos, got)
	}
}

// ---- Full ordering: schema → extension → seq → type → domain → table → index → func/proc → view/matview → trigger → policy/rule → unknowns

func TestEmit_FullOrdering(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateTableStmt{TableName: "t", Columns: []parser.ColumnDef{{Name: "id", DataType: "int"}}})
	mustApply(t, s, parser.CreateSequenceStmt{SeqName: "seq"})
	mustApply(t, s, parser.CreateTypeStmt{TypeName: "status", Labels: []string{"a"}})
	mustApply(t, s, parser.CreateIndexStmt{IndexName: "idx", TableName: "t", Body: "(id)"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjSchema, Name: "myschema", SQL: "CREATE SCHEMA myschema"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjExtension, Name: "pg_trgm", SQL: "CREATE EXTENSION pg_trgm"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjFunction, Name: "f", SQL: "CREATE FUNCTION f() RETURNS void AS $$ $$ LANGUAGE sql"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjView, Name: "v", SQL: "CREATE VIEW v AS SELECT 1"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjTrigger, Name: "trg_on_t", SQL: "CREATE TRIGGER trg AFTER INSERT ON t FOR EACH ROW EXECUTE FUNCTION f()"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjDomain, Name: "posint", SQL: "CREATE DOMAIN posint AS integer CHECK (VALUE > 0)"})
	mustApply(t, s, parser.CreateObjectStmt{Kind: parser.ObjPolicy, Name: "p_on_t", SQL: "CREATE POLICY p ON t USING (true)"})
	mustApply(t, s, parser.UnknownStmt{Raw: "SET search_path = public"})

	got := Emit(s)

	schemaPos := strings.Index(got, "CREATE SCHEMA")
	extPos := strings.Index(got, "CREATE EXTENSION")
	seqPos := strings.Index(got, "CREATE SEQUENCE")
	typePos := strings.Index(got, "CREATE TYPE")
	domainPos := strings.Index(got, "CREATE DOMAIN")
	tablePos := strings.Index(got, "CREATE TABLE")
	idxPos := strings.Index(got, "CREATE INDEX")
	funcPos := strings.Index(got, "CREATE FUNCTION")
	viewPos := strings.Index(got, "CREATE VIEW")
	trgPos := strings.Index(got, "CREATE TRIGGER")
	polPos := strings.Index(got, "CREATE POLICY")
	unkPos := strings.Index(got, "SET search_path")

	order := []struct {
		name string
		pos  int
	}{
		{"schema", schemaPos},
		{"extension", extPos},
		{"sequence", seqPos},
		{"type", typePos},
		{"domain", domainPos},
		{"table", tablePos},
		{"index", idxPos},
		{"function", funcPos},
		{"view", viewPos},
		{"trigger", trgPos},
		{"policy", polPos},
		{"unknown", unkPos},
	}

	for i := 0; i < len(order)-1; i++ {
		if order[i].pos < 0 {
			t.Errorf("%s not found in output", order[i].name)
			continue
		}
		if order[i+1].pos < 0 {
			t.Errorf("%s not found in output", order[i+1].name)
			continue
		}
		if order[i].pos >= order[i+1].pos {
			t.Errorf("ordering violation: %s (pos=%d) should come before %s (pos=%d)\n%s",
				order[i].name, order[i].pos, order[i+1].name, order[i+1].pos, got)
		}
	}
}

// ---- Generic objects --------------------------------------------------------

func TestEmit_GenericObject_View(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateObjectStmt{
		Kind: parser.ObjView,
		Name: "myview",
		SQL:  "CREATE VIEW myview AS SELECT 1",
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE VIEW myview AS SELECT 1;") {
		t.Errorf("missing view SQL: %s", got)
	}
}

func TestEmit_GenericObject_MatView(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateObjectStmt{
		Kind: parser.ObjMatView,
		Name: "mv",
		SQL:  "CREATE MATERIALIZED VIEW mv AS SELECT 1",
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE MATERIALIZED VIEW mv AS SELECT 1;") {
		t.Errorf("missing materialized view SQL: %s", got)
	}
}

func TestEmit_GenericObject_Function(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateObjectStmt{
		Kind: parser.ObjFunction,
		Name: "myfunc",
		SQL:  "CREATE FUNCTION myfunc() RETURNS void AS $$ $$ LANGUAGE sql",
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE FUNCTION myfunc()") {
		t.Errorf("missing function SQL: %s", got)
	}
}

func TestEmit_GenericObject_Schema(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateObjectStmt{
		Kind: parser.ObjSchema,
		Name: "myschema",
		SQL:  "CREATE SCHEMA myschema",
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE SCHEMA myschema;") {
		t.Errorf("missing schema: %s", got)
	}
}

func TestEmit_GenericObject_Extension(t *testing.T) {
	s := newSchema()
	mustApply(t, s, parser.CreateObjectStmt{
		Kind: parser.ObjExtension,
		Name: "pg_trgm",
		SQL:  "CREATE EXTENSION IF NOT EXISTS pg_trgm",
	})
	got := Emit(s)
	if !strings.Contains(got, "CREATE EXTENSION IF NOT EXISTS pg_trgm;") {
		t.Errorf("missing extension: %s", got)
	}
}
