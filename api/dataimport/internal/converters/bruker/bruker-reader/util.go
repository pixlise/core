package brukerreader

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// fixDecPatterns matches Bruker's occasional locale bug where decimal
// numbers in the XML use a comma instead of a dot, e.g. ">1,5<".
var fixDecPatterns = regexp.MustCompile(`(>-?\d+),(\d*([Ee]-?\d*)?<)`)

func fixDecCommas(b []byte) []byte {
	return fixDecPatterns.ReplaceAll(b, []byte("${1}.${2}"))
}

// ---------------------------------------------------------------------
// Small generic helpers
// ---------------------------------------------------------------------

// Interpret approximates Python's ast.literal_eval(string) for the limited
// vocabulary Bruker's XML actually uses: integers, floats, booleans, None,
// and otherwise the original string.
func Interpret(v interface{}) interface{} {
	s, ok := v.(string)
	if !ok {
		return v
	}
	return InterpretString(s)
}

func InterpretString(s string) interface{} {
	switch s {
	case "True":
		return true
	case "False":
		return false
	case "None":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

func asFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	case int:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func asInt(v interface{}) int {
	switch t := v.(type) {
	case int64:
		return int(t)
	case int:
		return t
	case float64:
		return int(t)
	case string:
		i, _ := strconv.Atoi(t)
		return i
	default:
		return 0
	}
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func ceilDiv(a, b int) int {
	return int(math.Ceil(float64(a) / float64(b)))
}
