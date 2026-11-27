package claim

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// How many live reservations one person may hold (iss-2610170717515674) is
// the repository's to set, and each machine may only lower it
// (itd-2611251809580662 decision 3, which refines dcn-2611222342270127: that
// record's rule for drain.bundle_max stands, and only the clause of its
// grounds naming reservation.max_live among the person's own settings no
// longer holds). It resolves in this order:
//
//  1. MaxLive, the bundled ten.
//  2. reservation.max_live in the repository's .abcd/config.json at origin's
//     default branch (decision 8), as the object store holds it (no replace
//     ref followed), and never from a working tree. Where it is set it
//     replaces the ten. Which commit that is depends on who asks:
//     - An act that adds a reservation, which has just fetched, asks origin
//       itself which branch is its default and at which commit (git
//       ls-remote --symref, as landing.ReadFetchedFacilitator asks it), and
//       reads the figure at the commit origin named, never at a name this
//       checkout tracks: origin/HEAD, a remote-tracking ref and the
//       conventional names the resolver falls back to are all local
//       guesses a lane's agent can move or a pushed branch can stand in for
//       (gitutil.TrackedDefaultBranch). An origin that answers but names no
//       branch with a commit as its default (none of its own, a detached
//       HEAD) has no file for the figure, and the ten applies, said in the
//       note. Where the act's fetch did not bring
//       that commit, it fetches the branch origin named. Where origin cannot
//       be asked, or the commit still is not here (the branch moved while it
//       was read), the act is refused as a failed fetch is, and is to be run
//       again: no figure is read and none is assumed, since the ten could
//       be more than the repository allows, and a commit that never reached
//       origin never sets anything.
//     - A view that only shows the figure (the ceiling row of abcd peers)
//       asks no network and reads it at origin's default branch as this
//       checkout last fetched it. The act that adds the reservation decides.
//       The drain's wait at the cap (MaxLiveFor) asks no network either: it
//       reads at the branch origin last named to an act here, and where no
//       act here has saved one it reads none and leaves the count to the
//       act, so a branch this checkout merely tracks does not hold it at a
//       wait.
//     A checkout that names no default branch of origin reads no repository
//     figure in a view, and the note says so.
//  3. reservation.max_live in ~/.abcd.noindex/config.json, the machine layer
//     loaded alone. Below the figure so far it replaces it; above it, it is
//     ignored and named in the note; equal, it changes nothing.
//
// It is one ceiling per person, counted across all of their sessions, copies
// and accounts whose holds are theirs (Self.Owns), the lanes whose pull
// requests are queued among them, so every machine of one person applies the
// same number unless that machine lowers it for itself.
//
// In the repository's file the reservation namespace admits max_live, and
// fork_origin, the fork check's repository-wide opt-out (fork.go,
// dcn-2610161141338923); reservation.github_username is a person's and is
// refused there, naming the file and the commit, so a repository can never
// say who a person is. A repository value that is not a whole number from 1
// to MaxLiveCeiling, an unknown key under reservation there, and a file that
// cannot be read at that commit each refuse an act that adds a reservation
// (reviving a lapsed one among them) naming the file at the branch: every
// misreading leans to a refusal, never to the ten. The machine's file keeps
// its three keys (reservationKeys), and a value an act would refuse there
// refuses the same way.
//
// The cap is read by no act but one that adds a reservation, so a bad value
// in either file never stops a release or the renewal of a live reservation.
// (An unknown key in the machine's file still stops every act of a person
// whose username is read from that file: the holder's reader claims the same
// namespace, holder.go.) A machine config.json that is there but cannot be
// read as the person's own (not a regular file, writable by others, not valid
// JSON) refuses an adding act the same way, naming the file, since reading
// past it could leave the figure above what the machine set; a view shows the
// figure so far and says so. A file that is not there, a home that cannot be
// searched, and a HOME inside a git working tree that is not the account's own
// (layered.MachineRoots) read as no machine file: none holds a file the person
// can be shown to have written, and a refusal there would let a HOME a
// checkout chose stop the act.

