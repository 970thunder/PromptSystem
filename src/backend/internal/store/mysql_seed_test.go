package store

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/database"

	_ "github.com/go-sql-driver/mysql"
)

// TestProductionSeedDoesNotCreateDemoData exercises the same seed entry point
// used by production startup with includeDemo=false. It uses a throwaway
// database so a regression cannot contaminate the shared integration schema.
func TestProductionSeedDoesNotCreateDemoData(t *testing.T) {
	dsn := testMySQLDSN(t)
	base, databaseName := seedTestDSNParts(t, dsn)
	testName := databaseName + "_seed_prod_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")

	admin, err := sql.Open("mysql", base+"mysql")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + testName); err != nil {
		t.Fatalf("drop stale test database: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + testName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS " + testName) }()

	db, err := sql.Open("mysql", base+testName+"?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	schema, err := os.ReadFile("../../sql/schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	script := strings.ReplaceAll(string(schema), "CREATE DATABASE IF NOT EXISTS promptos DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", "")
	script = strings.ReplaceAll(script, "USE promptos;", "USE `"+testName+"`;")
	if _, err := db.Exec(script); err != nil {
		t.Fatalf("initialize test schema: %v", err)
	}
	if err := database.RunMigrations(db, "../../sql/migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if err := SeedMySQLData(db, false); err != nil {
		t.Fatalf("production seed: %v", err)
	}
	var users, prompts, categories int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM prompts").Scan(&prompts); err != nil {
		t.Fatalf("count prompts: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categories); err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if users != 0 || prompts != 0 {
		t.Fatalf("production seed created demo data: users=%d prompts=%d", users, prompts)
	}
	if categories == 0 {
		t.Fatal("production seed must initialize reference categories")
	}
}

// TestMySQLPromptTagsMigrationNormalizesLegacyRows proves that the production
// migration repairs rows written before application-level normalization was
// introduced. It also verifies a second migration pass is a no-op.
func TestMySQLPromptTagsMigrationNormalizesLegacyRows(t *testing.T) {
	dsn := testMySQLDSN(t)
	base, databaseName := seedTestDSNParts(t, dsn)
	testName := databaseName + "_tags_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")

	admin, err := sql.Open("mysql", base+"mysql")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + testName); err != nil {
		t.Fatalf("drop stale test database: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + testName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS " + testName) }()

	db, err := sql.Open("mysql", base+testName+"?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	if err := database.RunMigrations(db, "../../sql/migrations"); err != nil {
		t.Fatalf("initialize schema and migrations: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version = '0015_normalize_prompt_tags'`); err != nil {
		t.Fatalf("reset tag migration ledger: %v", err)
	}

	const userID = 990001
	const promptID = 990001
	if _, err := db.Exec(`
		INSERT INTO users (id, username, email, password, status)
		VALUES (?, ?, ?, NULL, 1)
	`, userID, "tag-migration-user", "tag-migration@example.com"); err != nil {
		t.Fatalf("insert fixture user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO prompts (id, title, content, category_id, user_id, status)
		VALUES (?, 'Tag migration fixture', 'content', 1, ?, 1)
	`, promptID, userID); err != nil {
		t.Fatalf("insert fixture prompt: %v", err)
	}
	// Leading whitespace is legal in the legacy unique index, while the two
	// values collapse to the same canonical tag after normalization.
	if _, err := db.Exec(`INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?), (?, ?)`, promptID, "Brand", promptID, " brand "); err != nil {
		t.Fatalf("insert legacy tags: %v", err)
	}

	if err := database.RunMigrations(db, "../../sql/migrations"); err != nil {
		t.Fatalf("apply tag normalization migration: %v", err)
	}
	var tags []string
	rows, err := db.Query(`SELECT tag FROM prompt_tags WHERE prompt_id = ? ORDER BY id`, promptID)
	if err != nil {
		t.Fatalf("read normalized tags: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			t.Fatalf("scan normalized tag: %v", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate normalized tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "brand" {
		t.Fatalf("normalized tags = %#v, want [brand]", tags)
	}

	if err := database.RunMigrations(db, "../../sql/migrations"); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}
}

func seedTestDSNParts(t *testing.T, dsn string) (base, databaseName string) {
	t.Helper()
	idx := strings.IndexByte(dsn, '/')
	if idx < 0 || idx == len(dsn)-1 {
		t.Fatalf("invalid test DSN %q", dsn)
	}
	base = dsn[:idx+1]
	databaseName = strings.SplitN(dsn[idx+1:], "?", 2)[0]
	if databaseName == "" || strings.ContainsAny(databaseName, "` ;\t\r\n") {
		t.Fatalf("invalid test database name %q", databaseName)
	}
	return base, databaseName
}
