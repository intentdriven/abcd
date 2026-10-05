package interview

// treewatch.go holds an AI-written interview's role to the paths its contract
// lets it change. The role runs with the person's own route in the checkout,
// and with no host session nobody watches its edits between the questions
// abcd draws, so abcd does: before each dispatch it reads the state of every
// place the role could leave something behind, after the dispatch it reads it
// again, and a path whose state moved that the interview did not grant stops
// the interview, naming each. abcd cannot tell who wrote a path in that
// window, so a path anything else on the machine changes while the role runs
// stops it too (a second interview in the same checkout among them: one
// checkout is one session).
//
// Five places are read:
//
//   - What differs from HEAD, as git lists it (gitutil.Status, the one reader
//     of `git status --porcelain=v1 -z --untracked-files=all`): each path
//     paired with its content's mode and hash, so a second edit to a file
//     already changed before the dispatch moves its state too.
//   - What git ignores (the same listing's --ignored=matching entries, an
//     ignored directory walked whole), and an untracked nested repository
//     walked the same way: each file by its mode, size and modification time,
//     never its content, since an ignored tree (a build's output, the local
//     tier's scratch) can be large. The local tier is watched like any other
//     path: a push receipt under preflight-receipts/ or the handover changed
//     by the role stops the interview.
//   - git's own directory, which no status lists: every entry of hooks/ and
//     info/, the configuration (config, and config.worktree), HEAD, the refs
//     (refs/ and packed-refs), each submodule's git directory under modules/
//     (read as the common directory is: git status runs status inside a
//     populated submodule), and a linked worktree's .git file, by mode, size
//     and content hash. A hook runs on the person's next git command, the
//     configuration can name a hooks directory, an alias or a credential
//     helper, and HEAD and the refs decide what the next commit or push
//     carries. In a linked worktree these live in the repository's common
//     directory and are read there, besides the worktree's own HEAD and
//     config.worktree. Every
//     worktree's entry under the common directory's worktrees/ is read too:
//     the entry itself, so one made or removed is noticed, and its HEAD,
//     commondir, gitdir, config.worktree and locked, which decide where git
//     places that worktree, which configuration and hooks it runs there, and
//     where git lists it. git follows a link there, so a link (a
//     dotfiles-managed hooks directory, a hook or a configuration file) is
//     read where it leads too.
//   - The directory each core.hooksPath value names, in any scope the
//     person's git reads, when it is outside the working tree (inside, the
//     listing above already covers it), read as git's own directory is. The
//     value is resolved through every link first, so a hooks directory
//     reached through a link (a dotfiles-managed one, or an in-tree link to a
//     directory elsewhere) is read where git runs its hooks from, held to
//     the budget maxFollowedEntries and maxFollowedBytes set.
//   - The push receipts (.abcd/.work.local/preflight-receipts/) of every
//     worktree git lists, by mode, size and content hash: the pre-push gate
//     takes a commit's receipt from any of them, so a receipt written into
//     another worktree's local tier during a turn stops the interview too.
//     They are held to the same budget, since a worktree stands where its
//     registration says.
//
// git's own directory and common directory are placed by the run's first
// reading, taken before any role ran, and every later reading reads those
// same directories. Before it runs git, a later reading reads the two files
// that place them, a linked worktree's .git file and the git directory's
// commondir: one changed since is named as a change, and neither git nor the
// reading follows where it leads. It then sizes the files git reads whole on
// every command (HEAD, the configuration, the packed refs, info/exclude), in
// those directories and in each submodule's git directory, against
// maxHashedBytes, which also bounds what the reading hashes of the paths git
// lists and of git's own directory, so a file a role grows there is refused,
// named, before it is read. Only then does it ask git where its directories
// and its working tree are, refusing an answer other than the first
// reading's: a core.worktree written since would have git status walk
// another tree.
//
// Two places are left out because abcd itself writes them while a role runs:
// the run's own turn directory, where abcd writes each brief and the role its
// receipt, and the local transcript store a checkout can pull its runners'
// transcripts into. Every other run's turn directory is watched. Two paths are
// left out because something else on the machine writes them while a role
// runs and neither executes anything: a file named .DS_Store anywhere
// (Finder's metadata), and the host scheduler's lock at the repository's root,
// .claude/scheduled_tasks.lock, that one path alone. Every other path under
// .claude/ is watched: the host's settings there can name hooks.
//
// Paths are relative to the repository's root; a path outside it (a linked
// worktree's common directory, a hooks directory elsewhere) is named in full.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxStatusBytes bounds the status listing; a tree whose listing exceeds it
// is refused rather than read in part.
const maxStatusBytes = 8 << 20

