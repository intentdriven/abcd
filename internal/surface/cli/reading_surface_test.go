package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/reading"
)

// readingRepo lays out the minimum an `abcd reading assemble` run needs, so the
// surface test proves the WIRING end to end rather than mocking the core.
func readingRepo(t *testing.T) string {
	t.Helper()
	return readingRepoAt(t, t.TempDir())
}

// readingRepoAt builds the same fixture at a caller-chosen root, so a test about
// PATH RESOLUTION can place the repository deep enough that its two candidate
// bases both stay inside the test's own temporary directory. A test that wrote
// outside it would leave state behind and pass or fail on the leavings.
func readingRepoAt(t *testing.T, root string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The surface test drives the real core, so the repository it builds needs
	// the committed preset configuration an assembly now applies for the
	// position it was invoked at. One preset, naming every kind at every
	// assembling position, so these tests keep the item sets they were written
	// against.
	write(".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "default": {
      "positions": {
        "widening": {"kinds": ["brief-section", "glossary-term", "intent-projection", "discipline", "spec", "source", "test", "doc", "config"], "records": [], "paths": []},
        "entailment": {"kinds": ["brief-section", "glossary-term", "intent-projection", "discipline", "spec", "source", "test", "doc", "config"], "records": [], "paths": []},
        "comparative": {"kinds": ["discipline"], "records": [], "paths": []},
        "detection": {"kinds": ["brief-section", "glossary-term", "intent-projection", "discipline", "spec", "source", "test", "doc", "config"], "records": [], "paths": []}
      }
    }
  }
}
`)
	write(".abcd/record-lint.json", `{"schema_version": 1, "rules": {"record_schema": {"enabled": true,
  "severity": "blocker", "record_stores": {"itd": ".abcd/development/intents", "spc": ".abcd/development/specs",
  "rdi": ".abcd/work/issues/readings"}}}}`)
	write(".abcd/development/brief/01-product/06-framing.md", "# Framing\n\n## Construal\n\nA gap in the record.\n")
	write(".abcd/development/brief/02-constraints/03-invariants.md", "# Invariants\n\n1. One core.\n")
	// The rest of brief current text (itd-194). A walk row's source directory
	// must exist or the run refuses, and the table names six chapters.
	write(".abcd/development/brief/00-meta.md", "# Meta\n\nHow this brief is organised.\n")
	write(".abcd/development/brief/04-surfaces/23-reading.md", "# The reading surface\n\nWhat the verb does.\n")
	write(".abcd/development/brief/05-internals/03-configuration.md", "# Configuration\n\nWhere settings live.\n")
	write(".abcd/development/brief/06-delivery/01-shipping.md", "# Shipping\n\nHow a release is cut.\n")
	write(".abcd/development/specs/open/spc-1-a-spec.md", "---\nid: spc-1\n---\n\n# A spec\n\nThe mechanics.\n")
	write(".abcd/development/intents/shipped/itd-1-an-intent.md",
		"---\nid: itd-1\nspec_id: spc-1\n---\n\n# An intent\n\n## Press Release\n\nThe promise.\n")
	write(".abcd/development/intents/disciplines/itd-2-a-discipline.md",
		"---\nid: itd-2\n---\n\n# A discipline\n\nA standing commitment.\n")
	write(".abcd/development/intents/drafts/itd-3-a-draft.md",
		"---\nid: itd-3\n---\n\n# A draft\n\n## Press Release\n\nA candidate.\n")
	write(".abcd/development/brief/glossary/core/construal.md", "# Construal\n\nWhat it is treated as.\n")
	write("docs/reference/thing.md", "# Thing\n\nReference prose.\n")
	write("README.md", "# The repository\n")
	write("go.mod", "module example\n\ngo 1.24\n")

	gitCmd(t, root, "init", "-q", "-b", "main")
	gitCmd(t, root, "add", "-A")
	gitCommit(t, root, "commit", "-q", "-m", "fixture")
	return root
}

// treeOf lists every non-git path under root, so a test can assert that a dry
// run wrote nothing.
func treeOf(t *testing.T, root string) string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) || info.IsDir() {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}

// TestReadingVerbReachesBothPlanes holds "wired or it isn't done": the verb is
// registered in the command tree AND reachable from the plugin markdown surface.
func TestReadingVerbReachesBothPlanes(t *testing.T) {
	root := NewRootCommand()
	var readingCmd, assembleCmd bool
	for _, c := range root.Commands() {
		if c.Name() != "reading" {
			continue
		}
		readingCmd = true
		for _, sub := range c.Commands() {
			if sub.Name() == "assemble" {
				assembleCmd = true
			}
		}
	}
	if !readingCmd {
		t.Fatal("the command tree registers no `reading` verb")
	}
	if !assembleCmd {
		t.Fatal("`reading` registers no `assemble` sub-verb")
	}

	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "reading.md"))
	if err != nil {
		t.Fatalf("read the plugin surface: %v", err)
	}
	body := string(raw)
	for _, want := range []string{"assemble", "--position", "--target"} {
		if !strings.Contains(body, want) {
			t.Errorf("commands/reading.md does not mention %q", want)
		}
	}
}

// repoRootFromTest walks up to the directory holding go.mod.
func repoRootFromTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}

// TestAssembleRefusesFreeTextOperands is the invocation interface's whole claim:
// position and target state, and nothing else. A free-text operand has nowhere
// to go, so there is no channel a framing could travel through.
func TestAssembleRefusesFreeTextOperands(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	out, err := runCLIErr(t, "reading", "assemble", "read this against the ledger",
		"--position", "widening", "--target", "HEAD", "--dry-run")
	if err == nil {
		t.Fatalf("a free-text operand was accepted:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("a free-text operand exited %d, want 2", code)
	}
}

// TestAssembleRefusesAScopeOperand is itd-2609021003095168 ac-1 (framework v4
// section 8.2 and ruling M8; companion v4 section 4.1; adr-2609021016286571,
// which supersedes adr-58).
//
// The design fixes the invocation at a position and a target state. The scope
// operand adr-58 admitted is withdrawn, so --scope is an unknown flag: the verb
// refuses it, exits 2, and writes nothing. The refusal is required to name the
// two operands the design admits, because an operator refused an operand and
// told only "unknown flag" is not told what the invocation IS.
func TestAssembleRefusesAScopeOperand(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	out, err := runCLIErr(t, "reading", "assemble",
		"--position", "widening", "--target", "HEAD", "--scope", "everything", "--dry-run")
	if err == nil {
		t.Fatalf("--scope was accepted; the invocation is a position and a target state:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("--scope exited %d, want 2", code)
	}
	refusal := err.Error() + "\n" + string(out)
	if !strings.Contains(refusal, "--scope") {
		t.Errorf("the refusal does not name the operand it refused:\n%s", refusal)
	}
	for _, want := range []string{"--position", "--target"} {
		if !strings.Contains(refusal, want) {
			t.Errorf("the refusal does not name %s; an operator refused an operand is told which "+
				"two the design admits:\n%s", want, refusal)
		}
	}

	// Nothing may be written by a run that refuses at the door.
	runs := filepath.Join(repo, ".abcd", ".work.local", "scratch", "reading-runs")
	if entries, err := os.ReadDir(runs); err == nil && len(entries) > 0 {
		t.Errorf("a refused invocation wrote %d run directory/ies under %s", len(entries), runs)
	}
}

// TestPositionTokenIsClosed holds the first operand at the front door.
func TestPositionTokenIsClosed(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	for _, bad := range []string{"framing", "Widening", "", "widening extra"} {
		args := []string{"reading", "assemble", "--target", "HEAD", "--dry-run"}
		if bad != "" {
			args = append(args, "--position", bad)
		}
		if out, err := runCLIErr(t, args...); err == nil {
			t.Errorf("position %q was accepted:\n%s", bad, out)
		}
	}
	if _, err := runCLIErr(t, "reading", "assemble", "--position", "widening", "--target", "HEAD", "--dry-run"); err != nil {
		t.Errorf("the widening position was refused: %v", err)
	}
}

// TestTargetRefusesBranchAndTag holds the second operand at the front door, and
// with it the non-zero exit the evals bind to.
func TestTargetRefusesBranchAndTag(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	for _, bad := range []string{"main", "v1.0.0", "HEAD~1"} {
		out, err := runCLIErr(t, "reading", "assemble", "--position", "widening", "--target", bad, "--dry-run")
		if err == nil {
			t.Errorf("target %q was accepted:\n%s", bad, out)
			continue
		}
		if code := exitCodeOf(err); code == 0 {
			t.Errorf("target %q exited 0", bad)
		}
	}
	if out, err := runCLIErr(t, "reading", "assemble", "--position", "widening", "--dry-run"); err == nil {
		t.Errorf("a missing --target was accepted:\n%s", out)
	}
}

// TestAssembleDryRunWritesNothing holds the render-without-writing idiom at the
// front door.
func TestAssembleDryRunWritesNothing(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)
	before := treeOf(t, repo)

	out := runCLI(t, "reading", "assemble", "--position", "widening", "--target", "HEAD", "--dry-run", "--json")
	var res reading.AssembleResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("the --json envelope does not decode: %v\n%s", err, out)
	}
	if res.Written {
		t.Error("a dry run reported a write")
	}
	if res.ItemCount == 0 {
		t.Error("a dry run assembled nothing")
	}
	if !strings.HasPrefix(res.RunID, "rdg-") {
		t.Errorf("run id %q does not carry the readings family tag", res.RunID)
	}
	if after := treeOf(t, repo); after != before {
		t.Errorf("a dry run changed the tree:\nbefore\n%s\nafter\n%s", before, after)
	}
}

// TestAssembleWritesTwoArtefactsIntoANamedDirectory holds the interface the
// read-block and amnesia evals bind to: the assembled input and the manifest as
// two separate files in a directory the operator names.
func TestAssembleWritesTwoArtefactsIntoANamedDirectory(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)
	outDir := filepath.Join(t.TempDir(), "run")

	raw := runCLI(t, "reading", "assemble", "--position", "entailment", "--target", "HEAD",
		"--out", outDir, "--dry-run", "--json")
	var res reading.AssembleResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("the --json envelope does not decode: %v\n%s", err, raw)
	}
	if !res.Written {
		t.Fatal("an assembly into a named output directory reported no write")
	}
	bundleRaw, err := os.ReadFile(filepath.Join(outDir, reading.BundleFileName))
	if err != nil {
		t.Fatalf("read the assembled input: %v", err)
	}
	var bundle reading.Bundle
	if err := json.Unmarshal(bundleRaw, &bundle); err != nil {
		t.Fatalf("the assembled input does not decode: %v", err)
	}
	if len(bundle.Items) != res.ItemCount {
		t.Errorf("the assembled input holds %d items, the result reports %d", len(bundle.Items), res.ItemCount)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(outDir, reading.ManifestFileName))
	if err != nil {
		t.Fatalf("read the manifest: %v", err)
	}
	if _, err := reading.DecodeManifest(manifestRaw); err != nil {
		t.Fatalf("the written manifest does not decode strictly: %v", err)
	}
}

// TestBareReadingRenders holds the parent's read-only status render.
func TestBareReadingRenders(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	text := string(runCLI(t, "reading"))
	if !strings.Contains(text, reading.AssemblerVersion()) {
		t.Errorf("the bare render does not name the assembler version:\n%s", text)
	}
	for _, p := range reading.Positions() {
		if !strings.Contains(text, string(p)) {
			t.Errorf("the bare render does not name the %s position:\n%s", p, text)
		}
	}
	out := runCLI(t, "reading", "--json")
	var status reading.Status
	if err := json.Unmarshal(out, &status); err != nil {
		t.Fatalf("the --json status does not decode: %v\n%s", err, out)
	}
	if status.AssemblerVersion != reading.AssemblerVersion() {
		t.Errorf("status names assembler version %q", status.AssemblerVersion)
	}
}

// TestRelativeOutResolvesAgainstTheWorkingDirectory: the core takes a relative
// output directory against the repository root, which is right for the default
// it computes itself and wrong for a path an operator typed. Run from a
// subdirectory, `--out ../../x` must mean what the shell means by it.
func TestRelativeOutResolvesAgainstTheWorkingDirectory(t *testing.T) {
	repo := readingRepoAt(t, filepath.Join(t.TempDir(), "a", "b", "repo"))
	sub := filepath.Join(repo, "docs")
	t.Chdir(sub)

	raw := runCLI(t, "reading", "assemble", "--position", "widening", "--target", "HEAD",
		"--out", "../../outside-run", "--json")
	var res reading.AssembleResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("the --json envelope does not decode: %v\n%s", err, raw)
	}

	fromCwd := filepath.Join(filepath.Dir(filepath.Dir(sub)), "outside-run")
	fromRepoRoot := filepath.Join(filepath.Dir(filepath.Dir(repo)), "outside-run")
	if fromCwd == fromRepoRoot {
		t.Fatal("the two bases coincide, so this test proves nothing")
	}
	if _, err := os.Stat(filepath.Join(fromCwd, reading.BundleFileName)); err != nil {
		t.Errorf("the artefacts did not land beside the working directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fromRepoRoot, reading.BundleFileName)); err == nil {
		t.Error("the artefacts landed against the repository root instead")
	}
	// The core is handed the resolved path; the operator is shown what they
	// typed. An absolute path nobody asked for on the success surface is a local
	// path leaving the machine the moment the host reports the field.
	if res.OutDir != "../../outside-run" {
		t.Errorf("out_dir is %q, want the operator's own string", res.OutDir)
	}
	if filepath.IsAbs(res.OutDir) {
		t.Errorf("out_dir %q is an absolute path the operator did not type", res.OutDir)
	}
}

// TestOutRefusalsQuoteTheOperatorsOwnPath: a refusal that interpolates the
// resolved absolute directory puts a local path on the error surface, and the
// root redaction cannot reach it when the working directory is not a prefix.
func TestOutRefusalsQuoteTheOperatorsOwnPath(t *testing.T) {
	repo := readingRepo(t)
	// Run from a SUBDIRECTORY, so the resolved path is not under the working
	// directory and the root redaction has no prefix to strip. That is the case
	// the leak actually lives in.
	t.Chdir(filepath.Join(repo, "docs"))

	out, err := runCLIErr(t, "reading", "assemble", "--position", "widening",
		"--target", "HEAD", "--out", "../run-dir")
	if err == nil {
		t.Fatalf("an output directory inside the table's reach was accepted:\n%s", out)
	}
	msg := err.Error() + string(out)
	if !strings.Contains(msg, "../run-dir") {
		t.Errorf("the refusal does not quote the operator's own path: %s", msg)
	}
	if strings.Contains(msg, repo) {
		t.Errorf("the refusal quotes a resolved absolute path: %s", msg)
	}
}

// TestIngestRequiresOutputJSON holds the ingest verb's invocation interface: one
// operand, required, and no positional argument. There is no position operand
// and no regime operand, because the output states its own and the regime is the
// definition's — the standing guard on that is TestNoOperatorSurfaceSetsARegime.
func TestIngestRequiresOutputJSON(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)

	for _, args := range [][]string{
		{"reading", "ingest"},
		{"reading", "ingest", "some-output.json"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Errorf("%v was accepted:\n%s", args, out)
			continue
		}
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("%v exited %d, want 2", args, code)
		}
	}

	// The flag is registered and reaches the core: a path that resolves to
	// nothing is a refusal from the verb, not a cobra "unknown flag".
	out, err := runCLIErr(t, "reading", "ingest", "--reading-json", "absent.json")
	if err == nil {
		t.Fatalf("an absent output was accepted:\n%s", out)
	}
	if strings.Contains(err.Error()+string(out), "unknown flag") {
		t.Fatalf("--reading-json is not registered on the verb: %v\n%s", err, out)
	}
	if !strings.Contains(err.Error()+string(out), "reading ingest") {
		t.Errorf("the refusal does not name the verb: %v\n%s", err, out)
	}
	// The verb's name is printed ONCE. The refusal message is load-bearing for
	// this verb, and a doubled prefix is the first thing a reader skims past.
	if strings.Contains(err.Error()+string(out), "reading ingest: reading:") {
		t.Errorf("the refusal doubles its prefix: %v\n%s", err, out)
	}
}

// TestIngestReachesBothPlanes holds "wired or it isn't done" for the ingest verb:
// registered in the command tree AND reachable from the plugin markdown surface,
// with the operand the page tells a host to pass.
func TestIngestReachesBothPlanes(t *testing.T) {
	var found bool
	for _, c := range NewRootCommand().Commands() {
		if c.Name() != "reading" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "ingest" {
				continue
			}
			found = true
			if sub.Flags().Lookup("reading-json") == nil {
				t.Error("`reading ingest` registers no --reading-json flag")
			}
		}
	}
	if !found {
		t.Fatal("the command tree registers no `reading ingest` sub-verb")
	}

	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "reading.md"))
	if err != nil {
		t.Fatalf("read the plugin surface: %v", err)
	}
	body := string(raw)
	// The keys the page tells a host to REPORT are part of the wiring: a key
	// named on the page and absent from the result is a host instruction that
	// cannot be followed, and a durable delete the page never names is the
	// defect the sweep's own comment warns against.
	for _, want := range []string{"reading ingest", "--reading-json", "refusal.json", "pattern",
		"refused_items", "cleared_stages", "rolled_back_records"} {
		if !strings.Contains(body, want) {
			t.Errorf("commands/reading.md does not mention %q", want)
		}
	}
	for _, want := range []string{"refused_items", "cleared_stages", "rolled_back_records", "refusal_record"} {
		if !strings.Contains(string(mustMarshalIngestFields(t)), want) {
			t.Errorf("the ingest result carries no %q field for the page to report", want)
		}
	}
}

// TestIngestPayloadFlagNamesItsContent: the payload flag is named for what the
// JSON CONTAINS, the idiom its four siblings follow (--review-json,
// --verdict-json twice, --changelog-json), not for the role the flag plays.
// Role is the ambiguous axis on this verb: the global --json means the opposite
// direction of travel -- how abcd renders its own result, against a path to
// what a reader returned -- and the verb's own example puts both on one line.
func TestIngestPayloadFlagNamesItsContent(t *testing.T) {
	// Registered and reaching the core: a path that resolves to nothing is a
	// refusal from the verb, not a cobra "unknown flag".
	out, err := runCLIErr(t, "reading", "ingest", "--reading-json", "absent.json")
	if err == nil {
		t.Fatalf("an absent payload was accepted:\n%s", out)
	}
	if strings.Contains(err.Error()+string(out), "unknown flag") {
		t.Fatalf("--reading-json is not registered on the verb: %v\n%s", err, out)
	}
	// The role-named spelling is gone rather than aliased. An alias would keep
	// the collision the rename exists to remove, and the verb has no users.
	out, err = runCLIErr(t, "reading", "ingest", "--output-json", "absent.json")
	if err == nil {
		t.Fatalf("--output-json was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error()+string(out), "unknown flag") {
		t.Errorf("--output-json is still registered: %v\n%s", err, out)
	}
}

// parkedRunForIngest assembles one run in repo and installs the position's
// SHIPPED definition beside it, returning the two references an ingest payload
// has to cite. Three ingest cases need the same world; building it once keeps
// the fixture from drifting into three versions of itself.
func parkedRunForIngest(t *testing.T, srcRoot, repo string, pos reading.Position) (runID, manifestHash string, def reading.Definition) {
	t.Helper()
	out := runCLI(t, "reading", "assemble", "--position", string(pos), "--target", "HEAD", "--json")
	var assembled struct {
		RunID        string `json:"run_id"`
		ManifestHash string `json:"manifest_hash"`
	}
	if err := json.Unmarshal(out, &assembled); err != nil {
		t.Fatalf("decode the assemble render: %v\n%s", err, out)
	}
	def, err := reading.LoadDefinition(srcRoot, pos)
	if err != nil {
		t.Fatalf("load the shipped %s definition: %v", pos, err)
	}
	// The definition the run reads under has to be present in the repository the
	// verb is pointed at, so the fixture takes the shipped one verbatim.
	defRaw, err := os.ReadFile(filepath.Join(srcRoot, filepath.FromSlash(def.Path)))
	if err != nil {
		t.Fatal(err)
	}
	defPath := filepath.Join(repo, filepath.FromSlash(def.Path))
	if err := os.MkdirAll(filepath.Dir(defPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defPath, defRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	return assembled.RunID, assembled.ManifestHash, def
}

// detectionPayloadFile writes one legal detection payload, with a caller-chosen
// regime claim, and returns the path the verb reads.
func detectionPayloadFile(t *testing.T, runID, manifestHash, regime string, def reading.Definition) string {
	t.Helper()
	payload := map[string]any{
		"_type": "abcd.reading.output/1", "run_id": runID,
		"position": "detection", "regime": regime,
		"manifest_sha256": manifestHash,
		"instrument": map[string]any{
			"model": "a-model", "definition_sha256": def.SHA256,
			"assembler_version": reading.AssemblerVersion(),
		},
		"items": []any{map[string]any{
			"pattern": "the pattern this reading read under",
			"tension": "the record says one thing", "constraint_in_play": "a stated constraint",
			"why_a_tension": "the two cannot both hold",
		}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "output.json")
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return outPath
}

// TestIngestExecutesEndToEnd is the other half of the wiring claim: the verb is
// registered AND it runs. A surface test that only walked the command tree would
// pass against a sub-command whose RunE was never reached.
func TestIngestExecutesEndToEnd(t *testing.T) {
	// The source checkout is resolved BEFORE the working directory moves: the
	// shipped definition is read from this repository, and repoRootFromTest walks
	// up from the working directory, which is about to be a temporary tree with a
	// go.mod of its own.
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Chdir(repo)

	// One assembled run, through the sibling verb, so the ingest resolves a run
	// this repository actually parked.
	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)

	raw2 := runCLI(t, "reading", "ingest", "--reading-json", outPath, "--json")
	var res reading.IngestResult
	if err := json.Unmarshal(raw2, &res); err != nil {
		t.Fatalf("decode the ingest render: %v\n%s", err, raw2)
	}
	if len(res.Records) != 1 {
		t.Fatalf("the ingest landed %d record(s), want 1:\n%s", len(res.Records), raw2)
	}
	if res.Regime != def.Regime {
		t.Errorf("the run recorded regime %q, want the definition's %q", res.Regime, def.Regime)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(res.RunRecordPath))); err != nil {
		t.Errorf("the commit marker is not on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(res.Records[0].Path))); err != nil {
		t.Errorf("the reading record is not on disk: %v", err)
	}
}

// TestIngestRendersARefusalRecord: a refusal that produced a durable record
// renders it, so the `refusal_record` key the plugin page tells a host to report
// is reachable through the front door. Rendering only on success made that key —
// and the field behind it — dead surface.
func TestIngestRendersARefusalRecord(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Chdir(repo)

	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	// A regime the definition does not state: a list-level refusal, after the
	// run's identity is proven, so it records.
	outPath := detectionPayloadFile(t, runID, manifestHash, "generative", def)

	rendered, err := runCLIErr(t, "reading", "ingest", "--reading-json", outPath, "--json")
	if err == nil {
		t.Fatalf("a regime mismatch exited 0:\n%s", rendered)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("a refusal exited %d, want 2", code)
	}
	var res reading.IngestResult
	if jsonErr := json.Unmarshal(rendered, &res); jsonErr != nil {
		t.Fatalf("a refusal rendered no JSON result: %v\n%s", jsonErr, rendered)
	}
	if res.RefusalPath == "" {
		t.Fatal("the rendered result carries no refusal_record")
	}
	if _, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(res.RefusalPath))); statErr != nil {
		t.Errorf("the rendered refusal_record is not on disk: %v", statErr)
	}
	if len(res.Records) != 0 {
		t.Errorf("a refused run reported %d record(s)", len(res.Records))
	}
}

// TestTheTextRenderNamesNoItemZero: the elision entry a bounded refusal list
// carries is not an item, and the text surface must not print it as one — there
// is no item 0, and a reader sent looking for it finds nothing. The same render
// prints the refusal TOTAL, so the count is visible without reading the JSON.
func TestTheTextRenderNamesNoItemZero(t *testing.T) {
	var buf bytes.Buffer
	renderIngestResult(&buf, reading.IngestResult{
		RunID: "rdg-2608310000000041", Position: "detection", Regime: "registrative",
		RefusedCount: 199,
		RefusedItems: []reading.ItemRefusal{
			{Ordinal: 2, Rule: "named-provenance", Detail: "item 2 names no pattern"},
			{Rule: "refusals-elided", Detail: "and 178 more item(s) refused"},
		},
	})
	out := buf.String()
	if strings.Contains(out, "item 0") {
		t.Errorf("the render names an item 0:\n%s", out)
	}
	if !strings.Contains(out, "refused items: 199") {
		t.Errorf("the render does not print the refusal total:\n%s", out)
	}
	if !strings.Contains(out, "item 2 names no pattern") {
		t.Errorf("the render dropped a real item refusal:\n%s", out)
	}
	if !strings.Contains(out, "and 178 more") {
		t.Errorf("the render dropped the elision entry:\n%s", out)
	}
}

// TestDryRunRendersTheSizeReport is itd-198 ac-2 on the CLI side: the per-kind
// figures and the total appear on a run that writes nothing, which is the only
// run that can be made BEFORE deciding whether the artefact is worth producing.
func TestDryRunRendersTheSizeReport(t *testing.T) {
	res := reading.AssembleResult{
		RunID: "rdg-2608310000000001", Position: "widening", TargetCommit: "abcdef1234567890",
		ItemCount: 3, ManifestHash: "deadbeef", Written: false,
		Size: reading.SizeReport{
			ByKind: []reading.KindSize{
				{Kind: "source", Items: 2, Bytes: 3_400_000, TokensEst: 883_116},
				{Kind: "test", Items: 1, Bytes: 4_100_000, TokensEst: 1_064_935},
			},
			Items: 3, Bytes: 7_500_000, TokensEst: 1_948_051,
			Basis: "estimated: bytes / 3.85 (byte-derived, not a tokenizer's count)",
		},
	}
	var buf bytes.Buffer
	renderAssembleResult(&buf, res)
	out := buf.String()

	if !strings.Contains(out, "nothing (dry run") {
		t.Fatalf("the fixture is not rendering a dry run:\n%s", out)
	}
	for _, want := range []string{"source", "test", "7.5 MB", "1,948,051", "estimated", "bytes / 3.85"} {
		if !strings.Contains(out, want) {
			t.Errorf("the dry-run render omits %q:\n%s", want, out)
		}
	}
}

// TestSizeReportRendersBeforeTheWrittenLine guards the ordering trap: the
// dry-run branch returns early, so a report emitted after it would be invisible
// on exactly the runs that exist to carry it.
func TestSizeReportRendersBeforeTheWrittenLine(t *testing.T) {
	res := reading.AssembleResult{
		RunID: "rdg-2608310000000002", Position: "detection", TargetCommit: "abcdef1234567890",
		ItemCount: 1, ManifestHash: "cafe", Written: true, OutDir: "out",
		Size: reading.SizeReport{
			ByKind: []reading.KindSize{{Kind: "doc", Items: 1, Bytes: 1200, TokensEst: 311}},
			Items:  1, Bytes: 1200, TokensEst: 311, Basis: "estimated: bytes / 3.85",
		},
	}
	var buf bytes.Buffer
	renderAssembleResult(&buf, res)
	out := buf.String()

	size, written := strings.Index(out, "size (item text):"), strings.Index(out, "written:")
	if size < 0 || written < 0 {
		t.Fatalf("render is missing a line:\n%s", out)
	}
	if size > written {
		t.Errorf("the size report renders after the written line, so a dry run would "+
			"return before reaching it:\n%s", out)
	}
}

// TestHumanBytesAndThousands holds the two formatters the report leans on.
func TestHumanBytesAndThousands(t *testing.T) {
	for in, want := range map[int]string{
		0: "0 B", 999: "999 B", 1_000: "1.0 kB", 9_800_000: "9.8 MB", 2_300_000_000: "2.3 GB",
	} {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[int]string{
		0: "0", 42: "42", 999: "999", 1_000: "1,000", 2_295_107: "2,295,107", -1_234: "-1,234",
	} {
		if got := thousands(in); got != want {
			t.Errorf("thousands(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestPluginPageReportsTheSizeAndPreset is ac-2's plugin half and itd-199's,
// which were prose in commands/reading.md with nothing asserting it. The
// content-assertion pattern already existed twice in this file for other
// fields; it was simply not extended to the new ones.
func TestPluginPageReportsTheSizeAndPreset(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "reading.md"))
	if err != nil {
		t.Fatalf("read the plugin page: %v", err)
	}
	page := string(raw)
	for _, want := range []string{
		"size", "by_kind", "tokens_est", "basis",
		"preset", "preset_hash", "preset.selectors",
		// The comparative position's own fields. The page said "the comparative
		// position does not assemble" until adr-2609021016272867 gave it a
		// channel; what a host has to report now is which run the assembler
		// derived, how many candidates it held, whether the position was
		// exercised, and — on either derivation refusal — the listing of widening
		// runs the operator resolves it by.
		"candidate_run", "candidates", "not_exercised", "widening_runs",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("commands/reading.md does not mention %q, so the host is not told to "+
				"report it and the surface says less than the binary does", want)
		}
	}

	// Every shipped INVOCATION must actually run. Both examples on this page
	// were left stale when the scope operand landed, and only a retrospective
	// reading the branch tip noticed; the same guard now holds them to the two
	// operands the operand's withdrawal leaves.
	assertAssembleExamplesRun(t, "commands/reading.md", page)
}

// TestReadingSurfacesNameTwoOperands is itd-2609021003095168 ac-6 (framework v4
// section 8.2 and ruling M8; companion v4 section 4.1; adr-2609021016286571).
//
// Every surface that describes the invocation states two operands and none
// mentions a scope operand. It fails on a hit and on a miss alike: a page that
// still shows `--scope` ships an invocation that exits 2, and a page that shows
// an invocation missing `--position` or `--target` ships one that exits 2 for
// the other reason. This is the guard that used to require the third operand
// everywhere, inverted with the operand it required — the class of defect is
// one surface corrected and its siblings left behind, and the guard is over the
// whole set for that reason.
func TestReadingSurfacesNameTwoOperands(t *testing.T) {
	root := repoRootFromTest(t)
	for _, rel := range []string{
		"commands/reading.md",
		"docs/reference/cli/commands.md",
		".abcd/development/brief/04-surfaces/23-reading.md",
		"agents/cold-reading-widening.md",
		"agents/cold-reading-entailment.md",
		"agents/cold-reading-comparative.md",
		"agents/cold-reading-detection.md",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Errorf("read %s: %v", rel, err)
			continue
		}
		text := string(raw)
		if strings.Contains(text, "--scope") {
			t.Errorf("%s still names a --scope operand; the invocation is a position and a target "+
				"state, and the committed preset for the position supplies the rest", rel)
		}
		assertAssembleExamplesRun(t, rel, text)
		if !strings.HasPrefix(rel, "agents/") {
			continue
		}
		// The definitions' Object sections described what a run was given as
		// "the scope it was commissioned under", which is the operand's word
		// for it. There is no scope; there is the committed preset for the
		// position. The precedence sentence itself stays, and is held here so
		// the rewording cannot take it with it.
		for _, gone := range []string{
			"the scope it was commissioned under",
			"the scope you were given",
		} {
			if strings.Contains(flattenSpace(text), gone) {
				t.Errorf("%s still says %q; what a run is handed comes from the committed preset "+
					"for the position, not from an operand", rel, gone)
			}
		}
		if !strings.Contains(flattenSpace(text), "the bundle governs") {
			t.Errorf("%s lost the precedence sentence; where the definition and the bundle "+
				"disagree, the bundle governs", rel)
		}
	}

	// The verb's own Use and Example strings are the source the reference is
	// generated from, so they are checked at the source rather than only in the
	// artefact.
	cmd := newReadingCommand(new(bool))
	var sawAssemble bool
	for _, sub := range cmd.Commands() {
		if sub.Name() != "assemble" {
			continue
		}
		sawAssemble = true
		for _, field := range map[string]string{"Use": sub.Use, "Example": sub.Example, "Long": sub.Long} {
			if strings.Contains(field, "--scope") {
				t.Errorf("the assemble verb's own surface still names --scope:\n%s", field)
			}
		}
		for _, want := range []string{"--position", "--target"} {
			if !strings.Contains(sub.Use, want) {
				t.Errorf("the assemble verb's Use line omits %s: %q", want, sub.Use)
			}
		}
	}
	if !sawAssemble {
		t.Fatal("the reading command tree carries no assemble verb, so this guard checked nothing")
	}
}

// flattenSpace collapses every whitespace run to one space, so a prose
// assertion is about the sentence rather than about where the line broke.
func flattenSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// assertAssembleExamplesRun fails on any `reading assemble` invocation in text
// that omits either of the two operands the design admits.
func assertAssembleExamplesRun(t *testing.T, where, text string) {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "reading assemble") {
			continue
		}
		// Skip prose that merely names the verb; only look at invocations.
		if !strings.Contains(line, "--position") && !strings.Contains(line, "\\") {
			continue
		}
		// The examples span lines with a trailing backslash; take the whole
		// fenced command by scanning forward from the opening line.
		idx := strings.Index(text, line)
		invocation := text[idx:]
		if end := strings.Index(invocation, "```"); end > 0 {
			invocation = invocation[:end]
		}
		for _, want := range []string{"--position", "--target"} {
			if !strings.Contains(invocation, want) {
				t.Errorf("%s ships a `reading assemble` invocation with no %s, so it exits 2:\n%s",
					where, want, invocation)
			}
		}
	}
}

