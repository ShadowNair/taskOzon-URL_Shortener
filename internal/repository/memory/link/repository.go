package memory

import (
	"context"
	"sync"
	"time"
	"url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"
)

type Repository struct {
	mu	sync.RWMutex
	byOriginal	map[string]link.Link
	byShort		map[string]link.Link
}

func New() *Repository {
	return &Repository{
		byOriginal: make(map[string]link.Link),
		byShort: make(map[string]link.Link),
	}
}

func (r *Repository) RegisterShortURL(ctx context.Context, links link.Link) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if link, ok := r.byOriginal[links.OriginalURL]; ok {
		return link.ShortCode, nil
	}

	if existing, ok := r.byShort[links.ShortCode]; ok && existing.OriginalURL != links.OriginalURL {
		return "", globalerrors.ErrShortCodeConflict
	}

	link := link.Link{
		OriginalURL: links.OriginalURL,
		ShortCode: links.ShortCode,
		CreatedAt: time.Now().UTC(),
	}

	r.byOriginal[link.OriginalURL] = link
	r.byShort[link.ShortCode] = link

	return link.ShortCode, nil
}

func (r *Repository) GetByShortCode(ctx context.Context, shortCode string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if link, ok := r.byShort[shortCode]; ok {
		return link.OriginalURL, nil
	}
	return "", globalerrors.ErrNotFound
}