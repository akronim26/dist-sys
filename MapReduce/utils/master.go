package utils

import (
	"errors"
)

func (m *Master) RequestTask(args *GetTaskArgs, reply *GetTaskResults) error {
	if args == nil {
		return errors.New("arguments cannot be nil")
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	for i := range m.Tasks {
		if m.Tasks[i].Status == Incomplete {
			m.Tasks[i].Status = Progressing
			m.Tasks[i].WorkerId = args.WorkerId

			reply.TaskId = m.Tasks[i].Id
			reply.Type = m.Tasks[i].Type
			reply.FileLocation = m.Tasks[i].FileLocation
			reply.NReduce = m.NReduce
			reply.TaskFound = true

			return nil
		}
	}

	reply.TaskFound = false
	return nil
}
func (m *Master) ReportTaskDone(args *TaskDoneArgs, reply *TaskDoneResults) error {
	if args == nil {
		return errors.New("arguments cannot be nil")
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	for i := range m.Tasks {
		if m.Tasks[i].Id == args.TaskId {
			m.Tasks[i].Status = Completed
			m.Tasks[i].WorkerId = 0
			reply.Success = true
			return nil
		}
	}

	reply.Success = false
	return errors.New("task ID not found")
}
