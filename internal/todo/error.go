package todo

import (
	"errors"
	"fmt"
	"net/http"
)

// TaskNotFoundError should be returned by [TaskRepository.Update] and
// [TaskRepository.Delete] when the task with the specified ID does not exist.
type TaskNotFoundError struct {
	// ID is the ID of the task that was not found.
	ID string
}

// NewTaskNotFoundError creates a [TaskNotFoundError] for the task with the
// specified ID.
func NewTaskNotFoundError(id string) *TaskNotFoundError {
	return &TaskNotFoundError{ID: id}
}

// IsTaskNotFoundError checks if the provided error is a [TaskNotFoundError].
func IsTaskNotFoundError(err error) bool {
	var e *TaskNotFoundError
	return err != nil && errors.As(err, &e)
}

func (e *TaskNotFoundError) Error() string {
	return fmt.Sprintf("no such task: %s", e.ID)
}

type restError struct {
	cause   error
	status  int
	Message string `json:"message"`
}

func (e *restError) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("%s: %s", e.Message, e.cause.Error())
	}
	return e.Message
}

func (e *restError) Cause() error {
	return e.cause
}

func newBadRequestError(msg string, cause error) *restError {
	return &restError{cause, http.StatusBadRequest, msg}
}

func newInternalServerError(msg string, cause error) *restError {
	return &restError{cause, http.StatusInternalServerError, msg}
}
