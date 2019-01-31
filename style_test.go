package tblfmt

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var newlineRE = regexp.MustCompile(`(?ms)^`)

func TestFormats(t *testing.T) {
	for _, n := range []string{
		"unaligned",
		"aligned,border 0",
		"aligned,border 1",
		"aligned,border 2",
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
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, rs(), params); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", n, err)
			}
			t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
		})
	}
}
