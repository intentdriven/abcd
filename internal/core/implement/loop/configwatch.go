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
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
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

// maxHookEntries bounds the hooks directory entries one fingerprint hashes; a
// directory past it is fingerprinted by its first entries and its count.
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
// checkout's. A lane worktree whose config git cannot read is passed over:
// git cannot run there either. A live lane's path that is there but is not a
// real directory is recorded as what stands there (laneDirKind), whether or
// not git reads a config through it.
func takeFingerprint(repoRoot string, lanes []laneDir, salt string, now time.Time) (configFingerprint, error) {
	fp := configFingerprint{SchemaVersion: configFingerprintVersion, RecordedAt: now, Salt: salt,
		Keys: map[string]string{}, Worktrees: map[string]worktreeFingerprint{}, Files: map[string]string{},
		live: map[string]bool{}}
	entries, err := gitutil.RepoConfig(repoRoot, watchedConfig)
	if err != nil {
		return fp, err
	}
	hooksPath := hashEntries(fp.Keys, salt, entries)
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return fp, err
	}
	defaultHooks := filepath.Join(common, "hooks")
	hooks := defaultHooks
	if hooksPath != "" {
		hooks = expandHooksPath(repoRoot, hooksPath)
	}
	hashDir(fp.Files, salt, hooks)
	hashFile(fp.Files, salt, filepath.Join(common, "info", "attributes"))
	for _, ld := range lanes {
		lane := ld.Path
		wt := worktreeFingerprint{Keys: map[string]string{}, Hooks: map[string]string{}}
		if ld.Live {
			fp.live[lane] = true
			wt.Dir = laneDirKind(salt, lane)
		}
		entries, err := gitutil.RepoConfig(lane, watchedConfig)
		if errors.Is(err, gitutil.ErrConfigCap) {
			return fp, err
		}
		if err != nil {
			if wt.Dir != "" {
				fp.Worktrees[lane] = wt
			}
			continue
		}
		// The hooks directory git runs in the lane: the last core.hooksPath
		// the lane reads (its own, else the shared one), a relative value
		// taken under the lane's own working tree, as git takes it there.
		dir := defaultHooks
		if p := hashEntries(wt.Keys, salt, entries); p != "" {
			dir = expandHooksPath(lane, p)
		}
		if dir != hooks {
			hashDir(wt.Hooks, salt, dir)
		}
		fp.Worktrees[lane] = wt
	}
	return fp, nil
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

// expandHooksPath is where git looks for hooks under a core.hooksPath value:
// a leading ~/ is the home, and a relative value is relative to the working
// tree's root.
func expandHooksPath(repoRoot, v string) string {
	if rest, ok := strings.CutPrefix(v, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	if !filepath.IsAbs(v) {
		return filepath.Join(repoRoot, v)
	}
	return filepath.Clean(v)
}

// hashDir records each entry of dir under its path: a regular file by its
// mode and content, a symlink by its target, anything else by its kind. A
// directory that is not there records nothing; one that cannot be read
// records that, so its becoming readable is a change too.
func hashDir(into map[string]string, salt, dir string) {
	des, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		into[dir] = saltedHash(salt, "unreadable directory")
		return
	}
	if len(des) > maxHookEntries {
		into[dir] = saltedHash(salt, "entries", fmt.Sprint(len(des)))
		des = des[:maxHookEntries]
	}
	for _, de := range des {
		hashFile(into, salt, filepath.Join(dir, de.Name()))
	}
}

// hashFile records path as hashDir records an entry; a path that is not there
// records nothing.
func hashFile(into map[string]string, salt, path string) {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		into[path] = saltedHash(salt, "unreadable")
		return
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, _ := os.Readlink(path)
		into[path] = symlinkHash(salt, target)
	case fi.Mode().IsRegular():
		f, err := os.Open(path)
		if err != nil {
			into[path] = saltedHash(salt, "unreadable")
			return
		}
		defer f.Close()
		h := regularHasher(salt, fi.Mode().Perm())
		if _, err := io.Copy(h, f); err != nil {
			into[path] = saltedHash(salt, "unreadable")
			return
		}
		into[path] = hex.EncodeToString(h.Sum(nil))
	default:
		into[path] = kindHash(salt, fi.Mode())
	}
}

