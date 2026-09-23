package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MadsWettergren/over-engineered-calculator/internal/application"
	"github.com/MadsWettergren/over-engineered-calculator/internal/history"
)

// newTestHandler creates a Handler backed by a fresh in-memory repository.
//
// Each test gets its own handler so that state from one test cannot leak
// into another.
func newTestHandler() *Handler {
	repository := history.NewMemoryRepository()
	service := application.NewCalculatorService(repository)

	return NewHandler(service)
}

func TestCreateCalculation(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantErrCode string // empty means we expect success, not an error
	}{
		{
			name:       "performs a valid calculation",
			body:       `{"operation":"multiply","left":6,"right":7}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:        "rejects division by zero",
			body:        `{"operation":"divide","left":10,"right":0}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "division_by_zero",
		},
		{
			name:        "rejects an unknown operation",
			body:        `{"operation":"modulo","left":10,"right":3}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "unknown_operation",
		},
		{
			name:        "rejects malformed JSON",
			body:        `{"operation":`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_json",
		},
		{
			name:        "rejects unknown fields",
			body:        `{"operation":"add","left":1,"right":2,"unit":"meters"}`,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_json",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newTestHandler()

			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/calculations",
				bytes.NewBufferString(testCase.body),
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d (body: %s)",
					testCase.wantStatus, recorder.Code, recorder.Body.String())
			}

			if testCase.wantErrCode != "" {
				var response errorResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}

				if response.Error.Code != testCase.wantErrCode {
					t.Errorf("expected error code %q, got %q",
						testCase.wantErrCode, response.Error.Code)
				}

				return
			}

			var record history.Calculation
			if err := json.Unmarshal(recorder.Body.Bytes(), &record); err != nil {
				t.Fatalf("failed to decode calculation response: %v", err)
			}

			if record.ID == "" {
				t.Error("expected response to include an ID")
			}

			if record.CreatedAt.IsZero() {
				t.Error("expected response to include a creation time")
			}
		})
	}
}

func TestCreateCalculation_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodDelete, "/v1/calculations", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestListCalculations(t *testing.T) {
	handler := newTestHandler()

	// Seed two calculations through the real POST path, so this test also
	// exercises the create-then-list flow end to end.
	seedBodies := []string{
		`{"operation":"add","left":1,"right":2}`,
		`{"operation":"subtract","left":10,"right":3}`,
	}

	for _, body := range seedBodies {
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/calculations",
			bytes.NewBufferString(body),
		)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("seed request failed with status %d", recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/calculations", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var records []history.Calculation
	if err := json.Unmarshal(recorder.Body.Bytes(), &records); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Result != 3 {
		t.Errorf("expected first result 3, got %v", records[0].Result)
	}

	if records[1].Result != 7 {
		t.Errorf("expected second result 7, got %v", records[1].Result)
	}
}

func TestHandleHealth(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodPost, "/health/live", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestServeHTTP_UnknownPath(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestServeOpenAPISpec(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("openapi:")) {
		t.Errorf("expected response to start with 'openapi:', got: %s", recorder.Body.String())
	}
}
