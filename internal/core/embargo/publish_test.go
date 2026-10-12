package embargo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// publishFixture is a release v0.9.0 holding the fixes of two advisories
// judged vulnerabilities, A and B (A sorts first, so it is the canary), and a
// fake gh whose reads of each answer, in turn: as planned from (no versions),
// after the versions are set, and after publication.
type publishFixture struct {
	planFixture
}

var publishedAt = fixedNow.Add(3 * time.Hour)

// ghsaA sorts before ghsaB, so A is the canary.
const (
	ghsaA = "GHSA-rrrr-rrrr-rrrr"
	ghsaB = "GHSA-xxxx-xxxx-xxxx"
)

func newPublishFixture(t *testing.T, stateA string) publishFixture {
	t.Helper()
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaA, issA), vulnAdvisory(ghsaB, issB))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.resolve(issA, fix)
	pf.resolve(issB, fix)
	pf.tag("v0.9.0")
	px := publishFixture{pf}
	for ghsa, from := range map[string]string{ghsaA: stateA, ghsaB: "draft"} {
		px.answer(t, ghsa, 1, from, "", "")
		px.answer(t, ghsa, 2, from, "", "")
		px.answer(t, ghsa, 3, "draft", "< 0.9.0", "0.9.0")
		px.answer(t, ghsa, 4, "published", "< 0.9.0", "0.9.0")
		px.g.answer("PATCH", listPath+"/"+ghsa, "", 0, "{}")
	}
	px.g.answer("GET", listPath, "draft", 0, "[]")
	px.g.answer("GET", listPath, "triage", 0, "[]")
	return px
}

// answer sets the n-th read of ghsa: in state, with the versions given.
func (px publishFixture) answer(t *testing.T, ghsa string, n int, state, rng, patched string) {
	t.Helper()
	v := vulnJSON(examplePackage, rng)
	if patched != "" {
		v["patched_versions"] = patched
	}
	a := advisoryJSON(ghsa, state, "high", "a sample summary", "the report")
	a["vulnerabilities"] = []map[string]any{v}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	px.g.answer("GET", listPath+"/"+ghsa, "", n, string(b))
}

func (px publishFixture) publish(approve string) (PublishResult, error) {
	return Publish(PublishRequest{Home: px.f.home, RepoRoot: px.f.repo.Root(), Approve: approve,
		Now: func() time.Time { return publishedAt }})
}

// show is the bare publish: the plan, and its digest. It reads each
// advisory once, which the fixture's first answer covers.
func (px publishFixture) show(t *testing.T) PublishResult {
	t.Helper()
	res, err := px.publish("")
	if err != nil {
		t.Fatalf("Publish without approval: %v", err)
	}
	return res
}

// patches lists each PATCH as advisory id and body.
func (px publishFixture) patches(t *testing.T) [][2]string {
	t.Helper()
	var out [][2]string
	for _, c := range px.g.writes() {
		if c.method() != "PATCH" {
			t.Fatalf("a write that is not a PATCH: %q", c.argv)
		}
		id := ""
		for _, a := range c.argv {
			if strings.HasPrefix(a, listPath+"/") {
				id = strings.TrimPrefix(a, listPath+"/")
			}
		}
		out = append(out, [2]string{id, strings.TrimSpace(c.stdin)})
	}
	return out
}

func (px publishFixture) advisory(t *testing.T, ghsa string) Advisory {
	t.Helper()
	st, err := px.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range st.Advisories {
		if a.GHSAID == ghsa {
			return a
		}
	}
	t.Fatalf("%s not held", ghsa)
	return Advisory{}
}

// TestPublishWithoutApproveSendsNoPatch: the bare publish shows the plan with
// its approval command and sends only reads.
func TestPublishWithoutApproveSendsNoPatch(t *testing.T) {
	px := newPublishFixture(t, "draft")
	res := px.show(t)
	if len(res.Plan.Entries) != 2 || res.Plan.ApproveWith == "" || len(res.Published) != 0 {
		t.Fatalf("result %+v", res)
	}
	noWrites(t, px.g)
	if a := px.advisory(t, ghsaA); a.PublishedAt != nil {
		t.Fatal("published_at stamped without approval")
	}
}

