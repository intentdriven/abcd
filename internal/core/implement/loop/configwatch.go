package loop

// configwatch.go is the run tripwire (iss-2610111600058575). A program the
// repository's own git configuration names (a content filter, a hook, a
// transport or credential helper, a diff or merge driver, a signing program)
// runs on the person's own plain git commands with no abcd involved, so the
// loop's git steps keep running the repository's settings as they are and
// never blank, pin or override them (the abcd-fixes-what-it-adds principle).
// What the loop adds is a tripwire: the first check of a run records a
// fingerprint of every repository-level setting that can name a program, and
// of the hooks directory and info/attributes (and, for each lane worktree,
// every watched setting git reads there and the hooks directory git runs
// there), and every later check, made at the start of the loop's next step,
// compares against it. A change stops the step before it reaches its git,
// naming each changed key or file, until the person has looked and accepts it
// (`abcd implement step --accept-config-change`).
//
// The fingerprint lives in its own file beside the run's state. It holds
// salted hashes, never a value: a credential helper's value may carry a
// secret, so only key names and file paths are ever stored or reported. A key
// name can carry one too (a URL rewrite or a credential context names a URL,
// which may hold a token), so a subsection that can hold a URL is replaced by
// a salted-hash token wherever the key is stored or named.

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/procgroup"
)

// StageConfig is the refusal stage of a step the run tripwire stopped.
const StageConfig = "config"

// ConfigFingerprintFileName is the tripwire's file, beside the run's state.
const ConfigFingerprintFileName = "config-fingerprint.json"

// AcceptConfigFlag is the `implement step` flag that accepts a changed
// configuration once the person has looked at it.
const AcceptConfigFlag = "--accept-config-change"

// configFingerprintVersion is the fingerprint file's schema version.
const configFingerprintVersion = 1

// watchedConfig matches, as `git config --get-regexp` matches a canonical key
// (section and variable lower-cased), every repository-level setting that can
// name a program git starts, or a file that adds more settings.
const watchedConfig = `^(` +
	`filter\..+` +
	`|core\.(hookspath|sshcommand|fsmonitor|pager|editor|worktree|askpass|gitproxy)` +
	`|extensions\.worktreeconfig` +
	`|credential\..+` +
	`|remote\..+\.(uploadpack|receivepack|proxy|vcs|url|pushurl)` +
	`|url\..+\.(insteadof|pushinsteadof)` +
	`|protocol\.(.+\.)?allow` +
	`|diff\..+\.(command|textconv)` +
	`|merge\..+\.driver` +
	`|gpg\.(.+\.)?program` +
	`|gpg\.ssh\.defaultkeycommand` +
	`|include\.path` +
	`|includeif\..+\.path` +
	`)$`

// watchedKeyRE is watchedConfig as a Go expression: git's extended syntax and
// Go's agree on every construct the pattern uses.
var watchedKeyRE = regexp.MustCompile(watchedConfig)

// watchedKey says whether a canonical config key is one the run's fingerprint
// watches (watchedConfig). It is the one predicate for "watched": git applies
// the pattern for the fingerprint's read, and a drive's hold across its lanes
// (remoteIdentityKey) applies this, so the two never hold different lists.
func watchedKey(key string) bool { return watchedKeyRE.MatchString(key) }

// maxHookEntries bounds the entries, at every depth, one read of a hooks
// directory hashes. A tree past it is recorded by its first entries, in walk
// order, and a marker at the directory's own path saying the bound was passed
// and how many entries the tree holds (countEntries); a drive refuses to start
// on one (readRemoteIdentity) and its next lane on one grown past it
// (checkoutHooksUnchanged), and the lane's pin refuses one (provenHooks).
const maxHookEntries = 1024

// maxFingerprintBytes caps a fingerprint file read.
const maxFingerprintBytes = 1 << 20

// urlSections are the sections whose subsection is, or can name, a URL (a
// URL rewrite's base, a credential or http context, an includeIf condition):
// their subsection is never stored or named as written.
var urlSections = map[string]bool{"url": true, "credential": true, "http": true, "includeif": true}

// configFingerprint is the tripwire's file. Keys and Files map a watched key
// name (its URL subsection hidden, hiddenKey), or a watched file's path, to a
// salted hash of what it holds; Worktrees holds each lane worktree's own, by
// its path. The salt is the file's own, drawn when it is first written.
// live, never stored, holds the paths of the lanes still in progress when the
// fingerprint was taken: the lanes the loop still runs git in.
type configFingerprint struct {
	SchemaVersion int                            `json:"schema_version"`
	RecordedAt    time.Time                      `json:"recorded_at"`
	Salt          string                         `json:"salt"`
	Keys          map[string]string              `json:"keys"`
	Worktrees     map[string]worktreeFingerprint `json:"worktrees"`
	Files         map[string]string              `json:"files"`
	live          map[string]bool
}

// worktreeFingerprint is one lane worktree's part of the fingerprint: every
// watched key git reads there, whatever its scope (the repository's shared
// config, a conditional include that applies only there, the worktree's own
// config), and each entry of the hooks directory git runs there when that is
// not the checkout's (a relative core.hooksPath names the lane's own copy).
// Dir is set when a live lane's path is there but is not a real directory (a
// symlink, by its target, or another kind of file): git still runs at that
// path, so what stands there is part of the fingerprint. A real directory
// leaves it empty.
type worktreeFingerprint struct {
	Keys  map[string]string `json:"keys"`
	Hooks map[string]string `json:"hooks"`
	Dir   string            `json:"dir,omitempty"`
}

// laneDir is one lane worktree the fingerprint reads: its path, and whether
// its lane is still in progress (its stage neither done nor discarded).
type laneDir struct {
	Path string
	Live bool
}

// fingerprintRel is the run's fingerprint file, relative to the checkout.
func fingerprintRel(runID string) string { return runRel(runID) + "/" + ConfigFingerprintFileName }

// newSalt draws a fresh salt for a new fingerprint.
func newSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// saltedHash hashes parts, each length-prefixed, under salt.
func saltedHash(salt string, parts ...string) string {
	h := sha256.New()
	writePart(h, salt)
	for _, p := range parts {
		writePart(h, p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writePart(w io.Writer, s string) {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(s)))
	_, _ = w.Write(n[:])
	_, _ = io.WriteString(w, s)
}

// takeFingerprint reads the repository at repoRoot as it stands, under salt:
// every watched key it sets (each key's values, in order, hashed together),
// every entry of its effective hooks directory, and its info/attributes; and,
// for each lane worktree in lanes, every watched key git reads there and the
// entries of the hooks directory git runs there, when that is not the
// checkout's. A lane's path where nothing stands is passed over: git runs
// nothing there. A read at a lane's path that fails is passed over only where
// git, asked in the environment the lane's git runs in, says there is no
// repository there (noRepository); any other failure is a readFailure that
// stops the step (laneReadFailure). A live lane's path that is there but is
// not a real directory is recorded as what stands there (laneDirKind),
// whether or not git reads a config through it. Every read of the checkout
// that fails is a readFailure (checkoutHooks).
func takeFingerprint(repoRoot string, lanes []laneDir, salt string, now time.Time) (configFingerprint, error) {
	fp := configFingerprint{SchemaVersion: configFingerprintVersion, RecordedAt: now, Salt: salt,
		Keys: map[string]string{}, Worktrees: map[string]worktreeFingerprint{}, Files: map[string]string{},
		live: map[string]bool{}}
	hooks, common, err := checkoutHooks(repoRoot, salt, fp.Keys, fp.Files)
	if err != nil {
		return fp, err
	}
	hashFile(fp.Files, salt, filepath.Join(common, "info", "attributes"))
	for _, ld := range lanes {
		lane := ld.Path
		wt := worktreeFingerprint{Keys: map[string]string{}, Hooks: map[string]string{}}
		if ld.Live {
			fp.live[lane] = true
			wt.Dir = laneDirKind(salt, lane)
		}
		if _, err := os.Lstat(lane); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		entries, err := gitutil.RepoConfig(lane, watchedConfig)
		if errors.Is(err, gitutil.ErrConfigCap) {
			return fp, err
		}
		if err != nil {
			if err := laneReadFailure(lane, "reading the watched settings (git config --get-regexp), in "+isolatedEnv.name, err); err != nil {
				return fp, err
			}
			if wt.Dir != "" {
				fp.Worktrees[lane] = wt
			}
			continue
		}
		// The hooks directory git runs in the lane, asked of git there
		// (hooksDirAt): a relative core.hooksPath is the lane's own copy.
		// git reads the lane's config here, so a hooks directory it cannot
		// name, or names in a form abcd does not take, stops the step
		// (readFailure); it is never passed over as unchanged. Where git
		// says there is no repository at the lane's path (noRepository),
		// git runs no hook there, and the lane holds no hooks.
		hashEntries(wt.Keys, salt, entries)
		dir, err := hooksDirAt(lane)
		if err != nil {
			if err := laneReadFailure(lane, "asking git for the hooks directory it runs (git rev-parse --git-path hooks)", err); err != nil {
				return fp, err
			}
			fp.Worktrees[lane] = wt
			continue
		}
		if !samePath(repoRoot, dir, hooks) {
			hashDir(wt.Hooks, salt, dir)
		}
		fp.Worktrees[lane] = wt
	}
	return fp, nil
}

// checkoutHooks is how the fingerprint reads the checkout's hooks: each
// watched key the repository's own settings set is hashed into keys
// (hashEntries), and each entry, at every depth, of the hooks directory git
// runs at the checkout (hooksDirAt: asked of git, in the environment the
// lane's commit runs in, every scope that environment reads) into files
// (hashDir), all under salt. It returns that directory and the common git
// directory; git failing to name the directory is an error. A drive holds the
// same directory across its lanes (readCheckoutHooks in remoteid.go), hashed
// through the same walk. Every read here that fails is a readFailure naming
// it, the cap on the settings read apart (gitutil.ErrConfigCap): the check
// stops, and never takes a read that failed as nothing changed.
func checkoutHooks(repoRoot, salt string, keys, files map[string]string) (hooks, common string, err error) {
	entries, err := gitutil.RepoConfig(repoRoot, watchedConfig)
	if errors.Is(err, gitutil.ErrConfigCap) {
		return "", "", err
	}
	if err != nil {
		return "", "", &readFailure{what: "reading the repository's watched settings (git config --get-regexp), in " + isolatedEnv.name, why: fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))}
	}
	hashEntries(keys, salt, entries)
	if common, err = gitutil.RevParseAbsPath(repoRoot, "--git-common-dir"); err != nil {
		return "", "", &readFailure{what: "resolving the common git directory (git rev-parse --git-common-dir), in " + isolatedEnv.name, why: fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))}
	}
	if hooks, err = hooksDirAt(repoRoot); err != nil {
		return "", "", err
	}
	hashDir(files, salt, hooks)
	return hooks, common, nil
}

