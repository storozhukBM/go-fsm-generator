package examples

import (
	"testing"
	"errors"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()
	err := cb.Run(func() error { return nil })
	if err != nil {
		t.Fatal("error should be nil", err)
	}

	targetErr := errors.New("target")

	err = cb.Run(func() error { return targetErr })
	if err != targetErr {
		t.Fatal("expected target error not occured", err)
	}
	err = cb.Run(func() error { return targetErr })
	if err != targetErr {
		t.Fatal("expected target error not occured", err)
	}
	err = cb.Run(func() error { return targetErr })
	if err != targetErr {
		t.Fatal("expected target error not occured", err)
	}

	err = cb.Run(func() error { return nil })
	if err == nil || err.Error() != "circuit is open" {
		t.Fatal("circuit isn't open", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = cb.Run(func() error { return targetErr })
	if err != targetErr {
		t.Fatal("expected target error not occured", err)
	}

	err = cb.Run(func() error { return nil })
	if err == nil || err.Error() != "circuit is open" {
		t.Fatal("circuit isn't open", err)
	}

	time.Sleep(100 * time.Millisecond)
	err = cb.Run(func() error { return nil })
	if err != nil {
		t.Fatal("error should be nil", err)
	}
}
