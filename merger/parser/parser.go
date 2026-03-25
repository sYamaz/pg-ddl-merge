package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reCreateTable    = regexp.MustCompile(`(?i)^CREATE\s+(?:(?:GLOBAL|LOCAL)\s+)?(?:(TEMPORARY|TEMP|UNLOGGED)\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	reCollate        = regexp.MustCompile(`(?i)\s+COLLATE\s+("(?:[^"]|"")*"|\S+)`)
	reAlterTable     = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(?:ONLY\s+)?(\S+)\s+(.+)`)
	reDropTable      = regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reCreateIndex    = regexp.MustCompile(`(?i)^CREATE\s+(UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*(.*)`)
	reDropIndex      = regexp.MustCompile(`(?i)^DROP\s+INDEX\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reCreateSequence = regexp.MustCompile(`(?i)^CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)(.*)`)
	reDropSequence   = regexp.MustCompile(`(?i)^DROP\s+SEQUENCE\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reCreateTypeEnum = regexp.MustCompile(`(?i)^CREATE\s+TYPE\s+(\S+)\s+AS\s+ENUM\s*\((.+)\)`)
)

// normalizeIdent removes surrounding double-quotes and lowercases for key lookup.
func normalizeIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.ToLower(s)
}

// Parse parses a single SQL statement string into a Statement.
func Parse(sql string) (Statement, error) {
	sql = strings.TrimSpace(sql)
	upper := strings.ToUpper(sql)

	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"),
		strings.HasPrefix(upper, "CREATE TEMPORARY TABLE"),
		strings.HasPrefix(upper, "CREATE TEMP TABLE"),
		strings.HasPrefix(upper, "CREATE UNLOGGED TABLE"),
		strings.HasPrefix(upper, "CREATE GLOBAL TEMPORARY TABLE"),
		strings.HasPrefix(upper, "CREATE LOCAL TEMPORARY TABLE"),
		strings.HasPrefix(upper, "CREATE GLOBAL TEMP TABLE"),
		strings.HasPrefix(upper, "CREATE LOCAL TEMP TABLE"):
		return parseCreateTable(sql)
	case strings.HasPrefix(upper, "ALTER TABLE"):
		return parseAlterTable(sql)
	case strings.HasPrefix(upper, "DROP TABLE"):
		return parseDropTable(sql)
	case strings.HasPrefix(upper, "CREATE UNIQUE INDEX"), strings.HasPrefix(upper, "CREATE INDEX"):
		return parseCreateIndex(sql)
	case strings.HasPrefix(upper, "DROP INDEX"):
		return parseDropIndex(sql)
	case strings.HasPrefix(upper, "CREATE SEQUENCE"):
		return parseCreateSequence(sql)
	case strings.HasPrefix(upper, "DROP SEQUENCE"):
		return parseDropSequence(sql)
	case strings.HasPrefix(upper, "CREATE TYPE"):
		return parseCreateType(sql)
	default:
		return UnknownStmt{Raw: sql}, nil
	}
}

func parseCreateTable(sql string) (Statement, error) {
	m := reCreateTable.FindStringSubmatchIndex(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse CREATE TABLE: %s", sql[:min(len(sql), 60)])
	}
	// group 1: TEMPORARY|TEMP|UNLOGGED (m[2]:m[3], -1 if absent)
	// group 2: table name (m[4]:m[5])
	var temporary, unlogged bool
	if m[2] >= 0 {
		mod := strings.ToUpper(sql[m[2]:m[3]])
		temporary = mod == "TEMPORARY" || mod == "TEMP"
		unlogged = mod == "UNLOGGED"
	}
	tableName := sql[m[4]:m[5]]

	// find the body between outermost parens
	start := m[1] - 1 // position of '('
	body, err := extractParenBody(sql, start)
	if err != nil {
		return nil, fmt.Errorf("CREATE TABLE %s: %w", tableName, err)
	}

	cols, constraints, err := parseColumnList(body)
	if err != nil {
		return nil, fmt.Errorf("CREATE TABLE %s: %w", tableName, err)
	}

	return CreateTableStmt{
		TableName:   tableName,
		Temporary:   temporary,
		Unlogged:    unlogged,
		Columns:     cols,
		Constraints: constraints,
	}, nil
}

// extractParenBody returns the content between the first '(' and its matching ')'.
func extractParenBody(sql string, start int) (string, error) {
	if start >= len(sql) || sql[start] != '(' {
		return "", fmt.Errorf("expected '(' at position %d", start)
	}
	depth := 0
	for i := start; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return sql[start+1 : i], nil
			}
		case '\'':
			// skip string literal
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i++
					} else {
						break
					}
				}
				i++
			}
		}
	}
	return "", fmt.Errorf("unmatched parenthesis")
}

// splitAtDepthZeroCommas splits s by commas that are at paren depth 0.
func splitAtDepthZeroCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	inQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			if ch == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
				} else {
					inQuote = false
				}
			}
			continue
		}
		switch ch {
		case '\'':
			inQuote = true
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(s[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

var (
	reConstraintLine = regexp.MustCompile(`(?i)^CONSTRAINT\s+(\S+)\s+(.+)`)
	reInlineKeywords = regexp.MustCompile(`(?i)^(PRIMARY\s+KEY|UNIQUE|FOREIGN\s+KEY|CHECK|EXCLUDE|LIKE)\s*`)
)

func parseColumnList(body string) ([]ColumnDef, []TableConstraint, error) {
	parts := splitAtDepthZeroCommas(body)
	var cols []ColumnDef
	var constraints []TableConstraint

	for _, part := range parts {
		part = strings.TrimSpace(part)
		upper := strings.ToUpper(part)

		if m := reConstraintLine.FindStringSubmatch(part); m != nil {
			constraints = append(constraints, TableConstraint{
				Name:       m[1],
				Definition: strings.TrimSpace(m[2]),
			})
			continue
		}

		if reInlineKeywords.MatchString(upper) {
			constraints = append(constraints, TableConstraint{
				Definition: part,
			})
			continue
		}

		col, err := parseColumnDef(part)
		if err != nil {
			return nil, nil, err
		}
		cols = append(cols, col)
	}
	return cols, constraints, nil
}

var reDefaultVal = regexp.MustCompile(`(?i)\s+DEFAULT\s+`)

func parseColumnDef(s string) (ColumnDef, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ColumnDef{}, fmt.Errorf("empty column definition")
	}

	// Extract name (first token, possibly quoted)
	name, rest := extractIdent(s)
	if name == "" {
		return ColumnDef{}, fmt.Errorf("cannot parse column name from: %s", s)
	}

	col := ColumnDef{Name: name}
	upper := strings.ToUpper(rest)

	// Parse COLLATE collation_name (comes right after the data type)
	if m := reCollate.FindStringSubmatchIndex(rest); m != nil {
		col.Collation = rest[m[2]:m[3]]
		rest = rest[:m[0]] + rest[m[1]:]
		upper = strings.ToUpper(rest)
	}

	// Parse NOT NULL / NULL
	if strings.Contains(upper, "NOT NULL") {
		col.NotNull = true
		rest = strings.TrimSpace(replaceCI(rest, "NOT NULL", ""))
		upper = strings.ToUpper(rest)
	} else {
		// Explicit NULL keyword (redundant but valid) — strip it so it doesn't
		// contaminate the data type string.
		rest = strings.TrimSpace(replaceCI(rest, " NULL", ""))
		upper = strings.ToUpper(rest)
	}

	// Parse DEFAULT (skip if it's part of "GENERATED BY DEFAULT AS IDENTITY")
	if idx := indexCI(rest, " DEFAULT "); idx >= 0 && !strings.EqualFold(strings.TrimSpace(rest[max(0, idx-3):idx]), "BY") {
		before := rest[:idx]
		after := rest[idx+len(" DEFAULT "):]
		// the default value extends to the next keyword or end
		defVal, remainder := extractDefaultValue(after)
		col.Default = &defVal
		rest = strings.TrimSpace(before) + " " + strings.TrimSpace(remainder)
		rest = strings.TrimSpace(rest)
		upper = strings.ToUpper(rest)
	}

	// Remaining inline constraints (PRIMARY KEY, REFERENCES, UNIQUE, CHECK, GENERATED)
	var inlineConstraints []string
	dataType, inline := splitTypeAndConstraints(rest)
	if inline != "" {
		inlineConstraints = append(inlineConstraints, inline)
	}
	col.DataType = strings.TrimSpace(dataType)
	col.InlineConstraints = inlineConstraints
	_ = upper

	return col, nil
}

// extractIdent extracts the first identifier (possibly double-quoted) from s.
func extractIdent(s string) (name, rest string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "", ""
	}
	if s[0] == '"' {
		end := strings.Index(s[1:], `"`)
		if end < 0 {
			return s[1:], ""
		}
		return s[1 : end+1], strings.TrimSpace(s[end+2:])
	}
	// unquoted: read until whitespace
	for i, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			return s[:i], strings.TrimSpace(s[i:])
		}
	}
	return s, ""
}

