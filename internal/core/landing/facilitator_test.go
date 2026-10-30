package landing

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/ghcontract"
)

// The accounts below are made up and role-neutral: author-a writes,
// maintainer-b maintains, reader-c reads.

const (
	autoConfig  = `{"facilitator":{"mode":"auto"}}` + "\n"
	humanConfig = `{"facilitator":{"mode":"human"}}` + "\n"
)

// TestMergeGateDecisionTable is spc-2610161609350616 step 4's table, decided
// in the one place: an unverified author arms nothing whatever the approvals;
// a contributor's pull request needs a counted approval in either mode, after
// which the mode decides; human arms a maintainer's only once an approval
// counts; auto only once a review receipt names the head; and with no queue
// an armed merge is auto-merge.
func TestMergeGateDecisionTable(t *testing.T) {
	human := Facilitator{Mode: ModeHuman, Layer: "bundled", Origin: "bundled"}
	auto := Facilitator{Mode: ModeAuto, Layer: "repo", Origin: ".abcd/config.json at origin/main"}
	queue := MergeRule{Method: "SQUASH"}
	const head = "0123456789abcdef0123456789abcdef01234567"
	for _, c := range []struct {
		name             string
		mode             Facilitator
		rule             MergeRule
		role, approvedBy string
		reviewed, arm    bool
		method, says     string
	}{
		{"unverified, approved, auto, reviewed", auto, queue, RoleUnverified, "maintainer-b", true, false, "", "unverified"},
		{"unverified, approved, human", human, queue, RoleUnverified, "maintainer-b", false, false, "", "unverified"},
		{"contributor, unapproved, human", human, queue, RoleContributor, "", false, false, "", "by someone with write access is needed"},
		{"contributor, unapproved, auto, reviewed", auto, queue, RoleContributor, "", true, false, "", "by someone with write access is needed"},
		{"contributor, approved, human", human, queue, RoleContributor, "maintainer-b", false, true, "--squash", "merge queue"},
		{"contributor, approved, auto, unreviewed", auto, queue, RoleContributor, "maintainer-b", false, false, "", "no recorded review pair"},
		{"contributor, approved, auto, reviewed", auto, queue, RoleContributor, "maintainer-b", true, true, "--squash", "merge queue"},
		{"maintainer, unapproved, human", human, queue, RoleMaintainer, "", true, false, "", "technical facilitator's approval"},
		{"maintainer, approved, human", human, queue, RoleMaintainer, "maintainer-b", false, true, "--squash", "approved by maintainer-b"},
		{"maintainer, unapproved, auto, unreviewed", auto, queue, RoleMaintainer, "", false, false, "", head},
		{"maintainer, unapproved, auto, reviewed", auto, queue, RoleMaintainer, "", true, true, "--squash", "review receipt names"},
		{"maintainer, approved, human, no queue", human, MergeRule{}, RoleMaintainer, "maintainer-b", false, true, "--merge", "no merge queue gates main"},
		{"maintainer, auto, reviewed, no queue", auto, MergeRule{}, RoleMaintainer, "", true, true, "--merge", "auto-merge armed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := mergeGate(gateFacts{mode: c.mode, rule: c.rule, def: "main", head: head, author: "author-a", role: c.role, roleWhy: "read",
				approvedBy: c.approvedBy, approvalWhy: "no review approves it", reviewed: c.reviewed})
			if d.arm != c.arm || d.method != c.method || !strings.Contains(d.why, c.says) {
				t.Fatalf("mergeGate = %+v, want arm %v method %q saying %q", d, c.arm, c.method, c.says)
			}
		})
	}
}

// TestAnUnreadableReviewReceiptArmsNothingAndSaysWhy: in auto, receipts that
// cannot be read are not taken for a missing receipt; the gate arms nothing
// and its reason names the read that failed.
func TestAnUnreadableReviewReceiptArmsNothingAndSaysWhy(t *testing.T) {
	auto := Facilitator{Mode: ModeAuto, Layer: "repo", Origin: ".abcd/config.json at origin/main"}
	d := mergeGate(gateFacts{mode: auto, rule: MergeRule{Method: "SQUASH"}, def: "main", head: "0123456789abcdef0123456789abcdef01234567",
		author: "author-a", role: RoleMaintainer, roleWhy: "read", reviewed: false, reviewWhy: "git worktree list failed"})
	if d.arm || !strings.Contains(d.why, "review receipts cannot be read (git worktree list failed)") {
		t.Fatalf("mergeGate = %+v; want nothing armed, naming the failed read", d)
	}
}

