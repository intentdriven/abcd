package loop

// configwatch.go is the run tripwire (iss-2610111600058575). A program the
// repository's own git configuration names (a content filter, a hook, a
// transport or credential helper, a diff or merge driver, a signing program)
// runs on the person's own plain git commands with no abcd involved, so the
// loop's git steps keep running the repository's settings as they are and
// never blank, pin or override them (the abcd-fixes-what-it-adds principle).
// What the loop adds is a tripwire: the first check of a run records a
// fingerprint of every repository-level setting that can name a program, and
// of the hooks directory and info/attributes (and of each lane worktree's own
// per-worktree settings, which git reads there once extensions.worktreeConfig
// is set), and every later check, made
// before the loop's next git step, compares against it. A change stops the
// step before git starts, naming each changed key or file, until the person
// has looked and accepts it (`abcd implement step --accept-config-change`).
//
// The fingerprint lives in its own file beside the run's state. It holds
// salted hashes, never a value: a credential helper's value may carry a
// secret, so only key names and file paths are ever stored or reported.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	`|core\.(hookspath|sshcommand|fsmonitor|pager|editor|worktree)` +
	`|extensions\.worktreeconfig` +
	`|credential\..+` +
	`|remote\..+\.(uploadpack|receivepack)` +
	`|url\..+\.(insteadof|pushinsteadof)` +
	`|protocol\.(.+\.)?allow` +
	`|diff\..+\.(command|textconv)` +
	`|merge\..+\.driver` +
	`|gpg\.(.+\.)?program` +
	`|include\.path` +
	`|includeif\..+\.path` +
	`)$`

// maxHookEntries bounds the hooks directory entries one fingerprint hashes; a
// directory past it is fingerprinted by its first entries and its count.
const maxHookEntries = 1024

// maxFingerprintBytes caps a fingerprint file read.
const maxFingerprintBytes = 1 << 20

// configFingerprint is the tripwire's file. Keys and Files map a watched key
// name, or a watched file's path, to a salted hash of what it holds;
// WorktreeKeys maps each lane worktree's path to the watched keys its own
// per-worktree config sets, hashed the same way. The salt is the file's own,
// drawn when it is first written.
type configFingerprint struct {
	SchemaVersion int                          `json:"schema_version"`
	RecordedAt    time.Time                    `json:"recorded_at"`
	Salt          string                       `json:"salt"`
	Keys          map[string]string            `json:"keys"`
	WorktreeKeys  map[string]map[string]string `json:"worktree_keys"`
	Files         map[string]string            `json:"files"`
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
// for each lane worktree in lanes, the watched keys that worktree's own config
// sets and the hooks directory a core.hooksPath there names. A lane worktree
// whose config git cannot read is passed over: git cannot run there either.
func takeFingerprint(repoRoot string, lanes []string, salt string, now time.Time) (configFingerprint, error) {
	fp := configFingerprint{SchemaVersion: configFingerprintVersion, RecordedAt: now, Salt: salt,
		Keys: map[string]string{}, WorktreeKeys: map[string]map[string]string{}, Files: map[string]string{}}
	entries, err := gitutil.RepoConfig(repoRoot, watchedConfig)
	if err != nil {
		return fp, err
	}
	hooksPath := hashEntries(fp.Keys, salt, entries)
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return fp, err
	}
	hooks := filepath.Join(common, "hooks")
	if hooksPath != "" {
		hooks = expandHooksPath(repoRoot, hooksPath)
	}
	hashDir(fp.Files, salt, hooks)
	hashFile(fp.Files, salt, filepath.Join(common, "info", "attributes"))
	for _, lane := range lanes {
		entries, err := gitutil.RepoConfig(lane, watchedConfig)
		if errors.Is(err, gitutil.ErrConfigCap) {
			return fp, err
		}
		if err != nil {
			continue
		}
		own := entries[:0:0]
		for _, e := range entries {
			if e.Scope == "worktree" {
				own = append(own, e)
			}
		}
		keys := map[string]string{}
		if p := hashEntries(keys, salt, own); p != "" {
			hashDir(fp.Files, salt, expandHooksPath(lane, p))
		}
		fp.WorktreeKeys[lane] = keys
	}
	return fp, nil
}

