package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies numbered SQL files from dir that are not yet recorded in schema_migrations.
func RunMigrations(db *sql.DB, dir string) error {
	if strings.TrimSpace(dir) == "" {
		dir = defaultMigrationsDir()
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("migrations directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("migrations path %q is not a directory", dir)
	}

	// A new MySQL database has no tables yet, while the historical migration
	// chain starts with ALTER TABLE statements. Apply the checked-in baseline
	// exactly once for that empty-database case, then let the same migration
	// ledger drive every subsequent upgrade. Existing or partially migrated
	// databases are never overwritten.
	if err := ensureBaselineSchema(db, dir); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(64) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, filepath.Ext(name))
		var applied int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied > 0 {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := execMigrationSQL(db, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}

	return nil
}

func ensureBaselineSchema(db *sql.DB, migrationsDir string) error {
	var tableCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
	`).Scan(&tableCount); err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	if tableCount != 0 {
		return nil
	}

	schemaPath := filepath.Join(filepath.Dir(migrationsDir), "schema.sql")
	body, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read baseline schema %q: %w", schemaPath, err)
	}
	if err := execMigrationSQL(db, string(body)); err != nil {
		return fmt.Errorf("apply baseline schema %q: %w", schemaPath, err)
	}
	return nil
}

func defaultMigrationsDir() string {
	if dir := strings.TrimSpace(os.Getenv("PROMPTOS_MIGRATIONS_DIR")); dir != "" {
		return dir
	}

	candidates := []string{
		"sql/migrations",
		"/app/sql/migrations",
		filepath.Join("..", "sql", "migrations"),
		filepath.Join("..", "..", "sql", "migrations"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return "sql/migrations"
}

func execMigrationSQL(db *sql.DB, script string) error {
	statements := splitSQLStatements(script)
	if len(statements) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func splitSQLStatements(script string) []string {
	var (
		statements []string
		builder    strings.Builder
	)

	lines := strings.Split(script, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		// schema.sql is also used as the baseline for a database that the
		// Compose entrypoint has already created. Do not require the migration
		// account to have the global CREATE DATABASE privilege just to keep an
		// existing database in place.
		if strings.HasPrefix(strings.ToUpper(trimmed), "CREATE DATABASE IF NOT EXISTS ") {
			continue
		}
		if strings.EqualFold(trimmed, "USE promptos;") || strings.EqualFold(trimmed, "USE promptos") {
			continue
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(builder.String())
			builder.Reset()
			if stmt != "" {
				statements = append(statements, strings.TrimSuffix(stmt, ";"))
			}
		}
	}

	remainder := strings.TrimSpace(builder.String())
	if remainder != "" {
		statements = append(statements, strings.TrimSuffix(remainder, ";"))
	}

	return statements
}
