package batchflow

import (
	"errors"
	"testing"
)

func TestBuildErrorFingerprintNormalizesDynamicValues(t *testing.T) {
	errA := errors.New("ReceiveMessage failed with request 12345 token ABCDEF123456")
	errB := errors.New("ReceiveMessage failed with request 98765 token FEDCBA654321")

	gotA := buildErrorFingerprint("sqs_consumer", errA)
	gotB := buildErrorFingerprint("sqs_consumer", errB)

	if gotA != gotB {
		t.Fatalf("fingerprints should match after normalization:\nA=%s\nB=%s", gotA, gotB)
	}
}
