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
// Seven places are read:
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
//     path, the handover among them; its push receipts are read as below.
//   - git's own directory, which no status lists: every entry of hooks/ and
//     info/ but info/refs, which no local git command reads (serverInfo),
//     the configuration (config, and config.worktree), HEAD, the
//     branch HEAD names, each submodule's git directory under modules/ (read
//     as the common directory is, its refs included: git status runs status
//     inside a populated submodule), and a linked worktree's .git file, by
//     mode, size and content hash. A hook runs on the person's next git
//     command, the configuration can name a hooks directory, an alias or a
//     credential helper, and HEAD and the branch it names decide what the
//     next commit or push carries. Of the checkout's refs only that branch
//     is read by the value it holds (readOwnBranch), followed through every
//     symbolic ref it leads to, and the symbolic refs by what they name
//     (below): every worktree shares the common directory's refs, and a
//     peer's commit in its own worktree is not the interview's concern
//     (iss-2611012144422634). In a linked
//     worktree these live in the repository's common directory and are read
//     there, besides the worktree's own HEAD and config.worktree; the main
//     worktree's HEAD there is then a sibling's, and is not read. The
//     checkout's own entry under the common directory's worktrees/ is read
//     whole: the entry itself, and its HEAD, commondir, gitdir,
//     config.worktree and locked (worktreeFiles). Of a sibling worktree's
//     entry only what could make git run something there is read: its
//     commondir, gitdir and config.worktree (siblingFiles), which place its
//     configuration and hooks or carry its own configuration
//     (iss-2610040739124513), compared while the entry stands in both
//     readings. Its HEAD, what a rebase or a merge writes there, and the
//     entry made or removed are a peer's own work in its own worktree, which
//     is not the interview's concern, except an entry made carrying what
//     `git worktree add` writes there only where per-worktree configuration
//     is on, or never (carriesState): its own configuration, which it copies
//     from the worktree it runs in when extensions.worktreeConfig is set, a
//     commondir leading to another common directory, or no commondir at all,
//     so an entry removed in one turn and made again in the next is held as
//     one changed. git follows a link there, so a link (a
//     dotfiles-managed hooks directory, a hook or a configuration file) is
//     read where it leads too.
//   - The directory each core.hooksPath value names, in any scope the
//     person's git reads, when it is outside the working tree (inside, the
//     listing above already covers it), read as git's own directory is. The
//     value is resolved through every link first, so a hooks directory
//     reached through a link (a dotfiles-managed one, or an in-tree link to a
//     directory elsewhere) is read where git runs its hooks from, held to
//     the budget maxFollowedEntries and maxFollowedBytes set.
//   - The push receipts in this checkout's own local tier (receiptsRel), by
//     mode, size and content hash: the pre-push gate lets a commit through on
//     a receipt there, so one written while the role runs would let a push
//     skip the preflight. A receipts directory that is a link is read where
//     it leads, held to the same budget. A sibling worktree's receipts are
//     not read as files: a peer's preflight mints one in its own worktree at
//     any time, and that is a peer's own work.
//   - The pre-push gate's judgement of this checkout's HEAD commit
//     (readPushGate): whether .githooks/pre-push would let a push of it
//     through with no new preflight, by the gate's own rule, a receipt for
//     it in the local tier of any worktree git lists or a remote-tracking
//     ref outside every receipt-less prefix holding it. The judgement is
//     watched rather than the files it rests on, so a peer's fetch, commit
//     or preflight, which judge the peer's own commits, leave it as it was,
//     and it stops the interview only when it goes from shut to open,
//     naming what opened it: a role that plants a receipt in a sibling's
//     tier, registers a worktree holding one, or plants a ref at HEAD. It is
//     judged as the hook judges it: history read as stored, the worktrees as
//     the hook's line-by-line read of git's listing names them, a worktree
//     whose path holds a newline stopping the interview, and a receipt
//     counted by the hook's own checks (hookOwnReceipt).
//   - Every symbolic ref under the common directory's refs/, by its name and
//     the ref it names, read from the files (readSymbolicRefs): git resolves
//     one when the gate asks which remote-tracking refs hold a commit, so one
//     naming a ref outside its remote holds whatever the next commit or
//     stash puts there, whatever the gate reads of HEAD now. One made,
//     changed or removed stops the interview, unless it stays inside its
//     remote, as refs/remotes/<remote>/HEAD does.
//
// git's own directory and common directory are placed by the run's first
// reading, taken before any role ran, and every later reading reads those
// same directories. Before it runs git, a later reading reads the files that
// place them, a linked worktree's .git file and the git directory's
// commondir, and those that place each submodule's git directory, its
// commondir under modules/ and its gitlink's .git pointer (placingFiles): one
// changed since is named as a change, and neither git nor the reading follows
// where it leads. It then sizes the files git reads whole on
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
	"bufio"
	"context"
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
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

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
// directory core.hooksPath names outside the tree, and this checkout's push
// receipts. Every entry read there (a directory included) and every byte of
// the files hashed there is counted. A reading past either is refused, naming
// the link, the core.hooksPath value or the receipts directory, before the
// entry past it is read, rather than walking toward maxWatched: a role can
// plant a link to a large tree in one write where the guard follows links
// (this checkout's receipts directory among them), or point core.hooksPath
// at one, and hashing that whole would hold the turn for minutes before the
// guard refused. What the guard legitimately reads there is small: git's
// sample hooks are 14 files and 26 KiB, a checkout keeps at most 50 push
// receipts of 76 bytes each (scripts/preflight-receipt.sh), and a
// dotfiles-managed hooks directory holds tens of scripts. The budget holds a
// compiled hook binary, and reading all of it takes tens of milliseconds. It
// is shared by everything one reading reads there, so planting many links or
// hooks directories does not multiply it.
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
	// said is what moved of the checkout's own branch, in plain words
	// (branchSaid), leading the refusal.
	said []string
}

