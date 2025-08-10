package todo

import (
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
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

func (t *Task) toProto() *todopb.Task {
	return &todopb.Task{
		Id:          t.ID,
		Summary:     t.Summary,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
		CompletedAt: timestamppb.New(t.CompletedAt),
	}
}

func (ts Tasks) toProtos() []*todopb.Task {
	protos := make([]*todopb.Task, len(ts))
	for i := range ts {
		protos[i] = ts[i].toProto()
	}
	return protos
}

// TaskCreate encapsulates the data needed to create a new task.
type TaskCreate struct {
	// Summary is a concise description of the task.
	Summary string
}

func newTaskCreateFromProto(proto *todopb.NewTask) *TaskCreate {
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

func newTaskUpdateFromProto(proto *todopb.TaskUpdate, fields *fieldmaskpb.FieldMask) *TaskUpdate {
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
