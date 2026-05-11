package main

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
}

type Worker struct {
	Id     int
	Status WorkerStatus
	Master *Master
}

type Task struct {
	Id           int
	Type         TaskType
	Status       TaskStatus
	FileLocation string
	WorkerId     int
}

type Rpc struct {

}
