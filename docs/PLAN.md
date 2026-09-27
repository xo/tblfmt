# Decisions

Each decision that shapes tblfmt, in the order that it was made. The entries
are append only. A decision is never edited to change its conclusion. When a
later decision replaces one, the older entry keeps its text and its heading
names the decision that replaced it.

Each heading carries a status after the title: `Decided`, `Amends Dn`,
`Amended by Dn`, `Superseded by Dn` or `Conflicts with Dn`. The open questions
for Ken are at the end.

D1 to D30 were written on 2026-09-27 from the history of the repository, from
`ff80f92` (2018-06-23) to `04af38a` (2026-09-23). Each of them gives the
commits, what changed and the reason. The reason comes from a commit message,
a code comment or the README. If none of these gives a reason, the entry says
"Reason not recorded". Each entry also has a certainty:

1. High: a commit message, a comment or the README says that the change is
   deliberate, or gives a reason for it.
2. Medium: no text says so, but the rule holds across several commits.
3. Low: one commit made the change, and it reads like a fix.

A line number in D1 to D30 is at `04af38a`.

| Decision | Title | Status |
| --- | --- | --- |
| [D1](#d1-psql-is-the-reference-for-the-output-decided) | psql is the reference for the output | Decided |
| [D2](#d2-the-table-encoder-streams-and-buffers-only-enough-rows-to-size-the-columns-decided) | The table encoder streams, and buffers only enough rows to size the columns | Decided |
| [D3](#d3-a-sentinel-error-is-a-constant-of-type-error-decided) | A sentinel error is a constant of type Error | Decided |
| [D4](#d4-frommap-takes-psql-pset-names-decided) | FromMap takes psql `\pset` names | Decided |
| [D5](#d5-one-option-type-serves-every-encoder-decided) | One Option type serves every encoder | Decided |
| [D6](#d6-the-module-keeps-a-recent-minimum-go-version-decided) | The module keeps a recent minimum Go version | Decided |
| [D7](#d7-the-output-is-checked-against-golden-files-decided) | The output is checked against golden files | Decided |
| [D8](#d8-output-that-is-too-wide-switches-to-expanded-mode-as-with-psql-expanded-auto-decided) | Output that is too wide switches to expanded mode, as with psql `expanded auto` | Decided |
| [D9](#d9-the-random-test-data-uses-a-fixed-seed-decided) | The random test data uses a fixed seed | Decided |
| [D10](#d10-a-pager-that-quits-early-is-not-an-error-decided) | A pager that quits early is not an error | Decided |
| [D11](#d11-tests-run-in-github-actions-on-linux-and-lint-has-a-config-but-no-ci-job-decided) | Tests run in GitHub Actions on Linux, and lint has a config but no CI job | Decided |
| [D12](#d12-the-crosstab-view-follows-psql-crosstabview-decided) | The crosstab view follows psql `\crosstabview` | Decided |
| [D13](#d13-a-crosstab-view-covers-one-result-set-decided) | A crosstab view covers one result set | Decided |
| [D14](#d14-the-last-column-is-not-padded-when-it-is-left-aligned-decided-conflicts-with-d28) | The last column is not padded when it is left aligned | Decided. Conflicts with D28 |
| [D15](#d15-a-caller-can-choose-the-scan-type-of-each-column-decided) | A caller can choose the scan type of each column | Decided |
| [D16](#d16-a-header-can-be-recased-decided) | A header can be recased | Decided |
| [D17](#d17-an-encoder-writes-no-blank-line-after-the-last-result-set-decided) | An encoder writes no blank line after the last result set | Decided |
| [D18](#d18-the-unaligned-encoder-writes-a-footer-and-the-csv-encoder-does-not-decided) | The unaligned encoder writes a footer, and the csv encoder does not | Decided |
| [D19](#d19-tuples_only-turns-off-the-footer-in-every-format-decided) | `tuples_only` turns off the footer in every format | Decided |
| [D20](#d20-the-template-encoders-are-go-funcs-not-texttemplate-decided) | The template encoders are Go funcs, not text/template | Decided |
| [D21](#d21-a-table-format-writes-an-upper-case-table-with-no-borders-decided) | A `table` format writes an upper case table with no borders | Decided |
| [D22](#d22-a-generic-sqlnull-is-unwrapped-by-its-shape-decided) | A generic `sql.Null` is unwrapped by its shape | Decided |
| [D23](#d23-the-json-code-uses-encodingjsonv2-and-go-1271-is-the-minimum-decided) | The JSON code uses encoding/json/v2, and Go 1.27.1 is the minimum | Decided |
| [D24](#d24-csv-applies-numericlocale-with-quotes-and-json-ignores-it-decided) | csv applies `numericlocale` with quotes, and JSON ignores it | Decided |
| [D25](#d25-nan-and-the-infinities-use-the-postgresql-spellings-and-are-strings-in-json-decided) | NaN and the infinities use the PostgreSQL spellings, and are strings in JSON | Decided |
| [D26](#d26-a-uint64-above-the-int64-range-is-a-json-string-amended-by-d29) | A uint64 above the int64 range is a JSON string | Amended by D29 |
| [D27](#d27-a-null-value-is-aligned-by-its-column-decided) | A null value is aligned by its column | Decided |
| [D28](#d28-tblfmt-pads-the-last-column-where-psql-trims-it-decided-conflicts-with-d14) | tblfmt pads the last column where psql trims it | Decided. Conflicts with D14 |
| [D29](#d29-every-uint64-is-a-json-string-decided-amends-d26) | Every uint64 is a JSON string | Decided. Amends D26 |
| [D30](#d30-the-json-encoder-indents-its-output-decided) | The JSON encoder indents its output | Decided |
| [D31](#d31-a-result-set-that-has-no-columns-is-valid-and-each-encoder-writes-it-as-psql-does-decided) | A result set that has no columns is valid, and each encoder writes it as psql does | Decided |
| [D32](#d32-a-dependency-is-kept-at-the-version-that-usql-requires-or-higher-decided) | A dependency is kept at the version that usql requires, or higher | Decided |
| [D33](#d33-pr-18-is-closed-and-its-ideas-that-keep-the-api-wait-for-a-benchmark-decided) | PR 18 is closed, and its ideas that keep the API wait for a benchmark | Decided |
| [D34](#d34-the-repository-carries-its-agent-files-and-the-skills-are-copies-decided-amended-by-d38) | The repository carries its agent files, and the skills are copies | Decided. Amended by D38 |
| [D35](#d35-every-text-that-a-person-reads-is-written-in-plain-english-decided) | Every text that a person reads is written in plain English | Decided |
| [D36](#d36-an-agent-stages-a-change-and-ken-commits-it-decided) | An agent stages a change, and Ken commits it | Decided |
| [D37](#d37-tblfmt-makes-no-promise-of-backward-compatibility-decided) | tblfmt makes no promise of backward compatibility | Decided |
| [D38](#d38-the-agent-rules-are-in-agentsmd-and-claudemd-imports-them-amends-d34) | The agent rules are in AGENTS.md, and CLAUDE.md imports them | Amends D34 |

### D1. psql is the reference for the output. Decided.

Certainty: High.

Commits: `ff80f92` (2018-06-23), `faf1a88` (2019-01-31), `28c54ec`
(2019-06-09), `43905c0` (2021-04-04), `deafcc6` (2026-09-23), `ef17bef`
(2026-09-23).

The first README shows a psql style table and lists a TODO: "finish
compatibility with PostgreSQL (`psql`) output". `faf1a88` added a test that
ran each value through the real `psql` when `PSQL_CONN` was set, and put a
PostgreSQL service in the Travis config. `28c54ec` documented `PSQL_CONN` in
the README. The crosstab tests of `43905c0` are copied from the psql manual.
The tests added in September 2026 quote the output of psql 18.6 as the
expected result.

`deafcc6` states the rule in the README: tblfmt follows the output of psql
closely, the differences it lists are deliberate, and any other difference is
a bug.

`162708e` (2021-03-30) removed the live psql comparison test, and `2e553f9`
(2021-04-20) removed its README section. The helpers `internal.PsqlEncodeAll`
and `internal.PsqlEncode` remain, and nothing calls them.

Reason: the README says that tblfmt was designed for use by `usql`, whose
output follows psql. No text says why psql and not another client.

### D2. The table encoder streams, and buffers only enough rows to size the columns. Decided.

Certainty: High.

Commits: `ff80f92` (2018-06-23), `4aaadae` (2019-01-30, thda), `1761efe`
(2019-01-31, thda).

The first README calls the package "a streaming, bufferred table encoder".
The first `TableEncoder` has a `count` field, with this comment: "Note: when 0
all rows will be scanned (buffered) prior to encoding the table." `WithCount`
sets it. The default is 0, so by default the encoder buffers every row.

`4aaadae` changed the lookahead into batches. The encoder reads `count` rows,
measures them, writes them, and reads the next batch. The widths only grow
from one batch to the next. `1761efe` buffered the output in a `bufio.Writer`.

The header is written once, with the widths of the first batch. A later batch
that holds a wider value is wider than the header above it.

Reason, from the message of `4aaadae`: reading by batches "should make sure
that the width does not vary at each row".

### D3. A sentinel error is a constant of type Error. Decided.

Certainty: Medium.

Commits: `ff80f92` (2018-06-23), `37897df` (2021-03-30), `ee9e6a9`
(2024-02-10).

The first commit declares `type Error string` with an `Error` method, and
`ErrResultSetIsNil` as a constant of that type. `37897df` gathered the
constants in `error.go`. `ee9e6a9` moved them to `tblfmt.go`. There are now
21 constants, and each refactor kept the pattern.

Reason not recorded.

### D4. FromMap takes psql `\pset` names. Decided.

Certainty: High.

Commits: `8cb7903` (2019-01-27), `1c1f094` (2019-01-28), `b89f8cc`
(2021-03-28), and each commit that added a key.

`8cb7903` added `EncoderFromMap`, which read `format`, `border`, `linestyle`
and `unicode_border_linestyle`. These are psql `\pset` names and values.
`1c1f094` replaced it with `FromMap`, which returns a `Builder` and its
options. A comment lists the psql formats: unaligned, aligned, wrapped, html,
asciidoc, latex, latex-longtable, troff-ms, json and csv. Later commits added
`tuples_only`, `fieldsep`, `fieldsep_zero`, `recordsep`, `recordsep_zero`,
`csv_fieldsep`, `tableattr`, `title`, `null`, `footer`, `expanded`, `pager`,
`pager_min_lines`, `columns` and `numericlocale`.

Some keys are not psql names: `lower_column_names`, `use_column_types`,
`time`, `timezone` and `locale`, and the formats `vertical` and `table`.

Reason, from the comment that `b89f8cc` added on `FromMap`: it is "primarily
a helper func to accommodate psql-like format option names".

### D5. One Option type serves every encoder. Decided.

Certainty: High.

Commits: `1c1f094` (2019-01-28), `b082de3` (2021-03-30).

`1c1f094` replaced `TableEncoderOption` with one general `Option`. `b082de3`
made `Option` an interface with an unexported `apply` method. The `option`
struct holds one func for each encoder type. An option that has no func for
an encoder does nothing to it. An unknown encoder type panics.

Reason: the title of `b082de3` calls the new form "more idiomatic". No text
says why one type serves every encoder.

### D6. The module keeps a recent minimum Go version. Decided.

Certainty: Medium.

Commits: `f35e3c3` (2019-12-13), `b89f8cc` (2021-03-28), `4cf184a`
(2023-08-09), and the version bumps after them.

`f35e3c3` dropped support for Go before 1.8 and deleted `rs_go18.go`. After
that the `go` line rose with Go itself: 1.13, 1.16, 1.17, 1.18, 1.19, 1.20,
1.21, 1.22, 1.23, 1.25.0 and 1.27.1. `4cf184a` deleted a local `max` func when
Go 1.21 added the builtin. `4c2fd87` (2025-04-18) ran `modernize -fix`.

Reason not recorded. See D23 for 1.27.1.

### D7. The output is checked against golden files. Decided.

Certainty: High.

Commits: `076ccde` (2021-02-04, Jan Was), `0cff801` (2021-03-29), `162708e`
(2021-03-30), `6b9f344` (2021-03-30), `629b8a1` (2026-03-29).

`076ccde` added `testdata/*.expected.txt`. `0cff801` renamed them and added a
generator. `162708e` regenerated them. `6b9f344` compressed them with gzip.
`629b8a1` replaced the gzip files with one directory for each data set. Each
case is three files: `NN.in` holds the options, `NN.gld` holds the expected
bytes, and `NN.out` holds what the last run wrote. The test writes `NN.out` on
every run, and the `.out` files are committed. `629b8a1` also marks `.gld` and
`.out` as binary in `.gitattributes`.

`testdata/genformats.go` makes the format files from tblfmt itself, so those
files hold tblfmt's own output and not psql output. `testdata/crosstab.txt` is
the exception. `testdata/gencrosstab.go` makes it by running psql.

Reason not recorded in a commit. The `.gitattributes` comment from D34 says that the golden files hold the exact bytes an encoder writes, and that
some hold a carriage return on purpose.

### D8. Output that is too wide switches to expanded mode, as with psql `expanded auto`. Decided.

Certainty: Medium.

Commits: `2729f07` (2020-12-22, Jan Was), `184dbaa` (2021-02-05, Jan Was),
`b21d7b8` (2021-03-21), `402b42c` (2021-10-03).

`2729f07` added the expanded encoder. `184dbaa` added `WithMinExpandWidth`,
and `FromMap` sets it from the terminal width when `expanded` is `auto`.
`b21d7b8` lets the `columns` key set the width. `402b42c` left aligns the
headers in the switched output. The expanded encoder drops the default row
count footer unless the caller set a summary.

Reason: the titles describe the behavior, "Auto expand if width is greater
than screen". No text gives a reason beyond the psql option name.

### D9. The random test data uses a fixed seed. Decided.

Certainty: Medium.

Commit: `7f9dfc6` (2021-02-09, Jan Was).

The big random result set used a random seed unless `DETERMINISTIC` was set.
`7f9dfc6` reversed the default. The seed is now `1549508725559526476`, and a
value of `DETERMINISTIC` can choose another. The same commit writes each
random time in UTC. The golden tests of D7 depend on this seed.

Reason not recorded.

### D10. A pager that quits early is not an error. Decided.

Certainty: High.

Commits: `c2858ae` (2021-03-21), `95ec403` (2021-03-22).

`c2858ae` added pager support. `95ec403` added `checkErr`, which drops
`EPIPE` when a pager is running.

Reason, from the comment in `checkErr`: a broken pipe means that the pager
quit before it read all of the data, and that can be expected.

### D11. Tests run in GitHub Actions on Linux, and lint has a config but no CI job. Decided.

Certainty: Medium.

Commits: `c85b4cf` (2021-03-30), `b441505` (2021-03-30), `2faed8b`
(2025-05-20).

`c85b4cf` replaced Travis with GitHub Actions. The job runs `go test -v ./...`
on `ubuntu-latest` with the stable Go. `b441505` sets `core.autocrlf input`
so that the committed line endings reach the tests. `2faed8b` added
`.golangci.yml`. It turns on every linter and then turns off 33 by name,
`depguard` among them. No workflow runs golangci-lint.

Reason not recorded.

### D12. The crosstab view follows psql `\crosstabview`. Decided.

Certainty: High.

Commits: `73d1fa8` (2021-03-30), `3e603b1` (2021-04-02), `43905c0`
(2021-04-04).

`73d1fa8` added `view.go`, and `3e603b1` finished `CrosstabView`. It takes
the same four arguments as `\crosstabview`: the vertical, horizontal, data
and sort columns. Its errors match the psql checks, such as "data column must
be specified when query returns more than three columns".
`testdata/gencrosstab.go` runs each query through psql and records the
output. `43905c0` added two examples from the psql manual.

Reason: the generator says that its examples come from the PostgreSQL wiki
page on crosstabview. No text says why the view copies psql.

### D13. A crosstab view covers one result set. Decided.

Certainty: High.

Commit: `3e603b1` (2021-04-02).

`NextResultSet` on a `CrosstabView` always returns false. The doc comment on
the type says, under "CAUTION", that "A design decision was made to not
support multiple result sets". The caller makes a new view for each result
set.

Reason: the comment states the decision and its effect. It gives no reason
beyond that.

### D14. The last column is not padded when it is left aligned. Decided. Conflicts with D28.

Certainty: Medium.

Commits: `846f3aa` (2021-03-27, Jan Was), `e4096cb` (2021-04-17, Jan Was),
`ef17bef` (2026-09-23).

`846f3aa` stopped padding the last cell at border 0 and border 1. `e4096cb`
limited this to a left aligned cell, so a right aligned last column is still
padded. `ef17bef` made the rule use the column's alignment for a null cell.
The code is in `TableEncoder.row`, `encode.go:491-494`.

The encoder still writes one space after the last value, in the place of the
wrap marker. psql writes nothing there.

Reason not recorded. The rule moves the output toward psql, and D28 says that
tblfmt does the opposite. See D28.

### D15. A caller can choose the scan type of each column. Decided.

Certainty: Medium.

Commits: `d919550` (2021-04-30), `110a5bb` (2024-02-13).

By default the encoders scan every value into a `*any`. `d919550` added
`WithUseColumnTypes`, which scans into a new value of the driver's
`ScanType`. `110a5bb` added `WithColumnTypes` and `WithColumnTypesFunc`, so
that another package can build the scan destinations.

Reason: the title of `110a5bb` says it allows "using packages to build the
column types for result sets". No text says why the default is `*any`.

### D16. A header can be recased. Decided.

Certainty: Medium.

Commits: `79ceb9d` (2021-07-28), `38f2ce5` (2026-04-07).

`79ceb9d` added `WithLowerColumnNames` and the `lower_column_names` key. They
lower case a column name that is all upper case. `38f2ce5` replaced this with
the `Transformer` interface and the `TransformStyle` values, and added
`WithForceUpperColumnNames` for the `table` format of D21.

Reason not recorded.

### D17. An encoder writes no blank line after the last result set. Decided.

Certainty: Low.

Commit: `9cf97d8` (2021-11-20).

`EncodeAll` on the table, expanded, unaligned and template encoders wrote a
newline after the last result set. `9cf97d8` removed it, and the tests lost
their trailing blank line. The JSON encoder still writes a final newline.

Reason not recorded. The title is "Standardizing newline output".

### D18. The unaligned encoder writes a footer, and the csv encoder does not. Decided.

Certainty: Medium.

Commit: `b4e589b` (2024-04-01).

`b4e589b` gave the unaligned encoder `DefaultTableSummary()` and the csv
encoder an empty `Summary{}`. This matches psql, which writes "(1 row)" in
unaligned output and no footer in csv output.

Reason: the title is "Add missing summary for unaligned tables". It does not
name psql.

### D19. `tuples_only` turns off the footer in every format. Decided.

Certainty: High.

Commits: `97bbfd9` (2021-03-27, Jan Was), `3594abd` (2021-05-01, Jan Was),
`d89f7ad` (2026-03-03, Sylvain), `e8f8df8` (2026-04-07).

`97bbfd9` added `tuples_only` for the aligned format, and `3594abd` for the
unaligned format. `d89f7ad` made the unaligned and csv path turn off the
footer as well. `e8f8df8` did the same for the `table` format.

Reason, from the message of `d89f7ad`: the aligned format already did it, and
the unaligned and csv path still printed the row count. It fixes
xo/tblfmt#52 and relates to xo/usql#486.

### D20. The template encoders are Go funcs, not text/template. Decided.

Certainty: High.

Commits: `c8da282`, `7a1b1d0`, `fa8cb67`, `5af5ff3`, `a4a1779`, `42109eb`
(all 2026-03-29).

`c8da282` removed `text/template` and `html/template`. It added `Template`,
a struct, and `WriteHTMLTo`, `WriteAsciidocTo` and `WriteVerticalTo`. It
removed the public `WithRawTemplate` and the embedded `templates` package.
`WithExecutor` now takes `func(io.Writer, *Template) error`. Each func keeps
its old template text in a comment. `7a1b1d0` wrote the vertical func.
`fa8cb67` and `5af5ff3` fixed the asciidoc and html output, and `5af5ff3`
escapes each header and cell with `html.EscapeString`. `a4a1779` removed the
`testdata` package.

Reason not recorded. The title of `c8da282` says only "Initial work on
removing {text,html}/template".

### D21. A `table` format writes an upper case table with no borders. Decided.

Certainty: Medium.

Commits: `38f2ce5` (2026-04-07), `06ba293` (2026-04-07), `e8f8df8`
(2026-04-07).

`38f2ce5` added the `table` format to `FromMap` and `TableLineStyle`. The
format uses border 0, inline headers, upper case column names and left
alignment for every value. `06ba293` added an example to the doc comment.
psql has no such format.

Reason not recorded.

### D22. A generic `sql.Null` is unwrapped by its shape. Decided.

Certainty: High.

Commits: `c5684fa` (2023-05-07), `b496c3e` (2026-09-23).

`c5684fa` added the missing `sql.Null*` types to the formatter. `b496c3e`
added the generic `sql.Null[T]`. `unwrapNull` finds it with reflection: a
struct from `database/sql` with the fields `V` and `Valid`. Any other
`driver.Valuer` still goes through its `Value` method, up to 10 levels deep.

Reason, from the message of `b496c3e`: MySQL reports `sql.Null[uint64]` as the
scan type of a nullable `BIGINT UNSIGNED`, and no case matched it, so every
format wrote it as a JSON struct. The `Value` method of the generic type runs
`driver.DefaultParameterConverter`, which turns every `uint64` into an
`int64` and rejects a `uint64` that has its high bit set.

### D23. The JSON code uses encoding/json/v2, and Go 1.27.1 is the minimum. Decided.

Certainty: High.

Commit: `fa21f3a` (2026-09-23).

`fa21f3a` moved the formatter, the JSON encoder and the test helpers to
`encoding/json/v2` and `encoding/json/jsontext`, and raised the `go` line
from 1.25.0 to 1.27.1. The v2 defaults differ from v1, so `jsonOptions` sets
four of them back to the v1 behavior: sorted map keys, invalid UTF-8
accepted, and a nil map or slice written as `null`. `WithJSONConfig` drops a
prefix or indent that is not made of spaces and tabs, because `jsontext`
panics on one. `TestFormatJSON` now compares decoded values and not bytes.

Reason: the message says that the output of every format is unchanged, and
why each setting is pinned. It does not say why the package moved to v2. It
changes the two things together, and it does not say whether one needs the
other.

### D24. csv applies `numericlocale` with quotes, and JSON ignores it. Decided.

Certainty: High.

Commits: `02cd659` (2022-09-18), `fc9bb28` (2026-09-23).

`02cd659` added `numericlocale` with `golang.org/x/text`, with `en-US` as the
default locale. In csv, a grouped number such as `1,234,567` was then written
without quotes. In JSON, the same text was written as a bare value.
`fc9bb28` escapes a locale formatted number, so that csv quotes it as psql
does. JSON now ignores the locale. `useNumericLocale` and its comment in
`fmt.go` hold the rule.

Reason, from the message and the comment: the csv reader took the number as
three fields, and the JSON was not valid. JSON has a number type but no
syntax for a grouping separator. If JSON honors the locale, a number becomes
a string, and the JSON type of a column then follows a display option.

Why `en-US` is the default locale is not recorded.

### D25. NaN and the infinities use the PostgreSQL spellings, and are strings in JSON. Decided.

Certainty: High.

Commit: `eb6e816` (2026-09-23).

Every format now writes `NaN`, `Infinity` and `-Infinity`, in place of the Go
forms `NaN`, `+Inf` and `-Inf`. The JSON encoder writes them as strings. A
`uintptr` is a string in JSON as well. `floatString` in `fmt.go` holds the
rule.

Reason, from the message and the comment: a bare `NaN` or `+Inf` is not valid
JSON, and a strict parser rejects the whole document. JSON has no NaN and no
infinity. PostgreSQL's `to_jsonb` writes them as strings, and psql shows the
PostgreSQL spellings.

### D26. A uint64 above the int64 range is a JSON string. Amended by D29.

Certainty: High.

Commit: `eb6e816` (2026-09-23).

A JSON string of the exact digits replaced any unsigned integer above the
largest `int64`, such as the largest MySQL `BIGINT UNSIGNED`. The string is
never locale formatted.

Reason, from the message: a consumer that reads a JSON number as an `int64`
or a `float64` loses such a value.

D29 replaced the test by value with a test by type, on the same day.

### D27. A null value is aligned by its column. Decided.

Certainty: High.

Commit: `ef17bef` (2026-09-23).

The whole result set shared one empty value for its null cells. That value
had the zero `Align`, so the null string sat to the left in every column.
Now `columnAlign` takes each column's alignment from its values, and the null
cells follow it. This applies in the table encoder and the template encoder.
In html, a null cell in a numeric column gets `align="right"`, as the psql
html output does. A column whose values disagree keeps the default, and so
does a column that holds only nulls.

Reason, from the message: psql aligns the null string by the type of the
column. The message quotes the psql output.

### D28. tblfmt pads the last column where psql trims it. Decided. Conflicts with D14.

Certainty: High that the README states a decision. The code does not do what
the README says.

Commit: `deafcc6` (2026-09-23).

`deafcc6` added "Differences from `psql`" to the README, at `README.md:115`.
Its one entry, at `README.md:120-140`, says that tblfmt pads the last column
of a bordered table so that every line has the same width. It says that psql
pads the header and trims the data rows, and that tblfmt will not follow it.
A comment in the staged `TestEncodeAllNoColumnsFirst` points to this README
section.

Reason, from the README: lines of different widths are hard to select in a
terminal, to diff, and to lay out. The ragged edge carries no information,
and it is not consistent with the psql header, which is padded.

Conflict: the code at `04af38a` follows D14 and does not pad a left aligned
last column at border 0 or border 1. For the README's own example, a
`select 42 as n, 'a'::text as t` style result, a probe gives these line
widths: header 9, divider 9, row ` 42 | a ` 8, row `  7 | bb ` 9. The README
shows 9 for every line. The README describes the right aligned case and the
border 2 case. One of the two must change.

### D29. Every uint64 is a JSON string. Decided. Amends D26.

Certainty: High.

Commit: `96b7b49` (2026-09-23).

`unsignedInt64` now decides by type. A `uint64` or a `uint` is a JSON string
for every row, whatever its value. A `uint` counts whatever its width.

Reason, from the message and the comment: a test by value writes `{"n":42}`
for one row and `{"n":"18446744073709551615"}` for the next. A consumer then
cannot give the column a type without reading every value. MySQL's `BIGINT
UNSIGNED` crosses the limit with ordinary data. `uint` is included so that
the output is the same on every platform.

### D30. The JSON encoder indents its output. Decided.

Certainty: High.

Commit: `04af38a` (2026-09-23).

The JSON encoder wrote a compact document, but the formatter indented a
nested object or array. The nested value then broke across lines in the
middle of a compact row. Now the whole document is indented by two spaces for
each level. The formatter gets the indent of the value's column as its
prefix, so that a nested value lines up under its key. An empty result set is
still `[]`.

Reason, from the message: the mixed output broke a nested value across lines
inside a compact row.

### D31. A result set that has no columns is valid, and each encoder writes it as psql does. Decided.

Certainty: High.

Date: 2026-09-27. The change is staged, and Ken reviews it before it is
committed.

SQL permits a result set that has no columns, and psql writes one. `select;`
returns one row with no values, and `select from pg_class where false;`
returns no rows. psql 18.6 against PostgreSQL 18.6 writes `--` and `(1 row)`
for the first, at border 1. Some drivers have no column metadata and return
no columns for every statement that returns no rows. The SurrealDB and
Couchbase drivers in `xo/dbimp` do this, for example for `DELETE author`. In
each case the statement ran, and then usql printed
`error: result set has no columns` and exited with 1.

Every encoder now accepts a result set that has no columns, and `EncodeAll`
goes on to the next result set. This reverses the check that `4aaadae`
(2019-01-30) added. The output, measured against psql 18.6:

1. The aligned format draws the lines that psql draws, with no header line
   and no row line, and then the summary. At border 1 this is `--`, at
   border 0 it is an empty line, and at border 2 it is three `+--+` lines.
2. The expanded format writes no record. It writes only the summary.
3. The unaligned format writes an empty header line and no row line. The csv
   format writes an empty header line.
4. The JSON format writes `[]` for no rows and `{}` for each row.
5. The template formats write an empty header row and one empty row for each
   row.

`ErrResultSetHasNoColumns` is removed, because no encoder returns it. See
D37. usql compares with it in `handler/handler.go` for a SQL Server `EXEC`.
When usql takes the tag that carries this change, usql removes that comparison
and tests for zero columns itself.
`TestEncodeNoColumns` and `TestEncodeAllNoColumnsFirst` hold the psql output.

This decision does not cover a result set that has columns and no rows. psql
writes its header, and tblfmt does not. That is B1 in
[BACKLOG.md](BACKLOG.md).

### D32. A dependency is kept at the version that usql requires, or higher. Decided.

Certainty: High.

Date: 2026-09-27.

Under minimum version selection, the version in the `go.mod` of a library is
a minimum. usql required `go-runewidth` v0.0.30 and `golang.org/x/text`
v0.42.0, so usql built with those versions whatever tblfmt required.
Dependabot PR 61 (runewidth v0.0.24) and PR 62 (x/text v0.38.0) were based on
a commit older than D23, and both were below the usql versions. Every tblfmt
test passes with v0.0.30 and v0.42.0. The usql, dbmeta, Gemini and DeepSeek
reviews all recommended the same action.

Ken closed both PRs, and tblfmt moved to v0.0.30 and v0.42.0. A version above
the usql version forces usql up, so a bump above it needs a reason.

The removal of `.github/dependabot.yml` is staged, and no entry records who
made that choice or why. It is an open question below.

### D33. PR 18 is closed, and its ideas that keep the API wait for a benchmark. Decided.

Certainty: High.

Date: 2026-09-27.

PR 18 (nineinchnick, 2021) reduced allocations with a `sync.Pool` of
`*Value`. It was 89 commits behind main. These parts of it change the API or
the output, or are faults:

1. It added `HeaderInto`, `FormatInto` and `Free` to the exported
   `Formatter` interface, which breaks every `Formatter` outside tblfmt.
2. It removed the exported `FormatBytes`.
3. It changed a `uint8` from a character to a number.
4. It freed a `*Value` while the table encoder still held the batch for the
   widths, and while the template encoder still held the rows.

usql does not implement `Formatter` and does not call `FormatBytes`, but usql
users see the `uint8` change. The API changes alone are not a reason to refuse
a change. See D37. Ken closed the PR with thanks. Two ideas are still useful:
cache the row styles for each encoder, and reuse the scan buffer between
batches. They are B34 in [BACKLOG.md](BACKLOG.md). A benchmark of
memory use on large result sets comes first. usql issue 144 asked for this
work, and usql moved it to its backlog on 2026-09-23 as not measured.

### D34. The repository carries its agent files, and the skills are copies. Decided. Amended by D38.

Certainty: High.

Date: 2026-09-27.

tblfmt follows the layout of dbmeta, which dbmeta D89 records:

1. `CLAUDE.md` holds the rules for a coding agent. `CONTRIBUTING.md` holds
   the short form for a person. There is no `AGENTS.md`.
2. `docs/PLAN.md` holds the decisions, and `docs/BACKLOG.md` holds the known
   faults and the deferred work. dbimp and cql use the same two files.
3. `skills-lock.json` names the source of each skill. `.agents/skills/<name>`
   and `.claude/skills/<name>` each hold an ordinary copy.
4. `.gitignore` ignores `.claude/settings.local.json`.
5. `.gitattributes` makes every text file LF on every checkout. The golden
   files stay binary.

The `npx skills` command writes `.claude/skills/<name>` as a symbolic link
unless it gets `--copy`. A Windows checkout writes a symbolic link as a text
file, and Claude Code then loads no skill and reports nothing.
`TestSkillsAreCopies` fails on a link, and it fails when the two copies
differ.

### D35. Every text that a person reads is written in plain English. Decided.

Certainty: High.

Date: 2026-09-27.

Ken added the `simple-english` skill and asked that it apply to all user
facing documentation and code comments. It covers the README, the documents
in `docs/`, the Go doc comments, the comments in the code, error messages and
commit messages. The first pass rewrote the comments in every Go file and the
README. It changed no code, no identifier and no string value.

The pass found comments that were wrong about the code. The clear ones were
corrected. The others, and the faults in the code under them, are in
[BACKLOG.md](BACKLOG.md).

The values of the error constants did not change, because a caller can
compare with the text.

### D36. An agent stages a change, and Ken commits it. Decided.

Certainty: High.

Date: 2026-09-27.

An agent does not commit, push or tag in this repository. It stages the
change with `git add` and gives a proposed commit message. Ken reviews the
staged change and commits it. `AGENTS.md` holds the rule, as standing
rule 1.

### D37. tblfmt makes no promise of backward compatibility. Decided.

Certainty: High.

Date: 2026-09-27.

Ken said that none of the xo projects make a promise about their APIs. The
code is idiomatic Go, but the xo projects do not follow the compatibility
promise of the Go project.

A change can rename or remove an exported identifier, or add a method to an
exported interface, when that makes the code better. An exported identifier
that has no use is removed, and it is not marked `Deprecated:`.
`ErrResultSetHasNoColumns` is the first example, in D31.

usql imports tblfmt, so a change that breaks a caller in usql names that
caller in its commit message, and usql follows it when it takes the tag.

### D38. The agent rules are in AGENTS.md, and CLAUDE.md imports them. Amends D34.

Certainty: High.

Date: 2026-09-27.

Ken set one agent layout for every xo repository. dbmeta D110 records it.
Codex reads `AGENTS.md`, so `AGENTS.md` holds the rules. `CLAUDE.md` is an
ordinary file that holds one line, `@AGENTS.md`, which Claude Code imports. It
is not a symbolic link, for the reason in D34.

`AGENTS.md` starts with three standing rules: stage changes and commit only
when Ken says so, load `simple-english` before writing text that a person
reads, and load `go-pedantry` before writing or reviewing Go code. A rule of
the project wins where it conflicts with `go-pedantry`.

This replaces item 1 of D34, which said that the rules were in `CLAUDE.md`
and that there was no `AGENTS.md`.

## Open questions for Ken

An open question is not a decision. When Ken answers one, it becomes a
decision above, and the question goes.

1. D14 and D28 disagree. The README says that tblfmt pads the last column of
   a bordered table. The code pads a right aligned last column and does not
   pad a left aligned one at border 0 or border 1. For the README example,
   the row ` 42 | a` is 8 wide and the header is 9. Which one changes, the
   code or the README? See B10.
2. For a result set that has columns and no rows, psql writes the header and
   the divider, and tblfmt writes only `(0 rows)`. usql users see this. Must
   tblfmt write the header, as psql does? See B1.
3. psql writes `(0 rows)` for an expanded result set that has no rows, and
   `(1 row)` for an expanded result set that has no columns. `FromMap` gives
   the expanded encoder no summary, so tblfmt writes nothing. The automatic
   switch to expanded output keeps the summary. Which is right? See B7.
4. `EncodeAll` on the JSON encoder writes two result sets as two arrays that
   a comma joins. This is not one JSON document. Is that deliberate? See B18.
5. psql writes one blank line after an aligned table. tblfmt writes none, and
   usql writes the line itself. Is this a deliberate difference from psql
   that the README must list? See D17.
6. The removal of `.github/dependabot.yml` is staged in the index, and this
   session did not make it. Did Ken make it, and does Dependabot stay off?
7. D20 and D23 have no recorded reason. Why did the templates become Go
   funcs, and why did the JSON code move to `encoding/json/v2`?
8. `.golangci.yml` disables `exhaustruct`, and the current linter reports it
   as `exhaustruct_v5`, so 50 findings return. `gomodguard` is deprecated.
   No CI job runs the linter. Must CI run it, and must the configuration
   follow the new names? See B31 and B35.
9. `internal.PsqlEncode` and `internal.PsqlEncodeAll` have had no caller
   since 2021. Does a psql comparison test come back, or do the helpers go?
   See B30.
