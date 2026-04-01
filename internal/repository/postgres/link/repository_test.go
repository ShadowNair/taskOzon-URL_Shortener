package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRepo(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	repo := New(db)
	cleanup := func() {db.Close()}
	return repo, mock, cleanup
}

func TestRepository_RegisterURL_Success(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx, _ := context.WithTimeout(context.Background(), 1 * time.Second)
	testLink := link.Link{
		OriginalURL: "https://hello.org",
		ShortCode: "abc123",
	}

	mock.ExpectExec(sqlTextForRegisterShortURL).WithArgs(testLink.OriginalURL, testLink.ShortCode).WillReturnResult(sqlmock.NewResult(1, 1))

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.NoError(t, err)
	assert.Equal(t, testLink.ShortCode, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_DuplicateOriginalURL_LinkPKey(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	// Первый запрос — конфликт по unique-индексу (23505)
	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(&pgconn.PgError{
			Code:         "23505",
			ConstraintName: "link_pkey",
		})

	// Второй запрос — поиск существующего short_code по original_url
	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(testLink.OriginalURL).
		WillReturnRows(sqlmock.NewRows([]string{"short_code"}).AddRow("existing_code"))

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.NoError(t, err)
	assert.Equal(t, "existing_code", shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_DuplicateShortCode_ExplicitConstraint(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "link_short_code_key",
		})

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.ErrorIs(t, err, globalerrors.ErrShortCodeConflict)
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_DuplicateShortCode_IndexConstraint(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "idx_link_short_code",
		})

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.ErrorIs(t, err, globalerrors.ErrShortCodeConflict)
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_UnknownConstraint_FallbackToLookup(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	// Конфликт с неизвестным именем ограничения
	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "some_unknown_constraint",
		})

	// Fallback: попытка найти по original_url — успешно
	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(testLink.OriginalURL).
		WillReturnRows(sqlmock.NewRows([]string{"short_code"}).AddRow("fallback_code"))

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.NoError(t, err)
	assert.Equal(t, "fallback_code", shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_UnknownConstraint_LookupNotFound(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "unknown_constraint",
		})

	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(testLink.OriginalURL).
		WillReturnError(sql.ErrNoRows)

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.ErrorIs(t, err, globalerrors.ErrShortCodeConflict)
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RegisterShortURL_GenericDBError(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	testLink := link.Link{
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
	}

	mock.ExpectExec(sqlTextForRegisterShortURL).
		WithArgs(testLink.OriginalURL, testLink.ShortCode).
		WillReturnError(errors.New("connection refused"))

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "problem with insert link")
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_getByOriginalURL_Success(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	originalURL := "https://example.com"
	expectedShortCode := "abc123"

	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(originalURL).
		WillReturnRows(sqlmock.NewRows([]string{"short_code"}).AddRow(expectedShortCode))

	shortCode, err := repo.getByOriginalURL(ctx, originalURL)

	assert.NoError(t, err)
	assert.Equal(t, expectedShortCode, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_getByOriginalURL_NotFound(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	originalURL := "https://example.com"

	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(originalURL).
		WillReturnError(sql.ErrNoRows)

	shortCode, err := repo.getByOriginalURL(ctx, originalURL)

	assert.ErrorIs(t, err, globalerrors.ErrNotFound)
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_getByOriginalURL_DBError(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	originalURL := "https://example.com"

	mock.ExpectQuery(sqlTextForGetByOriginalURL).
		WithArgs(originalURL).
		WillReturnError(errors.New("query timeout"))

	shortCode, err := repo.getByOriginalURL(ctx, originalURL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query by original url")
	assert.Empty(t, shortCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByShortCode_Success(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	shortCode := "abc123"
	expectedOriginalURL := "https://example.com"

	mock.ExpectQuery(sqlTextForGetByShortCode).
		WithArgs(shortCode).
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow(expectedOriginalURL))

	originalURL, err := repo.GetByShortCode(ctx, shortCode)

	assert.NoError(t, err)
	assert.Equal(t, expectedOriginalURL, originalURL)
}

func TestRepository_GetByShortCode_NotFound(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	shortCode := "abc123"

	mock.ExpectQuery(sqlTextForGetByShortCode).
		WithArgs(shortCode).
		WillReturnError(sql.ErrNoRows)

	originalURL, err := repo.GetByShortCode(ctx, shortCode)

	assert.ErrorIs(t, err, globalerrors.ErrNotFound)
	assert.Empty(t, originalURL)
}

func TestRepository_GetByShortCode_DBError(t *testing.T) {
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	shortCode := "abc123"

	mock.ExpectQuery(sqlTextForGetByShortCode).
		WithArgs(shortCode).
		WillReturnError(errors.New("database unavailable"))

	originalURL, err := repo.GetByShortCode(ctx, shortCode)

	assert.ErrorContains(t, err, "database unavailable")
	assert.Empty(t, originalURL)
}