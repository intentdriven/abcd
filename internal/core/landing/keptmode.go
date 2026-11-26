package landing

// The kept reading of facilitator.mode (iss-2610302057380175). The status
// line's row has 300 ms for every git it asks, and reading the default
// branch's committed mode took most of the gits it started, so on a loaded
// machine the merging element dropped and every refresh wrote a note. The
// prompt hook takes that reading anyway, on every prompt and with a second to
// do it in (AutoForSessions); it keeps what the default branch's file says in
// one small file per repository under abcd's home, with the ref and the tip it
// was read at and the time. The row reads the kept file, the ref files that
// say whether that tip still stands, and the computer's own configuration as
// it is now, and asks git nothing.
//
// A default branch that moved since does not turn the mode unknown: the row
// goes on naming the kept reading's, with the time it was read, until a later
// read ends in time.
//
// The kept reading is for showing and for nothing else. The merge gate, abcd
// land, the build loop, the guard, the prompt line and the board each take
// their own (readFacilitatorAt, ReadFetchedFacilitator,
// readDefaultFacilitatorContext), and none of them opens this file
// (TestOnlyTheRowReadsTheKeptMode, TestTheKeptModeDecidesNothing).
//
// A kept reading that is missing, malformed, another repository's, or not
// this account's alone to change is NOT KNOWN: never auto, and never human.
// Where no default branch was ever fetched, and the repository has commits but
// no conventionally named branch of its own, there is no tip to key a reading
// on: the hook keeps none, and the row says the mode is not read rather than
// show the bundled default as if it were. A repository with no commit at all
// has no default branch that could say anything, so there the computer's file
// alone is asked (noCommit).

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// keptRel is the folder the kept readings are in, relative to the account's
// home: one file per repository, named for the sha256 of its common git
// directory's path, which its worktrees share.
var keptRel = abcdhome.Rel("facilitator-mode")

// The kept file's caps and its one schema.
const (
	keptSchema   = 2
	maxKeptBytes = 4 << 10
	// maxRefBytes bounds one loose ref, gitdir or commondir file, and
	// maxPackedRefsBytes the packed listing; past either the disk names no
	// default branch.
	maxRefBytes        = 4 << 10
	maxPackedRefsBytes = 16 << 20
	// maxRefEntries bounds the walk that looks for any ref at all.
	maxRefEntries = 4096
)

// What the default branch's .abcd/config.json said of facilitator.mode, as
// kept: one of the two modes, that it sets neither, or that what it says was
// refused.
const (
	keptUnset   = "unset"
	keptRefused = "refused"
)

// keptMode is the kept file's whole content. Repo is repoMark's, so a reading
// is never shown for another repository that came to stand at the same path.
// Person is what selects a person's own entry, as the reading found it
// (personmode.go), absent where no entry needed it.
type keptMode struct {
	Schema    int        `json:"schema_version"`
	Repo      string     `json:"repo"`
	Ref       string     `json:"ref"`
	Tip       string     `json:"tip"`
	Committed string     `json:"committed"`
	ReadAt    time.Time  `json:"read_at"`
	Person    *keptFacts `json:"person,omitempty"`
}

// keptFacts is a reading's projectFacts and git's identity against the
// record ("" where it was not asked), as kept.
type keptFacts struct {
	Roots    []string `json:"roots"`
	Unread   string   `json:"unread"`
	Address  string   `json:"address"`
	EnvIdent bool     `json:"env_ident"`
	Identity string   `json:"identity"`
}

// maxKeptUnread caps what a kept reading says of first commits not read.
const maxKeptUnread = 1024

// valid says whether kept facts are ones a reading could have kept.
func (f *keptFacts) valid() bool {
	if f == nil {
		return true
	}
	switch f.Identity {
	case "", gitIsRecord, gitIsOther:
	default:
		return false
	}
	for _, r := range f.Roots {
		if !gitutil.IsFullSHA(r) {
			return false
		}
	}
	return len(f.Unread) <= maxKeptUnread && (f.Address == "" || addressOf("https://"+f.Address) == f.Address)
}

// projectKey is the key the row selects an entry by: the kept facts, and
// git's identity as kept, asking git nothing.
func (f *keptFacts) projectKey() projectKey {
	if f == nil {
		return projectKey{facts: func() (projectFacts, error) {
			return projectFacts{unread: "the last reading did not read this project's first commits; the next prompt does"}, nil
		}}
	}
	facts := projectFacts{roots: f.Roots, unread: f.Unread, address: f.Address, envIdent: f.EnvIdent}
	return projectKey{facts: func() (projectFacts, error) { return facts, nil }, gitIs: f.Identity}
}

