# Backlog

The known faults, the TODO and FIXME comments, and the deferred work in
tblfmt. A decision about any of them goes in [PLAN.md](PLAN.md). When an item
is done, remove it here and name it in the commit message.

B1 to B33 were collected on 2026-09-27. Their line numbers are at `04af38a`,
and they can move. "Probe" means that a small program ran the encoder and
showed the fault. "Read" means that the fault comes from reading the code, and
no program ran it.

## Faults in the encoders

### B1. The table encoder writes no header for a result set that has columns and no rows.

- Where: `encode.go:204-208` writes the header, inside the batch loop. The loop
  ends at `encode.go:162-164` when a batch is empty, before the header.
- Text: `// print header if not already done`, then `if !wroteHeader {`.
- Plain description: for a query such as `select 1 as a where false`, tblfmt
  writes only `(0 rows)`, and psql writes the header, the divider and
  `(0 rows)`.
- Found: by the user. Probe: a result set with the columns `a` and `b` and no
  rows gives only `(0 rows)`, at border 1 and at border 2.
- Origin: `4aaadae` (2019-01-30) moved the header into the batch loop. The
  first commit, `ff80f92`, always wrote the header.
- D31 writes the lines for a result set that has no columns and no rows, in
  a block after the loop. That block runs only when `clen == 0`, and its
  comment names this item. Open question 2 in [PLAN.md](PLAN.md) asks Ken
  whether to fix this fault.

### B2. The expanded encoder writes a custom summary before the buffered records.

- Where: `encode.go:640`, in `ExpandedEncoder.Encode`.
- Text: `if err := summarize(w, enc.summary, enc.scanCount); err != nil {`
- Plain description: the call writes to `w`, the caller's writer, and the
  records are still in the `bufio.Writer` `enc.w`, so the summary comes out
  first. It must be `summarize(enc.w, ...)`, as in `TableEncoder.Encode` at
  `encode.go:218`.
- Found: by the user. Probe: with `WithSummary` on an expanded encoder, the
  output starts with the summary line and then `-[ RECORD 1 ]-`.
- Origin: `b4e589b` (2024-04-01). Before it, the method `enc.summarize` wrote
  to `enc.w` and ignored its argument.
- It also bypasses the pager when one is running.

### B3. The flush after every 1000 rows never runs.

- Where: `encode.go:236` in `TableEncoder.encodeVals` and `encode.go:658` in
  `ExpandedEncoder.encodeVals`.
- Text: `if i+1%1000 == 0 {` with the comment `// check error every 1k rows`.
- Plain description: `%` binds tighter than `+`, so the test is `i+1 == 0`,
  which is never true. A write error, such as a closed pager, is found only at
  the final flush.
- Origin: `184dbaa` (2021-02-05). Read.

### B4. With border 2 and a count, the table encoder draws an end border after every batch.

- Where: `encode.go:212-215`, inside the batch loop.
- Text: `// draw end border`, then `if enc.border >= 2 {`.
- Plain description: `WithCount(2)` with five rows and border 2 draws a
  bottom line after rows 2, 4 and 5, so one table looks like three.
- Origin: `184dbaa` (2021-02-05) moved the end border into the loop. Probe.

### B5. A pager that starts on a later batch loses the output before it.

- Where: `encode.go:195-203`, and the same pattern at `encode.go:181-189` and
  `encode.go:618-626`.
- Text: `enc.w = bufio.NewWriterSize(cmdBuf, 2048)`.
- Plain description: the pager check runs for each batch. When a later batch
  starts the pager, `enc.w` is replaced without a flush, so the header and the
  earlier rows in the old buffer are never written.
- Probe: `WithCount(1)`, `WithPager("cat")` and `WithMinPagerHeight(6)`, with
  a short first row and a tall second row. The output has only the second row
  and the summary.

### B6. The expanded encoder ignores the formatter options.

- Where: `encode.go:568`, in `NewExpandedEncoder`.
- Text: `t.formatter = NewEscapeFormatter()`.
- Plain description: the constructor applies the options and then replaces
  the formatter. So `numericlocale`, `time` and `timezone` have no effect in
  expanded mode.
- Probe: with `expanded: on`, `numericlocale: on` gives `1234567` and not
  `1,234,567`, and `time: 2006` still gives an RFC 3339 time.
- Origin: `076ccde` (2021-02-04).

### B7. Automatic expanded output keeps the row count footer, and expanded output does not.

- Where: `encode.go:167-193` switches to expanded output and keeps the table
  summary. `encode.go:569-571` drops the default summary for `expanded: on`.
- Plain description: the same data prints `(1 row)` after the records with
  `expanded: auto`, and prints no footer with `expanded: on`.
- Probe. Which one is right is not recorded. See open question 3 in
  [PLAN.md](PLAN.md).

### B8. The table height counts the mid divider in the wrong case.

- Where: `encode.go:432-435`, in `TableEncoder.tableHeight`.
- Text: `// mid divider`, then `if enc.inline {`.
- Plain description: `header` draws the mid divider when `inline` is false,
  and `tableHeight` counts it when `inline` is true. The height is one line off
  in both cases, which moves the pager threshold by one line.