func (e *UnexpectedChangesError) Error() string {
	lead := ""
	if len(e.said) > 0 {
		lead = strings.Join(e.said, "; ") + ". "
	}
	return lead + fmt.Sprintf("%d path(s) changed while the %s ran that this interview does not let it change, so the interview stopped: %s; "+
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
	// sibling is the key of the sibling worktree's entry being read, when
	// one is: put marks each state it records with it (siblingState).
	sibling string
}

// gitPin is where a run's first reading found git's own directory and
// common directory, and the state of each file that places them or a
// submodule's git directory (placingFiles). Every later reading
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
// this checkout's push receipts.
type reach struct {
	// what names it in a refusal, and fix says how to clear it.
	what, fix string
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
	r := &treeReader{repo: repo, realRepo: fsutil.RealExistingPath(repo), skip: []string{history.LocalStoreRelPath, receiptsRel}, st: treeState{}, pin: pin}
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
		// A submodule's git directory made since was not pinned: its
		// commondir is held to the state it had then, absent.
		mods, err := moduleGitDirs(r.pin.common)
		if err != nil {
			return err
		}
		for _, m := range mods {
			p := filepath.Join(r.pin.common, filepath.FromSlash(m), "commondir")
			key := r.keyOf(filepath.Dir(p), filepath.Base(p))
			if _, pinned := r.pin.pointers[key]; !pinned && pointerState(p) != "absent" {
				moved = append(moved, key)
			}
		}
		if len(moved) > 0 {
			slices.Sort(moved)
			return &redirectedError{Paths: moved}
		}
		r.gitDir, r.common = r.pin.gitDir, r.pin.common
		if err := r.sizeGitReads(); err != nil {
			return err
		}
		// A gitlink added to the index since was not pinned: its .git
		// pointer is held to the state it had then, absent, or git status
		// would place the submodule's git directory where the pointer leads.
		// What git reads whole is sized above and the files placing git's
		// own directory read unchanged, so listing the index reads the one
		// the pin placed (iss-2610050504094430). The listing names every
		// gitlink, with no cap, so this check's work grows with the index.
		links, err := gitutil.Gitlinks(r.repo)
		if err != nil {
			return err
		}
		for _, l := range links {
			p := filepath.Join(r.pin.top, filepath.FromSlash(l), ".git")
			key := r.keyOf(filepath.Dir(p), filepath.Base(p))
			if _, pinned := r.pin.pointers[key]; !pinned && pointerState(p) != "absent" {
				moved = append(moved, key)
			}
		}
		if len(moved) > 0 {
			slices.Sort(moved)
			return &redirectedError{Paths: moved}
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
		placing, err := placingFiles(r.repo, gitDir, common, top)
		if err != nil {
			return err
		}
		for _, p := range placing {
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

// placingFiles are the files that place a git directory git reads on every
// reading: the checkout's .git and its git directory's commondir, each
// submodule git directory's commondir under the common directory's modules/,
// and each gitlink's .git pointer in the working tree, which git status reads
// to place the submodule's git directory as it checks the submodule's HEAD.
// A submodule's own gitlinks are not listed: git status does not run inside
// a submodule (gitutil.Status's --ignore-submodules=dirty). The index's
// listing (gitutil.Gitlinks) names every gitlink, with no cap, so the files
// placed, and the work of reading them, grow with the index.
func placingFiles(repo, gitDir, common, top string) ([]string, error) {
	out := []string{filepath.Join(repo, ".git"), filepath.Join(gitDir, "commondir")}
	mods, err := moduleGitDirs(common)
	if err != nil {
		return nil, err
	}
	for _, m := range mods {
		out = append(out, filepath.Join(common, filepath.FromSlash(m), "commondir"))
	}
	links, err := gitutil.Gitlinks(repo)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		out = append(out, filepath.Join(top, filepath.FromSlash(l), ".git"))
	}
	return out, nil
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
				// A pipe or a device where git reads a file whole would hold
				// git's read, and the turn with it, until something writes
				// to it: refused before git runs, as a file past the bound is.
				return fmt.Errorf("%s is not a regular file, and git reads it whole on every command, so git is not run; restore it and run the interview again", r.keyOf(at.dir, f))
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
// tree, as git names them, in full. The working tree is asked through
// gitutil.Toplevel, the one place that holds git's toplevel answer to its
// shape, so a core.worktree pointed outside the repository is refused there.
func (r *treeReader) namedGitDirs() (gitDir, common, top string, err error) {
	out, err := gitutil.Run(r.repo, "rev-parse", "--git-dir", "--git-common-dir")
	if err != nil {
		return "", "", "", err
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || slices.Contains(lines, "") {
		return "", "", "", fmt.Errorf("git named no git directory and common directory (%q)", out)
	}
	top, err = gitutil.Toplevel(r.repo)
	if errors.Is(err, gitutil.ErrToplevelShape) {
		// Name what moved the working tree, so the person can see it: git's
		// core.worktree is the setting that places it elsewhere.
		named := "a working tree that does not contain " + r.repo
		if wt, cerr := gitutil.Run(r.repo, "config", "--get", "core.worktree"); cerr == nil && wt != "" && !strings.ContainsAny(wt, "\r\n") {
			named = "the working tree " + wt + " (core.worktree)"
		}
		return "", "", "", fmt.Errorf("git names %s, not the working tree of %s; the reading does not follow it: restore where git reads its working tree (core.worktree) and run the interview again", named, r.repo)
	}
	if err != nil {
		return "", "", "", err
	}
	return r.abs(lines[0]), r.abs(lines[1]), r.abs(top), nil
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
	if r.sibling != "" {
		state = siblingState + r.sibling + "\x00" + state
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
	r.followedEntries++
	if fi.Mode().IsRegular() {
		r.followedBytes += fi.Size()
	}
	if r.followedEntries > maxFollowedEntries || r.followedBytes > maxFollowedBytes {
		return fmt.Errorf("%s leads past what one reading follows links to (%d entries and %d MiB in all), "+
			"more than git's own directory, a hooks directory or the push receipts hold, so it is not read; "+
			"%s, and run the interview again", via.what, maxFollowedEntries, maxFollowedBytes>>20, via.fix)
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
	for _, rel := range commonEntries {
		if rel == "HEAD" && gitDir != common {
			// A linked worktree's HEAD is its own, read below; the common
			// directory's is the main worktree's, a sibling's.
			continue
		}
		if err := r.hashIn(common, rel); err != nil {
			return err
		}
	}
	if err := r.readOwnBranch(); err != nil {
		return err
	}
	if err := r.readSymbolicRefs(); err != nil {
		return err
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
	if err := r.readReceipts(); err != nil {
		return err
	}
	return r.readPushGate()
}

// worktreeFiles are the files of the checkout's own entry under the common
// directory's worktrees/ that git reads to place the worktree and run there:
// its HEAD, the common directory it uses (and so the configuration, the
// hooks and the refs), the working tree it belongs to (and so the place git
// lists it), its own configuration, and its lock.
var worktreeFiles = []string{"HEAD", "commondir", "gitdir", "config.worktree", "locked"}

// siblingFiles are the files of a sibling worktree's entry read: those that
// place the configuration and hooks git runs there, or carry its own
// configuration, so a role repointing one at a fake common directory whose
// configuration carries an alias, or writing it one, is stopped
// (iss-2610040739124513). Its HEAD and lock are a peer's own work.
var siblingFiles = []string{"commondir", "gitdir", "config.worktree"}

// siblingState leads the state of what a sibling worktree's entry holds:
// the entry's key, a NUL, then the state, so changedSince can leave out what
// came or went with the entry itself.
const siblingState = "sibling "

// carriesState leads, after the NUL, the state of what a sibling worktree's
// entry holds that a plain `git worktree add` does not write and that could make git
// run something there: its own configuration (config.worktree), a commondir
// leading to another common directory than this checkout's, or, on the
// entry itself, no commondir at all, which makes git read the entry as its
// own common directory. An entry made since the other reading that carries
// one is named all the same (changedSince).
const carriesState = "carries "

// siblingOf is the entry key a sibling state names, and the state after it.
func siblingOf(state string) (entry, rest string, ok bool) {
	after, ok := strings.CutPrefix(state, siblingState)
	if !ok {
		return "", "", false
	}
	return strings.Cut(after, "\x00")
}

// carrying is a sibling state marked as carrying what could make git run
// something in that worktree (carriesState); any other state is returned
// as it stands.
func carrying(state string) string {
	entry, rest, ok := siblingOf(state)
	if !ok || strings.HasPrefix(rest, carriesState) {
		return state
	}
	return siblingState + entry + "\x00" + carriesState + rest
}

// readWorktrees records every entry of common's worktrees/ by its kind and
// the files it holds: the checkout's own entry whole (worktreeFiles), and a
// sibling's siblingFiles, marked as the sibling's (siblingState), each by
// mode, size and content hash. A link, there or as worktrees/ itself, is
// recorded by its target's text and read where it leads, as git reads it.
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
	fold := fsutil.CaseFoldingFS()
	realGitDir := fsutil.RealExistingPath(r.gitDir)
	defer func() { r.sibling = "" }()
	for _, e := range entries {
		at, entryVia := filepath.Join(dir, e.Name()), via
		files := worktreeFiles
		r.sibling = ""
		if real := fsutil.RealExistingPath(at); !fsutil.PathWithin(real, realGitDir, fold) || !fsutil.PathWithin(realGitDir, real, fold) {
			files, r.sibling = siblingFiles, r.keyOf(dir, e.Name())
		}
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
		for _, f := range files {
			if err := r.hashAt(at, f, true, entryVia); err != nil {
				return err
			}
		}
		if r.sibling != "" {
			r.markCarried(at)
		}
	}
	return nil
}

// markCarried marks what the sibling worktree entry at (its real path) holds
// that a plain `git worktree add` does not write and that could make git run something
// there (carriesState): its config.worktree, a commondir leading to another
// common directory than this checkout's, or, on the entry itself, none at
// all. The commondir is read as git reads it, relative to the entry's real
// path, within maxPointerBytes; one past that, or not a file, is carried.
func (r *treeReader) markCarried(at string) {
	mark := func(key string) {
		if st, ok := r.st[key]; ok {
			r.st[key] = carrying(st)
		}
	}
	mark(r.keyOf(at, "config.worktree"))
	p := filepath.Join(at, "commondir")
	fi, err := os.Stat(p)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		mark(r.sibling)
		return
	case err != nil || !fi.Mode().IsRegular() || fi.Size() > maxPointerBytes:
		mark(r.keyOf(at, "commondir"))
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		mark(r.keyOf(at, "commondir"))
		return
	}
	named := strings.TrimRight(string(b), "\r\n")
	if !filepath.IsAbs(named) {
		named = filepath.Join(at, named)
	}
	fold := fsutil.CaseFoldingFS()
	leads, common := fsutil.RealExistingPath(filepath.Clean(named)), fsutil.RealExistingPath(r.common)
	if !fsutil.PathWithin(leads, common, fold) || !fsutil.PathWithin(common, leads, fold) {
		mark(r.keyOf(at, "commondir"))
	}
}

// receiptsRel is where a checkout's push receipts stand, under each worktree
// (scripts/preflight-receipt.sh). The walk of what git ignores leaves them
// to readReceipts.
const receiptsRel = ".abcd/.work.local/preflight-receipts"

// readReceipts records the push receipts in this checkout's own local tier by
// mode, size and content hash: the pre-push gate lets a commit through on a
// receipt there, so one written during a turn would pass a push the gates
// never read. A receipts directory that is a link is read where it leads.
// Everything is held to the budget, since the directory may lead anywhere on
// the machine. A sibling worktree's receipts are its own work and are not
// read as files; the gate's judgement (readPushGate) counts the one that
// names this checkout's HEAD.
func (r *treeReader) readReceipts() error {
	dir := filepath.Join(r.repo, filepath.FromSlash(receiptsRel))
	via := &reach{
		what: "the push receipts directory " + filepath.ToSlash(dir),
		fix:  "remove what is not a push receipt from it, or the link it is if you did not make it",
	}
	return r.hashAt(filepath.Dir(dir), filepath.Base(dir), true, via)
}

// pushGateKey keys the pre-push gate's judgement of this checkout's HEAD in a
// reading (readPushGate). It names no path: changedSince names, in its
// place, what opened the gate.
const pushGateKey = "\x00pre-push gate"

// gateShut and gateOpen lead the pre-push gate's judgement: shut, or open,
// then HEAD's commit, what opened it (a receipt's key or a remote-tracking
// ref's name) and how, each after a NUL.
const (
	gateShut = "shut"
	gateOpen = "open"
)

// gateBudget bounds the git calls one reading makes to judge the gate, all of
// them together: a git that does not answer within it refuses the reading.
const gateBudget = time.Minute

// maxRemoteListing bounds the remote-tracking refs holding HEAD that git
// lists; a listing past it refuses the reading rather than read part of it.
const maxRemoteListing = 4 << 20

// receiptlessPrefixes are the receipt-less prefixes of .githooks/pre-push's
// receiptless_prefixes table, the rows it declares in its own text. Its
// receiptless_tracking treats a remote-tracking ref under any of them, or
// holding the segment abcd-wip anywhere, as one that vouches for nothing,
// matched by segment after the remote's name and in any case; every row the
// hook adds for a host connector's prefix ends in that segment
// (scripts/cloud/connector.sh's WIP_PREFIX_RE), so the segment covers them.
// TestTheReceiptlessPrefixesMirrorThePrePushHook holds the two to one list.
var receiptlessPrefixes = []string{"abcd-reserved/", "abcd-status/"}

// wipSegment is the segment receiptless_tracking treats as receipt-less
// wherever it stands.
const wipSegment = "abcd-wip"

// receiptlessTracking reports whether the remote-tracking ref tracked, given
// without its refs/remotes/ part, is under a receipt-less prefix, by
// receiptless_tracking's rule: a segment match after the remote's name, in
// any case. Only ASCII letters are lowered, as tr lowers them in the C
// locale: lowering more would read as receipt-less a ref the hook may count.
func receiptlessTracking(tracked string) bool {
	t := "/" + asciiLower(tracked) + "/"
	if strings.Contains(t, "/"+wipSegment+"/") {
		return true
	}
	for _, p := range receiptlessPrefixes {
		if p != "" && strings.Contains(t, "/"+asciiLower(p)) {
			return true
		}
	}
	return false
}

// readPushGate records whether the pre-push gate (.githooks/pre-push) would
// let a push of this checkout's HEAD commit through with no new preflight,
// by the gate's own rule: a remote-tracking ref outside every receipt-less
// prefix holds the commit (on_a_remote), or a receipt named by it stands in
// the local tier of a worktree git lists, a regular file once links are
// followed, in a tier git tracks nothing in (scripts/preflight-receipt.sh
// check). A role that opens the gate during a turn would let the person's
// next plain push skip the preflight, whether it plants a receipt in a
// sibling's tier, registers a worktree holding one, or plants a ref; a peer's
// fetch, commit or preflight in its own worktree judges its own commits and
// leaves this one's judgement as it was. Every git call is bounded by
// gateBudget and by its output, and each worktree entry's files git reads to
// list it are sized first (sizeWorktreeEntries).
func (r *treeReader) readPushGate() error {
	ctx, cancel := context.WithTimeout(context.Background(), gateBudget)
	defer cancel()
	st, err := r.pushGate(ctx)
	if err != nil {
		return fmt.Errorf("whether the pre-push gate would let a push of HEAD through cannot be read: %w", err)
	}
	return r.put(pushGateKey, st)
}

// pushGate is readPushGate's judgement, as a state (gateShut, gateOpen).
// Every git call reads history as stored (gitutil.RunCappedBytesAsStoredContext),
// as the hook's own reads do (it exports GIT_NO_REPLACE_OBJECTS, no graft file
// and no commit-graph): a refs/replace/* ref standing a parentless commit in
// for one a planted ref names would otherwise hide that the ref holds HEAD.
func (r *treeReader) pushGate(ctx context.Context) (string, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, r.repo, "", maxPointerBytes, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	unborn := false
	if err != nil {
		if ctx.Err() != nil || gitutil.ExitCode(err) != 1 {
			return "", err
		}
		// git exits 1, saying nothing, when HEAD names no commit.
		unborn = true
	}
	head := strings.TrimSpace(string(out))
	if unborn || head == "" {
		// No commit: nothing a push could carry.
		return gateShut, nil
	}
	if strings.Trim(head, "0123456789abcdef") != "" || (len(head) != 40 && len(head) != 64) {
		return "", fmt.Errorf("git named HEAD's commit %s, not an object name", short(head))
	}
	open := func(where, how string) string {
		return gateOpen + "\x00" + head + "\x00" + where + "\x00" + how
	}
	refs, err := gitutil.RunCappedBytesAsStoredContext(ctx, r.repo, "", maxRemoteListing, "for-each-ref", "--format=%(refname)", "--contains", head, "refs/remotes")
	if err != nil {
		return "", err
	}
	for _, ref := range strings.Split(string(refs), "\n") {
		tracked, ok := strings.CutPrefix(ref, "refs/remotes/")
		if !ok || receiptlessTracking(tracked) {
			continue
		}
		return open(ref, "the remote-tracking ref "+ref+" holds it, and the gate takes a commit a remote-tracking ref holds as already on a remote"), nil
	}
	if err := r.sizeWorktreeEntries(); err != nil {
		return "", err
	}
	wts, err := r.hookWorktrees(ctx)
	if err != nil {
		return "", err
	}
	rel := receiptsRel + "/" + head
	// The hook looks in the pushing checkout's own tree first, then in every
	// worktree its listing names (check).
	for _, wt := range slices.Concat([]string{r.repo}, wts) {
		ok, err := hookOwnReceipt(ctx, wt, rel)
		if err != nil {
			return "", err
		}
		if ok {
			where := r.keyOf(wt, rel)
			return open(where, "a push receipt for it stands at "+where), nil
		}
	}
	return gateShut, nil
}

// maxTrackedListing bounds what git lists of one worktree's tracked paths
// when a receipt for HEAD stands there; a listing past it refuses the reading
// rather than judge part of it.
const maxTrackedListing = 64 << 20

// hookWorktrees is every worktree path as scripts/preflight-receipt.sh check
// reads them: the newline form of `git worktree list --porcelain`, every line
// that starts "worktree ", the rest of it verbatim, after the shell's command
// substitution has dropped any NUL and the trailing newlines. A worktree
// whose path holds a newline is refused first, by the NUL-separated listing,
// which carries such a path whole: the hook would read it as more than one
// worktree, and one of them could hold a receipt git lists nowhere. A git
// with no NUL-separated listing (before 2.36) prints the newline form alone,
// which is then what both read.
func (r *treeReader) hookWorktrees(ctx context.Context) ([]string, error) {
	z, err := gitutil.RunCappedBytesAsStoredContext(ctx, r.repo, "", gitutil.MaxWorktreeListing, "worktree", "list", "--porcelain", "-z")
	switch {
	case err == nil:
		for _, wt := range gitutil.ParseWorktreeList(string(z), "\x00") {
			if strings.Contains(wt.Path, "\n") {
				return nil, fmt.Errorf("a worktree git lists has a newline in its path (%s), and the pre-push gate reads git's worktree listing a line at a time, so it would take that path for more than one worktree; "+
					"the interview cannot tell what the gate would let through: move or remove that worktree (git worktree move, git worktree remove) and run the interview again", strconv.Quote(wt.Path))
			}
		}
	case ctx.Err() != nil:
		return nil, err
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, r.repo, "", gitutil.MaxWorktreeListing, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var wts []string
	for _, line := range strings.Split(strings.TrimRight(strings.ReplaceAll(string(out), "\x00", ""), "\n"), "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			wts = append(wts, p)
		}
	}
	return wts, nil
}

// trackedTier matches a line of `git ls-files` that the hook's own_receipt
// refuses a receipt on, once its ASCII letters are lowered (grep -i): .abcd
// itself, or anything at or under .abcd/.work.local, quoted or not.
var trackedTier = regexp.MustCompile(`^"?\.abcd(/\.work\.local(/|$)|$)`)

// hookOwnReceipt reports whether the worktree at dir holds a receipt at rel
// that scripts/preflight-receipt.sh's own_receipt counts, by its checks in
// its order: a regular file once links are followed, in a worktree git lists
// the tracked paths of, none of which is .abcd or lies in its local tier by
// any letter case of the name, and none of whose first two segments opens
// the same directories as .abcd and .abcd/.work.local. Paths are joined as
// the shell joins them, by a slash, never cleaned. git failing to list the
// paths counts the receipt for nothing, as it does there; git not answering
// (the deadline, the cap) refuses the reading.
func hookOwnReceipt(ctx context.Context, dir, rel string) (bool, error) {
	base := dir + "/"
	if fi, err := os.Stat(base + rel); err != nil || !fi.Mode().IsRegular() {
		return false, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, dir, "", maxTrackedListing, "ls-files")
	if err != nil {
		if ctx.Err() == nil && gitutil.ExitCode(err) > 0 {
			return false, nil
		}
		return false, err
	}
	listed := strings.TrimRight(strings.ReplaceAll(string(out), "\x00", ""), "\n")
	if listed == "" {
		return true, nil
	}
	lines := strings.Split(listed, "\n")
	for _, line := range lines {
		if trackedTier.MatchString(asciiLower(line)) {
			return false, nil
		}
	}
	same := func(a, b string) bool {
		fa, err := os.Stat(a)
		if err != nil {
			return false
		}
		fb, err := os.Stat(b)
		return err == nil && os.SameFile(fa, fb)
	}
	seen := map[string]bool{}
	for _, line := range lines {
		segs := strings.SplitN(line, "/", 3)
		if len(segs) < 2 {
			continue
		}
		pair := segs[0] + "/" + segs[1]
		if seen[pair] {
			continue
		}
		seen[pair] = true
		if same(base+segs[0], base+".abcd") && same(base+pair, base+".abcd/.work.local") {
			return false, nil
		}
	}
	return true, nil
}

// asciiLower is s with its ASCII letters lowered, as tr and grep -i fold
// them in the C locale.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// sizeWorktreeEntries refuses, before git lists the worktrees, an entry
// under the common directory's worktrees/ whose gitdir, HEAD, commondir or
// locked is not a regular file or holds more than maxPointerBytes: git reads
// each whole to list the entry, so one a role grows would hold the listing.
// Links are followed, as git follows them.
func (r *treeReader) sizeWorktreeEntries() error {
	dir := filepath.Join(r.common, "worktrees")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		for _, f := range []string{"gitdir", "HEAD", "commondir", "locked"} {
			fi, err := os.Stat(filepath.Join(dir, e.Name(), f))
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				continue
			}
			key := r.keyOf(dir, path.Join(e.Name(), f))
			if err != nil {
				return err
			}
			if !fi.Mode().IsRegular() || fi.Size() > maxPointerBytes {
				return fmt.Errorf("%s is not a file of at most %d KiB, and git reads it whole to list the worktrees, so git is not run; restore it and run the interview again", key, maxPointerBytes>>10)
			}
		}
	}
	return nil
}

// gateOpened reports whether the gate's judgement went from shut in before
// to open in after, with what opened it (a receipt's key or a ref's name)
// and the plain words for it. A judgement missing from before is taken as
// shut; open to open, by whatever, and open to shut are not an opening.
func gateOpened(before, after string) (where, said string, ok bool) {
	if strings.HasPrefix(before, gateOpen+"\x00") {
		return "", "", false
	}
	parts := strings.Split(after, "\x00")
	if len(parts) != 4 || parts[0] != gateOpen {
		return "", "", false
	}
	return parts[2], fmt.Sprintf("the pre-push gate would now let a push of %s through with no preflight run: %s", short(parts[1]), parts[3]), true
}

// gateSaid is the plain words for the gate opening between before and
// after, naming what opened it, or nothing.
func gateSaid(before, after treeState, role string) []string {
	if _, said, ok := gateOpened(before[pushGateKey], after[pushGateKey]); ok {
		return []string{said + ", written while the " + role + " ran"}
	}
	return nil
}

// gitDirEntries are the entries of each submodule's git directory read by
// mode, size and content hash. Of info/ the refs file is left out
// (serverInfo).
var gitDirEntries = []string{"hooks", "info", "config", "config.worktree", "HEAD", "packed-refs", "refs"}

// serverInfo reports whether p under base is the info/refs file of the common
// directory or of a submodule's git directory there. git update-server-info,
// which git gc and the repack a peer's commit can start run, rewrites it for
// clients of a dumb transport, listing every ref; no local git command reads
// it, nor does the pre-push gate, and it runs nothing, so it is not read.
func (r *treeReader) serverInfo(base, p string) bool {
	if base != r.common {
		return false
	}
	dir, ok := strings.CutSuffix(p, "info/refs")
	if !ok {
		return false
	}
	return dir == "" || slices.Contains(r.modules, strings.TrimSuffix(dir, "/")) && strings.HasSuffix(dir, "/")
}

// commonEntries are the entries of the common directory read by mode, size
// and content hash: gitDirEntries but the refs. Every worktree of the
// checkout shares the common directory's refs, so a peer committing in its
// own worktree moves one; the reading holds the checkout's own branch alone
// (readOwnBranch), the one ref the next commit or push there carries
// (iss-2611012144422634).
var commonEntries = slices.DeleteFunc(slices.Clone(gitDirEntries), func(e string) bool { return e == "refs" || e == "packed-refs" })

// branchState leads the state of the checkout's own branch: the ref's name,
// then the value it holds.
const branchState = "branch "

// readOwnBranch records the branch the checkout's HEAD names, keyed by its
// loose ref's path in the common directory, by the value it holds: the loose
// ref's content, or its line in the packed refs when there is no loose ref,
// so a peer packing the refs moves nothing while a commit, a reset or a
// rebase on the branch does. A detached HEAD names no branch: HEAD itself,
// read above, holds the commit. A branch that is itself a symbolic ref is
// followed as git follows it, to maxSymrefDepth, each ref's value joined to
// the next's, so a commit on the ref it leads to moves the state too: the
// next push of the branch carries that ref's commit. A loose ref on the way
// that is a link is refused: git follows it to a file whose content no
// reading holds.
func (r *treeReader) readOwnBranch() error {
	ref, ok := symbolicHead(r.gitDir)
	if !ok {
		return nil
	}
	common, err := os.OpenRoot(r.common)
	if err != nil {
		return err
	}
	defer common.Close()
	own, err := os.OpenRoot(r.gitDir)
	if err != nil {
		return err
	}
	defer own.Close()
	value, next := "", ref
	for depth := 0; ; depth++ {
		root := common
		if perWorktreeRef(next) {
			root = own
		}
		if fi, err := root.Lstat(next); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s, which the branch this checkout is on leads to, is a link, and git reads the ref where it leads, which the reading does not hold; replace it with the ref itself (git update-ref) and run the interview again", r.keyOf(r.common, next))
		}
		v, err := r.refValue(root, next)
		if err != nil {
			return err
		}
		if value != "" {
			value += " -> "
		}
		value += v
		target, ok := symrefTarget(v)
		if !ok || !plainRefName(target) || depth == maxSymrefDepth {
			break
		}
		next = target
	}
	return r.put(r.keyOf(r.common, ref), branchState+ref+" "+value)
}

// maxSymrefDepth is how many symbolic refs git follows from one before it
// gives up (SYMREF_MAXDEPTH), and so the reading.
const maxSymrefDepth = 5

// perWorktreeRef reports whether git keeps ref in each worktree's own git
// directory rather than the common one.
func perWorktreeRef(ref string) bool {
	for _, p := range []string{"refs/worktree/", "refs/bisect/", "refs/rewritten/"} {
		if strings.HasPrefix(ref, p) {
			return true
		}
	}
	return false
}

// symrefTarget is the ref a loose ref's content names when git reads it as a
// symbolic ref: content opening "ref:", the rest with the whitespace around it
// trimmed, as git's parse_loose_ref_contents takes it.
func symrefTarget(content string) (string, bool) {
	rest, ok := strings.CutPrefix(content, "ref:")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

// plainRefName reports whether ref is a ref name under refs/ in the shape git
// writes one: no empty, "." or ".." segment, no segment opening with a dot or
// ending in .lock, and none of the characters git refuses in a ref name, a
// control character among them. A name failing it is never taken as
// staying inside a remote.
func plainRefName(ref string) bool {
	if !strings.HasPrefix(ref, "refs/") || strings.HasSuffix(ref, ".") || strings.Contains(ref, "@{") {
		return false
	}
	for _, c := range []byte(ref) {
		if c < 0x20 || c == 0x7f || strings.IndexByte(" ~^:?*[\\", c) >= 0 {
			return false
		}
	}
	for _, seg := range strings.Split(ref, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") || strings.HasSuffix(seg, ".lock") {
			return false
		}
	}
	return true
}

// symrefState leads the state of a symbolic ref under the common directory's
// refs/ (readSymbolicRefs): symrefInside or symrefOutside, then, after NULs,
// how git reads it (a file opening "ref:", or a link), what it names, and
// its content's hash.
const symrefState = "symref "

const (
	symrefInside  = "inside"
	symrefOutside = "outside"
)

// maxLooseRefs bounds the entries one reading walks under the common
// directory's refs/; a tree past it is refused rather than read in part. git
// packs loose refs on every gc, so a refs/ holding this many has something in
// it worth packing.
var maxLooseRefs = 500_000

// readSymbolicRefs records every symbolic ref under the common directory's
// refs/, by its name and what it names (symrefState): a loose ref file whose
// content opens "ref:", and any link, which git reads as a symbolic ref when
// its text is a ref name and follows otherwise. The pre-push gate asks git
// which remote-tracking refs hold a commit (on_a_remote), and git resolves a
// symbolic ref there at that moment, so one naming a ref outside its remote
// (the checkout's own branch, refs/stash, a branch not yet made) holds
// whatever the person's next commit or stash puts there, a commit no
// preflight read (iss-2611012144422634). git's listings skip a symbolic ref
// whose target does not exist, so the files are read, not listed. The packed
// refs hold no symbolic ref: each line there is an object name and a ref
// name, or a peeled tag's object name. One staying inside its remote, a
// remote-tracking ref naming another under its own directory, as `git remote
// set-head`, a clone and a fetch write refs/remotes/<remote>/HEAD, follows
// what only a fetch moves, and is marked so (symrefInside); changedSince
// leaves out its coming, going and changing. Every other is named. The
// checkout's own branch is read by readOwnBranch, through every symbolic ref
// it leads to. A refs/ that is not a directory, a link under it leading to a
// directory, and a reftable, which keeps the refs in binary tables, are
// refused: what git reads through them cannot be held.
func (r *treeReader) readSymbolicRefs() error {
	root, err := os.OpenRoot(r.common)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Lstat("reftable"); err == nil {
		return fmt.Errorf("%s keeps its refs in a reftable, where the reading cannot tell a symbolic ref made pointing outside its remote, which would let a push skip the preflight; "+
			"the interview runs only in a repository that keeps its refs as files (git refs migrate --ref-format=files)", r.keyOf(r.common, ""))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	fi, err := root.Lstat("refs")
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return err
	case !fi.IsDir():
		return fmt.Errorf("%s is not a directory, and git reads the refs where it leads, which the reading does not hold; make it the directory itself and run the interview again", r.keyOf(r.common, "refs"))
	}
	own, _ := symbolicHead(r.gitDir)
	walked := 0
	return fs.WalkDir(root.FS(), "refs", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if walked++; walked > maxLooseRefs {
			return fmt.Errorf("%s holds more than %d entries, too many to read around each turn; pack them (git pack-refs --all) and run the interview again", r.keyOf(r.common, "refs"), maxLooseRefs)
		}
		if d.IsDir() || p == own {
			return nil
		}
		key := r.keyOf(r.common, p)
		if d.Type()&fs.ModeSymlink != 0 {
			text, err := root.Readlink(p)
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			if fi, err := os.Stat(filepath.Join(r.common, filepath.FromSlash(p))); err == nil && fi.IsDir() {
				return fmt.Errorf("%s is a link to a directory, and git reads the refs where it leads, which the reading does not hold; replace it with the refs themselves and run the interview again", key)
			}
			// git writes no link here unless core.preferSymlinkRefs is set,
			// and reads one by its text or where it leads, so none stays
			// inside its remote.
			return r.put(key, symrefState+symrefOutside+"\x00link\x00"+text+"\x00")
		}
		if !d.Type().IsRegular() {
			return nil
		}
		st, err := r.looseSymref(root, p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// Pruned by a peer's packing between the walk and the read.
			return nil
		case err != nil:
			return err
		case st == "":
			return nil
		}
		return r.put(key, st)
	})
}

// looseSymref is the state of the loose ref file p under root when git reads
// it as a symbolic ref, or "" when it holds anything else. Only its first
// bytes are read unless it opens "ref:", so the walk reads a few bytes of
// each plain ref; a symbolic ref is read whole, within maxPointerBytes, and
// counted against maxHashedBytes.
func (r *treeReader) looseSymref(root *os.Root, p string) (string, error) {
	f, err := root.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	lead := make([]byte, len("ref:"))
	if _, err := io.ReadFull(f, lead); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return "", nil
		}
		return "", err
	}
	if string(lead) != "ref:" {
		return "", nil
	}
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	if fi.Size() > maxPointerBytes {
		return fmt.Sprintf("%s%s\x00file\x00past %d bytes\x00", symrefState, symrefOutside, fi.Size()), nil
	}
	if fi.Size() > maxHashedBytes-r.hashedBytes {
		return "", fmt.Errorf("%s takes one reading past the %d MiB one reading hashes of the tree and git's own directory, so it is not read; restore it and run the interview again", r.keyOf(r.common, p), maxHashedBytes>>20)
	}
	r.hashedBytes += fi.Size()
	rest, err := io.ReadAll(io.LimitReader(f, fi.Size()))
	if err != nil {
		return "", err
	}
	content := append(lead, rest...)
	sum := sha256.Sum256(content)
	target, _ := symrefTarget(strings.TrimRight(string(content), " \t\n\v\f\r"))
	where := symrefOutside
	if symrefInsideRemote(p, target) {
		where = symrefInside
	}
	return symrefState + where + "\x00file\x00" + target + "\x00" + hex.EncodeToString(sum[:]), nil
}

