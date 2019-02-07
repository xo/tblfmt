package tblfmt

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var newlineRE = regexp.MustCompile(`(?ms)^`)

func TestEncodeFormats(t *testing.T) {
	for _, n := range []string{
		"unaligned",
		"aligned,border 0",
		"aligned,border 1",
		"aligned,border 2",
		"aligned,border 0,title 'test title'",
		"aligned,border 1,title 'test title'",
		"aligned,border 2,title 'test title'",
		//"wrapped",
		//"html",
		//"asciidoc",
		//"latex",
		//"latex-longtable",
		//"troff-ms",
		"json",
		"csv",
	} {
		// build params
		params := map[string]string{
			"format": n,
		}
		if i := strings.Index(n, ","); i != -1 {
			for _, p := range strings.Split(n, ",")[1:] {
				v := strings.SplitN(p, " ", 2)
				params[v[0]] = v[1]
			}
			params["format"] = n[:i]
		}

		if n != "json" && n != "csv" {
			t.Run("psql-"+n, func(t *testing.T) {
				t.Parallel()
				buf := new(bytes.Buffer)
				if err := psqlEncodeAll(buf, rs(), params); err != nil {
					if err == errPsqlConnNotDefined {
						t.Skipf("PSQL_CONN not defined, skipping psql query")
					}
					t.Fatalf("unable to run psql, got: %v", err)
				}
				t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
			})
		}

		t.Run(n, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, rs(), params); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", n, err)
			}
			t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
		})
	}
}

func TestBigAligned(t *testing.T) {
	resultSet := rsbig()
	buf := new(bytes.Buffer)
	if err := EncodeTableAll(buf, resultSet); err != nil {
		t.Fatalf("expected no error when encoding, got: %v", err)
	}
	t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
}

func BenchmarkEncodeFormats(b *testing.B) {
	encoders := []struct {
		name string
		f    Builder
		opts []Option
	}{
		{"aligned", NewTableEncoder, nil},
		{"json", NewJSONEncoder, nil},
		{"csv", NewCSVEncoder, nil},
	}

	for _, enc := range encoders {
		b.Run(enc.name, func(b *testing.B) {
			resultSet, w := rsbig(), &noopWriter{}
			for i := 0; i < b.N; i++ {
				enc, err := enc.f(resultSet, enc.opts...)
				if err != nil {
					b.Fatalf("expected no error, got: %v", err)
				}

				if err = enc.Encode(w); err != nil {
					b.Errorf("expected no error, got: %v", err)
				}
				resultSet.Reset()
			}
		})
	}
}
