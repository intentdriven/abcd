package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"

	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/core/ideate"
	"github.com/intentdriven/abcd/internal/core/source"
	"github.com/intentdriven/abcd/internal/gittest"
)

// ideateRepo lays out the minimum an `ideate record` run needs, so the surface
// test proves the WIRING end to end rather than mocking the core.
//
// It is a git working tree, not a bare temporary directory: a directory outside
// every repository is no longer a place a verdict is recorded, because the front
// door resolves the checkout root and refuses when there is none
// (iss-2609091729516940).
func ideateRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	gitInitAt(t, root)
	gitCommitAt(t, root, "root")
	root = realPath(t, root)
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(ideate.ResearchRelDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	write(decisions.SharedTierRel+"/CONTEXT.md", "# Context\n")
	write(".abcd/development/intents/planned/itd-104-ideate.md",
		"---\nid: itd-104\nslug: ideate\nspec_id: spc-18\nimpact: additive\n---\n\n# itd-104\n")
	return root
}

// ideateVerdictJSON is a valid verdict document; cited is the record id the grill
// hit cites, so a test can swap in an id that does not resolve.
func ideateVerdictJSON(cited string) string {
	return ideateVerdictJSONFrom(cited, "https://example.invalid/paper")
}

// ideateVerdictJSONFrom is ideateVerdictJSON with the research claim's primary
// source given too, so a test can cite a filed source key.
func ideateVerdictJSONFrom(cited, primary string) string {
	return `{
  "schema_version": 2,
  "prompt_version": "1.0.0",
  "idea": "abcd should gate a new idea before it becomes a record entry",
  "origin_sources": [],
  "legs": [
    {"kind": "research", "claims": [
      {"claim": "fresh-context evaluation is the measured debiasing effect",
       "primary_source": "` + primary + `", "status": "verified"}
    ]},
    {"kind": "record-grill", "hits": [
      {"record": "` + cited + `", "relation": "covered", "note": "already filed"}
    ], "searches": [
      {"terms": ["admission gate"], "places": [".abcd/development/intents/"], "count": 1}
    ]},
    {"kind": "adversarial-review", "reviewers": [
      {"label": "reviewer-a", "verdict": "survives", "sources": [], "kill_attempts": [
        {"attempt": "the gauntlet adds friction", "outcome": "survived"}
      ]}
    ]}
  ],
  "verdict": "survives",
  "rejected_alternatives": [
    {"alternative": "make it a blocking pre-capture gate", "why_rejected": "capture friction stays at one line"}
  ]
}`
}

// writeVerdict stages a verdict payload on disk and returns its path.
func writeVerdict(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "verdict.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestIdeateRecordWiredFromTheCLI is the wiring proof: the verb reaches
// ideate.Record from the front door and a durable verdict record lands on disk.
func TestIdeateRecordWiredFromTheCLI(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)

	out := runCLI(t, "ideate", "record", "the-ideate-gate", "--public", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")), "--json")
	var res ideate.Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, out)
	}
	if res.Verdict != ideate.VerdictSurvives || !res.Graduates {
		t.Errorf("verdict = %q graduates = %v; want survives/true", res.Verdict, res.Graduates)
	}
	// The literal directory, not ideate.ResearchRelDir: a check written against
	// the constant follows the constant wherever it moves, so it can never notice
	// the record landing outside the home research/README.md gives a dated note.
	wantDir := ".abcd/development/research/ideas/" + gittest.RootCommit(t, repo) + "/"
	if !strings.HasPrefix(res.Path, wantDir) || !strings.HasSuffix(res.Path, "-ideate-the-ideate-gate.md") {
		t.Errorf("path = %q; want a dated ideate record under %s", res.Path, wantDir)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(res.Path))); err != nil {
		t.Errorf("the reported record is not on disk: %v", err)
	}
	if strings.Join(res.CitedRecords, ",") != "itd-104" {
		t.Errorf("CitedRecords = %v, want [itd-104]", res.CitedRecords)
	}

	// The text render is what a human sees; it must name the verdict and the file.
	text := string(runCLI(t, "ideate", "record", "a-second-idea", "--public", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))))
	if !strings.Contains(text, "survives") || !strings.Contains(text, "a-second-idea") {
		t.Errorf("text render is missing the verdict or the slug:\n%s", text)
	}
}

