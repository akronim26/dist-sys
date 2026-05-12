package utils

import (
	"sync"
	"time"
)

type TaskType int

const (
	MapTask = iota
	ReduceTask
)

type TaskStatus int

const (
	Incomplete = iota
	Progressing
	Completed
)

type WorkerStatus int

const (
	Idle = iota
	Working
	Dead
)

type Phase int

const (
	MapPhase = iota
	ReducePhase
)

type Master struct {
	Tasks                []Task
	Mu                   sync.Mutex
	Pattern              string
	Phase                Phase
	MapTasksCompleted    int
	ReduceTasksCompleted int
	MapTasksTotal        int
	NReduce              int
}

type Worker struct {
	Id     int
	Status WorkerStatus
}

type Task struct {
	Id           int
	Type         TaskType
	Status       TaskStatus
	WorkerId     int
	FileLocation string
	StartTime    time.Time
}

type GetTaskArgs struct {
	WorkerId int
}

type GetTaskResults struct {
	TaskId       int
	Type         TaskType
	FileLocation string
	Pattern      string
	NReduce      int
	TaskFound    bool
	Done         bool
}

type TaskDoneResults struct {
	Success bool
}

type TaskDoneArgs struct {
	TaskId   int
	WorkerId int
}

type KeyValue struct {
	Key   string
	Value string
}
