package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"

	"github.com/jackc/pgconn"
)

const (
	sqlTextForRegisterShortURL = `
	INSERT INTO link (original_url, short_code) VALUES ($1, $2)
	`
	sqlTextForGetByOriginalURL = `
	SELECT short_code FROM link WHERE original_url = $1
	`
	sqlTextForGetByShortCode = `
	SELECT original_url FROM link WHERE short_code = $1
	`
)

type Repository struct {
	sql *sql.DB
}

func New(sql *sql.DB) *Repository {
	return &Repository{
		sql: sql,
	}
}

func (r *Repository) RegisterShortURL(ctx context.Context, links link.Link) (string, error) {
	_, err := r.sql.ExecContext(ctx, sqlTextForRegisterShortURL, links.OriginalURL, links.ShortCode)
	if err == nil {
		return links.ShortCode, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch strings.ToLower(pgErr.ConstraintName) {

		case "link_pkey":
			links.ShortCode, err = r.getByOriginalURL(ctx, links.OriginalURL)
			if err != nil {
				return "", fmt.Errorf("load existing link after original url conflict: %w", err)
			}
			return links.ShortCode, nil

		case "link_short_code_key", "idx_link_short_code":
			return "", globalerrors.ErrShortCodeConflict

		default:
			links.ShortCode, err = r.getByOriginalURL(ctx, links.OriginalURL)
			if err != nil {
				return "", globalerrors.ErrShortCodeConflict
			}
			return links.ShortCode, nil
		}
	}
	return "", fmt.Errorf("problem with insert link: %w", err)
}

func (r *Repository) getByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	err := r.sql.QueryRowContext(ctx, sqlTextForGetByOriginalURL, originalURL).Scan(&shortURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", globalerrors.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query by original url: %w", err)
	}
	return shortURL, nil
}

func (r *Repository) GetByShortCode(ctx context.Context, shortCode string) (string, error) {
	var originalURL string
	err := r.sql.QueryRowContext(ctx, sqlTextForGetByShortCode, shortCode).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", globalerrors.ErrNotFound
		}
		return "", fmt.Errorf("query by short code: %w", err)
	}
	return originalURL, nil
}