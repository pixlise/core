package wstestlib

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

func ExecQueuedActions(u *ScriptedTestUser) {
	caller := getCaller(2)

	// Run the actions
	fmt.Printf("Running actions [%v]\n", caller)

	for {
		running, err := u.RunNextAction()
		if err != nil {

			log.Fatalf("%v: %v\n", caller, err)
		}
		if !running {
			fmt.Println("Queued actions complete")
			fmt.Printf("-----------------------\n\n")
			break
		}
	}
}

func getCaller(skip int) string {
	// Program counter doesn't seem useful right now
	_, file, line, ok := runtime.Caller(skip)
	if ok {
		// dont need the whole path
		file = filepath.Base(file)
		return fmt.Sprintf("%v (%v)", file, line)
	}

	return "UNKNOWN_FILE"
}
