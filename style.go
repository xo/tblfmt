package tblfmt

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	runewidth "github.com/mattn/go-runewidth"
)

// Transformer is the interface for column transformers.
type Transformer interface {
	Transform(string) string
}

// TransformStyle is a transform style for column names in the header.
type TransformStyle int

// Transform styles.
const (
	TransformNone TransformStyle = iota
	TransformForceLower
	TransformForceUpper
	TransformUpperToLower
	TransformLowerToUpper
)

// Transform transforms s with the style. It satisfies the [Transformer]
// interface.
func (style TransformStyle) Transform(s string) string {
	switch style {
	case TransformForceLower:
		return strings.ToLower(s)
	case TransformForceUpper:
		return strings.ToUpper(s)
	case TransformUpperToLower:
		if j := strings.IndexFunc(s, func(r rune) bool {
			return unicode.IsLetter(r) && unicode.IsLower(r)
		}); j == -1 {
			return strings.ToLower(s)
		}
	case TransformLowerToUpper:
		if j := strings.IndexFunc(s, func(r rune) bool {
			return unicode.IsLetter(r) && unicode.IsUpper(r)
		}); j == -1 {
			return strings.ToUpper(s)
		}
	}
	return s
}

// Rule holds the strings of one kind of line in a table: the horizontal line,
// and the vertical lines at the left, between two columns, and at the right.
type Rule struct {
	Line   string
	Left   string
	Middle string
	Right  string
}

// LineStyle is a line style for tables. It holds the parts of a psql line
// style.
//
// The TableLineStyle, ASCIILineStyle, OldASCIILineStyle and UnicodeLineStyle
// funcs below return the predefined line styles.
//
// A table with a border of 2 looks like this:
//
//	+-----------+---------------------------+---+
//	| author_id |           name            | z |
//	+-----------+---------------------------+---+
//	|        14 | a       b       c       d |   |
//	|        15 | aoeu                     +|   |
//	|           | test                     +|   |
//	|           |                           |   |
//	+-----------+---------------------------+---+
//
// If the border is 1, the encoder writes the vertical lines between the
// columns:
//
//	 author_id |           name            | z
//	-----------+---------------------------+---
//	        14 | a       b       c       d | x
//	        15 | aoeu                     +|
//	           | test                     +|
//	           |                           |
//
// If the border is 0, the encoder writes no vertical line:
//
//	author_id           name            z
//	--------- ------------------------- -
//	       14 a       b       c       d x
//	       15 aoeu                     +
//	          test                     +
type LineStyle struct {
	// Top is the line above the header. Mid is the line under the header,
	// and End is the line under the last row. A rule with an empty Line
	// writes no line.
	Top Rule
	Mid Rule
	End Rule
	// Row holds the vertical lines of a row. Its Line is not used.
	Row Rule
	// MidNewline is the vertical line before a value that continues after a
	// newline. MidBlank is the vertical line before a value that has no more
	// lines. psql draws MidWrap before a wrapped value. In expanded output, it
	// also draws MidWrap beside each further line of a column name after a
	// value that is not empty ends.
	MidNewline string
	MidBlank   string
	MidWrap    string
	// HeaderNewlineLeft and HeaderNewlineRight mark a line of a column name
	// that continues after a newline.
	HeaderNewlineLeft  string
	HeaderNewlineRight string
	// NewlineLeft and NewlineRight mark a line of a value that continues
	// after a newline. WrapLeft and WrapRight mark a line of a value that
	// continues after a wrap.
	NewlineLeft  string
	NewlineRight string
	WrapLeft     string
	WrapRight    string
	// LeftMarks is true when the style marks a continued line on the left of
	// a value, as the psql old-ascii style does. With a border of 0, such a
	// style writes no mark at the right of a column name.
	LeftMarks bool
}

// TableLineStyle is the table line style. It draws no line.
//
// A table with this line style looks like this:
//
//	AUTHOR_ID NAME                      Z
//	14        a       b       c       d x
//	15        aoeu
//	          test
func TableLineStyle() LineStyle {
	return LineStyle{
		HeaderNewlineLeft:  " ",
		HeaderNewlineRight: " ",
		NewlineLeft:        " ",
		NewlineRight:       " ",
		WrapLeft:           " ",
		WrapRight:          " ",
	}
}

// ASCIILineStyle is the ASCII line style for tables, as psql draws it.
//
// A table with this line style looks like this:
//
//	+-----------+---------------------------+---+
//	| author_id |           name            | z |
//	+-----------+---------------------------+---+
//	|        14 | a       b       c       d | x |
//	|        15 | aoeu                     +|   |
//	|           | test                     +|   |
//	|           |                           |   |
//	+-----------+---------------------------+---+
func ASCIILineStyle() LineStyle {
	rule := Rule{Line: "-", Left: "+", Middle: "+", Right: "+"}
	return LineStyle{
		Top:                rule,
		Mid:                rule,
		End:                rule,
		Row:                Rule{Left: "|", Middle: "|", Right: "|"},
		MidNewline:         "|",
		MidBlank:           "|",
		MidWrap:            "|",
		HeaderNewlineLeft:  " ",
		HeaderNewlineRight: "+",
		NewlineLeft:        " ",
		NewlineRight:       "+",
		WrapLeft:           ".",
		WrapRight:          ".",
	}
}

