package tblfmt

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

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
	// newline is the record separator.
	newline []byte
	// resultSetSep is written between two result sets. When it is nil, the
	// encoder writes newline.
	resultSetSep []byte
	// border is the display border size.
	border int
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
	// widths are the column widths that the user set.
	widths []int
	// maxWidths are the calculated maximum widths of the columns. Each one is
	// at least the width that the user set.
	maxWidths []int
	// colWidths are the widths that the encoder draws the columns with. They
	// are maxWidths, unless wrapWidth makes a column narrower.
	colWidths []int
	// headerWidths are the widths of the column names, and sums are the sums
	// of the widths of the values in each column. psql wraps a column by
	// their average.
	headerWidths []int
	sums         []int
	// wide is true for each column that holds a character 2 columns wide.
	// Wrapping does not make such a column narrower than 2.
	wide []bool
	// wrapWidth is the width that the encoder wraps the table to, as the
	// psql format wrapped does. Zero turns off wrapping.
	wrapWidth int
	// columnAlign sets the alignment of a column from its type, and fixed
	// holds what it set for the columns of the result set.
	columnAlign func(*sql.ColumnType) (Align, bool)
	fixed       []fixedAlign
	// aligns are the calculated column alignments, followed by a column's null
	// values, which have no type of their own to align by.
	aligns []Align
	// minExpandWidth is the table width at which the encoder switches to the
	// ExpandedEncoder. Zero turns off the switch.
	minExpandWidth int
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
// By default, the table encoder has a border of 1.
func NewTableEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	enc := &TableEncoder{
		resultSet: resultSet,
		newline:   newline,
		border:    1,
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
	if !enc.lineStyle.valid() {
		return nil, ErrInvalidLineStyle
	}
	return enc, nil
}

// Encode encodes one result set to the writer with the options of the
// encoder.
//
// The encoder draws the table as psql does in print_aligned_text, in
// src/fe_utils/print.c.
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
	// start with the widths that the user set
	enc.maxWidths = make([]int, clen)
	enc.headerWidths = make([]int, clen)
	enc.sums = make([]int, clen)
	enc.wide = make([]bool, clen)
	enc.aligns = make([]Align, clen)
	copy(enc.maxWidths, enc.widths)
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	enc.fixed = columnAligns(enc.resultSet, clen, enc.columnAlign)
	// table is true after the encoder draws the header or a row as a table.
	// exp is set after the encoder switches a batch to expanded records.
	var table bool
	var exp *ExpandedEncoder
	for first := true; ; first = false {
		var vals [][]*Value
		// read the next batch
		vals, err = enc.nextResults()
		if err != nil {
			return err
		}
		// no more rows. psql draws the header of a result set that has no
		// rows, so the first batch is drawn even when it is empty.
		if len(vals) == 0 && !first {
			break
		}
		enc.calcWidth(vals)
		// psql does not switch a table of one column, because the records
		// would be wider than the table
		if enc.minExpandWidth != 0 && clen > 1 && enc.tableWidth() >= enc.minExpandWidth {
			if exp == nil {
				if exp, err = enc.expanded(cols); err != nil {
					return err
				}
				// psql writes the title of expanded records as it is
				if !table {
					exp.writeTitle()
				}
			}
			exp.calcWidth(vals, enc.scanCount)
			if err := exp.encodeVals(vals, enc.scanCount-len(vals)); err != nil {
				return err
			}
			if len(vals) == 0 {
				break
			}
			continue
		}
		// draw the header before the first row
		if !table && !enc.skipHeader {
			enc.header()
		}
		table = true
		if err := enc.encodeVals(vals); err != nil {
			return err
		}
		if len(vals) == 0 {
			break
		}
	}
	// draw the end border, and write the summary
	switch {
	case table:
		if enc.border >= 2 {
			enc.hline(enc.lineStyle.End)
		}
		err = summarize(enc.w, enc.summary, enc.scanCount)
	default:
		exp.scanCount = enc.scanCount
		err = exp.end(clen)
	}
	if err != nil {
		return err
	}
	return enc.w.Flush()
}

// expanded returns an expanded encoder that writes to the same writer, for a
// batch that is too wide for the table.
func (enc *TableEncoder) expanded(cols []string) (*ExpandedEncoder, error) {
	t := *enc
	t.formatter = NewEscapeFormatter()
	exp := &ExpandedEncoder{
		TableEncoder: t,
	}
	exp.maxWidths = make([]int, 2)
	var err error
	exp.headers, err = exp.formatter.Header(cols)
	return exp, err
}