// symrefInsideRemote reports whether the symbolic ref name, naming target,
// stays inside its remote: name is a remote-tracking ref, and target a plain
// ref name under name's own directory there, refs/remotes/<remote>/ for
// refs/remotes/<remote>/HEAD.
func symrefInsideRemote(name, target string) bool {
	dir := path.Dir(name)
	return strings.HasPrefix(dir, "refs/remotes/") && plainRefName(target) && strings.HasPrefix(target, dir+"/")
}

// insideSymref reports whether state is a symbolic ref's that stays inside
// its remote (symrefInside).
func insideSymref(state string) bool {
	return strings.HasPrefix(state, symrefState+symrefInside+"\x00")
}

// symrefSaid is the plain words for each symbolic ref outside its remote made,
// changed or removed between before and after.
func symrefSaid(before, after treeState, role string) []string {
	var out []string
	for _, k := range slices.Sorted(maps.Keys(after)) {
		s := after[k]
		rest, ok := strings.CutPrefix(s, symrefState)
		if !ok || insideSymref(s) || before[k] == s {
			continue
		}
		parts := strings.Split(rest, "\x00")
		names := ""
		if len(parts) > 2 {
			names = parts[2]
		}
		verb := "was made"
		if _, held := before[k]; held {
			verb = "was changed"
		}
		out = append(out, fmt.Sprintf("the symbolic ref %s %s naming %s, outside its remote, while the %s ran: the pre-push gate takes a commit it holds as already on a remote, and git resolves it when the gate asks, so it holds whatever the next commit or stash puts there",
			k, verb, short(names), role))
	}
	for _, k := range slices.Sorted(maps.Keys(before)) {
		if _, held := after[k]; held || !strings.HasPrefix(before[k], symrefState) || insideSymref(before[k]) {
			continue
		}
		out = append(out, fmt.Sprintf("the symbolic ref %s, outside its remote, was removed while the %s ran", k, role))
	}
	return out
}

