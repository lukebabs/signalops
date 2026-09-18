package main

import (
	"testing"
	"time"
)

func TestPolicyTargetsSubscriberUserActivity(t *testing.T) {
	cutoff := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	targets := policyTargets(policy{ID: "subscriber.user_activity_180d"}, "tenant-pilot-b", cutoff)
	if len(targets) != 1 {
		t.Fatalf("targets len = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.table != "subscriber_user_activity_events" {
		t.Fatalf("table = %q", target.table)
	}
	if target.timeColumn != "occurred_at" {
		t.Fatalf("time column = %q", target.timeColumn)
	}
	if target.where != "tenant_id=$1 AND occurred_at < $2" {
		t.Fatalf("where = %q", target.where)
	}
	if target.preserveReceipts {
		t.Fatal("subscriber activity retention should not use cyber evidence receipts")
	}
	if len(target.args) != 2 || target.args[0] != "tenant-pilot-b" || target.args[1] != cutoff {
		t.Fatalf("args = %#v", target.args)
	}
}
