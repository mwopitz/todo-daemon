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

func (c *Controller) respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		c.logger.Printf("cannot write response: %v", err)
	}
}

// CreateTask handles requests to create a new task.
func (c *Controller) CreateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	task, err := c.doCreateTask(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("created task %s: %s", task.ID, task.Summary)
	c.respond(w, http.StatusCreated, task)
}

func (c *Controller) doCreateTask(r *http.Request) (*taskDTO, *restError) {
	dto := &taskCreateDTO{}
	if err := json.NewDecoder(r.Body).Decode(dto); err != nil {
		return nil, newBadRequestError("invalid task", err)
	}
	taskCreate := newTaskCreateFromDTO(dto)
	task, err := c.tasks.Create(r.Context(), taskCreate)
	if err != nil {
		return nil, newInternalServerError("cannot create task", err)
	}
	return task.toDTO(), nil
}

// GetTasks handles the request to retrieve tasks.
func (c *Controller) GetTasks(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	tasks, err := c.doGetTasks(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("retrieved %d tasks", len(tasks))
	c.respond(w, http.StatusOK, tasks)
}

func (c *Controller) doGetTasks(r *http.Request) ([]taskDTO, *restError) {
	tasks, err := c.tasks.All(r.Context())
	if err != nil {
		return nil, newInternalServerError("cannot retrieve tasks", err)
	}
	return tasks.toDTOs(), nil
}

// UpdateTask handles requests to update an existing task.
func (c *Controller) UpdateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	task, err := c.doUpdateTask(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("updated task %s: %s", task.ID, task.Summary)
	c.respond(w, http.StatusOK, task)
}

func (c *Controller) doUpdateTask(r *http.Request) (*taskDTO, *restError) {
	updateDTO := taskUpdateDTO{}
	if err := json.NewDecoder(r.Body).Decode(&updateDTO); err != nil {
		return nil, newBadRequestError("invalid task data", err)
	}
	updateDTO.ID = r.PathValue("id")
	update := newTaskUpdateFromDTO(updateDTO)
	task, err := c.tasks.Update(r.Context(), update)
	if err != nil {
		return nil, newInternalServerError("cannot update task", err)
	}
	return task.toDTO(), nil
}

// DeleteTask handles requests to delete an existing task.
func (c *Controller) DeleteTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	if err := c.doDeleteTask(r); err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.respond(w, http.StatusNoContent, nil)
}

func (c *Controller) doDeleteTask(r *http.Request) *restError {
	err := c.tasks.Delete(r.Context(), r.PathValue("id"))
	if err != nil {
		if IsTaskNotFoundError(err) {
			return newNotFoundError("no such task", err)
		}
		return newInternalServerError("cannot delete task", err)
	}
	return nil
}