// hashEntries hashes each key's values, in the order git reads them, into
// into under the key's stored name (hiddenKey), and returns the last
// core.hooksPath among them (the one git uses).
func hashEntries(into map[string]string, salt string, entries []gitutil.ConfigEntry) string {
	values := map[string][]string{}
	hooksPath := ""
	for _, e := range entries {
		v := e.Scope + "\x00" + e.Value
		if e.NoValue {
			v = e.Scope + "\x00(no value)"
		}
		values[e.Key] = append(values[e.Key], v)
		if e.Key == "core.hookspath" {
			hooksPath = e.Value
		}
	}
	for k, vs := range values {
		into[hiddenKey(salt, k)] = saltedHash(salt, vs...)
	}
	return hooksPath
}

// hiddenKey is the name a canonical key is stored and named by. A key whose
// subsection can hold a URL (a section in urlSections, or a subsection with a
// colon, a slash or an at sign in it) may carry a token in its very name, so
// its subsection is replaced by a placeholder holding a short salted hash of
// it: url.<hidden-0123456789abcdef>.insteadof. Two such keys stay apart, and
// the same key keeps one name under one salt. Any other key is its own name.
func hiddenKey(salt, key string) string {
	section, rest, ok := strings.Cut(key, ".")
	i := strings.LastIndexByte(rest, '.')
	if !ok || i < 0 {
		return key
	}
	sub, variable := rest[:i], rest[i+1:]
	if !urlSections[section] && !strings.ContainsAny(sub, ":/@") {
		return key
	}
	return section + ".<hidden-" + saltedHash(salt, "subsection", sub)[:16] + ">." + variable
}

// laneWorktrees are the run's lane worktrees, other than the checkout: the
// linked worktrees the loop's lane steps run git in. A lane still in progress
// is listed whatever stands at its path, since its steps still run git there;
// a lane that landed or was discarded only while its worktree is a real
// directory on disk. A live lane is skipped as the checkout only when its
// recorded path is the checkout's own: a symlink at the lane's path that
// resolves to the checkout is still the lane's path, and is watched.
func laneWorktrees(repoRoot string, st State) []laneDir {
	var out []laneDir
	for _, l := range st.Lanes {
		if l.Worktree == "" {
			continue
		}
		live := l.Stage != StageDone && l.Stage != StageDiscarded
		if filepath.Clean(l.Worktree) == filepath.Clean(repoRoot) || (!live && samePath(repoRoot, l.Worktree, repoRoot)) {
			continue
		}
		if live || fsutil.IsRealDir(l.Worktree) {
			out = append(out, laneDir{Path: filepath.Clean(l.Worktree), Live: live})
		}
	}
	return out
}

// laneDirKind is what stands at a live lane's path, when that is not a real
// directory: a symlink by its target, anything else by its kind, a path that
// cannot be examined as such. A real directory, or nothing there, is empty.
func laneDirKind(salt, path string) string {
	fi, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ""
	case err != nil:
		return saltedHash(salt, "lane path", "unreadable")
	case fi.Mode()&fs.ModeSymlink != 0:
		target, _ := os.Readlink(path)
		return saltedHash(salt, "lane path", "symlink", target)
	case fi.IsDir():
		return ""
	default:
		return saltedHash(salt, "lane path", "kind", fi.Mode().Type().String())
	}
}

// noRepository reports whether git, asked at dir in the environment the
// lane's git runs in (stageEnv: `git rev-parse --git-dir`, in the C locale),
// says there is no repository there or above it: exit status 128, and its
// standard error beginning with git's own "fatal: not a git repository".
// Those words anywhere else in what git said (a path git repeats can hold
// them) are not that answer. Any other answer is not that, and a read that
// gives none is a readFailure.
func noRepository(dir string) (bool, error) {
	a, err := stageEnv.read(dir, "asking git whether a repository stands at the lane's path (git rev-parse --git-dir)", 4096, "rev-parse", "--git-dir")
	if err != nil {
		return false, err
	}
	return a.code == 128 && strings.HasPrefix(a.stderr, "fatal: not a git repository"), nil
}

// laneReadFailure is the readFailure of a read at a lane's path that failed
// where git does not say there is no repository there (noRepository): the
// step stops at the config stage, naming the read. A read that says so is
// nil, and the lane is passed over, since git runs nothing there.
func laneReadFailure(lane, what string, err error) error {
	none, nerr := noRepository(lane)
	if nerr != nil {
		return nerr
	}
	if none {
		return nil
	}
	var rf *readFailure
	if errors.As(err, &rf) {
		return err
	}
	return &readFailure{what: what + " at a lane's path", why: fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))}
}

// hooksDirAt is the hooks directory git runs in the working tree at dir in
// the environment the lane's add, commit, merge and reset run in
// (hooksDirIn, stageEnv).
func hooksDirAt(dir string) (string, error) { return hooksDirIn(stageEnv, dir) }

// hooksDirIn is the hooks directory git runs in the working tree at dir in
// e, asked of git itself (`git rev-parse --git-path hooks`) there: git
// expands core.hooksPath (a home form, ~user, %(prefix), :(optional)) under
// that environment's home and settings, as it does for the commands that run
// in it, and abcd expands nothing. A relative answer is under dir. git
// failing, not finishing, or answering with other than one path (a value
// holding a newline among them) is a readFailure naming the read: the caller
// stops rather than hash a directory git would not run.
func hooksDirIn(e *gitEnv, dir string) (string, error) { return hooksDirBy(e, dir, boundsNow()) }

