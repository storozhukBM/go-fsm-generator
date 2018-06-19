package examples

import (
	"errors"
	"time"
)

//go:generate ../go-fsm-generator -type CBMDeclaration -v

// FSMState placeholder type
type FSMState int

// CBMDeclaration of the circuit breaker state machine
type CBMDeclaration struct {
	Opened     FSMState `Try:"HalfOpened"`
	HalfOpened FSMState `Success:"Closed",Failure:"Opened",Panic:"Exit"`
	Closed     FSMState `Error:"Opened",Panic:"Exit"`
	Exit       FSMState
}

// CircuitBreaker type with state machine inside
type CircuitBreaker struct {
	fsm *CBM

	protectedFunc func() error
	lastErr       error

	failureCount     uint
	failureThreshold uint

	openedAt       time.Time
	coolDownPeriod time.Duration
}

// NewCircuitBreaker constructor
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		fsm:              NewCBM(Closed),
		failureThreshold: 3,
		coolDownPeriod:   100 * time.Millisecond,
	}
}

// Run executes protected func under circuit breaker
func (m *CircuitBreaker) Run(protectedFunc func() error) error {
	m.protectedFunc = protectedFunc
	m.fsm.Operate(m)
	if m.fsm.Current() == HalfOpened {
		m.fsm.Operate(m) // Try after transition to half opened
	}
	return m.lastErr
}

// OperateClosed state behaviour
func (m *CircuitBreaker) OperateClosed() (event CBMClosedEvent) {
	defer func() {
		if r := recover(); r != nil {
			m.lastErr = errors.New("panic happened")
			event = ClosedPanic
		}
	}()
	m.lastErr = m.protectedFunc()
	if m.lastErr != nil {
		m.failureCount++
		if m.failureCount >= m.failureThreshold {
			m.openedAt = time.Now()
			return ClosedError
		}
	}
	return ClosedNoop
}

// OperateHalfOpened state behaviour
func (m *CircuitBreaker) OperateHalfOpened() (event CBMHalfOpenedEvent) {
	defer func() {
		if r := recover(); r != nil {
			m.lastErr = errors.New("panic happened")
			event = HalfOpenedPanic
		}
	}()

	m.lastErr = m.protectedFunc()
	if m.lastErr != nil {
		m.openedAt = time.Now()
		return HalfOpenedFailure
	}

	m.failureCount = 0
	return HalfOpenedSuccess
}

// OperateOpened state behaviour
func (m *CircuitBreaker) OperateOpened() CBMOpenedEvent {
	if time.Since(m.openedAt) > m.coolDownPeriod {
		return OpenedTry
	}
	m.lastErr = errors.New("circuit is open")
	return OpenedNoop
}
