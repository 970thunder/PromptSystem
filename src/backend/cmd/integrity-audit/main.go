package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"promptos-backend/internal/config"
	"promptos-backend/internal/database"
	"promptos-backend/internal/store"
)

// The command is intentionally one-shot: deploy it with a systemd timer or
// cron so the no-Swap production host does not carry another resident worker.
func main() {
	cfg := config.Load()
	db, err := database.OpenMySQL(cfg)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	report, err := store.AuditMySQLPolymorphicIntegrity(db)
	if err != nil {
		log.Fatalf("audit polymorphic targets: %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		log.Fatalf("encode audit report: %v", err)
	}
	fmt.Println(string(encoded))
	if report.Total() > 0 {
		os.Exit(1)
	}
}
