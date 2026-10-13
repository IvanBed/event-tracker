package storage

import (
	"database/sql"
	"time"
)

func InitializeServiceSchema(db *sql.DB, schema string) error {
	_, err := db.Exec(schema)
	return err
}

func NewDatabase(dbPath string) (*sql.DB, error) {
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1) // SQLite supports only one writer at a time
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
