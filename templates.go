package tblfmt

import html "html/template"

var (
	templates = map[string]string{
		"html": `
<table {{.Attributes | attr}}>
	<caption>{{.Title}}</caption>
	<thead>
		<tr>
{{range .Headers}}			<th align="{{.Align}}">{{.}}</th>
{{end}}
		</tr>
	</thead>
	<tbody>
{{range .Rows}}		<tr>
{{range .}}			<td align="{{.Value.Align}}">{{.Value}}</td>
{{end}}		</tr>
{{end}}
	</tbody>
</table>`,
		"asciidoc": `
[%header]
{{if .Title.Buf}}.{{.Title}}
{{end}}|===
{{range .Headers}}|{{.}}
{{end}}
{{range .Rows}}{{range .}}|{{.Value}}
{{end}}
{{end}}|===`,
	}
	htmlFuncMap = html.FuncMap{
		"attr": func(s string) html.HTMLAttr {
			return html.HTMLAttr(s)
		},
		"safe": func(s string) html.HTML {
			return html.HTML(s)
		},
	}
)
