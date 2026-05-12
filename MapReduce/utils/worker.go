package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"
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

		err = client.Call("Master.RequestTask", args, &reply)
		if err != nil {
			return errors.New("internal error: " + err.Error())
		}
		intermediateFiles := make([][]KeyValue, reply.NReduce)

		if !reply.TaskFound {
			time.Sleep(time.Second)
			continue
		} else if reply.Type == MapTask {
			file, err := os.Open(reply.FileLocation)
			if err != nil {
				return errors.New("error while opening the file")
			}

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

			file.Close()

			for i := 0; i < reply.NReduce; i++ {
				fileName := fmt.Sprintf("mr-%d-%d", reply.TaskId, i)
				file, err := os.Create(fileName)

				if err != nil {
					return errors.New("error while creating file")
				}

				enc := json.NewEncoder(file)
				for _, kv := range intermediateFiles[i] {
					enc.Encode(&kv)
				}

				file.Close()
			}

			err = scanner.Err()
			if err != nil {
				return errors.New("internal error: " + err.Error())
			}

		} else if reply.Type == ReduceTask {
			files, err := filepath.Glob(fmt.Sprintf("mr-*-%d", reply.TaskId))
			if err != nil {
				return errors.New("error while fetching the assigned files")
			}

			var allMatches []string

			for _, val := range files {
				file, err := os.Open(val)
				if err != nil {
					return errors.New("error while opening the file")
				}

				var kv KeyValue

				for {

					err = json.NewDecoder(file).Decode(&kv)

					if err == io.EOF {
						break
					}

					if err != nil {
						return errors.New("error while decoding")
					}

					allMatches = append(allMatches, kv.Value)
				}
				file.Close()

			}
			file, err := os.Create(fmt.Sprintf("mr-final-%d", reply.TaskId))
			if err != nil {
				return errors.New("error while creating file")
			}

			err = os.WriteFile(fmt.Sprintf("mr-final-%d", reply.TaskId), []byte(strings.Join(allMatches, "\n")), 0644)
			if err != nil {
				return errors.New("error while writing the file")
			}

			file.Close()
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
