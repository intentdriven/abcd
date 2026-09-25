package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// intent_drain_test.go is the wiring proof for itd-53 (spc-2609211930059886):
// `abcd intent audit --owed [--max <n>]` prints the owed reviews oldest shipped
// first, capped, names how many remain, and emits the head's request so a host
// without the plugin page can drive the drain by hand. It runs no reviewer.

// commitShippedOn commits one file at a pinned author date, so the history walk
// reads that day as the day the file entered shipped/.
func commitShippedOn(t *testing.T, repo, rel, day string) {
	t.Helper()
	env := append(gittest.Env(t), "GIT_AUTHOR_DATE="+day+"T12:00:00Z", "GIT_COMMITTER_DATE="+day+"T12:00:00Z")
	for _, args := range [][]string{
		{"add", "--", rel},
		{"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "ship " + filepath.Base(rel)},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// drainRepo stages three owed intents — itd-20 committed 2026-02-01, itd-21
// committed 2026-01-01 with no marker, itd-22 shipped in the working tree only —
// plus an ingested one that is never queued.
func drainRepo(t *testing.T) string {
	t.Helper()
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliShipped+"/itd-20-s.md", cliShippedWithNotes("itd-20",
		"<!-- abcd-review: OWED receipt=rcp-0000000000a1 -->\nFidelity review OWED (receipt rcp-0000000000a1)."))
	commitShippedOn(t, repo, cliShipped+"/itd-20-s.md", "2026-02-01")
	writeRepoFile(t, repo, cliShipped+"/itd-21-s.md", cliShippedWithNotes("itd-21",
		"_Empty. Populated by intent-auditor when intent moves to shipped/._"))
	commitShippedOn(t, repo, cliShipped+"/itd-21-s.md", "2026-01-01")
	writeRepoFile(t, repo, cliShipped+"/itd-12-s.md", cliShippedWithNotes("itd-12",
		"<!-- abcd-review: INGESTED receipt=rcp-0000000000b2 -->\nFidelity review — receipt rcp-0000000000b2."))
	commitShippedOn(t, repo, cliShipped+"/itd-12-s.md", "2025-06-01")
	writeRepoFile(t, repo, cliShipped+"/itd-22-s.md", cliShippedWithNotes("itd-22",
		"<!-- abcd-review: OWED receipt=rcp-0000000000c3 -->\nFidelity review OWED (receipt rcp-0000000000c3)."))
	return repo
}

func TestIntentAuditOwedOrdersOldestFirstAndEmitsTheHead(t *testing.T) {
	repo := drainRepo(t)
	out, err := runCLIErr(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatalf("intent audit --owed must exit 0: %v\n%s", err, out)
	}
	s := string(out)
	i21, i20, i22 := strings.Index(s, "itd-21"), strings.Index(s, "itd-20"), strings.Index(s, "itd-22")
	if i21 < 0 || i20 < 0 || i22 < 0 || !(i21 < i20 && i20 < i22) {
		t.Fatalf("want itd-21 (2026-01-01), itd-20 (2026-02-01), itd-22 (uncommitted) in that order:\n%s", s)
	}
	if strings.Contains(s, "itd-12") {
		t.Fatalf("an ingested review is never queued:\n%s", s)
	}
	for _, want := range []string{"3 owed", "2026-01-01", "2026-02-01", "not yet committed", "runs no reviewer"} {
		if !strings.Contains(s, want) {
			t.Errorf("--owed output lacks %q:\n%s", want, s)
		}
	}
	// The head is itd-21, markerless: the emit mints its receipt and writes the
	// request, and the output names the request path.
	_, next, ok := strings.Cut(s, "next: ")
	if !ok || !strings.HasPrefix(next, "itd-21") {
		t.Fatalf("the next line does not name the head itd-21:\n%s", s)
	}
	_, reqLine, ok := strings.Cut(next, "request: ")
	if !ok {
		t.Fatalf("no request path printed:\n%s", s)
	}
	req := strings.TrimSpace(strings.SplitN(reqLine, "\n", 2)[0])
	if _, err := os.Stat(filepath.Join(repo, req)); err != nil {
		t.Fatalf("the printed request %q does not exist: %v", req, err)
	}
	body, err := os.ReadFile(filepath.Join(repo, cliShipped, "itd-21-s.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "abcd-review: OWED receipt=") {
		t.Fatalf("the head's emit parked no marker:\n%s", body)
	}
	// Only the head: the entries behind it have no request yet.
	if _, err := os.Stat(filepath.Join(repo, ".abcd/.work.local/reviews/rcp-0000000000a1.request.md")); err == nil {
		t.Fatal("a request was emitted for an entry behind the head")
	}
}

func TestIntentAuditOwedMaxCapsAndNamesTheRemainder(t *testing.T) {
	_ = drainRepo(t)
	out, err := runCLIErr(t, "intent", "audit", "--owed", "--max", "1")
	if err != nil {
		t.Fatalf("--owed --max 1 must exit 0: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "itd-21") || strings.Contains(s, "itd-20") || strings.Contains(s, "itd-22") {
		t.Fatalf("--max 1 lists the oldest alone:\n%s", s)
	}
	if !strings.Contains(s, "2 remain") {
		t.Fatalf("the summary does not name how many remain:\n%s", s)
	}

	var got struct {
		Queue []struct {
			IntentID  string `json:"intent_id"`
			State     string `json:"state"`
			ReceiptID string `json:"receipt_id"`
			Shipped   string `json:"shipped"`
		} `json:"queue"`
		Owed      int `json:"owed"`
		Max       int `json:"max"`
		Remaining int `json:"remaining"`
		Next      *struct {
			IntentID    string `json:"intent_id"`
			ReceiptID   string `json:"receipt_id"`
			RequestPath string `json:"request_path"`
		} `json:"next"`
	}
	raw := runCLI(t, "intent", "audit", "--owed", "--max", "2", "--json")
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("--owed --json is not JSON: %v\n%s", err, raw)
	}
	if len(got.Queue) != 2 || got.Owed != 3 || got.Max != 2 || got.Remaining != 1 {
		t.Fatalf("want 2 queued of 3 owed, 1 remaining:\n%s", raw)
	}
	if got.Queue[0].IntentID != "itd-21" || got.Queue[0].Shipped != "2026-01-01" || got.Queue[1].IntentID != "itd-20" {
		t.Fatalf("queue order wrong:\n%s", raw)
	}
	if got.Next == nil || got.Next.IntentID != "itd-21" || got.Next.RequestPath == "" ||
		got.Next.ReceiptID != got.Queue[0].ReceiptID || got.Queue[0].State != "OWED" {
		t.Fatalf("next does not name the head's request and receipt:\n%s", raw)
	}
}

func TestIntentAuditOwedNothingOwed(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliShipped+"/itd-12-s.md", cliShippedWithNotes("itd-12",
		"<!-- abcd-review: INGESTED receipt=rcp-0000000000b2 -->\nFidelity review — receipt rcp-0000000000b2."))
	before := snapshotTree(t, repo)
	out, err := runCLIErr(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatalf("nothing owed still exits 0: %v\n%s", err, out)
	}
	if s := string(out); !strings.Contains(s, "0 owed") || strings.Contains(s, "next:") {
		t.Fatalf("an empty drain names no next request:\n%s", s)
	}
	if after := snapshotTree(t, repo); len(after) != len(before) {
		t.Fatalf("an empty drain wrote: %d files -> %d", len(before), len(after))
	}
}

func TestIntentAuditOwedFlagRefusals(t *testing.T) {
	_ = drainRepo(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"intent", "audit", "--max", "2"}, "--max applies to --owed"},
		{[]string{"intent", "audit", "--owed", "itd-20"}, "takes no <itd-N>"},
		{[]string{"intent", "audit", "--owed", "--issue-drift"}, "--owed"},
		{[]string{"intent", "audit", "--owed", "--max", "-1"}, "got -1"},
	} {
		out, err := runCLIErr(t, c.args...)
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("%v: want exit 2, got %v\n%s", c.args, err, out)
			continue
		}
		if !strings.Contains(err.Error()+string(out), c.want) {
			t.Errorf("%v: refusal lacks %q: %v\n%s", c.args, c.want, err, out)
		}
	}
}