// keptFactsOf is what a person's reading kept, nil where no entry needed the
// project's facts.
func keptFactsOf(p personReading) *keptFacts {
	if p.facts == nil {
		return nil
	}
	return &keptFacts{Roots: p.facts.roots, Unread: p.facts.unread, Address: p.facts.address, EnvIdent: p.facts.envIdent, Identity: p.gitIs}
}

// keptClock is the clock a reading's time, and the row's today, are read from;
// a variable so a test can fix it.
var keptClock = time.Now

func (k keptMode) valid() bool {
	switch k.Committed {
	case ModeHuman, ModeAuto, keptUnset, keptRefused:
	default:
		return false
	}
	return k.Schema == keptSchema && gitutil.IsFullSHA(k.Tip) && k.Repo != "" && k.Ref != "" && !k.ReadAt.IsZero() && k.Person.valid()
}

// keepMode keeps what the default branch's own layer said of the mode, as the
// prompt hook just read it: vals and readErr are that read's, at the tip r
// names of ref. A read that failed in a way that says nothing of the branch (a
// git that did not answer, or the time spent) keeps nothing; a file the branch
// holds that is refused (not JSON, not a regular file, a value that is not a
// mode) is kept as refused, so the row says the mode was not read and why.
// person is the person's reading, whose facts are kept beside it. It
// answers nothing and fails quietly: a reading that could not be kept leaves
// the row saying the mode is not known.
func keepMode(ctx context.Context, r layered.Roots, ref string, vals []layerValue[string], readErr error, person personReading) {
	if r.Home == "" || abcdhome.Check(r.Home) != nil || ctx.Err() != nil {
		return
	}
	if readErr != nil {
		// The refusal may be the computer's own file's, which is not kept:
		// ask the branch's layer alone.
		branch := r
		branch.Home, branch.HomeRefused = "", true
		vals, _, readErr = modeLayers(ctx, branch)
	}
	k := keptMode{Schema: keptSchema, Ref: ref, Tip: r.RepoAt, Committed: repoLayerMode(vals), Person: keptFactsOf(person)}
	var unread *layered.UnreadError
	switch {
	case errors.As(readErr, &unread) && unread.GitFailed(), ctx.Err() != nil:
		return
	case readErr != nil:
		k.Committed = keptRefused
	case k.Committed == "":
		k.Committed = keptUnset
	}
	common, ok := commonGitDir(r.Repo)
	if !ok {
		return
	}
	// To the minute, as the row says it: a prompt that reads the same again
	// within it has nothing to write.
	k.Repo, k.ReadAt = repoMark(common), keptClock().UTC().Truncate(time.Minute)
	if !k.valid() {
		return
	}
	dir, err := fsutil.EnsureHomeDeclarationScope(r.Home, keptRel, abcdhome.DirMode)
	if err != nil {
		return
	}
	defer dir.Close()
	if now, ok := readKeptModeIn(dir, r.Home, common); ok && reflect.DeepEqual(now, k) {
		return
	}
	data, err := json.Marshal(k)
	if err != nil {
		return
	}
	// Unflushed: the next prompt keeps it again, and the hook waits on no disk.
	_ = fsutil.WriteRebuildableInRoot(dir, keptLeaf(common), data, abcdhome.FileMode)
}

// keptLeaf is the kept file's name for the repository whose common git
// directory is common.
func keptLeaf(common string) string {
	sum := sha256.Sum256([]byte(common))
	return hex.EncodeToString(sum[:]) + ".json"
}

// readKeptMode reads the kept reading of the repository whose common git directory
// is common, with a home declaration's guards; ok is false for one that is
// missing or that any of them refuses.
func readKeptMode(home, common string) (keptMode, bool) {
	dir, err := fsutil.OpenHomeDeclarationScope(home, keptRel)
	if err != nil {
		return keptMode{}, false
	}
	defer dir.Close()
	return readKeptModeIn(dir, home, common)
}

// readKeptModeIn is readKeptMode in the folder its caller holds: a regular file this
// account owns and no one else can read or write, within its cap, holding one
// JSON object of the kept shape and nothing after it.
func readKeptModeIn(dir *os.Root, home, common string) (keptMode, bool) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, path.Join(keptRel, keptLeaf(common)), maxKeptBytes, 0o077)
	if err != nil || refusal != fsutil.DeclarationOK {
		return keptMode{}, false
	}
	var k keptMode
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&k); err != nil {
		return keptMode{}, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) || !k.valid() || k.Repo != repoMark(common) {
		return keptMode{}, false
	}
	return k, true
}

