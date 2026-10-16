package cloudscripts

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// The hand-back (scripts/cloud/hand-back.sh, C6), spc-2610120941441282 step 8.
// The fixture is newCloudCase's clone with the committed pre-push hook active
// (core.hooksPath=.githooks), so every push the script makes goes through the
// same work-in-progress entry a cloud session's does. origin's URL is a plain
// github.com one, rewritten by git's own insteadOf to the bare repository, so
// the script reads a forge owner and name while every push and fetch stays on
// this disk. abcd's verbs run through a binary built from this checkout, named
// by ABCD_BIN through a wrapper that logs each call, so the outbound check
// judging a report is the real one; the fixture's `go` builds by copying that
// binary and runs it collapsing its exit code, as the real `go run` does. The
// forge's REST interface is a fake `gh` that records each call.

const (
	handBackScript = "scripts/cloud/hand-back.sh"
	forgeURL       = "https://github.com/acme/widgets.git"
	forgeRepo      = "acme/widgets"
	hostWIPPrefix  = "claude/abcd-wip/"
	handBackItem   = "iss-2610150000000011"
	sessionTrailer = "Assisted-by: Claude:claude-opus-5"
	trackingIssue  = "7"
)

func reportPath(item string) string { return ".abcd/work/cloud-reports/" + item + ".md" }

// newForgeCase is the hand-back and list fixture. tracking is cloud.conf's
// tracking_issue, committed on origin's default branch ("" leaves it empty).
// The clone ends on a branch `session`, cut from main, as a session's does.
func newForgeCase(t *testing.T, tracking string) *cloudCase {
	t.Helper()
	bin := builtAbcd.Path(t)
	c := newCloudCase(t)
	logged := filepath.Join(filepath.Dir(c.bin), "abcd-logged")
	if err := os.WriteFile(logged, []byte("#!/bin/sh\necho \"abcd $*\" >>\"$CALLS\"\nexec '"+bin+"' \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.env = append(c.env, "ABCD_BIN="+c.localAbcd(logged), "ABCD_OUTBOUND_BIN="+bin, "FAKE_ABCD="+logged)

	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))
	for _, rel := range []string{".githooks/pre-push", "scripts/preflight-receipt.sh"} {
		c.copyFrom(root, rel)
	}
	if tracking != "" {
		conf, err := os.ReadFile(filepath.Join(c.dir, "scripts", "cloud", "cloud.conf"))
		if err != nil {
			t.Fatal(err)
		}
		set := strings.Replace(string(conf), "\ntracking_issue=\n", "\ntracking_issue="+tracking+"\n", 1)
		if set == string(conf) {
			t.Fatal("scripts/cloud/cloud.conf carries no empty tracking_issue= line to set")
		}
		c.write("scripts/cloud/cloud.conf", set, 0o644)
	}
	c.git("add", "-A")
	c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
		"commit", "-q", "-m", "fixture: the hook and the cloud settings")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")

	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "alice@example.com")
	c.git("config", "url."+c.origin+".insteadOf", forgeURL)
	c.git("remote", "set-url", "origin", forgeURL)
	c.git("fetch", "-q", "origin")
	c.git("config", "core.hooksPath", ".githooks")
	c.git("checkout", "-q", "-b", "session")
	// A hand-back runs inside a cloud session; a test of the person's own
	// computer overrides the marker with its own value.
	c.env = append(c.env, remoteMarker)

	c.fake("go", goFakeForAbcd)
	c.fakeForge()
	c.resetCalls()
	return c
}

// fakeForge puts a fake `gh` on PATH. Each call appends one `gh ...` line to
// the call log, writes its argv NUL-separated to $CALLS.gh.<n>, and, given
// `--input -`, its standard input to $CALLS.gh.<n>.stdin. It prints the file
// FAKE_GH_OUT names, when set, and exits FAKE_GH (default 0).
func (c *cloudCase) fakeForge() {
	c.t.Helper()
	c.fake("gh", `echo "gh $*" >>"$CALLS"
n=0
while [ -e "$CALLS.gh.$n" ]; do n=$((n + 1)); done
: >"$CALLS.gh.$n"
for a in "$@"; do printf '%s\0' "$a" >>"$CALLS.gh.$n"; done
case " $* " in *" --input - "*) cat >"$CALLS.gh.$n.stdin" ;; esac
if [ -n "${FAKE_GH_OUT:-}" ]; then cat "$FAKE_GH_OUT"; fi
exit "${FAKE_GH:-0}"`)
}

// forgeCall is one recorded call to the fake gh.
type forgeCall struct {
	argv  []string
	stdin string
}

