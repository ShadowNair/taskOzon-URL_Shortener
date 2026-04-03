package link

import (
	"context"
	"errors"
	"fmt"
	domainlink "url-shortener/internal/domain/model/link"
	"url-shortener/pkg/globalerrors"
)

const (
	shortCodeLength     = 10
	maxGenerateAttempts = 32
)

type RepoI interface {
	RegisterShortURL(ctx context.Context, links domainlink.Link) (string, error)
	GetByShortCode(ctx context.Context, shortCode string) (string, error)
}

type generatorI interface {
	Generate(n int) (string, error)
}

type Usecase struct {
	repo      RepoI
	generator generatorI
}

func New(repo RepoI, generator generatorI) *Usecase {
	return &Usecase{
		repo:      repo,
		generator: generator,
	}
}

func (u *Usecase) CreateShortLink(ctx context.Context, originalURL string) (string, error) {
	for i := 0; i < maxGenerateAttempts; i++ {
		code, err := u.generator.Generate(shortCodeLength)
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}

		candidate := domainlink.Link{
			OriginalURL: originalURL,
			ShortCode:   code,
		}

		storedCode, err := u.repo.RegisterShortURL(ctx, candidate)
		switch {
		case errors.Is(err, globalerrors.ErrShortCodeConflict):
			continue
		case err == nil:
			return storedCode, nil
		default:
			return "", fmt.Errorf("register link: %w", err)
		}
	}

	return "", fmt.Errorf("failed to allocate short code after %d attempts", maxGenerateAttempts)
}

func (u *Usecase) GetOriginalURLByShort(ctx context.Context, shortCode string) (string, error) {
	return u.repo.GetByShortCode(ctx, shortCode)
}