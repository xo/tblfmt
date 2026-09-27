# About tblfmt

Package `tblfmt` writes result sets as text tables. A result set is the rows
and columns that a database query returns. `tblfmt` reads a result set one row
at a time, so it does not keep the whole result in memory. It writes tables
like this one:

```text
 author_id | name                  | z
-----------+-----------------------+---
        14 | a	b	c	d  |
        15 | aoeu                 +|
           | test                 +|
           |                       |
        16 | foo\bbar              |
        17 | a	b	\r        +|
           | 	a                  |
        18 | 袈	袈		袈 |
        19 | 袈	袈		袈+| a+
           |                       |
(6 rows)
```

`tblfmt` also has encoders for JSON, CSV, HTML, AsciiDoc, unaligned text and
the other formats that [`usql`][usql] supports.

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
	Scan(...interface{}) error
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
╔══════════════════════╦═══════════════════════════╦═══╗
║ author_id            ║ name                      ║ z ║
╠══════════════════════╬═══════════════════════════╬═══╣
║                   14 ║ a	b	c	d  ║   ║
║                   15 ║ aoeu                     ↵║   ║
║                      ║ test                     ↵║   ║
║                      ║                           ║   ║
║                    2 ║ 袈	袈		袈 ║   ║
╚══════════════════════╩═══════════════════════════╩═══╝
(3 rows)
```

The [Go Reference][goref-tblfmt] has the full API.

## Differences from `psql`

`tblfmt` writes the same output as `psql`. The differences below are
deliberate. If you find a different difference, report it as a bug.

### Trailing space on the last column

`tblfmt` pads the last column of a table that has a border, so that every
line of the table has the same width. `psql` pads the header, but it does not
pad the data rows. As a result, the right edge of a `psql` table is not
straight.

For `select 42 as n, 'a'::text as t union all select 7, 'bb';`, with trailing
spaces written as `·` and the width of each line at the right:

```text
psql 18.6                 tblfmt
 n  | t  ·         9       n  | t  ·         9
----+----          9      ----+----          9
 42 | a            7       42 | a ·          9
  7 | bb           8        7 | bb ·         9
```

`tblfmt` does not follow `psql` here, for these reasons:

1. A table that has lines of different widths is difficult to select in a
   terminal, to compare with `diff`, and to lay out in a program that measures
   the block.
2. The uneven edge gives no information.
3. `psql` pads its own header, so the data rows do not agree with it.

See D28 in [docs/PLAN.md](docs/PLAN.md). For a left aligned last column, the
code does not match this section yet. See open question 1 there.

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

[go-project]: https://golang.org/project
[usql]: https://github.com/xo/usql