// TestPublishWrongDigestSendsNoPatch: an approval that is not the digest of
// the plan as it stands now, including one given for a plan GitHub has since
// changed, sends nothing.
func TestPublishWrongDigestSendsNoPatch(t *testing.T) {
	px := newPublishFixture(t, "draft")
	shown := px.show(t)
	for _, wrong := range []string{strings.Repeat("0", 64), strings.ToUpper(shown.Plan.Digest), shown.Plan.Digest[:12]} {
		_, err := px.publish(wrong)
		mustRefuse(t, err, "without --approve")
	}
	noWrites(t, px.g)

	// The advisory edited on GitHub between the show and the approval.
	px = newPublishFixture(t, "draft")
	shown = px.show(t)
	a := advisoryJSON(ghsaA, "draft", "high", "a sample summary", "the report, edited on GitHub")
	a["vulnerabilities"] = []map[string]any{vulnJSON(examplePackage, "")}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	px.g.answer("GET", listPath+"/"+ghsaA, "", 2, string(b))
	_, err = px.publish(shown.Plan.Digest)
	mustRefuse(t, err, "approval")
	noWrites(t, px.g)
}

// TestCanaryFirstThenRest: the canary is set, read back, published and read
// back before the next advisory is touched, and the drafts are listed last.
func TestCanaryFirstThenRest(t *testing.T) {
	px := newPublishFixture(t, "draft")
	shown := px.show(t)
	if _, err := px.publish(shown.Plan.Digest); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	var seq []string
	for _, c := range px.g.calls()[4:] { // the show's two reads and the approval's rebuild
		what := c.method()
		for _, a := range c.argv {
			switch {
			case a == listPath+"/"+ghsaA:
				what += " A"
			case a == listPath+"/"+ghsaB:
				what += " B"
			case a == "state=draft" || a == "state=triage":
				what += " list " + strings.TrimPrefix(a, "state=")
			}
		}
		seq = append(seq, what)
	}
	want := []string{"PATCH A", "GET A", "PATCH A", "GET A", "PATCH B", "GET B", "PATCH B", "GET B", "GET list draft", "GET list triage"}
	if strings.Join(seq, ",") != strings.Join(want, ",") {
		t.Fatalf("calls\n got %q\nwant %q", seq, want)
	}
	p := px.patches(t)
	if p[0][1] != `{"vulnerabilities":[{"package":{"ecosystem":"go","name":"`+examplePackage+`"},"vulnerable_version_range":"< 0.9.0","patched_versions":"0.9.0","vulnerable_functions":[]}]}` {
		t.Fatalf("versions body %s", p[0][1])
	}
	if p[1][1] != `{"state":"published"}` {
		t.Fatalf("publish body %s", p[1][1])
	}
	for _, c := range px.g.calls() {
		if !c.has("github.com") {
			t.Fatalf("a call without the explicit host: %q", c.argv)
		}
	}
}

// TestCanaryMismatchStops: a canary that does not read back as published
// with the planned versions stops the run: nothing more is sent, and the
// failure says nothing was published.
func TestCanaryMismatchStops(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, px publishFixture){
		"versions not held": func(t *testing.T, px publishFixture) { px.answer(t, ghsaA, 3, "draft", "< 0.8.0", "0.8.0") },
		"not published":     func(t *testing.T, px publishFixture) { px.answer(t, ghsaA, 4, "draft", "< 0.9.0", "0.9.0") },
		"patch refused": func(t *testing.T, px publishFixture) {
			px.g.fail("PATCH", listPath+"/"+ghsaA, "", "HTTP 422: Validation Failed")
		},
	} {
		t.Run(name, func(t *testing.T) {
			px := newPublishFixture(t, "draft")
			setup(t, px)
			shown := px.show(t)
			_, err := px.publish(shown.Plan.Digest)
			var pe *PublishFailedError
			if !errors.As(err, &pe) {
				t.Fatalf("err = %v, want a failed publication", err)
			}
			if pe.GHSAID != ghsaA || !pe.Canary || len(pe.Published) != 0 || strings.Join(pe.Unpublished, ",") != ghsaA+","+ghsaB {
				t.Fatalf("failure %+v", pe)
			}
			for _, p := range px.patches(t) {
				if p[0] != ghsaA {
					t.Fatalf("a PATCH reached %s after the canary failed", p[0])
				}
			}
			if !strings.Contains(err.Error(), "nothing was published") {
				t.Fatalf("error %q", err)
			}
		})
	}
}

