package application

import (
	"errors"
	"testing"

	"github.com/MadsWettergren/over-engineered-calculator/internal/calculator"
	"github.com/MadsWettergren/over-engineered-calculator/internal/history"
)

func TestCalculatorServiceCalculatesAndStoresResult(t *testing.T) {
	repository := history.NewMemoryRepository()
	service := NewCalculatorService(repository)

	record, err := service.Calculate(calculator.Input{
		Operation: "multiply",
		Left:      6,
		Right:     7,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if record.Operation != "multiply" {
		t.Errorf("expected operation multiply, got %s", record.Operation)
	}

	if record.Result != 42 {
		t.Errorf("expected result 42, got %v", record.Result)
	}

	if record.ID == "" {
		t.Error("expected result to have an ID")
	}

	storedRecords := service.History()

	if len(storedRecords) != 1 {
		t.Fatalf("expected one stored record, got %d", len(storedRecords))
	}
}

func TestCalculatorServiceDoesNotStoreFailedCalculation(t *testing.T) {
	repository := history.NewMemoryRepository()
	service := NewCalculatorService(repository)

	_, err := service.Calculate(calculator.Input{
		Operation: "divide",
		Left:      10,
		Right:     0,
	})

	if !errors.Is(err, calculator.ErrDivisionByZero) {
		t.Fatalf("expected division-by-zero error, got %v", err)
	}

	storedRecords := service.History()

	if len(storedRecords) != 0 {
		t.Fatalf(
			"expected failed calculation not to be stored, got %d records",
			len(storedRecords),
		)
	}
}

func TestCalculatorServiceReturnsAllHistory(t *testing.T) {
	repository := history.NewMemoryRepository()
	service := NewCalculatorService(repository)

	operations := []calculator.Input{
		{
			Operation: "add",
			Left:      1,
			Right:     2,
		},
		{
			Operation: "subtract",
			Left:      10,
			Right:     3,
		},
	}

	for _, input := range operations {
		if _, err := service.Calculate(input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	records := service.History()

	if len(records) != 2 {
		t.Fatalf("expected two records, got %d", len(records))
	}

	if records[0].Result != 3 {
		t.Errorf("expected first result 3, got %v", records[0].Result)
	}

	if records[1].Result != 7 {
		t.Errorf("expected second result 7, got %v", records[1].Result)
	}
}