// hashEntries hashes each key's values, in the order git reads them, into
// into, and returns the last core.hooksPath among them (the one git uses).
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
		into[k] = saltedHash(salt, vs...)
	}
	return hooksPath
}

// laneWorktrees are the run's lane worktrees on disk, other than the checkout:
// the linked worktrees the loop's lane steps run git in.
func laneWorktrees(repoRoot string, st State) []string {
	var out []string
	for _, l := range st.Lanes {
		if l.Worktree != "" && fsutil.IsRealDir(l.Worktree) && !samePath(repoRoot, l.Worktree, repoRoot) {
			out = append(out, filepath.Clean(l.Worktree))
		}
	}
	return out
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
		into[path] = saltedHash(salt, "symlink", target)
	case fi.Mode().IsRegular():
		f, err := os.Open(path)
		if err != nil {
			into[path] = saltedHash(salt, "unreadable")
			return
		}
		defer f.Close()
		h := sha256.New()
		writePart(h, salt)
		writePart(h, "file "+fi.Mode().Perm().String())
		if _, err := io.Copy(h, f); err != nil {
			into[path] = saltedHash(salt, "unreadable")
			return
		}
		into[path] = hex.EncodeToString(h.Sum(nil))
	default:
		into[path] = saltedHash(salt, "kind", fi.Mode().Type().String())
	}
}

// changes names, in order, each key, each lane worktree's own key and then
// each file whose hash differs between was and now: added, removed or
// changed. Never a value. A lane worktree that is no longer there (its lane
// landed or was discarded) is not a change: nothing runs there any more; one
// that is new is compared against a worktree that set nothing.
func (was configFingerprint) changes(now configFingerprint) []string {
	var out []string
	diff := func(a, b map[string]string, name func(string) string) {
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
			old, had := a[k]
			cur, has := b[k]
			switch {
			case !had:
				out = append(out, name(k)+" (added)")
			case !has:
				out = append(out, name(k)+" (removed)")
			case old != cur:
				out = append(out, name(k)+" (changed)")
			}
		}
	}
	diff(was.Keys, now.Keys, func(k string) string { return k })
	paths := make([]string, 0, len(now.WorktreeKeys))
	for p := range now.WorktreeKeys {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		in := " in the worktree " + fsutil.RedactHome(p)
		diff(was.WorktreeKeys[p], now.WorktreeKeys[p], func(k string) string { return k + in })
	}
	diff(was.Files, now.Files, fsutil.RedactHome)
	return out
}

// readFingerprint reads the run's fingerprint; ok is false when the run has
// none. A file the loop did not write (unparseable, of another version, or
// with no salt) is refused rather than replaced.
func readFingerprint(root *os.Root, runID string) (configFingerprint, bool, error) {
	rel := fingerprintRel(runID)
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxFingerprintBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return configFingerprint{}, false, nil
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
	if fp.WorktreeKeys == nil {
		fp.WorktreeKeys = map[string]map[string]string{}
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

// configTripwire is the check made before a step's git: it takes the run
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
		return refuse(StageConfig, "", "", "the repository's git configuration is too large to read whole, so whether it changed during the run cannot be told: "+fsutil.RedactHome(err.Error()),
			"look at the repository's git configuration (`git config --local --list --show-origin`); nothing was changed")
	}
	if err != nil {
		return nil
	}
	changed := was.changes(cur)
	if len(changed) == 0 {
		return nil
	}
	return refuse(StageConfig, "", "",
		"the repository's git configuration changed during the run: "+strings.Join(changed, ", ")+
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
			return false, refuse(StageConfig, "", "", "the repository's git configuration cannot be read, so there is nothing to accept: "+fsutil.RedactHome(err.Error()),
				"settle what git reports, then accept again; nothing was changed")
		}
		if ok {
			again, err := takeFingerprint(repoRoot, lanes, was.Salt, now)
			if err != nil {
				return false, err
			}
			accepted = was.changes(again)
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
