//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/sYamaz/pg-ddl-merge/merger"
	"github.com/sYamaz/pg-ddl-merge/merger/parser"
)

var (
	pgContainerID string
	pgBaseURL     *url.URL
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer ctr.Terminate(ctx) //nolint:errcheck

	pgContainerID = ctr.GetContainerID()
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get connection string: %v\n", err)
		os.Exit(1)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse DSN: %v\n", err)
		os.Exit(1)
	}
	pgBaseURL = u

	// Verify postgres is truly ready to accept connections.
	if err := waitForPostgres(dsn, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "postgres not ready: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TestPGDDLMerge verifies that applying DDL files sequentially and applying the
// merged DDL both produce an identical schema, as reported by pg_dump.
func TestPGDDLMerge(t *testing.T) {
	integrationDir := filepath.Join("..", "testdata", "integration")
	entries, err := os.ReadDir(integrationDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	ctx := context.Background()
	adminDB := mustOpenDB(t, "postgres")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			inputDir := filepath.Join(integrationDir, name, "input")
			seqDB := "seq_" + sanitizeName(name)
			mergedDB := "mrg_" + sanitizeName(name)

			createDB(t, ctx, adminDB, seqDB)
			createDB(t, ctx, adminDB, mergedDB)

			// Apply optional setup.sql (prerequisites not part of merger input).
			setupSQL := filepath.Join(integrationDir, name, "setup.sql")
			for _, db := range []string{seqDB, mergedDB} {
				if err := applyFileIfExists(ctx, db, setupSQL); err != nil {
					t.Fatalf("setup.sql (%s): %v", db, err)
				}
			}

			// Apply input DDL files one by one (sequential)
			if err := applySequential(ctx, seqDB, inputDir); err != nil {
				t.Fatalf("applySequential: %v", err)
			}

			// Generate merged DDL and apply it as a single unit
			outFile := filepath.Join(t.TempDir(), "merged.sql")
			if _, err := merger.Run(merger.Config{InputDir: inputDir, OutputPath: outFile}); err != nil {
				t.Fatalf("merger.Run: %v", err)
			}
			if err := applyFile(ctx, mergedDB, outFile); err != nil {
				t.Fatalf("applyMerged: %v", err)
			}

			seqDump := pgDump(t, seqDB)
			mergedDump := pgDump(t, mergedDB)

			if seqDump != mergedDump {
				t.Errorf("schema mismatch for %s\n\n--- sequential ---\n%s\n\n--- merged ---\n%s",
					name, seqDump, mergedDump)
			}
		})
	}
}

// applySequential connects to dbName and executes each .sql file in inputDir in
// sorted order, splitting each file into individual statements.
func applySequential(ctx context.Context, dbName, inputDir string) error {
	db, err := openDB(dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(inputDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		if err := execSQLFile(ctx, db, f); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// applyFileIfExists calls applyFile only when the file exists; silently skips otherwise.
func applyFileIfExists(ctx context.Context, dbName, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	return applyFile(ctx, dbName, filePath)
}

// applyFile connects to dbName and executes all statements in the given SQL file.
func applyFile(ctx context.Context, dbName, filePath string) error {
	db, err := openDB(dbName)
	if err != nil {
		return err
	}
	defer db.Close()
	return execSQLFile(ctx, db, filePath)
}

func execSQLFile(ctx context.Context, db *sql.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, stmt := range parser.Split(string(raw)) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

// pgDump runs pg_dump inside the container and returns a normalized, sorted
// representation of the schema-only dump for dbName.
func pgDump(t *testing.T, dbName string) string {
	t.Helper()
	out, err := exec.Command("docker", "exec", pgContainerID,
		"pg_dump", "-U", "postgres",
		"--schema-only", "--no-owner", "--no-acl",
		dbName,
	).Output()
	if err != nil {
		t.Fatalf("pg_dump %s: %v", dbName, err)
	}
	return normalizeDump(string(out))
}

// normalizeDump strips pg_dump noise (comments, SET/SELECT config lines, blank
// lines) and sorts the remaining statements so that object-creation order does
// not affect comparison.
func normalizeDump(s string) string {
	var stmts []string
	var cur strings.Builder

	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip blanks, SQL comments, and psql metacommands (\connect, \restrict, etc.)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "\\") {
			continue
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(cur.String())
			cur.Reset()
			upper := strings.ToUpper(stmt)
			if strings.HasPrefix(upper, "SET ") ||
				strings.HasPrefix(upper, "SELECT PG_CATALOG.SET_CONFIG") {
				continue
			}
			stmts = append(stmts, stmt)
		}
	}

	sort.Strings(stmts)
	return strings.Join(stmts, "\n\n")
}

func openDB(dbName string) (*sql.DB, error) {
	u := *pgBaseURL
	u.Path = "/" + dbName
	return sql.Open("postgres", u.String())
}

func mustOpenDB(t *testing.T, dbName string) *sql.DB {
	t.Helper()
	db, err := openDB(dbName)
	if err != nil {
		t.Fatalf("sql.Open(%s): %v", dbName, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func createDB(t *testing.T, ctx context.Context, adminDB *sql.DB, name string) {
	t.Helper()
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("CREATE DATABASE %s: %v", name, err)
	}
	t.Cleanup(func() {
		// ignore error: best-effort cleanup
		adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+name) //nolint:errcheck
	})
}

// waitForPostgres retries pinging the database until it responds or the timeout elapses.
func waitForPostgres(dsn string, timeout time.Duration) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err = db.Ping(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for postgres: %w", err)
}

// sanitizeName converts an arbitrary string to a lowercase identifier safe for
// use as a PostgreSQL database name (max 40 chars).
func sanitizeName(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}
