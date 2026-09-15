package issueschema_test

import (
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestLedgerDirsNamesEveryConstant holds the derivation claim
// (spc-2609020626039834, "The manifest and the exclusions it asserts"): the
// comparative position's exclusion rows and the scribe's allow list are derived
// from this one function, so a ledger family this package declares and this
// function forgets would be admitted by one instrument and excluded by the
// other.
func TestLedgerDirsNamesEveryConstant(t *testing.T) {
	got := issueschema.LedgerDirs()
	want := []string{
		"open", "resolved", "wontfix",
		issueschema.ReadingsDir,
		issueschema.DispositionsDir,
		issueschema.AdmissionsDir,
		issueschema.SurprisesDir,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("LedgerDirs() = %v, want %v", got, want)
	}
	// The three status directories are the ones StatusDirs owns, and they are
	// carried rather than restated: a fourth status directory joins this list
	// the day it joins that one.
	for _, dir := range issueschema.StatusDirs {
		if !slices.Contains(got, dir) {
			t.Errorf("LedgerDirs() omits the status directory %q", dir)
		}
	}
	// Every sibling family constant this package declares is here. The list is
	// spelled out above; this is the other direction, so a constant added
	// without a line above fails rather than passing silently.
	for _, sibling := range []string{
		issueschema.ReadingsDir, issueschema.DispositionsDir,
		issueschema.AdmissionsDir, issueschema.SurprisesDir,
	} {
		if !slices.Contains(got, sibling) {
			t.Errorf("LedgerDirs() omits the sibling family directory %q", sibling)
		}
	}
}

// TestItemFateJudgement is the judgement half of the ordering guard
// (spc-2609020626039834, "The ordering guard"; adr-2609021016272867): given a
// disposition set and an admission set, what has happened to this candidate.
func TestItemFateJudgement(t *testing.T) {
	well := func(id, supersedes string) issueschema.DispositionRecord {
		return issueschema.DispositionRecord{ID: id, State: "accepted", Supersedes: supersedes, WellFormed: true}
	}

	t.Run("no records and no admissions is free", func(t *testing.T) {
		f := issueschema.JudgeItemFate(nil, nil)
		if !f.Free() {
			t.Fatalf("an item with nothing recorded against it is not free: %+v", f)
		}
		if f.Cyclic || f.Contested() {
			t.Errorf("an empty ledger reports cyclic=%v contested=%v", f.Cyclic, f.Contested())
		}
	})

	t.Run("one standing disposition is a fate", func(t *testing.T) {
		f := issueschema.JudgeItemFate([]issueschema.DispositionRecord{well("dsp-1", "")}, nil)
		if f.Free() {
			t.Fatal("an item carrying a standing disposition reads as free")
		}
		if !slices.Equal(f.Dispositions, []string{"dsp-1"}) {
			t.Errorf("Dispositions = %v, want [dsp-1]", f.Dispositions)
		}
	})

	t.Run("a superseded disposition does not stand", func(t *testing.T) {
		f := issueschema.JudgeItemFate([]issueschema.DispositionRecord{
			well("dsp-1", ""), well("dsp-2", "dsp-1"),
		}, nil)
		if !slices.Equal(f.Dispositions, []string{"dsp-2"}) {
			t.Errorf("Dispositions = %v, want [dsp-2]", f.Dispositions)
		}
	})

	t.Run("an admission alone is a fate", func(t *testing.T) {
		f := issueschema.JudgeItemFate(nil, []string{"adm-2", "adm-1"})
		if f.Free() {
			t.Fatal("an admitted item reads as free")
		}
		// Sorted, so a judgement over a directory listing does not depend on the
		// order the filesystem returned.
		if !slices.Equal(f.Admissions, []string{"adm-1", "adm-2"}) {
			t.Errorf("Admissions = %v, want [adm-1 adm-2]", f.Admissions)
		}
	})

	t.Run("a supersession cycle is unreadable, not unanswered", func(t *testing.T) {
		f := issueschema.JudgeItemFate([]issueschema.DispositionRecord{
			well("dsp-1", "dsp-2"), well("dsp-2", "dsp-1"),
		}, nil)
		if !f.Cyclic {
			t.Fatalf("every answer superseded and Cyclic is false: %+v", f)
		}
		if f.Free() {
			t.Error("a candidate whose fate cannot be read reads as free")
		}
	})

	t.Run("two standing answers are contested", func(t *testing.T) {
		f := issueschema.JudgeItemFate([]issueschema.DispositionRecord{
			well("dsp-1", ""), well("dsp-2", ""),
		}, nil)
		if !f.Contested() {
			t.Fatalf("two standing answers and Contested is false: %+v", f)
		}
		if f.Free() {
			t.Error("a contested item reads as free")
		}
	})
}
