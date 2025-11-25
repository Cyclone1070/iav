package tools

import (
	"sync"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

func TestTodoTools(t *testing.T) {
	// Helper to create a context with a fresh store
	createContext := func() *models.WorkspaceContext {
		return &models.WorkspaceContext{
			TodoStore: NewInMemoryTodoStore(),
		}
	}

	t.Run("Happy Path", func(t *testing.T) {
		ctx := createContext()

		// 1. Initial Read should be empty
		readResp, err := ReadTodos(ctx, models.ReadTodosRequest{})
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected empty todos, got %d", len(readResp.Todos))
		}

		// 2. Write some todos
		todos := []models.Todo{
			{Description: "Task 1", Status: models.TodoStatusPending},
			{Description: "Task 2", Status: models.TodoStatusInProgress},
		}
		writeResp, err := WriteTodos(ctx, models.WriteTodosRequest{Todos: todos})
		if err != nil {
			t.Fatalf("WriteTodos failed: %v", err)
		}
		if writeResp.Count != 2 {
			t.Errorf("expected count 2, got %d", writeResp.Count)
		}

		// 3. Read back and verify
		readResp, err = ReadTodos(ctx, models.ReadTodosRequest{})
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 2 {
			t.Fatalf("expected 2 todos, got %d", len(readResp.Todos))
		}
		if readResp.Todos[0].Description != "Task 1" {
			t.Errorf("expected Task 1, got %s", readResp.Todos[0].Description)
		}
		if readResp.Todos[1].Status != models.TodoStatusInProgress {
			t.Errorf("expected InProgress, got %s", readResp.Todos[1].Status)
		}
	})

	t.Run("Overwrite", func(t *testing.T) {
		ctx := createContext()

		// Write List A
		listA := []models.Todo{{Description: "A", Status: models.TodoStatusPending}}
		_, err := WriteTodos(ctx, models.WriteTodosRequest{Todos: listA})
		if err != nil {
			t.Fatalf("WriteTodos A failed: %v", err)
		}

		// Write List B
		listB := []models.Todo{{Description: "B", Status: models.TodoStatusCompleted}}
		_, err = WriteTodos(ctx, models.WriteTodosRequest{Todos: listB})
		if err != nil {
			t.Fatalf("WriteTodos B failed: %v", err)
		}

		// Read should return List B
		readResp, err := ReadTodos(ctx, models.ReadTodosRequest{})
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
		ctx := createContext()

		// Write something
		_, _ = WriteTodos(ctx, models.WriteTodosRequest{Todos: []models.Todo{{Description: "Task", Status: models.TodoStatusPending}}})

		// Write empty
		_, err := WriteTodos(ctx, models.WriteTodosRequest{Todos: []models.Todo{}})
		if err != nil {
			t.Fatalf("WriteTodos empty failed: %v", err)
		}

		// Read should be empty
		readResp, err := ReadTodos(ctx, models.ReadTodosRequest{})
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected empty list, got %d items", len(readResp.Todos))
		}
	})

	t.Run("Data Isolation", func(t *testing.T) {
		ctx := createContext()

		// Write initial data
		initial := []models.Todo{{Description: "Original", Status: models.TodoStatusPending}}
		_, _ = WriteTodos(ctx, models.WriteTodosRequest{Todos: initial})

		// Read data
		readResp, _ := ReadTodos(ctx, models.ReadTodosRequest{})

		// Modify returned slice
		readResp.Todos[0].Description = "Modified"

		// Read again - should be original
		readResp2, _ := ReadTodos(ctx, models.ReadTodosRequest{})
		if readResp2.Todos[0].Description != "Original" {
			t.Error("ReadTodos returned a reference to internal state, not a copy")
		}
	})

	t.Run("Context Isolation", func(t *testing.T) {
		// Verify that two different contexts have different stores
		ctx1 := createContext()
		ctx2 := createContext()

		// Write to ctx1
		_, err := WriteTodos(ctx1, models.WriteTodosRequest{Todos: []models.Todo{
			{Description: "Ctx1", Status: models.TodoStatusPending},
		}})

		// Read from ctx2 - should be empty
		readResp, err := ReadTodos(ctx2, models.ReadTodosRequest{})
		if err != nil {
			t.Fatalf("ReadTodos ctx2 failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected ctx2 to be empty, got %d items", len(readResp.Todos))
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		ctx := createContext()
		var wg sync.WaitGroup

		// Launch 100 goroutines reading and writing
		for i := range 100 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if id%2 == 0 {
					_, _ = WriteTodos(ctx, models.WriteTodosRequest{Todos: []models.Todo{{Description: "Concurrent", Status: models.TodoStatusPending}}})
				} else {
					_, _ = ReadTodos(ctx, models.ReadTodosRequest{})
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Missing Store", func(t *testing.T) {
		ctx := &models.WorkspaceContext{} // No store initialized

		_, err := ReadTodos(ctx, models.ReadTodosRequest{})
		if err == nil {
			t.Error("expected error when reading with missing store, got nil")
		}

		_, err = WriteTodos(ctx, models.WriteTodosRequest{Todos: []models.Todo{}})
		if err == nil {
			t.Error("expected error when writing with missing store, got nil")
		}
	})
}
