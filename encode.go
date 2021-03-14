package tblfmt

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

// TableEncoder is a buffered, lookahead table encoder for result sets.
type TableEncoder struct {
	// ResultSet is the result set to encode.
	resultSet ResultSet

	// count is the number of rows to scan ahead by (buffer), up to count, in
	// order to determine maximum column widths returned by the encoder's
	// formatter.
	//
	// Note: when 0 all rows will be scanned (buffered) prior to encoding the
	// table.
	count int

	// tab is the tab width.
	tab int

	// newline is the newline to use.
	newline []byte

	// border is the display border size.
	border int

	// inline toggles drawing the column header names inline with the top line.
	inline bool

	// lineStyle is the table line style.
	lineStyle LineStyle

	// formatter handles formatting values prior to output.
	formatter Formatter

	// summary is the summary map.
	summary map[int]func(io.Writer, int) (int, error)

	// isCustomSummary when summary has been set via options
	isCustomSummary bool

	// title is the title value.
	title *Value

	// empty is the empty value.
	empty *Value

	// headers contains formatted column names.
	headers []*Value

	// offsets are the column offsets.
	offsets []int

	// widths are the user-supplied column widths.
	widths []int

	// maxWidths are calculated max column widths.
	// They are at least as wide as user-supplied widths
	maxWidths []int

	// maxWidth of whole table, before switching to the ExpandedEncoder,
	// zero disables switching
	maxWidth int

	// scanCount is the number of scanned results in the result set.
	scanCount int

	// w is the undelying writer
	w *bufio.Writer
}

// NewTableEncoder creates a new table encoder using the provided options.
func NewTableEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	var err error

	enc := &TableEncoder{
		resultSet: resultSet,
		newline:   newline,
		border:    1,
		tab:       8,
		lineStyle: ASCIILineStyle(),
		formatter: NewEscapeFormatter(WithHeaderAlign(AlignCenter)),
		summary:   DefaultTableSummary(),
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}

	// apply options
	for _, o := range opts {
		if err = o(enc); err != nil {
			return nil, err
		}
	}

	// check linestyle runes
	// TODO: this check should be removed
	for _, l := range [][4]rune{
		enc.lineStyle.Top,
		enc.lineStyle.Mid,
		enc.lineStyle.Row,
		enc.lineStyle.Wrap,
		enc.lineStyle.End,
	} {
		for _, r := range l {
			if r != 0 && runewidth.RuneWidth(r) != 1 {
				return nil, ErrInvalidLineStyle
			}
		}
	}

	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *TableEncoder) Encode(w io.Writer) error {
	// reset scan count
	enc.scanCount = 0
	enc.w = bufio.NewWriterSize(w, 2048)

	var err error

	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}

	// get and check columns
	cols, err := enc.resultSet.Columns()
	if err != nil {
		return err
	}
	clen := len(cols)
	if clen == 0 {
		return ErrResultSetHasNoColumns
	}

	// setup offsets, widths
	enc.offsets = make([]int, clen)
	var wroteHeader bool

	// default to user-supplied widths
	if len(enc.widths) == clen {
		enc.maxWidths = enc.widths
	} else {
		enc.maxWidths = make([]int, clen)
	}

	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}

	for {
		var vals [][]*Value

		// buffer
		vals, err = enc.nextResults()
		if err != nil {
			return err
		}

		// no more values
		if len(vals) == 0 {
			break
		}

		enc.calcWidth(vals)

		if enc.maxWidth != 0 && enc.tableWidth() > enc.maxWidth {
			t := *enc
			t.formatter = NewEscapeFormatter()
			exp := ExpandedEncoder{
				TableEncoder: t,
			}
			exp.offsets = make([]int, 2)
			exp.maxWidths = make([]int, 2)
			exp.calcWidth(vals)

			if err := exp.encodeVals(vals); err != nil {
				return nil
			}
			continue
		}

		// print header if not already done
		if !wroteHeader {
			wroteHeader = true
			enc.header()
		}

		if err := enc.encodeVals(vals); err != nil {
			return err
		}

		// draw end border
		if enc.border >= 2 {
			enc.divider(enc.rowStyle(enc.lineStyle.End))
		}
	}

	// add summary
	enc.summarize(w)

	// flush will return the error code
	return enc.w.Flush()
}

