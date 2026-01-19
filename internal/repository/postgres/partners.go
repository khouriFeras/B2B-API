package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type partnerRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewPartnerRepository creates a new partner repository
func NewPartnerRepository(db *sql.DB, logger *zap.Logger) *partnerRepository {
	return &partnerRepository{
		db:     db,
		logger: logger,
	}
}

func (r *partnerRepository) GetByAPIKeyHash(ctx context.Context, apiKey string) (*domain.Partner, error) {
	// Since bcrypt hashes are salted and different each time, we can't do a direct lookup.
	// We need to iterate through active partners and verify the API key against each hash.
	// For production, consider adding a lookup_hash column (SHA256) for efficient lookup.
	
	query := `
		SELECT id, name, api_key_hash, webhook_url, is_active, created_at, updated_at
		FROM partners
		WHERE is_active = true
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to query partners", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var partner domain.Partner
		var webhookURL sql.NullString

		err := rows.Scan(
			&partner.ID,
			&partner.Name,
			&partner.APIKeyHash,
			&webhookURL,
			&partner.IsActive,
			&partner.CreatedAt,
			&partner.UpdatedAt,
		)

		if err != nil {
			continue
		}

		// Verify API key against stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(partner.APIKeyHash), []byte(apiKey)); err == nil {
			// Match found
			if webhookURL.Valid {
				partner.WebhookURL = &webhookURL.String
			}
			return &partner, nil
		}
	}

	return nil, &errors.ErrUnauthorized{Message: "invalid API key"}
}

func (r *partnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Partner, error) {
	query := `
		SELECT id, name, api_key_hash, webhook_url, is_active, created_at, updated_at
		FROM partners
		WHERE id = $1
	`

	var partner domain.Partner
	var webhookURL sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&partner.ID,
		&partner.Name,
		&partner.APIKeyHash,
		&webhookURL,
		&partner.IsActive,
		&partner.CreatedAt,
		&partner.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "partner", ID: id.String()}
	}
	if err != nil {
		r.logger.Error("Failed to get partner by ID", zap.Error(err))
		return nil, err
	}

	if webhookURL.Valid {
		partner.WebhookURL = &webhookURL.String
	}

	return &partner, nil
}

func (r *partnerRepository) Create(ctx context.Context, partner *domain.Partner) error {
	query := `
		INSERT INTO partners (id, name, api_key_hash, webhook_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	if partner.ID == uuid.Nil {
		partner.ID = uuid.New()
	}
	if partner.CreatedAt.IsZero() {
		partner.CreatedAt = now
	}
	if partner.UpdatedAt.IsZero() {
		partner.UpdatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		partner.ID,
		partner.Name,
		partner.APIKeyHash,
		partner.WebhookURL,
		partner.IsActive,
		partner.CreatedAt,
		partner.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create partner", zap.Error(err))
		return err
	}

	return nil
}

func (r *partnerRepository) Update(ctx context.Context, partner *domain.Partner) error {
	query := `
		UPDATE partners
		SET name = $2, api_key_hash = $3, webhook_url = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`

	partner.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		partner.ID,
		partner.Name,
		partner.APIKeyHash,
		partner.WebhookURL,
		partner.IsActive,
		partner.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update partner", zap.Error(err))
		return err
	}

	return nil
}
