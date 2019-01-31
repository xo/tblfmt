package tblfmt

import (
	"bytes"
	"log"
	"testing"
)

func TestFormats(t *testing.T) {
	for _, n := range []string{
		"psql",
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
		buf := new(bytes.Buffer)

		if n == "psql" {
			if err := psqlEncodeAll(buf, rs()); err != nil {
				if err == errPsqlConnNotDefined {
					t.Logf("PSQL_CONN not defined, skipping psql query")
					continue
				}
				t.Fatalf("unable to run psql, got: %v", err)
			}
		} else {
			if err := EncodeAll(buf, rs(), map[string]string{
				"format": n,
			}); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", n, err)
			}
		}

		log.Printf("format %q:\n%s", n, buf.String())
	}
}
