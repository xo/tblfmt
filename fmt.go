package tblfmt

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Formatter is the common interface for formatting values.
type Formatter interface {
	// Header returns a slice of formatted values for the provided headers.
	Header([]string) ([]*Value, error)
	// Format returns a slice of formatted value the provided row values.
	Format([]any) ([]*Value, error)
}

// EscapeFormatter is an escaping formatter, that handles formatting the
// standard Go types.
//
// When the encoder is not nil, then it will be passed any
// map[string]interface{} and []interface{} values encountered, otherwise the
// stdlib's [encoding/json/v2] will be used.
type EscapeFormatter struct {
	// mask is used to format header values when the formatted value (after
	// trimming spaces) is the empty string.
	//
	// Note: will have %d replaced with the column number (starting at 1).
	mask string
	// timeFormat is the format to use for time values.
	timeFormat string
	// timeLocation is the location to use for time values.
	timeLocation *time.Location
	// encoder will be used to encode map[string]interface{} and []interface{}
	// types.
	//
	// If nil, the standard [encoding/json/v2] will be used instead.
	encoder func(any) ([]byte, error)
	// prefix is indent prefix used by the JSON encoder when encoder is nil.
	prefix string
	// indent is the indent used by the JSON encoder when encoder is nil.
	indent string
	// isJSON sets escaping JSON characters.
	isJSON bool
	// escapeHTML sets the JSON encoder used when encoder is nil to escape HTML
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
	// align is the forced alignment value.
	align Align
	// numericLocalePrinter is the numeric locale printer.
	numericLocalePrinter *message.Printer
}