// maxWatched bounds the paths one reading holds. A tree past it is refused
// rather than read in part, before the first dispatch: a guard that stopped
// counting would let a role write past the bound unseen. It is several times
// what a large checkout's local tier and build output hold, so a tree
// reaching it has something in it worth clearing.
var maxWatched = 500_000

// maxFollowedEntries and maxFollowedBytes bound what one reading reads
// outside what it owns: what the links it follows lead to, every hooks
// directory core.hooksPath names outside the tree, and the push receipts of
// every worktree git lists. Every entry read there (a directory included)
// and every byte of the files hashed there is counted. A reading past either
// is refused, naming the link, the core.hooksPath value or the receipts
// directory, before the entry past it is read, rather than walking toward
// maxWatched: a role can plant a link to a large tree in one write where the
// guard follows links (its own turn directory's push receipts, registered as
// a worktree, among them), point core.hooksPath at one, or register a
// worktree whose receipts directory is one, and hashing that whole would
// hold the turn for minutes before the guard refused. What the guard
// legitimately reads there is small: git's sample hooks are 14 files and
// 26 KiB, a checkout keeps at most 50 push receipts of 76 bytes each
// (scripts/preflight-receipt.sh), and a dotfiles-managed hooks directory
// holds tens of scripts. The budget is some twenty times the receipts' cap
// and holds a compiled hook binary, and reading all of it takes tens of
// milliseconds. It is shared by everything one reading reads there, so
// planting many links or hooks directories does not multiply it, with one
// exception: each receipts directory is held to the entry budget on its own,
// since a checkout's receipts grow with its worktrees (twenty worktrees each
// keeping their 50 would pass a shared count), while their bytes, a few KiB
// for any real checkout, stay shared.
var (
	maxFollowedEntries       = 1024
	maxFollowedBytes   int64 = 64 << 20
)

// maxHashedBytes bounds the bytes one reading hashes of what it owns: the
// paths git lists as differing from HEAD, and git's own directory. Each
// listed path's size is counted from its Lstat before any is hashed, and a
// listing past the bound is refused, naming its largest path, so a tree the
// person already holds past it is refused before the first dispatch and a
// file a role writes past it is refused before it is read: a sparse file
// costs one write to make and minutes to hash whole. A file in git's own
// directory is counted as it is reached, before it is read. The bound is far
// past what a source tree's changes and git's own directory hold (a large
// repository's packed refs are tens of MiB), and hashing all of it takes
// about a second. A large file the person keeps in the tree belongs in
// .gitignore: an ignored path is read by its mode, size and modification
// time, never its content, which a role cannot set back without running a
// command, and a role whose contract allows no command cannot.
var maxHashedBytes int64 = 2 << 30

// maxPointerBytes bounds a file that places git's own directory (a linked
// worktree's .git file, a git directory's commondir): each holds one path.
const maxPointerBytes = 64 << 10

// treeState is each watched path mapped to its status and its state.
type treeState map[string]string

// schedulerLock is the one path under .claude/ left out of the reading: the
// host scheduler's lock, which executes nothing.
const schedulerLock = ".claude/scheduled_tasks.lock"

// finderMetadata is the base name of Finder's metadata file, left out of the
// reading wherever it stands.
const finderMetadata = ".DS_Store"

// UnexpectedChangesError is an interview stopped because paths the interview
// does not let its role change changed while the role ran. abcd cannot tell
// who changed them: the role, or anything else writing in that window.
type UnexpectedChangesError struct {
	Role string
	// Paths are the paths changed, relative to the repository's root.
	Paths []string
}

func (e *UnexpectedChangesError) Error() string {
	return fmt.Sprintf("%d path(s) changed while the %s ran that this interview does not let it change, so the interview stopped: %s; "+
		"read each (git status, git diff) and restore what you did not ask for",
		len(e.Paths), e.Role, strings.Join(e.Paths, ", "))
}

// treeReader is one reading of the places a role could change.
type treeReader struct {
	repo string
	// realRepo is repo with every link resolved, so a path read through its
	// real spelling is still keyed by its place in the tree.
	realRepo string
	// skip are repository-relative slash paths left out, with everything
	// under them.
	skip []string
	st   treeState
	// followedEntries and followedBytes are what this reading has read
	// outside what it owns so far, held to maxFollowedEntries and
	// maxFollowedBytes.
	followedEntries int
	followedBytes   int64
	// hashedBytes is what this reading has hashed of what it owns so far,
	// held to maxHashedBytes.
	hashedBytes int64
	// gitDir and common are git's own directory and common directory, as
	// pin placed them when it is set and as git names them when it is not.
	gitDir, common string
	pin            *gitPin
	// modules are the submodules' git directories under common's modules/,
	// relative to common, as sizeGitReads found them.
	modules []string
}

