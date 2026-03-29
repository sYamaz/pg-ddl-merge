package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	reCreateTable    = regexp.MustCompile(`(?i)^CREATE\s+(?:(?:GLOBAL|LOCAL)\s+)?(?:(TEMPORARY|TEMP|UNLOGGED)\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	rePartitionOf    = regexp.MustCompile(`(?i)^CREATE\s+(?:(?:GLOBAL|LOCAL)\s+)?(?:(?:TEMPORARY|TEMP|UNLOGGED)\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+PARTITION\s+OF\s+`)
	reCollate        = regexp.MustCompile(`(?i)\s+COLLATE\s+("(?:[^"]|"")*"|\S+)`)
	reAlterTable     = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(?:ONLY\s+)?(\S+)\s+(.+)`)
	reDropTable      = regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:(IF\s+EXISTS)\s+)?(.+)`)
	reCreateIndex    = regexp.MustCompile(`(?i)^CREATE\s+(UNIQUE\s+)?INDEX\s+(CONCURRENTLY\s+)?(IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*(.*)`)
	reDropIndex      = regexp.MustCompile(`(?i)^DROP\s+INDEX\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reAlterIndex     = regexp.MustCompile(`(?i)^ALTER\s+INDEX\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)
	reCreateSequence = regexp.MustCompile(`(?is)^CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)(.*)`)
	reDropSequence   = regexp.MustCompile(`(?i)^DROP\s+SEQUENCE\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reAlterSeqRename = regexp.MustCompile(`(?i)^ALTER\s+SEQUENCE\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)
	reAlterSeqBase   = regexp.MustCompile(`(?i)^ALTER\s+SEQUENCE\s+(?:IF\s+EXISTS\s+)?(\S+)\s+(.+)`)
	reCreateTypeEnum      = regexp.MustCompile(`(?i)^CREATE\s+TYPE\s+(\S+)\s+AS\s+ENUM\s*\((.+)\)`)
	reCreateTypeComposite = regexp.MustCompile(`(?i)^CREATE\s+TYPE\s+(\S+)\s+AS\s*\(`)
	reCreateTypeRange     = regexp.MustCompile(`(?i)^CREATE\s+TYPE\s+(\S+)\s+AS\s+RANGE\s*\(`)
	reDropType            = regexp.MustCompile(`(?i)^DROP\s+TYPE\s+(?:(IF\s+EXISTS)\s+)?(\S+)`)
	reAlterType           = regexp.MustCompile(`(?i)^ALTER\s+TYPE\s+(\S+)\s+(.+)`)
	reAlterColUsing       = regexp.MustCompile(`(?i)\s+USING\s+.+$`)
	reAlterFuncOrProcName  = regexp.MustCompile(`(?i)^ALTER\s+(?:FUNCTION|PROCEDURE)\s+(?:IF\s+EXISTS\s+)?(\S+?)(?:\s*\([^)]*\))?\s+`)
	reAlterExtensionUpdate = regexp.MustCompile(`(?i)^ALTER\s+EXTENSION\s+(\S+)\s+UPDATE\b`)
	reAlterDomainBase      = regexp.MustCompile(`(?i)^ALTER\s+DOMAIN\s+(?:IF\s+EXISTS\s+)?(\S+)\s+`)
	reAlterPolicyBase      = regexp.MustCompile(`(?i)^ALTER\s+POLICY\s+(\S+)\s+ON\s+(\S+)\s+`)
	reAlterViewBase        = regexp.MustCompile(`(?i)^ALTER\s+VIEW\s+(?:IF\s+EXISTS\s+)?(\S+)\s+`)
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
	case strings.HasPrefix(upper, "ALTER INDEX"):
		return parseAlterIndex(sql)
	case strings.HasPrefix(upper, "CREATE SEQUENCE"):
		return parseCreateSequence(sql)
	case strings.HasPrefix(upper, "DROP SEQUENCE"):
		return parseDropSequence(sql)
	case strings.HasPrefix(upper, "CREATE TYPE"):
		return parseCreateType(sql)
	case strings.HasPrefix(upper, "DROP TYPE"):
		return parseDropType(sql)
	case strings.HasPrefix(upper, "ALTER TYPE"):
		return parseAlterType(sql)
	case strings.HasPrefix(upper, "ALTER SEQUENCE"):
		return parseAlterSequence(sql)
	case strings.HasPrefix(upper, "ALTER VIEW"),
		strings.HasPrefix(upper, "ALTER MATERIALIZED VIEW"),
		strings.HasPrefix(upper, "ALTER SCHEMA"),
		strings.HasPrefix(upper, "ALTER FUNCTION"),
		strings.HasPrefix(upper, "ALTER PROCEDURE"),
		strings.HasPrefix(upper, "ALTER TRIGGER"),
		strings.HasPrefix(upper, "ALTER DOMAIN"),
		strings.HasPrefix(upper, "ALTER EXTENSION"),
		strings.HasPrefix(upper, "ALTER POLICY"),
		strings.HasPrefix(upper, "ALTER RULE"):
		return parseAlterObject(sql)
	// Generic tracked objects — check longer prefixes first
	case strings.HasPrefix(upper, "CREATE OR REPLACE MATERIALIZED VIEW"),
		strings.HasPrefix(upper, "CREATE MATERIALIZED VIEW"):
		return parseCreateObject(sql, ObjMatView)
	case strings.HasPrefix(upper, "DROP MATERIALIZED VIEW"):
		return parseDropObject(sql, ObjMatView)
	case strings.HasPrefix(upper, "CREATE OR REPLACE VIEW"),
		strings.HasPrefix(upper, "CREATE VIEW"):
		return parseCreateObject(sql, ObjView)
	case strings.HasPrefix(upper, "DROP VIEW"):
		return parseDropObject(sql, ObjView)
	case strings.HasPrefix(upper, "CREATE SCHEMA"):
		return parseCreateObject(sql, ObjSchema)
	case strings.HasPrefix(upper, "DROP SCHEMA"):
		return parseDropObject(sql, ObjSchema)
	case strings.HasPrefix(upper, "CREATE EXTENSION"):
		return parseCreateObject(sql, ObjExtension)
	case strings.HasPrefix(upper, "DROP EXTENSION"):
		return parseDropObject(sql, ObjExtension)
	case strings.HasPrefix(upper, "CREATE OR REPLACE FUNCTION"),
		strings.HasPrefix(upper, "CREATE FUNCTION"):
		return parseCreateObject(sql, ObjFunction)
	case strings.HasPrefix(upper, "DROP FUNCTION"):
		return parseDropObject(sql, ObjFunction)
	case strings.HasPrefix(upper, "CREATE OR REPLACE PROCEDURE"),
		strings.HasPrefix(upper, "CREATE PROCEDURE"):
		return parseCreateObject(sql, ObjProcedure)
	case strings.HasPrefix(upper, "DROP PROCEDURE"):
		return parseDropObject(sql, ObjProcedure)
	case strings.HasPrefix(upper, "CREATE CONSTRAINT TRIGGER"),
		strings.HasPrefix(upper, "CREATE OR REPLACE TRIGGER"),
		strings.HasPrefix(upper, "CREATE TRIGGER"):
		return parseCreateObject(sql, ObjTrigger)
	case strings.HasPrefix(upper, "DROP TRIGGER"):
		return parseDropObject(sql, ObjTrigger)
	case strings.HasPrefix(upper, "CREATE DOMAIN"):
		return parseCreateObject(sql, ObjDomain)
	case strings.HasPrefix(upper, "DROP DOMAIN"):
		return parseDropObject(sql, ObjDomain)
	case strings.HasPrefix(upper, "CREATE POLICY"):
		return parseCreateObject(sql, ObjPolicy)
	case strings.HasPrefix(upper, "DROP POLICY"):
		return parseDropObject(sql, ObjPolicy)
	case strings.HasPrefix(upper, "CREATE OR REPLACE RULE"),
		strings.HasPrefix(upper, "CREATE RULE"):
		return parseCreateObject(sql, ObjRule)
	case strings.HasPrefix(upper, "DROP RULE"):
		return parseDropObject(sql, ObjRule)
	case strings.HasPrefix(upper, "TRUNCATE"):
		return parseTruncate(sql)
	default:
		return UnknownStmt{Raw: sql}, nil
	}
}

