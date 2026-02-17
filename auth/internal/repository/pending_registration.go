package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/auth/internal/domain"
)

var ErrPendingRegistrationNotFound = errors.New("pending registration not found")

type PendingRegistrationRepository interface {
	// Create stores a new pending registration, replacing any existing one for the same email
	Create(ctx context.Context, pr *domain.PendingRegistration) error

	// GetByToken retrieves a pending registration by verification token
	GetByToken(ctx context.Context, token string) (*domain.PendingRegistration, error)

	// GetByEmail retrieves a pending registration by email
	GetByEmail(ctx context.Context, email string) (*domain.PendingRegistration, error)

	// UpdateToken updates the verification token and expiry for a pending registration
	UpdateToken(ctx context.Context, pr *domain.PendingRegistration) error

	// Delete removes a pending registration by ID
	Delete(ctx context.Context, id uint64) error

	// DeleteExpired removes all expired pending registrations and returns count deleted
	DeleteExpired(ctx context.Context) (int64, error)
}

type pendingRegistrationRepository struct {
	db *sql.DB
}

func NewPendingRegistrationRepository(db *sql.DB) PendingRegistrationRepository {
	return &pendingRegistrationRepository{db: db}
}

func (r *pendingRegistrationRepository) Create(ctx context.Context, pr *domain.PendingRegistration) error {
	// Upsert: insert or replace existing pending registration for the same email
	query := `
		INSERT INTO pending_registrations (email, username, display_name, password_hash, verification_token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (email) DO UPDATE SET
			username = EXCLUDED.username,
			display_name = EXCLUDED.display_name,
			password_hash = EXCLUDED.password_hash,
			verification_token = EXCLUDED.verification_token,
			expires_at = EXCLUDED.expires_at,
			created_at = CURRENT_TIMESTAMP
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query,
		pr.Email, pr.Username, pr.DisplayName, pr.PasswordHash, pr.VerificationToken, pr.ExpiresAt,
	).Scan(&pr.ID, &pr.CreatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *pendingRegistrationRepository) GetByToken(ctx context.Context, token string) (*domain.PendingRegistration, error) {
	query := `
		SELECT id, email, username, display_name, password_hash, verification_token, expires_at, created_at
		FROM pending_registrations
		WHERE verification_token = $1
	`
	pr := &domain.PendingRegistration{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&pr.ID, &pr.Email, &pr.Username, &pr.DisplayName,
		&pr.PasswordHash, &pr.VerificationToken, &pr.ExpiresAt, &pr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPendingRegistrationNotFound
		}
		return nil, err
	}

	return pr, nil
}

func (r *pendingRegistrationRepository) GetByEmail(ctx context.Context, email string) (*domain.PendingRegistration, error) {
	query := `
		SELECT id, email, username, display_name, password_hash, verification_token, expires_at, created_at
		FROM pending_registrations
		WHERE email = $1
	`
	pr := &domain.PendingRegistration{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&pr.ID, &pr.Email, &pr.Username, &pr.DisplayName,
		&pr.PasswordHash, &pr.VerificationToken, &pr.ExpiresAt, &pr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPendingRegistrationNotFound
		}
		return nil, err
	}

	return pr, nil
}

func (r *pendingRegistrationRepository) UpdateToken(ctx context.Context, pr *domain.PendingRegistration) error {
	query := `
		UPDATE pending_registrations
		SET verification_token = $1, expires_at = $2
		WHERE id = $3
	`
	result, err := r.db.ExecContext(ctx, query, pr.VerificationToken, pr.ExpiresAt, pr.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPendingRegistrationNotFound
	}

	return nil
}

func (r *pendingRegistrationRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM pending_registrations WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPendingRegistrationNotFound
	}

	return nil
}

func (r *pendingRegistrationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM pending_registrations WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
