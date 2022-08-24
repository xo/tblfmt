package tblfmt

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"
)

// Formatter is the common interface for formatting values.
type Formatter interface {
	// Header returns a slice of formatted values for the provided headers.
	Header([]string) ([]*Value, error)
	// Format returns a slice of formatted value the provided row values.
	Format([]interface{}) ([]*Value, error)
}

// EscapeFormatter is an escaping formatter, that handles formatting the
// standard Go types.
//
// If Marshaler is not nil, then it will be passed any map[string]interface{}
// and []interface{} values encountered. If nil, then the standard
// encoding/json.Encoder will be used instead.
type EscapeFormatter struct {
	// mask is used to format header values when the formatted value (after
	// trimming spaces) is the empty string.
	//
	// Note: will have %d replaced with the column number (starting at 1).
	mask string
	// timeFormat is the format to use for time values.
	timeFormat string
	// marshaler will be used to marshal map[string]interface{} and
	// []interface{} types.
	//
	// If nil, the standard encoding/json.Encoder will be used instead.
	marshaler func(interface{}) ([]byte, error)
	// prefix is indent prefix used by the JSON encoder when Marshaler is nil.
	prefix string
	// indent is the indent used by the JSON encoder when Marshaler is nil.
	indent string
	// isJSON sets escaping JSON characters.
	isJSON bool
	// escapeHTML sets the JSON encoder used when Marshaler is nil to escape HTML
	// characters.
	escapeHTML bool
	// isRaw sets raw escaping.
	isRaw bool
	// sep is the separator to use for raw (csv) encoding.
	sep rune
	// quote is the quote to use for raw (csv) encoding.
	quote rune
	// invalid is the value used for invalid utf8 runes when escaping.
	invalid []byte
	// invalidWidth is the rune width of the invalid value
	invalidWidth int
	// headerAlign is the default header values alignment
	headerAlign Align
}

