package tblfmt

import (
	"sort"
	"strconv"
	"strings"
)

// CrosstabView is a crosstab view for result sets.
//
// CAUTION:
//
// A design decision was made to not support multiple result sets, and to force
// the user to create a new crosstab view for each result set. As such,
// NextResultSet always returns false, and any use of this view should take
// care when using inside a loop or passing to other code that calls
// NextResultSet.
type CrosstabView struct {
	// resultSet is the wrapped result set.
	resultSet ResultSet
	// formatter is the formatter.
	formatter Formatter
	// v is the vertical header column.
	v string
	// h is the horizontal header column.
	h string
	// d is the data column.
	d string
	// s is the horizontal header sort column.
	s string
	// vmap is the map of vertical rows.
	vkeys []string
	// hmap is the map of horizontal columns.
	hkeys []hkey
	// vals are the result values.
	vals map[string]map[string]interface{}
	// pos is the index for the result.
	pos int
	// err is the last encountered error.
	err error
}

// NewCrosstabView creates a new crosstab view.
func NewCrosstabView(resultSet ResultSet, opts ...Option) (ResultSet, error) {
	view := &CrosstabView{
		resultSet: resultSet,
		formatter: NewEscapeFormatter(WithIsRaw(true, 0, 0)),
	}
	for _, o := range opts {
		if err := o.apply(view); err != nil {
			return nil, err
		}
	}
	if view.v != "" && view.h != "" && view.v == view.h {
		return nil, ErrCrosstabVerticalAndHorizontalColumnsMustNotBeSame
	}
	if err := view.build(); err != nil {
		return nil, err
	}
	return view, nil
}

// build builds the crosstab view.
func (view *CrosstabView) build() error {
	// reset
	view.pos = -1
	view.vals = make(map[string]map[string]interface{})
	// get columns
	cols, err := view.resultSet.Columns()
	if err != nil {
		return view.fail(err)
	}
	if len(cols) < 3 {
		return view.fail(ErrCrosstabResultMustHaveAtLeast3Columns)
	}
	if len(cols) > 3 && view.d == "" {
		return view.fail(ErrCrosstabDataColumnMustBeSpecifiedWhenQueryReturnsMoreThanThreeColumns)
	}
	vindex := findIndex(cols, view.v, 0)
	if vindex == -1 {
		return view.fail(ErrCrosstabVerticalColumnNotInResult)
	}
	view.v = cols[vindex]
	hindex := findIndex(cols, view.h, 1)
	if hindex == -1 {
		return view.fail(ErrCrosstabHorizontalColumnNotInResult)
	}
	view.h = cols[hindex]
	// this complicated bit of code is used to find the 'unused' column for c
	// (ie, when number of columns == 3, and v and h are specified)
	//
	// psql manual states (colD == c):
	//
	//   " If colD is not specified, then there must be exactly three columns
	//   in the query result, and the column that is neither colV nor colH is
	//   taken to be colD."
	ddef := 2
	if view.d == "" {
		used := map[int]bool{
			vindex: true,
			hindex: true,
		}
		for i := 0; i < 3; i++ {
			if !used[i] {
				ddef = i
			}
		}
	}
	dindex := findIndex(cols, view.d, ddef)
	if dindex == -1 {
		return view.fail(ErrCrosstabDataColumnNotInResult)
	}
	view.d = cols[dindex]
	sindex := -1
	if view.s != "" {
		if sidx := indexOf(cols, view.s); sidx != -1 {
			sindex = sidx
		} else {
			return view.fail(ErrCrosstabHorizontalSortColumnNotInResult)
		}
	}
	clen := len(cols)
	// process results
	for view.resultSet.Next() {
		row := make([]interface{}, clen)
		for i := 0; i < clen; i++ {
			row[i] = new(interface{})
		}
		if err := view.resultSet.Scan(row...); err != nil {
			return view.fail(err)
		}
		// raw format values
		vals := []interface{}{row[vindex], row[hindex]}
		if sindex != -1 {
			vals = append(vals, row[sindex])
		}
		v, err := view.formatter.Format(vals)
		if err != nil {
			return view.fail(err)
		}
		var s *Value
		if sindex != -1 {
			s = v[2]
		}
		if err := view.add(*(row[dindex].(*interface{})), v[0], v[1], s); err != nil {
			return view.fail(err)
		}
	}
	if err := view.resultSet.Err(); err != nil {
		return view.fail(err)
	}
	// sort
	if sindex != -1 {
		sort.Slice(view.hkeys, func(i, j int) bool {
			return view.hkeys[i].s < view.hkeys[j].s
		})
	}
	return nil
}

// fail sets the internal error to the passed error and returns it.
func (view *CrosstabView) fail(err error) error {
	view.err = err
	return err
}

// add processes and adds a val.
func (view *CrosstabView) add(d interface{}, v, h, s *Value) error {
	// determine sort value
	var sval int
	if s != nil {
		var err error
		sval, err = strconv.Atoi(s.String())
		if err != nil {
			return ErrCrosstabHorizontalSortColumnIsNotANumber
		}
	}
	// add v and h keys
	vk, hk := v.String(), h.String()
	view.vkeys = vkeyAppend(view.vkeys, vk)
	view.hkeys = hkeyAppend(view.hkeys, hkey{v: hk, s: sval})
	// store
	if _, ok := view.vals[vk]; !ok {
		view.vals[vk] = make(map[string]interface{})
	}
	if _, ok := view.vals[vk][hk]; ok {
		return ErrCrosstabDuplicateVerticalAndHorizontalValue
	}
	view.vals[vk][hk] = d
	return nil
}

// Next satisfies the ResultSet interface.
func (view *CrosstabView) Next() bool {
	if view.err != nil {
		return false
	}
	view.pos++
	return view.pos < len(view.vkeys)
}

// Scan satisfies the ResultSet interface.
func (view *CrosstabView) Scan(v ...interface{}) error {
	vkey := view.vkeys[view.pos]
	if len(v) > 0 {
		*(v[0].(*interface{})) = vkey
	}
	row := view.vals[vkey]
	for i := 0; i < len(view.hkeys) && i < len(v)-1; i++ {
		if z, ok := row[view.hkeys[i].v]; ok {
			*(v[i+1].(*interface{})) = z
		} else {
			*(v[i+1].(*interface{})) = nil
		}
	}
	return nil
}

// Columns satisfies the ResultSet interface.
func (view *CrosstabView) Columns() ([]string, error) {
	if view.err != nil {
		return nil, view.err
	}
	cols := make([]string, len(view.hkeys)+1)
	cols[0] = view.v
	for i := 0; i < len(view.hkeys); i++ {
		cols[i+1] = view.hkeys[i].v
	}
	return cols, nil
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
	return false
}

// vkeyAppend determines if k is in v, if so it returns the unmodified v.
// Otherwise, appends k to v.
func vkeyAppend(v []string, k string) []string {
	for _, z := range v {
		if z == k {
			return v
		}
	}
	return append(v, k)
}

// hkey wraps a horizontal column.
type hkey struct {
	v string
	s int
}

// hkeyAppend determines if k is in v, if so it returns the unmodified v.
// Otherwise, appends k to v.
func hkeyAppend(v []hkey, k hkey) []hkey {
	for _, z := range v {
		if z.v == k.v {
			return v
		}
	}
	return append(v, k)
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
			return i
		}
		return -1
	}
	return indexOf(v, s)
}
