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
