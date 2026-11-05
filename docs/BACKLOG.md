# Backlog

The known faults, the TODO and FIXME comments, and the deferred work in
tblfmt. A decision about any of them goes in [PLAN.md](PLAN.md). When an item
is done, remove it here and name it in the commit message.

B1 to B33 were collected on 2026-09-27. An item names the function that holds
the code, because a line number moves.

## Faults in the encoders

There is no known fault in the encoders.

## TODO and FIXME comments

### B21. `fmt.go`, in `format`: time formatting.

- Text: `// TODO: change time to v.AppendFormat() + pool`.
- Plain description: format a time into a pooled buffer with `AppendFormat`,
  and not with `Format`, to save allocations.

### B22. `fmt.go`, in `format`: numeric times.

- Text: `// TODO: use strconv.Format* for numeric times`.
- Plain description: use `strconv` for a time format that is only a number.
  The intent is not clear from the code.

### B23. `fmt.go`, in `format`: buffer pool.

- Text: `// TODO: use a pool`.
- Plain description: reuse value buffers from a pool in `format`.

### B25. `fmt.go`, in `encode`: buffer pool for encoded values.

- Text: `// TODO: use a pool`.
- Plain description: reuse buffers in `encode` as well.

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
  release, delete `emoji.go`, and delete the four calls to `emojiWidth`: in
  `FormatBytes`, in `hasWide` and `textWidth` in `encode.go`, and in
  `displayWidth` in `opts_test.go`. Then run
  `go test -run TestFromMapFormats -update .`. The golden files must not
  change.
- Status: on 2026-10-27, Ken opened mattn/go-runewidth#123 with the fix, at
  https://github.com/mattn/go-runewidth/pull/123. It is open. See D39 in
  [PLAN.md](PLAN.md).
