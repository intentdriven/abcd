//go:build unix

package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A copy controls its clone's git directory, so the tests-kept check reads
// the clone's configuration only as a regular file within its bound: a copy
// that made .git/config a FIFO, a link to one, a link to an endless device or
// a file past the bound, and left a test's bytes differing from the frozen
// blob, has its tests_kept unexamined, saying why without naming where
// anything sits, and the check returns rather than waiting on the file.
func TestTestsKeptReadsTheCloneConfigurationOnlyWithinItsBound(t *testing.T) {
	for name, tc := range map[string]func(t *testing.T, config string) (fifo string){
		"a FIFO": func(t *testing.T, config string) string {
			if err := syscall.Mkfifo(config, 0o600); err != nil {
				t.Fatal(err)
			}
			return config
		},
		"a link to a FIFO": func(t *testing.T, config string) string {
			fifo := filepath.Join(t.TempDir(), "fifo")
			if err := syscall.Mkfifo(fifo, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(fifo, config); err != nil {
				t.Fatal(err)
			}
			return fifo
		},
		"a link to an endless device": func(t *testing.T, config string) string {
			if err := os.Symlink("/dev/zero", config); err != nil {
				t.Fatal(err)
			}
			return ""
		},
		"a file past the bound": func(t *testing.T, config string) string {
			big := "[core]\n\tautocrlf = true\n" + strings.Repeat("# a comment the copy wrote\n", 4<<10)
			if err := os.WriteFile(config, []byte(big), 0o600); err != nil {
				t.Fatal(err)
			}
			return ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newComplianceCase(t)
			// An honest-looking CRLF rewrite: its bytes differ from the blob,
			// so git is asked, with the clone's line-ending settings.
			p := filepath.Join(c.clone, "widget", "widget_test.go")
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(data), "\n", "\r\n")), 0o644); err != nil {
				t.Fatal(err)
			}
			config := filepath.Join(c.clone, ".git", "config")
			if err := os.Remove(config); err != nil {
				t.Fatal(err)
			}
			fifo := tc(t, config)
			type result struct {
				got Compliance
				err error
			}
			done := make(chan result, 1)
			go func() {
				got, err := checkCompliance(c.in)
				done <- result{got, err}
			}()
			var r result
			select {
			case r = <-done:
			case <-time.After(30 * time.Second):
				// Release every git that waits on the FIFO, one setting after
				// another, so none is left behind, before failing: opened for
				// reading and writing, a FIFO's open does not block, and each
				// close hands the git reading it an end of file.
				for released := false; !released && fifo != ""; {
					if w, err := os.OpenFile(fifo, os.O_RDWR, 0); err == nil {
						time.Sleep(100 * time.Millisecond)
						_ = w.Close()
					}
					select {
					case r = <-done:
						released = true
					case <-time.After(100 * time.Millisecond):
					}
				}
				if fifo == "" {
					<-done
				}
				t.Fatal("the compliance check did not return within 30s: it waited on the clone's configuration")
			}
			if r.err != nil {
				t.Fatal(r.err)
			}
			if err := r.got.validate(); err != nil {
				t.Fatal(err)
			}
			ch := ruleOf(t, r.got, RuleTestsKept)
			if ch.State != StateUnexamined {
				t.Fatalf("tests_kept: %s (%s); want unexamined", ch.State, ch.Detail)
			}
			if !strings.Contains(ch.Detail, "configuration") {
				t.Errorf("the detail does not say the configuration could not be read: %q", ch.Detail)
			}
			for _, where := range []string{c.clone, c.store, os.TempDir(), "/dev/"} {
				if strings.Contains(ch.Detail, where) {
					t.Errorf("the detail names where something sits (%s): %q", where, ch.Detail)
				}
			}
		})
	}
}

// Every git command the package runs is bounded: one that waits (here on a
// FIFO no one writes) is stopped at its bound and refused, naming the bound,
// rather than holding the check forever.
func TestAGitThatWaitsIsStoppedAtItsBound(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "config")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err := gitRawWithin(500*time.Millisecond, "config", "--file", fifo, "--get", "core.eol")
	if err == nil || !strings.Contains(err.Error(), "did not finish within 500ms") {
		t.Fatalf("a git that waits: %v; want it stopped at its bound", err)
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Fatalf("a git that waits took %s to stop; want about its bound", took)
	}
}

// The whole compliance check, every git call and the transcript scan, runs
// under one overall deadline (spc-2610161949559050 step 8): a check whose git
// waits (here on the run checkout's configuration, a FIFO no one writes) is
// stopped at that deadline, well before any one call's own bound, and the
// rules it could not finish are unexamined, naming the deadline, so the copy
// is reported apart and one hung check never stalls the run.
func TestTheComplianceCheckStopsAtItsOverallDeadline(t *testing.T) {
	c := newComplianceCase(t)
	config := filepath.Join(c.src.Root(), ".git", "config")
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(config, 0o600); err != nil {
		t.Fatal(err)
	}
	c.in.Within = 2 * time.Second
	start := time.Now()
	got, err := checkCompliance(c.in)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 30*time.Second {
		t.Fatalf("the check took %s; want it stopped at its deadline of %s", took, c.in.Within)
	}
	if err := got.validate(); err != nil {
		t.Fatalf("the stopped check's result is refused by its own record: %v", err)
	}
	for _, rule := range []ComplianceRule{RuleTestsKept, RuleOwnCopyOnly} {
		if ch := ruleOf(t, got, rule); ch.State != StateUnexamined || !strings.Contains(ch.Detail, "did not finish within 2s") {
			t.Errorf("%s = %+v; want it unexamined, naming the deadline", rule, ch)
		}
	}
	if v := got.verdict(); v != StatusUnexamined {
		t.Fatalf("the stopped check makes the copy %q; want it unexamined", v)
	}
}
