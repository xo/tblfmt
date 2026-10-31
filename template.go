package tblfmt

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Template holds the data for a template.
type Template struct {
	Attributes string
	Headers    []*Value
	Rows       [][]*Value
	SkipHeader bool
	Title      *Value
	// Aligns are the alignments of the columns.
	Aligns []Align
	// Border is the border, as the psql variable border sets it.
	Border int
	// Expanded is true when the template writes each row as a record.
	Expanded bool
	// Footer is the summary line, or empty for none. The encoder follows the
	// rule of psql: a table gets the row count or the custom summary, and
	// expanded records get only a custom summary.
	Footer string
}

// WriteHTMLTo writes HTML output to the writer, as psql print_html_text and
// print_html_vertical do.
func WriteHTMLTo(w io.Writer, tpl *Template) error {
	fmt.Fprintf(w, "<table border=\"%d\"", tpl.Border)
	if tpl.Attributes != "" {
		fmt.Fprintf(w, " %s", tpl.Attributes)
	}
	fmt.Fprint(w, ">\n")
	if !tpl.SkipHeader && hasTitle(tpl) {
		fmt.Fprint(w, "  <caption>")
		writeHTML(w, tpl.Title.Buf)
		fmt.Fprint(w, "</caption>\n")
	}
	switch {
	case tpl.Expanded:
		for n, row := range tpl.Rows {
			if !tpl.SkipHeader {
				fmt.Fprintf(w, "\n  <tr><td colspan=\"2\" align=\"center\">Record %d</td></tr>\n", n+1)
			} else {
				fmt.Fprint(w, "\n  <tr><td colspan=\"2\">&nbsp;</td></tr>\n")
			}
			for i, v := range row {
				fmt.Fprint(w, "  <tr valign=\"top\">\n    <th>")
				writeHTML(w, tpl.Headers[i].Buf)
				fmt.Fprintf(w, "</th>\n    <td align=\"%s\">", alignWord(tpl, i))
				writeHTMLCell(w, v.Buf)
				fmt.Fprint(w, "</td>\n  </tr>\n")
			}
		}
	default:
		if !tpl.SkipHeader {
			fmt.Fprint(w, "  <tr>\n")
			for _, h := range tpl.Headers {
				fmt.Fprint(w, "    <th align=\"center\">")
				writeHTML(w, h.Buf)
				fmt.Fprint(w, "</th>\n")
			}
			fmt.Fprint(w, "  </tr>\n")
		}
		for _, row := range tpl.Rows {
			fmt.Fprint(w, "  <tr valign=\"top\">\n")
			for i, v := range row {
				fmt.Fprintf(w, "    <td align=\"%s\">", alignWord(tpl, i))
				writeHTMLCell(w, v.Buf)
				fmt.Fprint(w, "</td>\n")
			}
			fmt.Fprint(w, "  </tr>\n")
		}
	}
	fmt.Fprint(w, "</table>\n")
	if !tpl.SkipHeader && tpl.Footer != "" {
		fmt.Fprint(w, "<p>")
		writeHTML(w, []byte(tpl.Footer))
		fmt.Fprint(w, "<br />\n</p>")
	}
	fmt.Fprint(w, "\n")
	return nil
}

