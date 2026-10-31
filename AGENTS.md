# tblfmt

`tblfmt` writes database result sets as text. It writes each format of `psql`:
aligned, wrapped, unaligned, csv, html, asciidoc, latex, latex-longtable and
troff-ms, and expanded records. It also writes JSON, the vertical output that
the `mysql` client writes for `\G`, and a plain table. It also has a crosstab
view. [usql][usql] is its main consumer, and `usql` pins a tagged release.

`tblfmt` does not connect to a database and does not import a driver. The
caller gives it a `ResultSet`, which `*sql.Rows` satisfies.

## Standing rules

These hold in every `xo` repository, for every coding agent. dbmeta D110
records them.

1. Stage changes for review. Commit and push only when Ken says so.
2. Load the `simple-english` skill before you write any text that a person
   reads: a document, a code comment, an error message or a commit message.
   Follow it for that text.
3. Load the `go-pedantry` skill before you write or review Go code. Follow it
   where it does not conflict with a rule in this file. A rule here wins.

`CLAUDE.md` holds one line that imports this file, so that Claude Code and
every other agent read the same rules. Edit this file, not that one.

## Which document to read

| If you are | Read |
| --- | --- |
| asking why something is the way it is | the table at the top of [docs/PLAN.md](docs/PLAN.md) |
| looking for work that is known and not done | [docs/BACKLOG.md](docs/BACKLOG.md) |
| changing what an encoder writes | hard rule 1, then "Differences from `psql`" in [README.md](README.md) |
| changing an exported type, func or interface | hard rule 3 |
| adding or changing a test | "Tests" below |
| answering a lint finding | "Linting" below |
| writing a document, a code comment, an error message or a commit message | the `simple-english` skill. Load it first. See "Writing documentation" below |
| adding or updating an agent skill | "Agent skills" in [CONTRIBUTING.md](CONTRIBUTING.md) |

`CONTRIBUTING.md` holds the same rules for a person, and it is shorter.

A document that is not in that table does not exist. If you cannot find where
something is written down, it is not written down. Ask Ken. Do not decide it
yourself, and do not write it as though it were settled. The open questions
are at the end of `docs/PLAN.md`.

## Hard rules

1. `psql` is the reference. For the same result set and the same `\pset`
   values, an encoder writes the bytes that `psql` writes. Measure `psql`
   before you change an encoder, and write the command and its output in a
   comment beside the test, with the `psql` version. Run `TestPsql` (see
   "Tests") before you stage. A deliberate difference from `psql` is a
   decision. It goes in `docs/PLAN.md` and in "Differences from `psql`" in
   `README.md`. Any other difference is a fault.
2. `psql` writes one blank line after an aligned table, and `tblfmt` does not.
   `usql` writes that line itself. If `tblfmt` writes it, `usql` writes two.
3. `tblfmt` makes no promise of backward compatibility, and no `xo` project
   does. Rename or remove an exported identifier when that makes the code
   better. Remove an exported identifier that has no use. Do not mark it
   `Deprecated:`. Before you change one, find its uses in `usql`, and name each
   use in the commit message, so that `usql` can follow the change. See D37.
4. Keep the number of rows in memory small. The JSON and unaligned encoders
   write each row when they read it. The table and expanded encoders keep one
   batch of rows to calculate the column widths, and `WithCount` sets the
   size of the batch. Zero, the default, makes the batch the whole result
   set. The template encoder reads every row before it runs the template.
   Do not make a new encoder read a whole result set into memory without a
   reason that `docs/PLAN.md` records.
5. A result set that has no columns is valid, and so is a result set that has
   no rows. No encoder returns an error for either. `EncodeAll` goes on to the
   next result set.
6. Do not add a dependency without asking Ken. The module requires
   `go-runewidth`, `golang.org/x/text` and `consolesize-go`. Keep each one at
   the version that `usql` requires, or at a higher version. See
   [docs/PLAN.md](docs/PLAN.md). The `test` module is separate, and it
   requires the packages that its benchmark compares with tblfmt (D59).

## Layout

- `tblfmt.go` holds the `Encoder` and `ResultSet` interfaces, the `Encode*`
  funcs, and the error constants.
- `encode.go` holds every encoder: `TableEncoder`, `ExpandedEncoder`,
  `JSONEncoder`, `UnalignedEncoder` (and CSV) and `TemplateEncoder`.
- `fmt.go` holds the `Formatter` interface, `EscapeFormatter` and `Value`.
- `emoji.go` holds a temporary workaround for a width that go-runewidth
  counts wrong. See D41 in `docs/PLAN.md`.
- `opts.go` holds the `Option` funcs and `FromMap`, which reads `psql` `\pset`
  names.
- `style.go` holds the line styles and the summary.
- `template.go` holds the HTML, AsciiDoc, LaTeX, LaTeX longtable and troff
  writers, which follow the print code of `psql`, and the vertical writer,
  which follows the `mysql` client. They are Go funcs (D20).
