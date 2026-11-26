package landing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The landing's configuration keys, in .abcd/config.json and
// ~/.abcd.noindex/config.json, each a bool whose bundled default is true.
const (
	KeyPush    = "land.push"
	KeyEnqueue = "land.enqueue"
)

// Switch is one of the landing's two outward acts as configured: whether it is
// on, and the layer and origin that decided it — the bundled default when no
// layer turned it off, else the highest layer that did.
type Switch struct {
	Key    string `json:"key"`
	On     bool   `json:"on"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
}

// readSwitches resolves land.push and land.enqueue through the layered
// configuration reader, which claims the land namespace so a misspelt key is
// refused naming its file.
//
// The precedence is deliberately not the reader's ordinary one. Ordinarily the
// repository's file beats the machine's; here OFF wins from any layer, so a
// person can switch an outward act off for every repository on the machine and
// no repository's file switches it back on (the rules loader's kill switch is
// the precedent). Every layer that holds a key is decoded, strictly, and the
// values are ANDed. noPush and noEnqueue are the --no-push and --no-enqueue
// flags, which add a flag-layer false; there is no flag that turns a key on
// against a layer's off.
//
// branchRepo, when not "", is the landed branch's own checkout, whose
// repository layer is read beside r's and ANDed too: an off committed on the
// branch counts wherever land is run from. A home RootsFor refused hides the
// machine layer, and with it any machine-wide off, so both acts read as off.
func readSwitches(r layered.Roots, branchRepo string, noPush, noEnqueue bool) (push, enqueue Switch, err error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return Switch{}, Switch{}, err
	}
	if noPush {
		if err := s.SetFlag(KeyPush, false, "--no-push"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	if noEnqueue {
		if err := s.SetFlag(KeyEnqueue, false, "--no-enqueue"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	stacks := []stack{{s: s}}
	if branchRepo != "" {
		// The branch's checkout is read for its repository layer alone: the
		// machine layer is r's and is already read.
		b, err := layered.Load(layered.Config, layered.Roots{Repo: branchRepo, HomeRefused: true})
		if err != nil {
			return Switch{}, Switch{}, err
		}
		stacks = append(stacks, stack{s: b, where: fsutil.RedactHome(branchRepo)})
	}
	for _, st := range stacks {
		if err := st.s.Claim("land", "push", "enqueue"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	if push, err = lookupSwitch(stacks, KeyPush, r.HomeRefused); err != nil {
		return Switch{}, Switch{}, err
	}
	if enqueue, err = lookupSwitch(stacks, KeyEnqueue, r.HomeRefused); err != nil {
		return Switch{}, Switch{}, err
	}
	return push, enqueue, nil
}

// stack is one loaded configuration, and the checkout it was read in when it
// is the landed branch's ("" for the working directory's own).
type stack struct {
	s     *layered.Stack
	where string
}

// homeRefusedOrigin says why a refused home decides a key for its
// restrictive value: what names the machine-wide setting that cannot be read.
func homeRefusedOrigin(what string) string {
	return "the home directory was refused (HOME lies inside a git working tree that is not this account's), so a machine-wide " + what + " cannot be read; run abcd with your own HOME"
}

// layerValue is one layer's decoded value for a key, and where it came from.
type layerValue[T any] struct {
	v      T
	layer  string
	origin string
}

// decodeLayers decodes key from every layer of every stack that holds it,
// highest first within a stack, strictly, and through check when check is
// non-nil. Every layer is decoded, not only the one that decides, so a bad
// value is refused wherever it sits, naming its file and layer and what the
// key takes.
func decodeLayers[T any](stacks []stack, key string, check func(T) error, takes string) ([]layerValue[T], error) {
	var out []layerValue[T]
	for _, st := range stacks {
		found, err := st.s.Lookup(key)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			origin := f.Origin
			if st.where != "" {
				origin += " in the branch's checkout " + st.where
			}
			v, err := layered.Decode[T](f.Raw)
			if err == nil && check != nil {
				err = check(v)
			}
			if err != nil {
				return nil, fmt.Errorf("%s (%s layer): %s: %v; it takes %s", origin, f.Layer, key, err, takes)
			}
			out = append(out, layerValue[T]{v: v, layer: f.Layer.String(), origin: origin})
		}
	}
	return out, nil
}

// lookupSwitch ANDs key's value across every layer of every stack that holds
// it, and a refused home's hidden machine layer. Lookup lists the layers
// highest first, so the first off it meets is the one reported.
func lookupSwitch(stacks []stack, key string, homeRefused bool) (Switch, error) {
	vals, err := decodeLayers[bool](stacks, key, nil, "true or false")
	if err != nil {
		return Switch{}, err
	}
	for _, v := range vals {
		if !v.v {
			return Switch{Key: key, On: false, Layer: v.layer, Origin: v.origin}, nil
		}
	}
	if homeRefused {
		return Switch{Key: key, On: false, Layer: layered.Machine.String(), Origin: homeRefusedOrigin("off")}, nil
	}
	return Switch{Key: key, On: true, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}, nil
}

// offText says which layer turned a switch off, as every step names it.
func (s Switch) offText() string {
	return fmt.Sprintf("%s is off (%s layer, %s)", s.Key, s.Layer, s.Origin)
}

// KeyFacilitatorMode declares who merges: the repository's technical
// facilitator is a person (ModeHuman, the bundled default) or abcd (ModeAuto).
// It says nothing about abcd mode's facilitator state, which is whose answer
// the loop is waiting on right now.
const (
	KeyFacilitatorMode = "facilitator.mode"
	ModeHuman          = "human"
	ModeAuto           = "auto"
)

// Facilitator is facilitator.mode as resolved: the mode, and the layer and
// origin that decided it. Layer is "bundled" (Origin "bundled") when no layer
// declares the key; otherwise Origin names the file that set it
// (".abcd/config.json", or the machine file in its tilde form), or, for a
// refused home, why the machine layer could not be read.
type Facilitator struct {
	Key    string `json:"key"`
	Mode   string `json:"mode"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
	// HomeRefused says a refused home decided the mode (human), so Origin is
	// the reason, not a file.
	HomeRefused bool `json:"home_refused,omitempty"`
}

