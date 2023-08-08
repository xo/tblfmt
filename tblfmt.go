// Package tblfmt provides streaming table encoders for result sets (ie, from a
// database).
package tblfmt

import (
	"io"
	"runtime"
)

// Encoder is the shared interface for encoders.
type Encoder interface {
	Encode(io.Writer) error
	EncodeAll(io.Writer) error
}

// ResultSet is the shared interface for a result set.
type ResultSet interface {
	Next() bool
	Scan(...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
	NextResultSet() bool
}

// Encode encodes the result set to the writer using the supplied map options.
func Encode(w io.Writer, resultSet ResultSet, options map[string]string) error {
	f, opts := FromMap(options)
	enc, err := f(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeAll encodes all result sets to the writer using the supplied map
// options.
func EncodeAll(w io.Writer, resultSet ResultSet, options map[string]string) error {
	f, opts := FromMap(options)
	enc, err := f(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeTable encodes result set to the writer as a table using the supplied
// encoding options.
func EncodeTable(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTableAll encodes all result sets to the writer as a table using the
// supplied encoding options.
func EncodeTableAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeExpanded encodes result set to the writer as a table using the supplied
// encoding options.
func EncodeExpanded(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewExpandedEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeExpandedAll encodes all result sets to the writer as a table using the
// supplied encoding options.
func EncodeExpandedAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewExpandedEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeJSON encodes the result set to the writer as JSON using the supplied
// encoding options.
func EncodeJSON(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewJSONEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeJSONAll encodes all result sets to the writer as JSON using the
// supplied encoding options.
func EncodeJSONAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewJSONEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeUnaligned encodes the result set to the writer unaligned using the
// supplied encoding options.
func EncodeUnaligned(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewUnalignedEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeUnalignedAll encodes all result sets to the writer unaligned using the
// supplied encoding options.
func EncodeUnalignedAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewUnalignedEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeCSV encodes the result set to the writer unaligned using the
// supplied encoding options.
func EncodeCSV(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewCSVEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeCSVAll encodes all result sets to the writer unaligned using the
// supplied encoding options.
func EncodeCSVAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewCSVEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeTemplate encodes the result set to the writer using a template from
// the supplied encoding options.
func EncodeTemplate(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTemplateEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTemplateAll encodes all result sets to the writer using a template
// from the supplied encoding options.
func EncodeTemplateAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTemplateEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeHTML encodes the result set to the writer using the html template and
// the supplied encoding options.
func EncodeHTML(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewHTMLEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeHTML encodes the result set to the writer using the html template and
// the supplied encoding options.
func EncodeHTMLAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewHTMLEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeAsciiDoc encodes the result set to the writer using the asciidoc
// template and the supplied encoding options.
func EncodeAsciiDoc(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewAsciiDocEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeAsciiDoc encodes the result set to the writer using the asciidoc
// template and the supplied encoding options.
func EncodeAsciiDocAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewAsciiDocEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeVertical encodes the result set to the writer using the vertical
// template and the supplied encoding options.
func EncodeVertical(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewVerticalEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeVertical encodes the result set to the writer using the vertical
// template and the supplied encoding options.
func EncodeVerticalAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewVerticalEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// newline is the default newline used by the system.
var newline []byte

func init() {
	if runtime.GOOS == "windows" {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
}
