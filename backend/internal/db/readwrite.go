package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a primary (read-write) and optional replica (read-only) connection pool.
// Write operations always go to primary. Read operations go to the replica if available.
type Pool struct {
	primary *pgxpool.Pool
	replica *pgxpool.Pool
}

// NewPool creates a read/write splitting pool. If replicaDSN is empty, all queries go to primary.
func NewPool(ctx context.Context, primaryDSN, replicaDSN string) (*Pool, error) {
	primary, err := pgxpool.New(ctx, primaryDSN)
	if err != nil {
		return nil, err
	}

	pool := &Pool{primary: primary}

	if replicaDSN != "" {
		replica, err := pgxpool.New(ctx, replicaDSN)
		if err != nil {
			log.Printf("WARNING: replica not reachable, reads will use primary: %v", err)
		} else {
			if err := replica.Ping(ctx); err != nil {
				log.Printf("WARNING: replica ping failed, reads will use primary: %v", err)
				replica.Close()
			} else {
				pool.replica = replica
				log.Println("connected to read replica")
			}
		}
	}

	return pool, nil
}

// Primary returns the primary (write) pool.
func (p *Pool) Primary() *pgxpool.Pool {
	return p.primary
}

// Reader returns the replica pool if available, otherwise the primary.
func (p *Pool) Reader() *pgxpool.Pool {
	if p.replica != nil {
		return p.replica
	}
	return p.primary
}

// Query routes to the reader pool.
func (p *Pool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.Reader().Query(ctx, sql, args...)
}

// QueryRow routes to the reader pool.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.Reader().QueryRow(ctx, sql, args...)
}

// Exec routes to the primary pool.
func (p *Pool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return p.primary.Exec(ctx, sql, args...)
}

// Begin starts a transaction on the primary pool.
func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.primary.Begin(ctx)
}

// Ping checks both pools.
func (p *Pool) Ping(ctx context.Context) error {
	if err := p.primary.Ping(ctx); err != nil {
		return err
	}
	if p.replica != nil {
		return p.replica.Ping(ctx)
	}
	return nil
}

// Close closes both pools.
func (p *Pool) Close() {
	p.primary.Close()
	if p.replica != nil {
		p.replica.Close()
	}
}
