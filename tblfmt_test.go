package tblfmt

import (
	"bytes"
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
╔══════════════════════╦═══════════════════════════╦════════════════════════════╗
║      author_id       ║           name            ║             z              ║
╠══════════════════════╬═══════════════════════════╬════════════════════════════╣
║                   16 ║ foo\bbar                  ║                            ║
║                   17 ║ 袈	袈		袈 ║                            ║
║                   18 ║ a	b	\r        ↵║ a                         ↵║
║                      ║ 	a                  ║                            ║
║                   19 ║ 袈	袈		袈↵║                            ║
║                      ║                           ║                            ║
║                   20 ║ javascript                ║ {                         ↵║
║                      ║                           ║   "test21": "a value",    ↵║
║                      ║                           ║   "test22": "foo\u0008bar"↵║
║                      ║                           ║ }                          ║
║                   23 ║ slice                     ║ [                         ↵║
║                      ║                           ║   "a",                    ↵║
║                      ║                           ║   "b"                     ↵║
║                      ║                           ║ ]                          ║
╚══════════════════════╩═══════════════════════════╩════════════════════════════╝
(6 rows)

                                    my table
╔══════════════════════╦═══════════════════════════╦════════════════════════════╗
║      author_id       ║           name            ║             z              ║
╠══════════════════════╬═══════════════════════════╬════════════════════════════╣
║                   38 ║ a	b	c	d  ║ x                          ║
║                   39 ║ aoeu                     ↵║                            ║
║                      ║ test                     ↵║                            ║
║                      ║                           ║                            ║
║                   40 ║ foo\bbar                  ║                            ║
║                   41 ║ 袈	袈		袈 ║                            ║
║                   42 ║ a	b	\r        ↵║ a                         ↵║
║                      ║ 	a                  ║                            ║
║                   43 ║ 袈	袈		袈↵║                            ║
║                      ║                           ║                            ║
║                   44 ║ javascript                ║ {                         ↵║
║                      ║                           ║   "test45": "a value",    ↵║
║                      ║                           ║   "test46": "foo\u0008bar"↵║
║                      ║                           ║ }                          ║
║                   47 ║ slice                     ║ [                         ↵║
║                      ║                           ║   "a",                    ↵║
║                      ║                           ║   "b"                     ↵║
║                      ║                           ║ ]                          ║
╚══════════════════════╩═══════════════════════════╩════════════════════════════╝
(8 rows)
`
	buf := new(bytes.Buffer)
	if err := EncodeTableAll(buf, internal.NewRsetMulti(), WithBorder(2), WithLineStyle(UnicodeDoubleLineStyle()), WithTitle("my table"), WithWidths(20, 20)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestEncodeJSONAll(t *testing.T) {
	t.Parallel()
	exp := `[{"author_id":14,"name":"a\tb\tc\td","z":"x"},{"author_id":15,"name":"aoeu\ntest\n","z":null}],
[{"author_id":16,"name":"foo\bbar","z":null},{"author_id":17,"name":"袈\t袈\t\t袈","z":null},{"author_id":18,"name":"a\tb\t\r\n\ta","z":"a\n"},{"author_id":19,"name":"袈\t袈\t\t袈\n","z":null},{"author_id":20,"name":"javascript","z":{
  "test21": "a value",
  "test22": "foo\u0008bar"
}},{"author_id":23,"name":"slice","z":[
  "a",
  "b"
]}],
[{"author_id":38,"name":"a\tb\tc\td","z":"x"},{"author_id":39,"name":"aoeu\ntest\n","z":null},{"author_id":40,"name":"foo\bbar","z":null},{"author_id":41,"name":"袈\t袈\t\t袈","z":null},{"author_id":42,"name":"a\tb\t\r\n\ta","z":"a\n"},{"author_id":43,"name":"袈\t袈\t\t袈\n","z":null},{"author_id":44,"name":"javascript","z":{
  "test45": "a value",
  "test46": "foo\u0008bar"
}},{"author_id":47,"name":"slice","z":[
  "a",
  "b"
]}]
`
	buf := new(bytes.Buffer)
	if err := EncodeJSONAll(buf, internal.NewRsetMulti()); err != nil {
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
  "test22": "foo\u0008bar"
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
  "test46": "foo\u0008bar"
}"
Row 7:
  author_id = "47"
  name = "slice"
  z = "[
  "a",
  "b"
]"
`
	tpl := `{{ $headers := .Headers }}{{ range $i, $r := .Rows }}Row {{ $i }}:{{ range $j, $c := $r }}
  {{ index $headers $j }} = "{{ $c }}"{{ end }}
{{ end }}`
	buf := new(bytes.Buffer)
	if err := EncodeTemplateAll(buf, internal.NewRsetMulti(), WithRawTemplate(tpl, "text"), WithEmpty("<nil>")); err != nil {
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

author_id|name|z
16|foo` + "\b" + `bar|
17|袈	袈		袈|
18|a	b	` + "\r" + `
	a|a

19|袈	袈		袈
|
20|javascript|{
  "test21": "a value",
  "test22": "foo\u0008bar"
}
23|slice|[
  "a",
  "b"
]

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
  "test46": "foo\u0008bar"
}
47|slice|[
  "a",
  "b"
]
`
	buf := new(bytes.Buffer)
	if err := EncodeUnalignedAll(buf, internal.NewRsetMulti()); err != nil {
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
  ""test22"": ""foo\u0008bar""
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
  ""test46"": ""foo\u0008bar""
}"
47,slice,"[
  ""a"",
  ""b""
]"
`
	buf := new(bytes.Buffer)
	if err := EncodeCSVAll(buf, internal.NewRsetMulti()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}
