package store

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"promptos-backend/internal/database"

	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLSettingsSetupBootstrap covers the settings/setup store on a real
// MySQL: migration table present, one-shot bootstrap, settings persistence,
// and concurrent CompleteSetup producing exactly one winner. It uses a
// throwaway database so the one-shot semantics stay repeatable.
func TestMySQLSettingsSetupBootstrap(t *testing.T) {
	dsn := testMySQLDSN(t)
	base, databaseName := seedTestDSNParts(t, dsn)
	testName := databaseName + "_settings_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")

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
		t.Fatalf("run migrations: %v", err)
	}

	settings := NewMySQLSettingsStore(db)

	completed, err := settings.Completed()
	if err != nil || completed {
		t.Fatalf("fresh database must be uninitialized: completed=%v err=%v", completed, err)
	}

	// Concurrent bootstraps: exactly one wins on the marker primary key.
	const workers = 4
	var wg sync.WaitGroup
	results := make(chan int, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := settings.CompleteSetup(SetupInput{
				Username:  "it_bootstrap_admin",
				Email:     "it-bootstrap-admin@example.com",
				Password:  "StrongPass123!",
				Settings:  map[string]string{SettingSiteName: "IT Hub", SettingRegistrationOn: "false"},
				RequestID: "it-setup",
				Now:       time.Now().UTC(),
			})
			if err != nil {
				errs <- err
				return
			}
			results <- id
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	winners := 0
	for range results {
		winners++
	}
	var lastErr error
	for err := range errs {
		lastErr = err
	}
	if winners != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (last error: %v)", winners, lastErr)
	}

	completed, err = settings.Completed()
	if err != nil || !completed {
		t.Fatalf("completed after bootstrap: %v %v", completed, err)
	}
	hasAdmin, err := settings.HasAdmin()
	if err != nil || !hasAdmin {
		t.Fatalf("hasAdmin after bootstrap: %v %v", hasAdmin, err)
	}
	all, err := settings.GetAll()
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if all[SettingSiteName] != "IT Hub" || all[SettingRegistrationOn] != "false" {
		t.Fatalf("settings not persisted: %#v", all)
	}

	// The bootstrap transaction must also grant the admin role.
	var roleCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_roles ur JOIN users u ON u.id = ur.user_id WHERE ur.role = 'admin' AND u.email = 'it-bootstrap-admin@example.com'`).Scan(&roleCount); err != nil {
		t.Fatalf("count roles: %v", err)
	}
	if roleCount != 1 {
		t.Fatalf("expected exactly 1 admin role row, got %d", roleCount)
	}

	// A second, sequential attempt is still rejected.
	if _, err := settings.CompleteSetup(SetupInput{
		Username: "it_bootstrap_admin2",
		Email:    "it-bootstrap-admin2@example.com",
		Password: "StrongPass123!",
	}); err != ErrSetupCompleted {
		t.Fatalf("second setup err=%v", err)
	}

	// SetMany upsert + clear round trip.
	if err := settings.SetMany(map[string]string{SettingSiteDescription: "desc"}); err != nil {
		t.Fatalf("setmany: %v", err)
	}
	if err := settings.SetMany(map[string]string{SettingSiteDescription: ""}); err != nil {
		t.Fatalf("setmany clear: %v", err)
	}
	all, err = settings.GetAll()
	if err != nil {
		t.Fatalf("getall2: %v", err)
	}
	if _, exists := all[SettingSiteDescription]; exists {
		t.Fatal("cleared key must be deleted")
	}
}
