package tblfmt

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	//	_ "github.com/lib/pq"
	"github.com/xo/tblfmt/internal"
	"github.com/xo/tblfmt/testdata"
)

func TestNewCrosstabView(t *testing.T) {
	tests := loadViewTests(t, "crosstab")
	for i, test := range tests {
		view, err := NewCrosstabView(test.Rset(), WithParams(test.params...))
		if err != nil {
			t.Fatalf("test %d expected no error, got: %v", i, err)
		}
		checkView(t, i, test, view)
	}
}

/*
func TestNewCrosstabView_psql(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://postgres:P4ssw0rd@localhost")
	if err != nil {
		t.Fatal(err)
	}
	tests := loadViewTests(t, "crosstab")
	for i, test := range tests {
		resultSet, err := db.Query(test.PsqlQuery())
		if err != nil {
			t.Fatal(err)
		}
		view, err := NewCrosstabView(resultSet, WithParams(test.params...))
		if err != nil {
			t.Fatalf("test %d expected no error, got: %v", i, err)
		}
		checkView(t, i, test, view)
	}
}
*/

func checkView(t *testing.T, testNum int, test viewTest, view ResultSet) {
	t.Helper()
	cols, err := view.Columns()
	if err != nil {
		t.Fatalf("test %d expected no error, got: %v", testNum, err)
	}
	if !reflect.DeepEqual(cols, test.expCols) {
		t.Errorf("test %d expected columns to be %v, got: %v", testNum, test.expCols, cols)
	}
	clen := len(cols)
	var vals [][]any
	for view.Next() {
		r := make([]any, clen)
		for i := range clen {
			r[i] = new(any)
		}
		if err := view.Scan(r...); err != nil {
			t.Fatalf("test %d expected no error, got: %v", testNum, err)
		}
		vals = append(vals, r)
	}
	if err := view.Err(); err != nil {
		t.Fatalf("test %d expected no error, got: %v", testNum, err)
	}
	// log.Printf(">>> test.expVals: %+v", test.expVals)
	if len(vals) != len(test.expVals) {
		t.Fatalf("test %d expected len(vals) == len(test.expVals): %d != %d", testNum, len(vals), len(test.expVals))
	}
	for i := range len(test.expVals) {
		row := make([]any, len(vals[i]))
		for j := range len(vals[i]) {
			row[j] = *(vals[i][j].(*any))
		}
		rs := fmt.Sprintf("%v", row)
		es := fmt.Sprintf("%v", test.expVals[i])
		if rs != es {
			t.Errorf("test %d expected row %d to match result\ngot: %v\nexp: %v", testNum, i, row, test.expVals[i])
		}
	}
}

func loadViewTests(t *testing.T, name string) []viewTest {
	t.Helper()
	buf, err := testdata.Testdata.ReadFile(name + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	var tests []viewTest
	for _, b := range bytes.Split(buf, []byte("\n\n")) {
		z := bytes.Split(b, []byte("--\n"))
		if len(z) != 3 {
			t.Fatalf("t should be 3, got: %d", len(z))
		}
		s := strings.Split(string(z[0]), ` \crosstabview`)
		if len(s) != 2 {
			t.Fatalf("s should be len 2, got: %d", len(s))
		}
		cols, vals := parseViewTest(z[1])
		expCols, expVals := parseViewTest(z[2])
		tests = append(tests, viewTest{
			q:       strings.TrimSpace(string(z[0])),
			params:  strings.Split(strings.TrimSpace(s[1]), " "),
			cols:    cols,
			vals:    vals,
			expCols: expCols,
			expVals: expVals,
		})
	}
	return tests
}

type viewTest struct {
	q       string
	params  []string
	cols    []string
	vals    [][]any
	expCols []string
	expVals [][]any
}

func (test viewTest) Rset() *internal.RS {
	return internal.New(test.cols, test.vals)
}

var azRE = regexp.MustCompile(`[^0-9\-]`)

func (test viewTest) PsqlQuery() string {
	var vals []string
	for _, v := range test.vals {
		var z []string
		for _, x := range v {
			s := fmt.Sprintf("%v", x)
			if azRE.MatchString(s) {
				s = "'" + s + "'"
			}
			z = append(z, s)
		}
		vals = append(vals, "("+strings.Join(z, ",")+")")
	}
	return fmt.Sprintf(`select * from (values %s) as t (%s)`, strings.Join(vals, ", "), strings.Join(test.cols, ", "))
}

func parseViewTest(buf []byte) ([]string, [][]any) {
	lines := strings.Split(strings.TrimSpace(string(buf)), "\n")
	cols := strings.Split(lines[0], "|")
	var vals [][]any
	for _, line := range lines[1:] {
		var row []any
		for _, c := range strings.Split(line, "|") {
			switch i, err := strconv.Atoi(c); {
			case err == nil:
				row = append(row, i)
			case c == "":
				row = append(row, nil)
			default:
				row = append(row, c)
			}
		}
		vals = append(vals, row)
	}
	return cols, vals
}
