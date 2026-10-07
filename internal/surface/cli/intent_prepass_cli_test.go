package cli

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

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/intent"
)

// intent_prepass_cli_test.go — the front doors of the pre-pass (itd-42,
// spc-2609211918551301): `abcd intent prepass <itd-N>` prints the input and
// writes nothing, `--findings-json <path>` writes the planning brief and
// nothing else, every refusal exits 2 with nothing written, and the plugin
// page's planning interview opens from the verb (criterion 4). The judgement
// itself is the host's; these tests stand in for it with a findings file.

var cliPrepassInvariants = `# Invariants

## Properties the system must preserve regardless of how it's built

1. **Transparent prompts** — every prompt shows current state and how to change it later.

2. **Config stays home** — configuration is never written outside the
   machine-scoped home under ` + abcdhome.Display() + `/, whatever the caller asks.
`

const cliPrepassDraft = `---
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

func cliPrepassRecord(id, slug, title string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nkind: standalone\nspec_id: null\n---\n\n# " + title + "\n\nBody.\n"
}

// cliPrepassRepo lays the four inputs in a git working tree: the invariants,
// one principle, the draft under test and a sibling on three other shelves.
func cliPrepassRepo(t *testing.T) string {
	t.Helper()
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, ".abcd/development/brief/02-constraints/03-invariants.md", cliPrepassInvariants)
	writeRepoFile(t, repo, ".abcd/development/principles/the-users-directory-is-theirs.md",
		"# The user's directory is theirs\n\n**The rule.** A tool creates files only in space that was handed to it.\n")
	writeRepoFile(t, repo, cliDrafts+"/itd-500-shared-config.md", cliPrepassDraft)
	writeRepoFile(t, repo, cliPlanned+"/itd-19-stage-aware.md", cliPrepassRecord("itd-19", "stage-aware", "Behaviour Is Stage-Aware"))
	writeRepoFile(t, repo, ".abcd/development/intents/shipped/itd-7-home-store.md", cliPrepassRecord("itd-7", "home-store", "Every Store Lives Under The Home"))
	writeRepoFile(t, repo, ".abcd/development/intents/superseded/itd-3-old-config.md", cliPrepassRecord("itd-3", "old-config", "The Old Config Draft"))
	return repo
}

// cliTreeHash hashes every file of the working tree outside .git, keyed by
// its slash path: git's own bookkeeping is not the record.
func cliTreeHash(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
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

// cliTreeDiff names every path added, removed or changed between two hashes.
func cliTreeDiff(before, after map[string]string) []string {
	var out []string
	for p, h := range after {
		if before[p] != h {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func cliPrepassInput(t *testing.T) intent.PrepassInput {
	t.Helper()
	var in intent.PrepassInput
	out := runCLI(t, "intent", "prepass", "itd-500", "--json")
	if err := json.Unmarshal(out, &in); err != nil {
		t.Fatalf("prepass --json is not the input: %v\n%s", err, out)
	}
	return in
}

// cliFindingsFile writes a findings payload stamped for in OUTSIDE the
// repository, as a host's scratch would be, and returns its path.
func cliFindingsFile(t *testing.T, in intent.PrepassInput, body map[string]any) string {
	t.Helper()
	payload := map[string]any{"_type": intent.PrepassFindingsType, "intent": in.Intent, "input_digest": in.Digest}
	for k, v := range body {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Criterion 3, the emit half: the verb prints the four inputs and writes
// nothing.
func TestIntentPrepassEmitWritesNothing(t *testing.T) {
	repo := cliPrepassRepo(t)
	before := cliTreeHash(t, repo)

	in := cliPrepassInput(t)
	if in.Type != intent.PrepassInputType || in.Intent != "itd-500" || in.Digest == "" {
		t.Fatalf("the input's head = %q %q %q", in.Type, in.Intent, in.Digest)
	}
	if len(in.Invariants) != 2 || len(in.Principles) != 1 || !strings.Contains(in.Draft, "into the project directory") {
		t.Fatalf("the input does not carry the invariants, the principle and the draft: %+v", in)
	}
	shelves := map[string]bool{}
	for _, e := range in.Index {
		shelves[e.Shelf] = true
	}
	for _, s := range []string{"planned", "shipped", "superseded"} {
		if !shelves[s] {
			t.Fatalf("the index misses the %s shelf: %+v", s, in.Index)
		}
	}
	if len(in.Answers) != 4 || len(in.Rules) == 0 {
		t.Fatalf("the input does not hand the host the answers and the rules: %+v", in)
	}

	text := string(runCLI(t, "intent", "prepass", "itd-500"))
	for _, want := range []string{"abcd intent prepass — itd-500", "input digest " + in.Digest, "--findings-json"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the human render does not say %q:\n%s", want, text)
		}
	}
	if d := cliTreeDiff(before, cliTreeHash(t, repo)); len(d) != 0 {
		t.Fatalf("the emit wrote %v", d)
	}
}

// Criteria 1, 2, 3 and 5 end to end: the findings write the planning brief and
// nothing else, and the brief quotes the conflict, asks the overlap with the
// four answers and its recommendation in prose, and marks the unanchored
// concern.
func TestIntentPrepassFindingsWriteOnlyTheBrief(t *testing.T) {
	repo := cliPrepassRepo(t)
	in := cliPrepassInput(t)
	path := cliFindingsFile(t, in, map[string]any{
		"summary": "Shared config written into the project.",
		"conflicts": []map[string]any{{
			"invariant": 2, "anchor_quote": "configuration is never written outside the",
			"draft_quote": "into the project directory", "question": "Which gives, the invariant or the draft?",
		}},
		"overlaps": []map[string]any{{
			"sibling": "itd-7", "question": "itd-7 already keeps every store under the home; what does this add?",
			"recommendation": map[string]any{"answer": "refine", "reason": "the draft narrows where one file lives"},
		}},
		"unanchored": []map[string]any{{"question": "Who reads the shared file first?"}},
	})
	before := cliTreeHash(t, repo)

	var res intent.PrepassBriefResult
	out := runCLI(t, "intent", "prepass", "itd-500", "--findings-json", path, "--json")
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("--findings-json --json is not the result: %v\n%s", err, out)
	}
	wantPath := intent.PlanningBriefsRelDir + "/itd-500.md"
	if res.Intent != "itd-500" || res.BriefPath != wantPath || res.Conflicts != 1 || res.Overlaps != 1 || res.Unanchored != 1 {
		t.Fatalf("result = %+v", res)
	}
	if d := cliTreeDiff(before, cliTreeHash(t, repo)); len(d) != 1 || d[0] != wantPath {
		t.Fatalf("the pre-pass wrote %v, want only %s", d, wantPath)
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(wantPath)))
	if err != nil {
		t.Fatal(err)
	}
	brief := string(data)
	for _, want := range []string{
		// criterion 1: the invariant named, both lines quoted
		"Invariant 2", "> configuration is never written outside the", "> into the project directory",
		// criterion 2: the sibling, the four answers, the recommendation in prose
		"itd-7", "- **Keep both**", "- **Bundle**", "- **Supersede**", "- **Refine**",
		"The pre-pass leans towards **refine**: the draft narrows where one file lives.",
		// criterion 5: the unanchored concern is a question marked so
		"A question (unanchored)", "Who reads the shared file first?",
		// criterion 4's hand-off: every question says where its answer lands
		"Lands as:",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the brief does not carry %q:\n%s", want, brief)
		}
	}
	if strings.Contains(brief, "- **Refine** (recommended)") || strings.Contains(brief, "**Refine** ★") {
		t.Fatalf("the recommendation is a marked option:\n%s", brief)
	}

	text := string(runCLI(t, "intent", "prepass", "itd-500", "--findings-json", path))
	if !strings.Contains(text, "abcd intent prepass — itd-500 brief written") || !strings.Contains(text, wantPath) {
		t.Fatalf("the human render does not name the brief:\n%s", text)
	}
}

// A record that is not a draft has no interview to come: exit 2, naming the
// remedy, with nothing written — on both forms.
func TestIntentPrepassRefusesANonDraft(t *testing.T) {
	repo := cliPrepassRepo(t)
	in := cliPrepassInput(t)
	path := cliFindingsFile(t, in, map[string]any{"unanchored": []map[string]any{{"question": "Q?"}}})
	before := cliTreeHash(t, repo)
	for _, args := range [][]string{
		{"intent", "prepass", "itd-19"},
		{"intent", "prepass", "itd-19", "--findings-json", path},
		{"intent", "prepass", "itd-404"},
		{"intent", "prepass", "not-an-id"},
	} {
		_, err := runCLIErr(t, args...)
		if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "abcd intent prepass:") || !strings.Contains(err.Error(), "nothing written") {
			t.Fatalf("%v: want exit 2 naming the verb and that nothing was written; exit %d (%v)", args, exitCodeOf(err), err)
		}
	}
	_, err := runCLIErr(t, "intent", "prepass", "itd-19")
	if !strings.Contains(err.Error(), "planned/") || !strings.Contains(err.Error(), "abcd intent") {
		t.Fatalf("the non-draft refusal names neither the shelf nor the remedy: %v", err)
	}
	if d := cliTreeDiff(before, cliTreeHash(t, repo)); len(d) != 0 {
		t.Fatalf("a refusal wrote %v", d)
	}
}

// Malformed findings are refused on exit 2 with nothing written, the brief
// included.
func TestIntentPrepassRefusesMalformedFindings(t *testing.T) {
	repo := cliPrepassRepo(t)
	in := cliPrepassInput(t)
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	stamp := `"_type":"` + intent.PrepassFindingsType + `","intent":"itd-500","input_digest":"` + in.Digest + `"`
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(write("target.json", "{"+stamp+"}"), link); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"not json":       write("a.json", "{"),
		"unknown field":  write("b.json", "{"+stamp+`,"verdict":"pass"}`),
		"stale digest":   write("c.json", `{"_type":"`+intent.PrepassFindingsType+`","intent":"itd-500","input_digest":"sha256:00"}`),
		"blank question": write("d.json", "{"+stamp+`,"unanchored":[{"question":"  "}]}`),
		"missing file":   filepath.Join(dir, "absent.json"),
		"symlink":        link,
		"empty flag":     "",
	}
	before := cliTreeHash(t, repo)
	for name, p := range cases {
		out, err := runCLIErr(t, "intent", "prepass", "itd-500", "--findings-json", p)
		if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "nothing written") {
			t.Fatalf("%s: want exit 2 saying nothing was written; exit %d (%v)\n%s", name, exitCodeOf(err), err, out)
		}
	}
	if d := cliTreeDiff(before, cliTreeHash(t, repo)); len(d) != 0 {
		t.Fatalf("malformed findings wrote %v", d)
	}
	// The accept control: the same stamp with a well-formed body writes.
	runCLI(t, "intent", "prepass", "itd-500", "--findings-json", write("ok.json", "{"+stamp+`,"unanchored":[{"question":"Who reads it?"}]}`))
}

// Criterion 4 on the plugin page: the planning interview opens by running the
// verb, reads the brief it wrote, and asks each question in order, recording
// the answer where its "Lands as:" says.
func TestIntentPageInterviewOpensFromThePrepass(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(testRepoRoot(), "commands", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(body)
	if !strings.Contains(page[:strings.Index(page, "\n---\n")], "prepass <itd-N>") {
		t.Fatal("the page's argument-hint does not list `prepass <itd-N>`")
	}
	start := strings.Index(page, "## Planning interview")
	if start < 0 {
		t.Fatal("the page has no planning interview")
	}
	interview := page[start:]
	interview = interview[:strings.Index(interview[3:], "\n## ")+3]
	first := strings.Index(interview, "\n1. ")
	if first < 0 {
		t.Fatal("the interview has no numbered steps")
	}
	opening := interview[:first]
	for _, want := range []string{
		"intent prepass <itd-N> --json",
		"intent prepass <itd-N> --findings-json",
		"planning-briefs/<itd-N>.md",
		"Lands as:",
	} {
		if !strings.Contains(opening, want) {
			t.Fatalf("the interview's opening, before step 1, does not name %q:\n%s", want, opening)
		}
	}
	auto := page[strings.Index(page, "## Autonomous runs"):]
	auto = auto[:strings.Index(auto[3:], "\n## ")+3]
	if !strings.Contains(auto, "abcd intent prepass") {
		t.Fatalf("the Autonomous runs paragraph does not name the verb:\n%s", auto)
	}
}
