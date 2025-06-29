package todo

import (
	"cmp"
	"encoding/json"
	"log"
	"net/http"
)

// Controller handles requests to the REST API endpoints for managing tasks.
type Controller struct {
	logger *log.Logger
	tasks  TaskRepository
}

// NewController creates a new Controller instance with the provided TaskRepository
func NewController(tasks TaskRepository, logger *log.Logger) *Controller {
	return &Controller{
		logger: cmp.Or(logger, log.Default()),
		tasks:  tasks,
	}
}

// CreateTask handles requests to create a new task.
func (c *Controller) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var dto taskCreateDTO
	if err := dec.Decode(&dto); err != nil {
		c.logger.Printf("cannot decode task data: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(newErrorDTO("invalid task data"))
		return
	}

	taskCreate := newTaskCreateFromDTO(dto)
	task, err := c.tasks.Create(ctx, taskCreate)
	if err != nil {
		c.logger.Printf("cannot create task: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(newErrorDTO("cannot create task"))
		return
	}

	w.WriteHeader(http.StatusCreated)
	enc.Encode(task.toDTO())
}

// GetTasks handles the request to retrieve tasks.
func (c *Controller) GetTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	tasks, err := c.tasks.All(ctx)
	if err != nil {
		c.logger.Printf("cannot retrieve tasks: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(newErrorDTO("cannot retrieve tasks"))
		return
	}

	dtos := tasks.toDTOs()

	w.WriteHeader(http.StatusOK)
	enc.Encode(dtos)
}

// UpdateTask handles requests to update an existing task.
func (c *Controller) UpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	updateDTO := taskUpdateDTO{}
	if err := dec.Decode(&updateDTO); err != nil {
		c.logger.Printf("cannot decode task data: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(newErrorDTO("invalid task data"))
		return
	}

	updateDTO.ID = r.PathValue("id")

	update := newTaskUpdateFromDTO(updateDTO)
	task, err := c.tasks.Update(ctx, update)
	if err != nil {
		c.logger.Printf("cannot update task: %v", err)
		if IsTaskNotFoundError(err) {
			w.WriteHeader(http.StatusNotFound)
			enc.Encode(newErrorDTO("cannot find task: %s", update.ID))
		}
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(newErrorDTO("cannot update task"))
		return
	}

	w.WriteHeader(http.StatusOK)
	enc.Encode(task.toDTO())
}
