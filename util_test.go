package tblfmt

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type p struct {
	name string
	dob  time.Time
	f    float64
	hash []byte
	char []byte
	z    interface{}
}

// newp creates a new p using the rand source.
func newp(src *rand.Rand) p {
	hash := md5.Sum([]byte(randstr(src)))
	var char []byte
	if src.Intn(2) == 1 {
		char = []byte{byte(int('a') + src.Intn(26))}
	}
	var z interface{}
	switch src.Intn(4) {
	case 0, 1:
	case 2:
		c := 1 + src.Intn(5)
		m := make(map[string]interface{}, c)
		for i := 0; i < c; i++ {
			r := []rune(randstr(src))
			m[string(r[0:3])] = string(r[3:])
		}
		z = m
	case 3:
		y := make([]interface{}, 1+src.Intn(5))
		for i := range y {
			r := []rune(randstr(src))
			y[i] = string(r[0 : 1+src.Intn(6)])
		}
		z = y
	}
	return p{
		name: randstr(src),
		dob:  randtime(src),
		f:    src.Float64(),
		hash: []byte(fmt.Sprintf("%x", hash[:])),
		char: char,
		z:    z,
	}
}

var randval int64

func init() {
	d := strings.ToLower(os.Getenv("DETERMINISTIC"))
	if d == "" || d == "n" || d == "no" || d == "f" || d == "false" || d == "0" || d == "off" {
		randval = 1549508725559526476
	} else {
		if r, err := strconv.ParseInt(d, 10, 64); err == nil {
			randval = r
		} else {
			randval = time.Now().UnixNano()
		}
	}
}

func randsrc() *rand.Rand {
	return rand.New(rand.NewSource(randval))
}

var glyphs = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 _'\"\t\b\n\rゼ一二三四五六七八九十〇")

func randstr(src *rand.Rand) string {
	l := 6 + src.Intn(32)
	r := make([]rune, l)
	for i := 0; i < l; i++ {
		r[i] = glyphs[src.Intn(len(glyphs))]
	}
	return string(r)
}

func randtime(src *rand.Rand) time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	return time.Unix(src.Int63n(delta)+min, 0).UTC()
}

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

// rss creates a result set for the specified columns and vals.
func rss(cols []string, vals ...[][]interface{}) *rset {
	return &rset{
		cols: cols,
		vals: vals,
	}
}

// rsbig creates a large result set for testing / benchmarking purposes.
func rsbig() *rset {
	src := randsrc()
	count := src.Intn(1000)
	// generate rows
	vals := make([][]interface{}, count)
	for i := 0; i < count; i++ {
		p := newp(src)
		vals[i] = []interface{}{i + 1, p.name, p.dob, p.f, p.hash, p.char, p.z}
	}
	return &rset{
		cols: []string{"id", "name", "dob", "float", "hash", "", "z"},
		vals: [][][]interface{}{vals},
	}
}

func rstiny() *rset {
	return &rset{
		cols: []string{"z"},
		vals: [][][]interface{}{
			{
				{"x"},
			},
		},
	}
}

func rswide() *rset {
	return &rset{
		cols: []string{
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"cccccccccccccccccccccccccccccc",
			"dddddddddddddddddddddddddddddd",
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"ffffffffffffffffffffffffffffff",
			"gggggggggggggggggggggggggggggg",
			"hhhhhhhhhhhhhhhhhhhhhhhhhhhhhh",
			"iiiiiiiiiiiiiiiiiiiiiiiiiiiiii",
			"jjjjjjjjjjjjjjjjjjjjjjjjjjjjjj",
		},
		vals: [][][]interface{}{
			{
				{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			},
		},
	}
}

// rsset returns a predefined set of records for rs.
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

func (r *rset) Reset() {
	r.pos, r.rs = 0, 0
}

var errPsqlConnNotDefined = errors.New("PSQL_CONN not defined")

// psqlEncodeAll does a values query for each of the values in the result set,
// writing captured output to the writer.
func psqlEncodeAll(w io.Writer, resultSet ResultSet, params map[string]string) error {
	dsn := os.Getenv("PSQL_CONN")
	if len(dsn) == 0 {
		return errPsqlConnNotDefined
	}
	if err := psqlEncode(w, resultSet, params, dsn); err != nil {
		return err
	}
	for resultSet.NextResultSet() {
		if _, err := w.Write(newline); err != nil {
			return err
		}
		if err := psqlEncode(w, resultSet, params, dsn); err != nil {
			return err
		}
	}
	if _, err := w.Write(newline); err != nil {
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

// psqlEncode does a single value query using psql, writing the captured output
// to the writer.
func psqlEncode(w io.Writer, resultSet ResultSet, params map[string]string, dsn string) error {
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

type noopWriter struct{}

func (*noopWriter) Write(buf []byte) (int, error) {
	return len(buf), nil
}

// D parses s as a YYYY-MM-DD format.
func D(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// rsr creates a result set row for the passed values.
func rsr(vals ...interface{}) []interface{} {
	return vals
}
