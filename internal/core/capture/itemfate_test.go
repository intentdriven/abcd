package capture

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// writeFile writes one file under a fresh directory tree, for the two probes
// below. Neither writes through a verb: what is under test is what the probes
// READ, and hand-written records are exactly what the ordering guard exists to
// catch (spc-2609020626039834, "The ordering guard").
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestItemFateReadsBothStores is the ordering guard's probe
// (spc-2609020626039834, "The ordering guard"; adr-2609021016272867): one
// function answers "has anything happened to this candidate", reading the
// disposition store and the admission store, the latter keyed on the (run,
// proposal) pair exactly as core/lint's admittedProposals keys it.
func TestItemFateReadsBothStores(t *testing.T) {
	const run = "rdg-2608300000000001"
	const other = "rdg-2608300000000002"

	t.Run("an untouched item is free", func(t *testing.T) {
		repo, _ := ledger(t)
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !fate.Free() {
			t.Fatalf("an item with nothing recorded against it is not free: %+v", fate)
		}
	})

	t.Run("a standing disposition is reported", func(t *testing.T) {
		repo, ir := ledger(t)
		writeFile(t, filepath.Join(ir, issueschema.DispositionsDir, "rdi-1", "dsp-1.md"),
			"---\nschema_version: 1\nid: \"dsp-1\"\nitem: \"rdi-1\"\nstate: \"accepted\"\n"+
				"disposition_grounds: \"because\"\n---\n")
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !slices.Equal(fate.Dispositions, []string{"dsp-1"}) {
			t.Fatalf("Dispositions = %v, want [dsp-1]", fate.Dispositions)
		}
		if fate.Free() {
			t.Error("an item carrying a standing disposition reads as free")
		}
	})

	t.Run("an admission under the item's own run is reported", func(t *testing.T) {
		repo, ir := ledger(t)
		writeFile(t, filepath.Join(ir, issueschema.AdmissionsDir, run, "adm-1.md"),
			"---\nschema_version: 1\nid: \"adm-1\"\nrun: \""+run+"\"\nproposal: \"rdi-1\"\n"+
				"grounds: \"the frame is engaged\"\n---\n")
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !slices.Equal(fate.Admissions, []string{"adm-1"}) {
			t.Fatalf("Admissions = %v, want [adm-1]", fate.Admissions)
		}
	})

	t.Run("an admission filed under another run does not count", func(t *testing.T) {
		// The pair is the key, never the proposal alone: an admission filed under
		// one run naming an id that belongs to another would otherwise be a global
		// silencer (iss-2608300935215868, which core/lint's admittedProposals
		// already closes on the same key).
		repo, ir := ledger(t)
		writeFile(t, filepath.Join(ir, issueschema.AdmissionsDir, other, "adm-1.md"),
			"---\nschema_version: 1\nid: \"adm-1\"\nrun: \""+other+"\"\nproposal: \"rdi-1\"\n"+
				"grounds: \"the frame is engaged\"\n---\n")
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !fate.Free() {
			t.Fatalf("an admission under %s counted against an item of %s: %+v", other, run, fate)
		}
	})

	t.Run("an admission whose own run disagrees with its bucket counts under neither", func(t *testing.T) {
		repo, ir := ledger(t)
		writeFile(t, filepath.Join(ir, issueschema.AdmissionsDir, run, "adm-1.md"),
			"---\nschema_version: 1\nid: \"adm-1\"\nrun: \""+other+"\"\nproposal: \"rdi-1\"\n"+
				"grounds: \"the frame is engaged\"\n---\n")
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !fate.Free() {
			t.Fatalf("a self-contradicting admission counted: %+v", fate)
		}
	})

	t.Run("a supersession cycle is unreadable rather than unanswered", func(t *testing.T) {
		repo, ir := ledger(t)
		dir := filepath.Join(ir, issueschema.DispositionsDir, "rdi-1")
		writeFile(t, filepath.Join(dir, "dsp-1.md"),
			"---\nschema_version: 1\nid: \"dsp-1\"\nitem: \"rdi-1\"\nstate: \"accepted\"\n"+
				"supersedes_disposition: \"dsp-2\"\n---\n")
		writeFile(t, filepath.Join(dir, "dsp-2.md"),
			"---\nschema_version: 1\nid: \"dsp-2\"\nitem: \"rdi-1\"\nstate: \"accepted\"\n"+
				"supersedes_disposition: \"dsp-1\"\n---\n")
		fate, err := ItemFate(repo, run, "rdi-1")
		if err != nil {
			t.Fatalf("ItemFate: %v", err)
		}
		if !fate.Cyclic {
			t.Fatalf("a supersession cycle is not reported: %+v", fate)
		}
		if fate.Free() {
			t.Error("a candidate whose fate cannot be read reads as free")
		}
	})
}

