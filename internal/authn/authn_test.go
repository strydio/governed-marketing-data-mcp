package authn

import "testing"

func TestAuthenticate(t *testing.T) {
	authenticator, err := NewAuthenticator("secret-token")
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}

	principal, err := authenticator.Authenticate("Bearer secret-token")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.Ref != "uk_marketing_demo" {
		t.Fatalf("unexpected principal: %s", principal.Ref)
	}
}

func TestAuthenticateDenials(t *testing.T) {
	authenticator, err := NewAuthenticator("secret-token")
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}

	cases := []string{"", "Basic secret-token", "Bearer wrong", "Bearer"}
	for _, tc := range cases {
		if _, err := authenticator.Authenticate(tc); err == nil {
			t.Fatalf("expected authentication failure for %q", tc)
		}
	}
}

func TestNewAuthenticatorRequiresToken(t *testing.T) {
	if _, err := NewAuthenticator(""); err == nil {
		t.Fatal("expected missing credentials error")
	}
}
