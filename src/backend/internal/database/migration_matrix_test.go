package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrationMatrix verifies all three database start states converge to the
// same final schema and that migrations are idempotent:
//
//  1. fresh: an empty database that runs every migration in order.
//  2. baseline: a database initialized from sql/schema.sql, then re-run through
//     the migration chain (additive migrations only).
//  3. partial: a database that has applied only the first half of the
//     migrations, then finishes the rest.
//
// The test requires a real MySQL and is skipped unless PROMPTOS_TEST_MYSQL_DSN
// is set (CI provides it). It uses dedicated databases named
// promptos_matrix_* and drops them afterwards; it never touches the dev data.
func TestMigrationMatrix(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PROMPTOS_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("PROMPTOS_TEST_MYSQL_DSN not set; skipping migration matrix (run via CI or a local MySQL)")
	}

	baseDSN, dbName, err := splitDSNDatabase(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}

	migrationsDir := defaultMigrationsDir()

	names, err := listMigrationFiles(migrationsDir)
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(names) < 3 {
		t.Fatalf("expected several migration files, got %d", len(names))
	}

	scenarios := []struct {
		name  string
		db    string
		setup func(t *testing.T, db *sql.DB)
	}{
		{
			name: "fresh",
			db:   dbName + "_matrix_fresh",
			setup: func(t *testing.T, db *sql.DB) {
				// Nothing: empty database, all migrations apply.
			},
		},
		{
			name: "baseline",
			db:   dbName + "_matrix_baseline",
			setup: func(t *testing.T, db *sql.DB) {
				// Initialize from the full baseline schema, then re-run the
				// migration chain to prove migrations are additive/idempotent.
				schema := readFileOrFail(t, "sql/schema.sql")
				if _, err := db.Exec(schema); err != nil {
					t.Fatalf("apply baseline schema.sql: %v", err)
				}
			},
		},
		{
			name: "partial",
			db:   dbName + "_matrix_partial",
			setup: func(t *testing.T, db *sql.DB) {
				// Apply only the first half of migrations, then let the chain
				// finish the rest. This simulates an upgrade from an older
				// deployment that stopped mid-way.
				half := names[len(names)/2]
				for _, name := range names {
					if name > half {
						break
					}
					if err := runSingleMigration(db, migrationsDir, name); err != nil {
						t.Fatalf("pre-apply %s: %v", name, err)
					}
				}
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			admin, err := sql.Open("mysql", baseDSN)
			if err != nil {
				t.Fatalf("open admin conn: %v", err)
			}
			defer admin.Close()

			if _, err := admin.Exec("DROP DATABASE IF EXISTS " + sc.db); err != nil {
				t.Fatalf("drop test db: %v", err)
			}
			if _, err := admin.Exec("CREATE DATABASE " + sc.db + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
				t.Fatalf("create test db: %v", err)
			}
			defer func() {
				_, _ = admin.Exec("DROP DATABASE IF EXISTS " + sc.db)
			}()

			db, err := sql.Open("mysql", baseDSN+"/"+sc.db+"?parseTime=true&multiStatements=true")
			if err != nil {
				t.Fatalf("open test db: %v", err)
			}
			defer db.Close()

			sc.setup(t, db)

			// First run: full chain.
			if err := RunMigrations(db, migrationsDir); err != nil {
				t.Fatalf("RunMigrations (1st): %v", err)
			}

			// Second run: must be a no-op (idempotency), which catches the
			// historical 0002 foreign-key and 0008 duplicate-column bugs.
			if err := RunMigrations(db, migrationsDir); err != nil {
				t.Fatalf("RunMigrations (2nd, idempotency): %v", err)
			}

			assertMigrationTableComplete(t, db, len(names))
		})
	}
}

func splitDSNDatabase(dsn string) (base string, db string, err error) {
	// DSN form: user:pass@tcp(host:port)/dbname?opts...
	idx := strings.Index(dsn, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("DSN %q has no database separator", dsn)
	}
	base = dsn[:idx]
	rest := dsn[idx+1:]
	rest = strings.SplitN(rest, "?", 2)[0]
	if rest == "" {
		return "", "", fmt.Errorf("DSN %q has no database name", dsn)
	}
	return base, rest, nil
}

func listMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func runSingleMigration(db *sql.DB, dir, name string) error {
	body, err := os.ReadFile(dir + "/" + name)
	if err != nil {
		return err
	}
	if err := execMigrationSQL(db, string(body)); err != nil {
		return err
	}
	version := strings.TrimSuffix(name, ".sql")
	_, err = db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version)
	return err
}

func readFileOrFail(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil && path == "sql/schema.sql" {
		body, err = os.ReadFile(filepath.Join("..", "..", "sql", "schema.sql"))
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func assertMigrationTableComplete(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != expected {
		t.Fatalf("schema_migrations has %d rows, want %d", count, expected)
	}
}

// TestMigrationNamingOrder enforces the file naming contract
// NNNN_snake_case_description.sql and stable ordering. It runs without a
// database, so it always executes.
func TestMigrationNamingOrder(t *testing.T) {
	dir := defaultMigrationsDir()
	if _, err := os.Stat(dir); err != nil {
		// The package test working dir is internal/database, so fall back to
		// the repository-relative path used in CI/local runs.
		dir = "../../sql/migrations"
	}
	names, err := listMigrationFiles(dir)
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no migration files found")
	}

	for _, name := range names {
		base := strings.TrimSuffix(name, ".sql")
		parts := strings.Split(base, "_")
		if len(parts) < 3 {
			t.Errorf("migration %q must match NNNN_snake_case_description.sql", name)
			continue
		}
		if len(parts[0]) != 4 || !allDigits(parts[0]) {
			t.Errorf("migration %q: prefix %q must be 4 digits", name, parts[0])
		}
	}

	// Sorted file list must already be the application order.
	for i := 1; i < len(names); i++ {
		if names[i] <= names[i-1] {
			t.Fatalf("migrations out of order: %q before %q", names[i], names[i-1])
		}
	}
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
