package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSyncTemplatePinsPropagatesABump is the case this whole file exists for: a
// dependabot PR bumps a pinned action in the COMMITTED workflow and never touches
// the template it was rendered from, so TestSelfScaffoldParity fails and the PR
// can never go green on its own (iss-209). Propagating the bump back into the
// template is what restores parity.
func TestSyncTemplatePinsPropagatesABump(t *testing.T) {
	workflow := strings.Join([]string{
		"      - uses: actions/checkout@aaaa111 # v7.0.1",
		"        uses: actions/attest@NEWSHA # v4.2.2",
		"        uses: actions/attest-build-provenance@cccc333 # v4.1.1",
	}, "\n")
	tmpl := strings.Join([]string{
		"      - uses: actions/checkout@aaaa111 # v7.0.1",
		"        uses: actions/attest@OLDSHA # v4.2.0",
		"        uses: actions/attest-build-provenance@cccc333 # v4.1.1",
	}, "\n")

	got, changed, err := syncTemplatePins([]byte(workflow), []byte(tmpl))
	if err != nil {
		t.Fatalf("syncTemplatePins: %v", err)
	}
	if !changed {
		t.Fatal("syncTemplatePins reported no change; the attest pin differs and must be propagated")
	}
	if string(got) != workflow {
		t.Errorf("template not synced to the workflow pins:\ngot:\n%s\nwant:\n%s", got, workflow)
	}
}

// TestSyncTemplatePinsDoesNotConfuseAPrefix guards the one collision this
// line-matching could plausibly get wrong: actions/attest is a strict prefix of
// actions/attest-build-provenance, so a prefix match would rewrite the longer
// action with the shorter one's ref. Keys are exact action names for this reason.
func TestSyncTemplatePinsDoesNotConfuseAPrefix(t *testing.T) {
	workflow := "        uses: actions/attest@NEWSHA # v4.2.2\n"
	tmpl := "        uses: actions/attest-build-provenance@KEEPME # v4.1.1\n"

	got, changed, err := syncTemplatePins([]byte(workflow), []byte(tmpl))
	if err != nil {
		t.Fatalf("syncTemplatePins: %v", err)
	}
	if changed {
		t.Error("syncTemplatePins rewrote actions/attest-build-provenance from the actions/attest pin")
	}
	if string(got) != tmpl {
		t.Errorf("prefix collision rewrote the template:\ngot:  %q\nwant: %q", got, tmpl)
	}
}

// TestSyncTemplatePinsPreservesUntrackedActions: a template line whose action the
// workflow does not pin is left exactly as it stands. The bare-profile rendering
// guards regions the abcd workflow never emits, so "absent from the workflow" is
// a normal state and must never be read as "delete" or "blank".
func TestSyncTemplatePinsPreservesUntrackedActions(t *testing.T) {
	workflow := "        uses: actions/checkout@aaaa111 # v7.0.1\n"
	tmpl := strings.Join([]string{
		"        uses: actions/checkout@aaaa111 # v7.0.1",
		"        uses: some/other-action@dddd444 # v1.2.3",
		"",
	}, "\n")

	got, changed, err := syncTemplatePins([]byte(workflow), []byte(tmpl))
	if err != nil {
		t.Fatalf("syncTemplatePins: %v", err)
	}
	if changed {
		t.Error("syncTemplatePins reported a change when only an untracked action was present")
	}
	if string(got) != tmpl {
		t.Errorf("untracked action was not preserved:\ngot:\n%s\nwant:\n%s", got, tmpl)
	}
}

// TestSyncTemplatePinsRefusesAmbiguousWorkflow: if the workflow pins one action to
// two different refs there is no single answer to propagate, and guessing would
// silently pick one. Refuse instead — a wrong pin in a release workflow is exactly
// the failure the SHA pinning exists to prevent.
func TestSyncTemplatePinsRefusesAmbiguousWorkflow(t *testing.T) {
	workflow := strings.Join([]string{
		"        uses: actions/checkout@aaaa111 # v7.0.1",
		"        uses: actions/checkout@bbbb222 # v7.0.2",
		"",
	}, "\n")
	tmpl := "        uses: actions/checkout@aaaa111 # v7.0.1\n"

	if _, _, err := syncTemplatePins([]byte(workflow), []byte(tmpl)); err == nil {
		t.Fatal("syncTemplatePins accepted a workflow pinning actions/checkout to two refs; want a refusal")
	}
}

// TestSyncTemplatePinsPreservesLineEndingsAndIndent: the propagation rewrites only
// the ref, so indentation, the `- ` list marker, and a trailing newline survive.
// TestSelfScaffoldParity compares BYTES, so a stray whitespace change here would
// break the very parity this tool exists to restore.
func TestSyncTemplatePinsPreservesShape(t *testing.T) {
	workflow := "      - uses: actions/checkout@NEWSHA # v7.0.2\n"
	tmpl := "      - uses: actions/checkout@OLDSHA # v7.0.1\n"

	got, changed, err := syncTemplatePins([]byte(workflow), []byte(tmpl))
	if err != nil {
		t.Fatalf("syncTemplatePins: %v", err)
	}
	if !changed {
		t.Fatal("expected the checkout pin to be propagated")
	}
	if string(got) != workflow {
		t.Errorf("line shape not preserved:\ngot:  %q\nwant: %q", got, workflow)
	}
}

// TestSyncRepoPinsIsCleanToday is the standing guarantee: on a tree where nobody
// has bumped a workflow pin, running the sync changes nothing. It is the same
// invariant TestSelfScaffoldParity asserts, read from the other end — if this
// fails, a pin was bumped in a workflow without the template following.
func TestSyncRepoPinsIsCleanToday(t *testing.T) {
	root := repoRoot(t)
	changed, err := SyncRepoPins(root, false)
	if err != nil {
		t.Fatalf("SyncRepoPins: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("templates are out of sync with the committed workflows: %v\n"+
			"run `make scaffold-sync` to propagate the workflow pins into the templates", changed)
	}
}

