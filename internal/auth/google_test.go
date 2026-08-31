package auth

import (
	"testing"
)

func TestVerifyGoogleIDToken_Empty(t *testing.T) {
	_, err := VerifyGoogleIDToken("", "")
	if err == nil {
		t.Errorf("expected error for empty token, got nil")
	}
}

func TestVerifyGoogleIDToken_Invalid(t *testing.T) {
	_, err := VerifyGoogleIDToken("invalid-jwt-token", "test-client-id")
	if err == nil {
		t.Errorf("expected error for invalid token, got nil")
	}
}
