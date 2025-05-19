package tblfmt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"unicode"

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
	// newline is the record separator to use.
	newline []byte
	// border is the display border size.
	border int
	// inline toggles writing the column header names inline with the top line.
	inline bool
	// lineStyle is the table line style.
	lineStyle LineStyle
	// formatter handles formatting values prior to output.
	formatter Formatter
	// skipHeader disables writing header.
	skipHeader bool
	// summary is the summary map.
	summary Summary
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
	// maxWidths are calculated max column widths. Max widths are at least as
	// wide as user-supplied widths
	maxWidths []int
	// minExpandWidth of the table required to switch to the ExpandedEncoder
	// zero disables switching
	minExpandWidth int
	// minPagerWidth of the table required to redirect output to the pager,
	// zero disables pager
	minPagerWidth int
	// minPagerHeight of the table required to redirect output to the pager,
	// zero disables pager
	minPagerHeight int
	// pagerCmd is the pager command to run and redirect output to
	// if height or width is greater than minPagerHeight and minPagerWidth,
	pagerCmd string
	// scanCount is the number of scanned results in the result set.
	scanCount int
	// lowerColumnNames indicates lower casing the column names when column
	// names are all caps.
	lowerColumnNames bool
	// columnTypes is used to build column types for a result set.
	columnTypes func(ResultSet, []any, int) error
	// w is the undelying writer
	w *bufio.Writer
}

// NewTableEncoder creates a new table encoder using the provided options.
//
// The table encoder has a default value of border 1, and a tab width of 8.
func NewTableEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
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
		if err := o.apply(enc); err != nil {
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
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get and check columns
	clen, cols, err := buildColNames(enc.resultSet, enc.lowerColumnNames)
	switch {
	case err != nil:
		return err
	case clen == 0:
		return ErrResultSetHasNoColumns
	}
	// setup offsets, widths
	enc.offsets = make([]int, clen)
	wroteHeader := enc.skipHeader
	// default to user-supplied widths
	enc.maxWidths = make([]int, clen)
	copy(enc.maxWidths, enc.widths)
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	var cmdBuf io.WriteCloser
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
		if enc.minExpandWidth != 0 && enc.tableWidth() >= enc.minExpandWidth {
			t := *enc
			t.formatter = NewEscapeFormatter()
			exp := ExpandedEncoder{
				TableEncoder: t,
			}
			exp.offsets = make([]int, 2)
			exp.maxWidths = make([]int, 2)
			exp.headers, err = exp.formatter.Header(cols)
			if err != nil {
				return err
			}
			exp.calcWidth(vals)
			if exp.pagerCmd != "" && cmd == nil &&
				((exp.minPagerHeight != 0 && exp.tableHeight(vals) >= exp.minPagerHeight) ||
					(exp.minPagerWidth != 0 && exp.tableWidth() >= exp.minPagerWidth)) {
				cmd, cmdBuf, err = startPager(exp.pagerCmd, w)
				if err != nil {
					return err
				}
				exp.w = bufio.NewWriterSize(cmdBuf, 2048)
			}
			if err := exp.encodeVals(vals); err != nil {
				return checkErr(err, cmd)
			}
			continue
		}
		if enc.pagerCmd != "" && cmd == nil &&
			((enc.minPagerHeight != 0 && enc.tableHeight(vals) >= enc.minPagerHeight) ||
				(enc.minPagerWidth != 0 && enc.tableWidth() >= enc.minPagerWidth)) {
			cmd, cmdBuf, err = startPager(enc.pagerCmd, w)
			if err != nil {
				return err
			}
			enc.w = bufio.NewWriterSize(cmdBuf, 2048)
		}
		// print header if not already done
		if !wroteHeader {
			wroteHeader = true
			enc.header()
		}
		if err := enc.encodeVals(vals); err != nil {
			return checkErr(err, cmd)
		}
		// draw end border
		if enc.border >= 2 {
			enc.divider(enc.rowStyle(enc.lineStyle.End))
		}
	}
	// add summary
	if err := summarize(enc.w, enc.summary, enc.scanCount); err != nil {
		return err
	}
	if err := enc.w.Flush(); err != nil {
		return checkErr(err, cmd)
	}
	if cmd != nil {
		_ = cmdBuf.Close()
		return cmd.Wait()
	}
	return nil
}

