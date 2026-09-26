package lab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Check is one preflight check's verdict.
type Check struct {
	ID     string `json:"id"`
	Group  string `json:"group"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// The two check groups the preflight runs.
const (
	GroupIsolation  = "harness-isolation"
	GroupDualBinary = "dual-binary"
)

// Preflighted is the preflight's result: every check, the artefact it wrote,
// and — when a check failed — the finding the halt recorded.
type Preflighted struct {
	ID       string  `json:"id"`
	Passed   bool    `json:"passed"`
	Checks   []Check `json:"checks"`
	Artefact string  `json:"artefact"`
	Finding  string  `json:"finding,omitempty"`
	Lifted   string  `json:"lifted,omitempty"`
}

// vintageOf reads a binary's build vintage without running it. Tests replace it:
// a binary stamped at a fixture's pin would otherwise need a toolchain build.
var vintageOf = vintage.OfReader

// maxHomeEntries bounds the walk of the lab's HOME. A lab HOME holds caches and
// a store or two; past this the containment cannot be proved and the check fails.
const maxHomeEntries = 200000

// workBinPin is the work binary's identity as the first preflight to pass it
// recorded it. The work binary is built once and never rebuilt, so every later
// preflight compares against this.
type workBinPin struct {
	SHA256   string `json:"sha256"`
	Revision string `json:"revision"`
	At       string `json:"at"`
}

// Preflight runs the harness-isolation and dual-binary checks against the lab
// home and writes them as the lab's preflight artefact. A failed check halts the
// lab and records the refusal as a finding naming every failed check
// (ErrHalted); a passing preflight lifts a standing preflight halt. The finding
// stays in the log either way.
func Preflight(repoRoot, id string) (Preflighted, error) {
	l, err := open(repoRoot, id)
	if err != nil {
		return Preflighted{}, err
	}
	defer l.close()

	realDir, err := filepath.EvalSymlinks(l.dir)
	if err != nil {
		return Preflighted{}, fmt.Errorf("cannot resolve the lab home: %v", redact(err, l.store.home))
	}
	checks := []Check{
		l.checkHome(realDir),
		l.checkSnapshot(),
		l.checkRemotes(),
		l.checkHooks(realDir),
	}
	work, workHash := l.checkWorkBinary()
	checks = append(checks, work, l.checkPinned(work.OK, workHash), l.checkTestBinary())

	res := Preflighted{ID: id, Passed: true, Checks: checks, Artefact: l.display(preflightMD)}
	var failed []string
	for _, c := range checks {
		if !c.OK {
			res.Passed = false
			failed = append(failed, c.ID)
		}
	}
	if err := l.writeDoc(preflightMD, preflightDoc(l.entry, res)); err != nil {
		return Preflighted{}, err
	}
	js, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return Preflighted{}, err
	}
	if err := l.writeDoc(preflightJSON, string(js)+"\n"); err != nil {
		return Preflighted{}, err
	}
	if res.Passed {
		lifted, err := l.lift("preflight")
		if err != nil {
			return Preflighted{}, err
		}
		res.Lifted = lifted
		return res, nil
	}
	var detail strings.Builder
	detail.WriteString("The preflight refused:\n\n")
	for _, c := range checks {
		if !c.OK {
			fmt.Fprintf(&detail, "- %s (%s): %s\n", c.ID, c.Group, c.Detail)
		}
	}
	fid, err := l.haltAndRecord("preflight", failed,
		"Halted: the preflight refused "+strings.Join(failed, ", "),
		strings.TrimRight(detail.String(), "\n"), preflightMD)
	if err != nil {
		return Preflighted{}, err
	}
	res.Finding = fid
	return res, fmt.Errorf("%w: the preflight refused %s; recorded as %s in %s", ErrHalted, strings.Join(failed, ", "), fid, l.display(findingsName))
}

// checkHome proves the lab's own HOME is a real directory inside the lab and
// that nothing under it links outside the lab: a link out is the route by which
// operator-level state reaches the lab world.
func (l *lab) checkHome(realDir string) Check {
	c := Check{ID: "isolation.home", Group: GroupIsolation}
	homeDir := filepath.Join(l.dir, labHomeDir)
	if !fsutil.IsRealDir(homeDir) {
		c.Detail = "home/ is missing or not a real directory: the lab has no HOME of its own"
		return c
	}
	fold := fsutil.CaseFoldingFS()
	var escapes []string
	n := 0
	walkErr := filepath.WalkDir(homeDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		n++
		if n > maxHomeEntries {
			return errTooMany
		}
		if d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		target, rerr := filepath.EvalSymlinks(p)
		if rerr != nil {
			// A dangling link: judge where it points, lexically.
			t, lerr := os.Readlink(p)
			if lerr != nil {
				return lerr
			}
			if !filepath.IsAbs(t) {
				t = filepath.Join(filepath.Dir(p), t)
			}
			target = filepath.Clean(t)
		}
		if !fsutil.PathWithin(target, realDir, fold) && !fsutil.PathWithin(target, l.dir, fold) {
			rel, _ := filepath.Rel(l.dir, p)
			escapes = append(escapes, filepath.ToSlash(rel))
		}
		return nil
	})
	switch {
	case errors.Is(walkErr, errTooMany):
		c.Detail = fmt.Sprintf("home/ holds more than %d entries; its containment cannot be proved", maxHomeEntries)
	case walkErr != nil:
		c.Detail = "home/ could not be walked: " + redact(walkErr, l.store.home)
	case len(escapes) > 0:
		c.Detail = fmt.Sprintf("%d link(s) under home/ resolve outside the lab: %s", len(escapes), strings.Join(first(escapes, 5), ", "))
	default:
		c.OK = true
		c.Detail = "home/ is the lab's own HOME, and nothing under it links outside the lab"
	}
	return c
}

var errTooMany = errors.New("too many entries")

// checkSnapshot proves the snapshot is a standalone clone descending from the
// pin: its .git is a real directory (a linked worktree shares the operator's
// repository, hooks and config), it borrows no object store, and the pin is its
// HEAD or an ancestor of it.
func (l *lab) checkSnapshot() Check {
	c := Check{ID: "isolation.snapshot", Group: GroupIsolation}
	snap := filepath.Join(l.dir, snapshotDir)
	if !fsutil.IsRealDir(snap) {
		c.Detail = "snapshot/ is missing or not a real directory"
		return c
	}
	if !fsutil.IsRealDir(filepath.Join(snap, ".git")) {
		c.Detail = "snapshot/.git is not a real directory: a linked worktree or a gitfile shares a repository outside the lab"
		return c
	}
	if ok, _ := fsutil.ExistsNoFollow(filepath.Join(snap, ".git", "objects", "info", "alternates")); ok {
		c.Detail = "snapshot/ borrows another repository's object store (objects/info/alternates)"
		return c
	}
	head, err := gitutil.Run(snap, "rev-parse", "--verify", "HEAD")
	if err != nil {
		c.Detail = "git cannot read the snapshot's HEAD: " + redact(err, l.store.home)
		return c
	}
	ok, err := gitutil.IsAncestor(snap, l.entry.Pin, "HEAD")
	if err != nil || !ok {
		c.Detail = fmt.Sprintf("the snapshot's HEAD %s does not descend from the pin %s", short(head), short(l.entry.Pin))
		return c
	}
	c.OK = true
	if head == l.entry.Pin {
		c.Detail = "a standalone clone detached at the pin " + short(l.entry.Pin)
	} else {
		c.Detail = fmt.Sprintf("a standalone clone at %s, descending from the pin %s", short(head), short(l.entry.Pin))
	}
	return c
}

// checkRemotes proves the lab world has no remote: a remote is a path by which
// something done in the lab reaches a checkout or a forge outside it.
func (l *lab) checkRemotes() Check {
	c := Check{ID: "isolation.remotes", Group: GroupIsolation}
	snap := filepath.Join(l.dir, snapshotDir)
	if !fsutil.IsRealDir(snap) {
		c.Detail = "snapshot/ is missing"
		return c
	}
	out, err := gitutil.Run(snap, "remote")
	if err != nil {
		c.Detail = "git cannot list the snapshot's remotes: " + redact(err, l.store.home)
		return c
	}
	if names := strings.Fields(out); len(names) > 0 {
		c.Detail = "the snapshot has remote(s) " + strings.Join(first(names, 5), ", ") + ": the lab world must have no path back out (git remote remove)"
		return c
	}
	c.OK = true
	c.Detail = "the snapshot has no remote"
	return c
}

// checkHooks reads the hooks directory a session in the snapshot would run,
// with the operator's global and system configuration in force as they would be
// for that session, and refuses one outside the lab: an operator-level hooks
// path seeds operator state (a banlist, an identity gate) into the lab world.
// The value is judged as git itself expands it (--type=path: ~, ~user and
// %(prefix)/), never as raw text, so no spelling of an operator path reads as
// a relative one; a value git cannot expand is refused.
func (l *lab) checkHooks(realDir string) Check {
	c := Check{ID: "isolation.hooks", Group: GroupIsolation}
	snap := filepath.Join(l.dir, snapshotDir)
	if !fsutil.IsRealDir(snap) {
		c.Detail = "snapshot/ is missing"
		return c
	}
	cmd := exec.Command("git", "-C", snap, "config", "--type=path", "--get", "core.hooksPath")
	cmd.Env = gitutil.ScrubbedEnv()
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			c.OK = true
			c.Detail = "core.hooksPath is unset: the snapshot runs its own .git/hooks"
			return c
		}
		c.Detail = "git cannot read or expand the snapshot's hooks path: " + redact(err, l.store.home)
		return c
	}
	resolved := strings.TrimSuffix(string(out), "\n")
	if resolved == "" {
		// git joins the hook's name to the value, so an empty one runs hooks
		// from the filesystem root, never the snapshot.
		c.Detail = "core.hooksPath is set empty: git would run hooks from the filesystem root, outside the lab"
		return c
	}
	if !filepath.IsAbs(resolved) {
		// git runs hooks from the top of the work tree, so a relative path
		// names a directory in the snapshot.
		resolved = filepath.Join(snap, resolved)
	}
	resolved = fsutil.RealExistingPath(filepath.Clean(resolved))
	fold := fsutil.CaseFoldingFS()
	if fsutil.PathWithin(resolved, realDir, fold) || fsutil.PathWithin(resolved, l.dir, fold) {
		c.OK = true
		c.Detail = "core.hooksPath resolves inside the lab"
		return c
	}
	c.Detail = "core.hooksPath resolves outside the lab (" + clip(tilde(resolved, l.store.home)) + "): operator-level hooks would run in the lab world"
	return c
}

// checkWorkBinary proves bin/abcd is a regular file (never a link to an
// operator-level installation) built from the pristine snapshot: its embedded
// vintage is known, unmodified, and equals the pin. It returns the binary's
// sha256 for the pin check.
func (l *lab) checkWorkBinary() (Check, string) {
	c := Check{ID: "binary.work", Group: GroupDualBinary}
	fi, err := l.root.Lstat(workBinary)
	switch {
	case errors.Is(err, os.ErrNotExist):
		c.Detail = "bin/abcd is absent: build the work binary once from the pristine snapshot into bin/abcd"
		return c, ""
	case err != nil:
		c.Detail = "bin/abcd cannot be read: " + redact(err, l.store.home)
		return c, ""
	case fi.Mode()&os.ModeSymlink != 0:
		c.Detail = "bin/abcd is a link: the lab never drives its work with an operator-level installation"
		return c, ""
	case !fi.Mode().IsRegular():
		c.Detail = "bin/abcd is not a regular file"
		return c, ""
	}
	cur, hash, err := l.binaryIdentity(workBinary)
	switch {
	case err != nil:
		c.Detail = "bin/abcd carries no readable build metadata: " + redact(err, l.store.home)
	case cur.Revision == "":
		c.Detail = "bin/abcd has no vcs stamp: its vintage is unknown, so it cannot be tied to the pin"
	case !cur.Known:
		c.Detail = fmt.Sprintf("bin/abcd was built from a modified tree at %s: the work binary is built from the pristine snapshot", short(cur.Revision))
	case cur.Revision != l.entry.Pin:
		c.Detail = fmt.Sprintf("bin/abcd's vintage %s is not the pin %s", short(cur.Revision), short(l.entry.Pin))
	default:
		c.OK = true
		c.Detail = "bin/abcd's vintage equals the pin " + short(l.entry.Pin)
	}
	return c, hash
}

// checkPinned proves the work binary has not been rebuilt since the first
// preflight that passed it. The first such preflight records its identity.
func (l *lab) checkPinned(workOK bool, hash string) Check {
	c := Check{ID: "binary.pinned", Group: GroupDualBinary}
	data, err := fsutil.ReadGuardedInRoot(l.root, workBinPinName, 64<<10)
	if errors.Is(err, os.ErrNotExist) {
		if !workOK {
			c.Detail = "nothing to pin until bin/abcd passes"
			return c
		}
		pin, _ := json.Marshal(workBinPin{SHA256: hash, Revision: l.entry.Pin, At: stamp()})
		if err := fsutil.CreateExclusiveIn(l.root, workBinPinName, append(pin, '\n'), fileMode); err != nil {
			c.Detail = "cannot record the work binary's identity: " + redact(err, l.store.home)
			return c
		}
		c.OK = true
		c.Detail = "pinned now: sha256 " + short(hash) + "; the work binary is never rebuilt"
		return c
	}
	var pin workBinPin
	if err != nil || json.Unmarshal(data, &pin) != nil || pin.SHA256 == "" {
		c.Detail = "state/work-binary.json cannot be read"
		return c
	}
	if hash == "" {
		c.Detail = "bin/abcd is gone since it was pinned at sha256 " + short(pin.SHA256)
		return c
	}
	if hash != pin.SHA256 {
		c.Detail = fmt.Sprintf("bin/abcd was rebuilt mid-lab (sha256 %s, pinned %s): a provenance event; restore the pinned binary", short(hash), short(pin.SHA256))
		return c
	}
	c.OK = true
	c.Detail = "bin/abcd is the binary pinned at " + pin.At
	return c
}

// checkTestBinary proves the test binary, when one exists, is a regular file
// distinct from the work binary: mutations are tested with a second binary so
// the work binary's vintage survives them.
func (l *lab) checkTestBinary() Check {
	c := Check{ID: "binary.test", Group: GroupDualBinary}
	fi, err := l.root.Lstat(testBinary)
	switch {
	case errors.Is(err, os.ErrNotExist):
		c.OK = true
		c.Detail = "no test binary yet: rebuild bin/abcd-test for each mutation tested, never bin/abcd"
		return c
	case err != nil:
		c.Detail = "bin/abcd-test cannot be read: " + redact(err, l.store.home)
		return c
	case fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular():
		c.Detail = "bin/abcd-test is not a regular file"
		return c
	}
	if wi, err := l.root.Lstat(workBinary); err == nil && os.SameFile(fi, wi) {
		c.Detail = "bin/abcd-test is the work binary itself: the two binaries are distinct files"
		return c
	}
	c.OK = true
	c.Detail = "bin/abcd-test is a separate binary"
	if cur, _, err := l.binaryIdentity(testBinary); err == nil && cur.Revision != "" {
		c.Detail += " (vintage " + short(cur.Revision) + ")"
	}
	return c
}

// binaryIdentity reads a lab binary through the containment root: its vintage
// and its sha256.
func (l *lab) binaryIdentity(rel string) (vintage.Current, string, error) {
	f, err := l.root.Open(rel)
	if err != nil {
		return vintage.Current{}, "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return vintage.Current{}, "", err
	}
	cur, err := vintageOf(f)
	return cur, hex.EncodeToString(h.Sum(nil)), err
}

// preflightDoc renders the preflight artefact.
func preflightDoc(e Entry, res Preflighted) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Preflight — %s\n\n", e.ID)
	verdict := "PASSED"
	if !res.Passed {
		verdict = "HALTED"
	}
	fmt.Fprintf(&b, "Run %s against the pin %s: **%s**.\n\n", stamp(), e.Pin, verdict)
	b.WriteString("| Check | Group | Result | Detail |\n| --- | --- | --- | --- |\n")
	for _, c := range res.Checks {
		r := "pass"
		if !c.OK {
			r = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", c.ID, c.Group, r, strings.ReplaceAll(c.Detail, "|", "\\|"))
	}
	return b.String()
}

// short is a sha's first twelve digits.
func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func first(xs []string, n int) []string {
	if len(xs) <= n {
		return xs
	}
	return append(append([]string{}, xs[:n]...), fmt.Sprintf("and %d more", len(xs)-n))
}
