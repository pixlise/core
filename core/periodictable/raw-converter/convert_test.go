package rawconverter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Bit wonky but packaged this way because it's easy to run it from the IDE

func Example_convert() {
	r, err := os.ReadFile("rawPeriodicTable.ts")
	if err != nil {
		panic(err)
	}

	startToken := "> = {"
	pos := strings.Index(string(r), startToken)
	if pos == -1 {
		panic("Failed to find start of raw table")
	}

	pos += len(startToken) - 1

	// Fix names
	jsonFixed := ""
	rawJsonLines := strings.Split(string(r)[pos:], "\n")
	lineBlock := false
	for _, line := range rawJsonLines {
		pos = strings.Index(line, "lines: [],")
		if pos > -1 {
			continue
		}
		pos = strings.Index(line, "lines: [")
		if pos > -1 {
			lineBlock = true
		} else {
			pos = strings.Index(line, " ],")
			if pos > -1 {
				lineBlock = false
				continue
			}
		}

		if lineBlock {
			continue
		}

		pos = strings.Index(line, ":")

		if pos == -1 {
			jsonFixed += line + "\n"
		} else {
			if line[pos-1:pos] == "\"" {
				jsonFixed += line + "\n"
			} else {
				// It's a naked field name, dress it up
				start := strings.LastIndex(line[0:pos], " ")
				if start == -1 {
					panic("Bad line start: " + line)
				}
				start++

				symbolPos := strings.Index(line, "symbol: ")
				endIdx := len(line)
				if symbolPos > -1 {
					endIdx-- // snip off the ,
				}

				amendedLine := fmt.Sprintf("%v\"%v\"%v\n", line[0:start], line[start:pos], line[pos:endIdx])
				jsonFixed += amendedLine
			}
		}
	}

	// Fix end
	jsonFixed = jsonFixed[0:len(jsonFixed)-6] + "\n}"

	//fmt.Println(jsonFixed)

	var rawJsonObj map[string]interface{}
	err = json.Unmarshal([]byte(jsonFixed), &rawJsonObj)
	if err != nil {
		panic(err)
	}

	if len(rawJsonObj) != 119 {
		panic(fmt.Sprintf("Not enough elements read: %v", len(rawJsonObj)))
	}

	out := `package periodictable

func FillTable() []PeriodicTableItem {
	result := []PeriodicTableItem{}
`

	for c := 1; c <= 119; c++ {
		itemR, ok := rawJsonObj[fmt.Sprintf("%d", c)]
		if !ok {
			panic(fmt.Sprintf("Item %v not found", c))
		}

		item := itemR.(map[string]interface{})

		out += fmt.Sprintf(`    result = append(result, PeriodicTableItem{Name: "%v", AtomicMass: %v, Z: %v, Symbol: "%v"})`+"\n", item["name"], item["atomic_mass"], item["number"], item["symbol"])
	}

	out += `    return result
}`

	err = os.WriteFile("../filltable.go", []byte(out), 0777)
	if err != nil {
		panic(err)
	}

	// for k, _ := range rawJsonObj {
	// 	fmt.Println(k)
	// }
	fmt.Println(out)

	// Output:
	// <nil>
}