func (c *cloudCase) forgeCalls() []forgeCall {
	c.t.Helper()
	var calls []forgeCall
	for n := 0; ; n++ {
		p := c.calls + ".gh." + strconv.Itoa(n)
		b, err := os.ReadFile(p)
		if err != nil {
			return calls
		}
		args := strings.Split(string(b), "\x00")
		in, _ := os.ReadFile(p + ".stdin")
		calls = append(calls, forgeCall{argv: args[:len(args)-1], stdin: string(in)})
	}
}

// work is one attribution-clean commit of the session's own work.
func (c *cloudCase) work(rel string) string {
	c.t.Helper()
	c.write(rel, rel+"\n", 0o644)
	c.git("add", "--", rel)
	c.git("commit", "-q", "-m", "fix: "+rel+"\n\n"+sessionTrailer)
	return c.git("rev-parse", "HEAD")
}

// originRefuses makes the bare remote refuse every push with msg, as a forge
// proxy refusing the branch does.
func (c *cloudCase) originRefuses(msg string) {
	c.t.Helper()
	hook := filepath.Join(c.origin, "hooks", "pre-receive")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho '"+msg+"' >&2\nexit 1\n"), 0o755); err != nil {
		c.t.Fatal(err)
	}
}

// originRef is the commit origin holds for ref, or "".
func (c *cloudCase) originRef(ref string) string {
	c.t.Helper()
	out, err := c.run(c.origin, "git", "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// handBack runs the hand-back with the three texts and the flags given.
func (c *cloudCase) handBack(extra []string, args ...string) (stdout, stderr string, code int) {
	c.t.Helper()
	base := []string{"--did", "Fixed the walk over the readings folder.", "--failed", "Nothing failed.",
		"--needs", "A review of the walk."}
	return c.scriptSplit(extra, handBackScript, append(base, args...)...)
}

var finishedAtRe = regexp.MustCompile(`(?m)^finished_at: [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)

func TestHandBackCommitsReportAloneAsLastCommit(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	work := c.work("one.md")
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back", "--assisted-by", "Claude:claude-opus-5")
	if code != 0 {
		t.Fatalf("hand-back exited %d\n%s%s", code, out, errOut)
	}
	if got := c.git("rev-parse", "HEAD^"); got != work {
		t.Fatalf("the report commit's parent is %s, want the session's last work commit %s", got, work)
	}
	if got := c.git("diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != reportPath(handBackItem) {
		t.Fatalf("the last commit changes %q, want the report alone (%s)", got, reportPath(handBackItem))
	}
	msg := c.git("log", "-1", "--format=%B")
	if !strings.Contains(msg, "\nAssisted-by: Claude:claude-opus-5") {
		t.Errorf("the report commit carries no Assisted-by trailer with the value passed:\n%s", msg)
	}
	if strings.Contains(strings.ToLower(msg), "co-authored-by") {
		t.Errorf("the report commit carries a co-author trailer:\n%s", msg)
	}
	if got := c.git("log", "-1", "--format=%an <%ae>|%cn <%ce>"); got != "Alice Example <alice@example.com>|Alice Example <alice@example.com>" {
		t.Errorf("the report commit is authored as %q, want the checkout's configured identity", got)
	}
	if got := c.git("status", "--porcelain"); got != "" {
		t.Errorf("the hand-back left the tree changed:\n%s", got)
	}

	report := c.git("show", "HEAD:"+reportPath(handBackItem))
	for _, want := range []string{
		"---\nitem: " + handBackItem + "\nstate: handed-back\nchannel: branch\nfinished_at: ",
		"Fixed the walk over the readings folder.", "Nothing failed.", "A review of the walk.",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not carry %q\n%s", want, report)
		}
	}
	if !finishedAtRe.MatchString(report) {
		t.Errorf("the report's finished_at is not a UTC time to the second\n%s", report)
	}

	// Without --assisted-by the trailer is the one the session's own commits
	// carry; a session with no such commit must name it.
	c.work("two.md")
	if out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "stopped"); code != 0 {
		t.Fatalf("hand-back without --assisted-by after an assisted commit exited %d\n%s%s", code, out, errOut)
	}
	if msg := c.git("log", "-1", "--format=%B"); !strings.Contains(msg, "\n"+sessionTrailer) {
		t.Errorf("the default trailer is not the session's own:\n%s", msg)
	}
	c.git("checkout", "-q", "-b", "bare-session", "origin/main")
	head := c.git("rev-parse", "HEAD")
	_, errOut, code = c.handBack(nil, "--item", handBackItem, "--state", "stopped")
	if code == 0 || !strings.Contains(errOut, "--assisted-by") {
		t.Errorf("with no assisted commit and no --assisted-by the hand-back exited %d\n%s", code, errOut)
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("the refused hand-back committed %s", got)
	}
	if _, err := os.Stat(filepath.Join(c.dir, reportPath(handBackItem))); err == nil {
		t.Errorf("the refused hand-back wrote the report")
	}
}

// TestHandBackTakesTheTrailerFromALongHistory: the default disclosure is read
// from `git log` through a pipe. A reader that stops at the first value leaves
// git writing into a closed pipe once the history outruns the pipe's buffer,
// and under pipefail that SIGPIPE threw the value away: the merge queue saw the
// refusal on a fast runner. One commit whose disclosure lines total far more
// than any pipe buffer stands in for a long session, so the race always loses.
func TestHandBackTakesTheTrailerFromALongHistory(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	var msg strings.Builder
	msg.WriteString("fix: a long session\n\n")
	for i := 0; i < 8000; i++ {
		msg.WriteString(sessionTrailer + "\n")
	}
	c.write("long.md", "long\n", 0o644)
	c.git("add", "long.md")
	c.git("commit", "-q", "-m", msg.String())
	_, errOut, _ := c.handBack(nil, "--item", handBackItem, "--state", "handed-back")
	if strings.Contains(errOut, "REFUSING --assisted-by") {
		t.Fatalf("hand-back could not read the session's own disclosure from a long history:\n%s", errOut)
	}
	if got := c.git("log", "-1", "--format=%s"); !strings.HasPrefix(got, "chore(cloud): hand back the report on "+handBackItem) {
		t.Fatalf("the last commit is %q, want the report commit\n%s", got, errOut)
	}
	if got := c.git("log", "-1", "--format=%B"); !strings.Contains(got, "\n"+sessionTrailer) {
		t.Errorf("the report commit does not carry the session's own disclosure:\n%s", got)
	}
}

func TestHandBackRefusesAnItemThatIsNotARecordID(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	head := c.git("rev-parse", "HEAD")
	for _, item := range []string{"", "iss-26101500000000", "../iss-2610150000000011", "iss-2610150000000011/x", "main", "pr-1"} {
		_, errOut, code := c.handBack(nil, "--item", item, "--state", "handed-back")
		if code == 0 {
			t.Errorf("hand-back --item %q exited 0", item)
		}
		if !strings.Contains(errOut, "record id") {
			t.Errorf("hand-back --item %q did not say it wants a record id:\n%s", item, errOut)
		}
	}
	if _, _, code := c.handBack(nil, "--item", handBackItem, "--state", "finished"); code == 0 {
		t.Errorf("hand-back --state finished exited 0")
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("a refused hand-back committed %s", got)
	}
	if _, err := os.Stat(filepath.Join(c.dir, ".abcd", "work", "cloud-reports")); err == nil {
		t.Errorf("a refused hand-back wrote under the report folder")
	}
	if c.forgeCalls() != nil || strings.Contains(c.callLog(), "git push") {
		t.Errorf("a refused hand-back pushed or posted:\n%s", c.callLog())
	}
}

func TestHandBackPushesToWIPBranchThroughTheHook(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	// One clock reading for both hand-backs, as on a runner fast enough to
	// make them in one second: the report's finished_at then cannot tell the
	// two reports apart, and only what each says can.
	c.fake("date", "echo 2026-10-16T00:00:00Z")
	c.work("one.md")
	trace := filepath.Join(t.TempDir(), "git-trace")
	out, errOut, code := c.handBack([]string{"GIT_TRACE=" + trace}, "--item", handBackItem, "--state", "handed-back")
	if code != 0 {
		t.Fatalf("hand-back exited %d\n%s%s", code, out, errOut)
	}
	ref := "refs/heads/" + hostWIPPrefix + handBackItem
	if got, head := c.originRef(ref), c.git("rev-parse", "HEAD"); got != head {
		t.Fatalf("origin holds %q on %s, want the report commit %s\n%s%s", got, ref, head, out, errOut)
	}
	log := c.callLog()
	if !strings.Contains(log, "git push origin HEAD:"+ref) {
		t.Errorf("the hand-back did not push HEAD to %s:\n%s", ref, log)
	}
	// The hook ran: git's own trace names each hook it runs. (A hook's git
	// calls bypass the recording shim, since git puts its own directory first
	// on a hook's PATH.)
	if b, err := os.ReadFile(trace); err != nil || !regexp.MustCompile(`run_command: .*\.githooks/pre-push`).Match(b) {
		t.Errorf("the push did not run the committed pre-push hook (%v):\n%s", err, b)
	}
	for _, line := range strings.Split(log, "\n") {
		if !strings.HasPrefix(line, "git ") || strings.HasPrefix(line, "git config --get core.hooksPath") {
			continue
		}
		if strings.Contains(line, "--no-verify") || strings.Contains(line, "hooksPath") {
			t.Errorf("a git call skips or redirects the hooks: %s", line)
		}
	}

	// The hook judges the push: a work commit the attribution gate refuses
	// keeps the branch where it was.
	before := c.originRef(ref)
	c.write("two.md", "two\n", 0o644)
	c.git("add", "two.md")
	c.git("commit", "-q", "-m", "fix: two.md with no disclosure")
	_, errOut, _ = c.scriptSplit([]string{"FAKE_GH=1"}, handBackScript, "--item", handBackItem, "--state", "handed-back",
		"--did", "Added two.md.", "--failed", "Nothing failed.", "--needs", "A review of two.md.",
		"--assisted-by", "Claude:claude-opus-5")
	if got := c.originRef(ref); got != before {
		t.Errorf("a push carrying a commit the attribution gate refuses moved %s to %s", ref, got)
	}
	if !strings.Contains(errOut, "pre-push:") {
		t.Errorf("the refused push was not the hook's refusal:\n%s", errOut)
	}
}

func TestHandBackFallsBackToIssueCommentWhenPushFails(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	c.originRefuses("the proxy refuses this branch")
	// Text JSON must escape: quotes, a backslash, a tab, a non-ASCII letter.
	awkward := "Fixed the \"walk\" in C:\\cache\tand the café."
	out, errOut, code := c.scriptSplit(nil, handBackScript, "--item", handBackItem, "--state", "stopped",
		"--did", "Fixed the walk over the readings folder.\n"+awkward, "--failed", "Nothing failed.", "--needs", "A review of the walk.")
	if code != 0 {
		t.Fatalf("hand-back whose push failed and whose comment posted exited %d\n%s%s", code, out, errOut)
	}
	calls := c.forgeCalls()
	if len(calls) != 1 {
		t.Fatalf("the fake gh was called %d times, want once: %+v", len(calls), calls)
	}
	want := []string{"api", "--method", "POST", "repos/" + forgeRepo + "/issues/" + trackingIssue + "/comments", "--input", "-"}
	if strings.Join(calls[0].argv, " ") != strings.Join(want, " ") {
		t.Errorf("the REST call was %q, want %q", calls[0].argv, want)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(calls[0].stdin), &payload); err != nil {
		t.Fatalf("the comment's request body is not JSON: %v\n%s", err, calls[0].stdin)
	}
	body := payload["body"]
	for _, w := range []string{
		"item: " + handBackItem, "state: stopped", "channel: issue-comment",
		"Fixed the walk over the readings folder.", "Nothing failed.", "A review of the walk.",
		"the proxy refuses this branch", awkward,
	} {
		if !strings.Contains(body, w) {
			t.Errorf("the comment does not carry %q\n%s", w, body)
		}
	}
	if len(payload) != 1 {
		t.Errorf("the request body carries more than the comment: %v", payload)
	}
	if n := strings.Count(c.callLog(), "abcd lint outbound --label report"); n != 2 {
		t.Errorf("the outbound check ran %d times, want twice (the report, then the comment):\n%s", n, c.callLog())
	}
	if !strings.Contains(out+errOut, "#"+trackingIssue) {
		t.Errorf("the hand-back does not say where the report went:\n%s%s", out, errOut)
	}
}

// handBackSessionID is a synthetic opaque session id, built at run time
// (secret-shaped-fixtures-at-runtime), for a report the outbound check refuses.
var handBackSessionID = testsecret.Synthetic(62, 22)

func TestHandBackRefusesReportFailingOutbound(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	head := c.git("rev-parse", "HEAD")
	link := "https://agent-host.dev/code/session_" + handBackSessionID
	out, errOut, code := c.scriptSplit(nil, handBackScript, "--item", handBackItem, "--state", "handed-back",
		"--did", "Fixed it; see "+link, "--failed", "Nothing.", "--needs", "Nothing.")
	if code != 1 {
		t.Fatalf("a report carrying a session link: exit %d, want 1\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out+errOut, "breaks the outbound policy") {
		t.Errorf("the refusal does not print the outbound finding:\n%s%s", out, errOut)
	}
	if strings.Contains(out+errOut, handBackSessionID) {
		t.Errorf("the refusal repeats the session link it refused:\n%s%s", out, errOut)
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("a refused report was committed as %s", got)
	}
	if _, err := os.Stat(filepath.Join(c.dir, reportPath(handBackItem))); err == nil {
		t.Errorf("a refused report was left in the tree")
	}
	if c.originRef("refs/heads/"+hostWIPPrefix+handBackItem) != "" || c.forgeCalls() != nil {
		t.Errorf("a refused report was sent:\n%s", c.callLog())
	}
}

func TestHandBackReportNamesItsChannel(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	if out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back"); code != 0 {
		t.Fatalf("hand-back exited %d\n%s%s", code, out, errOut)
	}
	onBranch, err := c.run(c.origin, "git", "show", "refs/heads/"+hostWIPPrefix+handBackItem+":"+reportPath(handBackItem))
	if err != nil || !strings.Contains(onBranch, "\nchannel: branch\n") {
		t.Errorf("the pushed report does not name the branch as its channel: %v\n%s", err, onBranch)
	}

	other := "iss-2610150000000012"
	c.originRefuses("refused")
	if out, errOut, code := c.handBack(nil, "--item", other, "--state", "stopped"); code != 0 {
		t.Fatalf("hand-back with a failing push exited %d\n%s%s", code, out, errOut)
	}
	calls := c.forgeCalls()
	if len(calls) != 1 {
		t.Fatalf("the fake gh was called %d times, want once", len(calls))
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(calls[0].stdin), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload["body"], "\nchannel: issue-comment\n") || strings.Contains(payload["body"], "channel: branch") {
		t.Errorf("the comment does not name the issue comment as its channel\n%s", payload["body"])
	}
}

func TestHandBackPrintsReportWhenBothFail(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	c.originRefuses("the proxy refuses this branch")
	out, errOut, code := c.handBack([]string{"FAKE_GH=1"}, "--item", handBackItem, "--state", "handed-back")
	if code != 1 {
		t.Fatalf("hand-back whose push and comment both failed exited %d, want 1\n%s%s", code, out, errOut)
	}
	for _, w := range []string{"item: " + handBackItem, "Fixed the walk over the readings folder.", "Nothing failed.",
		"A review of the walk.", "the proxy refuses this branch"} {
		if !strings.Contains(out, w) {
			t.Errorf("the printed report does not carry %q\nstdout:\n%s\nstderr:\n%s", w, out, errOut)
		}
	}
	if len(c.forgeCalls()) != 1 {
		t.Errorf("the comment was tried %d times, want once", len(c.forgeCalls()))
	}
}

func TestHandBackRefusesWithoutTrackingIssue(t *testing.T) {
	c := newForgeCase(t, "")
	c.work("one.md")
	c.originRefuses("the proxy refuses this branch")
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back")
	if code != 1 {
		t.Fatalf("hand-back with no tracking issue and a failing push exited %d, want 1\n%s%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "no tracking issue configured") {
		t.Errorf("the fallback did not refuse by name:\n%s", errOut)
	}
	if !strings.Contains(out, "item: "+handBackItem) || !strings.Contains(out, "Fixed the walk over the readings folder.") {
		t.Errorf("the report was not printed:\n%s", out)
	}
	if calls := c.forgeCalls(); calls != nil {
		t.Errorf("with no tracking issue the hand-back still called the forge: %+v", calls)
	}
}

// TestHandBackOffCloudCommitsAndPushesNothing: the hand-back's first act is
// the connector's is_remote check. A brief started on the person's own
// computer must never commit onto their branch or push their unchecked
// commits to the public work-in-progress branch: off-cloud the hand-back
// prints the report, commits nothing, pushes and posts nothing, and exits 1.
func TestHandBackOffCloudCommitsAndPushesNothing(t *testing.T) {
	for _, marker := range []string{"CLAUDE_CODE_REMOTE=", "CLAUDE_CODE_REMOTE=false"} {
		t.Run(marker, func(t *testing.T) {
			c := newForgeCase(t, trackingIssue)
			c.work("one.md")
			head := c.git("rev-parse", "HEAD")
			out, errOut, code := c.handBack([]string{marker}, "--item", handBackItem, "--state", "stopped",
				"--assisted-by", "Claude:claude-opus-5")
			if code != 1 {
				t.Fatalf("an off-cloud hand-back exited %d, want 1\n%s%s", code, out, errOut)
			}
			for _, w := range []string{"item: " + handBackItem, "Fixed the walk over the readings folder.", "Nothing failed.", "A review of the walk."} {
				if !strings.Contains(out, w) {
					t.Errorf("the off-cloud hand-back did not print the report's %q\nstdout:\n%s\nstderr:\n%s", w, out, errOut)
				}
			}
			if !strings.Contains(errOut, "not running in the environment the brief was written for") {
				t.Errorf("the off-cloud hand-back does not say why it stopped\n%s", errOut)
			}
			if got := c.git("rev-parse", "HEAD"); got != head {
				t.Errorf("the off-cloud hand-back committed %s", got)
			}
			if got := c.git("status", "--porcelain"); got != "" {
				t.Errorf("the off-cloud hand-back changed the tree:\n%s", got)
			}
			if c.originRef("refs/heads/"+hostWIPPrefix+handBackItem) != "" {
				t.Errorf("the off-cloud hand-back pushed the work-in-progress branch")
			}
			log := c.callLog()
			for _, line := range strings.Split(log, "\n") {
				if strings.HasPrefix(line, "git ") && (strings.Contains(" "+line+" ", " push ") ||
					strings.Contains(" "+line+" ", " commit ") || strings.Contains(" "+line+" ", " add ")) {
					t.Errorf("the off-cloud hand-back ran %s", line)
				}
			}
			if c.forgeCalls() != nil {
				t.Errorf("the off-cloud hand-back posted to the forge: %+v", c.forgeCalls())
			}
		})
	}
}

// TestHandBackNeverShowsARefusedIdentity (the vendor identity is
// cloud_setup_test.go's vendorName and vendorMail): the product thinker ruled that a
// refused identity is shown nowhere. The hand-back judges the checkout's
// identity before it commits, and never puts raw git or hook output in what it
// prints or posts: neither standard output, standard error nor the comment on
// the tracking issue carries the vendor's name or address, whether the commit
// is what is blocked (the checkout's identity is the vendor's) or the push (a
// work commit carries it, and the hook's attribution gate prints it).
func TestHandBackNeverShowsARefusedIdentity(t *testing.T) {
	check := func(t *testing.T, c *cloudCase, out, errOut string) {
		t.Helper()
		calls := c.forgeCalls()
		if len(calls) != 1 {
			t.Fatalf("the fallback called the forge %d times, want once\nstdout:\n%s\nstderr:\n%s", len(calls), out, errOut)
		}
		var payload map[string]string
		if err := json.Unmarshal([]byte(calls[0].stdin), &payload); err != nil {
			t.Fatalf("the comment's request body is not JSON: %v\n%s", err, calls[0].stdin)
		}
		for where, text := range map[string]string{"stdout": out, "stderr": errOut, "the comment": payload["body"]} {
			for _, s := range []string{vendorName, vendorMail, vendorMail[strings.Index(vendorMail, "@")+1:]} {
				if strings.Contains(text, s) {
					t.Errorf("%s shows the refused identity's %q:\n%s", where, s, text)
				}
			}
		}
	}

	t.Run("the commit is blocked", func(t *testing.T) {
		c := newForgeCase(t, trackingIssue)
		c.work("one.md")
		head := c.git("rev-parse", "HEAD")
		c.git("config", "user.name", vendorName)
		c.git("config", "user.email", vendorMail)
		out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "stopped", "--assisted-by", "Claude:claude-opus-5")
		if code != 0 {
			t.Fatalf("hand-back whose comment posted exited %d\n%s%s", code, out, errOut)
		}
		check(t, c, out, errOut)
		if got := c.git("rev-parse", "HEAD"); got != head {
			t.Errorf("the hand-back committed %s under an identity that is not a person's", got)
		}
		if got := c.git("status", "--porcelain"); got != "" {
			t.Errorf("the hand-back left the tree changed:\n%s", got)
		}
		if strings.Contains(c.callLog(), "git push") {
			t.Errorf("the hand-back pushed under an identity that is not a person's:\n%s", c.callLog())
		}
		var payload map[string]string
		_ = json.Unmarshal([]byte(c.forgeCalls()[0].stdin), &payload)
		if !strings.Contains(payload["body"], "no person's identity is set in this checkout, so nothing was committed") {
			t.Errorf("the comment does not give the fixed reason:\n%s", payload["body"])
		}
	})

	t.Run("the push is blocked", func(t *testing.T) {
		c := newForgeCase(t, trackingIssue)
		c.write("one.md", "one\n", 0o644)
		c.git("add", "one.md")
		c.git("-c", "user.name="+vendorName, "-c", "user.email="+vendorMail, "commit", "-q", "-m", "fix: one.md\n\n"+sessionTrailer)
		out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back", "--assisted-by", "Claude:claude-opus-5")
		if code != 0 {
			t.Fatalf("hand-back whose comment posted exited %d\n%s%s", code, out, errOut)
		}
		if c.originRef("refs/heads/"+hostWIPPrefix+handBackItem) != "" {
			t.Fatalf("the push of a vendor-authored commit went ahead; the scenario needs the hook to refuse it")
		}
		check(t, c, out, errOut)
	})
}

// TestHandBackUnstagesReportWhenCommitFails: a commit hook that refuses the
// report commit leaves nothing staged, and its output is not repeated.
func TestHandBackUnstagesReportWhenCommitFails(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	head := c.git("rev-parse", "HEAD")
	c.write(".githooks/pre-commit", "#!/bin/sh\necho 'hook says: secret-hook-text' >&2\nexit 1\n", 0o755)
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "stopped", "--assisted-by", "Claude:claude-opus-5")
	if code != 0 {
		t.Fatalf("hand-back whose comment posted exited %d\n%s%s", code, out, errOut)
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("the refused commit moved HEAD to %s", got)
	}
	if got := c.git("diff", "--cached", "--name-only"); got != "" {
		t.Errorf("the report is left staged after the commit failed: %s", got)
	}
	if strings.Contains(out+errOut, "secret-hook-text") {
		t.Errorf("the hand-back repeated the commit hook's raw output:\n%s%s", out, errOut)
	}
	if !strings.Contains(errOut, "could not be committed") {
		t.Errorf("the hand-back does not say the commit failed:\n%s", errOut)
	}
}

// fakeOutbound writes a stand-in abcd whose lint outbound exits code, and
// returns its path.
func (c *cloudCase) fakeOutbound(code string) string {
	c.t.Helper()
	p := filepath.Join(filepath.Dir(c.bin), "abcd-outbound-"+code)
	body := "#!/bin/sh\necho \"abcd $*\" >>\"$CALLS\"\necho 'the check could not read its input' >&2\nexit " + code + "\n"
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		c.t.Fatal(err)
	}
	return p
}

// TestHandBackReadsTheOutboundChecksOwnExitCode: lint outbound exits 1 for a
// refused report and 2 for a check that could not run, and the hand-back acts
// differently on each. Through `go run` both read as 1.
func TestHandBackReadsTheOutboundChecksOwnExitCode(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	head := c.git("rev-parse", "HEAD")
	out, errOut, code := c.handBack([]string{"ABCD_BIN=" + c.localAbcd(c.fakeOutbound("2")), "FAKE_ABCD=" + c.fakeOutbound("2")},
		"--item", handBackItem, "--state", "handed-back", "--assisted-by", "Claude:claude-opus-5")
	if code != 1 {
		t.Fatalf("an outbound check that could not run: exit %d, want 1\n%s%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "could not run (exit 2)") || strings.Contains(errOut, "fails the outbound check") {
		t.Errorf("the hand-back did not read the check's own exit code, 2:\n%s", errOut)
	}
	if !strings.Contains(out, "item: "+handBackItem) {
		t.Errorf("a report the check could not judge is not printed for the transcript:\n%s", out)
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("a report the check could not judge was committed as %s", got)
	}
	if regexp.MustCompile(`(?m)^go `).MatchString(c.callLog()) {
		t.Errorf("with ABCD_BIN set the hand-back ran go:\n%s", c.callLog())
	}
}

// TestHandBackBuildsAbcdOnceWithoutABCDBIN: with no ABCD_BIN the hand-back
// builds abcd from this checkout once, and both outbound checks (the report,
// then the comment when the push fails) use that one binary.
func TestHandBackBuildsAbcdOnceWithoutABCDBIN(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	c.originRefuses("the proxy refuses this branch")
	out, errOut, code := c.handBack([]string{"ABCD_BIN="}, "--item", handBackItem, "--state", "handed-back")
	if code != 0 {
		t.Fatalf("hand-back exited %d\n%s%s", code, out, errOut)
	}
	log := c.callLog()
	if n := strings.Count(log, "go build -o "); n != 1 {
		t.Errorf("abcd was built %d times, want once per run:\n%s", n, log)
	}
	if strings.Contains(log, "go run") {
		t.Errorf("the hand-back still runs abcd through go run:\n%s", log)
	}
	if n := strings.Count(log, "abcd lint outbound --label report"); n != 2 {
		t.Errorf("the outbound check ran %d times, want twice:\n%s", n, log)
	}
	if len(c.forgeCalls()) != 1 {
		t.Errorf("the comment was posted %d times, want once", len(c.forgeCalls()))
	}
}

// TestHandBackScrubLeavesNoDomain: what a failed push's output may show has
// every identity replaced, addresses first and the longest string first. A
// name that is the start of the checkout's address (pat, pat@example.org)
// replaced first would leave "<identity>@example.org", the domain shown.
func TestHandBackScrubLeavesNoDomain(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.git("config", "user.name", "pat")
	c.git("config", "user.email", "pat@example.org")
	c.work("one.md")
	c.originRefuses("the proxy refuses this push by pat@example.org")
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back")
	if code != 0 {
		t.Fatalf("hand-back whose comment posted exited %d\n%s%s", code, out, errOut)
	}
	calls := c.forgeCalls()
	if len(calls) != 1 {
		t.Fatalf("the fallback called the forge %d times, want once\n%s", len(calls), errOut)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(calls[0].stdin), &payload); err != nil {
		t.Fatalf("the comment's request body is not JSON: %v\n%s", err, calls[0].stdin)
	}
	if !strings.Contains(payload["body"], "remote:") {
		t.Fatalf("the comment does not carry the remote's line, so the scrub is untested:\n%s", payload["body"])
	}
	for where, text := range map[string]string{"stdout": out, "stderr": errOut, "the comment": payload["body"]} {
		for _, s := range []string{"example.org", "pat@"} {
			if strings.Contains(text, s) {
				t.Errorf("%s shows %q of the identity:\n%s", where, s, text)
			}
		}
	}
}

// TestHandBackPrintsReportWhenConnectorCannotBeResolved: with no connector to
// say where it runs (two declared, so none is the default), the hand-back
// cannot tell the cloud from the person's computer. It commits, pushes and
// posts nothing, and still prints the report, so the transcript holds it.
func TestHandBackPrintsReportWhenConnectorCannotBeResolved(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	c.declare("other", "name=other\nwip_prefix=other/abcd-wip/\nis_remote=is-remote.sh\n")
	head := c.git("rev-parse", "HEAD")
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "stopped", "--assisted-by", "Claude:claude-opus-5")
	if code != 1 {
		t.Fatalf("a hand-back with no resolvable connector exited %d, want 1\n%s%s", code, out, errOut)
	}
	for _, w := range []string{"item: " + handBackItem, "Fixed the walk over the readings folder.", "Nothing failed.", "A review of the walk."} {
		if !strings.Contains(out, w) {
			t.Errorf("the report's %q was not printed\nstdout:\n%s\nstderr:\n%s", w, out, errOut)
		}
	}
	if !strings.Contains(errOut, "connector") {
		t.Errorf("the hand-back does not say the connector could not be resolved\n%s", errOut)
	}
	if got := c.git("rev-parse", "HEAD"); got != head {
		t.Errorf("the hand-back committed %s", got)
	}
	for _, line := range strings.Split(c.callLog(), "\n") {
		if strings.HasPrefix(line, "git ") && (strings.Contains(" "+line+" ", " push ") || strings.Contains(" "+line+" ", " commit ")) {
			t.Errorf("the hand-back ran %s", line)
		}
	}
	if c.forgeCalls() != nil {
		t.Errorf("the hand-back posted to the forge: %+v", c.forgeCalls())
	}
}

// TestHandBackFallbackRefersToNothingDropped: the fallback shows only the
// hook's own pre-push: lines and the remote's, and drops every other line, the
// connector loader's diagnostic among them. So a kept line never points at a
// diagnostic "above": the comment would point at nothing. The declaration the
// hook reads is the one HEAD commits, so a bad one committed and a good one in
// the working tree make the hand-back resolve its connector and the hook
// refuse the push.
func TestHandBackFallbackRefersToNothingDropped(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	conf := "scripts/cloud/connectors/claude/connector.conf"
	good, err := os.ReadFile(filepath.Join(c.dir, filepath.FromSlash(conf)))
	if err != nil {
		t.Fatal(err)
	}
	c.write(conf, string(good)+"exempt=all\n", 0o644)
	c.git("add", "--", conf)
	c.git("commit", "-q", "-m", "chore: a declaration\n\n"+sessionTrailer)
	c.write(conf, string(good), 0o644)
	out, errOut, code := c.handBack(nil, "--item", handBackItem, "--state", "handed-back")
	if code != 0 {
		t.Fatalf("hand-back whose comment posted exited %d\n%s%s", code, out, errOut)
	}
	calls := c.forgeCalls()
	if len(calls) != 1 {
		t.Fatalf("the fallback called the forge %d times, want once\n%s", len(calls), errOut)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(calls[0].stdin), &payload); err != nil {
		t.Fatalf("the comment's request body is not JSON: %v\n%s", err, calls[0].stdin)
	}
	if !strings.Contains(payload["body"], "pre-push:") {
		t.Fatalf("the comment carries no line of the hook's, so the scenario is untested:\n%s", payload["body"])
	}
	for where, text := range map[string]string{"stderr": errOut, "the comment": payload["body"]} {
		if strings.Contains(text, "above") {
			t.Errorf("%s points at a diagnostic \"above\" that the fallback dropped:\n%s", where, text)
		}
	}
}
