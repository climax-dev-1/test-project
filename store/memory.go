package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/example/auth0-gqlgen-demo/graph/model"
)

// MemoryStore is an in-memory storage for accounts
type MemoryStore struct {
	mu       sync.RWMutex
	accounts map[string]*model.Account // key is userID
	nextID   int
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		accounts: make(map[string]*model.Account),
		nextID:   1,
	}
}

// GetAccountByUserID retrieves an account by Auth0 user ID
func (s *MemoryStore) GetAccountByUserID(userID string) (*model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[userID]
	if !exists {
		return nil, fmt.Errorf("account not found for user ID: %s", userID)
	}

	return account, nil
}

// CreateAccount creates a new account
func (s *MemoryStore) CreateAccount(userID, email string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if account already exists
	if existing, exists := s.accounts[userID]; exists {
		return existing, nil
	}

	// Create new account
	account := &model.Account{
		ID:        fmt.Sprintf("%d", s.nextID),
		Email:     email,
		UserID:    userID,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	s.accounts[userID] = account
	s.nextID++

	return account, nil
}

// CreateAccountIfNotExists creates an account if it doesn't exist, otherwise returns existing
func (s *MemoryStore) CreateAccountIfNotExists(userID, email string) (*model.Account, error) {
	// First try to get existing account (read lock)
	s.mu.RLock()
	if existing, exists := s.accounts[userID]; exists {
		s.mu.RUnlock()
		return existing, nil
	}
	s.mu.RUnlock()

	// If not found, acquire write lock and create
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if existing, exists := s.accounts[userID]; exists {
		return existing, nil
	}

	// Create new account
	account := &model.Account{
		ID:        fmt.Sprintf("%d", s.nextID),
		Email:     email,
		UserID:    userID,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	s.accounts[userID] = account
	s.nextID++

	return account, nil
}

// ListAccounts returns all accounts (for debugging)
func (s *MemoryStore) ListAccounts() []*model.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]*model.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		accounts = append(accounts, account)
	}

	return accounts
}