// checkMode accepts the two modes, exactly as spelt.
func checkMode(m string) error {
	if m != ModeHuman && m != ModeAuto {
		return fmt.Errorf("%q is not a facilitator mode", layered.BoundKey(m))
	}
	return nil
}

// readMode resolves facilitator.mode through the layered configuration
// reader, which claims the facilitator namespace with its one key mode, so a
// misspelt key is refused naming its file.
//
// The precedence is land's switches' (readSwitches), not the reader's
// ordinary one: human from any layer wins, so a person can take the role back
// for every repository on their machine and no repository's file takes it from
// them. Every layer that holds the key is decoded and checked, so a value other
// than human or auto is refused in whichever layer it sits. With no human
// anywhere, the highest layer that says auto decides; with no layer at all,
// the bundled default human does. A home RootsFor refused hides the machine
// layer, and with it any machine-wide human, so the mode reads as human.
func readMode(r layered.Roots) (Facilitator, error) {
	return readModeContext(context.Background(), r)
}

// readModeContext is readMode with the git questions of a repository layer
// read at a commit bounded by ctx (layered.LoadContext).
func readModeContext(ctx context.Context, r layered.Roots) (Facilitator, error) {
	vals, _, err := modeLayers(ctx, r)
	if err != nil {
		return Facilitator{}, err
	}
	return resolveMode(vals, r.HomeRefused), nil
}

// modeLayers loads r's layers, claims the facilitator namespace and decodes
// facilitator.mode in every layer that holds it, highest first; a value that
// is neither mode, and a misspelt key, are refused. repoFile says whether the
// repository layer's file is there at all.
func modeLayers(ctx context.Context, r layered.Roots) (vals []layerValue[string], repoFile bool, err error) {
	s, err := layered.LoadContext(ctx, layered.Config, r)
	if err != nil {
		return nil, false, err
	}
	if err := s.Claim("facilitator", "mode"); err != nil {
		return nil, false, err
	}
	vals, err = decodeLayers([]stack{{s: s}}, KeyFacilitatorMode, checkMode, "human or auto")
	if err != nil {
		return nil, false, err
	}
	return vals, s.Present(layered.Repo), nil
}

