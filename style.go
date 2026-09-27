package tblfmt

import (
	"fmt"
	"io"
	"strings"
	"unicode"
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

// LineStyle is a line style for tables.
//
// The ASCII, OldASCII, and Unicode line styles below are predefined line
// styles.
//
// A table usually looks like this:
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
// If the border is 0, the encoder does not write a border around the table:
//
//	author_id           name            z
//	--------- ------------------------- -
//	       14 a       b       c       d
//	       15 aoeu                     +
//	          test                     +
//
// If the border is 1, the encoder writes a border between the columns:
//
//	 author_id |           name            | z
//	-----------+---------------------------+---
//	        14 | a       b       c       d |
//	        15 | aoeu                     +|
//	           | test                     +|
//	           |                           |
type LineStyle struct {
	Top  [4]rune
	Mid  [4]rune
	Row  [4]rune
	Wrap [4]rune
	End  [4]rune
}

// TableLineStyle is the table line style.
//
// A table with this line style looks like this:
//
//	AUTHOR_ID  NAME                      Z
//	14         a       b       c       d
//	15         aoeu
//	           test
func TableLineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{0, 0, 0, 0},
		Mid:  [4]rune{0, 0, 0, 0},
		Row:  [4]rune{0, ' ', 0, 0},
		Wrap: [4]rune{0, ' ', 0, 0},
		End:  [4]rune{0, 0, 0, 0},
	}
}

// ASCIILineStyle is the ASCII line style for tables.
//
// A table with this line style looks like this:
//
//	+-----------+---------------------------+---+
//	| author_id |           name            | z |
//	+-----------+---------------------------+---+
//	|        14 | a       b       c       d |   |
//	|        15 | aoeu                     +|   |
//	|           | test                     +|   |
//	|           |                           |   |
//	+-----------+---------------------------+---+
func ASCIILineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'+', '-', '+', '+'},
		Mid:  [4]rune{'+', '-', '+', '+'},
		Row:  [4]rune{'|', ' ', '|', '|'},
		Wrap: [4]rune{'|', '+', '|', '|'},
		End:  [4]rune{'+', '-', '+', '+'},
	}
}

// OldASCIILineStyle is the old ASCII line style for tables.
//
// A table with this line style looks like this:
//
//	+-----------+---------------------------+---+
//	| author_id |           name            | z |
//	+-----------+---------------------------+---+
//	|        14 | a       b       c       d |   |
//	|        15 | aoeu                      |   |
//	|           : test                          |
//	|           :                               |
//	+-----------+---------------------------+---+
func OldASCIILineStyle() LineStyle {
	s := ASCIILineStyle()
	s.Wrap[1], s.Wrap[2] = ' ', ':'
	return s
}

// UnicodeLineStyle is the Unicode line style for tables.
//
// A table with this line style looks like this:
//
//	┌───────────┬───────────────────────────┬───┐
//	│ author_id │           name            │ z │
//	├───────────┼───────────────────────────┼───┤
//	│        14 │ a       b       c       d │   │
//	│        15 │ aoeu                     ↵│   │
//	│           │ test                     ↵│   │
//	│           │                           │   │
//	└───────────┴───────────────────────────┴───┘
func UnicodeLineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'┌', '─', '┬', '┐'},
		Mid:  [4]rune{'├', '─', '┼', '┤'},
		Row:  [4]rune{'│', ' ', '│', '│'},
		Wrap: [4]rune{'│', '↵', '│', '│'},
		End:  [4]rune{'└', '─', '┴', '┘'},
	}
}

// UnicodeDoubleLineStyle is the Unicode double line style for tables.
//
// A table with this line style looks like this:
//
//	╔═══════════╦═══════════════════════════╦═══╗
//	║ author_id ║           name            ║ z ║
//	╠═══════════╬═══════════════════════════╬═══╣
//	║        14 ║ a       b       c       d ║   ║
//	║        15 ║ aoeu                     ↵║   ║
//	║           ║ test                     ↵║   ║
//	║           ║                           ║   ║
//	╚═══════════╩═══════════════════════════╩═══╝
func UnicodeDoubleLineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'╔', '═', '╦', '╗'},
		Mid:  [4]rune{'╠', '═', '╬', '╣'},
		Row:  [4]rune{'║', ' ', '║', '║'},
		Wrap: [4]rune{'║', '↵', '║', '║'},
		End:  [4]rune{'╚', '═', '╩', '╝'},
	}
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
