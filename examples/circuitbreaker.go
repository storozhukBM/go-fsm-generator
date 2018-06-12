package examples

import (
	"time"
	"errors"
)

//go:generate ../go-fsm-generator -type CBMDeclaration -v

type FSMState int

type CBMDeclaration struct {
	Opened     FSMState `Try:"HalfOpened"`
	HalfOpened FSMState `Success:"Closed",Failure:"Opened",Panic:"Exit"`
	Closed     FSMState `Error:"Opened",Panic:"Exit"`
	Exit       FSMState
}

type CircuitBreaker struct {
	fsm *CBM

	protectedFunc func() error
	lastErr       error

	failureCount     uint
	failureThreshold uint

	openedAt       time.Time
	coolDownPeriod time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		fsm:              NewCBM(Closed),
		failureThreshold: 3,
		coolDownPeriod:   100 * time.Millisecond,
	}
}

func (m *CircuitBreaker) Run(protectedFunc func() error) error {
	m.protectedFunc = protectedFunc
	m.fsm.Operate(m)
	if m.fsm.Current() == HalfOpened {
		m.fsm.Operate(m) // Try after transition to half opened
	}
	return m.lastErr
}

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

func (m *CircuitBreaker) OperateOpened() CBMOpenedEvent {
	if time.Since(m.openedAt) > m.coolDownPeriod {
		return OpenedTry
	}
	m.lastErr = errors.New("circuit is open")
	return OpenedNoop
}