// repoLayerMode is the repository layer's value among vals, "" when that
// layer does not set the key.
func repoLayerMode(vals []layerValue[string]) string {
	for _, v := range vals {
		if v.layer == layered.Repo.String() {
			return v.v
		}
	}
	return ""
}

// resolveMode is facilitator.mode's one precedence (readMode) over the layers'
// decoded values, highest first: every reader of the mode, the row's kept
// reading among them, answers through it.
func resolveMode(vals []layerValue[string], homeRefused bool) Facilitator {
	for _, v := range vals {
		if v.v == ModeHuman {
			return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: v.layer, Origin: v.origin}
		}
	}
	if homeRefused {
		return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: layered.Machine.String(), Origin: homeRefusedOrigin(ModeHuman), HomeRefused: true}
	}
	if len(vals) > 0 {
		return Facilitator{Key: KeyFacilitatorMode, Mode: ModeAuto, Layer: vals[0].layer, Origin: vals[0].origin}
	}
	return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
}

// readFacilitatorAt resolves facilitator.mode as the merge gate reads it: the
// repository layer is .abcd/config.json as the commit tip holds it (the
// default branch's tip, named name in every origin), read from the object
// store of the repository at repoRoot, beside r's machine layer, never the
// checkout of the branch being landed (dcn-2610230058333177). So a branch that
// commits auto, or deletes the default branch's human, does not decide the
// gate for itself. r's Repo is replaced; its Home and HomeRefused are kept.
// The error is the reader's raw text, for the caller to mask.
func readFacilitatorAt(r layered.Roots, repoRoot, tip, name string) (Facilitator, error) {
	r.Repo, r.RepoAt, r.RepoAtName = repoRoot, tip, name
	return readMode(r)
}

// CommittedMode is facilitator.mode as .abcd/config.json held it at commit, a
// full object name the repository at repoRoot holds, read from its object
// store as readFacilitatorAt reads the default branch's tip (bounded, a regular
// file only) and judged by checkMode: the repository layer alone, never the
// machine layer, since a reader of a past commit cannot read the machine as it
// was then. found says whether the commit holds the file at all; a file that
// sets no mode is human, the bundled default, and so is a commit without one.
// Its git questions are bound to ctx and read the commit's trees and the
// file's blob as stored (layered.Roots.RepoAtAsStored: no replace ref stands
// one in for another), as the landings reader, its one caller, reads the rest
// of history; the merge gate's own read (readFacilitatorAt) is unchanged. The
// error is the reader's raw text, for the caller to mask.
func CommittedMode(ctx context.Context, repoRoot, commit string) (mode string, found bool, err error) {
	// Refused here, not left to layered: an empty RepoAt there means the
	// working tree, so an empty name would read the working tree's mode as
	// if a commit held it.
	if !gitutil.IsFullSHA(commit) {
		return "", false, fmt.Errorf("%s is not a commit's full object name", clip(commit))
	}
	vals, found, err := modeLayers(ctx, layered.Roots{Repo: repoRoot, RepoAt: commit, RepoAtName: short(commit), RepoAtAsStored: true, HomeRefused: true})
	if err != nil {
		return "", false, err
	}
	if m := repoLayerMode(vals); m != "" {
		return m, found, nil
	}
	return ModeHuman, found, nil
}

// AutoAt reports whether the facilitator mode the merge gate reads at tip is auto.
func AutoAt(r layered.Roots, repoRoot, tip, name string) bool {
	m, err := readFacilitatorAt(r, repoRoot, tip, name)
	return err == nil && m.Mode == ModeAuto
}

// FetchedReadTimeout bounds ReadFetchedFacilitator as a whole: the shell
// guard asks it inside a host's hook, which may give up on a slow hook and
// run the command anyway, so a slow origin must end the read first, as human.
const FetchedReadTimeout = 15 * time.Second

