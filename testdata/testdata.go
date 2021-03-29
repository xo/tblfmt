package testdata

import (
	"embed"
)

// Testdata is the set of testdata.
//
//go:embed *.txt
var Testdata embed.FS