// TestComparativeRunForNamesTheLowestMatch is the probe the admission gate reads
// (spc-2609020626039834, "The fixed interpretation and the empty comparative
// run"): the outcome of a widening run is always a committed comparative run
// naming it, and this is how a later verb finds one.
func TestComparativeRunForNamesTheLowestMatch(t *testing.T) {
	repo, _ := ledger(t)
	const widening = "rdg-2608300000000001"
	runs := filepath.Join(repo, filepath.FromSlash(issueschema.ReadingsRecordDir))

	// Two comparative runs over the same widening run, plus a comparative run
	// over another and a widening run of its own. The lowest match is what comes
	// back, so the answer does not depend on directory order.
	writeFile(t, filepath.Join(runs, "rdg-2608300000000009", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2608300000000009","position":"comparative","candidate_run":"`+widening+`"}`)
	writeFile(t, filepath.Join(runs, "rdg-2608300000000005", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2608300000000005","position":"comparative","candidate_run":"`+widening+`"}`)
	writeFile(t, filepath.Join(runs, "rdg-2608300000000007", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2608300000000007","position":"comparative","candidate_run":"rdg-2608300000000002"}`)
	writeFile(t, filepath.Join(runs, widening, issueschema.RunRecordFileName),
		`{"run_id":"`+widening+`","position":"widening","candidate_run":""}`)

	got, err := ComparativeRunFor(repo, widening)
	if err != nil {
		t.Fatalf("ComparativeRunFor: %v", err)
	}
	if got != "rdg-2608300000000005" {
		t.Fatalf("ComparativeRunFor(%s) = %q, want rdg-2608300000000005", widening, got)
	}
}

// TestComparativeRunForIsEmptyBeforeAnyRun: a repository that has commissioned
// no comparative reading is in a state, not a fault, and the gate that reads
// this probe must be able to tell the two apart.
func TestComparativeRunForIsEmptyBeforeAnyRun(t *testing.T) {
	repo, _ := ledger(t)
	got, err := ComparativeRunFor(repo, "rdg-2608300000000001")
	if err != nil {
		t.Fatalf("ComparativeRunFor over an absent readings family: %v", err)
	}
	if got != "" {
		t.Fatalf("ComparativeRunFor = %q over a repository with no run, want \"\"", got)
	}

	// A widening run committed and nothing characterising it is still empty: the
	// probe answers about the comparative side alone.
	writeFile(t, filepath.Join(repo, filepath.FromSlash(issueschema.ReadingsRecordDir),
		"rdg-2608300000000001", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2608300000000001","position":"widening","candidate_run":""}`)
	got, err = ComparativeRunFor(repo, "rdg-2608300000000001")
	if err != nil {
		t.Fatalf("ComparativeRunFor: %v", err)
	}
	if got != "" {
		t.Fatalf("ComparativeRunFor = %q with only the widening run committed, want \"\"", got)
	}
}

// TestIngestReadingCommitsARunWithNoItems is the writer half of the clean-run
// idiom (framework section 13; the corrections ruling (4) of 2026-09-02;
// iss-2609021153269181): a run that returned nothing is recorded as a run with
// an empty item set, so the writer must accept one rather than refuse it.
func TestIngestReadingCommitsARunWithNoItems(t *testing.T) {
	repo, ir := ledger(t)
	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: strings.Repeat("a", 64),
		Position: "comparative", Regime: issueschema.ReadingRegime("comparative"),
		Items: nil,
	})
	if err != nil {
		t.Fatalf("IngestReading with no items: %v", err)
	}
	if len(res.Records) != 0 {
		t.Fatalf("an empty run wrote %d record(s), want 0", len(res.Records))
	}
	if res.Run != "rdg-2608300000000001" {
		t.Errorf("the result names run %q", res.Run)
	}
}
