package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	rdb   *redis.Client
	kafka *kafka.Writer
}

func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client, kw *kafka.Writer) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb, kafka: kw}
}

type healthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp := healthResponse{
		Status:   "ok",
		Services: make(map[string]string),
	}

	// Check PostgreSQL
	if err := h.db.Ping(ctx); err != nil {
		resp.Services["postgres"] = "down: " + err.Error()
		resp.Status = "degraded"
	} else {
		resp.Services["postgres"] = "up"
	}

	// Check Redis
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		resp.Services["redis"] = "down: " + err.Error()
		resp.Status = "degraded"
	} else {
		resp.Services["redis"] = "up"
	}

	// Kafka connectivity is checked passively (writer doesn't have a ping)
	resp.Services["kafka"] = "configured"

	w.Header().Set("Content-Type", "application/json")
	if resp.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}
