package readingitem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repo lays out a repository with an issues root under it and returns both.
func repo(t *testing.T) (root, issuesRoot string) {
	t.Helper()
	root = t.TempDir()
	issuesRoot = filepath.Join(root, ".abcd", "work", "issues")
	if err := os.MkdirAll(issuesRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, issuesRoot
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocateFindsOneItemAcrossRuns(t *testing.T) {
	_, ir := repo(t)
	write(t, filepath.Join(ir, "readings", "rdg-1", "rdi-11.md"), "a")
	write(t, filepath.Join(ir, "readings", "rdg-2", "rdi-22.md"), "b")
	run, path, err := Locate(ir, "rdi-22")
	if err != nil || run != "rdg-2" || filepath.Base(path) != "rdi-22.md" {
		t.Fatalf("Locate = %q %q %v", run, path, err)
	}
	if _, _, err := Locate(ir, "rdi-33"); !errors.Is(err, ErrUnknown) {
		t.Errorf("an absent item: err = %v, want ErrUnknown", err)
	}
	write(t, filepath.Join(ir, "readings", "rdg-3", "rdi-22.md"), "c")
	if _, _, err := Locate(ir, "rdi-22"); !errors.Is(err, ErrDuplicate) {
		t.Errorf("an item in two runs: err = %v, want ErrDuplicate", err)
	}
	if _, _, err := Locate(ir, "rdi-../x"); err == nil || !strings.Contains(err.Error(), "invalid rdi-N") {
		t.Errorf("a malformed id: err = %v", err)
	}
	// No readings tree at all is a state, not a fault.
	_, empty := repo(t)
	if _, _, err := Locate(empty, "rdi-1"); !errors.Is(err, ErrUnknown) {
		t.Errorf("no readings tree: err = %v, want ErrUnknown", err)
	}
}

func TestLocateRefusesSymlinkedRun(t *testing.T) {
	_, ir := repo(t)
	outside := t.TempDir()
	write(t, filepath.Join(outside, "rdi-11.md"), "a")
	if err := os.MkdirAll(filepath.Join(ir, "readings"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ir, "readings", "rdg-1")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := Locate(ir, "rdi-11"); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("a symlinked run directory: err = %v, want ErrPathUnsafe", err)
	}
}

func TestLocateDispositionRefusesASymlinkedItemDir(t *testing.T) {
	_, ir := repo(t)
	write(t, filepath.Join(ir, "dispositions", "rdi-11", "dsp-5.md"), "a")
	item, path, err := LocateDisposition(ir, "dsp-5")
	if err != nil || item != "rdi-11" || filepath.Base(path) != "dsp-5.md" {
		t.Fatalf("LocateDisposition = %q %q %v", item, path, err)
	}
	outside := t.TempDir()
	write(t, filepath.Join(outside, "dsp-6.md"), "b")
	if err := os.Symlink(outside, filepath.Join(ir, "dispositions", "rdi-12")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := LocateDisposition(ir, "dsp-6"); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("a symlinked item directory: err = %v, want ErrPathUnsafe", err)
	}
}

func TestResolveOccasionAcceptsShippedIntent(t *testing.T) {
	root, ir := repo(t)
	write(t, filepath.Join(root, ".abcd/development/intents/shipped/itd-7-a-delivery.md"), "---\nid: itd-7\n---\n")
	write(t, filepath.Join(ir, "readings", "rdg-1", "rdi-11.md"), "a")
	path, err := ResolveOccasion(root, "itd-7", FamilyItem, FamilyIntent)
	if err != nil || filepath.Base(path) != "itd-7-a-delivery.md" {
		t.Fatalf("ResolveOccasion(itd-7) = %q %v", path, err)
	}
	path, err = ResolveOccasion(root, "rdi-11", FamilyItem, FamilyIntent)
	if err != nil || filepath.Base(path) != "rdi-11.md" {
		t.Fatalf("ResolveOccasion(rdi-11) = %q %v", path, err)
	}
	if _, err := ResolveOccasion(root, "itd-8", FamilyItem, FamilyIntent); !errors.Is(err, ErrUnknown) {
		t.Errorf("an absent intent: err = %v, want ErrUnknown", err)
	}
}

func TestResolveOccasionRefusesPlannedIntent(t *testing.T) {
	root, _ := repo(t)
	write(t, filepath.Join(root, ".abcd/development/intents/planned/itd-7-a-plan.md"), "---\nid: itd-7\n---\n")
	_, err := ResolveOccasion(root, "itd-7", FamilyItem, FamilyIntent)
	if err == nil || !strings.Contains(err.Error(), "planned/") {
		t.Fatalf("a planned intent: err = %v, want a refusal naming planned/", err)
	}
}

func TestResolveOccasionRefusesAFamilyItIsNotHanded(t *testing.T) {
	root, ir := repo(t)
	write(t, filepath.Join(ir, "dispositions", "rdi-11", "dsp-5.md"), "a")
	for _, id := range []string{"dsp-5", "iss-1", "", "rdi", "../x"} {
		if _, err := ResolveOccasion(root, id, FamilyItem, FamilyIntent); err == nil || !strings.Contains(err.Error(), "is not one of rdi-N, itd-N") {
			t.Errorf("ResolveOccasion(%q): err = %v", id, err)
		}
	}
	if _, err := ResolveOccasion(root, "dsp-5", FamilyDisposition); err != nil {
		t.Errorf("a disposition handed its family: %v", err)
	}
}

// TestResolveOccasionReadsOnlyTheIntentStore: an intent occasion is looked up
// in the intent store alone, from the repository root the caller hands it, so a
// fault in an unrelated record family does not refuse it.
func TestResolveOccasionReadsOnlyTheIntentStore(t *testing.T) {
	root, _ := repo(t)
	write(t, filepath.Join(root, ".abcd/development/intents/shipped/itd-7-a-delivery.md"), "---\nid: itd-7\n---\n")
	write(t, filepath.Join(root, ".abcd/development/specs/open/spc-3-a-spec.md"), "a")
	specs := filepath.Join(root, ".abcd", "development", "specs")
	if err := os.Chmod(specs, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(specs, 0o755) })
	if _, err := os.ReadDir(specs); err == nil {
		t.Skip("the store stays readable at mode 000 (running as root)")
	}
	if path, err := ResolveOccasion(root, "itd-7", FamilyIntent); err != nil || filepath.Base(path) != "itd-7-a-delivery.md" {
		t.Fatalf("ResolveOccasion(itd-7) = %q %v", path, err)
	}
	if _, err := ResolveOccasion(t.TempDir(), "itd-7", FamilyIntent); !errors.Is(err, ErrUnknown) {
		t.Errorf("a root holding no intent store: err = %v, want ErrUnknown", err)
	}
}
