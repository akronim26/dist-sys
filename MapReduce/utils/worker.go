package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/rpc"
	"os"
	"strings"
)

func (w *Worker) Start(workerId int) error {
	client, err := rpc.DialHTTP("tcp", ":5000")
	if err != nil {
		return errors.New("failed to connect with client")
	}

	defer client.Close()

	for {
		args := GetTaskArgs{
			WorkerId: workerId,
		}

		reply := GetTaskResults{}
		intermediateFiles := make([][]KeyValue, reply.NReduce)

		err = client.Call("Master.RequestTask", args, &reply)
		if err != nil {
			return errors.New("internal error: " + err.Error())
		}

		if reply.Type == MapTask {
			file, err := os.Open(reply.FileLocation)
			if err != nil {
				return errors.New("error while opening the file")
			}

			defer file.Close()

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, reply.Pattern) {
					kv := KeyValue{
						Key:   reply.Pattern,
						Value: line,
					}

					bucketIndex := ihash(kv.Key) % reply.NReduce
					intermediateFiles[bucketIndex] = append(intermediateFiles[bucketIndex], kv)
				}
			}

			for i := 0; i < reply.NReduce; i++ {
				fileName := fmt.Sprintf("mr-%d-%d", reply.TaskId, i)
				file, err := os.Create(fileName)

				if err != nil {
					return errors.New("error while creating file")
				}
				defer file.Close()

				enc := json.NewEncoder(file)
				for _, kv := range intermediateFiles[i] {
					enc.Encode(&kv)
				}
			}

			err = scanner.Err()
			if err != nil {
				return errors.New("internal error: " + err.Error())
			}

		}

		taskDoneArgs := TaskDoneArgs{
			TaskId:   reply.TaskId,
			WorkerId: workerId,
		}

		taskDoneResults := TaskDoneResults{
			Success: false,
		}

		err = client.Call("Master.ReportTaskDone", taskDoneArgs, &taskDoneResults)
		if err != nil || !taskDoneResults.Success {
			return errors.New("failed to report task as done")
		}

	}
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}
