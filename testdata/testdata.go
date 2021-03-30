package testdata

//go:generate go run gencrosstab.go
//go:generate go run genformats.go

import (
	"embed"
)

// Testdata is the set of testdata.
//
//go:embed *.txt
var Testdata embed.FS
