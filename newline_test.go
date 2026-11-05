package tblfmt

import (
	"bytes"
	"maps"
	"strings"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestWithNewline makes sure that the table and expanded encoders end each
// line with the newline of WithNewline, the summary too (B9). psql has no such
// variable for the aligned format, so the expected output is not a measure of
// psql.
func TestWithNewline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		enc  func(ResultSet, ...Option) (Encoder, error)
		vals [][]any
		exp  string
	}{
		{"table", NewTableEncoder, [][]any{{1}}, " a \r\n---\r\n 1\r\n(1 row)\r\n"},
		{"expanded", NewExpandedEncoder, [][]any{{1}}, "-[ RECORD 1 ]\r\na | 1\r\n"},
		{"expanded no rows", NewExpandedEncoder, [][]any{}, "(0 rows)\r\n"},
	}
	for _, test := range tests {
		enc, err := test.enc(internal.New([]string{"a"}, test.vals), WithNewline("\r\n"))
		if err != nil {
			t.Fatalf("%s: expected no error, got: %v", test.name, err)
		}
		buf := new(bytes.Buffer)
		if err := enc.EncodeAll(buf); err != nil {
			t.Fatalf("%s: expected no error, got: %v", test.name, err)
		}
		if s := buf.String(); s != test.exp {
			t.Errorf("%s: expected %q, got: %q", test.name, test.exp, s)
		}
	}
}

// TestWithNewlineFormats makes sure that WithNewline sets the newline of each
// line in each format, other than unaligned and csv, which follow recordsep.
// The output with "\r\n" is the output with "\n", with each newline replaced.
func TestWithNewlineFormats(t *testing.T) {
	t.Parallel()
	formats := []map[string]string{
		{"format": "aligned", "title": "T", "border": "2"},
		{"format": "wrapped", "title": "T", "columns": "20"},
		{"format": "aligned", "expanded": "on", "title": "T"},
		{"format": "aligned", "expanded": "auto", "columns": "10"},
		{"format": "table", "title": "T"},
		{"format": "markdown"},
		{"format": "json"},
		{"format": "html", "title": "T"},
		{"format": "html", "expanded": "on"},
		{"format": "asciidoc", "title": "T"},
		{"format": "latex", "title": "T"},
		{"format": "latex-longtable", "title": "T"},
		{"format": "troff-ms", "title": "T"},
		{"format": "vertical"},
	}
	encode := func(params map[string]string, opts ...Option) string {
		rs := internal.New([]string{"a", "long\nname"}, [][]any{
			{1, "x\ny"},
			{22, "a long value that wraps"},
		}, [][]any{{3, "z"}})
		buf := new(bytes.Buffer)
		if err := EncodeAll(buf, rs, maps.Clone(params), opts...); err != nil {
			t.Fatalf("%v: expected no error, got: %v", params, err)
		}
		return buf.String()
	}
	for _, params := range formats {
		exp := strings.ReplaceAll(encode(params), "\n", "\r\n")
		if s := encode(params, WithNewline("\r\n")); s != exp {
			t.Errorf("%v: expected:\n%q\ngot:\n%q", params, exp, s)
		}
	}
}

// TestEncodeRecordSeparator makes sure that the unaligned encoder writes the
// record separator between two lines, and ends the last line as psql does: a
// newline, or a zero byte for recordsep_zero.
//
// Measured against psql 18.6, with od -c for the bytes:
//
//	$ psql -X -P format=unaligned -P recordsep=X -c "select g as a,
//	    chr(119+g) as b from generate_series(1,2) g"
//	a|bX1|xX2|yX(2 rows)\n
//
// The other cases are the same command with the other \pset values of each
// case.
func TestEncodeRecordSeparator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		params map[string]string
		exp    string
	}{
		{map[string]string{"recordsep": "X"}, "a|bX1|xX2|yX(2 rows)\n"},
		{map[string]string{"recordsep": "X", "expanded": "on"}, "a|1Xb|xXXa|2Xb|y\n"},
		{map[string]string{"recordsep": "X", "title": "T"}, "TXa|bX1|xX2|yX(2 rows)\n"},
		{map[string]string{"recordsep": "X", "expanded": "on", "title": "T"}, "TXXa|1Xb|xXXa|2Xb|y\n"},
		{map[string]string{"recordsep": "X", "footer": "off"}, "a|bX1|xX2|y\n"},
		{map[string]string{"recordsep": "X", "tuples_only": "on"}, "1|xX2|y\n"},
		{map[string]string{"recordsep": "X", "expanded": "on", "tuples_only": "on"}, "a|1Xb|xXXa|2Xb|y\n"},
		{map[string]string{"recordsep_zero": "on"}, "a|b\x001|x\x002|y\x00(2 rows)\x00"},
		{map[string]string{"recordsep_zero": "on", "expanded": "on"}, "a|1\x00b|x\x00\x00a|2\x00b|y\x00"},
	}
	for _, test := range tests {
		test.params["format"] = "unaligned"
		buf := new(bytes.Buffer)
		rs := internal.New([]string{"a", "b"}, [][]any{{1, "x"}, {2, "y"}})
		if err := EncodeAll(buf, rs, test.params); err != nil {
			t.Fatalf("%v: expected no error, got: %v", test.params, err)
		}
		if s := buf.String(); s != test.exp {
			t.Errorf("%v: expected %q, got: %q", test.params, test.exp, s)
		}
	}
}
