package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// peerCheckout is the ground the peer-listing surface tests stand on: a
// repository under a temp HOME holding one open issue, one planned intent and
// one shipped intent, committed, with the process standing in it. Worktrees the
// test adds go under HOME too, so every path the verb prints is one the home
// redaction must catch. HOME is left UNRESOLVED on purpose: on a host whose
// temp dir sits behind a symlink, git reports the resolved spelling, and the
// redaction must catch that one as well.
func peerCheckout(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	repo = filepath.Join(home, "repo")
	gitInitAt(t, repo)
	writeRel(t, repo, ".abcd/work/issues/open/iss-1-the-first-finding.md", peerIssue("iss-1", "the-first-finding", "The first finding"))
	writeRel(t, repo, ".abcd/development/intents/planned/itd-9-a-planned-intent.md", peerIntent("itd-9", "A planned intent"))
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "base ledger")
	t.Chdir(repo)
	return home, repo
}

func writeRel(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// peerIssue is a well-formed open record, so the ledger reader accepts it; the
// slug is the one both fixtures' filenames carry.
func peerIssue(id, slug, title string) string {
	return "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"" + slug + "\"\n" +
		"severity: \"minor\"\ncategory: \"bug\"\nsource: \"agent-finding\"\nfound_during: \"a test\"\n---\n\n" + title + "\n"
}

func peerIntent(id, title string) string {
	return "---\nid: " + id + "\nslug: x\n---\n\n# " + title + "\n"
}

// addPeer adds a linked worktree on a new branch under HOME and lays one
// uncommitted capture in it.
func addPeer(t *testing.T, home, repo, name string) string {
	t.Helper()
	dir := filepath.Join(home, "wt", name)
	gitCmd(t, repo, "worktree", "add", "-q", "-b", "feat/"+name, dir, "main")
	writeRel(t, dir, ".abcd/work/issues/open/iss-100-a-peer-finding.md", peerIssue("iss-100", "a-peer-finding", "A finding the peer captured"))
	return dir
}

// noHomePath fails when out carries the home directory in either spelling.
func noHomePath(t *testing.T, home, out string) {
	t.Helper()
	spellings := []string{home}
	if r, err := filepath.EvalSymlinks(home); err == nil {
		spellings = append(spellings, r)
	}
	for _, s := range spellings {
		if strings.Contains(out, s) {
			t.Fatalf("output carries an unredacted home path %q:\n%s", s, out)
		}
	}
}

// Criteria 1 and 11: the command names the peer's branch and home-redacted
// path, lists the row with its title, exits 0, and --json carries the same
// blocks with no home path in any value.
func TestPeersCommandListsASiblingCapture(t *testing.T) {
	home, repo := peerCheckout(t)
	addPeer(t, home, repo, "a")

	text := string(runCLI(t, "peers"))
	for _, want := range []string{"feat/a", "~/wt/a", "iss-100", "open there, absent here", "A finding the peer captured"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output lacks %q:\n%s", want, text)
		}
	}
	noHomePath(t, home, text)

	raw := runCLI(t, "peers", "--json")
	noHomePath(t, home, string(raw))
	var got struct {
		Live  int `json:"live"`
		IDs   int `json:"ids"`
		Peers []struct {
			Source string `json:"source"`
			Branch string `json:"branch"`
			Path   string `json:"path"`
			Rows   []struct {
				ID, Kind, Folder, Title string
			} `json:"rows"`
		} `json:"peers"`
		Skipped []any    `json:"skipped"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("--json is not JSON: %v\n%s", err, raw)
	}
	if got.Live != 1 || got.IDs != 1 || len(got.Peers) != 1 || got.Skipped == nil {
		t.Fatalf("payload = %+v, want one live peer, one id, and an empty skipped list", got)
	}
	p := got.Peers[0]
	if p.Source != "worktree" || p.Branch != "feat/a" || p.Path != "~/wt/a" {
		t.Fatalf("peer = %+v, want worktree feat/a at ~/wt/a", p)
	}
	if len(p.Rows) != 1 || p.Rows[0].ID != "iss-100" || p.Rows[0].Kind != "open-there" || p.Rows[0].Folder != "open" {
		t.Fatalf("rows = %+v", p.Rows)
	}
}

// Criteria 9 and 10: with no peers the command reports none and exits 0, and
// the board carries no peers line; with a non-empty diff the board carries one
// line naming the live-peer count and the id count.
func TestTheBoardCarriesAPeersLineOnlyWhenPeersHoldSomething(t *testing.T) {
	home, repo := peerCheckout(t)

	if out := string(runCLI(t, "peers")); !strings.Contains(out, "no peers") {
		t.Fatalf("a checkout with no peers must say so:\n%s", out)
	}
	if out := string(runCLI(t)); strings.Contains(out, "peers:") {
		t.Fatalf("the board carries a peers line with no peers:\n%s", out)
	}
	var board map[string]any
	if err := json.Unmarshal(runCLI(t, "--json"), &board); err != nil {
		t.Fatal(err)
	}
	if _, ok := board["peers"]; ok {
		t.Fatalf("the board's --json carries a peers member with no peers: %v", board)
	}

	addPeer(t, home, repo, "a")
	out := string(runCLI(t))
	if !strings.Contains(out, "peers:") || !strings.Contains(out, "1 record") || !strings.Contains(out, "1 live peer") {
		t.Fatalf("the board lacks the peers line (live-peer and id counts):\n%s", out)
	}
	board = nil
	if err := json.Unmarshal(runCLI(t, "--json"), &board); err != nil {
		t.Fatal(err)
	}
	pm, ok := board["peers"].(map[string]any)
	if !ok || pm["live"] != float64(1) || pm["ids"] != float64(1) {
		t.Fatalf("board peers member = %v, want live 1, ids 1", board["peers"])
	}
}

// Criterion 5: capture resolve, the record dispatcher and intent audit, asked
// for a record this checkout lacks and a peer holds, refuse naming the peer's
// branch, path and folder instead of answering not found.
func TestNotFoundPathsNameThePeerThatHoldsTheRecord(t *testing.T) {
	home, repo := peerCheckout(t)
	addPeer(t, home, repo, "a")
	// A branch checked out nowhere ships an intent this checkout has never seen.
	gitCmd(t, repo, "switch", "-q", "-c", "side")
	writeRel(t, repo, ".abcd/development/intents/shipped/itd-77-shipped-elsewhere.md", peerIntent("itd-77", "Shipped elsewhere"))
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "ship itd-77 on a branch")
	gitCmd(t, repo, "switch", "-q", "main")

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"capture resolve", []string{"capture", "resolve", "iss-100", "fixed", "--impact", "fix"}, []string{"feat/a", "~/wt/a", "open/"}},
		{"record dispatcher", []string{"iss-100"}, []string{"feat/a", "~/wt/a", "open/"}},
		{"intent audit", []string{"intent", "audit", "itd-77"}, []string{"side", "shipped/"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runCLIErr(t, c.args...)
			if err == nil {
				t.Fatalf("%v succeeded for a record this checkout lacks:\n%s", c.args, out)
			}
			msg := scrubPaths(err)
			for _, w := range c.want {
				if !strings.Contains(msg, w) {
					t.Errorf("refusal lacks %q: %s", w, msg)
				}
			}
			if strings.Contains(msg, "not found") {
				t.Errorf("refusal still answers not found: %s", msg)
			}
			noHomePath(t, home, err.Error())
		})
	}

	// The peer consult runs only on the not-found path: a record this checkout
	// holds describes as it always has.
	if out := string(runCLI(t, "iss-1")); !strings.Contains(out, "The first finding") {
		t.Fatalf("a local record no longer describes:\n%s", out)
	}
	if m, _ := filepath.Glob(filepath.Join(repo, ".abcd/work/issues/*/iss-100-*")); len(m) != 0 {
		t.Fatalf("the refused resolve wrote the peer's record here: %v", m)
	}
}

// A peer's branch name and worktree path are another checkout's bytes, and the
// not-found refusal interpolates both into the one line cli.Run writes to stderr
// and into the --json error envelope. Git accepts a UTF-8 C1 control (U+009B,
// the 8-bit CSI) and a bidi override (U+202E) in a refname and a directory name,
// so either would reach the terminal raw unless the refusal sanitises them. The
// runes are written numerically so this file carries none of them.
func TestThePeerHeldRefusalSanitisesThePeersBranchAndPath(t *testing.T) {
	home, repo := peerCheckout(t)
	hostile := "evil" + string(rune(0x9b)) + "31m" + string(rune(0x202e)) + "x"
	dir := filepath.Join(home, "wt", hostile)
	gitCmd(t, repo, "worktree", "add", "-q", "-b", "feat/"+hostile, dir, "main")
	writeRel(t, dir, ".abcd/work/issues/open/iss-100-a-peer-finding.md", peerIssue("iss-100", "a-peer-finding", "A finding the peer captured"))

	bad := []string{string(rune(0x9b)), string(rune(0x202e))}
	check := func(surface, s string) {
		t.Helper()
		if !strings.Contains(s, "a peer holds it") {
			t.Fatalf("%s is not the peer-held refusal:\n%q", surface, s)
		}
		for _, r := range bad {
			if strings.Contains(s, r) {
				t.Errorf("%s carries the raw rune %U from the peer's branch or path:\n%q", surface, []rune(r)[0], s)
			}
		}
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"iss-100"}, &stdout, &stderr); code == 0 {
		t.Fatalf("the dispatcher succeeded for a record this checkout lacks:\n%s", stdout.String())
	}
	check("the stderr refusal", stderr.String())

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"capture", "resolve", "iss-100", "fixed", "--impact", "fix", "--json"}, &stdout, &stderr); code == 0 {
		t.Fatalf("the resolve succeeded for a record this checkout lacks:\n%s", stdout.String())
	}
	var env errorEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("the --json refusal is not an envelope: %v\n%s", err, stdout.String())
	}
	check("the --json error envelope", env.Error)
}

// Criterion 13: AGENTS.md's scan-before-mutating step names the peer listing
// beside the harness's session listing, so the convention points at a surface
// abcd renders rather than at nothing.
func TestTheScanBeforeMutatingConventionNamesThePeerListing(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, step, ok := strings.Cut(string(body), "- **Scan before mutating git state.**")
	if !ok {
		t.Fatal("AGENTS.md has no scan-before-mutating step")
	}
	step, _, _ = strings.Cut(step, "\n- **")
	for _, want := range []string{"session listing", "abcd peers"} {
		if !strings.Contains(step, want) {
			t.Errorf("the scan-before-mutating step does not name %q:\n%s", want, step)
		}
	}
}