// hooksDirBy is hooksDirIn under the reading's bounds b (gitEnv.readBy).
func hooksDirBy(e *gitEnv, dir string, b readBounds) (string, error) {
	what := "asking git for the hooks directory it runs (git rev-parse --git-path hooks)"
	a, err := e.readBy(b, dir, what, 4096, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	if a.code != 0 {
		return "", e.exited(what, a)
	}
	p, ok := onePath(a.out)
	if !ok {
		return "", e.failed(what, "git did not answer with one path (a core.hooksPath holding a newline answers so), so the answer is not taken")
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p), nil
}

// workingTreeAsk runs one read-only git command at dir through
// gitutil.WorkingTreeCommitCommand, in the environment the lane's add and
// commit run in, as env gives it (stageEnv's environ, taken as the reading
// started), bounded by ctx, and returns its output, refused past limit
// bytes. git's output is read to its end before git is waited for
// (procgroup.RunToEOF), and when ctx ends git is killed and waited for at
// once, so a program git started that holds its output open holds the read
// no longer than ctx. git is asked in the C locale, which changes no setting
// it reads, so its own words are git's untranslated ones. A failure carries
// git's own words, home and credentials hidden, and git's standard error as
// written apart (gitStderr), never shown.
func workingTreeAsk(ctx context.Context, env []string, dir string, limit int, args ...string) (string, error) {
	cmd := gitutil.WorkingTreeCommitCommand(ctx, dir, args...)
	if env != nil {
		cmd.Env = slices.Clip(env)
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C", "LANG=C", "LANGUAGE=")
	out := &cappedBuffer{limit: limit}
	stderr := &cappedBuffer{limit: 4096}
	if err := procgroup.RunToEOF(ctx, cmd, out, stderr); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("git %s did not answer in the time it had: %w", args[0], ctx.Err())
		}
		return out.b.String(), &gitStderr{
			err:    fmt.Errorf("git %s: %w (%s)", args[0], err, fsutil.RedactHome(gitutil.ScrubURLCredentials(strings.TrimSpace(stderr.b.String())))),
			stderr: stderr.b.String(),
		}
	}
	if out.over {
		return "", fmt.Errorf("git %s answered past %d bytes", args[0], limit)
	}
	return out.b.String(), nil
}

// cappedBuffer keeps the first limit bytes written to it and notes whether
// more came.
type cappedBuffer struct {
	b     strings.Builder
	limit int
	over  bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := c.limit - c.b.Len(); room < len(p) {
		c.over = true
		if room > 0 {
			c.b.Write(p[:room])
		}
		return len(p), nil
	}
	c.b.Write(p)
	return len(p), nil
}

// hashDir records each entry of dir, at every depth, under its path: a
// regular file by its mode and content, a symlink by its target's text and by
// what it resolves to (linkHash), a subdirectory by its kind and then each
// entry in it, anything else by its kind. A symlink that resolves to a
// directory outside dir has that directory's entries recorded under the
// link's path, each such directory once; one that resolves to a directory
// inside dir is recorded as a link only, since that directory's entries are
// already recorded where they stand. Entries are walked in name order, depth
// first, and each is examined once, so the cost is linear in the tree. Past
// maxHookEntries entries the walk stops and records a marker at dir's own
// path; over says so. A directory that is not there records nothing; one that
// cannot be read records that, so its becoming readable is a change too.
// steps counts the entries and directories examined, for the test of cost.
func hashDir(into map[string]string, salt, dir string) (steps int, over bool) {
	des, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 1, false
	}
	if err != nil {
		into[dir] = saltedHash(salt, "unreadable directory")
		return 1, false
	}
	w := &hookWalk{into: into, salt: salt, seen: map[string]bool{}, steps: 1}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		w.root = real
		w.seen[real] = true
	}
	w.entries(dir, dir, des)
	if w.over {
		into[dir] = saltedHash(salt, "entries past", strconv.Itoa(maxHookEntries), "count", strconv.Itoa(countEntries(dir)))
	}
	return w.steps, w.over
}

// maxHookCount bounds countEntries.
const maxHookCount = 64 * maxHookEntries

// countEntries counts the entries at every depth of dir (resolved, when dir
// is itself a link), following no link below it, up to maxHookCount: the marker of a tree past maxHookEntries carries it, so
// an entry added or removed past the bound changes the record.
func countEntries(dir string) int {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real // a hooks directory that is itself a link is counted where it resolves
	}
	n := 0
	_ = filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
		if p == dir {
			return nil
		}
		if n++; n >= maxHookCount {
			return fs.SkipAll
		}
		return nil
	})
	return n
}

// hookWalk is one hashDir walk: root is the hooks directory's resolved path
// ("" when it cannot be resolved, in which case no link's directory is walked
// into), seen the resolved directories walked, n the entries recorded.
type hookWalk struct {
	into  map[string]string
	salt  string
	root  string
	seen  map[string]bool
	n     int
	steps int
	over  bool
}

// entries records under at each entry of the directory read at from as des:
// the same path, or, for a directory reached through a link, where the link
// resolves.
func (w *hookWalk) entries(at, from string, des []fs.DirEntry) {
	for _, de := range des {
		if w.n >= maxHookEntries {
			w.over = true
			return
		}
		w.n++
		w.steps++
		path, real := filepath.Join(at, de.Name()), filepath.Join(from, de.Name())
		fi := hashFileAs(w.into, w.salt, path, real)
		switch {
		case fi == nil:
		case fi.IsDir():
			w.dir(path, real)
		case fi.Mode()&fs.ModeSymlink != 0:
			w.linkedDir(path, real)
		}
		if w.over {
			return
		}
	}
}

// dir walks the directory read at from, recording its entries under at.
func (w *hookWalk) dir(at, from string) {
	des, err := os.ReadDir(from)
	w.steps++
	if err != nil {
		w.into[at] = saltedHash(w.salt, "unreadable directory")
		return
	}
	w.entries(at, from, des)
}

// linkedDir walks, under the link's path at, the directory the symlink read
// at from resolves to, when that is outside the hooks directory and not
// walked yet.
func (w *hookWalk) linkedDir(at, from string) {
	if w.root == "" {
		return
	}
	real, err := filepath.EvalSymlinks(from)
	if err != nil || w.seen[real] || within(w.root, real) {
		return
	}
	if fi, err := os.Stat(real); err != nil || !fi.IsDir() {
		return
	}
	w.seen[real] = true
	w.dir(at, real)
}

// within says whether path is root or lies under it, both resolved.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// hashFile records path as hashDir records an entry, and returns what it
// examined (nil when it recorded nothing or could not examine it); a path
// that is not there records nothing.
func hashFile(into map[string]string, salt, path string) fs.FileInfo {
	return hashFileAs(into, salt, path, path)
}

// hashFileAs is hashFile recording under at what it reads at from (an entry
// of a directory a link resolves to, recorded under the link's path).
func hashFileAs(into map[string]string, salt, at, from string) fs.FileInfo {
	fi, err := os.Lstat(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		into[at] = saltedHash(salt, "unreadable")
		return nil
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, _ := os.Readlink(from)
		into[at] = linkHash(salt, from, target)
	case fi.Mode().IsRegular():
		h, ok := regularHash(salt, from, fi.Mode().Perm())
		if !ok {
			into[at] = saltedHash(salt, "unreadable")
			return nil
		}
		into[at] = h
	default:
		into[at] = kindHash(salt, fi.Mode())
	}
	return fi
}

