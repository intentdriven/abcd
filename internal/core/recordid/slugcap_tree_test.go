//go:build slugcap

// This test is held out of the default run by the slugcap build tag until the
// rename of iss-2610100626320367 lands: today's tree holds some two thousand
// record files minted under the former 60-character cap, and the test fails on
// every one of them. The rename commit (cmd/record-slug-rename) deletes the
// build-tag line above, and from then on the test runs in every `go test ./...`.
// Run it before then with `go test -tags slugcap ./internal/core/recordid/`.

package recordid

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommittedRecordSlugsWithinTheCap walks every store whose filenames carry a
// minted slug — issues, intents, specs and ADRs, every bucket — and holds each
// record's slug at MaxSlugLen or fewer: the slug segment of its filename, and
// the `slug:` frontmatter field where the record has one. It logs each family's
// longest repository-relative path, the figure the cap exists to bound
// (iss-2610100626320367).
func TestCommittedRecordSlugsWithinTheCap(t *testing.T) {
	root := repoRootForTest(t)
	for _, fam := range familyRoots {
		fileFamily := fam.prefix
		if fileFamily == "adr" {
			fileFamily = "" // the ADR store's filenames carry no family tag
		}
		var longest string
		count := 0
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(fam.dir)), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			_, slug, ok := SplitRecordFilename(fileFamily, d.Name())
			if !ok {
				return nil
			}
			count++
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if len(rel) > len(longest) {
				longest = rel
			}
			if len(slug) > MaxSlugLen {
				t.Errorf("%s: filename slug is %d characters, over the %d cap", rel, len(slug), MaxSlugLen)
			}
			if field, ok := frontmatterSlug(t, path); ok && len(field) > MaxSlugLen {
				t.Errorf("%s: slug field is %d characters, over the %d cap", rel, len(field), MaxSlugLen)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", fam.dir, err)
		}
		if count == 0 {
			t.Errorf("%s: no records found under %s; the walk is not reading the store", fam.prefix, fam.dir)
		}
		t.Logf("%s: %d records, longest path %d characters (%s)", fam.prefix, count, len(longest), longest)
	}
}

// frontmatterSlug returns the `slug:` value of a record's leading frontmatter
// block, unquoted, and whether the block carries one.
func frontmatterSlug(t *testing.T, path string) (string, bool) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return "", false
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			return "", false
		}
		if v, ok := strings.CutPrefix(line, "slug:"); ok {
			return strings.Trim(strings.TrimSpace(v), `"'`), true
		}
	}
	return "", false
}

// repoRootForTest finds the repository root above this package by its go.mod.
func repoRootForTest(t *testing.T) string {
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
			t.Fatal("no go.mod above the test's directory")
		}
		dir = parent
	}
}
