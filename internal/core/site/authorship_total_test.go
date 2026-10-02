package site

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAuthorshipBarSumEqualsAssistedTotal pins the data invariant behind
// iss-2608231008315498: the Assisted-by panel's total (a.Assisted, the number it
// renders as the panel figure) MUST equal the sum of the per-model bars (ByModel),
// and the human-only `Assisted-by: None` declaration must never appear as a bar.
// The defect the issue reported was a note reading one total while the bars summed
// to another, because a `None` row was counted in one place and excluded in the
// other; the chart total and its bars have to agree. `None` is instead surfaced as
// its own DeclaredNone figure.
func TestAuthorshipBarSumEqualsAssistedTotal(t *testing.T) {
	r := gittest.NewRepo(t)
	// A controlled history: two single-model assisted commits, one commit naming two
	// models (two occurrences), two human-only `None` declarations, and one commit
	// with no trailer at all.
	//
	// Every declared value is trailer-shaped, because only a value that names a
	// model is charted at all — TestAuthorshipChartsOnlyTrailerShapedValues below
	// owns that rule, and free-form values here would test it twice while saying
	// nothing about the invariant this test exists for.
	r.Commit("feat: one\n\nAssisted-by: Vendor:model-a")
	r.Commit("feat: two\n\nAssisted-by: Vendor:model-b")
	r.Commit("feat: three\n\nAssisted-by: Vendor:model-a\nAssisted-by: Vendor:model-b")
	r.Commit("chore: human one\n\nAssisted-by: None")
	r.Commit("chore: human two\n\nAssisted-by: None")
	r.Commit("chore: undeclared")

	a, err := LoadAuthorship(r.Root())
	if err != nil {
		t.Fatalf("LoadAuthorship: %v", err)
	}

	barSum := 0
	for _, m := range a.ByModel {
		if m.Model == noneDeclaration {
			t.Fatalf("the None declaration must never be charted as a model bar: %+v", a.ByModel)
		}
		barSum += m.Commits
	}
	// The rendered panel total is a.Assisted; it must equal what the bars sum to.
	if barSum != a.Assisted {
		t.Fatalf("panel total (a.Assisted=%d) disagrees with the bar sum (%d); the note and the bars must match", a.Assisted, barSum)
	}
	// Sanity on the shape: 4 assisted occurrences (a, b, a, b), 2 None-only commits,
	// 1 undeclared.
	if a.Assisted != 4 {
		t.Errorf("assisted occurrences = %d, want 4", a.Assisted)
	}
	if a.DeclaredNone != 2 {
		t.Errorf("declared-none commits = %d, want 2", a.DeclaredNone)
	}
	if a.Undeclared != 1 {
		t.Errorf("undeclared commits = %d, want 1", a.Undeclared)
	}
}

// TestAuthorshipChartsOnlyTrailerShapedValues pins what may become a bar.
//
// The chart is a tally of DECLARED MODELS, and the convention says what a
// declaration of a model looks like: `Vendor:model-version`, optionally with the
// bracketed context-window suffix. A trailer line that says anything else is
// still a disclosure — the commit declares that a tool assisted, and it is
// counted as one — but it names no model, so there is no model to draw a bar
// for, and publishing the free text as a label puts whatever a commit author
// typed onto the site under a heading that reads as an inventory of models
// (iss-2609081940550352).
//
// The bar sum still has to equal the rendered panel total, so a value that is
// not charted is not counted in that total either; the commit-level figures —
// which are what the disclosure rate is computed from — keep counting it.
func TestAuthorshipChartsOnlyTrailerShapedValues(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("feat: one\n\nAssisted-by: Vendor:model-a")
	// The context-window variant is an exact model identifier, and the gate's
	// grammar admits it; the chart must too, unclipped.
	r.Commit("feat: two\n\nAssisted-by: Vendor:model-a[1m]")
	// A conformant line and a free-form second line on the same commit.
	r.Commit("feat: three\n\nAssisted-by: Vendor:model-a\nAssisted-by: and a second pair of hands")
	// A bare vendor with no version: the convention refuses it, so it names no
	// model — but the commit plainly declares that something assisted.
	r.Commit("feat: four\n\nAssisted-by: Vendor")
	r.Commit("chore: human\n\nAssisted-by: None")

	a, err := LoadAuthorship(r.Root())
	if err != nil {
		t.Fatalf("LoadAuthorship: %v", err)
	}

	want := map[string]int{"Vendor:model-a": 2, "Vendor:model-a[1m]": 1}
	got := map[string]int{}
	barSum := 0
	for _, m := range a.ByModel {
		got[m.Model] = m.Commits
		barSum += m.Commits
	}
	if len(got) != len(want) {
		t.Errorf("by_model: %+v, want the trailer-shaped values alone (%v)", a.ByModel, want)
	}
	for model, n := range want {
		if got[model] != n {
			t.Errorf("by_model[%q] = %d, want %d: %+v", model, got[model], n, a.ByModel)
		}
	}
	for _, m := range a.ByModel {
		if strings.Contains(m.Model, "second pair of hands") {
			t.Errorf("free text from a trailer line is published as a bar label: %q", m.Model)
		}
		if m.Model == "Vendor" {
			t.Errorf("a bare vendor with no version is charted as a model: %+v", a.ByModel)
		}
	}
	if barSum != a.Assisted {
		t.Errorf("panel total (a.Assisted=%d) disagrees with the bar sum (%d)", a.Assisted, barSum)
	}
	// The commit-level facts are untouched by what can be charted: four commits
	// declared assistance, one declared none, and NOTHING was silently read as a
	// human-only declaration because its value did not parse.
	if a.AssistedCommits != 4 {
		t.Errorf("assisted commits = %d, want 4", a.AssistedCommits)
	}
	if a.DeclaredNone != 1 {
		t.Errorf("declared-none commits = %d, want 1 — an unparsed value is not a declaration of no assistance", a.DeclaredNone)
	}
	if a.Undeclared != 0 {
		t.Errorf("undeclared commits = %d, want 0", a.Undeclared)
	}
}

