package tblfmt

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestJSONEncoder(t *testing.T) {
	resultSet := rs()

	var err error
	var i int
	for resultSet.Next() {
		exp := resultSet.toMap(i)

		buf := new(bytes.Buffer)
		if err = EncodeJSON(buf, resultSet); err != nil {
			t.Fatalf("expected no error when JSON encoding, got: %v", err)
		}
		var res []map[string]interface{}
		b := buf.Bytes()
		if err = json.Unmarshal(b, &res); err != nil {
			t.Fatalf("expected no error unmarshaling JSON, got: %v\n-- encoded --\n%s\n-- end--", err, string(b))
		}

		if !reflect.DeepEqual(res, exp) {
			t.Errorf("expected results to be equal, got:\n-- encoded --\n%s\n-- end--", string(b))
		}

		i++
	}
}

func TestTemplateEncoder(t *testing.T) {
	expected := `
Row 0:
  author_id = "14"
  name = "a	b	c	d"
  z = "x"

Row 1:
  author_id = "15"
  name = "aoeu
test
"
  z = ""

`
	template := `
{{ range $i, $r := .Rows }}Row {{ $i }}:
{{ range . }}  {{ .Name }} = "{{ .Value }}"
{{ end }}
{{ end }}`
	buf := new(bytes.Buffer)
	if err := EncodeTemplateAll(buf, rs(), WithTextTemplate(template)); err != nil {
		t.Fatalf("expected no error when Template encoding, got: %v", err)
	}
	actual := buf.String()
	if actual != expected {
		t.Fatalf("expected encoder to return:\n-- expected --\n%v\n-- end --\n\nbut got:\n-- encoded --\n%s\n-- end --", expected, actual)
	}
}