func (enc *TableEncoder) encodeVals(vals [][]*Value) error {
	// write the rows of the batch
	for i := range vals {
		enc.row(vals[i])
		// check for a write error every 1000 rows
		if (i+1)%1000 == 0 {
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
	return encodeAll(w, enc.resultSet, enc.Encode, enc.resultSetSep, enc.newline)
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
		applyAligns(v, enc.fixed)
		vals, i = append(vals, v), i+1
		// read in batches of enc.count rows
		if enc.count != 0 && i%enc.count == 0 {
			break
		}
	}
	return vals, enc.resultSet.Err()
}

func (enc *TableEncoder) calcWidth(vals [][]*Value) {
	for i, h := range enc.headers {
		// the width of the header is the minimum
		enc.headerWidths[i] = h.MaxWidth()
		enc.maxWidths[i] = max(enc.maxWidths[i], enc.headerWidths[i])
		// find the maximum width of the column, from the first row to the last
		for j := range vals {
			cell := vals[j][i]
			if cell == nil {
				cell = enc.empty
			}
			width := cell.MaxWidth()
			enc.maxWidths[i] = max(enc.maxWidths[i], width)
			enc.sums[i] += width
			if enc.wrapWidth > 0 && !enc.wide[i] {
				enc.wide[i] = hasWide(cell.Buf)
			}
		}
		// a null value has no type of its own to align by, so it follows the
		// column, the way psql aligns the null string by the column's type. A
		// batch that has no non-null value keeps the alignment of an earlier
		// batch.
		if align, ok := columnAlign(vals, i); ok {
			enc.aligns[i] = align
		}
		if i < len(enc.fixed) && enc.fixed[i].ok {
			enc.aligns[i] = enc.fixed[i].align
		}
	}
	enc.wrap()
}

// wrap sets the widths that the encoder draws the columns with. With a wrap
// width, it makes a column narrower until the table fits, as psql does. It
// takes one column from the column with the highest ratio of its maximum
// width to its average width, and so wraps the fewest values. A column is
// never narrower than its name.
//
// Note: psql can make a column 1 wide when it holds a character 2 columns
// wide, and the character then pushes the line out of line. tblfmt keeps
// such a column 2 wide (D56).
func (enc *TableEncoder) wrap() {
	enc.colWidths = slices.Clone(enc.maxWidths)
	if enc.wrapWidth <= 0 || enc.scanCount == 0 {
		return
	}
	// psql does not wrap when the column names alone do not fit
	headerWidth := enc.borderWidth()
	for _, w := range enc.headerWidths {
		headerWidth += w
	}
	if enc.wrapWidth < headerWidth {
		return
	}
	for width := enc.tableWidth(); width > enc.wrapWidth; width-- {
		worst, ratio := -1, 0.0
		for i, w := range enc.colWidths {
			average := enc.sums[i] / enc.scanCount
			minWidth := enc.headerWidths[i]
			if enc.wide[i] {
				minWidth = max(minWidth, 2)
			}
			if average == 0 || w <= minWidth {
				continue
			}
			// psql adds 1% of the maximum width, so that a wide column is
			// made narrower first
			if r := float64(w)/float64(average) + float64(enc.maxWidths[i])*0.01; r > ratio {
				worst, ratio = i, r
			}
		}
		if worst == -1 {
			return
		}
		enc.colWidths[worst]--
	}
}

// header draws the title, the top border, the column names and the line
// under them.
func (enc *TableEncoder) header() {
	ls := enc.lineStyle
	// psql centers the title over the table, unless the title is wider
	if enc.title != nil && enc.title.Width != 0 {
		if width := enc.tableWidth() - enc.title.MaxWidth(); width > 0 {
			enc.spaces(width / 2)
		}
		_, _ = enc.w.Write(enc.title.Buf)
		_, _ = enc.w.Write(enc.newline)
	}
	if enc.border >= 2 {
		enc.hline(ls.Top)
	}
	// done is true for each column name that has no more lines
	done := make([]bool, len(enc.headers))
	more := len(enc.headers)
	for l := 0; more > 0; l++ {
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Left)
		}
		for i, h := range enc.headers {
			if enc.border != 0 || (ls.LeftMarks && i > 0) {
				if l != 0 {
					_, _ = enc.w.WriteString(ls.HeaderNewlineLeft)
				} else {
					_, _ = enc.w.WriteString(" ")
				}
			}
			if !done[i] {
				padding := enc.colWidths[i] - h.LineWidth(l)
				var left int
				switch h.Align {
				case AlignCenter:
					left = padding / 2
				case AlignRight:
					left = padding
				}
				enc.spaces(left)
				enc.writeLine(h, l)
				enc.spaces(padding - left)
				if l == len(h.Newlines) {
					done[i] = true
					more--
				}
			} else {
				enc.spaces(enc.colWidths[i])
			}
			if enc.border != 0 || !ls.LeftMarks {
				if !done[i] {
					_, _ = enc.w.WriteString(ls.HeaderNewlineRight)
				} else {
					_, _ = enc.w.WriteString(" ")
				}
			}
			if enc.border != 0 && i != len(enc.headers)-1 {
				_, _ = enc.w.WriteString(ls.Row.Middle)
			}
		}
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Right)
		}
		_, _ = enc.w.Write(enc.newline)
	}
	enc.hline(ls.Mid)
}

