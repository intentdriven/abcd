package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// prepass_test.go — the core of `abcd intent prepass` (itd-42,
// spc-2609211918551301). Each test is written from one of the intent's
// Given/When/Then criteria; the front door (the CLI verb and the plugin page's
// interview) is phase 2 and is exercised there.

const prepassInvariants = `# Invariants

## Properties the system must preserve regardless of how it's built

1. **Transparent prompts** — every prompt shows current state and how to change it later.

2. **Config stays home** — configuration is never written outside the
   machine-scoped home under ~/.abcd/, whatever the caller asks.

3. **Remotes are read freely** — a remote is written only on explicit request.
`

const prepassPrinciple = `# The user's directory is theirs

**The rule.** A tool creates files only in space that was handed to it, never
beside the user's projects.

**Why.** A directory is the user's map of their own work.
`

const prepassDraft = `---
id: itd-500
slug: shared-config
kind: null
spec_id: null
---

# Shared Config Lands Beside The Project

## Press Release

abcd writes the team's shared config file into the project directory, next to
the code, so everyone sees it.

## Acceptance Criteria

- **Given** a repo, **when** config is saved, **then** it lands in the project.
`

func prepassRecord(id, slug, title string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nkind: standalone\nspec_id: null\n---\n\n# " + title + "\n\nBody.\n"
}

// prepassRepo lays a small record: three invariants, two principles, the draft
// under test and a sibling on every shelf the index reads.
func prepassRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		invariantsRelPath: prepassInvariants,
		principlesRelDir + "/the-users-directory-is-theirs.md": prepassPrinciple,
		principlesRelDir + "/less-but-better.md":               "# Less but better\n\n**The rule.** Build the smallest thing that works.\n",
		principlesRelDir + "/README.md":                        "# Principles\n\nThe index.\n",
		IntentsRelDir + "/drafts/itd-500-shared-config.md":     prepassDraft,
		IntentsRelDir + "/drafts/itd-501-team-settings.md":     prepassRecord("itd-501", "team-settings", "Team Settings Are Shared Through The Repo"),
		IntentsRelDir + "/planned/itd-19-stage-aware.md":       prepassRecord("itd-19", "stage-aware", "Behaviour Is Stage-Aware"),
		IntentsRelDir + "/shipped/itd-7-home-store.md":         prepassRecord("itd-7", "home-store", "Every Store Lives Under The Home"),
		IntentsRelDir + "/superseded/itd-3-old-config.md":      prepassRecord("itd-3", "old-config", "The Old Config Draft"),
		IntentsRelDir + "/disciplines/itd-84-decomposition.md": prepassRecord("itd-84", "decomposition", "An Intent Is Decomposed Before It Is Filed"),
	}
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func prepassAssemble(t *testing.T, root string) PrepassInput {
	t.Helper()
	in, err := AssemblePrepass(root, "itd-500")
	if err != nil {
		t.Fatalf("AssemblePrepass: %v", err)
	}
	return in
}

