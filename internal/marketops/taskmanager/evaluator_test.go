package taskmanager

import (
	"testing"
	"time"
)

func TestEvaluateBlocksMissingDependency(t *testing.T) {
	d := Evaluate(time.Now(), []Input{{JobID: "marketops-sri-refresh", Status: "failed"}})
	for _, x := range d {
		if x.JobID == "marketops-sri-refresh" && x.State != "blocked" {
			t.Fatal(x)
		}
	}
}
func TestEvaluateRetryableFailure(t *testing.T) {
	d := Evaluate(time.Now(), []Input{{JobID: "marketops-intraday", Status: "failed", Reason: "provider timeout"}})
	for _, x := range d {
		if x.JobID == "marketops-intraday" && (!x.RetryEligible || x.State != "retryable") {
			t.Fatal(x)
		}
	}
}
