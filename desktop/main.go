// MiMo-Reasonix Desktop - Wails Proof of Concept
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct bound to the frontend.
type App struct {
	ctx    context.Context
	mu     sync.RWMutex
	server *http.Server
	events chan []byte
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		events: make(chan []byte, 64),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("Desktop app started")
}

// shutdown is called when the app quits.
func (a *App) shutdown(ctx context.Context) {
	log.Println("Desktop app shutting down")
	if a.server != nil {
		a.server.Close()
	}
}

// Submit sends user input to the agent via the serve mode API.
func (a *App) Submit(input string) error {
	log.Printf("Submit: %s", input)

	// In a real implementation, this would call the Controller directly
	// or forward to the serve mode API.
	// For now, we'll simulate by emitting an event.
	a.emitEvent(map[string]any{
		"kind": "text",
		"text": "Echo: " + input,
	})

	return nil
}

// Cancel cancels the running turn.
func (a *App) Cancel() error {
	log.Println("Cancel")
	a.emitEvent(map[string]any{
		"kind": "notice",
		"text": "Turn cancelled",
	})
	return nil
}

// Approve approves or denies a tool call.
func (a *App) Approve(id string, allow bool) error {
	log.Printf("Approve: id=%s allow=%t", id, allow)
	return nil
}

// GetHistory returns the session message history.
func (a *App) GetHistory() ([]Message, error) {
	log.Println("GetHistory")
	// Placeholder - would call Controller.History()
	return []Message{
		{Role: "system", Content: "Ready"},
	}, nil
}

// GetStatus returns the session status.
func (a *App) GetStatus() (*Status, error) {
	log.Println("GetStatus")
	return &Status{
		Running: false,
		Plan:    false,
		Label:   "mimo-v2.5",
	}, nil
}

// emitEvent sends an event to the frontend.
func (a *App) emitEvent(event map[string]any) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	// Emit via Wails runtime
	runtime.EventsEmit(a.ctx, "agent:event", string(data))

	// Also send to internal channel
	select {
	case a.events <- data:
	default:
		// Drop if buffer full
	}
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Status represents the session status.
type Status struct {
	Running bool   `json:"running"`
	Plan    bool   `json:"plan"`
	Label   string `json:"label"`
}
