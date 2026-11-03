package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key violation")
	ErrInvalidInput = errors.New("invalid input")
)

type ServiceRepository struct {
	db *sql.DB
}

func InitServiceRepo(db *sql.DB) *ServiceRepository {

	return &ServiceRepository{db: db}
}

func (serviceRepo *ServiceRepository) CreateService(ctx context.Context, ServiceName string, PasswordHash string) (*ServiceDesc, error) {
	service := &ServiceDesc{
		Id:          uuid.New(),
		ServiceName: ServiceName,
		Password:    PasswordHash,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	query := `
        INSERT INTO services (id, service_name, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `

	fmt.Println(query)

	_, err := serviceRepo.db.Exec(query, service.Id, service.ServiceName, service.Password, service.CreatedAt, service.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (serviceRepo *ServiceRepository) GetServiceByName(serviceName string) (*ServiceDesc, error) {
	query := `SELECT id, service_name, password_hash, created_at, updated_at FROM services WHERE service_name = $1`
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

func (serviceRepo *ServiceRepository) GetServiceByID(id uuid.UUID) (*ServiceDesc, error) {
	query := `SELECT id, service_name, password_hash, created_at, updated_at FROM services WHERE id = $1`
	var service ServiceDesc
	var lastLogin sql.NullTime
	err := serviceRepo.db.QueryRow(query, id).Scan(
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
