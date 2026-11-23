package matrix

// hidden.go keeps a code task's hidden tests outside every builder's reach
// (decision 9, spc-2610161949559050 step 4). They are written from the
// acceptance criteria before any run and kept by content in the run's record;
// here they are placed, read-only, in the run's clone store beside the copies
// and never in one, and a copy is refused before it starts when its clone
// already holds one, at any path, since then every builder was handed it.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// hiddenTest is one hidden test file: its path in the tree it is run against,
// slash-separated, and its content.
type hiddenTest struct {
	Path string
	Data []byte
}

// maxHiddenBytes bounds one hidden test file; maxRefs bounds how many a task
// keeps, as the run record bounds its kept artefacts.
const maxHiddenBytes = 4 << 20

// hiddenDir is where task's hidden tests are kept for run: under the clone
// store's hidden/ directory, which no run id names (a run id is mx-...), so it
// is never inside any copy's clone, cloneDir(store, run, alias). The folder is
// named by hiddenName, never by the task's id, so nothing a copy can list in
// the store spells a task.
func hiddenDir(store, run, task string) string {
	return filepath.Join(store, "hidden", run, hiddenName(run, task))
}

// hiddenName is the folder task's hidden tests are kept in for run: tests_ and
// the first 128 bits of a digest of the run's and the task's ids, the same on
// every reading, so the run finds the folder again without a record of its
// own, and spelling nothing of the task. A copy that already knew the task's
// id and this derivation could compute it; the name is withheld from a copy's
// view, not a secret.
func hiddenName(run, task string) string {
	sum := sha256.Sum256([]byte("abcd matrix hidden tests\x00" + run + "\x00" + task))
	return "tests_" + hex.EncodeToString(sum[:16])
}

// placeHiddenTests writes task's hidden tests for run at hiddenDir, read-only,
// and returns each as the run record keeps it (by its path, its sha256 and
// its size), in the order given. It refuses before writing anything: a run or
// task that is not an id, a store that is not absolute or that sits inside a
// working tree, no tests or more than maxRefs, a path that is not a clean
// relative path a record can name (one that climbs, is absolute, enters a git
// directory, or carries a space or a control), a path named twice or both as
// a file and as a directory, and an empty or oversized file. Tests are placed
// once: a task that already has them is refused, so no copy is scored against
// tests other than the ones its run recorded. A failure part way removes what
// it wrote.
func placeHiddenTests(store, run, task string, tests []hiddenTest) ([]ContentRef, error) {
	if !runIDRe.MatchString(run) {
		return nil, fmt.Errorf("%q is not a run id", clip(run))
	}
	if !validID(task) {
		return nil, fmt.Errorf("%q is not a task id", clip(task))
	}
	if !filepath.IsAbs(store) {
		return nil, fmt.Errorf("the clone store %q is not an absolute path", clip(store))
	}
	if len(tests) == 0 || len(tests) > maxRefs {
		return nil, fmt.Errorf("a code task keeps from 1 to %d hidden tests, not %d", maxRefs, len(tests))
	}
	seen := map[string]bool{}
	refs := make([]ContentRef, 0, len(tests))
	for _, h := range tests {
		if err := hiddenPath(h.Path); err != nil {
			return nil, err
		}
		if seen[h.Path] {
			return nil, fmt.Errorf("the hidden test %s is named twice", h.Path)
		}
		seen[h.Path] = true
		if len(h.Data) == 0 || len(h.Data) > maxHiddenBytes {
			return nil, fmt.Errorf("the hidden test %s holds %d bytes; a hidden test holds from 1 to %d", h.Path, len(h.Data), maxHiddenBytes)
		}
		refs = append(refs, contentOf(h.Path, h.Data))
	}
	for p := range seen {
		for d := path.Dir(p); d != "."; d = path.Dir(d) {
			if seen[d] {
				return nil, fmt.Errorf("the hidden test %s is also named as a directory", d)
			}
		}
	}
	dir := hiddenDir(store, run, task)
	if err := outsideWorkingTrees(filepath.Dir(dir)); err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			return nil, fmt.Errorf("the clone store could not be read: %w", storeRelative(err, store, "<clone store>"))
		}
		if errors.Is(err, errStoreUnanswered) {
			return nil, errors.New("hidden tests are never placed where whether the clone store is inside a working tree or a repository could not be told, " +
				"and git gave no answer there; it lives outside every checkout")
		}
		return nil, errors.New("hidden tests are never placed inside a working tree or a repository, and the clone store is inside one; " +
			"it lives outside every checkout")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return nil, fmt.Errorf("the hidden tests' directory could not be made: %w", storeRelative(err, store, "<clone store>"))
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("task %s of run %s already has its hidden tests; they are placed once", task, run)
		}
		return nil, fmt.Errorf("the hidden tests' directory could not be made: %w", storeRelative(err, store, "<clone store>"))
	}
	for _, h := range tests {
		if err := writeHidden(dir, h); err != nil {
			_ = removeHidden(dir)
			return nil, fmt.Errorf("the hidden test %s was not placed: %w", h.Path, storeRelative(err, store, "<clone store>"))
		}
	}
	return refs, nil
}

