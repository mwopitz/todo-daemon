package todo

import (
	"context"
	"errors"
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
	Create(ctx context.Context, task *TaskCreate) (*Task, error)
	// Update modifies an existing task in the repository. If the task does not
	// exist, it returns a [TaskNotFoundError].
	Update(ctx context.Context, id string, update *TaskUpdate, fields FieldMask) (*Task, error)
	// Delete removes an existing task from the repository. If the task does not
	// exist, it returns a [TaskNotFoundError].
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
	tasks := slices.Collect(maps.Values(db.tasks))
	// Sort by creation time in ascending order.
	slices.SortFunc(tasks, func(a, b Task) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return tasks, nil
}

// Create adds a new task to the task map.
func (db *InMemoryTaskDB) Create(_ context.Context, task *TaskCreate) (*Task, error) {
	if task == nil {
		return nil, errors.New("task cannot be nil")
	}
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
func (db *InMemoryTaskDB) Update(_ context.Context, id string, update *TaskUpdate, fields FieldMask) (*Task, error) {
	if update == nil {
		return nil, errors.New("update cannot be nil")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	t, ok := db.tasks[id]
	if !ok {
		return nil, NewTaskNotFoundError(id)
	}
	now := time.Now()
	if slices.Contains(fields, "summary") {
		t.Summary = update.Summary
		t.UpdatedAt = now
	}
	if slices.Contains(fields, "completed_at") {
		t.CompletedAt = update.CompletedAt
		t.UpdatedAt = now
	}
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
