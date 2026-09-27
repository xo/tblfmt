package tblfmt

import (
	"database/sql"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/nathan-fiscaletti/consolesize-go"
)

// Builder is a func that creates an encoder for a result set.
type Builder = func(ResultSet, ...Option) (Encoder, error)

// Summary maps a row count to the func that writes the summary for that
// count. The key -1 holds the func for any other count.
type Summary = map[int]func(io.Writer, int) (int, error)

// Option is an encoder option.
type Option interface {
	apply(any) error
}

// option holds the funcs that set an option on each type of encoder.
type option struct {
	table     func(*TableEncoder) error
	expanded  func(*ExpandedEncoder) error
	json      func(*JSONEncoder) error
	unaligned func(*UnalignedEncoder) error
	template  func(*TemplateEncoder) error
	crosstab  func(*CrosstabView) error
	err       func(*errEncoder) error
}

// apply applies the option.
func (opt option) apply(o any) error {
	switch v := o.(type) {
	case *TableEncoder:
		if opt.table != nil {
			return opt.table(v)
		}
		return nil
	case *ExpandedEncoder:
		if opt.expanded != nil {
			return opt.expanded(v)
		}
		return nil
	case *JSONEncoder:
		if opt.json != nil {
			return opt.json(v)
		}
		return nil
	case *UnalignedEncoder:
		if opt.unaligned != nil {
			return opt.unaligned(v)
		}
		return nil
	case *TemplateEncoder:
		if opt.template != nil {
			return opt.template(v)
		}
		return nil
	case *CrosstabView:
		if opt.crosstab != nil {
			return opt.crosstab(v)
		}
		return nil
	case *errEncoder:
		if opt.err != nil {
			return opt.err(v)
		}
		return nil
	}
	panic(fmt.Sprintf("option cannot be applied to %T", o))
}

