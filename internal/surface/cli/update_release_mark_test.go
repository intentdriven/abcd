package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/update"
)

// iss-2609151847117977 at `abcd update`: the verb records the newest-release
// mark whenever it installs or verifies a release, refuses a bare update whose
// latest release has fallen back below the mark, and lowers the mark only when
// the person names the older tag themselves.

// latestUpdater resolves a bare update to a fixed latest tag and records
// whether the swap ran. With refusal set, Apply refuses with it instead of
// swapping, as the core does for a release it will not take.
type latestUpdater struct {
	latest  string
	applied bool
	refusal *update.Refusal
}

func (f *latestUpdater) ResolveTag(requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	return f.latest, nil
}

func (f *latestUpdater) Apply(_, tag string, _ io.Writer) (update.Report, error) {
	if f.refusal != nil {
		return update.Report{Action: update.ActionRefused, Tag: tag, Refusal: f.refusal}, nil
	}
	f.applied = true
	return swappedReceipt(tag), nil
}

// runUpdateWithMark drives `abcd update args...` down the swap path with the
// mark seeded at mark ("" for none), returning the fake, the output and the
// exit code.
func runUpdateWithMark(t *testing.T, mark, latest string, args ...string) (*latestUpdater, string, int) {
	t.Helper()
	fake := &latestUpdater{latest: latest}
	out, code := runUpdateWithMarkFake(t, mark, fake, args...)
	return fake, out, code
}

// runUpdateWithMarkFake is runUpdateWithMark with the updater supplied.
func runUpdateWithMarkFake(t *testing.T, mark string, fake *latestUpdater, args ...string) (string, int) {
	t.Helper()
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "abcd"), []byte("\x7fELFfake"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)
	if mark != "" {
		if err := ahoy.RecordReleaseMark(mark, true); err != nil {
			t.Fatal(err)
		}
	}
	orig := newUpdater
	newUpdater = func() updater { return fake }
	t.Cleanup(func() { newUpdater = orig })
	var out bytes.Buffer
	code := Run(append([]string{"update", "--yes"}, args...), &out, &out)
	return out.String(), code
}

// TestUpdateLeavesTheMarkWhenTheReleaseIsNotItsOwnBuild: a newer tag whose
// release carries an older release's genuine build is refused by the core as
// not this release, and the refusal leaves the mark where it was, so a later
// genuine release is not refused as older than a tag that installed nothing.
func TestUpdateLeavesTheMarkWhenTheReleaseIsNotItsOwnBuild(t *testing.T) {
	for _, named := range []bool{false, true} {
		fake := &latestUpdater{latest: "v99.0.0", refusal: &update.Refusal{
			Shape:  "not-this-release",
			Detail: "its build metadata names release v0.6.1, not v99.0.0, so it is not this release's build",
		}}
		var args []string
		if named {
			args = []string{"v99.0.0"}
		}
		out, code := runUpdateWithMarkFake(t, "v0.7.0", fake, args...)
		if code == 0 || !strings.Contains(out, "not-this-release") {
			t.Fatalf("named=%v: the core's refusal was not reached or not reported (exit %d):\n%s", named, code, out)
		}
		if got, _ := ahoy.ReadReleaseMark(); got != "v0.7.0" {
			t.Errorf("named=%v: the refusal moved the mark to %q", named, got)
		}
	}
}

// TestUpdateRefusesALatestReleaseOlderThanTheMark: a bare update whose latest
// release fell back below the recorded mark (a publisher withdrawal) swaps
// nothing and names the one command that lowers the mark on purpose.
func TestUpdateRefusesALatestReleaseOlderThanTheMark(t *testing.T) {
	fake, out, code := runUpdateWithMark(t, "v0.7.0", "v0.6.2")
	if code == 0 || fake.applied {
		t.Fatalf("a latest release older than the mark was applied (exit %d):\n%s", code, out)
	}
	if !strings.Contains(out, "abcd update v0.6.2") || !strings.Contains(out, "v0.7.0") {
		t.Errorf("the refusal must name the mark and the command that lowers it:\n%s", out)
	}
	if got, _ := ahoy.ReadReleaseMark(); got != "v0.7.0" {
		t.Errorf("a refused update moved the mark to %q", got)
	}
}

// TestUpdateRecordsTheMark: a bare update raises the mark to the release it
// installed; a named older tag lowers it, on purpose.
func TestUpdateRecordsTheMark(t *testing.T) {
	t.Run("a bare update raises it", func(t *testing.T) {
		_, out, code := runUpdateWithMark(t, "v0.6.2", "v0.7.0")
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		if got, _ := ahoy.ReadReleaseMark(); got != "v0.7.0" {
			t.Errorf("mark = %q after installing v0.7.0", got)
		}
	})
	t.Run("a named older tag lowers it", func(t *testing.T) {
		fake, out, code := runUpdateWithMark(t, "v0.7.0", "v0.7.0", "v0.6.2")
		if code != 0 || !fake.applied {
			t.Fatalf("a named older tag was refused (exit %d):\n%s", code, out)
		}
		if got, _ := ahoy.ReadReleaseMark(); got != "v0.6.2" {
			t.Errorf("mark = %q after the person named v0.6.2", got)
		}
	})
	t.Run("a first update records one", func(t *testing.T) {
		if _, out, code := runUpdateWithMark(t, "", "v0.7.0"); code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		if got, _ := ahoy.ReadReleaseMark(); got != "v0.7.0" {
			t.Errorf("mark = %q after a first update to v0.7.0", got)
		}
	})
}
