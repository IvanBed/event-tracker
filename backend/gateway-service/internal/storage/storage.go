package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS services (
    id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const schema2 = `
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY NOT NULL,
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);`

func InitializeServiceSchema(db *sql.DB, schema string) error {
	_, err := db.Exec(schema)
	return err
}

func NewDatabase(dbPath string) (*sql.DB, error) {
	//dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON"
	dsn := strings.TrimSpace(dbPath)
	fmt.Println("dsn", dsn)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {

		fmt.Println("Could not open sqlite file", err.Error())
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func CloseStorage(db *sql.DB) {
	db.Close()
}
