package tblfmt

import (
	"bytes"
	"regexp"
	"testing"
)

var newlineRE = regexp.MustCompile(`(?ms)^`)

func TestFormats(t *testing.T) {
	for _, n := range []string{
		"unaligned",
		"aligned",
		//"wrapped",
		//"html",
		//"asciidoc",
		//"latex",
		//"latex-longtable",
		//"troff-ms",
		"json",
		"csv",
	} {
		if n != "json" && n != "csv" {
			t.Run("psql-"+n, func(t *testing.T) {
				buf := new(bytes.Buffer)
				if err := psqlEncodeAll(buf, rs(), n); err != nil {
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
			if err := EncodeAll(buf, rs(), map[string]string{
				"format": n,
			}); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", n, err)
			}
			t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
		})
	}
}
