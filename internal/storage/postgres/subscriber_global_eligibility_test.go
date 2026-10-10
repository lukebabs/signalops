package postgres

import "testing"

func TestValidUSCommonStockEvidence(t *testing.T) {
	if !validUSCommonStockEvidence([]byte(`{"country_code":"US","security_type":"common_stock","exchange_listed":true,"provider_eligible":true,"is_active":true}`)) {
		t.Fatal("valid eligible evidence rejected")
	}
	if validUSCommonStockEvidence([]byte(`{"country_code":"CA","security_type":"common_stock","exchange_listed":true,"provider_eligible":true,"is_active":true}`)) {
		t.Fatal("non-US reference accepted")
	}
	if validUSCommonStockEvidence([]byte(`{"country_code":"US","security_type":"etf","exchange_listed":true,"provider_eligible":true,"is_active":true}`)) {
		t.Fatal("non-common stock accepted")
	}
}
