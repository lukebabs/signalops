package storage

import "testing"

func TestRunStatusConstants(t *testing.T) {
	statuses := []string{RunStatusStarted, RunStatusSucceeded, RunStatusFailed, RunStatusCanceled}
	for _, status := range statuses {
		if status == "" {
			t.Fatal("empty run status")
		}
	}
}

func TestIdempotencyStatusConstants(t *testing.T) {
	statuses := []string{
		IdempotencyStatusAccepted,
		IdempotencyStatusPublished,
		IdempotencyStatusProcessed,
		IdempotencyStatusFailed,
		IdempotencyStatusDuplicate,
	}
	for _, status := range statuses {
		if status == "" {
			t.Fatal("empty idempotency status")
		}
	}
}
