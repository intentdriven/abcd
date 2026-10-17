package cloudscripts

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// builtAbcd is abcd built from this checkout once for the whole package: the
// scripts run it as their ABCD_BIN and as the attribution gate's outbound
// half.
var builtAbcd gittest.Abcd

func TestMain(m *testing.M) {
	code := m.Run()
	builtAbcd.Remove()
	os.Exit(code)
}
