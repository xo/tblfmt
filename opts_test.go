package tblfmt

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/xo/tblfmt/internal"
	"github.com/xo/tblfmt/testdata"
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
			exp: NewUnalignedEncoder,
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
	for _, typ := range []string{
		"big",
		"multi",
		"tiny",
		"wide",
	} {
		typ := typ
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			z, err := testdata.Testdata.ReadFile(typ + ".gz")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			r, err := gzip.NewReader(bytes.NewReader(z))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			buf, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			var i int
		loop:
			for {
				var resultSet ResultSet
				switch typ {
				case "big":
					resultSet = internal.NewRsetBig(1549508725559526476)
				case "multi":
					resultSet = internal.NewRsetMulti()
				case "tiny":
					resultSet = internal.NewRsetTiny()
				case "wide":
					resultSet = internal.NewRsetWide()
				}
				var optMap map[string]string
				var exp []byte
				buf, optMap, exp, err = readFromOpts(buf)
				switch {
				case err != nil && err == io.EOF:
					break loop
				case err != nil:
					t.Fatalf("test %s (%d) expected no error, got: %v", typ, i, err)
				}
				f, opts := FromMap(optMap)
				enc, err := f(resultSet, opts...)
				if err != nil {
					t.Fatalf("test %s (%d) expected no error, got: %v", typ, i, err)
				}
				actual := new(bytes.Buffer)
				if err := enc.EncodeAll(actual); err != nil {
					t.Fatalf("test %s (%d) expected no error, got: %v", typ, i, err)
				}
				if !bytes.Equal(actual.Bytes(), exp) {
					t.Errorf("test %s (%d) actual != exp", typ, i)
				}
				i++
			}
		})
	}
}

func readFromOpts(buf []byte) ([]byte, map[string]string, []byte, error) {
	divider := append([]byte(internal.Divider), newline...)
	if len(buf) == 0 {
		return nil, nil, nil, io.EOF
	}
	if !bytes.HasPrefix(buf, divider) {
		return nil, nil, nil, errors.New("could not find option start divider")
	}
	buf = buf[len(divider):]
	end := bytes.Index(buf, divider)
	if end == -1 {
		return nil, nil, nil, errors.New("could not find option middle divider")
	}
	optMap, err := parseOpts(buf[:end])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to parse opts: %v", err)
	}
	buf = buf[end+len(divider):]
	if end = bytes.Index(buf, divider); end == -1 {
		end = len(buf)
	}
	return buf[end:], optMap, buf[:end], nil
}

func parseOpts(buf []byte) (map[string]string, error) {
	opts := make(map[string]string)
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
			return nil, errors.New("missing : in line")
		}
		opts[strings.TrimSpace(v[0])] = strings.TrimSpace(v[1])
		buf = buf[i+1:]
	}
	if opts["format"] == "" {
		return nil, errors.New("format was not defined")
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
	return opts, nil
}
