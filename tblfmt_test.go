package tblfmt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

var newlineRE = regexp.MustCompile(`(?ms)^`)

func TestEncodeFormats(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]string
	}{
		{
			name: "unaligned",
			params: map[string]string{
				"format": "unaligned",
				"title":  "won't print",
				// TODO psql does print the footer
				"footer": "off",
			},
		},
		{
			name: "aligned, border 0",
			params: map[string]string{
				"format": "aligned",
				"border": "0",
			},
		},
		{
			name: "aligned, border 1",
			params: map[string]string{
				"format": "aligned",
				"border": "1",
			},
		},
		{
			name: "aligned, border 2",
			params: map[string]string{
				"format": "aligned",
				"border": "2",
			},
		},
		{
			name: "aligned, border 0, title 'test title'",
			params: map[string]string{
				"format": "aligned",
				"border": "0",
				"title":  "test title",
			},
		},
		{
			name: "aligned, border 1, title 'test title'",
			params: map[string]string{
				"format": "aligned",
				"border": "1",
				"title":  "test title",
			},
		},
		{
			name: "aligned, border 2, title 'test title'",
			params: map[string]string{
				"format": "aligned",
				"border": "2",
				"title":  "test title",
			},
		},
		{
			name: "aligned, footer off",
			params: map[string]string{
				"format": "aligned",
				"footer": "off",
			},
		},
		{
			name: "aligned, border 0, expanded on",
			params: map[string]string{
				"format":   "aligned",
				"border":   "0",
				"expanded": "on",
			},
		},
		{
			name: "aligned, border 1, expanded on",
			params: map[string]string{
				"format":   "aligned",
				"border":   "1",
				"expanded": "on",
			},
		},
		{
			name: "aligned, border 2, expanded on",
			params: map[string]string{
				"format":   "aligned",
				"border":   "2",
				"expanded": "on",
			},
		},
		{
			name: "aligned, border 2, expanded on, title 'test title'",
			params: map[string]string{
				"format":   "aligned",
				"border":   "2",
				"expanded": "on",
				"title":    "test title",
			},
		},
		//"wrapped",
		//"html",
		//"asciidoc",
		//"latex",
		//"latex-longtable",
		//"troff-ms",
	}

	expected := "testdata/formats.expected.txt"

	var fe *os.File
	var err error
	dsn := os.Getenv("PSQL_CONN")
	if len(dsn) != 0 {
		fe, err = os.Create(expected)
		if err != nil {
			t.Fatalf("Cannot create expected file %s: %v", expected, err)
		}
	}

	actual := "testdata/formats.actual.txt"
	fa, err := os.Create(actual)
	if err != nil {
		t.Fatalf("Cannot create results file %s: %v", actual, err)
	}

	for _, c := range cases {
		t.Run("psql-"+c.name, func(t *testing.T) {
			//t.Parallel()
			buf := new(bytes.Buffer)
			if err := psqlEncodeAll(buf, rs(), c.params); err != nil {
				if err == errPsqlConnNotDefined {
					t.Skipf("PSQL_CONN not defined, skipping psql query")
				}
				t.Fatalf("unable to run psql, got: %v", err)
			}
			res := buf.String()
			t.Log("\n", newlineRE.ReplaceAllString(res, "\t"))
			if fe != nil {
				fmt.Fprintln(fe, c.name)
				fmt.Fprintln(fe, res)
			}
		})

		t.Run(c.name, func(t *testing.T) {
			//t.Parallel()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, rs(), c.params); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", c.name, err)
			}
			res := buf.String()
			t.Log("\n", newlineRE.ReplaceAllString(res, "\t"))
			fmt.Fprintln(fa, c.name)
			fmt.Fprintln(fa, res)
		})
	}

	if fe != nil {
		fe.Close()
	}
	fa.Close()

	err = filesEqual(expected, actual)
	if err != nil {
		t.Error(err)
	}
}

func TestEncodeExportFormats(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]string
	}{
		{
			name: "json",
			params: map[string]string{
				"format": "json",
			},
		},
		{
			name: "csv",
			params: map[string]string{
				"format": "csv",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, rs(), c.params); err != nil {
				t.Fatalf("expected no error when encoding format %q, got: %v", c.name, err)
			}
			t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
		})
	}
}

func TestTinyAligned(t *testing.T) {
	resultSet := rstiny()

	expected := "testdata/tiny.expected.txt"
	actual := "testdata/tiny.actual.txt"
	fa, err := os.Create(actual)
	if err != nil {
		t.Fatalf("Cannot create results file %s: %v", actual, err)
	}

	buf := new(bytes.Buffer)
	params := map[string]string{
		"format":   "aligned",
		"expanded": "on",
		"border":   "2",
	}
	if err := EncodeAll(buf, resultSet, params); err != nil {
		t.Fatalf("expected no error when encoding, got: %v", err)
	}
	res := buf.String()
	t.Log("\n", newlineRE.ReplaceAllString(res, "\t"))
	fmt.Fprintln(fa, res)
	fa.Close()

	err = filesEqual(expected, actual)
	if err != nil {
		t.Error(err)
	}
}

func TestWideExpanded(t *testing.T) {
	resultSet := rswide()
	buf := new(bytes.Buffer)
	params := map[string]string{
		"format":   "aligned",
		"expanded": "auto",
		"border":   "2",
	}
	if err := EncodeAll(buf, resultSet, params); err != nil {
		t.Fatalf("expected no error when encoding, got: %v", err)
	}
	t.Log("\n", newlineRE.ReplaceAllString(buf.String(), "\t"))
}

func TestBigAligned(t *testing.T) {
	resultSet := rsbig()

	expected := "testdata/big.expected.txt"
	actual := "testdata/big.actual.txt"
	fa, err := os.Create(actual)
	if err != nil {
		t.Fatalf("Cannot create results file %s: %v", actual, err)
	}

	buf := new(bytes.Buffer)
	if err := EncodeTableAll(buf, resultSet); err != nil {
		t.Fatalf("expected no error when encoding, got: %v", err)
	}
	res := buf.String()
	t.Log("\n", newlineRE.ReplaceAllString(res, "\t"))
	fmt.Fprintln(fa, res)
	fa.Close()

	err = filesEqual(expected, actual)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkEncodeFormats(b *testing.B) {
	encoders := []struct {
		name string
		f    Builder
		opts []Option
	}{
		{"aligned", NewTableEncoder, nil},
		{"aligned-batch10", NewTableEncoder, []Option{WithCount(10)}},
		{"aligned-batch100", NewTableEncoder, []Option{WithCount(100)}},
		{"json", NewJSONEncoder, nil},
		{"csv", NewCSVEncoder, nil},
		{"template-asciidoc", NewTemplateEncoder, []Option{WithNamedTemplate("asciidoc")}},
		{"template-html", NewTemplateEncoder, []Option{WithNamedTemplate("html")}},
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

func filesEqual(a, b string) error {
	// per comment, better to not read an entire file into memory
	// this is simply a trivial example.
	f1, err := ioutil.ReadFile(a)
	if err != nil {
		return fmt.Errorf("Cannot read file %s: %w", a, err)
	}

	f2, err := ioutil.ReadFile(b)
	if err != nil {
		return fmt.Errorf("Cannot read file %s: %w", b, err)
	}

	if !bytes.Equal(f1, f2) {
		return fmt.Errorf("Files %s and %s have different contents", a, b)
	}
	return nil
}
