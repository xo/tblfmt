package tblfmt

import (
	"bytes"
	"io"
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

	// expanded toggles expanded mode.
	//expanded bool

	// lineStyle is the table line style.
	lineStyle LineStyle

	// formatter handles formatting values prior to output.
	formatter Formatter

	// summary is the summary map.
	summary map[int]func(io.Writer, int) (int, error)

	// empty is the empty value.
	empty *Value

	// scanCount is the number of scanned results in the result set.
	scanCount int

	// offsets are the column offsets.
	offsets []int

	// widths are the column widths.
	widths []int
}

// NewTableEncoder creates a new table encoder using the provided
// options.
func NewTableEncoder(resultSet ResultSet, opts ...TableEncoderOption) *TableEncoder {
	enc := &TableEncoder{
		resultSet: resultSet,
		newline:   newline,
		border:    1,
		tab:       8,
		lineStyle: ASCIILineStyle(),
		formatter: NewEscapeFormatter(),
		summary:   DefaultTableSummary(),
		empty: &Value{
			Tabs: make([][][2]int, 1),
		},
	}
	for _, o := range opts {
		o(enc)
	}
	return enc
}

// Encode encodes a single result set to the writer using the formatting
// options specified in the encoder.
func (enc *TableEncoder) Encode(w io.Writer) error {
	var err error

	if enc.resultSet == nil {
		return ErrResultSetIsNil
	}

	// get columns
	cols, err := enc.resultSet.Columns()
	if err != nil {
		return err
	}

	clen := len(cols)

	// initialize
	r := make([]interface{}, clen)
	for i := 0; i < clen; i++ {
		r[i] = new(interface{})
	}

	var v []*Value
	var vals [][]*Value

	// format column names
	v, err = enc.formatter.Header(cols)
	if err != nil {
		return err
	}
	vals = append(vals, v)

	// buffer
	if enc.count >= 0 {
		if enc.count != 0 {
			vals = make([][]*Value, 0, enc.count)
		}

		// read to count (or all)
		var i int
		for enc.resultSet.Next() {
			v, err = enc.scanAndFormat(r)
			if err != nil {
				return err
			}
			vals, i = append(vals, v), i+1
			if enc.count != 0 && i >= enc.count {
				break
			}
		}
	}

	// calculate initial offsets and column widths
	enc.offsets, enc.widths = make([]int, clen), make([]int, clen)
	var offset int
	if enc.border > 1 {
		offset += 2
	}
	for i := 0; i < clen; i++ {
		if i == 0 && enc.border == 1 {
			offset++
		}
		if i != 0 && enc.border > 0 {
			offset += 2
		}

		// store offset
		enc.offsets[i] = offset

		// from top to bottom, find max column width
		for j := 0; j < len(vals); j++ {
			cell := vals[j][i]
			if cell == nil {
				cell = enc.empty
			}
			enc.widths[i] = max(enc.widths[i], cell.MaxWidth(offset, enc.tab))
		}

		// add column width, and one space for newline indicator
		offset += enc.widths[i] + 1
	}

	//fmt.Printf("offsets: %v, widths: %v\n", enc.offsets, enc.widths)

	// draw top border
	if enc.border >= 2 {
		if err = enc.divider(w, enc.lineStyle.Top); err != nil {
			return err
		}
	}

	// write header
	if err = enc.row(w, vals[0], false); err != nil {
		return err
	}

	// draw mid divider
	if err = enc.divider(w, enc.lineStyle.Mid); err != nil {
		return err
	}

	// marshal remaining buffered vals
	for i := 1; i < len(vals); i++ {
		if err = enc.row(w, vals[i], false); err != nil {
			return err
		}
	}

	// marshal remaining
	for enc.resultSet.Next() {
		v, err = enc.scanAndFormat(r)
		if err != nil {
			return err
		}
		if err = enc.row(w, v, true); err != nil {
			return err
		}
	}

	// draw end border
	if enc.border >= 2 {
		if err = enc.divider(w, enc.lineStyle.End); err != nil {
			return err
		}
	}

	// add summary
	if enc.summary != nil {
		if err = enc.summarize(w); err != nil {
			return nil
		}
	}

	_, err = w.Write(enc.newline)
	return err
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
//
// Mid:  [4]rune{'├', '─', '┼', '┤'},
//
// TODO: optimize / avoid multiple calls to w.Write.
func (enc *TableEncoder) divider(w io.Writer, r [4]rune) error {
	var err error

	// last column
	end := ' '
	if enc.border > 0 {
		end = r[1]
	}

	// left
	if enc.border > 1 {
		if err = condWrite(w, 1, r[0], r[1]); err != nil {
			return err
		}
	}

	for i, width := range enc.widths {
		if i == 0 && enc.border == 1 {
			if err = condWrite(w, 1, r[1]); err != nil {
				return err
			}
		}

		// left (column)
		if i != 0 {
			if enc.border > 0 {
				if err = condWrite(w, 1, r[2], r[1]); err != nil {
					return err
				}
			}
		}

		// column
		if err = condWrite(w, width, r[1]); err != nil {
			return err
		}

		// end
		if err = condWrite(w, 1, end); err != nil {
			return err
		}
	}

	// right
	if enc.border > 1 {
		if err = condWrite(w, 1, r[3]); err != nil {
			return err
		}
	}

	_, err = w.Write(enc.newline)
	return err
}

// row draws the a table row.
func (enc *TableEncoder) row(w io.Writer, vals []*Value, recalc bool) error {
	var err error

	//spew.Dump(vals)
	var l int
	for {
		// draw left border
		if enc.border > 1 {
			if err = condWrite(w, 1, enc.lineStyle.Row[0], enc.lineStyle.Row[1]); err != nil {
				return err
			}
		}

		var remaining bool
		for i, v := range vals {
			if v == nil {
				v = enc.empty
			}

			// draw column separator
			if i == 0 && enc.border == 1 {
				if err = condWrite(w, 1, ' '); err != nil {
					return err
				}
			}
			if i != 0 && enc.border > 0 {
				if err = condWrite(w, 1, enc.lineStyle.Row[2], enc.lineStyle.Row[1]); err != nil {
					return err
				}
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
				if len(v.Tabs[l]) != 0 {
					width += tabwidth(v.Tabs[l], enc.offsets[i], enc.tab)
				}
				if l == len(v.Newlines) {
					width += v.Width
				}

				// calc padding
				padding := enc.widths[i] - width

				// add padding left
				if v.Align == AlignRight && padding > 0 {
					_, err = w.Write(bytes.Repeat([]byte{' '}, padding))
					if err != nil {
						return err
					}
				}

				// write
				if _, err = w.Write(v.Buf[start:end]); err != nil {
					return err
				}

				// add padding right
				if v.Align == AlignLeft && padding > 0 {
					_, err = w.Write(bytes.Repeat([]byte{' '}, padding))
					if err != nil {
						return err
					}
				}
			} else if _, err = w.Write(bytes.Repeat([]byte{' '}, enc.widths[i])); err != nil {
				return err
			}

			// write newline wrap value
			if l < len(v.Newlines) {
				if err = condWrite(w, 1, enc.lineStyle.Wrap[1]); err != nil {
					return err
				}
			} else if err = condWrite(w, 1, ' '); err != nil {
				return err
			}

			remaining = remaining || l < len(v.Newlines)
		}

		// draw right border
		if enc.border > 1 {
			if err = condWrite(w, 1, enc.lineStyle.Row[3]); err != nil {
				return err
			}
		}

		_, err = w.Write(enc.newline)
		if err != nil {
			return err
		}

		if !remaining {
			break
		}

		l++
	}

	return nil
}

// condWrite conditionally writes runes to w.
func condWrite(w io.Writer, repeat int, runes ...rune) error {
	var buf []byte
	for _, r := range runes {
		if r != 0 {
			buf = append(buf, []byte(string(r))...)
		}
	}
	if repeat > 1 {
		buf = bytes.Repeat(buf, repeat)
	}
	_, err := w.Write(buf)
	return err
}

// summarize writes the table scan count summary.
func (enc *TableEncoder) summarize(w io.Writer) error {
	// do summary
	var f func(io.Writer, int) (int, error)
	if z, ok := enc.summary[-1]; ok {
		f = z
	}
	if z, ok := enc.summary[enc.scanCount]; ok {
		f = z
	}
	if f != nil {
		var err error
		if _, err = f(w, enc.scanCount); err != nil {
			return err
		}
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}
	}
	return nil
}