// ReadFetchedFacilitator is readDefaultFacilitatorContext for a caller that judges
// an act by the mode and so must not take it from refs an agent can move
// (`git remote set-head`, `git update-ref` on a remote-tracking ref): the
// shell guard, deciding a merge command. It asks Remote which branch is its
// default and at which commit (git ls-remote --symref), fetches that branch,
// and reads the mode at the commit Remote named, from the local object store,
// all within FetchedReadTimeout. A remote that cannot be asked, a fetch that
// fails or runs out of time, and a tip the fetch did not bring are errors,
// which the caller counts as human: it never falls back to the checkout's
// last fetch. So is a checkout where the forge client would act on another
// remote than Remote (one named upstream or github, which gh prefers, or one
// gh repo set-default chose), whose mode this read would not be. The fetch
// writes what any fetch of the branch writes (its remote-tracking ref), never
// origin/HEAD and never FETCH_HEAD, and starts no maintenance. The network
// calls run as Net's do, with the person's own git configuration and
// credentials and never a prompt, so a private repository reads as it fetches.
//
// It trusts Remote's URL as the checkout's configuration names it: an agent
// that rewrites the URL is out of its reach, as the guard's limit says.
func ReadFetchedFacilitator(cwd string) (Facilitator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), FetchedReadTimeout)
	defer cancel()
	return readFetchedFacilitator(ctx, cwd)
}

func readFetchedFacilitator(ctx context.Context, cwd string) (Facilitator, error) {
	r, _ := layered.HomeRoots()
	fresh := func(what string, err error) error {
		why := ""
		if err != nil {
			why = ": " + fsutil.RedactHome(err.Error())
		}
		return fmt.Errorf("%s, so the default branch's %s cannot be read as just fetched%s", what, KeyFacilitatorMode, why)
	}
	root, err := gitutil.ToplevelContext(ctx, cwd)
	if err != nil {
		return Facilitator{}, fmt.Errorf("%s is not inside a git checkout, so the default branch's %s cannot be read", fsutil.RedactHome(cwd), KeyFacilitatorMode)
	}
	if other, err := forgeRemote(ctx, root); err != nil {
		return Facilitator{}, fresh("this checkout's remotes cannot be read", err)
	} else if other != "" {
		return Facilitator{}, fresh("the forge client would act on the remote "+other+" here, not on "+Remote, nil)
	}
	out, err := gitutil.NetContext(ctx, root, "ls-remote", "--symref", Remote, "HEAD")
	if err != nil {
		return Facilitator{}, fresh(Remote+" could not say which branch is its default", err)
	}
	def, tip := "", ""
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "ref: "); ok {
			if target, name, ok := strings.Cut(rest, "\t"); ok && name == "HEAD" {
				def, _ = strings.CutPrefix(target, "refs/heads/")
			}
		} else if sha, name, ok := strings.Cut(line, "\t"); ok && name == "HEAD" {
			tip = sha
		}
	}
	if def == "" || def == "HEAD" || !gitutil.IsFullSHA(tip) {
		return Facilitator{}, fresh(Remote+" did not name a default branch and its tip", nil)
	}
	if _, err := gitutil.RunLimitedContext(ctx, root, 4096, "check-ref-format", "refs/heads/"+def); err != nil {
		return Facilitator{}, fresh(Remote+"'s HEAD names a branch whose name is not a valid ref", nil)
	}
	if _, err := gitutil.NetContext(ctx, root, "fetch", "--quiet", "--no-tags", "--no-write-fetch-head", "--no-auto-maintenance", Remote, "refs/heads/"+def); err != nil {
		return Facilitator{}, fresh("fetching "+Remote+"'s "+def+" failed", err)
	}
	if _, err := gitutil.RunLimitedContext(ctx, root, 4096, "cat-file", "-e", tip+"^{commit}"); err != nil {
		return Facilitator{}, fresh(Remote+"'s "+def+" moved while it was fetched", nil)
	}
	return readFacilitatorAt(r, root, tip, Remote+"/"+def)
}

