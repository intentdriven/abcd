package landing

// facilitator.go is the merge gate (itd-2610150602547271, spc-2610161609350616
// step 4): the one decision `abcd land` and the build loop's arm both ask
// before arming a pull request's merge. It replaces ruling AM1 (arm only where
// a ruleset requires a person's approval), whose reading of approvals it keeps
// for a contributor's pull request and for human mode, counting only an
// approval of the head commit by a maintainer other than the author.
//
// Its inputs are the facilitator mode as the default branch's committed
// .abcd/config.json and the machine layer declare it (readFacilitatorAt), the
// ruleset mirror's merge rule at the same tip, the author's permission on the
// repository, the pull request's reviews, the technical facilitator's recorded
// approval (RecordApproval), and in auto a review receipt naming the head
// (RecordReview). The table, in order:
//
//   - an author whose permission cannot be read, or a custom role whose base
//     cannot be read, is unverified: nothing is armed, whatever the approvals;
//   - a contributor's pull request (Read or Triage, or no permission) arms
//     nothing until an approval of its head counts, in either mode; then the
//     mode decides as for a maintainer's;
//   - human arms a maintainer's pull request only once an approval of its head
//     counts, and until then says it waits for the technical facilitator's;
//   - auto arms a maintainer's pull request only when a review receipt names
//     its head, and otherwise names the commit with no recorded review pair.
//
// An armed merge goes by the mirror's method: the merge queue's where one
// gates the default branch, auto-merge (a merge commit) where none does. abcd
// adds no approval rule or code-owner file to the forge. Every forge read here
// is a GET; a forge answer that cannot be read counts for nothing, so an
// unreadable answer never lets a merge through.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ApprovalsRelDir holds the technical facilitator's recorded approvals, one
// file per commit named by its full object name, beside the review receipts.
// The forge refuses an author's approval of their own pull request, and the
// facilitator authors most of the pull requests abcd opens, so their approval
// is also an act abcd records (`abcd land --approve`), run by a person; the
// shell guard refuses it to an agent.
const ApprovalsRelDir = TierRelDir + "/approvals"

// The author's role on the repository, as the merge gate reads it.
const (
	RoleMaintainer  = "maintainer"
	RoleContributor = "contributor"
	RoleUnverified  = "unverified"
)

// ForgeCall runs the forge client with args and returns its stdout, or an
// error carrying its stderr (so a 404 can be told from any other failure).
type ForgeCall func(args ...string) (string, error)

// maxReviews is the page of reviews the gate reads. A pull request with that
// many could hold a later review beyond the page, so a full page counts none.
const maxReviews = 100

// permissionArgs reads login's permission on the repository repo
// ("owner/name", or gh's "{owner}/{repo}" for the checkout's own): role_name
// is admin, maintain, write, triage, read or a custom role's name, and
// permission the legacy level (admin, write, read, none) a custom role's base
// maps to. Pinned in ghcontract (collaborator-permission).
func permissionArgs(repo, login string) []string {
	return []string{"api", "repos/" + repo + "/collaborators/" + login + "/permission"}
}

// pullArgs reads pull request n: its author (user.login). Pinned in
// ghcontract (pull-request).
func pullArgs(repo, n string) []string { return []string{"api", "repos/" + repo + "/pulls/" + n} }

// reviewsArgs reads pull request n's reviews, oldest first, a page of
// maxReviews: each one's reviewer (user.login), state and the commit it
// reviewed (commit_id). Pinned in ghcontract (pull-request-review).
func reviewsArgs(repo, n string) []string {
	return []string{"api", fmt.Sprintf("repos/%s/pulls/%s/reviews?per_page=%d", repo, n, maxReviews)}
}

type permissionAnswer struct {
	Permission string `json:"permission"`
	RoleName   string `json:"role_name"`
}

type forgeUser struct {
	Login string `json:"login"`
}

type pullAnswer struct {
	User *forgeUser `json:"user"`
}

type reviewAnswer struct {
	User     *forgeUser `json:"user"`
	State    string     `json:"state"`
	CommitID string     `json:"commit_id"`
}

// notFound reports whether the forge client failed on the forge's 404, which
// gh reports as "(HTTP 404)" on stderr.
func notFound(err error) bool { return err != nil && strings.Contains(err.Error(), "(HTTP 404)") }

// forgeWhy is a forge failure as a gate's reason clause: home-redacted,
// credential-free, sanitised and bounded.
func forgeWhy(err error) string {
	t := termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	if len(t) > maxForgeErrorText {
		t = t[:maxForgeErrorText]
	}
	return t
}