func (enc *TableEncoder) encodeVals(vals [][]*Value) error {
	rs := enc.rowStyle(enc.lineStyle.Row)
	// print buffered vals
	for i := 0; i < len(vals); i++ {
		enc.row(vals[i], rs)
		if i+1%1000 == 0 {
			// check error every 1k rows
			if err := enc.w.Flush(); err != nil {
				return err
			}
		}
	}

	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TableEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// nextResults reads the next enc.count values,
// or all values if enc.count = 0
func (enc *TableEncoder) nextResults() ([][]*Value, error) {
	var vals [][]*Value
	if enc.count != 0 {
		vals = make([][]*Value, 0, enc.count)
	}
	// set up storage for results
	r := make([]interface{}, len(enc.headers))
	for i := 0; i < len(enc.headers); i++ {
		r[i] = new(interface{})
	}

	// read to count (or all)
	var i int
	for enc.resultSet.Next() {
		v, err := enc.scanAndFormat(r)
		if err != nil {
			return vals, err
		}
		vals, i = append(vals, v), i+1

		// read by batches of enc.count rows
		if enc.count != 0 && i%enc.count == 0 {
			break
		}
	}
	return vals, nil
}

func (enc *TableEncoder) calcWidth(vals [][]*Value) {
	// calc offsets and widths for this batch of rows
	var offset int
	rs := enc.rowStyle(enc.lineStyle.Row)
	offset += runewidth.StringWidth(string(rs.left))
	for i, h := range enc.headers {
		if i != 0 {
			offset += runewidth.StringWidth(string(rs.middle))
		}

		// store offset
		enc.offsets[i] = offset

		// header's widths are the minimum
		enc.maxWidths[i] = max(enc.maxWidths[i], h.MaxWidth(offset, enc.tab))

		// from top to bottom, find max column width
		for j := 0; j < len(vals); j++ {
			cell := vals[j][i]
			if cell == nil {
				cell = enc.empty
			}
			enc.maxWidths[i] = max(enc.maxWidths[i], cell.MaxWidth(offset, enc.tab))
		}

		// add column width, and one space for newline indicator
		offset += enc.maxWidths[i]
		if rs.hasWrapping && enc.border != 0 {
			offset++
		}
	}
}

func (enc *TableEncoder) header() {
	rs := enc.rowStyle(enc.lineStyle.Row)

	if enc.title != nil {
		maxWidth := ((enc.tableWidth() - enc.title.Width) / 2) + enc.title.Width
		enc.writeAligned(enc.title.Buf, rs.filler, AlignRight, enc.title.Width, maxWidth)
		enc.w.Write(enc.newline)
	}
	// draw top border
	if enc.border >= 2 && !enc.inline {
		enc.divider(enc.rowStyle(enc.lineStyle.Top))
	}

	// draw the header row with top border style
	if enc.inline {
		rs = enc.rowStyle(enc.lineStyle.Top)
	}

	// write header
	enc.row(enc.headers, rs)

	if !enc.inline {
		// draw mid divider
		enc.divider(enc.rowStyle(enc.lineStyle.Mid))
	}
}

// the style for the current row, as arrays of bytes to print
type rowStyle struct {
	left, right, middle, filler, wrapper []byte
	hasWrapping                          bool
}

// rowStyle returns the left, right and midle borders.
// It also profides the filler string, and indicates
// if this style uses a wrapping indicator.
func (enc TableEncoder) rowStyle(r [4]rune) rowStyle {
	var left, right, middle, spacer, filler string
	spacer = strings.Repeat(string(r[1]), runewidth.RuneWidth(enc.lineStyle.Row[1]))
	filler = string(r[1])

	// compact output, r[1] is set to \0
	if r[1] == 0 {
		filler = " "
	}

	// outside borders
	if enc.border > 1 {
		left = string(r[0])
		right = string(r[3])
	}

	// initial spacer when borders are set
	if enc.border > 0 {
		left += spacer
	}

	middle = " "
	if enc.border >= 1 { // inside border
		middle = string(r[2]) + spacer
	}

	return rowStyle{
		left:        []byte(left),
		wrapper:     []byte(string(enc.lineStyle.Wrap[1])),
		middle:      []byte(middle),
		right:       []byte(right + string(enc.newline)),
		filler:      []byte(filler),
		hasWrapping: runewidth.RuneWidth(enc.lineStyle.Row[1]) > 0,
	}
}

// scanAndFormat scans and formats values from the result set.
func (enc *TableEncoder) scanAndFormat(vals []interface{}) ([]*Value, error) {
	var err error
	if err = enc.resultSet.Err(); err != nil {
		return nil, err
	}
	if err = enc.resultSet.Scan(vals...); err != nil {
		return nil, err
	}
	enc.scanCount++
	return enc.formatter.Format(vals)
}

// divider draws a divider.
func (enc *TableEncoder) divider(rs rowStyle) {
	// left
	enc.w.Write(rs.left)

	for i, width := range enc.maxWidths {
		// column
		enc.w.Write(bytes.Repeat(rs.filler, width))

		// line feed indicator
		if rs.hasWrapping && enc.border >= 1 {
			enc.w.Write(rs.filler)
		}

		// middle separator
		if i != len(enc.maxWidths)-1 {
			enc.w.Write(rs.middle)
		}
	}

	// right
	enc.w.Write(rs.right)
}

// tableWidth calculates total table width.
func (enc *TableEncoder) tableWidth() int {
	rs := enc.rowStyle(enc.lineStyle.Mid)
	width := runewidth.StringWidth(string(rs.left)) + runewidth.StringWidth(string(rs.right))

	for i, w := range enc.maxWidths {
		width += w

		if rs.hasWrapping && enc.border >= 1 {
			width += 1
		}

		if i != len(enc.maxWidths)-1 {
			width += runewidth.StringWidth(string(rs.middle))
		}
	}

	return width
}

// row draws the a table row.
func (enc *TableEncoder) row(vals []*Value, rs rowStyle) {
	var l int
	for {
		// left
		enc.w.Write(rs.left)

		var remaining bool
		for i, v := range vals {
			if v == nil {
				v = enc.empty
			}

			// write value
			if l <= len(v.Newlines) {
				// determine start, end, width
				start, end, width := 0, len(v.Buf), 0
				if l > 0 {
					start = v.Newlines[l-1][0] + 1
				}
				if l < len(v.Newlines) {
					end = v.Newlines[l][0]
					width += v.Newlines[l][1]
				}
				if len(v.Tabs) != 0 && len(v.Tabs[l]) != 0 {
					width += tabwidth(v.Tabs[l], enc.offsets[i], enc.tab)
				}
				if l == len(v.Newlines) {
					width += v.Width
				}

				enc.writeAligned(v.Buf[start:end], rs.filler, v.Align, width, enc.maxWidths[i])
			} else {
				enc.w.Write(bytes.Repeat(rs.filler, enc.maxWidths[i]))
			}

			// write newline wrap value
			if rs.hasWrapping {
				if l < len(v.Newlines) {
					enc.w.Write(rs.wrapper)
				} else {
					enc.w.Write(rs.filler)
				}
			}

			remaining = remaining || l < len(v.Newlines)

			// middle separator. If border == 0, the new line indicator
			// acts as the middle separator
			if i != len(enc.maxWidths)-1 && enc.border >= 1 {
				enc.w.Write(rs.middle)
			}
		}

		// right
		enc.w.Write(rs.right)

		if !remaining {
			break
		}

		l++
	}
}

func (enc *TableEncoder) writeAligned(b, filler []byte, a Align, width, max int) {
	// calc padding
	padding := max - width
	paddingLeft := 0
	paddingRight := 0
	switch a {
	case AlignRight:
		paddingLeft = padding
		paddingRight = 0
	case AlignCenter:
		paddingLeft = padding / 2
		paddingRight = padding/2 + padding%2
	case AlignLeft:
		paddingLeft = 0
		paddingRight = padding
	}

	// add padding left
	if paddingLeft > 0 {
		enc.w.Write(bytes.Repeat(filler, paddingLeft))
	}

	// write
	enc.w.Write(b)

	// add padding right
	if paddingRight > 0 {
		enc.w.Write(bytes.Repeat(filler, paddingRight))
	}
}

// summarize writes the table scan count summary.
func (enc *TableEncoder) summarize(w io.Writer) {
	// do summary
	if enc.summary == nil {
		return
	}
	var f func(io.Writer, int) (int, error)
	if z, ok := enc.summary[-1]; ok {
		f = z
	}
	if z, ok := enc.summary[enc.scanCount]; ok {
		f = z
	}
	if f != nil {
		f(enc.w, enc.scanCount)
		enc.w.Write(enc.newline)
	}
}

// ExpandedEncoder is a buffered, lookahead expanded table encoder for result sets.
type ExpandedEncoder struct {
	TableEncoder
}

// NewExpandedEncoder creates a new table encoder using the provided options.
func NewExpandedEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	tableEnc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return nil, err
	}
	t := tableEnc.(*TableEncoder)
	t.formatter = NewEscapeFormatter()
	if !t.isCustomSummary {
		t.summary = nil
	}

	enc := &ExpandedEncoder{
		TableEncoder: *t,
	}

	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *ExpandedEncoder) Encode(w io.Writer) error {
	// reset scan count
	enc.scanCount = 0
	enc.w = bufio.NewWriterSize(w, 2048)

	var err error

	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}

	// get and check columns
	cols, err := enc.resultSet.Columns()
	if err != nil {
		return err
	}
	clen := len(cols)
	if clen == 0 {
		return ErrResultSetHasNoColumns
	}

	// setup offsets, widths
	enc.offsets = make([]int, 2)
	enc.maxWidths = make([]int, 2)

	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}

	var wroteTitle bool
	for {
		var vals [][]*Value

		// buffer
		vals, err = enc.nextResults()
		if err != nil {
			return err
		}

		// no more values
		if len(vals) == 0 {
			break
		}

		enc.calcWidth(vals)

		// print title if not already done
		if !wroteTitle && enc.title != nil {
			wroteTitle = true

			enc.w.Write(enc.title.Buf)
			enc.w.Write(enc.newline)
		}

		if err := enc.encodeVals(vals); err != nil {
			return err
		}
	}

	// add summary
	enc.summarize(w)

	// flush will return the error code
	return enc.w.Flush()
}