// gitPin is where a run's first reading found git's own directory and
// common directory, and the state of each file that places them: a linked
// worktree's .git file and the git directory's commondir. Every later reading
// of the run reads those same directories, and before it runs git it reads
// those files: one changed since is named, as a change, without git or the
// reading following where it leads. So a role writing a commondir, or a .git
// file, that leads to a large tree cannot make the next reading read it.
type gitPin struct {
	// top is the working tree git names, which git status walks.
	gitDir, common, top string
	// pointers maps each placing file's key to its state.
	pointers map[string]string
}

// redirectedError is a reading refused because a file that places git's own
// directory changed since the run's first reading: git would now read
// another directory than the one the reading holds the role to.
type redirectedError struct {
	// Paths are the files changed, keyed as the reading keys them.
	Paths []string
}

func (e *redirectedError) Error() string {
	return fmt.Sprintf("%s changed since the interview's first reading, so git would read its own directory elsewhere; "+
		"the reading does not follow it: restore it and run the interview again", strings.Join(e.Paths, ", "))
}

// reach is what took a reading outside what it owns, and so to the budget: a
// followed link, a hooks directory core.hooksPath names outside the tree, or
// a worktree's push receipts.
type reach struct {
	// what names it in a refusal, and fix says how to clear it.
	what, fix string
	// own holds the reach to the entry budget on its own, counted in
	// entries, rather than sharing the reading's count.
	own     bool
	entries int
}

// linkReach is the link key, followed.
func linkReach(key string) *reach {
	return &reach{what: "the link " + key, fix: "remove the link, or point it at what git should read there"}
}

// readTree reads the state of every watched place under repo, leaving out
// turnDir, the run's own turn directory (relative to repo, slash-separated),
// placing git's own directories afresh.
func readTree(repo, turnDir string) (treeState, error) {
	return readTreePinned(repo, turnDir, nil)
}

// readTreePinned is readTree reading git's own directories where pin placed
// them. An empty pin is filled by this reading, the run's first.
func readTreePinned(repo, turnDir string, pin *gitPin) (treeState, error) {
	r := &treeReader{repo: repo, realRepo: fsutil.RealExistingPath(repo), skip: []string{history.LocalStoreRelPath}, st: treeState{}, pin: pin}
	if turnDir != "" {
		r.skip = append(r.skip, turnDir)
	}
	if err := r.read(); err != nil {
		return nil, fmt.Errorf("interview: the working tree's state cannot be read, so the role's changes cannot be held to its contract: %w", err)
	}
	return r.st, nil
}

func (r *treeReader) read() error {
	// git's own directories are placed first: git reads where they lead on
	// every command, the status listing's included.
	if err := r.placeGitDirs(); err != nil {
		return err
	}
	entries, err := gitutil.Status(r.repo, maxStatusBytes, gitutil.StatusOptions{Ignored: true})
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(r.repo)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := r.sizeListed(root, entries); err != nil {
		return err
	}
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Path, "/"):
			// An ignored directory, or an untracked nested repository: git
			// lists the directory alone, so it is walked.
			err = r.walkStat(e.XY, strings.TrimSuffix(e.Path, "/"))
		case e.XY == "!!":
			err = r.statOne(e.XY, e.Path)
		default:
			err = r.hashOne(root, e.XY, e.Path)
			if err == nil && e.Orig != "" {
				// A rename's or copy's source is watched with it.
				err = r.hashOne(root, e.XY+"<", e.Orig)
			}
		}
		if err != nil {
			return err
		}
	}
	return r.readGitDirs()
}

// placeGitDirs places git's own directory and common directory, and sizes
// what git reads whole there (sizeGitReads): where pin placed them, once the
// files that place them are read unchanged and what git reads there is sized,
// or as git names them, filling an empty pin. A pinned reading sizes before it
// asks git where its directories are, since git reads the configuration whole
// to answer, and asks for the working tree with them, so a core.worktree
// pointed at another tree is refused before git status walks it.
func (r *treeReader) placeGitDirs() error {
	pinned := r.pin != nil && r.pin.gitDir != ""
	if pinned {
		var moved []string
		for _, key := range slices.Sorted(maps.Keys(r.pin.pointers)) {
			if pointerState(r.pointerPath(key)) != r.pin.pointers[key] {
				moved = append(moved, key)
			}
		}
		if len(moved) > 0 {
			return &redirectedError{Paths: moved}
		}
		r.gitDir, r.common = r.pin.gitDir, r.pin.common
		if err := r.sizeGitReads(); err != nil {
			return err
		}
	}
	gitDir, common, top, err := r.namedGitDirs()
	if err != nil {
		return err
	}
	switch {
	case r.pin == nil:
	case r.pin.gitDir == "":
		r.pin.gitDir, r.pin.common, r.pin.top, r.pin.pointers = gitDir, common, top, map[string]string{}
		for _, p := range []string{filepath.Join(r.repo, ".git"), filepath.Join(gitDir, "commondir")} {
			st := pointerState(p)
			if strings.HasPrefix(st, "past ") {
				return fmt.Errorf("%s holds more than %d KiB, more than the one path it names; restore it and run the interview again", r.keyOf(filepath.Dir(p), filepath.Base(p)), maxPointerBytes>>10)
			}
			r.pin.pointers[r.keyOf(filepath.Dir(p), filepath.Base(p))] = st
		}
	case gitDir != r.pin.gitDir || common != r.pin.common || top != r.pin.top:
		// The files that place them read unchanged, so something else
		// redirected git (core.worktree among them): refused rather than
		// read.
		return fmt.Errorf("git names %s, %s and %s as its own directory, common directory and working tree, not %s, %s and %s as at the interview's first reading; "+
			"the reading does not follow it: restore where git reads its directories and its working tree (core.worktree) and run the interview again",
			gitDir, common, top, r.pin.gitDir, r.pin.common, r.pin.top)
	}
	r.gitDir, r.common = gitDir, common
	if pinned {
		return nil
	}
	return r.sizeGitReads()
}

