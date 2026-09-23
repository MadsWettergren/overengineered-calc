// Package calculator contains the core calculation rules.
//
// This package deliberately does not know anything about HTTP, JSON,
// databases, Docker, or web pages. Keeping the core logic independent
// makes it easy to test and reuse.
package calculator

import "errors"

// These errors describe expected calculator failures.
//
// We define them as package-level values so callers can check the exact
// kind of failure with errors.Is.
var (
	// ErrDivisionByZero is returned when division has a zero right operand.
	ErrDivisionByZero = errors.New("cannot divide by zero")

	// ErrUnknownOperation is returned when the operation is not supported.
	ErrUnknownOperation = errors.New("unknown operation")
)

// Input contains the values and operation required for a calculation.
type Input struct {
	// Operation is expected to be one of:
	// "add", "subtract", "multiply", or "divide".
	Operation string

	// Left is the first number in the calculation.
	Left float64

	// Right is the second number in the calculation.
	Right float64
}

// Calculate performs one basic arithmetic operation.
//
// It returns either:
//   - the calculated result and a nil error, or
//   - zero and a descriptive error.
//
// Keeping this function independent of the web layer is important. Later,
// an HTTP handler will call this function, but the function itself will
// remain unaware that HTTP exists.
func Calculate(input Input) (float64, error) {
	switch input.Operation {
	case "add":
		return input.Left + input.Right, nil

	case "subtract":
		return input.Left - input.Right, nil

	case "multiply":
		return input.Left * input.Right, nil

	case "divide":
		// Division by zero is not valid, so we handle it explicitly
		// instead of allowing an incorrect result to escape.
		if input.Right == 0 {
			return 0, ErrDivisionByZero
		}

		return input.Left / input.Right, nil

	default:
		// Returning a known error gives the HTTP layer a way to turn
		// this into a useful client response later.
		return 0, ErrUnknownOperation
	}
}
