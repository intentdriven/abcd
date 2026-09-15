package lint_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The release-gate manifest is the reproducibility anchor for the iss-35
// brief<->surface crosscheck: a receipt echoes its sha256 as manifestHash and
// receipt_gate refuses a mismatch. Pinning the inputs makes two runs comparable
// to each other; it does not make them comparable to the tree, and twice the
// pinned inputs were found describing a surface that had moved on
// (iss-387: five shipped chapters unpinned; iss-2609011423385217: the pinned
// context naming 10 of 22 command pages and no agents, the scope omitting the
// chapters where six of thirteen findings lived, checkerCount disagreeing with
// the lists, and promptHash left over from a prompt text edited three times
// since). In each case a full-tier receipt attested coverage the run could not
// have had. Nothing read the manifest against the tree, so nothing noticed.
//
// This test is that reader. Every value the manifest could derive from the tree
// is held to the tree, and every value it states about itself is held to its
// own contents. A surface that ships without joining the pin, or a prompt edit
// that leaves the hash behind, fails here rather than in a release receipt.
func TestReleaseGateManifestIsCurrent(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const manifestRel = ".abcd/development/release-gate/manifest.json"

	var m struct {
		BriefDocs    []string          `json:"briefDocs"`
		Surfaces     []json.RawMessage `json:"surfaces"`
		CheckerCount int               `json:"checkerCount"`
		PromptHash   string            `json:"promptHash"`
		Prompt       struct {
			Context    string `json:"context"`
			DirectionA string `json:"directionA"`
			DirectionB string `json:"directionB"`
		} `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, manifestRel)), &m); err != nil {
		t.Fatalf("parse %s: %v", manifestRel, err)
	}

	// checkerCount is one checker per brief doc (Direction A) plus one per
	// surface (Direction B). The field is informational to the detector, which
	// is exactly why it drifted: nothing consumed it.
	if want := len(m.BriefDocs) + len(m.Surfaces); m.CheckerCount != want {
		t.Errorf("checkerCount is %d; briefDocs (%d) + surfaces (%d) = %d", m.CheckerCount, len(m.BriefDocs), len(m.Surfaces), want)
	}

	// Every pinned brief doc exists, and every chapter under 04-surfaces/ is
	// pinned: a surface chapter is where a verb's claims canonically live, so a
	// new chapter is in scope by existing (iss-387's shape).
	for _, doc := range m.BriefDocs {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(doc))); err != nil {
			t.Errorf("briefDocs pins %s, which is not on disk: %v", doc, err)
		}
	}
	for _, chapter := range listMarkdown(t, root, ".abcd/development/brief/04-surfaces") {
		rel := ".abcd/development/brief/04-surfaces/" + chapter + ".md"
		if !slices.Contains(m.BriefDocs, rel) {
			t.Errorf("surface chapter %s is on disk but not pinned in briefDocs", rel)
		}
	}

	// The pinned context tells every checker what the shipped surface IS. It
	// enumerates the command pages and the agent prompts by name, so the roster
	// it states must be the roster on disk — fewer, and a checker working from
	// the context alone cannot find the missing ones; more, and it hunts for a
	// surface that does not ship.
	commands := listMarkdown(t, root, "commands", "README")
	agents := listMarkdown(t, root, "agents", "README", "CHANGELOG")
	assertRoster(t, "commands", pinnedRoster(t, m.Prompt.Context, "commands/ ("), commands)
	assertRoster(t, "agents", pinnedRoster(t, m.Prompt.Context, "agents/ ("), agents)

	// promptHash is sha256 over the three prompt parts joined by blank lines —
	// the algorithm the manifest's birth commit used, recovered by matching the
	// committed value against the committed text. It is pinned here so a prompt
	// edit that forgets the hash is caught, rather than deferred as unknowable.
	sum := sha256.Sum256([]byte(m.Prompt.Context + "\n\n" + m.Prompt.DirectionA + "\n\n" + m.Prompt.DirectionB))
	if want := "sha256:" + hex.EncodeToString(sum[:]); m.PromptHash != want {
		t.Errorf("promptHash is %s; sha256(context + \"\\n\\n\" + directionA + \"\\n\\n\" + directionB) is %s", m.PromptHash, want)
	}
}

// listMarkdown returns the basenames (extension stripped, sorted) of the *.md
// files directly under dir, minus the named exclusions.
func listMarkdown(t *testing.T, root, dir string, exclude ...string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		t.Fatalf("list %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if !ok || e.IsDir() || slices.Contains(exclude, name) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// pinnedRoster extracts the comma-separated, parenthesised name list that
// follows marker in the pinned context, e.g. "commands/ (a, b,\nc)". The list
// wraps across lines in the manifest, so every item is whitespace-trimmed.
func pinnedRoster(t *testing.T, context, marker string) []string {
	t.Helper()
	_, after, ok := strings.Cut(context, marker)
	if !ok {
		t.Fatalf("pinned context does not enumerate %q", strings.TrimSuffix(marker, " ("))
	}
	list, _, ok := strings.Cut(after, ")")
	if !ok {
		t.Fatalf("pinned context's %q list is not closed", marker)
	}
	var out []string
	for _, item := range strings.Split(list, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func assertRoster(t *testing.T, what string, pinned, onDisk []string) {
	t.Helper()
	if slices.Equal(pinned, onDisk) {
		return
	}
	for _, name := range onDisk {
		if !slices.Contains(pinned, name) {
			t.Errorf("%s/%s.md ships but the pinned context does not name it", what, name)
		}
	}
	for _, name := range pinned {
		if !slices.Contains(onDisk, name) {
			t.Errorf("the pinned context names %s/%s.md, which does not ship", what, name)
		}
	}
}
