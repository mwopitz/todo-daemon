package todo

import "time"

// Task represents a single to-do item.
type Task struct {
	ID          string
	Summary     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt time.Time
	DeletedAt   time.Time
}

// Tasks is a list of to-do items.
type Tasks []Task

func (t *Task) toDTO() *taskDTO {
	return &taskDTO{
		ID:          t.ID,
		Summary:     t.Summary,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}
}

func (ts Tasks) toDTOs() []taskDTO {
	dtos := make([]taskDTO, len(ts))
	for i, t := range ts {
		dtos[i].assign(t)
	}
	return dtos
}

type taskDTO struct {
	ID          string    `json:"id,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
	CompletedAt time.Time `json:"completed_at,omitzero"`
}

func (dto *taskDTO) assign(t Task) {
	dto.ID = t.ID
	dto.Summary = t.Summary
	dto.CreatedAt = t.CreatedAt
	dto.UpdatedAt = t.UpdatedAt
	dto.CompletedAt = t.CompletedAt
}

// TaskCreate encapsulates the data needed to create a new task.
type TaskCreate struct {
	// Summary is a concise description of the task.
	Summary string
}

type taskCreateDTO struct {
	Summary string `json:"summary"`
}

func newTaskCreateFromDTO(dto taskCreateDTO) TaskCreate {
	return TaskCreate(dto)
}

// TaskUpdate represents an modification to a task, which can include changing
// the summary or marking the task as completed.
type TaskUpdate struct {
	ID          string
	Summary     string
	CompletedAt time.Time
}

type taskUpdateDTO struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitzero"`
}

func newTaskUpdateFromDTO(dto taskUpdateDTO) TaskUpdate {
	return TaskUpdate(dto)
}