// gitDirReadsWhole and commonReadsWhole are the files git reads whole on
// every command, the status listing's included, in git's own directory and
// in the common directory, and in each submodule's git directory, which git
// status reads as it runs status inside the submodule. Other files git reads
// there bound themselves: a loose ref and the index are refused at their
// first bytes when they are not one.
var (
	gitDirReadsWhole = []string{"HEAD", "commondir", "config.worktree"}
	commonReadsWhole = []string{"config", "packed-refs", "info/exclude"}
)

// sizeGitReads sizes what git reads whole on every command, where links
// lead, in git's own directory, the common directory and every submodule's
// git directory under the common directory's modules/, refusing past what is
// left of maxHashedBytes before git runs, naming the largest: git reads it
// before the reading can bound its own hashing of it, so a file a role grows
// there would otherwise hold the turn while git read it.
func (r *treeReader) sizeGitReads() error {
	mods, err := moduleGitDirs(r.common)
	if err != nil {
		return err
	}
	r.modules = mods
	type place struct {
		dir   string
		files []string
	}
	places := []place{{r.gitDir, gitDirReadsWhole}, {r.common, commonReadsWhole}}
	for _, m := range mods {
		places = append(places, place{filepath.Join(r.common, filepath.FromSlash(m)), slices.Concat(gitDirReadsWhole, commonReadsWhole)})
	}
	var total, largest int64
	var largestKey string
	for _, at := range places {
		for _, f := range at.files {
			fi, err := os.Stat(filepath.Join(at.dir, filepath.FromSlash(f)))
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				continue
			}
			if err != nil {
				return err
			}
			if !fi.Mode().IsRegular() {
				continue
			}
			total += fi.Size()
			if fi.Size() > largest {
				largest, largestKey = fi.Size(), r.keyOf(at.dir, f)
			}
		}
	}
	if total > maxHashedBytes-r.hashedBytes {
		return fmt.Errorf("%s holds %d MiB, and git reads it whole on every command, past the %d MiB one reading hashes of the tree and git's own directory, so git is not run; "+
			"restore it and run the interview again", largestKey, largest>>20, maxHashedBytes>>20)
	}
	return nil
}

// pointerPath is the full path of a placing file's key.
func (r *treeReader) pointerPath(key string) string {
	if filepath.IsAbs(filepath.FromSlash(key)) {
		return filepath.FromSlash(key)
	}
	return filepath.Join(r.repo, filepath.FromSlash(key))
}

// namedGitDirs is git's own directory, its common directory and the working
// tree, as git names them, in full.
func (r *treeReader) namedGitDirs() (gitDir, common, top string, err error) {
	out, err := gitutil.Run(r.repo, "rev-parse", "--git-dir", "--git-common-dir", "--show-toplevel")
	if err != nil {
		return "", "", "", err
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 3 || slices.Contains(lines, "") {
		return "", "", "", fmt.Errorf("git named no git directory, common directory and working tree (%q)", out)
	}
	return r.abs(lines[0]), r.abs(lines[1]), r.abs(lines[2]), nil
}

// abs is p in full, a relative p taken from the repository.
func (r *treeReader) abs(p string) string {
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.repo, p)
	}
	return filepath.Clean(p)
}

// pointerState is the state of a file that places git's own directory: its
// kind, and a file's mode, size and content hash, or "past" and its size,
// unread, when it holds more than one path can.
func pointerState(p string) string {
	fi, err := os.Lstat(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "absent"
	case err != nil:
		return "unreadable " + err.Error()
	case fi.Mode().IsRegular() && fi.Size() > maxPointerBytes:
		return fmt.Sprintf("past %d", fi.Size())
	case fi.IsDir():
		return fi.Mode().String()
	}
	root, err := os.OpenRoot(filepath.Dir(p))
	if err != nil {
		return "unreadable " + err.Error()
	}
	defer root.Close()
	st, _, err := contentState(root, filepath.Base(p), maxPointerBytes)
	if err != nil {
		return "unreadable " + err.Error()
	}
	return st
}