// TestIdeateRecordRefusalsExitTwo proves every structural refusal reaches the
// operator as exit 2 with a diagnostic that names the fault — the unresolvable
// citation (AC 4) and the traversal slug being the two that matter most.
func TestIdeateRecordRefusalsExitTwo(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			"unresolvable-citation",
			[]string{"ideate", "record", "the-ideate-gate", "--public", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-9999"))},
			"itd-9999",
		},
		{
			"traversal-slug",
			[]string{"ideate", "record", "../../../etc/passwd", "--public", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))},
			"slug",
		},
		{
			// A host reading an older command page composes version 1; the
			// refusal names the version to compose instead.
			"version-1-payload",
			[]string{"ideate", "record", "the-ideate-gate", "--public", "--verdict-json",
				writeVerdict(t, strings.Replace(ideateVerdictJSON("itd-104"), `"schema_version": 2`, `"schema_version": 1`, 1))},
			"version 2",
		},
		{
			"verdict-json-required",
			[]string{"ideate", "record", "the-ideate-gate", "--public"},
			"--verdict-json",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runCLIErr(t, c.args...)
			if err == nil {
				t.Fatalf("the run succeeded; want a refusal\n%s", out)
			}
			if code := exitCodeOf(err); code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("diagnostic %q does not name %q", err.Error(), c.want)
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(ideate.ResearchRelDir)))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("a refused run wrote %s", e.Name())
	}
}

// TestIdeateRecordReadsStdin proves the "-" payload form works, matching every
// other host-delegated ingest verb.
func TestIdeateRecordReadsStdin(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)
	out := runCLIStdin(t, ideateVerdictJSON("itd-104"),
		"ideate", "record", "from-stdin", "--public", "--verdict-json", "-", "--json")
	if !strings.Contains(string(out), "from-stdin") {
		t.Errorf("stdin payload was not recorded:\n%s", out)
	}
}

// TestIdeateRoutingHintIsAPointerNotAGate is AC 1: the intent and capture status
// renders NAME /abcd:ideate for a big unproven idea, and neither one blocks,
// warns, or requires it.
func TestIdeateRoutingHintIsAPointerNotAGate(t *testing.T) {
	// The bare capture and intent boards resolve the checkout root and refuse
	// outside one (iss-2609090951291524, iss-2609091729516940), so the routing
	// hint is read in a working tree — which ideateRepo now always is.
	repo := ideateRepo(t)
	t.Chdir(repo)
	for _, verb := range []string{"intent", "capture"} {
		out := string(runCLI(t, verb))
		if !strings.Contains(out, "ideate") {
			t.Errorf("bare `abcd %s` does not name ideate as a route:\n%s", verb, out)
		}
		for _, forbidden := range []string{"must", "required", "before you", "warning"} {
			if strings.Contains(strings.ToLower(out), forbidden) {
				t.Errorf("bare `abcd %s` presents ideate as a gate (%q):\n%s", verb, forbidden, out)
			}
		}
	}
	// Capture still writes an issue with no ideate step in the way.
	if out, err := runCLIErr(t, "capture", "a plain observation", "--json"); err != nil {
		t.Errorf("capture was blocked: %v\n%s", err, out)
	}
}

// TestIdeateRenderSaysWhatEachVerdictLeavesNext is iss-2609100508573400: a
// reframed idea survives in another shape, so its next line must not read as
// the killed one's. Each verdict gets its own line, and the reframed one names
// the reframing as what may graduate, by the record section that holds it
// (iss-2610262154123870).
func TestIdeateRenderSaysWhatEachVerdictLeavesNext(t *testing.T) {
	next := func(v ideate.Verdict) string {
		out := renderIdeateResult(ideate.Result{Slug: "x", Verdict: v, Graduates: v == ideate.VerdictSurvives})
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "next:") {
				return line
			}
		}
		t.Fatalf("no next line for %s:\n%s", v, out)
		return ""
	}
	killed, reframed, survives := next(ideate.VerdictKilled), next(ideate.VerdictReframed), next(ideate.VerdictSurvives)
	if reframed == killed || reframed == survives {
		t.Errorf("the reframed verdict borrows another verdict's next line: %q", reframed)
	}
	for _, want := range []string{"reframing", "abcd intent", `"The reframing"`} {
		if !strings.Contains(reframed, want) {
			t.Errorf("the reframed next line must name %q: %q", want, reframed)
		}
	}
	if strings.Contains(reframed, "does not graduate;") {
		t.Errorf("the reframed next line reads as the killed one: %q", reframed)
	}
}

