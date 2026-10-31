# About tblfmt

Package `tblfmt` writes result sets as text tables. A result set is the rows
and columns that a database query returns. `tblfmt` reads a result set one row
at a time, so it does not keep the whole result in memory. It writes tables
like this one:

```text
 author_id |            name            | z 
-----------+----------------------------+---
        14 | a       b       c       d  | 
        15 | aoeu                      +| 
           | test                      +| 
           |                            | 
        16 | foo\x08bar                 | 
        17 | a       b       \r        +| 
           |         a                  | 
        18 | 袈      袈              袈 | 
        19 | 袈      袈              袈+| a+
           |                            | 
(6 rows)
```

`tblfmt` writes each format that `psql` has: aligned, wrapped, unaligned, csv,
html, asciidoc, latex, latex-longtable and troff-ms. It also writes JSON, the
vertical format that the `mysql` client writes for `\G`, and a plain table
with no lines. [`usql`][usql] uses
all of them.

[![Unit Tests][tblfmt-ci-status]][tblfmt-ci]
[![Go Reference][goref-tblfmt-status]][goref-tblfmt]
[![Discord Discussion][discord-status]][discord]

[tblfmt-ci]: https://github.com/xo/tblfmt/actions/workflows/test.yml
[tblfmt-ci-status]: https://github.com/xo/tblfmt/actions/workflows/test.yml/badge.svg
[goref-tblfmt]: https://pkg.go.dev/github.com/xo/tblfmt
[goref-tblfmt-status]: https://pkg.go.dev/badge/github.com/xo/tblfmt.svg
[discord]: https://discord.gg/yJKEzc7prt (Discord Discussion)
[discord-status]: https://img.shields.io/discord/829150509658013727.svg?label=Discord&logo=Discord&colorB=7289da&style=flat-square (Discord Discussion)

## Installing

Install `tblfmt` with the [Go][go-project] tool:

```sh
$ go get -u github.com/xo/tblfmt
```

## Using

`tblfmt` is for [`usql`][usql] and for the `database/sql` types of Go. It
accepts any type that has this interface:

```go
// ResultSet is the shared interface for a result set.
type ResultSet interface {
	Next() bool
	Scan(...any) error
	Columns() ([]string, error)
	Close() error
	Err() error
	NextResultSet() bool
}
```

This program uses `tblfmt`:

```go
// _example/example.go
package main

import (
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/xo/dburl"

	"github.com/xo/tblfmt"
)

func main() {
	db, err := dburl.Open("postgres://booktest:booktest@localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	res, err := db.Query("select * from authors")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()
	enc, err := tblfmt.NewTableEncoder(
		res,
		// force minimum column widths
		tblfmt.WithWidths(20, 20),
	)
	if err = enc.EncodeAll(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
```

The program writes output like this:

```text
      author_id       |            name            | z 
----------------------+----------------------------+---
                   14 | a       b       c       d  | 
                   15 | aoeu                      +| 
                      | test                      +| 
                      |                            | 
                    2 | 袈      袈              袈 | 
(3 rows)
```

The [Go Reference][goref-tblfmt] has the full API.

## Differences from `psql`

`tblfmt` writes the same output as `psql`. The differences below are
deliberate. If you find a different difference, report it as a bug.
[docs/BACKLOG.md](docs/BACKLOG.md) lists the differences that are known
faults.

### Blank line after a table

`psql` writes one blank line after an aligned table. `tblfmt` does not.
[`usql`][usql] is the client that matches `psql`, so `usql` writes the line.
If `tblfmt` wrote it too, `usql` would write two. See D48 in
[docs/PLAN.md](docs/PLAN.md).

### Quotes in csv

`tblfmt` quotes a csv field that holds a space or a tab. `psql` quotes a field
only when it holds the separator, a line break or a double quote. For a field
that holds `a`, a tab and `b`, `psql` writes `a<TAB>b`, and `tblfmt` writes
`"a<TAB>b"`. See D52 in [docs/PLAN.md](docs/PLAN.md).

### Separator between unaligned result sets

`psql` writes nothing between two result sets in the unaligned and csv
formats. `tblfmt` writes a newline by default. The caller chooses the
separator with `WithResultSetSeparator`. See D53 in
[docs/PLAN.md](docs/PLAN.md).

### Control characters in html and asciidoc

`tblfmt` escapes a control character in the html, asciidoc and vertical
formats, as it does in a table. For example, it writes U+0008 as `\x08`.
`psql` writes the character as it is in html and asciidoc. See D54 in
[docs/PLAN.md](docs/PLAN.md).

### Wrapped columns that hold a wide character

In the wrapped format, `psql` can make a column 1 column wide when its values
hold a character 2 columns wide, such as `日` or an emoji. The line with that
character is then 1 column too wide. `tblfmt` keeps such a column at least 2
columns wide. See D56 in [docs/PLAN.md](docs/PLAN.md).

### Width of an emoji

