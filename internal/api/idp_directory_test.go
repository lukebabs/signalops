package api

import "testing"

func TestKeycloakUserSearchURL(t *testing.T) {
	got, err := keycloakUserSearchURL("https://auth.example.test/realms/syncratic", "luke@example.test")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://auth.example.test/admin/realms/syncratic/users?briefRepresentation=true&max=20&search=luke%40example.test"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestKeycloakUserSearchURLRejectsInvalidIssuer(t *testing.T) {
	if _, err := keycloakUserSearchURL("", "lu"); err == nil {
		t.Fatal("expected invalid issuer error")
	}
}
