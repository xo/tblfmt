// Package test compares tblfmt with other packages that write text tables.
// It is a module of its own, so that tblfmt does not require them.
package test

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/alexeyco/simpletable"
	aquatable "github.com/aquasecurity/table"
	lipgloss "github.com/charmbracelet/lipgloss/table"
	pretty "github.com/jedib0t/go-pretty/v6/table"
	"github.com/olekukonko/tablewriter"
	rodaine "github.com/rodaine/table"

	"github.com/xo/tblfmt"
)

// writer is a package that writes a table of the rows to w.
type writer struct {
	name  string
	write func(w io.Writer, headers []string, rows [][]string) error
}

// writers are the packages that the benchmark compares. Each one writes the
// table in its default style.
var writers = []writer{
	{"tblfmt", func(w io.Writer, headers []string, rows [][]string) error {
		return tblfmt.EncodeAll(w, newResultSet(headers, rows), map[string]string{"format": "aligned"})
	}},
	{"tabwriter", func(w io.Writer, headers []string, rows [][]string) error {
		tw := tabwriter.NewWriter(w, 0, 8, 1, ' ', 0)
		if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
			return err
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
				return err
			}
		}
		return tw.Flush()
	}},
	{"tablewriter", func(w io.Writer, headers []string, rows [][]string) error {
		t := tablewriter.NewTable(w)
		t.Header(headers)
		for _, row := range rows {
			if err := t.Append(row); err != nil {
				return err
			}
		}
		return t.Render()
	}},
	{"go-pretty", func(w io.Writer, headers []string, rows [][]string) error {
		t := pretty.NewWriter()
		t.SetOutputMirror(w)
		t.AppendHeader(prettyRow(headers))
		for _, row := range rows {
			t.AppendRow(prettyRow(row))
		}
		t.Render()
		return nil
	}},
	{"lipgloss", func(w io.Writer, headers []string, rows [][]string) error {
		t := lipgloss.New().Headers(headers...).Rows(rows...)
		_, err := io.WriteString(w, t.Render())
		return err
	}},
	{"simpletable", func(w io.Writer, headers []string, rows [][]string) error {
		t := simpletable.New()
		t.Header = &simpletable.Header{Cells: simpleCells(headers)}
		for _, row := range rows {
			t.Body.Cells = append(t.Body.Cells, simpleCells(row))
		}
		_, err := io.WriteString(w, t.String())
		return err
	}},
	{"rodaine", func(w io.Writer, headers []string, rows [][]string) error {
		t := rodaine.New(anys(headers)...).WithWriter(w)
		for _, row := range rows {
			t.AddRow(anys(row)...)
		}
		t.Print()
		return nil
	}},
	{"aquasecurity", func(w io.Writer, headers []string, rows [][]string) error {
		t := aquatable.New(w)
		t.SetHeaders(headers...)
		for _, row := range rows {
			t.AddRow(row...)
		}
		t.Render()
		return nil
	}},
}

// BenchmarkWrite measures each package as it writes the same table of 1,000
// and 10,000 rows. Each row holds an integer, 100 bytes of text and a float,
// as strings. Run it in this directory with:
//
//	go test -run '^$' -bench BenchmarkWrite -benchmem
func BenchmarkWrite(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		headers, rows := table(n)
		for _, wr := range writers {
			b.Run(fmt.Sprintf("rows=%d/%s", n, wr.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if err := wr.write(io.Discard, headers, rows); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// TestWrite makes sure that each package writes each row, so that a change to
// the API of a package fails here and not in the benchmark.
func TestWrite(t *testing.T) {
	t.Parallel()
	headers, rows := table(3)
	for _, wr := range writers {
		t.Run(wr.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			if err := wr.write(&sb, headers, rows); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			for _, row := range rows {
				if !strings.Contains(sb.String(), row[2]) {
					t.Errorf("expected %q in the output:\n%s", row[2], sb.String())
				}
			}
		})
	}
}

// table returns the headers and n rows of the table.
func table(n int) ([]string, [][]string) {
	text := strings.Repeat("abcdefghij", 10)
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{strconv.Itoa(i + 1), text, strconv.FormatFloat(float64(i+1)/3, 'g', -1, 64)}
	}
	return []string{"id", "text", "f"}, rows
}

// prettyRow returns the row as a go-pretty row.
func prettyRow(row []string) pretty.Row {
	r := make(pretty.Row, len(row))
	for i, s := range row {
		r[i] = s
	}
	return r
}

// simpleCells returns the row as simpletable cells.
func simpleCells(row []string) []*simpletable.Cell {
	cells := make([]*simpletable.Cell, len(row))
	for i, s := range row {
		cells[i] = &simpletable.Cell{Text: s}
	}
	return cells
}

// anys returns the strings as a slice of any.
func anys(strs []string) []any {
	v := make([]any, len(strs))
	for i, s := range strs {
		v[i] = s
	}
	return v
}

// resultSet is a tblfmt result set of rows of strings.
type resultSet struct {
	headers []string
	rows    [][]string
	row     int
}

// newResultSet returns a result set of the rows.
func newResultSet(headers []string, rows [][]string) *resultSet {
	return &resultSet{headers: headers, rows: rows}
}

// Columns satisfies the tblfmt.ResultSet interface.
func (rs *resultSet) Columns() ([]string, error) {
	return rs.headers, nil
}

// Next satisfies the tblfmt.ResultSet interface.
func (rs *resultSet) Next() bool {
	rs.row++
	return rs.row <= len(rs.rows)
}

// Scan satisfies the tblfmt.ResultSet interface. The encoders scan into *any.
func (rs *resultSet) Scan(vals ...any) error {
	for i, s := range rs.rows[rs.row-1] {
		*vals[i].(*any) = s
	}
	return nil
}

// NextResultSet satisfies the tblfmt.ResultSet interface.
func (*resultSet) NextResultSet() bool {
	return false
}

// Err satisfies the tblfmt.ResultSet interface.
func (*resultSet) Err() error {
	return nil
}

// Close satisfies the tblfmt.ResultSet interface.
func (*resultSet) Close() error {
	return nil
}
