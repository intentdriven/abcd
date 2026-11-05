package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// The committed pre-push hook refuses a push that would fast-forward the head of
// an armed pull request, naming what to run. It changes nothing on the forge: a
// hook cannot tell a dry run from a push, so it asks and stops, and the person's
// next push asks again. It asks of the repository the push's own URL names. The
// hook is run directly with a stand-in forge client, and no network is touched.

// armingForge is a stand-in forge client. It answers the arming read by applying
// the hook's own --jq to the file armings.json, as the real client does, and logs
// every call. Only acme/widgets holds the pull requests armings.json lists.
const armingForge = `#!/bin/sh
d="$(cd "$(dirname "$0")" && pwd)"
printf '%s\n' "$*" >> "$d/gh.log"
jq_expr=""
prev=""
for a in "$@"; do
  [ "$prev" = "--jq" ] && jq_expr="$a"
  prev="$a"
done
case "$1 $2" in
  "api graphql")
    if [ -f "$d/refuse-read" ]; then echo "stand-in: the forge could not be read" >&2; exit 1; fi
    case "$(printf '%s' "$*" | tr '[:upper:]' '[:lower:]')" in
      *"o=acme"*"r=widgets"*) jq -r "$jq_expr" "$d/armings.json" ;;
      *) echo '{"data":{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}' | jq -r "$jq_expr" ;;
    esac ;;
  *) echo "stand-in: unexpected call: $*" >&2; exit 1 ;;
esac
`

// armingSSH is a stand-in ssh that resolves two host aliases, as `ssh -G` does
// from the person's own configuration.
const armingSSH = `#!/bin/sh
case "$2" in
  work-alias) echo "hostname github.com" ;;
  mirror-alias) echo "hostname gitlab.example.com" ;;
  *) echo "hostname $2" ;;
esac
`

// armingHook is a case whose feature branch is on origin at old, with one commit
// past it that a receipt names: a push of it fast-forwards the branch.
type armingHook struct {
	*prePushCase
	gh        string
	old, head string
}

const armedURL = "git@github.com:acme/widgets.git"

func newArmingHook(t *testing.T) *armingHook {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq unavailable: the stand-in applies the hook's --jq with it")
	}
	c := newPrePushCase(t)
	old := c.commit("feature.md", "a feature\n")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "feature")
	head := c.commit("more.md", "more\n")
	if out := c.preflight(c.dir); !strings.Contains(out, "receipt minted") {
		t.Fatalf("no receipt minted\n%s", out)
	}
	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(armingForge), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gh, "ssh"), []byte(armingSSH), 0o755); err != nil {
		t.Fatal(err)
	}
	return &armingHook{prePushCase: c, gh: gh, old: old, head: head}
}

// armings writes the nodes the forge answers the arming read with.
func (h *armingHook) armings(t *testing.T, nodes string) {
	t.Helper()
	h.armingsMore(t, nodes, false)
}