// regularHash is a regular file's record: its mode, then its bytes.
func regularHash(salt, path string, perm fs.FileMode) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	h := regularHasher(salt, perm)
	if _, err := io.Copy(h, f); err != nil {
		return "", false
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

// linkHash, regularHasher and kindHash are how hashFile records an entry: a
// symlink by its target's text and by what the link resolves to (a regular
// file's mode and bytes, anything else by its kind, a link that resolves to
// nothing as such), a regular file by its mode and then its bytes (written to
// the hasher), anything else by its kind. The lane's hooks pin (pinLaneHooks)
// records the entries it proves the same way, so a later check reads them as
// unchanged.
func linkHash(salt, link, target string) string {
	fi, err := os.Stat(link)
	switch {
	case err != nil:
		return saltedHash(salt, "symlink", target, "unresolved")
	case fi.Mode().IsRegular():
		if h, ok := regularHash(salt, link, fi.Mode().Perm()); ok {
			return saltedHash(salt, "symlink", target, "file", h)
		}
		return saltedHash(salt, "symlink", target, "unreadable")
	default:
		return saltedHash(salt, "symlink", target, "kind", fi.Mode().Type().String())
	}
}

func regularHasher(salt string, perm fs.FileMode) hash.Hash {
	h := sha256.New()
	writePart(h, salt)
	writePart(h, "file "+perm.String())
	return h
}

func kindHash(salt string, mode fs.FileMode) string {
	return saltedHash(salt, "kind", mode.Type().String())
}

// changes names, in order, each key, each lane worktree's key and hook and
// then each file whose hash differs between was and now: added, removed or
// changed. Never a value. A lane worktree that is no longer read is not a
// change when its lane landed or was discarded: nothing runs there any more;
// while its lane is in progress it is named gone, since the loop still runs
// git at its path. A live lane's path that became, or stopped being, other
// than a real directory is named changed. A lane worktree that is new is
// compared against the checkout as was recorded it
// (laneBaseline), for what it reads beyond or otherwise than the checkout: a
// key or hook the checkout has and it lacks starts nothing there. A lane's
// key whose change the checkout's own keys already name is not named again.
func (was configFingerprint) changes(repoRoot string, now configFingerprint) []string {
	var out []string
	named := map[string]bool{}
	diff := func(a, b map[string]string, name func(string) string, skip map[string]bool, removals bool) {
		var names []string
		for k := range a {
			names = append(names, k)
		}
		for k := range b {
			if _, ok := a[k]; !ok {
				names = append(names, k)
			}
		}
		slices.Sort(names)
		for _, k := range names {
			if skip[k] {
				continue
			}
			old, had := a[k]
			cur, has := b[k]
			change := ""
			switch {
			case !had:
				change = " (added)"
			case !has:
				if !removals {
					continue
				}
				change = " (removed)"
			case old != cur:
				change = " (changed)"
			default:
				continue
			}
			if skip == nil {
				named[k] = true
			}
			out = append(out, name(k)+change)
		}
	}
	diff(was.Keys, now.Keys, func(k string) string { return k }, nil, true)
	paths := make([]string, 0, len(now.Worktrees))
	for p := range now.Worktrees {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		cur := now.Worktrees[p]
		base, recorded := was.Worktrees[p]
		if !recorded {
			base = was.laneBaseline(repoRoot, p, cur)
		}
		in := " in the worktree " + fsutil.RedactHome(p)
		if base.Dir != cur.Dir {
			out = append(out, "the worktree "+fsutil.RedactHome(p)+" (changed)")
		}
		diff(base.Keys, cur.Keys, func(k string) string { return k + in }, named, recorded)
		diff(base.Hooks, cur.Hooks, fsutil.RedactHome, map[string]bool{}, recorded)
	}
	gone := make([]string, 0, len(was.Worktrees))
	for p := range was.Worktrees {
		if _, read := now.Worktrees[p]; !read && now.live[p] {
			gone = append(gone, p)
		}
	}
	slices.Sort(gone)
	for _, p := range gone {
		out = append(out, "the worktree "+fsutil.RedactHome(p)+" (gone)")
	}
	diff(was.Files, now.Files, fsutil.RedactHome, map[string]bool{}, true)
	return out
}

// laneBaseline is what a lane worktree the fingerprint does not hold (one
// made after it was recorded, and not pinned as it opened: pinLaneHooks) is
// compared against: the checkout as recorded.
// A lane reads the repository's shared config as the checkout does, so its
// keys are compared with the checkout's; a hook in its own copy of a relative
// hooks directory is compared with the checkout's file at the same relative
// path, and any other hook with the checkout's entry at the same path.
func (was configFingerprint) laneBaseline(repoRoot, lane string, cur worktreeFingerprint) worktreeFingerprint {
	base := worktreeFingerprint{Keys: was.Keys, Hooks: map[string]string{}}
	for p := range cur.Hooks {
		at := p
		if rel, err := filepath.Rel(lane, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			at = filepath.Join(repoRoot, rel)
		}
		if h, ok := was.Files[at]; ok {
			base.Hooks[p] = h
		}
	}
	return base
}

// nameChanges renders changed lines for anything printed or recorded:
// each through the canonical refusal redactor, so a secret that reached a
// name another way than a URL subsection is still sealed.
func nameChanges(repoRoot string, changed []string) []string {
	out := make([]string, len(changed))
	for i, c := range changed {
		out[i] = scanner.RedactRefusal(repoRoot, c)
	}
	return out
}

// readFingerprint reads the run's fingerprint; ok is false when the run has
// none. A file the loop did not write (unparseable, of another version, or
// with no salt) is refused rather than replaced.
func readFingerprint(root *os.Root, runID string) (configFingerprint, bool, error) {
	rel := fingerprintRel(runID)
	data, err := localtier.ReadInRoot(root, rel, maxFingerprintBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return configFingerprint{}, false, nil
	}
	if rf := tierRefusal(StageConfig, "", err); rf != nil {
		return configFingerprint{}, false, rf
	}
	remedy := "the loop is the file's only writer; restore it, or remove it and the run's next step records the configuration as it then stands"
	if err != nil {
		return configFingerprint{}, false, refuse(StageConfig, "", "", fmt.Sprintf("%s cannot be read as the run's configuration fingerprint: %v", rel, err), remedy)
	}
	var fp configFingerprint
	if err := json.Unmarshal(data, &fp); err != nil || fp.SchemaVersion != configFingerprintVersion || fp.Salt == "" {
		return configFingerprint{}, false, refuse(StageConfig, "", "", rel+" is not a configuration fingerprint this abcd wrote", remedy)
	}
	if fp.Keys == nil {
		fp.Keys = map[string]string{}
	}
	if fp.Files == nil {
		fp.Files = map[string]string{}
	}
	if fp.Worktrees == nil {
		fp.Worktrees = map[string]worktreeFingerprint{}
	}
	return fp, true, nil
}

// writeFingerprint replaces the run's fingerprint atomically. The caller holds
// the run tier's lock.
func writeFingerprint(root *os.Root, runID string, fp configFingerprint) error {
	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	rel := fingerprintRel(runID)
	if err := fsutil.WriteFileAtomicInRoot(root, rel, append(data, '\n'), filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// configTripwire is the check made at the start of a step: it takes the run
// tier's lock and runs configTripwireIn. A call this frame cannot place (a
// run id of the wrong shape, no tier, no run directory) passes, and the
// caller's own frame refuses it.
func configTripwire(repoRoot, runID string) error {
	if !ValidRunID(runID) || tierPresent(repoRoot) != nil ||
		!fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(runRel(runID)))) {
		return nil
	}
	return withLock(repoRoot, func(root *os.Root) error { return configTripwireIn(repoRoot, root, runID) })
}

// configTripwireIn checks the repository's configuration against the run's
// fingerprint, the run tier's lock held. A run with none (every run started
// before the tripwire) records one now and passes; a complete run passes, its
// steps taking no git step; an unchanged repository passes and writes
// nothing; a changed one is refused, naming each changed key or file and the
// way to accept it.
//
// Every read the check makes that gives no answer it can take stops the step
// at the config stage, naming the read (readRefusal): a configuration git
// cannot read, a hooks directory git cannot name or names otherwise than as
// one path, a read that did not finish. It is never read as nothing changed,
// whether or not a fingerprint was recorded: the check's reads run in abcd's
// isolated environment and the step's git in others, so a failure in one is
// not the same failure in the other. One read only in part (past the read's
// cap) is refused too, since part of it went unseen. The one answer of none
// it takes is a lane's path where git says there is no repository
// (takeFingerprint).
func configTripwireIn(repoRoot string, root *os.Root, runID string) error {
	st, err := readStateIn(root, runID)
	if err != nil || st.Complete() {
		return nil // the caller's frame reads the state and answers for it
	}
	was, ok, err := readFingerprint(root, runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Second)
	lanes := laneWorktrees(repoRoot, st)
	if !ok {
		salt, err := newSalt()
		if err != nil {
			return err
		}
		fp, err := takeFingerprint(repoRoot, lanes, salt, now)
		if err != nil {
			return readRefusal(repoRoot, err)
		}
		return writeFingerprint(root, runID, fp)
	}
	cur, err := takeFingerprint(repoRoot, lanes, was.Salt, now)
	if err != nil {
		return readRefusal(repoRoot, err)
	}
	changed := was.changes(repoRoot, cur)
	if len(changed) == 0 {
		return nil
	}
	return refuse(StageConfig, "", "",
		"the repository's git configuration changed during the run: "+strings.Join(nameChanges(repoRoot, changed), ", ")+
			"; a setting that names a program runs on the loop's next git step as on your own, so no git step is taken",
		"look at what changed (`git config --local --list --show-origin`, the hooks directory and info/attributes); "+
			"once you trust it, run the same `abcd implement step` again with "+AcceptConfigFlag+"; nothing was changed")
}

// readRefusal is the step's refusal at the config stage of a read the check
// depends on that gave no answer it can take, naming the read and why: a
// settings read past its cap says so; every other error (a readFailure, or
// one no read named) is refused as a read that gave no answer.
func readRefusal(repoRoot string, err error) error {
	if errors.Is(err, gitutil.ErrConfigCap) {
		return refuse(StageConfig, "", "", "the repository's git configuration is too large to read whole, so whether it changed during the run cannot be told: "+scanner.RedactRefusal(repoRoot, fsutil.RedactHome(err.Error())),
			"look at the repository's git configuration (`git config --local --list --show-origin`); nothing was changed")
	}
	return refuse(StageConfig, "", "", "a read the check of the repository's git configuration depends on gave no answer it can take, so whether it changed during the run cannot be told: "+scanner.RedactRefusal(repoRoot, fsutil.RedactHome(err.Error())),
		readRemedy("run the step again"))
}

// AcceptConfigChange re-records the run's fingerprint from the repository as
// it stands, on the person's word once they have looked at what changed
// (`abcd implement step --accept-config-change`), and names in the run record
// each key or file it accepted. It returns those names; nothing changed (or a
// run with no fingerprint yet) accepts nothing and records the fingerprint
// alone.
func AcceptConfigChange(repoRoot, runID string, o Options) ([]string, error) {
	var accepted []string
	err := saveRun(repoRoot, runID, o, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		was, ok, err := readFingerprint(root, runID)
		var bad *Refusal
		if err != nil && !errors.As(err, &bad) {
			return false, err
		}
		salt, serr := newSalt()
		if serr != nil {
			return false, serr
		}
		lanes := laneWorktrees(repoRoot, *st)
		cur, err := takeFingerprint(repoRoot, lanes, salt, now)
		if err != nil {
			return false, refuse(StageConfig, "", "", "the repository's git configuration cannot be read, so there is nothing to accept: "+scanner.RedactRefusal(repoRoot, fsutil.RedactHome(err.Error())),
				"settle what git reports, then accept again; nothing was changed")
		}
		if ok {
			again, err := takeFingerprint(repoRoot, lanes, was.Salt, now)
			if err != nil {
				return false, refuse(StageConfig, "", "", "the repository's git configuration cannot be read, so there is nothing to accept: "+scanner.RedactRefusal(repoRoot, fsutil.RedactHome(err.Error())),
					"settle what git reports, then accept again; nothing was changed")
			}
			accepted = nameChanges(repoRoot, was.changes(repoRoot, again))
		} else if bad != nil {
			accepted = []string{ConfigFingerprintFileName + " (replaced)"}
		}
		if err := writeFingerprint(root, runID, cur); err != nil {
			return false, err
		}
		if len(accepted) == 0 {
			return false, nil
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageConfig,
			Note: "the person accepted a change to the repository's git configuration: " + strings.Join(accepted, ", ")})
		st.UpdatedAt = now
		return true, nil
	})
	return accepted, err
}

