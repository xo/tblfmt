package tblfmt

import (
	htmltemplate "html/template"
	"io"
	"strconv"
	"strings"
	texttemplate "text/template"

	"github.com/nathan-fiscaletti/consolesize-go"
	"github.com/xo/tblfmt/templates"
)

// Builder is the shared builder interface.
type Builder = func(ResultSet, ...Option) (Encoder, error)

// Option is a Encoder option.
type Option = func(interface{}) error

// FromMap creates an encoder for the provided result set, applying the named
// options.
//
// Note: this func is primarily a helper func to accommodate psql-like format
// option names.
func FromMap(opts map[string]string) (Builder, []Option) {
	// unaligned, aligned, wrapped, html, asciidoc, latex, latex-longtable, troff-ms, json, csv
	switch format := opts["format"]; format {
	case "json":
		return NewJSONEncoder, nil
	case "csv", "unaligned":
		// determine separator, quote
		sep, quote, field := '|', rune(0), "fieldsep"
		if format == "csv" {
			sep, quote, field = ',', '"', "csv_fieldsep"
		}
		if s, ok := opts[field]; ok {
			if len(s) != 1 {
				return newErrEncoder, []Option{withError(ErrInvalidFieldSeparator)}
			}
			sep = []rune(s)[0]
		}
		if format != "csv" && opts["fieldsep_zero"] == "true" {
			sep = 0
		}
		// determine newline
		recordsep := newline
		if rs, ok := opts["recordsep"]; ok {
			recordsep = []byte(rs)
		}
		if opts["recordsep_zero"] == "true" {
			recordsep = []byte{0}
		}
		return NewUnalignedEncoder, []Option{
			WithSeparator(sep),
			WithQuote(quote),
			WithFormatter(NewEscapeFormatter(WithIsRaw(true, sep, quote))),
			WithNewline(string(recordsep)),
		}
	case "html", "asciidoc", "latex", "latex-longtable", "troff-ms":
		return NewTemplateEncoder, []Option{
			WithTemplate(format),
			WithTableAttributes(opts["tableattr"]),
			WithTitle(opts["title"]),
		}
	case "aligned":
		var tableOpts []Option
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
	default:
		return newErrEncoder, []Option{withError(ErrInvalidFormat)}
	}
}

// WithCount is a encoder option to set the buffered line count.
func WithCount(count int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.count = count
		case *ExpandedEncoder:
			enc.count = count
		}
		return nil
	}
}

// WithLineStyle is a encoder option to set the table line style.
func WithLineStyle(lineStyle LineStyle) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.lineStyle = lineStyle
		case *ExpandedEncoder:
			enc.lineStyle = lineStyle
		}
		return nil
	}
}

// WithFormatter is a encoder option to set a formatter for formatting values.
func WithFormatter(formatter Formatter) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.formatter = formatter
		case *ExpandedEncoder:
			enc.formatter = formatter
		case *UnalignedEncoder:
			enc.formatter = formatter
		case *JSONEncoder:
			enc.formatter = formatter
		}
		return nil
	}
}

// WithSummary is a encoder option to set a summary callback map.
func WithSummary(summary map[int]func(io.Writer, int) (int, error)) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.summary = summary
			enc.isCustomSummary = true
		case *ExpandedEncoder:
			enc.summary = summary
			enc.isCustomSummary = true
		}
		return nil
	}
}

// WithSkipHeader is a encoder option to skip drawing header.
func WithSkipHeader(s bool) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.skipHeader = s
		case *ExpandedEncoder:
			enc.skipHeader = s
		case *UnalignedEncoder:
			enc.skipHeader = s
		}
		return nil
	}
}

// WithInline is a encoder option to set the column headers as inline to the
// top line.
func WithInline(inline bool) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.inline = inline
		}
		return nil
	}
}

// WithTitle is a encoder option to set the title value used.
func WithTitle(title string) Option {
	return func(v interface{}) error {
		var formatter Formatter
		var val *Value
		switch enc := v.(type) {
		case *TableEncoder:
			formatter = enc.formatter
			val = enc.empty
		case *ExpandedEncoder:
			formatter = enc.formatter
			val = enc.empty
		case *TemplateEncoder:
			formatter = enc.formatter
			val = enc.empty
		}
		if title != "" {
			vals, err := formatter.Header([]string{title})
			if err != nil {
				return err
			}
			val = vals[0]
		}
		switch enc := v.(type) {
		case *TableEncoder:
			enc.title = val
		case *ExpandedEncoder:
			enc.title = val
		case *TemplateEncoder:
			enc.title = val
		}
		return nil
	}
}

