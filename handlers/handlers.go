package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/mharner33/telephone/hosts"
	"github.com/mharner33/telephone/message"
	"github.com/sirupsen/logrus"
)

// Using DataDog tracer from main.go

type Message struct {
	OriginalText string `json:"original_text"`
	ModifiedText string `json:"modified_text"`
}

// MessageHandler handles POST /message requests
func MessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	// Start DataDog span
	span, ctx := tracer.StartSpanFromContext(r.Context(), "receive-message")

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var originalText, modifiedText string

	// If modified message is blank, this is the first host
	if msg.ModifiedText == "" {
		originalText = msg.OriginalText
		modifiedText = message.Modify(ctx, msg.OriginalText)
		logrus.WithFields(logrus.Fields{
			"original_message": originalText,
			"modified_message": modifiedText,
		}).Info("First host - processing message")
	} else {
		originalText = msg.OriginalText
		modifiedText = message.Modify(ctx, msg.ModifiedText)
		logrus.WithFields(logrus.Fields{
			"original_message":     originalText,
			"previous_modified":    msg.ModifiedText,
			"new_modified_message": modifiedText,
		}).Info("Processing message")
	}

	span.SetTag("original.message", originalText)
	span.SetTag("modified.message", modifiedText)

	//nextServiceURL := os.Getenv("NEXT_SERVICE_URL")
	nextServiceURL := hosts.GetNextHostURL()
	if nextServiceURL != "" {
		go forwardMessage(ctx, originalText, modifiedText, nextServiceURL)
	} else {
		logrus.Info("End of the line. No NEXT_SERVICE_URL configured.")
	}

	span.Finish()
	io.WriteString(w, "Message received and forwarded (maybe)")
}

// HealthHandler handles GET /health requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

func forwardMessage(ctx context.Context, originalText string, modifiedText string, url string) {
	span, ctx := tracer.StartSpanFromContext(ctx, "forward-message")

	msg := Message{OriginalText: originalText, ModifiedText: modifiedText}
	body, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Error marshalling message")
		span.Finish(tracer.WithError(err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		logrus.WithError(err).Error("Error creating request")
		span.Finish(tracer.WithError(err))
		return
	}

	// Inject DataDog trace context into headers
	req.Header.Set("Content-Type", "application/json")
	tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error forwarding message")
		span.Finish(tracer.WithError(err))
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"url":    url,
		"status": resp.Status,
	}).Info("Forwarded message")
	span.SetTag("forward.url", url)
	span.SetTag("forward.status", resp.Status)
	span.Finish()
}
