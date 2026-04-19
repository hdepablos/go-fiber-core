package queue

import (
	"errors"
	"testing"
)

func TestPollingErrorGuardTriggersOnRepeatedSameError(t *testing.T) {
	guard := NewPollingErrorGuard(3)

	event := guard.Record(errors.New("queue missing 400"))
	if event.Triggered {
		t.Fatalf("first error should not trigger")
	}

	event = guard.Record(errors.New("queue missing 401"))
	if event.Triggered {
		t.Fatalf("second normalized error should not trigger")
	}

	event = guard.Record(errors.New("queue missing 402"))
	if !event.Triggered {
		t.Fatalf("third normalized error should trigger")
	}
}

func TestPollingErrorGuardResetsOnDifferentError(t *testing.T) {
	guard := NewPollingErrorGuard(2)

	guard.Record(errors.New("queue missing 400"))
	event := guard.Record(errors.New("other dependency failure"))
	if event.Count != 1 {
		t.Fatalf("different error should reset counter, got %d", event.Count)
	}
}