func parseCreateTable(sql string) (Statement, error) {
	m := reCreateTable.FindStringSubmatchIndex(sql)
	if m == nil {
		// PARTITION OF form → verbatim object, tracked by name.
		if m := rePartitionOf.FindStringSubmatch(sql); m != nil {
			return CreateObjectStmt{
				Kind: ObjPartition,
				Name: normalizeIdent(m[1]),
				SQL:  strings.TrimSuffix(strings.TrimSpace(sql), ";"),
			}, nil
		}
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

	// Extract optional PARTITION BY clause that follows the column list.
	var partitionBy string
	if closeIdx := parenClose(sql, start); closeIdx >= 0 {
		rest := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql[closeIdx+1:]), ";"))
		if idx := strings.Index(strings.ToUpper(rest), "PARTITION BY"); idx >= 0 {
			partitionBy = strings.TrimSpace(rest[idx+len("PARTITION BY"):])
		}
	}

	return CreateTableStmt{
		TableName:   tableName,
		Temporary:   temporary,
		Unlogged:    unlogged,
		Columns:     cols,
		Constraints: constraints,
		PartitionBy: partitionBy,
	}, nil
}

// parenClose returns the index of the ')' that closes the '(' at start,
// accounting for nested parens and single-quoted strings.
func parenClose(sql string, start int) int {
	if start >= len(sql) || sql[start] != '(' {
		return -1
	}
	depth := 0
	for i := start; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '\'':
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
	return -1
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
	reAddCol             = regexp.MustCompile(`(?i)^ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(.+)`)
	reDropCol            = regexp.MustCompile(`(?i)^DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\S+)`)
	reAlterColType       = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+(?:SET\s+DATA\s+)?TYPE\s+(.+)`)
	reAlterColSet        = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+(SET\s+DEFAULT\s+(.+)|DROP\s+DEFAULT|SET\s+NOT\s+NULL|DROP\s+NOT\s+NULL)`)
	reAlterColAddGen     = regexp.MustCompile(`(?is)^ALTER\s+COLUMN\s+(\S+)\s+ADD\s+(GENERATED\s+.+)`)
	reAlterColSetGen     = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+SET\s+GENERATED\s+(ALWAYS|BY\s+DEFAULT)`)
	reAlterColDropIdent  = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+DROP\s+IDENTITY(?:\s+(IF\s+EXISTS))?`)
	reAlterColSetStorage = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+SET\s+STORAGE\s+(\S+)`)
	reAlterColSetComp    = regexp.MustCompile(`(?i)^ALTER\s+COLUMN\s+(\S+)\s+SET\s+COMPRESSION\s+(\S+)`)
	reRenameCol          = regexp.MustCompile(`(?i)^RENAME\s+COLUMN\s+(\S+)\s+TO\s+(\S+)`)
	reRenameTo           = regexp.MustCompile(`(?i)^RENAME\s+TO\s+(\S+)`)
	reRenameConstr       = regexp.MustCompile(`(?i)^RENAME\s+CONSTRAINT\s+(\S+)\s+TO\s+(\S+)`)
	reAddConstr          = regexp.MustCompile(`(?i)^ADD\s+CONSTRAINT\s+(\S+)\s+(.+)`)
	reAddConstrAnon      = regexp.MustCompile(`(?i)^ADD\s+(PRIMARY\s+KEY|UNIQUE|FOREIGN\s+KEY|CHECK)\s*(.*)`)
	reDropConstr         = regexp.MustCompile(`(?i)^DROP\s+CONSTRAINT\s+(IF\s+EXISTS\s+)?(\S+)`)
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
		a, err := parseAlterAction(tableName, part)
		if err != nil {
			return nil, fmt.Errorf("ALTER TABLE %s: %w", tableName, err)
		}
		actions = append(actions, a)
	}

	return AlterTableStmt{TableName: tableName, Actions: actions}, nil
}

func parseAlterAction(tableName, s string) (AlterAction, error) {
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
		// Strip trailing USING clause — it's execution-time only, not part of the type name.
		dataType := strings.TrimSpace(reAlterColUsing.ReplaceAllString(m[2], ""))
		return AlterAction{Kind: ActionAlterColumnType, Column: m[1], DataType: dataType}, nil
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
	if m := reAlterColAddGen.FindStringSubmatch(s); m != nil {
		clause := strings.TrimSuffix(strings.TrimSpace(m[2]), ";")
		return AlterAction{Kind: ActionAddGenerated, Column: m[1], GeneratedClause: clause}, nil
	}
	if m := reAlterColSetGen.FindStringSubmatch(s); m != nil {
		kind := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(m[2])), ";")
		return AlterAction{Kind: ActionSetGenerated, Column: m[1], GeneratedKind: kind}, nil
	}
	if m := reAlterColDropIdent.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionDropIdentity, Column: m[1], IfExists: m[2] != ""}, nil
	}
	if m := reAlterColSetStorage.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionSetStorage, Column: m[1], StorageType: strings.TrimSuffix(m[2], ";")}, nil
	}
	if m := reAlterColSetComp.FindStringSubmatch(s); m != nil {
		return AlterAction{Kind: ActionSetCompression, Column: m[1], CompressionMethod: strings.TrimSuffix(m[2], ";")}, nil
	}
	if m := reRenameConstr.FindStringSubmatch(s); m != nil {
		oldName := strings.TrimSuffix(m[1], ";")
		newName := strings.TrimSuffix(m[2], ";")
		return AlterAction{Kind: ActionRenameConstraint, Constraint: TableConstraint{Name: oldName}, NewName: newName}, nil
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
		return AlterAction{Kind: ActionDropConstraint, Constraint: TableConstraint{Name: m[2]}, IfExists: m[1] != ""}, nil
	}

	// Unrecognized action — warn and skip rather than error
	if isSkippableAlterAction(upper) {
		fmt.Fprintf(os.Stderr, "warning: ALTER TABLE %s: unrecognized action skipped: %s\n", tableName, s[:min(len(s), 80)])
		return AlterAction{Kind: ActionSkip}, nil
	}

	fmt.Fprintf(os.Stderr, "warning: ALTER TABLE %s: unrecognized action skipped: %s\n", tableName, s[:min(len(s), 80)])
	return AlterAction{Kind: ActionSkip}, nil
}

// isSkippableAlterAction returns true for ALTER TABLE actions we know are safe to skip.
func isSkippableAlterAction(upper string) bool {
	skippablePrefixes := []string{
		"SET STATISTICS",
		"SET (", "RESET (",
		"SET SCHEMA",
		"SET TABLESPACE",
		"SET WITHOUT CLUSTER",
		"SET ACCESS METHOD",
		"CLUSTER ON",
		"ENABLE TRIGGER", "DISABLE TRIGGER",
		"ENABLE RULE", "DISABLE RULE",
		"ENABLE ROW LEVEL SECURITY", "DISABLE ROW LEVEL SECURITY",
		"FORCE ROW LEVEL SECURITY", "NO FORCE ROW LEVEL SECURITY",
		"ATTACH PARTITION",
		"DETACH PARTITION",
		"VALIDATE CONSTRAINT",
		"INHERIT ", "NO INHERIT ",
		"OF ", "NOT OF",
		"OWNER TO",
		"ENABLE ALWAYS TRIGGER",
		"ENABLE REPLICA TRIGGER",
		"ENABLE ALWAYS RULE",
		"ENABLE REPLICA RULE",
		"ALTER COLUMN",
	}
	for _, pfx := range skippablePrefixes {
		if strings.HasPrefix(upper, pfx) {
			return true
		}
	}
	return false
}

func parseDropTable(sql string) (Statement, error) {
	m := reDropTable.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse DROP TABLE: %s", sql[:min(len(sql), 60)])
	}
	ifExists := m[1] != ""
	// m[2] contains everything after IF EXISTS (or after DROP TABLE)
	// strip trailing CASCADE/RESTRICT
	rest := strings.TrimSpace(m[2])
	rest = strings.TrimSuffix(rest, ";")
	upper := strings.ToUpper(rest)
	for _, suffix := range []string{" CASCADE", " RESTRICT"} {
		if strings.HasSuffix(upper, suffix) {
			rest = rest[:len(rest)-len(suffix)]
			upper = strings.ToUpper(rest)
		}
	}
	// split multiple table names
	parts := strings.Split(rest, ",")
	var names []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("cannot parse DROP TABLE: no table names found in: %s", sql[:min(len(sql), 60)])
	}
	return DropTableStmt{
		TableNames: names,
		IfExists:   ifExists,
	}, nil
}

func parseCreateIndex(sql string) (Statement, error) {
	m := reCreateIndex.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse CREATE INDEX: %s", sql[:min(len(sql), 60)])
	}
	// m[1]=UNIQUE, m[2]=CONCURRENTLY, m[3]=IF NOT EXISTS, m[4]=name, m[5]=table, m[6]=body
	return CreateIndexStmt{
		Unique:       strings.TrimSpace(m[1]) != "",
		Concurrently: strings.TrimSpace(m[2]) != "",
		IfNotExists:  strings.TrimSpace(m[3]) != "",
		IndexName:    m[4],
		TableName:    m[5],
		Body:         strings.TrimSuffix(strings.TrimSpace(m[6]), ";"),
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

func parseAlterIndex(sql string) (Statement, error) {
	m := reAlterIndex.FindStringSubmatch(sql)
	if m == nil {
		// Other ALTER INDEX forms → pass through
		return UnknownStmt{Raw: sql}, nil
	}
	return AlterIndexStmt{
		IndexName: m[1],
		NewName:   strings.TrimSuffix(m[2], ";"),
	}, nil
}

func parseCreateSequence(sql string) (Statement, error) {
	m := reCreateSequence.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse CREATE SEQUENCE: %s", sql[:min(len(sql), 60)])
	}
	// Normalize multi-line bodies to a single space-separated line.
	body := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(m[2]), " ")
	return CreateSequenceStmt{
		SeqName: m[1],
		Body:    body,
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
	// ENUM
	if m := reCreateTypeEnum.FindStringSubmatch(sql); m != nil {
		labels := splitAtDepthZeroCommas(m[2])
		for i, l := range labels {
			labels[i] = strings.Trim(strings.TrimSpace(l), "'")
		}
		return CreateTypeStmt{TypeName: m[1], Labels: labels}, nil
	}
	// RANGE: check before COMPOSITE to avoid "AS RANGE (" being matched by COMPOSITE
	if m := reCreateTypeRange.FindStringSubmatch(sql); m != nil {
		return CreateObjectStmt{
			Kind: ObjType,
			Name: normalizeIdent(m[1]),
			SQL:  strings.TrimSuffix(strings.TrimSpace(sql), ";"),
		}, nil
	}
	// COMPOSITE: "CREATE TYPE name AS (..."
	if m := reCreateTypeComposite.FindStringSubmatch(sql); m != nil {
		return CreateObjectStmt{
			Kind: ObjType,
			Name: normalizeIdent(m[1]),
			SQL:  strings.TrimSuffix(strings.TrimSpace(sql), ";"),
		}, nil
	}
	// Other (base type, etc.) → pass through
	return UnknownStmt{Raw: sql}, nil
}

func parseDropType(sql string) (Statement, error) {
	m := reDropType.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("cannot parse DROP TYPE: %s", sql[:min(len(sql), 60)])
	}
	return DropTypeStmt{
		IfExists: m[1] != "",
		TypeName: strings.TrimSuffix(m[2], ";"),
	}, nil
}

var (
	reAlterTypeAddValue    = regexp.MustCompile(`(?i)^ADD\s+VALUE\s+(?:(IF\s+NOT\s+EXISTS)\s+)?'([^']*)'(?:\s+(BEFORE|AFTER)\s+'([^']*)')?`)
	reAlterTypeRenameValue = regexp.MustCompile(`(?i)^RENAME\s+VALUE\s+'([^']*)'\s+TO\s+'([^']*)'`)
	reAlterTypeRenameTo    = regexp.MustCompile(`(?i)^RENAME\s+TO\s+(\S+)`)
)

func parseAlterType(sql string) (Statement, error) {
	m := reAlterType.FindStringSubmatch(sql)
	if m == nil {
		return UnknownStmt{Raw: sql}, nil
	}
	typeName := m[1]
	actionStr := strings.TrimSpace(m[2])

	if am := reAlterTypeAddValue.FindStringSubmatch(actionStr); am != nil {
		act := AlterTypeAction{
			Kind:        AlterTypeAddValue,
			Value:       am[2],
			IfNotExists: am[1] != "",
		}
		if am[3] != "" {
			pos := strings.ToUpper(am[3])
			if pos == "BEFORE" {
				act.Before = am[4]
			} else {
				act.After = am[4]
			}
		}
		return AlterTypeStmt{TypeName: typeName, Action: act}, nil
	}

	if am := reAlterTypeRenameValue.FindStringSubmatch(actionStr); am != nil {
		return AlterTypeStmt{TypeName: typeName, Action: AlterTypeAction{
			Kind:     AlterTypeRenameValue,
			Value:    am[1],
			NewValue: am[2],
		}}, nil
	}

	if am := reAlterTypeRenameTo.FindStringSubmatch(actionStr); am != nil {
		return AlterTypeStmt{TypeName: typeName, Action: AlterTypeAction{
			Kind:    AlterTypeRenameTo,
			NewName: strings.TrimSuffix(am[1], ";"),
		}}, nil
	}

	// Other ALTER TYPE actions → pass through
	return UnknownStmt{Raw: sql}, nil
}

// --- Generic object parsing ---

// extractObjectName extracts the name from a CREATE/DROP statement for the given kind.
// Returns the normalized key (e.g. "name_on_tablename" for triggers/policies/rules).
func extractCreateObjectName(sql string, kind ObjectKind) string {
	upper := strings.ToUpper(sql)

	switch kind {
	case ObjView:
		// CREATE [OR REPLACE] VIEW [IF NOT EXISTS] name
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjMatView:
		// CREATE [OR REPLACE] MATERIALIZED VIEW [IF NOT EXISTS] name
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:OR\s+REPLACE\s+)?MATERIALIZED\s+VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjSchema:
		re := regexp.MustCompile(`(?i)^CREATE\s+SCHEMA\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjExtension:
		re := regexp.MustCompile(`(?i)^CREATE\s+EXTENSION\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjFunction:
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(\S+?)(?:\s*\(|$)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjProcedure:
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:OR\s+REPLACE\s+)?PROCEDURE\s+(\S+?)(?:\s*\(|$)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjTrigger:
		// CREATE [CONSTRAINT] [OR REPLACE] TRIGGER name ... ON tablename
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:CONSTRAINT\s+)?(?:OR\s+REPLACE\s+)?TRIGGER\s+(\S+)`)
		reOn := regexp.MustCompile(`(?i)\s+ON\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			name := normalizeIdent(m[1])
			if mOn := reOn.FindStringSubmatch(sql); mOn != nil {
				table := normalizeIdent(mOn[1])
				return name + "_on_" + table
			}
			return name
		}
	case ObjDomain:
		re := regexp.MustCompile(`(?i)^CREATE\s+DOMAIN\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjPolicy:
		// CREATE POLICY name ON tablename
		re := regexp.MustCompile(`(?i)^CREATE\s+POLICY\s+(\S+)\s+ON\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1]) + "_on_" + normalizeIdent(m[2])
		}
	case ObjRule:
		// CREATE [OR REPLACE] RULE name AS ON event TO tablename
		re := regexp.MustCompile(`(?i)^CREATE\s+(?:OR\s+REPLACE\s+)?RULE\s+(\S+)\s+AS\s+ON\s+\S+\s+TO\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1]) + "_on_" + normalizeIdent(m[2])
		}
	}
	_ = upper
	return ""
}