// sizeListed counts the bytes the listed paths git hashes would take,
// refusing past maxHashedBytes before any is read, naming the largest.
func (r *treeReader) sizeListed(root *os.Root, entries []gitutil.StatusEntry) error {
	var total, largest int64
	var largestPath string
	var largestTracked bool
	count := func(rel string, tracked bool) error {
		if r.skipped(rel) || exempt(rel) {
			return nil
		}
		fi, err := root.Lstat(rel)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		total += fi.Size()
		if fi.Size() > largest {
			largest, largestPath, largestTracked = fi.Size(), rel, tracked
		}
		return nil
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Path, "/") || e.XY == "!!" {
			continue
		}
		if err := count(e.Path, e.XY != "??"); err != nil {
			return err
		}
		// A rename's or copy's source is a path git tracks.
		if e.Orig != "" {
			if err := count(e.Orig, true); err != nil {
				return err
			}
		}
	}
	if total > maxHashedBytes {
		// .gitignore applies only to a path git does not track.
		fix := "remove it, or add it to .gitignore if git need not track it (an ignored path is read by its size and modification time, never its content)"
		if largestTracked {
			fix = "git tracks it, so .gitignore does not apply to it: restore it to what HEAD holds (git restore), unstage it, or commit it"
		}
		return fmt.Errorf("the paths git lists as differing from HEAD hold %d MiB, past the %d MiB one reading hashes of the tree and git's own directory, so none is read; "+
			"the largest is %s (%d MiB): %s, and run the interview again",
			total>>20, maxHashedBytes>>20, largestPath, largest>>20, fix)
	}
	return nil
}

// hash is the state of rel under root, keyed key, hashed within what is left
// to hash of it: for an entry reached through via, the size its charge
// counted, so a file grown since is not read past it; for one the reading
// owns, what is left of maxHashedBytes, which it then counts.
func (r *treeReader) hash(root *os.Root, rel, key string, via *reach, charged int64) (string, error) {
	limit := maxHashedBytes - r.hashedBytes
	if via != nil {
		limit = charged
	}
	st, n, err := contentState(root, rel, limit)
	if errors.Is(err, errPastBound) {
		if via != nil {
			return "", fmt.Errorf("%s leads past what one reading follows links to (%s grew to %d MiB as it was read), so it is not read; %s, and run the interview again",
				via.what, key, n>>20, via.fix)
		}
		return "", fmt.Errorf("%s holds %d MiB, which takes one reading past the %d MiB one reading hashes of the tree and git's own directory, so it is not read; "+
			"remove it, or add it to .gitignore if it is the tree's and git need not track it (an ignored path is read by its size and modification time, never its content), and run the interview again",
			key, n>>20, maxHashedBytes>>20)
	}
	if err != nil {
		return "", err
	}
	if via == nil {
		r.hashedBytes += n
	}
	return st, nil
}

// skipped reports whether rel is left out of the reading.
func (r *treeReader) skipped(rel string) bool {
	for _, s := range r.skip {
		if rel == s || strings.HasPrefix(rel, s+"/") {
			return true
		}
	}
	return false
}

// exempt reports whether key is one of the two paths something else on the
// machine writes while a role runs that execute nothing: a file named
// .DS_Store anywhere, and the scheduler's lock at the root.
func exempt(key string) bool {
	return key == schedulerLock || path.Base(key) == finderMetadata
}

// put records one path's state, refusing past maxWatched. An exempt path is
// not recorded.
func (r *treeReader) put(key, state string) error {
	if exempt(key) {
		return nil
	}
	if _, ok := r.st[key]; !ok && len(r.st) >= maxWatched {
		return fmt.Errorf("the tree holds more than %d paths outside what git tracks unchanged (untracked, ignored, or in git's own directory), "+
			"too many to read around each turn; remove what it no longer needs (a scratch directory, a build's output) and run the interview again", maxWatched)
	}
	r.st[key] = state
	return nil
}

