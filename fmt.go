package tblfmt

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"math"
	"math/big"
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
// The formatter marshals each value that it does not format otherwise, such
// as a map, a slice that is not []byte or a struct. It uses the marshal func
// set with [WithEncoder], or [encoding/json/v2] from the standard library.
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
	// encoder is the marshal func for a value that the formatter does not
	// format otherwise.
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
	// align is the alignment of a number, a bool, a time and a decimal, or -1
	// for the alignment of the type (D71).
	align Align
	// locale holds the symbols of the numeric locale, or nil.
	locale *numericLocale
}

// NewEscapeFormatter creates an escape formatter for basic Go values, such as
// []byte, string, time.Time, sql.Null*, a decimal with the Decompose method of
// [database/sql], a [fmt.Stringer] and any [database/sql/driver.Valuer]. The
// formatter marshals each other value, with the marshal func set with
// [WithEncoder] or with the standard [encoding/json/v2].
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
	switch v := val.(type) {
	case nil:
		return nil, nil
	case bool:
		return newValue(strconv.FormatBool(v), left, true), nil
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return f.number(f.localize(fmt.Sprintf("%d", v)), right, unsignedInt64(v)), nil
	case float32:
		s, notNumber := floatString(float64(v))
		if !notNumber {
			// note: strconv writes an exponent from 1e+06, as float4out does
			s = f.localize(strconv.FormatFloat(float64(v), 'g', -1, 32))
		}
		return f.number(s, right, notNumber), nil
	case float64:
		s, notNumber := floatString(v)
		if !notNumber {
			s = f.localize(float8(v))
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
			return f.number(f.localize(strconv.FormatUint(uint64(v.Byte), 10)), right, false), nil
		}
		return nil, nil
	case sql.NullFloat64:
		if v.Valid {
			s, notNumber := floatString(v.Float64)
			if !notNumber {
				s = f.localize(float8(v.Float64))
			}
			return f.number(s, right, notNumber), nil
		}
		return nil, nil
	case sql.NullInt16:
		if v.Valid {
			return f.number(f.localize(strconv.FormatInt(int64(v.Int16), 10)), right, false), nil
		}
		return nil, nil
	case sql.NullInt32:
		if v.Valid {
			return f.number(f.localize(strconv.FormatInt(int64(v.Int32), 10)), right, false), nil
		}
		return nil, nil
	case sql.NullInt64:
		if v.Valid {
			return f.number(f.localize(strconv.FormatInt(v.Int64, 10)), right, false), nil
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
	case decimalDecomposer:
		// note: before fmt.Stringer and driver.Valuer, so that a decimal type
		// that has them is written as a number too (D68)
		form, negative, coefficient, exponent := v.Decompose(nil)
		switch form {
		case 0:
			return f.number(f.decimal(negative, coefficient, exponent), right, false), nil
		case 1:
			if negative {
				return f.number("-Infinity", right, true), nil
			}
			return f.number("Infinity", right, true), nil
		case 2:
			return f.number("NaN", right, true), nil
		}
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
var nullPkgPath = reflect.TypeFor[sql.Null[bool]]().PkgPath()

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
	return f.locale != nil && !f.isJSON
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

// decimalDecomposer is the interface that [database/sql] documents for a
// decimal type. go-hdb returns a SAP HANA DECIMAL as such a type, and it has
// no String or Value method.
//
// Decompose returns the form of the value: 0 is finite, 1 is infinite and 2
// is NaN. A finite value is coefficient * 10^exponent, and coefficient is a
// big-endian unsigned integer.
type decimalDecomposer interface {
	Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32)
}

// maxDecimalDigits is the number of digits before the decimal point of the
// largest PostgreSQL numeric. A decimal with a larger exponent is written with
// an exponent, so that a value of a few bytes does not become a string of
// billions of zeros.
const maxDecimalDigits = 1 << 17

// decimal returns the text of a finite decimal, as psql writes a numeric. It
// keeps each digit of the coefficient, so that 1.5 with a scale of 2 is 1.50.
//
// Note: PostgreSQL has no negative zero, so a zero has no sign. The numeric
// locale writes the number of any length as the locale does (D73).
func (f *EscapeFormatter) decimal(negative bool, coefficient []byte, exponent int32) string {
	digits := new(big.Int).SetBytes(coefficient).String()
	sign := ""
	if negative && digits != "0" {
		sign = "-"
	}
	var integer, fraction string
	switch n := int(exponent); {
	case n > maxDecimalDigits || n < -maxDecimalDigits:
		return sign + digits + "e" + strconv.Itoa(n)
	case n >= 0:
		integer = digits
		if digits != "0" {
			integer += strings.Repeat("0", n)
		}
	default:
		n = -n
		if len(digits) <= n {
			digits = strings.Repeat("0", n-len(digits)+1) + digits
		}
		integer, fraction = digits[:len(digits)-n], digits[len(digits)-n:]
	}
	if fraction == "" {
		return f.localize(sign + integer)
	}
	return f.localize(sign + integer + "." + fraction)
}

// float8 returns the text of a finite float64, as psql writes a float8. It
// writes the shortest digits that read back as v, and it writes an exponent
// only when the decimal exponent is less than -4 or more than 14.
//
// Note: float8out in PostgreSQL writes the same shortest digits as strconv
// with a precision of -1. strconv 'g' writes an exponent from 1e+06, so only
// an exponent from 6 to 14 needs 'f'. The other values take one conversion.
func float8(v float64) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if _, e, ok := strings.CutLast(s, "e"); ok {
		if exp, _ := strconv.Atoi(e); 5 < exp && exp < 15 {
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
	return s
}

// localize returns s, the text of a finite number, in the numeric locale. It
// writes the digits, the grouping, the decimal point and the minus sign of the
// locale. An exponent keeps its letter and its sign. Without the numeric
// locale, it returns s.
//
// Note: psql writes ASCII digits, and it groups by the first group size of the
// locale only. tblfmt writes the number as the locale does (D73).
func (f *EscapeFormatter) localize(s string) string {
	if !f.useNumericLocale() {
		return s
	}
	return f.locale.format(s)
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

// lowerhex and upperhex hold the hex digits.
const (
	lowerhex = "0123456789abcdef"
	upperhex = "0123456789ABCDEF"
)

// FormatBytes escapes src to a Value. The Value holds the escaped (encoded)
// and unescaped runes, and the positions of the tabs and newlines in its Buf.
func FormatBytes(src []byte, invalid []byte, invalidWidth int, isJSON, isRaw bool, sep, quote rune) *Value {
	res := &Value{
		Tabs: make([][][2]int, 1),
	}
	// run is the start in Buf of the text that is copied as it is, or -1.
	//
	// note: the width of the text is measured by grapheme cluster when the
	// run ends, and not rune by rune. A cluster such as an emoji with a skin
	// tone holds two wide runes, and a terminal draws it 2 columns wide.
	run := -1
	measure := func() {
		if run >= 0 {
			res.Width += runewidth.StringWidth(string(res.Buf[run:])) + emojiWidth(res.Buf[run:])
			run = -1
		}
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
			measure()
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
				measure()
				res.Buf = append(res.Buf, '\\', 'u', '0', '0', '0', '7')
				res.Width += 6
				continue
			case '\b':
				measure()
				res.Buf = append(res.Buf, '\\', 'b')
				res.Width += 2
				continue
			case '\f':
				measure()
				res.Buf = append(res.Buf, '\\', 'f')
				res.Width += 2
				continue
			case '\n':
				measure()
				res.Buf = append(res.Buf, '\\', 'n')
				res.Width += 2
				continue
			case '\r':
				measure()
				res.Buf = append(res.Buf, '\\', 'r')
				res.Width += 2
				continue
			case '\t':
				measure()
				res.Buf = append(res.Buf, '\\', 't')
				res.Width += 2
				continue
			case '"':
				measure()
				res.Buf = append(res.Buf, '\\', '"')
				res.Width += 2
				continue
			case '\\':
				measure()
				res.Buf = append(res.Buf, '\\', '\\')
				res.Width += 2
				continue
			}
		}
		// raw (csv) escape
		if isRaw {
			if run < 0 {
				run = len(res.Buf)
			}
			n := utf8.EncodeRune(tmp[:], r)
			res.Buf = append(res.Buf, tmp[:n]...)
			switch {
			case r == sep:
				res.Quoted = true
			case r == quote && quote != 0:
				res.Buf = append(res.Buf, tmp[:n]...)
				res.Quoted = true
			default:
				res.Quoted = res.Quoted || unicode.IsSpace(r)
			}
			continue
		}
		// psql escapes only a control character, and writes each other rune
		// as it is. A joiner, a tag or a private use rune is part of what a
		// terminal draws, such as a joined emoji or a flag.
		if !unicode.IsControl(r) {
			if run < 0 {
				run = len(res.Buf)
			}
			n := utf8.EncodeRune(tmp[:], r)
			res.Buf = append(res.Buf, tmp[:n]...)
			continue
		}
		measure()
		switch r {
		case '\r':
			res.Buf = append(res.Buf, '\\', 'r')
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
			// psql writes a control character below U+0080 as \x00, and one
			// from U+0080 as \u0000, with upper case hex digits. JSON uses
			// \u0000 with lower case hex digits. A control character is below
			// U+00A0, so it never needs the \U00000000 form.
			hex := upperhex
			if isJSON {
				hex = lowerhex
			}
			switch {
			case r < utf8.RuneSelf && !isJSON:
				res.Buf = append(res.Buf, '\\', 'x', hex[byte(r)>>4], hex[byte(r)&0xf])
				res.Width += 4
			default:
				res.Buf = append(res.Buf, '\\', 'u')
				for s := 12; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, hex[r>>uint(s)&0xf])
				}
				res.Width += 6
			}
		}
	}
	measure()
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
	// Quoted is true when the csv encoder must quote a raw value. The value
	// holds the separator, the quote, a white space character or an invalid
	// rune.
	Quoted bool
}

