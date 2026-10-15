package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Starting a session through the connector (scripts/cloud/send.sh and the
// first host's launch.sh, C10 and C5), spc-2610120941441282 step 9. Every
// test runs against a FAKE host command-line tool first on PATH, which records
// its argv and serves fixture output; the real one is never reached.

const (
	sendScript      = "scripts/cloud/send.sh"
	environmentFile = ".abcd/.work.local/cloud/environment"
	fixtureEnv      = "env_fixture123"
	fixtureSession  = "session_01FixtureAbCdEf2345"
	fixtureLink     = "https://example.invalid/code/" + fixtureSession
)

func briefPreview(id string) string { return ".abcd/.work.local/cloud/briefs/" + id + ".md" }

// fakeHost puts the fake host tool on PATH. FAKE_HOST picks its answer:
//
//	ok       the start succeeds: JSON naming the session id and its link
//	noid     it exits 0 with JSON that names no session
//	nologin  no login
//	expired  the login has expired
//	badenv   the environment is unknown
//	refuse   any other refusal
//
// Each call appends one `host-call` line to the call log and writes its argv,
// NUL-separated, to $CALLS.host.<n>.
func (c *cloudCase) fakeHost() {
	c.t.Helper()
	c.fake("claude", `echo "host-call" >>"$CALLS"
n=0
while [ -e "$CALLS.host.$n" ]; do n=$((n + 1)); done
: >"$CALLS.host.$n"
for a in "$@"; do printf '%s\0' "$a" >>"$CALLS.host.$n"; done
case "${FAKE_HOST:-ok}" in
ok) printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"`+fixtureSession+`","result":"Started `+fixtureLink+`"}' ;;
noid) printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"result":"done"}' ;;
nologin) echo 'Invalid API key · Please run /login' >&2; exit 1 ;;
expired) echo 'OAuth token has expired. Please obtain a new token or refresh your existing token.' >&2; exit 1 ;;
badenv) echo 'Error: environment `+fixtureEnv+` not found' >&2; exit 1 ;;
refuse) echo 'Error: the service is unavailable' >&2; exit 1 ;;
esac`)
}

// hostArgv is the argv of each recorded call to the fake host tool.
func (c *cloudCase) hostArgv() [][]string {
	c.t.Helper()
	var calls [][]string
	for n := 0; ; n++ {
		b, err := os.ReadFile(c.calls + ".host." + strconv.Itoa(n))
		if err != nil {
			return calls
		}
		args := strings.Split(string(b), "\x00")
		calls = append(calls, args[:len(args)-1])
	}
}

func newSendCase(t *testing.T) *cloudCase {
	t.Helper()
	c := newBriefCase(t)
	c.fakeHost()
	c.write(environmentFile, fixtureEnv+"\n", 0o644)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.resetCalls()
	return c
}

// workTreeStamps is treeStamps over the working tree, leaving out .git, whose
// remote-tracking refs the brief's fetch may move.
func workTreeStamps(t *testing.T, c *cloudCase) map[string]fileStamp {
	t.Helper()
	all := treeStamps(t, c.dir)
	gitDir := filepath.Join(c.dir, ".git")
	for p := range all {
		if p == gitDir || strings.HasPrefix(p, gitDir+string(os.PathSeparator)) {
			delete(all, p)
		}
	}
	return all
}

func TestSendStartsOneSessionWithTheBrief(t *testing.T) {
	c := newSendCase(t)
	wantBrief, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("the brief itself is refused (exit %d)\n%s", code, errOut)
	}
	wantBrief = strings.TrimRight(wantBrief, "\n")
	c.resetCalls()

	out, code := c.script(nil, sendScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("send exited %d\n%s", code, out)
	}
	calls := c.hostArgv()
	if len(calls) != 1 || strings.Count(c.callLog(), "host-call") != 1 {
		t.Fatalf("send called the host %d times, want exactly once\n%s", len(calls), c.callLog())
	}
	want := []string{"-p", wantBrief, "--environment", fixtureEnv, "--ref", "main", "--output-format", "json"}
	got := calls[0]
	if len(got) != len(want) {
		t.Fatalf("the host was started with %d arguments, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argument %d = %q, want %q", i, got[i], want[i])
		}
	}
	preview, err := os.ReadFile(filepath.Join(c.dir, filepath.FromSlash(briefPreview(fixtureIssue))))
	if err != nil {
		t.Fatalf("send kept no preview of the brief it sent: %v", err)
	}
	if strings.TrimRight(string(preview), "\n") != wantBrief {
		t.Errorf("the preview is not the brief that was sent\n%s", preview)
	}
}

