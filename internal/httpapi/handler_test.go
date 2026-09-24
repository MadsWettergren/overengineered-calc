package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MadsWettergren/over-engineered-calculator/internal/application"
	"github.com/MadsWettergren/over-engineered-calculator/internal/auth"
	"github.com/MadsWettergren/over-engineered-calculator/internal/history"
)

// newTestHandler creates a Handler backed by fresh in-memory repositories.
//
// Each test gets its own handler so that state from one test cannot leak
// into another.
func newTestHandler() *Handler {
	repository := history.NewMemoryRepository()
	service := application.NewCalculatorService(repository)

	authRepository := auth.NewMemoryRepository()
	authService := auth.NewService(authRepository)

	return NewHandler(service, authService)
}

// newAuthenticatedTestHandler creates a handler and returns a bearer token
// for a freshly registered test user, for tests that exercise endpoints
// behind requireAuth.
func newAuthenticatedTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()

	handler := newTestHandler()

	if _, err := handler.auth.Register("test@example.com", "correct horse battery"); err != nil {
		t.Fatalf("failed to register test user: %v", err)
	}

	token, err := handler.auth.Login("test@example.com", "correct horse battery")
	if err != nil {
		t.Fatalf("failed to log in test user: %v", err)
	}

	return handler, token
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
			handler, token := newAuthenticatedTestHandler(t)

			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/calculations",
				bytes.NewBufferString(testCase.body),
			)
			request.Header.Set("Authorization", "Bearer "+token)
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
	handler, token := newAuthenticatedTestHandler(t)

	request := httptest.NewRequest(http.MethodDelete, "/v1/calculations", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestListCalculations(t *testing.T) {
	handler, token := newAuthenticatedTestHandler(t)

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
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("seed request failed with status %d", recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/calculations", nil)
	request.Header.Set("Authorization", "Bearer "+token)
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

func TestRequireAuth_MissingHeader(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/v1/calculations", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/v1/calculations", nil)
	request.Header.Set("Authorization", "Bearer not-a-real-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestHandleRegister(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d (body: %s)",
			http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response registerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Email != "ada@example.com" {
		t.Errorf("expected email to be echoed back, got %q", response.Email)
	}

	if response.ID == "" {
		t.Error("expected response to include an ID")
	}
}

func TestHandleRegister_DuplicateEmail(t *testing.T) {
	handler := newTestHandler()

	body := `{"email":"ada@example.com","password":"correct horse battery"}`

	for i, wantStatus := range []int{http.StatusCreated, http.StatusConflict} {
		request := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		if recorder.Code != wantStatus {
			t.Fatalf("attempt %d: expected status %d, got %d", i+1, wantStatus, recorder.Code)
		}
	}
}

func TestHandleLogin(t *testing.T) {
	handler := newTestHandler()

	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	handler.ServeHTTP(httptest.NewRecorder(), registerRequest)

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, loginRequest)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (body: %s)",
			http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response loginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Token == "" {
		t.Error("expected a non-empty token")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	handler := newTestHandler()

	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	handler.ServeHTTP(httptest.NewRecorder(), registerRequest)

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"totally wrong password"}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, loginRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestEndToEnd_RegisterLoginCalculate(t *testing.T) {
	handler := newTestHandler()

	register := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/register",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	handler.ServeHTTP(httptest.NewRecorder(), register)

	loginRecorder := httptest.NewRecorder()
	login := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewBufferString(`{"email":"ada@example.com","password":"correct horse battery"}`),
	)
	handler.ServeHTTP(loginRecorder, login)

	var loginResp loginResponse
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	calculateRecorder := httptest.NewRecorder()
	calculate := httptest.NewRequest(
		http.MethodPost,
		"/v1/calculations",
		bytes.NewBufferString(`{"operation":"add","left":2,"right":3}`),
	)
	calculate.Header.Set("Authorization", "Bearer "+loginResp.Token)

	handler.ServeHTTP(calculateRecorder, calculate)

	if calculateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d (body: %s)",
			http.StatusCreated, calculateRecorder.Code, calculateRecorder.Body.String())
	}
}

func TestServeWebPage(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte("<html")) {
		t.Errorf("expected response to contain HTML")
	}
}