// TestTriageStateAcceptedToDraftFirst: an advisory still in triage is
// accepted to draft in the same update that sets its versions.
func TestTriageStateAcceptedToDraftFirst(t *testing.T) {
	px := newPublishFixture(t, "triage")
	shown := px.show(t)
	if shown.Plan.Entries[0].ForgeState != "triage" {
		t.Fatalf("plan %+v", shown.Plan.Entries[0])
	}
	if _, err := px.publish(shown.Plan.Digest); err != nil {
		t.Fatal(err)
	}
	p := px.patches(t)
	var body map[string]any
	if err := json.Unmarshal([]byte(p[0][1]), &body); err != nil {
		t.Fatal(err)
	}
	if p[0][0] != ghsaA || body["state"] != "draft" || body["vulnerabilities"] == nil {
		t.Fatalf("first update %s", p[0][1])
	}
	if strings.Contains(p[2][1], `"state"`) {
		t.Fatalf("an advisory already in draft was sent a state with its versions: %s", p[2][1])
	}
}

// TestPartialRunReportsAndReplans: a failure after the canary stops the run,
// says which advisories are published and which are not, and a later plan
// holds only what is still unpublished.
func TestPartialRunReportsAndReplans(t *testing.T) {
	px := newPublishFixture(t, "draft")
	px.answer(t, ghsaB, 4, "draft", "< 0.9.0", "0.9.0")
	shown := px.show(t)
	_, err := px.publish(shown.Plan.Digest)
	var pe *PublishFailedError
	if !errors.As(err, &pe) || pe.GHSAID != ghsaB || pe.Canary ||
		strings.Join(pe.Published, ",") != ghsaA || strings.Join(pe.Unpublished, ",") != ghsaB {
		t.Fatalf("err = %v (%+v)", err, pe)
	}
	if !strings.Contains(err.Error(), "published: "+ghsaA) || !strings.Contains(err.Error(), "not published: "+ghsaB) {
		t.Fatalf("error %q", err)
	}
	if px.advisory(t, ghsaA).PublishedAt == nil || px.advisory(t, ghsaB).PublishedAt != nil {
		t.Fatal("published_at does not follow what was published")
	}
	px.answer(t, ghsaB, 5, "draft", "< 0.9.0", "0.9.0")
	p := px.plan(t, "")
	if len(p.Entries) != 1 || p.Entries[0].GHSAID != ghsaB {
		t.Fatalf("replan %+v", p)
	}
}

// TestNoDraftLeftConfirmed: after the last publication the draft and triage
// listings are read, and a planned advisory still in one fails the run.
func TestNoDraftLeftConfirmed(t *testing.T) {
	px := newPublishFixture(t, "draft")
	shown := px.show(t)
	res, err := px.publish(shown.Plan.Digest)
	if err != nil || !res.NoDraftLeft || len(res.Published) != 2 {
		t.Fatalf("result %+v, %v", res, err)
	}

	px = newPublishFixture(t, "draft")
	px.g.answer("GET", listPath, "triage", 0, page(t, advisoryJSON(ghsaB, "triage", "high", "a sample summary", "x")))
	shown = px.show(t)
	_, err = px.publish(shown.Plan.Digest)
	var pe *PublishFailedError
	if !errors.As(err, &pe) || !strings.Contains(err.Error(), "still lists") || len(pe.Published) != 2 {
		t.Fatalf("err = %v (%+v)", err, pe)
	}
}