// KeptMerging is the status line's merging element, read: the repository's
// facilitator.mode as the prompt hook last kept it, joined to the computer's
// own file as it stands, through the one precedence every reader uses
// (resolveMode). It asks git nothing. A reading whose tip no longer stands
// carries the time it was read. Under a refused home neither the computer's
// file nor a kept reading can be read, and the resolver's answer turns on the
// default branch, which the project may set to auto: the mode is not known,
// and the row says the computer's file was not read; only a repository with no
// commit, whose default branch can say nothing, takes the resolver's human
// there. The mode is not known, with no note, where there is no kept reading
// to use (the file comment says when); and not known, with a note for the
// front door's stderr, masked, where a file that declares it was refused.
func KeptMerging(ctx context.Context, root string) (statusline.Merging, string) {
	unknown := statusline.Merging{State: statusline.MergingUnknown}
	r, _ := layered.HomeRoots()
	vals, _, err := modeLayers(ctx, r)
	if err != nil {
		return unknown, unreadNote(root, err)
	}
	common, ok := commonGitDir(root)
	if r.HomeRefused {
		if ok && noCommit(common) {
			vals, in, _ := inputFor(r, vals, projectKey{})
			return resolveMode(vals, in).merging(), ""
		}
		return statusline.Merging{State: statusline.MergingUnknown, Whose: statusline.WhoseComputerUnread}, ""
	}
	if !ok {
		return unknown, ""
	}
	k, ok := readKeptMode(r.Home, common)
	if !ok {
		if noCommit(common) {
			vals, in, _ := inputFor(r, vals, projectKey{})
			return resolveMode(vals, in).merging(), ""
		}
		return unknown, ""
	}
	origin := layered.Config.RepoOrigin() + " at " + gitutil.ShortRef(k.Ref)
	switch k.Committed {
	case keptRefused:
		return unknown, unreadNote(root, errors.New(origin+" was refused when it was last read; abcd says why"))
	case ModeHuman, ModeAuto:
		vals = append([]layerValue[string]{{v: k.Committed, layer: layered.Repo.String(), origin: origin}}, vals...)
	}
	// The row asks git nothing: the entry is selected on what the reading
	// kept.
	vals, in, _ := inputFor(r, vals, k.Person.projectKey())
	m := resolveMode(vals, in).merging()
	if ref, tip, ok := defaultIn(common); !ok || ref != k.Ref || tip != k.Tip {
		m.Read = readTime(k.ReadAt.Local(), keptClock().Local())
	}
	return m, ""
}

// readTime is when a reading was made as the row says it: the time of day,
// with the date before it on any other day than now's.
func readTime(at, now time.Time) string {
	if at.Format(time.DateOnly) == now.Format(time.DateOnly) {
		return at.Format("15:04")
	}
	return at.Format("2006-01-02 15:04")
}

// repoMark tells the repository whose common git directory is common from
// another that comes to stand at the same path, from git's files alone: the
// directory itself, and git's description file, laid when it was made. A clone
// made again in its place is another repository to it, and so reads as not
// read until the next prompt; a change of remote is not.
func repoMark(common string) string {
	h := sha256.New()
	if fi, err := os.Stat(common); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			fmt.Fprintf(h, "ino %d\n", st.Ino)
		}
	}
	if fi, err := os.Lstat(filepath.Join(common, "description")); err == nil {
		fmt.Fprintf(h, "laid %d\n", fi.ModTime().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil))
}

