package examples

import "fmt"

//go:generate ../go-fsm-generator -type CBMDeclaration -v

type FSMState int

type CBMDeclaration struct {
	Opened     FSMState `Try:"HalfOpened"`
	HalfOpened FSMState `Success:"Closed",Failure:"Opened"`
	Closed     FSMState `Failure:"Opened",Panic:"Terminal",Error:"Terminal"`
	Terminal   FSMState
}

type CircuitBreaker struct {
	fsm *CBM
}

func (m *CircuitBreaker) OperateClosed() CBMClosedEvent {
	fmt.Printf(m.fsm.Current().String())
	return ClosedPanic
}

func (m *CircuitBreaker) OperateHalfOpened() CBMHalfOpenedEvent {
	return HalfOpenedSuccess
}

func (m *CircuitBreaker) OperateOpened() CBMOpenedEvent {
	return OpenedTry
}
