package site

import (
	"os"
	"testing"
)

// TestMain removes the installer PATH directories once the tests have run.
func TestMain(m *testing.M) {
	code := m.Run()
	removeInstallerPaths()
	os.Exit(code)
}
