package schema

import (
	"fmt"
	"os"
	"strings"

	"github.com/shunyamazaki/pg-ddl-merge/merger/parser"
)

func normIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.ToLower(s)
}

// Apply applies a parsed DDL statement to the schema model.
func (s *Schema) Apply(stmt parser.Statement) error {
	switch v := stmt.(type) {
	case parser.CreateTableStmt:
		return s.applyCreateTable(v)
	case parser.AlterTableStmt:
		return s.applyAlterTable(v)
	case parser.DropTableStmt:
		return s.applyDropTable(v)
	case parser.CreateIndexStmt:
		return s.applyCreateIndex(v)
	case parser.DropIndexStmt:
		return s.applyDropIndex(v)
	case parser.CreateSequenceStmt:
		return s.applyCreateSequence(v)
	case parser.DropSequenceStmt:
		return s.applyDropSequence(v)
	case parser.CreateTypeStmt:
		return s.applyCreateType(v)
	case parser.UnknownStmt:
		s.Unknowns = append(s.Unknowns, v.Raw)
	}
	return nil
}

func (s *Schema) applyCreateTable(v parser.CreateTableStmt) error {
	key := normIdent(v.TableName)
	if _, exists := s.tableIndex[key]; exists {
		return fmt.Errorf("duplicate CREATE TABLE: %s", v.TableName)
	}
	t := &Table{
		Name:        v.TableName,
		Temporary:   v.Temporary,
		Unlogged:    v.Unlogged,
		Columns:     v.Columns,
		Constraints: v.Constraints,
	}
	s.tableIndex[key] = len(s.Tables)
	s.Tables = append(s.Tables, t)
	return nil
}

func (s *Schema) applyAlterTable(v parser.AlterTableStmt) error {
	key := normIdent(v.TableName)
	idx, ok := s.tableIndex[key]
	if !ok {
		return fmt.Errorf("ALTER TABLE: table not found: %s", v.TableName)
	}
	t := s.Tables[idx]

	for _, action := range v.Actions {
		if err := s.applyAction(t, action, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) applyAction(t *Table, action parser.AlterAction, oldKey string) error {
	switch action.Kind {
	case parser.ActionAddColumn:
		col := action.ColDef
		if col == nil {
			return fmt.Errorf("ADD COLUMN: missing column definition")
		}
		colKey := normIdent(col.Name)
		for _, c := range t.Columns {
			if normIdent(c.Name) == colKey {
				return fmt.Errorf("ADD COLUMN: column already exists: %s", col.Name)
			}
		}
		t.Columns = append(t.Columns, *col)

	case parser.ActionDropColumn:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("DROP COLUMN: column not found: %s", action.Column)
		}
		t.Columns = append(t.Columns[:idx], t.Columns[idx+1:]...)

	case parser.ActionAlterColumnType:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("ALTER COLUMN TYPE: column not found: %s", action.Column)
		}
		t.Columns[idx].DataType = action.DataType

	case parser.ActionSetDefault:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("SET DEFAULT: column not found: %s", action.Column)
		}
		v := action.Default
		t.Columns[idx].Default = &v

	case parser.ActionDropDefault:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("DROP DEFAULT: column not found: %s", action.Column)
		}
		t.Columns[idx].Default = nil

	case parser.ActionSetNotNull:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("SET NOT NULL: column not found: %s", action.Column)
		}
		t.Columns[idx].NotNull = true

	case parser.ActionDropNotNull:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("DROP NOT NULL: column not found: %s", action.Column)
		}
		t.Columns[idx].NotNull = false

	case parser.ActionRenameColumn:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("RENAME COLUMN: column not found: %s", action.Column)
		}
		// warn if table has constraints that might reference the old name
		if len(t.Constraints) > 0 {
			fmt.Fprintf(os.Stderr, "warning: renaming column %s.%s — table constraints referencing this column by name are not updated\n", t.Name, action.Column)
		}
		t.Columns[idx].Name = action.NewName

	case parser.ActionRenameTo:
		newKey := normIdent(action.NewName)
		delete(s.tableIndex, oldKey)
		t.Name = action.NewName
		s.tableIndex[newKey] = s.tableIndex[oldKey]
		// re-set correctly
		for i, tbl := range s.Tables {
			if tbl == t {
				s.tableIndex[newKey] = i
				break
			}
		}
		// update indexes referencing old table name
		for i, idx := range s.Indexes {
			if normIdent(idx.TableName) == oldKey {
				s.Indexes[i].TableName = action.NewName
			}
		}

	case parser.ActionAddConstraint:
		if action.Constraint.Name != "" {
			for _, c := range t.Constraints {
				if normIdent(c.Name) == normIdent(action.Constraint.Name) {
					return fmt.Errorf("ADD CONSTRAINT: constraint already exists: %s", action.Constraint.Name)
				}
			}
		}
		t.Constraints = append(t.Constraints, action.Constraint)

	case parser.ActionDropConstraint:
		cKey := normIdent(action.Constraint.Name)
		idx := findConstraintIdx(t, cKey)
		if idx < 0 {
			return fmt.Errorf("DROP CONSTRAINT: constraint not found: %s", action.Constraint.Name)
		}
		t.Constraints = append(t.Constraints[:idx], t.Constraints[idx+1:]...)
	}
	return nil
}

