package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// matrixLabCLI mints a lab of a throwaway repository outside abcd, passes its
// preflight, lays the machine's configuration (an OpenRouter block with a pin,
// the opencode runner, and, when capped, a spending cap), and writes a
// declaration whose one builder is route. It returns the lab id and the
// declaration's path.
func matrixLabCLI(t *testing.T, route string, capped bool) (home, id, decl string) {
	t.Helper()
	home, repo := labCheckout(t)
	code, out, errOut := runLab(t, "lab", "mint", "does", "a", "cheaper", "model", "build", "it?")
	if code != 0 {
		t.Fatalf("lab mint exit %d: %s%s", code, out, errOut)
	}
	id = strings.Fields(out)[1]
	if code, out, _ := runLab(t, "lab", "preflight", id); code != 0 {
		t.Fatalf("lab preflight exit %d:\n%s", code, out)
	}
	cfg := `{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter","models":["vendor/model"],` +
		`"pins":{"vendor/model":{"order":["deepinfra"],"quantizations":["fp8"],"canonical_slug":"vendor/model-20240723"}}}}},` +
		`"runner":{"opencode":{"model":"openrouter/vendor/model"}}`
	if capped {
		cfg += `,"matrix":{"spend_cap":{"amount":25,"unit":"usd"}}`
	}
	cfg += "}"
	p := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	decl = filepath.Join(t.TempDir(), "matrix.json")
	body := `{"schema_version":1,"name":"routing check","tasks":[{"id":"feature-a","kind":"code","intent":"itd-2610161733316518"}],` +
		`"builders":[{"id":"host-big","route":"host"},{"id":"or-cheap","route":"` + route + `","cost_per_copy":{"amount":0.05,"unit":"usd"}}],` +
		`"repeats":{"pilot":3,"max":10},"concurrency":2,"judging":{"guests_may_judge":false}}`
	if err := os.WriteFile(decl, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = repo
	return home, id, decl
}

func TestLabMatrixShowsThePlanAndItsDigestFromTheCLI(t *testing.T) {
	home, id, decl := matrixLabCLI(t, "builder=runner:opencode@openrouter/vendor/model", true)
	code, out, errOut := runLab(t, "lab", "matrix", id, decl)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errOut)
	}
	for _, want := range []string{"copies:      6 in the pilot, at most 20", "spend cap:   25.00 usd (machine)", "openrouter   expected 0.50 usd at most",
		"load cap:    not used (no local cell)", "time:        known after the pilot", "digest:      ", "--approve ",
		"provider pin: order deepinfra; fallbacks off"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plan lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out+errOut, home) {
		t.Errorf("the plan carries the home path:\n%s%s", out, errOut)
	}
	code, out, _ = runLab(t, "lab", "--json", "matrix", id, decl)
	var got struct {
		Digest      string `json:"digest"`
		ApproveWith string `json:"approve_with"`
		Cells       []struct {
			Cell, Source string
		} `json:"cells"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &got) != nil || len(got.Digest) != 64 || !strings.HasSuffix(got.ApproveWith, "--approve "+got.Digest) || len(got.Cells) != 2 {
		t.Fatalf("--json (exit %d) = %s", code, out)
	}
	// Showing the plan writes nothing in the lab.
	if entries, _ := filepath.Glob(filepath.Join(abcdhome.Path(home, "lab"), "*", id, "state", "matrix", "*")); len(entries) != 0 {
		t.Fatalf("showing the plan wrote %v", entries)
	}
}

func TestLabMatrixRefusesAMeteredRunWithoutACapWithExit2(t *testing.T) {
	_, id, decl := matrixLabCLI(t, "builder=runner:opencode@openrouter/vendor/model", false)
	code, _, errOut := runLab(t, "lab", "matrix", id, decl)
	if code != 2 || !strings.Contains(errOut, "no spending cap") {
		t.Fatalf("exit %d, stderr %q; want 2 naming the missing cap", code, errOut)
	}
}

func TestLabMatrixWithTheShownDigestWritesTheRunRecord(t *testing.T) {
	home, id, decl := matrixLabCLI(t, "builder=runner:opencode@openrouter/vendor/model", true)
	if code, _, errOut := runLab(t, "lab", "matrix", id, decl, "--approve", strings.Repeat("0", 64)); code != 2 || !strings.Contains(errOut, "not the plan") {
		t.Fatalf("a stale approval: exit %d, stderr %q; want 2", code, errOut)
	}
	_, out, _ := runLab(t, "lab", "--json", "matrix", id, decl)
	var plan struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", id, decl, "--approve", plan.Digest)
	var started struct {
		RunID    string `json:"run_id"`
		Launched bool   `json:"launched"`
		Note     string `json:"note"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil || started.RunID == "" || started.Launched || !strings.Contains(started.Note, "nothing was launched") {
		t.Fatalf("approve (exit %d): %s%s", code, out, errOut)
	}
	runs, _ := filepath.Glob(filepath.Join(abcdhome.Path(home, "lab"), "*", id, "state", "matrix", started.RunID, "run.json"))
	if len(runs) != 1 {
		t.Fatalf("the consented run's record is not in the lab")
	}
}

// A matrix refusal reaches the terminal with a control or direction character
// escaped: the core returns raw text, and a declaration's ids, models and a
// scanner's kinds carry what the person or the repository wrote.
func TestAMatrixRefusalIsPrintedEscaped(t *testing.T) {
	err := matrixFailure(errors.New("cell or\u202echeap: the model vendor/\x1b[2Jmodel is refused"))
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("matrixFailure = %v; want exit 2", err)
	}
	if strings.ContainsAny(ee.Msg, "\u202e\x1b") {
		t.Errorf("the refusal carries a control or direction character raw: %q", ee.Msg)
	}
	if !strings.Contains(ee.Msg, "cell or") || !strings.Contains(ee.Msg, "model is refused") {
		t.Errorf("the refusal lost its text: %q", ee.Msg)
	}
}
