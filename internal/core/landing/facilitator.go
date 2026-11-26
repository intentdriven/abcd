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
// (RecordReview). A recorded approval or a review receipt is read from any
// worktree of the repository but one that is a drive lane's or lies inside one
// (insideDriveLane, on the path resolved): one there counts for nothing, and
// whenever the gate reads them the reason names each one it passed over.
// Asked from a checkout that is a drive lane's worktree or lies inside one, or
// about a branch checked out in one, the gate counts none at all, and the
// reason says so. The table, in order:
//
//   - a pull request whose diff, read from its merge base with the default
//     branch's tip to its head, changes the facilitator.mode line of
//     .abcd/config.json (sets, alters or removes it, or leaves it
//     unreadable) arms nothing in any mode, and one whose diff cannot be
//     read for it arms nothing either: the person merges it on the forge's
//     page (adr-2611262044127943);
//   - an author whose permission cannot be read, or a custom role whose base
//     cannot be read, is unverified: nothing is armed, whatever the approvals;
//   - a contributor's pull request (Read or Triage, or no permission) arms
//     nothing until an approval of its head counts, in either mode; then the
//     mode decides as for a maintainer's;
//   - human arms a maintainer's pull request only once an approval of its head
//     counts, and until then says it waits for the technical facilitator's;
//   - auto arms a maintainer's pull request only when a review receipt names
//     its head, and otherwise names the commit with no recorded review pair. A
//     delta's receipt names the head only while the delta stands (delta.go);
//     one that does not stand says why, and that the head needs a full pair;
//   - where auto is the person's own global setting rather than the
//     repository's, it arms only a pull request the forge account abcd runs
//     as opened ("only your own", ViewerQuery): anyone else's, or one where
//     the forge cannot say which account that is, is judged as human judges
//     it, waiting for an approval of its head or for its author to land it.
//
// An armed merge goes by the mirror's method: the merge queue's where one
// gates the default branch, auto-merge (a merge commit) where none does. abcd
// adds no approval rule or code-owner file to the forge. Every forge read here
// is a GET; a forge answer that cannot be read counts for nothing, so an
// unreadable answer never lets a merge through.

