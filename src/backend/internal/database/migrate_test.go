package database

import "testing"

func TestSplitSQLStatementsSkipsCommentsAndUse(t *testing.T) {
	script := `-- header
USE promptos;

ALTER TABLE users ADD COLUMN github_id BIGINT NULL;

UPDATE users SET bio = 'x' WHERE id = 1;
`
	statements := splitSQLStatements(script)
	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d: %#v", len(statements), statements)
	}
	if statements[0] != "ALTER TABLE users ADD COLUMN github_id BIGINT NULL" {
		t.Fatalf("unexpected first statement: %q", statements[0])
	}
}

func TestDefaultMigrationsDir(t *testing.T) {
	dir := defaultMigrationsDir()
	if dir == "" {
		t.Fatal("expected non-empty migrations dir")
	}
}
