package tblfmt

import "io"

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