// NewEscapeFormatter creates a escape formatter to handle basic Go values,
// such as []byte, string, time.Time, sql.Null*, and any
// [database/sql/driver.Valuer]. Formatting for
// map[string]interface{} and []interface{} will be passed to a marshaler
// provided by [WithEncoder], otherwise the standard [encoding/json/v2] will
// be used to marshal those values.
func NewEscapeFormatter(opts ...EscapeFormatterOption) *EscapeFormatter {
	f := &EscapeFormatter{
		mask:       "%d",
		timeFormat: time.RFC3339Nano,
		indent:     "  ",
		align:      -1,
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
	for i := range n {
		s := strings.TrimSpace(headers[i])
		switch {
		case s == "" && strings.ContainsRune(f.mask, '%'):
			s = fmt.Sprintf(f.mask, i+1)
		case s == "":
			s = f.mask
		}
		res[i] = FormatBytes([]byte(s), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
		res[i].Align = f.headerAlign
	}
	return res, nil
}

// Format satisfies the Formatter interface.
func (f *EscapeFormatter) Format(vals []any) ([]*Value, error) {
	n := len(vals)
	res := make([]*Value, n)
	left, right := AlignLeft, AlignRight
	if f.align != -1 {
		left, right = f.align, f.align
	}
	for i := range n {
		v, err := f.format(deref(vals[i]), left, right, 0)
		if err != nil {
			return nil, err
		}
		res[i] = v
	}
	return res, nil
}

// maxValuerDepth is the maximum number of times [EscapeFormatter.format] will
// unwrap a [database/sql/driver.Valuer] before giving up and encoding the
// value, bounding recursion for a value that wraps itself.
const maxValuerDepth = 10

// format formats a single value, returning nil for a null value. depth is the
// number of [database/sql/driver.Valuer] values already unwrapped.
func (f *EscapeFormatter) format(val any, left, right Align, depth int) (*Value, error) {
	// TODO: change time to v.AppendFormat() + pool
	// TODO: use strconv.Format* for numeric times
	// TODO: use pool
	// TODO: allow configurable runes that can be escaped
	switch v := val.(type) {
	case nil:
		return nil, nil
	case bool:
		return newValue(strconv.FormatBool(v), left, true), nil
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		var s string
		if f.useNumericLocale() {
			s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v))
		} else {
			s = fmt.Sprintf("%d", v)
		}
		return f.number(s, right, overflowsInt64(v)), nil
	case float32:
		s, notNumber := floatString(float64(v))
		switch {
		case notNumber:
		case f.useNumericLocale():
			s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v, number.MinFractionDigits(1)))
		default:
			s = strconv.FormatFloat(float64(v), 'g', -1, 32)
		}
		return f.number(s, right, notNumber), nil
	case float64:
		s, notNumber := floatString(v)
		switch {
		case notNumber:
		case f.useNumericLocale():
			s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v, number.MinFractionDigits(1)))
		default:
			s = strconv.FormatFloat(v, 'g', -1, 64)
		}
		return f.number(s, right, notNumber), nil
	case uintptr:
		return newValue(fmt.Sprintf("(0x%x)", v), right, false), nil
	case complex64:
		return newValue(fmt.Sprintf("%g", v), right, false), nil
	case complex128:
		return newValue(fmt.Sprintf("%g", v), right, false), nil
	case []byte:
		return FormatBytes(v, f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote), nil
	case string:
		return FormatBytes([]byte(v), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote), nil
	case time.Time:
		t := v
		if f.timeLocation != nil {
			t = t.In(f.timeLocation)
		}
		return newValue(t.Format(f.timeFormat), left, false), nil
	case sql.NullBool:
		if v.Valid {
			return newValue(strconv.FormatBool(v.Bool), left, true), nil
		}
		return nil, nil
	case sql.NullByte:
		if v.Valid {
			var s string
			if f.useNumericLocale() {
				s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v.Byte))
			} else {
				s = strconv.FormatUint(uint64(v.Byte), 10)
			}
			return f.number(s, right, false), nil
		}
		return nil, nil
	case sql.NullFloat64:
		if v.Valid {
			s, notNumber := floatString(v.Float64)
			switch {
			case notNumber:
			case f.useNumericLocale():
				s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v.Float64))
			default:
				s = strconv.FormatFloat(v.Float64, 'g', -1, 64)
			}
			return f.number(s, right, notNumber), nil
		}
		return nil, nil
	case sql.NullInt16:
		if v.Valid {
			var s string
			if f.useNumericLocale() {
				s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v.Int16))
			} else {
				s = strconv.FormatInt(int64(v.Int16), 10)
			}
			return f.number(s, right, false), nil
		}
		return nil, nil
	case sql.NullInt32:
		if v.Valid {
			var s string
			if f.useNumericLocale() {
				s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v.Int32))
			} else {
				s = strconv.FormatInt(int64(v.Int32), 10)
			}
			return f.number(s, right, false), nil
		}
		return nil, nil
	case sql.NullInt64:
		if v.Valid {
			var s string
			if f.useNumericLocale() {
				s = f.numericLocalePrinter.Sprintf("%v", number.Decimal(v.Int64))
			} else {
				s = strconv.FormatInt(v.Int64, 10)
			}
			return f.number(s, right, false), nil
		}
		return nil, nil
	case sql.NullString:
		if v.Valid {
			return FormatBytes([]byte(v.String), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote), nil
		}
		return nil, nil
	case sql.NullTime:
		if v.Valid {
			t := v.Time
			if f.timeLocation != nil {
				t = t.In(f.timeLocation)
			}
			return newValue(t.Format(f.timeFormat), left, false), nil
		}
		return nil, nil
	case sql.RawBytes:
		return FormatBytes(v, f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote), nil
	case fmt.Stringer:
		return FormatBytes([]byte(v.String()), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote), nil
	case driver.Valuer:
		if z, ok := unwrapNull(v); ok {
			if depth < maxValuerDepth {
				return f.format(deref(z), left, right, depth+1)
			}
			break
		}
		// note: a Valuer returning an error is encoded below, as encoding the
		// value is more useful to the reader than failing the entire result
		// set.
		if z, err := v.Value(); err == nil && depth < maxValuerDepth {
			return f.format(deref(z), left, right, depth+1)
		}
	}
	return f.encode(val)
}

// encode encodes a value not otherwise handled by [EscapeFormatter.format]
// using the configured encoder, or the standard [encoding/json/v2] when no
// encoder was configured.
func (f *EscapeFormatter) encode(val any) (*Value, error) {
	// TODO: pool
	if f.encoder != nil {
		buf, err := f.encoder(val)
		if err != nil {
			return nil, err
		}
		return &Value{
			Buf: buf,
			Raw: true,
		}, nil
	}
	// json encode
	buf, err := json.Marshal(val, f.jsonOptions()...)
	if err != nil {
		return nil, err
	}
	buf = bytes.TrimSpace(buf)
	if f.isJSON {
		return &Value{
			Buf: buf,
			Raw: true,
		}, nil
	}
	v := FormatBytes(buf, f.invalid, f.invalidWidth, false, f.isRaw, f.sep, f.quote)
	v.Raw = true
	return v, nil
}