// hline draws a horizontal line with the rule, as _print_horizontal_line in
// psql does. A rule with no line draws nothing.
func (enc *TableEncoder) hline(rule Rule) {
	if rule.Line == "" {
		return
	}
	switch enc.border {
	case 0:
	case 1:
		_, _ = enc.w.WriteString(rule.Line)
	default:
		_, _ = enc.w.WriteString(rule.Left + rule.Line)
	}
	for i, width := range enc.colWidths {
		_, _ = enc.w.WriteString(strings.Repeat(rule.Line, width))
		if i != len(enc.colWidths)-1 {
			if enc.border == 0 {
				_, _ = enc.w.WriteString(" ")
			} else {
				_, _ = enc.w.WriteString(rule.Line + rule.Middle + rule.Line)
			}
		}
	}
	switch enc.border {
	case 0:
	case 1:
		_, _ = enc.w.WriteString(rule.Line)
	default:
		_, _ = enc.w.WriteString(rule.Line + rule.Right)
	}
	_, _ = enc.w.Write(enc.newline)
}

// tableWidth returns the width of the table, as psql calculates it.
func (enc *TableEncoder) tableWidth() int {
	width := enc.borderWidth()
	for _, w := range enc.colWidths {
		width += w
	}
	return width
}

// borderWidth returns the width of the borders and the separators of the
// columns, as psql calculates it.
func (enc *TableEncoder) borderWidth() int {
	n := len(enc.headers)
	switch enc.border {
	case 0:
		return n
	case 1:
		return n*3 - min(n, 1)
	}
	return n*3 + 1
}

// row draws a table row. A value that holds newlines takes more than one
// line, and so does a value that the encoder wraps.
//
// Note: a row that has no values draws no line, as in psql.
func (enc *TableEncoder) row(vals []*Value) {
	if len(vals) == 0 {
		return
	}
	ls := enc.lineStyle
	// lines holds the next line of each value, and marks tells whether that
	// line follows a wrap or a newline. With wrapping, texts holds the line
	// with its tabs written as spaces, and offsets holds the part of it that
	// the encoder wrote.
	lines := make([]int, len(vals))
	marks := make([]mark, len(vals))
	var texts [][]byte
	var offsets []int
	if enc.wrapWidth > 0 {
		texts, offsets = make([][]byte, len(vals)), make([]int, len(vals))
	}
	value := func(i int) (*Value, Align) {
		if vals[i] == nil {
			return enc.empty, enc.aligns[i]
		}
		return vals[i], vals[i].Align
	}
	for more := true; more; {
		more = false
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Left)
		}
		for i := range vals {
			v, align := value(i)
			// psql pads the last column only when a border or a mark follows
			// it
			last := enc.border < 2 && i == len(vals)-1
			if enc.border != 0 {
				_, _ = enc.w.WriteString(marks[i].left(ls))
			}
			padding := 0
			switch l := lines[i]; {
			case l > len(v.Newlines):
				// the value has no more lines
				if !last {
					enc.spaces(enc.colWidths[i])
				}
			case texts == nil:
				padding = enc.colWidths[i] - v.LineWidth(l)
				padding = enc.alignLeft(align, padding)
				enc.writeLine(v, l)
				lines[i]++
				more = more || lines[i] <= len(v.Newlines)
			default:
				if offsets[i] == 0 {
					texts[i] = lineText(v, l)
				}
				text, width := cut(texts[i][offsets[i]:], enc.colWidths[i])
				padding = enc.alignLeft(align, enc.colWidths[i]-width)
				_, _ = enc.w.Write(text)
				offsets[i] += len(text)
				if offsets[i] < len(texts[i]) {
					more = true
				} else {
					lines[i]++
					offsets[i] = 0
					more = more || lines[i] <= len(v.Newlines)
				}
			}
			marks[i] = markNone
			if lines[i] <= len(v.Newlines) {
				switch {
				case offsets != nil && offsets[i] != 0:
					marks[i] = markWrap
				case lines[i] != 0:
					marks[i] = markNewline
				}
			}
			if !last || marks[i] != markNone {
				enc.spaces(padding)
			}
			switch {
			case marks[i] != markNone:
				_, _ = enc.w.WriteString(marks[i].right(ls))
			case !last:
				_, _ = enc.w.WriteString(" ")
			}
			// the vertical line before the next value tells whether that
			// value continues after a wrap or a newline, or has no more lines
			if enc.border != 0 && i != len(vals)-1 {
				next, _ := value(i + 1)
				switch {
				case marks[i+1] == markWrap:
					_, _ = enc.w.WriteString(ls.MidWrap)
				case marks[i+1] == markNewline:
					_, _ = enc.w.WriteString(ls.MidNewline)
				case lines[i+1] > len(next.Newlines):
					_, _ = enc.w.WriteString(ls.MidBlank)
				default:
					_, _ = enc.w.WriteString(ls.Row.Middle)
				}
			}
		}
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Right)
		}
		_, _ = enc.w.Write(enc.newline)
	}
}

