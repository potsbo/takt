package task

type Task struct {
	Name             string
	Command          string
	Depends          []string
	DoneNotification []chan TaskNotification
	Waiting          []chan TaskNotification
}

type TaskNotification struct {
	Ok   bool
	Name string
}

func (t *Task) DependsOn(dependedTask *Task) {
	c := make(chan TaskNotification)
	dependedTask.DoneNotification = append(dependedTask.DoneNotification, c)
	t.Waiting = append(t.Waiting, c)
}
