package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Step represents a single step in a saga with an execute and compensate action.
type Step struct {
	Name       string
	Execute    func(ctx context.Context, payload json.RawMessage) error
	Compensate func(ctx context.Context, payload json.RawMessage) error
}

// SagaState mirrors the saga_state DB table.
type SagaState struct {
	ID          string          `json:"id"`
	SagaType    string          `json:"saga_type"`
	Status      string          `json:"status"` // started, completed, compensating, failed
	Payload     json.RawMessage `json:"payload"`
	CurrentStep int             `json:"current_step"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Orchestrator manages saga execution with step-by-step forward progress and compensation on failure.
type Orchestrator struct {
	db       *pgxpool.Pool
	sagaType string
	steps    []Step
}

func NewOrchestrator(db *pgxpool.Pool, sagaType string, steps []Step) *Orchestrator {
	return &Orchestrator{
		db:       db,
		sagaType: sagaType,
		steps:    steps,
	}
}

// Execute runs the saga forward. On any step failure, it compensates all previously completed steps in reverse.
func (o *Orchestrator) Execute(ctx context.Context, payload json.RawMessage) (string, error) {
	// Persist saga state
	sagaID, err := o.createSaga(ctx, payload)
	if err != nil {
		return "", fmt.Errorf("create saga: %w", err)
	}

	// Execute steps forward
	for i, step := range o.steps {
		log.Printf("[saga:%s] executing step %d: %s", sagaID, i, step.Name)

		if err := o.updateStep(ctx, sagaID, i, "started"); err != nil {
			return sagaID, err
		}

		if err := step.Execute(ctx, payload); err != nil {
			log.Printf("[saga:%s] step %d failed: %v — starting compensation", sagaID, i, err)
			o.compensate(ctx, sagaID, i-1, payload)
			return sagaID, fmt.Errorf("step %d (%s) failed: %w", i, step.Name, err)
		}
	}

	if err := o.completeSaga(ctx, sagaID); err != nil {
		return sagaID, err
	}

	log.Printf("[saga:%s] completed successfully", sagaID)
	return sagaID, nil
}

// compensate runs compensation for steps [fromStep..0] in reverse order.
func (o *Orchestrator) compensate(ctx context.Context, sagaID string, fromStep int, payload json.RawMessage) {
	_ = o.updateStatus(ctx, sagaID, "compensating")

	for i := fromStep; i >= 0; i-- {
		step := o.steps[i]
		if step.Compensate == nil {
			continue
		}

		log.Printf("[saga:%s] compensating step %d: %s", sagaID, i, step.Name)
		if err := step.Compensate(ctx, payload); err != nil {
			log.Printf("[saga:%s] compensation step %d failed: %v", sagaID, i, err)
			// Continue compensating remaining steps even if one fails
		}
	}

	_ = o.updateStatus(ctx, sagaID, "failed")
}

func (o *Orchestrator) createSaga(ctx context.Context, payload json.RawMessage) (string, error) {
	var id string
	err := o.db.QueryRow(ctx,
		`INSERT INTO saga_state (saga_type, status, payload, current_step)
		 VALUES ($1, 'started', $2, 0)
		 RETURNING id`,
		o.sagaType, payload,
	).Scan(&id)
	return id, err
}

func (o *Orchestrator) updateStep(ctx context.Context, sagaID string, step int, status string) error {
	_, err := o.db.Exec(ctx,
		`UPDATE saga_state SET current_step = $1, status = $2, updated_at = now() WHERE id = $3`,
		step, status, sagaID,
	)
	return err
}

func (o *Orchestrator) updateStatus(ctx context.Context, sagaID, status string) error {
	_, err := o.db.Exec(ctx,
		`UPDATE saga_state SET status = $1, updated_at = now() WHERE id = $2`,
		status, sagaID,
	)
	return err
}

func (o *Orchestrator) completeSaga(ctx context.Context, sagaID string) error {
	return o.updateStatus(ctx, sagaID, "completed")
}

// GetState retrieves the current saga state from the DB.
func GetState(ctx context.Context, db *pgxpool.Pool, sagaID string) (*SagaState, error) {
	var s SagaState
	err := db.QueryRow(ctx,
		`SELECT id, saga_type, status, payload, current_step, created_at, updated_at
		 FROM saga_state WHERE id = $1`, sagaID,
	).Scan(&s.ID, &s.SagaType, &s.Status, &s.Payload, &s.CurrentStep, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
