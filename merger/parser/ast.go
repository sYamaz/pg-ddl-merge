package parser

type StatementKind int

const (
	StmtCreateTable StatementKind = iota
	StmtAlterTable
	StmtDropTable
	StmtCreateIndex
	StmtDropIndex
	StmtAlterIndex
	StmtCreateSequence
	StmtDropSequence
	StmtCreateType
	StmtDropType
	StmtAlterType
	StmtCreateObject
	StmtDropObject
	StmtAlterSequence
	StmtAlterSequenceOpts
	StmtAlterObject
	StmtTruncate
	StmtUnknown
)

type Statement interface {
	stmtKind() StatementKind
}

type AlterActionKind int

const (
	ActionAddColumn AlterActionKind = iota
	ActionDropColumn
	ActionAlterColumnType
	ActionSetDefault
	ActionDropDefault
	ActionSetNotNull
	ActionDropNotNull
	ActionRenameColumn
	ActionRenameTo
	ActionAddConstraint
	ActionDropConstraint
	ActionSkip // unrecognized action — silently ignored
)

type AlterAction struct {
	Kind       AlterActionKind
	Column     string
	NewName    string
	DataType   string
	Default    string
	Constraint TableConstraint
	ColDef     *ColumnDef // populated for ActionAddColumn
}

type TableConstraint struct {
	Name       string
	Definition string
}

type ColumnDef struct {
	Name              string
	DataType          string
	Collation         string // e.g. `"ja-x-icu"` or `pg_catalog.default`
	NotNull           bool
	Default           *string
	InlineConstraints []string
}

type CreateTableStmt struct {
	TableName   string
	Temporary   bool
	Unlogged    bool
	Columns     []ColumnDef
	Constraints []TableConstraint
	PartitionBy string // e.g. "RANGE (id)" — empty for non-partitioned tables
}

type AlterTableStmt struct {
	TableName string
	Actions   []AlterAction
}

type DropTableStmt struct {
	TableNames []string
	IfExists   bool
}

type CreateIndexStmt struct {
	IndexName string
	TableName string
	Unique    bool
	Body      string // everything after "ON tablename"
}

type DropIndexStmt struct {
	IndexName string
	IfExists  bool
}

type AlterIndexStmt struct {
	IndexName string
	NewName   string
}

type CreateSequenceStmt struct {
	SeqName string
	Body    string
}

type DropSequenceStmt struct {
	SeqName  string
	IfExists bool
}

type CreateTypeStmt struct {
	TypeName string
	Labels   []string
}

type DropTypeStmt struct {
	TypeName string
	IfExists bool
}

type AlterTypeActionKind int

const (
	AlterTypeAddValue AlterTypeActionKind = iota
	AlterTypeRenameValue
	AlterTypeRenameTo
)

type AlterTypeAction struct {
	Kind        AlterTypeActionKind
	Value       string  // label to add / old label to rename
	NewValue    string  // new label name (for RENAME VALUE)
	NewName     string  // new type name (for RENAME TO)
	IfNotExists bool    // ADD VALUE IF NOT EXISTS
	Before      string  // BEFORE existing_label
	After       string  // AFTER existing_label
}

type AlterTypeStmt struct {
	TypeName string
	Action   AlterTypeAction
}

// ObjectKind identifies the kind of generic tracked DDL object.
type ObjectKind string

const (
	ObjView      ObjectKind = "VIEW"
	ObjMatView   ObjectKind = "MATERIALIZED VIEW"
	ObjSchema    ObjectKind = "SCHEMA"
	ObjExtension ObjectKind = "EXTENSION"
	ObjFunction  ObjectKind = "FUNCTION"
	ObjProcedure ObjectKind = "PROCEDURE"
	ObjTrigger   ObjectKind = "TRIGGER"
	ObjDomain    ObjectKind = "DOMAIN"
	ObjPolicy    ObjectKind = "POLICY"
	ObjRule      ObjectKind = "RULE"
	ObjType      ObjectKind = "TYPE"      // composite / range / other non-ENUM types
	ObjPartition ObjectKind = "PARTITION" // CREATE TABLE ... PARTITION OF
)

