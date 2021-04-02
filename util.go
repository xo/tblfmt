package tblfmt

import (
	"runtime"
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
