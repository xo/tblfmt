package tblfmt

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// BenchmarkEncode measures each format at 10,000 and 1,000,000 rows. Each row
// holds an integer, 100 bytes of text and a float. The table formats also run
// with a batch of 1000 rows (WithCount), which keeps fewer rows in memory.
// See B34 in docs/BACKLOG.md.
//
// Run it with:
//
//	go test -run '^$' -bench BenchmarkEncode -benchmem
func BenchmarkEncode(b *testing.B) {
	tests := []struct {
		name   string
		params map[string]string
		batch  bool
	}{
		{"aligned", map[string]string{"format": "aligned"}, true},
		{"wrapped", map[string]string{"format": "wrapped", "columns": "80"}, true},
		{"expanded", map[string]string{"format": "aligned", "expanded": "on"}, true},
		{"unaligned", map[string]string{"format": "unaligned"}, false},
		{"csv", map[string]string{"format": "csv"}, false},
		{"json", map[string]string{"format": "json"}, false},
		{"html", map[string]string{"format": "html"}, false},
		{"latex", map[string]string{"format": "latex"}, false},
		{"troff-ms", map[string]string{"format": "troff-ms"}, false},
	}
	for _, rows := range []int{10_000, 1_000_000} {
		for _, test := range tests {
			counts := []int{0}
			if test.batch {
				counts = append(counts, 1000)
			}
			for _, count := range counts {
				b.Run(fmt.Sprintf("%s/rows=%d/count=%d", test.name, rows, count), func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						rs := &benchResultSet{rows: rows}
						if err := EncodeAll(io.Discard, rs, test.params, WithCount(count)); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
}

// benchText is the text of each row.
var benchText = strings.Repeat("abcdefghij", 10)

// benchResultSet is a result set that makes each row when the encoder scans
// it. It holds no rows, so the allocations of a benchmark are the encoder's.
type benchResultSet struct {
	rows int
	row  int
}

// Columns satisfies the ResultSet interface.
func (*benchResultSet) Columns() ([]string, error) {
	return []string{"id", "text", "f"}, nil
}

// Next satisfies the ResultSet interface.
func (rs *benchResultSet) Next() bool {
	rs.row++
	return rs.row <= rs.rows
}

// Scan satisfies the ResultSet interface. The encoders scan into *any.
func (rs *benchResultSet) Scan(vals ...any) error {
	*vals[0].(*any) = int64(rs.row)
	*vals[1].(*any) = benchText
	*vals[2].(*any) = float64(rs.row) / 3
	return nil
}

// NextResultSet satisfies the ResultSet interface.
func (*benchResultSet) NextResultSet() bool {
	return false
}

// Err satisfies the ResultSet interface.
func (*benchResultSet) Err() error {
	return nil
}

// Close satisfies the ResultSet interface.
func (*benchResultSet) Close() error {
	return nil
}
