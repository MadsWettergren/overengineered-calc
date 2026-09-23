// Package history contains storage-related types and interfaces for
// completed calculations.
//
// The package does not decide whether data is stored in memory, a database,
// or another system. It defines the contract that those implementations
// must follow.
package history

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Calculation represents one calculation that has been performed.
//
// We store the original inputs as well as the result. This means the
// history remains understandable even if the calculator gains new behavior
// in the future.
type Calculation struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Left      float64   `json:"left"`
	Right     float64   `json:"right"`
	Result    float64   `json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository describes the operations that the application needs from
// calculation history.
//
// The application will depend on this interface, not on a specific storage
// implementation. That makes the application easy to test and allows us to
// replace this in-memory implementation with PostgreSQL later.
type Repository interface {
	Add(calculation Calculation) error
	List() []Calculation
}

// MemoryRepository stores calculations in memory.
//
// This is useful for the first version because:
//   - it requires no database setup;
//   - it is fast;
//   - it is easy to use in tests.
//
// Data will disappear when the application stops. We will address that
// later with a database-backed repository.
type MemoryRepository struct {
	mu           sync.RWMutex
	calculations []Calculation
}

// NewMemoryRepository creates an empty in-memory history repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		calculations: make([]Calculation, 0),
	}
}

// Add stores one calculation in history.
//
// We use a mutex because HTTP requests may be handled concurrently. Without
// synchronization, two requests could modify the slice at the same time.
func (r *MemoryRepository) Add(calculation Calculation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create an ID here if the caller has not already provided one.
	if calculation.ID == "" {
		calculation.ID = uuid.NewString()
	}

	// Set the creation time here if it was not already provided.
	if calculation.CreatedAt.IsZero() {
		calculation.CreatedAt = time.Now().UTC()
	}

	r.calculations = append(r.calculations, calculation)

	return nil
}

// List returns all stored calculations.
//
// We return a copy of the slice rather than the internal slice itself.
// This prevents callers from accidentally modifying repository state.
func (r *MemoryRepository) List() []Calculation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Calculation, len(r.calculations))
	copy(result, r.calculations)

	return result
}