// armingsMore is armings with whether the forge holds more behind them.
func (h *armingHook) armingsMore(t *testing.T, nodes string, more bool) {
	t.Helper()
	next := "false"
	if more {
		next = "true"
	}
	body := `{"data":{"viewer":{"login":"alice"},"repository":{"pullRequests":{"pageInfo":{"hasNextPage":` + next + `},"nodes":` + nodes + `}}}}`
	if err := os.WriteFile(filepath.Join(h.gh, "armings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func armingAt(head string, auto, queued bool) string {
	arm, q := "null", "false"
	if auto {
		arm = `{"mergeMethod":"MERGE"}`
	}
	if queued {
		q = "true"
	}
	return `[{"number":7,"id":"PR_7","headRefOid":"` + head + `","isCrossRepository":false,"isInMergeQueue":` + q + `,"autoMergeRequest":` + arm + `}]`
}

// run runs the hook on one ref line for a remote, with path as PATH ("" is the
// stand-in first on the fixture's own). It returns what it printed and its exit
// error.
func (h *armingHook) runAs(t *testing.T, remote, url, line, path string) (string, error) {
	t.Helper()
	if path == "" {
		path = h.gh + string(os.PathListSeparator) + os.Getenv("PATH")
	}
	cmd := exec.Command(filepath.Join(h.dir, ".githooks", "pre-push"), remote, url)
	cmd.Dir = h.dir
	cmd.Env = append(append([]string{}, h.env...), "PATH="+path)
	cmd.Stdin = strings.NewReader(line + "\n")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

func (h *armingHook) run(t *testing.T, url, line, path string) (string, error) {
	t.Helper()
	return h.runAs(t, "origin", url, line, path)
}

func (h *armingHook) log(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(h.gh, "gh.log"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return string(raw)
}

// update is the ref line of a push fast-forwarding the feature branch.
func (h *armingHook) update() string {
	return "refs/heads/feature " + h.head + " refs/heads/feature " + h.old
}

// noForgeChange fails on any call that would change the forge.
func (h *armingHook) noForgeChange(t *testing.T) {
	t.Helper()
	if log := h.log(t); strings.Contains(log, "pr merge") || strings.Contains(log, "dequeuePullRequest") {
		t.Fatalf("the hook changed the forge:\n%s", log)
	}
}

func TestPrePushRefusesAPushThatWouldMoveAnArmedHead(t *testing.T) {
	for _, c := range []struct {
		name         string
		auto, queued bool
		says         string
	}{
		{"armed", true, false, "gh pr merge 7 --disable-auto"},
		{"queued", false, true, "merge queue"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newArmingHook(t)
			h.armings(t, armingAt(h.old, c.auto, c.queued))
			out, err := h.run(t, armedURL, h.update(), "")
			if err == nil {
				t.Fatalf("the push went ahead:\n%s", out)
			}
			for _, want := range []string{"#7", h.old[:12], h.head[:12], "abcd land", c.says} {
				if !strings.Contains(out, want) {
					t.Fatalf("the refusal does not say %q:\n%s", want, out)
				}
			}
			h.noForgeChange(t)
		})
	}
}

// A push that withdrew the arming first, by land or by hand, goes ahead: the
// next read finds it gone.
func TestPrePushPassesOnceTheArmingIsGone(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, `[]`)
	if out, err := h.run(t, armedURL, h.update(), ""); err != nil {
		t.Fatalf("the push was refused: %v\n%s", err, out)
	}
}

// The hook asks about a push that fast-forwards a branch.
func TestPrePushAsksOnlyAboutAFastForward(t *testing.T) {
	zero := strings.Repeat("0", 40)
	h := newArmingHook(t)
	h.armings(t, armingAt(h.old, true, false))
	for _, c := range []struct{ name, line string }{
		{"an update git rejects", "refs/heads/feature " + h.head + " refs/heads/feature " + strings.Repeat("b", 40)},
		{"a branch already at the commit", "refs/heads/feature " + h.head + " refs/heads/feature " + h.head},
		{"a branch the remote does not have", "refs/heads/feature " + h.head + " refs/heads/feature " + zero},
		{"a tag", "refs/tags/v1 " + h.head + " refs/tags/v1 " + zero},
		{"a deletion", "(delete) " + zero + " refs/heads/feature " + h.head},
	} {
		if out, err := h.run(t, armedURL, c.line, ""); err != nil {
			t.Fatalf("%s was refused: %v\n%s", c.name, err, out)
		}
		if log := h.log(t); log != "" {
			t.Fatalf("%s asked the forge:\n%s", c.name, log)
		}
	}
}

func TestPrePushLeavesAnArmingAtTheHeadItPushes(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, armingAt(h.head, true, false))
	if out, err := h.run(t, armedURL, h.update(), ""); err != nil {
		t.Fatalf("the push was refused: %v\n%s", err, out)
	}
}

func TestPrePushLeavesAForksPullRequestAndAnUnarmedOne(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, `[{"number":8,"id":"PR_8","headRefOid":"`+h.old+`","isCrossRepository":true,"isInMergeQueue":false,"autoMergeRequest":{"mergeMethod":"MERGE"}},`+
		`{"number":9,"id":"PR_9","headRefOid":"`+h.old+`","isCrossRepository":false,"isInMergeQueue":false,"autoMergeRequest":null}]`)
	if out, err := h.run(t, armedURL, h.update(), ""); err != nil {
		t.Fatalf("the push was refused: %v\n%s", err, out)
	}
}

// The push's own URL names the repository asked about, whatever the remote is
// called and whatever repository gh would answer for here.
func TestPrePushAsksTheRepositoryThePushReaches(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, armingAt(h.old, true, false))
	if out, err := h.runAs(t, "origin", "git@github.com:me/widgets.git", h.update(), ""); err != nil {
		t.Fatalf("a push to a fork with nothing armed was refused: %v\n%s", err, out)
	}
	if log := h.log(t); !strings.Contains(log, "o=me") || strings.Contains(log, "repo view") {
		t.Fatalf("the hook did not ask the fork:\n%s", log)
	}
}

// A push that reaches no repository on the forge is said in one line, and passes.
func TestPrePushSaysWhereTheCheckDoesNotApply(t *testing.T) {
	for _, c := range []struct{ name, remote, url string }{
		{"a mirror on another host", "mirror", "git@gitlab.example.com:acme/widgets.git"},
		{"a mirror over https", "origin", "https://gitlab.example.com/acme/widgets.git"},
		{"an alias for another host", "origin", "mirror-alias:acme/widgets.git"},
		{"a local path", "origin", "/srv/git/acme/widgets.git"},
		{"a file URL", "origin", "file:///srv/git/acme/widgets.git"},
		{"a credentialed URL on another host", "origin", "https://user:secret@gitlab.example.com/acme/widgets.git"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newArmingHook(t)
			h.armings(t, armingAt(h.old, true, false))
			out, err := h.runAs(t, c.remote, c.url, h.update(), "")
			if err != nil {
				t.Fatalf("the push was refused: %v\n%s", err, out)
			}
			if strings.Count(out, "github.com") != 1 || strings.Contains(out, "secret") {
				t.Fatalf("want one line naming where the check applies, and no credential:\n%s", out)
			}
			if log := h.log(t); log != "" {
				t.Fatalf("the forge was asked:\n%s", log)
			}
		})
	}
}

