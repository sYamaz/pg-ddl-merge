package schema

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/sYamaz/pg-ddl-merge/merger/parser"
)

// reAlterDomainAction extracts the action portion after "ALTER DOMAIN [IF EXISTS] name ".
var reAlterDomainAction = regexp.MustCompile(`(?i)^ALTER\s+DOMAIN\s+(?:IF\s+EXISTS\s+)?\S+\s+(.+)`)

func normIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.ToLower(s)
}

// objectKey returns the map key for an object: "KIND:normname".
func objectKey(kind parser.ObjectKind, name string) string {
	return strings.ToLower(string(kind)) + ":" + strings.ToLower(strings.TrimSpace(name))
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
	case parser.AlterIndexStmt:
		return s.applyAlterIndex(v)
	case parser.CreateSequenceStmt:
		return s.applyCreateSequence(v)
	case parser.DropSequenceStmt:
		return s.applyDropSequence(v)
	case parser.CreateTypeStmt:
		return s.applyCreateType(v)
	case parser.DropTypeStmt:
		return s.applyDropType(v)
	case parser.AlterTypeStmt:
		return s.applyAlterType(v)
	case parser.CreateObjectStmt:
		return s.applyCreateObject(v)
	case parser.DropObjectStmt:
		return s.applyDropObject(v)
	case parser.AlterSequenceStmt:
		return s.applyAlterSequence(v)
	case parser.AlterSequenceOptsStmt:
		return s.applyAlterSequenceOpts(v)
	case parser.TruncateStmt:
		s.applyTruncate(v)
	case parser.AlterObjectStmt:
		return s.applyAlterObject(v)
	case parser.AlterFunctionOptsStmt:
		return s.applyAlterFunctionOpts(v)
	case parser.AlterObjectOptsStmt:
		return s.applyAlterObjectOpts(v)
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
		PartitionBy: v.PartitionBy,
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
	case parser.ActionSkip:
		// silently do nothing
		return nil

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
		for _, ic := range t.Columns[idx].InlineConstraints {
			if strings.HasPrefix(strings.ToUpper(ic), "REFERENCES") {
				fmt.Fprintf(os.Stderr, "warning: ALTER TABLE %s: changing type of column %s which has an inline REFERENCES constraint — verify that the referenced column type matches\n", t.Name, action.Column)
				break
			}
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
		if idx >= 0 {
			t.Constraints = append(t.Constraints[:idx], t.Constraints[idx+1:]...)
			return nil
		}
		// Not found in named constraints — try inline constraints by PostgreSQL auto-naming convention.
		if dropInlineConstraintByAutoName(t, cKey) {
			return nil
		}
		if action.IfExists {
			return nil
		}
		return fmt.Errorf("DROP CONSTRAINT: constraint not found: %s", action.Constraint.Name)

	case parser.ActionAddGenerated:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("ADD GENERATED: column not found: %s", action.Column)
		}
		t.Columns[idx].InlineConstraints = replaceOrAppendInlineConstraint(
			t.Columns[idx].InlineConstraints, "GENERATED", action.GeneratedClause)

	case parser.ActionDropIdentity:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			if action.IfExists {
				return nil
			}
			return fmt.Errorf("DROP IDENTITY: column not found: %s", action.Column)
		}
		updated := removeInlineConstraint(t.Columns[idx].InlineConstraints, "GENERATED")
		if len(updated) == len(t.Columns[idx].InlineConstraints) && !action.IfExists {
			return fmt.Errorf("DROP IDENTITY: column %s has no identity", action.Column)
		}
		t.Columns[idx].InlineConstraints = updated

	case parser.ActionSetStorage:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("SET STORAGE: column not found: %s", action.Column)
		}
		t.Columns[idx].InlineConstraints = replaceOrAppendInlineConstraint(
			t.Columns[idx].InlineConstraints, "STORAGE", "STORAGE "+action.StorageType)

	case parser.ActionSetCompression:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("SET COMPRESSION: column not found: %s", action.Column)
		}
		t.Columns[idx].InlineConstraints = replaceOrAppendInlineConstraint(
			t.Columns[idx].InlineConstraints, "COMPRESSION", "COMPRESSION "+action.CompressionMethod)

	case parser.ActionRenameConstraint:
		cKey := normIdent(action.Constraint.Name)
		idx := findConstraintIdx(t, cKey)
		if idx < 0 {
			return fmt.Errorf("RENAME CONSTRAINT: constraint not found: %s", action.Constraint.Name)
		}
		t.Constraints[idx].Name = action.NewName

	case parser.ActionSetGenerated:
		colKey := normIdent(action.Column)
		idx := findColumnIdx(t, colKey)
		if idx < 0 {
			return fmt.Errorf("SET GENERATED: column not found: %s", action.Column)
		}
		updated := setGeneratedKind(t.Columns[idx].InlineConstraints, action.GeneratedKind)
		if updated == nil {
			return fmt.Errorf("SET GENERATED: column %s has no identity", action.Column)
		}
		t.Columns[idx].InlineConstraints = updated
	}
	return nil
}

