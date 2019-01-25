package tblfmt

import (
	"bytes"
	"log"
	"testing"
)

func TestNamedStyles(t *testing.T) {
	for _, n := range []string{
		"ascii",
		"old-ascii",
		"unicode",
		"double",
		"compact",
		"inline",
	} {
		enc, err := NewTableEncoder(rs(), WithNamedStyle(n))
		if err != nil {
			t.Fatalf("expected no error for %q, got: %v", n, err)
		}
		buf := new(bytes.Buffer)
		if err := enc.EncodeAll(buf); err != nil {
			t.Fatalf("expected no error when encoding for %s, got: %v", n, err)
		}
		log.Printf("%q style:\n%s", n, buf.String())
	}
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

// a set of records for rs.
func rsset(i int) [][]interface{} {
	return [][]interface{}{
		[]interface{}{i, "a\tb\tc\td", "x"},
		[]interface{}{i + 1, "aoeu\ntest\n", nil},
		[]interface{}{i + 2, "foo\bbar", nil},
		[]interface{}{i + 3, "袈\t袈\t\t袈", nil},
		[]interface{}{i + 4, "a\tb\t\r\n\ta", "a\n"},
		[]interface{}{i + 5, "袈\t袈\t\t袈\n", nil},
	}
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
		vals[i] = &r.vals[r.rs][r.pos][i]
	}
	r.pos++
	return nil
}

func (r *rset) NextResultSet() bool {
	r.rs, r.pos = r.rs+1, 0
	return r.rs < len(r.vals)
}

const (
	asciiBorder1 = ` author_id | name                  | z 
-----------+-----------------------+---
        14 | a	b	c	d  | x 
        15 | aoeu                 +|   
           | test                 +|   
           |                       |   
        16 | foo\bbar              |   
        17 | 袈	袈		袈 |   
        18 | a	b	\r        +| a+
           | 	a                  |   
        19 | 袈	袈		袈+|   
           |                       |   
(6 rows)`
)
