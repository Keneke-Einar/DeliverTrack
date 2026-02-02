package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name string

	// Configuration
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenMaxCalls int

	// State
	state            CircuitBreakerState
	failures         int
	lastFailureTime  time.Time
	halfOpenCalls    int
	halfOpenSuccesses int

	mutex sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMaxCalls: 3, // Allow 3 calls in half-open state
		state:            StateClosed,
	}
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit is open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) < cb.resetTimeout {
			return errors.New("circuit breaker is open")
		}
		// Transition to half-open
		cb.state = StateHalfOpen
		cb.halfOpenCalls = 0
		cb.halfOpenSuccesses = 0
	}

	// Check half-open call limit
	if cb.state == StateHalfOpen {
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			return errors.New("circuit breaker half-open call limit exceeded")
		}
		cb.halfOpenCalls++
	}

	// Execute the function
	err := fn()

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// onFailure handles failure scenarios
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateHalfOpen:
		// If we fail in half-open state, go back to open
		cb.state = StateOpen
	default:
		// If we reach failure threshold, open the circuit
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
		}
	}
}

// onSuccess handles success scenarios
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateHalfOpen:
		cb.halfOpenSuccesses++
		// If we have enough successes in half-open state, close the circuit
		if cb.halfOpenSuccesses >= cb.halfOpenMaxCalls {
			cb.state = StateClosed
			cb.failures = 0
		}
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == StateOpen
}

// IsClosed returns true if the circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == StateClosed
}

// IsHalfOpen returns true if the circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == StateHalfOpen
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// Retry executes a function with exponential backoff retry
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			lastErr = err

			// If this is the last attempt, don't wait
			if attempt == config.MaxAttempts {
				break
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Calculate next delay
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		} else {
			// Success
			return nil
		}
	}

	return lastErr
}