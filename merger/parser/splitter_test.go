package parser

import (
	"testing"
)

func TestSplit_Basic(t *testing.T) {
	got := Split("SELECT 1; SELECT 2;")
	if len(got) != 2 {
		t.Fatalf("got %d statements, want 2", len(got))
	}
	if got[0] != "SELECT 1" {
		t.Errorf("got %q, want %q", got[0], "SELECT 1")
	}
	if got[1] != "SELECT 2" {
		t.Errorf("got %q, want %q", got[1], "SELECT 2")
	}
}

func TestSplit_TrailingWithoutSemicolon(t *testing.T) {
	got := Split("SELECT 1; SELECT 2")
	if len(got) != 2 {
		t.Fatalf("got %d statements, want 2", len(got))
	}
}

func TestSplit_Empty(t *testing.T) {
	got := Split("")
	if len(got) != 0 {
		t.Errorf("got %d statements, want 0", len(got))
	}
}

func TestSplit_OnlyWhitespace(t *testing.T) {
	got := Split("   \n\t  ")
	if len(got) != 0 {
		t.Errorf("got %d statements, want 0", len(got))
	}
}

func TestSplit_SingleQuoteWithSemicolon(t *testing.T) {
	// semicolon inside single-quoted string must not split
	got := Split("INSERT INTO t VALUES ('a;b');")
	if len(got) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(got), got)
	}
}

func TestSplit_EscapedSingleQuote(t *testing.T) {
	got := Split("INSERT INTO t VALUES ('it''s here'); SELECT 1;")
	if len(got) != 2 {
		t.Fatalf("got %d statements, want 2: %v", len(got), got)
	}
}

func TestSplit_LineComment(t *testing.T) {
	sql := "SELECT 1; -- this is a comment\nSELECT 2;"
	got := Split(sql)
	if len(got) != 2 {
		t.Fatalf("got %d statements, want 2: %v", len(got), got)
	}
}

func TestSplit_BlockComment(t *testing.T) {
	sql := "SELECT /* ; */ 1; SELECT 2;"
	got := Split(sql)
	if len(got) != 2 {
		t.Fatalf("got %d statements, want 2: %v", len(got), got)
	}
}

func TestSplit_DollarQuote(t *testing.T) {
	// semicolon inside dollar-quoted block must not split
	sql := "CREATE FUNCTION f() RETURNS void AS $$BEGIN; END;$$ LANGUAGE plpgsql;"
	got := Split(sql)
	if len(got) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(got), got)
	}
}

func TestSplit_DollarQuoteTagged(t *testing.T) {
	sql := "DO $body$BEGIN; END;$body$;"
	got := Split(sql)
	if len(got) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(got), got)
	}
}

func TestSplit_WhitespaceOnlySections(t *testing.T) {
	got := Split(";   ;  SELECT 1;")
	if len(got) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(got), got)
	}
	if got[0] != "SELECT 1" {
		t.Errorf("got %q", got[0])
	}
}
