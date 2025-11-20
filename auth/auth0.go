package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserContext key for storing user info in context
type contextKey string

const UserContextKey contextKey = "user"

// UserInfo contains the authenticated user's information
type UserInfo struct {
	UserID string
	Email  string
}

// Auth0Config holds Auth0 configuration
type Auth0Config struct {
	Domain   string
	Audience string
}

// JWKS represents the JSON Web Key Set
type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

// JSONWebKey represents a single key in JWKS
type JSONWebKey struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// Middleware creates an HTTP middleware for Auth0 authentication
func Middleware(config Auth0Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			userInfo, err := validateToken(tokenString, config)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserContextKey, userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateToken validates the JWT token and extracts user information
func validateToken(tokenString string, config Auth0Config) (*UserInfo, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the kid from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found")
		}

		// Fetch JWKS
		jwks, err := fetchJWKS(config.Domain)
		if err != nil {
			return nil, err
		}

		// Find the key
		key := findKey(jwks, kid)
		if key == nil {
			return nil, errors.New("unable to find appropriate key")
		}

		// Convert to RSA public key
		return convertKey(key)
	})

	if err != nil {
		return nil, err
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Verify audience
	aud, ok := claims["aud"].(string)
	if !ok {
		// Try as array
		if audArray, ok := claims["aud"].([]interface{}); ok {
			found := false
			for _, a := range audArray {
				if audStr, ok := a.(string); ok && audStr == config.Audience {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.New("invalid audience")
			}
		} else {
			return nil, errors.New("audience claim not found")
		}
	} else if aud != config.Audience {
		return nil, errors.New("invalid audience")
	}

	// Verify issuer
	expectedIssuer := fmt.Sprintf("https://%s/", config.Domain)
	iss, ok := claims["iss"].(string)
	if !ok || iss != expectedIssuer {
		return nil, errors.New("invalid issuer")
	}

	// Verify expiration
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("expiration claim not found")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, errors.New("token expired")
	}

	// Extract user info
	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("sub claim not found")
	}

	email, _ := claims["email"].(string) // Email might not always be present

	return &UserInfo{
		UserID: userID,
		Email:  email,
	}, nil
}

// fetchJWKS fetches the JSON Web Key Set from Auth0
func fetchJWKS(domain string) (*JWKS, error) {
	url := fmt.Sprintf("https://%s/.well-known/jwks.json", domain)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	return &jwks, nil
}

// findKey finds a key in JWKS by kid
func findKey(jwks *JWKS, kid string) *JSONWebKey {
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return &key
		}
	}
	return nil
}

// convertKey converts a JSONWebKey to an RSA public key
func convertKey(key *JSONWebKey) (*rsa.PublicKey, error) {
	// Decode N
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}

	// Decode E
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	// Convert E to int
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}

// GetUserFromContext extracts user info from context
func GetUserFromContext(ctx context.Context) (*UserInfo, error) {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}

