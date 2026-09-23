// Package application contains use cases for the calculator application.
//
// This package coordinates domain logic and external dependencies. It does
// not know how requests arrived or how data is ultimately persisted.
package application

import (
	"github.com/MadsWettergren/over-engineered-calculator/internal/calculator"
	"github.com/MadsWettergren/over-engineered-calculator/internal/history"
)

// CalculatorService coordinates calculation and history storage.
type CalculatorService struct {
	// The calculator itself is kept as a dependency conceptually, although
	// the current calculator implementation exposes a package-level function.
	//
	// The repository is an interface, which means this service can work with
	// the in-memory repository now and a database repository later.
	repository history.Repository
}

// NewCalculatorService creates an application service.
func NewCalculatorService(repository history.Repository) *CalculatorService {
	return &CalculatorService{
		repository: repository,
	}
}

// Calculate performs a calculation and stores the result in history.
//
// The method first performs the calculation. If that fails, nothing is
// written to history. This is important: invalid calculations should not
// appear as successful historical records.
func (s *CalculatorService) Calculate(input calculator.Input) (history.Calculation, error) {
	result, err := calculator.Calculate(input)
	if err != nil {
		return history.Calculation{}, err
	}

	record := history.Calculation{
		Operation: input.Operation,
		Left:      input.Left,
		Right:     input.Right,
		Result:    result,
	}

	if err := s.repository.Add(record); err != nil {
		return history.Calculation{}, err
	}

	// The in-memory repository assigns the ID and timestamp when Add is
	// called. We retrieve the list and return the newest record.
	//
	// This is acceptable for our first in-memory implementation. When we
	// add a database repository, we will improve the Repository interface
	// so Add returns the stored record directly.
	records := s.repository.List()

	return records[len(records)-1], nil
}

// History returns all stored calculations.
func (s *CalculatorService) History() []history.Calculation {
	return s.repository.List()
}
