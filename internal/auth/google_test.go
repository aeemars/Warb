package auth

import (
	"testing"
)

func TestVerifyGoogleIDToken_Demo(t *testing.T) {
	profile, err := VerifyGoogleIDToken("demo:fatima@warbabank.com:Fatima Al-Rashidi", "")
	if err != nil {
		t.Fatalf("expected success for demo token, got error: %v", err)
	}
	if profile.Email != "fatima@warbabank.com" {
		t.Errorf("expected email fatima@warbabank.com, got %s", profile.Email)
	}
	if profile.Name != "Fatima Al-Rashidi" {
		t.Errorf("expected name Fatima Al-Rashidi, got %s", profile.Name)
	}
}

func TestVerifyGoogleIDToken_Empty(t *testing.T) {
	_, err := VerifyGoogleIDToken("", "")
	if err == nil {
		t.Errorf("expected error for empty token, got nil")
	}
}
