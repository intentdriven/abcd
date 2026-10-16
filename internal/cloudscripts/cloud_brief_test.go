package cloudscripts

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The brief (scripts/cloud/brief.sh and brief.md.tmpl, C4 and C13),
// spc-2610120941441282 step 5. The fixture is newCloudCase's clone with a
// bare origin; every abcd verb the script calls goes through a fake abcd
// (fakeAbcd), whose answer each test decides, so only the tests that say so
// build the real one.

const (
	briefScript   = "scripts/cloud/brief.sh"
	briefTemplate = "scripts/cloud/brief.md.tmpl"
	fixtureIssue  = "iss-2610150000000001"
	fixtureIntent = "itd-2610150000000002"
)

func issueRecord(id string) string {
	return ".abcd/work/issues/open/" + id + "-a-fixture-item.md"
}

func intentRecord(bucket, id string) string {
	return ".abcd/development/intents/" + bucket + "/" + id + "-a-fixture-item.md"
}

// fakeAbcd stands in for abcd itself: a fake `abcd` binary, written at
// $FAKE_ABCD, that answers the three verbs the brief runs, each by an
// environment variable a test sets:
//
//	FAKE_READY     intent ready's exit code (default 0)
//	FAKE_EMBARGO   embargo check's exit code, or `unknown` for a binary that
//	               has no such verb (default 0)
//	FAKE_OUTBOUND  lint outbound's exit code (default 0)
//
// Each call logs `abcd <argv>`. The texts embargo check was given are kept
// beside the call log, so a test can prove which they were.
//
// The fixture's `go` gives that binary both ways a script can reach abcd:
// `go build -o <path> ./cmd/abcd` copies it to <path> (FAKE_BUILD, when set,
// is the build's exit code instead), and `go run ./cmd/abcd ...` runs it the
// way the real `go run` does, turning every non-zero exit into 1. So a script
// still branching on `go run`'s exit code reads what it would really read.
func (c *cloudCase) fakeAbcd() {
	c.t.Helper()
	fakeAbcd := filepath.Join(filepath.Dir(c.bin), "fake-abcd")
	body := `#!/bin/sh
echo "abcd $*" >>"$CALLS"
case "${1:-} ${2:-}" in
"intent ready") exit "${FAKE_READY:-0}" ;;
"embargo check")
	: >"$CALLS.embargo-text"
	prev=""
	for a in "$@"; do
		if [ "$prev" = --file ]; then cat "$a" >>"$CALLS.embargo-text"; echo "$a" >>"$CALLS.embargo-files"; fi
		prev="$a"
	done
	if [ "${FAKE_EMBARGO:-0}" = unknown ]; then
		echo 'Error: unknown command "check" for "abcd embargo"' >&2
		exit 1
	fi
	exit "${FAKE_EMBARGO:-0}" ;;
"lint outbound") exit "${FAKE_OUTBOUND:-0}" ;;
esac
echo "fake abcd: unexpected $*" >&2
exit 3
`
	if err := os.WriteFile(fakeAbcd, []byte(body), 0o755); err != nil {
		c.t.Fatal(err)
	}
	c.env = append(c.env, "FAKE_ABCD="+fakeAbcd)
	c.fake("go", goFakeForAbcd)
}

// goFakeForAbcd is the fixture's `go` for a test whose abcd is $FAKE_ABCD:
// a build copies it, and a run collapses its exit code as the real go run does.
const goFakeForAbcd = `echo "go $*" >>"$CALLS"
if [ "${1:-}" = build ] && [ "${2:-}" = -o ] && [ $# -eq 4 ] && [ "$4" = ./cmd/abcd ]; then
	if [ -n "${FAKE_BUILD:-}" ]; then echo "fake go: the build failed" >&2; exit "$FAKE_BUILD"; fi
	cat "$FAKE_ABCD" >"$3" && chmod +x "$3"
	exit
fi
if [ "${1:-}" = run ] && [ "${2:-}" = ./cmd/abcd ]; then
	shift 2
	"$FAKE_ABCD" "$@" || exit 1
	exit 0
fi
echo "fake go: unexpected $*" >&2
exit 3`