// FromMap returns the Builder and the options for the format and the other
// parameters in the map.
//
// Note: this func is mainly a helper for parameter names that are like the
// format option names of psql.
func FromMap(opts map[string]string) (Builder, []Option) {
	// unaligned, aligned, wrapped, html, asciidoc, latex, latex-longtable, troff-ms, json, csv
	switch format := opts["format"]; format {
	case "json":
		return NewJSONEncoder, []Option{
			WithLowerColumnNames(opts["lower_column_names"] == "true"),
			WithUseColumnTypes(opts["use_column_types"] == "true"),
			FormatterOptionFromMap(opts),
		}
	case "csv", "unaligned":
		// determine separator, quote
		enc, sep, quote, field := NewUnalignedEncoder, '|', rune(0), "fieldsep"
		if format == "csv" {
			enc, sep, quote, field = NewCSVEncoder, ',', '"', "csv_fieldsep"
		}
		if s, ok := opts[field]; ok {
			r := []rune(s)
			if len(r) != 1 {
				err := ErrInvalidFieldSeparator
				if format == "csv" {
					err = ErrInvalidCSVFieldSeparator
				}
				return newErrEncoder, []Option{withError(err)}
			}
			sep = r[0]
		}
		if format != "csv" && opts["fieldsep_zero"] == "on" {
			sep = 0
		}
		// determine newline
		recordsep := newline
		if rs, ok := opts["recordsep"]; ok {
			recordsep = []byte(rs)
		}
		if opts["recordsep_zero"] == "on" {
			recordsep = []byte{0}
		}
		tableOpts := []Option{
			WithSeparator(sep),
			WithQuote(quote),
			WithFormatter(NewEscapeFormatter(WithIsRaw(true, sep, quote))),
			WithNewline(string(recordsep)),
			WithTitle(opts["title"]),
			WithEmpty(opts["null"]),
			WithSkipHeader(opts["tuples_only"] == "on"),
			WithLowerColumnNames(opts["lower_column_names"] == "true"),
			WithUseColumnTypes(opts["use_column_types"] == "true"),
			FormatterOptionFromMap(opts),
		}
		if opts["tuples_only"] == "on" {
			opts["footer"] = "off"
		}
		if s, ok := opts["footer"]; ok && s == "off" {
			// an empty summary map turns off the summary
			tableOpts = append(tableOpts, WithSummary(Summary{}))
		}
		return enc, tableOpts
	case "html", "asciidoc", "latex", "latex-longtable", "troff-ms", "vertical":
		return NewTemplateEncoder, []Option{
			WithTemplate(format),
			WithTableAttributes(opts["tableattr"]),
			WithTitle(opts["title"]),
			WithEmpty(opts["null"]),
			WithLowerColumnNames(opts["lower_column_names"] == "true"),
			WithUseColumnTypes(opts["use_column_types"] == "true"),
			FormatterOptionFromMap(opts),
		}
	case "table":
		tableOpts := []Option{
			WithForceUpperColumnNames(true),
			WithBorder(0),
			WithLineStyle(TableLineStyle()),
			WithInline(true),
			WithFormatterOptions(
				WithHeaderAlign(AlignLeft),
				WithAlign(AlignLeft),
			),
			FormatterOptionFromMap(opts),
		}
		if s, ok := opts["tuples_only"]; ok && s == "on" {
			tableOpts = append(tableOpts, WithSkipHeader(true))
			opts["footer"] = "off"
		}
		if s, ok := opts["null"]; ok {
			tableOpts = append(tableOpts, WithEmpty(s))
		}
		if s, ok := opts["footer"]; ok && s == "off" {
			// an empty summary map turns off the summary
			tableOpts = append(tableOpts, WithSummary(Summary{}))
		}
		tableOpts = pagerOpts(tableOpts, opts)
		return NewTableEncoder, tableOpts
	case "aligned":
		tableOpts := []Option{
			WithLowerColumnNames(opts["lower_column_names"] == "true"),
			WithUseColumnTypes(opts["use_column_types"] == "true"),
			FormatterOptionFromMap(opts),
		}
		if s, ok := opts["border"]; ok {
			border, _ := strconv.Atoi(s)
			tableOpts = append(tableOpts, WithBorder(border))
		}
		if s, ok := opts["tuples_only"]; ok && s == "on" {
			tableOpts = append(tableOpts, WithSkipHeader(true))
			opts["footer"] = "off"
		}
		if s, ok := opts["title"]; ok {
			tableOpts = append(tableOpts, WithTitle(s))
		}
		if s, ok := opts["null"]; ok {
			tableOpts = append(tableOpts, WithEmpty(s))
		}
		if s, ok := opts["footer"]; ok && s == "off" {
			// an empty summary map turns off the summary
			tableOpts = append(tableOpts, WithSummary(Summary{}))
		}
		if s, ok := opts["linestyle"]; ok {
			switch s {
			case "ascii":
				tableOpts = append(tableOpts, WithLineStyle(ASCIILineStyle()))
			case "old-ascii":
				tableOpts = append(tableOpts, WithLineStyle(OldASCIILineStyle()))
			case "unicode":
				switch opts["unicode_border_linestyle"] {
				case "single":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeLineStyle()))
				case "double":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeDoubleLineStyle()))
				}
			}
		}
		tableOpts = pagerOpts(tableOpts, opts)
		builder := NewTableEncoder
		if e, ok := opts["expanded"]; ok {
			switch e {
			case "auto":
				cols, _ := consolesize.GetConsoleSize()
				if cstr, ok := opts["columns"]; ok && cstr != "" {
					if c, err := strconv.ParseUint(cstr, 10, 32); err == nil && c != 0 {
						cols = int(c)
					}
				}
				tableOpts = append(tableOpts, WithMinExpandWidth(cols+1))
			case "on":
				builder = NewExpandedEncoder
			}
		}
		return builder, tableOpts
	}
	return newErrEncoder, []Option{withError(ErrInvalidFormat)}
}

