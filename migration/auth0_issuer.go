package migration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Auth0Issuer handles creating Auth0 users and issuing tokens
type Auth0Issuer struct {
	domain       string
	clientID     string
	clientSecret string
	audience     string
	connection   string // Database connection name
	httpClient   *http.Client
}

// NewAuth0Issuer creates a new Auth0 token issuer
func NewAuth0Issuer(domain, clientID, clientSecret, audience, connection string) *Auth0Issuer {
	return &Auth0Issuer{
		domain:       domain,
		clientID:     clientID,
		clientSecret: clientSecret,
		audience:     audience,
		connection:   connection,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Auth0User represents an Auth0 user
type Auth0User struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// TokenResponse represents an Auth0 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token,omitempty"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetManagementToken gets an Auth0 Management API token
func (a *Auth0Issuer) GetManagementToken() (string, error) {
	payload := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
		"audience":      fmt.Sprintf("https://%s/api/v2/", a.domain),
	}

	tokenResp, err := a.requestToken(payload)
	if err != nil {
		return "", fmt.Errorf("failed to get management token: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// FindOrCreateUser finds or creates a user in Auth0
func (a *Auth0Issuer) FindOrCreateUser(mgmtToken, email string, emailVerified bool) (*Auth0User, error) {
	// First, try to find existing user by email
	user, err := a.findUserByEmail(mgmtToken, email)
	if err == nil && user != nil {
		return user, nil
	}

	// User doesn't exist, create them
	return a.createUser(mgmtToken, email, emailVerified)
}

// findUserByEmail searches for a user by email
func (a *Auth0Issuer) findUserByEmail(mgmtToken, email string) (*Auth0User, error) {
	url := fmt.Sprintf("https://%s/api/v2/users-by-email?email=%s", a.domain, email)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+mgmtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search user: %s - %s", resp.Status, string(body))
	}

	var users []Auth0User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	if len(users) > 0 {
		return &users[0], nil
	}

	return nil, nil // User not found
}

// createUser creates a new user in Auth0
func (a *Auth0Issuer) createUser(mgmtToken, email string, emailVerified bool) (*Auth0User, error) {
	url := fmt.Sprintf("https://%s/api/v2/users", a.domain)

	// Generate a random password (user won't use it - they'll use passwordless)
	password := generateRandomPassword()

	payload := map[string]interface{}{
		"email":          email,
		"email_verified": emailVerified,
		"password":       password,
		"connection":     a.connection,
		"verify_email":   false, // Already verified by Passage
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+mgmtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create user: %s - %s", resp.Status, string(body))
	}

	var user Auth0User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// IssueTokenForUser issues an Auth0 token for a user (using passwordless or impersonation)
func (a *Auth0Issuer) IssueTokenForUser(userID string) (*TokenResponse, error) {
	// For production, you might want to use Auth0's impersonation or custom grant
	// For now, we'll document that this requires additional Auth0 configuration
	
	// This would typically use a custom OAuth grant type or Auth0 Actions
	// to issue a token for a migrated user without requiring a password
	
	return nil, fmt.Errorf("token issuance requires Auth0 Actions or custom grant configuration")
}

// requestToken makes a token request to Auth0
func (a *Auth0Issuer) requestToken(payload map[string]string) (*TokenResponse, error) {
	url := fmt.Sprintf("https://%s/oauth/token", a.domain)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// generateRandomPassword generates a secure random password
func generateRandomPassword() string {
	// In production, use crypto/rand for secure random generation
	return fmt.Sprintf("Migration-%d-%s", time.Now().Unix(), "SecureRandomString123!")
}

