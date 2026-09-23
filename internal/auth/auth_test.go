package auth

import (
	"errors"
	"testing"
)

func TestMemoryRepositoryCreatesAndFindsUser(t *testing.T) {
	repository := NewMemoryRepository()

	stored, err := repository.Create(User{
		Email:        "ada@example.com",
		PasswordHash: "hashed-value",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stored.ID == "" {
		t.Error("expected repository to assign an ID")
	}

	if stored.CreatedAt.IsZero() {
		t.Error("expected repository to assign a creation time")
	}

	found, err := repository.FindByEmail("ada@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found.ID != stored.ID {
		t.Errorf("expected to find the same user back")
	}
}

func TestMemoryRepositoryRejectsDuplicateEmail(t *testing.T) {
	repository := NewMemoryRepository()

	if _, err := repository.Create(User{
		Email: "ada@example.com", PasswordHash: "first",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := repository.Create(User{
		Email: "ada@example.com", PasswordHash: "second",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestMemoryRepositoryFindByEmail_NotFound(t *testing.T) {
	repository := NewMemoryRepository()

	_, err := repository.FindByEmail("missing@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
