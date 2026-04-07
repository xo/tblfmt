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

// TransformStyle is a column/header transform style.
type TransformStyle int

// Transform styles.
const (
	TransformNone TransformStyle = iota
	TransformForceLower
	TransformForceUpper
	TransformUpperToLower
	TransformLowerToUpper
)

// Transform transforms style. Satisifies the [Transformer] interface.
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

// LineStyle is a table line style.
//
// See the ASCII, OldASCII, and Unicode styles below for predefined table
// styles.
//
// Tables generally look like the following:
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
// When border is 0, then no surrounding borders will be shown:
//
//	author_id           name            z
//	--------- ------------------------- -
//	       14 a       b       c       d
//	       15 aoeu                     +
//	          test                     +
//
// When border is 1, then a border between columns will be shown:
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

// TableLineStyle is the table line style for tables.
//
// Tables using this style will look like the following:
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
// Tables using this style will look like the following:
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
// Tables using this style will look like the following:
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
// Tables using this style will look like the following:
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
// Tables using this style will look like the following:
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

// DefaultTableSummary is the default table summary.
//
// Default table summaries look like the following:
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
