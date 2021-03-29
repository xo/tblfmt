package tblfmt

import (
	"runtime"
	"strconv"
	"strings"
)

// newline is the default newline used by the system.
var newline []byte

func init() {
	if runtime.GOOS == "windows" {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
}

// max returns the max of a, b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// indexOf returns the index of s in v. If s is a integer, then it returns the
// converted value of s. If s is an integer, it needs to be 1-based.
func indexOf(v []string, s string) int {
	s = strings.TrimSpace(s)
	if i, err := strconv.Atoi(s); err == nil {
		i--
		if i >= 0 && i < len(v) {
			return i
		}
		return -1
	}
	for i, vv := range v {
		if strings.EqualFold(s, strings.TrimSpace(vv)) {
			return i
		}
	}
	return -1
}

// findIndex returns the index of s in v.
func findIndex(v []string, s string, i int) int {
	if s == "" {
		if i < len(v) {
			return -1
		}
		return i
	}
	return indexOf(v, s)
}
