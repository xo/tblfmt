package tblfmt

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"io"
	"testing"

	"github.com/xo/tblfmt/internal"
)

// TestWithBytesFormat makes sure that WithBytesFormat writes a binary value as
// the text that its func returns, in each format, and that a null value never
// reaches the func. The driver of the test returns the text column as bytes
// too, as the MySQL driver does, and the func returns false for it.
//
// Measured against psql 18.6, which writes a bytea value in hex:
//
//	$ psql -X -c "select '\x0102ff'::bytea as b, 'text'::text as t,
//	                     null::bytea as n, ''::bytea as e"
//	    b     |  t   | n | e
//	----------+------+---+----
//	 \x0102ff | text |   | \x
//	(1 row)
//
// The other cases are the same command with -P format=csv, unaligned and html.
func TestWithBytesFormat(t *testing.T) {
	t.Parallel()
	db := sql.OpenDB(bytesConnector{})
	t.Cleanup(func() { _ = db.Close() })
	var nulls int
	format := func(col *sql.ColumnType, b []byte) (string, bool) {
		if b == nil {
			nulls++
		}
		if col.DatabaseTypeName() != "BYTEA" {
			return "", false
		}
		return `\x` + hex.EncodeToString(b), true
	}
	tests := []struct {
		format string
		exp    string
	}{
		{"aligned", "    b     |  t   | n | e  \n----------+------+---+----\n \\x0102ff | text |   | \\x\n(1 row)\n"},
		{"csv", "b,t,n,e\n\\x0102ff,text,,\\x\n"},
		{"unaligned", "b|t|n|e\n\\x0102ff|text||\\x\n(1 row)\n"},
		{"html", "<table border=\"1\">\n  <tr>\n    <th align=\"center\">b</th>\n    <th align=\"center\">t</th>\n    <th align=\"center\">n</th>\n    <th align=\"center\">e</th>\n  </tr>\n  <tr valign=\"top\">\n    <td align=\"left\">\\x0102ff</td>\n    <td align=\"left\">text</td>\n    <td align=\"left\">&nbsp; </td>\n    <td align=\"left\">\\x</td>\n  </tr>\n</table>\n<p>(1 row)<br />\n</p>\n"},
		// JSON holds the text as a string
		{"json", "[{\"b\":\"\\\\x0102ff\",\"t\":\"text\",\"n\":null,\"e\":\"\\\\x\"}]\n"},
	}
	encode := func(name string) string {
		rows, err := db.QueryContext(context.Background(), "bytes")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		defer func() { _ = rows.Close() }()
		buf := new(bytes.Buffer)
		if err := EncodeAll(buf, rows, map[string]string{"format": name}, WithBytesFormat(format)); err != nil {
			t.Fatalf("%s: expected no error, got: %v", name, err)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("%s: expected no error, got: %v", name, err)
		}
		return buf.String()
	}
	for _, test := range tests {
		if s := encode(test.format); s != test.exp {
			t.Errorf("%s: expected:\n%q\ngot:\n%q", test.format, test.exp, s)
		}
	}
	if nulls != 0 {
		t.Errorf("expected no call for a null value, got: %d", nulls)
	}
	// a result set with no column types writes the bytes as before
	t.Run("no column types", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		rs := internal.New([]string{"b"}, [][]any{{[]byte("ab")}})
		if err := EncodeAll(buf, rs, map[string]string{"format": "csv"}, WithBytesFormat(format)); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if s, exp := buf.String(), "b\nab\n"; s != exp {
			t.Errorf("expected %q, got: %q", exp, s)
		}
	})
}

// bytesConnector is a database/sql connector for a driver that returns each
// value as bytes. Its one row has a bytea, a text value, a null bytea and an
// empty bytea.
type bytesConnector struct{}

func (bytesConnector) Connect(context.Context) (driver.Conn, error) { return bytesConn{}, nil }
func (bytesConnector) Driver() driver.Driver                        { return nil }

type bytesConn struct{}

func (bytesConn) Prepare(string) (driver.Stmt, error) { return bytesStmt{}, nil }
func (bytesConn) Close() error                        { return nil }
func (bytesConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

type bytesStmt struct{}

func (bytesStmt) Close() error                               { return nil }
func (bytesStmt) NumInput() int                              { return 0 }
func (bytesStmt) Exec([]driver.Value) (driver.Result, error) { return nil, driver.ErrSkip }
func (bytesStmt) Query([]driver.Value) (driver.Rows, error) {
	return &bytesRows{vals: [][]driver.Value{{[]byte{1, 2, 0xff}, []byte("text"), nil, []byte{}}}}, nil
}

type bytesRows struct {
	vals [][]driver.Value
}

func (*bytesRows) Columns() []string { return []string{"b", "t", "n", "e"} }
func (*bytesRows) Close() error      { return nil }
func (r *bytesRows) Next(dest []driver.Value) error {
	if len(r.vals) == 0 {
		return io.EOF
	}
	copy(dest, r.vals[0])
	r.vals = r.vals[1:]
	return nil
}

// ColumnTypeDatabaseTypeName satisfies the
// driver.RowsColumnTypeDatabaseTypeName interface.
func (*bytesRows) ColumnTypeDatabaseTypeName(i int) string {
	return []string{"BYTEA", "TEXT", "BYTEA", "BYTEA"}[i]
}
