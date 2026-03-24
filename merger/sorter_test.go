package merger

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSortedSQLFiles_NumericOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "10_c.sql", "")
	writeFile(t, dir, "02_b.sql", "")
	writeFile(t, dir, "01_a.sql", "")

	files, err := sortedSQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []int{1, 2, 10}
	for i, f := range files {
		if f.Prefix != want[i] {
			t.Errorf("index %d: got prefix %d, want %d", i, f.Prefix, want[i])
		}
	}
}

func TestSortedSQLFiles_DuplicatePrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "01_b.sql", "")
	writeFile(t, dir, "01_a.sql", "")

	files, err := sortedSQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].Name != "01_a.sql" {
		t.Errorf("got %s, want 01_a.sql", files[0].Name)
	}
	if files[1].Name != "01_b.sql" {
		t.Errorf("got %s, want 01_b.sql", files[1].Name)
	}
}

func TestSortedSQLFiles_IgnoresNonSQL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "01_a.sql", "")
	writeFile(t, dir, "README.md", "")
	writeFile(t, dir, ".gitkeep", "")
	writeFile(t, dir, "01.sql", "") // no underscore

	files, err := sortedSQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestSortedSQLFiles_IgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "01_a.sql", "")
	if err := os.Mkdir(filepath.Join(dir, "01_migrations"), 0755); err != nil {
		t.Fatal(err)
	}

	files, err := sortedSQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestSortedSQLFiles_NoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := sortedSQLFiles(dir)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSortedSQLFiles_DirNotExist(t *testing.T) {
	_, err := sortedSQLFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
