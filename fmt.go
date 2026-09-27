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

// Formatter is the common interface that formats values.
type Formatter interface {
	// Header returns a slice of formatted values for the column names of a
	// header.
	Header([]string) ([]*Value, error)
	// Format returns a slice of formatted values for the values of a row.
	Format([]any) ([]*Value, error)
}

// EscapeFormatter is a formatter that escapes values. It formats the standard
// Go types.
//
// If a marshal func is set with [WithEncoder], the formatter passes it each
// map[string]interface{} and []interface{} value. Otherwise, the formatter
// uses [encoding/json/v2] from the standard library.
type EscapeFormatter struct {
	// mask is the text for a column name in the header that is empty after the
	// formatter trims its spaces.
	//
	// Note: the formatter replaces %d in mask with the column number, which
	// starts at 1.
	mask string
	// timeFormat is the format to use for time values.
	timeFormat string
	// timeLocation is the location to use for time values.
	timeLocation *time.Location
	// encoder is the marshal func for map[string]interface{} and []interface{}
	// types.
	//
	// If it is nil, the formatter uses the standard [encoding/json/v2].
	encoder func(any) ([]byte, error)
	// prefix is the indent prefix for [encoding/json/v2] when encoder is nil.
	prefix string
	// indent is the indent for [encoding/json/v2] when encoder is nil.
	indent string
	// isJSON sets whether to escape JSON characters.
	isJSON bool
	// escapeHTML sets whether [encoding/json/v2] escapes HTML characters when
	// encoder is nil.
	escapeHTML bool
	// isRaw sets whether to use the raw escape.
	isRaw bool
	// sep is the separator for the raw (csv) escape.
	sep rune
	// quote is the quote for the raw (csv) escape.
	quote rune
	// invalid is the text that replaces an invalid UTF-8 rune in the escape.
	invalid []byte
	// invalidWidth is the rune width of invalid.
	invalidWidth int
	// headerAlign is the default alignment of the column names in the header.
	headerAlign Align
	// align is the forced alignment for values.
	align Align
	// numericLocalePrinter is the numeric locale printer.
	numericLocalePrinter *message.Printer
}

// NewEscapeFormatter creates an escape formatter for basic Go values, such as
// []byte, string, time.Time, sql.Null*, and any [database/sql/driver.Valuer].
// The formatter passes map[string]interface{} and []interface{} values to the
// marshal func set with [WithEncoder]. Otherwise, it uses the standard
// [encoding/json/v2] to marshal those values.
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

// maxValuerDepth is the maximum number of times that [EscapeFormatter.format]
// unwraps a [database/sql/driver.Valuer]. After that, it encodes the value.
// This limit stops the recursion for a value that wraps itself.
const maxValuerDepth = 10

// format formats one value. It returns nil for a null value. depth is the
// number of [database/sql/driver.Valuer] values that format already unwrapped.
func (f *EscapeFormatter) format(val any, left, right Align, depth int) (*Value, error) {
	// TODO: change time to v.AppendFormat() + pool
	// TODO: use strconv.Format* for numeric times
	// TODO: use a pool
	// TODO: let the caller set the runes to escape
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
		return f.number(s, right, unsignedInt64(v)), nil
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
		// note: a Valuer that returns an error is encoded below. the reader
		// gets more use from the encoded value than from an error for the
		// whole result set
		if z, err := v.Value(); err == nil && depth < maxValuerDepth {
			return f.format(deref(z), left, right, depth+1)
		}
	}
	return f.encode(val)
}