// commitRecord writes a record and commits it on the current branch; push
// sends the branch to origin.
func (c *cloudCase) commitRecord(rel string, push bool) {
	c.t.Helper()
	c.write(rel, "---\nid: fixture\n---\n\n# A fixture item\n", 0o644)
	c.git("add", "--", rel)
	c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
		"commit", "-q", "-m", "add "+rel)
	if push {
		branch := c.git("rev-parse", "--abbrev-ref", "HEAD")
		c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", branch)
	}
}

// scriptSplit runs one cloud script, keeping its standard output apart from
// its standard error: a refused brief must print nothing on standard output.
func (c *cloudCase) scriptSplit(extra []string, rel string, args ...string) (stdout, stderr string, code int) {
	c.t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(c.dir, filepath.FromSlash(rel))}, args...)...)
	cmd.Dir = c.dir
	cmd.Env = append(append([]string{}, c.env...), extra...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.String(), errb.String(), cloudExitCode(c.t, err)
}

func (c *cloudCase) home() string {
	c.t.Helper()
	home := ""
	for _, kv := range c.env {
		if strings.HasPrefix(kv, "HOME=") {
			home = strings.TrimPrefix(kv, "HOME=")
		}
	}
	if home == "" {
		c.t.Fatal("the fixture's environment carries no HOME")
	}
	return home
}

// embargoStore makes the fixture's embargo store, where abcd keeps it:
// worktrees/<root-sha>/embargo under abcd's home.
func (c *cloudCase) embargoStore() string {
	c.t.Helper()
	root := c.git("rev-list", "-n", "1", "--max-parents=0", "HEAD")
	p := abcdhome.Path(c.home(), "worktrees", root, "embargo")
	if err := os.MkdirAll(p, 0o700); err != nil {
		c.t.Fatal(err)
	}
	return p
}

func newBriefCase(t *testing.T) *cloudCase {
	t.Helper()
	c := newCloudCase(t)
	c.fakeAbcd()
	return c
}

func TestBriefNamesEveryPart(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.resetCalls()
	out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("brief for a public open issue exited %d\n%s", code, errOut)
	}
	for _, part := range []struct{ what, want string }{
		{"the item", fixtureIssue},
		{"its record path", issueRecord(fixtureIssue)},
		{"the first step", "scripts/cloud/begin.sh"},
		{"the off-cloud stop, which does nothing else", "do nothing else, not even a hand-back"},
		{"the other stops, which hand back", "If it stops for any other reason"},
		{"the author rule (the identity the setup took from the forge token)", "forge token"},
		{"the no-skip rule", "--no-verify"},
		{"the go-run rule", "go run ./cmd/abcd"},
		{"the disclosure trailer", "Assisted-by:"},
		{"the reserve rule", "go run ./cmd/abcd reserve " + fixtureIssue},
		{"the branch, from the connector's prefix", "claude/abcd-wip/" + fixtureIssue},
		{"the hand-back", "scripts/cloud/hand-back.sh --state"},
		{"the hand-back's item", "--item " + fixtureIssue},
		{"the report file", ".abcd/work/cloud-reports/" + fixtureIssue + ".md"},
		{"the fallback", "tracking issue"},
	} {
		if !strings.Contains(out, part.want) {
			t.Errorf("the brief does not name %s (%q)\n%s", part.what, part.want, out)
		}
	}
	if strings.Contains(out, "{{") || strings.Contains(out, "}}") {
		t.Errorf("the brief carries an unfilled placeholder\n%s", out)
	}
	if !strings.Contains(c.callLog(), "lint outbound --label brief") {
		t.Errorf("the brief was not passed through the outbound check:\n%s", c.callLog())
	}

	// The branch is the connector's: a second connector with its own prefix
	// names its own branch, and with two declared the brief will not guess.
	c.declare("other", "name=other\nwip_prefix=abcd-wip/\nis_remote=is-remote.sh\n")
	if out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue); code == 0 || out != "" ||
		!strings.Contains(errOut, "refusing to guess") {
		t.Errorf("with two connectors and none named, brief exited %d\nstdout %q\nstderr %s", code, out, errOut)
	}
	out, errOut, code = c.scriptSplit(nil, briefScript, "--connector", "other", fixtureIssue)
	if code != 0 || !strings.Contains(out, "`abcd-wip/"+fixtureIssue+"`") || strings.Contains(out, "claude/abcd-wip") {
		t.Errorf("--connector other did not name that connector's branch (exit %d)\n%s%s", code, out, errOut)
	}
}

