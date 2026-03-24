package merger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_BasicMerge(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")

	writeFile(t, dir, "01_create.sql", "CREATE TABLE users (id bigint, name text);")

	n, err := Run(Config{InputDir: dir, OutputPath: out})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 1 {
		t.Errorf("processed files: got %d, want 1", n)
	}

	got := readFile(t, out)
	if !strings.Contains(got, "CREATE TABLE users (") {
		t.Errorf("output missing CREATE TABLE:\n%s", got)
	}
}

func TestRun_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")

	writeFile(t, dir, "01_create.sql", "CREATE TABLE t (id int);")
	writeFile(t, dir, "02_alter.sql", "ALTER TABLE t ADD COLUMN name text;")

	if _, err := Run(Config{InputDir: dir, OutputPath: out}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := readFile(t, out)
	if !strings.Contains(got, "name text") {
		t.Errorf("output missing added column:\n%s", got)
	}
}

func TestRun_DropTable(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")

	writeFile(t, dir, "01_create.sql", "CREATE TABLE old (id int);")
	writeFile(t, dir, "02_drop.sql", "DROP TABLE old;")

	if _, err := Run(Config{InputDir: dir, OutputPath: out}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := readFile(t, out)
	if strings.Contains(got, "CREATE TABLE old") {
		t.Errorf("dropped table should not appear in output:\n%s", got)
	}
}

func TestRun_ReturnsFileCount(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")

	writeFile(t, dir, "01_a.sql", "CREATE TABLE a (id int);")
	writeFile(t, dir, "02_b.sql", "CREATE TABLE b (id int);")
	writeFile(t, dir, "03_c.sql", "CREATE TABLE c (id int);")

	n, err := Run(Config{InputDir: dir, OutputPath: out})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 3 {
		t.Errorf("processed files: got %d, want 3", n)
	}
}

func TestRun_InputDirNotExist(t *testing.T) {
	_, err := Run(Config{InputDir: "/nonexistent", OutputPath: "/tmp/out.sql"})
	if err == nil {
		t.Error("expected error for nonexistent input dir")
	}
}

func TestRun_OutputNotWritable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "01_a.sql", "CREATE TABLE a (id int);")

	_, err := Run(Config{InputDir: dir, OutputPath: "/nonexistent/dir/out.sql"})
	if err == nil {
		t.Error("expected error for non-writable output path")
	}
}

func TestRun_SeparatorFlagWarns(t *testing.T) {
	// -separator flag is ignored; Run should still succeed
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")
	writeFile(t, dir, "01_a.sql", "CREATE TABLE a (id int);")

	_, err := Run(Config{InputDir: dir, OutputPath: out, SeparatorTemplate: "---"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_WritesOutputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "merged.sql")
	writeFile(t, dir, "01_seq.sql", "CREATE SEQUENCE s START 1;")
	writeFile(t, dir, "02_tbl.sql", "CREATE TABLE t (id int);")

	if _, err := Run(Config{InputDir: dir, OutputPath: out}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := readFile(t, out)
	// Ordering: sequences appear before tables
	seqPos := strings.Index(got, "CREATE SEQUENCE")
	tblPos := strings.Index(got, "CREATE TABLE")
	if seqPos < 0 || tblPos < 0 || seqPos >= tblPos {
		t.Errorf("expected sequence before table in output:\n%s", got)
	}
}

// readFile is a test helper that reads a file and returns its content.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile(%s): %v", path, err)
	}
	return string(b)
}
