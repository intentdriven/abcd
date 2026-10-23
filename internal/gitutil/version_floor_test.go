package gitutil_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestGitAtLeastNamesTheVersionInDigitsAlone: the floor check says whether
// git is at least major.minor, and names its version as major.minor, digits
// alone; a probe that fails, or a line that does not parse, is below every
// floor and names none.
func TestGitAtLeastNamesTheVersionInDigitsAlone(t *testing.T) {
	for _, c := range []struct {
		line    string
		err     error
		ok      bool
		version string
	}{
		{"git version 2.28.1 (Apple Git-1)", nil, false, "2.28"},
		{"git version 2.29.0", nil, true, "2.29"},
		{"git version 3.0.0.windows.1", nil, true, "3.0"},
		{"git version two", nil, false, ""},
		{"", errors.New("no git"), false, ""},
	} {
		restore := gitutil.SwapGitVersionForTest(func() (string, error) { return c.line, c.err })
		ok, version := gitutil.GitAtLeast(2, 29)
		restore()
		if ok != c.ok || version != c.version {
			t.Errorf("GitAtLeast(2, 29) under %q = %v, %q; want %v, %q", c.line, ok, version, c.ok, c.version)
		}
	}
}

// TestAFailedVersionProbeIsAskedAgain: a git that could not run once (a fork
// refused on a loaded machine) is not kept as a git below every floor for the
// rest of the process; the next question probes again (iss-2610051434156079).
func TestAFailedVersionProbeIsAskedAgain(t *testing.T) {
	calls := 0
	restore := gitutil.SwapGitVersionForTest(func() (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("fork/exec git: resource temporarily unavailable")
		}
		return "git version 2.52.0", nil
	})
	defer restore()
	if ok, _ := gitutil.GitAtLeast(2, 44); ok {
		t.Fatal("a failed probe read as a git at the floor")
	}
	if ok, v := gitutil.GitAtLeast(2, 44); !ok || v != "2.52" {
		t.Fatalf("the probe after a failure = %v %q; want it asked again and at the floor", ok, v)
	}
	if ok, _ := gitutil.GitAtLeast(2, 44); !ok || calls != 2 {
		t.Fatalf("a probe that ran was asked again (%d calls); want it kept", calls)
	}
}

// TestConcurrentVersionQuestionsShareOneProbe: callers that ask while a probe
// is running wait for it rather than each starting their own, and all read
// its answer (iss-2610181802344551).
func TestConcurrentVersionQuestionsShareOneProbe(t *testing.T) {
	var calls atomic.Int32
	restore := gitutil.SwapGitVersionForTest(func() (string, error) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		return "git version 2.52.0", nil
	})
	defer restore()
	var wg sync.WaitGroup
	oks := make([]bool, 10)
	for i := range oks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			oks[i], _ = gitutil.GitAtLeast(2, 44)
		}()
	}
	wg.Wait()
	for i, ok := range oks {
		if !ok {
			t.Errorf("caller %d did not read the shared probe's answer", i)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("%d probes ran for 10 concurrent questions; want 1", n)
	}
}