// charge counts fi, an entry about to be read where via leads, against the
// budget for what one reading reads outside what it owns, refusing past it
// before the entry is read. An entry not reached that way (via nil) is held
// to maxHashedBytes as it is hashed instead.
func (r *treeReader) charge(via *reach, fi fs.FileInfo) error {
	if via == nil {
		return nil
	}
	entries, limit := &r.followedEntries, fmt.Sprintf("%d entries and %d MiB in all", maxFollowedEntries, maxFollowedBytes>>20)
	if via.own {
		entries, limit = &via.entries, fmt.Sprintf("%d entries for each worktree's push receipts and %d MiB in all", maxFollowedEntries, maxFollowedBytes>>20)
	}
	*entries++
	if fi.Mode().IsRegular() {
		r.followedBytes += fi.Size()
	}
	if *entries > maxFollowedEntries || r.followedBytes > maxFollowedBytes {
		return fmt.Errorf("%s leads past what one reading follows links to (%s), "+
			"more than git's own directory, a hooks directory or the push receipts hold, so it is not read; "+
			"%s, and run the interview again", via.what, limit, via.fix)
	}
	return nil
}

// hashOne records a path git lists by its content's mode and hash.
func (r *treeReader) hashOne(root *os.Root, xy, rel string) error {
	if r.skipped(rel) || exempt(rel) {
		return nil
	}
	h, err := r.hash(root, rel, rel, nil, 0)
	if err != nil {
		return err
	}
	return r.put(rel, xy+" "+h)
}

// statOne records an ignored path by its mode, size and modification time.
func (r *treeReader) statOne(xy, rel string) error {
	if r.skipped(rel) {
		return nil
	}
	fi, err := os.Lstat(filepath.Join(r.repo, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return r.put(rel, xy+" absent")
	}
	if err != nil {
		return err
	}
	return r.put(rel, xy+" "+statState(fi))
}

// walkStat records every file under the directory rel by its mode, size and
// modification time. A directory itself is not recorded: what it holds is,
// and git keeps no empty directory either. A link is recorded, never
// followed.
func (r *treeReader) walkStat(xy, rel string) error {
	if r.skipped(rel) {
		return nil
	}
	base := filepath.Join(r.repo, filepath.FromSlash(rel))
	return filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		sub, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		key := path.Join(rel, filepath.ToSlash(sub))
		if r.skipped(key) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return r.put(key, xy+" "+statState(fi))
	})
}

// statState is a file's mode, size and modification time.
func statState(fi fs.FileInfo) string {
	return fmt.Sprintf("%v %d %d", fi.Mode(), fi.Size(), fi.ModTime().UnixNano())
}

// readGitDirs records git's own directory and every hooks directory
// core.hooksPath names outside the working tree, each entry by its mode,
// size and content hash.
func (r *treeReader) readGitDirs() error {
	gitDir, common := r.gitDir, r.common
	// A linked worktree's .git is a file naming its git directory.
	// The git directory it names is read below, so a link there is not
	// followed.
	if fi, err := os.Lstat(filepath.Join(r.repo, ".git")); err == nil && !fi.IsDir() {
		if err := r.hashAt(r.repo, ".git", false, nil); err != nil {
			return err
		}
	}
	for _, rel := range gitDirEntries {
		if err := r.hashIn(common, rel); err != nil {
			return err
		}
	}
	if err := r.readModules(); err != nil {
		return err
	}
	if err := r.readWorktrees(common); err != nil {
		return err
	}
	if gitDir != common {
		for _, rel := range []string{"config.worktree", "HEAD"} {
			if err := r.hashIn(gitDir, rel); err != nil {
				return err
			}
		}
	}
	hooksPaths, err := gitutil.HooksPaths(r.repo)
	if err != nil {
		return err
	}
	fold := fsutil.CaseFoldingFS()
	realGitDir, realCommon := fsutil.RealExistingPath(gitDir), fsutil.RealExistingPath(common)
	for _, hp := range hooksPaths {
		// git runs a hook from where the value leads, so a link on the way
		// (the directory itself, or one above it) is resolved before the
		// directory is placed and read.
		dir := fsutil.RealExistingPath(r.abs(filepath.FromSlash(hp)))
		// Inside the working tree, and outside git's own directory, the
		// status listing already reads it.
		if fsutil.PathWithin(dir, r.realRepo, fold) && !fsutil.PathWithin(dir, realGitDir, fold) && !fsutil.PathWithin(dir, realCommon, fold) {
			continue
		}
		// Outside the tree it is held to the budget, named by the value as
		// configured: a hooks directory is tens of scripts, so one past the
		// budget is not one git should run from unread.
		via := &reach{
			what: "the hooks directory core.hooksPath names, " + hp + ",",
			fix:  "point core.hooksPath at a hooks directory, or unset it where it is set (git config --show-origin --get-all core.hooksPath names where)",
		}
		if err := r.hashAt(filepath.Dir(dir), filepath.Base(dir), true, via); err != nil {
			return err
		}
	}
	return r.readReceipts()
}

// worktreeFiles are the files of a linked worktree's entry under the common
// directory's worktrees/ that git reads to place the worktree and run there:
// its HEAD, the common directory it uses (and so the configuration, the
// hooks and the refs), the working tree it belongs to (and so the place git
// lists it, whose local tier the pre-push gate reads), its own configuration,
// and its lock.
var worktreeFiles = []string{"HEAD", "commondir", "gitdir", "config.worktree", "locked"}

