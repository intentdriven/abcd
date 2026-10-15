package cli

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
)

// The hand-back (scripts/cloud/hand-back.sh, C6), spc-2610120941441282 step 8.
// The fixture is newCloudCase's clone with the committed pre-push hook active
// (core.hooksPath=.githooks), so every push the script makes goes through the
// same work-in-progress entry a cloud session's does. origin's URL is a plain
// github.com one, rewritten by git's own insteadOf to the bare repository, so
// the script reads a forge owner and name while every push and fetch stays on
// this disk. abcd's verbs run through a binary built from this checkout (the
// fake `go` execs it), so the outbound check judging a report is the real one;
// the forge's REST interface is a fake `gh` that records each call.

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
	bin := outboundChecker(t)
	c := newCloudCase(t)
	c.env = append(c.env, "ABCD_BIN="+bin, "ABCD_OUTBOUND_BIN="+bin)

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

	c.fake("go", `echo "go $*" >>"$CALLS"
if [ "${1:-}" = run ] && [ "${2:-}" = ./cmd/abcd ]; then shift 2; exec "$ABCD_BIN" "$@"; fi
echo "fake go: unexpected $*" >&2
exit 3`)
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
	c.work("one.md")
	trace := filepath.Join(t.TempDir(), "git-trace")
	out, errOut, code := c.handBack([]string{"GIT_TRACE=" + trace}, "--item", handBackItem, "--state", "handed-back")
	if code != 0 {
		t.Fatalf("hand-back exited %d\n%s%s", code, out, errOut)
	}
	ref := "refs/heads/" + hostWIPPrefix + handBackItem
	if got, head := c.originRef(ref), c.git("rev-parse", "HEAD"); got != head {
		t.Fatalf("origin holds %q on %s, want the report commit %s", got, ref, head)
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
	_, errOut, _ = c.handBack([]string{"FAKE_GH=1"}, "--item", handBackItem, "--state", "handed-back",
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
	if !strings.Contains(c.callLog(), "go run ./cmd/abcd lint outbound --label report") {
		t.Errorf("the comment was not passed through the outbound check:\n%s", c.callLog())
	}
	if !strings.Contains(out+errOut, "#"+trackingIssue) {
		t.Errorf("the hand-back does not say where the report went:\n%s%s", out, errOut)
	}
}

func TestHandBackRefusesReportFailingOutbound(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.work("one.md")
	head := c.git("rev-parse", "HEAD")
	link := "https://agent-host.dev/code/session_" + testOutboundSessionID
	out, errOut, code := c.scriptSplit(nil, handBackScript, "--item", handBackItem, "--state", "handed-back",
		"--did", "Fixed it; see "+link, "--failed", "Nothing.", "--needs", "Nothing.")
	if code != 1 {
		t.Fatalf("a report carrying a session link: exit %d, want 1\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out+errOut, "breaks the outbound policy") {
		t.Errorf("the refusal does not print the outbound finding:\n%s%s", out, errOut)
	}
	if strings.Contains(out+errOut, testOutboundSessionID) {
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
