package todo

type TaskService struct {
	tasks TaskRepository
}

func NewTaskService(tasks TaskRepository) *TaskService {
	return &TaskService {
		tasks: tasks,
	}
}