// ConfigTripwire checks every run in progress in the checkout, as a step
// would, for a git step taken outside a run's own step (the shared network
// probe `abcd implement outage probe` runs): the first changed run refuses it.
func ConfigTripwire(repoRoot string) error {
	runs, err := Runs(repoRoot)
	if err != nil {
		return err
	}
	for _, st := range runs {
		if st.Complete() {
			continue
		}
		if err := configTripwire(repoRoot, st.RunID); err != nil {
			if r, ok := AsRefusal(err); ok {
				r.Reason = st.RunID + ": " + r.Reason
				r.Remedy = strings.Replace(r.Remedy, "run the same `abcd implement step` again with "+AcceptConfigFlag,
					"run `abcd implement step --run "+st.RunID+" "+AcceptConfigFlag+"`", 1)
			}
			return err
		}
	}
	return nil
}

// maxHookTreeBytes caps git's listing of a lane's hooks directory as a commit
// records it; a listing past it is refused whole.
const maxHookTreeBytes = 1 << 20

// hookTreeEntry is one entry of a hooks directory as a commit records it:
// its mode, its object and, for a blob, the size git stores for it (-1 for a
// tree).
type hookTreeEntry struct {
	mode, oid string
	size      int64
}

// pinLaneHooks is the lane's hooks pin (iss-2611071820538335), made by the
// worktree stage under the run tier's lock, before any commit in the lane.
// base is the commit the lane opens on and tip the default branch's tip as
// origin reported it: base is tip, or, where the lane's branch already stood
// (a worktree adopted, or a branch a killed step left), where that branch
// left tip's history, an ancestor of tip.
//
// A lane whose hooks directory is its own copy (the last core.hooksPath the
// lane reads is relative, so git takes it under the lane's working tree) is
// pinned from git's objects as stored (no replace ref or graft file stands in
// for them: hooksTreeAt, hookTreeEntries), never from what anything wrote in
// the lane: the directory's tree in base must be its tree in tip, and what
// stands on disk must be base's entries exactly, at every depth, each name,
// each file's size and bytes (its blob id recomputed from them), its exec
// bit, each symlink's target and each subdirectory's kind. A symlink inside
// the hooks directory is recorded by its target's text and by what it
// resolves to, itself an entry proved; one that points elsewhere in the
// lane's working tree is proved through the link against base's entry at the
// target's path (provenHooks); one that points outside the working tree is
// refused, with a remedy that says base itself holds it, since a lane made
// afresh would be refused again. The hooks directory is the one git names in
// the lane (hooksDirAt); where git cannot name one, the lane is refused. Any
// difference is refused at the worktree stage and nothing is pinned. The
// entries proved are recorded, hashed from the bytes
// that were proved, as the run's fingerprint holds the lane, with each watched
// key the lane reads at the hash the fingerprint recorded for the checkout, so
// from then on the lane's hooks are compared with its own commit and never
// with the checkout's copy, and a key the lane reads otherwise than the
// checkout as recorded, or that the checkout lacks, is still named at the next
// check.
//
// A lane whose hooks directory is not its own copy (the checkout's, or an
// absolute or home path) is not pinned and is compared with the checkout as
// recorded (laneBaseline). With no fingerprint yet the hooks are still proved
// but nothing is recorded: the run's first record takes the lane as it then
// stands. A lane whose config git cannot read is passed over only where git
// says there is no repository at its path (laneReadFailure), as
// takeFingerprint passes it over; any other failure stops the step at the
// config stage, naming the read.
func pinLaneHooks(c Context, laneID, lanePath, base, tip string) error {
	lane := filepath.Clean(lanePath)
	entries, err := gitutil.RepoConfig(lane, watchedConfig)
	if errors.Is(err, gitutil.ErrConfigCap) {
		return refuse(string(StageWorktree), "", laneID, "the lane's git configuration is too large to read whole, so the hooks git runs there cannot be told: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"look at the repository's git configuration (`git config --local --list --show-origin`); nothing was committed in the lane")
	}
	if err != nil {
		if err := laneReadFailure(lane, "reading the lane's watched settings (git config --get-regexp), in "+isolatedEnv.name, err); err != nil {
			return heldReadRefusal(c, laneID, err)
		}
		return nil
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	fp, recorded, err := readFingerprint(root, c.State.RunID)
	if err != nil {
		return err
	}
	read := map[string]string{}
	hashEntries(read, fp.Salt, entries)
	again := ", then run `abcd implement step` again; nothing was committed in the lane"
	hooksDir, err := hooksDirAt(lane)
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"git cannot name the hooks directory it runs in the lane, so the lane's hooks cannot be proved: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"look at the core.hooksPath git reads (`git config --show-scope --show-origin --get-all core.hooksPath`)"+again)
	}
	rel, err := ownHooksDir(lane, hooksDir)
	if err != nil {
		return refuse(StageConfig, "", laneID,
			"a read the lane's hooks proof depends on gave no answer it can take, so the lane's hooks cannot be proved: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			readRemedy("run `abcd implement step` again"))
	}
	if rel == "" {
		return nil
	}
	dir := filepath.Join(lane, filepath.FromSlash(rel))
	tree, err := hooksTreeAt(c.RepoRoot, base, rel)
	tipTree := tree
	if err == nil && base != tip {
		tipTree, err = hooksTreeAt(c.RepoRoot, tip, rel)
	}
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"the hooks directory "+scanner.RedactRefusal(c.RepoRoot, rel)+" of the lane's commit or of "+Remote+"'s tip cannot be read, so the lane's hooks cannot be proved: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"settle what git reports when it lists the hooks directory in "+Remote+"'s tip "+tip[:12]+" (git ls-tree)"+again)
	}
	if tipTree != tree {
		return refuse(string(StageWorktree), "", laneID,
			fmt.Sprintf("the lane's commit %s holds other hooks in %s than %s's tip %s, so they are not the default branch's reviewed hooks", base[:12], scanner.RedactRefusal(c.RepoRoot, rel), Remote, tip[:12]),
			"remove the lane's worktree and branch (`git worktree remove --force "+fsutil.RedactHome(lane)+"`, `git branch -D "+lanesBranch(c, laneID)+"`) so the stage cuts the lane afresh from the tip"+again)
	}
	want, err := hookTreeEntries(c.RepoRoot, tree)
	if err == nil {
		var hooks map[string]string
		if hooks, err = provenHooks(fp.Salt, dir, want, &proofTree{repoRoot: c.RepoRoot, commit: base, dir: lane}); err == nil {
			// Between this proof and the lane's first git step another
			// process of the account can still write in the lane; the pin
			// holds the hashes proved here, so that step's check names it.
			if !recorded {
				return nil
			}
			keys := map[string]string{}
			for k := range read {
				if h, ok := fp.Keys[k]; ok {
					keys[k] = h
				}
			}
			fp.Worktrees[lane] = worktreeFingerprint{Keys: keys, Hooks: hooks}
			return writeFingerprint(root, c.State.RunID, fp)
		}
	}
	var pe *proofError
	if errors.As(err, &pe) && len(pe.linkOut) > 0 {
		// The commit itself holds the link, so a lane made afresh would be
		// refused again: the remedy says so rather than send the person to
		// remake it.
		return refuse(string(StageWorktree), "", laneID,
			"the repository's committed hooks link outside the lane's working tree, so they cannot be proved against the commit the lane opens on, "+base[:12]+", and no lane is run there: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"the links are the repository's own and a lane made afresh holds them too; the loop is not run in this repository while its committed hooks link outside it; nothing was committed in the lane")
	}
	return refuse(string(StageWorktree), "", laneID,
		"the lane's hooks are not those of the commit it opens on, "+base[:12]+": "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
		"look at what wrote in the lane's hooks directory once git made it (the repository's post-checkout hook runs there); remove the lane's worktree (`git worktree remove --force "+fsutil.RedactHome(lane)+"`) so the stage makes it afresh"+again)
}

// ownHooksDir is the hooks directory hooksDir (as git names it: hooksDirAt),
// relative to dir and slash-separated, when it lies inside the working tree
// at dir, read as written or with both resolved (a checkout reached through a
// linked parent); "" when it lies outside it, is the tree itself, lies under
// the tree's own .git entry (named exactly so), or lies in the git directory
// or the common git directory git reports for dir, resolved (git's own hooks
// there, which no commit can hold, since git tracks no path in its own
// directory). A git directory git cannot report is
// a readFailure: the caller stops rather than compare or skip on a guess.
func ownHooksDir(dir, hooksDir string) (string, error) {
	if hooksDir == "" {
		return "", nil
	}
	in := func(dir, hooksDir string) string {
		rel, err := filepath.Rel(dir, hooksDir)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	rel := in(filepath.Clean(dir), filepath.Clean(hooksDir))
	if rel == "" {
		rel = in(fsutil.RealExistingPath(filepath.Clean(dir)), fsutil.RealExistingPath(filepath.Clean(hooksDir)))
	}
	if first, _, _ := strings.Cut(rel, "/"); rel == "" || first == ".git" {
		// git's own entry at the top of the tree (the git directory, or in a
		// linked worktree the file naming it), whose name git compares
		// exactly here and never tracks.
		return "", nil
	}
	real := fsutil.RealExistingPath(filepath.Clean(hooksDir))
	for _, flag := range []string{"--git-dir", "--git-common-dir"} {
		gitDir, err := gitutil.RevParseAbsPath(dir, flag)
		if err != nil {
			return "", &readFailure{what: "asking git for the git directory (git rev-parse " + flag + ")", why: fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))}
		}
		if g := fsutil.RealExistingPath(gitDir); real == g || within(g, real) {
			return "", nil
		}
	}
	return rel, nil
}

// checkoutHooksPath is the core.hooksPath git takes at the checkout, from
// the settings scopes the lane's add reads (gitutil.WorkingTreeCommitCommand,
// the person's global and system settings included), "" when none sets it.
func checkoutHooksPath(repoRoot string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	cmd := gitutil.WorkingTreeCommitCommand(ctx, repoRoot, "config", "--get", "core.hooksPath")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee) && ee.ExitCode() == 1 && len(out) == 0:
		return "", nil
	case err != nil:
		return "", fmt.Errorf("git config --get core.hooksPath: %v (%s)", err, strings.TrimSpace(stderr.String()))
	case len(out) > 4096:
		return "", errors.New("git config --get core.hooksPath answered past 4096 bytes")
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// standingBranchHooks is the worktree stage's check, before the add, of a
// branch already standing under the lane's name where origin reported tip.
// head is the branch's head, resolved once by the stage, which refuses the
// lane after the add if the branch then holds another commit, removing the
// worktree it added so the next step judges the branch again. Where hooksDir
// (the directory git names at the checkout, hooksDirAt, under the
// core.hooksPath the add is also told) is a working tree's own copy, head's
// hooks directory, as stored (hooksTreeAt), must be the tip's;
// either one that cannot be read is refused, never read as one that records
// nothing. The check is defensive: it stops nothing from running, since git's
// add runs the post-checkout hook of the checkout's own copy whatever the
// branch holds, and the proof after the add (pinLaneHooks) would refuse such
// a lane too; it refuses it before a worktree is made rather than after. The
// branch's content outside the hooks directory is its own and is not vouched
// for. A hooks directory outside the working tree passes.
func standingBranchHooks(c Context, laneID, head, branch, tip, hooksDir string) error {
	rel, err := ownHooksDir(c.RepoRoot, hooksDir)
	if err != nil {
		return heldReadRefusal(c, laneID, err)
	}
	if rel == "" {
		return nil
	}
	shown := scanner.RedactRefusal(c.RepoRoot, rel)
	mine, err := hooksTreeAt(c.RepoRoot, head, rel)
	if err == nil {
		var theirs string
		if theirs, err = hooksTreeAt(c.RepoRoot, tip, rel); err == nil {
			if mine == theirs {
				return nil
			}
			return refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the branch %s already stands, and its hooks in %s are not those of %s's tip %s, so it is not checked out", branch, shown, Remote, tip[:12]),
				"look at what made the branch; remove it (`git branch -D "+branch+"`) so the stage cuts the lane afresh from the tip, then run `abcd implement step` again; nothing was made")
		}
	}
	return refuse(string(StageWorktree), "", laneID,
		fmt.Sprintf("the hooks in %s of the branch %s, which already stands, or of %s's tip %s cannot be read, so the branch is not checked out: %s", shown, branch, Remote, tip[:12], scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error()))),
		"settle what git reports when it lists the branch's hooks directory (git ls-tree), or remove the branch (`git branch -D "+branch+"`) so the stage cuts the lane afresh from the tip, then run `abcd implement step` again; nothing was made")
}

