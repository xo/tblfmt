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
