package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
)

type fakeLots struct {
	filter  model.ParkingLotFilter
	lots    []model.ParkingLot
	getTTL  time.Duration
	listErr error
}

func (f *fakeLots) ListParkingLots(_ context.Context, filter model.ParkingLotFilter) ([]model.ParkingLot, error) {
	f.filter = filter
	return f.lots, f.listErr
}

func (f *fakeLots) GetParkingLot(_ context.Context, id int64, ttl time.Duration) (*model.ParkingLot, error) {
	f.getTTL = ttl
	for i := range f.lots {
		if f.lots[i].ID == id {
			return &f.lots[i], nil
		}
	}
	return &model.ParkingLot{ID: id}, nil
}

func (f *fakeLots) CreateParkingLot(
	_ context.Context, lot model.ParkingLot,
) (*model.ParkingLot, error) {
	return &lot, nil
}

func (f *fakeLots) UpdateParkingLot(
	_ context.Context, lot model.ParkingLot,
) (*model.ParkingLot, error) {
	return &lot, nil
}

func (f *fakeLots) DeleteParkingLot(_ context.Context, _ int64) error { return nil }

type fakeSpots struct {
	ttl    time.Duration
	spots  []model.ParkingSpot
	called bool
}

func (f *fakeSpots) ListSpotsByLot(
	_ context.Context, _ int64, ttl time.Duration,
) ([]model.ParkingSpot, error) {
	f.called = true
	f.ttl = ttl
	return f.spots, nil
}

func (f *fakeSpots) GetSpot(_ context.Context, _ int64) (*model.ParkingSpot, error) {
	return &model.ParkingSpot{}, nil
}

func (f *fakeSpots) CreateSpot(
	_ context.Context, spot model.ParkingSpot,
) (*model.ParkingSpot, error) {
	return &spot, nil
}

func (f *fakeSpots) UpdateSpot(
	_ context.Context, spot model.ParkingSpot,
) (*model.ParkingSpot, error) {
	return &spot, nil
}

func (f *fakeSpots) DeleteSpot(_ context.Context, _ int64) error { return nil }

func mapRouter(lots *fakeLots, spots *fakeSpots) *gin.Engine {
	return NewRouter(
		&config.Config{
			AppEnv:       "test",
			CORSOrigins:  []string{"http://localhost:5173"},
			FreshnessTTL: 15 * time.Second,
		},
		Deps{
			DB:     fakePinger{},
			Lots:   lots,
			Spots:  spots,
			Tokens: auth.NewManager("test-secret", time.Hour),
		},
	)
}

func TestListParkingLotsBBox(t *testing.T) {
	lots := &fakeLots{}
	router := mapRouter(lots, &fakeSpots{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/parking-lots?bbox=37.5,55.7,37.6,55.8", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if lots.filter.BBox == nil {
		t.Fatal("bbox was not passed to the store")
	}
	want := model.BBox{MinLon: 37.5, MinLat: 55.7, MaxLon: 37.6, MaxLat: 55.8}
	if *lots.filter.BBox != want {
		t.Fatalf("unexpected bbox: %+v", *lots.filter.BBox)
	}
	if lots.filter.TTL != 15*time.Second {
		t.Fatalf("unexpected ttl: %s", lots.filter.TTL)
	}
}

func TestListParkingLotsInvalidBBox(t *testing.T) {
	router := mapRouter(&fakeLots{}, &fakeSpots{})

	for _, bbox := range []string{
		"37.5,55.7,37.6",
		"a,b,c,d",
		"10,20,5,30",
		"-200,0,10,10",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/parking-lots?bbox="+bbox, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bbox=%s, got %d", bbox, rec.Code)
		}
	}
}

func TestListParkingLotsIncludeSpots(t *testing.T) {
	lots := &fakeLots{lots: []model.ParkingLot{{ID: 3, Name: "demo"}}}
	spots := &fakeSpots{spots: []model.ParkingSpot{{ID: 10, Code: "A-01", Status: "free"}}}
	router := mapRouter(lots, spots)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/parking-lots?include=spots", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !spots.called {
		t.Fatal("spots were not loaded for include=spots")
	}
	if spots.ttl != 15*time.Second {
		t.Fatalf("unexpected spots ttl: %s", spots.ttl)
	}

	var body []struct {
		Spots []model.ParkingSpot `json:"spots"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || len(body[0].Spots) != 1 || body[0].Spots[0].Code != "A-01" {
		t.Fatalf("unexpected spots payload: %+v", body)
	}
}
