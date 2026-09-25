package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/gittest"
)

const idBrief = `## Identity (canonical)

- **Title:** abcd — Agent-Based Configuration for Development
- **Tagline:** A host-agnostic configuration layer for intent-driven development.
- **Pitch:** A single Go binary that carries the why from idea to shipped
  reality, usable as a plugin in compatible agent harnesses.
`

const idConfig = `{
  "schema_version": 1,
  "block": {"file": ".abcd/development/brief.md", "heading": "Identity (canonical)"},
  "surfaces": [
    {"id": "readme-strapline", "files": ["README.md"], "kind": "regexp",
     "patterns": ["(?s)<p>(.*?)</p>"], "requires": ["tagline"], "template": "{tagline}"}
  ]
}`

const idCleanREADME = "<div>\n\n  <h1>abcd</h1>\n\n  <p>A host-agnostic configuration layer for intent-driven development.</p>\n\n</div>\n"

// The iss-143 README variant.
const idDriftedREADME = "<div>\n\n  <h1>abcd</h1>\n\n  <p>An opinionated, intent-driven development framework for product thinkers.</p>\n\n</div>\n"

func idRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func adoptedRepo(t *testing.T, readme string) string {
	return idRepo(t, map[string]string{
		".abcd/development/brief.md": idBrief,
		".abcd/positioning.json":     idConfig,
		"README.md":                  readme,
	})
}

// snapshotTree records content and mtime of every file under root.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		rel, _ := filepath.Rel(root, p)
		out[rel] = fi.ModTime().String() + "\x00" + string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestIdentityBareRendersTheBlockAndSurfaceStatus(t *testing.T) {
	t.Chdir(adoptedRepo(t, idCleanREADME))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"lint", "identity"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"abcd — Agent-Based Configuration for Development",
		"A host-agnostic configuration layer for intent-driven development.",
		"readme-strapline",
		"ok",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bare render missing %q:\n%s", want, out)
		}
	}
}

func TestIdentityJSONCarriesTheReport(t *testing.T) {
	t.Chdir(adoptedRepo(t, idDriftedREADME))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"lint", "identity", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	var rep positioning.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("identity --json is not a Report: %v\n%s", err, stdout.String())
	}
	if rep.Block.Tagline == "" || rep.Drifted() != 1 {
		t.Fatalf("report = %+v", rep)
	}
}

// AC3: render proposes; it never writes. Nothing under the repo may change,
// content or mtime.
func TestIdentityRenderWritesNothing(t *testing.T) {
	repo := adoptedRepo(t, idDriftedREADME)
	t.Chdir(repo)
	before := snapshotTree(t, repo)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "render"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "--- a/README.md") {
		t.Fatalf("render produced no diff to assert about:\n%s", stdout.String())
	}

	after := snapshotTree(t, repo)
	if len(before) != len(after) {
		t.Fatalf("render changed the file set: %d before, %d after", len(before), len(after))
	}
	for rel, was := range before {
		if now, ok := after[rel]; !ok || now != was {
			t.Errorf("render modified %s — it must write nothing", rel)
		}
	}
}

// No flag may turn the proposal into a write.
func TestIdentityRenderHasNoWriteFlag(t *testing.T) {
	repo := adoptedRepo(t, idDriftedREADME)
	t.Chdir(repo)
	for _, flag := range []string{"--write", "--apply", "--fix", "--in-place", "--force"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"identity", "render", flag}, &stdout, &stderr); code == 0 {
			t.Errorf("`identity render %s` was accepted; the proposal must never write", flag)
		}
	}
	data, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != idDriftedREADME {
		t.Fatal("README.md was rewritten by a rejected flag")
	}
}

func TestIdentityRenderPrintsTheProposedDiff(t *testing.T) {
	t.Chdir(adoptedRepo(t, idDriftedREADME))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "render"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"--- a/README.md",
		"+++ b/README.md",
		"-  <p>An opinionated, intent-driven development framework for product thinkers.</p>",
		"+  <p>A host-agnostic configuration layer for intent-driven development.</p>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render output missing %q:\n%s", want, out)
		}
	}
	// A diff is line-oriented: each marker must open its own line, or the
	// output is one unreadable smear and cannot be piped to `git apply`.
	for _, want := range []string{
		"\n--- a/README.md\n",
		"\n+++ b/README.md\n",
		"\n-  <p>An opinionated",
		"\n+  <p>A host-agnostic",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render output is not line-oriented, missing %q:\n%q", want, out)
		}
	}
	// Sanitising must not turn the diff's newlines into replacement glyphs.
	if strings.Contains(out, "README.md?") {
		t.Errorf("render output flattened its newlines:\n%q", out)
	}
}

