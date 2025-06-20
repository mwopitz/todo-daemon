package todo

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"
)

type TaskNotFoundError struct {
	ID string
}

func NewNoSuchTaskError(id string) *TaskNotFoundError {
	return &TaskNotFoundError{ID: id}
}

func (e *TaskNotFoundError) Error() string {
	return fmt.Sprintf("no such task: %s", e.ID)
}

type TaskRepository interface {
	GetTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, task NewTask) (*Task, error)
	UpdateTask(ctx context.Context, update TaskUpdate) (*Task, error)
	DeleteTask(ctx context.Context, id string) error
}

type InMemoryTaskDatabase struct {
	mu sync.Mutex
	tasks map[string]Task
}

func NewInMemoryTaskDatabase() *InMemoryTaskDatabase {
	return &InMemoryTaskDatabase{
		tasks: make(map[string]Task),
	}
}

func (db *InMemoryTaskDatabase) GetTasks(_ context.Context) ([]Task, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return slices.Collect(maps.Values(db.tasks)), nil
}

func (db *InMemoryTaskDatabase) CreateTask(_ context.Context, task NewTask) (*Task, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	t := Task{
		ID: strconv.Itoa(len(db.tasks) + 1),
		Summary: task.Summary,
		CreatedAt: time.Now(),
	}
	db.tasks[t.ID] = t
	return &t, nil
}

func (db *InMemoryTaskDatabase) UpdateTask(_ context.Context, update TaskUpdate) (*Task, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	t, ok := db.tasks[update.ID]
	if !ok {
		return nil, NewNoSuchTaskError(update.ID)
	}
	t.Summary = update.Summary
	db.tasks[t.ID] = t
	return &t, nil
}

func (db *InMemoryTaskDatabase) DeleteTask(_ context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, ok := db.tasks[id]
	if !ok {
		return NewNoSuchTaskError(id)
	}
	delete(db.tasks, id)
	return nil
}