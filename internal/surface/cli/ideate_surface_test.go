package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ideate"
)

// ideateRepo lays out the minimum an `ideate record` run needs, so the surface
// test proves the WIRING end to end rather than mocking the core.
func ideateRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
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
	write(ideate.DecisionsRelDir, "# Decisions\n")
	write(".abcd/development/intents/planned/itd-104-ideate.md",
		"---\nid: itd-104\nslug: ideate\nspec_id: spc-18\nimpact: additive\n---\n\n# itd-104\n")
	return root
}

// ideateVerdictJSON is a valid verdict document; cited is the record id the grill
// hit cites, so a test can swap in an id that does not resolve.
func ideateVerdictJSON(cited string) string {
	return `{
  "schema_version": 1,
  "prompt_version": "1.0.0",
  "idea": "abcd should gate a new idea before it becomes a record entry",
  "legs": [
    {"kind": "research", "claims": [
      {"claim": "fresh-context evaluation is the measured debiasing effect",
       "primary_source": "https://example.invalid/paper", "status": "verified"}
    ]},
    {"kind": "record-grill", "hits": [
      {"record": "` + cited + `", "relation": "covered", "note": "already filed"}
    ]},
    {"kind": "adversarial-review", "kill_attempts": [
      {"attempt": "the gauntlet adds friction", "outcome": "survived"}
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

	out := runCLI(t, "ideate", "record", "the-ideate-gate", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")), "--json")
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
	const wantDir = ".abcd/development/research/notes/"
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
	text := string(runCLI(t, "ideate", "record", "a-second-idea", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))))
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
			[]string{"ideate", "record", "the-ideate-gate", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-9999"))},
			"itd-9999",
		},
		{
			"traversal-slug",
			[]string{"ideate", "record", "../../../etc/passwd", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))},
			"slug",
		},
		{
			"verdict-json-required",
			[]string{"ideate", "record", "the-ideate-gate"},
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
		"ideate", "record", "from-stdin", "--verdict-json", "-", "--json")
	if !strings.Contains(string(out), "from-stdin") {
		t.Errorf("stdin payload was not recorded:\n%s", out)
	}
}

// TestIdeateRoutingHintIsAPointerNotAGate is AC 1: the intent and capture status
// renders NAME /abcd:ideate for a big unproven idea, and neither one blocks,
// warns, or requires it.
func TestIdeateRoutingHintIsAPointerNotAGate(t *testing.T) {
	repo := ideateRepo(t)
	// The bare capture board resolves the checkout root and refuses outside one
	// (iss-2609090951291524), so the routing hint is read in a working tree.
	gitInitAt(t, repo)
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
