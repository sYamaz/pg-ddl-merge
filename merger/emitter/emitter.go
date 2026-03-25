package emitter

import (
	"fmt"
	"strings"

	"github.com/shunyamazaki/pg-ddl-merge/merger/parser"
	"github.com/shunyamazaki/pg-ddl-merge/merger/schema"
)

// Emit generates clean DDL SQL from the schema model.
func Emit(s *schema.Schema) string {
	var sb strings.Builder

	for _, seq := range s.Sequences {
		sb.WriteString(fmt.Sprintf("CREATE SEQUENCE %s", seq.Name))
		if seq.Body != "" {
			sb.WriteString(" " + seq.Body)
		}
		sb.WriteString(";\n\n")
	}

	for _, t := range s.Types {
		labels := make([]string, len(t.Labels))
		for i, l := range t.Labels {
			labels[i] = fmt.Sprintf("'%s'", l)
		}
		sb.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (\n    %s\n);\n\n",
			t.Name, strings.Join(labels, ",\n    ")))
	}

	for _, t := range s.Tables {
		sb.WriteString(emitTable(t))
		sb.WriteString("\n")
	}

	for _, idx := range s.Indexes {
		unique := ""
		if idx.Unique {
			unique = "UNIQUE "
		}
		sb.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s %s;\n\n",
			unique, idx.Name, idx.TableName, idx.Body))
	}

	for _, u := range s.Unknowns {
		sb.WriteString(u)
		sb.WriteString(";\n\n")
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

func emitTable(t *schema.Table) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", t.Name))

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

	sb.WriteString(");\n")
	return sb.String()
}

func emitColumn(col parser.ColumnDef) string {
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