// symbolicHead is the ref the HEAD in gitDir names, when it names one in the
// shape git writes it.
func symbolicHead(gitDir string) (string, bool) {
	root, err := os.OpenRoot(gitDir)
	if err != nil {
		return "", false
	}
	defer root.Close()
	fi, err := root.Lstat("HEAD")
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxPointerBytes {
		return "", false
	}
	b, err := readAtMost(root, "HEAD", fi.Size())
	if err != nil {
		return "", false
	}
	ref, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "ref: ")
	if !ok || !strings.HasPrefix(ref, "refs/") || path.Clean(ref) != ref || strings.ContainsAny(ref, " \t\r\n\x00\\:?*[~^") || slices.Contains(strings.Split(ref, "/"), "..") {
		return "", false
	}
	return ref, true
}

// readAtMost is rel under root, read no further than n bytes.
func readAtMost(root *os.Root, rel string, n int64) ([]byte, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, n))
}

// refValue is the value ref holds under the common directory root: a loose
// ref's content, or what stands there when it is not a file, or its line in
// the packed refs, which are read too when the loose ref goes between its
// Lstat and its read.
func (r *treeReader) refValue(root *os.Root, ref string) (string, error) {
	fi, err := root.Lstat(ref)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return r.packedValue(root, ref)
	case err != nil:
		return "", err
	case !fi.Mode().IsRegular():
		st, _, err := contentState(root, ref, 0)
		return "not a file: " + st, err
	case fi.Size() > maxPointerBytes:
		return fmt.Sprintf("past %d", fi.Size()), nil
	}
	b, err := readRef(root, ref, fi.Size())
	if errors.Is(err, fs.ErrNotExist) {
		// A peer packing the refs removed the loose ref after it was found
		// and before it was opened. git writes the packed refs before it
		// prunes a loose ref, so they hold the branch now: read once there.
		return r.packedValue(root, ref)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// readRef reads a loose ref (readAtMost); a test stands a peer's packing of
// the refs between its Lstat and its read.
var readRef = readAtMost

// packedValue is ref's object name in the common directory's packed refs,
// "absent" when they hold none. The file is read a line at a time, within
// what is left of maxHashedBytes, which it then counts; a line longer than
// any ref's is refused rather than read.
func (r *treeReader) packedValue(root *os.Root, ref string) (string, error) {
	fi, err := root.Lstat("packed-refs")
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "absent", nil
	case err != nil:
		return "", err
	case !fi.Mode().IsRegular():
		st, _, err := contentState(root, "packed-refs", 0)
		return "packed refs not a file: " + st, err
	}
	key := r.keyOf(r.common, "packed-refs")
	if fi.Size() > maxHashedBytes-r.hashedBytes {
		return "", fmt.Errorf("%s holds %d MiB, which takes one reading past the %d MiB one reading hashes of the tree and git's own directory, so it is not read; restore it and run the interview again",
			key, fi.Size()>>20, maxHashedBytes>>20)
	}
	f, err := root.Open("packed-refs")
	if err != nil {
		return "", err
	}
	defer f.Close()
	r.hashedBytes += fi.Size()
	sc := bufio.NewScanner(io.LimitReader(f, fi.Size()))
	sc.Buffer(make([]byte, 0, 4096), maxPointerBytes+128)
	value := "absent"
	for sc.Scan() {
		name, packed, ok := strings.Cut(sc.Text(), " ")
		if ok && packed == ref && value == "absent" {
			value = name
		}
	}
	if errors.Is(sc.Err(), bufio.ErrTooLong) {
		return "", fmt.Errorf("%s holds a line longer than any ref's, so it is not read; restore it and run the interview again", key)
	}
	return value, sc.Err()
}