// jsonOptions returns the [encoding/json/v2] options used to marshal a value.
//
// Note: the first four restore what the v1 encoding/json encoder did, as the
// v2 defaults differ -- v2 leaves map keys in map order, rejects invalid
// UTF-8, and writes a nil map or slice as {} or [] rather than null.
func (f *EscapeFormatter) jsonOptions() []json.Options {
	opts := []json.Options{
		json.Deterministic(true),
		jsontext.AllowInvalidUTF8(true),
		json.FormatNilMapAsNull(true),
		json.FormatNilSliceAsNull(true),
		jsontext.EscapeForHTML(f.escapeHTML),
	}
	// note: jsontext panics when the prefix or indent is anything but spaces
	// and tabs, where the v1 encoder accepted any string, so both are dropped
	// when not usable. An empty indent indents by nothing rather than
	// producing compact output, so as with the v1 encoder the output is
	// compact only when the prefix and indent are both empty.
	prefix, indent := f.prefix, f.indent
	if !isSpaceOrTab(prefix) {
		prefix = ""
	}
	if !isSpaceOrTab(indent) {
		indent = ""
	}
	if prefix != "" || indent != "" {
		opts = append(opts, jsontext.WithIndent(indent))
		if prefix != "" {
			opts = append(opts, jsontext.WithIndentPrefix(prefix))
		}
	}
	return opts
}

// isSpaceOrTab reports whether s contains only spaces and tabs.
func isSpaceOrTab(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool {
		return r != ' ' && r != '\t'
	})
}

// nullPkgPath is the package path of the generic [database/sql.Null] type.
var nullPkgPath = reflect.TypeOf(sql.Null[bool]{}).PkgPath()

// unwrapNull returns the value contained in a generic [database/sql.Null]
// value, and whether v is such a value. A nil value is returned for a null
// (invalid) value.
//
// The generic Null's own Value method cannot be used for this: it passes the
// contained value through [database/sql/driver.DefaultParameterConverter],
// which converts every unsigned integer to an int64 and rejects outright a
// uint64 with its high bit set, such as MySQL's max BIGINT UNSIGNED.
func unwrapNull(v any) (any, bool) {
	typ := reflect.TypeOf(v)
	if typ == nil {
		return nil, false
	}
	// note: deref unwraps only one level, so a Null reached through a *any
	// destination arrives here as a pointer.
	val := reflect.ValueOf(v)
	if typ.Kind() == reflect.Pointer {
		typ, val = typ.Elem(), val.Elem()
	}
	if typ.Kind() != reflect.Struct || typ.PkgPath() != nullPkgPath ||
		typ.NumField() != 2 ||
		typ.Field(0).Name != "V" ||
		typ.Field(1).Name != "Valid" || typ.Field(1).Type.Kind() != reflect.Bool {
		return nil, false
	}
	// note: val is invalid for a nil pointer, whose Value method would panic.
	if !val.IsValid() || !val.Field(1).Bool() {
		return nil, true
	}
	return val.Field(0).Interface(), true
}

// useNumericLocale reports whether numbers are formatted for a locale.
//
// Note: never for JSON, which has a number type but no syntax for a grouping
// separator. Honoring it there would mean writing a number as a string, so
// that a column's JSON type would follow a display option; ignoring it keeps
// the type stable. csv, which has no types at all, applies it as psql does.
func (f *EscapeFormatter) useNumericLocale() bool {
	return f.numericLocalePrinter != nil && !f.isJSON
}

