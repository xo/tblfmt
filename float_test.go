package tblfmt

import (
	"bytes"
	"database/sql"
	"math"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestEncodeFloat makes sure that a float64 is written as psql writes a
// float8, and a float32 as psql writes a real, in each format. usql issue 448
// asked for this.
//
// Measured against psql 18.6, with LC_NUMERIC=en_US.UTF-8:
//
//	$ psql -X -c "select '1450817.032'::float8 as a, 123456789012345::float8 as b,
//	    1e15::float8 as c, 1e-5::float8 as d, 0.0001::float8 as e,
//	    -1234567.5::float8 as f, 2::float8 as g, '1.5e300'::float8 as h,
//	    '-0'::float8 as i, 0.1::float8 as j"
//	      a      |        b        |   c   |   d   |   e    |     f      | g |    h     | i  |  j
//	-------------+-----------------+-------+-------+--------+------------+---+----------+----+-----
//	 1450817.032 | 123456789012345 | 1e+15 | 1e-05 | 0.0001 | -1234567.5 | 2 | 1.5e+300 | -0 | 0.1
//	(1 row)
//
//	$ psql -X -c "select '1450817.032'::real as a, 1234.5::real as b,
//	    2::real as c, '1e-5'::real as d"
//	      a       |   b    | c |   d
//	--------------+--------+---+-------
//	 1.450817e+06 | 1234.5 | 2 | 1e-05
//	(1 row)
//
// The other cases are the same commands with -P numericlocale=on, format=csv
// and format=unaligned. The numeric locale groups only the digits before the
// decimal point, and it does not change an exponent. JSON writes the same
// digits as the other formats, as JSON numbers. The expanded encoder keeps the
// numeric locale too (B6):
//
//	$ psql -X -P numericlocale=on -P expanded=on -c "select
//	    '1450817.032'::float8 as a, 1e15::float8 as c"
//	-[ RECORD 1 ]----
//	a | 1,450,817.032
//	c | 1e+15
func TestEncodeFloat(t *testing.T) {
	t.Parallel()
	float8 := func() ResultSet {
		return internal.New([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, [][]any{{
			1450817.032, float64(123456789012345), 1e15, 1e-5, 0.0001, -1234567.5,
			float64(2), 1.5e300, math.Copysign(0, -1), 0.1,
		}})
	}
	real4 := func() ResultSet {
		return internal.New([]string{"a", "b", "c", "d"}, [][]any{{
			float32(1450817.032), float32(1234.5), float32(2), float32(1e-5),
		}})
	}
	expanded := func() ResultSet {
		return internal.New([]string{"a", "c"}, [][]any{{1450817.032, 1e15}})
	}
	tests := []struct {
		rs     func() ResultSet
		params map[string]string
		exp    string
	}{
		{float8, map[string]string{"format": "aligned"}, "      a      |        b        |   c   |   d   |   e    |     f      | g |    h     | i  |  j  \n-------------+-----------------+-------+-------+--------+------------+---+----------+----+-----\n 1450817.032 | 123456789012345 | 1e+15 | 1e-05 | 0.0001 | -1234567.5 | 2 | 1.5e+300 | -0 | 0.1\n(1 row)\n"},
		{float8, map[string]string{"format": "aligned", "numericlocale": "on"}, "       a       |          b          |   c   |   d   |   e    |      f       | g |    h     | i  |  j  \n---------------+---------------------+-------+-------+--------+--------------+---+----------+----+-----\n 1,450,817.032 | 123,456,789,012,345 | 1e+15 | 1e-05 | 0.0001 | -1,234,567.5 | 2 | 1.5e+300 | -0 | 0.1\n(1 row)\n"},
		{float8, map[string]string{"format": "csv"}, "a,b,c,d,e,f,g,h,i,j\n1450817.032,123456789012345,1e+15,1e-05,0.0001,-1234567.5,2,1.5e+300,-0,0.1\n"},
		{float8, map[string]string{"format": "unaligned"}, "a|b|c|d|e|f|g|h|i|j\n1450817.032|123456789012345|1e+15|1e-05|0.0001|-1234567.5|2|1.5e+300|-0|0.1\n(1 row)\n"},
		{float8, map[string]string{"format": "json"}, `[{"a":1450817.032,"b":123456789012345,"c":1e+15,"d":1e-05,"e":0.0001,"f":-1234567.5,"g":2,"h":1.5e+300,"i":-0,"j":0.1}]` + "\n"},
		{expanded, map[string]string{"format": "aligned", "numericlocale": "on", "expanded": "on"}, "-[ RECORD 1 ]----\na | 1,450,817.032\nc | 1e+15\n"},
		{real4, map[string]string{"format": "aligned"}, "      a       |   b    | c |   d   \n--------------+--------+---+-------\n 1.450817e+06 | 1234.5 | 2 | 1e-05\n(1 row)\n"},
		{real4, map[string]string{"format": "aligned", "numericlocale": "on"}, "      a       |    b    | c |   d   \n--------------+---------+---+-------\n 1.450817e+06 | 1,234.5 | 2 | 1e-05\n(1 row)\n"},
	}
	for _, test := range tests {
		buf := new(bytes.Buffer)
		if err := EncodeAll(buf, test.rs(), test.params); err != nil {
			t.Fatalf("%v: expected no error, got: %v", test.params, err)
		}
		if s := buf.String(); s != test.exp {
			t.Errorf("%v: expected:\n%q\ngot:\n%q", test.params, test.exp, s)
		}
	}
}

// TestFormatFloat makes sure that the formatter writes the edge cases of a
// float64 as psql writes a float8. psql writes an exponent when the decimal
// exponent is less than -4 or more than 14.
func TestFormatFloat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		v      any
		locale string
		exp    string
	}{
		{999999999999999.0, "", "999999999999999"},
		{99999999999999.9, "", "99999999999999.9"},
		{1e16, "", "1e+16"},
		{0.00012, "", "0.00012"},
		{0.00009, "", "9e-05"},
		{1.5e-300, "", "1.5e-300"},
		{float64(0), "", "0"},
		{sql.NullFloat64{Float64: 1450817.032, Valid: true}, "", "1450817.032"},
		{sql.Null[float64]{V: 1450817.032, Valid: true}, "", "1450817.032"},
		{sql.NullFloat64{Float64: 1234567.5, Valid: true}, "en-US", "1,234,567.5"},
		{1234567.25, "de-DE", "1.234.567,25"},
		{-1234567.25, "fr-FR", "-1 234 567,25"},
		{float64(2), "de-DE", "2"},
		{1.5e16, "de-DE", "1,5e+16"},
		{float32(1.5e16), "de-DE", "1,5e+16"},
	}
	for i, test := range tests {
		var opts []EscapeFormatterOption
		if test.locale != "" {
			opts = append(opts, WithNumericLocale(true, test.locale))
		}
		vals, err := NewEscapeFormatter(opts...).Format([]any{test.v})
		if err != nil {
			t.Fatalf("test %d expected no error, got: %v", i, err)
		}
		if s := vals[0].String(); s != test.exp {
			t.Errorf("test %d expected %q, got: %q", i, test.exp, s)
		}
		if vals[0].Align != AlignRight {
			t.Errorf("test %d expected %v, got: %v", i, AlignRight, vals[0].Align)
		}
	}
}
