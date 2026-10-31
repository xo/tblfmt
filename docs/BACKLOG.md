# Backlog

The known faults, the TODO and FIXME comments, and the deferred work in
tblfmt. A decision about any of them goes in [PLAN.md](PLAN.md). When an item
is done, remove it here and name it in the commit message.

B1 to B33 were collected on 2026-09-27. Their line numbers are at `04af38a`,
and they can move. "Probe" means that a small program ran the encoder and
showed the fault. "Read" means that the fault comes from reading the code, and
no program ran it.

## Faults in the encoders

### B6. The expanded encoder ignores the formatter options.

- Where: `encode.go:568`, in `NewExpandedEncoder`.
- Text: `t.formatter = NewEscapeFormatter()`.
- Plain description: the constructor applies the options and then replaces
  the formatter. So `numericlocale`, `time` and `timezone` have no effect in
  expanded mode.
- Probe: with `expanded: on`, `numericlocale: on` gives `1234567` and not
  `1,234,567`, and `time: 2006` still gives an RFC 3339 time.
- Origin: `076ccde` (2021-02-04).

### B9. The summary line ends with the platform newline and not the encoder's newline.

- Where: `encode.go:1390`, in `summarize`.
- Text: `if _, err := w.Write(newline); err != nil {`.
- Plain description: `summarize` writes the package `newline`. So with
  `WithNewline`, the rows of a table end with the chosen newline, and the
  summary line ends with `\n`, or `\r\n` on Windows.
- Origin: `b4e589b` (2024-04-01). The method before it wrote `enc.newline`.
  Probe. For the unaligned format with `recordsep`, the output is `aX1X(1
  row)` and a newline, which can match psql. Check psql before changing it.

### B17. FromMap changes the map that the caller passes in.

- Where: `opts.go:133`, `opts.go:164` and `opts.go:187`.
- Text: `opts["footer"] = "off"`.
- Plain description: with `tuples_only: on`, `FromMap` writes `footer: off`
  into the caller's map. A caller that reuses the map, such as a `\pset` store,
  keeps the footer off after it turns `tuples_only` off.
- Probe.

## TODO and FIXME comments

### B21. `fmt.go:143`: time formatting.

- Text: `// TODO: change time to v.AppendFormat() + pool`.
- Plain description: format a time into a pooled buffer with `AppendFormat`,
  and not with `Format`, to save allocations.

### B22. `fmt.go:144`: numeric times.

- Text: `// TODO: use strconv.Format* for numeric times`.
- Plain description: use `strconv` for a time format that is only a number.
  The intent is not clear from the code.

### B23. `fmt.go:145`: buffer pool.

- Text: `// TODO: use pool`.
- Plain description: reuse value buffers from a pool in `format`.

### B25. `fmt.go:298`: buffer pool for encoded values.

- Text: `// TODO: pool`.
- Plain description: reuse buffers in `encode` as well.

## Tests, tooling and dead code

### B33. A golden test option cannot hold a colon.

- Where: `opts_test.go:209`, in `loadOpts`.
- Text: `v := strings.Split(line, ":")` and then `if len(v) != 2 {`.
- Plain description: an option value such as a time format `15:04` fails the
  test with "missing : in line".
- Read.

## Deferred work

### B34. Make the encoders faster, and allocate less.

- Source: PR 18 (nineinchnick, 2021), closed on 2026-09-27. See D33 in
  [PLAN.md](PLAN.md). usql issue 144 asked for this work.
- Measure with `BenchmarkEncode` in `bench_test.go`, before and after each
  change, as "Tests" in `AGENTS.md` says. D59 added it.
- Measured on 2026-10-31: 10,000 rows of an integer, 100 bytes of text and a
  float cost about 0.9 µs and 18 allocations for each row in the aligned
  format. `FormatBytes` took more than half of the time. It handles one rune
  at a time, and it converts each run of text to a string to measure it. A run
  of printable ASCII can be copied at once, and its width is its length. A
  number goes through `fmt.Sprintf` and is copied again. Each value allocates
  its tab positions even when it has no tab.
- The latex format is the slowest: 2.3 s for 1,000,000 rows. `writeLaTeX`
  looks up each byte in a map, and writes each escape with `fmt.Fprint`.
- An idea from PR 18 keeps the API: reuse the scan buffer between batches.
- Do not pool a `*Value` while a batch or a template still holds it.

### B36. Comments that do not match the code.

- Plain description: the comment pass of D35 found these comments. Each one
  is left as it was, because the fix needs a look at the code first.
- `fmt.go`, the doc comments of `EscapeFormatter`, its `encoder` field and
  `NewEscapeFormatter`: they say that only `map[string]any` and `[]any`
  values go to the marshal func. Every value that `format` does not handle
  goes there, for example a struct, a `[]string` and a `driver.Valuer` whose
  `Value` returns an error.
- `fmt.go`, `Value.Quoted`: the comment says that a value is quoted when it
  holds a space or a character that does not print. In raw mode the code
  quotes a space, the separator and the quote, and it does not test for a
  character that does not print.
- `fmt.go`, `deref`: the comment says that it dereferences a pointer to an
  interface. It dereferences any pointer that is not nil, by one level.
- `opts.go`, the FIXME in the summary option: it says that all of these
  encoders need a summary. The unaligned encoder already has one. See B26.

### B38. go-runewidth counts an emoji variation sequence as 1 column.

- Where: `graphemeWidth` in go-runewidth v0.0.31, which `FormatBytes` calls
  through `runewidth.StringWidth`.
- Plain description: a character that is text by default, followed by U+FE0F,
  is one emoji 2 columns wide in a current terminal. Examples are U+2764
  U+FE0F and the keycap U+0031 U+FE0F U+20E3. go-runewidth counts each of them
  as 1 column, so tblfmt pads the cell by 1 column too many.
- Workaround: since D41, `emojiWidth` in `emoji.go` adds the missing column.
  It does nothing when go-runewidth counts the sequence as 2.
- To remove the workaround after go-runewidth is fixed: move to the fixed
  release, delete `emoji.go`, and delete the two calls to `emojiWidth`, in
  `FormatBytes` and in `displayWidth` in `opts_test.go`. Then run
  `go test -run TestFromMapFormats -update .`. The golden files must not
  change.
- Status: on 2026-10-27, Ken opened mattn/go-runewidth#123 with the fix, at
  https://github.com/mattn/go-runewidth/pull/123. It is open. See D39 in
  [PLAN.md](PLAN.md).