// TestAuthorshipCountsTheAbcdLabelApart pins the third accepted trailer form
// (iss-2610020727129199). `Assisted-by: abcd:<version>` says abcd composed the
// commit's text from record facts and that no model wrote it (ruling PC1), so it
// is neither assistance nor a declaration of none: it is never a model bar, never
// adds abcd to the vendor set, never counts toward the disclosure rate's
// numerator, and is stated as its own commit-level figure beside DeclaredNone.
//
// A commit that names a model as well is assisted, whatever else it declares: a
// model's text is in it. A label naming abcd in a shape the gate refuses
// (`ABCD:latest`) is still abcd's claimed provenance, never a model.
func TestAuthorshipCountsTheAbcdLabelApart(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("feat: one\n\nAssisted-by: Vendor:model-a")
	r.Commit("chore(implement): pick\n\nAssisted-by: abcd:dev")
	r.Commit("chore(implement): pick again\n\nAssisted-by: abcd:v0.12.0")
	r.Commit("chore: malformed label\n\nAssisted-by: ABCD:latest")
	r.Commit("feat: both\n\nAssisted-by: abcd:dev\nAssisted-by: Vendor:model-b")
	r.Commit("chore: human\n\nAssisted-by: None")
	r.Commit("chore: undeclared")

	a, err := LoadAuthorship(r.Root())
	if err != nil {
		t.Fatalf("LoadAuthorship: %v", err)
	}
	for _, m := range a.ByModel {
		if strings.HasPrefix(strings.ToLower(m.Model), "abcd:") {
			t.Errorf("the abcd label is charted as a model bar: %+v", a.ByModel)
		}
	}
	if len(a.ByModel) != 2 || a.Assisted != 2 {
		t.Errorf("by_model %+v (assisted %d), want the two vendor models alone", a.ByModel, a.Assisted)
	}
	if a.AssistedCommits != 2 {
		t.Errorf("assisted commits = %d, want 2: an abcd-composed commit discloses no AI assistance", a.AssistedCommits)
	}
	if a.ComposedByAbcd != 3 {
		t.Errorf("abcd-composed commits = %d, want 3", a.ComposedByAbcd)
	}
	if a.DeclaredNone != 1 || a.Undeclared != 1 {
		t.Errorf("declared none %d, undeclared %d, want 1 and 1", a.DeclaredNone, a.Undeclared)
	}
	if sum := a.AssistedCommits + a.ComposedByAbcd + a.DeclaredNone + a.Undeclared; sum != a.Authored {
		t.Errorf("the commit-level figures sum to %d, but %d commits were authored", sum, a.Authored)
	}
}

// TestTheAbcdLabelNamesNoVendor: the bots row is derived from the vendors the
// trailers name together with a machine's address, and abcd is not a vendor. An
// author named for the label's vendor, at an address no person reads, is not
// demoted by the label: only a vendor a model trailer names takes part in the
// conjunction.
func TestTheAbcdLabelNamesNoVendor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("chore(implement): pick\n\nAssisted-by: abcd:dev")
	r.Git("-c", "user.name=abcd", "-c", "user.email=noreply@abcd.example.invalid",
		"commit", "--allow-empty", "-m", "chore: an author named abcd\n\nAssisted-by: None")

	a, err := LoadAuthorship(r.Root())
	if err != nil {
		t.Fatalf("LoadAuthorship: %v", err)
	}
	for _, b := range a.Bots {
		if b.Name == "abcd" {
			t.Fatalf("the abcd label registered abcd as a vendor and demoted its author: bots %+v", a.Bots)
		}
	}
}
