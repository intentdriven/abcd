package landing

// land.go is `abcd land [<branch>]` (iss-2610150524235929,
// iss-2610150524246898, with iss-2610090642376032's queue confirmation for
// this verb): one landing path for any branch, a build lane's or not. Each
// step is idempotent, so a re-run resumes where the last one stopped:
//
//  1. resolve: the branch (the checkout's current one, or the one named, in
//     the worktree that has it checked out) and its head; refused on the
//     default branch, a detached HEAD and a tree with uncommitted changes.
//     With land.push and land.enqueue both off nothing else runs.
//  2. origin: origin's every fetch and push URL must be the repository the
//     forge client resolves here, and the forge's default branch must be the
//     one origin/HEAD names here and not the branch; checked before the gate,
//     so a landing the forge refuses never runs the preflight.
//  3. landed: origin's default branch is fetched, and a head already on it
//     has landed; so has a head the forge lists a merged pull request from
//     this repository's branch into the default branch at (a squash or
//     rebase merge leaves no ancestry). Either way the verb says so and stops,
//     pushing, gating and queueing nothing (a re-run after the merge never
//     recreates the deleted branch), and a forge that cannot be asked, or a
//     listing its bound filled with none at the head, gets no push.
//  4. gate: skipped when a preflight receipt already names the head (in any
//     worktree, as the pre-push hook looks); otherwise `make preflight` runs in
//     the branch's checkout (it waits on the machine load waiter itself), and
//     HEAD must be unchanged and a receipt present after it, and origin's
//     URLs as they were before it (the branch's preflight could rewrite them).
//  5. push (land.push): a plain `git push --porcelain origin
//     <head>:refs/heads/B` from the branch's checkout, so the commit pushed is
//     the one the gate checked whatever the branch names by then, the
//     repository's hooks running, never forced, never the default branch.
//     Off, it prints the command and names the layer that turned it off.
//  6. enqueue (land.enqueue): the open pull request for the branch, armed by
//     the merge gate (mergeGate), pinned to the head, and confirmed in the
//     merge queue (PRStateQuery, asked of the repository the forge client
//     named when origin was checked); with no pull request it prints the
//     `gh pr create` command and stops. It never opens a pull request, never overrides an arming
//     someone else made, and never edits the repository's settings.
//
// The trust boundaries: the remote is `origin` only, and its every fetch and
// push URL must be the repository the forge client resolves here; a pull
// request from another repository is never armed; the merge rule is read from
// the ruleset mirror at origin's default branch as just fetched, never at the
// branch's tip, so the branch cannot change the rule it lands by; the forge
// client's and git's environments are scrubbed of GIT_DIR and GIT_CONFIG_*; a
// credential in a URL git or the forge client quotes never reaches land's text.
// The receipt is a convenience check, not a security boundary: CI re-runs
// every gate on the pushed commit. The preflight is the branch's own `make
// preflight`, run with the person's environment, so land is for branches the
// person trusts.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Remote is the one remote a branch lands through.
const Remote = "origin"

// The statuses a step reports.
const (
	// StatusDone: the step did its act.
	StatusDone = "done"
	// StatusSkipped: the step's act was already done (a receipt names HEAD).
	StatusSkipped = "skipped"
	// StatusOff: a configuration key or a flag turned the act off.
	StatusOff = "off"
	// StatusStopped: the step cannot act yet; its note says what to do, and
	// running abcd land again resumes it.
	StatusStopped = "stopped"
	// StatusLeft: the act is a person's, and land leaves it to them.
	StatusLeft = "left"
	// StatusLanded: the branch's head is already on the default branch.
	StatusLanded = "landed"
)

// gateTimeout bounds one preflight run: the load waiter it starts with gives
// up after two hours, and the gates after it take well under one. A var so a
// test can shorten it.
var gateTimeout = 4 * time.Hour

// gateWaitDelay bounds the wait for the preflight's output once it has exited
// or been killed (procgroup.Bound).
const gateWaitDelay = 10 * time.Second

// Options are one landing's inputs and its seams. A nil seam takes the real
// one: Forge is GH, Git is gitutil.Net (the networked git runner, hooks
// running), Gate runs `make preflight`.
type Options struct {
	// Branch is the branch to land; "" is the checkout's current branch.
	Branch string
	// NoPush and NoEnqueue are --no-push and --no-enqueue.
	NoPush    bool
	NoEnqueue bool
	// Roots are where land.push and land.enqueue are read: the working
	// directory's repository layer and the machine layer. The landed branch's
	// own checkout is read as a repository layer beside them.
	Roots layered.Roots
	// Forge runs the forge client with args from dir.
	Forge func(dir string, args ...string) (string, error)
	// Git runs one networked git command (a push or a fetch) from dir.
	Git GitFunc
	// Gate runs the preflight in the branch's checkout at dir.
	Gate func(dir string) error
	// GateOutput receives the preflight's output; nil discards it.
	GateOutput io.Writer
}

// Step is what one step of the landing did.
type Step struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Note    string `json:"note"`
	Command string `json:"command,omitempty"`
}

// Result is the landing's report, filled as far as it got: a refused landing
// returns the steps it completed beside the refusal.
type Result struct {
	Branch   string `json:"branch"`
	Head     string `json:"head"`
	Checkout string `json:"checkout"`
	Default  string `json:"default_branch"`
	Push     Switch `json:"push"`
	Enqueue  Switch `json:"enqueue"`
	PR       int    `json:"pull_request,omitempty"`
	PRURL    string `json:"pull_request_url,omitempty"`
	// Pushed is the commit the push step sent, the head the gate checked.
	Pushed string `json:"pushed,omitempty"`
	// Landed is set when the head was already on the default branch, or the
	// forge lists PR as merged at it.
	Landed bool   `json:"landed,omitempty"`
	Steps  []Step `json:"steps"`
}

func (r *Result) add(name, status, note, command string) {
	r.Steps = append(r.Steps, Step{Name: name, Status: status, Note: clean(note), Command: clean(command)})
}

// clean drops any credential from a URL in text git or the forge client
// supplied (a remote URL carrying a token), before land reports it.
func clean(s string) string { return gitutil.ScrubURLCredentials(s) }

// Refusal is a landing refused: Reason says why, Fix what settles it.
type Refusal struct {
	Reason string
	Fix    string
}

func (r *Refusal) Error() string {
	if r.Fix == "" {
		return r.Reason
	}
	return r.Reason + "; " + r.Fix
}

func refuse(reason, fix string) error { return &Refusal{Reason: clean(reason), Fix: clean(fix)} }

func (o Options) withDefaults() Options {
	if o.Forge == nil {
		o.Forge = GH
	}
	if o.Git == nil {
		o.Git = gitutil.Net
	}
	if o.GateOutput == nil {
		o.GateOutput = io.Discard
	}
	if o.Gate == nil {
		o.Gate = makePreflight(o.GateOutput)
	}
	return o
}

