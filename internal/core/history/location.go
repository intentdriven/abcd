package history

// The store's location — where a transcript lands, and how two repos stay apart
// inside one store (iss-95).
//
// The DEFAULT is user-level and exists by construction:
//
//	~/.abcd/transcripts/<root-sha>/records/*.md    redacted records
//	~/.abcd/transcripts/<root-sha>/staging/*.raw   raw, awaiting redaction
//
// Two properties are load-bearing.
//
// It is USER-LEVEL, not repo-level, because a transcript is a session artefact
// of the machine, not a file of the checkout: it must survive a clone being
// deleted, must never be a candidate for `git add`, and must be one corpus a
// person can consult across every repo they work in.
//
// It is KEYED on the repo's root-commit SHA — the same immutable key ahoy's
// registry uses — because a single store still has to answer "this repo's
// transcripts". The key is a directory, so `list`, `show`, `staged` and `drain`
// each read exactly one repo's lane and no cross-repo filter is needed; and it
// is the root SHA rather than a path or a name because a checkout moves, gets
// renamed, and is cloned twice on one machine, while the root commit does not
// change under any of that.
//
// It is CREATED here rather than by `abcd ahoy install`. Requiring install to
// have run made capture fail closed on a machine where it had not: the
// SessionEnd hook logged to stderr, exited 0 (a shutdown hook must), and stored
// nothing — a store that appears wired while the corpus never accrues, which is
// exactly the failure the transcript work exists to prevent (iss-95). Creation
// is safe to do here because it needs no authority the caller does not already
// hold: every level is under the caller's own home (or, opted in, their own
// checkout). What the store still refuses is what `ownedDirsReal` was
// protecting — creating or writing THROUGH a planted symlink. ensureRealDir
// walks the chain top-down, creates each level itself (never MkdirAll, which
// would create the whole chain without judging any of it), and re-verifies every
// level as a real directory on EVERY call, so a parent swapped between calls is
// caught before the leaf is opened.
//
// The PER-REPO location is an opt-in pull, declared in the caller's home —
// see localDeclared.

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// userStoreRelPath is the user-level default, relative to the caller's home.
	userStoreRelPath = ".abcd/transcripts"

	// LocalStoreRelPath is the opt-in per-repo pull-in, relative to the repo
	// root. It is under .abcd/.work.local/ deliberately: that tier is gitignored
	// and per-worktree, so a pulled-in transcript is never a commit candidate and
	// never merge-conflicts between concurrent sessions.
	LocalStoreRelPath = ".abcd/.work.local/transcripts"

	// LocalRootsRelPath is the home-scoped declaration that pulls a repo's
	// transcripts into that repo, relative to the caller's home.
	LocalRootsRelPath = ".abcd/local-transcript-roots"

	// LocalRootsDisplay names that file in a diagnostic in tilde form, so a
	// message a user pastes into a shell works and no diagnostic carries the
	// caller's home path (iss-81, fsutil.RedactHome).
	LocalRootsDisplay = "~/" + LocalRootsRelPath

	// maxLocalRootsBytes caps the declaration read. A hand-maintained list of
	// checkout paths is a handful of lines; 64 KiB bounds a planted device or an
	// endless file without ever refusing a real one.
	maxLocalRootsBytes = 64 << 10

	// recordsDirName holds the redacted records; stagingDirName the raw ones.
	recordsDirName = "records"
	stagingDirName = "staging"

	// legacyStoreRelPath is where the store lived when it was a sub-tree of
	// ahoy's registry namespace; legacyRecordsDirName is the leaf under
	// <root-sha> that held the records. ahoy still owns ~/.abcd/history/ for
	// index.json and the per-repo meta.json; only the corpus moves out.
	legacyStoreRelPath    = ".abcd/history"
	legacyRecordsDirName  = "transcripts"
	legacyTombstoneName   = "transcripts.moved"
	legacyTombstoneHeader = "The transcript corpus for this repo lives at:\n"
)