// MaxLiveKey is the configuration key.
const MaxLiveKey = holderNamespace + ".max_live"

// MaxLiveCeiling bounds the setting.
const MaxLiveCeiling = 100

// The layers a ceiling is set by, as Ceiling.LimitLayer and
// CapError.LimitLayer name them.
const (
	LimitBundled    = "bundled"
	LimitRepository = "repository"
	LimitMachine    = "machine"
)

// liveLimit is one person's resolved ceiling: the number, the layer that set
// it and where.
type liveLimit struct {
	N      int
	Layer  string
	Origin string
}

// bundledLimit is the ceiling no file sets.
func bundledLimit() liveLimit {
	return liveLimit{N: MaxLive, Layer: LimitBundled, Origin: LimitBundled}
}

// describe names the figure for a sentence: "the default of 10", "the
// repository's 4 (.abcd/config.json at origin/main)".
func (l liveLimit) describe() string {
	switch l.Layer {
	case LimitRepository:
		return fmt.Sprintf("the repository's %d (%s)", l.N, l.Origin)
	case LimitMachine:
		return fmt.Sprintf("this machine's %d (%s)", l.N, l.Origin)
	}
	return fmt.Sprintf("the default of %d", l.N)
}

// maxLive resolves the ceiling a view shows for the checkout at root, whose
// origin's default branch as last fetched the caller named as ref ("" for
// none), and the account whose home is home, with a note saying what was
// read past or ignored.
func maxLive(root, ref, home string) (liveLimit, string, error) {
	lim, note, err := repoMaxLive(root, ref)
	if err != nil {
		return liveLimit{}, "", err
	}
	lim, mnote, err := machineMaxLive(home, lim, false)
	if err != nil {
		return liveLimit{}, "", err
	}
	return lim, joinNote(note, mnote), nil
}

// actMaxLive resolves the ceiling an act that adds a reservation judges by:
// the repository's figure at the commit origin itself names as its default
// branch's tip, never at a name this checkout tracks (tracked, the act's own
// resolution after its fetch, "" for none, is only compared with it), then
// the machine's lowering. Where origin cannot be asked, or the commit it names
// cannot be brought here, it refuses with a *FetchError, as the act's own
// fetch does: no figure is read and none is assumed. Where origin answers but
// names no branch with a commit as its default (it holds no commits, or its
// HEAD is detached or points outside refs/heads/), it has no file for the
// figure: the ten applies, lowered by the machine alone, and the note says
// so.
func actMaxLive(root, tracked, home string) (liveLimit, string, error) {
	lim, note := bundledLimit(), ""
	if root != "" {
		at, err := askOriginDefault(root, MaxLiveKey)
		if err != nil {
			return liveLimit{}, "", err
		}
		if at.None != "" {
			note = fmt.Sprintf("%s, so it has no repository figure for %s and %s applies", at.None, MaxLiveKey, lim.describe())
			return machineAfter(home, lim, note)
		}
		name, tip := at.Name, at.Tip
		if tracked != "refs/remotes/"+name {
			note = fmt.Sprintf("this checkout tracks %s as %s's default branch, but %s names %s, so the figure is read at the tip %s names",
				orNone(termsafe.Sanitize(gitutil.ShortRef(tracked))), Remote, Remote, name, Remote)
		}
		var rnote string
		if lim, rnote, err = repoMaxLiveAt(root, tip, name); err != nil {
			return liveLimit{}, "", err
		}
		note = joinNote(note, rnote)
	}
	return machineAfter(home, lim, note)
}

// OriginDefault is origin's default branch as origin itself names it, for a
// reader of a repository figure at that branch (AskOriginDefault): Name as a
// person names it ("origin/main") and Tip the commit origin named, brought
// into this checkout; or, where origin answers but names no branch with a
// commit as its default, None saying what it named instead, with Name and Tip
// "".
type OriginDefault struct {
	Name string
	Tip  string
	None string
}

