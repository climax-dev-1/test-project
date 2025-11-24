package migration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenExchangeService handles the migration token exchange
type TokenExchangeService struct {
	passageValidator *PassageValidator
	auth0Issuer      *Auth0Issuer
	
	// Cache to prevent duplicate migrations and track progress
	migrationCache map[string]*MigrationRecord
	cacheMutex     sync.RWMutex
}

// MigrationRecord tracks a user's migration status
type MigrationRecord struct {
	PassageUserID string
	Auth0UserID   string
	Email         string
	MigratedAt    time.Time
	LastExchange  time.Time
}

// ExchangeResult contains the result of a token exchange
type ExchangeResult struct {
	Auth0UserID   string
	Email         string
	AccessToken   string
	IDToken       string
	ExpiresIn     int
	IsNewMigration bool // true if this is the first time migrating this user
}

// NewTokenExchangeService creates a new token exchange service
func NewTokenExchangeService(
	passageAppID, passageAPIKey string,
	auth0Domain, auth0ClientID, auth0ClientSecret, auth0Audience, auth0Connection string,
) (*TokenExchangeService, error) {
	passageValidator, err := NewPassageValidator(passageAppID, passageAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create passage validator: %w", err)
	}

	auth0Issuer := NewAuth0Issuer(auth0Domain, auth0ClientID, auth0ClientSecret, auth0Audience, auth0Connection)

	return &TokenExchangeService{
		passageValidator: passageValidator,
		auth0Issuer:      auth0Issuer,
		migrationCache:   make(map[string]*MigrationRecord),
	}, nil
}

// ExchangeToken exchanges a Passage JWT for an Auth0 JWT
func (s *TokenExchangeService) ExchangeToken(ctx context.Context, passageToken string) (*ExchangeResult, error) {
	// Step 1: Validate the Passage token
	migratedUser, err := s.passageValidator.ValidateToken(ctx, passageToken)
	if err != nil {
		return nil, fmt.Errorf("invalid passage token: %w", err)
	}

	if migratedUser.Email == "" && migratedUser.Phone == "" {
		return nil, fmt.Errorf("passage user has no email or phone")
	}

	// Use email as primary identifier
	identifier := migratedUser.Email
	if identifier == "" {
		identifier = migratedUser.Phone
	}

	// Step 2: Check if user is already migrated
	s.cacheMutex.RLock()
	migration, exists := s.migrationCache[migratedUser.ID]
	s.cacheMutex.RUnlock()

	isNewMigration := !exists

	// Step 3: Get Auth0 management token
	mgmtToken, err := s.auth0Issuer.GetManagementToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth0 management token: %w", err)
	}

	// Step 4: Find or create user in Auth0
	auth0User, err := s.auth0Issuer.FindOrCreateUser(mgmtToken, identifier, migratedUser.EmailVerified)
	if err != nil {
		return nil, fmt.Errorf("failed to create/find auth0 user: %w", err)
	}

	// Step 5: Update migration cache
	s.cacheMutex.Lock()
	if migration == nil {
		migration = &MigrationRecord{
			PassageUserID: migratedUser.ID,
			Auth0UserID:   auth0User.UserID,
			Email:         identifier,
			MigratedAt:    time.Now(),
		}
	}
	migration.LastExchange = time.Now()
	s.migrationCache[migratedUser.ID] = migration
	s.cacheMutex.Unlock()

	// Step 6: For now, return a response indicating user is migrated
	// In production, you'd implement custom token issuance via Auth0 Actions
	return &ExchangeResult{
		Auth0UserID:    auth0User.UserID,
		Email:          identifier,
		IsNewMigration: isNewMigration,
		// Note: Actual token issuance requires Auth0 Actions or custom grant
		// See the implementation guide below
	}, nil
}

// GetMigrationStatus returns the migration status for a Passage user ID
func (s *TokenExchangeService) GetMigrationStatus(passageUserID string) (*MigrationRecord, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	
	record, exists := s.migrationCache[passageUserID]
	return record, exists
}

// GetMigrationStats returns statistics about the migration
func (s *TokenExchangeService) GetMigrationStats() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	
	return map[string]interface{}{
		"total_migrated_users": len(s.migrationCache),
		"cache_size":           len(s.migrationCache),
	}
}