// readRole is the one seam that says who is a maintainer: login's permission
// on the repository as the forge reports it. role_name admin, maintain and
// write are maintainers; triage and read contributors; a 404 (the forge knows
// no permission) or no role at all a contributor. Any other role_name is a
// custom role, judged by its base through the legacy level the forge maps it
// to (admin or write: Write, Maintain or Admin at base; read or none: Read or
// Triage, or nothing); a custom role with no such level is unverified, and so
// is a permission the forge cannot be asked for (any other failure, an answer
// that does not parse, a login the forge could not hold). why says what was
// read, for a reason clause.
func readRole(forge ForgeCall, repo, login string) (role, why string) {
	if !ownerName(login) {
		return RoleUnverified, fmt.Sprintf("the account %s is not a name the forge's permission read can take", clip(login))
	}
	out, err := forge(permissionArgs(repo, login)...)
	switch {
	case notFound(err):
		return RoleContributor, login + " has no permission on the repository (the forge answers 404)"
	case err != nil:
		return RoleUnverified, "the forge could not be asked for " + login + "'s permission: " + forgeWhy(err)
	}
	var a permissionAnswer
	if json.Unmarshal([]byte(out), &a) != nil {
		return RoleUnverified, "the forge's answer for " + login + "'s permission does not parse"
	}
	r := strings.ToLower(a.RoleName)
	switch r {
	case "admin", "maintain", "write":
		return RoleMaintainer, login + " has " + r + " permission"
	case "triage", "read":
		return RoleContributor, login + " has " + r + " permission"
	case "", "none":
		return RoleContributor, login + " has no role on the repository"
	}
	custom := termsafe.Sanitize(a.RoleName)
	switch strings.ToLower(a.Permission) {
	case "admin", "write":
		return RoleMaintainer, login + " has the custom role " + clip(custom) + ", based on write access or more"
	case "read", "none":
		return RoleContributor, login + " has the custom role " + clip(custom) + ", based on read or triage access"
	}
	return RoleUnverified, login + " has the custom role " + clip(custom) + ", whose base role cannot be read"
}

// readAuthor reads pull request n's author and their role.
func readAuthor(forge ForgeCall, repo, n string) (login, role, why string) {
	out, err := forge(pullArgs(repo, n)...)
	if err != nil {
		return "", RoleUnverified, "the forge could not be asked who opened pull request #" + n + ": " + forgeWhy(err)
	}
	var a pullAnswer
	if json.Unmarshal([]byte(out), &a) != nil || a.User == nil || a.User.Login == "" {
		return "", RoleUnverified, "the forge's answer naming pull request #" + n + "'s author does not parse"
	}
	role, why = readRole(forge, repo, a.User.Login)
	return a.User.Login, role, why
}

// countedApproval is the approval of head that counts, and by whom: the
// technical facilitator's recorded approval (approvalRecord), else the latest
// review of a reviewer other than the author that approves head, by a
// maintainer (readRole). An earlier commit's approval counts for nothing, and
// so does an approval a later review of the same reviewer superseded. "" with
// why when none counts.
func countedApproval(forge ForgeCall, repoRoot, repo, n, head, author string) (by, why string) {
	switch p, err := approvalRecord(repoRoot, head); {
	case err != nil:
		why = "the technical facilitator's recorded approval cannot be read (" + forgeWhy(err) + "); "
	case p != "":
		return "the technical facilitator's recorded approval (abcd land --approve)", ""
	}
	out, err := forge(reviewsArgs(repo, n)...)
	if err != nil {
		return "", why + "the forge could not be asked for pull request #" + n + "'s reviews: " + forgeWhy(err)
	}
	var reviews []reviewAnswer
	if json.Unmarshal([]byte(out), &reviews) != nil {
		return "", why + "the forge's answer listing pull request #" + n + "'s reviews does not parse"
	}
	if len(reviews) >= maxReviews {
		return "", why + fmt.Sprintf("pull request #%s has %d reviews or more, a later one may sit beyond the page read, so none counts", n, maxReviews)
	}
	// The latest submitted review of each reviewer, in the order they first
	// reviewed. A pending review is the reviewer's own draft, not a review.
	latest := map[string]reviewAnswer{}
	var order []string
	for _, r := range reviews {
		if r.User == nil || r.User.Login == "" || r.State == "PENDING" {
			continue
		}
		k := strings.ToLower(r.User.Login)
		if _, ok := latest[k]; !ok {
			order = append(order, k)
		}
		latest[k] = r
	}
	var notes []string
	for _, k := range order {
		r := latest[k]
		switch {
		case r.State != "APPROVED":
			continue
		case strings.EqualFold(r.User.Login, author):
			notes = append(notes, r.User.Login+" is its author")
			continue
		case r.CommitID != head:
			notes = append(notes, r.User.Login+" approved "+short(r.CommitID)+", not the head")
			continue
		}
		role, rwhy := readRole(forge, repo, r.User.Login)
		if role == RoleMaintainer {
			return r.User.Login + "'s approving review (" + rwhy + ")", ""
		}
		notes = append(notes, r.User.Login+"'s approval is not by someone with write access ("+rwhy+")")
	}
	if len(notes) == 0 {
		return "", why + "no review approves " + short(head)
	}
	return "", why + "no approval of " + short(head) + " counts: " + strings.Join(notes, "; ")
}