// Land lands the branch from the checkout at cwd.
func Land(cwd string, o Options) (Result, error) {
	o = o.withDefaults()
	var res Result
	root, err := gitutil.Toplevel(cwd)
	if err != nil {
		return res, refuse("the working directory is not inside a git checkout", "run abcd land from the branch's checkout")
	}
	dir, branch, err := resolveBranch(root, o.Branch)
	if err != nil {
		return res, err
	}
	res.Branch, res.Checkout = branch, fsutil.RedactHome(dir)
	ref := gitutil.DefaultRef(dir)
	def, ok := strings.CutPrefix(ref, "refs/remotes/"+Remote+"/")
	if !ok || def == "" || def == "HEAD" {
		return res, refuse("the repository has no default branch on the remote "+Remote+" to land on",
			"add the remote and fetch it (`git fetch "+Remote+"`), then run abcd land again")
	}
	res.Default = def
	if branch == def {
		return res, refuse(branch+" is the default branch, and land never pushes it",
			"land a branch of your own and merge it through a pull request")
	}
	head, err := headOf(dir)
	if err != nil {
		return res, err
	}
	res.Head = head
	if dirty, err := dirtyTree(dir); err != nil {
		return res, err
	} else if dirty {
		return res, refuse(fsutil.RedactHome(dir)+" holds uncommitted changes (staged, unstaged or untracked), and the preflight vouches only for a clean tree",
			"commit or remove them, then run abcd land again; nothing was pushed")
	}
	if res.Push, res.Enqueue, err = readSwitches(o.Roots, branchRepoRoot(o.Roots, dir), o.NoPush, o.NoEnqueue); err != nil {
		return res, refuse("the land configuration cannot be read: "+fsutil.RedactHome(err.Error()), "correct the file it names; nothing was pushed")
	}

	if !res.Push.On && !res.Enqueue.On {
		res.add("gate", StatusSkipped, "land.push and land.enqueue are both off, so nothing would land: the preflight did not run", "")
		addPushOff(&res)
		res.add("enqueue", StatusOff, res.Enqueue.offText()+": nothing was armed or queued", "")
		return res, nil
	}
	origin, err := checkOrigin(o, dir, branch, def)
	if err != nil {
		return res, err
	}
	tracking, on, err := Landed(o.Git, dir, def, head)
	var fe *FetchError
	switch {
	case errors.As(err, &fe):
		return res, refuse(fsutil.RedactHome(fe.Error()), "settle what git reports, then run abcd land again; nothing was pushed")
	case err != nil:
		return res, fmt.Errorf("placing %s on %s: %w", short(head), tracking, err)
	case on:
		res.Landed = true
		res.add("land", StatusLanded, short(head)+" is already on "+Remote+"/"+def+", so "+branch+" has landed: nothing was gated, pushed or queued", "")
		return res, nil
	}
	// A squash or rebase merge puts another commit on the default branch, so
	// the ancestor check says no: the forge's merged pull request at this head
	// is the proof, and a forge that cannot answer gets no push.
	pr, prURL, err := mergedAt(func(args ...string) (string, error) { return o.Forge(dir, args...) }, branch, def, head)
	var full *MergedListingFullError
	switch {
	case errors.Is(err, ErrNoForgeClient):
		return res, noForgeClient()
	case errors.As(err, &full):
		return res, refuse(full.Error()+", so land cannot prove a push would not recreate merged work",
			"if "+branch+" has not merged, push it yourself (git push "+Remote+" "+shellQuote("refs/heads/"+branch+":refs/heads/"+branch)+") or land it under a less common name; nothing was gated or pushed")
	case err != nil:
		return res, refuse("the forge client could not say whether a pull request for "+branch+" already merged at "+short(head)+" (a squash or rebase merge leaves it off "+Remote+"/"+def+"), so land cannot prove a push would not recreate merged work: "+fsutil.RedactHome(err.Error()),
			"settle what the forge client reports (gh auth status), then run abcd land again; nothing was gated or pushed")
	case pr > 0:
		res.Landed, res.PR, res.PRURL = true, pr, prURL
		res.add("land", StatusLanded, fmt.Sprintf("pull request #%d merged %s into %s (by a squash or rebase, so the commit itself is not on %s/%s), so %s has landed: nothing was gated, pushed or queued", pr, short(head), def, Remote, def, branch), "")
		return res, nil
	}
	if err := gate(&res, o, dir, head); err != nil {
		return res, err
	}
	// The preflight is the branch's own and runs with the person's
	// environment: it could rewrite origin's URLs (a pushurl, a pushInsteadOf)
	// after they were checked, so they are read again before the push.
	if err := origin.unchanged(dir); err != nil {
		return res, err
	}
	if err := push(&res, o, dir); err != nil {
		return res, err
	}
	return res, enqueue(&res, o, dir, tracking, origin.repo.NameWithOwner)
}

// branchRepoRoot is the repository layer's root for the landed branch's own
// checkout at dir, or "" when it is the working directory's own, so its
// .abcd/config.json is read beside the working directory's.
func branchRepoRoot(r layered.Roots, dir string) string {
	b, _ := layered.RootsFor(dir)
	if b.Repo == "" || fsutil.RealExistingPath(b.Repo) == fsutil.RealExistingPath(r.Repo) {
		return ""
	}
	return b.Repo
}

