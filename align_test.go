package tblfmt

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"
)

// TestWithColumnAlign makes sure that WithColumnAlign aligns a column by its
// database type: a number that the driver returns as a string, and the columns
// of a result set that has no rows (B44). psql right aligns a numeric column:
//
//	$ psql -X -P format=latex -c "select 1 as n, 'x'::text as t where false"
//	\begin{tabular}{r | l}
//	\textit{n} & \textit{t} \\
//	\hline
//	\end{tabular}
//
//	\noindent (0 rows) \\
func TestWithColumnAlign(t *testing.T) {
	t.Parallel()
	db := sql.OpenDB(alignConnector{})
	t.Cleanup(func() { _ = db.Close() })
	align := func(col *sql.ColumnType) (Align, bool) {
		if col.DatabaseTypeName() == "NUMERIC" {
			return AlignRight, true
		}
		return AlignLeft, true
	}
	tests := []struct {
		name   string
		query  string
		params map[string]string
		exp    string
	}{
		{"aligned", "rows", map[string]string{"format": "aligned"}, "  n  | t \n-----+---\n  10 | x\n 2.5 | y\n(2 rows)\n"},
		{"latex", "rows", map[string]string{"format": "latex"}, "\\begin{tabular}{r | l}\n\\textit{n} & \\textit{t} \\\\\n\\hline\n10 & x \\\\\n2.5 & y \\\\\n\\end{tabular}\n\n\\noindent (2 rows) \\\\\n\n"},
		{"latex no rows", "none", map[string]string{"format": "latex"}, "\\begin{tabular}{r | l}\n\\textit{n} & \\textit{t} \\\\\n\\hline\n\\end{tabular}\n\n\\noindent (0 rows) \\\\\n\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			rows, err := db.QueryContext(context.Background(), test.query)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			defer func() { _ = rows.Close() }()
			buf := new(bytes.Buffer)
			if err := EncodeAll(buf, rows, test.params, WithColumnAlign(align)); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if s := buf.String(); s != test.exp {
				t.Errorf("expected:\n%q\ngot:\n%q", test.exp, s)
			}
		})
	}
}

// alignConnector is a database/sql connector for a driver that returns a
// numeric column as strings, as many drivers do. The query "rows" returns two
// rows, and any other query returns none.
type alignConnector struct{}

func (alignConnector) Connect(context.Context) (driver.Conn, error) { return alignConn{}, nil }
func (alignConnector) Driver() driver.Driver                        { return nil }

type alignConn struct{}

func (alignConn) Prepare(query string) (driver.Stmt, error) { return alignStmt(query), nil }
func (alignConn) Close() error                              { return nil }
func (alignConn) Begin() (driver.Tx, error)                 { return nil, driver.ErrSkip }

type alignStmt string

func (alignStmt) Close() error                               { return nil }
func (alignStmt) NumInput() int                              { return 0 }
func (alignStmt) Exec([]driver.Value) (driver.Result, error) { return nil, driver.ErrSkip }
func (s alignStmt) Query([]driver.Value) (driver.Rows, error) {
	var vals [][]driver.Value
	if strings.EqualFold(string(s), "rows") {
		vals = [][]driver.Value{{"10", "x"}, {"2.5", "y"}}
	}
	return &alignRows{vals: vals}, nil
}

type alignRows struct {
	vals [][]driver.Value
}

func (*alignRows) Columns() []string { return []string{"n", "t"} }
func (*alignRows) Close() error      { return nil }
func (r *alignRows) Next(dest []driver.Value) error {
	if len(r.vals) == 0 {
		return io.EOF
	}
	copy(dest, r.vals[0])
	r.vals = r.vals[1:]
	return nil
}

// ColumnTypeDatabaseTypeName satisfies the
// driver.RowsColumnTypeDatabaseTypeName interface.
func (*alignRows) ColumnTypeDatabaseTypeName(i int) string {
	return []string{"NUMERIC", "TEXT"}[i]
}