// AskOriginDefault asks origin which branch is its default and at which
// commit, as an act that adds a reservation asks it for reservation.max_live,
// for the figure named key: never a name this checkout tracks, which a lane's
// agent can move or a pushed branch can stand in for. It keeps the branch for
// MaxLiveFor, and fetches it where this checkout does not hold the commit
// origin named. Where origin cannot be asked, or the commit cannot be brought
// here, it is a *FetchError and no figure is to be read.
func AskOriginDefault(root, key string) (OriginDefault, error) {
	return askOriginDefault(root, key)
}

func askOriginDefault(root, key string) (OriginDefault, error) {
	branch, tip, none, err := originHeadFor(root, key)
	if err != nil {
		return OriginDefault{}, err
	}
	if none != "" {
		return OriginDefault{None: none}, nil
	}
	keepOriginDefault(root, branch)
	if !commitHere(root, tip) {
		if err := fetchTip(root, branch, tip, key); err != nil {
			return OriginDefault{}, err
		}
	}
	return OriginDefault{Name: Remote + "/" + branch, Tip: tip}, nil
}

// machineAfter is an act's lim lowered by the machine, with note joined to
// the machine's own.
func machineAfter(home string, lim liveLimit, note string) (liveLimit, string, error) {
	lim, mnote, err := machineMaxLive(home, lim, true)
	if err != nil {
		return liveLimit{}, "", err
	}
	return lim, joinNote(note, mnote), nil
}

// gapOutcome ends a refusal of the ceiling's read from origin: the act is
// safe to run again as it stands.
const gapOutcome = nothingPushed + "; run it again once " + Remote + " answers"

// orNone is s, or "none" when it is "".
func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// originHead asks origin which branch is its default and at which commit
// (git ls-remote --symref, as landing.ReadFetchedFacilitator does, read from
// the lines naming HEAD alone: originHeadLines): the branch, a valid ref name
// under refs/heads/, and the tip, a full object name. Where origin answers but
// names no such branch with a commit (none of its own, a detached HEAD, a HEAD
// outside refs/heads/), none says what it named instead and branch and tip are
// "". Only an origin that cannot be asked is a *FetchError.
func originHead(root string) (branch, tip, none string, err error) {
	return originHeadFor(root, MaxLiveKey)
}

// originHeadFor is originHead for the figure named key, which its refusal
// names.
func originHeadFor(root, key string) (branch, tip, none string, err error) {
	target, symref, tip, err := originHeadLines(root)
	if err != nil {
		r := redacted(err)
		return "", "", "", &FetchError{Reason: r.text, cause: r.notRun, what: "which branch is its default, for " + key + ",", outcome: gapOutcome}
	}
	if symref {
		branch, _ = strings.CutPrefix(target, "refs/heads/")
		if branch == target {
			branch = ""
		}
	}
	switch {
	case !gitutil.IsFullSHA(tip):
		return "", "", Remote + " holds no commit at its HEAD", nil
	case !symref:
		return "", "", Remote + "'s HEAD is detached, naming no default branch", nil
	case branch == "" || branch == "HEAD":
		return "", "", Remote + "'s HEAD names no branch under refs/heads/", nil
	}
	if _, err := localGit(root, 4096, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", "", Remote + "'s HEAD names a branch whose name is not a valid ref", nil
	}
	return termsafe.Sanitize(branch), tip, "", nil
}

// fetchTip brings origin's default branch, which origin named at tip, into
// this checkout when the act's own fetch did not (it fetched the branch this
// checkout tracks, or the branch moved since): a *FetchError when the fetch
// fails or still does not bring tip (the branch moved again while it was
// read).
func fetchTip(root, branch, tip, key string) error {
	what := key + " at " + Remote + "/" + branch
	args := []string{"fetch", "--no-tags", Remote, "+refs/heads/" + branch + ":refs/remotes/" + Remote + "/" + branch}
	trace(args...)
	if _, err := gitutil.Net(root, args...); err != nil {
		r := redacted(err)
		return &FetchError{Reason: r.text, cause: r.notRun, what: what, outcome: gapOutcome}
	}
	if !commitHere(root, tip) {
		return &FetchError{Reason: fmt.Sprintf("%s named %s as the branch's tip, and the fetch did not bring it (the branch moved while it was read)", Remote, tip[:12]),
			what: what, outcome: gapOutcome}
	}
	return nil
}

