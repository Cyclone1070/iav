package tools

import (
	"sync"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

func TestTodoTools(t *testing.T) {
	// Helper to reset state between tests
	resetState := func() {
		todoMutex.Lock()
		sessionTodos = nil
		todoMutex.Unlock()
	}

	t.Run("Happy Path", func(t *testing.T) {
		resetState()
		ctx := &models.WorkspaceContext{}

		// 1. Initial Read should be empty
		readResp, err := ReadTodos(ctx)
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
		writeResp, err := WriteTodos(ctx, todos)
		if err != nil {
			t.Fatalf("WriteTodos failed: %v", err)
		}
		if writeResp.Count != 2 {
			t.Errorf("expected count 2, got %d", writeResp.Count)
		}

		// 3. Read back and verify
		readResp, err = ReadTodos(ctx)
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
		resetState()
		ctx := &models.WorkspaceContext{}

		// Write List A
		listA := []models.Todo{{Description: "A", Status: models.TodoStatusPending}}
		_, err := WriteTodos(ctx, listA)
		if err != nil {
			t.Fatalf("WriteTodos A failed: %v", err)
		}

		// Write List B
		listB := []models.Todo{{Description: "B", Status: models.TodoStatusCompleted}}
		_, err = WriteTodos(ctx, listB)
		if err != nil {
			t.Fatalf("WriteTodos B failed: %v", err)
		}

		// Read should return List B
		readResp, err := ReadTodos(ctx)
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
		resetState()
		ctx := &models.WorkspaceContext{}

		// Write something
		_, _ = WriteTodos(ctx, []models.Todo{{Description: "Task", Status: models.TodoStatusPending}})

		// Write empty
		_, err := WriteTodos(ctx, []models.Todo{})
		if err != nil {
			t.Fatalf("WriteTodos empty failed: %v", err)
		}

		// Read should be empty
		readResp, err := ReadTodos(ctx)
		if err != nil {
			t.Fatalf("ReadTodos failed: %v", err)
		}
		if len(readResp.Todos) != 0 {
			t.Errorf("expected empty list, got %d items", len(readResp.Todos))
		}
	})

	t.Run("Data Isolation", func(t *testing.T) {
		resetState()
		ctx := &models.WorkspaceContext{}

		// Write initial data
		initial := []models.Todo{{Description: "Original", Status: models.TodoStatusPending}}
		_, _ = WriteTodos(ctx, initial)

		// Read data
		readResp, _ := ReadTodos(ctx)

		// Modify returned slice
		readResp.Todos[0].Description = "Modified"

		// Read again - should be original
		readResp2, _ := ReadTodos(ctx)
		if readResp2.Todos[0].Description != "Original" {
			t.Error("ReadTodos returned a reference to internal state, not a copy")
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		resetState()
		ctx := &models.WorkspaceContext{}
		var wg sync.WaitGroup

		// Launch 100 goroutines reading and writing
		for i := range 100 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if id%2 == 0 {
					_, _ = WriteTodos(ctx, []models.Todo{{Description: "Concurrent", Status: models.TodoStatusPending}})
				} else {
					_, _ = ReadTodos(ctx)
				}
			}(i)
		}
		wg.Wait()
	})
}