// OldASCIILineStyle is the old ASCII line style for tables, as psql draws
// it. It marks a value that continues after a newline with a colon on its
// left.
//
// A table with this line style looks like this:
//
//	+-----------+---------------------------+---+
//	| author_id |           name            | z |
//	+-----------+---------------------------+---+
//	|        14 | a       b       c       d | x |
//	|        15 | aoeu                      |   |
//	|           : test                          |
//	|           :                               |
//	+-----------+---------------------------+---+
func OldASCIILineStyle() LineStyle {
	s := ASCIILineStyle()
	s.MidNewline = ":"
	s.MidBlank = " "
	s.MidWrap = ";"
	s.HeaderNewlineLeft = "+"
	s.HeaderNewlineRight = " "
	s.NewlineRight = " "
	s.WrapLeft = " "
	s.WrapRight = " "
	s.LeftMarks = true
	return s
}

// UnicodeLineStyle is the Unicode line style for tables, as psql draws it.
// Each argument draws one part of the table with double lines, as the psql
// variables unicode_border_linestyle, unicode_header_linestyle and
// unicode_column_linestyle do.
//
// A table with single lines looks like this:
//
//	┌───────────┬───────────────────────────┬───┐
//	│ author_id │           name            │ z │
//	├───────────┼───────────────────────────┼───┤
//	│        14 │ a       b       c       d │ x │
//	│        15 │ aoeu                     ↵│   │
//	│           │ test                     ↵│   │
//	│           │                           │   │
//	└───────────┴───────────────────────────┴───┘
func UnicodeLineStyle(doubleBorder, doubleHeader, doubleColumn bool) LineStyle {
	// each index is 0 for a single line, and 1 for a double line
	b, h, c := btoi(doubleBorder), btoi(doubleHeader), btoi(doubleColumn)
	border := [2][6]string{
		// up and right, vertical, down and right, horizontal, down and left,
		// up and left
		{"└", "│", "┌", "─", "┐", "┘"},
		{"╚", "║", "╔", "═", "╗", "╝"},
	}[b]
	header := [2][5]string{
		// horizontal, then vertical and right, and vertical and left, for a
		// single and a double border
		{"─", "├", "╟", "┤", "╢"},
		{"═", "╞", "╠", "╡", "╣"},
	}[h]
	column := [2][7]string{
		// vertical, then vertical and horizontal, up and horizontal, and down
		// and horizontal, for a single and a double crossing line
		{"│", "┼", "╪", "┴", "╧", "┬", "╤"},
		{"║", "╫", "╬", "╨", "╩", "╥", "╦"},
	}[c]
	return LineStyle{
		Top:                Rule{Line: border[3], Left: border[2], Middle: column[5+b], Right: border[4]},
		Mid:                Rule{Line: header[0], Left: header[1+b], Middle: column[1+h], Right: header[3+b]},
		End:                Rule{Line: border[3], Left: border[0], Middle: column[3+b], Right: border[5]},
		Row:                Rule{Left: border[1], Middle: column[0], Right: border[1]},
		MidNewline:         column[0],
		MidBlank:           column[0],
		MidWrap:            column[0],
		HeaderNewlineLeft:  " ",
		HeaderNewlineRight: "↵",
		NewlineLeft:        " ",
		NewlineRight:       "↵",
		WrapLeft:           "…",
		WrapRight:          "…",
	}
}

// btoi returns 1 for true, and 0 for false.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// valid reports whether each string of the line style is at most 1 column
// wide, as the encoders need.
func (ls LineStyle) valid() bool {
	strs := []string{
		ls.MidNewline, ls.MidBlank, ls.MidWrap,
		ls.HeaderNewlineLeft, ls.HeaderNewlineRight,
		ls.NewlineLeft, ls.NewlineRight,
		ls.WrapLeft, ls.WrapRight,
	}
	for _, r := range []Rule{ls.Top, ls.Mid, ls.End, ls.Row} {
		strs = append(strs, r.Line, r.Left, r.Middle, r.Right)
	}
	for _, s := range strs {
		if runewidth.StringWidth(s) > 1 {
			return false
		}
	}
	return true
}

// DefaultTableSummary is the default summary for tables.
//
// The default summary looks like this:
//
//	(3 rows)
func DefaultTableSummary() Summary {
	return map[int]func(io.Writer, int) (int, error){
		1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d row)", count)
		},
		-1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d rows)", count)
		},
	}
}
