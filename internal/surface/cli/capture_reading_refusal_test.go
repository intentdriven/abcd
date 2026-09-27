package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadingLedgerRefusalsExit2 holds the reading-ledger verbs — disposition,
// admit, surprise and reframe — and the mentions listing to the rule the issue
// verbs keep (iss-2609260552251398): a refusal of the verb's own input exits 2
// with nothing written, naming the verb once. Every case below exited 1, the
// code a fault takes, while the capture page said "every ledger verb".
func TestReadingLedgerRefusalsExit2(t *testing.T) {
	repo := captureLedgerRepo(t)
	// A HEAD, so an occasion no commit added is refused as that, not as a
	// history that cannot be read.
	gitCommitAt(t, repo, "an empty history to walk")
	writeWideningFixture(t, repo, false)
	const detectionRun, detectionItem = "rdg-2608300000000003", "rdi-2608300000000004"
	writeReadingFixture(t, repo, detectionRun, detectionItem)
	const ground = "the configuration engages the frame the reading characterised"
	unknownItem := "rdi-2608309999999999"

	cases := [][]string{
		// disposition: a malformed id, an item the ledger does not hold, a state
		// outside the vocabulary, a --recurs token outside its shape, and a
		// --supersedes naming no standing answer.
		{"capture", "disposition", "rdi-abc", "--state", "rejected", "--grounds", ground},
		{"capture", "disposition", unknownItem, "--state", "rejected", "--grounds", ground},
		{"capture", "disposition", detectionItem, "--state", "bogus", "--grounds", ground},
		{"capture", "disposition", detectionItem, "--state", "rejected", "--grounds", ground, "--recurs", "not-an-item"},
		{"capture", "disposition", detectionItem, "--state", "rejected", "--grounds", ground, "--supersedes", "dsp-2608309999999999"},
		// admit: a malformed id, an unknown item, a degenerate ground, an item
		// at another position, and an admission before characterisation.
		{"capture", "admit", "rdi-abc", "--grounds", ground},
		{"capture", "admit", unknownItem, "--grounds", ground},
		{"capture", "admit", admitItem, "--grounds", "x"},
		{"capture", "admit", detectionItem, "--grounds", ground},
		{"capture", "admit", admitItem, "--grounds", ground},
		// surprise: an occasion that is prose, one naming nothing, and a
		// degenerate text.
		{"capture", "surprise", "--occasioned-by", "a hunch", "the reading ranked the unexpected proposal first"},
		{"capture", "surprise", "--occasioned-by", unknownItem, "the reading ranked the unexpected proposal first"},
		{"capture", "surprise", "--occasioned-by", admitItem, "x"},
		// reframe: an occasion that is prose, one naming nothing, an occasion
		// not yet committed, and a --complete id malformed or naming nothing.
		{"capture", "reframe", "--occasioned-by", "a hunch", "--grounds", ground},
		{"capture", "reframe", "--occasioned-by", unknownItem, "--grounds", ground},
		{"capture", "reframe", "--occasioned-by", admitItem, "--grounds", ground, "--open"},
		{"capture", "reframe", "--complete", "rfm-abc"},
		{"capture", "reframe", "--complete", "rfm-2608309999999999"},
		// mentions: a --ref naming no commit.
		{"capture", "mentions", "--ref", "no-such-branch"},
	}
	for _, args := range cases {
		out, err := runCLIErr(t, args...)
		if exitCodeOf(err) != 2 {
			t.Errorf("%v: exit = %d (%v), want 2\n%s", args[1:], exitCodeOf(err), err, out)
			continue
		}
		if !strings.HasPrefix(err.Error(), "abcd capture "+args[1]+": ") ||
			strings.Contains(err.Error(), ": capture "+args[1]+": ") || strings.Contains(err.Error(), ": "+args[1]+": ") {
			t.Errorf("%v: the refusal does not name its verb once: %v", args[1:], err)
		}
	}
	for _, store := range []string{"dispositions", "admissions", "surprises", "reframes"} {
		if _, err := os.Stat(filepath.Join(repo, ".abcd", "work", "issues", store)); !os.IsNotExist(err) {
			t.Errorf("a refused request wrote into %s: %v", store, err)
		}
	}
}

// TestReadingLedgerStandingRefusalsExit2 covers the refusals a standing answer
// occasions: a second answer that does not cite the first, a second admission,
// and an admission over an answer in another state. Each is the reading ledger's
// transition conflict — the request does not fit the record as it stands — and
// exits 2 as the issue verbs' conflict does.
func TestReadingLedgerStandingRefusalsExit2(t *testing.T) {
	repo := captureLedgerRepo(t)
	writeWideningFixture(t, repo, true)
	const detectionRun, detectionItem = "rdg-2608300000000003", "rdi-2608300000000004"
	writeReadingFixture(t, repo, detectionRun, detectionItem)
	const ground = "the configuration engages the frame the reading characterised"

	runCLI(t, "capture", "disposition", detectionItem, "--state", "rejected", "--grounds", ground)
	runCLI(t, "capture", "admit", admitItem, "--grounds", ground)

	for _, args := range [][]string{
		{"capture", "disposition", detectionItem, "--state", "rejected", "--grounds", ground},
		{"capture", "admit", admitItem, "--grounds", ground},
	} {
		out, err := runCLIErr(t, args...)
		if exitCodeOf(err) != 2 || !strings.HasPrefix(err.Error(), "abcd capture "+args[1]+": ") {
			t.Errorf("%v: exit = %d (%v), want 2 naming the verb\n%s", args[1:], exitCodeOf(err), err, out)
		}
	}

	// An admission over a standing answer in another state.
	repo2 := captureLedgerRepo(t)
	writeWideningFixture(t, repo2, true)
	runCLI(t, "capture", "disposition", admitItem, "--state", "declined", "--grounds", ground)
	if out, err := runCLIErr(t, "capture", "admit", admitItem, "--grounds", ground); exitCodeOf(err) != 2 {
		t.Errorf("admit over a declined answer: exit = %d (%v), want 2\n%s", exitCodeOf(err), err, out)
	}
}

// TestCaptureMigrateFaultExits1: `capture migrate` takes no input it could
// refuse, so every failure it meets is a fault and takes a fault's exit 1. It
// mapped every error to exit 2, so a record it could not read looked, to a
// script, like a request the caller should fix.
func TestCaptureMigrateFaultExits1(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a file whatever its mode")
	}
	repo := captureLedgerRepo(t)
	var m struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(runCLI(t, "capture", "an observation migrate cannot read", "--json"), &m); err != nil || m.ID == "" {
		t.Fatalf("capture envelope unreadable: %v", err)
	}
	paths, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", m.ID+"-*.md"))
	if len(paths) != 1 {
		t.Fatalf("captured record not found: %v", paths)
	}
	if err := os.Chmod(paths[0], 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(paths[0], 0o644) })
	if out, err := runCLIErr(t, "capture", "migrate"); exitCodeOf(err) != 1 {
		t.Errorf("capture migrate over an unreadable record: exit = %d (%v), want 1\n%s", exitCodeOf(err), err, out)
	}
}
