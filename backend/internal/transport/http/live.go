package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/ws"
)

// live upgrades a browser to a WebSocket subscriber. Query parameter lot_id
// narrows the stream to one parking lot; without it the client receives all.
func (s *Server) live(c *gin.Context) {
	lotID := int64(0)
	if raw := strings.TrimSpace(c.Query("lot_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid lot_id")
			return
		}
		lotID = id
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin:      s.allowOrigin,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already replied with an HTTP error.
		return
	}

	client := ws.NewClient(s.hub, lotID)
	client.Serve(conn)
}

// allowOrigin restricts WebSocket upgrades to the configured frontend origins.
func (s *Server) allowOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	for _, allowed := range s.cfg.CORSOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// markCamerasFresh flags cameras whose last frame is still inside the TTL.
func (s *Server) markCamerasFresh(cameras []model.Camera) {
	now := time.Now()
	ttl := s.cfg.StatusTTL()
	for i := range cameras {
		cameras[i].Fresh = cameras[i].Status == "online" &&
			cameras[i].LastFrameAt != nil &&
			now.Sub(*cameras[i].LastFrameAt) <= ttl
	}
}
