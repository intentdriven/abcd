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
	if actAsBinary(os.Getenv(cliTestAsBinaryEnv), os.Args[1:]) {
		os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
	}
	pageRunnerExtraEnv = []string{cliTestAsBinaryEnv + "=1"}
	ahoy.SetCurrentVintageForTest(func() vintage.Current {
		return vintage.Current{Revision: "testvintage", Known: true}
	})
	os.Exit(m.Run())
}

// actAsBinary reports whether this test binary was started as the deep tier's
// child and must act as abcd rather than run the tests: the marker AND exactly
// the child's arguments (subprocessPageRunner runs `launch smoke-pages`), so a
// marker left in the environment of a test run never stands the suite down.
func actAsBinary(marker string, args []string) bool {
	return marker == "1" && len(args) == 2 && args[0] == "launch" && args[1] == "smoke-pages"
}

// TestActAsBinaryOnlyForTheSmokePagesChild: the marker alone never turns the
// test binary into abcd. An ambient marker in the environment of a `go test`
// run would otherwise run zero tests, print the status board and pass
// (iss-2609251902439148); only the deep tier's child invocation re-enters.
func TestActAsBinaryOnlyForTheSmokePagesChild(t *testing.T) {
	if !actAsBinary("1", []string{"launch", "smoke-pages"}) {
		t.Error("the deep tier's child must act as the abcd binary")
	}
	for _, args := range [][]string{
		{"-test.run=TestX", "-test.v"},
		{"-test.paniconexit0", "-test.timeout=10m0s"},
		{},
		{"launch", "smoke-pages", "extra"},
		{"launch"},
	} {
		if actAsBinary("1", args) {
			t.Errorf("the marker with %q must run the tests, not act as abcd", args)
		}
	}
	if actAsBinary("", []string{"launch", "smoke-pages"}) {
		t.Error("without the marker the binary always runs the tests")
	}
}
