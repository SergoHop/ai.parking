package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type ParkingLotLister interface {
	ListParkingLots(ctx context.Context) ([]model.ParkingLot, error)
}

type Server struct {
	cfg  *config.Config
	db   DBPinger
	lots ParkingLotLister
}

func NewRouter(cfg *config.Config, db DBPinger, lots ParkingLotLister) *gin.Engine {
	switch cfg.AppEnv {
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &Server{cfg: cfg, db: db, lots: lots}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Internal-Token"},
		MaxAge:       12 * time.Hour,
	}))
	_ = router.SetTrustedProxies(nil)

	router.GET("/api/health", srv.health)
	router.GET("/api/parking-lots", srv.listParkingLots)

	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "not_found", "route not found")
	})

	return router
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
