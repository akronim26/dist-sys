package main

import (
	"fmt"
	"mapReduce/utils"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [master|worker] ...")
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	case "master":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run main.go master [pattern] [files...]")
			os.Exit(1)
		}
		pattern := os.Args[2]
		files := os.Args[3:]

		fmt.Printf("Starting Master searching for: %s\n", pattern)
		m := utils.MakeMaster(files, pattern)
		m.Serve(":5000")

	case "worker":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go worker [worker_id]")
			os.Exit(1)
		}
		workerId, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("Invalid worker ID")
			os.Exit(1)
		}

		w := utils.Worker{}
		w.Start(workerId)

	default:
		fmt.Println("Invalid mode. Use 'master' or 'worker'.")
	}
}
