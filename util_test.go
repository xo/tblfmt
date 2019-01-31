package tblfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

type rset struct {
	rs   int
	pos  int
	cols []string
	vals [][][]interface{}
}

// rs creates a simple result set for testing purposes.
func rs() *rset {
	s, t := rsset(14), rsset(38)
	r := &rset{
		cols: []string{"author_id", "name", "z"},
		vals: [][][]interface{}{
			s[:2], s[2:], t,
		},
	}
	return r
}

// a set of records for rs.
func rsset(i int) [][]interface{} {
	return [][]interface{}{
		[]interface{}{float64(i), "a\tb\tc\td", "x"},
		[]interface{}{float64(i + 1), "aoeu\ntest\n", nil},
		[]interface{}{float64(i + 2), "foo\bbar", nil},
		[]interface{}{float64(i + 3), "袈\t袈\t\t袈", nil},
		[]interface{}{float64(i + 4), "a\tb\t\r\n\ta", "a\n"},
		[]interface{}{float64(i + 5), "袈\t袈\t\t袈\n", nil},
		[]interface{}{float64(i + 6), "javascript", map[string]interface{}{
			fmt.Sprintf("test%d", i+7): "a value",
			fmt.Sprintf("test%d", i+8): "foo\bbar",
		}},
		[]interface{}{float64(i + 9), "slice", []string{"a", "b"}},
	}
}

func (r *rset) toMap(set int) []map[string]interface{} {
	var m []map[string]interface{}
	for i := 0; i < len(r.vals[set]); i++ {
		row := make(map[string]interface{}, len(r.cols))
		for j := range r.cols {
			row[r.cols[j]] = r.vals[set][i][j]
		}
		m = append(m, row)
	}
	return m
}

func (*rset) Err() error {
	return nil
}

func (*rset) Close() error {
	return nil
}

func (r *rset) Columns() ([]string, error) {
	return r.cols, nil
}

func (r *rset) Next() bool {
	return r.pos < len(r.vals[r.rs])
}

func (r *rset) Scan(vals ...interface{}) error {
	for i := range vals {
		x, ok := vals[i].(*interface{})
		if !ok {
			return fmt.Errorf("scan for col %d expected *interface{}, got: %T", i, vals[i])
		}
		*x = r.vals[r.rs][r.pos][i]
	}
	r.pos++
	return nil
}

func (r *rset) NextResultSet() bool {
	r.rs, r.pos = r.rs+1, 0
	return r.rs < len(r.vals)
}

var errPsqlConnNotDefined = errors.New("PSQL_CONN not defined")

// psqlEncodeAll does a values query for each of the values in the result set,
// writing captured output to the writer.
func psqlEncodeAll(w io.Writer, resultSet ResultSet, format string) error {
	dsn := os.Getenv("PSQL_CONN")
	if len(dsn) == 0 {
		return errPsqlConnNotDefined
	}

	var err error

	if err = psqlEncode(w, resultSet, format, dsn); err != nil {
		return err
	}

	for resultSet.NextResultSet() {
		if _, err = w.Write(newline); err != nil {
			return err
		}

		if err = psqlEncode(w, resultSet, format, dsn); err != nil {
			return err
		}
	}

	if _, err = w.Write(newline); err != nil {
		return err
	}

	return nil
}

const (
	psqlValuesQuery = `%s
SELECT * FROM (
  VALUES%s
) AS t (author_id, name, z);`
)

// psqlEncode does a single value query using psql, writing the catpured output
// to the writer.
func psqlEncode(w io.Writer, resultSet ResultSet, format, dsn string) error {
	var err error

	// read values
	var vals string
	var i int
	for resultSet.Next() {
		var id, name, z interface{}
		if err = resultSet.Scan(&id, &name, &z); err != nil {
			return err
		}
		var extra string
		if i != 0 {
			extra = ","
		}

		n := name.(string)
		vals += fmt.Sprintf("%s\n    (%v,E'%s', %s)", extra, id, psqlEsc(n), psqlEnc(n, z))

		i++
	}
	if err = resultSet.Err(); err != nil {
		return err
	}

	// exec
	stdout := new(bytes.Buffer)
	q := fmt.Sprintf(psqlValuesQuery, `\pset format `+format, vals)
	cmd := exec.Command("psql", dsn, "-q")
	cmd.Stdin, cmd.Stdout = bytes.NewReader([]byte(q)), stdout
	if err = cmd.Run(); err != nil {
		return err
	}

	_, err = w.Write(bytes.TrimRightFunc(stdout.Bytes(), unicode.IsSpace))
	if err != nil {
		return err
	}
	_, err = w.Write(newline)
	return err
}

// psqlEsc escapes a string as a psql string.
func psqlEsc(s string) string {
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, "\r", `\r`, -1)
	s = strings.Replace(s, "\t", `\t`, -1)
	s = strings.Replace(s, "\b", `\b`, -1)
	s = strings.Replace(s, "袈", `\u8888`, -1)
	return s
}

// psqlEnc encodes v based on n.
func psqlEnc(n string, v interface{}) string {
	if n != "javascript" && n != "slice" {
		return "NULL"
	}
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	s := strconv.QuoteToASCII(string(buf))
	return "E'" + s[1:len(s)-1] + "'"
}
