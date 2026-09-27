package tblfmt

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	runewidth "github.com/mattn/go-runewidth"
)

// TableEncoder is an encoder that writes a result set as a table. It buffers
// a batch of rows ahead to find the widths of the columns.
type TableEncoder struct {
	// resultSet is the result set to encode.
	resultSet ResultSet
	// count is the number of rows in a batch. The encoder scans and buffers
	// up to count rows ahead to find the maximum widths of the columns that
	// the formatter of the encoder returns.
	//
	// Note: if count is 0, the encoder scans and buffers all the rows before
	// it encodes the table.
	count int
	// tab is the tab width.
	tab int
	// newline is the record separator.
	newline []byte
	// border is the display border size.
	border int
	// inline turns on writing the header inline with the top line.
	inline bool
	// lineStyle is the line style of the table.
	lineStyle LineStyle
	// formatter formats the values before the encoder writes them.
	formatter Formatter
	// skipHeader turns off the header.
	skipHeader bool
	// summary is the summary map.
	summary Summary
	// isCustomSummary is true if an option set the summary.
	isCustomSummary bool
	// title is the title value.
	title *Value
	// empty is the empty value.
	empty *Value
	// headers contains the formatted column names.
	headers []*Value
	// offsets are the column offsets.
	offsets []int
	// widths are the column widths that the user set.
	widths []int
	// maxWidths are the calculated maximum widths of the columns. Each one is
	// at least the width that the user set.
	maxWidths []int
	// aligns are the calculated column alignments, followed by a column's null
	// values, which have no type of their own to align by.
	aligns []Align
	// minExpandWidth is the table width at which the encoder switches to the
	// ExpandedEncoder. Zero turns off the switch.
	minExpandWidth int
	// minPagerWidth is the table width at which the encoder sends its output
	// to the pager. Zero turns off the width test. The height test can still
	// start the pager.
	minPagerWidth int
	// minPagerHeight is the table height at which the encoder sends its
	// output to the pager. Zero turns off the height test. The width test can
	// still start the pager.
	minPagerHeight int
	// pagerCmd is the pager command. The encoder starts it when the table
	// height is minPagerHeight or more, or when the table width is
	// minPagerWidth or more.
	pagerCmd string
	// scanCount is the number of rows that the encoder scanned from the result
	// set.
	scanCount int
	// headerTransformer is the transformer for the column names.
	headerTransformer Transformer
	// columnTypes builds the column types for a result set.
	columnTypes func(ResultSet, []any, int) error
	// w is the underlying writer
	w *bufio.Writer
}

// NewTableEncoder creates a table encoder with the options.
//
// By default, the table encoder has a border of 1 and a tab width of 8.
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
	// make sure that each rune of the line style has a width of 1
	// TODO: remove this loop
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