// branchOf is the ref and the value a branch state holds.
func branchOf(state string) (ref, value string, ok bool) {
	rest, ok := strings.CutPrefix(state, branchState)
	if !ok {
		return "", "", false
	}
	ref, value, ok = strings.Cut(rest, " ")
	return ref, value, ok
}

// branchSaid is what moved of the checkout's own branch between before and
// after, in plain words: the branch moved, made or deleted, or the checkout
// switched to another.
func branchSaid(before, after treeState, role string) []string {
	own := func(st treeState) (key, ref, value string) {
		for k, s := range st {
			if r, v, ok := branchOf(s); ok {
				return k, r, v
			}
		}
		return "", "", ""
	}
	name := func(ref string) string {
		if ref == "" {
			return "a detached commit"
		}
		return "the branch " + strings.TrimPrefix(ref, "refs/heads/")
	}
	bk, bref, bval := own(before)
	ak, aref, aval := own(after)
	switch {
	case bk != ak:
		return []string{fmt.Sprintf("this checkout was switched from %s to %s while the %s ran", name(bref), name(aref), role)}
	case bk == "" || bval == aval:
		return nil
	case bval == "absent":
		return []string{fmt.Sprintf("%s this checkout is on was made, at %s, while the %s ran", name(bref), short(aval), role)}
	case aval == "absent":
		return []string{fmt.Sprintf("%s this checkout is on was deleted while the %s ran; it was at %s", name(bref), role, short(bval))}
	}
	return []string{fmt.Sprintf("%s this checkout is on moved from %s to %s while the %s ran, as a commit, a reset or a rebase on it does; git log %s shows what it holds now",
		name(bref), short(bval), short(aval), role, strings.TrimPrefix(bref, "refs/heads/"))}
}