// WriteAsciidocTo writes AsciiDoc output to the writer, as psql
// print_asciidoc_text and print_asciidoc_vertical do.
func WriteAsciidocTo(w io.Writer, tpl *Template) error {
	// psql starts the table in a new paragraph
	fmt.Fprint(w, "\n")
	if !tpl.SkipHeader && hasTitle(tpl) {
		fmt.Fprint(w, ".")
		_, _ = w.Write(tpl.Title.Buf)
		fmt.Fprint(w, "\n")
	}
	frame := [...]string{`,frame="none",grid="none"`, `,frame="none"`, `,frame="all",grid="all"`}
	if tpl.Expanded {
		fmt.Fprint(w, `[cols="h,l"`)
	} else {
		if !tpl.SkipHeader {
			fmt.Fprint(w, `[options="header",cols="`)
		} else {
			fmt.Fprint(w, `[cols="`)
		}
		for i := range tpl.Headers {
			if i != 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, asciidocAlign(tpl, i))
		}
		fmt.Fprint(w, `"`)
	}
	if 0 <= tpl.Border && tpl.Border < len(frame) {
		fmt.Fprint(w, frame[tpl.Border])
	}
	fmt.Fprint(w, "]\n|====\n")
	switch {
	case tpl.Expanded:
		for n, row := range tpl.Rows {
			if !tpl.SkipHeader {
				fmt.Fprintf(w, "2+^|Record %d\n", n+1)
			} else {
				fmt.Fprint(w, "2+|\n")
			}
			for i, v := range row {
				fmt.Fprint(w, "<l|")
				writeAsciidoc(w, tpl.Headers[i].Buf)
				fmt.Fprintf(w, " %s|", asciidocAlign(tpl, i))
				if isBlank(v.Buf) {
					fmt.Fprint(w, " ")
				} else {
					writeAsciidoc(w, v.Buf)
				}
				fmt.Fprint(w, "\n")
			}
		}
	default:
		if !tpl.SkipHeader {
			for i, h := range tpl.Headers {
				if i != 0 {
					fmt.Fprint(w, " ")
				}
				fmt.Fprint(w, "^l|")
				writeAsciidoc(w, h.Buf)
			}
			fmt.Fprint(w, "\n")
		}
		for _, row := range tpl.Rows {
			for i, v := range row {
				if i != 0 {
					fmt.Fprint(w, " ")
				}
				fmt.Fprint(w, "|")
				// psql writes no needless space after the last cell
				switch {
				case !isBlank(v.Buf):
					writeAsciidoc(w, v.Buf)
				case i != len(row)-1:
					fmt.Fprint(w, " ")
				}
			}
			fmt.Fprint(w, "\n")
		}
	}
	fmt.Fprint(w, "|====\n")
	if !tpl.SkipHeader && tpl.Footer != "" {
		fmt.Fprint(w, "\n....\n", tpl.Footer, "\n....\n")
	}
	return nil
}

// WriteVerticalTo writes vertical output to the writer, as the mysql client
// writes a result for \G. It right aligns each column name to the widest name,
// by display width. With SkipHeader, it writes no names, as mysql does with
// --skip-column-names. See D60 in docs/PLAN.md.
func WriteVerticalTo(w io.Writer, tpl *Template) error {
	const divider = `***************************`
	var width int
	for _, h := range tpl.Headers {
		width = max(width, h.MaxWidth())
	}
	for i, r := range tpl.Rows {
		fmt.Fprintf(w, "%s %d. row %s\n", divider, i+1, divider)
		for j, c := range r {
			if !tpl.SkipHeader {
				fmt.Fprintf(w, "%s%s: ", strings.Repeat(" ", width-tpl.Headers[j].MaxWidth()), tpl.Headers[j])
			}
			fmt.Fprintf(w, "%s\n", c)
		}
	}
	return nil
}

// WriteLaTeXTo writes LaTeX output to the writer, as psql print_latex_text
// and print_latex_vertical do.
func WriteLaTeXTo(w io.Writer, tpl *Template) error {
	if tpl.Expanded {
		return writeLaTeXVertical(w, tpl)
	}
	border := min(tpl.Border, 3)
	if !tpl.SkipHeader && hasTitle(tpl) {
		fmt.Fprint(w, "\\begin{center}\n")
		writeLaTeX(w, tpl.Title.Buf)
		fmt.Fprint(w, "\n\\end{center}\n\n")
	}
	// the environment, and the alignments and the borders of the columns
	fmt.Fprint(w, "\\begin{tabular}{")
	if border >= 2 {
		fmt.Fprint(w, "| ")
	}
	for i := range tpl.Headers {
		fmt.Fprint(w, alignLetter(tpl, i))
		if border != 0 && i < len(tpl.Headers)-1 {
			fmt.Fprint(w, " | ")
		}
	}
	if border >= 2 {
		fmt.Fprint(w, " |")
	}
	fmt.Fprint(w, "}\n")
	if !tpl.SkipHeader && border >= 2 {
		fmt.Fprint(w, "\\hline\n")
	}
	if !tpl.SkipHeader {
		for i, h := range tpl.Headers {
			if i != 0 {
				fmt.Fprint(w, " & ")
			}
			fmt.Fprint(w, "\\textit{")
			writeLaTeX(w, h.Buf)
			fmt.Fprint(w, "}")
		}
		fmt.Fprint(w, " \\\\\n\\hline\n")
	}
	for _, row := range tpl.Rows {
		for i, v := range row {
			if i != 0 {
				fmt.Fprint(w, " & ")
			}
			writeLaTeX(w, v.Buf)
		}
		fmt.Fprint(w, " \\\\\n")
		if border == 3 {
			fmt.Fprint(w, "\\hline\n")
		}
	}
	if border == 2 {
		fmt.Fprint(w, "\\hline\n")
	}
	fmt.Fprint(w, "\\end{tabular}\n\n\\noindent ")
	if !tpl.SkipHeader && tpl.Footer != "" {
		writeLaTeX(w, []byte(tpl.Footer))
		fmt.Fprint(w, " \\\\\n")
	}
	fmt.Fprint(w, "\n")
	return nil
}

