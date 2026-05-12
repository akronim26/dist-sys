package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Start initializes the worker and enters a polling loop to request and execute tasks.
func (w *Worker) Start(workerId int) {
	client, err := rpc.DialHTTP("tcp", "localhost:5000")
	if err != nil {
		log.Fatalf("Worker %d failed to connect to master: %v", workerId, err)
	}
	defer client.Close()

	fmt.Printf("Worker %d started.\n", workerId)

	for {
		args := GetTaskArgs{WorkerId: workerId}
		reply := GetTaskResults{}

		err := client.Call("Master.RequestTask", &args, &reply)
		if err != nil {
			fmt.Printf("Worker %d: RPC call failed, exiting: %v\n", workerId, err)
			return
		}

		if reply.Done {
			fmt.Printf("Worker %d: Job complete. Exiting.\n", workerId)
			return
		}

		if !reply.TaskFound {
			time.Sleep(time.Second)
			continue
		}

		if reply.Type == MapTask {
			w.doMap(workerId, &reply)
		} else {
			w.doReduce(workerId, &reply)
		}

		doneArgs := TaskDoneArgs{TaskId: reply.TaskId, WorkerId: workerId}
		doneReply := TaskDoneResults{}
		client.Call("Master.ReportTaskDone", &doneArgs, &doneReply)
	}
}

// doMap performs the map task: it reads an input file, filters lines by pattern,
// and partitions results into intermediate files for reducers.
func (w *Worker) doMap(workerId int, reply *GetTaskResults) {
	fmt.Printf("Worker %d: Starting Map Task %d on %s\n", workerId, reply.TaskId, reply.FileLocation)

	file, err := os.Open(reply.FileLocation)
	if err != nil {
		log.Printf("Worker %d: Failed to open %s: %v", workerId, reply.FileLocation, err)
		return
	}
	defer file.Close()

	intermediate := make([][]KeyValue, reply.NReduce)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, reply.Pattern) {
			kv := KeyValue{Key: reply.Pattern, Value: line}
			bucket := ihash(kv.Key) % reply.NReduce
			intermediate[bucket] = append(intermediate[bucket], kv)
		}
	}

	for i := 0; i < reply.NReduce; i++ {
		oname := fmt.Sprintf("mr-%d-%d", reply.TaskId, i)
		ofile, _ := os.Create(oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range intermediate[i] {
			enc.Encode(&kv)
		}
		ofile.Close()
	}
}

// doReduce performs the reduce task: it gathers intermediate files,
// aggregates matching lines, and writes the final output for its partition.
func (w *Worker) doReduce(workerId int, reply *GetTaskResults) {
	fmt.Printf("Worker %d: Starting Reduce Task %d\n", workerId, reply.TaskId)

	files, err := filepath.Glob(fmt.Sprintf("mr-*-%d", reply.TaskId))
	if err != nil {
		log.Printf("Worker %d: Glob failed: %v", workerId, err)
		return
	}

	var allMatches []string
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err == io.EOF {
				break
			} else if err != nil {
				break
			}
			allMatches = append(allMatches, kv.Value)
		}
		file.Close()
	}

	oname := fmt.Sprintf("mr-out-%d", reply.TaskId)
	outputContent := strings.Join(allMatches, "\n")
	if len(allMatches) > 0 {
		outputContent += "\n"
	}
	err = os.WriteFile(oname, []byte(outputContent), 0644)
	if err != nil {
		log.Printf("Worker %d: Failed to write output %s: %v", workerId, oname, err)
	}
}

// ihash is a helper function to map a string key to a deterministic integer bucket.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}