// splitTypeAndConstraints splits the column type from trailing inline constraints.
func splitTypeAndConstraints(s string) (dataType, constraints string) {
	upper := strings.ToUpper(s)
	keywords := []string{
		"PRIMARY KEY", "REFERENCES ", "UNIQUE", "CHECK ", "GENERATED ",
		"COMPRESSION ", "STORAGE ",
	}
	earliest := len(s)
	for _, kw := range keywords {
		idx := strings.Index(upper, kw)
		if idx >= 0 && idx < earliest {
			earliest = idx
		}
	}
	if earliest < len(s) {
		return strings.TrimSpace(s[:earliest]), strings.TrimSpace(s[earliest:])
	}
	return s, ""
}

// extractDefaultValue extracts the default value expression, stopping at known clause keywords.
func extractDefaultValue(s string) (val, rest string) {
	upper := strings.ToUpper(s)
	stopWords := []string{"NOT NULL", "NULL", "PRIMARY KEY", "REFERENCES ", "UNIQUE", "CHECK ", "GENERATED "}
	earliest := len(s)
	for _, kw := range stopWords {
		idx := strings.Index(upper, kw)
		if idx >= 0 && idx < earliest {
			earliest = idx
		}
	}
	return strings.TrimSpace(s[:earliest]), strings.TrimSpace(s[earliest:])
}