func (s *Schema) applyDropTable(v parser.DropTableStmt) error {
	key := normIdent(v.TableName)
	idx, ok := s.tableIndex[key]
	if !ok {
		if v.IfExists {
			return nil
		}
		return fmt.Errorf("DROP TABLE: table not found: %s", v.TableName)
	}
	s.Tables = append(s.Tables[:idx], s.Tables[idx+1:]...)
	delete(s.tableIndex, key)
	// re-index remaining tables
	for i := idx; i < len(s.Tables); i++ {
		s.tableIndex[normIdent(s.Tables[i].Name)] = i
	}
	// remove associated indexes
	var kept []Index
	for _, ix := range s.Indexes {
		if normIdent(ix.TableName) != key {
			kept = append(kept, ix)
		}
	}
	s.Indexes = kept
	return nil
}

func (s *Schema) applyCreateIndex(v parser.CreateIndexStmt) error {
	key := normIdent(v.IndexName)
	if _, exists := s.indexIndex[key]; exists {
		return fmt.Errorf("duplicate CREATE INDEX: %s", v.IndexName)
	}
	s.indexIndex[key] = len(s.Indexes)
	s.Indexes = append(s.Indexes, Index{
		Name:      v.IndexName,
		TableName: v.TableName,
		Unique:    v.Unique,
		Body:      v.Body,
	})
	return nil
}

func (s *Schema) applyDropIndex(v parser.DropIndexStmt) error {
	key := normIdent(v.IndexName)
	idx, ok := s.indexIndex[key]
	if !ok {
		if v.IfExists {
			return nil
		}
		return fmt.Errorf("DROP INDEX: index not found: %s", v.IndexName)
	}
	s.Indexes = append(s.Indexes[:idx], s.Indexes[idx+1:]...)
	delete(s.indexIndex, key)
	for i := idx; i < len(s.Indexes); i++ {
		s.indexIndex[normIdent(s.Indexes[i].Name)] = i
	}
	return nil
}

func (s *Schema) applyCreateSequence(v parser.CreateSequenceStmt) error {
	key := normIdent(v.SeqName)
	if _, exists := s.seqIndex[key]; exists {
		return fmt.Errorf("duplicate CREATE SEQUENCE: %s", v.SeqName)
	}
	s.seqIndex[key] = len(s.Sequences)
	s.Sequences = append(s.Sequences, Sequence{Name: v.SeqName, Body: v.Body})
	return nil
}

func (s *Schema) applyDropSequence(v parser.DropSequenceStmt) error {
	key := normIdent(v.SeqName)
	idx, ok := s.seqIndex[key]
	if !ok {
		if v.IfExists {
			return nil
		}
		return fmt.Errorf("DROP SEQUENCE: sequence not found: %s", v.SeqName)
	}
	s.Sequences = append(s.Sequences[:idx], s.Sequences[idx+1:]...)
	delete(s.seqIndex, key)
	for i := idx; i < len(s.Sequences); i++ {
		s.seqIndex[normIdent(s.Sequences[i].Name)] = i
	}
	return nil
}

func (s *Schema) applyCreateType(v parser.CreateTypeStmt) error {
	key := normIdent(v.TypeName)
	if _, exists := s.typeIndex[key]; exists {
		return fmt.Errorf("duplicate CREATE TYPE: %s", v.TypeName)
	}
	s.typeIndex[key] = len(s.Types)
	s.Types = append(s.Types, EnumType{Name: v.TypeName, Labels: v.Labels})
	return nil
}

func findColumnIdx(t *Table, normName string) int {
	for i, c := range t.Columns {
		if normIdent(c.Name) == normName {
			return i
		}
	}
	return -1
}

func findConstraintIdx(t *Table, normName string) int {
	for i, c := range t.Constraints {
		if normIdent(c.Name) == normName {
			return i
		}
	}
	return -1
}