// FormatterOptionFromMap builds an option that sets the formatter options
// from the parameters in the map.
func FormatterOptionFromMap(opts map[string]string) Option {
	// time format
	timeFormat := opts["time"]
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}
	// time location
	var timeLocation *time.Location
	if tz := opts["timezone"]; tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			timeLocation = loc
		}
	}
	// numeric locale
	locale := opts["locale"]
	if locale == "" {
		locale = "en-US"
	}
	numericLocale := opts["numericlocale"] == "true" || opts["numericlocale"] == "on"
	return WithFormatterOptions(
		WithTimeFormat(timeFormat),
		WithTimeLocation(timeLocation),
		WithNumericLocale(numericLocale, locale),
	)
}

// WithCount is an encoder option that sets the number of rows to buffer.
func WithCount(count int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.count = count
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.count = count
			return nil
		},
	}
}

// WithLineStyle is an encoder option that sets the line style of the table.
func WithLineStyle(lineStyle LineStyle) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.lineStyle = lineStyle
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.lineStyle = lineStyle
			return nil
		},
	}
}

// WithFormatter is an encoder option that sets the formatter for values.
func WithFormatter(formatter Formatter) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.formatter = formatter
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.formatter = formatter
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.formatter = formatter
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.formatter = formatter
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.formatter = formatter
			return nil
		},
		crosstab: func(view *CrosstabView) error {
			view.formatter = formatter
			return nil
		},
	}
}

// WithFormatterOptions is an encoder option that adds more formatter
// options.
func WithFormatterOptions(opts ...EscapeFormatterOption) Option {
	apply := func(formatter Formatter) {
		f := formatter.(*EscapeFormatter)
		for _, o := range opts {
			o(f)
		}
	}
	return option{
		table: func(enc *TableEncoder) error {
			apply(enc.formatter)
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			apply(enc.formatter)
			return nil
		},
		json: func(enc *JSONEncoder) error {
			apply(enc.formatter)
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			apply(enc.formatter)
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			apply(enc.formatter)
			return nil
		},
		crosstab: func(view *CrosstabView) error {
			apply(view.formatter)
			return nil
		},
	}
}

// WithSummary is an encoder option that sets the summary of the table.
func WithSummary(summary Summary) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.summary = summary
			enc.isCustomSummary = true
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.summary = summary
			enc.isCustomSummary = true
			return nil
		},
		// FIXME: all of these encoders need a summary option too
		json: func(*JSONEncoder) error {
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.summary = summary
			return nil
		},
		template: func(*TemplateEncoder) error {
			return nil
		},
	}
}

// WithSkipHeader is an encoder option that stops the encoder from writing the
// header.
func WithSkipHeader(s bool) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.skipHeader = s
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.skipHeader = s
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.skipHeader = s
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.skipHeader = s
			return nil
		},
	}
}

// WithInline is an encoder option that writes the header inline with the top
// line.
func WithInline(inline bool) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.inline = inline
			return nil
		},
	}
}

// WithTitle is an encoder option that sets the title of the table.
func WithTitle(title string) Option {
	encode := func(formatter Formatter, empty *Value) *Value {
		if title == "" {
			return nil
		}
		if v, err := formatter.Header([]string{title}); err == nil {
			return v[0]
		}
		return empty
	}
	return option{
		table: func(enc *TableEncoder) error {
			enc.title = encode(enc.formatter, enc.empty)
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.title = encode(enc.formatter, enc.empty)
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.title = encode(enc.formatter, enc.empty)
			return nil
		},
	}
}

// WithEmpty is an encoder option that sets the value for empty (nil) cells.
func WithEmpty(empty string) Option {
	encode := func(formatter Formatter) *Value {
		z := new(any)
		*z = empty
		if v, err := formatter.Format([]any{z}); err == nil {
			return v[0]
		}
		panic(fmt.Sprintf("invalid empty value %q", empty))
	}
	return option{
		table: func(enc *TableEncoder) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
		crosstab: func(enc *CrosstabView) error {
			enc.empty = encode(enc.formatter)
			return nil
		},
	}
}