// approvalRecord finds the technical facilitator's recorded approval of sha in
// any worktree git lists for the repository at repoRoot, as reviewReceipt
// finds a review receipt, and returns its path, or "" when none names sha.
func approvalRecord(repoRoot, sha string) (string, error) {
	return findReceipt(repoRoot, ApprovalsRelDir, sha)
}

// approvalEntry is a recorded approval's content.
type approvalEntry struct {
	Commit     string `json:"commit"`
	Approval   string `json:"approval"`
	RecordedAt string `json:"recorded_at"`
}

// RecordApproval records the technical facilitator's approval of the commit
// sha, the full object name of a commit the repository at repoRoot holds, in
// that checkout's local tier, and returns the record's path. It is the act
// `abcd land --approve` runs for a person; nothing in abcd runs it on its own,
// and the shell guard refuses it to an agent.
func RecordApproval(repoRoot, sha string) (string, error) {
	if !fullSHA.MatchString(sha) {
		return "", fmt.Errorf("%s is not a commit's full object name", clip(sha))
	}
	if _, err := gitutil.Run(repoRoot, "cat-file", "-e", sha+"^{commit}"); err != nil {
		return "", fmt.Errorf("%s is not a commit this repository holds", sha)
	}
	tier := filepath.Join(repoRoot, filepath.FromSlash(TierRelDir))
	if fi, err := os.Lstat(tier); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%s/ is not a directory in this checkout, so there is nowhere to keep the approval", TierRelDir)
	}
	data, err := json.MarshalIndent(approvalEntry{Commit: sha, Approval: "the technical facilitator", RecordedAt: time.Now().UTC().Format(time.RFC3339)}, "", "  ")
	if err != nil {
		return "", err
	}
	// Written through the checkout's root, as the review receipt is.
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.MkdirAll(ApprovalsRelDir, 0o700); err != nil {
		return "", fmt.Errorf("the approvals directory cannot be made inside this checkout: %w", err)
	}
	rel := ApprovalsRelDir + "/" + sha
	if err := fsutil.WriteFileAtomicInRoot(root, rel, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("the approval cannot be written inside this checkout: %w", err)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(rel)), nil
}

// gateFacts are the merge gate's inputs for one pull request.
type gateFacts struct {
	mode Facilitator
	rule MergeRule
	def  string
	head string
	// author, role and roleWhy are the pull request's author and their role
	// (readRole).
	author, role, roleWhy string
	// approvedBy names the approval of head that counts, "" for none, and
	// approvalWhy why none does.
	approvedBy, approvalWhy string
	// reviewed is whether a review receipt names head, and reviewWhy why
	// the receipts could not be read, "" when they were.
	reviewed  bool
	reviewWhy string
}

// gateDecision is the merge gate's answer: whether to arm the merge and with
// which forge flag (method), and why, in words a step reports.
type gateDecision struct {
	arm    bool
	method string
	why    string
}

// modeText names the mode and the layer that set it, sanitised: its origin
// names a file, or a refused home's reason.
func modeText(m Facilitator) string {
	return termsafe.Sanitize(fmt.Sprintf("facilitator.mode is %s (%s layer, %s)", m.Mode, m.Layer, m.Origin))
}

