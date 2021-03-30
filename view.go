package tblfmt

import (
	"sort"
)

// CrosstabView is a crosstab view for result sets.
type CrosstabView struct {
	// resultSet is the wrapped result set.
	resultSet ResultSet
	// v is the vertical header column.
	v string
	// h is the horizontal header column.
	h string
	// c is the display column.
	c string
	// s is the sort column.
	s string
	// pos is the index for the result.
	pos int
	// cols are the result columns.
	cols []string
	// rows are the result rows.
	rows [][]interface{}
	// err is the last encountered error.
	err error
}

// NewCrosstabView creates a new crosstab view.
func NewCrosstabView(resultSet ResultSet, opts ...Option) (ResultSet, error) {
	view := &CrosstabView{
		resultSet: resultSet,
	}
	for _, o := range opts {
		if err := o.apply(view); err != nil {
			return nil, err
		}
	}
	if view.v != "" && view.h != "" && view.v == view.h {
		return nil, ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame
	}
	if view.err = view.build(); view.err != nil {
		return nil, view.err
	}
	return view, nil
}

// build builds the crosstab view.
func (view *CrosstabView) build() error {
	view.pos = 0
	cols, err := view.resultSet.Columns()
	if err != nil {
		return err
	}
	if len(cols) < 3 {
		return ErrCrosstabResultMustHaveAtLeast3Columns
	}
	vindex := findIndex(cols, view.v, 0)
	if vindex == -1 {
		return ErrCrosstabVerticalColumnNotInResult
	}
	hindex := findIndex(cols, view.h, 1)
	if hindex == -1 {
		return ErrCrosstabHorizontalColumnNotInResult
	}
	cindex := findIndex(cols, view.c, 2)
	if cindex == -1 {
		return ErrCrosstabContentColumnNotInResult
	}
	sindex := -1
	if view.s != "" {
		if sidx := indexOf(cols, view.s); sidx != -1 {
			sindex = sidx
		} else {
			return ErrCrosstabSortColumnNotInResult
		}
	}
	n := len(cols)
	var rows [][]interface{}
	for view.resultSet.Next() {
		row := make([]interface{}, n)
		for i := 0; i < n; i++ {
			row[i] = new(interface{})
		}
		if err := view.resultSet.Scan(row...); err != nil {
			return err
		}
		r := []interface{}{
			row[vindex],
			row[hindex],
			row[cindex],
		}
		if sindex != -1 {
			r = append(r, row[sindex])
		}
		rows = append(rows, r)
	}
	if err := view.resultSet.Err(); err != nil {
		return err
	}
	for i := 0; i < len(rows); i++ {
	}
	// sort
	if len(view.rows) != 0 && sindex != -1 {
		sort.Slice(view.rows, func(i, j int) bool {
			return false
		})
	}
	return nil
}

// Next satisfies the ResultSet interface.
func (view *CrosstabView) Next() bool {
	if view.err != nil {
		return false
	}
	view.pos++
	return view.pos < len(view.rows)
}

// Scan satisfies the ResultSet interface.
func (view *CrosstabView) Scan(v ...interface{}) error {
	for i := 0; i < len(v); i++ {
	}
	return nil
}

// Columns satisfies the ResultSet interface.
func (view *CrosstabView) Columns() ([]string, error) {
	if view.err != nil {
		return nil, view.err
	}
	return view.cols, nil
}

// Close satisfies the ResultSet interface.
func (view *CrosstabView) Close() error {
	return view.resultSet.Close()
}

// Err satisfies the ResultSet interface.
func (view *CrosstabView) Err() error {
	return view.err
}

// NextResultSet satisfies the ResultSet interface.
func (view *CrosstabView) NextResultSet() bool {
	// NOTE: it was decided that this should not be called multiple times.
	// This behavior might be revisited at a later date.
	return false
}