// WithWidths is an encoder option that sets the (minimum) width of each column.
func WithWidths(widths ...int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.widths = widths
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.widths = widths
			return nil
		},
		unaligned: func(*UnalignedEncoder) error {
			// FIXME: add support for minimum column widths to the
			// unaligned encoder
			// enc.widths = widths
			return nil
		},
		template: func(*TemplateEncoder) error {
			// FIXME: add support for minimum column widths to the
			// template encoder
			// enc.widths = widths
			return nil
		},
	}
}

// WithSeparator is an encoder option that sets the field separator.
func WithSeparator(sep rune) Option {
	return option{
		unaligned: func(enc *UnalignedEncoder) error {
			enc.sep = sep
			return nil
		},
	}
}

// WithQuote is an encoder option that sets the quote character for fields.
func WithQuote(quote rune) Option {
	return option{
		unaligned: func(enc *UnalignedEncoder) error {
			enc.quote = quote
			return nil
		},
	}
}

// WithNewline is an encoder option that sets the newline.
func WithNewline(newline string) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.newline = []byte(newline)
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.newline = []byte(newline)
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.newline = []byte(newline)
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.newline = []byte(newline)
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.newline = []byte(newline)
			return nil
		},
	}
}

// WithBorder is an encoder option that sets the border size.
func WithBorder(border int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.border = border
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.border = border
			return nil
		},
	}
}

// WithTableAttributes is an encoder option that sets the table attributes.
func WithTableAttributes(a string) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			enc.attributes = a
			return nil
		},
	}
}

// WithExecutor is an encoder option that sets the executor.
func WithExecutor(executor func(io.Writer, *Template) error) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			enc.executor = executor
			return nil
		},
	}
}

// WithTemplate is an encoder option that sets the template by its name.
func WithTemplate(name string) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			switch name {
			case "html":
				enc.executor = WriteHTMLTo
			case "asciidoc":
				enc.executor = WriteAsciidocTo
			case "vertical":
				enc.executor = WriteVerticalTo
			default:
				return ErrInvalidTemplate
			}
			return nil
		},
	}
}

// WithHeaderTransformer is an encoder option that sets the transform style for
// the header.
func WithHeaderTransformer(headerTransformer Transformer) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.headerTransformer = headerTransformer
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.headerTransformer = headerTransformer
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.headerTransformer = headerTransformer
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.headerTransformer = headerTransformer
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.headerTransformer = headerTransformer
			return nil
		},
		crosstab: func(view *CrosstabView) error {
			view.headerTransformer = headerTransformer
			return nil
		},
	}
}

// WithLowerColumnNames is an encoder option that changes the column names to
// lower case when they are all upper case. See [TransformUpperToLower].
func WithLowerColumnNames(lowerColumnNames bool) Option {
	transformStyle := TransformNone
	if lowerColumnNames {
		transformStyle = TransformUpperToLower
	}
	return WithHeaderTransformer(transformStyle)
}

// WithForceUpperColumnNames is an encoder option that changes all column names
// to upper case. See [TransformForceUpper].
func WithForceUpperColumnNames(forceUpper bool) Option {
	transformStyle := TransformNone
	if forceUpper {
		transformStyle = TransformForceUpper
	}
	return WithHeaderTransformer(transformStyle)
}

// WithColumnTypes is an encoder option that sets the func that builds the
// column types.
func WithColumnTypes(columnTypes func(ResultSet, []any, int) error) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.columnTypes = columnTypes
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.columnTypes = columnTypes
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.columnTypes = columnTypes
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.columnTypes = columnTypes
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.columnTypes = columnTypes
			return nil
		},
		crosstab: func(view *CrosstabView) error {
			view.columnTypes = columnTypes
			return nil
		},
	}
}