func (s *Schema) applyDropTable(v parser.DropTableStmt) error {
	for _, name := range v.TableNames {
		if err := s.dropOneTable(name, v.IfExists); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) dropOneTable(name string, ifExists bool) error {
	key := normIdent(name)
	idx, ok := s.tableIndex[key]
	if !ok {
		// Also handle partition tables stored as GenericObjects.
		partKey := objectKey(parser.ObjPartition, name)
		if partIdx, partOk := s.objectIdx[partKey]; partOk {
			s.Objects = append(s.Objects[:partIdx], s.Objects[partIdx+1:]...)
			delete(s.objectIdx, partKey)
			for i := partIdx; i < len(s.Objects); i++ {
				s.objectIdx[objectKey(s.Objects[i].Kind, s.Objects[i].Name)] = i
			}
			return nil
		}
		if ifExists {
			return nil
		}
		return fmt.Errorf("DROP TABLE: table not found: %s", name)
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
	// rebuild indexIndex
	s.indexIndex = make(map[string]int, len(s.Indexes))
	for i, ix := range s.Indexes {
		s.indexIndex[normIdent(ix.Name)] = i
	}
	return nil
}

func (s *Schema) applyCreateIndex(v parser.CreateIndexStmt) error {
	key := normIdent(v.IndexName)
	newIdx := Index{
		Name:         v.IndexName,
		TableName:    v.TableName,
		Unique:       v.Unique,
		Concurrently: v.Concurrently,
		IfNotExists:  v.IfNotExists,
		Body:         v.Body,
	}
	if pos, exists := s.indexIndex[key]; exists {
		if v.IfNotExists {
			return nil // IF NOT EXISTS: skip silently, matching PostgreSQL semantics
		}
		fmt.Fprintf(os.Stderr, "warning: duplicate CREATE INDEX %q — overwriting with new definition\n", v.IndexName)
		s.Indexes[pos] = newIdx
		return nil
	}
	s.indexIndex[key] = len(s.Indexes)
	s.Indexes = append(s.Indexes, newIdx)
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

func (s *Schema) applyAlterIndex(v parser.AlterIndexStmt) error {
	oldKey := normIdent(v.IndexName)
	idx, ok := s.indexIndex[oldKey]
	if !ok {
		return fmt.Errorf("ALTER INDEX: index not found: %s", v.IndexName)
	}
	newKey := normIdent(v.NewName)
	s.Indexes[idx].Name = v.NewName
	delete(s.indexIndex, oldKey)
	s.indexIndex[newKey] = idx
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

func (s *Schema) applyDropType(v parser.DropTypeStmt) error {
	key := normIdent(v.TypeName)
	if idx, ok := s.typeIndex[key]; ok {
		// ENUM type
		s.Types = append(s.Types[:idx], s.Types[idx+1:]...)
		delete(s.typeIndex, key)
		for i := idx; i < len(s.Types); i++ {
			s.typeIndex[normIdent(s.Types[i].Name)] = i
		}
		return nil
	}
	// Try generic type (composite / range)
	objKey := objectKey(parser.ObjType, v.TypeName)
	if objIdx, ok := s.objectIdx[objKey]; ok {
		s.Objects = append(s.Objects[:objIdx], s.Objects[objIdx+1:]...)
		delete(s.objectIdx, objKey)
		for i := objIdx; i < len(s.Objects); i++ {
			s.objectIdx[objectKey(s.Objects[i].Kind, s.Objects[i].Name)] = i
		}
		return nil
	}
	if v.IfExists {
		return nil
	}
	return fmt.Errorf("DROP TYPE: type not found: %s", v.TypeName)
}

func (s *Schema) applyAlterType(v parser.AlterTypeStmt) error {
	key := normIdent(v.TypeName)
	idx, ok := s.typeIndex[key]
	if !ok {
		// For RENAME TO, also accept generic types (composite / range)
		if v.Action.Kind == parser.AlterTypeRenameTo {
			objKey := objectKey(parser.ObjType, v.TypeName)
			if objIdx, objOk := s.objectIdx[objKey]; objOk {
				newObjKey := objectKey(parser.ObjType, v.Action.NewName)
				s.Objects[objIdx].Name = normIdent(v.Action.NewName)
				s.Objects[objIdx].SQL = replaceTypeNameInSQL(s.Objects[objIdx].SQL, v.TypeName, v.Action.NewName)
				delete(s.objectIdx, objKey)
				s.objectIdx[newObjKey] = objIdx
				return nil
			}
		}
		return fmt.Errorf("ALTER TYPE: type not found: %s", v.TypeName)
	}

	switch v.Action.Kind {
	case parser.AlterTypeAddValue:
		label := v.Action.Value
		// Check for existing label
		for _, l := range s.Types[idx].Labels {
			if l == label {
				if v.Action.IfNotExists {
					return nil
				}
				return fmt.Errorf("ALTER TYPE ADD VALUE: label already exists: %s", label)
			}
		}
		labels := s.Types[idx].Labels
		if v.Action.Before != "" {
			pos := findLabelIdx(labels, v.Action.Before)
			if pos < 0 {
				return fmt.Errorf("ALTER TYPE ADD VALUE: BEFORE label not found: %s", v.Action.Before)
			}
			labels = insertLabel(labels, pos, label)
		} else if v.Action.After != "" {
			pos := findLabelIdx(labels, v.Action.After)
			if pos < 0 {
				return fmt.Errorf("ALTER TYPE ADD VALUE: AFTER label not found: %s", v.Action.After)
			}
			labels = insertLabel(labels, pos+1, label)
		} else {
			labels = append(labels, label)
		}
		s.Types[idx].Labels = labels

	case parser.AlterTypeRenameValue:
		labels := s.Types[idx].Labels
		pos := findLabelIdx(labels, v.Action.Value)
		if pos < 0 {
			return fmt.Errorf("ALTER TYPE RENAME VALUE: label not found: %s", v.Action.Value)
		}
		labels[pos] = v.Action.NewValue

	case parser.AlterTypeRenameTo:
		newKey := normIdent(v.Action.NewName)
		s.Types[idx].Name = v.Action.NewName
		delete(s.typeIndex, key)
		s.typeIndex[newKey] = idx
	}
	return nil
}

func findLabelIdx(labels []string, label string) int {
	for i, l := range labels {
		if l == label {
			return i
		}
	}
	return -1
}

func insertLabel(labels []string, pos int, label string) []string {
	labels = append(labels, "")
	copy(labels[pos+1:], labels[pos:])
	labels[pos] = label
	return labels
}

func (s *Schema) applyCreateObject(v parser.CreateObjectStmt) error {
	key := objectKey(v.Kind, v.Name)
	if idx, exists := s.objectIdx[key]; exists {
		if v.OrReplace {
			s.Objects[idx].SQL = v.SQL
			s.Objects[idx].PostAlters = nil // fresh definition clears prior ALTERs
			return nil
		}
		return fmt.Errorf("duplicate CREATE %s: %s", v.Kind, v.Name)
	}
	s.objectIdx[key] = len(s.Objects)
	s.Objects = append(s.Objects, GenericObject{
		Kind: v.Kind,
		Name: v.Name,
		SQL:  v.SQL,
	})
	return nil
}

func (s *Schema) applyDropObject(v parser.DropObjectStmt) error {
	key := objectKey(v.Kind, v.Name)
	idx, ok := s.objectIdx[key]
	if !ok {
		if v.IfExists {
			return nil
		}
		return fmt.Errorf("DROP %s: not found: %s", v.Kind, v.Name)
	}
	s.Objects = append(s.Objects[:idx], s.Objects[idx+1:]...)
	delete(s.objectIdx, key)
	// re-index remaining objects
	for i := idx; i < len(s.Objects); i++ {
		k := objectKey(s.Objects[i].Kind, s.Objects[i].Name)
		s.objectIdx[k] = i
	}
	return nil
}

// seqBodySpecs defines how to upsert an option into a CREATE SEQUENCE body string.
// Key = SequenceOption.Kind. The removeRe strips all variants of the same family;
// newFmt is the text to append (%s for the value, or a literal string for flag options).
var seqBodySpecs = map[string]struct {
	removeRe *regexp.Regexp
	newFmt   string
}{
	// Use -?\d+ (not \S+) for numeric options so we never consume the next keyword.
	"INCREMENT BY": {regexp.MustCompile(`(?i)\s*INCREMENT\s+(?:BY\s+)?-?\d+`), "INCREMENT BY %s"},
	"NO MINVALUE":  {regexp.MustCompile(`(?i)\s*(?:NO\s+)?MINVALUE(?:\s+-?\d+)?`), "NO MINVALUE"},
	"MINVALUE":     {regexp.MustCompile(`(?i)\s*(?:NO\s+)?MINVALUE(?:\s+-?\d+)?`), "MINVALUE %s"},
	"NO MAXVALUE":  {regexp.MustCompile(`(?i)\s*(?:NO\s+)?MAXVALUE(?:\s+-?\d+)?`), "NO MAXVALUE"},
	"MAXVALUE":     {regexp.MustCompile(`(?i)\s*(?:NO\s+)?MAXVALUE(?:\s+-?\d+)?`), "MAXVALUE %s"},
	"START WITH":   {regexp.MustCompile(`(?i)\s*START(?:\s+WITH)?\s+-?\d+`), "START WITH %s"},
	"CACHE":        {regexp.MustCompile(`(?i)\s*CACHE\s+-?\d+`), "CACHE %s"},
	"NO CYCLE":     {regexp.MustCompile(`(?i)\s*(?:NO\s+)?CYCLE\b`), "NO CYCLE"},
	"CYCLE":        {regexp.MustCompile(`(?i)\s*(?:NO\s+)?CYCLE\b`), "CYCLE"},
	"OWNED BY":     {regexp.MustCompile(`(?i)\s*OWNED\s+BY\s+\S+`), "OWNED BY %s"},
	"AS":           {regexp.MustCompile(`(?i)\s*\bAS\s+\S+`), "AS %s"},
	"SET":          {regexp.MustCompile(`(?i)\s*SET\s+(?:LOGGED|UNLOGGED)\b`), "SET %s"},
	// "RESTART" intentionally absent: runtime state, no body update needed.
}

func applySeqBodyOption(body string, opt parser.SequenceOption) string {
	spec, ok := seqBodySpecs[opt.Kind]
	if !ok {
		return body // RESTART or unrecognized — skip
	}
	body = spec.removeRe.ReplaceAllString(body, "")
	body = strings.TrimSpace(body)
	var newText string
	if strings.Contains(spec.newFmt, "%s") {
		newText = fmt.Sprintf(spec.newFmt, opt.Value)
	} else {
		newText = spec.newFmt
	}
	if body == "" {
		return newText
	}
	return body + " " + newText
}

func (s *Schema) applyAlterSequenceOpts(v parser.AlterSequenceOptsStmt) error {
	key := normIdent(v.SeqName)
	idx, ok := s.seqIndex[key]
	if !ok {
		return fmt.Errorf("ALTER SEQUENCE: sequence not found: %s", v.SeqName)
	}
	for _, opt := range v.Opts {
		s.Sequences[idx].Body = applySeqBodyOption(s.Sequences[idx].Body, opt)
	}
	return nil
}

func (s *Schema) applyTruncate(v parser.TruncateStmt) {
	// Build a dedup key from the sorted normalized table names.
	sorted := make([]string, len(v.Tables))
	for i, t := range v.Tables {
		sorted[i] = normIdent(t)
	}
	sort.Strings(sorted)
	key := strings.Join(sorted, ",")

	if idx, ok := s.truncateIdx[key]; ok {
		s.Truncates[idx] = v // last one wins
	} else {
		s.truncateIdx[key] = len(s.Truncates)
		s.Truncates = append(s.Truncates, v)
	}
}

func (s *Schema) applyAlterSequence(v parser.AlterSequenceStmt) error {
	oldKey := normIdent(v.SeqName)
	idx, ok := s.seqIndex[oldKey]
	if !ok {
		return fmt.Errorf("ALTER SEQUENCE: sequence not found: %s", v.SeqName)
	}
	newKey := normIdent(v.NewName)
	s.Sequences[idx].Name = v.NewName
	delete(s.seqIndex, oldKey)
	s.seqIndex[newKey] = idx
	return nil
}

func (s *Schema) applyAlterObject(v parser.AlterObjectStmt) error {
	oldKey := objectKey(v.Kind, v.OldName)
	idx, ok := s.objectIdx[oldKey]
	if !ok {
		return fmt.Errorf("ALTER %s: not found: %s", v.Kind, v.OldName)
	}
	newKey := objectKey(v.Kind, v.NewName)
	s.Objects[idx].Name = v.NewName
	delete(s.objectIdx, oldKey)
	s.objectIdx[newKey] = idx
	return nil
}

func (s *Schema) applyAlterFunctionOpts(v parser.AlterFunctionOptsStmt) error {
	key := objectKey(v.Kind, v.Name)
	idx, ok := s.objectIdx[key]
	if !ok {
		// Function not found in schema (may be defined externally) — fall back to pass-through.
		s.Unknowns = append(s.Unknowns, v.SQL)
		return nil
	}
	s.Objects[idx].PostAlters = append(s.Objects[idx].PostAlters, v.SQL)
	return nil
}

var reDomainOwnerOrSchema = regexp.MustCompile(`(?i)^\s*(OWNER\s+TO|SET\s+SCHEMA)\s+`)

func (s *Schema) applyAlterObjectOpts(v parser.AlterObjectOptsStmt) error {
	// For DOMAIN: OWNER TO and SET SCHEMA are not tracked — warn and skip.
	if v.Kind == parser.ObjDomain {
		// Extract the action portion after "ALTER DOMAIN name "
		if m := reAlterDomainAction.FindStringSubmatch(v.SQL); m != nil {
			if reDomainOwnerOrSchema.MatchString(m[1]) {
				fmt.Fprintf(os.Stderr, "warning: ALTER DOMAIN OWNER TO / SET SCHEMA is not tracked: %s\n", v.SQL)
				return nil
			}
		}
	}
	key := objectKey(v.Kind, v.Name)
	idx, ok := s.objectIdx[key]
	if !ok {
		// Object not found in schema — fall back to pass-through.
		s.Unknowns = append(s.Unknowns, v.SQL)
		return nil
	}
	s.Objects[idx].PostAlters = append(s.Objects[idx].PostAlters, v.SQL)
	return nil
}

// replaceTypeNameInSQL replaces the type name token in "CREATE TYPE <name> ..." SQL.
func replaceTypeNameInSQL(sql, oldName, newName string) string {
	upper := strings.ToUpper(sql)
	const prefix = "CREATE TYPE "
	pos := strings.Index(upper, prefix)
	if pos < 0 {
		return sql
	}
	start := pos + len(prefix)
	rest := sql[start:]
	end := strings.IndexAny(rest, " \t\n\r")
	if end < 0 {
		end = len(rest)
	}
	if normIdent(rest[:end]) == normIdent(oldName) {
		return sql[:start] + newName + rest[end:]
	}
	return sql
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

// replaceOrAppendInlineConstraint replaces the first inline constraint that starts
// with prefix (case-insensitive) with newVal. If none exists, newVal is appended.
func replaceOrAppendInlineConstraint(constraints []string, prefix, newVal string) []string {
	upperPrefix := strings.ToUpper(prefix)
	for i, c := range constraints {
		if strings.HasPrefix(strings.ToUpper(c), upperPrefix) {
			constraints[i] = newVal
			return constraints
		}
	}
	return append(constraints, newVal)
}

// removeInlineConstraint removes the first inline constraint that starts with prefix
// (case-insensitive). Returns the slice unchanged if no match is found.
func removeInlineConstraint(constraints []string, prefix string) []string {
	upperPrefix := strings.ToUpper(prefix)
	for i, c := range constraints {
		if strings.HasPrefix(strings.ToUpper(c), upperPrefix) {
			return append(constraints[:i], constraints[i+1:]...)
		}
	}
	return constraints
}

// setGeneratedKind replaces "ALWAYS" or "BY DEFAULT" in a GENERATED identity inline
// constraint to match kind ("ALWAYS" or "BY DEFAULT"). Returns nil if no GENERATED
// constraint is found.
var reGeneratedAlwaysByDefault = regexp.MustCompile(`(?i)(GENERATED\s+)(ALWAYS|BY\s+DEFAULT)(\s+AS\s+IDENTITY.*)`)

func setGeneratedKind(constraints []string, kind string) []string {
	for i, c := range constraints {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(c)), "GENERATED") {
			updated := reGeneratedAlwaysByDefault.ReplaceAllStringFunc(c, func(match string) string {
				m := reGeneratedAlwaysByDefault.FindStringSubmatch(match)
				if m == nil {
					return match
				}
				return m[1] + kind + m[3]
			})
			result := make([]string, len(constraints))
			copy(result, constraints)
			result[i] = updated
			return result
		}
	}
	return nil
}

// dropInlineConstraintByAutoName tries to remove an inline constraint whose
// PostgreSQL auto-generated name matches cKey.
//
// PostgreSQL naming conventions:
//   - {table}_pkey          → column-level PRIMARY KEY
//   - {table}_{col}_key     → column-level UNIQUE
//   - {table}_{col}_fkey    → column-level REFERENCES (foreign key)
//   - {table}_{col}_check   → column-level CHECK
//
// Returns true if a matching inline constraint was found and removed.
func dropInlineConstraintByAutoName(t *Table, cKey string) bool {
	tKey := normIdent(t.Name)

	// {table}_pkey → inline PRIMARY KEY (only one per table, column unknown)
	if cKey == tKey+"_pkey" {
		for i := range t.Columns {
			before := len(t.Columns[i].InlineConstraints)
			t.Columns[i].InlineConstraints = removeInlineConstraint(t.Columns[i].InlineConstraints, "PRIMARY KEY")
			if len(t.Columns[i].InlineConstraints) < before {
				return true
			}
		}
	}

	// Per-column patterns: {table}_{col}_{suffix}
	for i, col := range t.Columns {
		colKey := normIdent(col.Name)
		switch cKey {
		case tKey + "_" + colKey + "_key":
			t.Columns[i].InlineConstraints = removeInlineConstraint(t.Columns[i].InlineConstraints, "UNIQUE")
			return true
		case tKey + "_" + colKey + "_fkey":
			t.Columns[i].InlineConstraints = removeInlineConstraint(t.Columns[i].InlineConstraints, "REFERENCES")
			return true
		case tKey + "_" + colKey + "_check":
			t.Columns[i].InlineConstraints = removeInlineConstraint(t.Columns[i].InlineConstraints, "CHECK")
			return true
		}
	}

	return false
}
