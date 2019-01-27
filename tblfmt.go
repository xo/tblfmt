// Package tblfmt provides encoders for streaming / writing rows of data.
package tblfmt

import (
	"io"
	"strconv"
)

// Encoder is the shared interface for tblfmt encoders.
type Encoder interface {
	Encode(io.Writer) error
	EncodeAll(io.Writer) error
}

// EncodeTable encodes result set to the writer as a table using the supplied
// encoding options.
func EncodeTable(w io.Writer, rs ResultSet, opts ...TableEncoderOption) error {
	enc, err := NewTableEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTableAll encodes all result sets to the writer as a table using the
// supplied encoding options.
func EncodeTableAll(w io.Writer, rs ResultSet, opts ...TableEncoderOption) error {
	enc, err := NewTableEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

/*

// EncodeJSON encodes the result set to the writer as JSON using the supplied
// encoding options.
func EncodeJSON(w io.Writer, rs ResultSet, opts ...JSONEncoderOption) error {
	enc, err := NewJSONEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeJSONAll encodes all result sets to the writer as JSON using the
// supplied encoding options.
func EncodeJSONAll(w io.Writer, rs ResultSet, opts ...JSONEncoderOption) error {
	enc, err := NewJSONEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

*/

// EncoderFromMap creates an encoder based on the passed map options.
func EncoderFromMap(rs ResultSet, opts map[string]string) (Encoder, error) {
	var tableOpts []TableEncoderOption

	switch opts["format"] {
	case "aligned":
		if s, ok := opts["border"]; ok {
			border, _ := strconv.Atoi(s)
			tableOpts = append(tableOpts, WithBorder(border))
		}
		if s, ok := opts["linestyle"]; ok {
			switch s {
			case "ascii":
				tableOpts = append(tableOpts, WithLineStyle(ASCIILineStyle()))
			case "old-ascii":
				tableOpts = append(tableOpts, WithLineStyle(OldASCIILineStyle()))
			case "unicode":
				switch opts["unicode_border_linestyle"] {
				case "single":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeLineStyle()))
				case "double":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeDoubleLineStyle()))
				}
			}
		}
	}

	return NewTableEncoder(rs, tableOpts...)
}

// Encode encodes the result set to the writer using the supplied map options.
func Encode(w io.Writer, rs ResultSet, opts map[string]string) error {
	enc, err := EncoderFromMap(rs, opts)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeAll encodes all result sets to the writer using the supplied map
// options.
func EncodeAll(w io.Writer, rs ResultSet, opts map[string]string) error {
	enc, err := EncoderFromMap(rs, opts)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}
