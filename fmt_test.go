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
	// note: the test compares what the two encodings decode to, not their
	// bytes. More than one escaped form of a string is valid JSON, and
	// FormatBytes and encoding/json write different forms. FormatBytes writes
	// an invalid UTF-8 byte as a \ufffd escape, and encoding/json writes the
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

// TestFormatNull makes sure that the generic database/sql Null type and other
// driver.Valuer implementations format as the value that they contain, and
// that a null value formats as the empty value.
//
// For a nullable BIGINT UNSIGNED, MySQL's ColumnType.ScanType reports
// sql.Null[uint64]. The test value is above math.MaxInt64, so it cannot make a
// round trip through the Value method of the Null.
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

// TestEncodeJSONNull makes sure that the generic database/sql Null type
// encodes as a bare JSON value, and as JSON null when it is not valid.
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
	// note: the test compares what the output decodes to, not its text,
	// because the golden tests cover the layout of the encoder. b is a JSON
	// string because it is a uint64. See TestEncodeJSONNumber.
	var v []map[string]jsontext.Value
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &v); err != nil {
		t.Fatalf("expected %s to be valid json, got: %v", buf.String(), err)
	}
	if len(v) != 2 {
		t.Fatalf("expected 2 json rows, got: %d", len(v))
	}
	if s, exp := string(v[0]["b"]), `"18446744073709551615"`; s != exp {
		t.Errorf("expected %s, got: %s", exp, s)
	}
	if s := string(v[1]["b"]); s != "null" {
		t.Errorf("expected null, got: %s", s)
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

// TestEncodeNumericLocale makes sure that a number formatted for a locale does
// not break the CSV and JSON encodings. If an encoder does not handle the
// grouping separator of the number, a reader reads it as a CSV field
// separator, or as part of an invalid JSON number.
func TestEncodeNumericLocale(t *testing.T) {
	t.Parallel()
	// note: the locales cover the three ways that a grouping separator
	// affects CSV. A comma is the field separator itself. A period comes with
	// a decimal comma, so only the float needs quotes. The encoder quotes a
	// non-breaking space because it is whitespace.
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
		// note: the JSON encoder ignores the numeric locale, so the numbers
		// stay numbers and do not become strings. The test compares their raw
		// text, so that a decode to a float does not round a uint64 above 2^53.
		var v []map[string]jsontext.Value
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &v); err != nil {
			t.Errorf("%s expected %s to be valid json, got: %v", test.locale, buf.String(), err)
			continue
		}
		// note: b is a uint64, so it is a JSON string. The string holds its
		// exact digits, and the numeric locale does not change them.
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

// TestEncodeJSONNumber makes sure that the JSON encoder writes a number as a
// string when JSON cannot write it as a number. One such number is above the
// int64 range, and a reader reads a JSON number as an int64. The others are
// the values that are not finite numbers. The string has the same text that
// the other formats show.
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
	// note: the test compares raw text, because decoding rounds the large
	// values. The quotes are there to keep these values exact.
	for _, test := range []struct {
		col string
		exp string
	}{
		{"u64max", `"18446744073709551615"`},
		{"u64min", `"9223372036854775808"`},
		{"i64max", `9223372036854775807`},
		// note: quoted although it fits, because the decision is by type, so
		// that a column is one JSON type for all of its rows
		{"u64ok", `"42"`},
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

// TestFormatNotANumber makes sure that the values that are not finite numbers
// use the PostgreSQL spellings in every format. psql shows these spellings.
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

// TestFormatBytesClusters makes sure that FormatBytes measures text by grapheme
// cluster, as a current terminal draws it, and not rune by rune.
//
// Note: psql counts each rune, so it pads a cluster such as 👍🏽 by 2 columns
// too few. tblfmt does not follow psql here. See D39 in docs/PLAN.md.
func TestFormatBytesClusters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		s     string
		raw   bool
		width int
		lines []int
		tabs  [][][2]int
	}{
		{"skin tone", "👍🏽", false, 2, nil, nil},
		{"flag", "🇯🇵", false, 2, nil, nil},
		{"combining mark", "é", false, 1, nil, nil},
		{"wide text", "日本語", false, 6, nil, nil},
		{"text around a cluster", "a👍🏽b", false, 4, nil, nil},
		// note: go-runewidth v0.0.31 counts these as 1 column, and emojiWidth
		// adds the missing column. See B38 in docs/BACKLOG.md.
		{"emoji form", "\u2764\ufe0f", false, 2, nil, nil},
		{"emoji form of a symbol", "\u00a9\ufe0f", false, 2, nil, nil},
		{"keycap", "1\ufe0f\u20e3", false, 2, nil, nil},
		{"emoji forms side by side", "\u2764\ufe0f\u2764\ufe0f", false, 4, nil, nil},
		{"wide base with the selector", "\u231a\ufe0f", false, 2, nil, nil},
		{"selector with no emoji form", "a\ufe0f", false, 1, nil, nil},
		{"selector alone", "\ufe0f", false, 0, nil, nil},
		{"joined in raw mode", "\U0001f3f3\ufe0f\u200d\U0001f308", true, 2, nil, nil},
		{"joined emoji", "\U0001f468\u200d\U0001f469\u200d\U0001f467", false, 2, nil, nil},
		{"joined emoji form", "\U0001f3f3\ufe0f\u200d\U0001f308", false, 2, nil, nil},
		{"flag with tags", "\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f", false, 2, nil, nil},
		// \a is escaped, and ends the text on each side of it
		{"escape between clusters", "👍🏽\a👍🏽", false, 2 + 2 + 2, nil, nil},
		{"tab after a cluster", "👍🏽\t👍🏽", false, 2, nil, [][][2]int{{{8, 2}}}},
		{"newline after a cluster", "👍🏽\n👍🏽👍🏽", false, 4, []int{2}, [][][2]int{nil, nil}},
		{"raw", "👍🏽,👍🏽", true, 5, nil, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			v := FormatBytes([]byte(test.s), nil, 0, false, test.raw, ',', '"')
			if v.Width != test.width {
				t.Errorf("expected width %d, got: %d", test.width, v.Width)
			}
			var lines []int
			for _, nl := range v.Newlines {
				lines = append(lines, nl[1])
			}
			if !reflect.DeepEqual(lines, test.lines) {
				t.Errorf("expected line widths %v, got: %v", test.lines, lines)
			}
			if test.tabs == nil {
				test.tabs = make([][][2]int, 1)
			}
			if !reflect.DeepEqual(v.Tabs, test.tabs) {
				t.Errorf("expected tabs %v, got: %v", test.tabs, v.Tabs)
			}
		})
	}
}