func TestBriefCarriesNoSessionLinkHomePathOrSecret(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("brief exited %d\n%s", code, errOut)
	}
	for _, bad := range []string{c.home(), c.dir, "://", "session_", "session/", "session="} {
		if strings.Contains(out, bad) {
			t.Errorf("the committed brief carries %q\n%s", bad, out)
		}
	}

	tmpl, err := os.ReadFile(filepath.Join(c.dir, filepath.FromSlash(briefTemplate)))
	if err != nil {
		t.Fatal(err)
	}
	token := "ghp_" + strings.Repeat("A1b2C3", 6)
	for _, tc := range []struct{ name, line, cause string }{
		{"a session link", "See https://example.invalid/code/session_01ABCDEFGHIJKLMNOPQR for context.", "session link"},
		{"this computer's home", "Notes are in " + c.home() + "/notes.txt.", "home"},
		{"another home path", "Notes are in /Users/someone/notes.txt.", "home"},
		{"a Linux home path", "Notes are in /home/someone/notes.txt.", "home"},
		{"a forge token", "Use " + token + " to push.", "secret"},
		{"a private key", "-----BEGIN OPENSSH PRIVATE KEY-----", "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c.write(briefTemplate, string(tmpl)+"\n"+tc.line+"\n", 0o644)
			out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
			if code != 1 || out != "" {
				t.Fatalf("a brief carrying %s exited %d with stdout %q, want a refusal printing no brief\n%s", tc.name, code, out, errOut)
			}
			if !strings.Contains(errOut, tc.cause) {
				t.Errorf("the refusal does not name its cause (%q)\n%s", tc.cause, errOut)
			}
			if strings.Contains(errOut, token) {
				t.Errorf("the refusal repeats the secret it found\n%s", errOut)
			}
		})
	}

	// The outbound check's own refusal is a refusal.
	c.write(briefTemplate, string(tmpl), 0o644)
	out, errOut, code = c.scriptSplit([]string{"FAKE_OUTBOUND=1"}, briefScript, fixtureIssue)
	if code != 1 || out != "" || !strings.Contains(errOut, "outbound") {
		t.Errorf("a brief the outbound check refuses exited %d with stdout %q\n%s", code, out, errOut)
	}
	// And so is an outbound check that cannot run.
	out, errOut, code = c.scriptSplit([]string{"FAKE_OUTBOUND=2"}, briefScript, fixtureIssue)
	if code != 1 || out != "" || !strings.Contains(errOut, "outbound") {
		t.Errorf("an outbound check that could not run let the brief through (exit %d, stdout %q)\n%s", code, out, errOut)
	}
}

func TestBriefRefusesUnreadyIntent(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(intentRecord("planned", fixtureIntent), true)
	c.resetCalls()
	out, errOut, code := c.scriptSplit([]string{"FAKE_READY=1"}, briefScript, fixtureIntent)
	if code != 1 || out != "" {
		t.Fatalf("an unready planned intent exited %d with stdout %q\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "not ready") {
		t.Errorf("the refusal does not say the intent is not ready\n%s", errOut)
	}
	if log := c.callLog(); !strings.Contains(log, "abcd intent ready "+fixtureIntent+" --json") {
		t.Errorf("the readiness gate was not asked:\n%s", log)
	}
	// A readiness gate that cannot run is a refusal too.
	if out, errOut, code := c.scriptSplit([]string{"FAKE_READY=3"}, briefScript, fixtureIntent); code != 1 || out != "" {
		t.Errorf("a readiness gate that could not run let the brief through (exit %d)\n%s", code, errOut)
	}

	// Records that are not ready to build, whatever the gate would say.
	for _, tc := range []struct{ name, rel, id string }{
		{"a draft intent", intentRecord("drafts", "itd-2610150000000003"), "itd-2610150000000003"},
		{"a shipped intent", intentRecord("shipped", "itd-2610150000000004"), "itd-2610150000000004"},
		{"a resolved issue", ".abcd/work/issues/resolved/iss-2610150000000005-a-fixture-item.md", "iss-2610150000000005"},
		{"no record at all", "", "iss-2610150000000006"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.rel != "" {
				c.commitRecord(tc.rel, true)
			}
			c.resetCalls()
			out, errOut, code := c.scriptSplit(nil, briefScript, tc.id)
			if code != 1 || out != "" || !strings.Contains(errOut, "ready to build") {
				t.Errorf("%s exited %d with stdout %q, want a refusal saying it is not ready to build\n%s", tc.name, code, out, errOut)
			}
			if strings.Contains(c.callLog(), "intent ready") {
				t.Errorf("%s reached the readiness gate:\n%s", tc.name, c.callLog())
			}
		})
	}

	// Not an item id at all: misuse, before anything runs.
	for _, bad := range []string{"", "spc-2610150000000007", "iss-1/../x", "itd-26101500000000021"} {
		c.resetCalls()
		if out, _, code := c.scriptSplit(nil, briefScript, bad); code != 2 || out != "" {
			t.Errorf("brief %q exited %d, want 2 (misuse)", bad, code)
		}
	}
}