// NewEscapeFormatter creates a escape formatter to handle basic Go values,
// such as []byte, string, time.Time. Formatting for map[string]interface{} and
// []interface{} will be passed to a marshaler provided by WithMarshaler,
// otherwise the standard encoding/json.Encoder will be used to marshal those
// values.
func NewEscapeFormatter(opts ...EscapeFormatterOption) *EscapeFormatter {
	f := &EscapeFormatter{
		mask:       "%d",
		timeFormat: time.RFC3339Nano,
		indent:     "  ",
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// Header satisfies the Formatter interface.
func (f *EscapeFormatter) Header(headers []string) ([]*Value, error) {
	n := len(headers)
	res := make([]*Value, n)
	useMask := strings.ContainsRune(f.mask, '%')
	for i := 0; i < n; i++ {
		s := strings.TrimSpace(headers[i])
		if s == "" && useMask {
			s = fmt.Sprintf(f.mask, i+1)
		} else if s == "" {
			s = f.mask
		}
		res[i] = FormatBytes([]byte(s), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		res[i].Align = f.headerAlign
	}
	return res, nil
}

// Format satisfies the Formatter interface.
func (f *EscapeFormatter) Format(vals []interface{}) ([]*Value, error) {
	n := len(vals)
	res := make([]*Value, n)
	// TODO: change time to v.AppendFormat() + pool
	// TODO: use strconv.Format* for numeric times
	// TODO: use pool
	// TODO: allow configurable runes that can be escaped
	// TODO: handler driver.Valuer
	for i := 0; i < n; i++ {
		val := deref(vals[i])
		switch v := val.(type) {
		case nil:
		case bool:
			res[i] = newValue(strconv.FormatBool(v), AlignLeft, true)
		case int:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case int8:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case int16:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case int32:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case int64:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case uint:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case uint8:
			res[i] = &Value{Buf: []byte(string(rune(v))), Width: 1, Align: AlignRight, Raw: true}
		case uint16:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case uint32:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case uint64:
			res[i] = newValue(strconv.FormatInt(int64(v), 10), AlignRight, true)
		case uintptr:
			res[i] = newValue(fmt.Sprintf("(0x%x)", v), AlignRight, true)
		case float32:
			res[i] = newValue(strconv.FormatFloat(float64(v), 'g', -1, 32), AlignRight, true)
		case float64:
			res[i] = newValue(strconv.FormatFloat(v, 'g', -1, 64), AlignRight, true)
		case complex64:
			res[i] = newValue(fmt.Sprintf("%g", v), AlignRight, false)
		case complex128:
			res[i] = newValue(fmt.Sprintf("%g", v), AlignRight, false)
		case []byte:
			res[i] = FormatBytes(v, f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		case string:
			res[i] = FormatBytes([]byte(v), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		case time.Time:
			res[i] = newValue(v.Format(f.timeFormat), AlignLeft, false)
		case sql.NullBool:
			if v.Valid {
				res[i] = newValue(strconv.FormatBool(v.Bool), AlignLeft, true)
			}
		case sql.NullInt32:
			if v.Valid {
				res[i] = newValue(strconv.FormatInt(int64(v.Int32), 10), AlignRight, true)
			}
		case sql.NullInt64:
			if v.Valid {
				res[i] = newValue(strconv.FormatInt(v.Int64, 10), AlignRight, true)
			}
		case sql.NullFloat64:
			if v.Valid {
				res[i] = newValue(strconv.FormatFloat(v.Float64, 'g', -1, 64), AlignRight, true)
			}
		case sql.NullTime:
			if v.Valid {
				res[i] = newValue(v.Time.Format(f.timeFormat), AlignLeft, false)
			}
		case sql.NullString:
			if v.Valid {
				res[i] = FormatBytes([]byte(v.String), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
			}
		case sql.RawBytes:
			res[i] = FormatBytes(v, f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		case fmt.Stringer:
			res[i] = FormatBytes([]byte(v.String()), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		default:
			// TODO: pool
			if f.marshaler != nil {
				buf, err := f.marshaler(v)
				if err != nil {
					return nil, err
				}
				res[i] = &Value{
					Buf: buf,
					Raw: true,
				}
			} else {
				// json encode
				buf := new(bytes.Buffer)
				enc := json.NewEncoder(buf)
				enc.SetIndent(f.prefix, f.indent)
				enc.SetEscapeHTML(f.escapeHTML)
				if err := enc.Encode(v); err != nil {
					return nil, err
				}
				if f.isJSON {
					res[i] = &Value{
						Buf: bytes.TrimSpace(buf.Bytes()),
						Raw: true,
					}
				} else {
					res[i] = FormatBytes(bytes.TrimSpace(buf.Bytes()), f.invalid, f.invalidWidth, false, f.isRaw, f.sep, f.quote)
					res[i].Raw = true
				}
			}
		}
	}
	return res, nil
}

// valueFromBuffer returns a value from a buffer known not to contain
// characters to escape.
func newValue(str string, align Align, raw bool) *Value {
	v := &Value{Buf: []byte(str), Align: align, Raw: raw}
	v.Width = len(v.Buf)
	return v
}

// lowerhex is the set of lower hex characters.
const lowerhex = "0123456789abcdef"

// FormatBytes parses src, saving escaped (encoded) and unescaped runes to a
// Value, along with tab and newline positions in the generated buf.
func FormatBytes(src []byte, invalid []byte, invalidWidth int, isJSON, isRaw bool, sep, quote rune) *Value {
	res := &Value{
		Tabs: make([][][2]int, 1),
	}
	var tmp [4]byte
	var r rune
	var l, w int
	for ; len(src) > 0; src = src[w:] {
		r, w = rune(src[0]), 1
		// lazy decode
		if r >= utf8.RuneSelf {
			r, w = utf8.DecodeRune(src)
		}
		// invalid rune decoded
		if w == 1 && r == utf8.RuneError {
			// replace with invalid (if set), otherwise hex encode
			if invalid != nil {
				res.Buf = append(res.Buf, invalid...)
				res.Width += invalidWidth
				res.Quoted = true
			} else {
				res.Buf = append(res.Buf, '\\', 'x', lowerhex[src[0]>>4], lowerhex[src[0]&0xf])
				res.Width += 4
				res.Quoted = true
			}
			continue
		}
		// handle json encoding
		if isJSON {
			switch r {
			case '\t':
				res.Buf = append(res.Buf, '\\', 't')
				res.Width += 2
				continue
			case '\n':
				res.Buf = append(res.Buf, '\\', 'n')
				res.Width += 2
				continue
			case '\\':
				res.Buf = append(res.Buf, '\\', '\\')
				res.Width += 2
				continue
			case '"':
				res.Buf = append(res.Buf, '\\', '"')
				res.Width += 2
				continue
			}
		}
		// handle raw encoding
		if isRaw {
			n := utf8.EncodeRune(tmp[:], r)
			res.Buf = append(res.Buf, tmp[:n]...)
			res.Width += runewidth.RuneWidth(r)
			switch {
			case r == sep:
				res.Quoted = true
			case r == quote && quote != 0:
				res.Buf = append(res.Buf, tmp[:n]...)
				res.Width += runewidth.RuneWidth(r)
				res.Quoted = true
			default:
				res.Quoted = res.Quoted || unicode.IsSpace(r)
			}
			continue
		}
		// printable character
		if strconv.IsGraphic(r) {
			n := utf8.EncodeRune(tmp[:], r)
			res.Buf = append(res.Buf, tmp[:n]...)
			res.Width += runewidth.RuneWidth(r)
			continue
		}
		switch r {
		// escape \a \b \f \r \v (Go special characters)
		case '\a':
			res.Buf = append(res.Buf, '\\', 'a')
			res.Width += 2
		case '\b':
			res.Buf = append(res.Buf, '\\', 'b')
			res.Width += 2
		case '\f':
			res.Buf = append(res.Buf, '\\', 'f')
			res.Width += 2
		case '\r':
			res.Buf = append(res.Buf, '\\', 'r')
			res.Width += 2
		case '\v':
			res.Buf = append(res.Buf, '\\', 'v')
			res.Width += 2
		case '\t':
			// save position
			res.Tabs[l] = append(res.Tabs[l], [2]int{len(res.Buf), res.Width})
			res.Buf = append(res.Buf, '\t')
			res.Width = 0
		case '\n':
			// save position
			res.Newlines = append(res.Newlines, [2]int{len(res.Buf), res.Width})
			res.Buf = append(res.Buf, '\n')
			res.Width = 0
			// increase line count
			res.Tabs = append(res.Tabs, nil)
			l++
		default:
			switch {
			// escape as \x00
			case r < ' ':
				res.Buf = append(res.Buf, '\\', 'x', lowerhex[byte(r)>>4], lowerhex[byte(r)&0xf])
				res.Width += 4
			// escape as \u0000
			case r > utf8.MaxRune:
				r = 0xfffd
				fallthrough
			case r < 0x10000:
				res.Buf = append(res.Buf, '\\', 'u')
				for s := 12; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, lowerhex[r>>uint(s)&0xf])
				}
				res.Width += 6
			// escape as \U00000000
			default:
				res.Buf = append(res.Buf, '\\', 'U')
				for s := 28; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, lowerhex[r>>uint(s)&0xf])
				}
				res.Width += 10
			}
		}
	}
	return res
}

// Value contains information pertaining to a formatted value.
type Value struct {
	// Buf is the formatted value.
	Buf []byte
	// Newlines are the positions of newline characters in Buf.
	Newlines [][2]int
	// Tabs are the positions of tab characters in Buf, split per line.
	Tabs [][][2]int
	// Width is the remaining width.
	Width int
	// Align indicates value alignment.
	Align Align
	// Raw tracks whether or not the value should be encoded or not.
	Raw bool
	// Quoted tracks whether or not a raw value should be quoted or not (ie,
	// contains a space or non printable character).
	Quoted bool
}

func (v Value) String() string {
	return string(v.Buf)
}

// LineWidth returns the line width (in runes) of line l.
func (v *Value) LineWidth(l, offset, tab int) int {
	var width int
	if l < len(v.Newlines) {
		width += v.Newlines[l][1]
	}
	if len(v.Tabs[l]) != 0 {
		width += tabwidth(v.Tabs[l], offset, tab)
	}
	if l == len(v.Newlines) {
		width += v.Width
	}
	return width
}

// MaxWidth calculates the maximum width (in runes) of the longest line
// contained in Buf, relative to starting offset and the tab width.
func (v *Value) MaxWidth(offset, tab int) int {
	// simple values do not have tabulations
	width := v.Width
	for l := 0; l < len(v.Tabs); l++ {
		width = max(width, v.LineWidth(l, offset, tab))
	}
	return width
}

// Align indicates an alignment direction for a value.
type Align int

// Align values.
const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// String satisfies the fmt.Stringer interface.
func (a Align) String() string {
	switch a {
	case AlignLeft:
		return "Left"
	case AlignRight:
		return "Right"
	case AlignCenter:
		return "Center"
	}
	return fmt.Sprintf("Align(%d)", a)
}

// tabwidth returns the rune width of buf containing tabs from start position
// in buf, a column offset, and given tab width.
func tabwidth(tabs [][2]int, offset, tab int) int {
	// log.Printf("tabs: %v, offset: %d, tab: %d", tabs, offset, tab)
	width := offset
	for i := 0; i < len(tabs); i++ {
		width += tabs[i][1]
		width += (tab - width%tab)
	}
	// log.Printf("res: %d", width-offset)
	return width - offset
}

// EscapeFormatterOption is an escape formatter option.
type EscapeFormatterOption func(*EscapeFormatter)

// WithMask is an escape formatter option to set the mask used for empty
// headers.
func WithMask(mask string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.mask = mask
	}
}

// WithTimeFormat is an escape formatter option to set the time format used for
// time values.
func WithTimeFormat(timeFormat string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.timeFormat = timeFormat
	}
}

// WithMarshaler is an escape formatter option to set a standard Go encoder to
// use for encoding the value.
func WithMarshaler(marshaler func(interface{}) ([]byte, error)) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.marshaler = marshaler
	}
}

