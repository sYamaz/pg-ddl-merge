package parser

import "strings"

// Split splits raw SQL text into individual statement strings.
// It handles single-quoted strings, dollar-quoted strings, line comments, and block comments.
func Split(sql string) []string {
	var stmts []string
	var buf strings.Builder

	type state int
	const (
		stNormal state = iota
		stSingleQuote
		stLineComment
		stBlockComment
	)

	st := stNormal
	i := 0
	dollarTag := ""

	for i < len(sql) {
		ch := sql[i]

		switch st {
		case stNormal:
			switch {
			case ch == '\'':
				st = stSingleQuote
				buf.WriteByte(ch)
				i++

			case ch == '-' && i+1 < len(sql) && sql[i+1] == '-':
				st = stLineComment
				i += 2

			case ch == '/' && i+1 < len(sql) && sql[i+1] == '*':
				st = stBlockComment
				i += 2

			case ch == '$':
				// Check for dollar-quoting: $tag$ or $$
				end := strings.IndexByte(sql[i+1:], '$')
				if end >= 0 {
					tag := sql[i : i+1+end+1]
					// tag is like $$ or $foo$
					closeIdx := strings.Index(sql[i+len(tag):], tag)
					if closeIdx >= 0 {
						// write the entire dollar-quoted block verbatim
						block := sql[i : i+len(tag)+closeIdx+len(tag)]
						buf.WriteString(block)
						dollarTag = ""
						_ = dollarTag
						i += len(block)
					} else {
						buf.WriteByte(ch)
						i++
					}
				} else {
					buf.WriteByte(ch)
					i++
				}

			case ch == ';':
				s := strings.TrimSpace(buf.String())
				if s != "" {
					stmts = append(stmts, s)
				}
				buf.Reset()
				i++

			default:
				buf.WriteByte(ch)
				i++
			}

		case stSingleQuote:
			buf.WriteByte(ch)
			if ch == '\'' {
				// check for escaped quote ''
				if i+1 < len(sql) && sql[i+1] == '\'' {
					buf.WriteByte(sql[i+1])
					i += 2
				} else {
					st = stNormal
					i++
				}
			} else {
				i++
			}

		case stLineComment:
			if ch == '\n' {
				st = stNormal
			}
			i++

		case stBlockComment:
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				st = stNormal
				i += 2
			} else {
				i++
			}
		}
	}

	// handle trailing statement without semicolon
	if s := strings.TrimSpace(buf.String()); s != "" {
		stmts = append(stmts, s)
	}

	return stmts
}
