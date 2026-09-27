# Contributing to tblfmt

`tblfmt` writes database result sets as text, and [usql][usql] is its main
consumer. `AGENTS.md` holds the full rules for a coding agent. This file is
the short form for a person.

## What a change must do

An encoder writes what `psql` writes. Before you change the output, run the
same query in `psql` with the same `\pset` values. Write the command, its
output and the `psql` version in a comment beside the test. If the change is a
deliberate difference from `psql`, it needs a decision in
[`docs/PLAN.md`](docs/PLAN.md) and a line in "Differences from `psql`" in
[`README.md`](README.md).

`tblfmt` makes no promise of backward compatibility. A change can rename or
remove an exported identifier when that makes the code better. `usql` imports
`tblfmt`, so name each such change in the pull request.

## Before you open a pull request

Run these in the repository root. Each must pass:

```sh
gofmt -l . && go vet ./... && go test -race -count=1 ./...
golangci-lint run --new-from-rev=HEAD ./...
```

If a golden file test fails, compare `testdata/<set>/NN.gld` with the
`NN.out` file that the test wrote.

## Writing

Write documents, code comments, error messages and commit messages in plain
English. Use short sentences and the active voice. Do not use contractions,
semicolons or em dashes. Use "must" and "can", not "should" and "may". The
`simple-english` skill in `.agents/skills` holds the full rules.

## Agent skills

The repository carries two agent skills. A skill is a set of instructions that
a coding agent loads for a task. `simple-english` sets how prose is written,
and `go-pedantry` sets how Go is written.

`skills-lock.json` names the source of each skill. The `skills` command from
npm writes that file. It writes each skill into two folders. Codex and other
agents read `.agents/skills/<name>`, and Claude Code reads
`.claude/skills/<name>`.

To add a skill or to update one, run this command in the repository root. The
example updates `simple-english`. Get the source of the other skills from
`skills-lock.json`:

```sh
npx skills@1.7.0 add AminBlg/SimpleEnglish --skill simple-english --agent codex claude-code --copy -y
```

Keep `--copy`. Without it, the command writes `.claude/skills/<name>` as a
symbolic link. A Windows checkout writes a symbolic link as a text file, and
Claude Code then loads no skill and reports nothing. `TestSkillsAreCopies`
fails on a link, and it fails when the two folders differ.

`.claude/settings.local.json` holds the Claude Code permissions of one person.
The root `.gitignore` ignores it.

[usql]: https://github.com/xo/usql
