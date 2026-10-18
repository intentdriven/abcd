package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/gittest"
)

// embargo_surface_test.go — step 10 of spc-2610120557432880: the front door of
// `abcd embargo`. The core's behaviour is held in internal/core/embargo; these
// tests hold what the surface adds: each sub-verb's refusals exit 2 with the
// remedy and write nothing, and each --json document has its shape. Advisory
// ids are of the GHSA-xxxx-xxxx-xxxx placeholder form
// (examples-use-reserved-identifiers).

const surfaceGHSA = "GHSA-xxxx-xxxx-xxxx"

// embargoSurfaceRepo is a checkout whose origin names an example GitHub
// repository, under a temp HOME, with a stand-in `gh` on PATH: the triage
// listing answers surfaceGHSA once the test has created the file named by
// ghTriage, every other listing an empty page, and every read of one
// advisory answers surfaceGHSA in triage. The working directory is the
// checkout.
func embargoSurfaceRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Write("README.md", "a project\n")
	r.Commit("initial")
	r.Git("remote", "add", "origin", "https://github.com/example/example.git")
	bin := t.TempDir()
	advisory := `{"ghsa_id":"` + surfaceGHSA + `","cve_id":null,"summary":"a sample report","description":"the report",` +
		`"severity":"high","state":"triage","vulnerabilities":[]}`
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in\n" +
		"  state=triage) if [ -f '" + filepath.Join(bin, "triage-on") + "' ]; then echo '[" + advisory + "]'; else echo '[]'; fi; exit 0 ;;\n" +
		"  state=*) echo '[]'; exit 0 ;;\nesac; done\necho '" + advisory + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(ghTriageEnv, filepath.Join(bin, "triage-on"))
	t.Chdir(r.Root())
	return r
}

const ghTriageEnv = "ABCD_TEST_EMBARGO_TRIAGE_ON"

// seedEmbargo runs intake with surfaceGHSA waiting in triage, so the store
// holds its private record.
func seedEmbargo(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(os.Getenv(ghTriageEnv), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, stderr := runEmbargo(t, "", "embargo", "intake"); code != 0 || !strings.Contains(out, "1 new private record") {
		t.Fatalf("intake: exit %d, %s%s", code, out, stderr)
	}
}

func runEmbargo(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func jsonKeys(t *testing.T, doc string) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, doc)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestEmbargoSurfaceRefusals: each sub-verb refuses a request it cannot act
// on at exit 2, saying what to do, and writes nothing.
func TestEmbargoSurfaceRefusals(t *testing.T) {
	r := embargoSurfaceRepo(t)
	loc, err := storepath.Locate(os.Getenv("HOME"), r.Root())
	if err != nil || loc.Registered {
		t.Fatalf("the fixture repository's store place: %+v, %v", loc, err)
	}
	store := loc.Path
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"embargo", "triage", surfaceGHSA, "vulnerability", "--by", "Alice"}, "abcd embargo intake"},
		{[]string{"embargo", "close", surfaceGHSA}, "abcd embargo intake"},
		{[]string{"embargo", "publish"}, "abcd embargo intake"},
		{[]string{"embargo", "check"}, "nothing to check"},
		{[]string{"embargo", "check", "--range", "nonsense"}, "<a>..<b>"},
		{[]string{"embargo", "triage", surfaceGHSA}, "accepts 2 arg"},
		{[]string{"embargo", "unknown"}, "unknown"},
	} {
		code, _, stderr := runEmbargo(t, "", tc.args...)
		if code != 2 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, stderr %q; want exit 2 saying %q", tc.args, code, stderr, tc.want)
		}
	}
	if _, err := os.Lstat(store); err == nil {
		t.Fatal("a refused sub-verb made the embargo store")
	}

	seedEmbargo(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"embargo", "triage", surfaceGHSA, "maybe", "--by", "Alice"}, "verdict"},
		{[]string{"embargo", "triage", surfaceGHSA, "not-a-vulnerability", "--by", "Alice"}, "--reason"},
		{[]string{"embargo", "triage", surfaceGHSA, "vulnerability"}, "--by"},
		{[]string{"embargo", "close", surfaceGHSA}, "verdict"},
		{[]string{"embargo", "publish", "--release", "v1"}, "vMAJOR.MINOR.PATCH"},
	} {
		code, _, stderr := runEmbargo(t, "", tc.args...)
		if code != 2 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, stderr %q; want exit 2 saying %q", tc.args, code, stderr, tc.want)
		}
	}

	t.Chdir(t.TempDir())
	if code, _, stderr := runEmbargo(t, "", "embargo", "intake"); code != 2 || !strings.Contains(stderr, "nothing written") {
		t.Errorf("intake outside a checkout: exit %d, stderr %q", code, stderr)
	}
}