func TestSendPrintsIdAndLink(t *testing.T) {
	c := newSendCase(t)
	out, code := c.script(nil, sendScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("send exited %d\n%s", code, out)
	}
	if !strings.Contains(out, fixtureSession) || !strings.Contains(out, fixtureLink) {
		t.Errorf("send does not report the session's id and link\n%s", out)
	}
	// The link is printed to the person and written nowhere.
	if err := filepath.WalkDir(c.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), fixtureSession) {
			t.Errorf("the session's id or link was written to %s", p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// A start the host reports with no session to name is not a success:
	// the person is told to look before sending again.
	out, code = c.script([]string{"FAKE_HOST=noid"}, sendScript, fixtureIssue)
	if code != 1 || !strings.Contains(out, "may have started") {
		t.Errorf("a start naming no session exited %d, want 1 saying a session may have started\n%s", code, out)
	}
}

func TestSendRefusesWhenHostToolMissing(t *testing.T) {
	c := newSendCase(t)
	if err := os.Remove(filepath.Join(c.bin, "claude")); err != nil {
		t.Fatal(err)
	}
	// The system directories alone after the fakes: the host tool a person
	// installs lives elsewhere, and this test must not find a real one.
	for _, d := range []string{"/usr/bin", "/bin"} {
		if _, err := os.Stat(filepath.Join(d, "claude")); err == nil {
			t.Skipf("a host tool is installed in %s", d)
		}
	}
	out, code := c.script([]string{"PATH=" + c.bin + ":/usr/bin:/bin"}, sendScript, fixtureIssue)
	if code != 1 {
		t.Fatalf("send with no host tool exited %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"cannot be reached", "not installed", "Nothing was started"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q\n%s", want, out)
		}
	}
}

func TestSendRefusesWithoutLogin(t *testing.T) {
	c := newSendCase(t)
	out, code := c.script([]string{"FAKE_HOST=nologin"}, sendScript, fixtureIssue)
	if code != 1 {
		t.Fatalf("send with no login exited %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"cannot be reached", "not logged in", "Nothing was started"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q\n%s", want, out)
		}
	}
	if strings.Contains(out, fixtureSession) {
		t.Errorf("a refused start reported a session\n%s", out)
	}
}

func TestSendExpiredLoginSaysRenewByHand(t *testing.T) {
	c := newSendCase(t)
	out, code := c.script([]string{"FAKE_HOST=expired"}, sendScript, fixtureIssue)
	if code != 1 {
		t.Fatalf("send with an expired login exited %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"cannot be reached", "expired", "renew", "by hand", "Nothing was started"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q\n%s", want, out)
		}
	}
	if n := strings.Count(c.callLog(), "host-call"); n != 1 {
		t.Errorf("send tried the host %d times; an expired login stops the run at once", n)
	}
}

func TestSendRefusesUnknownEnvironment(t *testing.T) {
	c := newSendCase(t)
	out, code := c.script([]string{"FAKE_HOST=badenv"}, sendScript, fixtureIssue)
	if code != 1 {
		t.Fatalf("send to an unknown environment exited %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"cannot be reached", "environment", environmentFile, "Nothing was started"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q\n%s", want, out)
		}
	}

	// No environment id, or one that is no id, stops before the host is asked
	// and before any brief is written.
	for _, tc := range []struct{ name, content string }{
		{"no environment file", ""},
		{"an empty environment file", "# only a comment\n\n"},
		{"an environment id that is no id", "env one; rm -rf /\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newSendCase(t)
			p := filepath.Join(c.dir, filepath.FromSlash(environmentFile))
			if tc.content == "" {
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
			} else {
				c.write(environmentFile, tc.content, 0o644)
			}
			out, code := c.script(nil, sendScript, fixtureIssue)
			if code != 1 || !strings.Contains(out, environmentFile) {
				t.Errorf("%s: send exited %d, want a refusal naming %s\n%s", tc.name, code, environmentFile, out)
			}
			if strings.Contains(c.callLog(), "host-call") {
				t.Errorf("%s: the host was asked anyway", tc.name)
			}
			if _, err := os.Stat(filepath.Join(c.dir, filepath.FromSlash(briefPreview(fixtureIssue)))); err == nil {
				t.Errorf("%s: a brief preview was written", tc.name)
			}
		})
	}
}