// Resolution is where this repo's transcripts live, and what resolving it had
// to say out loud.
//
// Notes is the out-of-band diagnostic channel, mirroring rules.Resolve: core
// never writes to stdout, so a migration it performed or a declaration it
// declined to honour is returned for the front door to print on stderr. A store
// that quietly moved a corpus, or quietly ignored an opt-in the caller wrote,
// would be indistinguishable from one that did neither.
type Resolution struct {
	Base    string   `json:"base"`
	Records string   `json:"records"`
	Staging string   `json:"staging"`
	Local   bool     `json:"local"`
	Notes   []string `json:"notes"`
}

// Resolve returns this repo's store, creating it when absent, and migrating a
// corpus left at the legacy location into it.
//
// It is the ONE seam every read and write path goes through, so the store can
// never be reached by a path that skipped the creation, the symlink discipline
// or the migration. It is idempotent and cheap: a resolved store is a handful of
// stat calls and, after the first call, a no-op migration.
//
// Front doors call it explicitly as well, in order to PRINT Notes — the verbs
// below discard them, because a verb that printed would be a core package
// speaking to a terminal.
func Resolve(repoRoot, rootSHA string) (Resolution, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return Resolution{}, errors.New(rootSHAErrMsg)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Resolution{}, fmt.Errorf("history: cannot resolve the caller's home directory: %v", err)
	}

	var res Resolution
	local, note := localDeclared(repoRoot)
	if note != "" {
		res.Notes = append(res.Notes, note)
	}

	// The chain is created and checked top-down, so a level that is a symlink is
	// refused BEFORE anything is created beneath it.
	var chain []string
	if local {
		res.Local = true
		res.Base = filepath.Join(repoRoot, filepath.FromSlash(LocalStoreRelPath))
		chain = []string{
			filepath.Join(repoRoot, ".abcd"),
			filepath.Join(repoRoot, ".abcd", ".work.local"),
			res.Base,
		}
	} else {
		res.Base = filepath.Join(home, filepath.FromSlash(userStoreRelPath))
		chain = []string{filepath.Join(home, ".abcd"), res.Base}
	}
	res.Records = filepath.Join(res.Base, rootSHA, recordsDirName)
	res.Staging = filepath.Join(res.Base, rootSHA, stagingDirName)
	chain = append(chain, filepath.Join(res.Base, rootSHA), res.Records)
	for _, d := range chain {
		if err := ensureRealDir(d); err != nil {
			return Resolution{}, err
		}
	}

	if n := migrateLegacy(home, rootSHA, res); n != "" {
		res.Notes = append(res.Notes, n)
	}
	return res, nil
}

// ensureRealDir creates one level of the store chain and proves it is a real
// directory afterwards.
//
// 0o700 because the store is private: records are redacted but still a verbatim
// account of the caller's sessions, and staging holds them unredacted. An
// already-existing level keeps whatever mode it has — this never widens or
// narrows a directory the caller made themselves.
func ensureRealDir(p string) error {
	if err := os.Mkdir(p, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return &StorePathError{Path: p, Msg: "cannot create the store directory: " + err.Error()}
	}
	if !fsutil.IsRealDir(p) {
		return &StorePathError{Path: p, Msg: "not a real directory (a symlink or a non-directory occupies it); refusing"}
	}
	return nil
}

