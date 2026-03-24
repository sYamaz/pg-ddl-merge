package merger

import (
	"fmt"
	"os"

	"github.com/shunyamazaki/pg-ddl-merge/merger/emitter"
	"github.com/shunyamazaki/pg-ddl-merge/merger/parser"
	"github.com/shunyamazaki/pg-ddl-merge/merger/schema"
)

type Config struct {
	InputDir          string
	OutputPath        string
	SeparatorTemplate string
}

func Run(cfg Config) (int, error) {
	if cfg.SeparatorTemplate != "" {
		fmt.Fprintln(os.Stderr, "warning: -separator flag is ignored in semantic merge mode")
	}

	files, err := sortedSQLFiles(cfg.InputDir)
	if err != nil {
		return 0, err
	}

	s := schema.New()

	for _, f := range files {
		raw, err := os.ReadFile(f.Path)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", f.Name, err)
		}

		stmts := parser.Split(string(raw))
		for _, stmtStr := range stmts {
			stmt, err := parser.Parse(stmtStr)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", f.Name, err)
			}
			if err := s.Apply(stmt); err != nil {
				return 0, fmt.Errorf("%s: %w", f.Name, err)
			}
		}
	}

	output := emitter.Emit(s)

	if err := os.WriteFile(cfg.OutputPath, []byte(output), 0644); err != nil {
		return 0, fmt.Errorf("failed to write output: %w", err)
	}

	return len(files), nil
}
