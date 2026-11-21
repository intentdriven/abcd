package interview

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// repoFile is a file of this repository, read from the package's directory.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// shellBlock is the lines of src from the first line equal to open through
// the first line after it equal to close.
func shellBlock(t *testing.T, src, open, close string) string {
	t.Helper()
	lines := strings.Split(src, "\n")
	start := slices.Index(lines, open)
	if start < 0 {
		t.Fatalf("no line %q", open)
	}
	end := slices.Index(lines[start:], close)
	if end < 0 {
		t.Fatalf("no line %q after %q", close, open)
	}
	return strings.Join(lines[start:start+end+1], "\n")
}

// TestTheReceiptlessPrefixesMirrorThePrePushHook: the tripwire's reading of
// the pre-push gate counts a remote-tracking ref as the hook's on_a_remote
// does, so the prefixes it treats as receipt-less are the rows the hook's
// receiptless_prefixes table declares, and for every ref name the hook's own
// receiptless_tracking, run in bash with the rows the host connectors add,
// answers as receiptlessTracking does. The hook's other half, the receipt
// lookup, reads every worktree git lists and a receipt that is a file once
// links are followed, as pushGate does.
func TestTheReceiptlessPrefixesMirrorThePrePushHook(t *testing.T) {
	hook := repoFile(t, ".githooks/pre-push")
	table := shellBlock(t, hook, "receiptless_prefixes=(", ")")
	var declared []string
	for _, m := range regexp.MustCompile(`(?m)^\s*"([^"|]*)\|`).FindAllStringSubmatch(table, -1) {
		declared = append(declared, m[1])
	}
	if !slices.Equal(declared, receiptlessPrefixes) {
		t.Fatalf("the hook declares the receipt-less prefixes %q, the tripwire mirrors %q", declared, receiptlessPrefixes)
	}
	connector := repoFile(t, "scripts/cloud/connector.sh")
	if !strings.Contains(connector, "WIP_PREFIX_RE='^([a-z0-9][a-z0-9-]*/)?"+wipSegment+"/$'") {
		t.Fatalf("scripts/cloud/connector.sh no longer holds every connector's prefix to one ending in the segment %s, which the tripwire relies on", wipSegment)
	}
	for _, want := range []string{
		`--contains "$1" refs/remotes`,
		`receiptless_tracking "${ref#refs/remotes/}" && continue`,
		`scripts/preflight-receipt.sh check "$commit"`,
	} {
		if !strings.Contains(hook, want) {
			t.Fatalf("the hook no longer holds %q, which the tripwire's reading of the gate mirrors", want)
		}
	}
	receipt := repoFile(t, "scripts/preflight-receipt.sh")
	for _, want := range []string{`[ -f "$1/$receipts_rel/$commit" ] || return 1`, `git worktree list --porcelain`, `receipts_rel=".abcd/.work.local/preflight-receipts"`} {
		if !strings.Contains(receipt, want) {
			t.Fatalf("scripts/preflight-receipt.sh no longer holds %q, which the tripwire's reading of the gate mirrors", want)
		}
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash to run the hook's own receiptless_tracking")
	}
	prefixes, err := exec.Command(bash, filepath.Join("..", "..", "..", "scripts", "cloud", "connector.sh"), "prefixes").Output()
	if err != nil {
		t.Fatal(err)
	}
	script := table + "\n"
	for _, p := range strings.Fields(string(prefixes)) {
		script += "receiptless_prefixes+=(\"" + p + "|wip_shape|adr|end\")\n"
	}
	script += shellBlock(t, hook, "lower() {", "}") + "\n" +
		shellBlock(t, hook, "receiptless_tracking() {", "}") + "\n" +
		"while IFS= read -r ref; do if receiptless_tracking \"$ref\"; then echo yes; else echo no; fi; done\n"
	refs := []string{
		"origin/main", "origin/feat/x", "origin/abcd-reserved/iss-1", "origin/ABCD-Reserved/iss-1",
		"origin/abcd-status/me", "origin/x/abcd-status/me", "origin/abcd-reservedx", "origin/xabcd-reserved/y",
		"abcd-reserved/x", "origin/claude/abcd-wip/x", "origin/other/abcd-wip/x", "origin/x/ABCD-WIP",
		"origin/abcd-wipe/x", "origin/abcd-wip", "origin/claude/x", "origin/claude/abcd-wipx",
	}
	cmd := exec.Command(bash, "-c", script)
	cmd.Stdin = strings.NewReader(strings.Join(refs, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the hook's receiptless_tracking did not run: %v", err)
	}
	got := strings.Fields(string(out))
	if len(got) != len(refs) {
		t.Fatalf("the hook answered %q for %d refs", got, len(refs))
	}
	for i, ref := range refs {
		if hookSays := got[i] == "yes"; hookSays != receiptlessTracking(ref) {
			t.Errorf("for %s the hook's receiptless_tracking says %v, receiptlessTracking %v", ref, hookSays, !hookSays)
		}
	}
}

// TestARefTheGateDoesNotCountLeavesTheInterviewRunning: what the tripwire
// watches of the remote-tracking refs is the gate's judgement, so a ref the
// gate counts for nothing, one under a receipt-less prefix, planted at HEAD
// during a turn leaves the interview running to its outcome; and once the
// gate already lets HEAD through before a turn (a receipt the person's own
// preflight minted), a ref planted during it opens nothing that was shut.
func TestARefTheGateDoesNotCountLeavesTheInterviewRunning(t *testing.T) {
	for _, c := range []struct {
		name string
		ref  string
		// receipted mints a receipt for HEAD in the checkout's own tier
		// before the run.
		receipted bool
	}{
		{name: "a reservation's tracking ref", ref: "refs/remotes/origin/abcd-reserved/iss-1"},
		{name: "a status branch's tracking ref", ref: "refs/remotes/origin/abcd-status/me"},
		{name: "a hosted session's work-in-progress ref", ref: "refs/remotes/origin/claude/abcd-wip/x"},
		{name: "a counted ref, the gate already open on a receipt", ref: "refs/remotes/origin/planted", receipted: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			head := r.git.Git("rev-parse", "HEAD")
			if c.receipted {
				r.git.Write(receiptsRel+"/"+head, "commit "+head+"\n")
			}
			stubAlso(t, script, 2, map[string]string{filepath.Join(r.repo, ".git", filepath.FromSlash(c.ref)): head + "\n"})
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want the interview run to its outcome", err)
			}
			if len(res.Changed) != 0 || len(r.asked) != 2 || len(r.finished) != 1 {
				t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
			}
		})
	}
}

