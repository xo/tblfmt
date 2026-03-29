package tblfmt

import (
	"fmt"
	"html"
	"io"
	"strings"
)

// Template is template data.
type Template struct {
	Attributes string
	Headers    []*Value
	Rows       [][]*Value
	SkipHeader bool
	Title      *Value
}

// WriteHTMLTo writes simple HTML output to the writer.
func WriteHTMLTo(w io.Writer, tpl *Template) error {
	// {{ $headers := .Headers }}{{ $rows := .Rows }}<table{{ .Attributes | attr }}>
	//   <caption>{{ .Title }}</caption>
	//   <thead>
	//     <tr>{{ range $i, $h := $headers }}
	//       <th align="{{ $h.Align.String | toLower }}">{{ $h }}</th>{{ end }}
	//     </tr>
	//   </thead>
	//   <tbody>{{ range $i, $r := $rows }}
	//     <tr>{{ range $j, $c := $r  }}
	//       <td align="{{ $c.Align.String | toLower }}">{{ $c }}</td>{{ end }}
	//     </tr>{{ end }}
	//   </tbody>
	// </table>
	fmt.Fprint(w, "<table")
	if len(tpl.Attributes) != 0 {
		fmt.Fprint(w, "", tpl.Attributes)
	}
	fmt.Fprintf(w, ">\n  <caption>%s</caption>\n  <thead>\n    <tr>\n", tpl.Title)
	for _, h := range tpl.Headers {
		fmt.Fprintf(w, "      <th align=%q>%s</th>\n", strings.ToLower(h.Align.String()), html.EscapeString(h.String()))
	}
	fmt.Fprint(w, "    </tr>\n  </thead>\n  <tbody>")
	for _, r := range tpl.Rows {
		fmt.Fprint(w, "\n    <tr>")
		for _, c := range r {
			fmt.Fprintf(w, "\n      <td align=%q>%s</td>", strings.ToLower(c.Align.String()), html.EscapeString(c.String()))
		}
		fmt.Fprint(w, "\n    </tr>")
	}
	fmt.Fprintln(w, "\n  </tbody>\n</table>")
	return nil
}

// WriteAsciidocTo writes simple asciidoc output to the writer.
func WriteAsciidocTo(w io.Writer, tpl *Template) error {
	// {{ $headers := .Headers }}{{ $rows := .Rows }}[%header]{{ if .Title.Buf }}
	// .{{ .Title }}{{ end }}
	// |==={{ range $i, $h := $headers }}
	// |{{ $h }}{{ end }}{{ range $i, $r := $rows }}
	// {{ range $j, $c := $r }}|{{ $c }}{{ end }}{{ end }}
	// |===
	fmt.Fprint(w, "[%header]")
	if s := tpl.Title.String(); s != "" {
		fmt.Fprintf(w, "\n%s", tpl.Title.String())
	}
	fmt.Fprint(w, "\n|===")
	for _, h := range tpl.Headers {
		fmt.Fprintf(w, "\n|%s", h)
	}
	for _, r := range tpl.Rows {
		fmt.Fprintln(w)
		for _, c := range r {
			fmt.Fprintf(w, "|%s", c)
		}
	}
	fmt.Fprintln(w, "\n|===")
	return nil
}

// WriteVerticalTo writes simple vertical output to the writer.
func WriteVerticalTo(w io.Writer, tpl *Template) error {
	// {{ $headers := .Headers }}{{ range $i, $r := .Rows }}*************************** {{ inc $i }}. row ***************************{{ range $j, $c := $r }}
	// {{ index $headers $j }}: {{ $c }}{{ end }}
	// {{ end -}}
	const divider = `***************************`
	for i, r := range tpl.Rows {
		fmt.Fprintf(w, "%s %d. %s\n", divider, i+1, divider)
		for j, c := range r {
			fmt.Fprintf(w, "%s: %s\n", tpl.Headers[j], c)
		}
	}
	return nil
}
