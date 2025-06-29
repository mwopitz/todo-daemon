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
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		c.logger.Printf("cannot write response: %v", err)
	}
}

// CreateTask handles requests to create a new task.
func (c *Controller) CreateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling request %s %s", r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	task, err := c.doCreateTask(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("created task %s", task.ID)

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
	c.logger.Printf("handling request %s %s", r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	tasks, err := c.doGetTasks(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
	}

	w.WriteHeader(http.StatusOK)

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
	c.logger.Printf("handling request %s %s", r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	task, err := c.doUpdateTask(r)
	if err != nil {
		c.logger.Println(err)
		w.WriteHeader(err.status)
		if e := json.NewEncoder(w).Encode(err); e != nil {
			c.logger.Printf("cannot write response: %v", e)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if e := json.NewEncoder(w).Encode(task); e != nil {
		c.logger.Printf("cannot write response: %v", e)
	}
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