// TestIntentAuditOwedCarriesTheRouting (iss-2609252052385551): the drain's head
// is emitted the way `audit <itd-N>` emits it, so the request the host hands the
// auditor carries its routing section, the JSON next carries the routing
// member, and a --route override reaches both rather than being dropped.
func TestIntentAuditOwedCarriesTheRouting(t *testing.T) {
	repo := drainRepo(t)
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "--owed", "--json", "--route", "intent-auditor=economy")
	if err != nil {
		t.Fatalf("--owed --route must exit 0: %v\n%s", err, stderr)
	}
	var got struct {
		Next struct {
			IntentID    string                 `json:"intent_id"`
			RequestPath string                 `json:"request_path"`
			Routing     *oracle.RequestRouting `json:"routing"`
		} `json:"next"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if got.Next.IntentID != "itd-21" || got.Next.Routing == nil ||
		got.Next.Routing.Tier != oracle.Economy || got.Next.Routing.Override != "intent-auditor=economy" {
		t.Fatalf("next carries no routing for the override:\n%s", stdout)
	}
	doc, err := os.ReadFile(filepath.Join(repo, got.Next.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "## Routing\n\nrouting:\n  agent: intent-auditor\n  tier: economy\n") {
		t.Fatalf("the drain's request carries no routing section:\n%s", doc)
	}

	text, _, err := runCLISplit(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "routing: intent-auditor at tier host-decides") {
		t.Fatalf("the human rendering carries no routing line:\n%s", text)
	}

	// A --route naming an agent the drain does not dispatch is refused, as the
	// single audit refuses it.
	if _, _, err := runCLISplit(t, "intent", "audit", "--owed", "--route", "lifeboat-reviewer=economy"); exitCodeOf(err) != 2 {
		t.Fatalf("a --route for another agent must exit 2, got %v", err)
	}
}

// TestIntentAuditOwedUnreadableHistoryIsUnknown (iss-2609252052381777): when the
// history walk fails, the shipped days are unknown, and the listing says so
// rather than reading every committed intent as not yet committed.
func TestIntentAuditOwedUnreadableHistoryIsUnknown(t *testing.T) {
	repo := drainRepo(t)
	// HEAD on a branch that does not exist: the history walk cannot read it.
	cmd := exec.Command("git", "-C", repo, "symbolic-ref", "HEAD", "refs/heads/no-such-branch")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	text, stderr, err := runCLISplit(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatalf("an unreadable history must not refuse the drain: %v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "shipped days are unknown") {
		t.Fatalf("stderr does not say the days are unknown: %q", stderr)
	}
	if strings.Contains(text, "not yet committed") || strings.Count(text, "shipped day unknown") != 3 {
		t.Fatalf("every entry must read as day unknown, none as not yet committed:\n%s", text)
	}
	stdout, _, err := runCLISplit(t, "intent", "audit", "--owed", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Queue []struct {
			ShippedState string `json:"shipped_state"`
		} `json:"queue"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	for _, e := range got.Queue {
		if e.ShippedState != "unknown" {
			t.Fatalf("shipped_state must be unknown:\n%s", stdout)
		}
	}
}

