package tblfmt

import (
	"fmt"
	"io"
	"runtime"
)

const lowerhex = "0123456789abcdef"

// newline is the default newline used by the system.
var newline []byte

func init() {
	if runtime.GOOS == "windows" {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
}

// Error is an error.
type Error string

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Error values.
const (
	// ErrResultSetIsNil is the result set is nil error.
	ErrResultSetIsNil Error = "result set is nil"

	// ErrResultSetHasNoColumns is the result set has no columns error.
	ErrResultSetHasNoColumns Error = "result set has no columns"

	// ErrInvalidLineStyle is the invalid line style error.
	ErrInvalidLineStyle Error = "invalid line style"

	// ErrInvalidStyleName is the invalid style name error.
	ErrInvalidStyleName Error = "invalid style name"
)

// DefaultTableSummary is the default table summary.
func DefaultTableSummary() map[int]func(io.Writer, int) (int, error) {
	return map[int]func(io.Writer, int) (int, error){
		1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d row)", count)
		},
		-1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d rows)", count)
		},
	}
}

// max returns the max of a, b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
