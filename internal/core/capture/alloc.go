package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const lockFilename = ".iss-alloc.lock"

// placeholderRetryBudget bounds the mint's redraw loop (spc-33 ruling 2).
const placeholderRetryBudget = 8

// orphanAgeThreshold is how old a zero-byte placeholder must be before the
// sweep removes it.
const orphanAgeThreshold = 60 * time.Second

// lockTimeout is the default flock acquisition budget. A var (not const) so a
// test can shorten it to exercise contention without a multi-second wait.
var lockTimeout = 5 * time.Second

// beforeOrphanRemoveHook, when non-nil, fires immediately before the sweep
// unlinks a classified orphan placeholder. It is a test-only seam (nil in
// production, zero overhead) used to force the iss-102 commit-in-the-unlink-window
// interleaving deterministically.
var beforeOrphanRemoveHook func(cand string)

// ensureLedgerDirs provisions issuesRoot and its status sub-directories, refusing
// symlinked leaves AND symlinked ancestors. The list is issueschema.StatusDirs —
// the same value the readers scan and the deterministic gates scope to, so a
// folder can never be provisioned that one of them does not look in. Idempotent.
//
// It is ledgerDirs with create, and nothing besides: one list of directories,
// judged by the same rule whether a verb is about to write the ledger or only
// read it.
func ensureLedgerDirs(repoRoot, issuesRoot string) error {
	if !filepath.IsAbs(issuesRoot) {
		return fmt.Errorf("issuesRoot must be absolute, got %q", issuesRoot)
	}
	return ledgerDirs(repoRoot, issuesRoot, true)
}

