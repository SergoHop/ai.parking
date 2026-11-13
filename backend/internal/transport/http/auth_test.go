package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

type fakeUsers struct {
	user *model.User
	err  error
}

func (f fakeUsers) GetByUsername(context.Context, string) (*model.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil {
		return nil, repository.ErrNotFound
	}
	return f.user, nil
}

func TestLoginSuccess(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}

	manager := auth.NewManager("test-secret", time.Hour)
	router := NewRouter(
		&config.Config{AppEnv: "test", CORSOrigins: []string{"http://localhost:5173"}},
		Deps{
			DB:     fakePinger{},
			Users:  fakeUsers{user: &model.User{ID: 1, Username: "admin", PasswordHash: hash, Role: "admin"}},
			Tokens: manager,
		},
	)

	body := `{"username":"admin","password":"secret"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Token == "" {
		t.Fatal("expected token in response")
	}
	if _, err := manager.Parse(resp.Token); err != nil {
		t.Fatalf("token does not parse: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}

	router := NewRouter(
		&config.Config{AppEnv: "test", CORSOrigins: []string{"http://localhost:5173"}},
		Deps{
			DB:     fakePinger{},
			Users:  fakeUsers{user: &model.User{ID: 1, Username: "admin", PasswordHash: hash, Role: "admin"}},
			Tokens: auth.NewManager("test-secret", time.Hour),
		},
	)

	body := `{"username":"admin","password":"wrong"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireAuth(t *testing.T) {
	manager := auth.NewManager("test-secret", time.Hour)
	srv := &Server{tokens: manager}

	router := gin.New()
	router.GET("/protected", srv.requireAuth, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	token, _, err := manager.Generate(model.User{ID: 1, Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rec.Code)
	}
}
