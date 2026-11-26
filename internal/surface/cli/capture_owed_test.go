package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/findings"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// capture_owed_test.go — `abcd capture owed` and `abcd capture … --fixed`
// (step 1 of spc-2611252115253843, itd-2611251831138387).

// findingsRepo is a managed checkout with one commit, so its list has a key,
// under a home of its own, outside any session and any stage.
func findingsRepo(t *testing.T) string {
	t.Helper()
	root := managedCheckout(t)
	gitCommitAt(t, root, "root")
	for _, k := range driveSessionMarkers {
		t.Setenv(k, "")
	}
	t.Setenv(runner.StageWritesEnv, "")
	t.Setenv(runner.StageHomeEnv, "")
	return root
}

// owedStore is the list of root's repository under the test's home.
func owedStore(t *testing.T, root string) findings.Store {
	t.Helper()
	sha, err := gitutil.RootCommit(root)
	if err != nil || sha == "" {
		t.Fatalf("root commit: %q %v", sha, err)
	}
	home, err := abcdhome.MachineHome()
	if err != nil {
		t.Fatal(err)
	}
	s, err := findings.NewStore(home, sha)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// plantEntry puts one entry of kind on root's list straight through the
// store, for the kinds the command cannot make, and returns its handle.
func plantEntry(t *testing.T, root string, kind findings.Kind, text string) string {
	t.Helper()
	got, bad, err := owedStore(t, root).Put(context.Background(), findings.Entry{
		Kind: kind, Checkout: root, Session: "s-plant", Text: text, At: time.Now(),
	})
	if err != nil || len(bad) != 0 || len(got) != 1 {
		t.Fatalf("plant %s: %v %v %v", kind, got, bad, err)
	}
	return got[0].Handle()
}

type owedEntryJSON struct {
	Handle   string `json:"handle"`
	Kind     string `json:"kind"`
	Session  string `json:"session"`
	Checkout string `json:"checkout"`
	Text     string `json:"text"`
	Settled  *struct {
		Act    string `json:"act"`
		Record string `json:"record"`
		Reason string `json:"reason"`
	} `json:"settled"`
}

type owedListJSON struct {
	Open         []owedEntryJSON `json:"open"`
	OpenCount    int             `json:"open_count"`
	More         int             `json:"more"`
	PrivateCount int             `json:"private_count"`
	Private      []owedEntryJSON `json:"private"`
	Settled      []owedEntryJSON `json:"settled"`
}

type owedPutJSON struct {
	Put []owedEntryJSON `json:"put"`
}

// owedPut puts one entry from stdin with the adding flag and returns it.
func owedPut(t *testing.T, flag, text string) owedEntryJSON {
	t.Helper()
	var got owedPutJSON
	if err := json.Unmarshal(runCLIStdin(t, text, "capture", "owed", flag, "-", "--json"), &got); err != nil || len(got.Put) != 1 {
		t.Fatalf("owed %s -: %v %+v", flag, err, got)
	}
	return got.Put[0]
}

// owedList is the bare list in --json.
func owedList(t *testing.T, args ...string) owedListJSON {
	t.Helper()
	var got owedListJSON
	raw := runCLI(t, append([]string{"capture", "owed", "--json"}, args...)...)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("owed --json: %v\n%s", err, raw)
	}
	return got
}

// owedRefusal runs a call that must refuse and returns its exit and code.
func owedRefusal(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	out, err := runCLIStdinErr(t, stdin, args...)
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("%v: want a refusal, got %v\n%s", args, err, out)
	}
	return ee.Code, string(ee.ref), ee.Msg
}

