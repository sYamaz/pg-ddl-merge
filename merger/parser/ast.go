package parser

type StatementKind int

const (
	StmtCreateTable StatementKind = iota
	StmtAlterTable
	StmtDropTable
	StmtCreateIndex
	StmtDropIndex
	StmtCreateSequence
	StmtDropSequence
	StmtCreateType
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
	NotNull           bool
	Default           *string
	InlineConstraints []string
}

type CreateTableStmt struct {
	TableName   string
	Columns     []ColumnDef
	Constraints []TableConstraint
}

type AlterTableStmt struct {
	TableName string
	Actions   []AlterAction
}

type DropTableStmt struct {
	TableName string
	IfExists  bool
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

type UnknownStmt struct {
	Raw string
}

func (s CreateTableStmt) stmtKind() StatementKind  { return StmtCreateTable }
func (s AlterTableStmt) stmtKind() StatementKind   { return StmtAlterTable }
func (s DropTableStmt) stmtKind() StatementKind    { return StmtDropTable }
func (s CreateIndexStmt) stmtKind() StatementKind  { return StmtCreateIndex }
func (s DropIndexStmt) stmtKind() StatementKind    { return StmtDropIndex }
func (s CreateSequenceStmt) stmtKind() StatementKind { return StmtCreateSequence }
func (s DropSequenceStmt) stmtKind() StatementKind { return StmtDropSequence }
func (s CreateTypeStmt) stmtKind() StatementKind   { return StmtCreateType }
func (s UnknownStmt) stmtKind() StatementKind      { return StmtUnknown }