func (v *Value) String() string {
	return string(v.Buf)
}

// tabWidth is the distance between two tab stops, as psql uses.
const tabWidth = 8

// line returns the start and the end of line l in Buf.
func (v *Value) line(l int) (int, int) {
	start, end := 0, len(v.Buf)
	if l > 0 {
		start = v.Newlines[l-1][0] + 1
	}
	if l < len(v.Newlines) {
		end = v.Newlines[l][0]
	}
	return start, end
}

// LineWidth returns the display width of line l. A tab moves to the next
// multiple of 8 columns from the start of the line, as psql writes it.
func (v *Value) LineWidth(l int) int {
	var width int
	if l < len(v.Tabs) && len(v.Tabs[l]) != 0 {
		width = tabwidth(v.Tabs[l])
	}
	if l < len(v.Newlines) {
		width += v.Newlines[l][1]
	} else {
		width += v.Width
	}
	return width
}

// MaxWidth returns the display width of the longest line in Buf.
func (v *Value) MaxWidth() int {
	// a simple value has no tabs
	width := v.Width
	for l := range len(v.Tabs) {
		width = max(width, v.LineWidth(l))
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

// tabwidth returns the width of a line up to and with its last tab. Each
// entry of tabs holds the width of the text before a tab.
func tabwidth(tabs [][2]int) int {
	var width int
	for i := range tabs {
		width += tabs[i][1]
		width += tabWidth - width%tabWidth
	}
	return width
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

// WithAlign sets the alignment of a number, a bool, a time and a decimal. A
// value that the formatter writes as text stays at the left (D71).
func WithAlign(a Align) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.align = a
	}
}

// WithNumericLocale sets the numeric locale. The formatter writes a number
// with the digits, the grouping, the decimal point and the minus sign of the
// locale, as golang.org/x/text writes them. This is the \pset numericlocale of
// psql, but psql writes ASCII digits and one group size (D73). If the locale
// does not parse, the formatter uses English.
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
			f.locale = newNumericLocale(tag)
		}
	}
}