// resolveBranch finds the checkout to land from: the one at root on its
// current branch, or the worktree that has branch checked out.
func resolveBranch(root, branch string) (dir, name string, err error) {
	wts, err := gitutil.ListWorktrees(root, gitutil.MaxWorktreeListing)
	if err != nil {
		return "", "", fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	if branch == "" {
		here := fsutil.RealExistingPath(root)
		for _, wt := range wts {
			if wt.Bare || fsutil.RealExistingPath(wt.Path) != here {
				continue
			}
			name, ok := strings.CutPrefix(wt.Branch, "refs/heads/")
			if !ok || name == "" {
				return "", "", refuse("HEAD is detached, so there is no branch to land",
					"check out the branch to land (or name it: abcd land <branch>), then run abcd land again")
			}
			return wt.Path, name, nil
		}
		return "", "", refuse("git lists no worktree at this checkout", "run abcd land from the branch's checkout")
	}
	if !gitutil.RefIsSafe(branch) {
		return "", "", refuse("the branch name is not one git takes", "name the branch as git lists it (git branch)")
	}
	if _, err := gitutil.Run(root, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", "", refuse("the branch name is not one git takes", "name the branch as git lists it (git branch)")
	}
	for _, wt := range wts {
		if !wt.Bare && wt.Branch == "refs/heads/"+branch {
			return wt.Path, branch, nil
		}
	}
	return "", "", refuse(branch+" is not checked out in any worktree, and the preflight runs in the branch's own checkout",
		"check it out (git switch "+branch+", or git worktree add), then run abcd land again")
}

func headOf(dir string) (string, error) {
	head, err := gitutil.Run(dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(head) {
		return "", refuse("the checkout's HEAD cannot be read", "commit the branch's work, then run abcd land again")
	}
	return head, nil
}

// dirtyTree reports whether the checkout at dir holds a staged, unstaged or
// untracked change, read as the person's own git reads it (their ignore files
// included), as the preflight's receipt reads it.
func dirtyTree(dir string) (bool, error) {
	out, err := gitutil.WorkingTreeCommand(dir, "status", "--porcelain", "--untracked-files=normal").Output()
	if err != nil {
		return false, fmt.Errorf("reading the checkout's status: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// gate is step 4.
func gate(res *Result, o Options, dir, head string) error {
	rcp, err := Receipt(dir, head)
	if err != nil {
		return err
	}
	if rcp != "" {
		res.add("gate", StatusSkipped, "a preflight receipt already names "+short(head)+" ("+fsutil.RedactHome(rcp)+")", "")
		return nil
	}
	if err := o.Gate(dir); err != nil {
		var r *Refusal
		if errors.As(err, &r) {
			return err
		}
		return refuse("the preflight did not pass on "+short(head)+": "+fsutil.RedactHome(err.Error()),
			"settle what it reports, then run abcd land again; nothing was pushed")
	}
	now, err := headOf(dir)
	if err != nil {
		return err
	}
	if now != head {
		return refuse(fmt.Sprintf("HEAD moved from %s to %s while the preflight ran, so its receipt does not vouch for what would be pushed", short(head), short(now)),
			"run abcd land again to gate the new head; nothing was pushed")
	}
	if rcp, err = Receipt(dir, head); err != nil {
		return err
	}
	if rcp == "" {
		return refuse("the preflight finished but no preflight receipt names "+short(head)+" (it mints one only on a tree that matched HEAD throughout)",
			"keep the tree clean while it runs, then run abcd land again; nothing was pushed")
	}
	res.add("gate", StatusDone, "the preflight passed and its receipt names "+short(head), "")
	return nil
}

// makePreflight is the real gate: `make preflight` in the branch's checkout,
// run directly (it waits on the machine load waiter itself, so it is never
// wrapped in another), its output to out. A repository whose make file
// declares no preflight target has no gate that mints a receipt, and gets no
// push.
func makePreflight(out io.Writer) func(dir string) error {
	return func(dir string) error {
		if !hasPreflightTarget(dir) {
			return refuse("the repository has no preflight gate that mints a receipt (its make file declares no preflight target), so land pushes nothing",
				"add a preflight target that mints "+ReceiptsRelDir+"/<commit> on a clean tree, or push the branch yourself")
		}
		bin, err := exec.LookPath("make")
		if err != nil {
			return refuse("make is not installed, and the preflight runs through it", "install make, then run abcd land again; nothing was pushed")
		}
		// The preflight leads its own process group, so the bound, or an
		// interrupt at the terminal (which a group of its own does not
		// receive), stops every process it started, not make alone.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		ctx, cancel := context.WithTimeout(ctx, gateTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "preflight")
		cmd.Dir = dir
		cmd.Env = gitutil.ScrubbedEnv()
		cmd.Stdout, cmd.Stderr = out, out
		procgroup.Bound(cmd, gateWaitDelay)
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("make preflight was stopped (%v), and its process group with it", context.Cause(ctx))
			}
			return fmt.Errorf("make preflight: %v", err)
		}
		return nil
	}
}

// preflightTargetRe is a make rule line naming the preflight target.
var preflightTargetRe = regexp.MustCompile(`(?m)^preflight[ \t]*:([^=]|$)`)

// hasPreflightTarget reports whether the make file make reads in dir (the
// first of GNUmakefile, makefile, Makefile) declares a preflight target.
func hasPreflightTarget(dir string) bool {
	for _, name := range []string{"GNUmakefile", "makefile", "Makefile"} {
		raw, err := fsutil.ReadGuarded(filepath.Join(dir, name), 1<<20)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			// A make file that cannot be read safely (a symlink, a device, one
			// past the cap) is no gate land can vouch for.
			return false
		}
		return preflightTargetRe.Match(raw)
	}
	return false
}

// forgeRepo is the forge client's view of the repository.
type forgeRepo struct {
	NameWithOwner    string `json:"nameWithOwner"`
	URL              string `json:"url"`
	SSHURL           string `json:"sshUrl"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

// checkOrigin refuses unless every fetch and push URL of origin is the
// repository the forge client resolves here, so the push and the pull request
// land in the same place, and unless the forge's default branch is def, the
// one origin/HEAD names here, and is not branch: a local origin/HEAD can be
// stale (the forge's default renamed after the clone), and the forge is the
// authority on which branch land must never push. One forge call asks both.
func checkOrigin(o Options, dir, branch, def string) (originURLs, error) {
	out, err := o.Forge(dir, "repo", "view", "--json", "nameWithOwner,url,sshUrl,defaultBranchRef")
	if err != nil {
		if errors.Is(err, ErrNoForgeClient) {
			return originURLs{}, noForgeClient()
		}
		return originURLs{}, refuse("the forge client could not say which repository origin is and which branch is its default, so land cannot prove the push is safe: "+fsutil.RedactHome(err.Error()),
			"settle what the forge client reports (gh auth status), then run abcd land again; nothing was gated or pushed")
	}
	var repo forgeRepo
	if json.Unmarshal([]byte(out), &repo) != nil {
		return originURLs{}, refuse("the forge client's view of the repository does not parse", "check the forge client (gh --version), then run abcd land again; nothing was pushed")
	}
	keys := map[string]bool{}
	for _, u := range []string{repo.URL, repo.SSHURL} {
		if k := repoKey(u); k != "" {
			keys[k] = true
		}
	}
	urls, err := readOriginURLs(dir, repo, keys)
	if err != nil {
		return originURLs{}, err
	}
	fdef := repo.DefaultBranchRef.Name
	switch {
	case fdef == "":
		return urls, refuse("the forge client reports no default branch for the repository, so land cannot prove "+branch+" is not it",
			"check the forge client (gh repo view), then run abcd land again; nothing was gated or pushed")
	case fdef == branch:
		return urls, refuse(branch+" is the forge's default branch, and land never pushes the default branch",
			"land a branch of your own and merge it through a pull request; refresh the local default with git remote set-head "+Remote+" --auto")
	case fdef != def:
		return urls, refuse("the forge's default branch is "+fdef+", but "+Remote+"/HEAD here names "+def+" (a stale or missing origin/HEAD), so land cannot trust which branch is the default",
			"refresh it (git fetch "+Remote+" && git remote set-head "+Remote+" --auto), then run abcd land again; nothing was gated or pushed")
	}
	return urls, nil
}

// originURLs are origin's fetch and push URLs as checked against the forge's
// repository, insteadOf and pushInsteadOf rewrites applied.
type originURLs struct {
	repo        forgeRepo
	keys        map[string]bool
	fetch, push []string
}

// readOriginURLs reads origin's every fetch and push URL locally (git applies
// the person's insteadOf and pushInsteadOf rules) and refuses unless each is
// the forge's repository, one of keys.
func readOriginURLs(dir string, repo forgeRepo, keys map[string]bool) (originURLs, error) {
	got := originURLs{repo: repo, keys: keys}
	for _, push := range []bool{false, true} {
		args := []string{"remote", "get-url", "--all"}
		what := "fetch URL"
		if push {
			args, what = append(args, "--push"), "push URL"
		}
		out, err := gitutil.LocalGit(dir, append(args, Remote)...)
		urls := strings.Fields(out)
		if err != nil || len(urls) == 0 {
			return got, refuse("the repository has no remote named "+Remote+", the one remote land pushes to", "add it, then run abcd land again; nothing was pushed")
		}
		if push {
			got.push = urls
		} else {
			got.fetch = urls
		}
		for _, u := range urls {
			k := repoKey(u)
			if k != "" && keys[k] {
				continue
			}
			return got, otherOrigin(u, k, what, push, repo, keys)
		}
	}
	return got, nil
}

// unchanged reads origin's URLs again, against the forge's repository as
// checked before, and refuses when any differs from what was checked.
func (v originURLs) unchanged(dir string) error {
	now, err := readOriginURLs(dir, v.repo, v.keys)
	if err == nil && slices.Equal(now.fetch, v.fetch) && slices.Equal(now.push, v.push) {
		return nil
	}
	why := "they no longer match what was checked"
	if r := (*Refusal)(nil); errors.As(err, &r) {
		why = r.Reason
	}
	return refuse(Remote+"'s URLs changed while the preflight ran ("+why+"), so land cannot prove the push reaches the forge's repository",
		"check what changed "+Remote+"'s URLs (git remote get-url --all --push "+Remote+"; git config --get-regexp '^url\\.'), then run abcd land again; nothing was pushed")
}

// otherOrigin is the refusal of origin's URL u (key k), which is not the
// forge's repository: an ssh host alias is named as one, with the insteadOf
// rule that maps it in u's own form, and a URL on another real host is said
// to be on another host, with only the set-url that points origin back.
func otherOrigin(u, k, what string, push bool, repo forgeRepo, keys map[string]bool) error {
	setURL := "git remote set-url "
	if push {
		setURL += "--push "
	}
	setURL += Remote + " " + shellQuote(repo.SSHURL)
	alias, real := sshAlias(u, k, keys)
	switch {
	case alias != "" && looksLikeSSHAlias(alias):
		from, to := sshPrefixes(u, real)
		return refuse(fmt.Sprintf("%s's %s names the host %s, but the forge's repository is on %s: %s looks like an SSH host alias (a Host entry in ssh config), which land does not resolve, so it cannot prove the push reaches the forge's repository",
			Remote, what, alias, real, alias),
			fmt.Sprintf("point %s at the forge's own host (%s, choosing the key with core.sshCommand), or map the alias for git with an insteadOf rule (git config url.%s.insteadOf %s), then run abcd land again; nothing was pushed",
				Remote, setURL, shellQuote(to), shellQuote(from)))
	case alias != "":
		return refuse(fmt.Sprintf("%s's %s (%s) is on the host %s, but the forge's repository is on %s, a different host", Remote, what, display(k), alias, real),
			fmt.Sprintf("point %s at the forge's repository (%s), or the forge client at %s (gh repo set-default), then run abcd land again; nothing was pushed", Remote, setURL, Remote))
	}
	return refuse(fmt.Sprintf("%s's %s (%s) is not the repository the forge client resolves here (%s)", Remote, what, display(k), display(repoKey(repo.URL))),
		"point every "+Remote+" URL at the forge's repository, or the forge client at "+Remote+" (gh repo set-default), then run abcd land again; nothing was pushed")
}

// sshAlias reports, for an origin URL u in ssh form (scp-like or ssh://)
// whose key k names a path one of the forge's keys names on another host, the
// host u names and the forge's: the shape of an ssh config host alias.
func sshAlias(u, k string, keys map[string]bool) (alias, real string) {
	scp := !strings.Contains(u, "://") && strings.Contains(u, ":")
	if !scp && !strings.HasPrefix(strings.ToLower(u), "ssh://") {
		return "", ""
	}
	host, path := splitKey(k)
	if host == "" {
		return "", ""
	}
	for fk := range keys {
		if fh, fp := splitKey(fk); fp == path && fh != "" && fh != host {
			return host, fh
		}
	}
	return "", ""
}

// sshAliasRe is a host as an ssh config Host alias names it: one label with
// no dot (so no public forge's host, and no IP address), with any port.
var sshAliasRe = regexp.MustCompile(`^[A-Za-z0-9_-]+(:[0-9]+)?$`)

// looksLikeSSHAlias reports whether host (with any port) is clearly an ssh
// config alias rather than a real host: a single label that is not localhost.
func looksLikeSSHAlias(host string) bool {
	name, _, _ := strings.Cut(host, ":")
	return sshAliasRe.MatchString(host) && !strings.EqualFold(name, "localhost")
}

// sshPrefixes are the insteadOf rule's two sides for the ssh URL u, in u's own
// form (scp-like or scheme): from is u's prefix up to its path as written, so
// the rule matches u; to is the same prefix on the host real, with no port,
// and any password dropped. "git@github-work:" maps to "git@github.com:", and
// "ssh://git@github-work:2222/" to "ssh://git@github.com/".
func sshPrefixes(u, real string) (from, to string) {
	if i := strings.Index(u, "://"); i >= 0 {
		scheme, rest := u[:i+3], u[i+3:]
		auth, _, _ := strings.Cut(rest, "/")
		user := ""
		if at := strings.LastIndexByte(auth, '@'); at >= 0 {
			user = auth[:at+1]
		}
		if name, _, hasPass := strings.Cut(user, ":"); hasPass {
			user = name + "@"
		}
		return scheme + auth + "/", scheme + user + real + "/"
	}
	c := strings.Index(u, ":")
	user := ""
	if at := strings.Index(u[:c], "@"); at >= 0 {
		user = u[:at+1]
	}
	if name, _, hasPass := strings.Cut(user, ":"); hasPass {
		user = name + "@"
	}
	return u[:c+1], user + real + ":"
}

func display(k string) string {
	if k == "" {
		return "unreadable"
	}
	return strings.TrimPrefix(k, ":")
}

// defaultPorts are the ports a URL scheme takes when it names none.
var defaultPorts = map[string]string{"ssh": "22", "git+ssh": "22", "https": "443", "http": "80", "git": "9418"}

// repoKey is a repository URL's comparable form, "host[:port]:/path" with the
// host and (on a host) the path lower-cased, any user, trailing slash and .git
// suffix dropped, and a port kept unless it is the scheme's own; a local path
// or file URL has an empty host. "" when the URL cannot be read.
func repoKey(u string) string {
	u = strings.TrimSpace(u)
	var host, path string
	switch c := strings.Index(u, ":"); {
	case u == "":
		return ""
	case strings.Contains(u, "://"):
		p, err := url.Parse(u)
		if err != nil {
			return ""
		}
		host, path = strings.ToLower(p.Hostname()), p.Path
		if port := p.Port(); port != "" && port != defaultPorts[strings.ToLower(p.Scheme)] {
			host += ":" + port
		}
		if p.Scheme == "file" {
			host = ""
		}
	case c > 1 && !strings.Contains(u[:c], "/"):
		h := u[:c]
		if at := strings.LastIndex(h, "@"); at >= 0 {
			h = h[at+1:]
		}
		host, path = strings.ToLower(h), "/"+strings.TrimPrefix(u[c+1:], "/")
	default:
		path = u
	}
	if host == "" {
		path = filepath.Clean(path)
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	path = strings.TrimSuffix(path, "/")
	if host != "" {
		path = strings.ToLower(path)
	}
	if path == "" || path == "/" || path == "." {
		return ""
	}
	return host + ":" + path
}

// splitKey splits a repoKey into its host (with any port) and its path.
func splitKey(k string) (host, path string) {
	if p, ok := strings.CutPrefix(k, ":"); ok {
		return "", p
	}
	if i := strings.Index(k, ":/"); i >= 0 {
		return k[:i], k[i+1:]
	}
	return "", k
}

// noForgeClient is the refusal when the forge client is not installed.
func noForgeClient() error {
	return refuse("the forge client `gh` is not installed, and land reads the repository and its pull request through it",
		"install gh and sign in with your own identity (gh auth login), or turn land.push and land.enqueue off")
}

// forge runs the forge client through the seam, its failures as refusals.
func forge(o Options, dir string, args ...string) (string, error) {
	out, err := o.Forge(dir, args...)
	if errors.Is(err, ErrNoForgeClient) {
		return "", noForgeClient()
	}
	if err != nil {
		return "", refuse(fsutil.RedactHome(err.Error()), "settle what the forge client reports, then run abcd land again")
	}
	return out, nil
}

// pushArgs is the push, plain: no force, no hook skipped, the commit head
// (the one the gate checked) to the branch's name on origin. A sha, not the
// branch's ref, so a branch that moves after the gate cannot change what is
// pushed.
func pushArgs(head, branch string) []string {
	return []string{"push", "--porcelain", Remote, head + ":refs/heads/" + branch}
}

// addPushOff is the push step when land.push is off.
func addPushOff(res *Result) {
	ref := "refs/heads/" + res.Branch
	res.add("push", StatusOff, res.Push.offText()+": nothing was pushed; push it yourself with the command", "git push "+Remote+" "+shellQuote(ref+":"+ref))
}

// maxPushReport bounds how much of a failed push's porcelain report its
// refusal carries.
const maxPushReport = 2048

// push is step 5.
func push(res *Result, o Options, dir string) error {
	if !res.Push.On {
		addPushOff(res)
		return nil
	}
	out, err := o.Git(dir, pushArgs(res.Head, res.Branch)...)
	if err != nil {
		reason := "git could not push " + res.Branch + " at " + short(res.Head) + " to " + Remote + ": " + fsutil.RedactHome(err.Error())
		if report := pushReport(out); report != "" {
			reason += "; git reports: " + report
		}
		return refuse(reason, "settle what git or the pre-push hook reports, then run abcd land again; land never forces a push or skips a hook")
	}
	res.Pushed = res.Head
	note := "pushed " + res.Branch + " at " + short(res.Head) + " to " + Remote
	if strings.Contains(out, "\n=\t") || strings.HasPrefix(out, "=\t") {
		note = Remote + " already held " + res.Branch + " at " + short(res.Head) + "; the push changed nothing"
	}
	res.add("push", StatusDone, note, "")
	return nil
}

// pushReport is a failed porcelain push's per-ref lines (git's own reasons
// for each rejection), one line, bounded and free of control characters.
func pushReport(out string) string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" && l != "Done" && !strings.HasPrefix(l, "To ") {
			lines = append(lines, strings.Join(strings.Fields(l), " "))
		}
	}
	r := termsafe.Sanitize(fsutil.RedactHome(strings.Join(lines, "; ")))
	if len(r) > maxPushReport {
		r = r[:maxPushReport]
	}
	return r
}

// openPR is the branch's open pull request as the forge lists it.
type openPR struct {
	Number            int    `json:"number"`
	URL               string `json:"url"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
}

// prState is what the forge reports of a pull request's way to the merge.
type prState struct {
	Mergeable         string          `json:"mergeable"`
	MergeStateStatus  string          `json:"mergeStateStatus"`
	IsInMergeQueue    bool            `json:"isInMergeQueue"`
	AutoMergeRequest  json.RawMessage `json:"autoMergeRequest"`
	HeadRefOid        string          `json:"headRefOid"`
	ReviewDecision    string          `json:"reviewDecision"`
	StatusCheckRollup *checkRollup    `json:"statusCheckRollup"`
	// viewer is the forge client's own login, from the same answer.
	viewer string
}

// checkRollup is the head commit's checks and statuses as the forge rolls
// them up, each node a check run (name, status, conclusion, completedAt) or a
// commit status (context, state, createdAt), with whether this pull request
// requires it; PageInfo says whether the forge holds more than land read.
type checkRollup struct {
	State    string `json:"state"`
	Contexts struct {
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
		Nodes []checkContext `json:"nodes"`
	} `json:"contexts"`
}

// newestRequiredPass is when the last of the required checks that have passed
// passed: a check run's completedAt, a commit status's createdAt (a status is
// created in its state, so a passing one's createdAt is when it passed).
// false when no passing required check gives a time that parses.
func (r *checkRollup) newestRequiredPass() (time.Time, bool) {
	var newest time.Time
	for _, c := range r.Contexts.Nodes {
		at := ""
		switch {
		case !c.IsRequired:
			continue
		case c.Context != "" || c.State != "":
			if c.State == "SUCCESS" {
				at = c.CreatedAt
			}
		case c.Status == "COMPLETED" && passingConclusions[c.Conclusion]:
			at = c.CompletedAt
		}
		if t, err := time.Parse(time.RFC3339, at); err == nil && t.After(newest) {
			newest = t
		}
	}
	return newest, !newest.IsZero()
}

// pendingRequiredListed reports whether a required check land read has not
// finished: a check run not completed, or a commit status pending or expected.
func (r *checkRollup) pendingRequiredListed() bool {
	for _, c := range r.Contexts.Nodes {
		switch {
		case !c.IsRequired:
		case c.Context != "" || c.State != "":
			if c.State == "PENDING" || c.State == "EXPECTED" {
				return true
			}
		case c.Status != "COMPLETED":
			return true
		}
	}
	return false
}

type checkContext struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	Context     string `json:"context"`
	State       string `json:"state"`
	IsRequired  bool   `json:"isRequired"`
	CompletedAt string `json:"completedAt"`
	CreatedAt   string `json:"createdAt"`
}

// passingConclusions are the check-run conclusions a required check passes
// with (the forge counts a skipped required check as passed).
var passingConclusions = map[string]bool{"SUCCESS": true, "NEUTRAL": true, "SKIPPED": true}

// maxCheckName bounds one check's name in a report.
const maxCheckName = 80

// heldChecks are the required checks that have not passed, each named as
// the forge names it (sanitised and bounded) with its state, in the forge's
// order.
func (s prState) heldChecks() []string {
	if s.StatusCheckRollup == nil {
		return nil
	}
	var held []string
	for _, c := range s.StatusCheckRollup.Contexts.Nodes {
		if !c.IsRequired {
			continue
		}
		name, why := c.Name, ""
		switch {
		case c.Context != "" || c.State != "":
			if name = c.Context; c.State != "SUCCESS" {
				why = c.State
			}
		case c.Status != "COMPLETED":
			why = c.Status
		case !passingConclusions[c.Conclusion]:
			why = c.Conclusion
			if why == "" {
				why = "no conclusion"
			}
		}
		if why == "" {
			continue
		}
		name = strings.TrimSpace(termsafe.Sanitize(name))
		if r := []rune(name); len(r) > maxCheckName {
			name = string(r[:maxCheckName])
		}
		if name == "" {
			name = "an unnamed check"
		}
		held = append(held, name+" ("+strings.ToLower(strings.ReplaceAll(termsafe.Sanitize(why), "_", " "))+")")
	}
	return held
}

func (s prState) armed() bool {
	t := strings.TrimSpace(string(s.AutoMergeRequest))
	return t != "" && t != "null"
}

// autoMerge is the forge's arming of a pull request.
type autoMerge struct {
	MergeMethod string `json:"mergeMethod"`
	EnabledAt   string `json:"enabledAt"`
	EnabledBy   *struct {
		Login string `json:"login"`
	} `json:"enabledBy"`
}

func (s prState) autoMerge() autoMerge {
	var a autoMerge
	_ = json.Unmarshal(s.AutoMergeRequest, &a)
	return a
}

// ownArming reports whether the forge says the account the forge client
// signs in as made the arming. The forge cannot tell land's arming from the
// person's own, so either counts as land's.
func (s prState) ownArming() bool {
	a := s.autoMerge()
	return s.armed() && s.viewer != "" && a.EnabledBy != nil && strings.EqualFold(a.EnabledBy.Login, s.viewer)
}

// armedFor is how long the arming has been in place (armed), by the forge's
// enabledAt, and how long it has sat armed with every required check
// passed: from the later of enabledAt and the newest passing required check's
// time (newestRequiredPass), so checks that take 25 minutes leave a re-run at
// arming+31 six minutes into the grace. false when the forge gives no arming
// time that parses.
//
// Both compare the forge's clock with the local one, and land makes no forge
// call to read the forge's: a local clock running fast shortens the grace by
// its lead, and one running slow lengthens it. A forge time later than the
// local clock counts as no time elapsed, so land never toggles an arming whose
// enabledAt, or whose last check's time, is in the future.
func (s prState) armedFor() (armed, clean time.Duration, ok bool) {
	if !s.armed() {
		return 0, 0, false
	}
	t, err := time.Parse(time.RFC3339, s.autoMerge().EnabledAt)
	if err != nil {
		return 0, 0, false
	}
	start := t
	if s.StatusCheckRollup != nil {
		if p, ok := s.StatusCheckRollup.newestRequiredPass(); ok && p.After(start) {
			start = p
		}
	}
	return max(now().Sub(t), 0), max(now().Sub(start), 0), true
}

// maxListedPRs bounds the forge's listing of a branch's open pull requests:
// a fork's pull requests from a branch of the same name are listed too.
const maxListedPRs = 20

// PRStateQuery is the forge's GraphQL query for a pull request's way to the
// merge: `gh pr view --json` has no merge-queue field (gh 2.100.0 refuses
// isInMergeQueue), so the state is read through the forge's API instead.
//
// It also reads the viewer's login (whose arming is land's own), the review
// decision, when and by whom the arming was made, and the head's checks with
// whether this pull request requires each (isRequired takes the pull
// request's number; with neither it nor an id the forge refuses the field).
// Every field is pinned in ghcontract, and testdata/ holds the forge's
// answers to this query.
//
// The checks are read a page at the forge's maximum (it refuses first:101,
// "exceeds the `first` limit of 100 records"), with whether more follow, and
// each with the time it finished (completedAt; a status's createdAt), which
// times the toggle's grace.
const PRStateQuery = `query($o:String!,$r:String!,$n:Int!){viewer{login} repository(owner:$o,name:$r){pullRequest(number:$n){isInMergeQueue mergeQueueEntry{state position} autoMergeRequest{mergeMethod enabledAt enabledBy{login}} reviewDecision mergeStateStatus mergeable headRefOid statusCheckRollup{state contexts(first:100){pageInfo{hasNextPage} nodes{... on CheckRun{name status conclusion completedAt isRequired(pullRequestNumber:$n)} ... on StatusContext{context state createdAt isRequired(pullRequestNumber:$n)}}}}}}}`

// prStateArgs are the forge client's arguments for PRStateQuery on pull
// request n of the repository owner/name. The owner and the name go as raw
// strings (-f), never read as a file or a number; n as a number (-F).
func prStateArgs(owner, name, n string) []string {
	return []string{"api", "graphql", "-f", "query=" + PRStateQuery, "-f", "o=" + owner, "-f", "r=" + name, "-F", "n=" + n}
}

// prStateAnswer is the forge's answer to PRStateQuery. An answer may carry
// errors beside its data (a field the token may not read comes back null),
// and gh exits 0 on some of them, so Errors is read, never left to gh.
type prStateAnswer struct {
	Data struct {
		Viewer *struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Repository *struct {
			PullRequest *prState `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// maxForgeErrorText bounds how much of the forge's error messages a refusal
// carries.
const maxForgeErrorText = 512

// errorText is the forge's messages in an answer's errors, sanitised and
// bounded; an error with no message is shown as its raw JSON.
func (a prStateAnswer) errorText() string {
	var msgs []string
	for _, raw := range a.Errors {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &e) != nil || strings.TrimSpace(e.Message) == "" {
			msgs = append(msgs, string(raw))
			continue
		}
		msgs = append(msgs, e.Message)
	}
	t := termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(strings.Join(msgs, "; "))))
	if len(t) > maxForgeErrorText {
		t = t[:maxForgeErrorText]
	}
	return t
}

// enqueue is step 6. tracking is origin's default branch as just fetched;
// nameWithOwner is the repository as the forge client resolved it when origin
// was checked, the one the pull request's state is asked of.
func enqueue(res *Result, o Options, dir, tracking, nameWithOwner string) error {
	if !res.Enqueue.On {
		res.add("enqueue", StatusOff, res.Enqueue.offText()+": nothing was armed or queued", "")
		return nil
	}
	out, err := forge(o, dir, "pr", "list", "--head="+res.Branch, "--state=open",
		"--json=number,url,isCrossRepository,headRefOid,baseRefName", fmt.Sprintf("--limit=%d", maxListedPRs))
	if err != nil {
		return err
	}
	var listed []openPR
	if json.Unmarshal([]byte(out), &listed) != nil {
		return refuse("the forge's listing of "+res.Branch+"'s pull requests does not parse", "check the forge client (gh --version), then run abcd land again")
	}
	// A fork's pull request from a branch of the same name is not this
	// branch's: it is set aside, and refused only when it is all there is.
	var prs []openPR
	for _, pr := range listed {
		if !pr.IsCrossRepository {
			prs = append(prs, pr)
		}
	}
	switch {
	case len(prs) == 0 && len(listed) > 0:
		return refuse("pull request #"+fmt.Sprint(listed[0].Number)+" for "+res.Branch+" comes from another repository (a fork), and land arms only a pull request from this repository's own branch",
			"merge it by hand on the forge, or open one from this repository's branch, then run abcd land again")
	case len(prs) == 0:
		res.add("enqueue", StatusStopped, "no open pull request for "+res.Branch+"; land opens none: open it with the command, then run abcd land again to queue it",
			"gh pr create --base "+shellQuote(res.Default)+" --head "+shellQuote(res.Branch))
		return nil
	case len(prs) > 1:
		return refuse("the forge lists more than one open pull request for "+res.Branch+" from this repository", "close all but one, then run abcd land again")
	}
	pr := prs[0]
	if pr.Number <= 0 {
		return refuse("the forge lists a pull request for "+res.Branch+" with no number", "check the forge client, then run abcd land again")
	}
	res.PR, res.PRURL = pr.Number, pr.URL
	n := fmt.Sprint(pr.Number)
	if pr.HeadRefOid != res.Head {
		res.add("enqueue", StatusStopped, fmt.Sprintf("pull request #%s's head on the forge is %s, not %s's head %s, so nothing was armed: push the branch (or let the forge see the push), then run abcd land again",
			n, short(pr.HeadRefOid), res.Branch, short(res.Head)), "")
		return nil
	}
	if pr.BaseRefName != res.Default {
		res.add("enqueue", StatusLeft, "pull request #"+n+" merges into "+pr.BaseRefName+", not the default branch "+res.Default+", whose ruleset land reads; its merge is left to a person", "")
		return nil
	}
	// The rule is read at the default branch's tip as just fetched: out of
	// the branch's reach as the merge base is, and fresher.
	tip, err := gitutil.Run(dir, "rev-parse", "--verify", "--quiet", tracking+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return refuse(Remote+"/"+res.Default+" cannot be read after the fetch, so the merge rule cannot be read", "fetch it (git fetch "+Remote+"), then run abcd land again")
	}
	rule, err := ReadMergeRule(dir, tip, res.Default, Remote+"/"+res.Default)
	var me *MirrorError
	switch {
	case errors.As(err, &me):
		return refuse(me.Reason, me.Fix+", then run abcd land again")
	case err != nil:
		return err
	}
	d := mergeGate(rule, res.Default)
	if !d.arm {
		res.add("enqueue", StatusLeft, "pull request #"+n+": "+d.why, "")
		return nil
	}
	owner, name, ok := strings.Cut(nameWithOwner, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return refuse("the forge client names the repository "+strconv.Quote(nameWithOwner)+", not owner/name, so land cannot ask the forge for pull request #"+n+"'s state, and nothing was armed",
			"check the forge client (gh repo view), then run abcd land again")
	}
	view := func() (prState, error) {
		out, err := forge(o, dir, prStateArgs(owner, name, n)...)
		if err != nil {
			return prState{}, err
		}
		var a prStateAnswer
		if json.Unmarshal([]byte(out), &a) != nil {
			return prState{}, refuse("the forge's view of pull request #"+n+" does not parse", "check the forge client, then run abcd land again")
		}
		if len(a.Errors) > 0 {
			return prState{}, refuse("the forge answered the query for pull request #"+n+"'s state with errors, so nothing was armed: "+a.errorText(),
				"settle what the forge reports (gh pr view "+n+"), then run abcd land again")
		}
		if a.Data.Repository == nil || a.Data.Repository.PullRequest == nil {
			return prState{}, refuse("the forge reports no pull request #"+n+" in "+nameWithOwner, "check the forge client (gh pr view "+n+"), then run abcd land again")
		}
		st := *a.Data.Repository.PullRequest
		if a.Data.Viewer != nil {
			st.viewer = a.Data.Viewer.Login
		}
		return st, nil
	}
	// The arming is pinned to the head land checked: the forge refuses it when
	// the pull request's head is another commit.
	armArgs := []string{"pr", "merge", n, "--auto", MergeMethods[rule.Method], "--match-head-commit=" + res.Head}
	arm := func() error {
		_, err := forge(o, dir, armArgs...)
		return err
	}
	s, err := view()
	if err != nil {
		return err
	}
	if s.Mergeable == "CONFLICTING" || s.MergeStateStatus == "DIRTY" {
		return refuse("pull request #"+n+" conflicts with "+res.Default+", and a conflicting pull request never enters the queue, so nothing was armed",
			"merge "+res.Default+" into "+res.Branch+" and settle the conflict, then run abcd land again")
	}
	note := "pull request #" + n + ": " + d.why
	if s.IsInMergeQueue {
		res.add("enqueue", StatusDone, note+reviewText(s)+"; it is already in the merge queue", "")
		return nil
	}
	// rearm makes the arming again after a toggle: land's own arguments for
	// an arming it makes now, the forge's method for one already made.
	rearm := armArgs
	switch {
	case s.armed() && !s.ownArming():
		// An arming another account made is someone's decision: it is left
		// as it is, never re-armed or toggled, and its queue state reported.
		res.add("enqueue", StatusDone, "pull request #"+n+" is already armed ("+armedMethod(s)+"), an arming another account made (or one the forge does not attribute), which land leaves as it is"+reviewText(s)+"; it is not in the merge queue yet: "+holdText(s, res.Default)+
			cleanAdvice(s, "land leaves an arming it did not make as it is, on this run and every re-run", n, rearmArgs(s, n, res.Head)), "")
		return nil
	case s.armed():
		// This account's arming, land's own from an earlier run (the forge
		// cannot tell land's arming from the person's own): the grace below
		// decides whether it has sat long enough to toggle.
		// An arming on a head other than the one land gated may merge a commit
		// land did not check: refused, as after a fresh arming.
		if s.HeadRefOid != res.Head {
			return refuse(fmt.Sprintf("pull request #%s is armed by this account, but its head on the forge is %s, not %s, the head land gated, so the arming may merge a commit land did not check", n, short(s.HeadRefOid), short(res.Head)),
				"withdraw the arming (gh pr merge "+n+" --disable-auto), then run abcd land again to gate and arm the forge's head")
		}
		note = "pull request #" + n + " is already armed (" + armedMethod(s) + ") by this account"
		rearm = rearmArgs(s, n, res.Head)
	default:
		if err := arm(); err != nil {
			return err
		}
		if s, err = confirmView(view, n, res.Head); err != nil {
			return err
		}
	}
	note += reviewText(s)
	// The confirmation (iss-2610090642376032): an armed pull request the
	// forge reports CLEAN can still sit outside the queue, and toggling
	// auto-merge once has queued it where the forge's enqueue call failed. It
	// also queues one by itself, after a wait (round 5), so the toggle waits
	// out toggleGrace, timed from the later of the forge's enabledAt and its
	// last required check's pass (armedFor). Both paths here hold the forge's
	// head at res.Head: confirmView refuses another after a fresh arming, and
	// the re-run path above refuses it.
	if _, clean, ok := s.armedFor(); ok && clean > toggleGrace && s.MergeStateStatus == "CLEAN" {
		if _, err := forge(o, dir, "pr", "merge", n, "--disable-auto"); err != nil {
			return err
		}
		if _, err := forge(o, dir, rearm...); err != nil {
			why := err.Error()
			if r := (*Refusal)(nil); errors.As(err, &r) {
				why = r.Reason
			}
			return refuse("pull request #"+n+" is now DISARMED: land withdrew its auto-merge to queue it again and the forge refused the new arming ("+why+")",
				"arm it again yourself with: gh "+strings.Join(shellWords(rearm), " ")+" (or run abcd land again)")
		}
		note += fmt.Sprintf("; it sat CLEAN outside the queue for %d min, armed with its required checks passed, past the %d-minute grace, so auto-merge was toggled once", int(clean/time.Minute), int(toggleGrace/time.Minute))
		if s, err = confirmView(view, n, res.Head); err != nil {
			return err
		}
	}
	if s.IsInMergeQueue {
		res.add("enqueue", StatusDone, note+"; it is in the merge queue", "")
		return nil
	}
	res.add("enqueue", StatusDone, note+"; it is not in the merge queue yet: "+holdText(s, res.Default)+waitText(s, n, rearm), "")
	return nil
}

// toggleGrace is how long this account's arming may sit CLEAN outside the
// merge queue (armedFor's clean time) before land toggles it. On 2026-10-15 pull
// request #1018, armed at 12:52:16Z, had its last required check pass at
// 12:59:42Z and was seen CLEAN, armed and unqueued at 12:59:47Z; the forge
// queued it by itself at 13:15:01Z, a 15m14s pickup: a normal wait, which a
// toggle would only have churned. The grace is about twice that.
const toggleGrace = 30 * time.Minute

// now is land's clock, a var so a test can fix it. An arming's age is read
// from the forge's enabledAt against it; land keeps no clock state of its own.
var now = time.Now

// waitText is what the person can do about this account's arming the forge
// reports CLEAN yet outside the merge queue (rearm arms it again): inside
// the grace, wait and run land again after it; past it (land has toggled it
// once on this run) or with no arming time from the forge, wait or toggle it
// by hand. "" for any other state, which the forge settles itself.
func waitText(s prState, n string, rearm []string) string {
	if s.MergeStateStatus != "CLEAN" {
		return ""
	}
	armed, clean, ok := s.armedFor()
	switch {
	case !ok:
		return cleanAdvice(s, "the forge gives no time for this arming, so land does not toggle it", n, rearm)
	case clean <= toggleGrace:
		since := fmt.Sprintf("armed %d min ago", int(armed/time.Minute))
		if clean < armed {
			since += fmt.Sprintf(", CLEAN for %d min since its last required check passed", int(clean/time.Minute))
		}
		return fmt.Sprintf("; armed; the forge has not queued it yet (%s); run abcd land again after %s if it is still not queued",
			since, ceilMinute(now().Add(toggleGrace-clean)).UTC().Format("15:04 UTC"))
	}
	return cleanAdvice(s, "land toggles an arming once a run", n, rearm)
}

// ceilMinute is t rounded up to the minute, so a time to come back after is
// never inside the grace.
func ceilMinute(t time.Time) time.Time {
	if r := t.Truncate(time.Minute); !r.Equal(t) {
		return r.Add(time.Minute)
	}
	return t
}

// cleanAdvice is what the person can do about an armed pull request the forge
// reports CLEAN yet outside the merge queue, which a re-run of land does not
// change (why says so): wait for the forge, or withdraw the arming and make it
// again with rearm (the forge client's arguments). "" for any other state,
// which the forge settles itself (a check passing, an approval).
func cleanAdvice(s prState, why, n string, rearm []string) string {
	if s.MergeStateStatus != "CLEAN" {
		return ""
	}
	return "; " + why + ": wait for the forge to queue it, or withdraw the arming and arm it again yourself (gh pr merge " + n + " --disable-auto, then gh " + strings.Join(shellWords(rearm), " ") + ")"
}

// rearmArgs arms pull request n again by the method of the arming the forge
// reports, pinned to head.
func rearmArgs(s prState, n, head string) []string {
	flag := MergeMethods[strings.ToUpper(armedMethod(s))]
	if flag == "" {
		flag = "--merge"
	}
	return []string{"pr", "merge", n, "--auto", flag, "--match-head-commit=" + head}
}

// confirmView views the pull request after an arming and refuses when its
// head on the forge is no longer head: the arming would merge a commit land
// did not check.
func confirmView(view func() (prState, error), n, head string) (prState, error) {
	s, err := view()
	if err != nil {
		return s, err
	}
	if s.HeadRefOid != head {
		return s, refuse(fmt.Sprintf("pull request #%s's head on the forge moved from %s to %s after land armed it, so the arming may merge a commit land did not check", n, short(head), short(s.HeadRefOid)),
			"withdraw the arming (gh pr merge "+n+" --disable-auto), then run abcd land again to gate and arm the new head")
	}
	return s, nil
}

// armedMethod is the merge method of the arming the forge reports.
func armedMethod(s prState) string {
	if m := s.autoMerge().MergeMethod; m != "" {
		return strings.ToLower(m)
	}
	return "auto-merge"
}

// shellWords quotes each word for a POSIX shell where it needs it.
func shellWords(ws []string) []string {
	out := slices.Clone(ws)
	for i, w := range out {
		out[i] = shellQuote(w)
	}
	return out
}

// maxHeldChecks bounds how many held required checks a report names.
const maxHeldChecks = 8

// holdText says what holds a pull request out of the queue: the required
// checks that have not passed, named first, then the forge's merge state.
// BEHIND is no holder where land arms, since a merge queue gates def and
// builds each entry on top of it, but only when land read every check and
// the forge's rollup is not waiting on one land did not list: a required
// check with no run yet (just after a push) or beyond the page land read can
// hold it too, and the forge's rollup state is the authority on that.
func holdText(s prState, def string) string {
	var b strings.Builder
	if held := s.heldChecks(); len(held) > 0 {
		shown := held[:min(len(held), maxHeldChecks)]
		b.WriteString("required checks have not passed: " + strings.Join(shown, ", "))
		if more := len(held) - len(shown); more > 0 {
			fmt.Fprintf(&b, ", and %d more", more)
		}
		b.WriteString("; ")
	}
	unread := false
	if r := s.StatusCheckRollup; r != nil {
		if (r.State == "PENDING" || r.State == "EXPECTED") && !r.pendingRequiredListed() {
			b.WriteString("checks are still pending (the forge's rollup is " + r.State + "), and no required check land read is pending: one may not have started yet; ")
			unread = true
		}
		if r.Contexts.PageInfo.HasNextPage {
			b.WriteString("the forge lists more checks than land read (the first 100), so one beyond them may hold it; ")
			unread = true
		}
	}
	st := s.MergeStateStatus
	if st == "" {
		st = "no merge state"
	}
	why := map[string]string{
		"BLOCKED":  "a required check or review has not passed",
		"BEHIND":   "",
		"UNSTABLE": "a check is failing",
		"UNKNOWN":  "the forge has not computed it yet",
		"DRAFT":    "the pull request is a draft",
		"CLEAN":    "it is mergeable, but the forge has not queued it",
	}[st]
	b.WriteString("the forge reports " + termsafe.Sanitize(st))
	switch {
	case st == "BEHIND" && unread:
		b.WriteString(" (the queue tests it on top of " + def + ")")
	case st == "BEHIND":
		b.WriteString(", which does not hold it: the queue tests it on top of " + def)
	case why != "":
		b.WriteString(" (" + why + ")")
	}
	return b.String()
}

// reviewText is the forge's review decision on the pull request, as a note
// clause: the forge's word, never the ruleset mirror's approval rule, which
// covers only the paths code owners own.
func reviewText(s prState) string {
	switch s.ReviewDecision {
	case "REVIEW_REQUIRED":
		return "; the forge holds it for a person's approval"
	case "APPROVED":
		return "; the forge reports it approved"
	case "CHANGES_REQUESTED":
		return "; a reviewer requested changes, which hold it"
	case "":
		return "; the forge requires no review for this pull request"
	}
	return "; the forge reports the review decision " + termsafe.Sanitize(s.ReviewDecision)
}

// gateDecision is the merge gate's answer: whether land arms the merge, and
// why, in words a step reports.
type gateDecision struct {
	arm bool
	why string
}

// mergeGate decides whether land arms a pull request's merge. It is the one
// place that decision is made, and the seat of the facilitator mode draft
// itd-97 describes (auto: queue at once; human: the merge waits for the
// facilitator), which replaces this function when its toggle is built. Until
// then every repository runs in human mode, and in human mode ruling AM1 is
// the behaviour: auto-merge is armed only where a merge queue gates the
// default branch AND a ruleset holds the merge for a person's approval, so
// what land arms a person must still approve; anywhere else the merge is left
// to a person. Nothing else in the landing decides human or auto.
func mergeGate(r MergeRule, def string) gateDecision {
	switch {
	case r.Method == "":
		return gateDecision{why: "no merge queue gates " + def + " (per the ruleset mirror), so its merge is left to a person"}
	case !r.Approval:
		return gateDecision{why: "the ruleset requires no person's approval on " + def + ", so under ruling AM1 its merge is left for a person to make"}
	}
	return gateDecision{arm: true, why: "auto-merge armed through " + def + "'s merge queue (" + strings.ToLower(r.Method) + ")"}
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// shellSafeRe is what a shell takes unquoted.
var shellSafeRe = regexp.MustCompile(`^[A-Za-z0-9._/:@%+=-]+$`)

// shellQuote quotes s for a POSIX shell when it needs it.
func shellQuote(s string) string {
	if shellSafeRe.MatchString(s) {
		return s
	}
	return shellquote.Single(s)
}