// TestIngestRendersWhatTheSweepDidOnAnErrorExit: the orphan sweep DELETES
// records from the committed ledger, and a delete in the committed tier is
// reported however the invocation ends. Rendering only when a refusal record
// exists made `rolled_back_records` and `cleared_stages` invisible on every
// other error path — an operator was told a write had failed and never told
// that records had left the ledger during it.
func TestIngestRendersWhatTheSweepDidOnAnErrorExit(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Chdir(repo)

	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)

	write := func(rel, body string) {
		abs := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// An orphan the sweep rolls back, and a stage path for THIS run that is a
	// regular file — so the staged write fails after the sweep has already
	// unlinked a committed record.
	orphan := "rdg-2608310000000051"
	write(reading.IngestStageDir+"/"+orphan+"/stage.json",
		`{"_type":"abcd.reading.ingest-stage/1","run_id":"`+orphan+`","records":[]}`)
	write(".abcd/work/issues/readings/"+orphan+"/rdi-2608310000000052.md",
		"---\nid: rdi-2608310000000052\n---\n")
	write(reading.IngestStageDir+"/"+runID, "not a directory\n")

	rendered, err := runCLIErr(t, "reading", "ingest", "--reading-json", outPath, "--json")
	if err == nil {
		t.Fatalf("the staged write did not fail, so this case proves nothing:\n%s", rendered)
	}
	var res reading.IngestResult
	if jsonErr := json.Unmarshal(rendered, &res); jsonErr != nil {
		t.Fatalf("an error exit rendered no JSON result, so the sweep's deletes were never reported: %v\n%s",
			jsonErr, rendered)
	}
	if len(res.RolledBack) != 1 || res.RolledBack[0] != "rdi-2608310000000052" {
		t.Errorf("the render reports rolled_back_records %v, want the record the sweep unlinked", res.RolledBack)
	}
	if len(res.ClearedStages) != 1 || res.ClearedStages[0] != orphan {
		t.Errorf("the render reports cleared_stages %v, want %s", res.ClearedStages, orphan)
	}
}