// Encode encodes one result set to the writer with the options of the
// encoder.
func (enc *TableEncoder) Encode(w io.Writer) error {
	// reset scan count
	enc.scanCount = 0
	enc.w = bufio.NewWriterSize(w, 2048)
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get the column names, and stop on an error
	clen, cols, err := buildColNames(enc.resultSet, enc.headerTransformer)
	if err != nil {
		return err
	}
	// set up the offsets and the widths
	enc.offsets = make([]int, clen)
	wroteHeader := enc.skipHeader
	// start with the widths that the user set
	enc.maxWidths = make([]int, clen)
	enc.aligns = make([]Align, clen)
	copy(enc.maxWidths, enc.widths)
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	var cmdBuf io.WriteCloser
	for {
		var vals [][]*Value
		// read the next batch
		vals, err = enc.nextResults()
		if err != nil {
			return err
		}
		// no more rows
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
			exp.aligns = make([]Align, 2)
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
		// write the header if the encoder did not write it yet
		if !wroteHeader {
			wroteHeader = true
			enc.header()
		}
		if err := enc.encodeVals(vals); err != nil {
			return checkErr(err, cmd)
		}
		// draw the end border
		if enc.border >= 2 {
			enc.divider(enc.rowStyle(enc.lineStyle.End))
		}
	}
	// psql draws the lines of a result set that has no columns and no rows.
	//
	// Note: psql also draws the header of a result set that has columns and
	// no rows, and this encoder does not. See B1 in docs/BACKLOG.md.
	if clen == 0 && enc.scanCount == 0 {
		if !wroteHeader {
			enc.header()
		}
		if enc.border >= 2 {
			enc.divider(enc.rowStyle(enc.lineStyle.End))
		}
	}
	// write the summary
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
	// write the rows of the batch
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

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
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

// nextResults reads the next batch of enc.count rows, or all the rows if
// enc.count is 0.
func (enc *TableEncoder) nextResults() ([][]*Value, error) {
	var vals [][]*Value
	if enc.count != 0 {
		vals = make([][]*Value, 0, enc.count)
	}
	// read enc.count rows, or all the rows
	var i int
	var r []any
	for enc.resultSet.Next() {
		if i == 0 {
			// set up the storage for the scanned values
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
		// read in batches of enc.count rows
		if enc.count != 0 && i%enc.count == 0 {
			break
		}
	}
	return vals, enc.resultSet.Err()
}

func (enc *TableEncoder) calcWidth(vals [][]*Value) {
	// calculate the offsets and the widths for this batch
	var offset int
	rs := enc.rowStyle(enc.lineStyle.Row)
	offset += runewidth.StringWidth(string(rs.left))
	for i, h := range enc.headers {
		if i != 0 {
			offset += runewidth.StringWidth(string(rs.middle))
		}
		// store the offset
		enc.offsets[i] = offset
		// the width of the header is the minimum
		enc.maxWidths[i] = max(enc.maxWidths[i], h.MaxWidth(offset, enc.tab))
		// find the maximum width of the column, from the first row to the last
		for j := range vals {
			cell := vals[j][i]
			if cell == nil {
				cell = enc.empty
			}
			enc.maxWidths[i] = max(enc.maxWidths[i], cell.MaxWidth(offset, enc.tab))
		}
		// a null value has no type of its own to align by, so it follows the
		// column, the way psql aligns the null string by the column's type. A
		// batch that has no non-null value keeps the alignment of an earlier
		// batch.
		if align, ok := columnAlign(vals, i); ok {
			enc.aligns[i] = align
		}
		// add the width of the column, and one space for the wrap indicator
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
	// draw the top border
	if enc.border >= 2 && !enc.inline {
		enc.divider(enc.rowStyle(enc.lineStyle.Top))
	}
	// draw the header with the style of the top line
	if enc.inline {
		rs = enc.rowStyle(enc.lineStyle.Top)
	}
	// write the header
	enc.row(enc.headers, rs)
	if !enc.inline {
		// draw the middle divider
		enc.divider(enc.rowStyle(enc.lineStyle.Mid))
	}
}

// rowStyle returns the left, right, and middle borders. It also returns the
// filler string, and tells if this style uses a wrap indicator.
func (enc *TableEncoder) rowStyle(r [4]rune) rowStyle {
	var left, right, middle, spacer, filler string
	spacer = strings.Repeat(string(r[1]), runewidth.RuneWidth(enc.lineStyle.Row[1]))
	filler = string(r[1])
	// for compact output, r[1] is \0
	if r[1] == 0 {
		filler = " "
	}
	// outside borders
	if enc.border > 1 {
		left = string(r[0])
		right = string(r[3])
	}
	// if the border is set, add a spacer at the start
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
		// wrap indicator
		if rs.hasWrapping && enc.border >= 1 {
			_, _ = enc.w.Write(rs.filler)
		}
		// middle separator
		if i != len(enc.maxWidths)-1 {
			_, _ = enc.w.Write(rs.middle)
		}
	}
	// psql draws one more filler when there are no columns
	if len(enc.maxWidths) == 0 && rs.hasWrapping && enc.border >= 1 {
		_, _ = enc.w.Write(rs.filler)
	}
	// right
	_, _ = enc.w.Write(rs.right)
}

// tableWidth calculates the total width of the table.
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
	// the filler that divider draws when there are no columns
	if len(enc.maxWidths) == 0 && rs.hasWrapping && enc.border >= 1 {
		width++
	}
	return width
}

// tableHeight calculates the total height of the table.
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
	// scanCount is not the final count at this point, but it is better than no count
	if enc.summary != nil && enc.summary[-1] != nil || enc.summary[enc.scanCount] != nil {
		height++
	}
	return height
}

