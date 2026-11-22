package reland

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// PullRequest is what the forge says of one pull request: what the
// dependency-reauthor workflow fills its environment from.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	// Author is the opener's login (`dependabot[bot]` for Dependabot).
	Author string `json:"author"`
	// HeadRef and HeadSHA are the branch and the commit it holds.
	HeadRef string `json:"head_ref"`
	HeadSHA string `json:"head_sha"`
	// HeadRepo is the repository the branch lives in, as owner/name, and
	// Repo the one the pull request is opened on.
	HeadRepo string `json:"head_repo"`
	Repo     string `json:"repo"`
}

// Forge is how the forge is asked. ReadPR reads one pull request; Pusher
// names the account whose push last moved ref (a full refs/heads/ ref) to
// sha, the workflow's "actor".
type Forge struct {
	ReadPR func(root string, n int) (PullRequest, error)
	Pusher func(ctx context.Context, root, ref, sha string) (string, error)
	// ListOpen lists the repository's open pull requests (List).
	ListOpen func(root string) ([]PullRequest, error)
}

// Options is one reland.
type Options struct {
	// Root is the checkout the branch is made in.
	Root string
	// PR is the bot's pull request.
	PR int
	// AssistedBy is the model form of the disclosure trailer
	// (`<Vendor>:<model-version>`) when an agent runs the reland; "" writes
	// `Assisted-by: None`.
	AssistedBy string
	// Forge asks the forge; its zero members are GitHub's (DefaultForge).
	Forge Forge
	// Fetch runs one networked git command from the checkout; nil is
	// gitutil.Net.
	Fetch func(dir string, args ...string) (string, error)
}