// alignLeft writes the padding that goes before a value with the alignment,
// and returns the padding that goes after it.
func (enc *TableEncoder) alignLeft(align Align, padding int) int {
	switch align {
	case AlignRight:
		enc.spaces(padding)
		return 0
	case AlignCenter:
		enc.spaces(padding / 2)
		return padding - padding/2
	}
	return padding
}

// mark is the state of the next line of a value, as psql tracks it.
type mark int

// The marks.
const (
	markNone mark = iota
	markWrap
	markNewline
)

// left returns the mark at the left of a line of a value.
func (m mark) left(ls LineStyle) string {
	switch m {
	case markWrap:
		return ls.WrapLeft
	case markNewline:
		return ls.NewlineLeft
	}
	return " "
}

// right returns the mark at the right of a line of a value.
func (m mark) right(ls LineStyle) string {
	switch m {
	case markWrap:
		return ls.WrapRight
	case markNewline:
		return ls.NewlineRight
	}
	return " "
}

// lineText returns line l of the value, with each tab written as spaces, as
// writeLine writes it.
func lineText(v *Value, l int) []byte {
	start, end := v.line(l)
	if l >= len(v.Tabs) || len(v.Tabs[l]) == 0 {
		return v.Buf[start:end]
	}
	var buf []byte
	var width int
	for _, tab := range v.Tabs[l] {
		buf = append(buf, v.Buf[start:tab[0]]...)
		width += tab[1]
		n := tabWidth - width%tabWidth
		buf = append(buf, spaces[:n]...)
		width += n
		start = tab[0] + 1
	}
	return append(buf, v.Buf[start:end]...)
}

// cut returns the start of the text that is at most width columns wide, and
// its width. It cuts between two grapheme clusters. As psql does, it takes the
// first cluster even when that is wider, and then reports the width.
func cut(text []byte, width int) ([]byte, int) {
	if w := textWidth(text); w <= width {
		return text, w
	}
	s := string(text)
	var prefix string
	// note: go-runewidth can count a cluster narrower than textWidth does,
	// so the prefix is cut again until it fits. See B38.
	for n := width; n >= 0; n-- {
		prefix = runewidth.Truncate(s, n, "")
		if textWidth([]byte(prefix)) <= width {
			break
		}
	}
	for n := width + 1; prefix == ""; n++ {
		prefix = runewidth.Truncate(s, n, "")
	}
	return text[:len(prefix)], min(textWidth([]byte(prefix)), width)
}

// hasWide reports whether the text holds a character 2 columns wide.
func hasWide(text []byte) bool {
	for _, r := range string(text) {
		if runewidth.RuneWidth(r) == 2 {
			return true
		}
	}
	return emojiWidth(text) != 0
}

// textWidth returns the display width of the text, as FormatBytes measures it.
func textWidth(text []byte) int {
	return runewidth.StringWidth(string(text)) + emojiWidth(text)
}

// writeLine writes line l of the value. It writes each tab as the spaces up
// to the next tab stop, as psql does. A tab stop is every 8 columns from the
// start of the line.
func (enc *TableEncoder) writeLine(v *Value, l int) {
	start, end := v.line(l)
	if l < len(v.Tabs) {
		var width int
		for _, tab := range v.Tabs[l] {
			_, _ = enc.w.Write(v.Buf[start:tab[0]])
			width += tab[1]
			n := tabWidth - width%tabWidth
			enc.spaces(n)
			width += n
			start = tab[0] + 1
		}
	}
	_, _ = enc.w.Write(v.Buf[start:end])
}

// spaces writes n spaces.
func (enc *TableEncoder) spaces(n int) {
	for ; n > len(spaces); n -= len(spaces) {
		_, _ = enc.w.WriteString(spaces)
	}
	if n > 0 {
		_, _ = enc.w.WriteString(spaces[:n])
	}
}

// spaces is a run of spaces for padding.
const spaces = "                                                                "

// ExpandedEncoder is an encoder that writes a result set as an expanded
// table. It writes each row as a record, with one line for each column. It
// buffers a batch of rows ahead, as TableEncoder does.
type ExpandedEncoder struct {
	TableEncoder

	// hmultiline is true when a column name holds a newline, and dmultiline
	// is true when a value holds a newline.
	hmultiline bool
	dmultiline bool
}

// NewExpandedEncoder creates an expanded table encoder with the options.
func NewExpandedEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	tableEnc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return nil, err
	}
	t := tableEnc.(*TableEncoder)
	t.formatter = NewEscapeFormatter()
	enc := &ExpandedEncoder{
		TableEncoder: *t,
	}
	return enc, nil
}

// Encode encodes one result set to the writer with the options of the
// encoder.
//
// The encoder draws the records as psql does in print_aligned_vertical, in
// src/fe_utils/print.c.
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
	enc.maxWidths = make([]int, 2)
	enc.hmultiline, enc.dmultiline = false, false
	enc.headers, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
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
		// write the title before the first record
		if enc.scanCount == len(vals) {
			enc.writeTitle()
		}
		enc.calcWidth(vals, enc.scanCount)
		if err := enc.encodeVals(vals, enc.scanCount-len(vals)); err != nil {
			return err
		}
	}
	if err := enc.end(clen); err != nil {
		return err
	}
	return enc.w.Flush()
}

