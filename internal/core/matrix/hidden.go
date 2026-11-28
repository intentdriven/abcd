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

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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

// readHiddenTests reads back task's hidden tests for run from hiddenDir, each
// as the run record keeps it in refs, in that order, and refuses unless the
// directory holds exactly those files with exactly that content: a test
// missing, changed, grown or replaced by a link, or a file the run did not
// place, refuses the whole read, so a copy is only ever scored against the
// tests its run recorded. Every entry is judged by Lstat before it is read,
// through an os.Root, and a link is never followed. A refusal names a test
// by its path in its tree, never where the directory sits on this machine.
func readHiddenTests(store, run, task string, refs []ContentRef) ([]hiddenTest, error) {
	if !runIDRe.MatchString(run) {
		return nil, fmt.Errorf("%q is not a run id", clip(run))
	}
	if !validID(task) {
		return nil, fmt.Errorf("%q is not a task id", clip(task))
	}
	if !filepath.IsAbs(store) {
		return nil, fmt.Errorf("the clone store %q is not an absolute path", clip(store))
	}
	if len(refs) == 0 || len(refs) > maxRefs {
		return nil, fmt.Errorf("a code task keeps from 1 to %d hidden tests, not %d", maxRefs, len(refs))
	}
	want := map[string]ContentRef{}
	for _, r := range refs {
		if err := hiddenPath(r.Name); err != nil {
			return nil, err
		}
		if _, dup := want[r.Name]; dup {
			return nil, fmt.Errorf("the hidden test %s is named twice", r.Name)
		}
		if r.Bytes < 1 || r.Bytes > maxHiddenBytes {
			return nil, fmt.Errorf("the hidden test %s is recorded as %d bytes; a hidden test holds from 1 to %d", r.Name, r.Bytes, maxHiddenBytes)
		}
		want[r.Name] = r
	}
	dir := hiddenDir(store, run, task)
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("task %s of run %s has no hidden tests placed; they are placed at the run's start", task, run)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("task %s of run %s's hidden tests cannot be read", task, run)
	}
	defer root.Close()
	var extra []string
	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.New("the hidden tests' directory could not be listed")
		}
		if p == "." || d.IsDir() {
			return nil
		}
		if _, ok := want[p]; !ok || !d.Type().IsRegular() {
			extra = append(extra, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(extra) > 0 {
		slices.Sort(extra)
		return nil, fmt.Errorf("task %s of run %s's hidden tests hold what the run did not place: %s",
			task, run, clipTo(termsafe.SanitizeLine(strings.Join(extra, ", ")), maxReasonBytes))
	}
	out := make([]hiddenTest, 0, len(refs))
	for _, r := range refs {
		data, err := fsutil.ReadGuardedInRoot(root, filepath.FromSlash(r.Name), maxHiddenBytes)
		if err != nil {
			return nil, fmt.Errorf("the hidden test %s cannot be read as the regular file its run placed", r.Name)
		}
		if got := contentOf(r.Name, data); got != r {
			return nil, fmt.Errorf("the hidden test %s is not the content its run recorded, so no copy is scored against it", r.Name)
		}
		out = append(out, hiddenTest{Path: r.Name, Data: data})
	}
	return out, nil
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

// hiddenFolderChecked is called once readHiddenFolder has judged where the
// folder lies, before it opens it: a test replaces it to swap the folder in
// that window.
var hiddenFolderChecked = func(folder string) {}

// readHiddenFolder reads a code task's hidden tests from folder, the one its
// declaration names, each file at its path under the folder, in path order.
// The person consented to the folder by its path, so it is read where that
// path lies and nowhere else. It refuses, reading nothing: a folder that is
// not an absolute, clean path to a directory; one reached through a link at
// any part of its path, its last included; one inside the run's checkout,
// any worktree of it or the clone store at store, or holding any of them,
// compared in any letter case, since on a case-insensitive filesystem two
// spellings name one folder; and one inside any working tree or repository.
// The folder is then opened once as a root, held to be the directory that
// was judged (one swapped in its place after the judging is refused), and
// every entry is read through that root, judged by Lstat, never following a
// link: a link, a FIFO, device or socket, a git directory, more than maxRefs
// files, a file past maxHiddenBytes or an empty one refuses the whole read.
func readHiddenFolder(repoRoot, store, folder string) ([]hiddenTest, error) {
	shown := clip(termsafe.SanitizeLine(folder))
	if !filepath.IsAbs(folder) || filepath.Clean(folder) != folder {
		return nil, fmt.Errorf("the hidden tests' folder %q is not an absolute, clean path", shown)
	}
	for at := folder; ; at = filepath.Dir(at) {
		fi, err := os.Lstat(at)
		if err != nil {
			return nil, fmt.Errorf("the hidden tests' folder %q cannot be read", shown)
		}
		if fi.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("the hidden tests' folder %q is reached through a link; name it by the path the link leads to, so the folder read is the one consented to", shown)
		}
		if filepath.Dir(at) == at {
			break
		}
	}
	judged, err := os.Lstat(folder)
	if err != nil || !judged.IsDir() {
		return nil, fmt.Errorf("the hidden tests' folder %q is not a directory", shown)
	}
	if err := hiddenFolderApart(repoRoot, store, folder); err != nil {
		return nil, fmt.Errorf("the hidden tests' folder %q %w", shown, err)
	}
	hiddenFolderChecked(folder)
	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("the hidden tests' folder %q cannot be opened", shown)
	}
	defer root.Close()
	if at, err := root.Stat("."); err != nil || !os.SameFile(judged, at) {
		return nil, fmt.Errorf("the hidden tests' folder %q was replaced while it was read, so nothing in it is placed", shown)
	}
	var tests []hiddenTest
	listed := 0
	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("the hidden tests' folder %q could not be listed whole", shown)
		}
		if p == "." {
			return nil
		}
		if listed++; listed > maxExportEntries {
			return fmt.Errorf("the hidden tests' folder %q holds more than %d entries", shown, maxExportEntries)
		}
		rel := clip(termsafe.SanitizeLine(p))
		if strings.EqualFold(d.Name(), ".git") {
			return fmt.Errorf("the hidden tests' folder %q holds a git directory or file, %s", shown, rel)
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("the hidden tests' folder %q holds %s, which is not a regular file (a link is never followed)", shown, rel)
		}
		if err := hiddenPath(p); err != nil {
			return err
		}
		if len(tests) == maxRefs {
			return fmt.Errorf("the hidden tests' folder %q holds more than %d files", shown, maxRefs)
		}
		data, err := fsutil.ReadGuardedInRoot(root, filepath.FromSlash(p), maxHiddenBytes)
		if err != nil || len(data) == 0 {
			return fmt.Errorf("the hidden test %s cannot be read as a regular file of 1 to %d bytes", rel, maxHiddenBytes)
		}
		tests = append(tests, hiddenTest{Path: p, Data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(tests) == 0 {
		return nil, fmt.Errorf("the hidden tests' folder %q holds no file", shown)
	}
	return tests, nil
}

// hiddenFolderApart refuses a hidden tests' folder inside, or holding, the
// run's checkout, any worktree of it or the clone store, compared in any
// letter case, and one inside any working tree or repository. Its error reads
// after the folder's name.
func hiddenFolderApart(repoRoot, store, folder string) error {
	top, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return errors.New("cannot be told apart from the run's checkout, which cannot be read")
	}
	out, err := gitRaw("-C", top, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return errors.New("cannot be told apart from the checkout's worktrees, which git could not list")
	}
	places := []string{top}
	for _, field := range strings.Split(string(out), "\x00") {
		if wt, ok := strings.CutPrefix(field, "worktree "); ok && filepath.IsAbs(wt) {
			places = append(places, filepath.Clean(wt))
		}
	}
	if rs, err := resolved(store); err == nil {
		places = append(places, rs)
	} else {
		return errors.New("cannot be told apart from the clone store, which cannot be resolved")
	}
	for _, p := range places {
		if withinFold(folder, p) || withinFold(p, folder) {
			return errors.New("lies inside the run's checkout, one of its worktrees or the clone store, or holds one of them, " +
				"where a copy, or the run's own export of it, would find the tests; keep them in a folder of their own outside all three")
		}
	}
	if err := outsideWorkingTrees(folder); err != nil {
		return errors.New("lies inside a working tree or a repository, or whether it does could not be told; keep the hidden tests outside every checkout")
	}
	return nil
}