// encode encodes a value that [EscapeFormatter.format] does not handle
// otherwise. It uses the marshal func set with [WithEncoder], or the standard
// [encoding/json/v2] if no marshal func is set.
func (f *EscapeFormatter) encode(val any) (*Value, error) {
	// TODO: use a pool
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
// Note: the first four options restore the behavior of the v1 encoding/json
// package, because the v2 defaults differ. By default, v2 leaves map keys in
// map order, rejects invalid UTF-8, and writes a nil map or slice as {} or []
// instead of null.
func (f *EscapeFormatter) jsonOptions() []json.Options {
	opts := []json.Options{
		json.Deterministic(true),
		jsontext.AllowInvalidUTF8(true),
		json.FormatNilMapAsNull(true),
		json.FormatNilSliceAsNull(true),
		jsontext.EscapeForHTML(f.escapeHTML),
	}
	// note: jsontext panics if the prefix or the indent holds a character
	// other than a space or a tab, but the v1 encoding/json package accepted
	// any string. so a prefix or an indent that jsontext cannot use is
	// dropped. an empty indent indents by nothing and does not give compact
	// output. so, as with the v1 package, the output is compact only if the
	// prefix and the indent are both empty
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

// unwrapNull returns the value in a generic [database/sql.Null] value, and
// whether v is such a value. For a null (invalid) value, it returns a nil
// value.
//
// The Value method of the generic Null cannot do this. That method passes the
// value through [database/sql/driver.DefaultParameterConverter]. The
// converter converts every unsigned integer to an int64. It rejects a uint64
// that has its high bit set, such as the maximum MySQL BIGINT UNSIGNED.
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
	// note: val is invalid for a nil pointer, and a call to its Value method
	// panics
	if !val.IsValid() || !val.Field(1).Bool() {
		return nil, true
	}
	return val.Field(0).Interface(), true
}

// useNumericLocale reports whether numbers are formatted for a locale.
//
// Note: never for JSON. JSON has a number type but no syntax for a grouping
// separator. To use the locale there, the formatter must write a number as a
// string, and then the JSON type of a column follows a display option. If the
// formatter ignores the locale, the type does not change. csv has no types at
// all, and it applies the locale as psql does.
func (f *EscapeFormatter) useNumericLocale() bool {
	return f.numericLocalePrinter != nil && !f.isJSON
}

// number returns a value for a formatted number. asString marks a number that
// JSON writes as a string instead of as a number.
//
// The formatter escapes a number formatted for a locale the same as any other
// value. If it did not, a csv reader reads the grouping separator as a field
// separator, and it reads a bare 1,234,567 as three fields.
func (f *EscapeFormatter) number(s string, align Align, asString bool) *Value {
	switch {
	case f.isJSON && asString:
		// note: a JSON string of the same text that the other formats show.
		// it is never locale formatted, because JSON ignores the locale
		return newValue(s, align, false)
	case !f.useNumericLocale():
		return newValue(s, align, true)
	}
	v := FormatBytes([]byte(s), f.invalid, f.invalidWidth, f.isJSON, f.isRaw, f.sep, f.quote)
	v.Align = align
	return v
}

// floatString returns the text of a float that is not a finite number, and
// whether f is such a float.
//
// Note: the spellings are those of PostgreSQL, which psql shows. JSON has no
// NaN or infinity, so the JSON output writes these as strings. The to_jsonb
// function of PostgreSQL does the same.
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

// unsignedInt64 reports whether v is an unsigned 64 bit integer, such as a
// MySQL BIGINT UNSIGNED. JSON writes such a value as a string of its exact
// digits. Then the value stays exact for a consumer that reads a JSON number
// as an int64 or a float64.
//
// Note: by type and not by value, so that a column has one JSON type in each
// of its rows. A decision by value writes {"n":42} for one row and
// {"n":"18446744073709551615"} for the next. Then a consumer cannot know the
// type of the column until it reads every value. uint is included whatever
// its width, so that the output does not differ between platforms.
func unsignedInt64(v any) bool {
	switch v.(type) {
	case uint, uint64:
		return true
	}
	return false
}

// newValue returns a value for a string that has no characters to escape.
func newValue(str string, align Align, raw bool) *Value {
	v := &Value{Buf: []byte(str), Align: align, Raw: raw}
	v.Width = len(v.Buf)
	return v
}

// lowerhex holds the lower case hex digits.
const lowerhex = "0123456789abcdef"

// FormatBytes escapes src to a Value. The Value holds the escaped (encoded)
// and unescaped runes, and the positions of the tabs and newlines in its Buf.
func FormatBytes(src []byte, invalid []byte, invalidWidth int, isJSON, isRaw bool, sep, quote rune) *Value {
	res := &Value{
		Tabs: make([][][2]int, 1),
	}
	var tmp [4]byte
	var r rune
	var l, w int
	for ; len(src) > 0; src = src[w:] {
		r, w = rune(src[0]), 1
		// decode only a rune that is not ASCII
		if r >= utf8.RuneSelf {
			r, w = utf8.DecodeRune(src)
		}
		// the decoded rune is invalid
		if w == 1 && r == utf8.RuneError {
			// replace with invalid if it is set, otherwise escape as hex
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
		// escape for JSON
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
		// raw (csv) escape
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

// Value holds a formatted value and data about it.
type Value struct {
	// Buf is the formatted value.
	Buf []byte
	// Newlines are the positions of newline characters in Buf.
	Newlines [][2]int
	// Tabs are the positions of tab characters in Buf, split per line.
	Tabs [][][2]int
	// Width is the remaining width.
	Width int
	// Align is the alignment of the value.
	Align Align
	// Raw is true when the JSON encoder writes Buf exactly, with no quotes.
	Raw bool
	// Quoted tracks whether a raw value must be quoted, that is, whether it
	// contains a space or a non printable character.
	Quoted bool
}

func (v *Value) String() string {
	return string(v.Buf)
}

// LineWidth returns the display width of line l.
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

// MaxWidth calculates the display width of the longest line in Buf, from the
// start offset and the tab width.
func (v *Value) MaxWidth(offset, tab int) int {
	// a simple value has no tabs
	width := v.Width
	for l := range len(v.Tabs) {
		width = max(width, v.LineWidth(l, offset, tab))
	}
	return width
}

// Align is the alignment direction of a value.
type Align int

// The Align directions.
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

// tabwidth returns the rune width of a line in buf that contains tabs. It
// uses the tab positions from the start of buf, a column offset, and the tab
// width.
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

// WithMask is an escape formatter option that sets the mask for an empty
// column name in the header.
func WithMask(mask string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.mask = mask
	}
}

// WithTimeFormat is an escape formatter option that sets the time format for
// time values.
func WithTimeFormat(timeFormat string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.timeFormat = timeFormat
	}
}

// WithTimeLocation is an escape formatter option that sets the time location
// for time values.
func WithTimeLocation(timeLocation *time.Location) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.timeLocation = timeLocation
	}
}

