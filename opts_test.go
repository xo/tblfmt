package tblfmt

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

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

func TestFromMapFormats(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{
		"big",
		"multi",
		"tiny",
		"wide",
	} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			tests := loadTests(t, typ)
			for _, test := range tests {
				t.Run(test.idstr, func(t *testing.T) {
					f, opts := FromMap(test.opts)
					var rs ResultSet
					switch typ {
					case "big":
						rs = internal.Big(1549508725559526476)
					case "multi":
						rs = internal.Multi()
					case "tiny":
						rs = internal.Tiny()
					case "wide":
						rs = internal.Wide()
					}
					enc, err := f(rs, opts...)
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
		id, err := strconv.ParseInt(strings.TrimPrefix(n, base+"/"), 10, 64)
		if err != nil {
			t.Fatalf("unable to parse %q: %v", name, err)
		}
		opts := loadOpts(t, name)
		gld := n + ".gld"
		exp, err := os.ReadFile(gld)
		if err != nil {
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