// ledgerDirs judges every directory the ledger is reached through and every
// directory the ledger IS, refusing any that exists as something other than a
// real directory. Exactly three groups, in this order:
//
//  1. each path segment between repoRoot and the ledger's parent — for the
//     default ledger, `.abcd` then `.abcd/work`;
//  2. issuesRoot itself;
//  3. every issueschema.StatusDirs entry under it — open, resolved, wontfix.
//
// All three, because a symlink at ANY of them redirects the store: a committed
// `.abcd` or `.abcd/work` sent the whole store — the lock, the status
// directories and the record — outside the checkout while the result still
// reported a repo-relative path (GHSA-865x-5m7q-qm79), and a committed `issues`
// or `open` did the same to a read, because os.ReadDir follows a symlinked
// directory and the guarded record reader's O_NOFOLLOW covers only the final
// component. Guarding the first group alone left `capture list --json` and the
// bare status render serializing a sibling checkout's records from a fresh
// clone, under repo-relative paths.
//
// With create, each is provisioned one level at a time with os.Mkdir
// (safeMkdirLeaf) — never MkdirAll, which follows whatever is already there.
// Without create — the form every verb runs through resolveRoots, so the READ
// paths are covered — an absent directory is not a fault: an unprovisioned
// ledger is a state the readers already tolerate, and nothing can hide behind a
// directory that is not there.
//
// The walk is modelled on memory.memoryDir (the GHSA-72rp fix) and carries the
// same residue: the window between one segment's Lstat and its Mkdir is a
// local-racer TOCTOU, closed only by opening the store as an os.Root, which is
// the package-wide follow-up iss-2609012037143368 records.
//
// An issuesRoot outside repoRoot is an operator-typed operand with no boundary
// to walk from: group 1 is skipped (with create, its parent is provisioned with
// MkdirAll as before), and groups 2 and 3 are judged as they are inside.
func ledgerDirs(repoRoot, issuesRoot string, create bool) error {
	var dirs []string
	parent := filepath.Dir(issuesRoot)
	if fsutil.PathWithin(parent, repoRoot, false) {
		rel, err := filepath.Rel(repoRoot, parent)
		if err != nil {
			return err
		}
		if rel != "." {
			current := repoRoot
			for _, segment := range strings.Split(rel, string(filepath.Separator)) {
				current = filepath.Join(current, segment)
				dirs = append(dirs, current)
			}
		}
	} else if create {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	dirs = append(dirs, issuesRoot)
	for _, sub := range issueschema.StatusDirs {
		dirs = append(dirs, filepath.Join(issuesRoot, sub))
	}
	for _, dir := range dirs {
		if create {
			if err := safeMkdirLeaf(dir); err != nil {
				return err
			}
			continue
		}
		if err := refuseSymlinkedDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// safeMkdirLeaf creates target if absent, then insists (via Lstat) that the
// result is a real directory and not a symlink.
func safeMkdirLeaf(target string) error {
	fi, err := os.Lstat(target)
	if os.IsNotExist(err) {
		if mkErr := os.Mkdir(target, 0o755); mkErr != nil && !os.IsExist(mkErr) {
			return fmt.Errorf("%w: mkdir failed for %s: %v", ErrPathUnsafe, target, mkErr)
		}
		fi, err = os.Lstat(target)
		if err != nil {
			return fmt.Errorf("%w: leaf disappeared after mkdir: %s", ErrPathUnsafe, target)
		}
	} else if err != nil {
		return fmt.Errorf("%w: lstat failed for %s: %v", ErrPathUnsafe, target, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return fmt.Errorf("%w: not a real directory: %s", ErrPathUnsafe, target)
	}
	return nil
}

// withLedgerLock runs fn while holding the exclusive allocator flock, so every
// ledger mutation — id allocation, status transitions, AND the capture commit
// write — serializes on one lock. It creates the ledger dirs, then routes through
// the shared fsutil.WithFileLock primitive (the one-canonical inter-process
// load-modify-write lock), mapping its sentinels back to the capture-facing
// ErrAllocatorContention / ErrPathUnsafe the ledger callers already test.
func withLedgerLock(repoRoot, issuesRoot string, fn func() error) error {
	if err := ensureLedgerDirs(repoRoot, issuesRoot); err != nil {
		return err
	}
	lockPath := filepath.Join(issuesRoot, lockFilename)
	err := fsutil.WithFileLock(lockPath, lockTimeout, fn)
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("%w: could not acquire allocator lock within %s", ErrAllocatorContention, lockTimeout)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("%w: allocator lock path is unsafe: %s", ErrPathUnsafe, lockPath)
	}
	return err
}

// WithLedgerLock runs fn while holding this ledger's exclusive allocator lock,
// for a caller OUTSIDE this package that mutates the ledger's files directly.
//
// The cold-reading ingest verb is the one such caller: its orphan sweep unlinks
// reading records whose run never committed, and without this that delete races
// this package's own readers. Exporting the lock rather than the path is what
// keeps one file the lock: a caller that rebuilt the path would be holding a
// different lock the day this one moves.
//
// It is NOT reentrant — an flock blocks a second acquisition in the same process
// — so a caller must not hold it across any exported verb of this package, every
// one of which takes it internally.
func WithLedgerLock(repoRoot string, fn func() error) error {
	rr, issuesRoot, err := resolveRoots(repoRoot, "")
	if err != nil {
		return err
	}
	return withLedgerLock(rr, issuesRoot, fn)
}

// minter is the capture family's mint seam (adr-45; mechanics per spc-33). The
// zero value is the production configuration — real clock, crypto entropy;
// tests inject both so same-instant and race cases are deterministic.
var minter recordid.Minter

// reservePath reserves a native timestamp-numeric iss id and creates a
// zero-byte placeholder under open/: flock -> mint -> presence check -> O_EXCL
// create, redrawing a fresh id on any clash (spc-33 ruling 2 — a redraw keeps
// candidates independent and uniform, where a bump would re-derive the next id
// from the ledger's occupancy, a miniature max+1). When forceID is non-empty it
// demands that exact id.
//
// The mint consults no maximum — not the ledger's, not the refs' (adr-45
// ruling 2) — so there is no scan here and no floor parameter: time orders the
// ids and entropy separates same-second minters on other branches. The
// presence check runs first because the O_EXCL create guards open/ alone; a
// clash with a resolved or wontfixed id also redraws.
func reservePath(repoRoot, issuesRoot, slug, forceID string) (string, string, error) {
	// Validate a caller-supplied ForceID against the iss-N shape BEFORE it is used
	// to build a path or create a placeholder — a traversal id (../../evil) must
	// never touch the filesystem outside the ledger, even transiently.
	if forceID != "" && !reIssID.MatchString(forceID) {
		return "", "", fmt.Errorf("%w: ForceID %q must match ^iss-[0-9]+$", ErrPathUnsafe, forceID)
	}
	var resID, resTarget string
	err := withLedgerLock(repoRoot, issuesRoot, func() error {
		if forceID != "" {
			if issPresent(issuesRoot, forceID) {
				return fmt.Errorf("%w: %s already exists in the ledger", ErrDuplicateIssueID, forceID)
			}
			target := filepath.Join(issuesRoot, "open", forceID+"-"+slug+".md")
			fd, cErr := createPlaceholder(target)
			if cErr != nil {
				if os.IsExist(cErr) {
					return fmt.Errorf("%w: %s appeared between scan and create", ErrDuplicateIssueID, forceID)
				}
				return cErr
			}
			syscall.Close(fd)
			resID, resTarget = forceID, target
			return nil
		}

		for attempt := 0; attempt < placeholderRetryBudget; attempt++ {
			issID, mErr := minter.Mint("iss")
			if mErr != nil {
				return mErr
			}
			if issPresent(issuesRoot, issID) {
				continue
			}
			target := filepath.Join(issuesRoot, "open", issID+"-"+slug+".md")
			fd, cErr := createPlaceholder(target)
			if cErr != nil {
				if os.IsExist(cErr) {
					continue
				}
				return cErr
			}
			syscall.Close(fd)
			resID, resTarget = issID, target
			return nil
		}
		return fmt.Errorf("%w: could not mint a free iss id after %d draws", ErrAllocatorContention, placeholderRetryBudget)
	})
	if err != nil {
		return "", "", err
	}
	return resID, resTarget, nil
}

// createPlaceholder does an O_EXCL|O_NOFOLLOW create of the placeholder file.
func createPlaceholder(target string) (int, error) {
	fd, err := syscall.Open(target, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if err == syscall.ELOOP {
			return -1, fmt.Errorf("%w: placeholder path is a symlink: %s", ErrPathUnsafe, target)
		}
		if err == syscall.EEXIST {
			if fi, lerr := os.Lstat(target); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
				return -1, fmt.Errorf("%w: placeholder path is a symlink: %s", ErrPathUnsafe, target)
			}
			return -1, os.ErrExist
		}
		return -1, err
	}
	return fd, nil
}

// issPresent reports whether issID exists in any status dir. It walks
// issueschema.StatusDirs rather than a literal: a status folder this scan does
// not visit is one the mint would happily re-issue an id into.
func issPresent(issuesRoot, issID string) bool {
	prefix := issID + "-"
	exact := issID + ".md"
	for _, sub := range issueschema.StatusDirs {
		entries, err := os.ReadDir(filepath.Join(issuesRoot, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			if n == exact || (len(n) > len(prefix) && n[:len(prefix)] == prefix && filepath.Ext(n) == ".md") {
				return true
			}
		}
	}
	return false
}

// cancelReservation removes a zero-byte placeholder idempotently. It refuses a
// symlinked or non-empty target (real content is the caller's transactional
// responsibility).
func cancelReservation(path string) error {
	fi, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: refusing to cancel a symlinked placeholder: %s", ErrPathUnsafe, path)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: placeholder is not a regular file: %s", ErrPathUnsafe, path)
	}
	if fi.Size() != 0 {
		return fmt.Errorf("refusing to cancel non-empty placeholder (%d bytes): %s", fi.Size(), path)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// cleanOrphanPlaceholders sweeps zero-byte iss-N placeholders older than the
// threshold from open/. Tolerates a virgin ledger. Refuses symlinked roots.
func cleanOrphanPlaceholders(issuesRoot string) error {
	fi, err := os.Lstat(issuesRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: issuesRoot is a symlink: %s", ErrPathUnsafe, issuesRoot)
	}
	openDir := filepath.Join(issuesRoot, "open")
	ofi, err := os.Lstat(openDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if ofi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: issuesRoot/open is a symlink: %s", ErrPathUnsafe, openDir)
	}
	entries, err := os.ReadDir(openDir)
	if err != nil {
		return nil
	}
	now := time.Now()
	for _, e := range entries {
		if !issFileNumRe.MatchString(e.Name()) {
			continue
		}
		cand := filepath.Join(openDir, e.Name())
		cfi, err := os.Lstat(cand)
		if err != nil {
			continue
		}
		if cfi.Mode()&os.ModeSymlink != 0 || !cfi.Mode().IsRegular() {
			continue
		}
		if cfi.Size() != 0 {
			continue
		}
		if now.Sub(cfi.ModTime()) <= orphanAgeThreshold {
			continue
		}
		if !orphanStillRemovable(cand, cfi) {
			continue
		}
		// Test-only seam (nil in production): fires in the residual window between
		// the pre-unlink re-check and the unlink, to force the documented iss-102
		// interleaving where a stalled commit's fill lands here.
		if beforeOrphanRemoveHook != nil {
			beforeOrphanRemoveHook(cand)
		}
		os.Remove(cand)
	}
	return nil
}

// orphanStillRemovable re-verifies, in the tightest possible window immediately
// before the unlink, that cand is still the same zero-byte inode the sweep
// classified as an orphan. A capture commits by atomically renaming a full issue
// file over its reserved placeholder, and that write happens OUTSIDE this
// (lockless) sweep — so between the caller's Lstat and its os.Remove a stalled
// capture can land its committed file at this exact path. Re-checking here means
// the sweep never deletes a placeholder a commit has since replaced or filled,
// closing the race for any commit that lands before this check. The residual
// micro-window between this stat and the unlink can only be fully eliminated by
// serialising the commit write on the ledger lock (see commitCapture in
// workflow.go), which is outside this sweep's reach.
func orphanStillRemovable(cand string, seen os.FileInfo) bool {
	recheck, err := os.Lstat(cand)
	if err != nil {
		return false
	}
	if recheck.Mode()&os.ModeSymlink != 0 || !recheck.Mode().IsRegular() {
		return false
	}
	if recheck.Size() != 0 || !os.SameFile(seen, recheck) {
		return false
	}
	return true
}

// findIssue locates issID across the three status dirs, mirroring find_issue.
func findIssue(issuesRoot, issID string) (string, State, error) {
	if !reIssID.MatchString(issID) {
		return "", "", fmt.Errorf("invalid iss-N identifier: %q", issID)
	}
	prefix := issID + "-"
	exact := issID + ".md"
	type match struct {
		path   string
		status State
	}
	var matches []match
	for _, sub := range statusDirs {
		dir := filepath.Join(issuesRoot, statusDirName[sub])
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			if filepath.Ext(n) != ".md" {
				continue
			}
			if n == exact {
				matches = append(matches, match{filepath.Join(dir, n), sub})
				continue
			}
			m := issFileNumRe.FindStringSubmatch(n)
			if len(n) > len(prefix) && n[:len(prefix)] == prefix && m != nil && issFamily+"-"+m[1] == issID {
				matches = append(matches, match{filepath.Join(dir, n), sub})
			}
		}
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("%w: %s not found in any status directory", ErrUnknownIssueID, issID)
	}
	if len(matches) > 1 {
		return "", "", fmt.Errorf("%w: %s present in multiple files", ErrDuplicateIssueID, issID)
	}
	return matches[0].path, matches[0].status, nil
}
