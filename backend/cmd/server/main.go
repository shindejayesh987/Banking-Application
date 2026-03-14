package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/jayeshinusshinde/banking-backend/internal/config"
	"github.com/jayeshinusshinde/banking-backend/internal/consumer"
	"github.com/jayeshinusshinde/banking-backend/internal/db"
	"github.com/jayeshinusshinde/banking-backend/internal/handler"
	intkafka "github.com/jayeshinusshinde/banking-backend/internal/kafka"
	"github.com/jayeshinusshinde/banking-backend/internal/middleware"
	"github.com/jayeshinusshinde/banking-backend/internal/repository"
	"github.com/jayeshinusshinde/banking-backend/internal/saga"
	"github.com/jayeshinusshinde/banking-backend/internal/service"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- PostgreSQL (read/write splitting) ---
	rwPool, err := db.NewPool(ctx, cfg.DB.DSN(), cfg.DB.ReplicaDSN)
	if err != nil {
		log.Fatalf("unable to create DB pool: %v", err)
	}
	defer rwPool.Close()

	if err := rwPool.Ping(ctx); err != nil {
		log.Printf("WARNING: postgres not reachable: %v", err)
	} else {
		log.Println("connected to PostgreSQL")
	}

	// --- Run migrations (always on primary) ---
	runMigrations(rwPool.Primary())

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: redis not reachable: %v", err)
	} else {
		log.Println("connected to Redis")
	}

	// --- Kafka Writer ---
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	defer kafkaWriter.Close()
	log.Println("kafka writer configured")

	// --- Dependencies ---
	producer := intkafka.NewProducer(kafkaWriter)
	primary := rwPool.Primary()

	accountRepo := repository.NewAccountRepo(primary)
	txnRepo := repository.NewTransactionRepo()
	userRepo := repository.NewUserRepo(primary)

	accountSvc := service.NewAccountService(accountRepo, producer)
	txnSvc := service.NewTransactionService(primary, accountRepo, txnRepo, producer)

	// --- Saga ---
	transferSaga := saga.NewTransferSaga(primary, accountRepo)

	// --- Handlers ---
	healthH := handler.NewHealthHandler(primary, rdb, kafkaWriter)
	accountH := handler.NewAccountHandler(accountSvc)
	txnH := handler.NewTransactionHandler(txnSvc)
	userH := handler.NewUserHandler(userRepo)
	sagaH := handler.NewSagaTransferHandler(transferSaga)

	// --- Kafka Consumers ---
	txnConsumer := consumer.NewRunner(cfg.Kafka.Brokers, "transactions", "banking-txn-logger", consumer.TransactionLogger())
	acctConsumer := consumer.NewRunner(cfg.Kafka.Brokers, "account-events", "banking-acct-handler", consumer.AccountEventHandler())
	defer txnConsumer.Close()
	defer acctConsumer.Close()

	go txnConsumer.Run(ctx)
	go acctConsumer.Run(ctx)

	// --- Rate Limiter: 100 requests per minute ---
	rateLimiter := middleware.NewRateLimiter(rdb, 100, time.Minute)

	// --- HTTP Router ---
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", healthH.Health)

	// Users
	mux.HandleFunc("POST /api/v1/users", userH.Create)
	mux.HandleFunc("GET /api/v1/users/{id}", userH.GetByID)

	// Accounts
	mux.HandleFunc("POST /api/v1/accounts", accountH.Create)
	mux.HandleFunc("GET /api/v1/accounts/{id}", accountH.GetByID)
	mux.HandleFunc("GET /api/v1/accounts", accountH.ListByUser)

	// Transactions
	mux.HandleFunc("POST /api/v1/transactions/deposit", txnH.Deposit)
	mux.HandleFunc("POST /api/v1/transactions/withdraw", txnH.Withdraw)
	mux.HandleFunc("POST /api/v1/transactions/transfer", txnH.Transfer)

	// Saga-based transfer
	mux.HandleFunc("POST /api/v1/transactions/saga-transfer", sagaH.Transfer)

	// Wrap with rate limiter
	rateLimited := rateLimiter.Middleware(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      rateLimited,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Graceful Shutdown ---
	go func() {
		log.Printf("server listening on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	cancel() // stop consumers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped gracefully")
}

// runMigrations reads SQL files from the migrations directory and executes them.
func runMigrations(db *pgxpool.Pool) {
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		log.Printf("WARNING: could not create migrations table: %v", err)
		return
	}

	var count int
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count)
	if err != nil {
		log.Printf("WARNING: could not check migration status: %v", err)
		return
	}
	if count > 0 {
		log.Println("migrations already applied")
		return
	}

	migrationSQL, err := os.ReadFile("migrations/001_initial_schema.up.sql")
	if err != nil {
		log.Printf("WARNING: could not read migration file: %v", err)
		return
	}

	_, err = db.Exec(ctx, string(migrationSQL))
	if err != nil {
		log.Printf("WARNING: migration failed: %v", err)
		return
	}

	_, err = db.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES (1)`)
	if err != nil {
		log.Printf("WARNING: could not record migration: %v", err)
		return
	}

	log.Println("migrations applied successfully")
}
