package utils

import (
	"encoding/json"
	"log"
	"strings"
)

func MakeDeterministicJSON(b []byte, flat bool) string {
	var anyJson map[string]interface{}
	err := json.Unmarshal(b, &anyJson)
	if err != nil {
		log.Fatalln(err)
	}
	indent := ""
	if !flat {
		indent = " "
	}
	b2, err := json.MarshalIndent(anyJson, "", indent)
	if err != nil {
		log.Fatalln(err)
	}

	result := string(b2)
	if flat {
		result = strings.ReplaceAll(result, "\n", "")
	}
	return result
}
