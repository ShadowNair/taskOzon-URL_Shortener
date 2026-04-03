package link

import (
	"context"
	"errors"
	"testing"
	domainlink "url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"
)

type repoStub struct {
	registerFn   func(ctx context.Context, links domainlink.Link) (string, error)
	getByShortFn func(ctx context.Context, shortCode string) (string, error)
}

func (r *repoStub) RegisterShortURL(ctx context.Context, links domainlink.Link) (string, error) {
	return r.registerFn(ctx, links)
}

func (r *repoStub) GetByShortCode(ctx context.Context, shortCode string) (string, error) {
	return r.getByShortFn(ctx, shortCode)
}

type generatorStub struct {
	values []string
	err    error
	index  int
}

func (g *generatorStub) Generate(_ int) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	value := g.values[g.index]
	g.index++
	return value, nil
}

func TestUsecaseCreateShortLinkSuccess(t *testing.T) {
	repo := &repoStub{
		registerFn: func(_ context.Context, links domainlink.Link) (string, error) {
			if links.ShortCode != "abcDEF123_" {
				t.Fatalf("unexpected short code: %s", links.ShortCode)
			}
			return links.ShortCode, nil
		},
	}
	gen := &generatorStub{values: []string{"abcDEF123_"}}
	uc := New(repo, gen)

	got, err := uc.CreateShortLink(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("CreateShortLink() error = %v", err)
	}
	if got != "abcDEF123_" {
		t.Fatalf("CreateShortLink() = %s, want %s", got, "abcDEF123_")
	}
}

func TestUsecaseCreateShortLinkRetriesOnConflict(t *testing.T) {
	calls := 0
	repo := &repoStub{
		registerFn: func(_ context.Context, links domainlink.Link) (string, error) {
			calls++
			if calls == 1 {
				return "", globalerrors.ErrShortCodeConflict
			}
			return links.ShortCode, nil
		},
	}
	gen := &generatorStub{values: []string{"aaaaaaaaaa", "bbbbbbbbbb"}}
	uc := New(repo, gen)

	got, err := uc.CreateShortLink(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("CreateShortLink() error = %v", err)
	}
	if got != "bbbbbbbbbb" {
		t.Fatalf("CreateShortLink() = %s, want %s", got, "bbbbbbbbbb")
	}
}

func TestUsecaseCreateShortLinkGeneratorError(t *testing.T) {
	uc := New(&repoStub{}, &generatorStub{err: errors.New("rng failed")})

	_, err := uc.CreateShortLink(context.Background(), "https://example.com")
	if err == nil || err.Error() != "generate short code: rng failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsecaseCreateShortLinkRepositoryError(t *testing.T) {
	repo := &repoStub{
		registerFn: func(_ context.Context, _ domainlink.Link) (string, error) {
			return "", errors.New("db down")
		},
	}
	uc := New(repo, &generatorStub{values: []string{"aaaaaaaaaa"}})

	_, err := uc.CreateShortLink(context.Background(), "https://example.com")
	if err == nil || err.Error() != "register link: db down" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsecaseCreateShortLinkMaxAttemptsExceeded(t *testing.T) {
	repo := &repoStub{
		registerFn: func(_ context.Context, _ domainlink.Link) (string, error) {
			return "", globalerrors.ErrShortCodeConflict
		},
	}
	values := make([]string, maxGenerateAttempts)
	for i := range values {
		values[i] = "aaaaaaaaaa"
	}
	uc := New(repo, &generatorStub{values: values})

	_, err := uc.CreateShortLink(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUsecaseGetOriginalURLByShort(t *testing.T) {
	repo := &repoStub{
		getByShortFn: func(_ context.Context, shortCode string) (string, error) {
			if shortCode != "abcdefghij" {
				t.Fatalf("unexpected shortCode: %s", shortCode)
			}
			return "https://example.com", nil
		},
	}
	uc := New(repo, &generatorStub{})

	got, err := uc.GetOriginalURLByShort(context.Background(), "abcdefghij")
	if err != nil {
		t.Fatalf("GetOriginalURLByShort() error = %v", err)
	}
	if got != "https://example.com" {
		t.Fatalf("GetOriginalURLByShort() = %s", got)
	}
}