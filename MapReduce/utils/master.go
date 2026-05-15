package utils

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

// RequestTask handles RPC requests from workers for a new task.
// It searches for an incomplete task and assigns it to the requesting worker.
func (m *Master) RequestTask(args *GetTaskArgs, reply *GetTaskResults) error {
	if args == nil {
		return errors.New("arguments cannot be nil")
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	if m.Phase == ReducePhase && m.ReduceTasksCompleted == m.NReduce {
		reply.Done = true
		return nil
	}

	for i := range m.Tasks {
		if m.Tasks[i].Status == Incomplete {
			m.Tasks[i].Status = Progressing
			m.Tasks[i].WorkerId = args.WorkerId
			m.Tasks[i].StartTime = time.Now()

			reply.TaskId = m.Tasks[i].Id
			reply.Type = m.Tasks[i].Type
			reply.FileLocation = m.Tasks[i].FileLocation
			reply.Pattern = m.Pattern
			reply.NReduce = m.NReduce
			reply.TaskFound = true

			return nil
		}
	}

	reply.TaskFound = false
	return nil
}

// ReportTaskDone handles RPC notifications from workers that a task is finished.
// It updates the task status and triggers phase transitions if necessary.
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

			if m.Tasks[i].Type == MapTask {
				m.MapTasksCompleted++
				if m.MapTasksCompleted == m.MapTasksTotal {
					m.transitionToReduce()
				}
			} else {
				m.ReduceTasksCompleted++
			}
			return nil
		}
	}

	reply.Success = false
	return errors.New("task ID not found")
}

// transitionToReduce switches the system from the Map phase to the Reduce phase.
// It generates a new set of tasks for reducers to process.
func (m *Master) transitionToReduce() {
	m.Phase = ReducePhase
	m.Tasks = make([]Task, m.NReduce)
	for i := 0; i < m.NReduce; i++ {
		m.Tasks[i] = Task{
			Id:     i,
			Type:   ReduceTask,
			Status: Incomplete,
		}
	}
	fmt.Println("All Map tasks finished. Transitioning to Reduce phase...")
}

// MakeMaster initializes a new Master instance with the given input files and search pattern.
func MakeMaster(files []string, pattern string) *Master {
	m := Master{
		Pattern:       pattern,
		MapTasksTotal: len(files),
		NReduce:       NReduce,
		Phase:         MapPhase,
	}

	m.Tasks = make([]Task, len(files))
	for i, val := range files {
		m.Tasks[i] = Task{
			Id:           i,
			Type:         MapTask,
			Status:       Incomplete,
			FileLocation: val,
		}
	}

	go m.watcher()

	return &m
}

// Serve starts the RPC server and listens for incoming worker connections.
func (m *Master) Serve(address string) {
	rpc.Register(m)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("error while listening: %s", err.Error())
	}
	fmt.Printf("Master serving on %s\n", address)
	http.Serve(listener, nil)
}

// watcher runs in a separate goroutine and periodically checks for timed-out tasks.
// If a task takes too long, it is reset to Incomplete so another worker can pick it up.
func (m *Master) watcher() {
	for {
		time.Sleep(2 * time.Second)
		m.Mu.Lock()
		for i := range m.Tasks {
			if m.Tasks[i].Status == Progressing && time.Since(m.Tasks[i].StartTime) > time.Duration(TimeoutInterval)*time.Second {
				fmt.Printf("Task %d (Type: %v) timed out. Resetting to Incomplete.\n", m.Tasks[i].Id, m.Tasks[i].Type)
				m.Tasks[i].Status = Incomplete
			}
		}
		m.Mu.Unlock()
	}
}
