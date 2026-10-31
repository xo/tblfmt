package tblfmt

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"

	"github.com/xo/tblfmt/internal"
)

func TestFromMap(t *testing.T) {
	tests := map[string]struct {
		name string
		opts map[string]string
		exp  Builder
	}{
		"empty": {
			exp: newErrEncoder,
		},
		"aligned": {
			opts: map[string]string{
				"format": "aligned",
			},
			exp: NewTableEncoder,
		},
		"unaligned": {
			opts: map[string]string{
				"format": "unaligned",
			},
			exp: NewUnalignedEncoder,
		},
		"csv": {
			opts: map[string]string{
				"format": "csv",
			},
			exp: NewCSVEncoder,
		},
		"any": {
			opts: map[string]string{
				"format":   "aligned",
				"border":   "2",
				"title":    "some title",
				"expanded": "on",
			},
			exp: NewExpandedEncoder,
		},
	}
	for n, test := range tests {
		t.Run(n, func(t *testing.T) {
			builder, _ := FromMap(test.opts)
			if reflect.ValueOf(builder).Pointer() != reflect.ValueOf(test.exp).Pointer() {
				t.Errorf("invalid builder, expected %T, got %T", test.exp, builder)
			}
		})
	}
}

// update makes TestFromMapFormats write each golden file from the output.
var update = flag.Bool("update", false, "update the golden files")

// goldenSets are the result sets of the golden tests, by the name of the
// directory in testdata that holds their files.
var goldenSets = map[string]func() *internal.RS{
	"big":         func() *internal.RS { return internal.Big(internal.BigSeed) },
	"multi":       internal.Multi,
	"tiny":        internal.Tiny,
	"wide":        internal.Wide,
	"big-emoji":   func() *internal.RS { return internal.BigEmoji(internal.BigEmojiSeed) },
	"multi-emoji": internal.MultiEmoji,
	"tiny-emoji":  internal.TinyEmoji,
	"wide-emoji":  internal.WideEmoji,
}

func TestFromMapFormats(t *testing.T) {
	t.Parallel()
	for _, typ := range slices.Sorted(maps.Keys(goldenSets)) {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			tests := loadTests(t, typ)
			for _, test := range tests {
				t.Run(test.idstr, func(t *testing.T) {
					f, opts := FromMap(test.opts)
					enc, err := f(goldenSets[typ](), opts...)
					if err != nil {
						t.Fatalf("expected no error, got: %v", err)
					}
					var buf bytes.Buffer
					if err := enc.EncodeAll(&buf); err != nil {
						t.Fatalf("expected no error, got: %v", err)
					}
					if err := os.WriteFile(test.out, buf.Bytes(), 0o644); err != nil {
						t.Fatalf("expected no error, got: %v", err)
					}
					if *update {
						if err := os.WriteFile(test.gld, buf.Bytes(), 0o644); err != nil {
							t.Fatalf("expected no error, got: %v", err)
						}
						test.exp = buf.Bytes()
					}
					if s, exp := buf.String(), string(test.exp); s != exp {
						t.Errorf("expected:\n%s\ngot:\n%s", exp, s)
					}
				})
			}
		})
	}
}

func TestFromMapTuplesOnly(t *testing.T) {
	tests := []struct {
		format     string
		tuplesOnly string
		exp        bool
	}{
		{"aligned", "on", false},
		{"aligned", "off", true},
		{"unaligned", "on", false},
		{"unaligned", "off", true},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Logf("format: %s tuples_only: %s has footer: %t", test.format, test.tuplesOnly, test.exp)
			rs := internal.New([]string{"id"}, [][]any{{"1"}})
			params := map[string]string{
				"format":      test.format,
				"tuples_only": test.tuplesOnly,
				"border":      "1",
				"linestyle":   "ascii",
				"fieldsep":    "|",
				"footer":      "on",
			}
			var buf bytes.Buffer
			if err := EncodeAll(&buf, rs, params); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			out := buf.String()
			t.Logf("output:\n%s", out)
			if b := strings.Contains(out, "(1 row)"); b != test.exp {
				t.Errorf("expected %t, got: %t", test.exp, b)
			}
		})
	}
}

type goldTest struct {
	id    int
	idstr string
	in    string
	gld   string
	out   string
	opts  map[string]string
	exp   []byte
}

