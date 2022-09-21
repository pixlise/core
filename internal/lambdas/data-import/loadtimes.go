package main

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v2/core/fileaccess"
)

type Loaded struct {
	LastLoaded []LastLoaded `json:"last_loaded"`
}
type LastLoaded struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

func saveLoadtime(configBucket string, name string, loads Loaded, fs fileaccess.FileAccess) error {
	var newloads []LastLoaded
	for _, l := range loads.LastLoaded {
		if l.Name == name {
			l.Timestamp = time.Now()
		}
		newloads = append(newloads, l)
	}
	var l = Loaded{newloads}

	return fs.WriteJSONNoIndent(configBucket, "DatasetConfig/lastloaded.json", l)
}

func lookupLoadtime(configBucket string, name string, fs fileaccess.FileAccess) (Loaded, bool) {
	var loads Loaded
	err := fs.ReadJSON(configBucket, "DatasetConfig/lastloaded.json", &loads, false)
	if err != nil {
		// REFACTOR: Return an error? What if this fails, is it bad? Should we use the "return empty if not found" flag above?
		fmt.Println(err)
	}
	for _, r := range loads.LastLoaded {
		if r.Name == name {
			if time.Now().Sub(r.Timestamp).Hours() < 1 {
				return loads, true
			} else {
				return loads, false
			}
		}
	}
	return loads, false
}