// withinFold is within with both paths read in one letter case.
func withinFold(p, base string) bool { return within(strings.ToLower(p), strings.ToLower(base)) }

// hiddenRecord is a code task's hidden tests as its run placed them, kept by
// content in the run's state beside its record (hiddenRecordRel): what a copy
// of the task is scored against, read back whole (readHiddenTests). Why it is
// a file of its own and not the run record's list is dcn-2611280309431219.
type hiddenRecord struct {
	SchemaVersion int          `json:"schema_version"`
	Run           string       `json:"run"`
	Task          string       `json:"task"`
	Tests         []ContentRef `json:"tests"`
}

// hiddenRecordSchema is a hidden record's schema version.
const hiddenRecordSchema = 1

// hiddenRecordRel is task's hidden record in run's state.
func hiddenRecordRel(run, task string) string { return run + "/hidden/" + task + ".json" }

// readHiddenRecord is task's hidden tests as run placed them, and whether it
// placed any: a run whose task named none, or one started before abcd placed
// hidden tests, has no record, and its copies' results are the host's report.
// A record that does not read as its own is an error.
func readHiddenRecord(proc stateReader, run, task string) ([]ContentRef, bool, error) {
	raw, err := proc.ReadState(hiddenRecordRel(run, task))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var h hiddenRecord
	if err := jsonstrict.Decode(raw, &h); err != nil || h.SchemaVersion != hiddenRecordSchema || h.Run != run || h.Task != task ||
		len(h.Tests) == 0 || len(h.Tests) > maxRefs {
		return nil, false, fmt.Errorf("task %s's hidden record does not read as its own", task)
	}
	for _, r := range h.Tests {
		if err := r.validate(); err != nil {
			return nil, false, fmt.Errorf("task %s's hidden record: %w", task, err)
		}
	}
	return h.Tests, true, nil
}
