package claim

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/testshard"
)

// TestMain runs the package as several test processes when make preflight
// asks: one is too long for its package budget on a loaded machine
// (iss-2610251004378066).
func TestMain(m *testing.M) {
	if code, ok := testshard.Shard(); ok {
		os.Exit(code)
	}
	os.Exit(m.Run())
}