// hiddenPath holds a hidden test's path to a clean, relative, slash-separated
// path a run record can name, entering no git directory.
func hiddenPath(p string) error {
	if err := token("hidden test path", p, maxRefName); err != nil {
		return err
	}
	if path.Clean(p) != p || path.IsAbs(p) || strings.ContainsRune(p, '\\') {
		return fmt.Errorf("the hidden test path %q is not a clean relative path", clip(p))
	}
	for _, seg := range strings.Split(p, "/") {
		// A git directory in any letter case: on a case-insensitive
		// filesystem .GIT is .git.
		if seg == ".." || seg == "." || strings.EqualFold(seg, ".git") {
			return fmt.Errorf("the hidden test path %q climbs out of its tree or enters a git directory", clip(p))
		}
	}
	return nil
}

// writeHidden writes h under dir, read-only, refusing to replace anything.
func writeHidden(dir string, h hiddenTest) error {
	p := filepath.Join(dir, filepath.FromSlash(h.Path))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return err
	}
	if _, err := f.Write(h.Data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// removeHidden removes a part-placed hidden directory, whose files are
// read-only and whose directories it made itself.
func removeHidden(dir string) error {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(p, 0o700)
		}
		return nil
	})
	return os.RemoveAll(dir)
}

// hiddenOutsideClone refuses a copy whose hidden tests are not hidden from it:
// the hidden tests' directory inside its clone, the clone inside that
// directory, or a file of the clone (outside its git directory) holding a
// hidden test's content, at any path. It is asked of a fresh clone, before the
// copy starts, and names the clone's file and the hidden test by their paths
// in their trees, never where either sits on this machine.
func hiddenOutsideClone(clone, dir string, refs []ContentRef) error {
	rc, err := filepath.EvalSymlinks(clone)
	if err != nil {
		return fmt.Errorf("the copy's clone cannot be read: %w", storeRelative(err, clone, "<clone>"))
	}
	rd, err := resolved(dir)
	if err != nil {
		return fmt.Errorf("the hidden tests' directory cannot be resolved: %w", storeRelative(err, dir, "<hidden tests>"))
	}
	if within(rd, rc) || within(rc, rd) {
		return errors.New("the hidden tests sit inside the copy's clone, or the clone inside them; they are kept beside the copies, never in one")
	}
	bySize := map[int64][]ContentRef{}
	for _, r := range refs {
		bySize[r.Bytes] = append(bySize[r.Bytes], r)
	}
	var leaks []string
	err = filepath.WalkDir(rc, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" && filepath.Dir(p) == rc {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		cands := bySize[fi.Size()]
		if len(cands) == 0 {
			return nil
		}
		sum, err := fileSHA256(p, fi.Size())
		if err != nil {
			return err
		}
		for _, r := range cands {
			if r.SHA256 == sum {
				rel, _ := filepath.Rel(rc, p)
				leaks = append(leaks, filepath.ToSlash(rel)+" holds the hidden test "+r.Name)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("the copy's clone could not be read for its hidden tests: %w", storeRelative(err, rc, "<clone>"))
	}
	if len(leaks) > 0 {
		slices.Sort(leaks)
		return fmt.Errorf("the frozen commit hands a hidden test to every builder, so no copy starts: %s",
			clipTo(strings.Join(leaks, "; "), maxReasonBytes))
	}
	return nil
}

// storeRelative is err with the path a filesystem error names given relative
// to base, under label ("<clone store>/hidden/..."), and a path outside base
// named by label alone, so an error names no place on this machine. An error
// that is no filesystem error is named by its cause alone.
func storeRelative(err error, base, label string) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return &fs.PathError{Op: pe.Op, Path: labelled(pe.Path, base, label), Err: pe.Err}
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return &os.LinkError{Op: le.Op, Old: labelled(le.Old, base, label), New: labelled(le.New, base, label), Err: le.Err}
	}
	if errors.Is(err, errNoAncestor) {
		return errNoAncestor
	}
	return errors.New("a filesystem error naming no path abcd can show")
}

// labelled is p relative to base, under label, or label alone when p is not
// beneath base.
func labelled(p, base, label string) string {
	if real, err := filepath.EvalSymlinks(base); err == nil && within(p, real) {
		base = real
	}
	if !within(p, base) {
		return label
	}
	rel, err := filepath.Rel(base, p)
	if err != nil || rel == "." {
		return label
	}
	return label + "/" + filepath.ToSlash(rel)
}

// errNoAncestor is resolved finding no directory above a path.
var errNoAncestor = errors.New("no directory above the path exists")

// resolved is p with the links of its nearest existing ancestor resolved,
// and the rest of p, which does not exist yet, joined back on.
func resolved(p string) (string, error) {
	rest := ""
	at := filepath.Clean(p)
	for {
		if r, err := filepath.EvalSymlinks(at); err == nil {
			return filepath.Join(r, rest), nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		up := filepath.Dir(at)
		if up == at {
			return "", errNoAncestor
		}
		rest = filepath.Join(filepath.Base(at), rest)
		at = up
	}
}

// within reports whether p is base or a path beneath it.
func within(p, base string) bool {
	rel, err := filepath.Rel(base, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// fileSHA256 is the sha256 of the file at p, which holds size bytes; a file
// that does not hold exactly that many is an error.
func fileSHA256(p string, size int64) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, size+1))
	if err != nil {
		return "", err
	}
	if n != size {
		return "", fmt.Errorf("a file changed size while it was read")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
