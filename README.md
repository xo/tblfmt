# About tblfmt [![Go Reference][goref-tblfmt-status]][goref-tblfmt]

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
all display variants [available in `usql`][usql].

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
// _example/main.go
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
	result, err := db.Query("select * from authors")
	if err != nil {
		log.Fatal(err)
	}
	defer result.Close()
	enc, err := tblfmt.NewTableEncoder(result,
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

## Testing

Run using standard `go test`:

```sh
$ go test -v
```

A few environment variables control how testing is done:

- `PSQL_CONN=<connection>` - specify local connection string to use with the `psql` tool for compatibility testing
- `DETERMINISTIC=1` - use a deterministic random seed for the big "random" test

Used like the following:

```sh
# retrieve the latest postgres docker image
$ docker pull postgres:latest

# run a postgres database with docker
$ docker run --rm -d -p 127.0.0.1:5432:5432 -e 'POSTGRES_PASSWORD=P4ssw0rd' --name postgres postgres

# do determininstic test and using psql:
$ export DETERMINISTIC=1 PSQL_CONN=postgres://postgres:P4ssw0rd@localhost/?sslmode=disable
$ go test -v
```

## TODO

1. add center alignment output
2. allow user to override alignment
3. Column encoder

[go-project]: https://golang.org/project
[goref-tblfmt]: https://pkg.go.dev/github.com/xo/tblfmt
[goref-tblfmt-status]: https://pkg.go.dev/badge/github.com/xo/tblfmt.svg
[usql]: https://github.com/xo/usql