func extractDropObjectName(sql string, kind ObjectKind) string {
	switch kind {
	case ObjView:
		re := regexp.MustCompile(`(?i)^DROP\s+VIEW\s+(?:IF\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(strings.TrimSuffix(m[1], ";"))
		}
	case ObjMatView:
		re := regexp.MustCompile(`(?i)^DROP\s+MATERIALIZED\s+VIEW\s+(?:IF\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(strings.TrimSuffix(m[1], ";"))
		}
	case ObjSchema:
		re := regexp.MustCompile(`(?i)^DROP\s+SCHEMA\s+(?:IF\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(strings.TrimSuffix(m[1], ";"))
		}
	case ObjExtension:
		re := regexp.MustCompile(`(?i)^DROP\s+EXTENSION\s+(?:IF\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(strings.TrimSuffix(m[1], ";"))
		}
	case ObjFunction:
		re := regexp.MustCompile(`(?i)^DROP\s+FUNCTION\s+(?:IF\s+EXISTS\s+)?(\S+?)(?:\s*\(|\s*;|$)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjProcedure:
		re := regexp.MustCompile(`(?i)^DROP\s+PROCEDURE\s+(?:IF\s+EXISTS\s+)?(\S+?)(?:\s*\(|\s*;|$)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1])
		}
	case ObjTrigger:
		// DROP TRIGGER [IF EXISTS] name ON tablename
		re := regexp.MustCompile(`(?i)^DROP\s+TRIGGER\s+(?:IF\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1]) + "_on_" + normalizeIdent(strings.TrimSuffix(m[2], ";"))
		}
	case ObjDomain:
		re := regexp.MustCompile(`(?i)^DROP\s+DOMAIN\s+(?:IF\s+EXISTS\s+)?(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(strings.TrimSuffix(m[1], ";"))
		}
	case ObjPolicy:
		re := regexp.MustCompile(`(?i)^DROP\s+POLICY\s+(?:IF\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1]) + "_on_" + normalizeIdent(strings.TrimSuffix(m[2], ";"))
		}
	case ObjRule:
		re := regexp.MustCompile(`(?i)^DROP\s+RULE\s+(?:IF\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)`)
		if m := re.FindStringSubmatch(sql); m != nil {
			return normalizeIdent(m[1]) + "_on_" + normalizeIdent(strings.TrimSuffix(m[2], ";"))
		}
	}
	return ""
}

func isOrReplace(sql string) bool {
	upper := strings.ToUpper(sql)
	return strings.Contains(upper, "OR REPLACE")
}

func parseCreateObject(sql string, kind ObjectKind) (Statement, error) {
	name := extractCreateObjectName(sql, kind)
	if name == "" {
		// Fall back to unknown if we cannot extract the name
		return UnknownStmt{Raw: sql}, nil
	}
	return CreateObjectStmt{
		Kind:      kind,
		Name:      name,
		OrReplace: isOrReplace(sql),
		SQL:       sql,
	}, nil
}

func parseDropObject(sql string, kind ObjectKind) (Statement, error) {
	name := extractDropObjectName(sql, kind)
	if name == "" {
		return UnknownStmt{Raw: sql}, nil
	}
	upper := strings.ToUpper(sql)
	ifExists := strings.Contains(upper, "IF EXISTS")
	return DropObjectStmt{
		Kind:     kind,
		Name:     name,
		IfExists: ifExists,
	}, nil
}

func parseAlterSequence(sql string) (Statement, error) {
	if m := reAlterSeqRename.FindStringSubmatch(sql); m != nil {
		return AlterSequenceStmt{
			SeqName: m[1],
			NewName: strings.TrimSuffix(m[2], ";"),
		}, nil
	}
	// Option-setting form: INCREMENT / MINVALUE / MAXVALUE / START / CACHE / CYCLE / OWNED BY etc.
	if m := reAlterSeqBase.FindStringSubmatch(sql); m != nil {
		if opts := parseSequenceOptions(m[2]); len(opts) > 0 {
			return AlterSequenceOptsStmt{SeqName: m[1], Opts: opts}, nil
		}
	}
	return UnknownStmt{Raw: sql}, nil
}

// seqOptSpecs defines parseable options in ALTER SEQUENCE ... (non-RENAME).
// NO variants must come before the bare keyword so family-based dedup works correctly.
var seqOptSpecs = []struct {
	re     *regexp.Regexp
	kind   string
	family string
}{
	{regexp.MustCompile(`(?i)\bNO\s+MINVALUE\b`), "NO MINVALUE", "MINVALUE"},
	{regexp.MustCompile(`(?i)\bNO\s+MAXVALUE\b`), "NO MAXVALUE", "MAXVALUE"},
	{regexp.MustCompile(`(?i)\bNO\s+CYCLE\b`), "NO CYCLE", "CYCLE"},
	{regexp.MustCompile(`(?i)\bINCREMENT\s+(?:BY\s+)?(-?\d+)`), "INCREMENT BY", "INCREMENT"},
	{regexp.MustCompile(`(?i)\bMINVALUE\s+(-?\d+)`), "MINVALUE", "MINVALUE"},
	{regexp.MustCompile(`(?i)\bMAXVALUE\s+(-?\d+)`), "MAXVALUE", "MAXVALUE"},
	{regexp.MustCompile(`(?i)\bSTART\s+(?:WITH\s+)?(-?\d+)`), "START WITH", "START"},
	{regexp.MustCompile(`(?i)\bCACHE\s+(-?\d+)`), "CACHE", "CACHE"},
	{regexp.MustCompile(`(?i)\bCYCLE\b`), "CYCLE", "CYCLE"},
	{regexp.MustCompile(`(?i)\bOWNED\s+BY\s+(\S+)`), "OWNED BY", "OWNED"},
	{regexp.MustCompile(`(?i)\bAS\s+(\S+)`), "AS", "AS"},
	{regexp.MustCompile(`(?i)\bSET\s+(LOGGED|UNLOGGED)\b`), "SET", "LOGGED"},
	// RESTART is runtime state: parse so we don't fall through to UnknownStmt,
	// but the schema layer will skip body updates for it.
	{regexp.MustCompile(`(?i)\bRESTART\s+(?:WITH\s+)?(-?\d+)`), "RESTART", "RESTART"},
	{regexp.MustCompile(`(?i)\bRESTART\b`), "RESTART", "RESTART"},
}

func parseSequenceOptions(s string) []SequenceOption {
	s = strings.TrimSpace(strings.TrimSuffix(s, ";"))
	var opts []SequenceOption
	seen := map[string]bool{}
	for _, spec := range seqOptSpecs {
		if seen[spec.family] {
			continue
		}
		m := spec.re.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		seen[spec.family] = true
		val := ""
		if len(m) > 1 {
			val = m[1]
		}
		opts = append(opts, SequenceOption{Kind: spec.kind, Value: val})
	}
	return opts
}

func parseTruncate(sql string) (Statement, error) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))

	restartIdentity := false
	cascade := false

	// Strip trailing modifier keywords (order matters: innermost last)
	for {
		upper := strings.ToUpper(s)
		switch {
		case strings.HasSuffix(upper, " CASCADE"):
			cascade = true
			s = s[:len(s)-len(" CASCADE")]
		case strings.HasSuffix(upper, " RESTRICT"):
			s = s[:len(s)-len(" RESTRICT")]
		case strings.HasSuffix(upper, " RESTART IDENTITY"):
			restartIdentity = true
			s = s[:len(s)-len(" RESTART IDENTITY")]
		case strings.HasSuffix(upper, " CONTINUE IDENTITY"):
			s = s[:len(s)-len(" CONTINUE IDENTITY")]
		default:
			goto done
		}
	}
done:

	// Strip "TRUNCATE [TABLE]" prefix
	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "TRUNCATE") {
		return UnknownStmt{Raw: sql}, nil
	}
	s = strings.TrimSpace(s[len("TRUNCATE"):])
	if strings.HasPrefix(strings.ToUpper(s), "TABLE ") {
		s = strings.TrimSpace(s[len("TABLE"):])
	}

	// Remove ONLY keyword (used for partitioned tables)
	reOnly := regexp.MustCompile(`(?i)\bONLY\s+`)
	s = reOnly.ReplaceAllString(s, "")

	// Parse comma-separated table names
	var tables []string
	for _, part := range strings.Split(s, ",") {
		t := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(part), "*"))
		t = strings.TrimSpace(t)
		if t != "" {
			tables = append(tables, t)
		}
	}
	if len(tables) == 0 {
		return UnknownStmt{Raw: sql}, nil
	}
	return TruncateStmt{Tables: tables, RestartIdentity: restartIdentity, Cascade: cascade}, nil
}

// alterObjRenameEntry describes how to detect a RENAME TO for a generic object kind.
type alterObjRenameEntry struct {
	prefix string
	kind   ObjectKind
	re     *regexp.Regexp
}

// alterObjRenameEntries lists RENAME TO patterns for each tracked generic object kind.
// Longer prefixes (MATERIALIZED VIEW) must appear before shorter ones (VIEW).
var alterObjRenameEntries = []alterObjRenameEntry{
	{"ALTER MATERIALIZED VIEW", ObjMatView, regexp.MustCompile(`(?i)^ALTER\s+MATERIALIZED\s+VIEW\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER VIEW", ObjView, regexp.MustCompile(`(?i)^ALTER\s+VIEW\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER SCHEMA", ObjSchema, regexp.MustCompile(`(?i)^ALTER\s+SCHEMA\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER EXTENSION", ObjExtension, regexp.MustCompile(`(?i)^ALTER\s+EXTENSION\s+(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER FUNCTION", ObjFunction, regexp.MustCompile(`(?i)^ALTER\s+FUNCTION\s+(?:IF\s+EXISTS\s+)?(\S+?)(?:\s*\([^)]*\))?\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER PROCEDURE", ObjProcedure, regexp.MustCompile(`(?i)^ALTER\s+PROCEDURE\s+(?:IF\s+EXISTS\s+)?(\S+?)(?:\s*\([^)]*\))?\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER DOMAIN", ObjDomain, regexp.MustCompile(`(?i)^ALTER\s+DOMAIN\s+(?:IF\s+EXISTS\s+)?(\S+)\s+RENAME\s+TO\s+(\S+)`)},
}

// alterObjTableScopedEntries handles objects whose key includes "_on_<table>".
type alterObjTableScopedEntry struct {
	prefix string
	kind   ObjectKind
	re     *regexp.Regexp // groups: 1=name, 2=table, 3=newname
}

var alterObjTableScopedEntries = []alterObjTableScopedEntry{
	{"ALTER TRIGGER", ObjTrigger, regexp.MustCompile(`(?i)^ALTER\s+TRIGGER\s+(?:IF\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER POLICY", ObjPolicy, regexp.MustCompile(`(?i)^ALTER\s+POLICY\s+(\S+)\s+ON\s+(\S+)\s+RENAME\s+TO\s+(\S+)`)},
	{"ALTER RULE", ObjRule, regexp.MustCompile(`(?i)^ALTER\s+RULE\s+(\S+)\s+ON\s+(\S+)\s+RENAME\s+TO\s+(\S+)`)},
}

func parseAlterObject(sql string) (Statement, error) {
	upper := strings.ToUpper(sql)

	// Table-scoped objects (TRIGGER, POLICY, RULE) whose key is "name_on_table"
	for _, e := range alterObjTableScopedEntries {
		if strings.HasPrefix(upper, e.prefix) {
			if m := e.re.FindStringSubmatch(sql); m != nil {
				table := normalizeIdent(strings.TrimSuffix(m[2], ";"))
				oldName := normalizeIdent(m[1]) + "_on_" + table
				newName := normalizeIdent(strings.TrimSuffix(m[3], ";")) + "_on_" + table
				return AlterObjectStmt{Kind: e.kind, OldName: oldName, NewName: newName}, nil
			}
			// For POLICY: associate non-RENAME content changes with the named policy.
			if e.kind == ObjPolicy {
				if m := reAlterPolicyBase.FindStringSubmatch(sql); m != nil {
					table := normalizeIdent(strings.TrimSuffix(m[2], ";"))
					name := normalizeIdent(m[1]) + "_on_" + table
					return AlterObjectOptsStmt{Kind: ObjPolicy, Name: name, SQL: sql}, nil
				}
			}
			return UnknownStmt{Raw: sql}, nil
		}
	}

	// Simple objects whose key is just the name
	for _, e := range alterObjRenameEntries {
		if strings.HasPrefix(upper, e.prefix) {
			if m := e.re.FindStringSubmatch(sql); m != nil {
				return AlterObjectStmt{
					Kind:    e.kind,
					OldName: normalizeIdent(m[1]),
					NewName: normalizeIdent(strings.TrimSuffix(m[2], ";")),
				}, nil
			}
			// For FUNCTION/PROCEDURE, associate non-RENAME actions with the named object.
			if e.kind == ObjFunction || e.kind == ObjProcedure {
				if m := reAlterFuncOrProcName.FindStringSubmatch(sql); m != nil {
					return AlterFunctionOptsStmt{
						Kind: e.kind,
						Name: normalizeIdent(m[1]),
						SQL:  sql,
					}, nil
				}
			}
			// For EXTENSION: UPDATE [TO version] action.
			if e.kind == ObjExtension {
				if m := reAlterExtensionUpdate.FindStringSubmatch(sql); m != nil {
					return AlterObjectOptsStmt{Kind: ObjExtension, Name: normalizeIdent(m[1]), SQL: sql}, nil
				}
			}
			// For DOMAIN: non-RENAME content changes (ADD/DROP CONSTRAINT, SET/DROP DEFAULT, etc.).
			if e.kind == ObjDomain {
				if m := reAlterDomainBase.FindStringSubmatch(sql); m != nil {
					return AlterObjectOptsStmt{Kind: ObjDomain, Name: normalizeIdent(m[1]), SQL: sql}, nil
				}
			}
			// For VIEW: non-RENAME content changes (ALTER COLUMN SET/DROP DEFAULT, SET/RESET options, etc.).
			if e.kind == ObjView {
				if m := reAlterViewBase.FindStringSubmatch(sql); m != nil {
					return AlterObjectOptsStmt{Kind: ObjView, Name: normalizeIdent(m[1]), SQL: sql}, nil
				}
			}
			return UnknownStmt{Raw: sql}, nil
		}
	}

	return UnknownStmt{Raw: sql}, nil
}

