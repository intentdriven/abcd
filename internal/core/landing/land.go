package landing

// land.go is `abcd land [<branch>]` (iss-2610150524235929,
// iss-2610150524246898, with iss-2610090642376032's queue confirmation for
// this verb): one landing path for any branch, a build lane's or not. Each
// step is idempotent, so a re-run resumes where the last one stopped:
//
//  1. resolve: the branch (the checkout's current one, or the one named, in
//     the worktree that has it checked out); refused on the default branch, a
//     detached HEAD and a tree with uncommitted changes.
//  2. gate: skipped when a preflight receipt already names HEAD (in any
//     worktree, as the pre-push hook looks); otherwise `make preflight` runs in
//     the branch's checkout (it waits on the machine load waiter itself), and
//     HEAD must be unchanged and a receipt present after it.
//  3. push (land.push): a plain `git push --porcelain origin
//     refs/heads/B:refs/heads/B` from the branch's checkout, the repository's
//     hooks running, never forced, never the default branch. Off, it prints
//     the command and names the layer that turned it off.
//  4. enqueue (land.enqueue): the open pull request for the branch, armed by
//     the merge gate (mergeGate) and confirmed in the merge queue; with no
//     pull request it prints the `gh pr create` command and stops. It never
//     opens a pull request and never edits the repository's settings.
//
// The trust boundaries: the remote is `origin` only, and both its fetch and
// its push URL must be the repository the forge client resolves here; a pull
// request from another repository is refused; the merge rule is read from the
// ruleset mirror at the merge base with origin's default branch, never at the
// branch's tip, so the branch cannot change the rule it lands by; the forge
// client's and git's environments are scrubbed of GIT_DIR and GIT_CONFIG_*.
// The receipt is a convenience check, not a security boundary: CI re-runs
// every gate on the pushed commit.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/shellquote"
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
)

// gateTimeout bounds one preflight run: the load waiter it starts with gives
// up after two hours, and the gates after it take well under one.
const gateTimeout = 4 * time.Hour

// Options are one landing's inputs and its seams. A nil seam takes the real
// one: Forge is GH, Git is gitutil.Net (the networked git runner, hooks
// running), Gate runs `make preflight`.
type Options struct {
	// Branch is the branch to land; "" is the checkout's current branch.
	Branch string
	// NoPush and NoEnqueue are --no-push and --no-enqueue.
	NoPush    bool
	NoEnqueue bool
	// Roots are where land.push and land.enqueue are read.
	Roots layered.Roots
	// Forge runs the forge client with args from dir.
	Forge func(dir string, args ...string) (string, error)
	// Git runs one networked git command (a push or a fetch) from dir.
	Git func(dir string, args ...string) (string, error)
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
	Steps    []Step `json:"steps"`
}

func (r *Result) add(name, status, note, command string) {
	r.Steps = append(r.Steps, Step{Name: name, Status: status, Note: note, Command: command})
}

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

func refuse(reason, fix string) error { return &Refusal{Reason: reason, Fix: fix} }

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
	if res.Push, res.Enqueue, err = readSwitches(o.Roots, o.NoPush, o.NoEnqueue); err != nil {
		return res, refuse("the land configuration cannot be read: "+fsutil.RedactHome(err.Error()), "correct the file it names; nothing was pushed")
	}

	if err := gate(&res, o, dir, head); err != nil {
		return res, err
	}
	if res.Push.On || res.Enqueue.On {
		if err := checkOrigin(o, dir); err != nil {
			return res, err
		}
	}
	if err := push(&res, o, dir); err != nil {
		return res, err
	}
	return res, enqueue(&res, o, dir)
}