// WithUseColumnTypes is an encoder option that makes the encoder use the column
// types of the result set.
func WithUseColumnTypes(useColumnTypes bool) Option {
	if !useColumnTypes {
		return WithColumnTypes(nil)
	}
	return WithColumnTypes(func(resultSet ResultSet, r []any, n int) error {
		cols, err := resultSetColumns(resultSet, n)
		if err != nil {
			return err
		}
		for i := range n {
			r[i] = reflect.New(cols[i].ScanType()).Interface()
		}
		return nil
	})
}

// WithColumnTypesFunc is an encoder option that sets a func that builds the
// type of each column.
func WithColumnTypesFunc(f func(*sql.ColumnType) (any, error)) Option {
	return WithColumnTypes(func(resultSet ResultSet, r []any, n int) error {
		cols, err := resultSetColumns(resultSet, n)
		if err != nil {
			return err
		}
		for i := range n {
			if r[i], err = f(cols[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// WithParams is a view option that sets the column parameters.
func WithParams(params ...string) Option {
	return option{
		crosstab: func(view *CrosstabView) error {
			if len(params) > 4 {
				return ErrInvalidColumnParams
			}
			if len(params) > 0 {
				view.v = params[0]
			}
			if len(params) > 1 {
				view.h = params[1]
			}
			if len(params) > 2 {
				view.d = params[2]
			}
			if len(params) > 3 {
				view.s = params[3]
			}
			return nil
		},
	}
}

// WithMinExpandWidth is an encoder option that sets the maximum width before
// the encoder switches to the expanded format.
func WithMinExpandWidth(w int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.minExpandWidth = w
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.minExpandWidth = w
			return nil
		},
	}
}

// WithMinPagerWidth is an encoder option that sets the maximum width before
// the encoder sends the output to the pager.
func WithMinPagerWidth(w int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.minPagerWidth = w
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.minPagerWidth = w
			return nil
		},
	}
}

// WithMinPagerHeight is an encoder option that sets the maximum height before
// the encoder sends the output to the pager.
func WithMinPagerHeight(h int) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.minPagerHeight = h
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.minPagerHeight = h
			return nil
		},
	}
}

// WithPager is an encoder option that sets the pager command.
func WithPager(p string) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.pagerCmd = p
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.pagerCmd = p
			return nil
		},
	}
}

// withError is an encoder option that forces an error.
func withError(err error) Option {
	return option{
		err: func(enc *errEncoder) error {
			enc.err = err
			return err
		},
	}
}

// resultSetColumns gets the column types from a result set, and makes sure
// that the number of column types is n.
func resultSetColumns(resultSet ResultSet, n int) ([]*sql.ColumnType, error) {
	rs, ok := resultSet.(interface {
		ColumnTypes() ([]*sql.ColumnType, error)
	})
	if !ok {
		return nil, ErrResultSetHasNoColumnTypes
	}
	cols, err := rs.ColumnTypes()
	switch {
	case err != nil:
		return nil, err
	case len(cols) != n:
		return nil, ErrResultSetReturnedInvalidColumnTypes
	}
	return cols, nil
}

// pagerOpts adds pager options to the table options.
func pagerOpts(tableOpts []Option, opts map[string]string) []Option {
	pager, pagerCmd := opts["pager"], opts["pager_cmd"]
	if pager == "" || pagerCmd == "" {
		return tableOpts
	}
	tableOpts = append(tableOpts, WithPager(pagerCmd))
	switch pager {
	case "on":
		cols, rows := consolesize.GetConsoleSize()
		if cstr, ok := opts["columns"]; ok && cstr != "" {
			if c, err := strconv.ParseUint(cstr, 10, 32); err == nil && c != 0 {
				cols = int(c)
			}
		}
		if rstr, ok := opts["pager_min_lines"]; ok && rstr != "" {
			if r, err := strconv.ParseUint(rstr, 10, 32); err == nil && r != 0 {
				rows = int(r)
			}
		}
		tableOpts = append(tableOpts, WithMinPagerWidth(cols+1), WithMinPagerHeight(rows+1))
	case "always":
		tableOpts = append(tableOpts, WithMinPagerWidth(-1), WithMinPagerHeight(-1))
	}
	return tableOpts
}
