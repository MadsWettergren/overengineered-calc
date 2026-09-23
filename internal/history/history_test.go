package history

import (
	"testing"
	"time"
)

func TestMemoryRepositoryStoresCalculations(t *testing.T) {
	repository := NewMemoryRepository()

	calculation := Calculation{
		Operation: "add",
		Left:      2,
		Right:     3,
		Result:    5,
	}

	if err := repository.Add(calculation); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calculations := repository.List()

	if len(calculations) != 1 {
		t.Fatalf("expected one calculation, got %d", len(calculations))
	}

	stored := calculations[0]

	if stored.Operation != "add" {
		t.Errorf("expected operation add, got %s", stored.Operation)
	}

	if stored.Result != 5 {
		t.Errorf("expected result 5, got %v", stored.Result)
	}

	if stored.ID == "" {
		t.Error("expected repository to assign an ID")
	}

	if stored.CreatedAt.IsZero() {
		t.Error("expected repository to assign a creation time")
	}
}

func TestMemoryRepositoryReturnsACopy(t *testing.T) {
	repository := NewMemoryRepository()

	originalTime := time.Now().UTC()

	if err := repository.Add(Calculation{
		Operation: "multiply",
		Left:      4,
		Right:     5,
		Result:    20,
		CreatedAt: originalTime,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	firstRead := repository.List()
	firstRead[0].Result = 999

	secondRead := repository.List()

	if secondRead[0].Result != 20 {
		t.Fatalf("repository was modified through returned slice: %v", secondRead[0].Result)
	}
}
