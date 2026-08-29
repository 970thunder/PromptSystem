package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"promptos-backend/internal/config"
)

func OpenMySQL(cfg config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		cfg.MySQLUser,
		cfg.MySQLPass,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDB,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	maxOpen := cfg.DBMaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.DBMaxIdleConns
	if maxIdle <= 0 || maxIdle > maxOpen {
		maxIdle = min(5, maxOpen)
	}
	lifetime := cfg.DBConnMaxLifetimeMin
	if lifetime <= 0 {
		lifetime = 30
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(lifetime) * time.Minute)

	var pingErr error
	for attempt := 0; attempt < 10; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			return db, nil
		}

		time.Sleep(2 * time.Second)
	}

	_ = db.Close()
	return nil, pingErr
}
