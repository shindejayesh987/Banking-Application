package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort string
	DB         DBConfig
	Redis      RedisConfig
	Kafka      KafkaConfig
}

type DBConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	Name       string
	SSLMode    string
	ReplicaDSN string // optional read replica DSN
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
}

func Load() Config {
	return Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DB: DBConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "banking"),
			Password: getEnv("POSTGRES_PASSWORD", "banking123"),
			Name:     getEnv("POSTGRES_DB", "banking_db"),
			SSLMode:    getEnv("POSTGRES_SSLMODE", "disable"),
			ReplicaDSN: getEnv("POSTGRES_REPLICA_DSN", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", "redis123"),
			DB:       0,
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:29092")},
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
