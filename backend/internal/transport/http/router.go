package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/ws"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type ParkingLotStore interface {
	ListParkingLots(ctx context.Context, filter model.ParkingLotFilter) ([]model.ParkingLot, error)
	GetParkingLot(ctx context.Context, id int64, ttl time.Duration) (*model.ParkingLot, error)
	CreateParkingLot(ctx context.Context, lot model.ParkingLot) (*model.ParkingLot, error)
	UpdateParkingLot(ctx context.Context, lot model.ParkingLot) (*model.ParkingLot, error)
	DeleteParkingLot(ctx context.Context, id int64) error
}

type SpotStore interface {
	ListSpotsByLot(ctx context.Context, lotID int64, ttl time.Duration) ([]model.ParkingSpot, error)
	GetSpot(ctx context.Context, id int64) (*model.ParkingSpot, error)
	CreateSpot(ctx context.Context, spot model.ParkingSpot) (*model.ParkingSpot, error)
	UpdateSpot(ctx context.Context, spot model.ParkingSpot) (*model.ParkingSpot, error)
	DeleteSpot(ctx context.Context, id int64) error
}

type CameraStore interface {
	ListCamerasByLot(ctx context.Context, lotID int64) ([]model.Camera, error)
	GetCamera(ctx context.Context, id int64) (*model.Camera, error)
	CreateCamera(ctx context.Context, camera model.Camera) (*model.Camera, error)
	UpdateCamera(ctx context.Context, camera model.Camera) (*model.Camera, error)
	DeleteCamera(ctx context.Context, id int64) error
}

type UserStore interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
}

// InternalStore is used by the CV worker endpoints (trusted, token-protected).
type InternalStore interface {
	GetCamera(ctx context.Context, id int64) (*model.Camera, error)
	UpdateCameraHeartbeat(
		ctx context.Context,
		id int64,
		status string,
		lastFrameAt *time.Time,
		fps *int,
		resolution *string,
	) error
	ApplySpotEvents(ctx context.Context, cameraID *int64, events []model.SpotEvent) ([]model.SpotChange, error)
}

type Deps struct {
	DB       DBPinger
	Lots     ParkingLotStore
	Spots    SpotStore
	Cameras  CameraStore
	Users    UserStore
	Internal InternalStore
	Tokens   *auth.Manager
	Hub      *ws.Hub
}

type Server struct {
	cfg      *config.Config
	db       DBPinger
	lots     ParkingLotStore
	spots    SpotStore
	cameras  CameraStore
	users    UserStore
	internal InternalStore
	tokens   *auth.Manager
	hub      *ws.Hub
}

func NewRouter(cfg *config.Config, deps Deps) *gin.Engine {
	switch cfg.AppEnv {
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	hub := deps.Hub
	if hub == nil {
		hub = ws.NewHub()
	}

	srv := &Server{
		cfg:      cfg,
		db:       deps.DB,
		lots:     deps.Lots,
		spots:    deps.Spots,
		cameras:  deps.Cameras,
		users:    deps.Users,
		internal: deps.Internal,
		tokens:   deps.Tokens,
		hub:      hub,
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Internal-Token"},
		MaxAge:       12 * time.Hour,
	}))
	_ = router.SetTrustedProxies(nil)

	api := router.Group("/api")
	api.GET("/health", srv.health)
	api.POST("/auth/login", srv.login)

	// Публичное чтение: водитель не авторизуется.
	api.GET("/parking-lots", srv.listParkingLots)
	api.GET("/parking-lots/:id", srv.getParkingLot)
	api.GET("/parking-lots/:id/spots", srv.listSpots)
	api.GET("/parking-lots/:id/cameras", srv.listCameras)

	// Запись только для админа.
	admin := api.Group("")
	admin.Use(srv.requireAuth)
	admin.POST("/parking-lots", srv.createParkingLot)
	admin.PUT("/parking-lots/:id", srv.updateParkingLot)
	admin.DELETE("/parking-lots/:id", srv.deleteParkingLot)

	admin.POST("/parking-lots/:id/spots", srv.createSpot)
	admin.PUT("/spots/:id", srv.updateSpot)
	admin.DELETE("/spots/:id", srv.deleteSpot)

	admin.POST("/cameras", srv.createCamera)
	admin.PUT("/cameras/:id", srv.updateCamera)
	admin.DELETE("/cameras/:id", srv.deleteCamera)

	// Внутренние эндпоинты CV-сервиса: общий секрет X-Internal-Token.
	internal := api.Group("/internal")
	internal.Use(srv.requireInternalToken)
	internal.POST("/cameras/:id/heartbeat", srv.cameraHeartbeat)
	internal.POST("/events", srv.spotEvents)

	// Live-статусы для карты: /ws/live?lot_id=N (без параметра — все лоты).
	router.GET("/ws/live", srv.live)

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