// numericLocale holds the symbols that a locale uses to write a number.
//
// Note: golang.org/x/text formats only a number that fits a uint64 or a
// float64, so the symbols come from its printer once, and format applies them
// to a number of any length, such as a decimal.
type numericLocale struct {
	// digits holds the digit of the locale for each of 0 to 9.
	digits [10]string
	// point is the decimal point, and sep is the grouping separator.
	point string
	sep   string
	// primary is the size of the group next to the decimal point, and
	// secondary is the size of each other group. primary is 0 for a locale
	// that does not group.
	primary   int
	secondary int
	// minus and minusEnd are the text before and after a negative number.
	minus    string
	minusEnd string
}

// newNumericLocale reads the symbols of the locale from the printer of
// golang.org/x/text.
func newNumericLocale(tag language.Tag) *numericLocale {
	p := message.NewPrinter(tag)
	sprint := func(v any) string {
		return p.Sprintf("%v", number.Decimal(v))
	}
	l := &numericLocale{point: "."}
	for i := range l.digits {
		l.digits[i] = sprint(i)
	}
	// note: the locale writes one half as a zero, the decimal point and a
	// five
	half := sprint(0.5)
	if zero, five := l.digits[0], l.digits[5]; strings.HasPrefix(half, zero) && strings.HasSuffix(half, five) && len(zero)+len(five) < len(half) {
		l.point = half[len(zero) : len(half)-len(five)]
	}
	// note: 20 digits show the size of each group, and the separator
	var groups []int
	n := 0
	for rest := sprint(uint64(12345678901234567890)); rest != ""; {
		if d := l.digit(rest); d != "" {
			n++
			rest = rest[len(d):]
			continue
		}
		i := 0
		for i < len(rest) && l.digit(rest[i:]) == "" {
			_, w := utf8.DecodeRuneInString(rest[i:])
			i += w
		}
		l.sep, rest = rest[:i], rest[i:]
		groups = append(groups, n)
		n = 0
	}
	// groups holds the size of each group, from the left. The first group can
	// be shorter than the others
	groups = append(groups, n)
	if len(groups) > 1 {
		l.primary = groups[len(groups)-1]
		l.secondary = l.primary
		if len(groups) > 2 {
			l.secondary = groups[len(groups)-2]
		}
	}
	// note: the minus sign of a locale can hold a direction mark
	l.minus = "-"
	if minus, end, ok := strings.Cut(sprint(-1), l.digits[1]); ok {
		l.minus, l.minusEnd = minus, end
	}
	return l
}

