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

// Schema holds the complete in-memory representation of the database schema.
type Schema struct {
	Sequences  []Sequence
	Types      []EnumType
	Tables     []*Table
	Indexes    []Index
	Unknowns   []string

	tableIndex map[string]int // normalized name -> index in Tables
	seqIndex   map[string]int
	typeIndex  map[string]int
	indexIndex map[string]int
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
	}
}
