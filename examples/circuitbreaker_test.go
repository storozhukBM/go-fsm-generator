package examples

import "testing"

func TestCircuitBreaker(t *testing.T) {
	cbm, err := NewCBMFromString("Opened")
	if err != nil {
		t.Fatal("state parsing error should be nil", err)
	}
	circuitBreaker := &CircuitBreaker{
		fsm: cbm,
	}
	cbm.Operate(circuitBreaker)
	if cbm.state != Terminal {
		t.Fatalf("unexpected state: %s", cbm.state)
	}
}
