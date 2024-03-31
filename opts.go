package tblfmt

import (
	"database/sql"
	"fmt"
	htmltemplate "html/template"
	"io"
	"reflect"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/nathan-fiscaletti/consolesize-go"
	"github.com/xo/tblfmt/templates"
)

// Builder is the shared builder interface.
type Builder = func(ResultSet, ...Option) (Encoder, error)

// Option is a Encoder option.
type Option interface {
	apply(interface{}) error
}

// option wraps setting an option on an encoder.
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
func (opt option) apply(o interface{}) error {
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

// FromMap creates an encoder for the provided result set, applying the named
// options.
//
// Note: this func is primarily a helper func to accommodate psql-like format
// option names.
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
		sep, quote, field := '|', rune(0), "fieldsep"
		if format == "csv" {
			sep, quote, field = ',', '"', "csv_fieldsep"
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
		return NewUnalignedEncoder, []Option{
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
			// use an empty summary map to skip drawing the footer
			tableOpts = append(tableOpts, WithSummary(map[int]func(io.Writer, int) (int, error){}))
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
		pager := opts["pager"]
		pagerCmd := opts["pager_cmd"]
		if pager != "" && pagerCmd != "" {
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
		}
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

// FormatterOptionFromMap builds formatter encoding options from the named
// options.
func FormatterOptionFromMap(opts map[string]string) Option {
	// time format
	timeFormat := opts["time"]
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}
	// numeric locale
	locale := opts["locale"]
	if locale == "" {
		locale = "en-US"
	}
	numericLocale := opts["numericlocale"] == "true" || opts["numericlocale"] == "on"
	return WithFormatterOptions(WithTimeFormat(timeFormat), WithNumericLocale(numericLocale, locale))
}

// WithCount is a encoder option to set the buffered line count.
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

// WithLineStyle is a encoder option to set the table line style.
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

// WithFormatter is a encoder option to set a formatter for formatting values.
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

// WithFormatterOptions is a encoder option to add additional formatter
// options.
func WithFormatterOptions(opts ...EscapeFormatterOption) Option {
	apply := func(formatter Formatter) error {
		f := formatter.(*EscapeFormatter)
		for _, o := range opts {
			o(f)
		}
		return nil
	}
	return option{
		table: func(enc *TableEncoder) error {
			return apply(enc.formatter)
		},
		expanded: func(enc *ExpandedEncoder) error {
			return apply(enc.formatter)
		},
		json: func(enc *JSONEncoder) error {
			return apply(enc.formatter)
		},
		unaligned: func(enc *UnalignedEncoder) error {
			return apply(enc.formatter)
		},
		template: func(enc *TemplateEncoder) error {
			return apply(enc.formatter)
		},
		crosstab: func(view *CrosstabView) error {
			return apply(view.formatter)
		},
	}
}

// WithSummary is a encoder option to set a summary map.
func WithSummary(summary map[int]func(io.Writer, int) (int, error)) Option {
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
		// FIXME: all of these should have a summary option as well ...
		json: func(enc *JSONEncoder) error {
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			return nil
		},
	}
}

// WithSkipHeader is a encoder option to disable writing a header.
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

// WithInline is a encoder option to set the column headers as inline to the
// top line.
func WithInline(inline bool) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.inline = inline
			return nil
		},
	}
}

// WithTitle is a encoder option to set the table title.
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

// WithEmpty is a encoder option to set the value used in empty (nil)
// cells.
func WithEmpty(empty string) Option {
	encode := func(formatter Formatter) *Value {
		z := new(interface{})
		*z = empty
		if v, err := formatter.Format([]interface{}{z}); err == nil {
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

// WithWidths is a encoder option to set (minimum) widths for a column.
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
		unaligned: func(enc *UnalignedEncoder) error {
			// FIXME: unaligned encoder should be able to support minimum
			// column widths
			// enc.widths = widths
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			// FIXME: template encoder should be able to support minimum column
			// widths
			// enc.widths = widths
			return nil
		},
	}
}

// WithSeparator is a encoder option to set the field separator.
func WithSeparator(sep rune) Option {
	return option{
		unaligned: func(enc *UnalignedEncoder) error {
			enc.sep = sep
			return nil
		},
	}
}

// WithQuote is a encoder option to set the field quote.
func WithQuote(quote rune) Option {
	return option{
		unaligned: func(enc *UnalignedEncoder) error {
			enc.quote = quote
			return nil
		},
	}
}

// WithNewline is a encoder option to set the newline.
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

// WithBorder is a encoder option to set the border size.
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

// WithTableAttributes is a encoder option to set the table attributes.
func WithTableAttributes(a string) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			enc.attributes = a
			return nil
		},
	}
}

