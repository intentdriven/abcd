package lint_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The release-gate manifest pins the inputs of the iss-35 brief<->surface
// crosscheck: a receipt echoes its sha256 as manifestHash and receipt_gate
// refuses a mismatch. Pinning the inputs makes two runs comparable in scope to
// each other (never in findings, which LLM checkers do not reproduce —
// iss-2608231409595789); it does not make them comparable to the tree, and twice the
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
//
// The manifest also stores nothing a surface addition would rewrite on a line
// every other such branch rewrites too (iss-2610150526192138): the checker
// count is computed rather than stored, the command and agent rosters are
// sorted one-per-line arrays the prompt names through placeholders (so the
// prompt text and its hash stay put when a surface ships), and briefDocs is
// ordered by chapter slug, so two branches adding chapters insert at different
// lines. manifestShapeProblems holds those properties; this test holds the
// lists to the tree as well.
func TestReleaseGateManifestIsCurrent(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const manifestRel = ".abcd/development/release-gate/manifest.json"
	raw := readRepoFile(t, root, manifestRel)
	for _, p := range manifestShapeProblems([]byte(raw)) {
		t.Error(p)
	}
	m, err := parseReleaseGateManifest([]byte(raw))
	if err != nil {
		t.Fatalf("parse %s: %v", manifestRel, err)
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

	// The pinned context tells every checker what the shipped surface IS: the
	// detector fills its ${commands} and ${agents} placeholders from these two
	// rosters, so each must be the roster on disk — fewer, and a checker
	// working from the context alone cannot find the missing ones; more, and it
	// hunts for a surface that does not ship.
	assertRoster(t, "commands", m.Commands, listMarkdown(t, root, "commands", "README"))
	assertRoster(t, "agents", m.Agents, listMarkdown(t, root, "agents", "README", "CHANGELOG"))

	// The detector is what turns the placeholders into names. A manifest that
	// moved the rosters out of the context while the script still sent the
	// context unfilled would hand every checker a literal ${commands}.
	js := readRepoFile(t, root, ".abcd/development/release-gate/brief-surface-crosscheck.js")
	for _, want := range []string{"commands: input.commands.join(', ')", "agents: input.agents.join(', ')"} {
		if !strings.Contains(js, want) {
			t.Errorf("brief-surface-crosscheck.js does not fill the context's placeholders from the manifest's rosters (no %q)", want)
		}
	}
}

// releaseGateManifest is the part of the release-gate manifest these tests read.
type releaseGateManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	BriefDocs     []string          `json:"briefDocs"`
	Commands      []string          `json:"commands"`
	Agents        []string          `json:"agents"`
	Surfaces      []json.RawMessage `json:"surfaces"`
	PromptHash    string            `json:"promptHash"`
	Prompt        struct {
		Context    string `json:"context"`
		DirectionA string `json:"directionA"`
		DirectionB string `json:"directionB"`
	} `json:"prompt"`
}

func parseReleaseGateManifest(raw []byte) (releaseGateManifest, error) {
	var m releaseGateManifest
	err := json.Unmarshal(raw, &m)
	return m, err
}

