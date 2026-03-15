package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env overrides
	vars := []string{
		"SERVER_PORT", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER",
		"POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
		"POSTGRES_REPLICA_DSN", "REDIS_ADDR", "REDIS_PASSWORD", "KAFKA_BROKERS",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}

	cfg := Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort: got %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host: got %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != "5432" {
		t.Errorf("DB.Port: got %q, want %q", cfg.DB.Port, "5432")
	}
	if cfg.DB.User != "banking" {
		t.Errorf("DB.User: got %q, want %q", cfg.DB.User, "banking")
	}
	if cfg.DB.Password != "banking123" {
		t.Errorf("DB.Password: got %q, want %q", cfg.DB.Password, "banking123")
	}
	if cfg.DB.Name != "banking_db" {
		t.Errorf("DB.Name: got %q, want %q", cfg.DB.Name, "banking_db")
	}
	if cfg.DB.SSLMode != "disable" {
		t.Errorf("DB.SSLMode: got %q, want %q", cfg.DB.SSLMode, "disable")
	}
	if cfg.DB.ReplicaDSN != "" {
		t.Errorf("DB.ReplicaDSN: got %q, want empty", cfg.DB.ReplicaDSN)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr: got %q, want %q", cfg.Redis.Addr, "localhost:6379")
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "localhost:29092" {
		t.Errorf("Kafka.Brokers: got %v, want [localhost:29092]", cfg.Kafka.Brokers)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("POSTGRES_HOST", "db.prod")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_USER", "admin")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "prod_db")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("POSTGRES_REPLICA_DSN", "postgres://replica:5432/db")
	t.Setenv("REDIS_ADDR", "redis.prod:6380")
	t.Setenv("REDIS_PASSWORD", "rpass")
	t.Setenv("KAFKA_BROKERS", "kafka.prod:9092")

	cfg := Load()

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort: got %q, want 9090", cfg.ServerPort)
	}
	if cfg.DB.Host != "db.prod" {
		t.Errorf("DB.Host: got %q, want db.prod", cfg.DB.Host)
	}
	if cfg.DB.Port != "5433" {
		t.Errorf("DB.Port: got %q, want 5433", cfg.DB.Port)
	}
	if cfg.DB.User != "admin" {
		t.Errorf("DB.User: got %q, want admin", cfg.DB.User)
	}
	if cfg.DB.Password != "secret" {
		t.Errorf("DB.Password: got %q, want secret", cfg.DB.Password)
	}
	if cfg.DB.Name != "prod_db" {
		t.Errorf("DB.Name: got %q, want prod_db", cfg.DB.Name)
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("DB.SSLMode: got %q, want require", cfg.DB.SSLMode)
	}
	if cfg.DB.ReplicaDSN != "postgres://replica:5432/db" {
		t.Errorf("DB.ReplicaDSN: got %q", cfg.DB.ReplicaDSN)
	}
	if cfg.Redis.Addr != "redis.prod:6380" {
		t.Errorf("Redis.Addr: got %q", cfg.Redis.Addr)
	}
	if cfg.Kafka.Brokers[0] != "kafka.prod:9092" {
		t.Errorf("Kafka.Brokers: got %v", cfg.Kafka.Brokers)
	}
}

func TestDBConfig_DSN(t *testing.T) {
	d := DBConfig{
		User:     "user",
		Password: "pass",
		Host:     "localhost",
		Port:     "5432",
		Name:     "mydb",
		SSLMode:  "disable",
	}
	want := "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
	got := d.DSN()
	if got != want {
		t.Errorf("DSN(): got %q, want %q", got, want)
	}
}