// TestSyncRepoPinsPropagatesABumpIntoTheTemplate runs against a COPY of the real
// tree: simulate exactly what dependabot does — bump a pin in the committed
// workflow only — and show the sync carries it into the template, touching that
// one line and nothing else.
//
// Deliberately NOT named for parity, and it does not assert it: Render reads the
// EMBEDDED templatesFS, not this temp-dir copy, so no test operating on a copy can
// prove the rendered output matches. Byte-parity is TestSelfScaffoldParity's job,
// against the real embedded template; this proves the propagation that makes that
// test passable again, and the one-line bound below is what rules out the
// collateral edits a byte-comparison would otherwise have caught.
func TestSyncRepoPinsPropagatesABumpIntoTheTemplate(t *testing.T) {
	root := t.TempDir()
	realRoot := repoRoot(t)

	for _, rel := range []string{ReleaseYMLPath, AutoReleaseYMLPath} {
		copyInto(t, realRoot, root, rel)
	}
	for _, rel := range []string{releaseTemplateRel, autoReleaseTemplateRel} {
		copyInto(t, realRoot, root, rel)
	}

	// Bump actions/attest in the workflow alone, the way dependabot would: every
	// occurrence at once, since the workflow calls it more than once (iss-273).
	//
	// The pin is READ from the tree rather than written here on purpose: hard-coding
	// the current SHA would make this test fail on the very first dependabot bump —
	// the exact brittleness this whole file exists to remove — and it did, once.
	wfPath := filepath.Join(root, filepath.FromSlash(ReleaseYMLPath))
	before := readFile(t, wfPath)
	attestRe := regexp.MustCompile(`actions/attest@\S+ # \S+`)
	current := attestRe.FindString(before)
	if current == "" {
		t.Fatal("release.yml no longer pins actions/attest; pick another action for this test")
	}
	const bumpedPin = "actions/attest@1111111111111111111111111111111111111111 # v9.9.9"
	occurrences := strings.Count(before, current)
	bumped := strings.ReplaceAll(before, current, bumpedPin)
	if bumped == before {
		t.Fatalf("failed to rewrite the actions/attest pin %q in release.yml", current)
	}
	if err := os.WriteFile(wfPath, []byte(bumped), 0o644); err != nil {
		t.Fatalf("write bumped workflow: %v", err)
	}

	// Dry run must SEE the drift without writing.
	seen, err := SyncRepoPins(root, false)
	if err != nil {
		t.Fatalf("SyncRepoPins(dry): %v", err)
	}
	if len(seen) != 1 || seen[0] != releaseTemplateRel {
		t.Fatalf("dry run reported %v; want exactly [%s]", seen, releaseTemplateRel)
	}
	if strings.Contains(readFile(t, filepath.Join(root, filepath.FromSlash(releaseTemplateRel))), "1111111111") {
		t.Fatal("the dry run wrote to the template; it must only report")
	}

	// Applying it propagates the bump.
	tmplBefore := readFile(t, filepath.Join(root, filepath.FromSlash(releaseTemplateRel)))
	if _, err := SyncRepoPins(root, true); err != nil {
		t.Fatalf("SyncRepoPins(write): %v", err)
	}
	tmplNow := readFile(t, filepath.Join(root, filepath.FromSlash(releaseTemplateRel)))

	// EXACTLY the attest pin lines move. TestSelfScaffoldParity compares bytes, so a
	// sync that also normalised whitespace, reordered a key or touched another pin
	// would leave the template unrenderable to the committed workflow — and this
	// test, working on a copy, cannot see that. Bounding the edit to the bumped
	// lines is the substitute for the byte-comparison it cannot make.
	if n := changedLineCount(t, tmplBefore, tmplNow); n != occurrences {
		t.Errorf("sync changed %d lines in the template; want exactly %d (the attest pins)", n, occurrences)
	}
	if !strings.Contains(tmplNow, bumpedPin) {
		t.Errorf("the bumped attest pin %q was not propagated into the template", bumpedPin)
	}
	// The sibling whose name actions/attest is a strict prefix of must be untouched,
	// read from the tree for the same reason the bump above is.
	siblingRe := regexp.MustCompile(`actions/attest-build-provenance@\S+ # \S+`)
	if sibling := siblingRe.FindString(before); sibling != "" && !strings.Contains(tmplNow, sibling) {
		t.Errorf("the attest-build-provenance pin %q was damaged by the attest propagation", sibling)
	}

	// And a second run is a no-op: the sync converges.
	again, err := SyncRepoPins(root, true)
	if err != nil {
		t.Fatalf("SyncRepoPins(second): %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second run reported %v; the sync must converge in one pass", again)
	}
}

func copyInto(t *testing.T, fromRoot, toRoot, rel string) {
	t.Helper()
	dst := filepath.Join(toRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	data, err := os.ReadFile(filepath.Join(fromRoot, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// changedLineCount reports how many lines differ between two same-length files,
// failing if the sync added or removed any.
func changedLineCount(t *testing.T, before, after string) int {
	t.Helper()
	bl, al := strings.Split(before, "\n"), strings.Split(after, "\n")
	if len(bl) != len(al) {
		t.Fatalf("sync changed the template's line count: %d -> %d", len(bl), len(al))
	}
	n := 0
	for i := range bl {
		if bl[i] != al[i] {
			n++
		}
	}
	return n
}