func replaceCI(s, old, new string) string {
	lower := strings.ToLower(s)
	lold := strings.ToLower(old)
	idx := strings.Index(lower, lold)
	if idx < 0 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

func indexCI(s, sub string) int {
	return strings.Index(strings.ToUpper(s), strings.ToUpper(sub))
}

// --- ALTER TABLE ---

var (
	reAddCol       = regexp.MustCompile(`(?i)^ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(.+)`)
	reDropCol      = regexp.MustCompile(`(?i)^DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\S+)`)
	reAlterColType = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+(?:SET\s+DATA\s+)?TYPE\s+(.+)`)
	reAlterColSet  = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+(SET\s+DEFAULT\s+(.+)|DROP\s+DEFAULT|SET\s+NOT\s+NULL|DROP\s+NOT\s+NULL)`)
	reRenameCol    = regexp.MustCompile(`(?i)^RENAME\s+COLUMN\s+(\S+)\s+TO\s+(\S+)`)
	reRenameTo     = regexp.MustCompile(`(?i)^RENAME\s+TO\s+(\S+)`)
	reAddConstr    = regexp.MustCompile(`(?i)^ADD\s+CONSTRAINT\s+(\S+)\s+(.+)`)
	reAddConstrAnon = regexp.MustCompile(`(?i)^ADD\s+(PRIMARY\s+KEY|UNIQUE|FOREIGN\s+KEY|CHECK)\s*(.*)`)
	reDropConstr   = regexp.MustCompile(`(?i)^DROP\s+CONSTRAINT\s+(?:IF\s+EXISTS\s+)?(\S+)`)
)

func parseAlterTable(sql string) (Statement, error) {
	m := reAlterTable.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse ALTER TABLE: %s", sql[:min(len(sql), 60)])
	}
	tableName := strings.TrimSuffix(m[1], ";")
	actionsStr := strings.TrimSpace(m[2])

	// split comma-separated actions at depth 0
	actionParts := splitAtDepthZeroCommas(actionsStr)
	var actions []AlterAction
	for _, part := range actionParts {
		part = strings.TrimSpace(part)
		a, err := parseAlterAction(part)
		if err != nil {
			return nil, fmt.Errorf("ALTER TABLE %s: %w", tableName, err)
		}
		actions = append(actions, a)
	}

	return AlterTableStmt{TableName: tableName, Actions: actions}, nil
}

