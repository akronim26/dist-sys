package utils

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
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
			reply.Pattern = m.Pattern
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

func MakeMaster(files []string) *Master {
	m := Master{}
	m.Tasks = make([]Task, len(files))

	for i, val := range files {
		m.Tasks[i].FileLocation = val
		m.Tasks[i].Id = i + 1
		m.Tasks[i].NReduce = NReduce
		m.Tasks[i].Status = Incomplete
		m.Tasks[i].Type = MapTask
	}

	return &m
}

func (m *Master) Serve(address string) {
	// Resister the master
	rpc.Register(m)
	// Setup the standard RPC handlers
	rpc.HandleHTTP()
	// Listen
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("error while listening: %s", err.Error())
	}
	// Serve
	http.Serve(listener, nil)
}
