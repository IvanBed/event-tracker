package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key violation")
	ErrInvalidInput = errors.New("invalid input")
)

type ServiceRepo struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS service (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);`

func (serviceRepo *ServiceRepo) initializeSchema() error {
	_, err := serviceRepo.db.Exec(schema)
	return err
}

func newDatabase(dbPath string) (*sql.DB, error) {
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

func InitServiceRepo(dbPath string) (*ServiceRepo, error) {
	var serviceRepo *ServiceRepo

	db, err := newDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	serviceRepo = &ServiceRepo{db: db}
	err = serviceRepo.initializeSchema()

	if err != nil {
		return nil, err
	}
	return serviceRepo, nil
}

func (serviceRepo *ServiceRepo) CloseServiceRepo() {
	serviceRepo.db.Close()
}

func (serviceRepo *ServiceRepo) CreateService(ctx context.Context, ServiceName string, PasswordHash string) (*ServiceDesc, error) {

	query := `
        INSERT INTO service (name, password_hash, created_at, updated_at)
        VALUES (?, ?, ?, ?)
    `
	var service ServiceDesc

	now := time.Now()
	result, err := serviceRepo.db.ExecContext(ctx, query,
		ServiceName,
		PasswordHash,
		now,
		now,
	)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			return nil, ErrDuplicateKey
		}
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	service.Id = string(id)
	service.ServiceName = ServiceName
	service.Password = PasswordHash
	service.CreatedAt = now
	service.UpdatedAt = now

	return &service, nil
}

func (serviceRepo *ServiceRepo) GetServiceByName(serviceName string) (*ServiceDesc, error) {
	query := `SELECT id, service_name, password_hash, created_at, updated_at FROM service WHERE email = $1`
	var service ServiceDesc
	var lastLogin sql.NullTime
	err := serviceRepo.db.QueryRow(query, serviceName).Scan(
		&service.Id,
		&service.ServiceName,
		&service.Password,
		&service.CreatedAt,
		&lastLogin,
	)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		service.UpdatedAt = lastLogin.Time
	}
	return &service, nil
}

func isUniqueConstraintError(err error) bool {
	// SQLite error message contains "UNIQUE constraint failed"
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") ||
		contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