func parseAlterAction(s string) (AlterAction, error) {
	upper := strings.ToUpper(s)

	if m := reAddCol.FindStringSubmatch(s); m != nil {
		col, err := parseColumnDef(m[1])
		if err != nil {
			return AlterAction{}, err
		}
		return AlterAction{Kind: ActionAddColumn, Column: col.Name, ColDef: &col}, nil
	}
	if m := reDropCol.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionDropColumn, Column: m[1]}, nil
	}
	if m := reAlterColType.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionAlterColumnType, Column: m[1], DataType: strings.TrimSpace(m[2])}, nil
	}
	if m := reAlterColSet.FindStringSubmatch(s); m != nil {
		col := m[1]
		action := strings.ToUpper(m[2])
		switch {
		case strings.HasPrefix(action, "SET NOT NULL"):
			return AlterAction{Kind: ActionSetNotNull, Column: col}, nil
		case strings.HasPrefix(action, "DROP NOT NULL"):
			return AlterAction{Kind: ActionDropNotNull, Column: col}, nil
		case strings.HasPrefix(action, "DROP DEFAULT"):
			return AlterAction{Kind: ActionDropDefault, Column: col}, nil
		case strings.HasPrefix(action, "SET DEFAULT"):
			return AlterAction{Kind: ActionSetDefault, Column: col, Default: strings.TrimSpace(m[3])}, nil
		}
	}
	if m := reRenameCol.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionRenameColumn, Column: m[1], NewName: m[2]}, nil
	}
	if m := reRenameTo.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionRenameTo, NewName: m[1]}, nil
	}
	if m := reAddConstr.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionAddConstraint, Constraint: TableConstraint{Name: m[1], Definition: strings.TrimSpace(m[2])}}, nil
	}
	if m := reAddConstrAnon.FindStringSubmatch(s); m != nil {
		def := strings.TrimSpace(m[1]) + " " + strings.TrimSpace(m[2])
		return AlterAction{Kind: ActionAddConstraint, Constraint: TableConstraint{Definition: strings.TrimSpace(def)}}, nil
	}
	if m := reDropConstr.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionDropConstraint, Constraint: TableConstraint{Name: m[1]}}, nil
	}

	_ = upper
	return AlterAction{}, fmt.Errorf("unrecognized ALTER TABLE action: %s", s[:min(len(s), 60)])
}

func buildColumnDefStr(col ColumnDef) string {
	parts := []string{col.Name, col.DataType}
	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if col.Default != nil {
		parts = append(parts, "DEFAULT "+*col.Default)
	}
	parts = append(parts, col.InlineConstraints...)
	return strings.Join(parts, " ")
}

func parseDropTable(sql string) (Statement, error) {
	m := reDropTable.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse DROP TABLE: %s", sql[:min(len(sql), 60)])
	}
	return DropTableStmt{
		TableName: strings.TrimSuffix(m[2], ";"),
		IfExists:  m[1] != "",
	}, nil
}

func parseCreateIndex(sql string) (Statement, error) {
	m := reCreateIndex.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse CREATE INDEX: %s", sql[:min(len(sql), 60)])
	}
	return CreateIndexStmt{
		Unique:    strings.TrimSpace(m[1]) != "",
		IndexName: m[2],
		TableName: m[3],
		Body:      strings.TrimSpace(m[4]),
	}, nil
}

func parseDropIndex(sql string) (Statement, error) {
	m := reDropIndex.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse DROP INDEX: %s", sql[:min(len(sql), 60)])
	}
	return DropIndexStmt{
		IfExists:  m[1] != "",
		IndexName: strings.TrimSuffix(m[2], ";"),
	}, nil
}

func parseCreateSequence(sql string) (Statement, error) {
	m := reCreateSequence.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse CREATE SEQUENCE: %s", sql[:min(len(sql), 60)])
	}
	return CreateSequenceStmt{
		SeqName: m[1],
		Body:    strings.TrimSpace(m[2]),
	}, nil
}

func parseDropSequence(sql string) (Statement, error) {
	m := reDropSequence.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse DROP SEQUENCE: %s", sql[:min(len(sql), 60)])
	}
	return DropSequenceStmt{
		IfExists: m[1] != "",
		SeqName:  strings.TrimSuffix(m[2], ";"),
	}, nil
}

func parseCreateType(sql string) (Statement, error) {
	m := reCreateTypeEnum.FindStringSubmatch(sql)
	if m == nil {
		// Not an enum, treat as unknown
		return UnknownStmt{Raw: sql}, nil
	}
	labels := splitAtDepthZeroCommas(m[2])
	for i, l := range labels {
		labels[i] = strings.Trim(strings.TrimSpace(l), "'")
	}
	return CreateTypeStmt{TypeName: m[1], Labels: labels}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
