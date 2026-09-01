package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GoogleProfile represents verified user info extracted from a Google ID Token.
type GoogleProfile struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Audience      string `json:"aud"`
	Issuer        string `json:"iss"`
}

type tokenInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
	Iss           string `json:"iss"`
	Exp           string `json:"exp"`
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// VerifyGoogleIDToken validates the Google ID token against Google's tokeninfo API.
func VerifyGoogleIDToken(idToken, expectedClientID string) (*GoogleProfile, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, errors.New("empty ID token provided")
	}

	// FINDING-07 FIX: Always require a configured Client ID for audience validation.
	expectedClientID = strings.TrimSpace(expectedClientID)
	if expectedClientID == "" {
		return nil, errors.New("server misconfigured: no Google Client ID set for audience validation")
	}

	tokenInfoURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", url.QueryEscape(idToken))
	resp, err := httpClient.Get(tokenInfoURL)
	if err != nil {
		return nil, fmt.Errorf("calling Google tokeninfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp tokenInfoResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		errMsg := errResp.ErrorDesc
		if errMsg == "" {
			errMsg = errResp.Error
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("Google verification failed with status code %d", resp.StatusCode)
		}
		return nil, errors.New(errMsg)
	}

	var ti tokenInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ti); err != nil {
		return nil, fmt.Errorf("parsing Google tokeninfo response: %w", err)
	}

	// Validate issuer
	if ti.Iss != "accounts.google.com" && ti.Iss != "https://accounts.google.com" {
		return nil, fmt.Errorf("invalid token issuer: %s", ti.Iss)
	}

	// FINDING-07 FIX: Audience validation is now unconditional.
	if ti.Aud != expectedClientID {
		return nil, fmt.Errorf("token audience does not match expected Client ID")
	}

	return &GoogleProfile{
		Sub:           ti.Sub,
		Email:         ti.Email,
		EmailVerified: ti.EmailVerified == "true" || ti.EmailVerified == "1",
		Name:          ti.Name,
		Picture:       ti.Picture,
		Audience:      ti.Aud,
		Issuer:        ti.Iss,
	}, nil
}