// The patch the HUMAN sees must be as applicable as the one in --json. The
// core's CRLF round-trip is worthless if the render path flattens it, and a
// core-level test cannot catch that — this asserts it at the front door.
func TestIdentityRenderKeepsCRLFOnTheHumanPath(t *testing.T) {
	crlf := strings.ReplaceAll(idDriftedREADME, "\n", "\r\n")
	repo := idRepo(t, map[string]string{
		".abcd/development/brief.md": idBrief,
		".abcd/positioning.json":     idConfig,
		"README.md":                  crlf,
	})
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "render"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "-  <p>An opinionated, intent-driven development framework for product thinkers.</p>\r\n") {
		t.Errorf("human patch dropped the carriage return a CRLF surface needs:\n%q", out)
	}
	// And it is a patch git will take.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	runGitT(t, repo, "init", "-q")
	patch := filepath.Join(t.TempDir(), "p.patch")
	// Drop the two header lines the human render prefixes the diffs with.
	body := out[strings.Index(out, "--- a/"):]
	if err := os.WriteFile(patch, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", repo, "apply", "--check", patch)
	cmd.Env = gittest.Env(t)
	if o, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git apply --check refused the rendered patch: %v: %s\n%q", err, o, body)
	}
}

func TestIdentityRenderOnACleanRepoProposesNothing(t *testing.T) {
	t.Chdir(adoptedRepo(t, idCleanREADME))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "render"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "--- a/") {
		t.Fatalf("clean repo produced a diff:\n%s", stdout.String())
	}
}

// A repo that has not adopted the check is told how to, not crashed at.
func TestIdentityWithoutARegistryExplainsHowToAdopt(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint", "identity"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("un-adopted repo exited 0:\n%s", stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "identity init") {
		t.Errorf("no pointer to `abcd identity init`:\n%s", combined)
	}
}

