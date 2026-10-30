package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeCitationAgreesWithCitationCFF pins the copy-and-paste citation in
// README.md to CITATION.cff, which owns those facts.
//
// The README prints the citation so a reader can take it without opening another
// file, and a printed copy of a fact is a fact in two places: rename the software,
// correct an author's name or move the repository, and the CFF changes while the
// README keeps handing out the old citation. A wrong citation is the one kind of
// documentation error the reader carries into their own bibliography, where
// nothing in this repository can reach it.
//
// So every field the README repeats is derived here from the CFF rather than
// written down, and the page fails with the rename instead of after it.
func TestReadmeCitationAgreesWithCitationCFF(t *testing.T) {
	root := repoRootForTest(t)

	cff := readRepoFile(t, root, "CITATION.cff")
	readme := readRepoFile(t, root, "README.md")

	// CFF names a person by family and given names and an organisation by a
	// single name; BibTeX spells the first "family, given" and keeps the second
	// whole inside a second pair of braces, so it is never split into names.
	author := ""
	if family, ok := cffLookup(t, cff, "family-names"); ok {
		author = "{" + family + ", " + cffValue(t, cff, "given-names") + "}"
	} else {
		author = "{{" + cffValue(t, cff, "name") + "}}"
	}
	repo := cffValue(t, cff, "repository-code")
	license := cffValue(t, cff, "license")

	cases := []struct {
		name string
		want string
	}{
		// The BibTeX form is the only one the page prints, so it carries every
		// fact the CFF owns: the author in BibTeX's spelling, the repository URL,
		// and the declared licence.
		{"bibtex names the author", author},
		{"bibtex names the repository", "{" + repo + "}"},
		{"bibtex names the licence", "license = {" + license + "}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(readme, tc.want) {
				t.Errorf("README.md citation does not carry %q from CITATION.cff", tc.want)
			}
		})
	}
}

// readRepoFile reads one repo-relative path or fails the test.
func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// cffValue pulls one scalar off CITATION.cff. The file is flat YAML with quoted
// scalars, so a line scan reads it without taking a parser dependency for four
// fields; a key that is absent is a failure rather than an empty string, because
// an empty expectation would make every Contains check below pass vacuously.
func cffValue(t *testing.T, cff, key string) string {
	t.Helper()
	value, ok := cffLookup(t, cff, key)
	if !ok {
		t.Fatalf("CITATION.cff declares no %q", key)
	}
	return value
}

// cffLookup is cffValue for a key that may be absent: ok is false when no line
// declares it, and a declared key with no value still fails the test.
func cffLookup(t *testing.T, cff, key string) (string, bool) {
	t.Helper()
	for _, line := range strings.Split(cff, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "- ")
		rest, ok := strings.CutPrefix(trimmed, key+":")
		if !ok {
			continue
		}
		value := strings.Trim(strings.TrimSpace(rest), `"'`)
		if value == "" {
			t.Fatalf("CITATION.cff declares %q with no value", key)
		}
		return value, true
	}
	return "", false
}