func (enc *TableEncoder) encodeVals(vals [][]*Value) error {
	rs := enc.rowStyle(enc.lineStyle.Row)
	// print buffered vals
	for i := range vals {
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
	if err := enc.Encode(w); err != nil {
		return err
	}
	for enc.resultSet.NextResultSet() {
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
		if err := enc.Encode(w); err != nil {
			return err
		}
	}
	return nil
}

// nextResults reads the next enc.count values, or all values if enc.count = 0.
func (enc *TableEncoder) nextResults() ([][]*Value, error) {
	var vals [][]*Value
	if enc.count != 0 {
		vals = make([][]*Value, 0, enc.count)
	}
	// read to count (or all)
	var i int
	var r []any
	for enc.resultSet.Next() {
		if i == 0 {
			// set up storage for results
			var err error
			r, err = buildColumnTypes(enc.resultSet, len(enc.headers), enc.columnTypes)
			if err != nil {
				return nil, err
			}
		}
		v, err := scanAndFormat(enc.resultSet, r, enc.formatter, &enc.scanCount)
		if err != nil {
			return vals, err
		}
		vals, i = append(vals, v), i+1
		// read by batches of enc.count rows
		if enc.count != 0 && i%enc.count == 0 {
			break
		}
	}
	return vals, enc.resultSet.Err()
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
		for j := range vals {
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
	if enc.title != nil && enc.title.Width != 0 {
		maxWidth := ((enc.tableWidth() - enc.title.Width) / 2) + enc.title.Width
		enc.writeAligned(enc.title.Buf, rs.filler, AlignRight, maxWidth-enc.title.Width)
		_, _ = enc.w.Write(enc.newline)
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

// rowStyle returns the left, right and midle borders. It also profides the
// filler string, and indicates if this style uses a wrapping indicator.
func (enc *TableEncoder) rowStyle(r [4]rune) rowStyle {
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

// divider draws a divider.
func (enc *TableEncoder) divider(rs rowStyle) {
	// left
	_, _ = enc.w.Write(rs.left)
	for i, width := range enc.maxWidths {
		// column
		_, _ = enc.w.Write(bytes.Repeat(rs.filler, width))
		// line feed indicator
		if rs.hasWrapping && enc.border >= 1 {
			_, _ = enc.w.Write(rs.filler)
		}
		// middle separator
		if i != len(enc.maxWidths)-1 {
			_, _ = enc.w.Write(rs.middle)
		}
	}
	// right
	_, _ = enc.w.Write(rs.right)
}

// tableWidth calculates total table width.
func (enc *TableEncoder) tableWidth() int {
	rs := enc.rowStyle(enc.lineStyle.Mid)
	width := runewidth.StringWidth(string(rs.left)) + runewidth.StringWidth(string(rs.right))
	for i, w := range enc.maxWidths {
		width += w
		if rs.hasWrapping && enc.border >= 1 {
			width++
		}
		if i != len(enc.maxWidths)-1 {
			width += runewidth.StringWidth(string(rs.middle))
		}
	}
	return width
}

// tableHeight calculates total table height.
func (enc *TableEncoder) tableHeight(rows [][]*Value) int {
	height := 0
	if enc.title != nil && enc.title.Width != 0 {
		height += strings.Count(string(enc.title.Buf), "\n")
	}
	// top border
	if enc.border >= 2 && !enc.inline {
		height++
	}
	// header
	height++
	// mid divider
	if enc.inline {
		height++
	}
	for _, row := range rows {
		largest := 1
		for _, cell := range row {
			if cell == nil {
				cell = enc.empty
			}
			if len(cell.Newlines) > largest {
				largest = len(cell.Newlines)
			}
		}
		height += largest
	}
	// end border
	if enc.border >= 2 {
		height++
	}
	// scanCount at this point is not the final value but this is better than nothing
	if enc.summary != nil && enc.summary[-1] != nil || enc.summary[enc.scanCount] != nil {
		height++
	}
	return height
}

// row draws the a table row.
func (enc *TableEncoder) row(vals []*Value, rs rowStyle) {
	var l int
	for {
		// left
		_, _ = enc.w.Write(rs.left)
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
				padding := enc.maxWidths[i] - width
				// no padding for last cell if no border and aligned left
				if enc.border <= 1 && v.Align == AlignLeft && i == len(vals)-1 && (!rs.hasWrapping || l >= len(v.Newlines)) {
					padding = 0
				}
				enc.writeAligned(v.Buf[start:end], rs.filler, v.Align, padding)
			} else if enc.border > 1 || i != len(vals)-1 {
				_, _ = enc.w.Write(bytes.Repeat(rs.filler, enc.maxWidths[i]))
			}
			// write newline wrap value
			if rs.hasWrapping {
				if l < len(v.Newlines) {
					_, _ = enc.w.Write(rs.wrapper)
				} else {
					_, _ = enc.w.Write(rs.filler)
				}
			}
			remaining = remaining || l < len(v.Newlines)
			// middle separator. If border == 0, the new line indicator
			// acts as the middle separator
			if i != len(enc.maxWidths)-1 && enc.border >= 1 {
				_, _ = enc.w.Write(rs.middle)
			}
		}
		// right
		_, _ = enc.w.Write(rs.right)
		if !remaining {
			break
		}
		l++
	}
}

func (enc *TableEncoder) writeAligned(b, filler []byte, a Align, padding int) {
	// calc padding
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
		_, _ = enc.w.Write(bytes.Repeat(filler, paddingLeft))
	}
	// write
	_, _ = enc.w.Write(b)
	// add padding right
	if paddingRight > 0 {
		_, _ = enc.w.Write(bytes.Repeat(filler, paddingRight))
	}
}

// rowStyle is the row style for a row, as arrays of bytes to print.
type rowStyle struct {
	left, right, middle, filler, wrapper []byte
	hasWrapping                          bool
}

// ExpandedEncoder is a buffered, lookahead expanded table encoder for result sets.
type ExpandedEncoder struct {
	TableEncoder
}

// NewExpandedEncoder creates a new expanded table encoder using the provided options.
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
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get and check columns
	clen, cols, err := buildColNames(enc.resultSet, enc.lowerColumnNames)
	switch {
	case err != nil:
		return err
	case clen == 0:
		return ErrResultSetHasNoColumns
	}
	// setup offsets, widths
	enc.offsets = make([]int, 2)
	enc.maxWidths = make([]int, 2)
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	var cmdBuf io.WriteCloser
	wroteTitle := enc.skipHeader
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
		if enc.pagerCmd != "" && cmd == nil &&
			((enc.minPagerHeight != 0 && enc.tableHeight(vals) >= enc.minPagerHeight) ||
				(enc.minPagerWidth != 0 && enc.tableWidth() >= enc.minPagerWidth)) {
			cmd, cmdBuf, err = startPager(enc.pagerCmd, w)
			if err != nil {
				return err
			}
			enc.w = bufio.NewWriterSize(cmdBuf, 2048)
		}
		// print title if not already done
		if !wroteTitle {
			wroteTitle = true
			if enc.title != nil && enc.title.Width != 0 {
				_, _ = enc.w.Write(enc.title.Buf)
				_, _ = enc.w.Write(enc.newline)
			}
		}
		if err := enc.encodeVals(vals); err != nil {
			return checkErr(err, cmd)
		}
	}
	// add summary
	if err := summarize(w, enc.summary, enc.scanCount); err != nil {
		return err
	}
	if err := enc.w.Flush(); err != nil {
		return checkErr(err, cmd)
	}
	if cmd != nil {
		_ = cmdBuf.Close()
		return cmd.Wait()
	}
	return nil
}