// TestIdeateRecordRefusesBothAnswers pins the front door's one remaining
// refusal: --private and --public together name two stores, so the verb
// writes nothing and names both flags.
func TestIdeateRecordRefusesBothAnswers(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)
	payload := writeVerdict(t, ideateVerdictJSON("itd-104"))

	out, err := runCLIErr(t, "ideate", "record", "a-doubly-answered-idea", "--private", "--public", "--verdict-json", payload)
	if err == nil {
		t.Fatalf("the run succeeded; want a refusal\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	for _, want := range []string{"--private", "--public"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("diagnostic %q does not name %s", err.Error(), want)
		}
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(ideate.ResearchRelDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("a refused run wrote %s", e.Name())
	}
	if _, err := os.Stat(abcdhome.Path(os.Getenv("HOME"), "ideas")); !os.IsNotExist(err) {
		t.Errorf("a refused run laid the private store: %v", err)
	}
}

// TestIdeateRecordDefaultsToPrivate is iss-2610300838089246: ideate is
// private unless the person says public, and nobody is asked. A run with
// neither flag records the verdict in the private store, exactly as
// --private does, and leaves the checkout as it was.
func TestIdeateRecordDefaultsToPrivate(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)

	out, err := runCLIErr(t, "ideate", "record", "an-unflagged-idea",
		"--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")), "--json")
	if err != nil {
		t.Fatalf("a run with neither flag refused: %v\n%s", err, out)
	}
	var res ideate.Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, out)
	}
	sha := gittest.RootCommit(t, repo)
	wantPrefix := abcdhome.Display("ideas", sha) + "/"
	if res.Visibility != ideate.VisibilityPrivate || !strings.HasPrefix(res.Path, wantPrefix) || res.DecisionPath != "" {
		t.Errorf("result = visibility %q path %q decision %q; want private under %s with no pointer",
			res.Visibility, res.Path, res.DecisionPath, wantPrefix)
	}
	if _, err := os.Stat(abcdhome.Path(os.Getenv("HOME"), "ideas", sha, filepath.Base(res.Path))); err != nil {
		t.Errorf("the private record is not on disk: %v", err)
	}
	if log := decisionTexts(t, repo); log != "" {
		t.Errorf("an unflagged verdict recorded a committed decision:\n%s", log)
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(ideate.ResearchRelDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("an unflagged verdict wrote %s into the checkout", e.Name())
	}
}

