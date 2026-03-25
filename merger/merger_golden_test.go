package merger

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestRun_Golden(t *testing.T) {
	integrationDir := filepath.Join("..", "testdata", "integration")
	entries, err := os.ReadDir(integrationDir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", integrationDir, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			inputDir := filepath.Join(integrationDir, name, "input")
			wantPath := filepath.Join(integrationDir, name, "want.sql")
			outPath := filepath.Join(t.TempDir(), "merged.sql")

			if _, err := Run(Config{InputDir: inputDir, OutputPath: outPath}); err != nil {
				t.Fatalf("Run: %v", err)
			}

			got := readFile(t, outPath)

			if *update {
				if err := os.WriteFile(wantPath, []byte(got), 0644); err != nil {
					t.Fatalf("WriteFile %s: %v", wantPath, err)
				}
				t.Logf("updated %s", wantPath)
				return
			}

			want := readFile(t, wantPath)
			if got != want {
				t.Errorf("output mismatch for %s:\ngot:\n%s\nwant:\n%s", name, got, want)
			}
		})
	}
}