// resolveBranch finds the checkout to land from: the one at root on its
// current branch, or the worktree that has branch checked out.
func resolveBranch(root, branch string) (dir, name string, err error) {
	wts, err := gitutil.ListWorktrees(root, maxWorktreeListing)
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

// gate is step 2.
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
		ctx, cancel := context.WithTimeout(context.Background(), gateTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "preflight")
		cmd.Dir = dir
		cmd.Env = gitutil.ScrubbedEnv()
		cmd.Stdout, cmd.Stderr = out, out
		if err := cmd.Run(); err != nil {
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

// checkOrigin refuses unless origin's fetch and push URLs are both the
// repository the forge client resolves here, so the push and the pull request
// land in the same place.
func checkOrigin(o Options, dir string) error {
	out, err := forge(o, dir, "repo", "view", "--json", "nameWithOwner,url,sshUrl")
	if err != nil {
		return err
	}
	var repo struct {
		NameWithOwner string `json:"nameWithOwner"`
		URL           string `json:"url"`
		SSHURL        string `json:"sshUrl"`
	}
	if json.Unmarshal([]byte(out), &repo) != nil {
		return refuse("the forge client's view of the repository does not parse", "check the forge client (gh --version), then run abcd land again; nothing was pushed")
	}
	keys := map[string]bool{}
	for _, u := range []string{repo.URL, repo.SSHURL} {
		if k := repoKey(u); k != "" {
			keys[k] = true
		}
	}
	for _, push := range []bool{false, true} {
		args := []string{"remote", "get-url"}
		what := "fetch URL"
		if push {
			args, what = append(args, "--push"), "push URL"
		}
		u, err := userGit(dir, append(args, Remote)...)
		if err != nil {
			return refuse("the repository has no remote named "+Remote+", the one remote land pushes to", "add it, then run abcd land again; nothing was pushed")
		}
		if k := repoKey(u); k == "" || !keys[k] {
			return refuse(fmt.Sprintf("%s's %s (%s) is not the repository the forge client resolves here (%s)", Remote, what, display(k), display(repoKey(repo.URL))),
				"point "+Remote+" at the forge's repository, or the forge client at "+Remote+" (gh repo set-default), then run abcd land again; nothing was pushed")
		}
	}
	return nil
}

func display(k string) string {
	if k == "" {
		return "unreadable"
	}
	return strings.TrimPrefix(k, ":")
}

// userGit runs one git read under the person's own configuration (an
// insteadOf rewrite included, as the push will see it), less any injected
// GIT_DIR or GIT_CONFIG_*.
func userGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gitutil.ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")
	out, err := RunCapped(cmd, "git "+args[0])
	return strings.TrimSpace(out), err
}

// repoKey is a repository URL's comparable form, "host:/path" with the host
// and (on a host) the path lower-cased, any user, port, trailing slash and
// .git suffix dropped; a local path or file URL has an empty host. "" when the
// URL cannot be read.
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

// forge runs the forge client through the seam, its failures as refusals.
func forge(o Options, dir string, args ...string) (string, error) {
	out, err := o.Forge(dir, args...)
	if errors.Is(err, ErrNoForgeClient) {
		return "", refuse("the forge client `gh` is not installed, and land reads the repository and its pull request through it",
			"install gh and sign in with your own identity (gh auth login), or turn land.push and land.enqueue off")
	}
	if err != nil {
		return "", refuse(fsutil.RedactHome(err.Error()), "settle what the forge client reports, then run abcd land again")
	}
	return out, nil
}

// pushArgs is the push, plain: no force, no hook skipped, the branch to the
// same name on origin.
func pushArgs(branch string) []string {
	ref := "refs/heads/" + branch
	return []string{"push", "--porcelain", Remote, ref + ":" + ref}
}

// push is step 3.
func push(res *Result, o Options, dir string) error {
	ref := "refs/heads/" + res.Branch
	if !res.Push.On {
		res.add("push", StatusOff, res.Push.offText()+": nothing was pushed; push it yourself with the command", "git push "+Remote+" "+shellQuote(ref+":"+ref))
		return nil
	}
	if _, err := o.Git(dir, pushArgs(res.Branch)...); err != nil {
		return refuse("git could not push "+res.Branch+" to "+Remote+": "+fsutil.RedactHome(err.Error()),
			"settle what git or the pre-push hook reports, then run abcd land again; land never forces a push or skips a hook")
	}
	res.add("push", StatusDone, "pushed "+res.Branch+" at "+short(res.Head)+" to "+Remote, "")
	return nil
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
	Mergeable        string          `json:"mergeable"`
	MergeStateStatus string          `json:"mergeStateStatus"`
	IsInMergeQueue   bool            `json:"isInMergeQueue"`
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

func (s prState) armed() bool {
	t := strings.TrimSpace(string(s.AutoMergeRequest))
	return t != "" && t != "null"
}

// enqueue is step 4.
func enqueue(res *Result, o Options, dir string) error {
	if !res.Enqueue.On {
		res.add("enqueue", StatusOff, res.Enqueue.offText()+": nothing was armed or queued", "")
		return nil
	}
	out, err := forge(o, dir, "pr", "list", "--head="+res.Branch, "--state=open",
		"--json=number,url,isCrossRepository,headRefOid,baseRefName", "--limit=2")
	if err != nil {
		return err
	}
	var prs []openPR
	if json.Unmarshal([]byte(out), &prs) != nil {
		return refuse("the forge's listing of "+res.Branch+"'s pull requests does not parse", "check the forge client (gh --version), then run abcd land again")
	}
	switch {
	case len(prs) == 0:
		res.add("enqueue", StatusStopped, "no open pull request for "+res.Branch+"; land opens none: open it with the command, then run abcd land again to queue it",
			"gh pr create --base "+shellQuote(res.Default)+" --head "+shellQuote(res.Branch))
		return nil
	case len(prs) > 1:
		return refuse("the forge lists more than one open pull request for "+res.Branch, "close all but one, then run abcd land again")
	}
	pr := prs[0]
	if pr.Number <= 0 {
		return refuse("the forge lists a pull request for "+res.Branch+" with no number", "check the forge client, then run abcd land again")
	}
	res.PR, res.PRURL = pr.Number, pr.URL
	n := fmt.Sprint(pr.Number)
	if pr.IsCrossRepository {
		return refuse("pull request #"+n+" comes from another repository (a fork), and land arms only a pull request from this repository's own branch",
			"merge it by hand on the forge")
	}
	if pr.HeadRefOid != res.Head {
		res.add("enqueue", StatusStopped, fmt.Sprintf("pull request #%s's head on the forge is %s, not %s's head %s, so nothing was armed: push the branch (or let the forge see the push), then run abcd land again",
			n, short(pr.HeadRefOid), res.Branch, short(res.Head)), "")
		return nil
	}
	if pr.BaseRefName != res.Default {
		res.add("enqueue", StatusLeft, "pull request #"+n+" merges into "+pr.BaseRefName+", not the default branch "+res.Default+", whose ruleset land reads; its merge is left to a person", "")
		return nil
	}
	tracking := "refs/remotes/" + Remote + "/" + res.Default
	if _, err := o.Git(dir, "fetch", "--quiet", "--no-tags", Remote, "+refs/heads/"+res.Default+":"+tracking); err != nil {
		return refuse("git could not fetch "+res.Default+" from "+Remote+": "+fsutil.RedactHome(err.Error()), "settle what git reports, then run abcd land again")
	}
	base, err := gitutil.Run(dir, "merge-base", res.Head, tracking)
	if err != nil || !gitutil.IsFullSHA(base) {
		return refuse(res.Branch+" shares no history with "+Remote+"/"+res.Default+", so the merge rule cannot be read", "rebase the branch on "+res.Default+", then run abcd land again")
	}
	rule, err := ReadMergeRule(dir, base, res.Default, "the merge base with "+res.Default)
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
	view := func() (prState, error) {
		out, err := forge(o, dir, "pr", "view", n, "--json", "mergeable,mergeStateStatus,isInMergeQueue,autoMergeRequest")
		if err != nil {
			return prState{}, err
		}
		var s prState
		if json.Unmarshal([]byte(out), &s) != nil {
			return prState{}, refuse("the forge's view of pull request #"+n+" does not parse", "check the forge client, then run abcd land again")
		}
		return s, nil
	}
	arm := func() error {
		_, err := forge(o, dir, "pr", "merge", n, "--auto", MergeMethods[rule.Method])
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
		res.add("enqueue", StatusDone, note+"; it is already in the merge queue", "")
		return nil
	}
	if !s.armed() {
		if err := arm(); err != nil {
			return err
		}
	}
	// The confirmation (iss-2610090642376032): an armed pull request the
	// forge reports CLEAN can still sit outside the queue; toggling auto-merge
	// once has queued it where the forge's enqueue call failed.
	if s, err = view(); err != nil {
		return err
	}
	if !s.IsInMergeQueue && s.MergeStateStatus == "CLEAN" {
		if _, err := forge(o, dir, "pr", "merge", n, "--disable-auto"); err != nil {
			return err
		}
		if err := arm(); err != nil {
			return err
		}
		note += "; it sat CLEAN outside the queue, so auto-merge was toggled once"
		if s, err = view(); err != nil {
			return err
		}
	}
	if s.IsInMergeQueue {
		res.add("enqueue", StatusDone, note+"; it is in the merge queue", "")
		return nil
	}
	res.add("enqueue", StatusDone, note+"; it is not in the merge queue yet: the forge reports "+holdText(s), "")
	return nil
}

// holdText says what the forge reports holds a pull request out of the queue.
func holdText(s prState) string {
	st := s.MergeStateStatus
	if st == "" {
		st = "no merge state"
	}
	why := map[string]string{
		"BLOCKED":  "a required check or review has not passed",
		"BEHIND":   "the branch is behind the default branch",
		"UNSTABLE": "a check is failing",
		"UNKNOWN":  "the forge has not computed it yet",
		"DRAFT":    "the pull request is a draft",
		"CLEAN":    "it is mergeable; run abcd land again to check the queue",
	}[st]
	if why == "" {
		return st
	}
	return st + " (" + why + ")"
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
	return gateDecision{arm: true, why: "auto-merge armed through " + def + "'s merge queue (" + strings.ToLower(r.Method) + "), which holds it for a person's approval"}
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