// TestIdeateRecordPrivateFromTheCLI is the private store's wiring proof: the
// verdict lands owner-only under $HOME/.abcd.noindex/ideas/<root-sha>/, the
// result names it home-relative, and the checkout is left exactly as it was.
func TestIdeateRecordPrivateFromTheCLI(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)

	out := runCLI(t, "ideate", "record", "a-private-idea", "--private",
		"--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")), "--json")
	var res ideate.Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, out)
	}
	sha := gittest.RootCommit(t, repo)
	wantPrefix := abcdhome.Display("ideas", sha) + "/"
	if res.Visibility != ideate.VisibilityPrivate || !strings.HasPrefix(res.Path, wantPrefix) || res.DecisionPath != "" {
		t.Errorf("result = visibility %q path %q decision %q; want private under %s with no pointer",
			res.Visibility, res.Path, res.DecisionPath, wantPrefix)
	}
	onDisk := abcdhome.Path(os.Getenv("HOME"), "ideas", sha, filepath.Base(res.Path))
	if fi, err := os.Stat(onDisk); err != nil {
		t.Errorf("the private record is not on disk: %v", err)
	} else if fi.Mode().Perm() != 0o600 {
		t.Errorf("private record mode = %o, want 600", fi.Mode().Perm())
	}
	if log := decisionTexts(t, repo); log != "" {
		t.Errorf("a private verdict recorded a committed decision:\n%s", log)
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(ideate.ResearchRelDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("a private verdict wrote %s into the checkout", e.Name())
	}

	text := string(runCLI(t, "ideate", "record", "a-second-private-idea", "--private",
		"--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))))
	if !strings.Contains(text, "private") || strings.Contains(text, os.Getenv("HOME")) {
		t.Errorf("the text render must say private and never echo the absolute home:\n%s", text)
	}
}

// TestIdeateRecordRefusesAShallowCheckout proves neither store is keyed on a
// shallow clone's graft: there `rev-list --max-parents=0 HEAD` names the graft
// commit, not the repository's root, so a verdict would land in a folder no
// full clone of the same repository ever reads. The run refuses, writing
// nothing in either store.
func TestIdeateRecordRefusesAShallowCheckout(t *testing.T) {
	src := ideateRepo(t)
	for _, args := range [][]string{
		{"add", "-A"},
		{"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "seed"},
	} {
		cmd := exec.Command("git", append([]string{"-C", src}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	dst := filepath.Join(t.TempDir(), "shallow")
	clone := exec.Command("git", "clone", "--quiet", "--depth", "1", "file://"+src, dst)
	clone.Env = gittest.Env(t)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("shallow clone: %v (%s)", err, out)
	}
	t.Chdir(dst)
	payload := writeVerdict(t, ideateVerdictJSON("itd-104"))
	for _, answer := range []string{"--public", "--private"} {
		t.Run(answer, func(t *testing.T) {
			out, err := runCLIErr(t, "ideate", "record", "a-shallow-idea", answer, "--verdict-json", payload)
			if err == nil {
				t.Fatalf("the run succeeded in a shallow clone\n%s", out)
			}
			if code := exitCodeOf(err); code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if !strings.Contains(err.Error(), "shallow") {
				t.Errorf("diagnostic %q does not say the checkout is shallow", err.Error())
			}
			// The remedy is named once, in the person's terms: no second copy of it
			// and no aside about how CI checks out.
			if n := strings.Count(err.Error(), "git fetch --unshallow"); n != 1 || strings.Contains(err.Error(), "CI") {
				t.Errorf("diagnostic %q names the remedy %d times or speaks of CI", err.Error(), n)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dst, filepath.FromSlash(ideate.ResearchRelDir))); !os.IsNotExist(err) {
		t.Errorf("a refused run laid the public store: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(os.Getenv("HOME"), "ideas")); !os.IsNotExist(err) {
		t.Errorf("a refused run laid the private store: %v", err)
	}
}

// TestIdeateRecordResolvesAFiledSourceAgainstTheDefaultCorpus is the front
// door's half of spc-2610142138047122's source check: a `source:<key>` primary
// source is resolved against the person's default sources corpus when one
// exists, and accepted on its shape when none does.
func TestIdeateRecordResolvesAFiledSourceAgainstTheDefaultCorpus(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)

	// No corpus under this HOME: a well-formed key is accepted.
	runCLI(t, "ideate", "record", "a-filed-idea", "--public",
		"--verdict-json", writeVerdict(t, ideateVerdictJSONFrom("itd-104", "source:survey2026")), "--json")

	for k, v := range map[string]string{
		"GIT_AUTHOR_NAME": "Fixture", "GIT_AUTHOR_EMAIL": "fixture@example.invalid",
		"GIT_COMMITTER_NAME": "Fixture", "GIT_COMMITTER_EMAIL": "fixture@example.invalid",
	} {
		t.Setenv(k, v)
	}
	dir := abcdhome.Path(os.Getenv("HOME"), "sources")
	if _, err := source.Init(dir); err != nil {
		t.Fatalf("source.Init: %v", err)
	}
	doc := filepath.Join(t.TempDir(), "survey.md")
	if err := os.WriteFile(doc, []byte("# survey\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Add(source.AddRequest{Corpus: dir, Key: "survey2026", Title: "A survey", Class: source.ClassPublic, Original: doc}); err != nil {
		t.Fatalf("source.Add: %v", err)
	}
	runCLI(t, "ideate", "record", "a-second-filed-idea", "--public",
		"--verdict-json", writeVerdict(t, ideateVerdictJSONFrom("itd-104", "source:survey2026")), "--json")

	out, err := runCLIErr(t, "ideate", "record", "an-unfiled-idea", "--public",
		"--verdict-json", writeVerdict(t, ideateVerdictJSONFrom("itd-104", "source:never-filed")))
	if err == nil {
		t.Fatalf("a key the default corpus does not carry was accepted\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "source:never-filed") {
		t.Errorf("exit %d, diagnostic %q; want 2 naming the key", code, err.Error())
	}
}