// WithExecutor is a encoder option to set the executor.
func WithExecutor(executor func(io.Writer, interface{}) error) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			enc.executor = executor
			return nil
		},
	}
}

// WithRawTemplate is a encoder option to set a raw template of either "text"
// or "html" type.
func WithRawTemplate(text, typ string) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			switch typ {
			case "html":
				tpl, err := htmltemplate.New(typ).Funcs(htmltemplate.FuncMap{
					"attr":    func(s string) htmltemplate.HTMLAttr { return htmltemplate.HTMLAttr(s) },
					"safe":    func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) },
					"toLower": func(s string) htmltemplate.HTML { return htmltemplate.HTML(strings.ToLower(s)) },
					"toUpper": func(s string) htmltemplate.HTML { return htmltemplate.HTML(strings.ToUpper(s)) },
					"inc":     func(i int) int { return i + 1 },
				}).Parse(text)
				if err != nil {
					return err
				}
				enc.executor = tpl.Execute
				return nil
			case "text":
				tpl, err := texttemplate.New(typ).Funcs(texttemplate.FuncMap{
					"inc": func(i int) int { return i + 1 },
				}).Parse(text)
				if err != nil {
					return err
				}
				enc.executor = tpl.Execute
				return nil
			}
			return ErrInvalidTemplate
		},
	}
}

// WithTemplate is a encoder option to set a named template.
func WithTemplate(name string) Option {
	return option{
		template: func(enc *TemplateEncoder) error {
			typ := "text"
			if name == "html" {
				typ = "html"
			}
			buf, err := templates.Templates.ReadFile(name + ".txt")
			if err != nil {
				return err
			}
			return WithRawTemplate(string(buf), typ).apply(enc)
		},
	}
}

// WithLowerColumnNames is a encoder option to lower case column names when
// column names are all caps.
func WithLowerColumnNames(lowerColumnNames bool) Option {
	return option{
		table: func(enc *TableEncoder) error {
			enc.lowerColumnNames = lowerColumnNames
			return nil
		},
		expanded: func(enc *ExpandedEncoder) error {
			enc.lowerColumnNames = lowerColumnNames
			return nil
		},
		json: func(enc *JSONEncoder) error {
			enc.lowerColumnNames = lowerColumnNames
			return nil
		},
		unaligned: func(enc *UnalignedEncoder) error {
			enc.lowerColumnNames = lowerColumnNames
			return nil
		},
		template: func(enc *TemplateEncoder) error {
			enc.lowerColumnNames = lowerColumnNames
			return nil
		},
		crosstab: func(view *CrosstabView) error {
			view.lowerColumnNames = lowerColumnNames
			return nil
		},
	}
}

// WithColumnTypes is a encoder option to set a func to use for building column
// types.
func WithColumnTypes(columnTypes func(ResultSet, []interface{}, int) error) Option {
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

// WithUseColumnTypes is a encoder option to use the result set's column types.
func WithUseColumnTypes(useColumnTypes bool) Option {
	if !useColumnTypes {
		return WithColumnTypes(nil)
	}
	return WithColumnTypes(func(resultSet ResultSet, r []interface{}, n int) error {
		cols, err := resultSetColumns(resultSet, n)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			r[i] = reflect.New(cols[i].ScanType()).Interface()
		}
		return nil
	})
}

// WithColumnTypesFunc is a encoder option to set a func to build each column's
// type.
func WithColumnTypesFunc(f func(*sql.ColumnType) (interface{}, error)) Option {
	return WithColumnTypes(func(resultSet ResultSet, r []interface{}, n int) error {
		cols, err := resultSetColumns(resultSet, n)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if r[i], err = f(cols[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// WithParams is a view option to set the column parameters.
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

// WithMinExpandWidth is a encoder option to set maximum width before switching
// to expanded format.
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

// WithMinPagerWidth is a encoder option to set maximum width before
// redirecting output to pager.
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

// WithMinPagerHeight is a encoder option to set maximum height before
// redirecting output to pager.
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

// WithPager is a encoder option to set the pager command.
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

// withError is a encoder option to force an error.
func withError(err error) Option {
	return option{
		err: func(enc *errEncoder) error {
			enc.err = err
			return err
		},
	}
}

// resultSetColumns retrieves the columns from a result set and checks the
// length.
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
