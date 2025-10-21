package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chitrace "github.com/DataDog/dd-trace-go/contrib/go-chi/chi/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/go-chi/chi/v5"
	"github.com/mharner33/telephone/handlers"
	"github.com/mharner33/telephone/message"
	"github.com/sirupsen/logrus"
)

func main() {
	// Set up logrus with JSON formatting
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// CLI flag to select LLM provider (ollama or gemini). Default: gemini
	llmProvider := flag.String("llm", "gemini", "LLM provider: 'ollama' or 'gemini'")
	flag.Parse()

	tracer.Start(
		tracer.WithService("dd-telephone"),
		tracer.WithEnv("dev"),
		tracer.WithServiceVersion("0.1.0"),
	)
	defer tracer.Stop()

	// If you expect your application to be shut down by SIGTERM (for example, a container in Kubernetes),
	// you might want to listen for that signal and explicitly stop the tracer to ensure no data is lost
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		<-sigChan
		tracer.Stop()
	}()

	// Select LLM backend based on flag
	if *llmProvider == "ollama" {
		message.SetUseOllama(true)
		logrus.Info("Using LLM provider: ollama")
	} else {
		message.SetUseOllama(false)
		logrus.Info("Using LLM provider: gemini")
	}

	// Create Chi router with API v1 base path
	r := chi.NewRouter()

	// Add DataDog tracing middleware
	r.Use(chitrace.Middleware(chitrace.WithService("dd-telephone")))

	// Group routes under /api/v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/message", handlers.MessageHandler)
		r.Get("/health", handlers.HealthHandler)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logrus.WithFields(logrus.Fields{
		"port": port,
	}).Info("Starting server")
	logrus.Fatal(server.ListenAndServe())
}

// Removed local handlers; now using handlers package