// lanesBranch is the branch the worktree stage cuts for laneID.
func lanesBranch(c Context, laneID string) string {
	if lw, err := laneWorktree(c.RepoRoot, c.State.RunID, laneID); err == nil {
		return lw.Branch
	}
	return BranchPrefix + c.State.RunID + "-" + laneID
}

// hooksTreeAt is the object a commit records at rel, "" when it records
// nothing there. It is read as stored (gitutil.RunCappedAsStored): a replace
// ref or a graft file cannot stand another commit or tree in for the one the
// commit stores. It lists rel from the commit's root (`ls-tree --full-tree`,
// whose paths git matches literally), so git listing nothing is nothing
// recorded, and git failing to answer (no such commit, an object it cannot
// read) is an error, never read as nothing recorded.
func hooksTreeAt(repoRoot, commit, rel string) (string, error) {
	out, err := gitutil.RunCappedAsStored(repoRoot, maxHookTreeBytes, "ls-tree", "-z", "--full-tree", commit, "--", rel)
	if err != nil {
		return "", fmt.Errorf("git ls-tree %s -- %s: %v", commit, rel, err)
	}
	for _, rec := range strings.Split(out, "\x00") {
		meta, name, ok := strings.Cut(rec, "\t")
		if !ok || name != rel {
			continue
		}
		if f := strings.Fields(meta); len(f) == 3 && gitutil.IsFullSHA(f[2]) {
			return f[2], nil
		}
		return "", errors.New("git ls-tree listed the hooks directory in a form it does not document")
	}
	return "", nil
}

