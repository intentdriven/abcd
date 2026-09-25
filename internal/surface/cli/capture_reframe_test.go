package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// reframeCLIRepo lays out a checkout carrying the three frame surfaces and one
// committed reading item, and changes into it.
func reframeCLIRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/development/brief/01-product/06-framing.md", "# Framing\n\n## Construal\n\nThe first construal.\n")
	r.Write(".abcd/development/brief/01-product/04-scope.md", "# Scope\n\nThe first scope.\n")
	r.Write(".abcd/development/brief/glossary/core/term.md", "# Term\n")
	r.Write(".abcd/work/issues/readings/rdg-1/rdi-11.md", "---\nid: rdi-11\n---\n")
	r.Commit("base")
	t.Chdir(r.Root())
	return r
}

// `capture reframe` is a front door (spc-2609020626048705): reachable from the
// CLI, and each of its renders names the half it wrote.
func TestCaptureReframeSurface(t *testing.T) {
	r := reframeCLIRepo(t)
	const ground = "the detection reading showed the construal was the wrong frame"

	out := runCLI(t, "capture", "reframe", "--occasioned-by", "rdi-11", "--grounds", ground, "--open")
	if !strings.Contains(string(out), "first half written") || !strings.Contains(string(out), "capture reframe --complete rfm-") {
		t.Fatalf("the open render does not name its half and the completion:\n%s", out)
	}
	id := strings.Fields(strings.TrimSpace(string(out[strings.Index(string(out), "rfm-"):])))[0]

	// --complete takes the record id alone.
	if _, err := runCLIErr(t, "capture", "reframe", "--complete", id, "--occasioned-by", "rdi-11"); err == nil {
		t.Fatal("--complete with --occasioned-by was accepted")
	}

	r.Write(".abcd/development/brief/01-product/06-framing.md", "# Framing\n\n## Construal\n\nThe second construal.\n")
	r.Git("add", ".abcd/development/brief")
	r.Git("commit", "-q", "-m", "rewrite the construal")
	out = runCLI(t, "capture", "reframe", "--complete", id)
	if !strings.Contains(string(out), id+"  completed across 1 commit(s)") || !strings.Contains(string(out), "construal") {
		t.Fatalf("the completion render does not name its half:\n%s", out)
	}

	r.Write(".abcd/development/brief/01-product/04-scope.md", "# Scope\n\nThe second scope.\n")
	r.Git("add", ".abcd/development/brief")
	r.Git("commit", "-q", "-m", "rewrite the scope")
	out = runCLI(t, "capture", "reframe", "--occasioned-by", "rdi-11", "--grounds", ground, "--json")
	var res struct {
		ID      string   `json:"id"`
		Half    string   `json:"half"`
		Changed []string `json:"changed"`
		Commits int      `json:"commits"`
		Before  struct {
			Scope string `json:"scope"`
		} `json:"before"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("reframe output not JSON: %v\n%s", err, out)
	}
	if res.Half != "whole" || len(res.Changed) != 1 || res.Changed[0] != "scope" || res.Commits != 1 || len(res.Before.Scope) != 64 {
		t.Fatalf("whole write = %+v", res)
	}

	// Neither flag set is a usage error naming what is missing.
	if _, err := runCLIErr(t, "capture", "reframe", "--grounds", ground); err == nil ||
		!strings.Contains(err.Error(), "--occasioned-by") {
		t.Fatalf("a reframe without an occasion: err = %v", err)
	}
}