// writeLaTeXVertical writes LaTeX records, as psql print_latex_vertical does.
func writeLaTeXVertical(w io.Writer, tpl *Template) error {
	border := min(tpl.Border, 2)
	if !tpl.SkipHeader && hasTitle(tpl) {
		fmt.Fprint(w, "\\begin{center}\n")
		writeLaTeX(w, tpl.Title.Buf)
		fmt.Fprint(w, "\n\\end{center}\n\n")
	}
	fmt.Fprint(w, "\\begin{tabular}{", []string{"cl", "c|l", "|c|l|"}[max(border, 0)], "}\n")
	for n, row := range tpl.Rows {
		if !tpl.SkipHeader {
			if border == 2 {
				fmt.Fprintf(w, "\\hline\n\\multicolumn{2}{|c|}{\\textit{Record %d}} \\\\\n", n+1)
			} else {
				fmt.Fprintf(w, "\\multicolumn{2}{c}{\\textit{Record %d}} \\\\\n", n+1)
			}
		}
		if border >= 1 {
			fmt.Fprint(w, "\\hline\n")
		}
		for i, v := range row {
			writeLaTeX(w, tpl.Headers[i].Buf)
			fmt.Fprint(w, " & ")
			writeLaTeX(w, v.Buf)
			fmt.Fprint(w, " \\\\\n")
		}
	}
	if border == 2 {
		fmt.Fprint(w, "\\hline\n")
	}
	fmt.Fprint(w, "\\end{tabular}\n\n\\noindent ")
	if !tpl.SkipHeader && tpl.Footer != "" {
		writeLaTeX(w, []byte(tpl.Footer))
		fmt.Fprint(w, " \\\\\n")
	}
	fmt.Fprint(w, "\n")
	return nil
}

// WriteLaTeXLongtableTo writes LaTeX longtable output to the writer, as psql
// print_latex_longtable_text does. The attributes hold the widths of the left
// aligned columns, as fractions of the text width. Expanded output is the
// same as for WriteLaTeXTo.
func WriteLaTeXLongtableTo(w io.Writer, tpl *Template) error {
	if tpl.Expanded {
		return writeLaTeXVertical(w, tpl)
	}
	border := min(tpl.Border, 3)
	// the environment, and the alignments and the borders of the columns. A
	// left aligned column takes the next width in the attributes, or the last
	// one when there are no more.
	fmt.Fprint(w, "\\begin{longtable}{")
	if border >= 2 {
		fmt.Fprint(w, "| ")
	}
	widths := strings.Fields(tpl.Attributes)
	var last string
	for i := range tpl.Headers {
		switch letter := alignLetter(tpl, i); {
		case letter == "l" && len(widths) != 0:
			last, widths = widths[0], widths[1:]
			fmt.Fprintf(w, "p{%s\\textwidth}", last)
		case letter == "l" && last != "":
			fmt.Fprintf(w, "p{%s\\textwidth}", last)
		default:
			fmt.Fprint(w, letter)
		}
		if border != 0 && i < len(tpl.Headers)-1 {
			fmt.Fprint(w, " | ")
		}
	}
	if border >= 2 {
		fmt.Fprint(w, " |")
	}
	fmt.Fprint(w, "}\n")
	if !tpl.SkipHeader {
		// the head of the first page, and then of each page after it
		for _, end := range []string{"\\midrule\n\\endfirsthead\n", "\\endhead\n"} {
			if border >= 2 {
				fmt.Fprint(w, "\\toprule\n")
			}
			for i, h := range tpl.Headers {
				if i != 0 {
					fmt.Fprint(w, " & ")
				}
				fmt.Fprint(w, "\\small\\textbf{\\textit{")
				writeLaTeX(w, h.Buf)
				fmt.Fprint(w, "}}")
			}
			fmt.Fprint(w, " \\\\\n")
			// a line under each row already ends the head
			if end == "\\endhead\n" && border != 3 {
				fmt.Fprint(w, "\\midrule\n")
			}
			fmt.Fprint(w, end)
		}
		switch {
		case hasTitle(tpl):
			for _, end := range []string{" (Continued)]{", "]{"} {
				if border == 2 {
					fmt.Fprint(w, "\\bottomrule\n")
				}
				fmt.Fprint(w, "\\caption[")
				writeLaTeX(w, tpl.Title.Buf)
				fmt.Fprint(w, end)
				writeLaTeX(w, tpl.Title.Buf)
				if end == "]{" {
					fmt.Fprint(w, "}\n\\endlastfoot\n")
				} else {
					fmt.Fprint(w, "}\n\\endfoot\n")
				}
			}
		case border >= 2:
			fmt.Fprint(w, "\\bottomrule\n\\endfoot\n\\bottomrule\n\\endlastfoot\n")
		}
	}
	for _, row := range tpl.Rows {
		for i, v := range row {
			if i != 0 {
				fmt.Fprint(w, "\n&\n")
			}
			fmt.Fprint(w, "\\raggedright{")
			writeLaTeX(w, v.Buf)
			fmt.Fprint(w, "}")
		}
		fmt.Fprint(w, " \\tabularnewline\n")
		if border == 3 {
			fmt.Fprint(w, " \\hline\n")
		}
	}
	fmt.Fprint(w, "\\end{longtable}\n")
	return nil
}

