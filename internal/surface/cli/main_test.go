package cli

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// TestMain defaults the ahoy build-vintage seam to a fresh, determinable,
// non-dogfood value for the whole package. runCLI executes the commands
// in-process through this unstamped go-test binary, so without the override
// every `ahoy install` exercised here would hit the itd-111 unknown-vintage
// refusal.
// cliTestAsBinaryEnv marks a child process of this test binary that must act
// as the abcd binary rather than run the tests.
const cliTestAsBinaryEnv = "ABCD_CLI_TEST_AS_BINARY"

func TestMain(m *testing.M) {
	// The deep installability tier re-executes the running binary as its
	// isolated child. Under `go test` that binary is this test binary, so it
	// stands in for abcd when the runner marks it: the real subprocess path runs
	// in every test that reaches it, and no test runs the suite recursively.
	if os.Getenv(cliTestAsBinaryEnv) == "1" {
		os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
	}
	pageRunnerExtraEnv = []string{cliTestAsBinaryEnv + "=1"}
	ahoy.SetCurrentVintageForTest(func() vintage.Current {
		return vintage.Current{Revision: "testvintage", Known: true}
	})
	os.Exit(m.Run())
}
