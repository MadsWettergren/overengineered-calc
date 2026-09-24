package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MadsWettergren/over-engineered-calculator/internal/auth"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registerResponse deliberately excludes the password hash. There is no
// reason for it to ever leave the server.
type registerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// handleRegister creates a new account.
func (h *Handler) handleRegister(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		responseWriter.Header().Set("Allow", "POST")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body registerRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeError(
			responseWriter,
			http.StatusBadRequest,
			"invalid_json",
			"request body must contain valid JSON",
		)
		return
	}

	user, err := h.auth.Register(body.Email, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidEmail):
			writeError(responseWriter, http.StatusBadRequest, "invalid_email", "email must not be empty")

		case errors.Is(err, auth.ErrWeakPassword):
			writeError(responseWriter, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")

		case errors.Is(err, auth.ErrEmailTaken):
			writeError(responseWriter, http.StatusConflict, "email_taken", "email is already registered")

		default:
			writeError(responseWriter, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		}

		return
	}

	writeJSON(responseWriter, http.StatusCreated, registerResponse{
		ID:    user.ID,
		Email: user.Email,
	})
}

// handleLogin verifies credentials and returns a bearer token.
func (h *Handler) handleLogin(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		responseWriter.Header().Set("Allow", "POST")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body loginRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeError(
			responseWriter,
			http.StatusBadRequest,
			"invalid_json",
			"request body must contain valid JSON",
		)
		return
	}

	token, err := h.auth.Login(body.Email, body.Password)
	if err != nil {
		// auth.Login already collapses "unknown email" and "wrong
		// password" into one error. We do the same here rather than
		// re-introducing a distinction at the HTTP layer.
		writeError(responseWriter, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	writeJSON(responseWriter, http.StatusOK, loginResponse{Token: token})
}
