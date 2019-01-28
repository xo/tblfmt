package tblfmt

import (
	"fmt"
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
