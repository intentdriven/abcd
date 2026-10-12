package embargo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const reporterText = "the reporter's original text\n\nwith a second paragraph"

// closeFixture is a seeded store whose advisory carries a "not a
// vulnerability" verdict and its draft, and a fake gh whose first two reads of
// the advisory answer with it as reported and every later read as closed with
// the approved text placed by the one placement function.
type closeFixture struct {
	seeded
	g    *ghFake
	path string
}

func newCloseFixture(t *testing.T) closeFixture {
	t.Helper()
	sd := seedStore(t)
	if _, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictNotAVulnerability, By: "Alice",
		Reason: "the behaviour is documented and intended"}); err != nil {
		t.Fatal(err)
	}
	cf := closeFixture{seeded: sd, g: newGHFake(t), path: listPath + "/" + sd.ghsa}
	reported := cf.remote(t, "triage", reporterText)
	cf.g.answer("GET", cf.path, "", 1, reported)
	cf.g.answer("GET", cf.path, "", 2, reported)
	cf.g.answer("GET", cf.path, "", 3, cf.remote(t, "closed", composeClosedDescription(cf.draft(t), reporterText)))
	cf.g.answer("PATCH", cf.path, "", 0, cf.remote(t, "closed", composeClosedDescription(cf.draft(t), reporterText)))
	return cf
}

