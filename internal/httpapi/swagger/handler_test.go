package swagger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoc(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	res := httptest.NewRecorder()

	Doc().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Doc() status = %d", res.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode swagger json: %v", err)
	}
	if payload["openapi"] != "3.0.3" {
		t.Fatalf("unexpected openapi version: %v", payload["openapi"])
	}
}

func TestUI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	res := httptest.NewRecorder()

	UI().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("UI() status = %d", res.Code)
	}
	if got := res.Body.String(); got == "" {
		t.Fatal("swagger UI body is empty")
	}
}