// digit returns the digit of the locale at the start of s, or "".
func (l *numericLocale) digit(s string) string {
	for _, d := range l.digits {
		if d != "" && strings.HasPrefix(s, d) {
			return d
		}
	}
	return ""
}

// format returns s, the text of a finite number with ASCII digits, in the
// locale. s is an optional minus sign, the digits of the integer part, and an
// optional fraction and exponent.
func (l *numericLocale) format(s string) string {
	var b strings.Builder
	neg := false
	if rest, ok := strings.CutPrefix(s, "-"); ok {
		neg, s = true, rest
		b.WriteString(l.minus)
	}
	n := strings.IndexFunc(s, func(r rune) bool { return r < '0' || '9' < r })
	if n == -1 {
		n = len(s)
	}
	integer, rest := s[:n], s[n:]
	// a separator comes before a digit when the digits after it, r, make
	// whole groups
	group := l.primary != 0 && len(integer) > l.primary
	for i := range len(integer) {
		if r := len(integer) - i; group && i != 0 && (r == l.primary || r > l.primary && (r-l.primary)%l.secondary == 0) {
			b.WriteString(l.sep)
		}
		b.WriteString(l.digits[integer[i]-'0'])
	}
	if fraction, ok := strings.CutPrefix(rest, "."); ok {
		b.WriteString(l.point)
		rest = fraction
	}
	for i := range len(rest) {
		if c := rest[i]; '0' <= c && c <= '9' {
			b.WriteString(l.digits[c-'0'])
		} else {
			b.WriteByte(c)
		}
	}
	if neg {
		b.WriteString(l.minusEnd)
	}
	return b.String()
}

// deref returns the value that v points to, by one level. It returns nil for
// a nil pointer, and for a pointer to a nil pointer. A scan of a null value
// into a destination of type **T leaves such a pointer.
func deref(v any) any {
	if z, ok := v.(*any); ok {
		if z == nil {
			return nil
		}
		v = *z
	} else if val := reflect.ValueOf(v); val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		v = val.Elem().Interface()
	}
	// note: a method of a nil pointer, such as Decompose or String, can panic
	if val := reflect.ValueOf(v); val.Kind() == reflect.Pointer && val.IsNil() {
		return nil
	}
	return v
}