- `view.go` holds the crosstab view, which follows `psql` `\crosstabview`.
- `internal/` holds the result sets that the tests use.
- `testdata/` holds the golden files, and the program that writes the
  crosstab file.
- `bench_test.go` holds `BenchmarkEncode`, which measures each format.
- `test/` is a separate module with its own `go.mod` and `.golangci.yml`. Its
  `BenchmarkWrite` compares tblfmt with other packages that write tables, and
  `TestWrite` makes sure that each package still writes the table.
  `TestPsql` compares tblfmt with `psql` (D65).

## Tests

`TestFromMapFormats` reads each `testdata/<set>/NN.in` for the `\pset`
values, encodes one of the result sets in `internal`, and compares the output
with `NN.gld`. It writes what it got to `NN.out`, so `diff NN.gld NN.out` shows
a failure. `.gitattributes` marks both files as binary, because some of them
hold a carriage return on purpose.

Each set has an emoji version, such as `tiny-emoji`, with the same `.in`
files. To write the golden files again after a change to the output, run
`go test -run TestFromMapFormats -update .`. Then read the diff of the `.gld`
files before you stage them. A golden file holds the output of `tblfmt`
itself, so it cannot show that a column is padded wrong.
`TestGoldenAlignment` makes sure that each line of a table with border 2 has
the same display width.

A test for a `psql` behavior holds the `psql` command and output in its
comment. `TestEncodeNullAlign` and `TestEncodeNoColumns` are examples.

`TestSkillsAreCopies` makes sure that each agent skill is an ordinary folder in
`.agents/skills` and in `.claude/skills`, and that the two copies agree.

`TestPsql` in `test/psql_test.go` runs each of about 1900 cases with `psql` 18
and with tblfmt, and compares the bytes. Run it before you stage a change to
what an encoder writes. It needs a PostgreSQL server from `dbrun` in a dbmeta
checkout. Read the rules in `docs/DBRUN.md` of dbmeta first: run `dbrun
status`, share a server that runs, and stop only a server that you started.

```sh
cd ../dbmeta/test && DBMETA_OWNER_NAME=tblfmt go run ./cmd/dbrun start postgres-18
export DBMETA_POSTGRES=$(go run ./cmd/dbrun dsn --json postgres-18 | jq -r '.[0].url')
cd ../../tblfmt/test && go test -count=1 -run TestPsql ./...
```

To add a case, add a SELECT statement or a set of `\pset` values to
`psqlGroups`. A case that a decision makes different from `psql` stays out of
the test, and its comment names the decision.

`BenchmarkEncode` in `bench_test.go` measures each format at 10,000 and
1,000,000 rows. If a change can make an encoder slower, run it before and
after the change, and give Ken both results:

```sh
go test -run '^$' -bench BenchmarkEncode -benchmem -count 10
```

Compare the two results with `benchstat`. One run of each case takes about a
minute, so `-count 10` takes about 10 minutes.

## Go conventions

Match the code around you. Comments are short, lower case, and they say why.

Errors are constants of `type Error string`, declared in `tblfmt.go`, and
named with an `Err` prefix. Do not use `errors.New`. Compare errors with
`errors.Is`.

Write every JSON value with `encoding/json/v2` and `encoding/json/jsontext`.
Do not import `encoding/json`.

A receiver name is one to three letters, and it is the same on every method of
a type. The encoders use `enc`.

## Linting

`.golangci.yml` enables every linter and disables a list, as dburl, dbmeta
and dbimp do. See D49. Read a finding as a question, not as a task. If a
linter makes idiomatic Go worse, disable it in `.golangci.yml` and write the
reason beside it. Only a real defect gets a code change.

CI pins golangci-lint v2.14.0. The module has no finding with that version.
Run the linter on the whole module:

```sh
golangci-lint run ./...
```

## Before you stage

Standing rule 1 applies: stage the change with `git add`, and give Ken a
proposed commit message. Do not tag. All of these must pass first:

```sh
gofmt -l . && go vet ./... && go test -race -count=1 ./...
golangci-lint run ./...
cd test && go vet ./... && go test -count=1 ./... && golangci-lint run ./...
```

`gofmt -l .` must print nothing.

CI runs `go test -v ./...` in the root and in `test`, on `ubuntu-latest` with
the stable Go release. A second job runs golangci-lint in both modules, and
`gofmt -l .`.

## Writing documentation

A new document goes in `docs/`. Only `README.md`, `AGENTS.md`, `CLAUDE.md`
and `CONTRIBUTING.md` belong in the repository root. Add a new document to the
table at the top of this file.

A decision goes in `docs/PLAN.md`, and nowhere else. Put its status in the
heading after the title: `Decided`, or `Amends D3`, or `Superseded by D9`.
Add it to the index at the top. Deferred work and known faults go in
`docs/BACKLOG.md`.

Standing rule 2 applies to every such text. In brief: short sentences, the active voice, no contractions,
no semicolons and no em dashes. Use `can`, `will` and `must`, never `should`,
`may` or `might`. Put the condition before the command. Ken asked for this.

[usql]: https://github.com/xo/usql