// readWorktrees records every entry of common's worktrees/ by its kind, so
// one made or removed is noticed, and each entry's worktreeFiles by mode,
// size and content hash. A link, there or as worktrees/ itself, is recorded
// by its target's text and read where it leads, as git reads it.
func (r *treeReader) readWorktrees(common string) error {
	dir := filepath.Join(common, "worktrees")
	fi, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// via is the link the entries are reached through, when worktrees/ is
	// one: what it leads to is held to the followed-link budget.
	var via *reach
	if fi.Mode()&fs.ModeSymlink != 0 {
		if err := r.hashAt(common, "worktrees", false, nil); err != nil {
			return err
		}
		via = linkReach(r.keyOf(common, "worktrees"))
		dir = fsutil.RealExistingPath(dir)
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		at, entryVia := filepath.Join(dir, e.Name()), via
		if via != nil {
			fi, err := e.Info()
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if err := r.charge(via, fi); err != nil {
				return err
			}
		}
		if e.Type()&fs.ModeSymlink != 0 {
			if err := r.hashAt(dir, e.Name(), false, nil); err != nil {
				return err
			}
			entryVia = linkReach(r.keyOf(dir, e.Name()))
			at = fsutil.RealExistingPath(at)
		} else if err := r.put(r.keyOf(dir, e.Name()), "worktree "+e.Type().String()); err != nil {
			return err
		}
		if fi, err := os.Stat(at); err != nil || !fi.IsDir() {
			continue
		}
		for _, f := range worktreeFiles {
			if err := r.hashAt(at, f, true, entryVia); err != nil {
				return err
			}
		}
	}
	return nil
}

// receiptsRel is where a checkout's push receipts stand, under each worktree
// (scripts/preflight-receipt.sh).
const receiptsRel = ".abcd/.work.local/preflight-receipts"

// readReceipts records the push receipts of every worktree git lists by
// mode, size and content hash: the pre-push gate takes a receipt for the
// commit being pushed from any of them, so one written into another
// worktree's local tier, or into a worktree the role made git list, would
// pass a push the gates never read.
func (r *treeReader) readReceipts() error {
	wts, err := gitutil.ListWorktrees(r.repo, maxStatusBytes)
	if err != nil {
		return err
	}
	for _, wt := range wts {
		if wt.Path == "" {
			continue
		}
		dir := filepath.Join(wt.Path, filepath.FromSlash(receiptsRel))
		// Held to the budget: a worktree's path is what its registration
		// says, which a role can write, so its receipts directory may be any
		// tree on the machine.
		via := &reach{
			what: "the push receipts directory " + filepath.ToSlash(dir),
			fix:  "remove what is not a push receipt from it, or the worktree registration that names it if you did not make that worktree (git worktree list names each)",
			own:  true,
		}
		if err := r.hashAt(filepath.Dir(dir), filepath.Base(dir), true, via); err != nil {
			return err
		}
	}
	return nil
}

// gitDirEntries are the entries of a common directory, and of each
// submodule's git directory, read by mode, size and content hash.
var gitDirEntries = []string{"hooks", "info", "config", "config.worktree", "HEAD", "packed-refs", "refs"}

// readModules records every submodule's git directory sizeGitReads found
// under the common directory's modules/ as the common directory is read
// (gitDirEntries): git status runs status inside a populated submodule, and
// the submodule's hooks, configuration, HEAD and refs decide what runs there
// and what its next commit or push carries.
func (r *treeReader) readModules() error {
	for _, m := range r.modules {
		for _, rel := range gitDirEntries {
			if err := r.hashIn(r.common, path.Join(m, rel)); err != nil {
				return err
			}
		}
	}
	return nil
}

// moduleGitDirs is every submodule's git directory under common's modules/,
// relative to common and slash-separated. A directory there holding a HEAD, a
// config or a hooks entry is a submodule's git directory, and its own
// modules/ is read the same way; any other directory is the leading part of a
// submodule's name, which may hold a slash, and is read through. A link is
// never followed.
func moduleGitDirs(common string) ([]string, error) {
	root, err := os.OpenRoot(common)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var out []string
	err = moduleGitDirsIn(root.FS(), "modules", &out)
	return out, err
}

func moduleGitDirsIn(fsys fs.FS, dir string, out *[]string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := path.Join(dir, e.Name())
		if isGitDir(fsys, sub) {
			*out = append(*out, sub)
			sub = path.Join(sub, "modules")
		}
		if err := moduleGitDirsIn(fsys, sub, out); err != nil {
			return err
		}
	}
	return nil
}

