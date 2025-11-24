package migration

import (
	"context"
	"fmt"

	passage "github.com/passageidentity/passage-go/v2"
)

// PassageValidator handles validation of Passage JWTs
type PassageValidator struct {
	client *passage.Passage
	appID  string
}

// NewPassageValidator creates a new Passage validator
func NewPassageValidator(appID, apiKey string) (*PassageValidator, error) {
	if appID == "" {
		return nil, fmt.Errorf("passage app ID is required")
	}

	client, err := passage.New(appID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create passage client: %w", err)
	}

	return &PassageValidator{
		client: client,
		appID:  appID,
	}, nil
}

// MigratedPassageUser represents a validated Passage user for migration
type MigratedPassageUser struct {
	ID            string
	Email         string
	Phone         string
	EmailVerified bool
	PhoneVerified bool
}

// ValidateToken validates a Passage JWT and returns user information
func (v *PassageValidator) ValidateToken(ctx context.Context, token string) (*MigratedPassageUser, error) {
	// Authenticate the token with Passage
	userID, err := v.client.Auth.ValidateJWT(token)
	if err != nil {
		return nil, fmt.Errorf("invalid passage token: %w", err)
	}

	// Get user details from Passage
	user, err := v.client.User.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	migratedUser := &MigratedPassageUser{
		ID:            user.ID,
		Email:         user.Email,
		Phone:         user.Phone,
		EmailVerified: user.EmailVerified,
		PhoneVerified: user.PhoneVerified,
	}

	return migratedUser, nil
}