// hookTreeEntries lists the tree tree at every depth (`ls-tree -r -t`), by
// each entry's slash-separated path under it (none for ""), as stored
// (gitutil.RunCappedAsStored): its mode, its object and a blob's stored size.
// An object that is not a tree, a listing past its cap or past maxHookEntries
// entries, or a record git does not document is an error.
func hookTreeEntries(repoRoot, tree string) (map[string]hookTreeEntry, error) {
	want := map[string]hookTreeEntry{}
	if tree == "" {
		return want, nil
	}
	out, err := gitutil.RunCappedAsStored(repoRoot, maxHookTreeBytes, "ls-tree", "-r", "-t", "-z", "-l", tree)
	if err != nil {
		return nil, fmt.Errorf("the commit's hooks directory is not a directory git can list: %v", err)
	}
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		meta, name, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 4 || name == "" || !gitutil.IsFullSHA(f[2]) {
			return nil, errors.New("git ls-tree listed the hooks directory in a form it does not document")
		}
		size := int64(-1)
		if f[1] == "blob" {
			n, err := strconv.ParseInt(f[3], 10, 64)
			if err != nil || n < 0 {
				return nil, errors.New("git ls-tree listed a hook's size in a form it does not document")
			}
			size = n
		}
		want[name] = hookTreeEntry{mode: f[0], oid: f[2], size: size}
		if len(want) > maxHookEntries {
			return nil, fmt.Errorf("the commit's hooks directory holds more than %d entries", maxHookEntries)
		}
	}
	return want, nil
}

// proofTree is the working tree a hooks directory stands in (dir) and the
// commit it is proved against: a symlink whose target leaves the hooks
// directory and stays in the tree is proved by what the commit holds at the
// target's path (hookProof.link).
type proofTree struct{ repoRoot, commit, dir string }

// proofError is provenHooks's refusal: every difference, each named by its
// path (problems), with the entries that stand there and the commit lacks
// (extra) and the symlinks whose target leaves the working tree (linkOut)
// told apart, so a caller can say which it met.
type proofError struct {
	problems []string
	extra    []string
	linkOut  []string
}

func (e *proofError) Error() string { return strings.Join(e.problems, "; ") }

// maxLinkDepth bounds the symlinks one hook's link is followed through.
const maxLinkDepth = 8

// provenHooks proves what stands in dir, at every depth, is want exactly and
// returns each entry's record, as hashDir keys and hashes it, from what was
// proved. A directory that is not there holds nothing. A symlink whose target
// stays in dir is recorded and its target proved as an entry of dir; one
// whose target leaves dir and stays in tree's working tree is proved by the
// commit's own content at the target's path, followed through each link on
// the way and, for a directory, at every depth, its entries recorded under
// the link's path as hashDir records them, each such directory once; one
// whose target leaves the working tree, or any link where tree is nil, cannot
// be proved and is refused. Every difference is named, each by its path, in
// the one error (a *proofError); a tree past maxHookEntries entries, or a
// directory in it that cannot be read, is refused whole.
func provenHooks(salt, dir string, want map[string]hookTreeEntry, tree *proofTree) (map[string]string, error) {
	p := &hookProof{salt: salt, root: dir, tree: tree, out: map[string]string{}, dirs: map[string]bool{}}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		p.realRoot = real
		p.dirs[real] = true
	}
	if tree != nil {
		if real, err := filepath.EvalSymlinks(tree.dir); err == nil {
			p.realTree = real
		}
	}
	des, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		des, err = nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s cannot be read: %v", dir, err)
	}
	if err := p.prove(&proofScope{want: want, disk: dir, record: dir, seen: map[string]bool{}}, des); err != nil {
		return nil, err
	}
	if len(p.err.problems) > 0 {
		return nil, &p.err
	}
	return p.out, nil
}

// hookProof is one provenHooks walk: the entries proved so far, by the path
// hashDir records them under, the resolved directories walked (as hashDir's
// seen), the count of entries examined, and each difference found.
type hookProof struct {
	salt, root, realRoot, realTree string
	tree                           *proofTree
	out                            map[string]string
	dirs                           map[string]bool
	n                              int
	err                            proofError
}

// proofScope is one directory proved against one listing: the hooks
// directory itself, or one a link resolves to, read at disk and recorded
// under record.
type proofScope struct {
	want         map[string]hookTreeEntry
	disk, record string
	seen, under  map[string]bool
}

// prove walks the scope from its top, read as des, and names each entry the
// listing holds that is not there.
func (p *hookProof) prove(s *proofScope, des []fs.DirEntry) error {
	if err := p.walk(s, "", des); err != nil {
		return err
	}
	var missing []string
	for name := range s.want {
		if !s.seen[name] && !s.under[parentOf(name)] {
			missing = append(missing, filepath.Join(s.record, filepath.FromSlash(name)))
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		p.err.problems = append(p.err.problems, strings.Join(missing, ", ")+", which the commit holds, is not there")
	}
	return nil
}

// parentOf is the slash-separated directory a tree path stands in, "" at the
// top.
func parentOf(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		return name[:i]
	}
	return ""
}

// walk proves each entry of the scope's directory at rel (slash-separated, ""
// for its top), read as des, and each real subdirectory's in turn.
func (p *hookProof) walk(s *proofScope, rel string, des []fs.DirEntry) error {
	for _, de := range des {
		if p.n++; p.n > maxHookEntries {
			return fmt.Errorf("%s holds more than %d entries", p.root, maxHookEntries)
		}
		name := path.Join(rel, de.Name())
		from := filepath.Join(s.disk, filepath.FromSlash(name))
		at := filepath.Join(s.record, filepath.FromSlash(name))
		s.seen[name] = true
		e, ok := s.want[name]
		if !ok {
			p.err.problems = append(p.err.problems, at+" stands there, and the commit holds no such entry")
			p.err.extra = append(p.err.extra, at)
			continue
		}
		h, target, err := provenEntry(p.salt, from, e)
		if err == nil && e.mode == "120000" {
			err = p.link(at, from, target, 0)
		}
		if err != nil {
			p.err.problems = append(p.err.problems, at+" "+err.Error())
			if e.mode == "040000" {
				// What the commit holds under a directory that is not one
				// here is named by the directory alone.
				s.markUnder(name)
			}
			continue
		}
		p.out[at] = h
		if e.mode == "040000" {
			sub, err := os.ReadDir(from)
			if err != nil {
				return fmt.Errorf("%s cannot be read: %v", at, err)
			}
			if err := p.walk(s, name, sub); err != nil {
				return err
			}
		}
	}
	return nil
}

// markUnder records that nothing under the tree path name is named missing,
// since name itself was named.
func (s *proofScope) markUnder(name string) {
	if s.under == nil {
		s.under = map[string]bool{}
	}
	for n := range s.want {
		if strings.HasPrefix(n, name+"/") {
			s.under[parentOf(n)] = true
		}
	}
}

