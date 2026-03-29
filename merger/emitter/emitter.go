package emitter

import (
	"fmt"
	"strings"

	"github.com/sYamaz/pg-ddl-merge/merger/parser"
	"github.com/sYamaz/pg-ddl-merge/merger/schema"
)

// Emit generates clean DDL SQL from the schema model.
// Output order:
//  1. SCHEMA objects
//  2. EXTENSION objects
//  3. Sequences
//  4. Types (ENUM, then composite/range)
//  5. DOMAIN objects
//  6. Tables
//  7. Indexes
//  8. FUNCTION + PROCEDURE objects
//  9. VIEW + MATERIALIZED VIEW objects
//  10. TRIGGER objects
//  11. POLICY + RULE objects
//  12. Unknowns (pass-through)
func Emit(s *schema.Schema) string {
	var sb strings.Builder

	// 1. SCHEMA
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjSchema {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n\n")
		}
	}

	// 2. EXTENSION
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjExtension {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n")
			for _, alter := range obj.PostAlters {
				sb.WriteString(alter)
				sb.WriteString(";\n")
			}
			sb.WriteString("\n")
		}
	}

	// 3. Sequences
	for _, seq := range s.Sequences {
		sb.WriteString(fmt.Sprintf("CREATE SEQUENCE %s", seq.Name))
		if seq.Body != "" {
			sb.WriteString(" " + seq.Body)
		}
		sb.WriteString(";\n\n")
	}

	// 4. Types (ENUM)
	for _, t := range s.Types {
		labels := make([]string, len(t.Labels))
		for i, l := range t.Labels {
			labels[i] = fmt.Sprintf("'%s'", l)
		}
		sb.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (\n    %s\n);\n\n",
			t.Name, strings.Join(labels, ",\n    ")))
	}
	// 4b. Types (composite / range — verbatim)
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjType {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n\n")
		}
	}

	// 5. DOMAIN
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjDomain {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n")
			for _, alter := range obj.PostAlters {
				sb.WriteString(alter)
				sb.WriteString(";\n")
			}
			sb.WriteString("\n")
		}
	}

	// 6. Tables
	for _, t := range s.Tables {
		sb.WriteString(emitTable(t))
		sb.WriteString("\n")
	}

	// 6b. Partition tables (CREATE TABLE ... PARTITION OF — verbatim)
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjPartition {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n\n")
		}
	}

	// 7. Indexes
	for _, idx := range s.Indexes {
		unique := ""
		if idx.Unique {
			unique = "UNIQUE "
		}
		concurrently := ""
		if idx.Concurrently {
			concurrently = "CONCURRENTLY "
		}
		ifNotExists := ""
		if idx.IfNotExists {
			ifNotExists = "IF NOT EXISTS "
		}
		sb.WriteString(fmt.Sprintf("CREATE %sINDEX %s%s%s ON %s %s;\n\n",
			unique, concurrently, ifNotExists, idx.Name, idx.TableName, idx.Body))
	}

	// 8. FUNCTION + PROCEDURE
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjFunction || obj.Kind == parser.ObjProcedure {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n")
			for _, alter := range obj.PostAlters {
				sb.WriteString(alter)
				sb.WriteString(";\n")
			}
			sb.WriteString("\n")
		}
	}

	// 9. VIEW + MATERIALIZED VIEW
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjView || obj.Kind == parser.ObjMatView {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n")
			for _, alter := range obj.PostAlters {
				sb.WriteString(alter)
				sb.WriteString(";\n")
			}
			sb.WriteString("\n")
		}
	}

	// 10. TRIGGER
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjTrigger {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n\n")
		}
	}

	// 11. POLICY + RULE
	for _, obj := range s.Objects {
		if obj.Kind == parser.ObjPolicy || obj.Kind == parser.ObjRule {
			sb.WriteString(obj.SQL)
			sb.WriteString(";\n")
			for _, alter := range obj.PostAlters {
				sb.WriteString(alter)
				sb.WriteString(";\n")
			}
			sb.WriteString("\n")
		}
	}

	// 12. Truncates (deduped per table set)
	for _, trunc := range s.Truncates {
		sb.WriteString("TRUNCATE ")
		sb.WriteString(strings.Join(trunc.Tables, ", "))
		if trunc.RestartIdentity {
			sb.WriteString(" RESTART IDENTITY")
		}
		if trunc.Cascade {
			sb.WriteString(" CASCADE")
		}
		sb.WriteString(";\n\n")
	}

	// 13. Unknowns
	for _, u := range s.Unknowns {
		sb.WriteString(u)
		sb.WriteString(";\n\n")
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

func emitTable(t *schema.Table) string {
	var sb strings.Builder
	keyword := "TABLE"
	if t.Temporary {
		keyword = "TEMPORARY TABLE"
	} else if t.Unlogged {
		keyword = "UNLOGGED TABLE"
	}
	sb.WriteString(fmt.Sprintf("CREATE %s %s (\n", keyword, t.Name))

	totalItems := len(t.Columns) + len(t.Constraints)
	i := 0

	for _, col := range t.Columns {
		i++
		comma := ","
		if i == totalItems {
			comma = ""
		}
		sb.WriteString("    " + emitColumn(col) + comma + "\n")
	}

	for _, c := range t.Constraints {
		i++
		comma := ","
		if i == totalItems {
			comma = ""
		}
		line := ""
		if c.Name != "" {
			line = fmt.Sprintf("CONSTRAINT %s %s", c.Name, c.Definition)
		} else {
			line = c.Definition
		}
		sb.WriteString("    " + line + comma + "\n")
	}

	if t.PartitionBy != "" {
		sb.WriteString(") PARTITION BY " + t.PartitionBy + ";\n")
	} else {
		sb.WriteString(");\n")
	}
	return sb.String()
}

func emitColumn(col parser.ColumnDef) string {
	parts := []string{col.Name, col.DataType}
	if col.Collation != "" {
		parts = append(parts, "COLLATE "+col.Collation)
	}
	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if col.Default != nil {
		parts = append(parts, "DEFAULT "+*col.Default)
	}
	parts = append(parts, col.InlineConstraints...)
	return strings.Join(parts, " ")
}