// noCommit says whether the repository whose common git directory is common
// plainly has no commit: its HEAD names a branch, and no ref of any kind
// stands. A layout it does not read counts as having one, so the answer is
// never yes on a guess.
func noCommit(common string) bool {
	refs := refFiles{common: common}
	if head, err := refs.loose("HEAD"); err != nil || !strings.HasPrefix(head, "ref: refs/heads/") {
		return false
	}
	if raw, err := fsutil.ReadGuarded(filepath.Join(common, "packed-refs"), maxPackedRefsBytes); err == nil {
		for line := range strings.Lines(string(raw)) {
			if line = strings.TrimSpace(line); line != "" && line[0] != '#' {
				return false
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	entries, none := 0, true
	err := filepath.WalkDir(filepath.Join(common, "refs"), func(_ string, d fs.DirEntry, err error) error {
		if entries++; err != nil || !d.IsDir() || entries > maxRefEntries {
			none = false
			return fs.SkipAll
		}
		return nil
	})
	return err == nil && none
}

// ShownFacilitator resolves facilitator.mode for the repository holding cwd as
// the board shows it: readDefaultFacilitatorContext's answer, and for a repository
// with no commit the computer's file alone. Nothing that decides a merge reads
// it.
func ShownFacilitator(cwd string) (Facilitator, error) {
	return shownFacilitator(context.Background(), cwd)
}

func shownFacilitator(ctx context.Context, cwd string) (Facilitator, error) {
	f, err := readDefaultFacilitatorContext(ctx, cwd)
	if !errors.Is(err, ErrNoDefaultBranch) {
		return f, err
	}
	root, topErr := gitutil.ToplevelContext(ctx, cwd)
	if topErr != nil {
		return Facilitator{}, err
	}
	if common, ok := commonGitDir(root); !ok || !noCommit(common) {
		return Facilitator{}, err
	}
	r, _ := layered.HomeRoots()
	return readModeContext(ctx, r, false)
}

// diskDefault is gitutil.DefaultRef's answer for the repository holding the
// checkout at root, and the tip it stands at, read from the ref files and
// starting no git. It follows DefaultRef's order (origin/HEAD's target, then
// the conventional names under origin, then the same names locally), which
// TestTheDiskReadsTheDefaultBranchGitNames holds it to. ok is false wherever
// the files do not say plainly: a layout it does not read (another ref
// backend), a ref that names another, a listing past its cap. The answer only
// says whether a kept reading's tip still stands, so one that differs from
// git's costs the row a time beside the mode and never shows another.
func diskDefault(root string) (ref, tip string, ok bool) {
	common, ok := commonGitDir(root)
	if !ok {
		return "", "", false
	}
	return defaultIn(common)
}

func defaultIn(common string) (ref, tip string, ok bool) {
	const origin = "refs/remotes/" + Remote + "/"
	refs := refFiles{common: common}
	switch head, err := refs.loose(origin + "HEAD"); {
	case err != nil:
		return "", "", false
	default:
		if target, sym := strings.CutPrefix(head, "ref: "); sym && strings.HasPrefix(target, origin) && fsutil.ValidRelPath(target) {
			if tip, err := refs.tip(target); err != nil {
				return "", "", false
			} else if tip != "" {
				return target, tip, true
			}
		}
	}
	for _, prefix := range []string{origin, "refs/heads/"} {
		for _, n := range []string{"main", "master", "trunk", "develop"} {
			tip, err := refs.tip(prefix + n)
			if err != nil {
				return "", "", false
			}
			if tip != "" {
				return prefix + n, tip, true
			}
		}
	}
	return "", "", false
}

// refFiles reads the refs a common git directory keeps as files.
type refFiles struct {
	common string
	packed map[string]string // nil until read
}

// loose is the content of ref's own file, trimmed, "" when it has none.
func (f *refFiles) loose(ref string) (string, error) {
	raw, err := fsutil.ReadGuarded(filepath.Join(f.common, filepath.FromSlash(ref)), maxRefBytes)
	switch {
	case err == nil:
		return strings.TrimSpace(string(raw)), nil
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR), errors.Is(err, fsutil.ErrNotRegular):
		// No file: never there, or a folder left by (or holding) other refs.
		return "", nil
	}
	return "", err
}

// tip is the object ref names, its own file's over the packed listing's, ""
// when neither has it.
func (f *refFiles) tip(ref string) (string, error) {
	sha, err := f.loose(ref)
	if err != nil {
		return "", err
	}
	if sha == "" {
		if f.packed == nil {
			if f.packed, err = readPackedRefs(f.common); err != nil {
				return "", err
			}
		}
		if sha = f.packed[ref]; sha == "" {
			return "", nil
		}
	}
	if !gitutil.IsFullSHA(sha) {
		return "", errors.New("the ref does not hold an object name")
	}
	return sha, nil
}

// readPackedRefs is the packed listing's refs by name, empty when there is
// none.
func readPackedRefs(common string) (map[string]string, error) {
	out := map[string]string{}
	raw, err := fsutil.ReadGuarded(filepath.Join(common, "packed-refs"), maxPackedRefsBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for line := range strings.Lines(string(raw)) {
		sha, ref, ok := strings.Cut(strings.TrimRight(line, "\r\n"), " ")
		if ok && (strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/remotes/"+Remote+"/")) {
			out[ref] = sha
		}
	}
	return out, nil
}

// commonGitDir is the directory the checkout at root keeps its refs in, every
// link resolved: its .git, or for a linked worktree the directory its git
// file and that directory's commondir lead to.
func commonGitDir(root string) (string, bool) {
	gitDir, err := filepath.EvalSymlinks(filepath.Join(root, ".git"))
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(gitDir)
	if err != nil {
		return "", false
	}
	if !fi.IsDir() {
		raw, err := fsutil.ReadGuarded(gitDir, maxRefBytes)
		if err != nil {
			return "", false
		}
		to, ok := strings.CutPrefix(strings.TrimRight(string(raw), "\r\n"), "gitdir: ")
		if !ok {
			return "", false
		}
		gitDir = from(root, to)
	}
	raw, err := fsutil.ReadGuarded(filepath.Join(gitDir, "commondir"), maxRefBytes)
	switch {
	case err == nil:
		gitDir = from(gitDir, strings.TrimRight(string(raw), "\r\n"))
	case !errors.Is(err, fs.ErrNotExist):
		return "", false
	}
	common, err := filepath.EvalSymlinks(gitDir)
	if err != nil {
		return "", false
	}
	return common, true
}

// from is p as a path, taken from dir when it is relative.
func from(dir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(dir, p)
}