import (
	"context"
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

// ViewerQuery asks the forge which account its client runs as, the one a
// person's own global auto arms for ("only your own", adr-2611262044127943).
const ViewerQuery = `query{viewer{login}}`

// viewerArgs are the forge client's arguments for ViewerQuery.
func viewerArgs() []string { return []string{"api", "graphql", "-f", "query=" + ViewerQuery} }

// viewerAnswer is the forge's answer to ViewerQuery.
type viewerAnswer struct {
	Data struct {
		Viewer *struct {
			Login string `json:"login"`
		} `json:"viewer"`
	} `json:"data"`
}

// readViewer says why pull request n, opened by author, is not known to be
// one the account the forge client runs as opened: "" when it is, the two
// logins compared as the forge compares them, without regard to case. unread
// says the forge could not be asked, or its answer does not parse, which
// counts as not the author's, so a failed read never arms.
func readViewer(forge ForgeCall, n, author string) (notOwn string, unread bool) {
	out, err := forge(viewerArgs()...)
	if err != nil {
		return "the forge could not say which account abcd runs as (" + forgeWhy(err) + "), so #" + n + " is not known to be your own", true
	}
	var a viewerAnswer
	if json.Unmarshal([]byte(out), &a) != nil || a.Data.Viewer == nil || a.Data.Viewer.Login == "" {
		return "the forge's answer naming which account abcd runs as does not parse, so #" + n + " is not known to be your own", true
	}
	if strings.EqualFold(a.Data.Viewer.Login, author) {
		return "", false
	}
	return "#" + n + " was opened by " + loginText(author) + ", not by " + loginText(a.Data.Viewer.Login) + ", the account abcd runs as", false
}

// loginText is a forge login as a reason shows it: as it is where it is one
// the forge could hold (ownerName, within maxLoginRunes).
func loginText(login string) string {
	if ownerName(login) && len(login) <= maxLoginRunes {
		return login
	}
	return "an account the forge names in a form land does not show"
}

// modeLineChange says why the pull request whose head is head changes the
// committed facilitator.mode line, read at its merge base with the default
// branch's tip and at head (committedModeLine), "" when the line is the same
// at both; unread says the diff could not be read for it, which arms nothing
// either. The line is the person's own act to change: abcd never arms such a
// merge in any mode, and the person merges it on the forge's page.
func modeLineChange(root, tip, tipName, head string) (why string, unread bool) {
	const never = "abcd never arms a change to that line, in any facilitator mode: the person merges it on the forge's page, so nothing is armed"
	cannot := func(what string) (string, bool) {
		return "whether this pull request changes the facilitator.mode line in .abcd/config.json cannot be judged (" + what + "), and " + never, true
	}
	if !gitutil.IsFullSHA(tip) || !gitutil.IsFullSHA(head) {
		return cannot("the head or " + tipName + " is not a full object name")
	}
	base, err := gitutil.Run(root, "merge-base", tip, head)
	if err != nil || !gitutil.IsFullSHA(base) {
		return cannot("no merge base of " + short(head) + " and " + tipName + " can be read")
	}
	ctx := context.Background()
	was, _, err := committedModeLine(ctx, root, base)
	if err != nil {
		return cannot("at the merge base " + short(base) + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	is, _, err := committedModeLine(ctx, root, head)
	if err != nil {
		return cannot("at the head " + short(head) + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	if was == is {
		return "", false
	}
	unset := func(v string) string {
		if v == "" {
			return "unset"
		}
		return v
	}
	return "it changes the facilitator.mode line in .abcd/config.json (" + unset(was) + " at the merge base " + short(base) + ", " + unset(is) + " at the head " + short(head) + "), and " + never, false
}

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
// credential-free, sanitised and bounded at a character (boundForgeText).
func forgeWhy(err error) string {
	return boundForgeText(termsafe.Sanitize(fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))))
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

// Review is one review as a page of a pull request's reviews lists it, oldest
// first: its reviewer's login ("" for none the forge names), its state, and
// the commit it reviewed. Index is its place on the page, which LatestReviews
// sets so a caller can go back to the rest of what it read.
type Review struct {
	Login, State, Commit string
	Index                int
}

// LatestReviews is the merge gate's reading of a page of reviews: the latest
// submitted review of each reviewer, in the order they first reviewed. A
// pending review is the reviewer's own draft, not a review, and a review that
// names no reviewer is no one's; every other state, a dismissed one included,
// is the reviewer's latest word, so it supersedes an earlier approval.
// Reviewers are told apart without regard to case, as the forge's logins are.
func LatestReviews(page []Review) []Review {
	latest := map[string]Review{}
	var order []string
	for i, r := range page {
		if r.Login == "" || r.State == "PENDING" {
			continue
		}
		r.Index = i
		k := strings.ToLower(r.Login)
		if _, ok := latest[k]; !ok {
			order = append(order, k)
		}
		latest[k] = r
	}
	out := make([]Review, 0, len(order))
	for _, k := range order {
		out = append(out, latest[k])
	}
	return out
}

// Approves says whether r, a reviewer's latest review, approves head on a pull
// request author opened, before anything is asked of the reviewer's access: it
// is APPROVED, of head, by someone other than author. note says why one that
// approved something does not count, "" for a review that approves nothing;
// with no author known, no review can be told from the author's own, so none
// counts.
func (r Review) Approves(head, author string) (note string, ok bool) {
	switch {
	case r.State != "APPROVED":
		return "", false
	case author == "":
		return "the pull request's author is not known, so " + r.Login + "'s approval cannot be told from the author's own", false
	case strings.EqualFold(r.Login, author):
		return r.Login + " is its author", false
	case r.Commit == "":
		return r.Login + "'s approval names no commit", false
	case r.Commit != head:
		return r.Login + " approved " + short(r.Commit) + ", not the head", false
	}
	return "", true
}

// countedApproval is the approval of head that counts, and by whom: the
// technical facilitator's recorded approval (approvalRecord), else the latest
// review of a reviewer other than the author that approves head, by a
// maintainer (readRole). An earlier commit's approval counts for nothing, and
// so does an approval a later review of the same reviewer superseded. "" with
// why when none counts. passed names the recorded approvals of head it passed
// over in a drive lane's worktree, "" for none, whatever the answer. When
// fromLane is set (fromDriveLane: the gate was asked from a drive lane's
// worktree or about a branch checked out in one, repoRoot's own among them) no
// recorded approval is read, and a forge review alone can count: none counting
// is an answer it could not read, never a refusal.
func countedApproval(forge ForgeCall, repoRoot, repo, n, head, author string, fromLane bool) (by, why, passed string, unread bool) {
	if fromLane {
		// The gate's reason names the checkout and says why (fromDriveLane).
		why, unread = "no recorded approval was read from a drive lane's worktree; ", true
	} else {
		p, lanes, err := approvalRecords(repoRoot, head)
		if len(lanes) > 0 {
			passed = "the recorded approval of " + short(head) + " at " + shownPaths(lanes) + " was passed over, since no approval in a drive lane's worktree counts"
		}
		switch {
		case err != nil:
			why, unread = "the technical facilitator's recorded approval cannot be read ("+forgeWhy(err)+"); ", true
		case p != "":
			return "the technical facilitator's recorded approval (abcd land --approve)", "", passed, false
		}
	}
	out, err := forge(reviewsArgs(repo, n)...)
	if err != nil {
		return "", why + "the forge could not be asked for pull request #" + n + "'s reviews: " + forgeWhy(err), passed, true
	}
	var reviews []reviewAnswer
	if json.Unmarshal([]byte(out), &reviews) != nil {
		return "", why + "the forge's answer listing pull request #" + n + "'s reviews does not parse", passed, true
	}
	if len(reviews) >= maxReviews {
		return "", why + fmt.Sprintf("pull request #%s has %d reviews or more, a later one may sit beyond the page read, so none counts", n, maxReviews), passed, true
	}
	page := make([]Review, len(reviews))
	for i, r := range reviews {
		page[i] = Review{State: r.State, Commit: r.CommitID}
		if r.User != nil {
			page[i].Login = r.User.Login
		}
	}
	var notes []string
	for _, r := range LatestReviews(page) {
		if note, ok := r.Approves(head, author); !ok {
			if note != "" {
				notes = append(notes, note)
			}
			continue
		}
		role, rwhy := readRole(forge, repo, r.Login)
		if role == RoleMaintainer {
			return r.Login + "'s approving review (" + rwhy + ")", "", passed, false
		}
		unread = unread || role == RoleUnverified
		notes = append(notes, r.Login+"'s approval is not by someone with write access ("+rwhy+")")
	}
	if len(notes) == 0 {
		return "", why + "no review approves " + short(head), passed, unread
	}
	return "", why + "no approval of " + short(head) + " counts: " + strings.Join(notes, "; "), passed, unread
}

// approvalRecord finds the technical facilitator's recorded approval of sha in
// any worktree git lists for the repository at repoRoot, as findReceipts finds
// a review receipt, and returns its path, or "" when none names sha. One in a
// drive lane's worktree is passed over, and every one is when repoRoot is a
// drive lane's worktree or lies inside one.
func approvalRecord(repoRoot, sha string) (string, error) {
	return findReceipt(repoRoot, ApprovalsRelDir, sha)
}

// approvalRecords is approvalRecord with the paths of the approvals it passed
// over in a drive lane's worktree, for a reason to name.
func approvalRecords(repoRoot, sha string) (path string, passed []string, err error) {
	found, err := findReceipts(repoRoot, ApprovalsRelDir, sha)
	for _, f := range found {
		if f.lane {
			passed = append(passed, f.path)
		}
	}
	path, err = firstReceipt(found, err)
	return path, passed, err
}

// ApprovalRecord is approvalRecord for a reader of past landings
// (internal/core/landings), bound to ctx: the technical facilitator's recorded
// approval of sha in any of wts, by findReceipt's rule (a regular file, never a
// symlink, and never one git tracks). wts nil lists the worktrees git knows for
// the repository at repoRoot, as findReceipt does; a reader asking about many
// commits lists them once and passes them in. Either way one in a drive lane's
// worktree is passed over, and every one is when repoRoot is a drive lane's
// worktree or lies inside one (insideDriveLane).
func ApprovalRecord(ctx context.Context, repoRoot, sha string, wts []gitutil.Worktree) (string, error) {
	if !fullSHA.MatchString(sha) {
		return "", fmt.Errorf("%s is not a commit's full object name", clip(sha))
	}
	if wts == nil {
		found, err := findReceiptsContext(ctx, repoRoot, ApprovalsRelDir, sha)
		return firstReceipt(found, err)
	}
	return firstReceipt(receiptsIn(ctx, repoRoot, wts, ApprovalsRelDir, sha))
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
// and the shell guard refuses it to an agent. A checkout that is a drive
// lane's worktree or lies inside one (insideDriveLane) is refused
// (ErrDriveLaneRecord), since the gate never counts an approval there.
func RecordApproval(repoRoot, sha string) (string, error) {
	if err := refuseInDriveLane(repoRoot); err != nil {
		return "", err
	}
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
	// approvalWhy why none does; approvalUnread says a read it rests on failed.
	approvedBy, approvalWhy string
	approvalUnread          bool
	// approvalPassed says which recorded approvals of head were passed over in
	// a drive lane's worktree, "" for none; like reviewPassed, the reason ends
	// with it whatever the answer.
	approvalPassed string
	// reviewed is whether a review receipt that stands names head, and
	// reviewWhy why the receipts could not be read, "" when they were.
	reviewed  bool
	reviewWhy string
	// reviewUnread says the gate could not judge the receipts, so whether one stands
	// is unknown; a receipt read and found wanting is the head refused.
	reviewUnread bool
	// deltaWhy is why the delta receipt naming head does not stand, "" when
	// there is none or it does; reviewNote says what a standing one rests on.
	deltaWhy, reviewNote string
	// reviewPassed says which review receipts naming head were passed over in
	// a drive lane's worktree, "" for none; the reason ends with it, whatever
	// the answer, so a person sees one was not counted.
	reviewPassed string
	// modeLine is why the pull request's diff changes the committed
	// facilitator.mode line, or why that cannot be judged, "" when it does
	// not (modeLineChange); modeLineUnread says the read failed.
	modeLine       string
	modeLineUnread bool
	// notOwn is why, where auto is the person's own global setting, the pull
	// request is not known to be one the account abcd runs as opened, ""
	// when it is or the repository's own value decides (readViewer);
	// viewerUnread says the forge could not be asked.
	notOwn       string
	viewerUnread bool
	// fromLane says the gate was asked from a checkout that is a drive lane's
	// worktree or lies inside one, or about a branch checked out in one
	// (fromDriveLane), "" when it was not: it counted no review receipt or
	// recorded approval, and the reason ends with it.
	fromLane string
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
	// auto is the mode the gate judges by: a person's own global auto judges
	// a pull request it does not know to be their own as human does.
	auto := f.mode.Mode == ModeAuto && f.notOwn == ""
	switch {
	case f.modeLine != "":
		return gateDecision{why: f.modeLine}
	case f.role != RoleMaintainer && f.role != RoleContributor:
		return gateDecision{why: "its author's role on the repository is unverified (" + f.roleWhy + "), so nothing is armed, whatever approvals it has"}
	case f.role == RoleContributor && f.approvedBy == "":
		return gateDecision{why: "its author is a contributor (" + f.roleWhy + "), so an approval of " + head + " by someone with write access is needed before anything is armed, in either facilitator mode: " + f.approvalWhy}
	case f.notOwn != "" && f.approvedBy == "":
		return gateDecision{why: modeText(f.mode) + ", and a person's own global auto arms only a pull request your own account opened: " + f.notOwn +
			", so it waits for an approval of " + head + " (a forge review approving that commit by someone with write access other than its author, or abcd land --approve run by the facilitator), or for its author to land it: " + f.approvalWhy}
	case !auto && f.approvedBy == "":
		return gateDecision{why: modeText(f.mode) + ", so it waits for the technical facilitator's approval of " + head +
			" (a forge review approving that commit by someone with write access other than its author, or abcd land --approve run by the facilitator): " + f.approvalWhy}
	case auto && f.reviewWhy != "" && f.reviewUnread:
		return gateDecision{why: modeText(f.mode) + ", but the review receipts cannot be read (" + f.reviewWhy + "), so whether a review pair passed " + head + " is unknown and nothing is armed"}
	case auto && f.reviewWhy != "":
		return gateDecision{why: modeText(f.mode) + ", but no review receipt for " + head + " stands (" + f.reviewWhy + "): the head needs a recorded review pair (abcd land --record-review), so nothing is armed"}
	case auto && f.deltaWhy != "" && f.reviewUnread:
		return gateDecision{why: modeText(f.mode) + ", but whether the delta review recorded for " + head + " stands cannot be judged (" + f.deltaWhy + "), so nothing is armed"}
	case auto && f.deltaWhy != "":
		return gateDecision{why: modeText(f.mode) + ", but the delta review recorded for " + head + " does not stand (" + f.deltaWhy + "), so it needs a full review pair (record it with abcd land --record-review), and nothing is armed"}
	case auto && !f.reviewed:
		return gateDecision{why: modeText(f.mode) + ", but no review receipt names " + f.head + ": the head has no recorded review pair (record it with abcd land --record-review), so nothing is armed"}
	}
	why := modeText(f.mode)
	if f.notOwn != "" {
		why += ", judged as human since " + f.notOwn
	}
	if f.approvedBy != "" {
		why += "; approved by " + f.approvedBy
	}
	if auto {
		why += "; a review receipt names " + head
		if f.reviewNote != "" {
			why += " (" + f.reviewNote + ")"
		}
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
	// From is the checkout the landing was asked from, when it is not
	// RepoRoot's (abcd land run in one checkout for a branch checked out in
	// another); "" is RepoRoot itself. When From, or else RepoRoot, is a drive
	// lane's worktree or lies inside one, no review receipt or recorded
	// approval is read (fromDriveLane).
	From string
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
	// Unread marks a decision not to arm that rests on an answer the gate could
	// not read: a forge read that failed, or receipts it could not judge. It is
	// no refusal of the head, so nothing is withdrawn on it.
	Unread bool
	// Mode is the facilitator mode the gate ran in (ModeAuto or ModeHuman),
	// as it read it.
	Mode string
	// AuthorCanWrite says the forge named the pull request's author as one
	// who can write to the repository (RoleMaintainer); false for a
	// contributor and for an author whose role is unverified.
	AuthorCanWrite bool
	// ApprovalCounted says a person's approval of the head counted in the
	// answer: a forge review or a recorded approval the gate found standing.
	ApprovalCounted bool
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
	f := gateFacts{mode: mode, rule: q.Rule, def: q.Default, head: q.Head, fromLane: fromDriveLane(q.RepoRoot, q.From)}
	f.modeLine, f.modeLineUnread = modeLineChange(q.RepoRoot, q.Tip, q.TipName, q.Head)
	if f.modeLine != "" {
		return decided(f), nil
	}
	f.author, f.role, f.roleWhy = readAuthor(q.Forge, q.Repo, q.PR)
	if mode.Mode == ModeAuto && mode.Layer != layered.Repo.String() && f.role != RoleUnverified {
		f.notOwn, f.viewerUnread = readViewer(q.Forge, q.PR, f.author)
	}
	auto := mode.Mode == ModeAuto && f.notOwn == ""
	if f.role == RoleContributor || (f.role == RoleMaintainer && !auto) {
		f.approvedBy, f.approvalWhy, f.approvalPassed, f.approvalUnread = countedApproval(q.Forge, q.RepoRoot, q.Repo, q.PR, q.Head, f.author, f.fromLane != "")
	}
	switch {
	case !auto:
	case f.fromLane != "":
		// The worktrees a lane lists are the lane's own, and a lane's
		// worktree is the drive's, so no receipt is read: whether one stands
		// is unknown, which is no refusal.
		f.reviewWhy, f.reviewUnread = "none is read from a drive lane's worktree", true
	default:
		proof := readReviewProof(q.RepoRoot, q.Head, q.Tip)
		f.reviewed, f.reviewNote, f.deltaWhy, f.reviewWhy, f.reviewUnread = proof.reviewed, proof.note, proof.deltaWhy, proof.unreadable, proof.unread
		f.reviewPassed = proof.passed
	}
	return decided(f), nil
}

// decided is the gate's answer over f (mergeGate) as a MergeDecision, with
// the facts the gate decided on that a caller reads as fields, never from
// the words of its reason: the mode, whether the author can write to the
// repository and whether a person's approval counted.
func decided(f gateFacts) MergeDecision {
	d := mergeGate(f)
	for _, note := range []string{f.reviewPassed, f.approvalPassed, f.fromLane} {
		if note != "" {
			d.why += "; " + note
		}
	}
	return MergeDecision{Arm: d.arm, Method: d.method, Why: d.why,
		Unread: !d.arm && (f.role == RoleUnverified || f.reviewUnread || f.approvalUnread || f.modeLineUnread || f.viewerUnread),
		Mode:   f.mode.Mode, AuthorCanWrite: f.role == RoleMaintainer, ApprovalCounted: f.approvedBy != ""}
}