// roleForge answers one permission read.
func roleForge(answer string, err error) ForgeCall {
	return func(args ...string) (string, error) {
		if len(args) != 2 || args[0] != "api" || args[1] != "repos/o/r/collaborators/author-a/permission" {
			return "", errors.New("unexpected call")
		}
		return answer, err
	}
}

// TestTheMaintainerSeamReadsThePermission is the one seam that says who is a
// maintainer: Write, Maintain and Admin are; Read and Triage are not; a 404 or
// no role is a contributor; a custom role is its base; a permission the forge
// cannot be asked for, an answer that does not parse, or a custom role whose
// base cannot be read is unverified.
func TestTheMaintainerSeamReadsThePermission(t *testing.T) {
	perm := func(p, r string) string {
		return `{"permission":"` + p + `","role_name":"` + r + `","user":{"login":"author-a"}}`
	}
	for _, c := range []struct {
		name   string
		answer string
		err    error
		want   string
	}{
		{"admin", perm("admin", "admin"), nil, RoleMaintainer},
		{"maintain", perm("write", "maintain"), nil, RoleMaintainer},
		{"write", perm("write", "write"), nil, RoleMaintainer},
		{"triage", perm("read", "triage"), nil, RoleContributor},
		{"read", perm("read", "read"), nil, RoleContributor},
		{"no role", perm("none", ""), nil, RoleContributor},
		{"a 404", "", errors.New("gh api: exit status 1 (gh: Not Found (HTTP 404))"), RoleContributor},
		{"a custom role based on write", perm("write", "release-keeper"), nil, RoleMaintainer},
		{"a custom role based on triage", perm("read", "labeller"), nil, RoleContributor},
		{"a custom role whose base cannot be read", perm("", "release-keeper"), nil, RoleUnverified},
		{"a forbidden read", "", errors.New("gh api: exit status 1 (gh: Resource not accessible by integration (HTTP 403))"), RoleUnverified},
		{"an answer that does not parse", "not json", nil, RoleUnverified},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got, why := readRole(roleForge(c.answer, c.err), "o/r", "author-a"); got != c.want || why == "" {
				t.Fatalf("readRole = %q (%s), want %q", got, why, c.want)
			}
		})
	}
	if got, _ := readRole(roleForge("", nil), "o/r", "../../evil"); got != RoleUnverified {
		t.Fatalf("a login the forge could not hold is unverified, never put in a path: %q", got)
	}
}

// TestTheDecodedRESTFieldsArePinned: every field the gate decodes is one the
// forge sends, as ghcontract pins it after a live read-only call.
func TestTheDecodedRESTFieldsArePinned(t *testing.T) {
	for resource, v := range map[string]any{"collaborator-permission": permissionAnswer{}, "pull-request-review": reviewAnswer{}, "pull-request": pullAnswer{}} {
		ty := reflect.TypeOf(v)
		for i := range ty.NumField() {
			name := strings.Split(ty.Field(i).Tag.Get("json"), ",")[0]
			if !ghcontract.RESTField(resource, name) {
				t.Errorf("%s.%s is decoded but not pinned", resource, name)
			}
			if name == "user" && !ghcontract.RESTField(resource, "user.login") {
				t.Errorf("%s.user.login is decoded but not pinned", resource)
			}
		}
	}
}

// reviewsForge answers the reviews read with reviews and every permission
// read from roles (a missing account is a 404).
func reviewsForge(reviews string, roles map[string]string) ForgeCall {
	return func(args ...string) (string, error) {
		switch {
		case len(args) == 2 && args[1] == "repos/o/r/pulls/7/reviews?per_page=100":
			return reviews, nil
		case len(args) == 2 && strings.HasPrefix(args[1], "repos/o/r/collaborators/"):
			login := strings.TrimSuffix(strings.TrimPrefix(args[1], "repos/o/r/collaborators/"), "/permission")
			if r, ok := roles[login]; ok {
				return `{"permission":"write","role_name":"` + r + `"}`, nil
			}
			return "", errors.New("gh: Not Found (HTTP 404)")
		}
		return "", errors.New("unexpected call")
	}
}