func (enc *ExpandedEncoder) encodeVals(vals [][]*Value) error {
	rs := enc.rowStyle(enc.lineStyle.Row)
	// print buffered vals
	for i := range vals {
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
	if err := enc.Encode(w); err != nil {
		return err
	}
	for enc.resultSet.NextResultSet() {
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
		if err := enc.Encode(w); err != nil {
			return err
		}
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

// tableHeight calculates total table height.
func (enc *ExpandedEncoder) tableHeight(rows [][]*Value) int {
	height := 0
	if enc.title != nil && enc.title.Width != 0 {
		height += strings.Count(string(enc.title.Buf), "\n")
	}
	for _, row := range rows {
		// header
		height++
		for _, cell := range row {
			if cell == nil {
				cell = enc.empty
			}
			height += 1 + len(cell.Newlines)
		}
	}
	// end border
	if enc.border >= 2 {
		height++
	}
	// scanCount at this point is not the final value but this is better than nothing
	if enc.summary != nil && enc.summary[-1] != nil || enc.summary[enc.scanCount] != nil {
		height++
	}
	return height
}

func (enc *ExpandedEncoder) record(i int, vals []*Value, rs rowStyle) {
	if !enc.skipHeader {
		// write record header as a single record
		headerRS := rs
		header := enc.recordHeader(i)
		if enc.border != 0 {
			headerRS = enc.rowStyle(enc.lineStyle.Top)
			if i != 0 {
				headerRS = enc.rowStyle(enc.lineStyle.Mid)
			}
		}
		_, _ = enc.w.Write(headerRS.left)
		_, _ = enc.w.WriteString(header)
		padding := enc.maxWidths[0] + enc.maxWidths[1] + runewidth.StringWidth(string(headerRS.middle))*2 - len(header) - 1
		if padding > 0 {
			_, _ = enc.w.Write(bytes.Repeat(headerRS.filler, padding))
		}
		// write newline wrap value
		_, _ = enc.w.Write(headerRS.filler)
		_, _ = enc.w.Write(headerRS.right)
	}
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
	// newline is the record separator to use.
	newline []byte
	// formatter handles formatting values prior to output.
	formatter Formatter
	// empty is the empty value.
	empty *Value
	// lowerColumnNames indicates lower casing the column names when column
	// names are all caps.
	lowerColumnNames bool
	// columnTypes is used to build column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewJSONEncoder creates a new JSON encoder using the provided options.
func NewJSONEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	enc := &JSONEncoder{
		resultSet: resultSet,
		newline:   newline,
		formatter: NewEscapeFormatter(WithIsJSON(true)),
		empty: &Value{
			Buf:  []byte("null"),
			Tabs: make([][][2]int, 1),
			Raw:  true,
		},
	}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
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
	var (
		start = []byte{'['}
		end   = []byte{']'}
		open  = []byte{'{'}
		cls   = []byte{'}'}
		q     = []byte{'"'}
		cma   = []byte{','}
	)
	// get and check columns
	clen, cols, err := buildColNames(enc.resultSet, enc.lowerColumnNames)
	switch {
	case err != nil:
		return err
	case clen == 0:
		return ErrResultSetHasNoColumns
	}
	cb := make([][]byte, clen)
	for i := range clen {
		if cb[i], err = json.Marshal(cols[i]); err != nil {
			return err
		}
		cb[i] = append(cb[i], ':')
	}
	// set up storage for results
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
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
		vals, err = scanAndFormat(enc.resultSet, r, enc.formatter, &count)
		if err != nil {
			return err
		}
		if _, err = w.Write(open); err != nil {
			return err
		}
		for i := range clen {
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
	err = enc.resultSet.Err()
	if err != nil {
		return err
	}
	// end
	_, err = w.Write(end)
	return err
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *JSONEncoder) EncodeAll(w io.Writer) error {
	if err := enc.Encode(w); err != nil {
		return err
	}
	for enc.resultSet.NextResultSet() {
		if _, err := w.Write([]byte{','}); err != nil {
			return err
		}
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
		if err := enc.Encode(w); err != nil {
			return err
		}
	}
	if _, err := w.Write(enc.newline); err != nil {
		return err
	}
	return nil
}

// UnalignedEncoder is an unbuffered, unaligned encoder for result sets.
//
// Provides a way of encoding unaligned result sets in formats such as
// comma-separated value (CSV) or tab-separated value (TSV) files.
//
// By default uses a field separator of '|', no quote separator, and record
// separator using the default newline for the platfom ("\r\n" on Windows, "\n"
// otherwise).
type UnalignedEncoder struct {
	// resultSet is the result set to encode.
	resultSet ResultSet
	// sep is the separator to use.
	sep rune
	// quote is the quote to use.
	quote rune
	// newline is the record separator to use.
	newline []byte
	// formatter handles formatting values prior to output.
	formatter Formatter
	// skipHeader disables writing header.
	skipHeader bool
	// summary is the summary map.
	summary map[int]func(io.Writer, int) (int, error)
	// empty is the empty value.
	empty *Value
	// lowerColumnNames indicates lower casing the column names when column
	// names are all caps.
	lowerColumnNames bool
	// columnTypes is used to build column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewUnalignedEncoder creates a new unaligned encoder using the provided
// options.
func NewUnalignedEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	sep, quote := rune('|'), rune(0)
	enc := &UnalignedEncoder{
		resultSet: resultSet,
		sep:       sep,
		quote:     quote,
		newline:   newline,
		formatter: NewEscapeFormatter(WithIsRaw(true, sep, quote)),
		summary:   DefaultTableSummary(),
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// NewCSVEncoder creates a new csv encoder using the provided options.
//
// Creates an unaligned encoder using the default field separator ',' and field
// quote of '"'.
func NewCSVEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	sep, quote := rune(','), rune('"')
	enc := &UnalignedEncoder{
		resultSet: resultSet,
		sep:       sep,
		quote:     quote,
		newline:   newline,
		formatter: NewEscapeFormatter(WithIsRaw(true, sep, quote)),
		summary:   Summary{},
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *UnalignedEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get and check columns
	clen, cols, err := buildColNames(enc.resultSet, enc.lowerColumnNames)
	switch {
	case err != nil:
		return err
	case clen == 0:
		return ErrResultSetHasNoColumns
	}
	sep, quote := []byte(string(enc.sep)), []byte(string(enc.quote))
	// write header
	if !enc.skipHeader {
		headers, err := enc.formatter.Header(cols)
		if err != nil {
			return err
		}
		for i := range clen {
			if i != 0 {
				if _, err := w.Write(sep); err != nil {
					return err
				}
			}
			buf := headers[i].Buf
			if enc.quote != 0 && headers[i].Quoted {
				buf = append(quote, append(buf, quote...)...)
			}
			if _, err := w.Write(buf); err != nil {
				return err
			}
		}
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
	}
	// set up storage for results
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// process
	var count int
	for enc.resultSet.Next() {
		vals, err := scanAndFormat(enc.resultSet, r, enc.formatter, &count)
		if err != nil {
			return err
		}
		for i := range clen {
			if i != 0 {
				if _, err := w.Write(sep); err != nil {
					return err
				}
			}
			v := vals[i]
			if v == nil {
				v = enc.empty
			}
			buf := v.Buf
			if enc.quote != 0 && v.Quoted {
				buf = append(quote, append(buf, quote...)...)
			}
			if _, err := w.Write(buf); err != nil {
				return err
			}
		}
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
	}
	if err := summarize(w, enc.summary, count); err != nil {
		return err
	}
	return enc.resultSet.Err()
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *UnalignedEncoder) EncodeAll(w io.Writer) error {
	if err := enc.Encode(w); err != nil {
		return err
	}
	for enc.resultSet.NextResultSet() {
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
		if err := enc.Encode(w); err != nil {
			return err
		}
	}
	return nil
}

// TemplateEncoder is an unbuffered template encoder for result sets.
type TemplateEncoder struct {
	// ResultSet is the result set to encode.
	resultSet ResultSet
	// executor is the template executor function.
	executor func(io.Writer, any) error
	// newline is the record separator to use.
	newline []byte
	// formatter handles formatting values prior to output.
	formatter Formatter
	// title is the title value.
	title *Value
	// empty is the empty value.
	empty *Value
	// skipHeader disables writing header.
	skipHeader bool
	// attributes are extra table attributes.
	attributes string
	// lowerColumnNames indicates lower casing the column names when column
	// names are all caps.
	lowerColumnNames bool
	// columnTypes is used to build column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewTemplateEncoder creates a new template encoder using the provided options.
func NewTemplateEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	enc := &TemplateEncoder{
		resultSet: resultSet,
		executor:  func(io.Writer, any) error { return ErrInvalidTemplate },
		newline:   newline,
		formatter: NewEscapeFormatter(),
		empty: &Value{
			Buf: []byte(""),
		},
	}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

// NewHTMLEncoder creates a new html template encoder using the provided
// options.
func NewHTMLEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("html")}, opts...)...)
}

// NewAsciiDocEncoder creates a new asciidoc template encoder using the
// provided options.
func NewAsciiDocEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("asciidoc")}, opts...)...)
}