// WithEncoder is an escape formatter option that sets a standard Go marshal
// func to encode the value.
func WithEncoder(encoder func(any) ([]byte, error)) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.encoder = encoder
	}
}

// WithIsJSON is an escape formatter option that turns on a special escape for
// JSON characters in values that are not complex.
func WithIsJSON(isJSON bool) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.isJSON = isJSON
	}
}

// WithJSONConfig is an escape formatter option that sets the JSON prefix, the
// JSON indent, and whether to escape HTML. The formatter passes them to
// [encoding/json/v2] if no marshal func is set on the escape formatter.
//
// The prefix and indent must contain only spaces and tabs, and are ignored
// otherwise. Output is compact when both are empty.
func WithJSONConfig(prefix, indent string, escapeHTML bool) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.prefix, f.indent, f.escapeHTML = prefix, indent, escapeHTML
	}
}

// WithIsRaw is an escape formatter option that turns on a special escape for
// raw characters in values.
func WithIsRaw(isRaw bool, sep, quote rune) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.isRaw, f.sep, f.quote = isRaw, sep, quote
	}
}

// WithInvalid is an escape formatter option that sets the text that replaces
// an invalid rune in the escape.
func WithInvalid(invalid string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.invalid = []byte(invalid)
		f.invalidWidth = runewidth.StringWidth(invalid)
	}
}

// WithHeaderAlign sets the alignment of the column names in the header.
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

// WithNumericLocale sets the numeric locale printer. The printer groups the
// digits of a number as the locale does, the same as \pset numericlocale in
// psql.
//
// It has no effect on JSON output. A grouped number is not a JSON number. A
// JSON string for it makes the JSON type of a column depend on a display
// option. Every other format applies it. The csv output quotes each field
// that needs it, exactly as psql does.
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
