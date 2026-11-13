package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

const claimsContextKey = "auth_claims"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "username and password are required")
		return
	}

	user, err := s.users.GetByUsername(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
			return
		}
		writeStoreError(c, err)
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}

	token, expiresAt, err := s.tokens.Generate(*user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to issue token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (s *Server) requireAuth(c *gin.Context) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeError(c, http.StatusUnauthorized, "unauthorized", "bearer token is required")
		c.Abort()
		return
	}

	claims, err := s.tokens.Parse(strings.TrimSpace(parts[1]))
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
		c.Abort()
		return
	}

	c.Set(claimsContextKey, claims)
	c.Next()
}
