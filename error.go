package tblfmt

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
