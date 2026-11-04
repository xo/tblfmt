package tblfmt

import (
	"bytes"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestEncodeMarkdown makes sure that the markdown encoder writes a table that
// a GitHub markdown reader reads as the result set (D69). psql has no
// markdown format, so the expected output is not a measure of psql.
func TestEncodeMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]string
		cols   []string
		vals   [][][]any
		exp    string
	}{
		{
			"pipe and newline", map[string]string{"format": "markdown"},
			[]string{"n", "a|b"},
			[][][]any{{{1, "x|y"}, {22, "two\nlines"}, {nil, nil}}},
			"| n  |     a\\|b     |\n" +
				"|---:|:-------------|\n" +
				"|  1 | x\\|y         |\n" +
				"| 22 | two<br>lines |\n" +
				"|    |              |\n",
		},
		{
			// a column name that holds a newline is one line too
			"newline in a name", map[string]string{"format": "markdown", "header_align": "column"},
			[]string{"long\nname", "b"},
			[][][]any{{{1, "x"}}},
			"| long<br>name | b |\n" +
				"|-------------:|:--|\n" +
				"|            1 | x |\n",
		},
		{
			// a markdown table has no title, no footer and no expanded form,
			// and always has a header
			"ignored options", map[string]string{"format": "markdown", "title": "T", "footer": "on", "tuples_only": "on", "border": "0", "expanded": "on", "columns": "5"},
			[]string{"a"},
			[][][]any{{{"x"}}},
			"| a |\n" +
				"|:--|\n" +
				"| x |\n",
		},
		{
			// psql writes the header of a result set with no rows
			"no rows", map[string]string{"format": "markdown"},
			[]string{"a", "b"},
			[][][]any{{}},
			"| a | b |\n" +
				"|:--|:--|\n",
		},
		{
			// a markdown table needs a column
			"no columns", map[string]string{"format": "markdown"},
			[]string{},
			[][][]any{{{}, {}}},
			"",
		},
		{
			// a blank line ends a markdown table
			"result sets", map[string]string{"format": "markdown"},
			[]string{"a"},
			[][][]any{{{"x"}}, {{"y"}}},
			"| a |\n" +
				"|:--|\n" +
				"| x |\n" +
				"\n" +
				"| a |\n" +
				"|:--|\n" +
				"| y |\n",
		},
		{
			// a column with only null values aligns to the left
			"null column", map[string]string{"format": "markdown", "header_align": "column", "null": "(null)"},
			[]string{"a", "b"},
			[][][]any{{{1, nil}}},
			"| a | b      |\n" +
				"|--:|:-------|\n" +
				"| 1 | (null) |\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, internal.New(test.cols, test.vals...), test.params); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if s := buf.String(); s != test.exp {
				t.Errorf("expected:\n%q\ngot:\n%q", test.exp, s)
			}
		})
	}
}

// TestEncodeMarkdownCenter makes sure that a centered column has a colon at
// each side of its delimiter. WithAlign aligns a number, and not a string.
func TestEncodeMarkdownCenter(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	rs := internal.New([]string{"a", "b"}, [][]any{{1, "x"}, {333, "y"}})
	if err := EncodeMarkdownAll(buf, rs, WithFormatterOptions(WithAlign(AlignCenter))); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	exp := "|  a  | b |\n" +
		"|:---:|:--|\n" +
		"|  1  | x |\n" +
		"| 333 | y |\n"
	if s := buf.String(); s != exp {
		t.Errorf("expected:\n%q\ngot:\n%q", exp, s)
	}
}
