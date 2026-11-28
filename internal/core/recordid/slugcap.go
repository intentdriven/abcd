package recordid

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/fsutil"
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
	// OverRecordCap is true when the record itself is larger than the record cap
	// (issueschema.RecordReadLimit), so its slug field was not read; Where is then
	// "record" and Line and Len are 0.
	OverRecordCap bool
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
// tree. A record larger than the record cap is not read, and is named with
// OverRecordCap set, never passed.
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
		field, line, ok, err := frontmatterSlug(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			out = append(out, SlugOverCap{Rel: rel, Where: "record", OverRecordCap: true})
			continue
		}
		if ok && len(field) > maxLen {
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
//
// It reads the block through frontmatter.Fields, the scanner record-lint reads
// the same field with, so the commit-message hook and the lint agree on what a
// record's slug is: a byte-order mark ahead of the opening rule, a key spelled
// `slug :`, a trailing comment and a long line above the field read here as they
// read there. A private line scan missed the first two, read a trailing comment
// into the value and counted it toward the slug's length, and gave up at a line
// over its buffer (iss-2611070737092992). The block may open below blank lines
// and HTML comments, located by mdrecord.FirstContent, as record-lint's
// frontmatterOpen admits an attribution comment above it.
//
// The file is taken through fsutil.ReadGuarded under the record cap, the read
// record-lint's record_schema takes of the same file: one open that refuses a
// symlinked or non-regular leaf, and no more than issueschema.RecordReadLimit
// bytes. A file that is absent or not a regular file carries no field, as
// before; a record over the cap returns fsutil.ErrTooBig, which the caller names.
func frontmatterSlug(file string) (string, int, bool, error) {
	data, err := fsutil.ReadGuarded(file, issueschema.RecordReadLimit)
	if errors.Is(err, fsutil.ErrTooBig) {
		return "", 0, false, err
	}
	if err != nil {
		return "", 0, false, nil
	}
	lines := strings.Split(string(data), "\n")
	open, col := mdrecord.FirstContent(lines)
	if open >= len(lines) || frontmatter.TrimBOM(lines[open][:col]) != "" || !frontmatter.IsDelimiter(lines[open][col:]) {
		return "", 0, false, nil
	}
	f, ok := frontmatter.Fields(lines[open:])["slug"]
	if !ok {
		return "", 0, false, nil
	}
	return strings.Trim(f.Value, `"'`), f.Line + open, true, nil
}