// commitHere reports whether the commit named tip is in this checkout's
// object store, read as stored.
func commitHere(root, tip string) bool {
	_, err := storedGit(root, 256, "cat-file", "-e", tip+"^{commit}")
	return err == nil
}

// repoMaxLive is the ceiling the repository sets at ref, origin's default
// branch as last fetched, over the bundled ten.
func repoMaxLive(root, ref string) (liveLimit, string, error) {
	lim := bundledLimit()
	if root == "" {
		return lim, "", nil
	}
	if ref == "" {
		return lim, fmt.Sprintf("this checkout names no default branch of %s as last fetched, so no repository figure for %s is read and %s applies",
			Remote, MaxLiveKey, lim.describe()), nil
	}
	name := termsafe.Sanitize(gitutil.ShortRef(ref))
	tip, err := storedGit(root, 256, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return liveLimit{}, "", refuse("%s at %s cannot be read for %s: %s does not name a commit here; %s",
			layered.Config.RepoOrigin(), name, MaxLiveKey, name, nothingPushed)
	}
	return repoMaxLiveAt(root, tip, name)
}

// repoMaxLiveAt is the ceiling the repository's file sets at the commit tip,
// named name ("origin/main") in every origin and refusal, over the bundled
// ten.
func repoMaxLiveAt(root, tip, name string) (liveLimit, string, error) {
	lim := bundledLimit()
	at := layered.Config.RepoOrigin() + " at " + name
	stack, err := layered.Load(layered.Config, layered.Roots{Repo: root, RepoAt: tip, RepoAtName: name, RepoAtAsStored: true, HomeRefused: true})
	if err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	// A person's own key is refused before the namespace is claimed, so the
	// refusal says why it may not be there rather than that nothing reads it.
	members, err := stack.Members(layered.Repo, holderNamespace)
	if err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	if slices.Contains(members, "github_username") {
		return liveLimit{}, "", refuse("%s (commit %s) sets %s, which names a person: a repository's file cannot say who you are, "+
			"and the key is read only from %s; remove it from the repository's file; %s",
			at, tip[:12], holderKey, layered.Config.MachineOrigin(), nothingPushed)
	}
	if err := stack.Claim(holderNamespace, repoReservationKeys...); err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	v, err := layered.Get(stack, MaxLiveKey, MaxLive, checkMaxLive)
	if err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	if v.Layer == layered.Repo {
		lim = liveLimit{N: v.V, Layer: LimitRepository, Origin: at}
	}
	return lim, "", nil
}