// forgeRemote names the remote the forge client would act on in the checkout
// at root when that is not Remote, "" when it is Remote: gh takes a remote
// gh repo set-default marked (remote.<name>.gh-resolved) first, then one
// named upstream, then github, then origin. A set-default on Remote that
// names another repository rather than "base" counts as another.
func forgeRemote(ctx context.Context, root string) (string, error) {
	out, err := gitutil.RunLimitedContext(ctx, root, 64<<10, "config", "--get-regexp", `^remote\..*\.(url|gh-resolved)$`)
	if err != nil && strings.TrimSpace(out) != "" {
		return "", err
	}
	urls := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		key, _, _ := strings.Cut(line, " ")
		name, ok := strings.CutPrefix(key, "remote.")
		if !ok {
			continue
		}
		if n, ok := strings.CutSuffix(name, ".gh-resolved"); ok {
			// gh repo set-default marks the remote it chose "base", or, on
			// a remote it does not act on as itself (a fork's parent),
			// names the repository it acts on instead.
			_, val, _ := strings.Cut(line, " ")
			if n != Remote || strings.TrimSpace(val) != "base" {
				return n + " (gh repo set-default)", nil
			}
		}
		if n, ok := strings.CutSuffix(name, ".url"); ok {
			urls[n] = true
		}
	}
	for _, n := range []string{"upstream", "github"} {
		if urls[n] {
			return n, nil
		}
	}
	return "", nil
}

// ErrNoDefaultBranch is readDefaultFacilitatorContext's answer for a repository with
// no default branch to read facilitator.mode at: one with no commit yet, say.
// It is not a refusal of anything a file declares, so a caller that only
// shows the mode (the status line) may stay quiet over it.
var ErrNoDefaultBranch = errors.New("the repository has no default branch to read " + KeyFacilitatorMode + " at")

// readDefaultFacilitatorContext resolves facilitator.mode for the repository
// holding cwd as the merge gate does (readFacilitatorAt), at the tip of the
// default branch git tracks for it (gitutil.DefaultRef), with the machine layer
// of the caller's home (layered.HomeRoots, as RootsFor resolves it). The board
// shows the mode through it (ShownFacilitator); the shell guard reads
// ReadFetchedFacilitator. A repository with no default branch to read is an
// error (ErrNoDefaultBranch): a caller that cannot read the mode takes it as
// human. Every git question is bounded by ctx: a git still running when ctx
// ends is killed and the read fails with ctx's error wrapped, never with
// ErrNoDefaultBranch. It asks
// the network nothing: the default branch is the one last fetched, and its
// configuration is read from the local object store. The board and the prompt
// hook read the mode through it; the status line's row reads what the hook
// kept of it (KeptMerging).
func readDefaultFacilitatorContext(ctx context.Context, cwd string) (Facilitator, error) {
	r, _, err := defaultTipRoots(ctx, cwd)
	if err != nil {
		return Facilitator{}, err
	}
	return readModeContext(ctx, r)
}

// defaultTipRoots is the Roots readDefaultFacilitatorContext reads the mode
// at, and the full name of the ref whose tip it names: the default branch git
// tracks for the repository holding cwd, with the machine layer of the
// caller's home.
func defaultTipRoots(ctx context.Context, cwd string) (r layered.Roots, ref string, err error) {
	r, _ = layered.HomeRoots()
	root, err := gitutil.ToplevelContext(ctx, cwd)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return layered.Roots{}, "", deadlineErr(ctxErr)
	}
	if errors.Is(err, gitutil.ErrGitTimedOut) {
		return layered.Roots{}, "", deadlineErr(err)
	}
	if err != nil {
		return layered.Roots{}, "", fmt.Errorf("%s is not inside a git checkout, so the default branch's %s cannot be read", fsutil.RedactHome(cwd), KeyFacilitatorMode)
	}
	ref = gitutil.DefaultRefContext(ctx, root)
	var tip string
	if ref != "" {
		tip, err = gitutil.RunLimitedContext(ctx, root, 4096, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
		tip = strings.TrimSpace(tip)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return layered.Roots{}, "", deadlineErr(ctxErr)
	}
	if ref == "" || err != nil || !gitutil.IsFullSHA(tip) {
		return layered.Roots{}, "", ErrNoDefaultBranch
	}
	r.Repo, r.RepoAt, r.RepoAtName = root, tip, gitutil.ShortRef(ref)
	return r, ref, nil
}

