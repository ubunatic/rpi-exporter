package collector_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// When running locally, add bin/ (project root) to PATH so the vcgencmd-stub
	// binary is found by IsRpi() and runVCGenCmd(). On a real Pi, vcgencmd is
	// already in PATH and this is a no-op.
	if stub, err := filepath.Abs("../bin"); err == nil {
		if _, err := os.Stat(filepath.Join(stub, "vcgencmd")); err == nil {
			os.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
	}
	os.Exit(m.Run())
}