// TestEmbargoJSONShapes: every sub-verb's --json is one document on stdout of
// the core's shape, a check finding included (exit 2, findings in the
// document), and no document repeats a matched advisory id.
func TestEmbargoJSONShapes(t *testing.T) {
	r := embargoSurfaceRepo(t)

	code, out, stderr := runEmbargo(t, "", "embargo", "intake", "--json")
	if code != 0 {
		t.Fatalf("intake: exit %d, %s", code, stderr)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "ended,held,intake_at,minted,needs_severity,repo,restores,store,store_created,unread" {
		t.Errorf("intake keys %s", got)
	}
	if strings.Contains(out, os.Getenv("HOME")) {
		t.Errorf("intake's document names the home directory: %s", out)
	}

	code, out, _ = runEmbargo(t, "fix: tighten the parser\n", "embargo", "check", "--file", "-", "--json")
	if code != 0 {
		t.Fatalf("clean check: exit %d", code)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "allowed,commits,files,findings,intake_at,intake_stale,no_store,under_embargo" {
		t.Errorf("check keys %s", got)
	}

	seedEmbargo(t)
	code, out, _ = runEmbargo(t, "fixes "+surfaceGHSA+"\n", "embargo", "check", "--file", "-", "--json")
	var res embargo.CheckResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || code != 2 || len(res.Findings) != 1 ||
		res.Findings[0].Kind != embargo.KindAdvisoryID || strings.Contains(out, "xxxx") {
		t.Fatalf("check finding: exit %d, %v, %s", code, err, out)
	}
	code, out, _ = runEmbargo(t, "fixes "+surfaceGHSA+"\n", "embargo", "check", "--file", "-")
	if code != 2 || !strings.Contains(out, "REFUSED, 1 finding") || strings.Contains(out, "xxxx") {
		t.Fatalf("check text: exit %d, %s", code, out)
	}

	code, out, stderr = runEmbargo(t, "", "embargo", "triage", surfaceGHSA, "not-a-vulnerability", "--by", "Alice",
		"--reason", "the behaviour is documented", "--json")
	if code != 0 {
		t.Fatalf("triage: exit %d, %s%s", code, stderr, out)
	}
	for _, k := range []string{"decided_at", "decided_by", "draft", "draft_path", "ghsa_id", "issue_id", "record_status", "verdict"} {
		if !strings.Contains(strings.Join(jsonKeys(t, out), ","), k) {
			t.Errorf("triage document has no %s: %s", k, out)
		}
	}

	code, out, stderr = runEmbargo(t, "", "embargo", "close", surfaceGHSA, "--json")
	if code != 0 {
		t.Fatalf("close: exit %d, %s", code, stderr)
	}
	var cr embargo.CloseResult
	if err := json.Unmarshal([]byte(out), &cr); err != nil || cr.Closed || cr.Digest == "" ||
		!strings.Contains(cr.ApproveWith, "--approve "+cr.Digest) || cr.Text == "" {
		t.Fatalf("close document %v %s", err, out)
	}
	code, out, _ = runEmbargo(t, "", "embargo", "close", surfaceGHSA)
	if code != 0 || !strings.Contains(out, "nothing was sent") || !strings.Contains(out, "approve with: ") {
		t.Fatalf("close text: exit %d, %s", code, out)
	}

	r.Git("tag", "v0.1.0")
	code, out, stderr = runEmbargo(t, "", "embargo", "publish", "--json")
	if code != 0 {
		t.Fatalf("publish: exit %d, %s", code, stderr)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "approved,no_draft_left,plan,published,restores" {
		t.Errorf("publish keys %s", got)
	}
	var pr embargo.PublishResult
	if err := json.Unmarshal([]byte(out), &pr); err != nil || pr.Approved || pr.Plan.Release != "v0.1.0" || pr.Plan.Entries == nil || pr.Plan.LeftOut == nil {
		t.Fatalf("publish document %v %s", err, out)
	}
}

// TestEmbargoCountsReadAsEnglish: a count of advisories is spelled as one.
func TestEmbargoCountsReadAsEnglish(t *testing.T) {
	for n, want := range map[int]string{0: "0 advisories", 1: "1 advisory", 6: "6 advisories"} {
		if got := advisoryCount(n); got != want {
			t.Errorf("advisoryCount(%d) = %q, want %q", n, got, want)
		}
	}
}

// TestEmbargoCheckRendersItsNote: a check that read a store with no live
// worktree registration says so in its text form too, not only in --json.
func TestEmbargoCheckRendersItsNote(t *testing.T) {
	const note = "1 embargo store under this home was read with no live worktree registration; its records stay under embargo"
	var b bytes.Buffer
	renderEmbargoCheck(&b, embargo.CheckResult{Note: note, Findings: []embargo.CheckFinding{}})
	if !strings.Contains(b.String(), "note: "+note+"\n") {
		t.Fatalf("the text form is %q; want the note on a line of its own", b.String())
	}
}

// TestEmbargoPublishRendersStoreEvidence: the text form says which planned
// advisory takes its fix from the store, and after publication prints each
// restore the result names, once.
func TestEmbargoPublishRendersStoreEvidence(t *testing.T) {
	const restore = "cp -- ~/.abcd.noindex/worktrees/0123/embargo/.abcd/work/issues/resolved/iss-1-private-report.md .abcd/work/issues/resolved/"
	fix := strings.Repeat("a", 40)
	plan := embargo.PublicationPlan{Repo: "example/example", Release: "v0.9.0", LeftOut: []embargo.LeftOut{}, Digest: "d", ApproveWith: "abcd embargo publish --release v0.9.0 --approve d",
		Entries: []embargo.PlanEntry{
			{GHSAID: surfaceGHSA, IssueID: "iss-1", FixCommit: fix, FirstTag: "v0.9.0", PatchedVersion: "0.9.0", StoreEvidence: true},
			{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", FixCommit: fix, FirstTag: "v0.9.0", PatchedVersion: "0.9.0"},
		}}
	var b bytes.Buffer
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: plan, Published: []embargo.PublishedAdvisory{}})
	lines := strings.Split(b.String(), "\n")
	if !strings.Contains(lines[1], surfaceGHSA) || !strings.Contains(lines[1], "fix recorded in the store") || strings.Contains(lines[2], "store") {
		t.Fatalf("the plan's text form is %q", b.String())
	}

	b.Reset()
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: plan, Approved: true, NoDraftLeft: true, Published: []embargo.PublishedAdvisory{
		{GHSAID: surfaceGHSA, IssueID: "iss-1"}, {GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2"},
	}, Restores: []embargo.Restore{{GHSAID: surfaceGHSA, IssueID: "iss-1", RestoreWith: restore}}})
	if got := b.String(); !strings.Contains(got, "  "+surfaceGHSA+" (iss-1) is published; restore its record as resolved, from the checkout's root:\n    "+restore+"\n") ||
		strings.Count(got, "restore its record") != 1 {
		t.Fatalf("the publication's text form is %q", got)
	}
}

// TestEmbargoRendersRestores: a plan and an intake each print every restore
// the store's state owes, and a restore that names no command says why.
func TestEmbargoRendersRestores(t *testing.T) {
	const restore = "cp -- ~/.abcd.noindex/worktrees/0123/embargo/.abcd/work/issues/resolved/iss-1-private-report.md .abcd/work/issues/resolved/"
	rs := []embargo.Restore{
		{GHSAID: surfaceGHSA, IssueID: "iss-1", RestoreWith: restore},
		{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", Reason: "its private record in the store carries another id; no command restoring it is named"},
	}
	wants := []string{
		"  " + surfaceGHSA + " (iss-1) is published; restore its record as resolved, from the checkout's root:\n    " + restore + "\n",
		"  GHSA-rrrr-rrrr-rrrr (iss-2) is published, and its record is not restored: its private record in the store carries another id; no command restoring it is named\n",
	}
	var b bytes.Buffer
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: embargo.PublicationPlan{Repo: "example/example", Release: "v0.9.0", Entries: []embargo.PlanEntry{},
		LeftOut: []embargo.LeftOut{}, Restores: rs}})
	for _, want := range wants {
		if got := b.String(); !strings.Contains(got, want) {
			t.Fatalf("the plan's text form is %q; want %q", got, want)
		}
	}
	b.Reset()
	renderEmbargoIntake(&b, embargo.IntakeResult{Repo: "example/example", Minted: []embargo.MintedRecord{}, Held: []string{}, NeedsSeverity: []string{}, Restores: rs})
	for _, want := range wants {
		if got := b.String(); !strings.Contains(got, want) {
			t.Fatalf("the intake's text form is %q; want %q", got, want)
		}
	}
}

