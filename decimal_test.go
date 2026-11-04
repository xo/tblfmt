package tblfmt

import (
	"bytes"
	"database/sql"
	"math/big"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestEncodeDecimal makes sure that a value with the Decompose method of a
// database/sql decimal is written as psql writes a numeric, in each format
// (D68). go-hdb returns a SAP HANA DECIMAL as such a value.
//
// Measured against psql 18.6:
//
//	$ psql -X -c "select 1.5::numeric(9,2) as a, -0.05::numeric as b,
//	    12e3::numeric as c, 'Infinity'::numeric as d, '-Infinity'::numeric as e,
//	    'NaN'::numeric as f, 0::numeric(5,3) as g, 123456789.25::numeric as h"
//	  a   |   b   |   c   |    d     |     e     |  f  |   g   |      h
//	------+-------+-------+----------+-----------+-----+-------+--------------
//	 1.50 | -0.05 | 12000 | Infinity | -Infinity | NaN | 0.000 | 123456789.25
//	(1 row)
//
// The other cases are the same command with -P numericlocale=on, format=csv
// and format=unaligned. The JSON case follows to_jsonb of the same row:
//
//	{"a": 1.50, "b": -0.05, "c": 12000, "d": "Infinity", "e": "-Infinity",
//	 "f": "NaN", "g": 0.000, "h": 123456789.25}
func TestEncodeDecimal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		params map[string]string
		exp    string
	}{
		{map[string]string{"format": "aligned"}, "  a   |   b   |   c   |    d     |     e     |  f  |   g   |      h       \n------+-------+-------+----------+-----------+-----+-------+--------------\n 1.50 | -0.05 | 12000 | Infinity | -Infinity | NaN | 0.000 | 123456789.25\n(1 row)\n"},
		{map[string]string{"format": "aligned", "numericlocale": "on"}, "  a   |   b   |   c    |    d     |     e     |  f  |   g   |       h        \n------+-------+--------+----------+-----------+-----+-------+----------------\n 1.50 | -0.05 | 12,000 | Infinity | -Infinity | NaN | 0.000 | 123,456,789.25\n(1 row)\n"},
		{map[string]string{"format": "csv"}, "a,b,c,d,e,f,g,h\n1.50,-0.05,12000,Infinity,-Infinity,NaN,0.000,123456789.25\n"},
		{map[string]string{"format": "unaligned"}, "a|b|c|d|e|f|g|h\n1.50|-0.05|12000|Infinity|-Infinity|NaN|0.000|123456789.25\n(1 row)\n"},
		// JSON writes a finite decimal as a number, and the others as strings
		{map[string]string{"format": "json"}, `[{"a":1.50,"b":-0.05,"c":12000,"d":"Infinity","e":"-Infinity","f":"NaN","g":0.000,"h":123456789.25}]` + "\n"},
	}
	for _, test := range tests {
		rs := internal.New([]string{"a", "b", "c", "d", "e", "f", "g", "h"}, [][]any{{
			hdbDecimal{coef: 150, exp: -2},
			hdbDecimal{negative: true, coef: 5, exp: -2},
			hdbDecimal{coef: 12, exp: 3},
			hdbDecimal{form: 1},
			hdbDecimal{form: 1, negative: true},
			hdbDecimal{form: 2},
			hdbDecimal{exp: -3},
			hdbDecimal{coef: 12345678925, exp: -2},
		}})
		buf := new(bytes.Buffer)
		if err := EncodeAll(buf, rs, test.params); err != nil {
			t.Fatalf("%v: expected no error, got: %v", test.params, err)
		}
		if s := buf.String(); s != test.exp {
			t.Errorf("%v: expected:\n%q\ngot:\n%q", test.params, test.exp, s)
		}
	}
}

// TestFormatDecimal makes sure that the formatter writes the edge cases of a
// decimal as psql writes a numeric. psql 18.6 writes -0.00::numeric(5,2) as
// 0.00, and 1.5e-10::numeric as 0.00000000015.
func TestFormatDecimal(t *testing.T) {
	t.Parallel()
	maxDigits := hdbDecimal{coef: 15, exp: maxDecimalDigits}
	// note: the encoders scan into *any, so a driver value of *apdDecimal
	// reaches the formatter as a pointer, with its Decompose method
	ptr := any(&apdDecimal{coef: 15, exp: 9})
	tests := []struct {
		v      any
		locale string
		exp    string
	}{
		{hdbDecimal{negative: true, exp: -2}, "", "0.00"},
		{hdbDecimal{coef: 15, exp: -11}, "", "0.00000000015"},
		{hdbDecimal{exp: 3}, "", "0"},
		{hdbDecimal{negative: true, coef: 15, exp: -1}, "", "-1.5"},
		{hdbDecimal{coef: 15, exp: maxDecimalDigits + 1}, "", "15e131073"},
		{hdbDecimal{coef: 15, exp: -maxDecimalDigits - 1}, "", "15e-131073"},
		{&maxDigits, "", "15" + string(bytes.Repeat([]byte{'0'}, maxDecimalDigits))},
		{sql.Null[hdbDecimal]{V: hdbDecimal{coef: 25, exp: -1}, Valid: true}, "", "2.5"},
		{&ptr, "", "15000000000"},
		{hdbDecimal{negative: true, coef: 123456725, exp: -2}, "de-DE", "-1.234.567,25"},
		{hdbDecimal{coef: 123456725, exp: -2}, "fr-FR", "1 234 567,25"},
		// note: an integer part that does not fit a uint64 is not grouped
		{hdbDecimal{coef: 1, exp: 20}, "en-US", "100000000000000000000"},
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

// hdbDecimal is a decimal type like the one of go-hdb. It has the Decompose
// method of a database/sql decimal, unexported fields, and no String or Value
// method.
type hdbDecimal struct {
	form     byte
	negative bool
	coef     uint64
	exp      int32
}

// Decompose satisfies the decimal interface of database/sql.
func (d hdbDecimal) Decompose([]byte) (byte, bool, []byte, int32) {
	return d.form, d.negative, new(big.Int).SetUint64(d.coef).Bytes(), d.exp
}

// apdDecimal is a decimal type like the one of apd, which dbimp returns for a
// decimal column. Its Decompose method has a pointer receiver.
type apdDecimal hdbDecimal

// Decompose satisfies the decimal interface of database/sql.
func (d *apdDecimal) Decompose(buf []byte) (byte, bool, []byte, int32) {
	return hdbDecimal(*d).Decompose(buf)
}
