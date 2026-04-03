// package link

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/url"
// 	"strings"
// 	"time"
// 	"url-shortener/pkg/response"
// )

// const timeContext = 2 * time.Second

// type usecaseI interface {
// 	CreateShortLink(ctx context.Context, originalURL string) (string, error)
// 	GetOriginalURLByShort(ctx context.Context, shortCode string) (string, error)
// }

// type Handler struct {
// 	usecase	usecaseI
// }

// func New(usecase usecaseI) *Handler {
// 	return &Handler{
// 		usecase: usecase,
// 	}
// }

// type createRequest struct {
// 	URL string `json:"url"`
// }

// func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
// 		response.JSONResponse(w, http.StatusOK, "server ready", nil)
// 	}

// func (h *Handler) CreateShortLink(w http.ResponseWriter, r *http.Request) {
// 	var req createRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		response.JSONResponse(w, http.StatusBadRequest, "invalid json body", struct{}{})
// 		return
// 	}

// 	err := validateURL(req.URL)
// 	if err != nil {
// 		response.JSONResponse(w, http.StatusBadRequest, "invalid url", map[string]interface{}{
// 			"bad_url": req.URL,
// 		})
// 		return
// 	}

// 	ctx, cansel := context.WithTimeout(context.Background(), timeContext)
// 	defer cansel()

// 	res, err := h.usecase.CreateShortLink(ctx, req.URL)
// 	// вставить обработчик ошибок(создать)
	
// 	response.JSONResponse(w, http.StatusCreated, "Short code created", map[string] interface{}{
// 		"ShortURL": res,
// 	})
// }

// func (h *Handler) GetOriginalURLByShort(w http.ResponseWriter, r *http.Request) {
// 	var req createRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		response.JSONResponse(w, http.StatusBadRequest, "invalid json body", nil)
// 	}

// 	for _, c := range req.URL {
// 		if !isAllowedChar(c) {
// 			response.JSONResponse(w, http.StatusBadRequest, "invalid short url", map[string]interface{}{
// 				"shortURL": req.URL, 
// 			})
// 		}
// 	}

// 	ctx, cansel := context.WithTimeout(context.Background(), timeContext)
// 	defer cansel()

// 	originalURL, err := h.usecase.GetOriginalURLByShort(ctx, req.URL)
// 	if err != nil {
// 		//обработать
// 	}

// 	response.JSONResponse(w, http.StatusOK, "find original url by short url", map[string]interface{}{
// 		"originalURL":	originalURL,
// 	})
// }

// func validateURL(raw string) error {
// 	raw = strings.TrimSpace(raw)
// 	if raw == "" {
// 		return fmt.Errorf("url is required")
// 	}
// 	parsed, err := url.ParseRequestURI(raw)
// 	if err != nil {
// 		return fmt.Errorf("invalid url: %w", err)
// 	}
// 	if parsed.Scheme != "http" && parsed.Scheme != "https" {
// 		return fmt.Errorf("url scheme must be http or https")
// 	}
// 	if parsed.Host == "" {
// 		return fmt.Errorf("url host is required")
// 	}
// 		return nil
// }

// func isAllowedChar(ch rune) bool {
// 	switch {
// 	case ch >= 'a' && ch <= 'z':
// 		return true
// 	case ch >= 'A' && ch <= 'Z':
// 		return true
// 	case ch >= '0' && ch <= '9':
// 		return true
// 	case ch == '_':
// 		return true
// 	default:
// 		return false
// 	}
// }

package link

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"url-shortener/pkg/globalerrors"
	"url-shortener/pkg/response"

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
}

func New(usecase usecaseI) *Handler {
	return &Handler{usecase: usecase}
}

type createRequest struct {
	URL string `json:"url"`
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

	if err := validateURL(req.URL); err != nil {
		h.handleError(w, err)
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

func validateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: url is required", globalerrors.ErrInvalidURL)
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("%w: malformed url", globalerrors.ErrInvalidURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https", globalerrors.ErrInvalidURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%w: host is required", globalerrors.ErrInvalidURL)
	}
	return nil
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