// row draws a table row.
//
// Note: a row that has no values draws no line, as in psql.
func (enc *TableEncoder) row(vals []*Value, rs rowStyle) {
	if len(vals) == 0 {
		return
	}
	var l int
	for {
		// left
		_, _ = enc.w.Write(rs.left)
		var remaining bool
		for i, v := range vals {
			align := enc.aligns[i]
			if v == nil {
				v = enc.empty
			} else {
				align = v.Align
			}
			// write value
			if l <= len(v.Newlines) {
				// find the start, the end, and the width
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
				// no padding for the last value if there is no outside border and
				// the value is left aligned
				if enc.border <= 1 && align == AlignLeft && i == len(vals)-1 && (!rs.hasWrapping || l >= len(v.Newlines)) {
					padding = 0
				}
				enc.writeAligned(v.Buf[start:end], rs.filler, align, padding)
			} else if enc.border > 1 || i != len(vals)-1 {
				_, _ = enc.w.Write(bytes.Repeat(rs.filler, enc.maxWidths[i]))
			}
			// write the wrap indicator, or a filler
			if rs.hasWrapping {
				if l < len(v.Newlines) {
					_, _ = enc.w.Write(rs.wrapper)
				} else {
					_, _ = enc.w.Write(rs.filler)
				}
			}
			remaining = remaining || l < len(v.Newlines)
			// middle separator. If the border is 0, the wrap indicator is the
			// middle separator
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
	// calculate the padding
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
	// add the left padding
	if paddingLeft > 0 {
		_, _ = enc.w.Write(bytes.Repeat(filler, paddingLeft))
	}
	// write
	_, _ = enc.w.Write(b)
	// add the right padding
	if paddingRight > 0 {
		_, _ = enc.w.Write(bytes.Repeat(filler, paddingRight))
	}
}

// rowStyle is the style for a row, as the bytes to write.
type rowStyle struct {
	left, right, middle, filler, wrapper []byte
	hasWrapping                          bool
}

// ExpandedEncoder is an encoder that writes a result set as an expanded
// table. It writes each row as a record, with one line for each column. It
// buffers a batch of rows ahead, as TableEncoder does.
type ExpandedEncoder struct {
	TableEncoder
}

// NewExpandedEncoder creates an expanded table encoder with the options.
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

// Encode encodes one result set to the writer with the options of the
// encoder.
func (enc *ExpandedEncoder) Encode(w io.Writer) error {
	// reset scan count
	enc.scanCount = 0
	enc.w = bufio.NewWriterSize(w, 2048)
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get the column names, and stop on an error
	clen, cols, err := buildColNames(enc.resultSet, enc.headerTransformer)
	if err != nil {
		return err
	}
	// set up the offsets and the widths
	enc.offsets = make([]int, 2)
	enc.maxWidths = make([]int, 2)
	enc.aligns = make([]Align, 2)
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	var cmdBuf io.WriteCloser
	wroteTitle := enc.skipHeader
	for {
		var vals [][]*Value
		// read the next batch
		vals, err = enc.nextResults()
		if err != nil {
			return err
		}
		// no more rows
		if len(vals) == 0 {
			break
		}
		// psql writes no record for a result set that has no columns. It
		// writes only the summary.
		if clen == 0 {
			continue
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
		// write the title if the encoder did not write it yet
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
	// write the summary
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
	// write the rows of the batch
	for i := range vals {
		enc.record(i, vals[i], rs)
		if i+1%1000 == 0 {
			// check error every 1k rows
			if err := enc.w.Flush(); err != nil {
				return err
			}
		}
	}
	// draw the end border
	if enc.border >= 2 && enc.scanCount != 0 {
		enc.divider(enc.rowStyle(enc.lineStyle.End))
	}
	return nil
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
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
	// the first column is always the column name
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
	// the second column is as wide as the widest value in any row, but not
	// narrower than the record header
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

// tableHeight calculates the total height of the table.
func (enc *ExpandedEncoder) tableHeight(rows [][]*Value) int {
	height := 0
	if enc.title != nil && enc.title.Width != 0 {
		height += strings.Count(string(enc.title.Buf), "\n")
	}
	for _, row := range rows {
		// record header
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
	// scanCount is not the final count at this point, but it is better than no count
	if enc.summary != nil && enc.summary[-1] != nil || enc.summary[enc.scanCount] != nil {
		height++
	}
	return height
}

func (enc *ExpandedEncoder) record(i int, vals []*Value, rs rowStyle) {
	if !enc.skipHeader {
		// write the record header as one line
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
		// write a filler in place of the wrap indicator
		_, _ = enc.w.Write(headerRS.filler)
		_, _ = enc.w.Write(headerRS.right)
	}
	// write each value, with the column name in the first column
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

// columnAlign returns the alignment shared by column i's non-null values, and
// whether the column had any to share.
//
// Note: a column whose values do not agree on an alignment has none to pass
// on, and is left aligned.
func columnAlign(vals [][]*Value, i int) (Align, bool) {
	var align Align
	var found bool
	for j := range vals {
		switch cell := vals[j][i]; {
		case cell == nil:
		case !found:
			align, found = cell.Align, true
		case cell.Align != align:
			return AlignLeft, true
		}
	}
	return align, found
}

// JSONEncoder is an encoder that writes a result set as JSON. It does not
// buffer rows.
type JSONEncoder struct {
	resultSet ResultSet
	// newline is the record separator.
	newline []byte
	// formatter formats the values before the encoder writes them.
	formatter Formatter
	// empty is the empty value.
	empty *Value
	// headerTransformer is the transformer for the column names.
	headerTransformer Transformer
	// columnTypes builds the column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewJSONEncoder creates a JSON encoder with the options.
func NewJSONEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	enc := &JSONEncoder{
		resultSet: resultSet,
		newline:   newline,
		// note: the prefix is the indent of a value, so that the lines of a
		// nested object or array line up under the value.
		formatter: NewEscapeFormatter(WithIsJSON(true), WithJSONConfig(jsonValueIndent, jsonIndent, false)),
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

// jsonIndent is the indent for one level of the output of the JSON encoder.
// jsonRowIndent is the indent of a row, and jsonValueIndent is the indent of a
// value.
const (
	jsonIndent      = "  "
	jsonRowIndent   = jsonIndent
	jsonValueIndent = jsonIndent + jsonIndent
)

// Encode encodes one result set to the writer with the options of the
// encoder.
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
		rowNL = append(append([]byte{}, enc.newline...), jsonRowIndent...)
		valNL = append(append([]byte{}, enc.newline...), jsonValueIndent...)
	)
	// get the column names, and stop on an error
	clen, cols, err := buildColNames(enc.resultSet, enc.headerTransformer)
	if err != nil {
		return err
	}
	cb := make([][]byte, clen)
	for i := range clen {
		// note: matches the v1 encoding/json defaults, which escaped HTML and
		// replaced invalid UTF-8 rather than rejecting it.
		if cb[i], err = json.Marshal(cols[i], jsontext.AllowInvalidUTF8(true), jsontext.EscapeForHTML(true)); err != nil {
			return err
		}
		cb[i] = append(cb[i], ':', ' ')
	}
	// set up the storage for the scanned values
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// start
	if _, err = w.Write(start); err != nil {
		return err
	}
	// write the rows
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
		if _, err = w.Write(rowNL); err != nil {
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
			if _, err = w.Write(valNL); err != nil {
				return err
			}
			if _, err = w.Write(cb[i]); err != nil {
				return err
			}
			// if the value is raw, write it as is
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
		// a row that has no columns is {}
		if clen != 0 {
			if _, err = w.Write(rowNL); err != nil {
				return err
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
	// end. A result set that has no rows stays [], with no newline in it.
	if count != 0 {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}
	}
	_, err = w.Write(end)
	return err
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
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

// UnalignedEncoder is an encoder that writes a result set with no alignment.
// It does not buffer rows.
//
// You can use it to encode a result set in formats such as comma-separated
// values (CSV) or tab-separated values (TSV).
//
// By default, the field separator is '|', there is no quote character, and the
// record separator is the default newline for the platform ("\r\n" on Windows,
// "\n" otherwise).
type UnalignedEncoder struct {
	// resultSet is the result set to encode.
	resultSet ResultSet
	// sep is the field separator.
	sep rune
	// quote is the quote character.
	quote rune
	// newline is the record separator.
	newline []byte
	// formatter formats the values before the encoder writes them.
	formatter Formatter
	// skipHeader turns off the header.
	skipHeader bool
	// summary is the summary map.
	summary map[int]func(io.Writer, int) (int, error)
	// empty is the empty value.
	empty *Value
	// headerTransformer is the transformer for the column names.
	headerTransformer Transformer
	// columnTypes builds the column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewUnalignedEncoder creates an unaligned encoder with the options.
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

// NewCSVEncoder creates a CSV encoder with the options.
//
// It creates an unaligned encoder. By default, the field separator is ',' and
// the field quote is '"'.
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

// Encode encodes one result set to the writer with the options of the
// encoder.
func (enc *UnalignedEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get the column names, and stop on an error
	clen, cols, err := buildColNames(enc.resultSet, enc.headerTransformer)
	if err != nil {
		return err
	}
	sep, quote := []byte(string(enc.sep)), []byte(string(enc.quote))
	// write the header
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
	// set up the storage for the scanned values
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// write the rows
	var count int
	for enc.resultSet.Next() {
		vals, err := scanAndFormat(enc.resultSet, r, enc.formatter, &count)
		switch {
		case err != nil:
			return err
		case clen == 0:
			// psql writes no line for a row that has no columns
			continue
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

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
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

// TemplateEncoder is a template encoder for result sets.
//
// Note: the encoder reads every row of a result set into memory before it
// runs the template, because the template receives all the rows at once.
type TemplateEncoder struct {
	// resultSet is the result set to encode.
	resultSet ResultSet
	// executor is the function that runs the template.
	executor func(io.Writer, *Template) error
	// newline is the record separator.
	newline []byte
	// formatter formats the values before the encoder writes them.
	formatter Formatter
	// title is the title value.
	title *Value
	// empty is the empty value.
	empty *Value
	// skipHeader turns off the header.
	skipHeader bool
	// attributes are extra table attributes.
	attributes string
	// headerTransformer is the transformer for the column names.
	headerTransformer Transformer
	// columnTypes builds the column types for a result set.
	columnTypes func(ResultSet, []any, int) error
}

// NewTemplateEncoder creates a template encoder with the options.
func NewTemplateEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	enc := &TemplateEncoder{
		resultSet: resultSet,
		executor:  func(io.Writer, *Template) error { return ErrInvalidTemplate },
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

// NewHTMLEncoder creates a template encoder for HTML with the options.
func NewHTMLEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("html")}, opts...)...)
}

// NewAsciiDocEncoder creates a template encoder for AsciiDoc with the
// options.
func NewAsciiDocEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("asciidoc")}, opts...)...)
}

// NewVerticalEncoder creates a vertical template encoder with the options.
func NewVerticalEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("vertical")}, opts...)...)
}

// Encode encodes one result set to the writer with the options of the
// encoder.
func (enc *TemplateEncoder) Encode(w io.Writer) error {
	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}
	// get the column names, and stop on an error
	clen, cols, err := buildColNames(enc.resultSet, enc.headerTransformer)
	if err != nil {
		return err
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
	// set up the storage for the scanned values
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// write the rows
	var rows [][]*Value
	var count int
	for enc.resultSet.Next() {
		vals, err := scanAndFormat(enc.resultSet, r, enc.formatter, &count)
		if err != nil {
			return err
		}
		rows = append(rows, vals)
	}
	if err := enc.resultSet.Err(); err != nil {
		return err
	}
	// a null value has no type of its own to align by, so it follows the
	// column, the way psql aligns the null string by the column's type
	for i := range clen {
		empty := enc.empty
		if align, ok := columnAlign(rows, i); ok && align != empty.Align {
			v := *empty
			v.Align = align
			empty = &v
		}
		for _, vals := range rows {
			if vals[i] == nil {
				vals[i] = empty
			}
		}
	}
	title := enc.title
	if title == nil {
		title = enc.empty
	}
	return enc.executor(w, &Template{
		Attributes: enc.attributes,
		Headers:    headers,
		Rows:       rows,
		SkipHeader: enc.skipHeader,
		Title:      title,
	})
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
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

// errEncoder is an encoder that does nothing and always returns the wrapped
// error.
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

// newErrEncoder creates an errEncoder, which does nothing.
func newErrEncoder(_ ResultSet, opts ...Option) (Encoder, error) {
	enc := &errEncoder{}
	for _, o := range opts {
		if err := o.apply(enc); err != nil {
			return nil, err
		}
	}
	return enc, enc.err
}

// scanAndFormat scans the values of one row from the result set, and formats
// them.
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
func buildColNames(resultSet ResultSet, transformer Transformer) (int, []string, error) {
	cols, err := resultSet.Columns()
	if err != nil {
		return 0, nil, err
	}
	clen := len(cols)
	if transformer != nil {
		for i := range clen {
			cols[i] = transformer.Transform(cols[i])
		}
	}
	return clen, cols, nil
}

// buildColumnTypes builds a []interface{} to store the scanned values.
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

// summarize writes the summary for the count of scanned rows.
func summarize(w io.Writer, summary Summary, count int) error {
	// write the summary
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
		// a broken pipe means that the pager stopped before it read all the
		// data, and this can be normal
		return nil
	}
	return err
}
