package testdata

// SomeState placeholder type
type FSMState int

// SomeDeclaration of the circuit breaker state machine
type SomeDeclaration struct {
	First  FSMState `Aa:"Second"`
	Second FSMState `Bb:"Third",Cc:"First",Zz:"Fourth"`
	Third  FSMState `Dd:"First",Zz:"Fourth"`
	Fourth FSMState
}
