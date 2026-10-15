package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/decisions"
)

// decide_record_test.go — the verb's three forms (spc-2610110931381316,
// decision 17): `abcd decide "<text>"` records a decision, bare `abcd decide`
// lists them, and `abcd decide --adr "<title>"` mints an ADR. The ADR form's
// own behaviour is held by decide_surface_test.go and decide_root_test.go.

// decisionsRepo is a git working tree holding the shared tier, chdir'd into.
func decisionsRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	gitInitAt(t, root)
	root = realPath(t, root)
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

var dcnPathRe = regexp.MustCompile(`^\.abcd/work/decisions/dcn-[0-9]{16}-[a-z0-9-]+\.md$`)

// TestDecideRecordsADecisionFromTheCLI: the operand form reaches
// decisions.Record and writes one decision record, never an ADR.
func TestDecideRecordsADecisionFromTheCLI(t *testing.T) {
	repo := decisionsRepo(t)

	var d decisions.Decision
	out := runCLI(t, "decide", "Decisions are one flat file each", "--json")
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("decide --json: not JSON: %v\n%s", err, out)
	}
	if !dcnPathRe.MatchString(d.Path) {
		t.Errorf("recorded path %q is not a decision record in the folder", d.Path)
	}
	if d.Text != d.Date+" — Decisions are one flat file each" {
		t.Errorf("recorded text = %q, want `<date> — <text>`", d.Text)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(d.Path))); err != nil {
		t.Errorf("the reported record is not on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(decide.ADRsRelDir))); err == nil {
		t.Error("recording a decision laid the ADR store: the operand form must not mint an ADR")
	}

	text := string(runCLI(t, "decide", "A second decision"))
	if !strings.Contains(text, "decision recorded") || !strings.Contains(text, ".abcd/work/decisions/dcn-") {
		t.Errorf("the text render does not name the recording and its path:\n%s", text)
	}
}

// TestBareDecideListsTheDecisions: with no operand the verb lists the folder
// newest first, in both forms, and writes nothing.
func TestBareDecideListsTheDecisions(t *testing.T) {
	repo := decisionsRepo(t)

	empty := string(runCLI(t, "decide"))
	if !strings.Contains(empty, "no decisions recorded") {
		t.Errorf("bare decide over an empty folder does not say so:\n%s", empty)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(decisions.RelDir))); err == nil {
		t.Error("listing laid the decisions folder")
	}

	var first, second decisions.Decision
	if err := json.Unmarshal(runCLI(t, "decide", "The first decision", "--json"), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(runCLI(t, "decide", "The second decision", "--json"), &second); err != nil {
		t.Fatal(err)
	}

	text := string(runCLI(t, "decide"))
	for _, d := range []decisions.Decision{first, second} {
		if !strings.Contains(text, d.ID+"\n"+d.Text+"\n") {
			t.Errorf("the listing does not carry %s as its id then its body:\n%s", d.ID, text)
		}
	}

	var listed []decisions.Decision
	raw := runCLI(t, "decide", "--json")
	if err := json.Unmarshal(raw, &listed); err != nil {
		t.Fatalf("bare decide --json is not an array of records: %v\n%s", err, raw)
	}
	if len(listed) != 2 {
		t.Fatalf("bare decide --json listed %d records, want 2:\n%s", len(listed), raw)
	}
	if listed[0].ID < listed[1].ID {
		t.Errorf("the listing is not newest first: %s before %s", listed[0].ID, listed[1].ID)
	}
}

// TestDecideAdrMintsAnADR: --adr takes the operand as an ADR title and mints
// exactly what bare `abcd decide "<title>"` minted before, and records no
// decision.
func TestDecideAdrMintsAnADR(t *testing.T) {
	repo := decisionsRepo(t)

	var d decide.Decision
	out := runCLI(t, "decide", "--adr", "Decisions are one flat file each", "--json")
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("decide --adr --json: not JSON: %v\n%s", err, out)
	}
	if !regexp.MustCompile(`^adr-[0-9]{16}$`).MatchString(d.ID) {
		t.Errorf("--adr minted %q, not an ADR id", d.ID)
	}
	if !strings.HasPrefix(d.Path, ".abcd/development/decisions/adrs/") {
		t.Errorf("--adr wrote %q, outside the ADR store", d.Path)
	}
	body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(d.Path)))
	if err != nil || !strings.Contains(string(body), "status: proposed") {
		t.Errorf("the ADR skeleton is not on disk as proposed: %v\n%s", err, body)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(decisions.RelDir))); err == nil {
		t.Error("--adr recorded a decision as well as minting the ADR")
	}
}

// TestDecideFormRefusalsExitTwo: every operand fault of the three forms exits
// 2 and writes nothing.
func TestDecideFormRefusalsExitTwo(t *testing.T) {
	repo := decisionsRepo(t)
	for _, tc := range []struct {
		name string
		args []string
		says string
	}{
		{"adr-without-a-title", []string{"decide", "--adr"}, `abcd decide --adr "<title>"`},
		{"two-operands", []string{"decide", "one", "two"}, "one quoted operand at most"},
		{"unslugable-text", []string{"decide", "///"}, "no letters or digits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCLIErr(t, tc.args...)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("%v exited %d, want 2:\n%s", tc.args, code, out)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("%v refusal %q does not say %q", tc.args, err, tc.says)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "development")); err == nil {
		t.Error("a refusal laid the ADR store")
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(decisions.RelDir))); err == nil {
		t.Error("a refusal laid the decisions folder")
	}

	// Without the shared tier there is nowhere to record, and the refusal says
	// how to lay it.
	if err := os.RemoveAll(filepath.Join(repo, ".abcd")); err != nil {
		t.Fatal(err)
	}
	out, err := runCLIErr(t, "decide", "A decision with nowhere to go")
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "ahoy install") {
		t.Errorf("recording without the shared tier = exit %d, %v; want exit 2 naming `ahoy install`\n%s", code, err, out)
	}
}

// TestDecideOutsideAnyRepositoryRefusesEveryForm: the record and list forms
// resolve the checkout root as the ADR form does, and refuse outside one.
func TestDecideOutsideAnyRepositoryRefusesEveryForm(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)
	for _, args := range [][]string{{"decide", "a decision outside any repository"}, {"decide"}} {
		out, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "repository") {
			t.Errorf("%v outside a repository = exit %d, %v; want exit 2 naming the repository\n%s", args, code, err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(plain, ".abcd")); err == nil {
		t.Error("a refused call laid .abcd/ outside the repository")
	}
}
