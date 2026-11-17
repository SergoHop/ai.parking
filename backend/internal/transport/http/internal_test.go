package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
	"github.com/SergoHop/ai.parking/backend/internal/ws"
)

type heartbeatCall struct {
	cameraID    int64
	status      string
	lastFrameAt *time.Time
	fps         *int
	resolution  *string
}

type fakeInternal struct {
	camera    *model.Camera
	cameraErr error
	heartbeat *heartbeatCall
	events    []model.SpotEvent
	applyErr  error
}

func (f *fakeInternal) GetCamera(_ context.Context, id int64) (*model.Camera, error) {
	if f.cameraErr != nil {
		return nil, f.cameraErr
	}
	if f.camera == nil || f.camera.ID != id {
		return nil, repository.ErrNotFound
	}
	return f.camera, nil
}

func (f *fakeInternal) UpdateCameraHeartbeat(
	_ context.Context,
	id int64,
	status string,
	lastFrameAt *time.Time,
	fps *int,
	resolution *string,
) error {
	if f.cameraErr != nil {
		return f.cameraErr
	}
	if f.camera == nil || f.camera.ID != id {
		return repository.ErrNotFound
	}
	f.heartbeat = &heartbeatCall{id, status, lastFrameAt, fps, resolution}
	return nil
}

func (f *fakeInternal) ApplySpotEvents(
	_ context.Context,
	cameraID *int64,
	events []model.SpotEvent,
) ([]model.SpotChange, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	f.events = append(f.events, events...)

	lotID := int64(0)
	if f.camera != nil {
		lotID = f.camera.LotID
	}
	changes := make([]model.SpotChange, 0, len(events))
	for i, event := range events {
		changes = append(changes, model.SpotChange{
			SpotID:     event.SpotID,
			LotID:      lotID,
			Code:       "S-" + strconv.Itoa(i+1),
			Status:     event.Status,
			Confidence: event.Confidence,
			At:         event.CapturedAt,
		})
	}
	return changes, nil
}

func internalRouter(token string, store *fakeInternal) *gin.Engine {
	return internalRouterWithHub(token, store, nil)
}

func internalRouterWithHub(token string, store *fakeInternal, hub *ws.Hub) *gin.Engine {
	return NewRouter(
		&config.Config{
			AppEnv:        "test",
			CORSOrigins:   []string{"http://localhost:5173"},
			InternalToken: token,
		},
		Deps{
			DB:       fakePinger{},
			Internal: store,
			Tokens:   auth.NewManager("test-secret", time.Hour),
			Hub:      hub,
		},
	)
}

func TestInternalRequiresToken(t *testing.T) {
	router := internalRouter("secret", &fakeInternal{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "wrong")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", rec.Code)
	}
}