// title writes the title as it is, as psql does for expanded records.
func (enc *ExpandedEncoder) writeTitle() {
	if !enc.skipHeader && enc.title != nil && enc.title.Width != 0 {
		_, _ = enc.w.Write(enc.title.Buf)
		_, _ = enc.w.Write(enc.newline)
	}
}

// encodeVals writes the records of the batch. The first record of the batch
// is record n+1.
func (enc *ExpandedEncoder) encodeVals(vals [][]*Value, n int) error {
	for i := range vals {
		enc.record(n+i+1, vals[i])
		// check for a write error every 1000 rows
		if (i+1)%1000 == 0 {
			if err := enc.w.Flush(); err != nil {
				return err
			}
		}
	}
	return nil
}

// end draws the end border, and writes the summary. psql writes the row count
// only for a result set that has no rows or no columns. It writes a custom
// summary after the records, with a blank line before it when the border is
// less than 2.
func (enc *ExpandedEncoder) end(clen int) error {
	if enc.scanCount == 0 || clen == 0 {
		return summarize(enc.w, enc.summary, enc.scanCount)
	}
	if enc.border >= 2 {
		enc.recordLine(0, enc.maxWidths[0], enc.lineStyle.End)
	}
	if !enc.isCustomSummary || len(enc.summary) == 0 {
		return nil
	}
	if enc.border < 2 {
		if _, err := enc.w.Write(enc.newline); err != nil {
			return err
		}
	}
	return summarize(enc.w, enc.summary, enc.scanCount)
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
func (enc *ExpandedEncoder) EncodeAll(w io.Writer) error {
	return encodeAll(w, enc.resultSet, enc.Encode, enc.resultSetSep, enc.newline)
}

// calcWidth calculates the width of the column names and of the values. The
// batch ends with record n. The values are at least as wide as the record
// header line needs.
func (enc *ExpandedEncoder) calcWidth(vals [][]*Value, n int) {
	for _, h := range enc.headers {
		enc.maxWidths[0] = max(enc.maxWidths[0], h.MaxWidth())
		enc.hmultiline = enc.hmultiline || len(h.Newlines) != 0
	}
	for _, row := range vals {
		for _, cell := range row {
			if cell == nil {
				cell = enc.empty
			}
			enc.maxWidths[1] = max(enc.maxWidths[1], cell.MaxWidth())
			enc.dmultiline = enc.dmultiline || len(cell.Newlines) != 0
		}
	}
	// swidth is the width of the separator between the column names and the
	// values, and of the marks around it
	old := enc.lineStyle.LeftMarks
	var swidth int
	switch enc.border {
	case 0:
		swidth = 1
		if enc.hmultiline {
			swidth++
		}
	case 1:
		swidth = 3
		if enc.hmultiline && old {
			swidth++
		}
	default:
		swidth = 7
	}
	if enc.dmultiline && enc.border < 2 && !old {
		swidth++
	}
	// rwidth is the width of the record header line
	var rwidth int
	if !enc.skipHeader {
		rwidth = len(strconv.Itoa(n))
		switch enc.border {
		case 0:
			rwidth += len("* Record ")
		case 1:
			rwidth += len("-[ RECORD  ]")
		default:
			rwidth += len("+-[ RECORD  ]-+")
		}
	}
	hwidth, dwidth := enc.maxWidths[0], enc.maxWidths[1]
	for {
		width := max(hwidth+swidth+dwidth, rwidth)
		newWidth := width - hwidth - swidth
		// with a wrap width, psql fits the records in it, but leaves at
		// least 3 columns for a value
		if enc.wrapWidth > 0 {
			minWidth := max(hwidth+swidth+3, rwidth)
			switch {
			case enc.wrapWidth < minWidth:
				newWidth = minWidth - hwidth - swidth
			case enc.wrapWidth < width:
				newWidth = enc.wrapWidth - hwidth - swidth
			}
		}
		// a value that wraps needs a column for its mark
		if newWidth < dwidth && !enc.dmultiline && enc.border < 2 && !old {
			enc.dmultiline = true
			swidth++
			continue
		}
		enc.maxWidths[1] = newWidth
		return
	}
}

// record writes record n: the record header line, and a line for each column.
func (enc *ExpandedEncoder) record(n int, vals []*Value) {
	ls := enc.lineStyle
	rule := ls.Mid
	if n == 1 {
		rule = ls.Top
	}
	hwidth := enc.maxWidths[0]
	if enc.border < 2 && enc.hmultiline && ls.LeftMarks {
		hwidth++
	}
	switch {
	case !enc.skipHeader:
		enc.recordLine(n, hwidth, rule)
	case n != 1 || enc.border >= 2:
		// psql separates records with a line that has no record number
		enc.recordLine(0, hwidth, rule)
	}
	for i, v := range vals {
		if v == nil {
			v = enc.empty
		}
		enc.field(enc.headers[i], v)
	}
}

// recordLine draws the line above record n, or a line with no record number
// when n is 0, as print_aligned_vertical_line in psql does.
func (enc *ExpandedEncoder) recordLine(n, hwidth int, rule Rule) {
	fill := " "
	if enc.border > 0 {
		fill = rule.Line
	}
	switch enc.border {
	case 0:
	case 1:
		_, _ = enc.w.WriteString(rule.Line)
	default:
		_, _ = enc.w.WriteString(rule.Left + rule.Line)
	}
	var reclen int
	if n != 0 {
		label := "[ RECORD " + strconv.Itoa(n) + " ]"
		if enc.border == 0 {
			label = "* Record " + strconv.Itoa(n)
		}
		_, _ = enc.w.WriteString(label)
		reclen = len(label)
	}
	if enc.border != 2 {
		reclen++
	}
	for i := reclen; i < hwidth; i++ {
		_, _ = enc.w.WriteString(fill)
	}
	reclen -= hwidth
	// each part of the separator is drawn only where the label does not
	// cover it
	parts := []string{" "}
	if enc.border > 0 {
		parts = []string{rule.Line, rule.Middle, rule.Line}
	}
	for _, s := range parts {
		if reclen <= 0 {
			_, _ = enc.w.WriteString(s)
		}
		reclen--
	}
	reclen = max(reclen, 0)
	for i := reclen; i < max(enc.maxWidths[1], reclen); i++ {
		_, _ = enc.w.WriteString(fill)
	}
	if enc.border >= 2 {
		_, _ = enc.w.WriteString(rule.Line + rule.Right)
	}
	_, _ = enc.w.Write(enc.newline)
}

// field writes a column name and its value. Either one can take more than one
// line.
func (enc *ExpandedEncoder) field(h, v *Value) {
	ls := enc.lineStyle
	old := ls.LeftMarks
	hwidth, dwidth := enc.maxWidths[0], enc.maxWidths[1]
	hline, dline := 0, 0
	hdone, ddone := false, false
	// text is the line of the value with its tabs written as spaces, and
	// offset is the part of it that the encoder wrote. psql keeps the offset
	// after the value ends.
	text := lineText(v, 0)
	var offset int
	for !hdone || !ddone {
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Left)
		}
		// the column name
		if !hdone {
			if enc.border >= 2 || (enc.hmultiline && old) {
				if hline != 0 {
					_, _ = enc.w.WriteString(ls.HeaderNewlineLeft)
				} else {
					_, _ = enc.w.WriteString(" ")
				}
			}
			enc.writeLine(h, hline)
			enc.spaces(hwidth - h.LineWidth(hline))
			mark := enc.border > 0 || (enc.hmultiline && !old)
			if hline < len(h.Newlines) {
				if mark {
					_, _ = enc.w.WriteString(ls.HeaderNewlineRight)
				}
				hline++
			} else {
				if mark {
					_, _ = enc.w.WriteString(" ")
				}
				hdone = true
			}
		} else {
			swidth := hwidth + enc.border
			if enc.border < 2 && enc.hmultiline && old {
				swidth++
			}
			if enc.border == 0 && !old && enc.hmultiline {
				swidth++
			}
			// note: psql writes at least one space here
			enc.spaces(max(swidth, 1))
		}
		// the separator
		if enc.border > 0 {
			switch {
			case offset != 0:
				_, _ = enc.w.WriteString(ls.MidWrap)
			case dline == 0:
				_, _ = enc.w.WriteString(ls.Row.Middle)
			default:
				_, _ = enc.w.WriteString(ls.MidNewline)
			}
		}
		// the value
		if ddone {
			if enc.border >= 2 {
				enc.spaces(dwidth + 2)
				_, _ = enc.w.WriteString(ls.Row.Right)
			}
			_, _ = enc.w.Write(enc.newline)
			continue
		}
		if offset == 0 {
			_, _ = enc.w.WriteString(" ")
		} else {
			_, _ = enc.w.WriteString(ls.WrapLeft)
		}
		chunk, width := cut(text[offset:], dwidth)
		_, _ = enc.w.Write(chunk)
		offset += len(chunk)
		padding := dwidth - width
		marks := enc.border > 1 || (enc.dmultiline && !old)
		switch {
		case offset < len(text):
			// the value wraps
			if marks {
				enc.spaces(padding)
				_, _ = enc.w.WriteString(ls.WrapRight)
			}
		case dline < len(v.Newlines):
			if marks {
				enc.spaces(padding)
				_, _ = enc.w.WriteString(ls.NewlineRight)
			}
			dline++
			offset = 0
			text = lineText(v, dline)
		default:
			if enc.border > 1 {
				enc.spaces(padding)
				_, _ = enc.w.WriteString(" ")
			}
			ddone = true
		}
		if enc.border >= 2 {
			_, _ = enc.w.WriteString(ls.Row.Right)
		}
		_, _ = enc.w.Write(enc.newline)
	}
}