func (cf closeFixture) remote(t *testing.T, state, description string) string {
	t.Helper()
	b, err := json.Marshal(advisoryJSON(cf.ghsa, state, "high", seededTitle, description))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (cf closeFixture) draft(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(cf.draftPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (cf closeFixture) close(approve string) (CloseResult, error) {
	return Close(CloseRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Target: cf.issID, Approve: approve,
		Now: func() time.Time { return fixedNow.Add(2 * time.Hour) }})
}

func (cf closeFixture) show(t *testing.T) CloseResult {
	t.Helper()
	res, err := cf.close("")
	if err != nil {
		t.Fatalf("Close without approval: %v", err)
	}
	return res
}

func noWrites(t *testing.T, g *ghFake) {
	t.Helper()
	if w := g.writes(); len(w) != 0 {
		t.Fatalf("%d write(s) sent: %q", len(w), w[0].argv)
	}
}

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// TestCloseWithoutApproveSendsOnlyGet: the bare close reads the advisory,
// shows exactly the text it would post with its digest and the approval
// command, and sends nothing else.
func TestCloseWithoutApproveSendsOnlyGet(t *testing.T) {
	cf := newCloseFixture(t)
	res := cf.show(t)
	if res.Text != cf.draft(t) {
		t.Fatalf("shown text is not the draft:\n%s", res.Text)
	}
	if !hex64.MatchString(res.Digest) || res.Closed || res.ForgeState != "triage" {
		t.Fatalf("result %+v", res)
	}
	if res.ApproveWith != "abcd embargo close "+cf.ghsa+" --approve "+res.Digest {
		t.Fatalf("approve_with = %q", res.ApproveWith)
	}
	calls := cf.g.calls()
	if len(calls) != 1 || calls[0].method() != "GET" || !calls[0].has(cf.path) || !calls[0].has("github.com") {
		t.Fatalf("calls %+v, want the one GET of the advisory", calls)
	}
	if a := cf.state(t); a.ClosedAt != nil {
		t.Fatal("a bare close stamped closed_at")
	}
}

// TestCloseWrongDigestSendsNothing: an approval that is not the digest of the
// text shown sends nothing and says how to see the current text.
func TestCloseWrongDigestSendsNothing(t *testing.T) {
	cf := newCloseFixture(t)
	// Every read answers with the advisory as reported: no close is made.
	cf.g.answer("GET", cf.path, "", 3, cf.remote(t, "triage", reporterText))
	shown := cf.show(t)
	for _, wrong := range []string{strings.Repeat("0", 64), strings.ToUpper(shown.Digest), shown.Digest[:12]} {
		_, err := cf.close(wrong)
		mustRefuse(t, err, "without --approve")
	}
	noWrites(t, cf.g)
	if a := cf.state(t); a.ClosedAt != nil {
		t.Fatal("closed_at stamped")
	}
}

// TestCloseEditedDraftInvalidatesApproval: an approval binds to the text the
// person saw; a draft edited since is a different text.
func TestCloseEditedDraftInvalidatesApproval(t *testing.T) {
	cf := newCloseFixture(t)
	shown := cf.show(t)
	if err := os.WriteFile(cf.draftPath(), []byte(cf.draft(t)+"\nOne more sentence.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := cf.close(shown.Digest)
	mustRefuse(t, err, "approval")
	noWrites(t, cf.g)
}

// TestCloseAdvisoryChangedOnGitHubInvalidatesApproval: the digest covers the
// description as read, so a reporter's edit on GitHub since the person looked
// voids the approval.
func TestCloseAdvisoryChangedOnGitHubInvalidatesApproval(t *testing.T) {
	cf := newCloseFixture(t)
	cf.g.answer("GET", cf.path, "", 2, cf.remote(t, "triage", reporterText+"\n\nan added line"))
	shown := cf.show(t)
	_, err := cf.close(shown.Digest)
	mustRefuse(t, err, "approval")
	noWrites(t, cf.g)
}

// TestClosePatchesStateClosedWithApprovedText: with the approval of exactly the
// shown text, one PATCH sets the advisory closed with that text, its body on
// stdin, the host named.
func TestClosePatchesStateClosedWithApprovedText(t *testing.T) {
	cf := newCloseFixture(t)
	shown := cf.show(t)
	res, err := cf.close(shown.Digest)
	if err != nil {
		t.Fatalf("approved close: %v", err)
	}
	if !res.Closed {
		t.Fatalf("result %+v", res)
	}
	w := cf.g.writes()
	if len(w) != 1 {
		t.Fatalf("%d writes, want one PATCH", len(w))
	}
	p := w[0]
	if p.method() != "PATCH" || !p.has(cf.path) || !p.has("--input") || !p.has("github.com") {
		t.Fatalf("PATCH argv %q", p.argv)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(p.stdin), &body); err != nil {
		t.Fatalf("PATCH body %q: %v", p.stdin, err)
	}
	if len(body) != 2 || body["state"] != "closed" {
		t.Fatalf("PATCH body keys %v", body)
	}
	if !strings.HasPrefix(body["description"], strings.TrimRight(shown.Text, "\n")) {
		t.Fatalf("the description does not begin with the approved text:\n%s", body["description"])
	}
	// The PATCH comes after the re-read and before the read-back.
	calls := cf.g.calls()
	if len(calls) != 4 || calls[2].method() != "PATCH" || calls[3].method() != "GET" {
		t.Fatalf("call order %v", calls)
	}
}

// TestCloseKeepsReporterDescriptionBelowSeparator: the reporter's text is kept
// whole under a separator line, below the approved explanation.
func TestCloseKeepsReporterDescriptionBelowSeparator(t *testing.T) {
	cf := newCloseFixture(t)
	shown := cf.show(t)
	if _, err := cf.close(shown.Digest); err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	_ = json.Unmarshal([]byte(cf.g.writes()[0].stdin), &body)
	want := strings.TrimRight(shown.Text, "\n") + "\n\n---\n\n" + reporterText
	if body["description"] != want {
		t.Fatalf("description:\n%q\nwant\n%q", body["description"], want)
	}
	if got := composeClosedDescription("an explanation\n", ""); got != "an explanation" {
		t.Fatalf("with no reporter text: %q", got)
	}
}

// TestCloseReadsBackAndReportsMismatch: a read-back that is not closed, or
// whose description does not begin with the approved text, is a failed close
// naming the half that did not hold, and closed_at is not stamped.
func TestCloseReadsBackAndReportsMismatch(t *testing.T) {
	for name, tc := range map[string]struct {
		state, description  string
		stateHeld, textHeld bool
	}{
		"still triage":  {"triage", "", false, true},
		"text missing":  {"closed", reporterText, true, false},
		"neither holds": {"draft", reporterText, false, false},
	} {
		t.Run(name, func(t *testing.T) {
			cf := newCloseFixture(t)
			desc := tc.description
			if desc == "" {
				desc = composeClosedDescription(cf.draft(t), reporterText)
			}
			cf.g.answer("GET", cf.path, "", 3, cf.remote(t, tc.state, desc))
			shown := cf.show(t)
			_, err := cf.close(shown.Digest)
			var cfe *CloseFailedError
			if !errors.As(err, &cfe) {
				t.Fatalf("err = %v, want a failed close", err)
			}
			if cfe.StateHeld != tc.stateHeld || cfe.TextHeld != tc.textHeld {
				t.Fatalf("failed close %+v", cfe)
			}
			if !tc.stateHeld && !strings.Contains(err.Error(), "state") || !tc.textHeld && !strings.Contains(err.Error(), "text") {
				t.Fatalf("the failure does not name the half: %v", err)
			}
			if a := cf.state(t); a.ClosedAt != nil {
				t.Fatal("closed_at stamped on a failed close")
			}
		})
	}
}

// TestCloseStampsClosedAt: a confirmed close records when, from the clock, and
// the advisory then refuses a second close without a call.
func TestCloseStampsClosedAt(t *testing.T) {
	cf := newCloseFixture(t)
	shown := cf.show(t)
	res, err := cf.close(shown.Digest)
	if err != nil {
		t.Fatal(err)
	}
	want := fixedNow.Add(2 * time.Hour)
	a := cf.state(t)
	if a.ClosedAt == nil || !a.ClosedAt.Equal(want) || a.ForgeState != "closed" {
		t.Fatalf("state entry %+v", a)
	}
	if res.ClosedAt == nil || !res.ClosedAt.Equal(want) {
		t.Fatalf("result closed_at %v", res.ClosedAt)
	}
	n := len(cf.g.calls())
	_, err = cf.close("")
	mustRefuse(t, err, "already closed")
	if len(cf.g.calls()) != n {
		t.Fatal("a refused close called gh")
	}
}

// TestCloseRefusesVulnerabilityOrUntriagedVerdict: only the two closing
// verdicts close; any other refuses before anything is sent.
func TestCloseRefusesVulnerabilityOrUntriagedVerdict(t *testing.T) {
	for name, req := range map[string]*TriageRequest{
		"untriaged":     nil,
		"vulnerability": {Verdict: VerdictVulnerability, By: "Alice"},
		"public-track":  {Verdict: VerdictPublicTrack, By: "Alice", Reason: "it discloses nothing"},
	} {
		t.Run(name, func(t *testing.T) {
			sd := seedStore(t)
			if req != nil {
				req.Target = sd.issID
				if _, err := sd.triage(*req); err != nil {
					t.Fatal(err)
				}
			}
			g := newGHFake(t)
			_, err := Close(CloseRequest{Home: sd.f.home, RepoRoot: sd.f.repo.Root(), Target: sd.issID, Approve: strings.Repeat("0", 64)})
			mustRefuse(t, err, "verdict")
			if len(g.calls()) != 0 {
				t.Fatal("a refused close called gh")
			}
		})
	}
}

// TestCloseRefusesAlreadyClosedOrPublished: an advisory the state records as
// closed or published refuses before any call; one GitHub reports closed or
// published refuses after the read, sending nothing.
func TestCloseRefusesAlreadyClosedOrPublished(t *testing.T) {
	for _, field := range []string{"closed_at", "published_at"} {
		t.Run(field, func(t *testing.T) {
			cf := newCloseFixture(t)
			st, _ := cf.s.Verify()
			at := fixedNow
			if field == "closed_at" {
				st.Advisories[0].ClosedAt = &at
			} else {
				st.Advisories[0].PublishedAt = &at
			}
			if err := cf.s.SaveState(st); err != nil {
				t.Fatal(err)
			}
			_, err := cf.close("")
			mustRefuse(t, err, "already")
			if len(cf.g.calls()) != 0 {
				t.Fatal("a refused close called gh")
			}
		})
	}
	for _, state := range []string{"closed", "published"} {
		t.Run("GitHub "+state, func(t *testing.T) {
			cf := newCloseFixture(t)
			cf.g.answer("GET", cf.path, "", 1, cf.remote(t, state, reporterText))
			_, err := cf.close("")
			mustRefuse(t, err, state)
			noWrites(t, cf.g)
		})
	}
}

// TestCloseScanFindingRefusesShowAndClose: the reporter reads the posted text,
// so a secret or a name the private-names layer bans refuses the show and the
// close alike, before anything is sent, and the refusal does not repeat it.
func TestCloseScanFindingRefusesShowAndClose(t *testing.T) {
	token := "gh" + "p_" + strings.Repeat("D", 36) // synthetic, split so no secret scanner reads it as one
	for name, setup := range map[string]func(cf closeFixture){
		"token": func(cf closeFixture) {
			if err := os.WriteFile(cf.draftPath(), []byte(cf.draft(t)+"\nUse "+token+" to see it.\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"private name": func(cf closeFixture) {
			p := filepath.Join(cf.f.repo.Root(), ".abcd", ".work.local", "private-names.txt")
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("# abcd-banlist: keyed\nproject-name zanzibarite\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cf.draftPath(), []byte(cf.draft(t)+"\nAs zanzibarite does.\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			cf := newCloseFixture(t)
			setup(cf)
			for _, approve := range []string{"", strings.Repeat("0", 64)} {
				_, err := cf.close(approve)
				mustRefuse(t, err, "closing text")
				if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "zanzibarite") {
					t.Fatalf("the refusal repeats the finding: %v", err)
				}
			}
			if len(cf.g.calls()) != 0 {
				t.Fatal("a refused close called gh")
			}
		})
	}
}
