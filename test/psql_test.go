package test

import (
	"bytes"
	"context"
	"database/sql"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/xo/tblfmt"
)

// TestPsql makes sure that tblfmt writes what psql writes, for the same query
// and the same \pset values. It needs a PostgreSQL server and psql 18. Start
// the server with dbrun from a dbmeta checkout, and set DBMETA_POSTGRES to the
// URL that it prints:
//
//	cd ../dbmeta/test && go run ./cmd/dbrun start postgres-18
//	export DBMETA_POSTGRES=$(go run ./cmd/dbrun dsn --json postgres-18 | jq -r '.[0].url')
//
// The queries are plain SELECT statements, and make no table. Each case leaves
// out what a decision in docs/PLAN.md makes different from psql: the blank
// line after a table (D48), the quotes of csv (D52), the escapes of a control
// character in the template formats (D54) and the width of an emoji (D39).
func TestPsql(t *testing.T) {
	dsn := os.Getenv("DBMETA_POSTGRES")
	if dsn == "" {
		t.Skip("DBMETA_POSTGRES is not set. See the comment of TestPsql")
	}
	out, err := exec.CommandContext(t.Context(), "psql", "--version").Output()
	if err != nil {
		t.Fatalf("expected psql on the PATH, got: %v", err)
	}
	if !bytes.Contains(out, []byte(" 18.")) {
		t.Fatalf("expected psql 18, got: %s", out)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer func() { _ = db.Close() }()
	for _, g := range psqlGroups() {
		for _, params := range g.params {
			for i, q := range g.queries {
				name := psqlName(params) + "/" + strings.Fields(q)[0] + "-" + string(rune('a'+i))
				t.Run(name, func(t *testing.T) {
					exp := runPsql(t, dsn, params, q)
					got := runTblfmt(t, db, params, q)
					// psql writes a blank line after an aligned table, and usql
					// writes it for tblfmt (D48)
					switch params["format"] {
					case "aligned", "wrapped":
						got += "\n"
					}
					if got != exp {
						t.Errorf("query: %s\nexpected:\n%s\ngot:\n%s", q, exp, got)
					}
				})
			}
		}
	}
}

// psqlGroup is a set of \pset values, and the queries to run with each.
type psqlGroup struct {
	params  []map[string]string
	queries []string
}

// psqlGroups returns the cases of TestPsql.
func psqlGroups() []psqlGroup {
	// basic queries hold no space or tab, so that csv can run them (D52)
	basic := []string{
		`select 1 as a, 'x'::text as b where false`,
		`select`,
		`select from generate_series(1, 2)`,
		`select g as n, md5(g::text) as h, (g * 1.5)::float8 as f, (g * 2.5)::numeric as d from generate_series(1, 3) g`,
		`select 42 as n, E'bb\ncc\nd' as t, null::text as z union all select 7, 'x', 'q'`,
	}
	// text queries hold spaces, tabs, newlines and the characters that the
	// template formats escape, and no control character (D54)
	text := append(slices.Clone(basic),
		`select 42 as n, E'a\tb' as t union all select 7, E'bb\ncc\nd' union all select null, 'x'`,
		`select 1 as "a`+"\n"+`bb", 'x'::text as c union all select 22, E'y\nz'`,
		`select 'a'::text as t, 42 as n union all select E'bb\ncc', 7`,
		`select g as n, repeat('x', g) as v from generate_series(1, 11) g`,
		`select E'\tx\ty' as t, 'z' as u`,
		`select 42 as n, E'a#b$c%d&e<f>g\\h^i_j{k|l}m~n "q"' as t, null::text as z union all select 7, E'two\nlines', '  lead'`,
		`select '日本語のテキストはとても長いです日本語のテキスト' as j, 1 as k`,
	)
	// aligned queries hold control characters too
	aligned := append(slices.Clone(text),
		`select E'a\x01b\x07c\x7fd' as c, U&'e\0085f' as d, E'g\x1bh' as e`,
		`select g as n, repeat(md5(g::text), 2) as long, (g * 1.5)::text as other from generate_series(1, 4) g`,
		`select E'a\tb\tccccccccccccccccccccccccccccccc' as t, 2 as u`,
	)
	var table, templates, unaligned, csv []map[string]string
	for _, format := range []string{"aligned", "wrapped"} {
		for _, style := range [][2]string{{"ascii", ""}, {"old-ascii", ""}, {"unicode", ""}, {"unicode", "double"}} {
			for _, border := range []string{"0", "1", "2"} {
				for _, expanded := range []string{"off", "on"} {
					p := map[string]string{"format": format, "linestyle": style[0], "border": border, "expanded": expanded}
					if style[1] != "" {
						p["unicode_border_linestyle"] = style[1]
						p["unicode_column_linestyle"] = style[1]
					}
					if format == "wrapped" {
						p["columns"] = "30"
					}
					table = append(table, p)
				}
			}
		}
	}
	table = append(table,
		map[string]string{"format": "aligned", "title": "My Title"},
		map[string]string{"format": "aligned", "border": "2", "title": "T"},
		map[string]string{"format": "aligned", "tuples_only": "on"},
		map[string]string{"format": "aligned", "tuples_only": "on", "border": "2"},
		map[string]string{"format": "aligned", "footer": "off"},
		map[string]string{"format": "aligned", "expanded": "on", "title": "T"},
		map[string]string{"format": "aligned", "expanded": "on", "tuples_only": "on"},
		map[string]string{"format": "aligned", "expanded": "on", "tuples_only": "on", "border": "2"},
		map[string]string{"format": "aligned", "expanded": "on", "footer": "off"},
		map[string]string{"format": "aligned", "expanded": "auto", "columns": "12"},
		map[string]string{"format": "wrapped", "columns": "12"},
		map[string]string{"format": "wrapped", "columns": "200"},
		map[string]string{"format": "wrapped", "columns": "8", "expanded": "on"},
	)
	for _, format := range []string{"html", "asciidoc", "latex", "latex-longtable", "troff-ms"} {
		for _, border := range []string{"0", "1", "2", "3"} {
			templates = append(templates, map[string]string{"format": format, "border": border})
		}
		templates = append(templates,
			map[string]string{"format": format, "title": "My_Title #1"},
			map[string]string{"format": format, "border": "2", "title": "T"},
			map[string]string{"format": format, "tuples_only": "on"},
			map[string]string{"format": format, "tuples_only": "on", "border": "2"},
			map[string]string{"format": format, "footer": "off"},
			map[string]string{"format": format, "expanded": "on"},
			map[string]string{"format": format, "expanded": "on", "border": "2"},
			map[string]string{"format": format, "expanded": "on", "title": "T"},
			map[string]string{"format": format, "expanded": "on", "tuples_only": "on"},
		)
	}
	templates = append(templates,
		map[string]string{"format": "html", "tableattr": `class="x"`},
		map[string]string{"format": "latex-longtable", "tableattr": "0.2 0.3"},
		map[string]string{"format": "latex-longtable", "tableattr": "0.1 0.2 0.3 0.4", "border": "2", "title": "T"},
	)
	for _, expanded := range []string{"off", "on"} {
		unaligned = append(unaligned,
			map[string]string{"format": "unaligned", "expanded": expanded},
			map[string]string{"format": "unaligned", "expanded": expanded, "fieldsep": ";"},
			map[string]string{"format": "unaligned", "expanded": expanded, "title": "T"},
			map[string]string{"format": "unaligned", "expanded": expanded, "tuples_only": "on"},
			map[string]string{"format": "unaligned", "expanded": expanded, "footer": "off"},
		)
		csv = append(csv,
			map[string]string{"format": "csv", "expanded": expanded},
			map[string]string{"format": "csv", "expanded": expanded, "csv_fieldsep": ";"},
			map[string]string{"format": "csv", "expanded": expanded, "tuples_only": "on"},
		)
	}
	return []psqlGroup{
		{table, aligned},
		{templates, text},
		{unaligned, aligned},
		{csv, basic},
	}
}

// psqlName returns a name for the \pset values.
func psqlName(params map[string]string) string {
	var parts []string
	for _, k := range slices.Sorted(maps.Keys(params)) {
		parts = append(parts, k+"="+strings.ReplaceAll(params[k], " ", "_"))
	}
	return strings.Join(parts, ",")
}

// runPsql runs the query with psql, and returns what it writes.
func runPsql(t *testing.T, dsn string, params map[string]string, q string) string {
	t.Helper()
	args := []string{dsn, "-X", "-q", "-P", "pager=off"}
	for _, k := range slices.Sorted(maps.Keys(params)) {
		args = append(args, "-P", k+"="+params[k])
	}
	cmd := exec.CommandContext(t.Context(), "psql", append(args, "-c", q)...)
	cmd.Env = append(os.Environ(), "PGCLIENTENCODING=UTF8")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("psql: %v: %s", err, stderr.String())
	}
	return string(out)
}

// runTblfmt runs the query, and returns what tblfmt writes for it.
func runTblfmt(t *testing.T, db *sql.DB, params map[string]string, q string) string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), q)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var buf bytes.Buffer
	if err := tblfmt.EncodeAll(&buf, rows, maps.Clone(params), tblfmt.WithColumnAlign(psqlAlign)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return buf.String()
}

// psqlAlign aligns a column as psql does, by its type: column_type_alignment
// in src/bin/psql/common.c right aligns the numbers and the money type.
func psqlAlign(col *sql.ColumnType) (tblfmt.Align, bool) {
	switch col.DatabaseTypeName() {
	case "INT2", "INT4", "INT8", "FLOAT4", "FLOAT8", "NUMERIC", "OID", "XID", "XID8", "CID", "MONEY":
		return tblfmt.AlignRight, true
	}
	return tblfmt.AlignLeft, true
}
