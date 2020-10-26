package task

import "errors"

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Task struct {
	Name             string
	Command          string
	Depends          []string
	DoneNotification []chan TaskNotification
	waiting          []chan TaskNotification
}

type TaskNotification struct {
	Ok   bool
	Name string
}

func (t *Task) DependsOn(dependedTask *Task) {
	c := make(chan TaskNotification)
	dependedTask.DoneNotification = append(dependedTask.DoneNotification, c)
	t.waiting = append(t.waiting, c)
}

func (t Task) WaitDependecies() error {
	okToGo := true
	for _, c := range t.waiting {
		notification := <-c
		if !notification.Ok {
			okToGo = notification.Ok
		}
	}
	if !okToGo {
		return DependencyNotFulfilledErr
	}

	return nil
}
