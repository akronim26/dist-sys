package utils

import "sync"

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

type Master struct {
	Workers []Worker
	Tasks   []Task
	NReduce int
	Mu      sync.Mutex
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
	NReduce      int
}

type GetTaskArgs struct {
	WorkerId int
}

type GetTaskResults struct {
	TaskId       int
	Type         TaskType
	FileLocation string
	NReduce      int
	TaskFound    bool
}

type TaskDoneResults struct {
	Success bool
}

type TaskDoneArgs struct {
	TaskId   int
	WorkerId int
}
