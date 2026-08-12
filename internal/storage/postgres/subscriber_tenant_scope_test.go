package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestValidSubscriberTenantID(t *testing.T) {
	for _, testCase := range []struct {
		tenantID string
		want     bool
	}{
		{tenantID: "tenant-local", want: true},
		{tenantID: "tenant_123", want: true},
		{tenantID: " tenant-a ", want: true},
		{tenantID: "", want: false},
		{tenantID: "tenant/a", want: false},
		{tenantID: "tenant a", want: false},
		{tenantID: "-tenant", want: false},
	} {
		if got := validSubscriberTenantID(testCase.tenantID); got != testCase.want {
			t.Fatalf("validSubscriberTenantID(%q) = %t, want %t", testCase.tenantID, got, testCase.want)
		}
	}
}

func TestWithSubscriberTenantScopeAgainstPostgres(t *testing.T) {
	if os.Getenv("SIGNALOPS_SUBSCRIBER_RLS_INTEGRATION") != "1" {
		t.Skip("set SIGNALOPS_SUBSCRIBER_RLS_INTEGRATION=1 to run")
	}
	dsn := os.Getenv("SIGNALOPS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer repo.Close()
	repo.db.SetMaxOpenConns(1)
	repo.db.SetMaxIdleConns(1)

	if err := repo.WithSubscriberTenantScope(ctx, "tenant-scope-test", func(ctx context.Context, tx *sql.Tx) error {
		var scoped string
		if err := tx.QueryRowContext(ctx, "SELECT current_setting('signalops.tenant_id', true)").Scan(&scoped); err != nil {
			return err
		}
		if scoped != "tenant-scope-test" {
			t.Fatalf("scoped tenant = %q", scoped)
		}
		return nil
	}); err != nil {
		t.Fatalf("with subscriber tenant scope: %v", err)
	}

	var after string
	if err := repo.db.QueryRowContext(ctx, "SELECT COALESCE(current_setting('signalops.tenant_id', true), '')").Scan(&after); err != nil {
		t.Fatalf("read tenant context after commit: %v", err)
	}
	if after != "" {
		t.Fatalf("tenant context leaked after transaction: %q", after)
	}
}
