package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/pixlise/core/v4/core/utils"
)

type importTask struct {
	id        string
	params    string
	resultErr error
	complete  bool
}

var importTaskMutex sync.Mutex
var importTasks = map[string]importTask{}
var importTaskErrorCount = 0

func addImportTask(params string) string {
	id := utils.RandStringBytesMaskImpr(10)

	importTaskMutex.Lock()
	defer importTaskMutex.Unlock()

	importTasks[id] = importTask{id: id, params: params, resultErr: nil, complete: false}
	return id
}

func finishImportTask(id string, err error) {
	importTaskMutex.Lock()
	defer importTaskMutex.Unlock()

	task, ok := importTasks[id]
	if !ok {
		fatalError(fmt.Errorf("Failed to find import task: %v", id))
	}

	// mark this task as complete
	//delete(importTasks, id)
	task.complete = true
	task.resultErr = err

	if err != nil {
		importTaskErrorCount++
		printTaskError(importTaskErrorCount, task)
	} else {
		fmt.Printf("Task (of total %v) finished: %v\n", len(importTasks), task.params)
	}
}

func reportFailedTasks() {
	importTaskMutex.Lock()
	defer importTaskMutex.Unlock()

	fmt.Printf("Total tasks: %v. Failures: %v\n", len(importTasks), importTaskErrorCount)
	fmt.Printf("List of all errors:\n")

	errCount := 0
	for _, task := range importTasks {
		if task.resultErr != nil {
			errCount++
			printTaskError(errCount, task)
		}
	}
}

func printTaskError(idx int, task importTask) {
	log.Printf("==================\nImport Task Failed\n------------------\nTask (%v of %v): %v\nError: %v\n==================\n", idx, len(importTasks), task.params, task.resultErr)
}