func TestBriefAcceptsOpenIssueAndReadyIntent(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.commitRecord(intentRecord("planned", fixtureIntent), true)
	for _, tc := range []struct{ id, record string }{
		{fixtureIssue, issueRecord(fixtureIssue)},
		{fixtureIntent, intentRecord("planned", fixtureIntent)},
	} {
		out, errOut, code := c.scriptSplit(nil, briefScript, tc.id)
		if code != 0 {
			t.Errorf("brief %s exited %d\n%s", tc.id, code, errOut)
			continue
		}
		if !strings.Contains(out, tc.record) || !strings.Contains(out, "claude/abcd-wip/"+tc.id) {
			t.Errorf("brief %s does not name its record and branch\n%s", tc.id, out)
		}
	}
	// Ordinal ids are items too.
	c.commitRecord(issueRecord("iss-42"), true)
	if out, errOut, code := c.scriptSplit(nil, briefScript, "iss-42"); code != 0 || !strings.Contains(out, issueRecord("iss-42")) {
		t.Errorf("brief for the ordinal iss-42 exited %d\n%s%s", code, out, errOut)
	}
}

func TestBriefRefusesItemNotOnRemoteDefault(t *testing.T) {
	notPublic := func(t *testing.T, c *cloudCase, id string) {
		t.Helper()
		c.resetCalls()
		out, errOut, code := c.scriptSplit(nil, briefScript, id)
		if code != 1 || out != "" {
			t.Fatalf("exited %d with stdout %q, want a refusal printing no brief\n%s", code, out, errOut)
		}
		if !strings.Contains(errOut, "not public") {
			t.Errorf("the refusal does not say the item is not public\n%s", errOut)
		}
		// Refused before a brief was composed.
		if log := c.callLog(); strings.Contains(log, "embargo") || strings.Contains(log, "lint outbound") {
			t.Errorf("a brief was composed for an item that is not public:\n%s", log)
		}
	}

	t.Run("a draft never pushed", func(t *testing.T) {
		c := newBriefCase(t)
		c.write(issueRecord(fixtureIssue), "# never committed\n", 0o644)
		notPublic(t, c, fixtureIssue)
		c.git("add", "--", issueRecord(fixtureIssue))
		c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
			"commit", "-q", "-m", "committed, not pushed")
		notPublic(t, c, fixtureIssue)
	})
	t.Run("a record only on another branch", func(t *testing.T) {
		c := newBriefCase(t)
		c.git("checkout", "-q", "-b", "topic")
		c.commitRecord(issueRecord(fixtureIssue), true)
		notPublic(t, c, fixtureIssue)
	})
	t.Run("a remote whose default branch cannot be read", func(t *testing.T) {
		c := newBriefCase(t)
		c.commitRecord(issueRecord(fixtureIssue), true)
		if out, err := c.run(c.origin, "git", "symbolic-ref", "HEAD", "refs/heads/nosuch"); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		c.resetCalls()
		out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
		if code != 1 || out != "" || !strings.Contains(errOut, "default branch") {
			t.Errorf("an undeterminable default branch exited %d with stdout %q, want a refusal naming it\n%s", code, out, errOut)
		}
	})
	t.Run("a remote that cannot be reached", func(t *testing.T) {
		c := newBriefCase(t)
		c.commitRecord(issueRecord(fixtureIssue), true)
		c.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
		out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
		if code != 1 || out != "" || !strings.Contains(errOut, "fetch") {
			t.Errorf("an unreachable remote exited %d with stdout %q, want a refusal naming the fetch\n%s", code, out, errOut)
		}
	})
}

