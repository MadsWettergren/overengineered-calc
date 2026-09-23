// Package auth contains user accounts and credential verification.
//
// Like internal/history, this package defines a storage contract as an
// interface rather than committing to a specific backend. The in-memory
// implementation here is a deliberate scope decision (see the README),
// not a placeholder for something more sophisticated.
package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrEmailTaken is returned when registering an email that already
	// belongs to an account.
	ErrEmailTaken = errors.New("email is already registered")

	// ErrUserNotFound is returned when no account matches the given email.
	ErrUserNotFound = errors.New("user not found")
)

// User represents one registered account.
//
// PasswordHash is a bcrypt hash, never the plaintext password. Nothing in
// this package ever stores or logs the plaintext value.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Repository describes the storage operations the auth service needs.
type Repository interface {
	Create(user User) (User, error)
	FindByEmail(email string) (User, error)
}

// MemoryRepository stores users in memory, keyed by email.
type MemoryRepository struct {
	mu    sync.RWMutex
	users map[string]User
}

// NewMemoryRepository creates an empty in-memory user repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users: make(map[string]User),
	}
}

// Create stores a new user, rejecting duplicate emails.
func (r *MemoryRepository) Create(user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Email]; exists {
		return User{}, ErrEmailTaken
	}

	if user.ID == "" {
		user.ID = uuid.NewString()
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	r.users[user.Email] = user

	return user, nil
}

// FindByEmail looks up a user by email.
func (r *MemoryRepository) FindByEmail(email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[email]
	if !exists {
		return User{}, ErrUserNotFound
	}

	return user, nil
}
