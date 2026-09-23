package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidEmail is returned when an email is empty or blank.
	ErrInvalidEmail = errors.New("email must not be empty")

	// ErrWeakPassword is returned when a password is too short to be
	// worth hashing and storing.
	ErrWeakPassword = errors.New("password must be at least 8 characters")

	// ErrInvalidCredentials covers both "unknown email" and "wrong
	// password". These are deliberately not distinguished — see Login.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidToken is returned when a token is missing or unrecognized.
	ErrInvalidToken = errors.New("invalid or expired token")
)

const minPasswordLength = 8

// Service handles registration, login, and token verification.
//
// Tokens are opaque random strings held in memory, mapped to a user ID.
// There is no expiry yet — see the README for that limitation and what a
// production version would add.
type Service struct {
	users Repository

	mu     sync.RWMutex
	tokens map[string]string // token -> user ID
}

// NewService creates an auth service backed by the given user repository.
func NewService(users Repository) *Service {
	return &Service{
		users:  users,
		tokens: make(map[string]string),
	}
}

// Register creates a new account with a hashed password.
func (s *Service) Register(email, password string) (User, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return User{}, ErrInvalidEmail
	}

	if len(password) < minPasswordLength {
		return User{}, ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	return s.users.Create(User{
		Email:        email,
		PasswordHash: string(hash),
	})
}

// Login verifies credentials and issues a new token on success.
func (s *Service) Login(email, password string) (string, error) {
	user, err := s.users.FindByEmail(strings.TrimSpace(email))
	if err != nil {
		// We return the same error whether the email doesn't exist or the
		// password is wrong. Distinguishing the two in the response would
		// let a caller enumerate which emails are registered.
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash), []byte(password),
	); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.tokens[token] = user.ID
	s.mu.Unlock()

	return token, nil
}

// Authenticate resolves a token to a user ID.
func (s *Service) Authenticate(token string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.tokens[token]
	if !ok {
		return "", ErrInvalidToken
	}

	return userID, nil
}

// generateToken produces a random 32-byte token, hex-encoded.
func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