- Read.

### B9. The summary line ends with the platform newline and not the encoder's newline.

- Where: `encode.go:1390`, in `summarize`.
- Text: `if _, err := w.Write(newline); err != nil {`.
- Plain description: `summarize` writes the package `newline`. So with
  `WithNewline`, the rows of a table end with the chosen newline, and the
  summary line ends with `\n`, or `\r\n` on Windows.
- Origin: `b4e589b` (2024-04-01). The method before it wrote `enc.newline`.
  Probe. For the unaligned format with `recordsep`, the output is `aX1X(1
  row)` and a newline, which can match psql. Check psql before changing it.

### B10. The README describes last column padding that the code does not do.

- Where: `README.md:120-140`, and `encode.go:491-494` in `TableEncoder.row`.
- Text: the README says that tblfmt "pads the last column of a bordered
  table". The code comment says `// no padding for last cell if no border and
  aligned left`.
- Plain description: at border 1, a left aligned last column is not padded.
  For the README example, the rows are 8 and 9 columns wide and the header is
  9, where the README shows 9 for every line. Either the README or the code
  must change. See D14 and D28, and open question 1, in [PLAN.md](PLAN.md).
- Probe.

### B11. The html encoder writes the table attributes with no space before them.

- Where: `template.go:36`, in `WriteHTMLTo`.
- Text: `fmt.Fprint(w, "", tpl.Attributes)`.
- Plain description: `fmt.Fprint` adds a space only between two operands that
  are not strings, so `tableattr` of `border="1"` gives `<tableborder="1">`.
  psql writes a space. The old template, `<table{{ .Attributes | attr }}>`,
  had the same fault.
- Probe.

### B12. The html encoder does not escape the title.

- Where: `template.go:38`, in `WriteHTMLTo`.
- Text: `fmt.Fprintf(w, ">\n  <caption>%s</caption>\n ...", tpl.Title)`.
- Plain description: a title of `<b>x</b>` is written as markup. `5af5ff3`
  escapes every header and cell, but not the caption. Before `c8da282`,
  `html/template` escaped it.
- Probe.

### B13. The asciidoc encoder writes the title without its leading period.

- Where: `template.go:63-64`, in `WriteAsciidocTo`.
- Text: `fmt.Fprintf(w, "\n%s", tpl.Title.String())`.
- Plain description: asciidoc marks a block title with a leading `.`. The old
  template, kept in the comment above, wrote `.{{ .Title }}`, and psql writes
  `.title` too. The output now has a bare `T` line.
- Origin: `c8da282` and `fa8cb67` (2026-03-29). Probe.

### B14. The template encoders ignore `tuples_only`.

- Where: `template.go:10-17` (the `SkipHeader` field), `WriteHTMLTo`,
  `WriteAsciidocTo` and `WriteVerticalTo`, and `opts.go:140-149` in `FromMap`.
- Plain description: `FromMap` passes no `WithSkipHeader` to the template
  formats, and none of the three writers reads `tpl.SkipHeader`. With
  `format: html` and `tuples_only: on`, the header row is still written.
- Probe.

### B15. FromMap accepts latex, latex-longtable and troff-ms, and then fails.

- Where: `opts.go:140` in `FromMap`, and `opts.go:591-607` in `WithTemplate`.
- Text: `case "html", "asciidoc", "latex", "latex-longtable", "troff-ms",
  "vertical":`.
- Plain description: `FromMap` returns the template builder for these three
  formats, and `WithTemplate` has no func for them, so the builder returns
  `ErrInvalidTemplate`.
- Deferred idea: the first README listed "finish template implementations for
  HTML, LaTeX, etc." as a TODO. `2e553f9` (2021-04-20) removed the TODO list,
  and the LaTeX and troff formats were never written.
- Probe.

### B16. The psql `wrapped` format is not supported.

- Where: `opts.go:84`.
- Text: `// unaligned, aligned, wrapped, html, asciidoc, latex, latex-longtable,
  troff-ms, json, csv`.
- Plain description: the comment lists `wrapped`, and `FromMap` returns
  `ErrInvalidFormat` for it. The format test of `faf1a88` had `"wrapped"`
  commented out.
- Read.

### B17. FromMap changes the map that the caller passes in.

- Where: `opts.go:133`, `opts.go:164` and `opts.go:187`.
- Text: `opts["footer"] = "off"`.
- Plain description: with `tuples_only: on`, `FromMap` writes `footer: off`
  into the caller's map. A caller that reuses the map, such as a `\pset` store,
  keeps the footer off after it turns `tuples_only` off.
- Probe.

### B18. The JSON encoder writes several result sets as arrays joined by a comma.

- Where: `encode.go:964-984`, in `JSONEncoder.EncodeAll`.
- Plain description: two result sets give `[...],` and then `[...]`, which is
  not one JSON document, so a JSON parser rejects it. The staged
  `TestEncodeAllNoColumnsFirst` expects this form, so it can be deliberate.
- Origin: `1c1f094` (2019-01-28). Read.

### B19. With a count, a later wide value is wider than the header.

- Where: `encode.go:155-216`, in `TableEncoder.Encode`.
- Plain description: the header is sized from the first batch, and a later
  batch widens the columns under it. This follows from D2. It is a known cost
  of streaming, not a slip.
- Probe: `WithCount(2)` and the values 1, 2 and 33333 give a 1 column wide
  header over a 5 column wide value.

## TODO and FIXME comments

### B20. `encode.go:108`: the line style check.

- Text: `// TODO: this check should be removed`.
- Plain description: `NewTableEncoder` rejects a line style that holds a rune
  wider than one column, and the TODO says that this check is to go.

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