func TestBriefRefusesEmbargoedItem(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.embargoStore()
	c.resetCalls()
	out, errOut, code := c.scriptSplit([]string{"FAKE_EMBARGO=2"}, briefScript, fixtureIssue)
	if code != 1 || out != "" {
		t.Fatalf("an embargoed item exited %d with stdout %q\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "embargo") {
		t.Errorf("the refusal does not name the embargo\n%s", errOut)
	}
	if !strings.Contains(c.callLog(), "abcd embargo check --file ") {
		t.Errorf("the embargo check was not run over a file:\n%s", c.callLog())
	}
	checked, err := os.ReadFile(c.calls + ".embargo-text")
	if err != nil || !strings.Contains(string(checked), "scripts/cloud/begin.sh") || !strings.Contains(string(checked), fixtureIssue) {
		t.Errorf("the embargo check was not given the composed brief (%v)\n%s", err, checked)
	}
}

func TestBriefFailsClosedWhenEmbargoCheckCannotRun(t *testing.T) {
	for _, tc := range []struct{ name, mode string }{
		{"the verb is unknown", "unknown"},
		{"the check fails otherwise", "3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newBriefCase(t)
			c.commitRecord(issueRecord(fixtureIssue), true)
			c.embargoStore()
			out, errOut, code := c.scriptSplit([]string{"FAKE_EMBARGO=" + tc.mode}, briefScript, fixtureIssue)
			if code != 1 || out != "" {
				t.Fatalf("a store with a check that cannot run exited %d with stdout %q, want a refusal\n%s", code, out, errOut)
			}
			if !strings.Contains(errOut, "embargo") || !strings.Contains(errOut, "could not run") {
				t.Errorf("the refusal does not say the embargo check could not run\n%s", errOut)
			}
		})
	}
}

func TestBriefWithoutStoreSkipsEmbargoCheck(t *testing.T) {
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.resetCalls()
	// FAKE_EMBARGO=2 would refuse if the check ran.
	out, errOut, code := c.scriptSplit([]string{"FAKE_EMBARGO=2"}, briefScript, fixtureIssue)
	if code != 0 || !strings.Contains(out, fixtureIssue) {
		t.Fatalf("with no embargo store, brief exited %d\n%s", code, errOut)
	}
	if strings.Contains(c.callLog(), "embargo") {
		t.Errorf("the embargo check ran with no store on this machine:\n%s", c.callLog())
	}
}

func TestBriefReadsNoAuthorFromThisComputer(t *testing.T) {
	const name, email = "Zebulon Quarterfield-Fixture", "zebulon.quarterfield@fixture.invalid"
	c := newBriefCase(t)
	c.commitRecord(issueRecord(fixtureIssue), true)
	c.git("config", "user.name", name)
	c.git("config", "user.email", email)
	c.embargoStore()
	c.resetCalls()
	env := []string{"GIT_AUTHOR_NAME=" + name, "GIT_AUTHOR_EMAIL=" + email, "GIT_COMMITTER_NAME=" + name, "GIT_COMMITTER_EMAIL=" + email}
	out, errOut, code := c.scriptSplit(env, briefScript, fixtureIssue)
	if code != 0 {
		t.Fatalf("brief exited %d\n%s", code, errOut)
	}
	checked, _ := os.ReadFile(c.calls + ".embargo-text")
	for _, s := range []string{name, email, "Zebulon", "fixture.invalid"} {
		for where, text := range map[string]string{"the brief": out, "the diagnostics": errOut, "the checked text": string(checked)} {
			if strings.Contains(text, s) {
				t.Errorf("%s carries this computer's identity (%q)\n%s", where, s, text)
			}
		}
	}
	// The script never asks git for an identity at all.
	for _, l := range strings.Split(c.callLog(), "\n") {
		for _, ask := range []string{"user.name", "user.email", "var GIT_", "%an", "%ae", "%cn", "%ce"} {
			if strings.HasPrefix(l, "git ") && strings.Contains(l, ask) {
				t.Errorf("brief asked git for an identity: %q", l)
			}
		}
	}
}

