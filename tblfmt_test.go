package tblfmt

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/xo/tblfmt/internal"
)

func TestEncodeTableAll(t *testing.T) {
	t.Parallel()
	exp := `                       my table
╔══════════════════════╦══════════════════════════╦═══╗
║      author_id       ║           name           ║ z ║
╠══════════════════════╬══════════════════════════╬═══╣
║                   14 ║ a	b	c	d ║ x ║
║                   15 ║ aoeu                    ↵║   ║
║                      ║ test                    ↵║   ║
║                      ║                          ║   ║
╚══════════════════════╩══════════════════════════╩═══╝
(2 rows)

                                  my table
╔══════════════════════╦═══════════════════════════╦════════════════════════╗
║      author_id       ║           name            ║           z            ║
╠══════════════════════╬═══════════════════════════╬════════════════════════╣
║                   16 ║ foo\bbar                  ║                        ║
║                   17 ║ 袈	袈		袈 ║                        ║
║                   18 ║ a	b	\r        ↵║ a                     ↵║
║                      ║ 	a                  ║                        ║
║                   19 ║ 袈	袈		袈↵║                        ║
║                      ║                           ║                        ║
║                   20 ║ javascript                ║ {                     ↵║
║                      ║                           ║   "test21": "a value",↵║
║                      ║                           ║   "test22": "foo\bbar"↵║
║                      ║                           ║ }                      ║
║                   23 ║ slice                     ║ [                     ↵║
║                      ║                           ║   "a",                ↵║
║                      ║                           ║   "b"                 ↵║
║                      ║                           ║ ]                      ║
╚══════════════════════╩═══════════════════════════╩════════════════════════╝
(6 rows)

                                  my table
╔══════════════════════╦═══════════════════════════╦════════════════════════╗
║      author_id       ║           name            ║           z            ║
╠══════════════════════╬═══════════════════════════╬════════════════════════╣
║                   38 ║ a	b	c	d  ║ x                      ║
║                   39 ║ aoeu                     ↵║                        ║
║                      ║ test                     ↵║                        ║
║                      ║                           ║                        ║
║                   40 ║ foo\bbar                  ║                        ║
║                   41 ║ 袈	袈		袈 ║                        ║
║                   42 ║ a	b	\r        ↵║ a                     ↵║
║                      ║ 	a                  ║                        ║
║                   43 ║ 袈	袈		袈↵║                        ║
║                      ║                           ║                        ║
║                   44 ║ javascript                ║ {                     ↵║
║                      ║                           ║   "test45": "a value",↵║
║                      ║                           ║   "test46": "foo\bbar"↵║
║                      ║                           ║ }                      ║
║                   47 ║ slice                     ║ [                     ↵║
║                      ║                           ║   "a",                ↵║
║                      ║                           ║   "b"                 ↵║
║                      ║                           ║ ]                      ║
╚══════════════════════╩═══════════════════════════╩════════════════════════╝
(8 rows)
`
	buf := new(bytes.Buffer)
	if err := EncodeTableAll(buf, internal.Multi(), WithBorder(2), WithLineStyle(UnicodeDoubleLineStyle()), WithTitle("my table"), WithWidths(20, 20)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestEncodeJSONAll(t *testing.T) {
	t.Parallel()
	exp := `[
  {
    "author_id": 14,
    "name": "a\tb\tc\td",
    "z": "x"
  },
  {
    "author_id": 15,
    "name": "aoeu\ntest\n",
    "z": null
  }
],
[
  {
    "author_id": 16,
    "name": "foo\bbar",
    "z": null
  },
  {
    "author_id": 17,
    "name": "袈\t袈\t\t袈",
    "z": null
  },
  {
    "author_id": 18,
    "name": "a\tb\t\r\n\ta",
    "z": "a\n"
  },
  {
    "author_id": 19,
    "name": "袈\t袈\t\t袈\n",
    "z": null
  },
  {
    "author_id": 20,
    "name": "javascript",
    "z": {
      "test21": "a value",
      "test22": "foo\bbar"
    }
  },
  {
    "author_id": 23,
    "name": "slice",
    "z": [
      "a",
      "b"
    ]
  }
],
[
  {
    "author_id": 38,
    "name": "a\tb\tc\td",
    "z": "x"
  },
  {
    "author_id": 39,
    "name": "aoeu\ntest\n",
    "z": null
  },
  {
    "author_id": 40,
    "name": "foo\bbar",
    "z": null
  },
  {
    "author_id": 41,
    "name": "袈\t袈\t\t袈",
    "z": null
  },
  {
    "author_id": 42,
    "name": "a\tb\t\r\n\ta",
    "z": "a\n"
  },
  {
    "author_id": 43,
    "name": "袈\t袈\t\t袈\n",
    "z": null
  },
  {
    "author_id": 44,
    "name": "javascript",
    "z": {
      "test45": "a value",
      "test46": "foo\bbar"
    }
  },
  {
    "author_id": 47,
    "name": "slice",
    "z": [
      "a",
      "b"
    ]
  }
]
`
	buf := new(bytes.Buffer)
	if err := EncodeJSONAll(buf, internal.Multi()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestEncodeTemplateAll(t *testing.T) {
	t.Parallel()
	exp := `Row 0:
  author_id = "14"
  name = "a	b	c	d"
  z = "x"
Row 1:
  author_id = "15"
  name = "aoeu
test
"
  z = "<nil>"

Row 0:
  author_id = "16"
  name = "foo\bbar"
  z = "<nil>"
Row 1:
  author_id = "17"
  name = "袈	袈		袈"
  z = "<nil>"
Row 2:
  author_id = "18"
  name = "a	b	\r
	a"
  z = "a
"
Row 3:
  author_id = "19"
  name = "袈	袈		袈
"
  z = "<nil>"
Row 4:
  author_id = "20"
  name = "javascript"
  z = "{
  "test21": "a value",
  "test22": "foo\bbar"
}"
Row 5:
  author_id = "23"
  name = "slice"
  z = "[
  "a",
  "b"
]"

Row 0:
  author_id = "38"
  name = "a	b	c	d"
  z = "x"
Row 1:
  author_id = "39"
  name = "aoeu
test
"
  z = "<nil>"
Row 2:
  author_id = "40"
  name = "foo\bbar"
  z = "<nil>"
Row 3:
  author_id = "41"
  name = "袈	袈		袈"
  z = "<nil>"
Row 4:
  author_id = "42"
  name = "a	b	\r
	a"
  z = "a
"
Row 5:
  author_id = "43"
  name = "袈	袈		袈
"
  z = "<nil>"
Row 6:
  author_id = "44"
  name = "javascript"
  z = "{
  "test45": "a value",
  "test46": "foo\bbar"
}"
Row 7:
  author_id = "47"
  name = "slice"
  z = "[
  "a",
  "b"
]"
`
	buf := new(bytes.Buffer)
	if err := EncodeTemplateAll(buf, internal.Multi(), WithExecutor(testWriteTemplateTo), WithEmpty("<nil>")); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestEncodeUnalignedAll(t *testing.T) {
	t.Parallel()
	exp := `author_id|name|z
14|a	b	c	d|x
15|aoeu
test
|
(2 rows)

author_id|name|z
16|foo` + "\b" + `bar|
17|袈	袈		袈|
18|a	b	` + "\r" + `
	a|a

19|袈	袈		袈
|
20|javascript|{
  "test21": "a value",
  "test22": "foo\bbar"
}
23|slice|[
  "a",
  "b"
]
(6 rows)

author_id|name|z
38|a	b	c	d|x
39|aoeu
test
|
40|foo` + "\b" + `bar|
41|袈	袈		袈|
42|a	b	` + "\r" + `
	a|a

43|袈	袈		袈
|
44|javascript|{
  "test45": "a value",
  "test46": "foo\bbar"
}
47|slice|[
  "a",
  "b"
]
(8 rows)
`
	buf := new(bytes.Buffer)
	if err := EncodeUnalignedAll(buf, internal.Multi()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestEncodeCSVAll(t *testing.T) {
	t.Parallel()
	exp := `author_id,name,z
14,"a	b	c	d",x
15,"aoeu
test
",

author_id,name,z
16,foo` + "\b" + `bar,
17,"袈	袈		袈",
18,"a	b	` + "\r" + `
	a","a
"
19,"袈	袈		袈
",
20,javascript,"{
  ""test21"": ""a value"",
  ""test22"": ""foo\bbar""
}"
23,slice,"[
  ""a"",
  ""b""
]"

author_id,name,z
38,"a	b	c	d",x
39,"aoeu
test
",
40,foo` + "\b" + `bar,
41,"袈	袈		袈",
42,"a	b	` + "\r" + `
	a","a
"
43,"袈	袈		袈
",
44,javascript,"{
  ""test45"": ""a value"",
  ""test46"": ""foo\bbar""
}"
47,slice,"[
  ""a"",
  ""b""
]"
`
	buf := new(bytes.Buffer)
	if err := EncodeCSVAll(buf, internal.Multi()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func testWriteTemplateTo(w io.Writer, tpl *Template) error {
	// {{ $headers := .Headers }}{{ range $i, $r := .Rows }}Row {{ $i }}:{{ range $j, $c := $r }}
	//   {{ index $headers $j }} = "{{ $c }}"{{ end }}
	// {{ end }}
	for i, r := range tpl.Rows {
		fmt.Fprintf(w, "Row %d:\n", i)
		for j, c := range r {
			fmt.Fprintf(w, "  %s = \"%s\"\n", tpl.Headers[j], c)
		}
	}
	return nil
}

// TestEncodeNullAlign checks that a null value follows the alignment of its
// column, the way psql aligns the null string by the column's type, and does
// not always sit to the left.
//
// Measured against psql 18.6:
//
//	$ psql -P null='(null)' -c "select 12345678901234567890::numeric as n,
//	                                   'txt'::text as t
//	                            union all select null, null;"
//	          n           |   t
//	----------------------+--------
//	 12345678901234567890 | txt
//	               (null) | (null)
func TestEncodeNullAlign(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		vals [][]any
		exp  string
		html string
	}{
		// note: the values are wider than the null string on purpose. A null
		// string exactly as wide as its column fills it, and then the two
		// alignments are indistinguishable and the test proves nothing.
		{
			"right aligned column",
			[][]any{{uint64(12345678901234567890)}, {nil}},
			"               (null)",
			`align="right"`,
		},
		{
			"left aligned column",
			[][]any{{"aaaaaaaaaaaaaaaaaaaa"}, {nil}},
			" (null)",
			`align="left"`,
		},
		{
			// a column of more than one alignment has none to pass on
			"mixed alignment column",
			[][]any{{uint64(12345678901234567890)}, {"aaaaaaaaaaaaaaaaaaaa"}, {nil}},
			" (null)",
			`align="left"`,
		},
		{
			"all null column",
			[][]any{{nil}, {nil}},
			" (null)",
			`align="left"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, internal.New([]string{"n"}, test.vals), map[string]string{
				"format": "aligned",
				"null":   "(null)",
			}); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			t.Logf("aligned:\n%s", buf.String())
			// the null row is the last row before the summary
			lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
			if len(lines) < 3 {
				t.Fatalf("expected at least 3 lines, got: %d", len(lines))
			}
			if s := strings.TrimRight(lines[len(lines)-2], " "); s != test.exp {
				t.Errorf("expected null row %q, got: %q", test.exp, s)
			}
			buf.Reset()
			if err := EncodeAll(buf, internal.New([]string{"n"}, test.vals), map[string]string{
				"format": "html",
				"null":   "(null)",
			}); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			t.Logf("html:\n%s", buf.String())
			if s := test.html + ">(null)</td>"; !strings.Contains(buf.String(), s) {
				t.Errorf("expected html to contain %q", s)
			}
		})
	}
}
