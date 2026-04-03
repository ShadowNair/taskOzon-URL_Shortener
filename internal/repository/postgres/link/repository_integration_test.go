package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	domainlink "url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

const testInitSchema = `
CREATE TABLE IF NOT EXISTS link (
    original_url TEXT PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

var loadEnvOnce sync.Once

func TestRepository_RegisterShortURL_RealDB_Success(t *testing.T) {
	repo, db := newRealDBRepository(t)

	got, err := repo.RegisterShortURL(context.Background(), domainlink.Link{
		OriginalURL: "https://example.com/success",
		ShortCode:   "Abc123_XyZ",
	})
	require.NoError(t, err)
	require.Equal(t, "Abc123_XyZ", got)

	storedOriginal := mustGetOriginalURL(t, db, "Abc123_XyZ")
	require.Equal(t, "https://example.com/success", storedOriginal)
}

func TestRepository_RegisterShortURL_RealDB_IdempotentByOriginalURL(t *testing.T) {
	repo, _ := newRealDBRepository(t)

	first, err := repo.RegisterShortURL(context.Background(), domainlink.Link{
		OriginalURL: "https://example.com/same",
		ShortCode:   "FirstCode1",
	})
	require.NoError(t, err)
	require.Equal(t, "FirstCode1", first)

	second, err := repo.RegisterShortURL(context.Background(), domainlink.Link{
		OriginalURL: "https://example.com/same",
		ShortCode:   "OtherCode2",
	})
	require.NoError(t, err)
	require.Equal(t, "FirstCode1", second)
}

func TestRepository_RegisterShortURL_RealDB_ShortCodeConflict(t *testing.T) {
	repo, _ := newRealDBRepository(t)

	_, err := repo.RegisterShortURL(context.Background(), domainlink.Link{
		OriginalURL: "https://example.com/one",
		ShortCode:   "SameCode_1",
	})
	require.NoError(t, err)

	_, err = repo.RegisterShortURL(context.Background(), domainlink.Link{
		OriginalURL: "https://example.com/two",
		ShortCode:   "SameCode_1",
	})
	require.ErrorIs(t, err, globalerrors.ErrShortCodeConflict)
}

func TestRepository_GetByShortCode_RealDB_Success(t *testing.T) {
	repo, db := newRealDBRepository(t)
	insertFixtureLink(t, db, "https://example.com/found", "FindCode_1")

	got, err := repo.GetByShortCode(context.Background(), "FindCode_1")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/found", got)
}

func TestRepository_GetByShortCode_RealDB_NotFound(t *testing.T) {
	repo, _ := newRealDBRepository(t)

	got, err := repo.GetByShortCode(context.Background(), "Missing__1")
	require.ErrorIs(t, err, globalerrors.ErrNotFound)
	require.Empty(t, got)
}

func newRealDBRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()

	loadEnvOnce.Do(func() {
		_ = godotenv.Load()
	})

	if os.Getenv("RUN_PG_TESTS") != "1" {
		t.Skip("set RUN_PG_TESTS=1 and start PostgreSQL before running repository integration tests")
	}

	db, err := sql.Open("pgx", testConnectionString())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return db.PingContext(ctx) == nil
	}, 10*time.Second, 300*time.Millisecond, "postgres is not reachable")

	_, err = db.Exec(testInitSchema)
	require.NoError(t, err)

	_, err = db.Exec(`TRUNCATE TABLE link`)
	require.NoError(t, err)

	return New(db), db
}

func testConnectionString() string {
	host := envOrDefaultTest("POSTGRES_HOST", "localhost")
	port := envOrDefaultTest("POSTGRES_PORT", "5432")
	user := envOrDefaultTest("POSTGRES_USER", "shortener")
	password := envOrDefaultTest("POSTGRES_PASSWORD", "shortener")
	database := envOrDefaultTest("POSTGRES_DB", "shortener")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		database,
	)
}

func envOrDefaultTest(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func insertFixtureLink(t *testing.T, db *sql.DB, originalURL, shortCode string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO link (original_url, short_code) VALUES ($1, $2)`, originalURL, shortCode)
	require.NoError(t, err)
}

func mustGetOriginalURL(t *testing.T, db *sql.DB, shortCode string) string {
	t.Helper()
	var originalURL string
	err := db.QueryRow(`SELECT original_url FROM link WHERE short_code = $1`, shortCode).Scan(&originalURL)
	require.NoError(t, err)
	return originalURL
}
