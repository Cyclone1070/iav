//go:build integration

package ui

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	uiservices "github.com/Cyclone1070/iav/internal/ui/services"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
)

func TestUIOrchestrator_MessageFlow(t *testing.T) {
	t.Parallel()

	// Create UI with real channels
	channels := NewUIChannels()
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	ui := NewUI(channels, renderer, spinnerFactory)

	// Track if message was received
	var receivedMsg string
	var msgReceived bool
	var mu sync.Mutex

	// Start goroutine listening to message channel
	done := make(chan bool)
	go func() {
		for msg := range channels.MessageChan {
			mu.Lock()
			receivedMsg = msg
			msgReceived = true
			mu.Unlock()
			done <- true
			return
		}
	}()

	// Call WriteMessage from main thread
	channelSendCompleted := false
	ui.WriteMessage("Task complete")
	channelSendCompleted = true

	// Message received
	select {
	case <-done:
		mu.Lock()
		assert.Equal(t, "Task complete", receivedMsg)
		mu.Unlock()
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for message")
	}

	// Channel not blocked
	assert.True(t, channelSendCompleted, "WriteMessage should not block")
	assert.True(t, msgReceived, "Message should have been received")
}

func TestUIOrchestrator_InputRequest(t *testing.T) {
	t.Parallel()

	// Create UI with real channels
	channels := NewUIChannels()
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	ui := NewUI(channels, renderer, spinnerFactory)

	// Start ReadInput in goroutine
	resultChan := make(chan string)
	errChan := make(chan error)
	ctx := context.Background()

	go func() {
		result, err := ui.ReadInput(ctx, "What next?")
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for request on inputReq channel
	select {
	case req := <-channels.InputReq:
		assert.Equal(t, "What next?", req.Prompt)
		// Send response
		channels.InputResp <- "Run tests"
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not receive input request")
	}

	// Response delivered
	select {
	case result := <-resultChan:
		assert.Equal(t, "Run tests", result)
	case err := <-errChan:
		t.Fatalf("ReadInput returned error: %v", err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for result")
	}
}

func TestUIOrchestrator_Deadlock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock test in short mode")
	}

	initialGoroutines := runtime.NumGoroutine()

	// Create UI with real channels
	channels := NewUIChannels()
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	ui := NewUI(channels, renderer, spinnerFactory)

	// Service input requests in background
	go func() {
		for req := range channels.InputReq {
			channels.InputResp <- "response"
			_ = req
		}
	}()

	// Also consume messages
	go func() {
		for range channels.MessageChan {
			// Consume
		}
	}()

	// Spawn 200 goroutines
	const numOps = 200
	allDone := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(numOps)

	// Half call WriteMessage, half call ReadInput
	for i := 0; i < numOps; i++ {
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				ui.WriteMessage("test message")
			}()
		} else {
			go func() {
				defer wg.Done()
				_, _ = ui.ReadInput(context.Background(), "test prompt")
			}()
		}
	}

	go func() {
		wg.Wait()
		allDone <- true
	}()

	// All operations complete
	select {
	case <-allDone:
		assert.True(t, true, "All operations completed")
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected - operations timed out")
	}

	// Allow cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// No goroutine leaks (allow some margin for test goroutines)
	finalGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+5, "Should not have significant goroutine leaks")
}
