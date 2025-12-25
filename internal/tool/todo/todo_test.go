package todo

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

// mockTodoStoreWithErrors is a local mock for testing store errors
type mockTodoStoreWithErrors struct {
	readErr  error
	writeErr error
}

func (m *mockTodoStoreWithErrors) Read() ([]Todo, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return []Todo{}, nil
}

func (m *mockTodoStoreWithErrors) Write(todos []Todo) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	return nil
}

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
		for i := range 100 {
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
}

func TestTodoValidation(t *testing.T) {
	cfg := config.DefaultConfig()
	store := NewInMemoryTodoStore()
	writeTool := NewWriteTodosTool(store, cfg)

	t.Run("InvalidStatus", func(t *testing.T) {
		req := &WriteTodosRequest{
			Todos: []Todo{{Description: "Foo", Status: "unknown"}},
		}
		_, err := writeTool.Run(context.Background(), req)
		if err == nil {
			t.Error("expected error for invalid status")
		}
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("expected ErrInvalidStatus, got %v", err)
		}
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		req := &WriteTodosRequest{
			Todos: []Todo{{Description: "", Status: TodoStatusPending}},
		}
		_, err := writeTool.Run(context.Background(), req)
		if err == nil {
			t.Error("expected error for empty description")
		}
		if !errors.Is(err, ErrEmptyDescription) {
			t.Errorf("expected ErrEmptyDescription, got %v", err)
		}
	})

	t.Run("AllValidStatuses", func(t *testing.T) {
		statuses := []TodoStatus{
			TodoStatusPending,
			TodoStatusInProgress,
			TodoStatusCompleted,
			TodoStatusCancelled,
		}
		for _, s := range statuses {
			req := &WriteTodosRequest{
				Todos: []Todo{{Description: "Valid", Status: s}},
			}
			_, err := writeTool.Run(context.Background(), req)
			if err != nil {
				t.Errorf("expected success for status %s, got %v", s, err)
			}
		}
	})
}

func TestTodoStoreErrors(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("ReadError", func(t *testing.T) {
		mockStore := &mockTodoStoreWithErrors{readErr: errors.New("read failed")}
		readTool := NewReadTodosTool(mockStore, cfg)

		resp, err := readTool.Run(context.Background(), &ReadTodosRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Todos) != 0 {
			t.Errorf("expected empty todos on read error, got %d", len(resp.Todos))
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		mockStore := &mockTodoStoreWithErrors{writeErr: errors.New("write failed")}
		writeTool := NewWriteTodosTool(mockStore, cfg)

		_, err := writeTool.Run(context.Background(), &WriteTodosRequest{Todos: []Todo{}})
		if err == nil {
			t.Error("expected error for store write failure")
		}
	})
}