// symlinkHash, regularHasher and kindHash are how hashFile records an entry:
// a symlink by its target, a regular file by its mode and then its bytes
// (written to the hasher), anything else by its kind. The lane's hooks pin
// (pinLaneHooks) records the entries it proves the same way, so a later check
// reads them as unchanged.
func symlinkHash(salt, target string) string { return saltedHash(salt, "symlink", target) }

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
// A configuration git cannot read at all (no git, a config file git refuses)
// passes: git that cannot read it cannot start a program it names either, and
// the step's own git meets the same failure. One read only in part (past the
// read's cap) is refused, since part of it went unseen.
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
			// Nothing was recorded to compare against, so the run is not
			// refused for it; the next check tries again.
			return nil
		}
		return writeFingerprint(root, runID, fp)
	}
	cur, err := takeFingerprint(repoRoot, lanes, was.Salt, now)
	if errors.Is(err, gitutil.ErrConfigCap) {
		return refuse(StageConfig, "", "", "the repository's git configuration is too large to read whole, so whether it changed during the run cannot be told: "+scanner.RedactRefusal(repoRoot, fsutil.RedactHome(err.Error())),
			"look at the repository's git configuration (`git config --local --list --show-origin`); nothing was changed")
	}
	if err != nil {
		return nil
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

// AcceptConfigChange re-records the run's fingerprint from the repository as
// it stands, on the person's word once they have looked at what changed
// (`abcd implement step --accept-config-change`), and names in the run record
// each key or file it accepted. It returns those names; nothing changed (or a
// run with no fingerprint yet) accepts nothing and records the fingerprint
// alone.
func AcceptConfigChange(repoRoot, runID string, o Options) ([]string, error) {
	var accepted []string
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
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
// stands on disk must be base's entries exactly, each name, each file's size
// and bytes (its blob id recomputed from them), its exec bit, each symlink's
// target and each subdirectory's kind. A subdirectory's contents are neither
// proved nor pinned, and a symlink is proved by its target's text, not by the
// file it points to. Any difference is refused at the worktree stage and
// nothing is pinned. The entries proved are recorded, hashed from the bytes
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
// stands. A lane whose config git cannot read is passed over, as
// takeFingerprint passes it over: git cannot run there either.
func pinLaneHooks(c Context, laneID, lanePath, base, tip string) error {
	lane := filepath.Clean(lanePath)
	entries, err := gitutil.RepoConfig(lane, watchedConfig)
	if errors.Is(err, gitutil.ErrConfigCap) {
		return refuse(string(StageWorktree), "", laneID, "the lane's git configuration is too large to read whole, so the hooks git runs there cannot be told: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"look at the repository's git configuration (`git config --local --list --show-origin`); nothing was committed in the lane")
	}
	if err != nil {
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
	hooksPath := hashEntries(read, fp.Salt, entries)
	if hooksPath == "" {
		return nil
	}
	rel := ownHooksDir(lane, hooksPath)
	if rel == "" {
		return nil
	}
	dir := filepath.Join(lane, filepath.FromSlash(rel))
	again := ", then run `abcd implement step` again; nothing was committed in the lane"
	tree, err := hooksTreeAt(c.RepoRoot, base, rel)
	tipTree := tree
	if err == nil && base != tip {
		tipTree, err = hooksTreeAt(c.RepoRoot, tip, rel)
	}
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"the hooks directory "+rel+" of the lane's commit or of "+Remote+"'s tip cannot be read, so the lane's hooks cannot be proved: "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
			"settle what git reports (`git ls-tree "+tip[:12]+" -- "+rel+"`)"+again)
	}
	if tipTree != tree {
		return refuse(string(StageWorktree), "", laneID,
			fmt.Sprintf("the lane's commit %s holds other hooks in %s than %s's tip %s, so they are not the default branch's reviewed hooks", base[:12], rel, Remote, tip[:12]),
			"remove the lane's worktree and branch (`git worktree remove --force "+fsutil.RedactHome(lane)+"`, `git branch -D "+lanesBranch(c, laneID)+"`) so the stage cuts the lane afresh from the tip"+again)
	}
	want, err := hookTreeEntries(c.RepoRoot, tree)
	if err == nil {
		var hooks map[string]string
		if hooks, err = provenHooks(fp.Salt, dir, want); err == nil {
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
	return refuse(string(StageWorktree), "", laneID,
		"the lane's hooks are not those of the commit it opens on, "+base[:12]+": "+scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error())),
		"look at what wrote in the lane's hooks directory once git made it (the repository's post-checkout hook runs there); remove the lane's worktree (`git worktree remove --force "+fsutil.RedactHome(lane)+"`) so the stage makes it afresh"+again)
}

// ownHooksDir is the hooks directory, relative and slash-separated, that git
// runs in a working tree at dir when hooksPath (the last core.hooksPath the
// repository's own settings give) names the tree's own copy; "" when it names
// none or a directory outside the tree.
func ownHooksDir(dir, hooksPath string) string {
	if hooksPath == "" {
		return ""
	}
	rel, err := filepath.Rel(dir, expandHooksPath(dir, hooksPath))
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// checkoutHooksPath is the core.hooksPath git takes at the checkout, from
// every settings scope the lane's add reads (gitutil.WorkingTreeCommitCommand,
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
// lane after the add if the branch then holds another commit. Where hooksPath
// (checkoutHooksPath, which the add is also told) names a working tree's own
// copy, head's hooks directory, as stored (hooksTreeAt), must be the tip's;
// either one that cannot be read is refused, never read as one that records
// nothing. The check is defensive: it stops nothing from running, since git's
// add runs the post-checkout hook of the checkout's own copy whatever the
// branch holds, and the proof after the add (pinLaneHooks) would refuse such
// a lane too; it refuses it before a worktree is made rather than after. The
// branch's content outside the hooks directory is its own and is not vouched
// for. A hooks directory outside the working tree passes.
func standingBranchHooks(c Context, laneID, head, branch, tip, hooksPath string) error {
	rel := ownHooksDir(c.RepoRoot, hooksPath)
	if rel == "" {
		return nil
	}
	mine, err := hooksTreeAt(c.RepoRoot, head, rel)
	if err == nil {
		var theirs string
		if theirs, err = hooksTreeAt(c.RepoRoot, tip, rel); err == nil {
			if mine == theirs {
				return nil
			}
			return refuse(string(StageWorktree), "", laneID,
				fmt.Sprintf("the branch %s already stands, and its hooks in %s are not those of %s's tip %s, so it is not checked out", branch, rel, Remote, tip[:12]),
				"look at what made the branch; remove it (`git branch -D "+branch+"`) so the stage cuts the lane afresh from the tip, then run `abcd implement step` again; nothing was made")
		}
	}
	return refuse(string(StageWorktree), "", laneID,
		fmt.Sprintf("the hooks in %s of the branch %s, which already stands, or of %s's tip %s cannot be read, so the branch is not checked out: %s", rel, branch, Remote, tip[:12], scanner.RedactRefusal(c.RepoRoot, fsutil.RedactHome(err.Error()))),
		"settle what git reports (`git ls-tree "+branch+" -- "+rel+"`), or remove the branch (`git branch -D "+branch+"`) so the stage cuts the lane afresh from the tip, then run `abcd implement step` again; nothing was made")
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

// hookTreeEntries lists the tree tree by entry name (none for ""), as stored
// (gitutil.RunCappedAsStored): its mode, its object and a blob's stored size.
// An object that is not a tree, a listing past its cap or past maxHookEntries
// entries, or a record git does not document is an error.
func hookTreeEntries(repoRoot, tree string) (map[string]hookTreeEntry, error) {
	want := map[string]hookTreeEntry{}
	if tree == "" {
		return want, nil
	}
	out, err := gitutil.RunCappedAsStored(repoRoot, maxHookTreeBytes, "ls-tree", "-z", "-l", tree)
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
	}
	if len(want) > maxHookEntries {
		return nil, fmt.Errorf("the commit's hooks directory holds more than %d entries", maxHookEntries)
	}
	return want, nil
}

// provenHooks proves what stands in dir is want exactly and returns each
// entry's record, as hashDir keys and hashes it, from what was proved. A
// directory that is not there holds nothing.
func provenHooks(salt, dir string, want map[string]hookTreeEntry) (map[string]string, error) {
	des, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		des, err = nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s cannot be read: %v", dir, err)
	}
	if len(des) > maxHookEntries {
		return nil, fmt.Errorf("%s holds more than %d entries", dir, maxHookEntries)
	}
	out := map[string]string{}
	for _, de := range des {
		path := filepath.Join(dir, de.Name())
		e, ok := want[de.Name()]
		if !ok {
			return nil, fmt.Errorf("%s stands there, and the commit holds no such entry", path)
		}
		h, err := provenEntry(salt, path, e)
		if err != nil {
			return nil, fmt.Errorf("%s %v", path, err)
		}
		out[path] = h
	}
	if len(out) != len(want) {
		var missing []string
		for name := range want {
			if _, ok := out[filepath.Join(dir, name)]; !ok {
				missing = append(missing, filepath.Join(dir, name))
			}
		}
		slices.Sort(missing)
		return nil, fmt.Errorf("%s, which the commit holds, is not there", strings.Join(missing, ", "))
	}
	return out, nil
}

// provenEntry proves the entry at path is e and returns its record: a
// symlink whose target is e's blob, a regular file whose bytes are e's blob
// and whose owner exec bit is set exactly when e is 100755, a directory where
// e is a tree. A file is read once, through the descriptor its record and its
// blob id are both taken from.
func provenEntry(salt, path string, e hookTreeEntry) (string, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("cannot be examined: %v", err)
	}
	switch e.mode {
	case "120000":
		if fi.Mode()&fs.ModeSymlink == 0 {
			return "", errors.New("is not the symlink the commit holds")
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", fmt.Errorf("cannot be read: %v", err)
		}
		if int64(len(target)) != e.size {
			return "", errors.New("points elsewhere than the commit's symlink")
		}
		blob, ok := blobHasher(e.oid, int64(len(target)))
		if !ok {
			return "", errors.New("is recorded under an object name of no known length")
		}
		_, _ = io.WriteString(blob, target)
		if hex.EncodeToString(blob.Sum(nil)) != e.oid {
			return "", errors.New("points elsewhere than the commit's symlink")
		}
		return symlinkHash(salt, target), nil
	case "100644", "100755":
		if !fi.Mode().IsRegular() {
			return "", errors.New("is not the regular file the commit holds")
		}
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("cannot be read: %v", err)
		}
		defer f.Close()
		held, err := f.Stat()
		if err != nil || !os.SameFile(fi, held) || !held.Mode().IsRegular() {
			return "", errors.New("changed while it was read")
		}
		if exec := held.Mode().Perm()&0o100 != 0; exec != (e.mode == "100755") {
			return "", fmt.Errorf("has its exec bit %s, unlike the commit's mode %s", map[bool]string{true: "set", false: "clear"}[exec], e.mode)
		}
		// The size is judged before a byte is read, so a file of any size at
		// a hook's name costs one stat, and the read below is bounded by the
		// size git stores for the blob.
		if held.Size() != e.size {
			return "", fmt.Errorf("is %d bytes, and the commit's file %d", held.Size(), e.size)
		}
		blob, ok := blobHasher(e.oid, held.Size())
		if !ok {
			return "", errors.New("is recorded under an object name of no known length")
		}
		rec := regularHasher(salt, held.Mode().Perm())
		n, err := io.Copy(io.MultiWriter(blob, rec), io.LimitReader(f, held.Size()+1))
		if err != nil || n != held.Size() {
			return "", errors.New("changed while it was read")
		}
		if hex.EncodeToString(blob.Sum(nil)) != e.oid {
			return "", errors.New("differs from the commit's file")
		}
		return hex.EncodeToString(rec.Sum(nil)), nil
	case "040000":
		if !fi.IsDir() {
			return "", errors.New("is not the directory the commit holds")
		}
		return kindHash(salt, fi.Mode()), nil
	}
	return "", fmt.Errorf("is an entry of mode %s, which the pin does not prove", e.mode)
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