// TestEncodeClusters makes sure that an aligned table pads a column of
// grapheme clusters to the same width on each line.
// TestEncodeFormatChars makes sure that a table escapes only a control
// character, as psql does. A joiner, a tag, a private use rune, a soft hyphen,
// a zero width space and a direction control are written as they are.
//
// Measured against psql 18.6. In the output, <U+200C> and the like stand for
// the character itself, which psql writes as it is:
//
//	$ psql -X -q -P border=2 -c "select v from (values
//	    (1, U&'\+01F3F4\+0E0067\+0E0062\+0E0065\+0E006E\+0E0067\+0E007F'),
//	    (2, U&'\0645\06CC\200C\062E\0648\0627\0647\0645'),
//	    (3, U&'\E0A0 main'),
//	    (4, U&'co\00ADop'),
//	    (5, U&'a\200Bb'),
//	    (6, U&'a\202Eb'),
//	    (7, U&'a\0085b')
//	  ) as t(n, v) order by n;"
//	+----------+
//	|    v     |
//	+----------+
//	| 🏴<U+E0067><U+E0062><U+E0065><U+E006E><U+E0067><U+E007F>       |
//	| می<U+200C>خواهم  |
//	| <U+E0A0> main   |
//	| co<U+00AD>op     |
//	| a<U+200B>b       |
//	| a<U+202E>b       |
//	| a\u0085b |
//	+----------+
//	(7 rows)
//
// Note: psql writes the row of U+0085 as the text \u0085.
func TestEncodeFormatChars(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	rs := internal.New([]string{"v"}, [][]any{
		{"\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f"},
		{"\u0645\u06cc\u200c\u062e\u0648\u0627\u0647\u0645"},
		{"\ue0a0 main"},
		{"co\u00adop"},
		{"a\u200bb"},
		{"a\u202eb"},
		{"a\u0085b"},
	})
	if err := EncodeAll(buf, rs, map[string]string{"format": "aligned", "border": "2"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	exp := "+----------+\n" +
		"|    v     |\n" +
		"+----------+\n" +
		"| \U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f       |\n" +
		"| \u0645\u06cc\u200c\u062e\u0648\u0627\u0647\u0645  |\n" +
		"| \ue0a0 main   |\n" +
		"| co\u00adop     |\n" +
		"| a\u200bb       |\n" +
		"| a\u202eb       |\n" +
		"| a\\u0085b |\n" +
		"+----------+\n" +
		"(7 rows)\n"
	if s := buf.String(); s != exp {
		t.Errorf("expected:\n%q\ngot:\n%q", exp, s)
	}
}

func TestEncodeClusters(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	rs := internal.New([]string{"e", "n"}, [][]any{{"👍🏽", 1}, {"ab", 2}})
	if err := EncodeAll(buf, rs, map[string]string{"format": "aligned", "border": "2"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	exp := "+----+---+\n" +
		"| e  | n |\n" +
		"+----+---+\n" +
		"| 👍🏽 | 1 |\n" +
		"| ab | 2 |\n" +
		"+----+---+\n" +
		"(2 rows)\n"
	if s := buf.String(); s != exp {
		t.Errorf("expected:\n%s\ngot:\n%s", exp, s)
	}
}