// fixedAlign is the alignment that WithColumnAlign sets for a column. ok is
// false for a column that is aligned by its values.
type fixedAlign struct {
	align Align
	ok    bool
}

// columnAligns returns the alignment that f sets for each column. It returns
// nil when f is nil, or when the result set has no column types.
func columnAligns(resultSet ResultSet, n int, f func(*sql.ColumnType) (Align, bool)) []fixedAlign {
	if f == nil {
		return nil
	}
	cols, err := resultSetColumns(resultSet, n)
	if err != nil {
		return nil
	}
	aligns := make([]fixedAlign, n)
	for i, col := range cols {
		aligns[i].align, aligns[i].ok = f(col)
	}
	return aligns
}

// applyAligns sets the fixed alignment of each value of the row.
func applyAligns(row []*Value, aligns []fixedAlign) {
	for i, a := range aligns {
		if a.ok && i < len(row) && row[i] != nil {
			row[i].Align = a.align
		}
	}
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
	// resultSetSep is written between two result sets. When it is nil, the
	// encoder writes newline.
	resultSetSep []byte
	// indent is the indent of one level. When it is empty, the encoder writes
	// compact JSON.
	indent string
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
		// note: WithIndent sets the prefix to the indent of a value, so that
		// the lines of a nested object or array line up under the value.
		formatter: NewEscapeFormatter(WithIsJSON(true), WithJSONConfig("", "", false)),
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
		colon = []byte{':'}
		// rowNL and valNL start a row and a value. Compact JSON has neither.
		rowNL, valNL []byte
	)
	if enc.indent != "" {
		rowNL = append(append([]byte{}, enc.newline...), enc.indent...)
		valNL = append(append([]byte{}, enc.newline...), enc.indent+enc.indent...)
		colon = []byte{':', ' '}
	}
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
		cb[i] = append(cb[i], colon...)
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
	if count != 0 && enc.indent != "" {
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
	if err := encodeAll(w, enc.resultSet, enc.Encode, enc.resultSetSep, enc.newline); err != nil {
		return err
	}
	_, err := w.Write(enc.newline)
	return err
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
	// resultSetSep is written between two result sets. When it is nil, the
	// encoder writes newline.
	resultSetSep []byte
	// formatter formats the values before the encoder writes them.
	formatter Formatter
	// skipHeader turns off the header.
	skipHeader bool
	// title is the title value. psql writes it in the unaligned format, and
	// not in csv.
	title *Value
	// csv is true for the csv format.
	csv bool
	// expanded is true when the encoder writes each row as a record.
	expanded bool
	// summary is the summary map, and isCustomSummary is true if an option
	// set it.
	summary         map[int]func(io.Writer, int) (int, error)
	isCustomSummary bool
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
		csv:       true,
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
//
// The encoder writes the result set as psql does in print_unaligned_text,
// print_unaligned_vertical, print_csv_text and print_csv_vertical, in
// src/fe_utils/print.c.
func (enc *UnalignedEncoder) Encode(w io.Writer) error {
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
	ew := &errWriter{w: w}
	// psql writes the title of the unaligned format, and not of csv
	title := !enc.csv && !enc.skipHeader && enc.title != nil && enc.title.Width != 0
	if title {
		ew.write(enc.title.Buf)
	}
	// write the header
	if !enc.expanded && !enc.skipHeader {
		if title {
			ew.write(enc.newline)
		}
		for i := range clen {
			if i != 0 {
				ew.writeRune(enc.sep)
			}
			enc.write(ew, headers[i])
		}
		ew.write(enc.newline)
	} else if title && !enc.expanded {
		ew.write(enc.newline)
	}
	// set up the storage for the scanned values
	r, err := buildColumnTypes(enc.resultSet, clen, enc.columnTypes)
	if err != nil {
		return err
	}
	// write the rows. In expanded mode, each row is a record with a line for
	// each column. The unaligned format puts a blank line between two
	// records, and after the title.
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
		if enc.expanded && !enc.csv && (count > 1 || title) && clen != 0 {
			ew.write(enc.newline)
			ew.write(enc.newline)
		}
		for i := range clen {
			v := vals[i]
			if v == nil {
				v = enc.empty
			}
			switch {
			case enc.expanded:
				enc.write(ew, headers[i])
				ew.writeRune(enc.sep)
				enc.write(ew, v)
				if enc.csv || i != clen-1 {
					ew.write(enc.newline)
				}
			default:
				if i != 0 {
					ew.writeRune(enc.sep)
				}
				enc.write(ew, v)
			}
		}
		if !enc.expanded {
			ew.write(enc.newline)
		}
		if ew.err != nil {
			return ew.err
		}
	}
	if err := enc.resultSet.Err(); err != nil {
		return err
	}
	switch {
	case !enc.expanded:
		if err := summarize(ew, enc.summary, count); err != nil {
			return err
		}
	case enc.csv:
	default:
		// psql writes only a custom summary after the records, after a blank
		// line, and ends the last line
		// a row that has no columns writes no record
		records := count != 0 && clen != 0
		if enc.isCustomSummary && len(enc.summary) != 0 && records {
			ew.write(enc.newline)
			ew.write(enc.newline)
			var buf bytes.Buffer
			if err := summarize(&buf, enc.summary, count); err != nil {
				return err
			}
			ew.write(bytes.TrimSuffix(buf.Bytes(), newline))
		}
		if records || title {
			ew.write(enc.newline)
		}
	}
	return ew.err
}