// releaseGatePromptHash is the manifest's promptHash definition: sha256 over
// the three prompt parts joined by blank lines — the algorithm the manifest's
// birth commit used, recovered by matching the committed value against the
// committed text. The rosters are not part of it: like ${doc} and ${s.name},
// ${commands} and ${agents} are inputs the detector substitutes, which the
// manifestHash a receipt echoes pins along with the rest of the file.
func releaseGatePromptHash(m releaseGateManifest) string {
	sum := sha256.Sum256([]byte(m.Prompt.Context + "\n\n" + m.Prompt.DirectionA + "\n\n" + m.Prompt.DirectionB))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// briefDocOrder orders briefDocs by directory, then by chapter slug, the file
// name with its leading "NN-" chapter number removed. Chapter numbers are
// handed out in sequence, so numeric order puts every new chapter on the same
// line and any two branches that add one conflict there; slug order puts a
// new chapter beside its neighbours by name, as the command roster does.
func briefDocOrder(a, b string) int {
	key := func(p string) string {
		dir, file := path.Split(p)
		if num, rest, ok := strings.Cut(file, "-"); ok && num != "" && strings.Trim(num, "0123456789") == "" {
			file = rest
		}
		return dir + file
	}
	if c := strings.Compare(key(a), key(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// manifestShapeProblems reports every way the manifest's bytes break the rules
// that keep it both self-consistent and mergeable, with no reference to the
// tree: schemaVersion 2, no stored checkerCount, sorted rosters with no
// duplicate, briefDocs in briefDocOrder, each roster placeholder present once
// in the context, and promptHash equal to its definition.
func manifestShapeProblems(raw []byte) []string {
	var out []string
	m, err := parseReleaseGateManifest(raw)
	if err != nil {
		return []string{"the manifest does not parse: " + err.Error()}
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return []string{"the manifest is not a JSON object: " + err.Error()}
	}

	// schemaVersion 2 is what arms the release gate's closed disposition
	// vocabulary for this tree: every finding a release receipt records is
	// fixed, invalid or deferred to its own open issue record. Lowering it
	// would disarm the deferred-finding discipline for every later release.
	if m.SchemaVersion != 2 {
		out = append(out, fmt.Sprintf("schemaVersion is %d; the committed manifest is schemaVersion 2, which arms the closed disposition vocabulary", m.SchemaVersion))
	}

	// The checker count is one checker per brief doc (Direction A) plus one per
	// surface (Direction B). It is computed from the two lists, never stored:
	// nothing consumes a stored copy, which is why it drifted once
	// (iss-2609011423385217), and a stored copy is rewritten on one line by
	// every branch that adds a chapter (iss-2610150526192138).
	if _, ok := keys["checkerCount"]; ok {
		out = append(out, fmt.Sprintf("the manifest stores checkerCount; the count is computed, briefDocs (%d) + surfaces (%d) = %d, and a stored copy goes silently wrong when two branches that each add a chapter merge",
			len(m.BriefDocs), len(m.Surfaces), len(m.BriefDocs)+len(m.Surfaces)))
	}

	for _, roster := range []struct {
		name  string
		items []string
	}{{"commands", m.Commands}, {"agents", m.Agents}} {
		if len(roster.items) == 0 {
			out = append(out, "the manifest has no "+roster.name+" roster")
		}
		for i := 1; i < len(roster.items); i++ {
			if roster.items[i-1] >= roster.items[i] {
				out = append(out, fmt.Sprintf("%s is not strictly sorted: %q then %q; keep it sorted so two branches adding different names insert at different lines", roster.name, roster.items[i-1], roster.items[i]))
			}
		}
		if n := strings.Count(m.Prompt.Context, "${"+roster.name+"}"); n != 1 {
			out = append(out, fmt.Sprintf("the pinned context names ${%s} %d times; it names the roster through that one placeholder, never inline", roster.name, n))
		}
	}
	for i := 1; i < len(m.BriefDocs); i++ {
		if briefDocOrder(m.BriefDocs[i-1], m.BriefDocs[i]) >= 0 {
			out = append(out, fmt.Sprintf("briefDocs is not in chapter-slug order: %s then %s", m.BriefDocs[i-1], m.BriefDocs[i]))
		}
	}

	// promptHash is pinned so a prompt edit that forgets the hash is caught,
	// rather than deferred as unknowable. Receipts echo it as policy.promptHash.
	if want := releaseGatePromptHash(m); m.PromptHash != want {
		out = append(out, fmt.Sprintf("promptHash is %s; sha256(context + \"\\n\\n\" + directionA + \"\\n\\n\" + directionB) is %s", m.PromptHash, want))
	}
	return out
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

func assertRoster(t *testing.T, what string, pinned, onDisk []string) {
	t.Helper()
	if slices.Equal(pinned, onDisk) {
		return
	}
	for _, name := range onDisk {
		if !slices.Contains(pinned, name) {
			t.Errorf("%s/%s.md ships but the manifest's %s roster does not name it", what, name, what)
		}
	}
	for _, name := range pinned {
		if !slices.Contains(onDisk, name) {
			t.Errorf("the manifest's %s roster names %s/%s.md, which does not ship", what, what, name)
		}
	}
}