func loadTests(t *testing.T, typ string) []goldTest {
	t.Helper()
	base := "testdata/" + typ
	var tests []goldTest
	err := filepath.Walk(base, func(name string, fi os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case fi.IsDir(), !strings.HasSuffix(name, ".in"):
			return nil
		}
		n := strings.TrimSuffix(name, ".in")
		// note: filepath.Walk returns the separator of the platform, so the
		// number comes from the base name
		id, err := strconv.ParseInt(filepath.Base(n), 10, 64)
		if err != nil {
			t.Fatalf("unable to parse %q: %v", name, err)
		}
		opts := loadOpts(t, name)
		gld := n + ".gld"
		exp, err := os.ReadFile(gld)
		switch {
		case *update && errors.Is(err, fs.ErrNotExist):
			// -update writes the missing golden file
		case err != nil:
			t.Fatalf("unable to open %s: %v", gld, err)
		}
		tests = append(tests, goldTest{
			id:    int(id),
			idstr: strconv.Itoa(int(id)),
			in:    name,
			gld:   gld,
			out:   n + ".out",
			opts:  opts,
			exp:   exp,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return tests
}

func loadOpts(t *testing.T, in string) map[string]string {
	t.Helper()
	opts := make(map[string]string)
	buf, err := os.ReadFile(in)
	if err != nil {
		t.Fatalf("unable to open %q", in)
	}
loop:
	for {
		i := bytes.Index(buf, []byte{'\n'})
		if i == -1 {
			break loop
		}
		line := string(bytes.TrimSpace(buf[:i]))
		if line == "" {
			break loop
		}
		v := strings.Split(line, ":")
		if len(v) != 2 {
			t.Fatalf("missing : in line")
		}
		opts[strings.TrimSpace(v[0])] = strings.TrimSpace(v[1])
		buf = buf[i+1:]
	}
	if opts["format"] == "" {
		t.Fatalf("format was not defined")
	}
	if s := opts["format"]; s == "expanded" {
		opts["format"] = "aligned"
		opts["expanded"] = "on"
	}
	if s := opts["linestyle"]; strings.HasPrefix(s, "unicode") {
		opts["linestyle"] = "unicode"
		opts["unicode_border_linestyle"] = "single"
		if strings.Contains(s, "double") {
			opts["unicode_border_linestyle"] = "double"
		}
	}
	return opts
}

// TestGoldenAlignment makes sure that each line of a table with border 2 in a
// golden file has the same display width. A tab moves to the next multiple of
// 8 columns, as in a terminal.
//
// Note: the golden files hold the output of tblfmt itself, so they cannot show
// that a column is padded wrong. This test can.
func TestGoldenAlignment(t *testing.T) {
	t.Parallel()
	for _, typ := range slices.Sorted(maps.Keys(goldenSets)) {
		for _, test := range loadTests(t, typ) {
			if test.opts["border"] != "2" || (test.opts["format"] != "aligned" && test.opts["format"] != "wrapped") {
				continue
			}
			t.Run(typ+"/"+test.idstr, func(t *testing.T) {
				t.Parallel()
				first := -1
				for i, line := range strings.Split(string(test.exp), "\n") {
					r, _ := utf8.DecodeRuneInString(line)
					if !strings.ContainsRune("+|┌│├└╔║╠╚", r) {
						first = -1
						continue
					}
					w := displayWidth(line)
					switch {
					case first == -1:
						first = w
					case w != first:
						t.Errorf("line %d is %d columns wide, expected %d: %q", i+1, w, first, line)
					}
				}
			})
		}
	}
}

// displayWidth returns the width of the line in a terminal with a tab stop
// every 8 columns. It counts an emoji variation sequence as 2 columns, as a
// current terminal draws it. See B38 in docs/BACKLOG.md.
func displayWidth(line string) int {
	var w int
	for i, s := range strings.Split(line, "\t") {
		if i != 0 {
			w += 8 - w%8
		}
		w += runewidth.StringWidth(s) + emojiWidth([]byte(s))
	}
	return w
}

// TestWithFormatterOptionsError makes sure that WithFormatterOptions returns
// an error, and does not panic, when the formatter is not an EscapeFormatter.
func TestWithFormatterOptionsError(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"aligned", "json", "csv", "html"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			f, opts := FromMap(map[string]string{"format": format})
			opts = append(opts, WithFormatter(otherFormatter{}), WithFormatterOptions(WithTimeFormat(time.Kitchen)))
			if _, err := f(internal.Tiny(), opts...); !errors.Is(err, ErrFormatterIsNotEscapeFormatter) {
				t.Errorf("expected %v, got: %v", ErrFormatterIsNotEscapeFormatter, err)
			}
		})
	}
}

// otherFormatter is a formatter that is not an EscapeFormatter.
type otherFormatter struct{}

// Header satisfies the Formatter interface.
func (otherFormatter) Header([]string) ([]*Value, error) {
	return nil, nil
}

// Format satisfies the Formatter interface.
func (otherFormatter) Format([]any) ([]*Value, error) {
	return nil, nil
}
