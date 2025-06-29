package todo

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"
)

// TaskRepository defines functions for querying and persisting [Task]s.
type TaskRepository interface {
	// All retrieves all tasks from the repository.
	All(ctx context.Context) (Tasks, error)
	// Create adds a new task to the repository.
	Create(ctx context.Context, task TaskCreate) (*Task, error)
	// Update modifies an existing task in the repository.
	Update(ctx context.Context, task TaskUpdate) (*Task, error)
	// Delete removes an existing task from the repository.
	Delete(ctx context.Context, id string) error
}

// InMemoryTaskDB is an in-memory implementation of [TaskRepository]. It just
// stores tasks in a map.
type InMemoryTaskDB struct {
	mu    sync.Mutex
	tasks map[string]Task
}

// NewInMemoryTaskDB creates a new instance of [InMemoryTaskDB] with an empty
// map of tasks.
func NewInMemoryTaskDB() *InMemoryTaskDB {
	return &InMemoryTaskDB{
		tasks: make(map[string]Task),
	}
}

// All returns all tasks stored in the task map.
func (db *InMemoryTaskDB) All(_ context.Context) (Tasks, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return slices.Collect(maps.Values(db.tasks)), nil
}

// Create adds a new task to the task map.
func (db *InMemoryTaskDB) Create(_ context.Context, task TaskCreate) (*Task, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	t := Task{
		ID:        strconv.Itoa(len(db.tasks) + 1),
		Summary:   task.Summary,
		CreatedAt: time.Now(),
	}
	db.tasks[t.ID] = t
	return &t, nil
}

// Update modifies an existing task in the task map
func (db *InMemoryTaskDB) Update(_ context.Context, task TaskUpdate) (*Task, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	t, ok := db.tasks[task.ID]
	if !ok {
		return nil, NewTaskNotFoundError(task.ID)
	}
	t.Summary = task.Summary
	db.tasks[t.ID] = t
	return &t, nil
}

// Delete removes a task from the task map by its ID.
func (db *InMemoryTaskDB) Delete(_ context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, ok := db.tasks[id]
	if !ok {
		return NewTaskNotFoundError(id)
	}
	delete(db.tasks, id)
	return nil
}
