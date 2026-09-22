package tblfmt

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
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
			"👀",
			"🤰",
			"foo",
			"15\u00f8C",
		},
		";",
	)
	exp, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	exp = exp[1 : len(exp)-1]
	t.Logf("exp: %q", string(exp))
	v := FormatBytes([]byte(s), nil, 0, true, false, 0, 0)
	t.Logf("v  : %q", v)
	if b := []byte(v.String()); !slices.Equal(b, exp) {
		t.Errorf("\nexpected:\n%q\ngot:\n%q", string(exp), string(b))
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
	const exp = `[{"b":18446744073709551615},{"b":null}]`
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
