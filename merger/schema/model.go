package schema

import "github.com/shunyamazaki/pg-ddl-merge/merger/parser"

type Sequence struct {
	Name string
	Body string
}

type EnumType struct {
	Name   string
	Labels []string
}

type Index struct {
	Name      string
	TableName string
	Unique    bool
	Body      string
}

// GenericObject holds a verbatim SQL definition for tracked objects
// (VIEW, MATERIALIZED VIEW, SCHEMA, EXTENSION, FUNCTION, PROCEDURE,
// TRIGGER, DOMAIN, POLICY, RULE).
type GenericObject struct {
	Kind parser.ObjectKind
	Name string // normalized key
	SQL  string // verbatim full SQL
}

// Schema holds the complete in-memory representation of the database schema.
type Schema struct {
	Sequences []Sequence
	Types     []EnumType
	Tables    []*Table
	Indexes   []Index
	Objects   []GenericObject
	Unknowns  []string

	tableIndex map[string]int // normalized name -> index in Tables
	seqIndex   map[string]int
	typeIndex  map[string]int
	indexIndex map[string]int
	objectIdx  map[string]int // "KIND:normname" -> index in Objects
}

type Table struct {
	Name        string
	Temporary   bool
	Unlogged    bool
	Columns     []parser.ColumnDef
	Constraints []parser.TableConstraint
}

func New() *Schema {
	return &Schema{
		tableIndex: map[string]int{},
		seqIndex:   map[string]int{},
		typeIndex:  map[string]int{},
		indexIndex: map[string]int{},
		objectIdx:  map[string]int{},
	}
}
