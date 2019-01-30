package tblfmt

import (
	"bytes"
	"log"
	"testing"
)

func TestFormats(t *testing.T) {
	for _, n := range []map[string]string{
		map[string]string{"format": "unaligned"},
		//"wrapped",
		//"html",
		//"asciidoc",
		//"latex",
		//"latex-longtable",
		//"troff-ms",
		map[string]string{"format": "json"},
		map[string]string{"format": "csv"},
		map[string]string{"format": "aligned"},
		map[string]string{"format": "aligned", "border": "2",
			"linestyle": "unicode", "unicode_border_linestyle": "single"},
		map[string]string{"format": "aligned", "border": "2",
			"linestyle": "unicode-compact"},
		map[string]string{"format": "aligned", "border": "1",
			"linestyle": "unicode-compact"},
		map[string]string{"format": "aligned", "border": "1",
			"linestyle": "unicode", "unicode_border_linestyle": "double"},
		map[string]string{"format": "aligned", "border": "2",
			"linestyle": "unicode-inline"},
		map[string]string{"format": "aligned", "border": "1",
			"linestyle": "unicode-inline"},
	} {
		buf := new(bytes.Buffer)
		if err := EncodeAll(buf, rs(), n); err != nil {
			t.Fatalf("expected no error when encoding format %q, got: %v", n, err)
		}
		log.Printf("format %q:\n%s", n, buf.String())
	}
}