// CreateObjectStmt represents CREATE for any generic tracked object.
type CreateObjectStmt struct {
	Kind      ObjectKind
	Name      string // normalized key (e.g. "name_on_table" for triggers/policies/rules)
	OrReplace bool
	SQL       string // verbatim full SQL
}

// DropObjectStmt represents DROP for any generic tracked object.
type DropObjectStmt struct {
	Kind     ObjectKind
	Name     string // normalized key
	IfExists bool
}

// AlterSequenceStmt represents ALTER SEQUENCE ... RENAME TO.
type AlterSequenceStmt struct {
	SeqName string
	NewName string
}

// SequenceOption represents a single option in ALTER SEQUENCE ... (non-RENAME).
// Kind values: "INCREMENT BY", "MINVALUE", "NO MINVALUE", "MAXVALUE", "NO MAXVALUE",
// "START WITH", "CACHE", "CYCLE", "NO CYCLE", "OWNED BY", "AS", "SET", "RESTART".
type SequenceOption struct {
	Kind  string
	Value string // numeric/identifier value; empty for flag-only kinds (NO MINVALUE etc.)
}

// AlterSequenceOptsStmt represents ALTER SEQUENCE with options other than RENAME TO.
// The schema layer applies each option to the corresponding sequence Body.
type AlterSequenceOptsStmt struct {
	SeqName string
	Opts    []SequenceOption
}

// TruncateStmt represents TRUNCATE TABLE name [, ...] [RESTART IDENTITY] [CASCADE].
type TruncateStmt struct {
	Tables          []string
	RestartIdentity bool
	Cascade         bool
}

// AlterObjectStmt represents ALTER <generic-object> ... RENAME TO.
// OldName and NewName use the same normalized key format as CreateObjectStmt.Name
// (e.g. "trigname_on_tablename" for triggers/policies/rules).
type AlterObjectStmt struct {
	Kind    ObjectKind
	OldName string
	NewName string
}

type UnknownStmt struct {
	Raw string
}

func (s CreateTableStmt) stmtKind() StatementKind   { return StmtCreateTable }
func (s AlterTableStmt) stmtKind() StatementKind    { return StmtAlterTable }
func (s DropTableStmt) stmtKind() StatementKind     { return StmtDropTable }
func (s CreateIndexStmt) stmtKind() StatementKind   { return StmtCreateIndex }
func (s DropIndexStmt) stmtKind() StatementKind     { return StmtDropIndex }
func (s AlterIndexStmt) stmtKind() StatementKind    { return StmtAlterIndex }
func (s CreateSequenceStmt) stmtKind() StatementKind { return StmtCreateSequence }
func (s DropSequenceStmt) stmtKind() StatementKind  { return StmtDropSequence }
func (s CreateTypeStmt) stmtKind() StatementKind    { return StmtCreateType }
func (s DropTypeStmt) stmtKind() StatementKind      { return StmtDropType }
func (s AlterTypeStmt) stmtKind() StatementKind     { return StmtAlterType }
func (s CreateObjectStmt) stmtKind() StatementKind  { return StmtCreateObject }
func (s DropObjectStmt) stmtKind() StatementKind    { return StmtDropObject }
func (s AlterSequenceStmt) stmtKind() StatementKind     { return StmtAlterSequence }
func (s AlterSequenceOptsStmt) stmtKind() StatementKind { return StmtAlterSequenceOpts }
func (s AlterObjectStmt) stmtKind() StatementKind       { return StmtAlterObject }
func (s TruncateStmt) stmtKind() StatementKind          { return StmtTruncate }
func (s UnknownStmt) stmtKind() StatementKind           { return StmtUnknown }