// short is an object name cut to twelve characters, anything else quoted
// and cut to 64 bytes.
func short(v string) string {
	if len(v) >= 40 && strings.Trim(v, "0123456789abcdef") == "" {
		return v[:12]
	}
	if len(v) > 64 {
		v = v[:64] + "..."
	}
	return strconv.Quote(v)
}

// moduleEntries are the entries of each submodule's git directory read by
// mode, size and content hash: the common directory's (gitDirEntries), and
// its commondir, which places the directory git reads the submodule's
// configuration, hooks and refs from, so a write there is named.
var moduleEntries = slices.Concat(gitDirEntries, []string{"commondir"})

// readModules records every submodule's git directory sizeGitReads found
// under the common directory's modules/ (moduleEntries): git status runs
// status inside a populated submodule, and the submodule's hooks,
// configuration, HEAD and refs decide what runs there and what its next
// commit or push carries.
func (r *treeReader) readModules() error {
	for _, m := range r.modules {
		for _, rel := range moduleEntries {
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
		if d.IsDir() || r.serverInfo(base, p) {
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
// sorted. Two kinds of path held by one reading alone are left out: the
// checkout's own branch, since HEAD named another between them and HEAD's own
// change names that; and what a sibling worktree's entry holds when the
// other reading does not hold the entry, since a peer made or removed that
// worktree, unless the entry was made since and what it holds carries what
// a plain `git worktree add` does not write (carriesState). The pre-push gate's
// judgement is named only when it opened between them, by what opened it
// (gateOpened). A symbolic ref staying inside its remote, made, changed or
// removed, is left out (readSymbolicRefs).
func changedSince(before, after treeState) []string {
	alone := func(s string, other treeState, made bool) bool {
		if strings.HasPrefix(s, branchState) {
			return true
		}
		entry, rest, ok := siblingOf(s)
		if !ok {
			return false
		}
		if _, held := other[entry]; held {
			return false
		}
		return !made || !strings.HasPrefix(rest, carriesState)
	}
	var out []string
	if where, _, ok := gateOpened(before[pushGateKey], after[pushGateKey]); ok {
		out = append(out, where)
	}
	for p, s := range after {
		if p == pushGateKey {
			continue
		}
		b, ok := before[p]
		if !ok && (alone(s, before, true) || insideSymref(s)) {
			continue
		}
		if b != s && !(insideSymref(b) && insideSymref(s)) {
			out = append(out, p)
		}
	}
	for p, s := range before {
		if p == pushGateKey {
			continue
		}
		if _, ok := after[p]; !ok && !alone(s, after, false) && !insideSymref(s) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return slices.Compact(out)
}