// TestAPeerPackingTheRefsMidReadIsReadInThePackedRefs: a peer's git
// pack-refs can remove this checkout's loose branch ref between the reading
// finding it and opening it. git writes the packed refs before it prunes the
// loose ref, so the reading reads the branch there once, and holds it at the
// value it had, rather than refusing the reading.
func TestAPeerPackingTheRefsMidReadIsReadInThePackedRefs(t *testing.T) {
	r := newWrittenRun(t, "")
	r.git.Commit("base")
	head := r.git.Git("rev-parse", "HEAD")
	saved := readRef
	t.Cleanup(func() { readRef = saved })
	packed := false
	readRef = func(root *os.Root, rel string, n int64) ([]byte, error) {
		if !packed {
			packed = true
			r.git.Git("pack-refs", "--all")
			if _, err := os.Lstat(filepath.Join(r.repo, ".git", filepath.FromSlash(rel))); !os.IsNotExist(err) {
				t.Fatalf("the loose ref %s still stands after packing (%v); the case proves nothing", rel, err)
			}
		}
		return readAtMost(root, rel, n)
	}
	st, err := readTree(r.repo, "")
	if err != nil {
		t.Fatalf("a peer packing the refs mid-read refused the reading: %v", err)
	}
	if !packed {
		t.Fatal("the reading never read the loose ref; the case proves nothing")
	}
	if got, want := st[".git/refs/heads/main"], branchState+"refs/heads/main "+head; got != want {
		t.Fatalf("the branch reads %q, want %q", got, want)
	}
}