// WithIsJSON is an escape formatter option to enable special escaping for JSON
// characters in non-complex values.
func WithIsJSON(isJSON bool) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.isJSON = isJSON
	}
}

// WithJSONConfig is an escape formatter option to set the JSON encoding
// prefix, indent value, and whether or not to escape HTML. Passed to the
// standard encoding/json.Encoder when a marshaler has not been set on the
// escape formatter.
func WithJSONConfig(prefix, indent string, escapeHTML bool) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.prefix, f.indent, f.escapeHTML = prefix, indent, escapeHTML
	}
}

// WithIsRaw is an escape formatter option to enable special escaping for raw
// characters in values.
func WithIsRaw(isRaw bool, sep, quote rune) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.isRaw, f.sep, f.quote = isRaw, sep, quote
	}
}

// WithInvalid is an escape formatter option to set the invalid value used when
// an invalid rune is encountered during escaping.
func WithInvalid(invalid string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.invalid = []byte(invalid)
		f.invalidWidth = runewidth.StringWidth(invalid)
	}
}

// WithHeaderAlign sets the alignment of header values.
func WithHeaderAlign(a Align) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.headerAlign = a
	}
}

// deref dereferences a pointer to an interface.
func deref(v interface{}) interface{} {
	switch z := v.(type) {
	case *interface{}:
		return *z
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val.Interface()
}