// isGitDir reports whether dir holds a HEAD, a config or a hooks entry, as
// a submodule's git directory does.
func isGitDir(fsys fs.FS, dir string) bool {
	for _, n := range []string{"HEAD", "config", "hooks"} {
		if _, err := fs.Lstat(fsys, path.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// hashIn records rel under base, and every entry under it when it is a
// directory, by mode, size and content hash, keyed by its path relative to
// the repository when it is inside it and in full when not. A link is
// recorded by its target's text, and what it leads to is read too, once: git
// follows a link in its own directory or in a hooks directory, so a hooks
// directory, a configuration file or a hook that is a link (a
// dotfiles-managed one) runs or is read where it leads.
func (r *treeReader) hashIn(base, rel string) error {
	return r.hashAt(base, rel, true, nil)
}

// hashAt is hashIn, following each link it meets only when follow is set,
// and never further than the link's own target. via, when set, is what took
// the reading to base and rel: every entry read is held to the budget before
// it is read.
func (r *treeReader) hashAt(base, rel string, follow bool, via *reach) error {
	root, err := os.OpenRoot(base)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		// Nothing can stand under a base that is absent or is not a
		// directory.
		return nil
	}
	if err != nil {
		return err
	}
	defer root.Close()
	fi, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		// A file, or a link standing where a directory is expected: recorded
		// as itself, never followed.
		if err := r.charge(via, fi); err != nil {
			return err
		}
		key := r.keyOf(base, rel)
		h, err := r.hash(root, rel, key, via, fi.Size())
		if err != nil {
			return err
		}
		if err := r.put(key, "git "+h); err != nil {
			return err
		}
		if follow && fi.Mode()&fs.ModeSymlink != 0 {
			return r.followLink(base, rel)
		}
		return nil
	}
	return fs.WalkDir(root.FS(), rel, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		var charged int64
		if via != nil {
			fi, err := d.Info()
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := r.charge(via, fi); err != nil {
				return err
			}
			charged = fi.Size()
		}
		if d.IsDir() {
			return nil
		}
		key := r.keyOf(base, p)
		h, err := r.hash(root, p, key, via, charged)
		if err != nil {
			return err
		}
		if err := r.put(key, "git "+h); err != nil {
			return err
		}
		if follow && d.Type()&fs.ModeSymlink != 0 {
			return r.followLink(base, p)
		}
		return nil
	})
}

// followLink reads what the link rel under base leads to, every link on the
// way resolved, without following any link it finds there, held to the
// followed-link budget and naming the link past it. A link leading nowhere
// reads as nothing.
func (r *treeReader) followLink(base, rel string) error {
	target := fsutil.RealExistingPath(filepath.Join(base, filepath.FromSlash(rel)))
	return r.hashAt(filepath.Dir(target), filepath.Base(target), false, linkReach(r.keyOf(base, rel)))
}

// keyOf is the key of rel under base: its path relative to the repository
// when it is inside it, spelled through the repository's path or its real
// one, and in full when not.
func (r *treeReader) keyOf(base, rel string) string {
	full := filepath.Join(base, filepath.FromSlash(rel))
	for _, repo := range []string{r.repo, r.realRepo} {
		if k, err := filepath.Rel(repo, full); err == nil && k != ".." && !strings.HasPrefix(k, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(k)
		}
	}
	return filepath.ToSlash(full)
}

// errPastBound is a file larger than what is left to hash of it.
var errPastBound = errors.New("past what is left to hash")

// contentState is rel's kind, mode, size and content hash, and the bytes
// hashed: a link's target is hashed, never followed. A regular file larger
// than limit is not read: errPastBound is returned with its size. A file is
// hashed no further than the size its Lstat gave, so one growing as it is
// read is not read past the limit; its size or content differs at the next
// reading.
func contentState(root *os.Root, rel string, limit int64) (string, int64, error) {
	fi, err := root.Lstat(rel)
	if errors.Is(err, os.ErrNotExist) {
		return "absent", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	var n int64
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		target, err := root.Readlink(rel)
		if err != nil {
			return "", 0, err
		}
		h.Write([]byte(target))
	case fi.Mode().IsRegular():
		if fi.Size() > limit {
			return "", fi.Size(), errPastBound
		}
		f, err := root.Open(rel)
		if err != nil {
			return "", 0, err
		}
		n, err = io.CopyN(h, f, fi.Size())
		f.Close()
		if err != nil && !errors.Is(err, io.EOF) {
			return "", 0, err
		}
	}
	return fmt.Sprintf("%v %d %s", fi.Mode(), fi.Size(), hex.EncodeToString(h.Sum(nil))), n, nil
}

// changedSince is every path whose state differs between before and after,
// sorted.
func changedSince(before, after treeState) []string {
	var out []string
	for p, s := range after {
		if before[p] != s {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