// TestEmbargoIntakeRendersForgeEnds: the text form names each held
// advisory GitHub reports published or closed, now recorded, and each one
// GitHub could not tell, left as it was.
func TestEmbargoIntakeRendersForgeEnds(t *testing.T) {
	var b bytes.Buffer
	renderEmbargoIntake(&b, embargo.IntakeResult{Repo: "example/example", Minted: []embargo.MintedRecord{}, Held: []string{}, NeedsSeverity: []string{},
		Ended:  []embargo.ForgeEnd{{GHSAID: surfaceGHSA, IssueID: "iss-1", State: "published"}},
		Unread: []embargo.UnreadAdvisory{{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", Reason: "GitHub could not be read for it; its state in the store is unchanged"}}})
	got := b.String()
	for _, want := range []string{
		"  " + surfaceGHSA + " (iss-1) is published on GitHub; published_at is recorded from that, at 0001-01-01T00:00:00Z\n",
		"  GHSA-rrrr-rrrr-rrrr (iss-2): GitHub could not be read for it; its state in the store is unchanged\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the text form is %q; want %q", got, want)
		}
	}
	b.Reset()
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: embargo.PublicationPlan{Repo: "example/example", Release: "v0.9.0", Entries: []embargo.PlanEntry{}, LeftOut: []embargo.LeftOut{},
		Unread: []embargo.UnreadAdvisory{{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", Reason: "GitHub could not be read for it; its state in the store is unchanged"}}}})
	if got := b.String(); !strings.Contains(got, "  GHSA-rrrr-rrrr-rrrr (iss-2): GitHub could not be read for it; its state in the store is unchanged\n") {
		t.Fatalf("the plan's text form is %q", got)
	}
	b.Reset()
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: embargo.PublicationPlan{Repo: "example/example", Release: "v0.9.0", Entries: []embargo.PlanEntry{}, LeftOut: []embargo.LeftOut{},
		Ended: []embargo.ForgeEnd{{GHSAID: surfaceGHSA, IssueID: "iss-1", State: "published"}}}})
	if got := b.String(); !strings.Contains(got, "  "+surfaceGHSA+" (iss-1) is published on GitHub; published_at is recorded from that, at 0001-01-01T00:00:00Z\n") {
		t.Fatalf("the plan's text form does not name what it recorded: %q", got)
	}
}

// TestEmbargoPublishShowsUnreadCandidateOnce: a candidate GitHub could not
// be read for is on one line of the plan's text form, its left-out line,
// carrying the unread reason when that is not already its reason; a held
// advisory that is no candidate keeps its own unread line.
func TestEmbargoPublishShowsUnreadCandidateOnce(t *testing.T) {
	const unread = "GitHub could not be read for it; its state in the store is unchanged"
	var b bytes.Buffer
	renderEmbargoPublish(&b, embargo.PublishResult{Plan: embargo.PublicationPlan{Repo: "example/example", Release: "v0.9.0", Entries: []embargo.PlanEntry{},
		LeftOut: []embargo.LeftOut{
			{GHSAID: surfaceGHSA, IssueID: "iss-1", Reason: unread},
			{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", Reason: "its record is not resolved in v0.9.0"},
		},
		Unread: []embargo.UnreadAdvisory{
			{GHSAID: surfaceGHSA, IssueID: "iss-1", Reason: unread},
			{GHSAID: "GHSA-rrrr-rrrr-rrrr", IssueID: "iss-2", Reason: unread},
			{GHSAID: "GHSA-hhhh-hhhh-hhhh", IssueID: "iss-3", Reason: unread},
		}}})
	got := b.String()
	for id, n := range map[string]int{surfaceGHSA: 1, "GHSA-rrrr-rrrr-rrrr": 1, "GHSA-hhhh-hhhh-hhhh": 1} {
		if c := strings.Count(got, id); c != n {
			t.Fatalf("%s is on %d lines, want %d: %q", id, c, n, got)
		}
	}
	for _, want := range []string{
		"  left out " + surfaceGHSA + " (iss-1): " + unread + "\n",
		"  left out GHSA-rrrr-rrrr-rrrr (iss-2): its record is not resolved in v0.9.0; also, " + unread + "\n",
		"  GHSA-hhhh-hhhh-hhhh (iss-3): " + unread + "\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the plan's text form is %q; want %q", got, want)
		}
	}
}
