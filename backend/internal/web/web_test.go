package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterServesEmbeddedIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register(router)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Geopress") {
		t.Fatalf("expected index response to contain Geopress, got %q", recorder.Body.String())
	}
}

func TestRegisterFallsBackToIndexForSPARoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register(router)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Geopress") {
		t.Fatalf("expected fallback response to contain Geopress, got %q", recorder.Body.String())
	}
}

func TestRegisterReturnsAPI404ForUnknownAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register(router)

	req := httptest.NewRequest(http.MethodGet, "/api/not-found", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}
