package cli

// reading_comparative_test.go holds the comparative channel at the FRONT DOOR:
// what the operator is told when the derivation refuses, and what they are told
// when the position is not exercised (spc-2609020626039834; adr-2609021016272867).
//
// The invocation itself is pinned elsewhere and deliberately: readingOperands in
// regime_surface_test.go fails closed on any addition to this verb, and no
// operand names the run.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/reading"
)

// plantComparativeRuns writes n widening runs of two items each into the CLI's
// fixture repository and commits them, then writes each run's commit marker at
// the resulting HEAD.
//
// The marker is written after the commit and left untracked: the durable
// readings family is admitted by no include row and excluded at every position,
// so it sits outside the dirty gate and can name the very commit its records
// landed in.
func plantComparativeRuns(t *testing.T, repo string, runs ...string) {
	t.Helper()
	write := func(rel, body string) {
		abs := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The criteria discipline, without which a comparative assembly refuses for
	// a different reason than the one under test.
	write(".abcd/development/intents/disciplines/"+reading.CriteriaDiscipline+"-criteria.md",
		"---\nid: "+reading.CriteriaDiscipline+"\n---\n\n# The selection criteria\n\n"+
			"## The rule\n\n- Plausibility — it could work by a mechanism we can state.\n"+
			"- Cost — what building and carrying it consumes.\n\n## The gate\n\nDeclared here.\n")
	for _, run := range runs {
		for i := 1; i <= 2; i++ {
			id := run[len(run)-3:] + "0" + string(rune('0'+i))
			write(".abcd/work/issues/readings/"+run+"/rdi-"+id+".md",
				"---\nschema_version: 1\nid: \"rdi-"+id+"\"\nrun: \""+run+"\"\n"+
					"manifest: \""+strings.Repeat("a", 64)+"\"\nposition: \"widening\"\n"+
					"regime: \"generative\"\npattern: \"the pattern read under\"\n"+
					"configuration: \"a configuration the construal admits\"\n"+
					"what_admits_it: \"what admits it\"\n---\n")
		}
	}
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "plant widening runs")
	head := gitCmd(t, repo, "rev-parse", "HEAD")
	for _, run := range runs {
		write(".abcd/development/readings/"+run+"/run.json",
			`{"_type":"`+reading.RunType+`","run_id":"`+run+`","position":"widening",`+
				`"target_commit":"`+head+`"}`+"\n")
	}
}

// TestComparativeRefusalRendersTheWideningRuns is the operator's whole handle on
// the derivation: with none or more than one qualifying the verb refuses, and
// BOTH renderings carry the listing — the plain one in the message, the JSON one
// under `widening_runs` — because the remedy is to look at the runs and
// disposition one (adr-2609021016272867, "The operator is told what the
// assembler selected and why").
func TestComparativeRefusalRendersTheWideningRuns(t *testing.T) {
	t.Run("none qualifies", func(t *testing.T) {
		repo := readingRepo(t)
		t.Chdir(repo)
		out, err := runCLIErr(t, "reading", "assemble", "--position", "comparative",
			"--target", "HEAD", "--dry-run")
		if err == nil {
			t.Fatalf("a repository with no widening run assembled at the comparative position:\n%s", out)
		}
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("the refusal exited %d, want 2", code)
		}
		msg := err.Error()
		for _, want := range []string{"comparative", "no widening run is committed at this target"} {
			if !strings.Contains(strings.ToLower(msg), strings.ToLower(want)) {
				t.Errorf("the refusal does not carry %q: %s", want, msg)
			}
		}
	})

	t.Run("more than one qualifies", func(t *testing.T) {
		repo := readingRepo(t)
		t.Chdir(repo)
		const a, b = "rdg-2608301200000031", "rdg-2608301200000032"
		plantComparativeRuns(t, repo, a, b)

		// Plain: the listing is the message, one run per line.
		out, err := runCLIErr(t, "reading", "assemble", "--position", "comparative",
			"--target", "HEAD", "--dry-run")
		if err == nil {
			t.Fatalf("two qualifying widening runs assembled:\n%s", out)
		}
		msg := err.Error()
		for _, want := range []string{a, b, "2 item(s)", "no disposition and no admission"} {
			if !strings.Contains(msg, want) {
				t.Errorf("the plain refusal does not carry %q: %s", want, msg)
			}
		}

		// JSON: the same listing as data, so a host can act on it.
		raw, err := runCLIErr(t, "reading", "assemble", "--position", "comparative",
			"--target", "HEAD", "--dry-run", "--json")
		if err == nil {
			t.Fatal("the json rendering assembled")
		}
		var doc struct {
			WideningRuns []reading.WideningRun `json:"widening_runs"`
		}
		if uerr := json.Unmarshal([]byte(raw), &doc); uerr != nil {
			t.Fatalf("the json refusal does not decode: %v\n%s", uerr, raw)
		}
		if len(doc.WideningRuns) != 2 {
			t.Fatalf("the json refusal lists %d run(s), want 2: %s", len(doc.WideningRuns), raw)
		}
		seen := map[string]reading.WideningRun{}
		for _, r := range doc.WideningRuns {
			seen[r.ID] = r
		}
		for _, id := range []string{a, b} {
			r, ok := seen[id]
			if !ok {
				t.Errorf("the json listing omits %s", id)
				continue
			}
			if r.Items != 2 {
				t.Errorf("%s is listed with %d item(s), want 2", id, r.Items)
			}
			if r.Dispositioned || r.Admitted || !r.Committed {
				t.Errorf("%s is listed with the wrong state: %+v", id, r)
			}
		}
	})
}

// TestComparativeAssemblyNamesTheDerivedRun: with exactly one qualifying run the
// verb assembles, and the operator is told which run the reading is about —
// no operand named it, so the result is where they learn it.
func TestComparativeAssemblyNamesTheDerivedRun(t *testing.T) {
	repo := readingRepo(t)
	t.Chdir(repo)
	const run = "rdg-2608301200000033"
	plantComparativeRuns(t, repo, run)

	raw := runCLI(t, "reading", "assemble", "--position", "comparative",
		"--target", "HEAD", "--dry-run", "--json")
	var res reading.AssembleResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("the --json envelope does not decode: %v\n%s", err, raw)
	}
	if res.CandidateRun != run {
		t.Errorf("the result names the candidate run %q, want %q", res.CandidateRun, run)
	}
	if res.Candidates != 2 {
		t.Errorf("the result reports %d candidates, want 2", res.Candidates)
	}
	if res.NotExercised {
		t.Error("a two-candidate run was reported as not exercised")
	}

	// The plain rendering says the same thing.
	plain := runCLI(t, "reading", "assemble", "--position", "comparative",
		"--target", "HEAD", "--dry-run")
	for _, want := range []string{run, "candidate run", "derived from the record"} {
		if !strings.Contains(string(plain), want) {
			t.Errorf("the plain rendering does not carry %q:\n%s", want, plain)
		}
	}
}
