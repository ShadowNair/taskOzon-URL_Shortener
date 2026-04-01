package memory

import (
	"context"
	"testing"
	"time"

	"url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRepo() *Repository {
	return New()
}

func makeTestLink(originalURL, shortCode string) link.Link {
	return link.Link{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		CreatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}


func TestRepository_RegisterShortURL_Success_NewLink(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	testLink := makeTestLink("https://example.com", "abc123")

	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.NoError(t, err)
	assert.Equal(t, "abc123", shortCode)

	// Проверяем, что ссылка действительно сохранилась в обоих индексах
	original, err := repo.GetByShortCode(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", original)
}

func TestRepository_RegisterShortURL_DuplicateOriginalURL_ReturnsExisting(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	// Первый раз регистрируем
	firstLink := makeTestLink("https://example.com", "abc123")
	_, err := repo.RegisterShortURL(ctx, firstLink)
	require.NoError(t, err)

	// Второй раз — тот же original_url, но другой short_code (игнорируется)
	secondLink := makeTestLink("https://example.com", "xyz789")
	returnedCode, err := repo.RegisterShortURL(ctx, secondLink)

	assert.NoError(t, err)
	assert.Equal(t, "abc123", returnedCode) // Вернули старый код, а не новый!

	// Проверяем, что в хранилище остался первый вариант
	original, err := repo.GetByShortCode(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", original)

	// "xyz789" не должен быть зарегистрирован
	_, err = repo.GetByShortCode(ctx, "xyz789")
	assert.ErrorIs(t, err, globalerrors.ErrNotFound)
}

func TestRepository_RegisterShortURL_ShortCodeConflict_DifferentOriginal(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	firstLink := makeTestLink("https://first.com", "same123")
	_, err := repo.RegisterShortURL(ctx, firstLink)
	require.NoError(t, err)

	secondLink := makeTestLink("https://second.com", "same123")
	returnedCode, err := repo.RegisterShortURL(ctx, secondLink)

	assert.ErrorIs(t, err, globalerrors.ErrShortCodeConflict)
	assert.Empty(t, returnedCode)

	original, err := repo.GetByShortCode(ctx, "same123")
	assert.NoError(t, err)
	assert.Equal(t, "https://first.com", original)
}

func TestRepository_RegisterShortURL_SameLinkTwice_Idempotent(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	testLink := makeTestLink("https://example.com", "abc123")

	code1, err1 := repo.RegisterShortURL(ctx, testLink)
	assert.NoError(t, err1)
	assert.Equal(t, "abc123", code1)

	code2, err2 := repo.RegisterShortURL(ctx, testLink)
	assert.NoError(t, err2)
	assert.Equal(t, "abc123", code2)
}

func TestRepository_RegisterShortURL_ContextCanceled(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testLink := makeTestLink("https://example.com", "abc123")
	shortCode, err := repo.RegisterShortURL(ctx, testLink)

	assert.NoError(t, err)
	assert.Equal(t, "abc123", shortCode)
}

func TestRepository_GetByShortCode_Success(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	// Предварительно регистрируем ссылку
	testLink := makeTestLink("https://example.com", "abc123")
	_, err := repo.RegisterShortURL(ctx, testLink)
	require.NoError(t, err)

	// Получаем по short_code
	originalURL, err := repo.GetByShortCode(ctx, "abc123")

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", originalURL)
}

func TestRepository_GetByShortCode_NotFound(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	originalURL, err := repo.GetByShortCode(ctx, "nonexistent")

	assert.ErrorIs(t, err, globalerrors.ErrNotFound)
	assert.Empty(t, originalURL)
}

func TestRepository_GetByShortCode_EmptyShortCode(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	ctx := context.Background()

	originalURL, err := repo.GetByShortCode(ctx, "")

	assert.ErrorIs(t, err, globalerrors.ErrNotFound)
	assert.Empty(t, originalURL)
}

func TestRepository_ConcurrentAccess_NoRace(t *testing.T) {
	t.Parallel()
	repo := New()
	ctx := context.Background()

	// Запускаем горутины на запись и чтение
	done := make(chan bool)

	// 10 горутин пишут разные ссылки
	for i := 0; i < 10; i++ {
		go func(idx int) {
			link := makeTestLink(
				"https://example.com/"+string(rune(idx+'0')),
				"code"+string(rune(idx+'0')),
			)
			_, _ = repo.RegisterShortURL(ctx, link)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func(idx int) {
			_, _ = repo.GetByShortCode(ctx, "code"+string(rune(idx+'0')))
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}