// TestIntentAuditOwedBadHeadDoesNotBlock (iss-2609252052386874): the oldest
// owed intent's request cannot be emitted (its spec_id carries no number); the
// drain lists it with its error, exits 0, and emits the next entry's request.
func TestIntentAuditOwedBadHeadDoesNotBlock(t *testing.T) {
	repo := drainRepo(t)
	bad := strings.Replace(cliShippedWithNotes("itd-19", "_Empty._"), "spec_id: spc-1", "spec_id: none", 1)
	writeRepoFile(t, repo, cliShipped+"/itd-19-s.md", bad)
	commitShippedOn(t, repo, cliShipped+"/itd-19-s.md", "2025-12-01")

	text, stderr, err := runCLISplit(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatalf("a bad head must not block the drain: %v\n%s", err, stderr)
	}
	if !strings.Contains(text, "itd-19") || !strings.Contains(text, "not emitted") || !strings.Contains(text, "next: itd-21") {
		t.Fatalf("want itd-19 listed as not emitted and next itd-21:\n%s", text)
	}
	stdout, _, err := runCLISplit(t, "intent", "audit", "--owed", "--json", "--max", "1")
	if err != nil {
		t.Fatalf("a queue of one bad entry must still exit 0: %v", err)
	}
	var got struct {
		Queue []struct {
			IntentID  string `json:"intent_id"`
			EmitError string `json:"emit_error"`
		} `json:"queue"`
		Next *struct{} `json:"next"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if len(got.Queue) != 1 || got.Queue[0].IntentID != "itd-19" || got.Queue[0].EmitError == "" || got.Next != nil {
		t.Fatalf("want itd-19 with its emit_error and no next:\n%s", stdout)
	}
}