// localDeclared reports whether the caller has pulled this repo's transcripts
// into the repo itself, and — when a declaration exists but was not honoured —
// why, so an opt-in that silently did nothing is never mistaken for one that was
// never written.
//
// The declaration is home-scoped and line-oriented, following the
// ~/.abcd/path-entry and ~/.abcd/trusted-roots idiom rather than inventing one:
// an abcd-owned record under the caller's own home, where an absent or
// unvouched-for file declares nothing. It is a sibling file rather than a
// section of either, for the reason trusted-roots is: each of those records
// exactly one thing and is parsed by its own owner.
//
// It is deliberately NOT a file inside the repo, and deliberately not an
// environment variable. A repo-shipped declaration would let a checkout redirect
// the machine's transcripts into its own working tree, where its own .gitignore
// governs whether they are a commit candidate — the tree asserting where the
// machine's session record is kept. An env var clears the letter of that bar and
// not its spirit: a repo can ship the shell, direnv or task-runner configuration
// that sets it, and the tree asserts it one indirection out. Writing this file
// needs write access to the caller's own home, the authority they already hold
// over everything abcd trusts.
//
// A declaration this process does not own, or one anyone can write, is not the
// caller's word and pulls nothing in.
func localDeclared(repoRoot string) (bool, string) {
	if repoRoot == "" {
		return false, ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false, ""
	}
	path := filepath.Join(home, filepath.FromSlash(LocalRootsRelPath))
	fi, err := os.Lstat(path)
	if err != nil {
		return false, "" // no declaration is the ordinary case, not a diagnostic.
	}
	switch {
	case !fi.Mode().IsRegular():
		return false, ignoredDeclaration("it is not a regular file")
	case fi.Mode().Perm()&0o022 != 0:
		return false, ignoredDeclaration("it is writable by others, so its contents are not necessarily yours")
	}
	if owner, err := fsutil.OwnerUID(path); err != nil || owner != uint32(os.Getuid()) {
		return false, ignoredDeclaration("it is not owned by this session's uid")
	}
	raw, err := fsutil.ReadGuarded(path, maxLocalRootsBytes)
	if err != nil {
		return false, ignoredDeclaration("it could not be read (" + termsafe.Sanitize(err.Error()) + ")")
	}
	fold := fsutil.CaseFoldingFS()
	want := fsutil.FoldPath(repoRoot, fold)
	for _, line := range strings.Split(string(raw), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		if !filepath.IsAbs(entry) {
			continue // a relative entry names a different directory per caller.
		}
		// Both spellings: the entry as written, and symlink-resolved, because a
		// declared path may not be resolved and repoRoot may be either.
		for _, cand := range []string{filepath.Clean(entry), resolveOrClean(entry)} {
			if fsutil.FoldPath(cand, fold) == want || fsutil.FoldPath(cand, fold) == fsutil.FoldPath(resolveOrClean(repoRoot), fold) {
				return true, ""
			}
		}
	}
	return false, ""
}

// ignoredDeclaration renders the one-line reason a present declaration was not
// honoured, naming the file in tilde form so no home path is carried.
func ignoredDeclaration(why string) string {
	return "history: IGNORED " + LocalRootsDisplay + " — " + why + "; transcripts stay in " + "~/" + userStoreRelPath
}