// TestBriefReadsAbcdsRealExitCodes: the brief branches on abcd's exit codes
// (2 is the embargo check's refusal; 1 a gate's refusal, anything else a gate
// that could not run), and `go run` turns every non-zero exit into 1. So the
// brief runs a binary built from this checkout, once per run, or the one
// ABCD_BIN names.
func TestBriefReadsAbcdsRealExitCodes(t *testing.T) {
	t.Run("go run collapses an exit code", func(t *testing.T) {
		if _, err := exec.LookPath("go"); err != nil {
			t.Skip("go unavailable")
		}
		mod := t.TempDir()
		for name, body := range map[string]string{
			"go.mod":  "module exitcode\n\ngo 1.21\n",
			"main.go": "package main\n\nimport \"os\"\n\nfunc main() { os.Exit(2) }\n",
		} {
			if err := os.WriteFile(filepath.Join(mod, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		run := exec.Command("go", "run", ".")
		run.Dir = mod
		run.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=")
		out, err := run.CombinedOutput()
		if code := cloudExitCode(t, err); code != 1 {
			t.Fatalf("go run of a program exiting 2 exited %d, want 1 (the collapse this test is about)\n%s", code, out)
		}
	})

	held := "it is held under embargo"
	for _, tc := range []struct {
		name, env, want, never string
	}{
		{"the embargo check's refusal", "FAKE_EMBARGO=2", held, "could not run"},
		{"a readiness gate that cannot run", "FAKE_READY=2", "readiness gate (abcd intent ready) could not run (exit 2)", "it is not ready:"},
		{"an outbound check that cannot run", "FAKE_OUTBOUND=2", "could not run (exit 2)", "fails the outbound check"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newBriefCase(t)
			c.commitRecord(intentRecord("planned", fixtureIntent), true)
			c.embargoStore()
			c.resetCalls()
			out, errOut, code := c.scriptSplit([]string{tc.env}, briefScript, fixtureIntent)
			if code != 1 || out != "" {
				t.Fatalf("exited %d with stdout %q, want a refusal printing no brief\n%s", code, out, errOut)
			}
			if !strings.Contains(errOut, tc.want) || strings.Contains(errOut, tc.never) {
				t.Errorf("the refusal does not read abcd's own exit code: want %q and never %q\n%s", tc.want, tc.never, errOut)
			}
			log := c.callLog()
			if n := strings.Count(log, "go build -o "); n != 1 {
				t.Errorf("abcd was built %d times, want once per run:\n%s", n, log)
			}
			if strings.Contains(log, "go run") {
				t.Errorf("the brief still runs abcd through go run, which collapses its exit code:\n%s", log)
			}
		})
	}

	t.Run("ABCD_BIN names a prebuilt binary", func(t *testing.T) {
		c := newBriefCase(t)
		c.commitRecord(issueRecord(fixtureIssue), true)
		c.embargoStore()
		c.resetCalls()
		bin := ""
		for _, kv := range c.env {
			if strings.HasPrefix(kv, "FAKE_ABCD=") {
				bin = strings.TrimPrefix(kv, "FAKE_ABCD=")
			}
		}
		out, errOut, code := c.scriptSplit([]string{"ABCD_BIN=" + c.localAbcd(bin), "FAKE_EMBARGO=2"}, briefScript, fixtureIssue)
		if code != 1 || out != "" || !strings.Contains(errOut, held) {
			t.Errorf("an embargo refusal through ABCD_BIN exited %d with stdout %q, want the embargo refusal\n%s", code, out, errOut)
		}
		if log := c.callLog(); regexp.MustCompile(`(?m)^go `).MatchString(log) {
			t.Errorf("with ABCD_BIN set the brief ran go:\n%s", log)
		}

		// A value that is not an executable file is not used: abcd is built.
		c.resetCalls()
		out, errOut, code = c.scriptSplit([]string{"ABCD_BIN=" + filepath.Join(t.TempDir(), "absent")}, briefScript, fixtureIssue)
		if code != 0 || !strings.Contains(out, fixtureIssue) {
			t.Errorf("with ABCD_BIN naming no file the brief exited %d\n%s", code, errOut)
		}
		if n := strings.Count(c.callLog(), "go build -o "); n != 1 {
			t.Errorf("with ABCD_BIN naming no file abcd was built %d times, want once:\n%s", n, c.callLog())
		}
	})

	t.Run("a build that fails is a refusal", func(t *testing.T) {
		c := newBriefCase(t)
		c.commitRecord(issueRecord(fixtureIssue), true)
		out, errOut, code := c.scriptSplit([]string{"FAKE_BUILD=1"}, briefScript, fixtureIssue)
		if code != 1 || out != "" || !strings.Contains(errOut, "could not be built") {
			t.Errorf("a failed build exited %d with stdout %q, want a refusal saying abcd could not be built\n%s", code, out, errOut)
		}
	})
}

// embargoTitle is the fixture advisory's title, an invented one
// (examples-use-reserved-identifiers), long enough for the check to match.
const embargoTitle = "Sample Overflow In The Example Parser"

// TestBriefChecksTheRecordAgainstTheEmbargo runs the real abcd, built from this
// checkout, against a real embargo store: the store is made by `abcd embargo
// intake`, reading one advisory in triage from a fake `gh`. The brief carries
// the item's id and path, never its text, so the check is also given the
// item's record: a record naming an embargoed advisory's title is refused.
func TestBriefChecksTheRecordAgainstTheEmbargo(t *testing.T) {
	bin := builtAbcd.Path(t)
	c := newCloudCase(t)
	c.env = append(c.env, "ABCD_BIN="+c.localAbcd(bin))
	advisory := `{"ghsa_id":"GHSA-xxxx-xxxx-xxxx","cve_id":null,"summary":"` + embargoTitle + `",` +
		`"description":"the report","severity":"high","state":"triage","vulnerabilities":[]}`
	c.fake("gh", `echo "gh $*" >>"$CALLS"
for a in "$@"; do case "$a" in
	state=triage) echo '[`+advisory+`]'; exit 0 ;;
	state=*) echo '[]'; exit 0 ;;
esac; done
echo '`+advisory+`'`)

	clean, named := issueRecord(fixtureIssue), issueRecord("iss-2610150000000009")
	for rel, body := range map[string]string{
		clean: "---\nid: fixture\n---\n\n# A fixture item\n\nThe walk skips a folder.\n",
		named: "---\nid: fixture\n---\n\n# A fixture item\n\nFix the " + embargoTitle + " before it is published.\n",
	} {
		c.write(rel, body, 0o644)
	}
	c.git("add", "-A")
	c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
		"commit", "-q", "-m", "two records")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")

	// Intake reads origin as a github.com repository; the brief fetches from
	// the bare one.
	c.git("remote", "set-url", "origin", "https://github.com/example/example.git")
	if out, err := c.run(c.dir, bin, "embargo", "intake"); err != nil {
		t.Fatalf("abcd embargo intake: %v\n%s", err, out)
	}
	c.git("remote", "set-url", "origin", c.origin)

	out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue)
	if code != 0 || !strings.Contains(out, fixtureIssue) {
		t.Fatalf("a record naming no embargoed advisory: the brief exited %d\n%s", code, errOut)
	}
	out, errOut, code = c.scriptSplit(nil, briefScript, "iss-2610150000000009")
	if code != 1 || out != "" {
		t.Fatalf("a record naming an embargoed advisory's title: the brief exited %d with stdout %q, want a refusal\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "held under embargo") {
		t.Errorf("the refusal does not say the item is held under embargo\n%s", errOut)
	}
	if strings.Contains(errOut, embargoTitle) {
		t.Errorf("the refusal repeats the embargoed title\n%s", errOut)
	}
}

// TestBriefRefusesRecordWithUnpushedEdits: readiness is judged on this
// computer's copy of the record, and the cloud session reads origin's. So the
// two must be the same blob.
func TestBriefRefusesRecordWithUnpushedEdits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		commit bool
	}{
		{"an edit not committed", false},
		{"an edit committed but not pushed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newBriefCase(t)
			rel := intentRecord("planned", fixtureIntent)
			c.commitRecord(rel, true)
			c.write(rel, "---\nid: fixture\n---\n\n# A fixture item, edited here\n", 0o644)
			if tc.commit {
				c.git("add", "--", rel)
				c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
					"commit", "-q", "-m", "edit "+rel)
			}
			c.resetCalls()
			out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIntent)
			if code != 1 || out != "" {
				t.Fatalf("exited %d with stdout %q, want a refusal printing no brief\n%s", code, out, errOut)
			}
			if !strings.Contains(errOut, "changes not on origin's default branch") || !strings.Contains(errOut, "push them first") {
				t.Errorf("the refusal does not say the record has changes not on origin\n%s", errOut)
			}
			if log := c.callLog(); strings.Contains(log, "lint outbound") || strings.Contains(log, "embargo") {
				t.Errorf("a brief was composed for a record that differs from origin's:\n%s", log)
			}
		})
	}
}

