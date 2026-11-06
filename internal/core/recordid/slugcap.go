package recordid

import (
	"bufio"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SlugOverCap is one place a record's slug runs past a repository's cap.
type SlugOverCap struct {
	// Rel is the record's repository-relative, slash-separated path.
	Rel string
	// Where is "filename slug" or "slug field".
	Where string
	// Line is the slug field's line, 0 for the filename.
	Line int
	// Len is the slug's length.
	Len int
}

// familyFileTag is the family tag a store's filenames carry for
// SplitRecordFilename: the ADR store's carry none.
func familyFileTag(prefix string) string {
	if prefix == "adr" {
		return ""
	}
	return prefix
}

// RecordFiles lists every file in every id-bearing store that a store could have
// written, each bucket included, as repository-relative slash paths. A store the
// repository does not have is no error, and a symlink is not a record.
func RecordFiles(root string) ([]string, error) {
	var out []string
	for _, fam := range familyRoots {
		tag := familyFileTag(fam.prefix)
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(fam.dir)), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if _, _, ok := SplitRecordFilename(tag, d.Name()); !ok {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SlugsOverCap judges the named paths against maxLen: the slug in the filename of
// each that sits in an id-bearing store, and the `slug:` field of its leading
// frontmatter where the working tree holds it as a regular file. A path outside
// every store, or one no store could have written, says nothing, and a path with
// no file is judged by its name alone: a staged rename need not be in the working
// tree.
func SlugsOverCap(root string, rels []string, maxLen int) []SlugOverCap {
	var out []SlugOverCap
	for _, rel := range rels {
		tag, ok := storeFileTag(rel)
		if !ok {
			continue
		}
		_, slug, ok := SplitRecordFilename(tag, path.Base(rel))
		if !ok {
			continue
		}
		if len(slug) > maxLen {
			out = append(out, SlugOverCap{Rel: rel, Where: "filename slug", Len: len(slug)})
		}
		if field, line, ok := frontmatterSlug(filepath.Join(root, filepath.FromSlash(rel))); ok && len(field) > maxLen {
			out = append(out, SlugOverCap{Rel: rel, Where: "slug field", Line: line, Len: len(field)})
		}
	}
	return out
}

// storeFileTag is the family tag of the store rel sits in, and whether it sits in
// one.
func storeFileTag(rel string) (string, bool) {
	for _, fam := range familyRoots {
		if strings.HasPrefix(rel, fam.dir+"/") {
			return familyFileTag(fam.prefix), true
		}
	}
	return "", false
}

// frontmatterSlug returns the `slug:` value of a regular file's leading
// frontmatter block, unquoted, its line, and whether the block carries one.
func frontmatterSlug(file string) (string, int, bool) {
	fi, err := os.Lstat(file)
	if err != nil || !fi.Mode().IsRegular() {
		return "", 0, false
	}
	f, err := os.Open(file)
	if err != nil {
		return "", 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return "", 0, false
	}
	for line := 2; sc.Scan(); line++ {
		text := sc.Text()
		if strings.TrimSpace(text) == "---" {
			return "", 0, false
		}
		if v, ok := strings.CutPrefix(text, "slug:"); ok {
			return strings.Trim(strings.TrimSpace(v), `"'`), line, true
		}
	}
	return "", 0, false
}