// TestOnlyTheLatestApprovalOfTheHeadByAnotherMaintainerCounts: an approval
// counts when it is its reviewer's latest review, approves the head commit,
// and its reviewer is a maintainer other than the author.
func TestOnlyTheLatestApprovalOfTheHeadByAnotherMaintainerCounts(t *testing.T) {
	f := newFixture(t, "")
	head, earlier := f.head(), f.repo.Git("rev-parse", "HEAD~1")
	review := func(login, state, commit string) string {
		return `{"user":{"login":"` + login + `"},"state":"` + state + `","commit_id":"` + commit + `"}`
	}
	list := func(rs ...string) string { return "[" + strings.Join(rs, ",") + "]" }
	roles := map[string]string{"author-a": "write", "maintainer-b": "maintain", "reader-c": "read"}
	for _, c := range []struct {
		name, reviews string
		counts        bool
	}{
		{"a maintainer's approval of the head", list(review("maintainer-b", "APPROVED", head)), true},
		{"an approval of an earlier commit", list(review("maintainer-b", "APPROVED", earlier)), false},
		{"an approval a later review withdrew", list(review("maintainer-b", "APPROVED", head), review("maintainer-b", "CHANGES_REQUESTED", head)), false},
		{"a later approval of the head", list(review("maintainer-b", "CHANGES_REQUESTED", earlier), review("maintainer-b", "APPROVED", head)), true},
		{"the author's own approval", list(review("author-a", "APPROVED", head)), false},
		{"a reader's approval", list(review("reader-c", "APPROVED", head)), false},
		{"an approval by an account with no permission", list(review("stranger-d", "APPROVED", head)), false},
		{"no review", "[]", false},
		{"a full page", "[" + strings.Repeat(review("maintainer-b", "APPROVED", head)+",", maxReviews-1) + review("maintainer-b", "APPROVED", head) + "]", false},
		{"an answer that does not parse", "{", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			by, why := countedApproval(reviewsForge(c.reviews, roles), f.repo.Root(), "o/r", "7", head, "author-a")
			if (by != "") != c.counts || (by == "" && why == "") {
				t.Fatalf("countedApproval = %q, %q; want counted %v", by, why, c.counts)
			}
		})
	}
	// The technical facilitator's recorded approval of the head counts, with
	// no forge review at all.
	if err := os.MkdirAll(filepath.Join(f.repo.Root(), TierRelDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordApproval(f.repo.Root(), head); err != nil {
		t.Fatal(err)
	}
	if by, _ := countedApproval(reviewsForge("[]", roles), f.repo.Root(), "o/r", "7", head, "author-a"); !strings.Contains(by, "recorded approval") {
		t.Fatalf("the recorded approval counts: %q", by)
	}
	if by, _ := countedApproval(reviewsForge("[]", roles), f.repo.Root(), "o/r", "7", earlier, "author-a"); by != "" {
		t.Fatalf("an approval recorded for another commit counts for nothing: %q", by)
	}
}

// landGate lands the fixture's branch with an open pull request at its head
// and returns the enqueue step.
func landGate(t *testing.T, f *fixture) Step {
	t.Helper()
	f.forge.prs = f.prFor(f.head())
	if f.forge.views == nil {
		f.forge.views = []string{viewBlocked, viewQueued}
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	return step(t, res, "enqueue")
}

// reviewed writes the review receipt for the fixture's head, as
// abcd land --record-review does.
func (f *fixture) reviewed(t *testing.T) {
	t.Helper()
	p := filepath.Join(f.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), f.head())
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func wantArmed(t *testing.T, f *fixture, s Step) {
	t.Helper()
	if f.forge.called("pr", "merge", "7", "--auto") != 1 || s.Status != StatusDone {
		t.Fatalf("enqueue step %+v, calls %v: want the merge armed", s, f.forge.calls)
	}
}

func wantLeft(t *testing.T, f *fixture, s Step, says string) {
	t.Helper()
	if f.forge.called("pr", "merge") != 0 || s.Status != StatusLeft || !strings.Contains(s.Note, says) {
		t.Fatalf("enqueue step %+v, calls %v: want nothing armed, saying %q", s, f.forge.calls, says)
	}
}

// With no facilitator.mode anywhere, a maintainer's pull request with no
// approval of its head waits for the technical facilitator's, and arms once a
// maintainer other than the author approves the head.
func TestHumanWaitsForTheFacilitatorsApproval(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.reviews = "[]"
	wantLeft(t, f, landGate(t, f), "waits for the technical facilitator's approval")

	g := newFixture(t, queueRuleset)
	wantArmed(t, g, landGate(t, g))
}

// The facilitator authors most pull requests and the forge refuses an
// author's approval of their own, so their approval recorded through abcd
// arms in human mode.
func TestTheFacilitatorsRecordedApprovalArms(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.reviews = approvedBy("author-a", "@HEAD@")
	if err := os.MkdirAll(filepath.Join(f.repo.Root(), TierRelDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordApproval(f.repo.Root(), f.head()); err != nil {
		t.Fatal(err)
	}
	wantArmed(t, f, landGate(t, f))
}

// A contributor's pull request needs a maintainer's approval of its head in
// either mode; once it counts, the mode decides.
func TestAContributorsPullRequestNeedsAMaintainersApproval(t *testing.T) {
	roles := map[string]string{"author-a": `{"permission":"read","role_name":"triage"}`, "maintainer-b": `{"permission":"write","role_name":"write"}`}
	for _, mode := range []string{"", autoConfig} {
		files := map[string]string{}
		if mode != "" {
			files[".abcd/config.json"] = mode
		}
		f := newFixtureFiles(t, queueRuleset, files)
		f.forge.roles, f.forge.reviews = roles, "[]"
		f.reviewed(t)
		wantLeft(t, f, landGate(t, f), "by someone with write access is needed")

		g := newFixtureFiles(t, queueRuleset, files)
		g.forge.roles = roles
		g.reviewed(t)
		wantArmed(t, g, landGate(t, g))
	}
	// An author the forge knows no permission for is a contributor too.
	f := newFixture(t, queueRuleset)
	f.forge.roles = map[string]string{"maintainer-b": `{"permission":"write","role_name":"write"}`}
	f.forge.reviews = "[]"
	wantLeft(t, f, landGate(t, f), "by someone with write access is needed")
}

// An author's permission the forge cannot be asked for is unverified, and
// nothing is armed whatever the approvals.
func TestAnUnverifiedAuthorArmsNothing(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.roles = map[string]string{"author-a": "!gh: Resource not accessible by integration (HTTP 403)", "maintainer-b": defaultRoles["maintainer-b"]}
	f.reviewed(t)
	wantLeft(t, f, landGate(t, f), "unverified")
}

// In auto a maintainer's pull request arms by the mirror's method only when a
// review receipt names its head, and otherwise names the commit.
func TestAutoArmsOnlyAReviewedHead(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.reviews = "[]"
	wantLeft(t, f, landGate(t, f), f.head())

	g := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	g.forge.reviews = "[]"
	g.reviewed(t)
	s := landGate(t, g)
	wantArmed(t, g, s)
	if g.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+g.head()) != 1 || !strings.Contains(s.Note, "in the merge queue") {
		t.Fatalf("enqueue step %+v, calls %v: armed by the queue's method and confirmed queued", s, g.forge.calls)
	}
}

// The mode is read at the default branch's tip, never the lane's checkout
// (dcn-2610230058333177): a lane that commits auto over a default branch at
// human, or deletes the default branch's human, arms nothing.
func TestTheLaneDoesNotDecideItsOwnMode(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": humanConfig})
	f.repo.Write(".abcd/config.json", autoConfig)
	f.repo.Commit("the lane says auto")
	f.forge.reviews = "[]"
	f.reviewed(t)
	wantLeft(t, f, landGate(t, f), "facilitator.mode is human")

	g := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": humanConfig})
	g.repo.Remove(".abcd/config.json")
	g.repo.Commit("the lane deletes the default branch's human")
	g.forge.reviews = "[]"
	g.reviewed(t)
	wantLeft(t, g, landGate(t, g), "facilitator.mode is human")
}

// human in the machine layer beats auto in the repository's.
func TestTheMachinesHumanWins(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	dir := abcdhome.Path(f.home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(humanConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	f.forge.reviews = "[]"
	f.reviewed(t)
	wantLeft(t, f, landGate(t, f), "machine layer")
}

// A facilitator.mode refusal can quote a key a configuration file wrote; it
// reaches land's refusal escaped, and nothing is armed.
func TestAHostileKeyInTheModeRefusalIsEscaped(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": `{"facilitator":{"mode":"auto","\u001b]0;owned\u0007":1}}` + "\n"})
	f.forge.prs = f.prFor(f.head())
	_, err := f.land(f.options())
	var r *Refusal
	if !errors.As(err, &r) || !strings.Contains(r.Error(), "facilitator") || strings.ContainsAny(r.Error(), "\x1b\x07") {
		t.Fatalf("err = %q, want a refusal naming facilitator with its control bytes escaped", err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
}
