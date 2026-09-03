package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"promptos-backend/internal/config"
	"promptos-backend/internal/database"
	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

type taskResult struct {
	Task                  string                            `json:"task"`
	Polymorphic           *store.PolymorphicIntegrityReport `json:"polymorphic,omitempty"`
	Counters              *store.CounterIntegrityReport     `json:"counters,omitempty"`
	UploadsTrashed        int                               `json:"uploadsTrashed,omitempty"`
	UploadsDeleteFailures int                               `json:"uploadsDeleteFailures,omitempty"`
}

func main() {
	task := flag.String("task", "all", "maintenance task: integrity, counters, uploads, or all")
	olderThan := flag.Duration("older-than", 24*time.Hour, "age threshold for pending uploads")
	flag.Parse()

	if *olderThan <= 0 {
		log.Fatal("older-than must be positive")
	}
	if *task != "all" && *task != "integrity" && *task != "counters" && *task != "uploads" {
		log.Fatalf("unknown task %q", *task)
	}
	cfg := config.Load()
	db, err := database.OpenMySQL(cfg)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	result := taskResult{Task: *task}
	failed := false
	if *task == "all" || *task == "integrity" {
		report, auditErr := store.AuditMySQLPolymorphicIntegrity(db)
		if auditErr != nil {
			log.Printf("polymorphic audit failed: %v", auditErr)
			failed = true
		} else {
			result.Polymorphic = &report
			failed = failed || report.Total() > 0
		}
	}
	if *task == "all" || *task == "counters" {
		report, auditErr := store.AuditMySQLPromptCounters(db)
		if auditErr != nil {
			log.Printf("counter audit failed: %v", auditErr)
			failed = true
		} else {
			result.Counters = &report
			failed = failed || report.Total() > 0
		}
	}
	if *task == "all" || *task == "uploads" {
		count, deleteFailures, cleanupErr := cleanupUploads(cfg, db, *olderThan)
		if cleanupErr != nil {
			log.Printf("upload cleanup failed: %v", cleanupErr)
			failed = true
		} else {
			result.UploadsTrashed = count
			result.UploadsDeleteFailures = deleteFailures
			failed = failed || deleteFailures > 0
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("encode maintenance result: %v", err)
	}
	log.Print(string(encoded))
	if failed {
		os.Exit(1)
	}
}

func cleanupUploads(cfg config.Config, db *sql.DB, olderThan time.Duration) (int, int, error) {
	imageStorage, err := storage.NewImageStorage(cfg)
	if err != nil {
		return 0, 0, err
	}
	uploads := store.NewMySQLUploadStore(db)
	return cleanupUploadRecords(cfg.UploadProvider, uploads, imageStorage, olderThan)
}

func cleanupUploadRecords(configuredProvider string, uploads store.UploadManager, imageStorage storage.ImageStorage, olderThan time.Duration) (int, int, error) {
	if uploads == nil || imageStorage == nil {
		return 0, 0, fmt.Errorf("upload cleanup dependencies are unavailable")
	}
	records, err := uploads.ListUnreferencedUploads(time.Now().UTC().Add(-olderThan))
	if err != nil {
		return 0, 0, err
	}
	trashed, failures := 0, 0
	for _, rec := range records {
		if !sameStorageProvider(rec.Provider, configuredProvider) {
			// Never delete an object through a provider selected by a different
			// deployment configuration. Leave the row pending for an operator
			// to rerun with the matching provider credentials.
			failures++
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		deleteErr := imageStorage.Delete(ctx, rec.ObjectKey)
		cancel()
		if deleteErr != nil {
			failures++
			continue
		}
		if err := uploads.SoftDeleteUpload(rec.ObjectKey, rec.OwnerID); err != nil {
			failures++
			continue
		}
		trashed++
	}
	return trashed, failures, nil
}

func sameStorageProvider(recorded, configured string) bool {
	normalize := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "rustfs" || value == "s3" {
			return "r2"
		}
		if value == "" {
			return "local"
		}
		return value
	}
	return normalize(recorded) == normalize(configured)
}
