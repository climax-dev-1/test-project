package migration

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler provides HTTP handlers for token exchange
type Handler struct {
	service *TokenExchangeService
}

// NewHandler creates a new migration handler
func NewHandler(service *TokenExchangeService) *Handler {
	return &Handler{
		service: service,
	}
}

// ExchangeTokenRequest represents the request body for token exchange
type ExchangeTokenRequest struct {
	PassageToken string `json:"passage_token"`
}

// ExchangeTokenResponse represents the response for token exchange
type ExchangeTokenResponse struct {
	Success        bool   `json:"success"`
	Auth0UserID    string `json:"auth0_user_id,omitempty"`
	Email          string `json:"email,omitempty"`
	IsNewMigration bool   `json:"is_new_migration"`
	Message        string `json:"message,omitempty"`
	
	// In production with Auth0 Actions, these would be populated:
	// AccessToken string `json:"access_token,omitempty"`
	// IDToken     string `json:"id_token,omitempty"`
	// ExpiresIn   int    `json:"expires_in,omitempty"`
	// TokenType   string `json:"token_type,omitempty"`
}

// HandleExchangeToken handles the POST /migrate/exchange-token endpoint
func (h *Handler) HandleExchangeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ExchangeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ExchangeTokenResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate passage token is provided
	if req.PassageToken == "" {
		respondJSON(w, http.StatusBadRequest, ExchangeTokenResponse{
			Success: false,
			Message: "passage_token is required",
		})
		return
	}

	// Remove "Bearer " prefix if present
	passageToken := strings.TrimPrefix(req.PassageToken, "Bearer ")

	// Exchange token
	result, err := h.service.ExchangeToken(r.Context(), passageToken)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, ExchangeTokenResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Return success response
	respondJSON(w, http.StatusOK, ExchangeTokenResponse{
		Success:        true,
		Auth0UserID:    result.Auth0UserID,
		Email:          result.Email,
		IsNewMigration: result.IsNewMigration,
		Message:        "User migrated successfully. Use passwordless authentication to get Auth0 token.",
	})
}

// HandleMigrationStats handles the GET /migrate/stats endpoint
func (h *Handler) HandleMigrationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.service.GetMigrationStats()
	respondJSON(w, http.StatusOK, stats)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