// write writes the value, in quotes when the csv format needs them.
func (enc *UnalignedEncoder) write(ew *errWriter, v *Value) {
	if enc.quote != 0 && v.Quoted {
		ew.writeRune(enc.quote)
		ew.write(v.Buf)
		ew.writeRune(enc.quote)
		return
	}
	ew.write(v.Buf)
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
func (enc *UnalignedEncoder) EncodeAll(w io.Writer) error {
	return encodeAll(w, enc.resultSet, enc.Encode, enc.resultSetSep, enc.newline)
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
	// resultSetSep is written between two result sets. When it is nil, the
	// encoder writes newline.
	resultSetSep []byte
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
	// border is the border, as the psql variable border sets it.
	border int
	// expanded is true when the template writes each row as a record.
	expanded bool
	// summary is the summary map, and isCustomSummary is true if an option
	// set it.
	summary         Summary
	isCustomSummary bool
	// columnAlign sets the alignment of a column from its type.
	columnAlign func(*sql.ColumnType) (Align, bool)
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
		border:    1,
		summary:   DefaultTableSummary(),
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

// NewLaTeXEncoder creates a template encoder for LaTeX with the options.
func NewLaTeXEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("latex")}, opts...)...)
}

// NewLaTeXLongtableEncoder creates a template encoder for the LaTeX longtable
// package with the options.
func NewLaTeXLongtableEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("latex-longtable")}, opts...)...)
}