// NewVerticalEncoder creates a new vertical template encoder using the
// provided options.
func NewVerticalEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("vertical")}, opts...)...)
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *TemplateEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get and check columns
	clen, cols, err := buildColNames(enc.resultSet, enc.lowerColumnNames)
	switch {
	case err != nil:
		return err
	case clen == 0:
		return ErrResultSetHasNoColumns
	}
	headers, err := enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	for i := range clen {
		if headers[i] == nil {
			headers[i] = enc.empty
		}
	}
	// set up storage for results
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// process
	var rows [][]*Value
	var count int
	for enc.resultSet.Next() {
		vals, err := scanAndFormat(enc.resultSet, r, enc.formatter, &count)
		if err != nil {
			return err
		}
		for i := range clen {
			if vals[i] == nil {
				vals[i] = enc.empty
			}
		}
		rows = append(rows, vals)
	}
	if err := enc.resultSet.Err(); err != nil {
		return err
	}
	title := enc.title
	if title == nil {
		title = enc.empty
	}
	return enc.executor(w, map[string]any{
		"Attributes": enc.attributes,
		"Headers":    headers,
		"Rows":       rows,
		"SkipHeader": enc.skipHeader,
		"Title":      title,
	})
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TemplateEncoder) EncodeAll(w io.Writer) error {
	if err := enc.Encode(w); err != nil {
		return err
	}
	for enc.resultSet.NextResultSet() {
		if _, err := w.Write(enc.newline); err != nil {
			return err
		}
		if err := enc.Encode(w); err != nil {
			return err
		}
	}
	return nil
}

