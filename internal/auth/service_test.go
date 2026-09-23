package auth

import (
	"errors"
	"testing"
)

func TestServiceRegisterAndLogin(t *testing.T) {
	service := NewService(NewMemoryRepository())

	user, err := service.Register("ada@example.com", "correct horse battery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Email != "ada@example.com" {
		t.Errorf("expected email to be stored, got %q", user.Email)
	}

	if user.PasswordHash == "" || user.PasswordHash == "correct horse battery" {
		t.Error("expected password to be hashed, not stored as-is")
	}

	token, err := service.Login("ada@example.com", "correct horse battery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("expected a non-empty token")
	}

	userID, err := service.Authenticate(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if userID != user.ID {
		t.Errorf("expected authenticated user ID %q, got %q", user.ID, userID)
	}
}

func TestServiceRegisterRejectsShortPassword(t *testing.T) {
	service := NewService(NewMemoryRepository())

	_, err := service.Register("ada@example.com", "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestServiceRegisterRejectsDuplicateEmail(t *testing.T) {
	service := NewService(NewMemoryRepository())

	if _, err := service.Register("ada@example.com", "correct horse battery"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := service.Register("ada@example.com", "a different password")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestServiceLoginRejectsWrongPassword(t *testing.T) {
	service := NewService(NewMemoryRepository())

	if _, err := service.Register("ada@example.com", "correct horse battery"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := service.Login("ada@example.com", "wrong password entirely")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceLoginRejectsUnknownEmail(t *testing.T) {
	service := NewService(NewMemoryRepository())

	_, err := service.Login("missing@example.com", "whatever password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceAuthenticateRejectsUnknownToken(t *testing.T) {
	service := NewService(NewMemoryRepository())

	_, err := service.Authenticate("not-a-real-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