// machineMaxLive is lim lowered by this machine's own figure, never raised:
// a higher figure is ignored and named in the note. A machine file that is
// there but cannot be read as the person's own (not a regular file, writable
// by others, not valid JSON) refuses an act (act) naming the file, since
// reading past it could leave the figure above what the machine set; a view
// shows the figure so far and says the act will refuse. A file that is not
// there, a home that cannot be searched, and a HOME that is not this
// account's own read as no file.
func machineMaxLive(home string, lim liveLimit, act bool) (liveLimit, string, error) {
	if home == "" {
		return lim, "", nil
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		return lim, fmt.Sprintf("HOME lies inside a git working tree and is not this account's home, so %s is not read for %s and %s applies",
			layered.Config.MachineOrigin(), MaxLiveKey, lim.describe()), nil
	}
	if _, err := os.Lstat(abcdhome.Path(roots.Home, layered.Config.MachineRel)); err != nil {
		return lim, "", nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		if act {
			return liveLimit{}, "", refuse("%s cannot be read for %s, which may lower the ceiling on this machine (%s); fix or remove the file; %s",
				layered.Config.MachineOrigin(), MaxLiveKey, redacted(err), nothingPushed)
		}
		return lim, fmt.Sprintf("%s cannot be read for %s (%s), so an act that adds a reservation refuses until it can; %s is shown",
			layered.Config.MachineOrigin(), MaxLiveKey, redacted(err), lim.describe()), nil
	}
	if err := stack.Claim(holderNamespace, reservationKeys...); err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	v, err := layered.Get(stack, MaxLiveKey, 0, checkMaxLive)
	if err != nil {
		return liveLimit{}, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	switch {
	case v.Layer != layered.Machine || v.V == lim.N:
		return lim, "", nil
	case v.V > lim.N:
		return lim, fmt.Sprintf("%s sets %s to %d, above %s; a machine's figure may only lower the ceiling, so it is ignored",
			layered.Config.MachineOrigin(), MaxLiveKey, v.V, lim.describe()), nil
	}
	return liveLimit{N: v.V, Layer: LimitMachine, Origin: layered.Config.MachineOrigin()}, "", nil
}

// checkMaxLive holds a figure to 1 to MaxLiveCeiling.
func checkMaxLive(n int) error {
	if n < 1 || n > MaxLiveCeiling {
		return fmt.Errorf("want a whole number from 1 to %d", MaxLiveCeiling)
	}
	return nil
}

// MaxLiveFor is the cap of the person whose home o names (o.Home, else the
// process's own) in the checkout at root, for a caller that counts from a view
// it already holds (the drain's wait). It asks no network: it reads the
// repository's figure at the remote-tracking ref of the branch origin last
// named as its default when an act here asked it (keepOriginDefault). Where
// no act here has saved that name (errNoKeptDefault), it reads no figure,
// never one at the default branch this checkout tracks, which a pushed branch
// under a conventional name can stand in for. Whatever would refuse the act
// (the ref naming no commit here, an unusable figure, a machine file that
// cannot be read) is refused here too. On any error such a caller leaves the
// count to the act, which asks origin and decides. Beside the figure it
// returns the layer that set it (LimitBundled, LimitRepository or
// LimitMachine), which the drain's wait sentences name beside the count.
func MaxLiveFor(root string, o Options) (int, string, error) {
	b := keptOriginDefault(root)
	if b == "" {
		return 0, "", errNoKeptDefault
	}
	lim, _, err := repoMaxLive(root, "refs/remotes/"+Remote+"/"+b)
	if err != nil {
		return 0, "", err
	}
	lim, _, err = machineMaxLive(o.home(), lim, true)
	if err != nil {
		return 0, "", err
	}
	return lim.N, lim.Layer, nil
}

// errNoKeptDefault is MaxLiveFor's answer in a checkout where no act has
// saved the branch origin names as its default.
var errNoKeptDefault = errors.New("no act in this checkout has saved which branch " + Remote + " names as its default, so no figure is read and the act that adds a reservation counts")

// originDefaultRel keeps the branch origin last named as its default when an
// act asked it, for MaxLiveFor.
const originDefaultRel = stateRelDir + "/origin-default"

// keepOriginDefault records branch, origin's answer, for MaxLiveFor; a
// checkout without its local tier, or a write that fails, keeps nothing, and
// MaxLiveFor then reads no figure.
func keepOriginDefault(root, branch string) {
	if tierPresent(root) != nil {
		return
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	if mkStateDir(r, stateRelDir) != nil {
		return
	}
	_ = fsutil.WriteFileAtomicInRoot(r, originDefaultRel, []byte(branch+"\n"), stateMode)
}

// keptOriginDefault is the branch keepOriginDefault last recorded, "" when
// none reads as a valid branch name, or the file is tracked by git (a
// checkout's committed files never speak for origin).
func keptOriginDefault(root string) string {
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, originDefaultRel, 512)
	if err != nil || localtier.Untracked(root, originDefaultRel) != nil {
		return ""
	}
	b := strings.TrimSpace(string(data))
	if b == "" || b == "HEAD" {
		return ""
	}
	if _, err := localGit(root, 4096, "check-ref-format", "refs/heads/"+b); err != nil {
		return ""
	}
	return b
}
