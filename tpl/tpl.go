package tblfmt

// TemplateEncoder is a Go template encoder for a result set.
type TemplateEncoder struct {
	escape bool
}

// NewTemplateEncoder creates a new Go template encoder for a result set.
func NewTemplateEncoder(resultSet ResultSet, opts ...TemplateEncoderOption) *TemplateEncoder {
	return nil
}

// TemplateEncoderOption is a Template encoder option.
type TemplateEncoderOption func(*TemplateEncoder)

// WithEscape is a template encoder option to toggle escape.
func WithEscape(escape bool) TemplateEncoderOption {
	return func(enc *TemplateEncoder) {
		enc.escape = escape
	}
}

// NewASCIIDocEncoder creates a new template encoder with a ascii doc
// template.
func NewASCIIDocEncoder() *TemplateEncoder {
	return nil
}

// NewLatexEncoder creates a new template encoder with a latex (using
// tabular) template.
func NewLatexEncoder() *TemplateEncoder {
	return nil
}

// NewLatexLongtableEncoder creates a new template encoder with a latex,
// longtable template.
func NewLatexLongtableEncoder() *TemplateEncoder {
	return nil
}

// NewTroffMsEncoder creates a new template encoder with a troff-ms
// template.
func NewTroffMsEncoder() *TemplateEncoder {
	return nil
}

// NewHTMLEncoder creates a new template encoder with a HTML template.
func NewHTMLEncoder() *TemplateEncoder {
	return nil
}
