package todo

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestTodoTools(t *testing.T) {
	cfg := config.DefaultConfig()

	// Helper to create tools with a fresh store
	createTools := func() (*ReadTodosTool, *WriteTodosTool, *InMemoryTodoStore) {
		store := NewInMemoryTodoStore()
		readTool := NewReadTodosTool(store, cfg)
		writeTool := NewWriteTodosTool(store, cfg)
		return readTool, writeTool, store
	}

	t.Run("Happy Path", func(t *testing.T) {
		readTool, writeTool, _ := createTools()

		// 1. Initial Read should be empty
		req := &ReadTodosRequest{}
		readResp, err := readTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected empty todos, got %d", len(readResp.Todos))
		}

		// 2. Write some todos
		todos := []Todo{
			{Description: "Task 1", Status: TodoStatusPending},
			{Description: "Task 2", Status: TodoStatusInProgress},
		}
		writeReq := &WriteTodosRequest{Todos: todos}
		writeResp, err := writeTool.Run(context.Background(), writeReq)
		if err != nil {
			t.Fatalf("WriteTodos failed: %v", err)
		}
		if writeResp.Count != 2 {
			t.Errorf("expected count 2, got %d", writeResp.Count)
		}

		// 3. Read back and verify
		req = &ReadTodosRequest{}
		readResp, err = readTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 2 {
			t.Fatalf("expected 2 todos, got %d", len(readResp.Todos))
		}
		if readResp.Todos[0].Description != "Task 1" {
			t.Errorf("expected Task 1, got %s", readResp.Todos[0].Description)
		}
		if readResp.Todos[1].Status != TodoStatusInProgress {
			t.Errorf("expected InProgress, got %s", readResp.Todos[1].Status)
		}
	})

	t.Run("Overwrite", func(t *testing.T) {
		readTool, writeTool, _ := createTools()

		// Write List A
		listA := []Todo{{Description: "A", Status: TodoStatusPending}}
		writeReq := &WriteTodosRequest{Todos: listA}
		_, err := writeTool.Run(context.Background(), writeReq)
		if err != nil {
			t.Fatalf("WriteTodos A failed: %v", err)
		}

		// Write List B
		listB := []Todo{{Description: "B", Status: TodoStatusCompleted}}
		writeReq = &WriteTodosRequest{Todos: listB}
		_, err = writeTool.Run(context.Background(), writeReq)
		if err != nil {
			t.Fatalf("WriteTodos B failed: %v", err)
		}

		// Read should return List B
		req := &ReadTodosRequest{}
		readResp, err := readTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 1 {
			t.Fatalf("expected 1 todo, got %d", len(readResp.Todos))
		}
		if readResp.Todos[0].Description != "B" {
			t.Errorf("expected B, got %s", readResp.Todos[0].Description)
		}
	})

	t.Run("Empty Write Clears", func(t *testing.T) {
		readTool, writeTool, _ := createTools()

		// Write something
		writeReq := &WriteTodosRequest{Todos: []Todo{{Description: "Task", Status: TodoStatusPending}}}
		_, _ = writeTool.Run(context.Background(), writeReq)

		// Write empty
		writeReq = &WriteTodosRequest{Todos: []Todo{}}
		_, err := writeTool.Run(context.Background(), writeReq)
		if err != nil {
			t.Fatalf("WriteTodos empty failed: %v", err)
		}

		// Read should be empty
		readReq := &ReadTodosRequest{}
		readResp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected empty list, got %d items", len(readResp.Todos))
		}
	})

	t.Run("Data Isolation", func(t *testing.T) {
		readTool, writeTool, _ := createTools()

		// Write initial data
		initial := []Todo{{Description: "Original", Status: TodoStatusPending}}
		writeReq := &WriteTodosRequest{Todos: initial}
		_, _ = writeTool.Run(context.Background(), writeReq)

		// Read data
		readReq := &ReadTodosRequest{}
		readResp, _ := readTool.Run(context.Background(), readReq)

		// Modify returned slice
		readResp.Todos[0].Description = "Modified"

		// Read again - should be original
		readReq2 := &ReadTodosRequest{}
		readResp2, _ := readTool.Run(context.Background(), readReq2)
		if readResp2.Todos[0].Description != "Original" {
			t.Error("ReadTodos returned a reference to internal state, not a copy")
		}
	})

	t.Run("Context Isolation", func(t *testing.T) {
		// Verify that two different tool instances with different stores are isolated
		_, writeTool1, _ := createTools()
		readTool2, _, _ := createTools()

		// Write to store 1
		writeReq := &WriteTodosRequest{Todos: []Todo{
			{Description: "Ctx1", Status: TodoStatusPending},
		}}
		_, err := writeTool1.Run(context.Background(), writeReq)
		if err != nil {
			t.Fatalf("WriteTodos ctx1 failed: %v", err)
		}

		// Read from store 2 - should be empty
		readReq := &ReadTodosRequest{}
		readResp, err := readTool2.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("ReadTodos ctx2 failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected ctx2 to be empty, got %d items", len(readResp.Todos))
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		readTool, writeTool, _ := createTools()
		var wg sync.WaitGroup

		// Launch 100 goroutines reading and writing
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if id%2 == 0 {
					writeReq := &WriteTodosRequest{Todos: []Todo{{Description: "Concurrent", Status: TodoStatusPending}}}
					_, _ = writeTool.Run(context.Background(), writeReq)
				} else {
					readReq := &ReadTodosRequest{}
					_, _ = readTool.Run(context.Background(), readReq)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Missing Store", func(t *testing.T) {
		// Create tools with nil store
		readTool := NewReadTodosTool(nil, cfg)
		writeTool := NewWriteTodosTool(nil, cfg)

		readReq := &ReadTodosRequest{}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil || !errors.Is(err, ErrStoreNotConfigured) {
			t.Errorf("expected ErrStoreNotConfigured when reading with missing store, got %v", err)
		}

		writeReq := &WriteTodosRequest{Todos: []Todo{}}
		_, err = writeTool.Run(context.Background(), writeReq)
		if err == nil || !errors.Is(err, ErrStoreNotConfigured) {
			t.Errorf("expected ErrStoreNotConfigured when writing with missing store, got %v", err)
		}
	})
}
