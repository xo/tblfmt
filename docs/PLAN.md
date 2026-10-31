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
| [D8](#d8-output-that-is-too-wide-switches-to-expanded-mode-as-with-psql-expanded-auto-decided-amended-by-d45) | Output that is too wide switches to expanded mode, as with psql `expanded auto` | Decided. Amended by D45 |
| [D9](#d9-the-random-test-data-uses-a-fixed-seed-decided) | The random test data uses a fixed seed | Decided |
| [D10](#d10-a-pager-that-quits-early-is-not-an-error-superseded-by-d42) | A pager that quits early is not an error | Superseded by D42 |
| [D11](#d11-tests-run-in-github-actions-on-linux-and-lint-has-a-config-but-no-ci-job-decided) | Tests run in GitHub Actions on Linux, and lint has a config but no CI job | Decided |
| [D12](#d12-the-crosstab-view-follows-psql-crosstabview-decided) | The crosstab view follows psql `\crosstabview` | Decided |
| [D13](#d13-a-crosstab-view-covers-one-result-set-decided) | A crosstab view covers one result set | Decided |
| [D14](#d14-the-last-column-is-not-padded-when-it-is-left-aligned-superseded-by-d44) | The last column is not padded when it is left aligned | Superseded by D44 |
| [D15](#d15-a-caller-can-choose-the-scan-type-of-each-column-decided) | A caller can choose the scan type of each column | Decided |
| [D16](#d16-a-header-can-be-recased-decided) | A header can be recased | Decided |
| [D17](#d17-an-encoder-writes-no-blank-line-after-the-last-result-set-decided-amended-by-d48) | An encoder writes no blank line after the last result set | Decided. Amended by D48 |
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
| [D28](#d28-tblfmt-pads-the-last-column-where-psql-trims-it-superseded-by-d44) | tblfmt pads the last column where psql trims it | Superseded by D44 |
| [D29](#d29-every-uint64-is-a-json-string-decided-amends-d26) | Every uint64 is a JSON string | Decided. Amends D26 |
| [D30](#d30-the-json-encoder-indents-its-output-superseded-by-d47) | The JSON encoder indents its output | Superseded by D47 |
| [D31](#d31-a-result-set-that-has-no-columns-is-valid-and-each-encoder-writes-it-as-psql-does-decided) | A result set that has no columns is valid, and each encoder writes it as psql does | Decided |
| [D32](#d32-a-dependency-is-kept-at-the-version-that-usql-requires-or-higher-decided-amended-by-d50) | A dependency is kept at the version that usql requires, or higher | Decided. Amended by D50 |
| [D33](#d33-pr-18-is-closed-and-its-ideas-that-keep-the-api-wait-for-a-benchmark-decided) | PR 18 is closed, and its ideas that keep the API wait for a benchmark | Decided |
| [D34](#d34-the-repository-carries-its-agent-files-and-the-skills-are-copies-decided-amended-by-d38) | The repository carries its agent files, and the skills are copies | Decided. Amended by D38 |
| [D35](#d35-every-text-that-a-person-reads-is-written-in-plain-english-decided) | Every text that a person reads is written in plain English | Decided |
| [D36](#d36-an-agent-stages-a-change-and-ken-commits-it-decided) | An agent stages a change, and Ken commits it | Decided |
| [D37](#d37-tblfmt-makes-no-promise-of-backward-compatibility-decided) | tblfmt makes no promise of backward compatibility | Decided |
| [D38](#d38-the-agent-rules-are-in-agentsmd-and-claudemd-imports-them-amends-d34) | The agent rules are in AGENTS.md, and CLAUDE.md imports them | Amends D34 |
| [D39](#d39-a-value-is-measured-by-grapheme-cluster-and-not-rune-by-rune-as-psql-does-decided-amended-by-d41) | A value is measured by grapheme cluster, and not rune by rune as psql does | Decided. Amended by D41 |
| [D40](#d40-each-golden-set-has-an-emoji-version-and-the-test-writes-the-golden-files-amends-d7) | Each golden set has an emoji version, and the test writes the golden files | Amends D7 |
| [D41](#d41-tblfmt-counts-an-emoji-variation-sequence-as-2-columns-until-go-runewidth-does-amends-d39) | tblfmt counts an emoji variation sequence as 2 columns until go-runewidth does | Amends D39 |
| [D42](#d42-tblfmt-does-not-run-a-pager-and-usql-does-supersedes-d10) | tblfmt does not run a pager, and usql does | Supersedes D10 |
| [D43](#d43-a-table-escapes-only-a-control-character-as-psql-does-decided-amended-by-d46) | A table escapes only a control character, as psql does | Decided. Amended by D46 |
| [D44](#d44-the-table-encoder-draws-a-table-as-psql-does-supersedes-d14-and-d28) | The table encoder draws a table as psql does | Supersedes D14 and D28 |
| [D45](#d45-the-expanded-encoder-draws-records-as-psql-does-amends-d8) | The expanded encoder draws records as psql does | Amends D8 |
| [D46](#d46-escapes-follow-psql-and-unicode-text-is-drawn-as-a-terminal-draws-it-amends-d43) | Escapes follow psql, and Unicode text is drawn as a terminal draws it | Amends D43 |
| [D47](#d47-the-caller-sets-the-separator-between-result-sets-and-json-is-compact-supersedes-d30) | The caller sets the separator between result sets, and JSON is compact | Supersedes D30 |
| [D48](#d48-usql-writes-the-blank-line-after-a-table-amends-d17) | usql writes the blank line after a table | Amends D17 |
| [D49](#d49-the-lint-configuration-follows-the-other-xo-repositories-decided) | The lint configuration follows the other xo repositories | Decided |
| [D50](#d50-dependabot-stays-off-decided) | Dependabot stays off | Decided |
| [D51](#d51-tblfmt-has-no-numeric-type-decided-amended-by-d63) | tblfmt has no Numeric type | Decided. Amended by D63 |
| [D52](#d52-the-csv-encoder-quotes-a-field-that-holds-a-space-or-a-tab-decided) | The csv encoder quotes a field that holds a space or a tab | Decided |
| [D53](#d53-the-caller-chooses-the-separator-between-unaligned-result-sets-decided) | The caller chooses the separator between unaligned result sets | Decided |
| [D54](#d54-the-template-formats-escape-a-control-character-decided) | The template formats escape a control character | Decided |
| [D55](#d55-withformatteroptions-returns-an-error-for-a-formatter-it-cannot-change-decided) | WithFormatterOptions returns an error for a formatter it cannot change | Decided |
| [D56](#d56-the-wrapped-format-is-an-option-of-the-table-encoder-decided) | The wrapped format is an option of the table encoder | Decided |
| [D57](#d57-tblfmt-writes-each-psql-format-decided) | tblfmt writes each psql format | Decided |
| [D58](#d58-the-json-encoder-ignores-withsummary-and-withwidths-decided) | The JSON encoder ignores WithSummary and WithWidths | Decided |
| [D59](#d59-tblfmt-has-benchmarks-and-a-module-that-compares-it-with-other-packages-decided) | tblfmt has benchmarks, and a module that compares it with other packages | Decided |
| [D60](#d60-the-vertical-format-is-the-g-layout-of-the-mysql-client-with-the-rules-of-psql-decided) | The vertical format is the `\G` layout of the mysql client, with the rules of psql | Decided |
| [D61](#d61-the-options-stay-small-and-a-caller-changes-the-escapes-with-a-formatter-decided) | The options stay small, and a caller changes the escapes with a formatter | Decided |
| [D62](#d62-the-tests-run-on-linux-windows-and-macos-decided) | The tests run on Linux, Windows and macOS | Decided |
| [D63](#d63-a-caller-sets-the-alignment-of-a-column-by-its-type-decided-amends-d51) | A caller sets the alignment of a column by its type | Decided. Amends D51 |
| [D64](#d64-the-html-and-asciidoc-formats-follow-psql-and-each-format-has-its-expanded-form-decided) | The html and asciidoc formats follow psql, and each format has its expanded form | Decided |
| [D65](#d65-testpsql-compares-tblfmt-with-psql-on-a-server-that-dbrun-starts-decided) | TestPsql compares tblfmt with psql, on a server that dbrun starts | Decided |
| [D66](#d66-a-later-batch-can-be-wider-than-the-header-decided) | A later batch can be wider than the header | Decided |

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

### D8. Output that is too wide switches to expanded mode, as with psql `expanded auto`. Decided. Amended by D45.

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

### D10. A pager that quits early is not an error. Superseded by D42.

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

### D14. The last column is not padded when it is left aligned. Superseded by D44.

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

### D17. An encoder writes no blank line after the last result set. Decided. Amended by D48.

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

Reason, from Ken on 2026-10-31: speed, and the escapes, which were hard to get
right in a template. A Go func with `fmt.Fprint` was as easy to write as a
template, and Ken does not like to write Go templates. The templates were
simple enough that the change made little difference. The comments kept the
text of the templates that the funcs replaced. D64 replaced the html and
asciidoc funcs with ports of psql, and their comments went.

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

Reason, from Ken on 2026-10-31: `encoding/json/v2` is in the standard library
from Go 1.27, so tblfmt uses it. The minimum Go version follows from that.

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

### D28. tblfmt pads the last column where psql trims it. Superseded by D44.

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

### D30. The JSON encoder indents its output. Superseded by D47.

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

### D32. A dependency is kept at the version that usql requires, or higher. Decided. Amended by D50.

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

### D39. A value is measured by grapheme cluster, and not rune by rune as psql does. Decided. Amended by D41.

Certainty: High.

Date: 2026-10-28.

A grapheme cluster is the group of runes that a terminal draws as one
character. Ken saw a column in usql that did not line up, and asked for the
alignment to be fixed where possible. He does not want tblfmt to follow psql
here, because psql counts each rune too.

`FormatBytes` added the width of each rune. An emoji with a skin tone, such as
U+1F44D U+1F3FD, is two wide runes, so it was counted 4 columns wide and a
terminal draws it 2 wide. `FormatBytes` now copies the text that it does not
escape, and measures each such run with `runewidth.StringWidth` when the run
ends. A run ends at an escaped rune, a tab, a newline and the end of the
value. `StringWidth` segments the run into grapheme clusters. tblfmt needs no
new dependency for this, because go-runewidth v0.0.31 already segments text.

The measure follows the current terminals that usql users are likely to have.
xterm and some tmux configurations still draw each rune on its own, and there
a column with such a cluster does not line up.

One case is still wrong. go-runewidth counts a character that is text by
default followed by U+FE0F, such as U+2764 U+FE0F, as 1 column, and a
terminal draws it 2 wide. That fault is in go-runewidth. It is B38 in
[BACKLOG.md](BACKLOG.md). Ken reports it upstream.

`TestFormatBytesClusters` and `TestEncodeClusters` hold the cases. "Differences
from `psql`" in `README.md` lists this difference.

### D40. Each golden set has an emoji version, and the test writes the golden files. Amends D7.

Certainty: High.

Date: 2026-10-28.

`testdata` has four more sets: `big-emoji`, `multi-emoji`, `tiny-emoji` and
`wide-emoji`. Each uses the same 33 option files as the set it copies.
`internal.BigEmoji` uses the seed `internal.BigEmojiSeed`, and its text adds
emoji to the glyphs of `Big`. Some of the emoji are clusters of more than one
rune: a skin tone, a flag, a keycap, U+FE0F, and emoji that U+200D joins.
`MultiEmoji`, `TinyEmoji` and `WideEmoji` hold emoji next to a tab, a newline
and an escaped rune. `Big` gives the same output as before.

`go test -run TestFromMapFormats -update` writes each `NN.gld` from the output,
and it writes a golden file that is missing. Before this, nothing could write
a golden file. See B29.

A golden file holds the output of tblfmt itself, so it cannot show that a
column is padded wrong. `TestGoldenAlignment` can. It makes sure that each
line of a table with border 2 has the same display width, with a tab stop
every 8 columns. With the rune by rune measure of D39, all 32 tables with
border 2 in the emoji sets fail it.

### D41. tblfmt counts an emoji variation sequence as 2 columns until go-runewidth does. Amends D39.

Certainty: High.

Date: 2026-10-28.

D39 left one case wrong: go-runewidth v0.0.31 counts a narrow base followed
by U+FE0F, such as U+2764 U+FE0F, as 1 column. Ken saw the emoji golden
tables in a terminal, and the borders after such a sequence did not line up.
He does not maintain go-runewidth, and a `replace` in `go.mod` does not reach
a module that imports tblfmt. So tblfmt has a temporary workaround.

`emojiWidth` in `emoji.go` adds 1 column for each base followed by U+FE0F that
go-runewidth counts as 1 column. The bases come from
`emoji-variation-sequences.txt` in Unicode 18.0.0, which names 371. A sequence
that U+200D joins to another emoji is not counted, because go-runewidth
already counts the joined cluster as 2. `emojiWidth` asks go-runewidth for the
width of each sequence, so it adds nothing once go-runewidth is fixed.

Each golden file is the same, byte for byte, as the output of tblfmt built
with the patch that Ken sends to go-runewidth. `TestGoldenAlignment` counts
these sequences as 2 columns too, so it fails on the golden files that D40
wrote. B38 in [BACKLOG.md](BACKLOG.md) says how to remove the workaround.

### D42. tblfmt does not run a pager, and usql does. Supersedes D10.

Certainty: High.

Date: 2026-10-29.

Ken decided to remove the pager from tblfmt completely. usql runs it in a new
package, `usql/pager`. An encoder writes only to the `io.Writer` that the
caller gives it.

This change removes the options `WithPager`, `WithMinPagerWidth` and
`WithMinPagerHeight`, the funcs `startPager`, `checkErr` and `pagerOpts`, and
the `tableHeight` methods. `FromMap` does not read the keys `pager`,
`pager_cmd` and `pager_min_lines`. It ignores them, as it ignores each key
that it does not know.

The pager in tblfmt had these faults:

1. It ran `exec.Command(pagerCmd)`, so a command with arguments, such as
   `PAGER="less -S"`, failed.
2. Its output went to the writer that usql passed in. That writer is the
   rline session and not the terminal, so `less` acted like `cat`.
3. Only the table and expanded encoders started it. Other formats were never
   paged.
4. It decided from one batch of rows. When a later batch started it, the
   output of the earlier batches was lost.

usql now pages all of its output with one writer. The writer holds the output
until it is taller or wider than the terminal, and then starts the pager on
the terminal. tblfmt has no compatibility promise (D37). usql does not call
the removed options, and it does not set the pager keys. `consolesize-go`
stays, because `expanded: auto` uses it.

### D43. A table escapes only a control character, as psql does. Decided. Amended by D46.

Certainty: High.

Date: 2026-10-30.

`FormatBytes` escaped each rune that `strconv.IsGraphic` rejects. Go counts
a format character, a private use character and an unassigned code point as
not graphic. So tblfmt escaped text that a terminal draws:

1. U+200D in each joined emoji, such as U+1F468 U+200D U+1F469 U+200D
   U+1F467, and in Indic text.
2. U+200C in Persian and Indic text.
3. The tag characters in the flags of England, Scotland and Wales.
4. The signs that span the digits after them, such as U+0600.
5. The private use characters, such as the Nerd Font and Powerline icons.
6. A character that Unicode 18 added. Go 1.27 knows Unicode 17.

psql 18.6 escapes only a control character, U+0000 to U+001F, U+007F and
U+0080 to U+009F, and writes each other rune as it is. The fault came up on
2026-10-30, when the rline session asked about the width of a flag. Ken
decided that tblfmt follows psql. `FormatBytes` now
escapes a rune only when `unicode.IsControl` is true. A control character is
below U+00A0, so the `\U00000000` form went.

tblfmt now also writes the invisible characters as psql does. These are the
soft hyphen, the zero width space, the word joiner, U+FEFF, U+2028, U+2029
and the direction controls, such as U+202E. A direction control can reverse
part of a line in a terminal that supports text from right to left. psql has
the same risk.

tblfmt measures the text by grapheme cluster with go-runewidth (D39). For two
of these characters, go-runewidth and psql disagree by 1 column. go-runewidth
counts U+0600 as 1 column, and psql counts it as 0. go-runewidth counts U+2028
as 0 columns, and psql counts it as 1. Terminals do not agree either.

`TestEncodeFormatChars` holds the psql command and its output. The golden
files of the emoji sets changed only where they hold U+200D.

### D44. The table encoder draws a table as psql does. Supersedes D14 and D28.

Certainty: High.

Date: 2026-10-31.

Ken answered open questions 1 and 2 on 2026-10-31: tblfmt follows psql. The
table encoder now follows `print_aligned_text` in `src/fe_utils/print.c` of
PostgreSQL 18, one branch at a time:

1. psql pads a left aligned value only when a border, a newline mark or
   another column follows it. So the last column of a table with a border of
   0 or 1 has no trailing space. This replaces D14 and D28, and the README
   section about the trailing space.
2. For a result set that has columns and no rows, tblfmt writes the header
   and the line under it, as psql does. B1 is fixed.
3. A tab moves to the next multiple of 8 columns from the start of the value,
   and tblfmt writes it as spaces. The `tab` field, the column offsets and the
   `offset` and `tab` arguments of `Value.LineWidth` and `Value.MaxWidth` went.
4. A `LineStyle` holds the parts of a psql line style: a `Rule` for each
   horizontal line, the vertical line before a value that continues, is blank
   or wraps, and the newline marks. The old-ascii style marks a continued value
   with a colon on its left, as psql does.
5. `UnicodeLineStyle` takes the psql variables `unicode_border_linestyle`,
   `unicode_header_linestyle` and `unicode_column_linestyle`, and `FromMap`
   reads all three. `UnicodeDoubleLineStyle` went. psql draws the lines between
   the columns single when only the border is double.
6. The title is centered over the width that psql calculates. The end border
   is drawn once, after the last batch. B4 is fixed.
7. `WithInline` went. A rule with no line draws nothing, so the `table` format
   of D21 needs no option.

The rows, the column names and the border lines of every aligned and expanded
golden case are the same as psql 18.6 output, apart from the blank line of
D48. A temporary program checked this on 2026-10-31. It also compared 243
cases of real queries, such as empty results, titles, `tuples_only`,
column names with newlines and control characters, with each line style. The
emoji cases differ only in the padding of D39.

### D45. The expanded encoder draws records as psql does. Amends D8.

Certainty: High.

Date: 2026-10-31.

Ken answered open question 3 on 2026-10-31: tblfmt follows psql. The
expanded encoder now follows `print_aligned_vertical` in
`src/fe_utils/print.c`:

1. psql writes the row count only for a result set that has no rows or no
   columns. It writes `(0 rows)` for no rows, and `(1 row)` for one row with
   no columns. A summary that the caller sets follows the records, with a
   blank line before it when the border is less than 2. The automatic switch
   of D8 follows the same rule. B7 is fixed.
2. The record header line has no extra dash at the end, and a line has no
   space at the start when the border is 1.
3. The values are at least as wide as the record header line needs. psql
   counts the digits of the last record number for this.
4. The records are numbered across batches. Before, each batch started again
   at record 1.
5. The custom summary goes to the buffered writer, after the records. B2 is
   fixed.

### D46. Escapes follow psql, and Unicode text is drawn as a terminal draws it. Amends D43.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that tblfmt follows psql in escaping. In a table, a
control character below U+0080 is `\x` and two upper case hex digits, such
as `\x07`, and one from U+0080 is `\u` and four upper case hex digits.
U+000D stays `\r`. Before, tblfmt wrote `\a`, `\b`, `\v` and `\f`, and lower
case hex digits. B39 is fixed. JSON keeps its own escapes.

Ken also decided that tblfmt differs from psql where that draws Unicode text
correctly and lines it up in a terminal, such as emoji and flags. D39 and D41
are cases of this rule. A new case follows the same rule, and needs a
decision of its own only where it changes what tblfmt escapes.

### D47. The caller sets the separator between result sets, and JSON is compact. Supersedes D30.

Certainty: High.

Date: 2026-10-31.

Ken answered open question 4 on 2026-10-31. Each encoder has the option
`WithResultSetSeparator`, so that the caller decides what `EncodeAll` writes
between two result sets. The default of each encoder is the newline, as
before.

The JSON encoder writes compact JSON by default, with each result set as one
array on its own line, as JSON Lines does. The option `WithIndent` indents
the output as D30 did. No comma joins two arrays now. B18 is fixed.

### D48. usql writes the blank line after a table. Amends D17.

Certainty: High.

Date: 2026-10-31.

Ken answered open question 5 on 2026-10-31. psql writes one blank line after
an aligned table, and tblfmt does not. This is a deliberate difference.
tblfmt makes the table formats of psql, but usql is the client that matches
psql, so usql writes the line. "Differences from `psql`" in `README.md` lists
it.

### D49. The lint configuration follows the other xo repositories. Decided.

Certainty: High.

Date: 2026-10-31.

Ken answered open question 8 on 2026-10-31. `.golangci.yml` follows dburl,
dbmeta and dbimp: `default: all`, with each disabled linter and its reason
beside it. depguard allows only the standard library and the three modules
of hard rule 6. The formatters are gofmt and goimports. A lint job in CI
installs golangci-lint v2.14.0, which is pinned so that a new release cannot
turn on a linter that nobody chose. The job also runs `gofmt -l .`.

The findings that were real defects got a code change. A template writer did
not check its writes, so `TemplateEncoder.Encode` now returns the first write
error. The csv encoder built a quoted field with `append`, which could write
into the buffer of a value, and now uses `slices.Concat`. B31, B35 and B37 are
fixed.

### D50. Dependabot stays off. Decided.

Certainty: High.

Date: 2026-10-31.

Ken answered open question 6 on 2026-10-31. `62d9721` removed
`.github/dependabot.yml`, and it stays off. D32 ties each dependency to the
version that usql requires, and Dependabot would move a dependency past it.

### D51. tblfmt has no Numeric type. Decided. Amended by D63.

Certainty: High.

Date: 2026-10-31.

On 2026-09-27, usql asked for an exported `type Numeric string`, so that a
numeric or money column is right aligned as in psql. Ken decided on
2026-10-31 that `Numeric` is not the right design, and that tblfmt does not
add it. He has not chosen another design yet.

### D52. The csv encoder quotes a field that holds a space or a tab. Decided.

Certainty: High.

Date: 2026-10-31.

psql 18.6 quotes a csv field only when it holds the separator, a carriage
return, a line feed or a double quote, when it is exactly `\.`, or when the
separator is `\` or `.`. The source is `csv_print_field` in
`src/fe_utils/print.c`. tblfmt also quotes a field that holds a space or a
tab. For the golden case `multi/28`, psql writes `14,a<TAB>b<TAB>c<TAB>d,x`,
and tblfmt writes `14,"a<TAB>b<TAB>c<TAB>d",x`.

Ken decided on 2026-10-31 that psql is wrong here, and that tblfmt keeps
quoting such a field. "Differences from `psql`" in `README.md` lists it.

### D53. The caller chooses the separator between unaligned result sets. Decided.

Certainty: High.

Date: 2026-10-31.

psql 18.6 writes no line between two result sets in the unaligned and csv
formats. tblfmt writes the newline by default. Ken decided on 2026-10-31 that
the client chooses this with `WithResultSetSeparator` (D47), so `FromMap` does
not set it. "Differences from `psql`" in `README.md` lists it.

### D54. The template formats escape a control character. Decided.

Certainty: High.

Date: 2026-10-31.

psql 18.6 escapes only its own markup in the html and asciidoc formats, and
writes a control character as it is. The source is `html_escaped_print` and
`asciidoc_escaped_print` in `src/fe_utils/print.c`. tblfmt escapes a control
character in these formats as it does in a table, so U+0008 is `\x08`.

Ken decided on 2026-10-31 that the escapes are correct and psql is wrong
here. "Differences from `psql`" in `README.md` lists it.

### D55. WithFormatterOptions returns an error for a formatter it cannot change. Decided.

Certainty: High.

Date: 2026-10-31.

`WithFormatterOptions` asserted that the formatter of the encoder is an
`EscapeFormatter`, and did not check the assertion. After `WithFormatter` set
a formatter of another type, the encoder panicked. Ken decided on 2026-10-31
to fix this without a change to the API. The option now returns
`ErrFormatterIsNotEscapeFormatter`, and the constructor of the encoder returns
that error. `TestWithFormatterOptionsError` holds the cases.

### D56. The wrapped format is an option of the table encoder. Decided.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that the psql format `wrapped` is an option of the
table encoder, and not an encoder of its own. `WithWrapWidth` sets the width
for the table and the expanded encoders. `FromMap` reads `format: wrapped`,
and takes the width from `columns` or from the terminal, as psql does.

The encoders follow psql in `src/fe_utils/print.c`. The table encoder makes
the column with the highest ratio of its maximum width to its average width
narrower, one column at a time, until the table fits. A column is never
narrower than its name. The expanded encoder fits the values in the width,
and keeps at least 3 columns for a value. A wrapped line ends with the wrap
mark of the line style, such as `.` in ascii and `…` in unicode. `LineStyle`
has `WrapLeft` and `WrapRight` for these marks. The cut falls between two
grapheme clusters, and go-runewidth finds the clusters (D39).

One difference from psql follows D46. psql can make a column 1 wide when it
holds a character 2 columns wide, and the line with that character is then 1
column too wide. tblfmt keeps such a column at least 2 wide.

The same work found a fault in the switch of D8: psql does not switch a table
of one column to expanded records, because the records are wider than the
table. tblfmt now does the same.

A temporary program compared 559 cases with psql 18.6 on 2026-10-31, and each
one matched. `TestEncodePsqlFormats` holds a case with the psql output.

### D57. tblfmt writes each psql format. Decided.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that tblfmt supports each format that psql 18
names: aligned, asciidoc, csv, html, latex, latex-longtable, troff-ms,
unaligned and wrapped. `FromMap` takes each of them. The formats json,
vertical and table are tblfmt formats.

`WriteLaTeXTo`, `WriteLaTeXLongtableTo` and `WriteTroffMSTo` in `template.go`
follow `print_latex_text`, `print_latex_longtable_text`,
`print_latex_vertical`, `print_troff_ms_text` and `print_troff_ms_vertical`.
They read the new `Template` fields `Aligns`, `Border`, `Expanded` and
`Footer`. `WithBorder`, `WithSummary` and the new `WithExpanded` work for the
template encoder, and `FromMap` passes `border`, `tuples_only`, `footer` and
`expanded` to it. A control character in a value is escaped first, as D54
says, and then the writer adds the escapes of LaTeX or troff. B15 and B16 are
fixed.

A temporary program compared 216 cases of the three formats with psql 18.6
on 2026-10-31. Each case matched, apart from the 33 cases of a result set
that has no rows. B44 records those.

### D58. The JSON encoder ignores WithSummary and WithWidths. Decided.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that the JSON encoder does nothing for the options
`WithSummary` and `WithWidths`. The FIXME in `WithSummary` asked for a summary
in each encoder. The template encoder takes a summary since D57, so B26 is
fixed.

### D59. tblfmt has benchmarks, and a module that compares it with other packages. Decided.

Certainty: High.

Date: 2026-10-31.

Ken asked on 2026-10-31 for benchmarks, and for a `test` module with its own
`go.mod` that compares tblfmt with other packages that write tables.

`BenchmarkEncode` in `bench_test.go` measures each format at 10,000 and
1,000,000 rows, and the table formats with a batch of 0 and of 1000 rows. Its
result set makes each row when the encoder scans it, so the allocations are
the encoder's. This is the first step of B34.

`BenchmarkWrite` in `test/compare_test.go` writes the same table with tblfmt,
`text/tabwriter`, `tablewriter`, `go-pretty`, `lipgloss`, `simpletable`,
`rodaine/table` and `aquasecurity/table`. `TestWrite` runs in CI, so that a
change to the API of one of them is found. The module replaces tblfmt with
the code of the checkout. Hard rule 6 is about the root module, and Ken's
request covers the packages of the `test` module. `test/.golangci.yml` allows
them in depguard. "Comparison with other packages" in `README.md` holds the
features and the results.

A comparison of the staged code with v0.20.0 on 2026-10-31 found the aligned,
expanded, JSON and html formats 6% to 18% faster, and the expanded format
with 43% fewer allocations. The test of D43 for a character that prints,
`unicode.IsControl` and not `strconv.IsGraphic`, gave most of the gain.

### D60. The vertical format is the `\G` layout of the mysql client, with the rules of psql. Decided.

Certainty: High.

Date: 2026-10-31.

Ken stated on 2026-10-31 that the `vertical` format is the layout that the
`mysql` command line client writes for a query that ends with `\G`. tblfmt
has it so that a usql user has each feature of the other clients, or more.
The mysql client is not a reference for the bytes, as psql is (hard rule 1).
tblfmt keeps the rules of psql in this format too, such as the null text of
the `null` variable, and the escapes of D54. A caller can change them with
the options.

The client writes `print_table_data_vertically` in `client/mysql.cc`, and
MySQL and MariaDB have the same code. Before each row it writes
`*************************** N. row ***************************`. It writes
each column as `%*s: `, with the name right aligned to the longest name, and
then the value, or `NULL` for a null value.

Ken decided on 2026-10-31 that tblfmt right aligns the names too. It aligns
them by display width and not by bytes, as D46 asks. With `tuples_only`, it
writes no names, as mysql does with `--skip-column-names`.
`TestEncodeVertical` holds the output of the MariaDB 13.0.2 client.

### D61. The options stay small, and a caller changes the escapes with a formatter. Decided.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that `WithWidths` sets the width of a column only in
the table encoders, because only they pad a column. The unaligned and
template encoders ignore it, as the JSON encoder does (D58). The FIXME
comments that asked for widths there went. B27 and B28 are closed.

A caller that wants other escapes passes a formatter of its own with
`WithFormatter`. tblfmt adds no option to choose the escaped characters. B24
is closed.

### D62. The tests run on Linux, Windows and macOS. Decided.

Certainty: High.

Date: 2026-10-31.

Ken asked on 2026-10-31 for CI on Windows and macOS. The test job runs on
`ubuntu-latest`, `windows-latest` and `macos-latest`.

An encoder writes the newline of the platform by default, which is `\r\n` on
Windows (`tblfmt.go`). `TestMain` in `main_test.go` sets the newline to `\n`,
so that the golden files and each expected string hold on every platform, and
`TestPlatformNewline` tests the default. `loadTests` takes the number of a
golden case from the base name of its file, because `filepath.Walk` returns
the separator of the platform. `.gitattributes` keeps every text file LF on
every checkout, and keeps the golden files and `testdata/crosstab.txt` as
they are. B32 is closed.

### D63. A caller sets the alignment of a column by its type. Decided. Amends D51.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that a caller needs a general way to set the
alignment of a column by its type, before the encoder reads a row, and not a
`Numeric` type (D51). `WithColumnAlign` takes a func that gets the
`*sql.ColumnType` of each column, and returns the alignment and true, or false
to align the column by its values. The table and template encoders read it.
The alignment holds for each value of the column, a null value too, and for a
result set that has no rows. B44 is fixed.

A driver can return a number as a string, such as a PostgreSQL `numeric`. With
`WithColumnAlign`, usql can right align such a column, as psql does in
`column_type_alignment` of `src/bin/psql/common.c`. `psqlAlign` in
`test/psql_test.go` is an example. `TestWithColumnAlign` tests it with a
driver of its own.

### D64. The html and asciidoc formats follow psql, and each format has its expanded form. Decided.

Certainty: High.

Date: 2026-10-31.

Ken decided on 2026-10-31 that tblfmt compares the html and asciidoc formats
with psql and follows it, and that each format has the expanded form that
psql has. `WriteHTMLTo` and `WriteAsciidocTo` are now ports of
`print_html_text`, `print_html_vertical`, `print_asciidoc_text` and
`print_asciidoc_vertical`. They read the border, the title, the footer, the
alignments and `tuples_only`. B11, B12, B13 and B14 are fixed. The escapes of
D54 stay.

The unaligned and csv encoders have the expanded forms of
`print_unaligned_vertical` and `print_csv_vertical`. `WithExpanded` sets it,
and `FromMap` passes `expanded: on` to them. The unaligned format writes the
title, as psql does, and csv does not. B45 is fixed.

### D65. TestPsql compares tblfmt with psql, on a server that dbrun starts. Decided.

Certainty: High.

Date: 2026-10-31.

Ken asked on 2026-10-31 for a permanent comparison with psql. A test of that
kind was removed in 2021 (D1). `internal.PsqlEncode`, which it called, was
removed on 2026-10-31, because nothing called it.
`TestPsql` in `test/psql_test.go` runs each query with psql 18 and with tblfmt,
on the same server, and compares the bytes. It reads the URL of the server
from `DBMETA_POSTGRES`, and skips when that is not set. The queries are plain
SELECT statements that make no table. Each case leaves out what a decision
makes different from psql: D39, D48, D52 and D54.

The server comes from `dbrun` in dbmeta, as dbmeta D68 says each database
must. The job `psql` in CI checks out dbmeta at the commit that dbimp pins,
installs psql 18, starts `postgres-18`, runs `TestPsql` and removes the
server. On 2026-10-31, each of 1911 cases matched psql 18.6.

### D66. A later batch can be wider than the header. Decided.

Certainty: High.

Date: 2026-10-31.

D2 reads the rows in batches, and draws the header with the widths of the
first batch. A later batch that holds a wider value is wider than the header
above it. Ken decided this when he wrote the package, and confirmed it on
2026-10-31: it is the cost of streaming. A usql user seldom streams a large
query to the terminal, and usually stops it before the table changes width.
B19 is closed.

## Open questions for Ken

An open question is not a decision. When Ken answers one, it becomes a
decision above, and the question goes.

There is no open question.
