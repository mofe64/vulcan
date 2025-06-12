package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetUserIdByOIDCSub(ctx context.Context, sub string) (uuid.UUID, error)
	InsertNewUser(ctx context.Context, sub string, email string) (uuid.UUID, error)
	InsertNewUserWithTx(ctx context.Context, tx pgx.Tx, sub string, email string) (uuid.UUID, error)
	UpsertUser(ctx context.Context, sub string, email string) (uuid.UUID, error)
	UpsertUserWithTx(ctx context.Context, tx pgx.Tx, sub string, email string) (uuid.UUID, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserIdByOIDCSub(ctx context.Context, sub string) (uuid.UUID, error) {
	if sub == "" {
		return uuid.Nil, errors.New("missing oidc_sub")
	}

	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id FROM users WHERE oidc_sub = $1;
	`, sub).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, nil // user not found
		}
		return uuid.Nil, err // other error
	}
	return id, nil
}

func (r *userRepository) InsertNewUser(ctx context.Context, sub string, email string) (uuid.UUID, error) {
	if sub == "" {
		return uuid.Nil, errors.New("missing oidc_sub")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	err = tx.QueryRow(ctx, `
		INSERT INTO users (id, oidc_sub, email)
		VALUES ($1, $2, $3)
		RETURNING id;
	`, id, sub, sql.NullString{String: email, Valid: email != ""}).Scan(&id)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *userRepository) InsertNewUserWithTx(ctx context.Context, tx pgx.Tx, sub string, email string) (uuid.UUID, error) {
	if sub == "" {
		return uuid.Nil, errors.New("missing oidc_sub")
	}
	id := uuid.New()
	err := tx.QueryRow(ctx, `
		INSERT INTO users (id, oidc_sub, email)
		VALUES ($1, $2, $3)
		RETURNING id;
	`, id, sub, sql.NullString{String: email, Valid: email != ""}).Scan(&id)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpsertUser inserts a new user or updates the e-mail of an existing one.
// Parameters
//
//	sub   – the stable OIDC “subject” claim (Dex makes this unique per connector)
//	email – the user’s primary e-mail; can be empty if GitHub kept it private
//
// Returns
//
//	userID – the UUID in your `users` table
//	err    – any database error
func (r *userRepository) UpsertUser(ctx context.Context, sub string, email string) (uuid.UUID, error) {
	if sub == "" {
		return uuid.Nil, errors.New("missing oidc_sub")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}

	// if the row exists we keep its id, else we generate a fresh UUID.
	id := uuid.New()

	err = tx.QueryRow(ctx, `
		INSERT INTO users (id, oidc_sub, email)
		VALUES ($1, $2, $3)
		ON CONFLICT (oidc_sub)
		DO UPDATE
		SET email = EXCLUDED.email
		WHERE users.email IS DISTINCT FROM EXCLUDED.email
		RETURNING id;
	`, id, sub, sql.NullString{String: email, Valid: email != ""}).Scan(&id)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *userRepository) UpsertUserWithTx(ctx context.Context, tx pgx.Tx, sub string, email string) (uuid.UUID, error) {
	if sub == "" {
		return uuid.Nil, errors.New("missing oidc_sub")
	}

	// if the row exists we keep its id, else we generate a fresh UUID.
	id := uuid.New()

	err := tx.QueryRow(ctx, `
		INSERT INTO users (id, oidc_sub, email)
		VALUES ($1, $2, $3)
		ON CONFLICT (oidc_sub)
		DO UPDATE
		SET email = EXCLUDED.email
		WHERE users.email IS DISTINCT FROM EXCLUDED.email
		RETURNING id;
	`, id, sub, sql.NullString{String: email, Valid: email != ""}).Scan(&id)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