// prepassFindings marshals a findings payload stamped for in.
func prepassFindings(t *testing.T, in PrepassInput, body map[string]any) []byte {
	t.Helper()
	payload := map[string]any{
		"_type":        PrepassFindingsType,
		"intent":       in.Intent,
		"input_digest": in.Digest,
	}
	for k, v := range body {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func prepassBrief(t *testing.T, root string, raw []byte) (PrepassBriefResult, string) {
	t.Helper()
	res, err := WritePrepassBrief(root, "itd-500", raw)
	if err != nil {
		t.Fatalf("WritePrepassBrief: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.BriefPath)))
	if err != nil {
		t.Fatalf("reading the brief: %v", err)
	}
	return res, string(data)
}

// prepassTree hashes every file under root, keyed by its slash path.
func prepassTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		sum := sha256.Sum256(data)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// AC3 (the read half), AC2 (the index): the input is the invariants, the
// principles, a one-line index of every intent on every shelf, and the draft,
// and nothing else.
func TestPrepassInputIsTheInvariantsThePrinciplesTheIndexAndTheDraft(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)

	if in.Type != PrepassInputType || in.Intent != "itd-500" {
		t.Fatalf("input header = %q %q", in.Type, in.Intent)
	}
	if in.DraftPath != IntentsRelDir+"/drafts/itd-500-shared-config.md" || in.Draft != prepassDraft {
		t.Fatalf("draft = %q (%d bytes), want the draft verbatim", in.DraftPath, len(in.Draft))
	}
	if len(in.Invariants) != 3 {
		t.Fatalf("invariants = %d, want 3: %+v", len(in.Invariants), in.Invariants)
	}
	inv2 := in.Invariants[1]
	if inv2.Number != 2 || inv2.Title != "Config stays home" ||
		!strings.Contains(collapseSpace(inv2.Text), "never written outside the machine-scoped home") {
		t.Fatalf("invariant 2 = %+v", inv2)
	}
	if len(in.Principles) != 2 {
		t.Fatalf("principles = %+v, want two (README is not a principle)", in.Principles)
	}
	for _, p := range in.Principles {
		if strings.Contains(p.Rule, "Why.") {
			t.Fatalf("principle %s carries more than its rule: %q", p.Path, p.Rule)
		}
	}
	got := map[string]string{}
	for _, e := range in.Index {
		got[e.ID] = e.Shelf + " | " + e.Title
	}
	want := map[string]string{
		"itd-501": "drafts | Team Settings Are Shared Through The Repo",
		"itd-19":  "planned | Behaviour Is Stage-Aware",
		"itd-7":   "shipped | Every Store Lives Under The Home",
		"itd-3":   "superseded | The Old Config Draft",
		"itd-84":  "disciplines | An Intent Is Decomposed Before It Is Filed",
	}
	if len(got) != len(want) {
		t.Fatalf("index = %v, want %v (the draft itself is not its own sibling)", got, want)
	}
	for id, line := range want {
		if got[id] != line {
			t.Fatalf("index[%s] = %q, want %q", id, got[id], line)
		}
	}
	if len(in.Answers) != 4 || strings.Join(in.Answers, ",") != "keep-both,bundle,supersede,refine" {
		t.Fatalf("answers = %v", in.Answers)
	}
	if !strings.HasPrefix(in.Digest, "sha256:") {
		t.Fatalf("digest = %q", in.Digest)
	}
}

// The pre-pass prepares the interview of a draft; a planned or shipped record
// has had its interview.
func TestPrepassRefusesARecordThatIsNotADraft(t *testing.T) {
	root := prepassRepo(t)
	_, err := AssemblePrepass(root, "itd-19")
	if err == nil || !strings.Contains(err.Error(), "planned") || !strings.Contains(err.Error(), "draft") {
		t.Fatalf("err = %v, want a refusal naming the shelf and drafts", err)
	}
	if _, err := AssemblePrepass(root, "itd-999"); err == nil {
		t.Fatal("an absent intent was assembled")
	}
	if _, err := AssemblePrepass(root, "../etc"); err == nil {
		t.Fatal("a malformed id was assembled")
	}
}

// AC1: the brief names every invariant the draft conflicts with, quoting the
// invariant's line and the draft's.
func TestPrepassBriefQuotesEveryInvariantConflict(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	raw := prepassFindings(t, in, map[string]any{
		"conflicts": []map[string]any{
			{
				"invariant":    2,
				"anchor_quote": "configuration is never written outside the machine-scoped home",
				"draft_quote":  "writes the team's shared config file into the project directory",
				"question":     "Invariant 2 keeps config under the home; the draft writes it beside the code. Which gives?",
			},
			{
				"invariant":    1,
				"anchor_quote": "every prompt shows current state",
				"draft_quote":  "so everyone sees it",
				"question":     "Does the draft's save prompt show where the file will land?",
			},
		},
	})
	res, brief := prepassBrief(t, root, raw)
	if res.Conflicts != 2 || res.Unanchored != 0 {
		t.Fatalf("result = %+v, want two conflicts", res)
	}
	for _, want := range []string{
		"Invariant 2 — Config stays home",
		"> configuration is never written outside the machine-scoped home",
		"> writes the team's shared config file into the project directory",
		"Invariant 1 — Transparent prompts",
		"> every prompt shows current state",
		"Which gives?",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief lacks %q:\n%s", want, brief)
		}
	}
}

// AC2: an overlap names the sibling from the index, is asked with the four
// answers, and carries the recommendation in the prose beside the question,
// never as a marked option.
func TestPrepassOverlapIsAskedWithTheFourAnswersAndTheRecommendationInProse(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	overlap := func(rec map[string]any) []byte {
		o := map[string]any{"sibling": "itd-501", "question": "itd-501 already shares team settings through the repo; how is this different?"}
		if rec != nil {
			o["recommendation"] = rec
		}
		return prepassFindings(t, in, map[string]any{"overlaps": []map[string]any{o}})
	}

	_, plain := prepassBrief(t, root, overlap(nil))
	res, recd := prepassBrief(t, root, overlap(map[string]any{"answer": "supersede", "reason": "the sibling's scope contains this draft's whole press release"}))
	if res.Overlaps != 1 {
		t.Fatalf("result = %+v, want one overlap", res)
	}
	for _, want := range []string{"itd-501", "Team Settings Are Shared Through The Repo", "(drafts)", "how is this different?"} {
		if !strings.Contains(recd, want) {
			t.Fatalf("brief lacks %q:\n%s", want, recd)
		}
	}
	options := func(brief string) string {
		var lines []string
		for _, l := range strings.Split(brief, "\n") {
			if strings.HasPrefix(l, "- **") {
				lines = append(lines, l)
			}
		}
		return strings.Join(lines, "\n")
	}
	want := "- **Keep both**\n- **Bundle**\n- **Supersede**\n- **Refine**"
	if !strings.Contains(options(recd), want) {
		t.Fatalf("options = %q, want the four answers in order:\n%s", options(recd), recd)
	}
	if options(plain) != options(recd) {
		t.Fatalf("the recommendation changed the options:\nplain %q\nrecd  %q", options(plain), options(recd))
	}
	if strings.Contains(strings.ToLower(options(recd)), "recommend") {
		t.Fatalf("an option is marked: %q", options(recd))
	}
	if !strings.Contains(recd, "The pre-pass leans towards **supersede**: the sibling's scope contains this draft's whole press release.") {
		t.Fatalf("the recommendation is not in the prose beside the question:\n%s", recd)
	}
	if strings.Contains(plain, "leans towards") {
		t.Fatalf("a brief with no recommendation carries one:\n%s", plain)
	}
}

// AC3: after the pre-pass, the draft, the brief and the siblings are unchanged,
// and the only file written is the planning brief under the local tier.
func TestPrepassWritesOnlyThePlanningBrief(t *testing.T) {
	root := prepassRepo(t)
	before := prepassTree(t, root)
	in := prepassAssemble(t, root)
	if after := prepassTree(t, root); len(after) != len(before) {
		t.Fatalf("assembling the input wrote files: %d -> %d", len(before), len(after))
	}
	res, _ := prepassBrief(t, root, prepassFindings(t, in, map[string]any{
		"unanchored": []map[string]any{{"question": "Who reads the shared file?"}},
	}))
	if res.BriefPath != PlanningBriefsRelDir+"/itd-500.md" {
		t.Fatalf("brief path = %q", res.BriefPath)
	}
	after := prepassTree(t, root)
	for rel, sum := range before {
		if after[rel] != sum {
			t.Fatalf("%s changed", rel)
		}
	}
	var added []string
	for rel := range after {
		if _, ok := before[rel]; !ok {
			added = append(added, rel)
		}
	}
	sort.Strings(added)
	if strings.Join(added, ",") != res.BriefPath {
		t.Fatalf("files written = %v, want only %s", added, res.BriefPath)
	}
}

// AC5: a concern the pre-pass cannot anchor is a question marked unanchored,
// not a finding — whether the host said so itself or named an anchor the record
// does not hold.
func TestPrepassUnanchoredConcernIsAQuestionMarkedUnanchored(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	raw := prepassFindings(t, in, map[string]any{
		"unanchored": []map[string]any{{"question": "Will a team of one ever need this?"}},
		"conflicts": []map[string]any{
			{"invariant": 9, "anchor_quote": "a ninth invariant nobody wrote", "draft_quote": "writes the team's shared config file", "question": "Invariant 9 forbids this?"},
			{"invariant": 3, "anchor_quote": "a remote is pushed on every save", "draft_quote": "writes the team's shared config file", "question": "Does the save push?"},
			{"invariant": 3, "anchor_quote": "a remote is written only on explicit request", "draft_quote": "the draft never says this at all", "question": "Is the file pushed?"},
			{"principle": principlesRelDir + "/no-such-principle.md", "anchor_quote": "a principle nobody wrote", "draft_quote": "writes the team's shared config file", "question": "Principle?"},
		},
		"overlaps": []map[string]any{{"sibling": "itd-999", "question": "itd-999 did this already?"}},
	})
	res, brief := prepassBrief(t, root, raw)
	if res.Conflicts != 0 || res.Overlaps != 0 || res.Unanchored != 6 || res.Demoted != 5 {
		t.Fatalf("result = %+v, want every concern unanchored and five demoted", res)
	}
	if strings.Contains(brief, "Conflict with") || strings.Contains(brief, "Overlap with") {
		t.Fatalf("an unanchorable concern was written as a finding:\n%s", brief)
	}
	if got := strings.Count(brief, "(unanchored)"); got != 6 {
		t.Fatalf("questions marked unanchored = %d, want 6:\n%s", got, brief)
	}
	for _, why := range []string{
		"the brief carries no invariant 9",
		"not found in invariant 3",
		"not found in the draft",
		"no principle at " + principlesRelDir + "/no-such-principle.md",
		"no intent on any shelf is itd-999",
	} {
		if !strings.Contains(brief, why) {
			t.Fatalf("brief does not say why a concern is unanchored (%q):\n%s", why, brief)
		}
	}
}

// A principle is a named record too: a conflict quoting one is anchored.
func TestPrepassPrincipleConflictIsAnchored(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	res, brief := prepassBrief(t, root, prepassFindings(t, in, map[string]any{
		"conflicts": []map[string]any{{
			"principle":    principlesRelDir + "/the-users-directory-is-theirs.md",
			"anchor_quote": "never beside the user's projects",
			"draft_quote":  "into the project directory, next to the code",
			"question":     "The principle keeps tools out of the project; which gives?",
		}},
	}))
	if res.Conflicts != 1 || res.Unanchored != 0 {
		t.Fatalf("result = %+v", res)
	}
	if !strings.Contains(brief, "Principle — The user's directory is theirs") {
		t.Fatalf("brief does not name the principle:\n%s", brief)
	}
}

// AC4 (the core half): each question is numbered and says where its answer
// lands on the record — a decision, or a typed link — so the interview can ask
// each one and record it.
func TestPrepassEveryQuestionSaysWhereItsAnswerLands(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	_, brief := prepassBrief(t, root, prepassFindings(t, in, map[string]any{
		"conflicts": []map[string]any{{
			"invariant": 2, "anchor_quote": "configuration is never written outside the machine-scoped home",
			"draft_quote": "into the project directory", "question": "Which gives?",
		}},
		"overlaps":   []map[string]any{{"sibling": "itd-19", "question": "How is this different?"}},
		"unanchored": []map[string]any{{"question": "Who reads it?"}},
	}))
	for i, q := range []string{"### Q1.", "### Q2.", "### Q3."} {
		at := strings.Index(brief, q)
		if at < 0 {
			t.Fatalf("question %d is not numbered:\n%s", i+1, brief)
		}
		rest := brief[at+len(q):]
		if next := strings.Index(rest, "### Q"); next >= 0 {
			rest = rest[:next]
		}
		if !strings.Contains(rest, "Lands as:") {
			t.Fatalf("Q%d does not say where its answer lands:\n%s", i+1, rest)
		}
	}
	for _, link := range []string{"superseded_by", "--bundle", "## Decisions", `--kind superseded --by <itd-M> --reason "<why>"`} {
		if !strings.Contains(brief, link) {
			t.Fatalf("the overlap's landing does not name %q:\n%s", link, brief)
		}
	}
}

// The findings are untrusted: a malformed payload is refused with nothing
// written.
func TestPrepassRefusesMalformedFindings(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	good := map[string]any{"unanchored": []map[string]any{{"question": "Q?"}}}
	cases := map[string][]byte{
		"not json":       []byte("{"),
		"trailing data":  append(prepassFindings(t, in, good), []byte(" {}")...),
		"unknown field":  prepassFindings(t, in, map[string]any{"verdict": "pass"}),
		"wrong type":     mustJSON(t, map[string]any{"_type": "abcd/other/v1", "intent": in.Intent, "input_digest": in.Digest}),
		"wrong intent":   mustJSON(t, map[string]any{"_type": PrepassFindingsType, "intent": "itd-501", "input_digest": in.Digest}),
		"stale digest":   mustJSON(t, map[string]any{"_type": PrepassFindingsType, "intent": in.Intent, "input_digest": "sha256:00"}),
		"blank question": prepassFindings(t, in, map[string]any{"unanchored": []map[string]any{{"question": "  "}}}),
		"bad answer":     prepassFindings(t, in, map[string]any{"overlaps": []map[string]any{{"sibling": "itd-19", "question": "Q?", "recommendation": map[string]any{"answer": "merge", "reason": "r"}}}}),
		"no reason":      prepassFindings(t, in, map[string]any{"overlaps": []map[string]any{{"sibling": "itd-19", "question": "Q?", "recommendation": map[string]any{"answer": "bundle"}}}}),
		"two anchors":    prepassFindings(t, in, map[string]any{"conflicts": []map[string]any{{"invariant": 1, "principle": "p", "anchor_quote": "x", "draft_quote": "y", "question": "Q?"}}}),
		"no anchor":      prepassFindings(t, in, map[string]any{"conflicts": []map[string]any{{"anchor_quote": "x", "draft_quote": "y", "question": "Q?"}}}),
	}
	for name, raw := range cases {
		if _, err := WritePrepassBrief(root, "itd-500", raw); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(PlanningBriefsRelDir))); !os.IsNotExist(err) {
		t.Fatalf("a refused payload wrote under the local tier (stat err %v)", err)
	}
	// The control: the same payload, well formed, is accepted.
	if _, err := WritePrepassBrief(root, "itd-500", prepassFindings(t, in, good)); err != nil {
		t.Fatalf("the well-formed control was refused: %v", err)
	}
}

// A record moved since the input was assembled refuses, rather than quoting text
// the host never read.
func TestPrepassRefusesFindingsOverAMovedRecord(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	raw := prepassFindings(t, in, map[string]any{"unanchored": []map[string]any{{"question": "Q?"}}})
	inv := filepath.Join(root, filepath.FromSlash(invariantsRelPath))
	if err := os.WriteFile(inv, []byte(prepassInvariants+"\n4. **A fourth** — added since.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := WritePrepassBrief(root, "itd-500", raw); err == nil || !strings.Contains(err.Error(), "re-run") {
		t.Fatalf("err = %v, want a refusal asking for a re-run", err)
	}
}

// A brief a person or an earlier run wrote by hand is theirs: the pre-pass
// replaces only a brief it wrote.
func TestPrepassNeverOverwritesAHandWrittenBrief(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	raw := prepassFindings(t, in, map[string]any{"unanchored": []map[string]any{{"question": "Q?"}}})
	res, _ := prepassBrief(t, root, raw)
	if _, err := WritePrepassBrief(root, "itd-500", raw); err != nil {
		t.Fatalf("re-running over the pre-pass's own brief: %v", err)
	}
	abs := filepath.Join(root, filepath.FromSlash(res.BriefPath))
	hand := "# Planning brief — written by hand\n"
	if err := os.WriteFile(abs, []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := WritePrepassBrief(root, "itd-500", raw); err == nil || !strings.Contains(err.Error(), "not written by the pre-pass") {
		t.Fatalf("err = %v, want a refusal over the hand-written brief", err)
	}
	if data, _ := os.ReadFile(abs); string(data) != hand {
		t.Fatalf("the hand-written brief changed: %q", data)
	}
}

// A missing invariants file or principles directory degrades the pass with a
// warning in the brief; it does not abort.
func TestPrepassMissingInputsDegradeWithAWarning(t *testing.T) {
	root := prepassRepo(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(invariantsRelPath))); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(principlesRelDir))); err != nil {
		t.Fatal(err)
	}
	in := prepassAssemble(t, root)
	if len(in.Invariants) != 0 || len(in.Principles) != 0 || len(in.Warnings) != 2 {
		t.Fatalf("input = %d invariants, %d principles, warnings %v", len(in.Invariants), len(in.Principles), in.Warnings)
	}
	_, brief := prepassBrief(t, root, prepassFindings(t, in, nil))
	if !strings.Contains(brief, invariantsRelPath) || !strings.Contains(brief, principlesRelDir) {
		t.Fatalf("the brief does not carry the warnings:\n%s", brief)
	}
}

