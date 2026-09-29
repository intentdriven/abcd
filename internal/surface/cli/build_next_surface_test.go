package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildNextPicksAndCarriesTheReasonInJSON is itd-2609211116005482's
// criterion 10 over a pick: the payload carries the candidate set with each
// score, the pick with its rule and runner-up, the reason's text, and the run
// the pick started; the text form names the pick and the reason.
func TestBuildNextPicksAndCarriesTheReasonInJSON(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(".abcd/development/intents/planned/itd-11-beta.md",
		strings.NewReplacer("itd-10", "itd-11", "alpha", "beta", "spc-1", "spc-2", "- Given x, when y, then z.", "- The page states the flags.").Replace(buildIntent()))
	repo.Write(".abcd/development/specs/open/spc-2-beta.md", "---\nid: spc-2\nslug: beta\nintent: itd-11\n---\n# beta\n\n## Summary\n\nWritten.\n")
	repo.Commit("a second intent")

	out := mustImplement(t, "build", "next", "--json")
	var res struct {
		Candidates []struct {
			ID    string `json:"id"`
			Score struct {
				Total       int  `json:"total"`
				NoFootprint bool `json:"no_footprint"`
			} `json:"score"`
		} `json:"candidates"`
		Excluded []any `json:"excluded"`
		Pick     struct {
			Chosen struct {
				ID string `json:"id"`
			} `json:"chosen"`
			RunnerUp *struct {
				ID string `json:"id"`
			} `json:"runner_up"`
			TieBrokenByAge bool   `json:"tie_broken_by_age"`
			Rule           string `json:"rule"`
			Falsifier      string `json:"falsifier"`
		} `json:"pick"`
		Entry string `json:"entry"`
		Start struct {
			RunID string `json:"run_id"`
			Lane  struct {
				ID    string `json:"id"`
				Stage string `json:"stage"`
			} `json:"lane"`
		} `json:"start"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if len(res.Candidates) != 2 || res.Candidates[0].ID != "itd-10" || res.Candidates[0].Score.Total != 100 ||
		res.Candidates[1].Score.Total != 0 || !res.Candidates[0].Score.NoFootprint {
		t.Fatalf("the candidates carry their scores in the pick order: %s", out)
	}
	if res.Excluded == nil || res.Pick.Chosen.ID != "itd-10" || res.Pick.RunnerUp == nil || res.Pick.RunnerUp.ID != "itd-11" ||
		res.Pick.Rule == "" || res.Pick.Falsifier == "" {
		t.Fatalf("the pick carries the chosen, the runner-up, the rule and the falsifier: %s", out)
	}
	if !strings.HasPrefix(res.Entry, "picked by run "+res.Start.RunID+" on ") || res.Start.Lane.ID != "lane-1" || res.Start.Lane.Stage != "worktree" {
		t.Fatalf("the entry is the run's, and the run is the build's: %s", out)
	}
}

// TestBuildNextRefusalCarriesEveryExclusion is criterion 1's refusal on the
// surface and criterion 10's refusal half: with no candidate, --json renders
// the refusal with every excluded intent and its check, exits 2 and writes
// nothing; the text form names them.
func TestBuildNextRefusalCarriesEveryExclusion(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(buildPlanned, strings.Replace(buildIntent(), "kind: standalone\n", "kind: standalone\nheld: \"not yet\"\n", 1))
	repo.Commit("hold it")

	ref := refusalDocs(t, 2, "build", "next", "--json")
	ex, _ := ref["excluded"].([]any)
	if ref["stage"] != "pick" || len(ex) != 1 {
		t.Fatalf("the refusal is the pick's, with every exclusion: %v", ref)
	}
	if e, _ := ex[0].(map[string]any); e["id"] != "itd-10" || e["check"] != "hold" || e["reason"] == "" {
		t.Fatalf("each exclusion names the intent, the check and the reason: %v", ex[0])
	}
	code, _, errOut := implementCLI(t, "build", "next")
	if code != 2 || !strings.Contains(errOut, "itd-10 (hold") {
		t.Fatalf("the text refusal names each exclusion: %d %q", code, errOut)
	}
	code, _, errOut = implementCLI(t, "build", "next", "--until-empty")
	if code != 2 || !strings.Contains(errOut, "criterion 5") {
		t.Fatalf("--until-empty is refused by name: %d %q", code, errOut)
	}
	runDirAbsent(t, repo.Root())
}
