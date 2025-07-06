package todo

import (
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/mwopitz/todo-daemon/api/todopb"
)

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

func (t *Task) toProto() *pb.Task {
	return &pb.Task{
		Id:          t.ID,
		Summary:     t.Summary,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
		CompletedAt: timestamppb.New(t.CompletedAt),
	}
}

func (ts Tasks) toDTOs() []taskDTO {
	dtos := make([]taskDTO, len(ts))
	for i := range ts {
		dtos[i].assign(&ts[i])
	}
	return dtos
}

func (ts Tasks) toProtos() []*pb.Task {
	protos := make([]*pb.Task, len(ts))
	for i := range ts {
		protos[i] = ts[i].toProto()
	}
	return protos
}

type taskDTO struct {
	ID          string    `json:"id,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
	CompletedAt time.Time `json:"completed_at,omitzero"`
}

func (dto *taskDTO) assign(t *Task) {
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

func newTaskCreateFromDTO(dto *taskCreateDTO) *TaskCreate {
	return &TaskCreate{
		Summary: dto.Summary,
	}
}

func newTaskCreateFromProto(proto *pb.NewTask) *TaskCreate {
	return &TaskCreate{
		Summary: proto.GetSummary(),
	}
}

// TaskUpdate represents an modification to a task, which can include changing
// the summary or marking the task as completed.
type TaskUpdate struct {
	Summary     *string
	CompletedAt *time.Time
}

type taskUpdateDTO struct {
	Summary     *string    `json:"summary,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func newTaskUpdateFromDTO(dto taskUpdateDTO) *TaskUpdate {
	return &TaskUpdate{
		Summary:     dto.Summary,
		CompletedAt: dto.CompletedAt,
	}
}

func newTaskUpdateFromProto(proto *pb.TaskUpdate, fields *fieldmaskpb.FieldMask) *TaskUpdate {
	u := &TaskUpdate{}
	for _, path := range fields.GetPaths() {
		switch path {
		case "summary":
			summary := proto.GetSummary()
			u.Summary = &summary
		case "completed_at":
			completedAt := proto.GetCompletedAt().AsTime()
			u.CompletedAt = &completedAt
		}
	}
	return u
}
