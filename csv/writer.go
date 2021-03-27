package csv

import (
	"bufio"
	"io"
)

// Writer identical to "encoding/csv", but does not quote values and allows any newline.
type Writer struct {
	Comma   rune
	Newline string
	w       *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Comma:   ',',
		Newline: "\n",
		w:       bufio.NewWriter(w),
	}
}

// Write writes a single CSV record to w.
// A record is a slice of strings with each string being one field.
// Writes are buffered, so Flush must eventually be called to ensure
// that the record is written to the underlying io.Writer.
func (w *Writer) Write(record []string) error {
	for n, field := range record {
		if n > 0 {
			if _, err := w.w.WriteRune(w.Comma); err != nil {
				return err
			}
		}

		if _, err := w.w.WriteString(field); err != nil {
			return err
		}
	}
	_, err := w.w.WriteString(w.Newline)
	return err
}

// Flush writes any buffered data to the underlying io.Writer.
// To check if an error occurred during the Flush, call Error.
func (w *Writer) Flush() {
	w.w.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.w.Write(nil)
	return err
}