func TestPrePushRefusesAMoveOfTheArmedHeadUnderAnyRemoteName(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, armingAt(h.old, true, false))
	if out, err := h.runAs(t, "upstream", armedURL, h.update(), ""); err == nil {
		t.Fatalf("a push moving the armed head under another remote's name went ahead:\n%s", out)
	}
}

// The hook reads the first 100 pull requests from the branch, and says so where
// the forge lists more.
func TestPrePushSaysItReadTheFirstHundred(t *testing.T) {
	h := newArmingHook(t)
	h.armingsMore(t, `[]`, true)
	out, err := h.run(t, armedURL, h.update(), "")
	if err != nil || !strings.Contains(out, "first 100") {
		t.Fatalf("err %v:\n%s", err, out)
	}
	h = newArmingHook(t)
	h.armingsMore(t, armingAt(h.old, true, false), true)
	if out, err := h.run(t, armedURL, h.update(), ""); err == nil {
		t.Fatalf("an armed head on the page read was passed:\n%s", out)
	}
}

// A push the receipt gate refuses says that first.
func TestPrePushChecksTheReceiptBeforeTheArming(t *testing.T) {
	h := newArmingHook(t)
	h.armings(t, armingAt(h.old, true, false))
	unreceipted := h.commit("again.md", "again\n")
	line := "refs/heads/feature " + unreceipted + " refs/heads/feature " + h.old
	out, err := h.run(t, armedURL, line, "")
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("the receipt gate did not refuse: %v\n%s", err, out)
	}
	if log := h.log(t); log != "" {
		t.Fatalf("a refused push asked the forge:\n%s", log)
	}
}

// The hook reports the forge client's answer, and the push goes on.
func TestPrePushSaysWhenTheForgeClientGivesNoAnswer(t *testing.T) {
	t.Run("the forge cannot be read", func(t *testing.T) {
		h := newArmingHook(t)
		h.armings(t, armingAt(h.old, true, false))
		if err := os.WriteFile(filepath.Join(h.gh, "refuse-read"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := h.run(t, armedURL, h.update(), "")
		if err != nil || !strings.Contains(out, "no answer") {
			t.Fatalf("err %v:\n%s", err, out)
		}
	})
	t.Run("no forge client", func(t *testing.T) {
		h := newArmingHook(t)
		tools := t.TempDir()
		for _, name := range []string{"git", "bash", "sh", "env", "sed", "grep", "tr", "cut", "mktemp", "tar", "rm", "cat", "dirname", "basename", "head", "tail", "sleep", "find", "sort", "uname", "awk", "wc", "date", "mkdir", "ls", "stat", "shasum", "sha256sum"} {
			if p, err := exec.LookPath(name); err == nil {
				if err := os.Symlink(p, filepath.Join(tools, name)); err != nil {
					t.Fatal(err)
				}
			}
		}
		out, err := h.run(t, armedURL, h.update(), tools)
		if err != nil || !strings.Contains(out, "gh is not installed") {
			t.Fatalf("err %v:\n%s", err, out)
		}
	})
}

// Each form of a github.com URL names its repository, an ssh alias included.
func TestPrePushReadsEachFormOfTheForgesURL(t *testing.T) {
	for _, url := range []string{
		"git@github.com:acme/widgets.git",
		"git@github.com:ACME/Widgets.git",
		"github.com:acme/widgets",
		"ssh://git@ssh.github.com:443/acme/widgets.git",
		"https://github.com/acme/widgets",
		"https://x-access-token:secret@github.com/acme/widgets.git",
		"work-alias:acme/widgets.git",
	} {
		h := newArmingHook(t)
		h.armings(t, armingAt(h.old, true, false))
		out, err := h.run(t, url, h.update(), "")
		if err == nil {
			t.Fatalf("%s: the push went ahead:\n%s", url, out)
		}
		log := strings.ToLower(h.log(t))
		if !strings.Contains(log, "o=acme") || !strings.Contains(log, "r=widgets") || strings.Contains(log, "secret") {
			t.Fatalf("%s did not name acme/widgets:\n%s", url, log)
		}
	}
}

// The hook and the landing ask the forge the one question.
func TestPrePushAsksTheLandingsArmingQuery(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(prePushSourceRoot(t), ".githooks", "pre-push"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), landing.ArmingsQuery) {
		t.Fatalf("the hook's arming read is not landing.ArmingsQuery:\n%s", landing.ArmingsQuery)
	}
}