// TestIdentityInitFaultExitsTwo pins that an init that cannot proceed takes the
// tree-wide fault code 2 (like `identity`, `identity render`, and every other
// "could not be evaluated" path), not the code 1 an already-rendered refusal
// reserves — init renders nothing on this path.
func TestIdentityInitFaultExitsTwo(t *testing.T) {
	repo := idRepo(t, map[string]string{"README.md": idCleanREADME})
	t.Chdir(repo)

	// Init with no --title/--tagline in a repo that has no block yet is a fault.
	var stdout, stderr bytes.Buffer
	code := Run([]string{"identity", "init"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("identity init fault exit = %d, want 2; stderr = %s", code, stderr.String())
	}
}

// AC1: init lands a parseable block and the pointer that records its location.
func TestIdentityInitScaffoldsTheBlockAndThePointer(t *testing.T) {
	repo := idRepo(t, map[string]string{"README.md": idCleanREADME})
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"identity", "init",
		"--title", "abcd — Agent-Based Configuration for Development",
		"--tagline", "A host-agnostic configuration layer for intent-driven development.",
		"--pitch", "A single Go binary that carries the why from idea to shipped reality."},
		&stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}

	cfg, ok, err := positioning.LoadConfig(repo)
	if err != nil || !ok {
		t.Fatalf("LoadConfig after init = %v, %v", ok, err)
	}
	block, err := positioning.ParseBlock(repo, cfg.Block)
	if err != nil {
		t.Fatalf("the scaffolded block does not parse: %v", err)
	}
	if block.Tagline != "A host-agnostic configuration layer for intent-driven development." {
		t.Errorf("Tagline = %q", block.Tagline)
	}
	if block.Title == "" || block.Pitch == "" {
		t.Errorf("block = %+v", block)
	}
	// The check the init just enabled must now run, and find no drift: the
	// repo's README already carries the tagline it was interviewed for.
	var auditOut, auditErr bytes.Buffer
	Run([]string{"lint", "--json"}, &auditOut, &auditErr)
	var res struct {
		Findings []struct {
			RuleID  string `json:"ruleId"`
			Message string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(auditOut.Bytes(), &res); err != nil {
		t.Fatalf("lint --json: %v\n%s", err, auditOut.String())
	}
	for _, f := range res.Findings {
		if f.RuleID == "identity-positioning" {
			t.Errorf("audit after init reports positioning drift: %s", f.Message)
		}
	}
}

// The pitch is skippable at onboarding; title and tagline are not.
func TestIdentityInitRequiresTitleAndTagline(t *testing.T) {
	t.Chdir(idRepo(t, map[string]string{"README.md": idCleanREADME}))
	for _, args := range [][]string{
		{"identity", "init", "--tagline", "t"},
		{"identity", "init", "--title", "T"},
		{"identity", "init"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code == 0 {
			t.Errorf("%v was accepted without both required answers", args)
		}
	}
}

func TestIdentityInitAcceptsNoPitch(t *testing.T) {
	repo := idRepo(t, map[string]string{"README.md": idCleanREADME})
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "init", "--title", "T", "--tagline", "A host-agnostic configuration layer for intent-driven development."}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	cfg, _, err := positioning.LoadConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	block, err := positioning.ParseBlock(repo, cfg.Block)
	if err != nil {
		t.Fatalf("block without a pitch does not parse: %v", err)
	}
	if block.Pitch != "" {
		t.Errorf("Pitch = %q, want empty", block.Pitch)
	}
}

// Onboarding detects an existing block and registers it rather than
// re-interviewing and minting a second canon.
func TestIdentityInitDetectsAnExistingBlockInsteadOfReinterviewing(t *testing.T) {
	repo := idRepo(t, map[string]string{
		".abcd/development/brief.md": idBrief,
		"README.md":                  idCleanREADME,
	})
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"identity", "init",
		"--file", ".abcd/development/brief.md", "--heading", "Identity (canonical)"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "existing") {
		t.Errorf("init did not report that it adopted the existing block:\n%s", stdout.String())
	}
	// The block file is untouched — no second canon, no rewrite.
	data, err := os.ReadFile(filepath.Join(repo, ".abcd/development/brief.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != idBrief {
		t.Fatal("init rewrote an existing identity block")
	}
	cfg, ok, err := positioning.LoadConfig(repo)
	if err != nil || !ok || cfg.Block.Heading != "Identity (canonical)" {
		t.Fatalf("pointer not registered: %+v %v %v", cfg, ok, err)
	}
}

// Re-running init over a registered block is a no-op, not a clobber.
func TestIdentityInitIsIdempotent(t *testing.T) {
	repo := adoptedRepo(t, idCleanREADME)
	t.Chdir(repo)
	before := snapshotTree(t, repo)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"identity", "init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	for rel, was := range snapshotTree(t, repo) {
		if before[rel] != was {
			t.Errorf("re-running init modified %s", rel)
		}
	}
}

// A heading that already exists but carries incomplete bullets cannot be
// repaired by appending a second copy of itself: init must refuse before it
// writes anything, name the file, and stay refusing on a re-run.
func TestIdentityInitRefusesAnIncompleteExistingBlock(t *testing.T) {
	const partial = "## Identity (canonical)\n\n- **Title:** Widget\n"
	repo := idRepo(t, map[string]string{
		".abcd/development/IDENTITY.md": partial,
		"README.md":                     idCleanREADME,
	})
	t.Chdir(repo)

	for attempt := 1; attempt <= 3; attempt++ {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"identity", "init", "--title", "Widget", "--tagline", "A sharper widget."}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("attempt %d: init accepted an incomplete existing block", attempt)
		}
		if !strings.Contains(stderr.String()+stdout.String(), ".abcd/development/IDENTITY.md") {
			t.Errorf("attempt %d: refusal does not name the file to repair: %s", attempt, stderr.String())
		}
	}
	// Nothing was appended, and no registry was left half-adopted.
	data, err := os.ReadFile(filepath.Join(repo, ".abcd/development/IDENTITY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != partial {
		t.Fatalf("init rewrote or appended to the existing block:\n%s", data)
	}
	if _, ok, err := positioning.LoadConfig(repo); err != nil || ok {
		t.Fatalf("a refused init left a registry behind: ok=%v err=%v", ok, err)
	}
}

// A repo whose registry already points somewhere must never be silently
// repointed by an init with different answers.
func TestIdentityInitRefusesToRepointAnAdoptedRegistry(t *testing.T) {
	repo := adoptedRepo(t, idCleanREADME)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"identity", "init", "--title", "Other", "--tagline", "Different."}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("init overwrote an adopted registry")
	}
	cfg, _, err := positioning.LoadConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Block.File != ".abcd/development/brief.md" {
		t.Fatalf("registry was repointed to %q", cfg.Block.File)
	}
}
