# About tblfmt

Package `tblfmt` provides streaming table encoders for result sets (ie, from a
database), creating tables like the following:

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

Additionally, there are standard encoders for JSON, CSV, HTML, unaligned and
other display variants [supported by `usql`][usql].

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

Install in the usual [Go][go-project] fashion:

```sh
$ go get -u github.com/xo/tblfmt
```

## Using

`tblfmt` was designed for use by [`usql`][usql] and Go's native `database/sql`
types, but will handle any type with the following interface:

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

`tblfmt` can be used similar to the following:

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

Which can produce output like the following:

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

Please see the [Go Reference][goref-tblfmt] for the full API.

## Differences from `psql`

`tblfmt` follows `psql`'s output closely. The differences below are deliberate;
anything else is a bug worth reporting.

### Trailing space on the last column

`tblfmt` pads the last column of a bordered table, so that every line of a
table is the same width. `psql` pads the header but trims the data rows,
leaving them ragged.

For `select 42 as n, 'a'::text as t union all select 7, 'bb';`, with trailing
spaces written as `·` and the width of each line at the right:

```text
psql 18.6                 tblfmt
 n  | t  ·         9       n  | t  ·         9
----+----          9      ----+----          9
 42 | a            7       42 | a ·          9
  7 | bb           8        7 | bb ·         9
```

`psql` is wrong here. A table whose lines do not share a width is awkward to
select from a terminal, to diff, and to lay out in anything that measures the
block, and the ragged edge carries no information. It is also inconsistent with
`psql`'s own header, which is padded. `tblfmt` will not follow it.

## Testing

Run using standard `go test`:

```sh
$ go test -v
```

[go-project]: https://golang.org/project
[usql]: https://github.com/xo/usql
