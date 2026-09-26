package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/scribe"
)

// scribe_surface_test.go — the front door onto internal/core/scribe
// (spc-2609020626045177). `scribe` is a top-level verb, never a sub-verb of
// `reading`, because the two contexts must never share a front door.

// scribeOperands is the scribe verb's closed operand set, pinned on the
// readingOperands idiom: it fails CLOSED, so an operand added to either verb
// has to say what it does before it can ship. There is no operand that could
// widen the context, name a path the allow list does not, or set a record's
// content: the supplied dispositions are the researcher's, read whole.
var scribeOperands = map[string][]string{
	"abcd scribe":          {},
	"abcd scribe assemble": {"dispositions", "dry-run", "out", "run"},
	"abcd scribe ingest":   {"context", "dispositions", "scribe-json"},
}

// TestScribeOperandsArePinned walks the registered tree and holds each scribe
// command to its pinned operand set, then proves the pin by adding an operand
// to a tree built for the purpose.
func TestScribeOperandsArePinned(t *testing.T) {
	check := func(commands map[string][]string) []string {
		var out []string
		for path, want := range scribeOperands {
			got, ok := commands[path]
			if !ok {
				out = append(out, path+" is not registered")
				continue
			}
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				out = append(out, path+" declares "+strings.Join(got, ",")+", want "+strings.Join(want, ","))
			}
		}
		return out
	}
	walk := func() map[string][]string {
		out := map[string][]string{}
		for _, cmd := range commandSurface(NewRootCommand()) {
			if !strings.HasPrefix(cmd.Path, "abcd scribe") {
				continue
			}
			names := []string{}
			for _, f := range cmd.Flags {
				names = append(names, f.Name)
			}
			out[cmd.Path] = names
		}
		return out
	}
	for _, msg := range check(walk()) {
		t.Error(msg)
	}

	// Armed: a third operand on assemble is named by the same comparison.
	commands := walk()
	commands["abcd scribe assemble"] = append(commands["abcd scribe assemble"], "include")
	if msgs := check(commands); len(msgs) != 1 || !strings.Contains(msgs[0], "include") {
		t.Fatalf("an added operand was not caught: %v", msgs)
	}
}

// scribeRepo is a repository holding one ingested detection run of one item,
// with its commit marker and the scribe definition, entered as the working
// directory.
func scribeRepo(t *testing.T) (repo, item string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	res, err := capture.IngestReading(capture.IngestReadingRequest{
		RepoRoot: repo, Run: "rdg-2609250000000001", Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: "detection", Regime: issueschema.ReadingRegime("detection"),
		Items: []capture.ReadingItem{{Pattern: "a stated constraint", Body: map[string]string{
			"tension": "t", "constraint_in_play": "c", "why_a_tension": "w"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		".abcd/development/readings/rdg-2609250000000001/run.json": `{"run_id":"rdg-2609250000000001"}`,
		scribe.DefinitionPath: "---\nname: scribe\n---\n",
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: nowhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo, res.Records[0].ID
}

// TestScribeAssembleEchoesTheOperatorsOut: a relative --out means what the
// shell means by it, and the result shows the operator the string they typed
// rather than an absolute path nobody did.
func TestScribeAssembleEchoesTheOperatorsOut(t *testing.T) {
	repo, _ := scribeRepo(t)
	outside := t.TempDir()
	t.Chdir(outside)
	disp := filepath.Join(outside, "dispositions.md")
	if err := os.WriteFile(disp, []byte("nothing decided\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Run from a directory outside the repository, the repository named by
	// changing back into it with the --out still relative to where we typed it.
	t.Chdir(repo)
	rel, err := filepath.Rel(repo, filepath.Join(outside, "session"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scribe", "assemble", "--run", "rdg-2609250000000001",
		"--dispositions", disp, "--out", rel, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scribe assemble exited %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var res scribe.AssembleResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("--json is not a result: %v\n%s", err, stdout.String())
	}
	if res.OutDir != rel {
		t.Errorf("out_dir = %q, want the operator's own %q", res.OutDir, rel)
	}
	if strings.Contains(stdout.String(), outside) {
		t.Errorf("the result carries an absolute path nobody typed:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "session", scribe.ContextFileName)); err != nil {
		t.Fatalf("the context did not land where the operator's relative --out points: %v", err)
	}

	// Every refusal exits 2: a positional argument, a missing --run.
	for _, args := range [][]string{
		{"scribe", "assemble", "rdg-2609250000000001"},
		{"scribe", "assemble", "--dispositions", disp},
		{"scribe", "assemble", "--run", "rdg-2609250000000001"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := Run(args, &stdout, &stderr); code != 2 {
			t.Errorf("%v exited %d, want 2\n%s", args, code, stderr.String())
		}
	}
}

// TestScribeIngestRendersOnRefusal: a refusal after something landed renders
// what landed before it exits 2, so the operator can see the partial state.
func TestScribeIngestRendersOnRefusal(t *testing.T) {
	repo, item := scribeRepo(t)
	ground := "the constraint the reading names is real and binds the verb as shipped"
	disp := filepath.Join(t.TempDir(), "dispositions.md")
	if err := os.WriteFile(disp, []byte(item+": accepted — "+ground+".\nSurprise at "+item+": none here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scribe", "assemble", "--run", "rdg-2609250000000001", "--dispositions", disp, "--json"},
		&stdout, &stderr); code != 0 {
		t.Fatalf("assemble exited %d: %s", code, stderr.String())
	}
	var asm scribe.AssembleResult
	if err := json.Unmarshal(stdout.Bytes(), &asm); err != nil {
		t.Fatal(err)
	}
	// The disposition lands; the surprise's text is below the substance floor,
	// so the surprise verb refuses it after the disposition landed.
	payload := map[string]any{
		"_type": scribe.OutputType, "run": "rdg-2609250000000001", "context_sha256": asm.ContextSHA256,
		"dispositions": []map[string]any{{"item": item, "state": "accepted", "grounds": ground}},
		"surprises":    []map[string]any{{"occasioned_by": item, "text": "none here"}},
	}
	raw, _ := json.Marshal(payload)
	p := filepath.Join(t.TempDir(), "out.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code := Run([]string{"scribe", "ingest", "--scribe-json", p, "--dispositions", disp, "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("a refused ingest exited %d, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var res scribe.IngestResult
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	if err := dec.Decode(&res); err != nil {
		t.Fatalf("the refusal rendered no result first: %v\n%s", err, stdout.String())
	}
	if len(res.Dispositions) != 1 || res.Dispositions[0].Item != item {
		t.Fatalf("the refusal's render names %+v as landed, want the disposition", res.Dispositions)
	}
	_ = repo
}
