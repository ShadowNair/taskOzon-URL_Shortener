package link

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"url-shortener/pkg/globalerrors"
	"url-shortener/pkg/response"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

const timeContext = 2 * time.Second
const shortCodeLength = 10

type usecaseI interface {
	CreateShortLink(ctx context.Context, originalURL string) (string, error)
	GetOriginalURLByShort(ctx context.Context, shortCode string) (string, error)
}

type Handler struct {
	usecase usecaseI
	validate *validator.Validate
}

func New(usecase usecaseI) *Handler {
	return &Handler{
		usecase: usecase,
		validate: validator.New(),
	}
}

type createRequest struct {
	URL string `json:"url" validate:"required,http_url"`
}

type shortLinkResponse struct {
	URL       string `json:"url"`
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

type originalURLResponse struct {
	URL string `json:"url"`
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSONResponse(w, http.StatusOK, "server ready", map[string]string{"status": "ok"})
}

func (h *Handler) CreateShortLink(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONResponse(w, http.StatusBadRequest, "invalid json body", nil)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.handleError(w, fmt.Errorf("%w: invalid url format", globalerrors.ErrInvalidURL))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeContext)
	defer cancel()

	shortCode, err := h.usecase.CreateShortLink(ctx, req.URL)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shortURL := buildShortURL(r, shortCode)
	response.JSONResponse(w, http.StatusCreated, "short url created", shortLinkResponse{
		URL:       req.URL,
		ShortCode: shortCode,
		ShortURL:  shortURL,
	})
}

func (h *Handler) GetOriginalURLByShort(w http.ResponseWriter, r *http.Request) {
	shortCode := mux.Vars(r)["shortCode"]
	if err := validateShortCode(shortCode); err != nil {
		h.handleError(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeContext)
	defer cancel()

	originalURL, err := h.usecase.GetOriginalURLByShort(ctx, shortCode)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSONResponse(w, http.StatusOK, "original url found", originalURLResponse{URL: originalURL})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, globalerrors.ErrInvalidURL), errors.Is(err, globalerrors.ErrInvalidShortCode):
		response.JSONResponse(w, http.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, globalerrors.ErrNotFound):
		response.JSONResponse(w, http.StatusNotFound, err.Error(), nil)
	default:
		response.JSONResponse(w, http.StatusInternalServerError, "internal server error", nil)
	}
}

func validateShortCode(shortCode string) error {
	if len(shortCode) != shortCodeLength {
		return fmt.Errorf("%w: short code must be %d characters long", globalerrors.ErrInvalidShortCode, shortCodeLength)
	}
	for _, c := range shortCode {
		if !isAllowedChar(c) {
			return fmt.Errorf("%w: contains invalid characters", globalerrors.ErrInvalidShortCode)
		}
	}
	return nil
}

func buildShortURL(r *http.Request, shortCode string) string {
	baseURL := fmt.Sprintf("%s://%s", requestScheme(r), r.Host)
	if r.Host == "" {
		baseURL = requestScheme(r) + "://localhost"
	}
	return strings.TrimRight(baseURL, "/") + "/" + shortCode
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		return forwardedProto
	}
	return "http"
}

func isAllowedChar(ch rune) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	case ch == '_':
		return true
	default:
		return false
	}
}