func (enc *ExpandedEncoder) encodeVals(vals [][]*Value) error {
	rs := enc.rowStyle(enc.lineStyle.Row)
	// print buffered vals
	for i := 0; i < len(vals); i++ {
		enc.record(i, vals[i], rs)
		if i+1%1000 == 0 {
			// check error every 1k rows
			if err := enc.w.Flush(); err != nil {
				return err
			}
		}
	}

	// draw end border
	if enc.border >= 2 && enc.scanCount != 0 {
		enc.divider(enc.rowStyle(enc.lineStyle.End))
	}
	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *ExpandedEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

func (enc *ExpandedEncoder) calcWidth(vals [][]*Value) {
	rs := enc.rowStyle(enc.lineStyle.Row)

	offset := runewidth.StringWidth(string(rs.left))

	enc.offsets[0] = offset

	// first column is always the column name
	for _, h := range enc.headers {
		enc.maxWidths[0] = max(enc.maxWidths[0], h.MaxWidth(offset, enc.tab))
	}

	offset += enc.maxWidths[0]
	if rs.hasWrapping && enc.border != 0 {
		offset++
	}

	mw := runewidth.StringWidth(string(rs.middle))
	offset += mw

	enc.offsets[1] = offset

	// second column is any value from any row but no less than the record header
	enc.maxWidths[1] = max(0, len(enc.recordHeader(len(vals)-1))-enc.maxWidths[0]-mw-1)
	for _, row := range vals {
		for _, cell := range row {
			if cell == nil {
				cell = enc.empty
			}
			enc.maxWidths[1] = max(enc.maxWidths[1], cell.MaxWidth(offset, enc.tab))
		}
	}
}

func (enc *ExpandedEncoder) record(i int, vals []*Value, rs rowStyle) {
	// write record header as a single record
	headerRS := rs
	header := enc.recordHeader(i)
	if enc.border != 0 {
		headerRS = enc.rowStyle(enc.lineStyle.Top)
		if i != 0 {
			headerRS = enc.rowStyle(enc.lineStyle.Mid)
		}
	}

	enc.w.Write(headerRS.left)
	enc.w.WriteString(header)
	padding := enc.maxWidths[0] + enc.maxWidths[1] + runewidth.StringWidth(string(headerRS.middle))*2 - len(header) - 1
	if padding > 0 {
		enc.w.Write(bytes.Repeat(headerRS.filler, padding))
	}
	// write newline wrap value
	enc.w.Write(headerRS.filler)
	enc.w.Write(headerRS.right)

	// write each value with column name in first col
	for j, v := range vals {
		if v != nil {
			v.Align = AlignLeft
		}
		enc.row([]*Value{enc.headers[j], v}, rs)
	}
}

func (enc *ExpandedEncoder) recordHeader(i int) string {
	header := fmt.Sprintf("* Record %d", i+1)
	if enc.border != 0 {
		header = fmt.Sprintf("[ RECORD %d ]", i+1)
	}
	return header
}

// JSONEncoder is an unbuffered JSON encoder for result sets.
type JSONEncoder struct {
	resultSet ResultSet

	// newline is the newline to use.
	newline []byte

	// formatter handles formatting values prior to output.
	formatter Formatter

	// title is the title value.
	title *Value

	// empty is the empty value.
	empty *Value
}

// NewJSONEncoder creates a new JSON encoder using the provided options.
func NewJSONEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	var err error
	enc := &JSONEncoder{
		resultSet: resultSet,
		newline:   newline,
		formatter: NewEscapeFormatter(WithEscapeJSON(true)),
		empty: &Value{
			Buf:  []byte("null"),
			Tabs: make([][][2]int, 1),
			Raw:  true,
		},
	}
	for _, o := range opts {
		if err = o(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *JSONEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}

	var i int
	var err error

	var (
		start = []byte{'['}
		end   = []byte{']'}
		open  = []byte{'{'}
		cls   = []byte{'}'}
		q     = []byte{'"'}
		cma   = []byte{','}
	)

	// get and check columns
	cols, err := enc.resultSet.Columns()
	if err != nil {
		return err
	}
	clen := len(cols)
	if clen == 0 {
		return ErrResultSetHasNoColumns
	}
	cb := make([][]byte, clen)
	for i = 0; i < clen; i++ {
		if cb[i], err = json.Marshal(cols[i]); err != nil {
			return err
		}
		cb[i] = append(cb[i], ':')
	}

	// set up storage for results
	r := make([]interface{}, clen)
	for i = 0; i < clen; i++ {
		r[i] = new(interface{})
	}

	// start
	if _, err = w.Write(start); err != nil {
		return err
	}

	// process
	var v *Value
	var vals []*Value
	var count int
	for enc.resultSet.Next() {
		if count != 0 {
			if _, err = w.Write(cma); err != nil {
				return err
			}
		}
		count++
		vals, err = enc.scanAndFormat(r)
		if err != nil {
			return err
		}

		if _, err = w.Write(open); err != nil {
			return err
		}

		for i = 0; i < clen; i++ {
			v = vals[i]
			if v == nil {
				v = enc.empty
			}

			// write "column":
			if _, err = w.Write(cb[i]); err != nil {
				return err
			}

			// if raw, write the exact value
			if v.Raw {
				if _, err = w.Write(v.Buf); err != nil {
					return err
				}
			} else {
				if _, err = w.Write(q); err != nil {
					return err
				}
				if _, err = w.Write(v.Buf); err != nil {
					return err
				}
				if _, err = w.Write(q); err != nil {
					return err
				}
			}

			if i != clen-1 {
				if _, err = w.Write(cma); err != nil {
					return err
				}
			}
		}

		if _, err = w.Write(cls); err != nil {
			return err
		}
	}

	// end
	_, err = w.Write(end)
	return err
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *JSONEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write([]byte{','}); err != nil {
			return err
		}

		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// scanAndFormat scans and formats values from the result set.
func (enc *JSONEncoder) scanAndFormat(vals []interface{}) ([]*Value, error) {
	var err error
	if err = enc.resultSet.Err(); err != nil {
		return nil, err
	}
	if err = enc.resultSet.Scan(vals...); err != nil {
		return nil, err
	}
	return enc.formatter.Format(vals)
}

// CSVEncoder is an unbuffered CSV encoder for result sets.
type CSVEncoder struct {
	// ResultSet is the result set to encode.
	resultSet ResultSet

	// fieldsep is the field separator to use.
	fieldsep rune

	// newline is the newline to use.
	newline []byte

	// formatter handles formatting values prior to output.
	formatter Formatter

	// title is the title value.
	title *Value

	// empty is the empty value.
	empty *Value
}

// NewCSVEncoder creates a new CSV encoder using the provided options.
func NewCSVEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	var err error
	enc := &CSVEncoder{
		resultSet: resultSet,
		newline:   newline,
		formatter: NewEscapeFormatter(),
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}
	for _, o := range opts {
		if err = o(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *CSVEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}

	var i int
	var err error

	c := csv.NewWriter(w)
	if enc.fieldsep != 0 {
		c.Comma = enc.fieldsep
	}

	// get and check columns
	cols, err := enc.resultSet.Columns()
	if err != nil {
		return err
	}
	clen := len(cols)
	if clen == 0 {
		return ErrResultSetHasNoColumns
	}

	if err = c.Write(cols); err != nil {
		return err
	}

	// set up storage for results
	r := make([]interface{}, clen)
	for i = 0; i < clen; i++ {
		r[i] = new(interface{})
	}

	// process
	var v *Value
	var vals []*Value
	z := make([]string, clen)
	for enc.resultSet.Next() {
		c.Flush()
		if err = c.Error(); err != nil {
			return err
		}
		vals, err = enc.scanAndFormat(r)
		if err != nil {
			return err
		}

		for i = 0; i < clen; i++ {
			v = vals[i]
			if v == nil {
				v = enc.empty
			}
			z[i] = string(v.Buf)
		}
		if err = c.Write(z); err != nil {
			return err
		}
	}

	// flush
	c.Flush()
	return c.Error()
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *CSVEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// scanAndFormat scans and formats values from the result set.
func (enc *CSVEncoder) scanAndFormat(vals []interface{}) ([]*Value, error) {
	var err error
	if err = enc.resultSet.Err(); err != nil {
		return nil, err
	}
	if err = enc.resultSet.Scan(vals...); err != nil {
		return nil, err
	}
	return enc.formatter.Format(vals)
}

// TemplateEncoder is an unbuffered template encoder for result sets.
type TemplateEncoder struct {
	// ResultSet is the result set to encode.
	resultSet ResultSet

	// newline is the newline to use.
	newline []byte

	// formatter handles formatting values prior to output.
	formatter Formatter

	// title is the title value.
	title *Value

	// empty is the empty value.
	empty *Value
}

// NewTemplateEncoder creates a new template encoder using the provided options.
func NewTemplateEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	var err error
	enc := &TemplateEncoder{
		resultSet: resultSet,
		newline:   newline,
		formatter: NewEscapeFormatter(),
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}
	for _, o := range opts {
		if err = o(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *TemplateEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TemplateEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}
		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// scanAndFormat scans and formats values from the result set.
func (enc *TemplateEncoder) scanAndFormat(vals []interface{}) ([]*Value, error) {
	var err error
	if err = enc.resultSet.Err(); err != nil {
		return nil, err
	}
	if err = enc.resultSet.Scan(vals...); err != nil {
		return nil, err
	}
	return enc.formatter.Format(vals)
}