// TestPublishedAtStamped: each advisory GitHub confirms published is stamped
// in the state with the clock's time.
func TestPublishedAtStamped(t *testing.T) {
	px := newPublishFixture(t, "draft")
	shown := px.show(t)
	res, err := px.publish(shown.Plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, ghsa := range []string{ghsaA, ghsaB} {
		a := px.advisory(t, ghsa)
		if a.PublishedAt == nil || !a.PublishedAt.Equal(publishedAt) || a.ForgeState != "published" {
			t.Fatalf("%s: %+v", ghsa, a)
		}
	}
	if res.Published[0].GHSAID != ghsaA || !res.Published[0].PublishedAt.Equal(publishedAt) {
		t.Fatalf("result %+v", res.Published)
	}
}

// TestPublishedAtRecordedFromGitHubAfterSaveFails: an advisory GitHub
// confirmed published whose published_at could not be saved is reported as
// published, and the next plan reads it published on GitHub and records it,
// leaving it out, instead of holding it under embargo for ever.
func TestPublishedAtRecordedFromGitHubAfterSaveFails(t *testing.T) {
	px := newPublishFixture(t, "draft")
	stateDir := filepath.Dir(px.s.StatePath())
	t.Cleanup(func() { _ = os.Chmod(stateDir, 0o700) })
	// The publishing PATCH of the canary leaves the state's folder unwritable.
	px.g.hook(`if [ "$method" = PATCH ] && grep -q '"published"' "$d/calls/$n.stdin"; then chmod 500 '` + stateDir + `'; fi`)
	shown := px.show(t)
	_, err := px.publish(shown.Plan.Digest)
	var pe *PublishFailedError
	if !errors.As(err, &pe) || pe.GHSAID != ghsaA || strings.Join(pe.Published, ",") != ghsaA {
		t.Fatalf("err = %v (%+v)", err, pe)
	}
	if !strings.Contains(err.Error(), "the next `abcd embargo publish` reads it published on GitHub and records it") {
		t.Fatalf("the failure does not say how published_at is recorded: %v", err)
	}
	px.g.hook(":")
	if err := os.Chmod(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if a := px.advisory(t, ghsaA); a.PublishedAt != nil {
		t.Fatalf("published_at recorded though its save failed: %+v", a)
	}
	px.answer(t, ghsaA, 5, "published", "< 0.9.0", "0.9.0")
	p := px.plan(t, "")
	if len(p.Entries) != 1 || p.Entries[0].GHSAID != ghsaB {
		t.Fatalf("replan %+v", p)
	}
	if r := leftOutReason(t, p, ghsaA); !strings.Contains(r, "published already") {
		t.Fatalf("left-out reason %q", r)
	}
	if a := px.advisory(t, ghsaA); a.PublishedAt == nil || a.ForgeState != "published" {
		t.Fatalf("the plan did not record what GitHub reports: %+v", a)
	}
}

// TestPlanRecordsClosedOnGitHub: an advisory judged a vulnerability that
// GitHub reports closed is left out, and its closed_at is recorded.
func TestPlanRecordsClosedOnGitHub(t *testing.T) {
	px := newPublishFixture(t, "draft")
	px.answer(t, ghsaA, 1, "closed", "", "")
	p := px.plan(t, "")
	if r := leftOutReason(t, p, ghsaA); !strings.Contains(r, "closed already") {
		t.Fatalf("left-out reason %q", r)
	}
	if a := px.advisory(t, ghsaA); a.ClosedAt == nil || a.ForgeState != "closed" || a.PublishedAt != nil {
		t.Fatalf("the plan did not record what GitHub reports: %+v", a)
	}
	noWrites(t, px.g)
}
