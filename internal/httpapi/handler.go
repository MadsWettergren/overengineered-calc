// Package httpapi contains HTTP handlers for the calculator application.
//
// This package is responsible for HTTP concerns such as:
//   - routing;
//   - decoding JSON;
//   - encoding JSON;
//   - choosing HTTP status codes.
//
// It does not perform arithmetic itself. Arithmetic is delegated to the
// application service.
package httpapi

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MadsWettergren/over-engineered-calculator/internal/application"
	"github.com/MadsWettergren/over-engineered-calculator/internal/auth"
	"github.com/MadsWettergren/over-engineered-calculator/internal/calculator"
)

// openAPISpecification is embedded into the compiled binary so the running
// service can serve its own documentation without depending on a file that
// might be missing at deploy time.
//
//go:embed openapi.yaml
var openAPISpecification []byte

// webPage is the small HTML/JS client bundled with the API. Embedding it
// keeps the binary self-contained, the same reasoning as the OpenAPI spec
// above.
//
//go:embed web/index.html
var webPage []byte

// Handler contains the dependencies required by the HTTP API.
type Handler struct {
	service *application.CalculatorService
	auth    *auth.Service
}

// NewHandler creates an HTTP handler using the provided application and
// auth services.
func NewHandler(service *application.CalculatorService, authService *auth.Service) *Handler {
	return &Handler{
		service: service,
		auth:    authService,
	}
}

// calculateRequest is the JSON structure accepted by the calculation
// endpoint.
//
// We use a separate HTTP request type instead of exposing the domain's Input
// type directly. This keeps the transport format separate from the domain
// model and gives us freedom to change either one later.
type calculateRequest struct {
	Operation string  `json:"operation"`
	Left      float64 `json:"left"`
	Right     float64 `json:"right"`
}

// errorResponse is the consistent JSON format used for API errors.
type errorResponse struct {
	Error errorDetails `json:"error"`
}

// errorDetails contains a machine-readable code and a human-readable
// message.
type errorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ServeHTTP routes incoming requests to the appropriate handler.
//
// We start with a small manual router because there are only a few endpoints.
// If the API grows, we can introduce a router library later.
func (h *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/v1/calculations":
		requireAuth(h.auth, h.handleCalculations)(responseWriter, request)

	case "/v1/auth/register":
		h.handleRegister(responseWriter, request)

	case "/v1/auth/login":
		h.handleLogin(responseWriter, request)

	case "/health/live":
		h.handleHealth(responseWriter, request)

	case "/openapi.yaml":
		h.serveOpenAPISpec(responseWriter, request)

	case "/":
		h.serveWebPage(responseWriter, request)

	default:
		http.NotFound(responseWriter, request)
	}
}

// handleCalculations handles both creating and listing calculations.
//
// POST creates a calculation.
// GET returns calculation history.
func (h *Handler) handleCalculations(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	switch request.Method {
	case http.MethodPost:
		h.createCalculation(responseWriter, request)

	case http.MethodGet:
		h.listCalculations(responseWriter, request)

	default:
		responseWriter.Header().Set("Allow", "GET, POST")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// createCalculation decodes a request, delegates the calculation to the
// application service, and returns the stored record.
func (h *Handler) createCalculation(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	var input calculateRequest

	decoder := json.NewDecoder(request.Body)

	// Reject unknown JSON fields. This helps catch client typos such as
	// "operaton" instead of "operation".
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeError(
			responseWriter,
			http.StatusBadRequest,
			"invalid_json",
			"request body must contain valid JSON",
		)
		return
	}

	record, err := h.service.Calculate(calculator.Input{
		Operation: input.Operation,
		Left:      input.Left,
		Right:     input.Right,
	})
	if err != nil {
		switch {
		case errors.Is(err, calculator.ErrDivisionByZero):
			writeError(
				responseWriter,
				http.StatusUnprocessableEntity,
				"division_by_zero",
				"cannot divide by zero",
			)

		case errors.Is(err, calculator.ErrUnknownOperation):
			writeError(
				responseWriter,
				http.StatusBadRequest,
				"unknown_operation",
				"operation is not supported",
			)

		default:
			writeError(
				responseWriter,
				http.StatusInternalServerError,
				"internal_error",
				"an unexpected error occurred",
			)
		}

		return
	}

	writeJSON(responseWriter, http.StatusCreated, record)
}

// listCalculations returns all calculation history.
func (h *Handler) listCalculations(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	history := h.service.History()

	writeJSON(responseWriter, http.StatusOK, history)
}

// handleHealth reports that the process is running.
//
// This endpoint does not check a database yet because we do not have one.
// Later we can add a readiness endpoint that checks dependencies.
func (h *Handler) handleHealth(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		responseWriter.Header().Set("Allow", "GET")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(responseWriter, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// serveOpenAPISpec returns the embedded OpenAPI document.
//
// Serving this from the running binary means the documentation can never
// drift out of sync with what was actually deployed — there is only one
// copy of the file, and it travels with the compiled service.
func (h *Handler) serveOpenAPISpec(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		responseWriter.Header().Set("Allow", "GET")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/yaml")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write(openAPISpecification)
}

// serveWebPage returns the embedded web client.
func (h *Handler) serveWebPage(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		responseWriter.Header().Set("Allow", "GET")
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write(webPage)
}

// writeJSON serializes a value and writes it as a JSON response.
func writeJSON(
	responseWriter http.ResponseWriter,
	statusCode int,
	value any,
) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)

	// Encoding directly to the response avoids manually converting the
	// value to a byte slice first.
	_ = json.NewEncoder(responseWriter).Encode(value)
}

// writeError writes an API error in the standard error format.
func writeError(
	responseWriter http.ResponseWriter,
	statusCode int,
	code string,
	message string,
) {
	writeJSON(responseWriter, statusCode, errorResponse{
		Error: errorDetails{
			Code:    code,
			Message: message,
		},
	})
}
