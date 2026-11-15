package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestFingerprintFollowsWhatTheScannerWasBuiltFrom: two scanners built from
// the same inputs share a fingerprint, and a change to the pattern set, the
// identity or the augmenter changes it. The values are made up.
func TestFingerprintFollowsWhatTheScannerWasBuiltFrom(t *testing.T) {
	root := t.TempDir()
	build := func(opts ...Option) string {
		t.Helper()
		s, err := New(root, opts...)
		if err != nil {
			t.Fatal(err)
		}
		return s.Fingerprint()
	}
	base := build(WithAugmenter(nil))
	if base == "" || build(WithAugmenter(nil)) != base {
		t.Fatal("two scanners built from the same inputs have different fingerprints")
	}
	if build(WithAugmenter(&fakeAug{})) == base {
		t.Error("arming an augmenter left the fingerprint unchanged")
	}

	s, err := New(root, WithAugmenter(nil))
	if err != nil {
		t.Fatal(err)
	}
	s.identity.GitUserEmail = "someone-else@example.invalid"
	if s.Fingerprint() == base {
		t.Error("a changed identity left the fingerprint unchanged")
	}
	s, err = New(root, WithAugmenter(nil))
	if err != nil {
		t.Fatal(err)
	}
	s.patterns[0].Re = regexp.MustCompile(`made-up-[0-9]+`)
	if s.Fingerprint() == base {
		t.Error("a pattern's changed expression left the fingerprint unchanged")
	}

	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"house_token":{"regex":"HOUSE-[0-9]{6}","kind":"house_token","label":"house token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if build(WithAugmenter(nil)) == base {
		t.Error("a pattern added to the repository's config left the fingerprint unchanged")
	}
}
