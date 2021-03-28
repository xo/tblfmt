package tblfmt

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestJSONEncoder(t *testing.T) {
	resultSet := rs()
	var i int
	for resultSet.Next() {
		exp := resultSet.toMap(i)
		buf := new(bytes.Buffer)
		if err := EncodeJSON(buf, resultSet); err != nil {
			t.Fatalf("expected no error when JSON encoding, got: %v", err)
		}
		var res []map[string]interface{}
		b := buf.Bytes()
		if err := json.Unmarshal(b, &res); err != nil {
			t.Fatalf("expected no error unmarshaling JSON, got: %v\n-- encoded --\n%s\n-- end--", err, string(b))
		}
		if !reflect.DeepEqual(res, exp) {
			t.Errorf("expected results to be equal, got:\n-- encoded --\n%s\n-- end--", string(b))
		}
		i++
	}
}

func TestTemplateEncoder(t *testing.T) {
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
`
	tpl := `{{ $headers := .Headers }}{{ range $i, $r := .Rows }}Row {{ $i }}:{{ range $j, $c := $r }}
  {{ index $headers $j }} = "{{ $c }}"{{ end }}
{{ end }}`
	buf := new(bytes.Buffer)
	if err := EncodeTemplateAll(buf, rs(), WithRawTemplate(tpl, "text")); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestUnalignedEncoder(t *testing.T) {
	exp := `author_id|name|z
14|a	b	c	d|x
15|aoeu
test
|
`
	buf := new(bytes.Buffer)
	if err := EncodeUnalignedAll(buf, rs()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}

func TestCSVEncoder(t *testing.T) {
	exp := `author_id,name,z
14,"a	b	c	d",x
15,"aoeu
test
",
`
	buf := new(bytes.Buffer)
	if err := EncodeCSVAll(buf, rs()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	actual := buf.String()
	if actual != exp {
		t.Errorf("expected:\n%q\n---\ngot:\n%q", exp, actual)
	}
}
