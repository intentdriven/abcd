package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The brief (scripts/cloud/brief.sh and brief.md.tmpl, C4 and C13),
// spc-2610120941441282 step 5. The fixture is newCloudCase's clone with a
// bare origin; every abcd verb the script calls goes through a fake `go` on
// PATH (fakeAbcd), whose answer each test decides, so nothing here builds abcd.

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

// fakeAbcd replaces the fixture's `go` with one that answers the three abcd
// verbs the brief runs, each by an environment variable a test sets:
//
//	FAKE_READY     intent ready's exit code (default 0)
//	FAKE_EMBARGO   embargo check's exit code, or `unknown` for a binary that
//	               has no such verb (default 0)
//	FAKE_OUTBOUND  lint outbound's exit code (default 0)
//
// The text embargo check was given is kept beside the call log, so a test can
// prove it was the composed brief.
func (c *cloudCase) fakeAbcd() {
	c.t.Helper()
	c.fake("go", `echo "go $*" >>"$CALLS"
if [ "${1:-}" != run ] || [ "${2:-}" != ./cmd/abcd ]; then echo "fake go: unexpected $*" >&2; exit 3; fi
shift 2
case "${1:-} ${2:-}" in
"intent ready") exit "${FAKE_READY:-0}" ;;
"embargo check")
	prev=""
	for a in "$@"; do
		if [ "$prev" = --file ]; then cat "$a" >"$CALLS.embargo-text"; fi
		prev="$a"
	done
	if [ "${FAKE_EMBARGO:-0}" = unknown ]; then
		echo 'Error: unknown command "check" for "abcd embargo"' >&2
		exit 1
	fi
	exit "${FAKE_EMBARGO:-0}" ;;
"lint outbound") exit "${FAKE_OUTBOUND:-0}" ;;
esac
echo "fake go: unexpected abcd $*" >&2
exit 3`)
}

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
	p := filepath.Join(c.home(), ".abcd.noindex", "worktrees", root, "embargo")
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
		{"the author rule (the identity the setup took from the forge token)", "forge token"},
		{"the no-skip rule", "--no-verify"},
		{"the go-run rule", "go run ./cmd/abcd"},
		{"the disclosure trailer", "Assisted-by:"},
		{"the reserve rule", "go run ./cmd/abcd reserve " + fixtureIssue},
		{"the branch, from the connector's prefix", "claude/abcd-wip/" + fixtureIssue},
		{"the hand-back", "scripts/cloud/hand-back.sh --state"},
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
	if log := c.callLog(); !strings.Contains(log, "go run ./cmd/abcd intent ready "+fixtureIntent+" --json") {
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
	if !strings.Contains(c.callLog(), "go run ./cmd/abcd embargo check --file ") {
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
