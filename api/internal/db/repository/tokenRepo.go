package repository

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error
	StoreRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, raw string, exp time.Time) error
	CheckRefreshToken(ctx context.Context, raw string) (uuid.UUID, error)
	RotateRefreshToken(ctx context.Context, userID uuid.UUID, oldRaw, newRaw string, newExp time.Time) error
	RotateRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, oldRaw, newRaw string, newExp time.Time) error
}

type tokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepo(db *pgxpool.Pool) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(raw))
	_, err = tx.Exec(ctx, `
	  INSERT INTO refresh_tokens(token_id, user_id, expires_at)
	  VALUES ($1, $2, $3)
	`, hash[:], userID, exp)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return err
	}

	return tx.Commit(ctx)
}

func (r *tokenRepository) StoreRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, raw string, exp time.Time) error {
	hash := sha256.Sum256([]byte(raw))
	_, err := tx.Exec(ctx, `
	  INSERT INTO refresh_tokens(token_id, user_id, expires_at)
	  VALUES ($1, $2, $3)
	`, hash[:], userID, exp)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return err
	}

	return tx.Commit(ctx)
}

func (r *tokenRepository) CheckRefreshToken(ctx context.Context, raw string) (uuid.UUID, error) {
	hash := sha256.Sum256([]byte(raw))
	var userID uuid.UUID
	err := r.db.QueryRow(ctx, `
	  SELECT user_id FROM refresh_tokens
	  WHERE token_id = $1 AND expires_at > now()
	`, hash[:]).Scan(&userID)
	return userID, err
}

func (r *tokenRepository) RotateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	oldRaw, newRaw string,
	newExp time.Time,
) error {

	oldHash := sha256.Sum256([]byte(oldRaw))
	newHash := sha256.Sum256([]byte(newRaw))

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	// immediately expire (or delete) the previous token.
	_, err = tx.Exec(ctx, `
		UPDATE refresh_tokens
		   SET expires_at = now()
		 WHERE token_id = $1
		   AND user_id  = $2
	`, oldHash[:], userID)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	// insert the new token.
	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (token_id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, newHash[:], userID, newExp)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return err
	}

	return tx.Commit(ctx)
}

func (r *tokenRepository) RotateRefreshTokenWithTx(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	oldRaw, newRaw string,
	newExp time.Time,
) error {

	oldHash := sha256.Sum256([]byte(oldRaw))
	newHash := sha256.Sum256([]byte(newRaw))

	// 1️⃣  Immediately expire (or delete) the previous token.
	_, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		   SET expires_at = now()
		 WHERE token_id = $1
		   AND user_id  = $2
	`, oldHash[:], userID)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	// 2️⃣  Insert the new token.
	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (token_id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, newHash[:], userID, newExp)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return err
	}

	return tx.Commit(ctx)
}
