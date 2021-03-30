package tblfmt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xo/tblfmt/internal"
	"github.com/xo/tblfmt/testdata"
)

func TestNewCrosstabView(t *testing.T) {
	tests := crosstabTests(t)
	for _, test := range tests {
		test = test
	}
}

type crosstabTest struct {
	q   string
	p   []string
	rs  *internal.Rset
	exp [][]interface{}
}

func crosstabTests(t *testing.T) []crosstabTest {
	buf, err := testdata.Testdata.ReadFile("crosstab.txt")
	if err != nil {
		t.Fatal(err)
	}
	var tests []crosstabTest
	for _, b := range bytes.Split(buf, []byte("\n\n")) {
		z := bytes.Split(b, []byte("--\n"))
		if len(z) != 3 {
			t.Fatalf("t should be 3, got: %d", len(z))
		}
		test := crosstabTest{
			q: strings.TrimSpace(string(z[0])),
		}
		t.Logf(">>> %q", test.q)
		tests = append(tests, test)
	}
	return tests
}