// deadlineErr is the read's error when git did not answer before ctx ended.
func deadlineErr(err error) error {
	return fmt.Errorf("git did not answer in time to read the default branch's %s: %w", KeyFacilitatorMode, err)
}

// AutoSessionLine is the one line the prompt hook gives a session on every
// prompt while facilitator.mode is auto (iss-2611211700231699), beside the
// away block and whether or not the person is away: what the setting means
// for a session, as the person ruled it (dcn-2611070734169709). It grants
// nothing an act did not already have: the question gate, the stop check and
// the guard's acts reserved to the person are unchanged.
const AutoSessionLine = "The technical facilitator is on auto (" + KeyFacilitatorMode + ", as the default branch and this machine set it): a technical question is decided and recorded with `abcd decide`, not asked, and finished work does not wait for a person to read it. An act a command page reserves to the person is still the person's: it waits for them and holds back nothing else, so carry on with the rest of the work."

// AutoForSessions reports whether facilitator.mode is auto as the board reads
// it (readDefaultFacilitatorContext): the default branch's committed
// .abcd/config.json as last fetched and the machine layer, never the working
// tree's copy, its git questions bounded by ctx and none of them asking the
// network, so a hook can ask it on every prompt. A repository with no
// default branch yet reads as not auto, with no error. The error is the
// reader's raw text, for the caller to mask; a caller takes it as human.
//
// It is also what keeps the reading the status line's row shows (keepMode):
// the prompt hook has the second that the row does not, and asks git this
// anyway. The kept reading never answers it.
func AutoForSessions(ctx context.Context, root string) (bool, error) {
	r, ref, err := defaultTipRoots(ctx, root)
	switch {
	case errors.Is(err, ErrNoDefaultBranch):
		return false, nil
	case err != nil:
		return false, err
	}
	vals, _, err := modeLayers(ctx, r)
	keepMode(ctx, r, ref, vals, err)
	if err != nil {
		return false, err
	}
	return resolveMode(vals, r.HomeRefused).Mode == ModeAuto, nil
}

// LiveMerging is the presence line's merging element on the board, read: the
// repository's facilitator.mode through the resolver the merge gate reads with
// (shownFacilitator), as the board's own line reads it, its git questions
// bounded by ctx and asking the network nothing. A default branch that cannot
// be told is not known and notes nothing; any other failure is not known, with
// the note saying why, masked, for the front door's stderr. The status line's
// row reads KeptMerging instead.
func LiveMerging(ctx context.Context, root string) (statusline.Merging, string) {
	f, err := shownFacilitator(ctx, root)
	switch {
	case errors.Is(err, ErrNoDefaultBranch):
		return statusline.Merging{State: statusline.MergingUnknown}, ""
	case err != nil:
		return statusline.Merging{State: statusline.MergingUnknown}, unreadNote(root, err)
	}
	return f.merging(), ""
}

// unreadNote says on the front door's stderr why the mode was not read, with
// the home and the checkout's own path masked; the element itself says
// `facilitator: not read`.
func unreadNote(root string, err error) string {
	msg := fsutil.RedactHome(err.Error())
	if root != "" {
		msg = strings.ReplaceAll(msg, root, ".")
	}
	return "statusline: " + KeyFacilitatorMode + " was not read (" + termsafe.CleanProseLine(msg, 300) + ")"
}

// merging is the mode in the row's own type, with whose setting it is. The
// row's package cannot import this one: landing reaches ahoy (through embargo,
// capture and site), and ahoy reads statusline, so the front door hands the
// readers over.
func (f Facilitator) merging() statusline.Merging {
	m := statusline.Merging{State: statusline.MergingHuman}
	if f.Mode == ModeAuto {
		m.State = statusline.MergingAuto
	}
	switch {
	case f.HomeRefused:
		m.Whose = statusline.WhoseComputerUnread
	case f.Layer == layered.Repo.String():
		m.Whose = statusline.WhoseProject
	case f.Layer == layered.Machine.String():
		m.Whose = statusline.WhoseComputer
	default:
		m.Whose = statusline.WhoseUnset
	}
	return m
}