// errEncoder provides a no-op encoder that always returns the wrapped error.
type errEncoder struct {
	err error
}

// Encode satisfies the Encoder interface.
func (err *errEncoder) Encode(io.Writer) error {
	return err.err
}

// EncodeAll satisfies the Encoder interface.
func (err *errEncoder) EncodeAll(io.Writer) error {
	return err.err
}

// newErrEncoder creates a no-op error encoder.
func newErrEncoder(_ ResultSet, opts ...Option) (Encoder, error) {
	enc := &errEncoder{}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
			return nil, err
		}
	}
	return enc, enc.err
}

// scanAndFormat scans and formats values from the result set.
func scanAndFormat(resultSet ResultSet, vals []any, formatter Formatter, count *int) ([]*Value, error) {
	if err := resultSet.Err(); err != nil {
		return nil, err
	}
	if err := resultSet.Scan(vals...); err != nil {
		return nil, err
	}
	*count++
	return formatter.Format(vals)
}

// buildColNames builds the column names for the result set.
func buildColNames(resultSet ResultSet, lower bool) (int, []string, error) {
	cols, err := resultSet.Columns()
	if err != nil {
		return 0, nil, err
	}
	clen := len(cols)
	if lower {
		for i, s := range cols {
			if j := strings.IndexFunc(s, func(r rune) bool {
				return unicode.IsLetter(r) && unicode.IsLower(r)
			}); j == -1 {
				cols[i] = strings.ToLower(s)
			}
		}
	}
	return clen, cols, nil
}

