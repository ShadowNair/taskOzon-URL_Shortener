package link

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"url-shortener/pkg/globalerrors"

	"github.com/gorilla/mux"
)

type usecaseStub struct {
	createFn func(ctx context.Context, originalURL string) (string, error)
	getFn    func(ctx context.Context, shortCode string) (string, error)
}

func (u *usecaseStub) CreateShortLink(ctx context.Context, originalURL string) (string, error) {
	return u.createFn(ctx, originalURL)
}

func (u *usecaseStub) GetOriginalURLByShort(ctx context.Context, shortCode string) (string, error) {
	return u.getFn(ctx, shortCode)
}

func TestHealth(t *testing.T) {
	h := New(&usecaseStub{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.Health(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Health() status = %d", res.Code)
	}
}

func TestCreateShortLinkSuccess(t *testing.T) {
	h := New(&usecaseStub{
		createFn: func(_ context.Context, originalURL string) (string, error) {
			if originalURL != "https://example.com" {
				t.Fatalf("unexpected url: %s", originalURL)
			}
			return "abcDEF123_", nil
		},
	})

	body := bytes.NewBufferString(`{"url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	req.Host = "localhost:8080"
	res := httptest.NewRecorder()

	h.CreateShortLink(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("CreateShortLink() status = %d", res.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := payload["data"].(map[string]any)
	if data["short_code"] != "abcDEF123_" {
		t.Fatalf("short_code = %v", data["short_code"])
	}
	if data["short_url"] != "http://localhost:8080/abcDEF123_" {
		t.Fatalf("short_url = %v", data["short_url"])
	}
}

func TestCreateShortLinkInvalidJSON(t *testing.T) {
	h := New(&usecaseStub{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewBufferString(`{"url":`))
	res := httptest.NewRecorder()

	h.CreateShortLink(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("CreateShortLink() status = %d", res.Code)
	}
}

func TestCreateShortLinkInvalidURL(t *testing.T) {
	h := New(&usecaseStub{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewBufferString(`{"url":"ftp://example.com"}`))
	res := httptest.NewRecorder()

	h.CreateShortLink(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("CreateShortLink() status = %d", res.Code)
	}
}

func TestCreateShortLinkInternalError(t *testing.T) {
	h := New(&usecaseStub{
		createFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("boom")
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewBufferString(`{"url":"https://example.com"}`))
	res := httptest.NewRecorder()

	h.CreateShortLink(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("CreateShortLink() status = %d", res.Code)
	}
}

func TestGetOriginalURLByShortSuccess(t *testing.T) {
	h := New(&usecaseStub{
		getFn: func(_ context.Context, shortCode string) (string, error) {
			if shortCode != "abcDEF123_" {
				t.Fatalf("unexpected shortCode: %s", shortCode)
			}
			return "https://example.com", nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/abcDEF123_", nil)
	req = mux.SetURLVars(req, map[string]string{"shortCode": "abcDEF123_"})
	res := httptest.NewRecorder()

	h.GetOriginalURLByShort(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("GetOriginalURLByShort() status = %d", res.Code)
	}
}

func TestGetOriginalURLByShortInvalidCode(t *testing.T) {
	h := New(&usecaseStub{})
	req := httptest.NewRequest(http.MethodGet, "/short", nil)
	req = mux.SetURLVars(req, map[string]string{"shortCode": "short"})
	res := httptest.NewRecorder()

	h.GetOriginalURLByShort(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("GetOriginalURLByShort() status = %d", res.Code)
	}
}

func TestGetOriginalURLByShortNotFound(t *testing.T) {
	h := New(&usecaseStub{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", globalerrors.ErrNotFound
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/abcDEF123_", nil)
	req = mux.SetURLVars(req, map[string]string{"shortCode": "abcDEF123_"})
	res := httptest.NewRecorder()

	h.GetOriginalURLByShort(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("GetOriginalURLByShort() status = %d", res.Code)
	}
}