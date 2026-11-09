package tblfmt

import (
	"bytes"
	"errors"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestEncodeExpandedHeaderWidth makes sure that xheader_width sets the header
// line of each expanded record as psql does (D76). TestPsql compares more
// cases with psql.
//
// Measured against psql 18.6:
//
//	$ psql -X -P expanded=on -P border=1 -P xheader_width=20 -c "select 1 as a,
//	    'a much longer value that is wide' as b"
//	-[ RECORD 1 ]-------
//	a | 1
//	b | a much longer value that is wide
//
// The other cases are the same command with the other values. psql refuses 0,
// a word and an empty value with "\pset: allowed xheader_width values are
// "full" (default), "column", "page", or a number specifying the exact
// width". It reads " 20px" as 20, as atoi in C does.
func TestEncodeExpandedHeaderWidth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		params map[string]string
		exp    string
	}{
		{map[string]string{"border": "1", "xheader_width": "full"}, "-[ RECORD 1 ]-----------------------\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": "column"}, "-[ RECORD 1 ]\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": "20"}, "-[ RECORD 1 ]-------\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": " 20px"}, "-[ RECORD 1 ]-------\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": "5"}, "-[ RECORD 1 ]\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": "-3"}, "-[ RECORD 1 ]-----------------------\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "1", "xheader_width": "Page", "columns": "30"}, "-[ RECORD 1 ]-----------------\na | 1\nb | a much longer value that is wide\n"},
		{map[string]string{"border": "0", "xheader_width": "20"}, "* Record 1          \na 1\nb a much longer value that is wide\n"},
		{map[string]string{"border": "2", "xheader_width": "20"}, "+-[ RECORD 1 ]-----+\n| a | 1                                |\n| b | a much longer value that is wide |\n+---+--------------+\n"},
		{map[string]string{"border": "2", "xheader_width": "column"}, "+-[ RECORD 1 ]\n| a | 1                                |\n| b | a much longer value that is wide |\n+---+\n"},
	}
	for _, test := range tests {
		test.params["format"], test.params["expanded"] = "aligned", "on"
		buf := new(bytes.Buffer)
		rs := internal.New([]string{"a", "b"}, [][]any{{1, "a much longer value that is wide"}})
		if err := EncodeAll(buf, rs, test.params, WithSummary(Summary{})); err != nil {
			t.Fatalf("%v: expected no error, got: %v", test.params, err)
		}
		if s := buf.String(); s != test.exp {
			t.Errorf("%v: expected:\n%q\ngot:\n%q", test.params, test.exp, s)
		}
	}
	for _, s := range []string{"0", "abc", ""} {
		rs := internal.New([]string{"a"}, [][]any{{1}})
		params := map[string]string{"format": "aligned", "expanded": "on", "xheader_width": s}
		if err := EncodeAll(new(bytes.Buffer), rs, params); !errors.Is(err, ErrInvalidExpandedHeaderWidth) {
			t.Errorf("%q: expected %v, got: %v", s, ErrInvalidExpandedHeaderWidth, err)
		}
	}
}
