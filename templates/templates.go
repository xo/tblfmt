// Package templates contains embedded templates for the tblfmt package.
package templates

import (
	"embed"
)

// Templates are embedded html and go templates.
//go:embed *.txt
var Templates embed.FS