// TableEncoderOption is a table encoder option.
type TableEncoderOption func(*TableEncoder)

// WithCount is a table encoder option to set the buffered line count.
func WithCount(count int) TableEncoderOption {
	return func(enc *TableEncoder) {
		enc.count = count
	}
}

// WithStyle is a table encoder option to set the table line style.
func WithStyle(lineStyle LineStyle) TableEncoderOption {
	return func(enc *TableEncoder) {
		enc.lineStyle = lineStyle
	}
}

// WithFormatter is a table encoder option to set a formatter for formatting
// values.
func WithFormatter(formatter Formatter) TableEncoderOption {
	return func(enc *TableEncoder) {
		enc.formatter = formatter
	}
}

// WithSummary is a table encoder option to set a summary callback map.
func WithSummary(summary map[int]func(io.Writer, int) (int, error)) TableEncoderOption {
	return func(enc *TableEncoder) {
		enc.summary = summary
	}
}

// WithEmpty is a table encoder option to set the value used in empty (nil)
// cells.
func WithEmpty(empty string) TableEncoderOption {
	return func(enc *TableEncoder) {
		cell := interface{}(empty)
		v, err := enc.formatter.Format([]interface{}{&cell})
		if err != nil {
			panic(err)
		}
		enc.empty = v[0]
	}
}