// TestCaptureOwedPutsEachAddingKindAndListsIt: --untested, --doubt and
// --handed-up each put one entry from standard input for this checkout and
// the session of the environment; the bare list names each by handle, kind
// and text, in the text form and in --json, and the same text put twice in
// one session is one entry.
func TestCaptureOwedPutsEachAddingKindAndListsIt(t *testing.T) {
	root := findingsRepo(t)
	t.Setenv(hostSessionIDEnv, "s-adding")
	texts := map[string]string{
		"--untested":  "the parser drops the last line of a file",
		"--doubt":     "the board may count a lane twice",
		"--handed-up": "the writer loses the final newline",
	}
	handles := map[string]string{}
	for flag, text := range texts {
		e := owedPut(t, flag, text)
		if want := strings.TrimPrefix(flag, "--"); e.Kind != want || e.Session != "s-adding" || e.Text != text || !strings.HasPrefix(e.Handle, "f-") {
			t.Errorf("%s put %+v, want kind %s, the session and the text", flag, e, want)
		}
		handles[flag] = e.Handle
	}
	if again := owedPut(t, "--untested", texts["--untested"]); again.Handle != handles["--untested"] {
		t.Errorf("the same finding said twice in one session is %s and %s, want one entry", again.Handle, handles["--untested"])
	}
	got := owedList(t)
	if got.OpenCount != 3 || len(got.Open) != 3 {
		t.Fatalf("the list holds %d (%d shown), want 3: %+v", got.OpenCount, len(got.Open), got.Open)
	}
	text := string(runCLI(t, "capture", "owed"))
	for flag, h := range handles {
		if !strings.Contains(text, h) || !strings.Contains(text, texts[flag]) || !strings.Contains(text, strings.TrimPrefix(flag, "--")) {
			t.Errorf("the text list does not name %s, its kind and its text:\n%s", h, text)
		}
	}
	if !strings.Contains(text, "abcd capture owed <handle>") {
		t.Errorf("the text list does not name the clearing act:\n%s", text)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
}

// TestCaptureOwedSettlesByTheKindTable holds every cell of the spec's kind
// table at the command: an act the table marks settles the entry, which
// leaves the open list for the settled one carrying the act and its record
// or reason; an act it does not mark is refused and the entry stays open.
func TestCaptureOwedSettlesByTheKindTable(t *testing.T) {
	root := findingsRepo(t)
	issue := captureIssueForProbe(t, "the parser drops the last line of a file", "minor")
	acts := map[string][]string{
		"captured":      {"--captured", issue},
		"settled":       {"--settled=" + issue},
		"refuted":       {"--refuted", "ran the parser over an empty file; it keeps the line"},
		"own":           {"--own", "the base passes the same check at its head"},
		"not-a-finding": {"--not-a-finding", "the word was prose about a test name"},
		"private":       {"--private"},
	}
	table := map[findings.Kind][]string{
		findings.KindUntested: {"captured", "settled", "refuted", "private"},
		findings.KindHandedUp: {"captured", "settled", "refuted", "private"},
		findings.KindDoubt:    {"captured", "settled"},
		findings.KindKnown:    {"captured", "settled", "refuted", "private"},
		findings.KindLine:     {"captured", "settled", "refuted", "not-a-finding", "private"},
		findings.KindCheck:    {"captured", "settled", "own", "private"},
	}
	n := 0
	for kind, allowed := range table {
		for act, flags := range acts {
			n++
			h := plantEntry(t, root, kind, "a finding of kind "+string(kind)+" for the act "+act)
			args := append([]string{"capture", "owed", h}, flags...)
			out, err := runCLIStdinErr(t, "", args...)
			want := false
			for _, a := range allowed {
				want = want || a == act
			}
			open := map[string]bool{}
			for _, e := range owedList(t).Open {
				open[e.Handle] = true
			}
			if !want {
				var ee *exitError
				if !errors.As(err, &ee) || ee.Code != 2 || ee.ref != "capture/invalid-input" {
					t.Errorf("%s --%s: want refused capture/invalid-input, got %v\n%s", kind, act, err, out)
				}
				if !open[h] {
					t.Errorf("%s --%s: a refused settle took %s off the list", kind, act, h)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s --%s: refused: %v\n%s", kind, act, err, out)
				continue
			}
			if open[h] {
				t.Errorf("%s --%s: %s is still open", kind, act, h)
			}
		}
	}
	settled := owedList(t, "--settled").Settled
	byAct := map[string]bool{}
	for _, e := range settled {
		if e.Settled != nil {
			byAct[e.Settled.Act] = true
			if e.Settled.Act == "refuted" && e.Settled.Reason != "ran the parser over an empty file; it keeps the line" {
				t.Errorf("the refutation's test is not kept: %+v", e.Settled)
			}
			if e.Settled.Act == "captured" && e.Settled.Record != issue {
				t.Errorf("the captured act does not name its record: %+v", e.Settled)
			}
		}
	}
	for _, act := range []string{"captured", "settled", "refuted", "own", "not-a-finding"} {
		if !byAct[act] {
			t.Errorf("--settled lists no entry settled by --%s: %+v", act, settled)
		}
	}
	if got := owedList(t).PrivateCount; got != 5 {
		t.Errorf("the private list counts %d, want the 5 entries --private moved there", got)
	}
	_ = n
}

// TestCaptureOwedSettledTakesItsRecordAsAWord: --settled given bare beside
// handles takes the one word that is not a handle as its record, the way the
// block writes it (settled <record>).
func TestCaptureOwedSettledTakesItsRecordAsAWord(t *testing.T) {
	root := findingsRepo(t)
	issue := captureIssueForProbe(t, "the reader breaks on an empty input file", "minor")
	h := plantEntry(t, root, findings.KindUntested, "the reader breaks on an empty input")
	runCLI(t, "capture", "owed", h, "--settled", issue)
	if open := owedList(t).Open; len(open) != 0 {
		t.Fatalf("--settled <record> left %+v open", open)
	}
}

// TestCaptureOwedAsksTheRecords holds criterion 12 at the settle: a captured
// claim on an issue the ledger does not hold is capture/unknown-issue, and
// one on a wontfix, which does not answer a capture, is refused naming the
// state; either way the entry stays open.
func TestCaptureOwedAsksTheRecords(t *testing.T) {
	root := findingsRepo(t)
	gone := captureIssueForProbe(t, "the writer loses the final newline", "minor")
	runCLI(t, "capture", "wontfix", gone, "the behaviour is the documented one")
	h := plantEntry(t, root, findings.KindUntested, "the writer loses the final newline")
	if code, ref, _ := owedRefusal(t, "", "capture", "owed", h, "--captured", unknownIssueID); code != 2 || ref != "capture/unknown-issue" {
		t.Errorf("an absent issue: exit %d %s, want 2 capture/unknown-issue", code, ref)
	}
	code, ref, msg := owedRefusal(t, "", "capture", "owed", h, "--captured", gone)
	if code != 2 || ref != "capture/invalid-input" || !strings.Contains(msg, "wontfix") {
		t.Errorf("a wontfix: exit %d %s %q, want 2 capture/invalid-input naming the state", code, ref, msg)
	}
	if open := owedList(t).Open; len(open) != 1 || open[0].Handle != h {
		t.Errorf("a refused claim took the entry off the list: %+v", open)
	}
}

// TestCaptureOwedRefusesWhatItCannotRead: the flags' shapes. Two acts at
// once conflict; a handle with no act, an act with no handle, an adding flag
// with no "-" and an empty text are each missing something; an adding flag
// beside handles conflicts; nothing is written by any.
func TestCaptureOwedRefusesWhatItCannotRead(t *testing.T) {
	root := findingsRepo(t)
	h := plantEntry(t, root, findings.KindUntested, "a finding the refusals leave alone")
	for _, tc := range []struct {
		stdin string
		args  []string
		want  string
	}{
		{"", []string{h, "--refuted", "a test", "--own", "a run"}, "usage/flag-conflict"},
		{"", []string{h, "--private", "--captured", "iss-1"}, "usage/flag-conflict"},
		{"x y", []string{"--untested", "--doubt", "-"}, "usage/flag-conflict"},
		{"x y", []string{h, "--untested"}, "usage/missing-argument"},
		{"", []string{h}, "usage/missing-required-flag"},
		{"", []string{"--refuted", "a test"}, "usage/missing-argument"},
		{"x y", []string{"--untested"}, "usage/missing-argument"},
		{"  \n", []string{"--untested", "-"}, "usage/missing-argument"},
		{"", []string{h, "--settled"}, "usage/missing-argument"},
		{"", []string{"f-zzzz", "--refuted", "a test"}, "capture/invalid-input"},
		{"", []string{"f-00000000", "--refuted", "a test"}, "capture/invalid-input"},
	} {
		code, ref, msg := owedRefusal(t, tc.stdin, append([]string{"capture", "owed"}, tc.args...)...)
		if code != 2 || ref != tc.want {
			t.Errorf("%v: exit %d %s (%q), want 2 %s", tc.args, code, ref, msg, tc.want)
		}
	}
	if got := owedList(t); got.OpenCount != 1 || got.PrivateCount != 0 {
		t.Errorf("a refusal changed the list: %+v", got)
	}
}

// TestCaptureOwedBareWritesNothing: a bare call in a fresh checkout makes no
// findings store under abcd's home, and lists nothing.
func TestCaptureOwedBareWritesNothing(t *testing.T) {
	findingsRepo(t)
	home, err := abcdhome.MachineHome()
	if err != nil {
		t.Fatal(err)
	}
	text := string(runCLI(t, "capture", "owed"))
	if !strings.Contains(text, "nothing owed") {
		t.Errorf("an empty list does not say so:\n%s", text)
	}
	owedList(t)
	owedList(t, "--settled")
	if _, err := os.Lstat(abcdhome.Path(home, "findings")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a bare call made the findings store: %v", err)
	}
}

// TestCaptureOwedRunsInASharedMainCheckout: the list is under abcd's home,
// not in the checkout, so a main checkout two live sessions started in is
// not refused.
func TestCaptureOwedRunsInASharedMainCheckout(t *testing.T) {
	repo, _ := sharedCheckout(t)
	t.Chdir(repo)
	if _, err := runCLIStdinErr(t, "a finding met in a shared checkout", "capture", "owed", "--untested", "-"); err != nil {
		t.Fatalf("a shared main checkout refused the put: %v", err)
	}
	if got := owedList(t); got.OpenCount != 1 {
		t.Errorf("the put did not reach the list: %+v", got)
	}
}

// TestCaptureOwedShowsAPrivateTextOnlyToThePerson is criterion 11 at the
// command: a private text is printed only where the input and the output are
// a terminal, no session marker is set and no stage runs; everywhere else,
// and always in --json, the count alone. Only the person settles it.
func TestCaptureOwedShowsAPrivateTextOnlyToThePerson(t *testing.T) {
	findingsRepo(t)
	const planted = "the token check compares by prefix-7f3a only"
	put := owedPut(t, "--private", planted)
	if put.Kind != "private" || put.Text != "" {
		t.Fatalf("the put echoed the private entry's text or kind: %+v", put)
	}
	terminal := func(on bool) {
		was := connectIsTerminal
		t.Cleanup(func() { connectIsTerminal = was })
		connectIsTerminal = func(any) bool { return on }
	}
	absent := func(where string) {
		t.Helper()
		text := string(runCLI(t, "capture", "owed"))
		if strings.Contains(text, planted) || !strings.Contains(text, "1 private") {
			t.Errorf("%s: want the count and not the text:\n%s", where, text)
		}
		if raw := runCLI(t, "capture", "owed", "--json"); strings.Contains(string(raw), planted) {
			t.Errorf("%s: --json carries the text:\n%s", where, raw)
		}
	}
	absent("a pipe")
	terminal(true)
	t.Setenv("CLAUDECODE", "1")
	absent("a session marker")
	t.Setenv("CLAUDECODE", "")
	t.Setenv(runner.StageWritesEnv, "/stage")
	absent("a stage")
	if code, ref, _ := owedRefusal(t, "", "capture", "owed", put.Handle, "--refuted", "an agent's test"); code != 2 || ref != "capture/invalid-input" {
		t.Errorf("an agent settled a private entry: exit %d %s", code, ref)
	}
	t.Setenv(runner.StageWritesEnv, "")
	if raw := runCLI(t, "capture", "owed", "--json"); strings.Contains(string(raw), planted) {
		t.Errorf("--json at the person's terminal carries the text:\n%s", raw)
	}
	if text := string(runCLI(t, "capture", "owed")); !strings.Contains(text, planted) || !strings.Contains(text, put.Handle) {
		t.Errorf("the person's terminal does not show the private text:\n%s", text)
	}
	runCLI(t, "capture", "owed", put.Handle, "--refuted", "the person ran the check")
	if got := owedList(t); got.PrivateCount != 0 {
		t.Errorf("the person's settle left the private entry: %+v", got)
	}
}

// TestCaptureOwedListsAHundredAndCountsTheRest: the bare list shows a hundred
// entries at most, this checkout's first, and then the count of the rest.
func TestCaptureOwedListsAHundredAndCountsTheRest(t *testing.T) {
	root := findingsRepo(t)
	es := []findings.Entry{{Kind: findings.KindUntested, Checkout: "/elsewhere/checkout", Session: "s-far", Text: "a finding met in another checkout", At: time.Now().Add(-time.Hour)}}
	for i := 0; i < 101; i++ {
		es = append(es, findings.Entry{Kind: findings.KindUntested, Checkout: root, Session: "s-many",
			Text: "finding number " + strings.Repeat("i", i+1), At: time.Now()})
	}
	if _, _, err := owedStore(t, root).Put(context.Background(), es...); err != nil {
		t.Fatal(err)
	}
	got := owedList(t)
	if got.OpenCount != 102 || len(got.Open) != 100 || got.More != 2 {
		t.Fatalf("the list shows %d of %d with %d more, want 100 of 102 with 2 more", len(got.Open), got.OpenCount, got.More)
	}
	for _, e := range got.Open {
		if strings.Contains(e.Text, "another checkout") {
			t.Errorf("another checkout's entry is shown before this checkout's are all shown")
		}
	}
	if text := string(runCLI(t, "capture", "owed")); !strings.Contains(text, "and 2 more") {
		t.Errorf("the text list does not count the rest:\n%s", text)
	}
}

// TestCaptureFixedFilesOneResolvedRecord is criterion 9 at the front door:
// `capture … --fixed … --impact …` writes the one record into resolved/
// naming what fixed it, and a refusal writes nothing.
func TestCaptureFixedFilesOneResolvedRecord(t *testing.T) {
	repo := captureLedgerRepo(t)
	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	raw := runCLI(t, "capture", "the gate read the base and not the head", "--fixed", "the gate reads the head it is offered", "--impact", "fix", "--json")
	if err := json.Unmarshal(raw, &got); err != nil || got.Status != "resolved" {
		t.Fatalf("--fixed: %v %+v\n%s", err, got, raw)
	}
	recs := issueRecords(t, repo)
	if len(recs) != 1 || !strings.Contains(filepath.ToSlash(recs[0]), "/resolved/"+got.ID+"-") {
		t.Fatalf("--fixed left %v, want one record in resolved/", recs)
	}
	body, _ := os.ReadFile(recs[0])
	if !strings.Contains(string(body), "the gate reads the head it is offered") {
		t.Errorf("the record does not name what fixed it:\n%s", body)
	}
	for _, args := range [][]string{
		{"capture", "the gate read the base once more", "--fixed", "the gate reads the head"},
		{"capture", "the gate read the base once more", "--impact", "fix"},
	} {
		if _, err := runCLIErr(t, args...); err == nil {
			t.Errorf("%v was not refused", args)
		}
	}
	if recs := issueRecords(t, repo); len(recs) != 1 {
		t.Errorf("a refused --fixed capture wrote: %v", recs)
	}
}