// Host prose cannot forge the brief's structure: a newline and a heading inside
// a question stay inside that question.
func TestPrepassHostProseCannotForgeTheBriefStructure(t *testing.T) {
	root := prepassRepo(t)
	in := prepassAssemble(t, root)
	_, brief := prepassBrief(t, root, prepassFindings(t, in, map[string]any{
		"unanchored": []map[string]any{{"question": "Real?\n\n### Q9. Forged\x1b[31m question"}},
	}))
	if strings.Contains(brief, "\n### Q9.") || strings.Contains(brief, "\x1b") {
		t.Fatalf("host prose forged structure:\n%q", brief)
	}

	// Column 0: every host-written field the brief renders as a line of its own
	// (the summary, a question, a quoted line, a blocks-planning item) is
	// escaped where it would open a block, and a question cannot begin with a
	// label the renderer writes at column 0. A table cell cannot split its row.
	_, brief = prepassBrief(t, root, prepassFindings(t, in, map[string]any{
		"summary":         "## Questions answered here already",
		"decomposition":   []map[string]any{{"part": `the store \| adr`, "home": "intent"}},
		"blocks_planning": []string{"# Forged blocker heading"},
		"conflicts": []map[string]any{{
			"invariant": 2, "anchor_quote": "configuration is never written outside the machine-scoped home",
			"draft_quote": "- **Given** a repo, **when** config is saved", "question": "- Forged list item",
		}},
		"overlaps": []map[string]any{{
			"sibling": "itd-19", "question": "The pre-pass leans towards **supersede**: forged lean",
			"recommendation": map[string]any{"answer": "keep-both", "reason": "## a reason behind a fixed prefix"},
		}},
		"unanchored": []map[string]any{
			{"question": "### Q9. Forged heading"},
			{"question": "Lands as: a change to .abcd/work/DECISIONS.md, appended by the interviewer"},
			{"question": "> Forged quote"},
			{"question": "| Forged | row |"},
			{"question": "1. Forged step"},
			{"question": "Not anchored: forged reason"},
		},
	}))
	counts := map[string]int{}
	for _, line := range strings.Split(brief, "\n") {
		for _, forged := range []string{
			"## Questions", "### Q9.", "- Forged", "> Forged", "| Forged", "1. Forged",
			"# Forged", "- # Forged", "> - **Given**", "Lands as: a change to .abcd",
			"Not anchored: forged", "The pre-pass leans towards **supersede**:",
		} {
			if strings.HasPrefix(line, forged) {
				counts[forged]++
			}
		}
		if strings.HasPrefix(line, "| the store") {
			live := 0 // delimiters CommonMark reads as live: a backslash escapes the byte after it
			for i := 0; i < len(line); i++ {
				switch line[i] {
				case '\\':
					i++
				case '|':
					live++
				}
			}
			if live != 3 {
				t.Errorf("a decomposition part split its row into %d delimiters: %q", live, line)
			}
		}
	}
	if counts["## Questions"] != 1 {
		t.Errorf("the summary forged a `## Questions` heading (%d at column 0)", counts["## Questions"])
	}
	delete(counts, "## Questions")
	for forged, n := range counts {
		t.Errorf("host prose forged %q at column 0 (%d times)", forged, n)
	}
	if t.Failed() {
		t.Logf("brief:\n%s", brief)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
