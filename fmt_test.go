package tblfmt

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/xo/tblfmt/internal"
)

func TestTabwidthCalc(t *testing.T) {
	tests := []struct {
		s      string
		offset int
		tab    int
		exp    int
	}{
		{"", 0, 8, 0},
		{" ", 0, 8, 1},
		{"    ", 0, 8, 4},
		{"\u8888", 0, 8, 2},
		{"\t", 0, 8, 8},
		{"\t\t", 0, 8, 16}, // 5
		{"\t\t ", 0, 8, 17},
		{" \t\t ", 0, 8, 17},
		{" \t\t\t ", 0, 8, 25},
		{"foo\tbar\t", 0, 8, 16},
		{"\t\t\u8888", 0, 8, 18}, // 10
		{"\u8888\t\u8888", 0, 8, 10},
		{"\u8888\t\u8888\t", 0, 8, 16},
		{"", 1, 8, 0}, // 13
		{"\t", 1, 4, 3},
		{" \t", 1, 4, 3},
		{" \t ", 1, 4, 4},
		{"\u8888\t\u8888\t", 1, 4, 7},
		/*
		   ---xxxxxxxxx (width == 9)
		  |   袈   袈  |
		*/
		{"\u8888\t\t\u8888\t", 3, 2, 9}, // 18
		/*
		   --------------xxxxxxxxxxxxxxxxxxxxxxxxxxxx (width == 28)
		  |              袈        袈              袈|
		*/
		{"\u8888\t\u8888\t\t\u8888", 14, 8, 28}, // 19
		{"袈	袈		袈", 14, 8, 28},
	}
	for i, test := range tests {
		tabs, w := tabpositions(test.s)
		w += tabwidth(tabs, test.offset, test.tab)
		if test.exp != w {
			t.Errorf("test %d %q expected tabwidth(%v, %d, %d) = %d, got: %d", i, test.s, tabs, test.offset, test.tab, test.exp, w)
		}
	}
}

var tabRE = regexp.MustCompile("\t")

// tabpositions returns a list of tab positions in s.
func tabpositions(s string) ([][2]int, int) {
	var tabs [][2]int
	var last int
	for _, m := range tabRE.FindAllStringIndex(s, -1) {
		tabs = append(tabs, [2]int{m[0], runewidth.StringWidth(s[last:m[0]])})
		last = m[0] + 1
	}
	return tabs, runewidth.StringWidth(s[last:])
}

func TestFormatBytesTabs(t *testing.T) {
	tests := []escTest{
		v("", 0),
		v("\u8888\t\u8888", 4),
		v(" \u8888 \t \u8888 ", 8),
	}
	for i, test := range tests {
		v := FormatBytes([]byte(test.s), nil, 0, false, false, 0, 0)
		if !reflect.DeepEqual(v, test.exp) {
			t.Errorf("test %d %q expected %v, got: %v", i, test.s, test.exp, v)
			width := runewidth.StringWidth(string(v.Buf))
			if v.Width != width {
				t.Errorf("test %d %q expected width %d, got: %d", i, test.s, width, v.Width)
			}
			if width != test.check {
				t.Errorf("test %d %q expected check width %d, got: %d", i, test.s, test.check, width)
			}
		}
	}
}

func TestFormatBytesComplex(t *testing.T) {
	s := `{
  "2011": "Team Garmin - Cervelo",
  "2012": "AA Drink - Leontien.nl",
  "2013": "Boels-Dolmans Cycling Team",
  "2015": "Boels-Dolmans"
}`
	v := FormatBytes([]byte(s), nil, 0, false, false, 0, 0)
	if w := v.MaxWidth(0, 8); w != 39 {
		t.Errorf("expected width of 39, got: %d", w)
	}
}

