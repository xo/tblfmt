package tblfmt

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
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
	// ErrInvalidFormat is the invalid format error.
	ErrInvalidFormat Error = "invalid format"
	// ErrInvalidLineStyle is the invalid line style error.
	ErrInvalidLineStyle Error = "invalid line style"
	// ErrInvalidTemplate is the invalid template error.
	ErrInvalidTemplate Error = "invalid template"
	// ErrInvalidFieldSeparator is the invalid field separator error.
	ErrInvalidFieldSeparator Error = "invalid field separator"
	// ErrCrosstabResultMustHaveAtLeast3Columns is the crosstab result must
	// have at least 3 columns error.
	ErrCrosstabResultMustHaveAtLeast3Columns Error = "crosstab result must have at least 3 columns"
	// ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame is the crosstab
	// vertical and horizontal columns must not be same error.
	ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame Error = "crosstab vertical and horizontal columns must not be same"
	// ErrCrosstabVerticalColumnNotInResult is the crosstab vertical column not
	// in result error.
	ErrCrosstabVerticalColumnNotInResult Error = "crosstab vertical column not in result"
	// ErrCrosstabHorizontalColumnNotInResult is the crosstab horizontal column
	// not in result error.
	ErrCrosstabHorizontalColumnNotInResult Error = "crosstab horizontal column not in result"
	// ErrCrosstabContentColumnNotInResult is the crosstab content column not
	// in result error.
	ErrCrosstabContentColumnNotInResult Error = "crosstab content column not in result"
	// ErrCrosstabSortColumnNotInResult is the crosstab sort column not in
	// result error.
	ErrCrosstabSortColumnNotInResult Error = "crosstab sort column not in result"
)

// errEncoder provides a no-op encoder that always returns the wrapped error.
type errEncoder struct {
	err error
}

// Encode satisfies the Encoder interface.
func (err errEncoder) Encode(io.Writer) error {
	return err.err
}

// EncodeAll satisfies the Encoder interface.
func (err errEncoder) EncodeAll(io.Writer) error {
	return err.err
}

// newErrEncoder creates a no-op error encoder.
func newErrEncoder(_ ResultSet, opts ...Option) (Encoder, error) {
	enc := &errEncoder{}
	for _, o := range opts {
		if err := o(enc); err != nil {
			return nil, err
		}
	}
	return enc, enc.err
}

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

/*
// condWrite conditionally writes runes to w.
func condWrite(w io.Writer, repeat int, runes ...rune) error {
	var buf []byte
	for _, r := range runes {
		if r != 0 {
			buf = append(buf, []byte(string(r))...)
		}
	}
	if repeat > 1 {
		buf = bytes.Repeat(buf, repeat)
	}
	_, err := w.Write(buf)
	return err
}
*/