// mustMarshalIngestFields renders one fully-populated IngestResult, so the
// wiring test can check that every key the plugin page tells a host to report
// is a key the result actually emits.
func mustMarshalIngestFields(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(reading.IngestResult{
		RunID: "rdg-2608310000000061", Position: "detection", Regime: "registrative",
		RefusedItems:  []reading.ItemRefusal{{Ordinal: 1, Rule: "named-provenance", Detail: "d"}},
		RefusedCount:  1,
		RunRecordPath: "run.json", RefusalPath: "refusal.json",
		ClearedStages: []string{"rdg-2608310000000062"},
		RolledBack:    []string{"rdi-2608310000000063"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestTheTextRenderSaysWhenTheRegimeDidNotResolve: a run refused because its
// definition does not resolve has no regime, and the header interpolated the
// empty string into "under the  regime" — two spaces and a sentence asserting a
// regime that is not there. The refusal record leaves the field empty on
// purpose; the render has to say so rather than render the emptiness.
func TestTheTextRenderSaysWhenTheRegimeDidNotResolve(t *testing.T) {
	var buf bytes.Buffer
	renderIngestResult(&buf, reading.IngestResult{
		RunID: "rdg-2608310000000071", Position: "detection",
		RefusalPath: ".abcd/development/readings/rdg-2608310000000071/refusal.json",
	})
	out := buf.String()
	if strings.Contains(out, "the  regime") {
		t.Errorf("the render interpolates an empty regime:\n%s", out)
	}
	if !strings.Contains(out, "did not resolve") {
		t.Errorf("the render does not say the regime is unresolved:\n%s", out)
	}

	// And a resolved regime is still named.
	buf.Reset()
	renderIngestResult(&buf, reading.IngestResult{
		RunID: "rdg-2608310000000072", Position: "detection", Regime: "registrative",
	})
	if !strings.Contains(buf.String(), "under the registrative regime") {
		t.Errorf("the render dropped a resolved regime:\n%s", buf.String())
	}
}

// TestTheStatusRenderPrintsAnOrphanedIngest: the line is conditional, so
// without this the branch that reports an abnormal state would never run. An
// orphan means reading records are sitting in the committed ledger for a run
// that never happened, which is the one thing on this render an operator has to
// act on.
func TestTheStatusRenderPrintsAnOrphanedIngest(t *testing.T) {
	base := reading.Status{
		AssemblerVersion: reading.AssemblerVersion(), SchemaVersion: 1,
		Positions: reading.Positions(), Definitions: []string{"cold-reading-detection"},
	}
	var quiet bytes.Buffer
	renderReadingStatus(&quiet, base)
	if strings.Contains(quiet.String(), "interrupted") {
		t.Errorf("a repository with no orphan prints an interrupted line:\n%s", quiet.String())
	}

	var loud bytes.Buffer
	base.OrphanedIngests = []string{"rdg-2608310000000081"}
	renderReadingStatus(&loud, base)
	out := loud.String()
	if !strings.Contains(out, "rdg-2608310000000081") {
		t.Errorf("the render does not name the orphaned ingest:\n%s", out)
	}
	if !strings.Contains(out, "commit marker") {
		t.Errorf("the render does not say what an orphan means:\n%s", out)
	}
}

// TestTheStatusRenderTellsALeftoverStageFromAnOrphan: the interrupted line says
// a run's records "are in the ledger for a run with no commit marker" and will
// be swept. That is true of an orphan and false of a committed run whose stage
// merely failed to clear, so the two are rendered apart and the page that tells
// a host what to report names both keys.
func TestTheStatusRenderTellsALeftoverStageFromAnOrphan(t *testing.T) {
	base := reading.Status{
		AssemblerVersion: reading.AssemblerVersion(), SchemaVersion: 1,
		Positions: reading.Positions(), Definitions: []string{"cold-reading-detection"},
		LeftoverStages: []string{"rdg-2608310000000091"},
	}
	var buf bytes.Buffer
	renderReadingStatus(&buf, base)
	out := buf.String()
	if !strings.Contains(out, "rdg-2608310000000091") {
		t.Errorf("the render does not name the leftover stage:\n%s", out)
	}
	if !strings.Contains(out, "committed") || !strings.Contains(out, "clears the stage") {
		t.Errorf("the render does not say the run committed and only its stage goes:\n%s", out)
	}
	if strings.Contains(out, "interrupted") || strings.Contains(out, "commit marker") {
		t.Errorf("a committed run's leftover stage is rendered as an orphan:\n%s", out)
	}

	// Both at once render as two lines, each naming its own run.
	base.OrphanedIngests = []string{"rdg-2608310000000092"}
	buf.Reset()
	renderReadingStatus(&buf, base)
	out = buf.String()
	if !strings.Contains(out, "interrupted:    rdg-2608310000000092") {
		t.Errorf("the orphan line lost its run:\n%s", out)
	}
	if strings.Contains(out, "interrupted:    rdg-2608310000000091") {
		t.Errorf("the leftover stage is listed on the orphan line:\n%s", out)
	}

	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "reading.md"))
	if err != nil {
		t.Fatalf("read the plugin surface: %v", err)
	}
	if !strings.Contains(string(raw), "leftover_stages") {
		t.Error("commands/reading.md does not name leftover_stages, so a host is never told to report it")
	}
}

// TestRenderSizeReportNamesUnscannedItems is itd-194's operator-facing half:
// the assembler discloses per item that the exclusion floor did not examine it,
// and the report totals that per run so an operator deciding whether to
// dispatch a bundle can see how much of it travels on disclosure rather than on
// a scan (spc-2609021003136831, "The size report counts what was not
// examined").
//
// Absent at zero, and that is the deliberate half: a cold run has nothing to
// disclose, and a line reading "0 item(s)" would be a disclosure about nothing.
func TestRenderSizeReportNamesUnscannedItems(t *testing.T) {
	render := func(unscanned int) string {
		var buf bytes.Buffer
		renderSizeReport(&buf, reading.SizeReport{
			ByKind:    []reading.KindSize{{Kind: "doc", Items: 4, Bytes: 1200, TokensEst: 311}},
			Items:     4,
			Unscanned: unscanned,
			Bytes:     1200,
			TokensEst: 311,
			Basis:     "estimated: bytes / 3.85",
		})
		return buf.String()
	}

	above := render(3)
	for _, want := range []string{
		"unscanned: 3 item(s)",
		"travel whole, not examined by the exclusion floor",
	} {
		if !strings.Contains(above, want) {
			t.Errorf("the report above zero omits %q:\n%s", want, above)
		}
	}
	// After the per-kind rows: the count is a fact about the assembly as a
	// whole, and a line above them would read as another kind.
	if kind, unscanned := strings.Index(above, "doc"), strings.Index(above, "unscanned:"); kind > unscanned {
		t.Errorf("the unscanned line renders before the per-kind rows:\n%s", above)
	}

	if at := render(0); strings.Contains(at, "unscanned") {
		t.Errorf("a run with nothing unscanned still discloses one:\n%s", at)
	}
}

// TestRenderSizeReportSaysWhetherAWindowIsDeclared is spc-2609020626048722's
// ac-2 and ac-3 at the surface: the report carries the declaration beside the
// measurement, says plainly when a run exceeds it, and says "none declared"
// rather than rendering a zero a reader would take for a bound when the
// committed file is at schema version 1.
func TestRenderSizeReportSaysWhetherAWindowIsDeclared(t *testing.T) {
	render := func(s reading.SizeReport) string {
		var buf bytes.Buffer
		renderSizeReport(&buf, s)
		return buf.String()
	}
	base := reading.SizeReport{
		ByKind:    []reading.KindSize{{Kind: "doc", Items: 4, Bytes: 1200, TokensEst: 311}},
		Items:     4,
		Bytes:     1200,
		TokensEst: 311,
		Basis:     "estimated: bytes / 3.85",
	}

	none := render(base)
	if !strings.Contains(none, "window:        none declared (preset schema version 1)") {
		t.Errorf("a run under a version 1 file does not say so:\n%s", none)
	}

	within := base
	within.Window = &reading.Window{
		TokensEst: 630000, MeasuredTokensEst: 624000, MeasuredBytes: 2402400,
		MeasuredAt: "255543c1fa30",
	}
	got := render(within)
	for _, want := range []string{
		"window:        630,000 tokens declared",
		"(measured ~624,000 at 255543c1fa30)",
		"this run is within it",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the declared-window line omits %q:\n%s", want, got)
		}
	}

	over := within
	over.ExceedsWindow = true
	if got := render(over); !strings.Contains(got, "this run EXCEEDS it") {
		t.Errorf("a run past its declaration does not say so:\n%s", got)
	}

	// The over-target line is a separate statement from the declaration: an
	// entry can be inside its own declared window and still over the target the
	// maintainer's ruling of 2026-09-02 fixes at two hundred thousand.
	target := within
	target.OverTarget = true
	target.TokensEst = 624000
	got = render(target)
	for _, want := range []string{
		"over target: 624,000 estimated tokens against a target of 200,000",
		"the reader's window decides whether this is acceptable",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the over-target line omits %q:\n%s", want, got)
		}
	}
	if got := render(within); strings.Contains(got, "over target") {
		t.Errorf("a run under the target still states one:\n%s", got)
	}
}

// TestRenderSizeReportStatesTheMechanismLine is spc-2609020626048722's ac-9 at
// the surface (the readings companion's section 6.6): the entailment report
// states how many projected intents carry a mechanism claim beside the figures,
// and no other position's report carries the statement.
func TestRenderSizeReportStatesTheMechanismLine(t *testing.T) {
	base := reading.SizeReport{
		ByKind:    []reading.KindSize{{Kind: "intent-projection", Items: 4, Bytes: 1200, TokensEst: 311}},
		Items:     4,
		Bytes:     1200,
		TokensEst: 311,
		Basis:     "estimated: bytes / 3.85",
	}
	var buf bytes.Buffer
	renderSizeReport(&buf, base)
	if strings.Contains(buf.String(), "mechanism") {
		t.Errorf("a report carrying no mechanism proportion still states one:\n%s", buf.String())
	}

	buf.Reset()
	withIt := base
	withIt.Mechanism = &reading.MechanismReport{Intents: 23, Stated: 2, NoneStated: 6, Absent: 15}
	renderSizeReport(&buf, withIt)
	want := "mechanism: 2 of 23 projected intents carry a mechanism claim; " +
		"6 state none; 15 carry neither"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("the mechanism line is not %q:\n%s", want, buf.String())
	}
}
