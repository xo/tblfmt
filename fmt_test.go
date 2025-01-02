package tblfmt

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	runewidth "github.com/mattn/go-runewidth"
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
		t.Errorf("expected: %q, got: %q", string(exp), string(b))
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
	s = strings.Replace(s, `\t`, "\t", -1)
	s = strings.Replace(s, `\n`, "\n", -1)
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
