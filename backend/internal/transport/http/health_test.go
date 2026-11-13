package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func testRouter(pinger DBPinger) *gin.Engine {
	return NewRouter(
		&config.Config{AppEnv: "test", CORSOrigins: []string{"http://localhost:5173"}},
		Deps{
			DB:     pinger,
			Tokens: auth.NewManager("test-secret", time.Hour),
		},
	)
}

func TestHealthOK(t *testing.T) {
	router := testRouter(fakePinger{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHealthDBDown(t *testing.T) {
	router := testRouter(fakePinger{err: context.DeadlineExceeded})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"db":"down"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