`tblfmt` measures the width of text by grapheme cluster. A grapheme cluster is
the group of code points that a terminal draws as one character. `psql`
measures each code point. An emoji with a skin tone, such as `👍🏽`, is two
code points, and a current terminal draws it 2 columns wide. `psql` counts it
4 columns wide and pads the cell by 2 columns too few. `tblfmt` counts it 2
columns wide. A joined emoji, such as `👨‍👩‍👧`, and a flag are one
grapheme cluster each, and `tblfmt` counts each of them 2 columns wide. See
D39 in [docs/PLAN.md](docs/PLAN.md).

A terminal that draws each code point on its own, such as xterm, does not line
up such a column.

## Comparison with other packages

Other Go packages write text tables too. They take the rows as strings and
draw the table in their own style. `tblfmt` reads a result set from a
database, and writes it as `psql` does. The table below compares the
versions that the benchmark uses. "Some" means that the package has the
feature only in a mode or an option.

| Feature | `tblfmt` | [`text/tabwriter`][tabwriter] | [`tablewriter`][tablewriter] v1.1.5 | [`go-pretty`][go-pretty] v6.8.3 | [`lipgloss`][lipgloss] v1.1.0 | [`simpletable`][simpletable] v1.0.0 | [`rodaine/table`][rodaine] v1.4.1 | [`aquasecurity/table`][aquasecurity] v1.11.0 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Reads a `*sql.Rows` | yes | no | no | no | no | no | no | no |
| Writes a row before it reads the next row | some | no | some | no | no | no | no | no |
| Writes a table as `psql` does | yes | no | no | no | no | no | no | no |
| Writes each row as a record (expanded) | yes | no | no | no | no | no | no | no |
| Wraps a table to a width | yes | no | yes | yes | yes | no | no | yes |
| Formats other than a text table | csv, json, html, asciidoc, latex, troff | no | markdown, html, svg | csv, tsv, html, markdown | no | no | no | no |
| Colors and text styles | no | no | yes | yes | yes | no | yes | yes |

`tblfmt` writes each row of the unaligned, csv and json formats when it reads
it. The table formats keep one batch of rows, and `WithCount` sets the size of
the batch. `tablewriter` writes each row in its streaming mode.

The benchmark in [`test/`](test) writes a table of 10,000 rows with each
package. Each row holds an integer, 100 bytes of text and a float, as strings.
Each package draws its default style. `text/tabwriter` and `rodaine/table`
draw no lines, and `tblfmt` also escapes each value and measures it by
grapheme cluster. The results on an AMD Ryzen 9 9950X with Go 1.27.1, on
2026-10-31:

| Package | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| `text/tabwriter` | 2.8 ms | 9.4 MiB | 30,000 |
| `rodaine/table` | 6.3 ms | 7.6 MiB | 210,000 |
| `tblfmt` | 8.8 ms | 9.2 MiB | 213,000 |
| `go-pretty` | 11.1 ms | 12.4 MiB | 125,000 |
| `tablewriter` | 40.1 ms | 49.0 MiB | 1,086,000 |
| `simpletable` | 45.0 ms | 25.4 MiB | 545,000 |
| `lipgloss` | 82.0 ms | 78.1 MiB | 735,000 |
| `aquasecurity/table` | 215.5 ms | 344.7 MiB | 9,421,000 |

To run the benchmark, run this command in the `test` directory:

```sh
go test -run '^$' -bench BenchmarkWrite -benchmem
```

[tabwriter]: https://pkg.go.dev/text/tabwriter
[tablewriter]: https://github.com/olekukonko/tablewriter
[go-pretty]: https://github.com/jedib0t/go-pretty
[lipgloss]: https://github.com/charmbracelet/lipgloss
[simpletable]: https://github.com/alexeyco/simpletable
[rodaine]: https://github.com/rodaine/table
[aquasecurity]: https://github.com/aquasecurity/table

## Documentation

| Document | What it holds |
| --- | --- |
| [CONTRIBUTING.md](CONTRIBUTING.md) | What a change must do, and the commands to run before a pull request |
| [docs/PLAN.md](docs/PLAN.md) | Each decision that shapes `tblfmt`, and the reason for it |
| [docs/BACKLOG.md](docs/BACKLOG.md) | Known faults and work that is not done |
| [AGENTS.md](AGENTS.md) | The rules for a coding agent. `CLAUDE.md` imports it |

## Testing

Run the tests with `go test`:

```sh
$ go test -v
```

To measure each format at 10,000 and 1,000,000 rows, run the benchmarks:

```sh
$ go test -run '^$' -bench BenchmarkEncode -benchmem
```

The benchmark against other packages is in the `test` module. See
"Comparison with other packages".

`TestPsql` in the `test` module compares the output with `psql` 18, query by
query. It needs a PostgreSQL server. "Tests" in [AGENTS.md](AGENTS.md) says
how to start one with `dbrun` from [dbmeta][dbmeta].

[go-project]: https://golang.org/project
[usql]: https://github.com/xo/usql
[dbmeta]: https://github.com/xo/dbmeta