// buildColumnTypes builds a []interface{} for storing scan results.
func buildColumnTypes(resultSet ResultSet, n int, columnTypes func(ResultSet, []any, int) error) ([]any, error) {
	r := make([]any, n)
	if columnTypes != nil {
		if err := columnTypes(resultSet, r, n); err != nil {
			return nil, err
		}
		return r, nil
	}
	for i := range n {
		r[i] = new(any)
	}
	return r, nil
}

// summarize writes the table scan count summary.
func summarize(w io.Writer, summary Summary, count int) error {
	// do summary
	if summary == nil {
		return nil
	}
	var f func(io.Writer, int) (int, error)
	if z, ok := summary[-1]; ok {
		f = z
	}
	if z, ok := summary[count]; ok {
		f = z
	}
	if f != nil {
		if _, err := f(w, count); err != nil {
			return err
		}
		if _, err := w.Write(newline); err != nil {
			return err
		}
	}
	return nil
}

func startPager(pagerCmd string, w io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	cmd := exec.Command(pagerCmd)
	cmd.Stdout = w
	cmd.Stderr = w
	cmdBuf, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	return cmd, cmdBuf, cmd.Start()
}

func checkErr(err error, cmd *exec.Cmd) error {
	if cmd != nil && errors.Is(err, syscall.EPIPE) {
		// broken pipe means pager quit before consuming all data, which might be expected
		return nil
	}
	return err
}