// number returns a value for a formatted number. notNumber marks a value that
// JSON cannot write as a number.
//
// A number formatted for a locale is escaped as any other value would be, as
// its grouping separator would otherwise be read as a csv field separator,
// leaving a bare 1,234,567 to be read as three fields.
func (f *EscapeFormatter) number(s string, align Align, notNumber bool) *Value {
	switch {
	case f.isJSON && notNumber:
		// note: written as a JSON string of the same text the other formats
		// show, never locale formatted, which JSON ignores anyway.
		return newValue(s, align, false)
	case !f.useNumericLocale():
		return newValue(s, align, true)
	}
	v := FormatBytes([]byte(s), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
	v.Align = align
	return v
}

// floatString returns the text of a float that is not a finite number, and
// whether it is one.
//
// Note: the spellings are PostgreSQL's, which are what psql displays. JSON has
// no NaN or infinity at all, so these are written there as strings, as
// PostgreSQL's own to_jsonb does.
func floatString(f float64) (string, bool) {
	switch {
	case math.IsNaN(f):
		return "NaN", true
	case math.IsInf(f, 1):
		return "Infinity", true
	case math.IsInf(f, -1):
		return "-Infinity", true
	}
	return "", false
}

// overflowsInt64 reports whether v is an unsigned integer above the int64 a
// JSON number is read as, such as MySQL's max BIGINT UNSIGNED, and so is
// written as a JSON string of its exact digits.
func overflowsInt64(v any) bool {
	switch z := v.(type) {
	case uint:
		return uint64(z) > math.MaxInt64
	case uint64:
		return z > math.MaxInt64
	}
	return false
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
			switch {
			case invalid != nil:
				res.Buf = append(res.Buf, invalid...)
				res.Width += invalidWidth
				res.Quoted = true
			case isJSON:
				res.Buf = append(res.Buf, '\\', 'u')
				for s := 12; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, lowerhex[r>>uint(s)&0xf])
				}
			default:
				res.Buf = append(res.Buf, '\\', 'x', lowerhex[src[0]>>4], lowerhex[src[0]&0xf])
				res.Width += 4
				res.Quoted = true
			}
			continue
		}
		// handle json encoding
		if isJSON {
			switch r {
			case '\a':
				res.Buf = append(res.Buf, '\\', 'u', '0', '0', '0', '7')
				res.Width += 6
				continue
			case '\b':
				res.Buf = append(res.Buf, '\\', 'b')
				res.Width += 2
				continue
			case '\f':
				res.Buf = append(res.Buf, '\\', 'f')
				res.Width += 2
				continue
			case '\n':
				res.Buf = append(res.Buf, '\\', 'n')
				res.Width += 2
				continue
			case '\r':
				res.Buf = append(res.Buf, '\\', 'r')
				res.Width += 2
				continue
			case '\t':
				res.Buf = append(res.Buf, '\\', 't')
				res.Width += 2
				continue
			case '"':
				res.Buf = append(res.Buf, '\\', '"')
				res.Width += 2
				continue
			case '\\':
				res.Buf = append(res.Buf, '\\', '\\')
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
			case r < ' ' && !isJSON:
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

func (v *Value) String() string {
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
	for l := range len(v.Tabs) {
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
	for i := range tabs {
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

// WithTimeLocation is an escape formatter option to set the time location used
// for time values.
func WithTimeLocation(timeLocation *time.Location) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.timeLocation = timeLocation
	}
}

// WithEncoder is an escape formatter option to set a standard Go encoder to
// use for encoding the value.
func WithEncoder(encoder func(any) ([]byte, error)) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.encoder = encoder
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
// prefix, indent value, and whether or not to escape HTML. Passed to
// [encoding/json/v2] when a marshaler has not been set on the escape
// formatter.
//
// The prefix and indent must contain only spaces and tabs, and are ignored
// otherwise. Output is compact when both are empty.
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

// WithAlign sets the forced alignment for values.
func WithAlign(a Align) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.align = a
	}
}

// WithNumericLocale sets the numeric locale printer, which groups the digits
// of a number as the locale does, as psql's \pset numericlocale does.
//
// It has no effect on JSON output: a grouped number is not a JSON number, and
// writing it as a JSON string would make a column's type depend on a display
// option. Every other format applies it, csv quoting the fields that need it
// exactly as psql does.
func WithNumericLocale(enable bool, locale string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		if enable {
			tag := language.English
			if t, err := language.Parse(locale); err == nil {
				tag = t
			}
			f.numericLocalePrinter = message.NewPrinter(tag)
		}
	}
}

// deref dereferences a pointer to an interface.
func deref(v any) any {
	switch z := v.(type) {
	case nil:
		return nil
	case *any:
		return *z
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	return val.Interface()
}