// WriteTroffMSTo writes troff output with the ms macros to the writer, as psql
// print_troff_ms_text and print_troff_ms_vertical do.
func WriteTroffMSTo(w io.Writer, tpl *Template) error {
	border := min(tpl.Border, 2)
	if !tpl.SkipHeader && hasTitle(tpl) {
		fmt.Fprint(w, ".LP\n.DS C\n")
		writeTroff(w, tpl.Title.Buf)
		fmt.Fprint(w, "\n.DE\n")
	}
	fmt.Fprint(w, ".LP\n.TS\n")
	if border == 2 {
		fmt.Fprint(w, "center box;\n")
	} else {
		fmt.Fprint(w, "center;\n")
	}
	if tpl.Expanded {
		writeTroffVertical(w, tpl, border)
	} else {
		for i := range tpl.Headers {
			fmt.Fprint(w, alignLetter(tpl, i))
			if border > 0 && i < len(tpl.Headers)-1 {
				fmt.Fprint(w, " | ")
			}
		}
		fmt.Fprint(w, ".\n")
		if !tpl.SkipHeader {
			for i, h := range tpl.Headers {
				if i != 0 {
					fmt.Fprint(w, "\t")
				}
				fmt.Fprint(w, "\\fI")
				writeTroff(w, h.Buf)
				fmt.Fprint(w, "\\fP")
			}
			fmt.Fprint(w, "\n_\n")
		}
		for _, row := range tpl.Rows {
			for i, v := range row {
				if i != 0 {
					fmt.Fprint(w, "\t")
				}
				writeTroff(w, v.Buf)
			}
			fmt.Fprint(w, "\n")
		}
	}
	fmt.Fprint(w, ".TE\n.DS L\n")
	if !tpl.SkipHeader && tpl.Footer != "" {
		writeTroff(w, []byte(tpl.Footer))
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, ".DE\n")
	return nil
}

// writeTroffVertical writes troff records, as psql print_troff_ms_vertical
// does. A record header line and the lines of a record need different
// formats, so the writer changes the format with .T& between them.
func writeTroffVertical(w io.Writer, tpl *Template, border int) {
	const (
		formatNone = iota
		formatHeader
		formatBody
	)
	format := formatNone
	if tpl.SkipHeader {
		fmt.Fprint(w, "c l;\n")
	}
	for n, row := range tpl.Rows {
		if !tpl.SkipHeader {
			if format != formatHeader {
				if border == 2 && n > 0 {
					fmt.Fprint(w, "_\n")
				}
				if format != formatNone {
					fmt.Fprint(w, ".T&\n")
				}
				fmt.Fprint(w, "c s.\n")
				format = formatHeader
			}
			fmt.Fprintf(w, "\\fIRecord %d\\fP\n", n+1)
		}
		if border >= 1 {
			fmt.Fprint(w, "_\n")
		}
		for i, v := range row {
			if !tpl.SkipHeader && format != formatBody {
				if format != formatNone {
					fmt.Fprint(w, ".T&\n")
				}
				if border != 1 {
					fmt.Fprint(w, "c l.\n")
				} else {
					fmt.Fprint(w, "c | l.\n")
				}
				format = formatBody
			}
			writeTroff(w, tpl.Headers[i].Buf)
			fmt.Fprint(w, "\t")
			writeTroff(w, v.Buf)
			fmt.Fprint(w, "\n")
		}
	}
}