// mergeGate decides whether a pull request's merge is armed, by the table in
// this file's doc. It is the one place that decision is made.
func mergeGate(f gateFacts) gateDecision {
	head := short(f.head)
	switch {
	case f.role != RoleMaintainer && f.role != RoleContributor:
		return gateDecision{why: "its author's role on the repository is unverified (" + f.roleWhy + "), so nothing is armed, whatever approvals it has"}
	case f.role == RoleContributor && f.approvedBy == "":
		return gateDecision{why: "its author is a contributor (" + f.roleWhy + "), so an approval of " + head + " by someone with write access is needed before anything is armed, in either facilitator mode: " + f.approvalWhy}
	case f.mode.Mode != ModeAuto && f.approvedBy == "":
		return gateDecision{why: modeText(f.mode) + ", so it waits for the technical facilitator's approval of " + head +
			" (a forge review approving that commit by someone with write access other than its author, or abcd land --approve run by the facilitator): " + f.approvalWhy}
	case f.mode.Mode == ModeAuto && f.reviewWhy != "":
		return gateDecision{why: modeText(f.mode) + ", but the review receipts cannot be read (" + f.reviewWhy + "), so whether a review pair passed " + head + " is unknown and nothing is armed"}
	case f.mode.Mode == ModeAuto && !f.reviewed:
		return gateDecision{why: modeText(f.mode) + ", but no review receipt names " + f.head + ": the head has no recorded review pair (record it with abcd land --record-review), so nothing is armed"}
	}
	why := modeText(f.mode)
	if f.approvedBy != "" {
		why += "; approved by " + f.approvedBy
	}
	if f.mode.Mode == ModeAuto {
		why += "; a review receipt names " + head
	}
	if f.rule.Method == "" {
		return gateDecision{arm: true, method: "--merge", why: why + "; auto-merge armed (merge commit): no merge queue gates " + f.def + " (per the ruleset mirror)"}
	}
	return gateDecision{arm: true, method: MergeMethods[f.rule.Method], why: why + "; auto-merge armed through " + f.def + "'s merge queue (" + strings.ToLower(f.rule.Method) + ")"}
}

// MergeQuery is what the merge gate is asked about one pull request.
type MergeQuery struct {
	// RepoRoot is a checkout of the repository: its object store holds the
	// default branch's tip, and its worktrees the receipts and approvals.
	RepoRoot string
	// Roots supplies the machine layer (Home, HomeRefused); its Repo is not
	// read.
	Roots layered.Roots
	// Tip is the default branch's tip as just fetched, and TipName how it is
	// named ("origin/main"); Default is the branch's name.
	Tip, TipName, Default string
	// Rule is the ruleset mirror's merge rule at Tip.
	Rule MergeRule
	// Repo is the repository as the forge names it ("owner/name"), or gh's
	// "{owner}/{repo}" for the checkout's own.
	Repo string
	// PR is the pull request's number and Head its head commit, the one the
	// landing gated.
	PR, Head string
	// Forge runs the forge client, its error carrying stderr.
	Forge ForgeCall
}

// MergeDecision is the merge gate's answer to a MergeQuery: whether to arm,
// with which forge flag, and why.
type MergeDecision struct {
	Arm    bool
	Method string
	Why    string
}

// ErrModeUnreadable wraps a facilitator.mode the gate could not read, whose
// text a front door masks before printing: nothing is armed.
var ErrModeUnreadable = errors.New("facilitator.mode cannot be read")

// DecideMerge gathers the merge gate's inputs for q and decides. Its error is
// only a facilitator.mode that cannot be read (wrapping ErrModeUnreadable,
// sanitised); a forge answer that cannot be read is a reason in the decision,
// never an arming.
func DecideMerge(q MergeQuery) (MergeDecision, error) {
	mode, err := readFacilitatorAt(q.Roots, q.RepoRoot, q.Tip, q.TipName)
	if err != nil {
		return MergeDecision{}, fmt.Errorf("%w: %s", ErrModeUnreadable, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	f := gateFacts{mode: mode, rule: q.Rule, def: q.Default, head: q.Head}
	f.author, f.role, f.roleWhy = readAuthor(q.Forge, q.Repo, q.PR)
	if f.role == RoleContributor || (f.role == RoleMaintainer && mode.Mode != ModeAuto) {
		f.approvedBy, f.approvalWhy = countedApproval(q.Forge, q.RepoRoot, q.Repo, q.PR, q.Head, f.author)
	}
	if mode.Mode == ModeAuto {
		p, err := reviewReceipt(q.RepoRoot, q.Head)
		f.reviewed = err == nil && p != ""
		if err != nil {
			f.reviewWhy = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		}
	}
	d := mergeGate(f)
	return MergeDecision{Arm: d.arm, Method: d.method, Why: d.why}, nil
}
