// The api command starts the calculator HTTP service.
package main

import (
	"log"
	"net/http"

	"github.com/MadsWettergren/over-engineered-calculator/internal/application"
	"github.com/MadsWettergren/over-engineered-calculator/internal/history"
	"github.com/MadsWettergren/over-engineered-calculator/internal/httpapi"
)

func main() {
	// Create the repository implementation.
	//
	// For now, history is stored in memory. Later, main.go will choose
	// between an in-memory repository and a database repository based on
	// configuration.
	repository := history.NewMemoryRepository()

	// Wire the application service to the repository.
	service := application.NewCalculatorService(repository)

	// Wire the HTTP handler to the application service.
	handler := httpapi.NewHandler(service)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("calculator API listening on http://localhost:8080")

	// ListenAndServe blocks while the server handles requests.
	//
	// log.Fatal exits the process if the server cannot start or stops with
	// an unexpected error.
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