func TestFormatJSON(t *testing.T) {
	s := strings.Join(
		[]string{
			"\a",
			"\b",
			"\f",
			"\n",
			"\r",
			"\t",
			"\x1a",
			"\x2b",
			"\x3f",
			"\\",
			" ",
			"\x9f",
			"\xaf",
			"\xff",
			"\u1998",
			"\U0001f440",
			"\U0001f930",
			"foo",
			"15\u00f8C",
		},
		";",
	)
	// note: compared by what the encodings decode to and not byte for byte,
	// as more than one escaping of the same string is valid JSON, and
	// FormatBytes and encoding/json legitimately differ: an invalid UTF-8
	// byte becomes a \ufffd escape here, where encoding/json writes the
	// U+FFFD replacement rune itself.
	exp, err := json.Marshal(s, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	t.Logf("exp: %s", string(exp))
	v := FormatBytes([]byte(s), nil, 0, true, false, 0, 0)
	buf := []byte(`"` + v.String() + `"`)
	t.Logf("v  : %s", string(buf))
	var expStr string
	if err := json.Unmarshal(exp, &expStr); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	var s2 string
	if err := json.Unmarshal(buf, &s2); err != nil {
		t.Fatalf("expected %s to be valid json, got: %v", string(buf), err)
	}
	if s2 != expStr {
		t.Errorf("\nexpected:\n%q\ngot:\n%q", expStr, s2)
	}
}

func TestFormatBytesRaw(t *testing.T) {
	tests := []struct {
		s   string
		exp string
	}{
		{"", ""},
		{"a", "a"},
		{" ", `" "`},
		{"  ", `"  "`},
		{"    ", `"    "`},
		{"\n", "\"\n\""},
		{"\t", "\"\t\""},
		{",", "\",\""},
		{",\t", "\",\t\""},
		{",\t\"", "\",\t\"\"\""},
	}
	for i, test := range tests {
		v := FormatBytes([]byte(test.s), nil, 0, false, true, ',', '"')
		buf := v.Buf
		if v.Quoted {
			buf = append([]byte{'"'}, append(buf, '"')...)
		}
		if string(buf) != test.exp {
			t.Errorf("test %d %q expected %q == %q", i, test.s, string(buf), test.exp)
		}
	}
}

type escTest struct {
	s     string
	exp   *Value
	check int
}

func quote(s string) string {
	s = strconv.Quote(s)
	s = s[1 : len(s)-1]
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\n`, "\n")
	return s
}

func v(s string, check int) escTest {
	c := quote(s)
	tabs, width := tabpositions(c)
	var buf []byte
	if len(c) != 0 {
		buf = []byte(c)
	}
	v := &Value{
		Buf:   buf,
		Tabs:  [][][2]int{tabs},
		Width: width,
	}
	return escTest{s, v, check}
}

// TestFormatNull checks that the generic database/sql Null type and other
// driver.Valuer implementations format as their contained value, and that a
// null value formats as the empty value.
//
// MySQL's ColumnType.ScanType reports sql.Null[uint64] for a nullable BIGINT
// UNSIGNED, and the value is above math.MaxInt64, so it cannot round trip
// through the Null's own Value method.
func TestFormatNull(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	tm := time.Date(2026, 1, 2, 3, 4, 5, 0, loc)
	maxUint64 := sql.Null[uint64]{V: math.MaxUint64, Valid: true}
	tests := []struct {
		v     any
		exp   string
		align Align
	}{
		{sql.Null[uint64]{V: math.MaxUint64, Valid: true}, "18446744073709551615", AlignRight},
		{&maxUint64, "18446744073709551615", AlignRight},
		{sql.Null[uint64]{V: 42, Valid: true}, "42", AlignRight},
		{sql.Null[uint64]{}, "", AlignLeft},
		{&sql.Null[uint64]{}, "", AlignLeft},
		{sql.Null[int64]{V: -5, Valid: true}, "-5", AlignRight},
		{sql.Null[float64]{V: 3.25, Valid: true}, "3.25", AlignRight},
		{sql.Null[bool]{V: true, Valid: true}, "true", AlignLeft},
		{sql.Null[string]{V: "hello", Valid: true}, "hello", AlignLeft},
		{sql.Null[string]{}, "", AlignLeft},
		{sql.Null[time.Time]{V: tm, Valid: true}, "2026-01-02T03:04:05Z", AlignLeft},
		{sql.Null[[]byte]{V: []byte("bytes"), Valid: true}, "bytes", AlignLeft},
		{valuer{}, "", AlignLeft},
		{valuer{v: "from valuer"}, "from valuer", AlignLeft},
		{errValuer{}, "{}", AlignLeft},
		{(*sql.Null[uint64])(nil), "", AlignLeft},
		{nil, "", AlignLeft},
	}
	f := NewEscapeFormatter(WithTimeLocation(loc))
	for i, test := range tests {
		vals, err := f.Format([]any{test.v})
		switch {
		case err != nil:
			t.Errorf("test %d expected no error, got: %v", i, err)
			continue
		case len(vals) != 1:
			t.Errorf("test %d expected 1 value, got: %d", i, len(vals))
			continue
		}
		var s string
		var align Align
		if vals[0] != nil {
			s, align = string(vals[0].Buf), vals[0].Align
		}
		if s != test.exp {
			t.Errorf("test %d %#v expected %q, got: %q", i, test.v, test.exp, s)
		}
		if test.exp != "" && align != test.align {
			t.Errorf("test %d %#v expected align %d, got: %d", i, test.v, test.align, align)
		}
	}
}

// TestEncodeJSONNull checks that the generic database/sql Null type encodes as
// a bare JSON value, and as JSON null when not valid.
func TestEncodeJSONNull(t *testing.T) {
	t.Parallel()
	resultSet := internal.New([]string{"b"}, [][]any{
		{&sql.Null[uint64]{V: math.MaxUint64, Valid: true}},
		{&sql.Null[uint64]{}},
	})
	buf := new(bytes.Buffer)
	if err := EncodeJSONAll(buf, resultSet); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// note: quoted because it is above the int64 a JSON number is read as,
	// with its digits intact; see TestEncodeJSONNumber.
	const exp = `[{"b":"18446744073709551615"},{"b":null}]`
	if s := strings.TrimSpace(buf.String()); s != exp {
		t.Errorf("expected %s, got: %s", exp, s)
	}
}

// valuer is a driver.Valuer returning a string.
type valuer struct {
	v string
}

func (v valuer) Value() (driver.Value, error) {
	if v.v == "" {
		return nil, nil
	}
	return v.v, nil
}

// errValuer is a driver.Valuer always returning an error.
type errValuer struct{}

func (errValuer) Value() (driver.Value, error) {
	return nil, errors.New("invalid value")
}

// TestEncodeNumericLocale checks that a locale formatted number does not break
// the csv and JSON encodings, where its grouping separator would otherwise be
// read as a field separator, or as an invalid JSON number.
func TestEncodeNumericLocale(t *testing.T) {
	t.Parallel()
	// note: the locales cover the three ways a grouping separator meets csv --
	// a comma, which is the separator itself; a period paired with a decimal
	// comma, where only the float needs quoting; and a non-breaking space,
	// which is quoted as whitespace.
	tests := []struct {
		locale string
		n      string
		f      string
		b      string
	}{
		{"en-US", "1,234,567", "1,234,567.25", "18,446,744,073,709,551,615"},
		{"en-IN", "12,34,567", "12,34,567.25", "1,84,46,74,40,73,70,95,51,615"},
		{"de-DE", "1.234.567", "1.234.567,25", "18.446.744.073.709.551.615"},
		{"fr-FR", "1\u00a0234\u00a0567", "1\u00a0234\u00a0567,25", "18\u00a0446\u00a0744\u00a0073\u00a0709\u00a0551\u00a0615"},
	}
	for _, test := range tests {
		resultSet := func() ResultSet {
			return internal.New([]string{"n", "f", "b"}, [][]any{
				{1234567, 1234567.25, &sql.Null[uint64]{V: math.MaxUint64, Valid: true}},
				{nil, nil, &sql.Null[uint64]{}},
			})
		}
		buf := new(bytes.Buffer)
		if err := EncodeCSVAll(buf, resultSet(), WithFormatter(NewEscapeFormatter(
			WithNumericLocale(true, test.locale),
			WithIsRaw(true, ',', '"'),
		))); err != nil {
			t.Fatalf("%s expected no error, got: %v", test.locale, err)
		}
		t.Logf("%s csv:\n%s", test.locale, buf.String())
		records, err := csv.NewReader(bytes.NewReader(buf.Bytes())).ReadAll()
		if err != nil {
			t.Errorf("%s expected no error, got: %v", test.locale, err)
			continue
		}
		if len(records) != 3 {
			t.Errorf("%s expected 3 csv records, got: %d", test.locale, len(records))
			continue
		}
		for i, record := range records {
			if len(record) != 3 {
				t.Errorf("%s csv record %d expected 3 fields, got: %d (%q)", test.locale, i, len(record), record)
			}
		}
		if exp := []string{test.n, test.f, test.b}; !slices.Equal(records[1], exp) {
			t.Errorf("%s expected csv %q, got: %q", test.locale, exp, records[1])
		}
		buf.Reset()
		if err := EncodeJSONAll(buf, resultSet(), WithFormatter(NewEscapeFormatter(
			WithNumericLocale(true, test.locale),
			WithIsJSON(true),
		))); err != nil {
			t.Fatalf("%s expected no error, got: %v", test.locale, err)
		}
		t.Logf("%s json: %s", test.locale, buf.String())
		// note: json ignores the numeric locale, so the numbers stay numbers
		// rather than becoming strings; they are compared as their raw text so
		// that a uint64 above 2^53 is not rounded by decoding it to a float.
		var v []map[string]jsontext.Value
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &v); err != nil {
			t.Errorf("%s expected %s to be valid json, got: %v", test.locale, buf.String(), err)
			continue
		}
		// note: b is above the int64 a JSON number is read as, so it is a
		// JSON string -- of its exact digits, never locale formatted.
		exp := map[string]string{
			"n": "1234567",
			"f": "1.23456725e+06",
			"b": `"18446744073709551615"`,
		}
		if len(v) != 2 {
			t.Errorf("%s expected 2 json rows, got: %d", test.locale, len(v))
			continue
		}
		for _, col := range []string{"n", "f", "b"} {
			if s := string(v[0][col]); s != exp[col] {
				t.Errorf("%s expected json %s to be %s, got: %s", test.locale, col, exp[col], s)
			}
		}
		if s := string(v[1]["b"]); s != "null" {
			t.Errorf("%s expected null, got: %s", test.locale, s)
		}
	}
}

// TestEncodeJSONNumber checks that a number JSON cannot write as a number --
// one above the int64 a JSON number is read as, and the values that are not
// finite numbers at all -- is written as a string of the same text the other
// formats show.
func TestEncodeJSONNumber(t *testing.T) {
	t.Parallel()
	resultSet := internal.New(
		[]string{"u64max", "u64min", "i64max", "u64ok", "nan", "inf", "ninf", "f"},
		[][]any{{
			uint64(math.MaxUint64),
			uint64(math.MaxInt64) + 1,
			int64(math.MaxInt64),
			uint64(42),
			math.NaN(),
			math.Inf(1),
			math.Inf(-1),
			1.5,
		}},
	)
	buf := new(bytes.Buffer)
	if err := EncodeJSONAll(buf, resultSet); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	t.Logf("json: %s", buf.String())
	var v []map[string]jsontext.Value
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &v); err != nil {
		t.Fatalf("expected %s to be valid json, got: %v", buf.String(), err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 json row, got: %d", len(v))
	}
	// note: compared as raw text, as decoding would round the large values it
	// is the point of the quoting to keep exact.
	for _, test := range []struct {
		col string
		exp string
	}{
		{"u64max", `"18446744073709551615"`},
		{"u64min", `"9223372036854775808"`},
		{"i64max", `9223372036854775807`},
		{"u64ok", `42`},
		{"nan", `"NaN"`},
		{"inf", `"Infinity"`},
		{"ninf", `"-Infinity"`},
		{"f", `1.5`},
	} {
		if s := string(v[0][test.col]); s != test.exp {
			t.Errorf("expected %s to be %s, got: %s", test.col, test.exp, s)
		}
	}
}

// TestFormatNotANumber checks that the values that are not finite numbers use
// PostgreSQL's spellings, which are what psql displays, in every format.
func TestFormatNotANumber(t *testing.T) {
	t.Parallel()
	f := NewEscapeFormatter()
	for i, test := range []struct {
		v   any
		exp string
	}{
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{float32(math.Inf(1)), "Infinity"},
		{sql.NullFloat64{Float64: math.NaN(), Valid: true}, "NaN"},
		{sql.Null[float64]{V: math.Inf(-1), Valid: true}, "-Infinity"},
	} {
		vals, err := f.Format([]any{test.v})
		if err != nil {
			t.Errorf("test %d expected no error, got: %v", i, err)
			continue
		}
		if s := string(vals[0].Buf); s != test.exp {
			t.Errorf("test %d expected %q, got: %q", i, test.exp, s)
		}
		if vals[0].Align != AlignRight {
			t.Errorf("test %d expected right align, got: %d", i, vals[0].Align)
		}
	}
}
