package todo

import "time"

type Task struct {
	ID string
	Summary string
	CreatedAt time.Time
	UpdatedAt time.Time
	CompletedAt time.Time
	DeletedAt time.Time
}

type NewTask struct {
	Summary string
}

type TaskUpdate struct {
	ID string
	Summary string
}
