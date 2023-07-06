package wstestlib

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

func ExecQueuedActions(u *ScriptedTestUser) {
	// Program counter doesn't seem useful right now
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "UNKNOWN file"
		line = -1
	} else {
		// dont need the whole path
		file = filepath.Base(file)
	}

	// Run the actions
	fmt.Printf("Running actions [%v (%v)]\n", file, line)

	for {
		running, err := u.RunNextAction()
		if err != nil {

			log.Fatalf("%v (%v): %v\n", file, line, err)
		}
		if !running {
			fmt.Println("Queued actions complete")
			fmt.Printf("-----------------------\n\n")
			break
		}
	}
}
