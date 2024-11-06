package sdfToRSI

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Returns a map of RTTs, with the line they are found
type EventEntry struct {
	Line  int
	What  string
	Value string
}

func scanSDF(sdfPath string) ([]EventEntry, error) {
	refs := []EventEntry{}
	file, err := os.Open(sdfPath)
	if err != nil {
		return refs, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	rttMap := map[int64]bool{}

	lineNo := 0
	firstTimeStamp := int64(0)
	maxTimeStamp := int64(0)

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		// If we haven't found the start yet, keep looking
		if strings.Trim(line, " ") == ":: SDF_Peek complete" {
			if len(refs) > 0 && refs[0].What == "start" {
				return refs, fmt.Errorf("Found duplicate start at line %v", lineNo)
			} else {
				refs = append(refs, EventEntry{Line: lineNo, What: "start", Value: ""})
				continue
			}
		}

		// If we haven't started reading the file yet, stop here
		if len(refs) <= 0 {
			continue
		}

		// Check the time stamp, we're ignoring th ones starting with 2000
		tok, lineData, ok := takeToken(line, " : ")
		if !ok || len(tok) <= 0 {
			return refs, fmt.Errorf("Expected timestamp at start of line %v", lineNo)
		}

		// Ignore startup timestamps
		if strings.HasPrefix(tok, "2000-") {
			continue
		}

		// Valid time stamp, see if it's the first we're reading...
		ts := readTimestamp(tok)

		if firstTimeStamp == 0 {
			// Must be the first time stamp
			refs = append(refs, EventEntry{Line: lineNo, What: "first-time", Value: tok})
			firstTimeStamp = ts
		} else {
			// Lets make sure this is incrementing
			if ts < maxTimeStamp {
				// Nope, this time stamp is older than what we recently read
				return refs, fmt.Errorf("Timestamp is not incremental line %v", lineNo)
			}
		}

		if ts > maxTimeStamp {
			maxTimeStamp = ts
		}

		// See if there's an RTT on this line, if so, note what line it starts on
		tok, ok = findToken(lineData, " RTT: ", " ")
		if ok && len(tok) > 0 && !strings.HasPrefix(tok, "0x") {
			// Read the RTT as hex number
			thisRTT, err := strconv.ParseInt(tok, 16, 32)
			if err != nil {
				return refs, fmt.Errorf("Failed to read RTT from line %v: \"%v\". Error: %v", lineNo, line, err)
			}

			if thisRTT > 0 {
				if !rttMap[thisRTT] {
					// First mention of this RTT
					rttMap[thisRTT] = true
					refs = append(refs, EventEntry{Line: lineNo, What: "new-rtt", Value: tok})
				}
			}
		}

		if strings.Contains(lineData, "\"Science Placement\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "science", Value: "begin"})
		}

		if strings.Contains(lineData, "termination of Science Placement\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "science", Value: "end"})
		}

		if strings.Contains(lineData, "Open the Dust Cover\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "dust-cover", Value: "opening"})
		}

		if strings.Contains(lineData, "Close the Dust Cover\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "dust-cover", Value: "closing"})
		}

		if strings.Contains(lineData, "Termination of Cover Open\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "dust-cover", Value: "opened"})
		}

		if strings.Contains(lineData, "Cover Close termination\"") {
			refs = append(refs, EventEntry{Line: lineNo, What: "dust-cover", Value: "closed"})
		}

		sciPlace := "Sci_Place: "
		pos := strings.Index(lineData, sciPlace)
		if pos > -1 {
			lineData = lineData[pos+len(sciPlace):]
			refs = append(refs, EventEntry{Line: lineNo, What: "sci-place", Value: lineData})
		}
	}

	return refs, nil
}
