package tblfmt

import (
	"os"
	"runtime"
	"testing"
)

// platformNewline is the newline of the platform, before TestMain sets it.
var platformNewline string

// TestMain sets the newline to "\n", so that the golden files and the expected
// output of each test hold on every platform. TestPlatformNewline tests the
// newline of the platform.
func TestMain(m *testing.M) {
	platformNewline = string(newline)
	newline = []byte("\n")
	os.Exit(m.Run())
}

// TestPlatformNewline makes sure that the encoders write "\r\n" on Windows,
// and "\n" on any other platform, by default.
func TestPlatformNewline(t *testing.T) {
	t.Parallel()
	exp := "\n"
	if runtime.GOOS == "windows" {
		exp = "\r\n"
	}
	if platformNewline != exp {
		t.Errorf("expected %q, got: %q", exp, platformNewline)
	}
}