// link proves the symlink at from, recorded at at, whose text is target and
// is already proved to be the commit's. A target that stays in the hooks
// directory is an entry of it, proved on its own. One that leaves it and
// stays in the working tree is proved by the commit's own entry at the
// target's path, reached through real directories only (no directory on the
// way is itself a link), and followed on through a link there, up to
// maxLinkDepth; a directory there is proved whole (linkedDir). One that
// leaves the working tree is refused and noted as such.
func (p *hookProof) link(at, from, target string, depth int) error {
	if linkStaysIn(p.root, from, target) {
		return nil
	}
	// Whether the target leaves the tree is read from the link's text, which
	// is the commit's own. One that stays in it is resolved as the system
	// resolves it, component by component from the tree (linkTarget): a
	// ".." after a component that is itself a link is never read as cleaned
	// text.
	lex := target
	if !filepath.IsAbs(lex) {
		lex = filepath.Join(filepath.Dir(from), lex)
	}
	lex = filepath.Clean(lex)
	if p.tree == nil || p.realTree == "" || (!within(filepath.Clean(p.tree.dir), lex) && !within(p.realTree, lex)) {
		p.err.linkOut = append(p.err.linkOut, at)
		return errors.New("is a symlink whose target leaves the working tree, which cannot be proved against the commit")
	}
	start, text := filepath.Dir(from), target
	if filepath.IsAbs(target) {
		start, text = p.realTree, ""
		for _, base := range []string{filepath.Clean(p.tree.dir), p.realTree} {
			if rest, ok := strings.CutPrefix(target, base+string(filepath.Separator)); ok {
				text = rest
				break
			}
		}
		if text == "" {
			return errors.New("names the working tree in a form the proof does not follow")
		}
	}
	to, err := linkTarget(start, text)
	if err != nil {
		return err
	}
	rel := ""
	if within(p.realTree, to) {
		if r, err := filepath.Rel(p.realTree, to); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	if rel == "" {
		return fmt.Errorf("resolves to %s, outside the working tree its text names, which the proof does not prove", to)
	}
	if rel == "." {
		return errors.New("points at the working tree itself, which the proof does not prove")
	}
	if depth >= maxLinkDepth {
		return fmt.Errorf("reaches its target through more than %d symlinks, which the proof does not follow", maxLinkDepth)
	}
	e, ok, err := treeEntryAt(p.tree.repoRoot, p.tree.commit, rel)
	if err != nil {
		return fmt.Errorf("points at %s, which cannot be listed in the commit: %v", rel, err)
	}
	if !ok {
		return fmt.Errorf("points at %s, which the commit does not hold", rel)
	}
	_, next, err := provenEntry(p.salt, to, e)
	if err != nil {
		return fmt.Errorf("points at %s, which %s", rel, err.Error())
	}
	switch e.mode {
	case "120000":
		return p.link(at, to, next, depth+1)
	case "040000":
		return p.linkedDir(at, to, e.oid)
	}
	return nil
}

// linkTarget is the path target, a link's text relative to the directory
// start, names, resolved as the system resolves it: from start (resolved),
// one component at a time, a ".." taking the parent of the directory reached
// so far. Every component but the last must be a real directory: one that
// is a link, or is not there, is refused, since the proof does not follow a
// link on the way. The last component is the target itself, proved by the
// caller.
func linkTarget(start, target string) (string, error) {
	cur, err := filepath.EvalSymlinks(start)
	if err != nil {
		return "", fmt.Errorf("stands in a directory that cannot be resolved: %v", err)
	}
	parts := strings.Split(filepath.ToSlash(target), "/")
	last := len(parts) - 1
	for last >= 0 && (parts[last] == "" || parts[last] == ".") {
		last--
	}
	for i, part := range parts {
		switch {
		case part == "" || part == ".":
			continue
		case part == "..":
			cur = filepath.Dir(cur)
			continue
		}
		cur = filepath.Join(cur, part)
		if i == last {
			break
		}
		fi, err := os.Lstat(cur)
		switch {
		case err != nil:
			return "", fmt.Errorf("names %s on its way, which cannot be examined, so the proof does not follow it: %v", cur, err)
		case fi.Mode()&fs.ModeSymlink != 0:
			return "", fmt.Errorf("reaches its target through %s, itself a symlink, which the proof does not follow", cur)
		case !fi.IsDir():
			return "", fmt.Errorf("names %s on its way, which is not a directory", cur)
		}
	}
	return cur, nil
}

// linkedDir proves the directory at to, which a link recorded at at resolves
// to and the commit holds as the tree oid, at every depth, recording its
// entries under at as hashDir does: only when it is outside the hooks
// directory and not walked yet.
func (p *hookProof) linkedDir(at, to, oid string) error {
	real, err := filepath.EvalSymlinks(to)
	if err != nil {
		return fmt.Errorf("resolves to a directory that cannot be examined: %v", err)
	}
	if p.dirs[real] || (p.realRoot != "" && within(p.realRoot, real)) {
		return nil
	}
	p.dirs[real] = true
	want, err := hookTreeEntries(p.tree.repoRoot, oid)
	if err != nil {
		return err
	}
	des, err := os.ReadDir(to)
	if err != nil {
		return fmt.Errorf("resolves to a directory that cannot be read: %v", err)
	}
	return p.prove(&proofScope{want: want, disk: to, record: at, seen: map[string]bool{}}, des)
}

// treeEntryAt is the entry a commit records at rel, slash-separated from its
// root, as stored (gitutil.RunCappedAsStored): its mode, object and a blob's
// size; ok is false when it records nothing there.
func treeEntryAt(repoRoot, commit, rel string) (hookTreeEntry, bool, error) {
	out, err := gitutil.RunCappedAsStored(repoRoot, maxHookTreeBytes, "ls-tree", "-z", "-l", "--full-tree", commit, "--", rel)
	if err != nil {
		return hookTreeEntry{}, false, fmt.Errorf("git ls-tree: %v", err)
	}
	for _, rec := range strings.Split(out, "\x00") {
		meta, name, ok := strings.Cut(rec, "\t")
		if !ok || name != rel {
			continue
		}
		f := strings.Fields(meta)
		if len(f) != 4 || !gitutil.IsFullSHA(f[2]) {
			return hookTreeEntry{}, false, errors.New("git ls-tree listed the entry in a form it does not document")
		}
		size := int64(-1)
		if f[1] == "blob" {
			n, err := strconv.ParseInt(f[3], 10, 64)
			if err != nil || n < 0 {
				return hookTreeEntry{}, false, errors.New("git ls-tree listed a size in a form it does not document")
			}
			size = n
		}
		return hookTreeEntry{mode: f[0], oid: f[2], size: size}, true, nil
	}
	return hookTreeEntry{}, false, nil
}

// provenEntry proves the entry at path is e and returns its record and, for a
// symlink, its target: a symlink whose target is e's blob, a regular file
// whose bytes are e's blob and whose owner exec bit is set exactly when e is
// 100755, a directory where e is a tree. A file is read once, through the
// descriptor its record and its blob id are both taken from. A symlink's
// record holds what it resolves to (linkHash); where its target may lie is
// the caller's to judge (hookProof.link). Any other mode (a submodule's) is
// refused.
func provenEntry(salt, path string, e hookTreeEntry) (string, string, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot be examined: %v", err)
	}
	switch e.mode {
	case "120000":
		if fi.Mode()&fs.ModeSymlink == 0 {
			return "", "", errors.New("is not the symlink the commit holds")
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", "", fmt.Errorf("cannot be read: %v", err)
		}
		if int64(len(target)) != e.size {
			return "", "", errors.New("points elsewhere than the commit's symlink")
		}
		blob, ok := blobHasher(e.oid, int64(len(target)))
		if !ok {
			return "", "", errors.New("is recorded under an object name of no known length")
		}
		_, _ = io.WriteString(blob, target)
		if hex.EncodeToString(blob.Sum(nil)) != e.oid {
			return "", "", errors.New("points elsewhere than the commit's symlink")
		}
		return linkHash(salt, path, target), target, nil
	case "100644", "100755":
		if !fi.Mode().IsRegular() {
			return "", "", errors.New("is not the regular file the commit holds")
		}
		f, err := os.Open(path)
		if err != nil {
			return "", "", fmt.Errorf("cannot be read: %v", err)
		}
		defer f.Close()
		held, err := f.Stat()
		if err != nil || !os.SameFile(fi, held) || !held.Mode().IsRegular() {
			return "", "", errors.New("changed while it was read")
		}
		if exec := held.Mode().Perm()&0o100 != 0; exec != (e.mode == "100755") {
			return "", "", fmt.Errorf("has its exec bit %s, unlike the commit's mode %s", map[bool]string{true: "set", false: "clear"}[exec], e.mode)
		}
		// The size is judged before a byte is read, so a file of any size at
		// a hook's name costs one stat, and the read below is bounded by the
		// size git stores for the blob.
		if held.Size() != e.size {
			return "", "", fmt.Errorf("is %d bytes, and the commit's file %d", held.Size(), e.size)
		}
		blob, ok := blobHasher(e.oid, held.Size())
		if !ok {
			return "", "", errors.New("is recorded under an object name of no known length")
		}
		rec := regularHasher(salt, held.Mode().Perm())
		n, err := io.Copy(io.MultiWriter(blob, rec), io.LimitReader(f, held.Size()+1))
		if err != nil || n != held.Size() {
			return "", "", errors.New("changed while it was read")
		}
		if hex.EncodeToString(blob.Sum(nil)) != e.oid {
			return "", "", errors.New("differs from the commit's file")
		}
		return hex.EncodeToString(rec.Sum(nil)), "", nil
	case "040000":
		if !fi.IsDir() {
			return "", "", errors.New("is not the directory the commit holds")
		}
		return kindHash(salt, fi.Mode()), "", nil
	}
	return "", "", fmt.Errorf("is an entry of mode %s, which the proof does not prove", e.mode)
}

// linkStaysIn says whether the symlink at path, whose target text is target,
// names a path inside root, read as written and, where it resolves, as
// resolved. The text is judged against root as given and against root
// resolved, so an absolute target written in resolved form stays in a root
// reached through a linked parent; a link that resolves nowhere but is
// written inside root stays in.
func linkStaysIn(root, path, target string) bool {
	to := target
	if !filepath.IsAbs(to) {
		to = filepath.Join(filepath.Dir(path), to)
	}
	to = filepath.Clean(to)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	if !within(filepath.Clean(root), to) && !within(realRoot, to) {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return true
	}
	return within(realRoot, real)
}

// blobHasher is the hash git names a blob of size bytes by, in the object
// format oid's length says (SHA-1 for 40 hex digits, SHA-256 for 64), its
// header written.
func blobHasher(oid string, size int64) (hash.Hash, bool) {
	var h hash.Hash
	switch len(oid) {
	case 40:
		h = sha1.New()
	case 64:
		h = sha256.New()
	default:
		return nil, false
	}
	fmt.Fprintf(h, "blob %d\x00", size)
	return h, true
}
