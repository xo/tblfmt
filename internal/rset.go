package internal

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"time"
)

// RS is a result set.
type RS struct {
	rs   int
	pos  int
	cols []string
	vals [][][]any
}

// New creates a new result set
func New(cols []string, vals ...[][]any) *RS {
	return &RS{
		cols: cols,
		vals: vals,
	}
}

// Multi creates a result set with multiple result sets.
func Multi() *RS {
	s, t := rset(14), rset(38)
	r := &RS{
		cols: []string{"author_id", "name", "z"},
		vals: [][][]any{
			s[:2], s[2:], t,
		},
	}
	return r
}

// Big creates a random, big result set using the provided seed.
func Big(seed int64) *RS {
	src := rand.New(rand.NewSource(seed))
	count := src.Intn(1000)
	// generate rows
	vals := make([][]any, count)
	for i := range count {
		p := newRrow(src)
		vals[i] = []any{i + 1, p.name, p.dob, p.f, p.hash, p.char, p.z}
	}
	return &RS{
		cols: []string{"id", "name", "dob", "float", "hash", "", "z"},
		vals: [][][]any{vals},
	}
}

// Tiny creates a tiny result set.
func Tiny() *RS {
	return &RS{
		cols: []string{"z"},
		vals: [][][]any{
			{
				{"x"},
			},
		},
	}
}

// Wide creates a wide result set.
func Wide() *RS {
	return &RS{
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
		vals: [][][]any{
			{
				{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			},
		},
	}
}

// Err satisfies the ResultSet interface.
func (*RS) Err() error {
	return nil
}

// Err satisfies the ResultSet interface.
func (*RS) Close() error {
	return nil
}

// Err satisfies the ResultSet interface.
func (r *RS) Columns() ([]string, error) {
	return r.cols, nil
}

// Err satisfies the ResultSet interface.
func (r *RS) Next() bool {
	return r.pos < len(r.vals[r.rs])
}

// Err satisfies the ResultSet interface.
func (r *RS) Scan(vals ...any) error {
	for i := range vals {
		x, ok := vals[i].(*any)
		if !ok {
			return fmt.Errorf("scan for col %d expected *interface{}, got: %T", i, vals[i])
		}
		*x = r.vals[r.rs][r.pos][i]
	}
	r.pos++
	return nil
}

// Err satisfies the ResultSet interface.
func (r *RS) NextResultSet() bool {
	r.rs, r.pos = r.rs+1, 0
	return r.rs < len(r.vals)
}

// Reset resets the rset so that it can be used repeatedly.
func (r *RS) Reset() {
	r.pos, r.rs = 0, 0
}

// rrow is a random row.
type rrow struct {
	name string
	dob  time.Time
	f    float64
	hash []byte
	char []byte
	z    any
}

// newRrow creates a new random row using the rand source.
func newRrow(src *rand.Rand) rrow {
	hash := md5.Sum([]byte(rstr(src)))
	var char []byte
	if src.Intn(2) == 1 {
		char = []byte{byte(int('a') + src.Intn(26))}
	}
	var z any
	switch src.Intn(4) {
	case 0, 1:
	case 2:
		c := 1 + src.Intn(5)
		m := make(map[string]any, c)
		for range c {
			r := []rune(rstr(src))
			m[string(r[0:3])] = string(r[3:])
		}
		z = m
	case 3:
		y := make([]any, 1+src.Intn(5))
		for i := range y {
			r := []rune(rstr(src))
			y[i] = string(r[0 : 1+src.Intn(6)])
		}
		z = y
	}
	return rrow{
		name: rstr(src),
		dob:  rtime(src),
		f:    src.Float64(),
		hash: fmt.Appendf(nil, "%x", hash[:]),
		char: char,
		z:    z,
	}
}

var glyphs = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 _'\"\t\b\n\rゼ一二三四五六七八九十〇")

// rstr creates a random string using the rand source.
func rstr(src *rand.Rand) string {
	l := 6 + src.Intn(32)
	r := make([]rune, l)
	for i := range l {
		r[i] = glyphs[src.Intn(len(glyphs))]
	}
	return string(r)
}

// rtime creates a random time using the rand source.
func rtime(src *rand.Rand) time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	return time.Unix(src.Int63n(delta)+min, 0).UTC()
}

// rset returns predefined record set values.
func rset(i int) [][]any {
	return [][]any{
		{float64(i), "a\tb\tc\td", "x"},
		{float64(i + 1), "aoeu\ntest\n", nil},
		{float64(i + 2), "foo\bbar", nil},
		{float64(i + 3), "袈\t袈\t\t袈", nil},
		{float64(i + 4), "a\tb\t\r\n\ta", "a\n"},
		{float64(i + 5), "袈\t袈\t\t袈\n", nil},
		{float64(i + 6), "javascript", map[string]any{
			fmt.Sprintf("test%d", i+7): "a value",
			fmt.Sprintf("test%d", i+8): "foo\bbar",
		}},
		{float64(i + 9), "slice", []string{"a", "b"}},
	}
}

const Divider = "==============================================="