// WithEmpty is a encoder option to set the value used in empty (nil)
// cells.
func WithEmpty(empty string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			cell := interface{}(empty)
			v, err := enc.formatter.Format([]interface{}{&cell})
			if err != nil {
				return err
			}
			enc.empty = v[0]
		case *ExpandedEncoder:
			cell := interface{}(empty)
			v, err := enc.formatter.Format([]interface{}{&cell})
			if err != nil {
				return err
			}
			enc.empty = v[0]
		}
		return nil
	}
}

// WithWidths is a encoder option to set (minimum) widths for a column.
func WithWidths(widths []int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.widths = widths
		case *ExpandedEncoder:
			enc.widths = widths
		}
		return nil
	}
}

// WithMinExpandWidth is a encoder option to set maximum width before switching
// to expanded format.
func WithMinExpandWidth(w int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.minExpandWidth = w
		case *ExpandedEncoder:
			enc.minExpandWidth = w
		}
		return nil
	}
}

// WithMinPagerWidth is a encoder option to set maximum width before
// redirecting output to pager.
func WithMinPagerWidth(w int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.minPagerWidth = w
		case *ExpandedEncoder:
			enc.minPagerWidth = w
		}
		return nil
	}
}

// WithMinPagerHeight is a encoder option to set maximum height before
// redirecting output to pager.
func WithMinPagerHeight(h int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.minPagerHeight = h
		case *ExpandedEncoder:
			enc.minPagerHeight = h
		}
		return nil
	}
}

// WithPager is a encoder option to set the pager command.
func WithPager(p string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.pagerCmd = p
		case *ExpandedEncoder:
			enc.pagerCmd = p
		}
		return nil
	}
}

// WithSeparator is a encoder option to set the field separator.
func WithSeparator(sep rune) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *UnalignedEncoder:
			enc.sep = sep
		}
		return nil
	}
}

// WithQuote is a encoder option to set the field quote.
func WithQuote(quote rune) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *UnalignedEncoder:
			enc.quote = quote
		}
		return nil
	}
}

// WithNewline is a encoder option to set the newline.
func WithNewline(newline string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.newline = []byte(newline)
		case *ExpandedEncoder:
			enc.newline = []byte(newline)
		case *JSONEncoder:
			enc.newline = []byte(newline)
		case *UnalignedEncoder:
			enc.newline = []byte(newline)
		case *TemplateEncoder:
			enc.newline = []byte(newline)
		}
		return nil
	}
}

// WithBorder is a encoder option to set the border size.
func WithBorder(border int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.border = border
		case *ExpandedEncoder:
			enc.border = border
		}
		return nil
	}
}

// WithTableAttributes is a encoder option to set the table attributes.
func WithTableAttributes(a string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TemplateEncoder:
			enc.attributes = a
		}
		return nil
	}
}

// WithExecutor is a encoder option to set the executor.
func WithExecutor(executor func(io.Writer, interface{}) error) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TemplateEncoder:
			enc.executor = executor
		}
		return nil
	}
}

// WithRawTemplate is a encoder option to set a raw template of either "text"
// or "html" type.
func WithRawTemplate(text, typ string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TemplateEncoder:
			switch typ {
			case "html":
				tpl, err := htmltemplate.New("").Funcs(htmltemplate.FuncMap{
					"attr":    func(s string) htmltemplate.HTMLAttr { return htmltemplate.HTMLAttr(s) },
					"safe":    func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) },
					"toLower": func(s string) htmltemplate.HTML { return htmltemplate.HTML(strings.ToLower(s)) },
					"toUpper": func(s string) htmltemplate.HTML { return htmltemplate.HTML(strings.ToUpper(s)) },
				}).Parse(text)
				if err != nil {
					return err
				}
				enc.executor = tpl.Execute
			case "text":
				tpl, err := texttemplate.New("").Parse(text)
				if err != nil {
					return err
				}
				enc.executor = tpl.Execute
			default:
				return ErrInvalidTemplate
			}
		}
		return nil
	}
}

// WithTemplate is a encoder option to set a named template.
func WithTemplate(name string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TemplateEncoder:
			typ := "text"
			if name == "html" {
				typ = "html"
			}
			buf, err := templates.Templates.ReadFile(name + ".txt")
			if err != nil {
				return err
			}
			return WithRawTemplate(string(buf), typ)(enc)
		}
		return nil
	}
}

// withError is a encoder option to force an error.
func withError(err error) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *errEncoder:
			enc.err = err
		}
		return err
	}
}
