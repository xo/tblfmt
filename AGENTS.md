# tblfmt

`tblfmt` writes database result sets as text. It has encoders for aligned
tables, expanded records, unaligned text, CSV, JSON, HTML, AsciiDoc and
MySQL-style vertical output. It also has a crosstab view. [usql][usql] is its
main consumer, and `usql` pins a tagged release.

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
   comment beside the test, with the `psql` version. A deliberate difference
   from `psql` is a decision. It goes in `docs/PLAN.md` and in "Differences
   from `psql`" in `README.md`. Any other difference is a fault.
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
   [docs/PLAN.md](docs/PLAN.md).

## Layout

- `tblfmt.go` holds the `Encoder` and `ResultSet` interfaces, the `Encode*`
  funcs, and the error constants.
- `encode.go` holds every encoder: `TableEncoder`, `ExpandedEncoder`,
  `JSONEncoder`, `UnalignedEncoder` (and CSV) and `TemplateEncoder`.
- `fmt.go` holds the `Formatter` interface, `EscapeFormatter` and `Value`.
- `opts.go` holds the `Option` funcs and `FromMap`, which reads `psql` `\pset`
  names.
- `style.go` holds the line styles and the summary.
- `template.go` holds the HTML, AsciiDoc and vertical writers. They are Go
  funcs, and each keeps the template text that it replaced in a comment.
- `view.go` holds the crosstab view, which follows `psql` `\crosstabview`.
- `internal/` holds the result sets that the tests use.
- `testdata/` holds the golden files and the programs that write them.

## Tests

`TestFromMapFormats` reads each `testdata/<set>/NN.in` for the `\pset`
values, encodes one of the result sets in `internal`, and compares the output
with `NN.gld`. It writes what it got to `NN.out`, so `diff NN.gld NN.out` shows
a failure. `.gitattributes` marks both files as binary, because some of them
hold a carriage return on purpose.

A test for a `psql` behavior holds the `psql` command and output in its
comment. `TestEncodeNullAlign` and `TestEncodeNoColumns` are examples.

`TestSkillsAreCopies` makes sure that each agent skill is an ordinary folder in
`.agents/skills` and in `.claude/skills`, and that the two copies agree.

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

`.golangci.yml` enables every linter and disables a list. Read a finding as a
question, not as a task. If a linter makes idiomatic Go worse, disable it in
`.golangci.yml` and write the reason beside it. Only a real defect gets a code
change.

Main has many findings today. A new linter version renamed `exhaustruct` to
`exhaustruct_v5`, so the disable line no longer matches, and the new name
gives most of the findings. `docs/BACKLOG.md` holds this. To see only the
findings in your change, run:

```sh
golangci-lint run --new-from-rev=HEAD ./...
```

## Before you stage

Standing rule 1 applies: stage the change with `git add`, and give Ken a
proposed commit message. Do not tag. All of these must pass first:

```sh
gofmt -l . && go vet ./... && go test -race -count=1 ./...
golangci-lint run --new-from-rev=HEAD ./...
```

`gofmt -l .` must print nothing. `_example/mysqltest.go` is the one file that
it prints on main today.

CI runs `go test -v ./...` on `ubuntu-latest` with the stable Go release.

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