// NewTroffMSEncoder creates a template encoder for troff with the ms macros,
// with the options.
func NewTroffMSEncoder(resultSet ResultSet, opts ...Option) (Encoder, error) {
	return NewTemplateEncoder(resultSet, append([]Option{WithTemplate("troff-ms")}, opts...)...)
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
	// the column types are read before the rows, which closes them
	fixed := columnAligns(enc.resultSet, clen, enc.columnAlign)
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
	for _, vals := range rows {
		applyAligns(vals, fixed)
	}
	// a null value has no type of its own to align by, so it follows the
	// column, the way psql aligns the null string by the column's type
	for i := range clen {
		empty := enc.empty
		align, ok := columnAlign(rows, i)
		if i < len(fixed) && fixed[i].ok {
			align, ok = fixed[i].align, true
		}
		if ok && align != empty.Align {
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
	aligns := make([]Align, clen)
	for i := range clen {
		aligns[i], _ = columnAlign(rows, i)
		if i < len(fixed) && fixed[i].ok {
			aligns[i] = fixed[i].align
		}
	}
	// psql writes the row count or the custom summary after a table, and only
	// a custom summary after expanded records
	var footer bytes.Buffer
	if !enc.expanded || enc.isCustomSummary {
		if err := summarize(&footer, enc.summary, len(rows)); err != nil {
			return err
		}
	}
	// a template writer does not check each write, so ew keeps the first
	// write error
	ew := &errWriter{w: w}
	if err := enc.executor(ew, &Template{
		Attributes: enc.attributes,
		Headers:    headers,
		Rows:       templateRows(rows, clen),
		SkipHeader: enc.skipHeader,
		Title:      title,
		Aligns:     aligns,
		Border:     enc.border,
		Expanded:   enc.expanded,
		Footer:     strings.TrimSuffix(footer.String(), string(newline)),
	}); err != nil {
		return err
	}
	return ew.err
}

// templateRows returns the rows for a template. psql writes no line for a row
// that has no columns, and still counts it in the summary.
func templateRows(rows [][]*Value, clen int) [][]*Value {
	if clen == 0 {
		return nil
	}
	return rows
}

// errWriter is a writer that keeps the first error of its writes. After an
// error, it writes nothing.
type errWriter struct {
	w   io.Writer
	err error
}

// write writes p, and keeps the error.
func (ew *errWriter) write(p []byte) {
	_, _ = ew.Write(p)
}

// writeRune writes r as UTF-8, and keeps the error. A zero rune is a zero
// byte, as psql writes fieldsep_zero.
func (ew *errWriter) writeRune(r rune) {
	ew.write(utf8.AppendRune(nil, r))
}

// Write satisfies the io.Writer interface.
func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	ew.err = err
	return n, err
}

// EncodeAll encodes each result set to the writer with the options of the
// encoder.
func (enc *TemplateEncoder) EncodeAll(w io.Writer) error {
	return encodeAll(w, enc.resultSet, enc.Encode, enc.resultSetSep, enc.newline)
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