### B24. `fmt.go:146`: escaped runes.

- Text: `// TODO: allow configurable runes that can be escaped`.
- Plain description: let a caller choose which runes the formatter escapes.

### B25. `fmt.go:298`: buffer pool for encoded values.

- Text: `// TODO: pool`.
- Plain description: reuse buffers in `encode` as well.

### B26. `opts.go:371`: a summary for the JSON and template encoders.

- Text: `// FIXME: all of these should have a summary option as well ...`
- Plain description: `WithSummary` does nothing for the JSON and template
  encoders.

### B27. `opts.go:496-498`: minimum widths for the unaligned encoder.

- Text: `// FIXME: unaligned encoder should be able to support minimum column
  widths`.
- Plain description: `WithWidths` does nothing for the unaligned encoder.

### B28. `opts.go:502-504`: minimum widths for the template encoder.

- Text: `// FIXME: template encoder should be able to support minimum column
  widths`.
- Plain description: `WithWidths` does nothing for the template encoder.

## Tests, tooling and dead code

### B29. The format generator still writes gzip files.

- Where: `testdata/genformats.go:73` and `testdata/genformats.go:83`.
- Text: `w := gzip.NewWriter(out)` and `os.WriteFile(name+".gz", ...)`.
- Plain description: `629b8a1` (2026-03-29) replaced the `.gz` files with the
  `NN.in` and `NN.gld` files, and the tests no longer read `.gz`. So nothing
  can regenerate the golden files. `a4a1779` removed the `go:generate` lines
  that ran the generator.

### B30. The psql comparison helpers are dead code.

- Where: `internal/internal.go:245-338`.
- Text: `func PsqlEncodeAll(...)` and `func PsqlEncode(...)`.
- Plain description: `162708e` (2021-03-30) removed the test that called them,
  and nothing else calls them. Either restore a psql comparison test or
  remove them.

### B31. golangci-lint has a config and does not run in CI.

- Where: `.golangci.yml` and `.github/workflows/test.yml`.
- Plain description: the workflow runs only `go test -v ./...`, so nothing
  checks the lint config that `2faed8b` added.

### B32. The golden tests can fail on Windows.

- Where: `tblfmt.go:291-297` and `.gitattributes`.
- Plain description: on Windows the encoders write `\r\n`, and the golden
  files hold `\n`. CI runs only on Linux, so nothing has tried it.
- Read. Not verified.

### B33. A golden test option cannot hold a colon.

- Where: `opts_test.go:209`, in `loadOpts`.
- Text: `v := strings.Split(line, ":")` and then `if len(v) != 2 {`.
- Plain description: an option value such as a time format `15:04` fails the
  test with "missing : in line".
- Read.

## Deferred work

### B34. Reduce the allocations of the table encoder, after a benchmark.

- Source: PR 18 (nineinchnick, 2021), closed on 2026-09-27. See D33 in
  [PLAN.md](PLAN.md). usql issue 144 asked for this work.
- Plain description: two ideas from PR 18 keep the API. Cache the row styles
  for each encoder, because `rowStyle` builds new byte slices for each row.
  Reuse the scan buffer between batches.
- First step: write a benchmark of memory use and allocations. Measure the
  aligned, JSON, csv and html formats, at 10,000 rows and at 1,000,000 rows,
  with a text column of 100 bytes. Measure with a count of 0 and of 1000.
- Do not pool a `*Value` while a batch or a template still holds it.

### B35. The lint configuration uses linter names that changed.

- Where: `.golangci.yml`.
- Plain description: the configuration disables `exhaustruct`, and the
  current golangci-lint reports the same linter as `exhaustruct_v5`. As a
  result, about 50 findings on main come back. golangci-lint also says that
  `gomodguard` is deprecated and that `gomodguard_v2` replaces it. See open
  question 8 in [PLAN.md](PLAN.md).

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
- `template.go`, the template text in the comments of `WriteAsciidocTo` and
  `WriteVerticalTo`: the comments no longer match the code. See B13.
- `opts.go`, the FIXME in the summary option: it says that all of these
  encoders need a summary. The unaligned encoder already has one. See B26.

### B37. `_example/mysqltest.go` is not gofmt clean.

- Where: `_example/mysqltest.go`.
- Plain description: `gofmt -l .` prints this file on main, so the check in
  `AGENTS.md` cannot print nothing until the file is formatted.
