package calculator

import (
	"errors"
	"testing"
)

// TestCalculate verifies the calculator's normal and error behavior.
//
// Go test functions must begin with "Test" and receive *testing.T.
func TestCalculate(t *testing.T) {
	// Each entry describes one independent behavior.
	//
	// Keeping test cases in a table makes it easy to add more cases
	// without duplicating the test structure.
	tests := []struct {
		name      string
		input     Input
		want      float64
		wantError error
	}{
		{
			name: "adds two numbers",
			input: Input{
				Operation: "add",
				Left:      2,
				Right:     3,
			},
			want: 5,
		},
		{
			name: "subtracts two numbers",
			input: Input{
				Operation: "subtract",
				Left:      10,
				Right:     4,
			},
			want: 6,
		},
		{
			name: "multiplies two numbers",
			input: Input{
				Operation: "multiply",
				Left:      6,
				Right:     7,
			},
			want: 42,
		},
		{
			name: "divides two numbers",
			input: Input{
				Operation: "divide",
				Left:      20,
				Right:     5,
			},
			want: 4,
		},
		{
			name: "supports negative numbers",
			input: Input{
				Operation: "add",
				Left:      -2,
				Right:     5,
			},
			want: 3,
		},
		{
			name: "rejects division by zero",
			input: Input{
				Operation: "divide",
				Left:      10,
				Right:     0,
			},
			wantError: ErrDivisionByZero,
		},
		{
			name: "rejects unknown operations",
			input: Input{
				Operation: "power",
				Left:      2,
				Right:     3,
			},
			wantError: ErrUnknownOperation,
		},
	}

	for _, test := range tests {
		// Run each table entry as a separately named test.
		t.Run(test.name, func(t *testing.T) {
			got, err := Calculate(test.input)

			// Check the expected error first.
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf(
						"expected error %v, got %v",
						test.wantError,
						err,
					)
				}

				// This test expects an error, so there is no result
				// value that needs to be checked.
				return
			}

			// For successful calculations, an error should not occur.
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check the calculated result.
			if got != test.want {
				t.Fatalf("expected result %v, got %v", test.want, got)
			}
		})
	}
}
