package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

// ResultSet is the shared interface for a result set.
type ResultSet interface {
	Next() bool
	Scan(...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
	NextResultSet() bool
}

// PsqlEncodeAll does a values query for each of the values in the result set,
// writing captured output to the writer.
func PsqlEncodeAll(w io.Writer, resultSet ResultSet, params map[string]string, dsn string) error {
	if err := PsqlEncode(w, resultSet, params, dsn); err != nil {
		return err
	}
	for resultSet.NextResultSet() {
		if _, err := w.Write(newline); err != nil {
			return err
		}
		if err := PsqlEncode(w, resultSet, params, dsn); err != nil {
			return err
		}
	}
	if _, err := w.Write(newline); err != nil {
		return err
	}
	return nil
}

// PsqlEncode does a single value query using psql, writing the captured output
// to the writer.
func PsqlEncode(w io.Writer, resultSet ResultSet, params map[string]string, dsn string) error {
	// read values
	var vals string
	var i int
	for resultSet.Next() {
		var id, name, z interface{}
		if err := resultSet.Scan(&id, &name, &z); err != nil {
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
	if err := resultSet.Err(); err != nil {
		return err
	}
	// build pset
	var pset string
	for k, v := range params {
		pset += fmt.Sprintf("\n\\pset %s '%s'", k, v)
	}
	// exec
	stdout := new(bytes.Buffer)
	q := fmt.Sprintf(psqlValuesQuery, pset, vals)
	cmd := exec.Command("psql", dsn, "-qX")
	cmd.Stdin, cmd.Stdout = bytes.NewReader([]byte(q)), stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	if _, err := w.Write(bytes.TrimRightFunc(stdout.Bytes(), unicode.IsSpace)); err != nil {
		return err
	}
	_, err := w.Write(newline)
	return err
}

const (
	psqlValuesQuery = `%s
SELECT * FROM (
  VALUES%s
) AS t (author_id, name, z);`
)

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

var newline []byte

func init() {
	if runtime.GOOS == "windows" {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
}