// TestBriefRefusesUnpushedSpecEdits: `abcd intent ready` reads more than the
// intent's record: the spec store, the linked spec's body and steps among it.
// The cloud session reads origin's copy of all of them, so the brief refuses
// an intent while any path under .abcd/development/ here differs from origin's
// default branch: an edit not committed, one committed and not pushed, and a
// spec never added.
func TestBriefRefusesUnpushedSpecEdits(t *testing.T) {
	spec := ".abcd/development/specs/open/spc-2610150000000008-a-fixture-item.md"
	for _, tc := range []struct {
		name string
		edit func(c *cloudCase)
	}{
		{"a spec edit not committed", func(c *cloudCase) {
			c.write(spec, "---\nid: spc\n---\n\n# Edited here\n", 0o644)
		}},
		{"a spec edit committed but not pushed", func(c *cloudCase) {
			c.write(spec, "---\nid: spc\n---\n\n# Edited here\n", 0o644)
			c.git("add", "--", spec)
			c.git("-c", "user.name=Seed Person", "-c", "user.email=seed@example.com", "-c", "core.hooksPath=/dev/null",
				"commit", "-q", "-m", "edit the spec")
		}},
		{"a spec never added", func(c *cloudCase) {
			c.write(".abcd/development/specs/open/spc-2610150000000009-another.md", "---\nid: spc\n---\n", 0o644)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newBriefCase(t)
			c.commitRecord(intentRecord("planned", fixtureIntent), true)
			c.commitRecord(spec, true)
			tc.edit(c)
			c.resetCalls()
			out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIntent)
			if code != 1 || out != "" {
				t.Fatalf("exited %d with stdout %q, want a refusal printing no brief\n%s", code, out, errOut)
			}
			if !strings.Contains(errOut, ".abcd/development/") || !strings.Contains(errOut, "push") {
				t.Errorf("the refusal does not say the design records here differ from origin's\n%s", errOut)
			}
			if log := c.callLog(); strings.Contains(log, "lint outbound") || strings.Contains(log, "embargo") {
				t.Errorf("a brief was composed while the records differ from origin's:\n%s", log)
			}
		})
	}
	t.Run("an issue is judged on its record alone", func(t *testing.T) {
		c := newBriefCase(t)
		c.commitRecord(issueRecord(fixtureIssue), true)
		c.write(spec, "---\nid: spc\n---\n\n# Not pushed\n", 0o644)
		if out, errOut, code := c.scriptSplit(nil, briefScript, fixtureIssue); code != 0 || !strings.Contains(out, fixtureIssue) {
			t.Errorf("an open issue was refused for a spec edit its readiness never reads (exit %d)\n%s", code, errOut)
		}
	})
}