// hasTitle reports whether the template has a title.
func hasTitle(tpl *Template) bool {
	return tpl.Title != nil && len(tpl.Title.Buf) != 0
}

// alignLetter returns the letter that LaTeX and troff use for the alignment
// of column i: r for a right aligned column, and l for any other.
func alignLetter(tpl *Template, i int) string {
	if i < len(tpl.Aligns) && tpl.Aligns[i] == AlignRight {
		return "r"
	}
	return "l"
}

// alignWord returns the word that HTML uses for the alignment of column i.
func alignWord(tpl *Template, i int) string {
	if alignLetter(tpl, i) == "r" {
		return "right"
	}
	return "left"
}

// asciidocAlign returns the cell format that AsciiDoc uses for the alignment
// of column i.
func asciidocAlign(tpl *Template, i int) string {
	if alignLetter(tpl, i) == "r" {
		return ">l"
	}
	return "<l"
}

// isBlank reports whether the text holds only spaces and tabs, as psql tests a
// cell with strspn.
func isBlank(text []byte) bool {
	return len(bytes.Trim(text, " \t")) == 0
}

// writeHTML writes the text with the escapes of HTML, as psql
// html_escaped_print does. A newline is a line break, and a space at the start
// is a space that does not break, which keeps the indent of EXPLAIN.
func writeHTML(w io.Writer, text []byte) {
	leading := true
	for _, c := range text {
		switch c {
		case '&':
			fmt.Fprint(w, "&amp;")
		case '<':
			fmt.Fprint(w, "&lt;")
		case '>':
			fmt.Fprint(w, "&gt;")
		case '\n':
			fmt.Fprint(w, "<br />\n")
		case '"':
			fmt.Fprint(w, "&quot;")
		case ' ':
			if leading {
				fmt.Fprint(w, "&nbsp;")
			} else {
				fmt.Fprint(w, " ")
			}
		default:
			_, _ = w.Write([]byte{c})
		}
		if c != ' ' {
			leading = false
		}
	}
}

// writeHTMLCell writes a cell. psql writes a cell that holds only spaces and
// tabs as a space that does not break.
func writeHTMLCell(w io.Writer, text []byte) {
	if isBlank(text) {
		fmt.Fprint(w, "&nbsp; ")
		return
	}
	writeHTML(w, text)
}

// writeAsciidoc writes the text with the escape of AsciiDoc for the cell
// separator, as psql asciidoc_escaped_print does.
func writeAsciidoc(w io.Writer, text []byte) {
	_, _ = w.Write(bytes.ReplaceAll(text, []byte{'|'}, []byte(`\|`)))
}

// latexEscapes are the escapes of LaTeX, as psql latex_escaped_print writes
// them. psql writes a newline as a line break, and notes that this is not
// right.
var latexEscapes = map[byte]string{
	'#':  `\#`,
	'$':  `\$`,
	'%':  `\%`,
	'&':  `\&`,
	'<':  `\textless{}`,
	'>':  `\textgreater{}`,
	'\\': `\textbackslash{}`,
	'^':  `\^{}`,
	'_':  `\_`,
	'{':  `\{`,
	'|':  `\textbar{}`,
	'}':  `\}`,
	'~':  `\~{}`,
	'\n': `\\`,
}

// writeLaTeX writes the text with the escapes of LaTeX.
func writeLaTeX(w io.Writer, text []byte) {
	start := 0
	for i, c := range text {
		if s, ok := latexEscapes[c]; ok {
			_, _ = w.Write(text[start:i])
			fmt.Fprint(w, s)
			start = i + 1
		}
	}
	_, _ = w.Write(text[start:])
}

// writeTroff writes the text with the escape of troff for a backslash, as psql
// troff_ms_escaped_print does.
func writeTroff(w io.Writer, text []byte) {
	_, _ = w.Write(bytes.ReplaceAll(text, []byte{'\\'}, []byte(`\(rs`)))
}
