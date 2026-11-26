package claim

import (
	"fmt"
	"slices"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gitutil"
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
//  2. reservation.max_live in the repository's .abcd/config.json, read at the
//     commit origin's default branch stood at when this checkout last fetched
//     it (decision 8), as the object store holds it (no replace ref followed),
//     and never from a working tree: a lane's agent can write its worktree and
//     its own branch, never origin's default branch. Where it is set it
//     replaces the ten. A checkout that names no default branch of origin
//     reads no repository figure, and the note says so.
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
// namespace, holder.go.) A machine config.json that cannot be read at all
// (writable by others, not valid JSON) is read past instead, as is a HOME
// inside a git working tree that is not the account's own
// (layered.MachineRoots): neither is a file the person can be shown to have
// written, and reading past it can only leave the figure higher than the
// machine meant, never higher than the repository's, so the figure so far
// applies and the note says why.

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

// maxLive resolves the ceiling for the checkout at root, whose origin's
// default branch the caller named as ref ("" for none), and the account whose
// home is home, with a note saying what was read past or ignored.
func maxLive(root, ref, home string) (liveLimit, string, error) {
	lim, note, err := repoMaxLive(root, ref)
	if err != nil {
		return liveLimit{}, "", err
	}
	lim, mnote, err := machineMaxLive(home, lim)
	if err != nil {
		return liveLimit{}, "", err
	}
	return lim, joinNote(note, mnote), nil
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
	at := layered.Config.RepoOrigin() + " at " + name
	tip, err := storedGit(root, 256, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return liveLimit{}, "", refuse("%s cannot be read for %s: %s does not name a commit here; %s", at, MaxLiveKey, name, nothingPushed)
	}
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
// a higher figure is ignored and named in the note.
func machineMaxLive(home string, lim liveLimit) (liveLimit, string, error) {
	if home == "" {
		return lim, "", nil
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		return lim, fmt.Sprintf("HOME lies inside a git working tree and is not this account's home, so %s is not read for %s and %s applies",
			layered.Config.MachineOrigin(), MaxLiveKey, lim.describe()), nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		return lim, fmt.Sprintf("the machine configuration could not be read for %s (%s), so %s applies",
			MaxLiveKey, redacted(err), lim.describe()), nil
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
// process's own) in the checkout at root, as an act that adds a reservation
// judges it: a caller that counts from a view it already holds (the drain's
// wait) compares with this. A setting the act would refuse is refused here the
// same way.
func MaxLiveFor(root string, o Options) (int, error) {
	lim, _, err := maxLive(root, originDefault(root), o.home())
	return lim.N, err
}
