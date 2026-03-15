package middleware

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func succeed() error { return nil }
func fail() error    { return errTest }

func TestNewCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 10*time.Second)
	if cb.State() != StateClosed {
		t.Errorf("initial state: got %s, want closed", cb.State())
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestExecute_SuccessStaysClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 10*time.Second)
	for i := 0; i < 10; i++ {
		if err := cb.Execute(succeed); err != nil {
			t.Errorf("Execute should succeed, got: %v", err)
		}
	}
	if cb.State() != StateClosed {
		t.Errorf("state after successes: got %s, want closed", cb.State())
	}
}

func TestExecute_FailuresOpenCircuit(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 10*time.Second)

	// 2 failures — still closed
	cb.Execute(fail)
	cb.Execute(fail)
	if cb.State() != StateClosed {
		t.Errorf("state after 2 failures: got %s, want closed", cb.State())
	}

	// 3rd failure — opens
	cb.Execute(fail)
	if cb.State() != StateOpen {
		t.Errorf("state after 3 failures: got %s, want open", cb.State())
	}
}

func TestExecute_OpenRejectsImmediately(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Second)
	cb.Execute(fail) // opens

	err := cb.Execute(succeed)
	if err == nil {
		t.Fatal("expected error from open circuit, got nil")
	}
	if err.Error() != "circuit breaker is open" {
		t.Errorf("error message: got %q, want %q", err.Error(), "circuit breaker is open")
	}
}

func TestExecute_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond)
	cb.Execute(fail) // opens

	// Wait for timeout to expire
	time.Sleep(20 * time.Millisecond)

	// Next call should transition to half-open and execute
	err := cb.Execute(succeed)
	if err != nil {
		t.Errorf("expected success in half-open, got: %v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("state after timeout+success: got %s, want half-open", cb.State())
	}
}

func TestExecute_HalfOpenSuccessCloses(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond)
	cb.Execute(fail) // opens
	time.Sleep(20 * time.Millisecond)

	cb.Execute(succeed) // → half-open, 1 success
	if cb.State() != StateHalfOpen {
		t.Errorf("after 1st success: got %s, want half-open", cb.State())
	}

	cb.Execute(succeed) // 2nd success → closes
	if cb.State() != StateClosed {
		t.Errorf("after 2nd success: got %s, want closed", cb.State())
	}
}

func TestExecute_HalfOpenFailureReOpens(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond)
	cb.Execute(fail) // opens
	time.Sleep(20 * time.Millisecond)

	cb.Execute(fail) // enters half-open then fails → re-opens
	if cb.State() != StateOpen {
		t.Errorf("after half-open failure: got %s, want open", cb.State())
	}
}

func TestExecute_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(100, 2, 10*time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Execute(succeed)
		}()
	}
	wg.Wait()
	// Should not panic or deadlock; state should still be closed
	if cb.State() != StateClosed {
		t.Errorf("concurrent state: got %s, want closed", cb.State())
	}
}

func TestExecute_ErrorPropagated(t *testing.T) {
	cb := NewCircuitBreaker(5, 2, 10*time.Second)
	customErr := errors.New("custom")
	err := cb.Execute(func() error { return customErr })
	if !errors.Is(err, customErr) {
		t.Errorf("error not propagated: got %v, want %v", err, customErr)
	}
}
