package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/mharner33/telephone/handlers"
	"github.com/mharner33/telephone/message"
)

func main() {
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
		log.Println("Using LLM provider: ollama")
	} else {
		message.SetUseOllama(false)
		log.Println("Using LLM provider: gemini")
	}

	http.HandleFunc("/message", handlers.MessageHandler)
	http.HandleFunc("/health", handlers.HealthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           nil,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(server.ListenAndServe())
}

// Removed local handlers; now using handlers package