func TestInternalDisabledWithoutTokenConfig(t *testing.T) {
	router := internalRouter("", &fakeInternal{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "anything")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestCameraHeartbeat(t *testing.T) {
	store := &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3}}
	router := internalRouter("secret", store)

	body := `{"status":"online","last_frame_at":"2026-11-14T10:00:00Z","fps":5,"resolution":"1280x720"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cameras/7/heartbeat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.heartbeat == nil {
		t.Fatal("heartbeat was not stored")
	}
	if store.heartbeat.cameraID != 7 || store.heartbeat.status != "online" {
		t.Fatalf("unexpected heartbeat: %+v", store.heartbeat)
	}
	if store.heartbeat.fps == nil || *store.heartbeat.fps != 5 {
		t.Fatalf("unexpected fps: %+v", store.heartbeat.fps)
	}
}

func TestCameraHeartbeatValidation(t *testing.T) {
	router := internalRouter("secret", &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cameras/7/heartbeat",
		strings.NewReader(`{"status":"sleeping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCameraHeartbeatNotFound(t *testing.T) {
	store := &fakeInternal{cameraErr: repository.ErrNotFound}
	router := internalRouter("secret", store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cameras/99/heartbeat",
		strings.NewReader(`{"status":"offline"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSpotEventsAccepted(t *testing.T) {
	store := &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3}}
	router := internalRouter("secret", store)

	captured := time.Now().UTC().Add(-time.Second).Format(time.RFC3339)
	body, _ := json.Marshal(map[string]any{
		"camera_id": 7,
		"events": []map[string]any{
			{"spot_id": 1, "status": "occupied", "confidence": 0.93, "captured_at": captured},
			{"spot_id": 2, "status": "free", "captured_at": captured},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"accepted":2`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if len(store.events) != 2 || store.events[0].Status != "occupied" {
		t.Fatalf("unexpected events: %+v", store.events)
	}
	if store.events[0].CameraID == nil || *store.events[0].CameraID != 7 {
		t.Fatalf("camera id not propagated: %+v", store.events[0])
	}
}

func TestSpotEventsValidation(t *testing.T) {
	router := internalRouter("secret", &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3}})

	cases := []string{
		`{"camera_id":7,"events":[]}`,
		`{"camera_id":7,"events":[{"spot_id":1,"status":"busy","captured_at":"2026-11-14T10:00:00Z"}]}`,
		`{"camera_id":7,"events":[{"spot_id":0,"status":"free","captured_at":"2026-11-14T10:00:00Z"}]}`,
		`{"camera_id":7,"events":[{"spot_id":1,"status":"free","confidence":1.5,"captured_at":"2026-11-14T10:00:00Z"}]}`,
		`{"camera_id":7,"events":[{"spot_id":1,"status":"free"}]}`,
	}
	for _, body := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(internalTokenHeader, "secret")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", body, rec.Code)
		}
	}
}

func TestSpotEventsUnknownCamera(t *testing.T) {
	store := &fakeInternal{cameraErr: repository.ErrNotFound}
	router := internalRouter("secret", store)

	captured := time.Now().UTC().Format(time.RFC3339)
	body := `{"camera_id":99,"events":[{"spot_id":1,"status":"free","captured_at":"` + captured + `"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSpotEventsInvalidReference(t *testing.T) {
	store := &fakeInternal{
		camera:   &model.Camera{ID: 7, LotID: 3},
		applyErr: repository.ErrInvalidReference,
	}
	router := internalRouter("secret", store)

	captured := time.Now().UTC().Format(time.RFC3339)
	body := `{"camera_id":7,"events":[{"spot_id":9999,"status":"free","captured_at":"` + captured + `"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_reference") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestSpotEventsBroadcast(t *testing.T) {
	hub := ws.NewHub()
	subscriber := ws.NewClient(hub, 3)
	store := &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3}}
	router := internalRouterWithHub("secret", store, hub)

	captured := time.Now().UTC().Format(time.RFC3339)
	body := `{"camera_id":7,"events":[{"spot_id":21,"status":"occupied","confidence":0.9,"captured_at":"` + captured + `"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	select {
	case raw := <-subscriber.Send():
		var msg ws.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal message: %v", err)
		}
		if msg.Type != "spot_status" || msg.LotID != 3 || msg.SpotID != 21 ||
			msg.Status != "occupied" || msg.Code == "" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no live message received")
	}
}

func TestCameraHeartbeatBroadcast(t *testing.T) {
	hub := ws.NewHub()
	subscriber := ws.NewClient(hub, 3)
	store := &fakeInternal{camera: &model.Camera{ID: 7, LotID: 3, Status: "offline"}}
	router := internalRouterWithHub("secret", store, hub)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cameras/7/heartbeat",
		strings.NewReader(`{"status":"online","last_frame_at":"2026-11-17T10:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, "secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	select {
	case raw := <-subscriber.Send():
		var msg ws.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal message: %v", err)
		}
		if msg.Type != "camera_status" || msg.LotID != 3 ||
			msg.CameraID != 7 || msg.Status != "online" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no camera message received")
	}
}