func TestSendHostRefusalStartsNothingLocal(t *testing.T) {
	for _, mode := range []string{"nologin", "expired", "badenv", "refuse", "missing"} {
		t.Run(mode, func(t *testing.T) {
			c := newSendCase(t)
			env := []string{"FAKE_HOST=" + mode}
			if mode == "missing" {
				for _, d := range []string{"/usr/bin", "/bin"} {
					if _, err := os.Stat(filepath.Join(d, "claude")); err == nil {
						t.Skipf("a host tool is installed in %s", d)
					}
				}
				if err := os.Remove(filepath.Join(c.bin, "claude")); err != nil {
					t.Fatal(err)
				}
				env = append(env, "PATH="+c.bin+":/usr/bin:/bin")
			}
			worktreesBefore := c.git("worktree", "list", "--porcelain")
			before := workTreeStamps(t, c)
			storeBefore := treeStamps(t, c.home())
			c.resetCalls()

			out, code := c.script(env, sendScript, fixtureIssue)
			if code != 1 || !strings.Contains(out, "Nothing was started") {
				t.Fatalf("a refused start exited %d, want 1 saying nothing was started\n%s", code, out)
			}
			log := c.callLog()

			after := workTreeStamps(t, c)
			preview := filepath.Join(c.dir, filepath.FromSlash(briefPreview(fixtureIssue)))
			for p, s := range after {
				if b, ok := before[p]; ok {
					if b != s && !strings.HasPrefix(preview, p) {
						t.Errorf("a refused start changed %s", p)
					}
					continue
				}
				// The preview and the folders that hold it are the one thing
				// a refused start may leave.
				if p != preview && !strings.HasPrefix(preview, p+string(os.PathSeparator)) {
					t.Errorf("a refused start created %s", p)
				}
			}
			for p := range before {
				if _, ok := after[p]; !ok {
					t.Errorf("a refused start removed %s", p)
				}
			}
			if got := c.git("worktree", "list", "--porcelain"); got != worktreesBefore {
				t.Errorf("a refused start changed the worktrees:\n%s", got)
			}
			for p := range treeStamps(t, c.home()) {
				if _, ok := storeBefore[p]; !ok {
					t.Errorf("a refused start created %s under the home folder", p)
				}
			}
			// Nothing ran after the host refused: its call is the last thing
			// send did, and no worktree or local session was reached for.
			lines := strings.Split(strings.TrimSpace(log), "\n")
			if mode != "missing" && lines[len(lines)-1] != "host-call" {
				t.Errorf("send went on after the host refused:\n%s", log)
			}
			if strings.Contains(log, "worktree") {
				t.Errorf("send touched a worktree:\n%s", log)
			}
		})
	}
}

func TestSendRefusesPrivateOrUnreadyItem(t *testing.T) {
	t.Run("not public", func(t *testing.T) {
		c := newSendCase(t)
		c.commitRecord(issueRecord("iss-2610150000000009"), false)
		c.resetCalls()
		out, code := c.script(nil, sendScript, "iss-2610150000000009")
		if code != 1 || !strings.Contains(out, "not public") {
			t.Errorf("send of an unpushed item exited %d, want a refusal saying it is not public\n%s", code, out)
		}
		if strings.Contains(c.callLog(), "host-call") {
			t.Error("the host was asked to start a session for an item that is not public")
		}
		if _, err := os.Stat(filepath.Join(c.dir, filepath.FromSlash(briefPreview("iss-2610150000000009")))); err == nil {
			t.Error("a brief preview was written for an item that is not public")
		}
	})
	t.Run("not ready", func(t *testing.T) {
		c := newSendCase(t)
		c.commitRecord(intentRecord("planned", fixtureIntent), true)
		c.resetCalls()
		out, code := c.script([]string{"FAKE_READY=1"}, sendScript, fixtureIntent)
		if code != 1 || !strings.Contains(out, "not ready") {
			t.Errorf("send of an unready intent exited %d, want a refusal saying it is not ready\n%s", code, out)
		}
		if strings.Contains(c.callLog(), "host-call") {
			t.Error("the host was asked to start a session for an item that is not ready")
		}
	})
	t.Run("under embargo", func(t *testing.T) {
		c := newSendCase(t)
		c.embargoStore()
		out, code := c.script([]string{"FAKE_EMBARGO=2"}, sendScript, fixtureIssue)
		if code != 1 || !strings.Contains(out, "embargo") {
			t.Errorf("send of an embargoed item exited %d, want a refusal naming the embargo\n%s", code, out)
		}
		if strings.Contains(c.callLog(), "host-call") {
			t.Error("the host was asked to start a session for an item under embargo")
		}
	})
}
