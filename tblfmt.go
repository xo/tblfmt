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

// Error is an  error.
type Error string

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Error values.
const (
	// ErrResultSetIsNil is the result set is nil error.
	ErrResultSetIsNil Error = "result set is nil"
	// ErrResultSetHasNoColumnTypes is the result set has no column types error.
	ErrResultSetHasNoColumnTypes Error = "result set has no column types"
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
	// ErrInvalidColumnParams is the invalid column params error.
	ErrInvalidColumnParams Error = "invalid column params"
	// ErrCrosstabResultMustHaveAtLeast3Columns is the crosstab result must
	// have at least 3 columns error.
	ErrCrosstabResultMustHaveAtLeast3Columns Error = "crosstab result must have at least 3 columns"
	// ErrCrosstabDataColumnMustBeSpecifiedWhenQueryReturnsMoreThanThreeColumnsA
	// is the data column must be specified when query returns more than three
	// columns error.
	ErrCrosstabDataColumnMustBeSpecifiedWhenQueryReturnsMoreThanThreeColumns Error = "data column must be specified when query returns more than three columns"
	// ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame is the crosstab
	// vertical and horizontal columns must not be same error.
	ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame Error = "crosstab vertical and horizontal columns must not be same"
	// ErrCrosstabVerticalColumnNotInResult is the crosstab vertical column not
	// in result error.
	ErrCrosstabVerticalColumnNotInResult Error = "crosstab vertical column not in result"
	// ErrCrosstabHorizontalColumnNotInResult is the crosstab horizontal column
	// not in result error.
	ErrCrosstabHorizontalColumnNotInResult Error = "crosstab horizontal column not in result"
	// ErrCrosstabDataColumnNotInResult is the crosstab data column not in
	// result error.
	ErrCrosstabDataColumnNotInResult Error = "crosstab data column not in result"
	// ErrCrosstabHorizontalSortColumnNotInResult is the crosstab horizontal
	// sort column not in result error.
	ErrCrosstabHorizontalSortColumnNotInResult Error = "crosstab horizontal sort column not in result"
	// ErrCrosstabDuplicateVerticalAndHorizontalValue is the crosstab duplicate
	// vertical and horizontal value error.
	ErrCrosstabDuplicateVerticalAndHorizontalValue Error = "crosstab duplicate vertical and horizontal value"
	// ErrCrosstabHorizontalSortColumnIsNotANumber is the crosstab horizontal
	// sort column is not a number error.
	ErrCrosstabHorizontalSortColumnIsNotANumber Error = "crosstab horizontal sort column is not a number"
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
