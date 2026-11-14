package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestMain keeps the developer's own ~/.abcd.noindex/rules.json out of every test that
// did not lay one out: while HOME is still the process's own, the user layer
// reads as absent, and a test that sets HOME to a fixture gets that fixture's
// user layer (spc-23). Without it, a machine carrying a user layer would change
// what every Load in this package returns.
//
// It also doubles as the LOAD idiom test's burner: run with loadBurnerEnv set,
// the test binary runs no test and only sleeps (runLoadBurner).
func TestMain(m *testing.M) {
	if os.Getenv(loadBurnerEnv) != "" {
		os.Exit(runLoadBurner(os.Args))
	}
	real := os.Getenv("HOME")
	restore := SwapUserHomeForTest(func() (string, error) {
		if home := os.Getenv("HOME"); home != real {
			return home, nil
		}
		return "", nil
	})
	code := m.Run()
	restore()
	os.Exit(code)
}

// loadBurnerEnv marks the one process that is TestLoadDomainIdiomOwnsTheWholeGroup's
// burner. The test sets it only in the environment of the load script it runs,
// never in its own, so no process the tests start outside that script sees it.
const loadBurnerEnv = "ABCD_RULES_TEST_BURNER"

// loadBurnerPrefix starts every burner's name (burnerName).
const loadBurnerPrefix = "abcdburn"

// runLoadBurner is the burner: the test binary run through a symlink under the
// test's unique burner name, standing in for the system's sleep. The system's
// sleep cannot be the burner: where coreutils is one multi-call binary that
// picks its tool from the name it runs under, it is not sleep under the
// burner's name and exits at once (iss-2611141634129814). It sleeps for the
// seconds its one argument names, 300 without one, and installs no signal
// handler, so SIGTERM and SIGKILL end it as they end sleep. It refuses to run
// under any other name, so the variable leaking into a real test run fails
// that run at once instead of leaving it asleep.
func runLoadBurner(args []string) int {
	if len(args) == 0 || !strings.HasPrefix(filepath.Base(args[0]), loadBurnerPrefix) {
		fmt.Fprintf(os.Stderr, "%s is set, but this process is not a load burner (%q); unset it to run the tests\n", loadBurnerEnv, args)
		return 2
	}
	secs := 300
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "load burner: %q is not a whole number of seconds\n", args[1])
			return 2
		}
		secs = n
	}
	time.Sleep(time.Duration(secs) * time.Second)
	return 0
}