// TestBriefTemplateAndCloudFilesCheckOutLF: the brief template is read line by
// line into the brief, the cloud scripts run under bash and their settings are
// read with sed, so a CRLF checkout would put a \r into every line of each.
// Every file tracked under scripts/cloud/, brief.md.tmpl among them, is pinned
// to LF in .gitattributes, whatever its extension.
func TestBriefTemplateAndCloudFilesCheckOutLF(t *testing.T) {
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))
	ls := exec.Command("git", "-C", root, "ls-files", "--", "scripts/cloud")
	ls.Env = gittest.Env(t)
	files, err := ls.Output()
	if err != nil {
		t.Fatalf("git ls-files scripts/cloud: %v", err)
	}
	if !strings.Contains(string(files), briefTemplate+"\n") {
		t.Fatalf("%s is not tracked under scripts/cloud/:\n%s", briefTemplate, files)
	}
	attr := exec.Command("git", "-C", root, "check-attr", "--stdin", "eol")
	attr.Env = gittest.Env(t)
	attr.Stdin = bytes.NewReader(files)
	got, err := attr.Output()
	if err != nil {
		t.Fatalf("git check-attr eol: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(got)), "\n") {
		if !strings.HasSuffix(line, ": eol: lf") {
			t.Errorf("not pinned to LF in .gitattributes: %s", line)
		}
	}
}