// Result is a reland made.
type Result struct {
	PR         int    `json:"pull_request"`
	Subject    string `json:"subject"`
	Superseded string `json:"superseded_head"`
	Default    string `json:"default_branch"`
	Base       string `json:"base"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	Author     string `json:"author"`
	Bound      string `json:"bound"`
	Next       string `json:"next"`
}

// Refusal is a reland refused: the pull request is outside the bound, its
// change does not apply, or the checkout's identity is a machine's. Nothing
// was written.
type Refusal struct{ Reason string }

func (r *Refusal) Error() string { return r.Reason }

func refuse(format string, a ...any) error { return &Refusal{Reason: fmt.Sprintf(format, a...)} }

// headRefRe is the charset the workflow's script admits a branch name in
// before any message repeats it; it is checked here before the name reaches
// a refspec, and the script checks it again.
var headRefRe = regexp.MustCompile(`^[A-Za-z0-9._/+-]{1,255}$`)

// gitCap bounds a git read here: object names, a commit, a short listing.
const gitCap = 1 << 20

// branchFor is the local branch a reland of pull request n is made on.
func branchFor(n int) string { return "reland/pr-" + strconv.Itoa(n) }

// Reland relands pull request o.PR as the checkout's own commit: it asks the
// forge for the pull request, fetches origin's default branch and the
// pull request's branch, judges it with the workflow's script against the
// declaration the default branch stores, applies the bot's commit onto the
// default branch's tip as stored (a three-way merge from the commit's own
// parent, writing no working tree), and commits the result under the
// checkout's git identity on a new local branch. It pushes nothing and opens
// nothing.
func Reland(o Options) (Result, error) {
	res := Result{PR: o.PR}
	if o.PR <= 0 {
		return res, errors.New("a pull request number is a positive integer")
	}
	if o.AssistedBy != "" && (!assistedby.IsModelValue(o.AssistedBy) || assistedby.NamesAbcd(o.AssistedBy)) {
		return res, fmt.Errorf("--assisted-by takes the model form <Vendor>:<model-version> (for example Claude:claude-opus-5-5)")
	}
	f := o.Forge.withDefaults()
	fetch := o.Fetch
	if fetch == nil {
		fetch = gitutil.Net
	}
	root := o.Root

	// The identity first: a machine identity refuses before the network.
	who, err := checkoutIdentity(root)
	if err != nil {
		return res, err
	}
	res.Author = who

	pr, err := f.ReadPR(root, o.PR)
	if err != nil {
		return res, fmt.Errorf("reading pull request #%d from the forge: %w", o.PR, err)
	}
	if !strings.EqualFold(pr.State, "open") {
		return res, refuse("pull request #%d is %s, not open: there is nothing to reland", o.PR, strings.ToLower(pr.State))
	}
	if !headRefRe.MatchString(pr.HeadRef) || !gitutil.RefIsSafe(pr.HeadRef) {
		return res, refuse("pull request #%d's branch name carries a character outside [A-Za-z0-9._/+-]", o.PR)
	}
	if !gitutil.IsFullSHA(pr.HeadSHA) {
		return res, fmt.Errorf("the forge named no full commit for pull request #%d's head", o.PR)
	}
	if pr.HeadRepo != pr.Repo {
		return res, refuse("pull request #%d's branch lives in %s, not in %s: a bump is the repository's own branch", o.PR, pr.HeadRepo, pr.Repo)
	}
	res.Subject, res.Superseded = pr.Title, pr.HeadSHA

	def, err := gitutil.RemoteHeadBranch(fetch, root, "origin")
	if err != nil {
		return res, err
	}
	res.Default = def
	tracking := "refs/remotes/origin/" + def
	headTracking := "refs/remotes/origin/" + pr.HeadRef
	if _, err := fetch(root, "fetch", "--no-tags", "--quiet", "origin",
		"+refs/heads/"+def+":"+tracking, "+refs/heads/"+pr.HeadRef+":"+headTracking); err != nil {
		return res, fmt.Errorf("fetching %s and %s from origin: %w", def, pr.HeadRef, err)
	}
	base, err := gitutil.RunCappedAsStored(root, gitCap, "rev-parse", "--verify", "--quiet", tracking+"^{commit}")
	if err != nil || !gitutil.IsFullSHA(base) {
		return res, fmt.Errorf("origin's %s did not resolve to a commit after the fetch", def)
	}
	res.Base = base
	if head, err := gitutil.RunCappedAsStored(root, gitCap, "rev-parse", "--verify", "--quiet", headTracking+"^{commit}"); err != nil || head != pr.HeadSHA {
		return res, refuse("origin's %s does not hold %s, the commit the forge names for #%d: the bot moved it since; run this again", pr.HeadRef, pr.HeadSHA, o.PR)
	}

	actor, err := f.Pusher(context.Background(), root, "refs/heads/"+pr.HeadRef, pr.HeadSHA)
	if err != nil {
		return res, refuse("the forge's record does not say who pushed %s to %s (%v), so the bound's actor clause cannot be judged", pr.HeadSHA, pr.HeadRef, err)
	}

	// The bound, judged by the workflow's own script against the declaration
	// the default branch stores, never the checkout's working copy.
	conf, err := gitutil.RunCappedAsStored(root, scriptOutputCap, "cat-file", "blob", base+":"+scaffold.ReauthorConfPath)
	if err != nil {
		return res, refuse("origin's %s declares no dependency re-authoring (%s), so it states no bound to reland inside", def, scaffold.ReauthorConfPath)
	}
	dir, err := os.MkdirTemp("", "abcd-reland-")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(dir)
	confPath := filepath.Join(dir, "dependency-reauthor.conf")
	if err := os.WriteFile(confPath, []byte(conf+"\n"), 0o600); err != nil {
		return res, err
	}
	out, err := runScript(root, "check", map[string]string{
		"REAUTHOR_CONF": confPath, "PR_AUTHOR": pr.Author, "ACTOR": actor, "HEAD_REF": pr.HeadRef,
		"HEAD_SHA": pr.HeadSHA, "BASE_SHA": base, "HEAD_REPO": pr.HeadRepo, "GITHUB_REPOSITORY": pr.Repo,
	})
	var br *boundRefusal
	if errors.As(err, &br) {
		return res, &Refusal{Reason: br.msg}
	}
	if err != nil {
		return res, err
	}
	verdict := ""
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "dependency-reauthor: "); ok {
			verdict = v
		}
	}
	switch {
	case strings.HasPrefix(verdict, "in bound: "):
		res.Bound = strings.TrimPrefix(verdict, "in bound: ")
	case strings.HasPrefix(verdict, "left alone "):
		return res, refuse("pull request #%d is outside the re-authoring bound, so it is not relanded: %s", o.PR, strings.TrimPrefix(verdict, "left alone "))
	default:
		return res, fmt.Errorf("the re-authoring script gave no verdict on #%d", o.PR)
	}

	// The change: the bot's commit, three-way merged from its own parent onto
	// the default branch's tip, with git's built-in merge in place of any
	// driver the repository configures. No working tree is written.
	fork, err := gitutil.RunCappedAsStored(root, gitCap, "merge-base", base, pr.HeadSHA)
	if err != nil || !gitutil.IsFullSHA(fork) {
		return res, fmt.Errorf("no merge base between %s and %s", base, pr.HeadSHA)
	}
	overrides, err := gitutil.MergeDriverOverrides(root)
	if err != nil {
		return res, err
	}
	args := append(overrides, "-c", "merge.union.driver="+gitutil.BuiltinMergeDriver(),
		"merge-tree", "--write-tree", "--name-only", "--no-messages", "--merge-base="+fork, base, pr.HeadSHA)
	merged, conflicted, err := gitutil.RunCappedExitOneAsStored(root, gitCap, args...)
	if err != nil {
		return res, fmt.Errorf("applying #%d's commit onto %s: %w", o.PR, def, err)
	}
	lines := strings.Split(strings.TrimSpace(string(merged)), "\n")
	if conflicted {
		files := strings.Join(slices.DeleteFunc(lines[1:], func(s string) bool { return strings.TrimSpace(s) == "" }), ", ")
		return res, refuse("#%d's change does not apply onto %s at %s without a conflict (in %s): %s has moved under the bump, "+
			"so redo the bump on %s with the ecosystem's own tool (for example `go get <module>@<version>` and `go mod tidy`, "+
			"or the pip pin in the requirements file) and commit it as your own", o.PR, def, short(base), files, def, def)
	}
	tree := strings.TrimSpace(lines[0])
	if !gitutil.IsFullSHA(tree) {
		return res, fmt.Errorf("applying #%d's commit onto %s wrote no tree", o.PR, def)
	}
	baseTree, err := gitutil.RunCappedAsStored(root, gitCap, "rev-parse", base+"^{tree}")
	if err != nil {
		return res, err
	}
	if tree == baseTree {
		return res, refuse("%s at %s already holds #%d's change: there is nothing to reland, and the bot closes its pull request itself", def, short(base), o.PR)
	}
	// Only the files the bot's commit changed may differ from the default
	// branch: a merge that reached further is refused, not committed.
	theirs, err := nameOnlyDiff(root, fork, pr.HeadSHA)
	if err != nil {
		return res, err
	}
	ours, err := nameOnlyDiff(root, base, tree)
	if err != nil {
		return res, err
	}
	for _, p := range ours {
		if !slices.Contains(theirs, p) {
			return res, refuse("applying #%d's commit onto %s changed %s, which the bot's commit does not change", o.PR, def, p)
		}
	}

	// The commit, under the checkout's identity, with the disclosure trailer.
	msg := message(pr, def, base, res.Bound, o.AssistedBy)
	msgPath := filepath.Join(dir, "message")
	if err := os.WriteFile(msgPath, []byte(msg), 0o600); err != nil {
		return res, err
	}
	commit, err := gitutil.LocalGit(root, "commit-tree", tree, "-p", base, "-F", msgPath)
	if err != nil {
		return res, fmt.Errorf("composing the relanded commit: %w", err)
	}
	commit = strings.TrimSpace(commit)
	if !gitutil.IsFullSHA(commit) {
		return res, fmt.Errorf("composing the relanded commit named no commit")
	}
	branch := branchFor(o.PR)
	// An empty old value creates the branch only where none exists, so a
	// branch already there is never moved.
	if _, err := gitutil.LocalGit(root, "update-ref", "-m", "abcd launch reland #"+strconv.Itoa(o.PR), "refs/heads/"+branch, commit, ""); err != nil {
		return res, refuse("the branch %s already exists here (%v): land or delete it, then run this again", branch, err)
	}
	res.Branch, res.Commit = branch, commit
	res.Next = "git switch " + branch + ", then abcd land; once it merges, the bot closes #" + strconv.Itoa(o.PR) + " itself"
	return res, nil
}

// message is the relanded commit's message: the bot's subject, what was done
// and under which bound, the pull request it supersedes, and the trailer.
func message(pr PullRequest, def, base, bound, assisted string) string {
	subject := strings.TrimSpace(strings.SplitN(pr.Title, "\n", 2)[0])
	if subject == "" {
		subject = "bump dependencies"
	}
	trailer := "Assisted-by: None"
	if assisted != "" {
		trailer = "Assisted-by: " + assisted
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", subject)
	fmt.Fprintf(&b, "Proposed by %s as %s on %s; relanded as a person's commit onto\n", pr.Author, pr.HeadSHA, pr.HeadRef)
	fmt.Fprintf(&b, "%s at %s by `abcd launch reland`, inside the bound declared in\n", def, base)
	fmt.Fprintf(&b, "%s (%s).\n\n", scaffold.ReauthorConfPath, bound)
	fmt.Fprintf(&b, "Supersedes #%d\n\n", pr.Number)
	fmt.Fprintf(&b, "%s\n", trailer)
	return b.String()
}

// nameOnlyDiff lists the paths that differ between two commits or trees.
func nameOnlyDiff(root, a, b string) ([]string, error) {
	out, err := gitutil.RunCappedAsStored(root, gitCap, "diff", "--name-only", "--no-renames", "--no-ext-diff", "--no-textconv", a, b)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(out, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// checkoutIdentity is the author and committer git stamps in root, refused
// when either is a machine's (the attribution gate's list) or incomplete.
func checkoutIdentity(root string) (string, error) {
	var author string
	for _, role := range []identity.Role{identity.RoleAuthor, identity.RoleCommitter} {
		v := "GIT_AUTHOR_IDENT"
		if role == identity.RoleCommitter {
			v = "GIT_COMMITTER_IDENT"
		}
		ident, err := gitutil.LocalGit(root, "var", v)
		if err != nil {
			return "", refuse("git has no complete %s identity in this checkout (%v): set user.name and user.email to your own", role, err)
		}
		name, email, ok := splitIdent(ident)
		if !ok {
			return "", refuse("git's %s identity in this checkout is not a name and an address", role)
		}
		if identity.IsToolIdentity(role, name, email) {
			return "", refuse("this checkout commits as %s <%s>, a machine identity the attribution gate refuses as %s: "+
				"a reland is a person's commit, so set user.name and user.email to your own", name, email, role)
		}
		if role == identity.RoleAuthor {
			author = name + " <" + email + ">"
		}
	}
	return author, nil
}

// splitIdent reads `Name <email> <time> <zone>`, git var's form.
func splitIdent(ident string) (name, email string, ok bool) {
	open := strings.LastIndex(ident, " <")
	end := strings.LastIndex(ident, ">")
	if open < 0 || end < open {
		return "", "", false
	}
	name, email = strings.TrimSpace(ident[:open]), ident[open+2:end]
	return name, email, name != "" && email != ""
}

// short is a commit's first twelve characters, for a message.
func short(sha string) string { return sha[:min(12, len(sha))] }