// resolveOrClean is EvalSymlinks with a lexical fallback, so a declared path
// that does not currently exist still compares.
func resolveOrClean(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// migrateLegacy moves a corpus stored under the legacy location into the
// resolved store, and returns the one-line note saying so (empty when there was
// nothing to move).
//
// Move, not read-both and not refuse. Read-both would leave two stores that
// diverge from the first capture onward, and every read path would have to merge
// them forever. Refusing would re-open iss-95 from the other end: a machine that
// HAD installed would stop capturing until someone ran a remedy, which is the
// silent-corpus failure wearing a louder hat.
//
// It moves file by file rather than renaming the directory: that is idempotent
// under a peer doing the same thing concurrently, needs no atomicity cliff, and
// works when the destination is on another filesystem (the per-repo opt-in can
// be). A file it could not move is LEFT where it is and counted, and the
// tombstone — which is what stops the next resolve looking again — is written
// only when nothing was left behind. Nothing is ever deleted except a source
// whose bytes are already at the destination.
func migrateLegacy(home, rootSHA string, dst Resolution) string {
	legacyRepo := filepath.Join(home, filepath.FromSlash(legacyStoreRelPath), rootSHA)
	if _, err := os.Lstat(filepath.Join(legacyRepo, legacyTombstoneName)); err == nil {
		return "" // already migrated; the tombstone is the receipt.
	}
	legacyRecords := filepath.Join(legacyRepo, legacyRecordsDirName)
	legacyStaging := filepath.Join(legacyRepo, stagingDirName)
	if !fsutil.IsRealDir(legacyRecords) && !fsutil.IsRealDir(legacyStaging) {
		return "" // this repo never had a store at the legacy location.
	}

	// Each legacy leaf is judged before it is read, on the same bar the store's
	// own levels are held to: a symlink at the old path points somewhere the
	// caller did not put a corpus, and moving its contents INTO the store would
	// import files nothing here wrote under names the store then trusts.
	var movedRecords, leftRecords, movedStaged, leftStaged int
	complete := true
	if fsutil.IsRealDir(legacyRecords) {
		movedRecords, leftRecords = moveAll(legacyRecords, dst.Records, ".md")
	}
	if fsutil.IsRealDir(legacyStaging) {
		if err := ensureRealDir(dst.Staging); err == nil {
			movedStaged, leftStaged = moveAll(legacyStaging, dst.Staging, stagedSuffix)
		} else {
			// The staging leaf could not be opened, so how much is still at the
			// old path is unknown. Unknown is not "nothing": the tombstone is
			// withheld and the next resolve looks again.
			complete = false
		}
	}
	left := leftRecords + leftStaged
	complete = complete && left == 0

	if complete {
		// The tombstone is the loud half of the move: a person who goes looking at
		// the old path finds a pointer, not an unexplained absence.
		body := legacyTombstoneHeader + fsutil.RedactHome(dst.Records) + "\n" +
			"Moved " + time.Now().UTC().Format(time.RFC3339) + " by abcd (iss-95).\n"
		_ = fsutil.WriteFileAtomic(filepath.Join(legacyRepo, legacyTombstoneName), []byte(body), 0o600)
	}
	if movedRecords == 0 && movedStaged == 0 && complete {
		return "" // the legacy dirs existed but were empty: nothing worth saying.
	}
	note := fmt.Sprintf("history: moved %d transcript(s) and %d staged file(s) out of %s into %s",
		movedRecords, movedStaged, fsutil.RedactHome(legacyRepo), fsutil.RedactHome(dst.Base))
	if !complete {
		note += fmt.Sprintf("; %d file(s) COULD NOT be moved and are still at the old path", left)
	}
	return note
}

// moveAll moves every regular file with the given suffix from src to dst,
// returning how many moved and how many were left behind. A non-regular entry
// (a planted symlink or FIFO) is never moved into the store — it is left and
// counted, so the tombstone is withheld and the note says so.
func moveAll(src, dst, suffix string) (moved, left int) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		from := filepath.Join(src, e.Name())
		if fi, err := os.Lstat(from); err != nil || !fi.Mode().IsRegular() {
			left++
			continue
		}
		if err := moveFile(from, filepath.Join(dst, e.Name())); err != nil {
			left++
			continue
		}
		moved++
	}
	return moved, left
}

// moveFile moves one file, tolerating both a peer that already moved it and a
// destination on another filesystem.
//
// A destination that already exists is only accepted when it holds the SAME
// bytes — the record filename carries a nanosecond stamp and the session id, so
// an identical name is the same record a peer already migrated, and removing the
// source is finishing that move. Different bytes under one name is not something
// this may resolve by guessing, so the source stays where it is.
func moveFile(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		same, err := sameBytes(from, to)
		if err != nil || !same {
			return errors.New("destination exists with different content")
		}
		return os.Remove(from)
	}
	if err := os.Rename(from, to); err == nil {
		return nil
	}
	// Cross-device, or a rename the filesystem refused: copy then remove, and
	// never remove the source until the copy is on disk.
	data, err := fsutil.ReadGuarded(from, maxTranscriptBytes)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(to, data, 0o600); err != nil {
		return err
	}
	return os.Remove(from)
}

// sameBytes reports whether two files hold identical content.
func sameBytes(a, b string) (bool, error) {
	sa, err := hashFile(a)
	if err != nil {
		return false, err
	}
	sb, err := hashFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(sa, sb), nil
}

func hashFile(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(f, maxTranscriptBytes)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
