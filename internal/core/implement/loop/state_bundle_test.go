package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A run carries members (itd-2611201450052326, spc-2611201455051897 step 1):
// an issue run keyed by its lead carries member issues beside it, each with
// its own footprint and reservation, and the run state's schema moves from 29
// to 30 to hold them. A run with no members writes what it wrote before, its
// version aside.

// The bundle fixture's issues: one category in one directory, each found at
// a file of its own there.
const (
	bundleLead  = "iss-2611201500000001"
	bundleTwo   = "iss-2611201500000002"
	bundleThree = "iss-2611201500000003"
)

// bundleFiles are the files each bundle issue is found at, in issue order.
var bundleFiles = map[string]string{bundleLead: "docs/a.md", bundleTwo: "docs/b.md", bundleThree: "docs/c.md"}

// bundleRepo is a drain repository on a local bare origin, P's checkout, whose
// ledger holds the three bundle issues, each a minor documentation issue
// found at its own page under docs/.
func bundleRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	_, w := builtOnce(t, worldKey("bundleRepo"), func(w *fixtureWorld) struct{} {
		repo := briefRepo(t, agentsMarked)
		issueRepoRule(t, repo)
		for _, id := range []string{bundleLead, bundleTwo, bundleThree} {
			repo.Write(bundleFiles[id], "# "+id+"\n")
		}
		for _, id := range []string{bundleLead, bundleTwo, bundleThree} {
			fileIssueAt(t, repo, id, capture.SeverityMinor, capture.Category("documentation"), "Correct the page's sentence.", bundleFiles[id]+":1")
		}
		repo.Commit("the ledger")
		gittest.InitBare(t, w.bare)
		repo.Git("remote", "add", "origin", w.bare)
		repo.Git("push", "-q", "origin", "main")
		repo.Git("fetch", "-q", "origin")
		asPat(t, repo)
		return struct{}{}
	})
	return w.repo, w.bare
}

// bundleRun starts the lead's run at at, reserving it on origin, and gives it
// the two members, each reserved on origin at at with its own page, as the
// drain's start of a bundle will (step 4). It returns the run's id.
func bundleRun(t *testing.T, repo *gittest.Repo, at time.Time) string {
	t.Helper()
	start, err := Start(repo.Root(), bundleLead, Options{Now: func() time.Time { return at }})
	if err != nil {
		t.Fatalf("starting the lead's run: %v", err)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}}, clockAt(at)); err != nil {
			t.Fatalf("reserving %s: %v", id, err)
		}
	}
	if err := mutate(repo.Root(), start.RunID, func(_ *os.Root, st *State) (bool, error) {
		for _, id := range []string{bundleTwo, bundleThree} {
			when := at
			st.Bundle = append(st.Bundle, Member{Issue: id, Footprint: []string{bundleFiles[id]}, ReservedAt: &when})
			st.Record = append(st.Record, Entry{At: at, Stage: StageReservation, Note: "reserved " + id, Issue: id})
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return start.RunID
}

// bundleIssues are the three issues of a bundle run, the lead first.
var bundleIssues = []string{bundleLead, bundleTwo, bundleThree}

// recordLines decodes the run state's record lines as stored.
func recordLines(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var doc struct {
		Record []map[string]any `json:"record"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Record
}

// TestARunWithNoMembersWritesNoBundleAndNoIssueOnItsLines: a run with no
// members, its reservation recorded, writes no `bundle` key, no
// `lead_hand_back` and no `issue` on any record line, so its file differs
// from version 29's only in its version.
func TestARunWithNoMembersWritesNoBundleAndNoIssueOnItsLines(t *testing.T) {
	repo, _ := bundleRepo(t)
	start, err := Start(repo.Root(), bundleLead, Options{})
	if err != nil {
		t.Fatal(err)
	}
	data := stateBytes(t, repo.Root(), start.RunID)
	top := topLevelFields(t, data)
	if _, ok := top["bundle"]; ok {
		t.Fatalf("a run with no members writes a bundle:\n%s", data)
	}
	if _, ok := top["lead_hand_back"]; ok {
		t.Fatalf("a run with no members writes a lead's hand-back:\n%s", data)
	}
	if !strings.Contains(string(data), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)) || SchemaVersion != 30 {
		t.Fatalf("the file states the current version, 30:\n%s", data)
	}
	lines := recordLines(t, data)
	if !slices.ContainsFunc(lines, func(l map[string]any) bool { return l["stage"] == StageReservation }) {
		t.Fatalf("premise: the run's reservation line is recorded: %v", lines)
	}
	for i, l := range lines {
		if _, ok := l["issue"]; ok {
			t.Fatalf("record line %d of a run with no members names an issue: %v", i+1, l)
		}
	}
}

// TestAVersion29FileReadsAsARunWithNoMembersAndIsWrittenBackAtTheCurrentVersion:
// version 29 is version 30's strict subset, read as a run with no members, and
// the run's next write takes it to the current version.
func TestAVersion29FileReadsAsARunWithNoMembersAndIsWrittenBackAtTheCurrentVersion(t *testing.T) {
	if schemaVersionUndigested != schemaVersionUnbundled+1 || schemaVersionUnbundled != schemaVersionUnreasoned+1 {
		t.Fatalf("a run's members are the version after %d: %d", schemaVersionUnbundled, schemaVersionUndigested)
	}
	repo, _ := bundleRepo(t)
	start, err := Start(repo.Root(), bundleLead, Options{})
	if err != nil {
		t.Fatal(err)
	}
	stateAtVersion(t, repo.Root(), start.RunID, schemaVersionUnbundled)
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatalf("a version-29 file reads: %v", err)
	}
	if st.SchemaVersion != SchemaVersion || st.hasMembers() || st.LeadHandBack != nil || !slices.Equal(st.Issues(), []string{bundleLead}) {
		t.Fatalf("it reads as the current version, a run of its lead alone: %d %+v %v", st.SchemaVersion, st.Bundle, st.Issues())
	}
	// The run's next write is its driver's, claimed as a step claims it.
	if err := claimToSave(repo.Root(), start.RunID, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := mutate(repo.Root(), start.RunID, func(_ *os.Root, st *State) (bool, error) {
		st.UpdatedAt = st.UpdatedAt.Add(time.Second)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if data := stateBytes(t, repo.Root(), start.RunID); !strings.Contains(string(data), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)) {
		t.Fatalf("the next write takes the file to version %d:\n%s", SchemaVersion, data)
	}
}

// TestAVersion29FileCarryingMembersIsRefusedNamingThem: a run's members, the
// lead's own hand-back and a record line's issue are version 30's, so a file
// of version 29 carrying any of them is refused naming it.
func TestAVersion29FileCarryingMembersIsRefusedNamingThem(t *testing.T) {
	cases := map[string]struct {
		edit func(doc map[string]any)
		want string
	}{
		"bundle": {func(doc map[string]any) {
			doc["bundle"] = []any{map[string]any{"issue": bundleTwo, "footprint": []any{"docs/b.md"}}}
		}, "`bundle`"},
		"lead_hand_back": {func(doc map[string]any) {
			doc["lead_hand_back"] = map[string]any{"at": "2026-11-20T09:00:00Z", "kind": "decision"}
		}, "`lead_hand_back`"},
		"issue on a line": {func(doc map[string]any) {
			rec := doc["record"].([]any)
			rec[0].(map[string]any)["issue"] = bundleLead
		}, "`issue` on record line 1"},
	}
	repo, _ := bundleRepo(t)
	start, err := Start(repo.Root(), bundleLead, Options{})
	if err != nil {
		t.Fatal(err)
	}
	good := stateBytes(t, repo.Root(), start.RunID)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(runStatePath(repo.Root(), start.RunID), good, 0o600); err != nil {
				t.Fatal(err)
			}
			data := stateAtVersion(t, repo.Root(), start.RunID, schemaVersionUnbundled)
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			tc.edit(doc)
			out, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(runStatePath(repo.Root(), start.RunID), out, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = ReadState(repo.Root(), start.RunID)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "version 29 never wrote") {
				t.Fatalf("a version-29 file carrying %s is refused naming it: %v", name, err)
			}
		})
	}
}

// TestAVersion30FileIsANewerAbcdsToAnOlderReader: a reader refuses a file of a
// version above its own as a newer abcd's before any strict decode, so a field
// the newer version added is never read as a file that does not parse. Version
// 30's `bundle` stands for that field here: a file one version above this
// reader's, carrying a member and a key no version writes yet, is refused as a
// newer abcd's, as a reader at 29 refuses a version-30 file carrying `bundle`.
func TestAVersion30FileIsANewerAbcdsToAnOlderReader(t *testing.T) {
	repo, _ := bundleRepo(t)
	runID := bundleRun(t, repo, time.Now().UTC().Truncate(time.Second))
	data := stateBytes(t, repo.Root(), runID)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc["schema_version"] = SchemaVersion + 1
	doc["a_later_field"] = true
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runStatePath(repo.Root(), runID), out, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ReadState(repo.Root(), runID)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("is schema version %d, which a newer abcd wrote", SchemaVersion+1)) {
		t.Fatalf("a file of a later version is refused as a newer abcd's: %v", err)
	}
}

// TestARunWithMembersRoundTrips: a run's members, each with its footprint,
// reservation, hand-back and split, the lead's own hand-back and a line's
// issue are read back as written.
func TestARunWithMembersRoundTrips(t *testing.T) {
	repo, _ := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, at)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Bundle[1].HandBack = &HandBack{At: at, Kind: "decision", Reason: "the page's owner decides", Home: "the record"}
		st.LeadHandBack = &HandBack{At: at, Kind: "decision", Reason: "the lead's owner decides", Home: "the record"}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Bundle) != 2 || st.Bundle[0].Issue != bundleTwo || !slices.Equal(st.Bundle[0].Footprint, []string{"docs/b.md"}) ||
		st.Bundle[0].ReservedAt == nil || !st.Bundle[0].ReservedAt.Equal(at) || st.Bundle[1].HandBack == nil || st.Bundle[1].HandBack.Reason != "the page's owner decides" {
		t.Fatalf("the members are read back as written: %+v", st.Bundle)
	}
	if st.LeadHandBack == nil || st.LeadHandBack.Reason != "the lead's owner decides" {
		t.Fatalf("the lead's own hand-back is read back: %+v", st.LeadHandBack)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Issue == bundleThree && e.Stage == StageReservation }) {
		t.Fatalf("a line naming its issue is read back: %+v", st.Record)
	}
}

// TestAHandEditedMemberIsRefused: a run's members are ones the loop writes,
// so a member on an intent's run, an id that is no issue's or is the run's
// twice, a footprint that is no plain path, a lead's own hand-back with no
// members, and a line naming an issue the run does not carry are each refused.
func TestAHandEditedMemberIsRefused(t *testing.T) {
	repo, _ := bundleRepo(t)
	runID := bundleRun(t, repo, time.Now().UTC().Truncate(time.Second))
	good := stateBytes(t, repo.Root(), runID)
	cases := map[string]func(doc map[string]any){
		"an intent's run": func(doc map[string]any) { doc["key"] = "itd-10" },
		"no issue id": func(doc map[string]any) {
			doc["bundle"].([]any)[0].(map[string]any)["issue"] = "itd-11"
		},
		"the lead twice": func(doc map[string]any) {
			doc["bundle"].([]any)[0].(map[string]any)["issue"] = bundleLead
		},
		"a member twice": func(doc map[string]any) {
			doc["bundle"].([]any)[1].(map[string]any)["issue"] = bundleTwo
		},
		"a path that is not plain": func(doc map[string]any) {
			doc["bundle"].([]any)[0].(map[string]any)["footprint"] = []any{"../docs/b.md"}
		},
		"no footprint": func(doc map[string]any) {
			doc["bundle"].([]any)[0].(map[string]any)["footprint"] = []any{}
		},
		"a hand-back with a control": func(doc map[string]any) {
			doc["bundle"].([]any)[0].(map[string]any)["hand_back"] = map[string]any{"at": "2026-11-20T09:00:00Z", "reason": "bad\x1b[31m"}
		},
		"a lead's hand-back with no members": func(doc map[string]any) {
			delete(doc, "bundle")
			rec := doc["record"].([]any)
			for _, l := range rec {
				delete(l.(map[string]any), "issue")
			}
			doc["lead_hand_back"] = map[string]any{"at": "2026-11-20T09:00:00Z", "kind": "decision"}
		},
		"a line naming another issue": func(doc map[string]any) {
			rec := doc["record"].([]any)
			rec[0].(map[string]any)["issue"] = eligibleIssue
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(good, &doc); err != nil {
				t.Fatal(err)
			}
			edit(doc)
			out, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(runStatePath(repo.Root(), runID), out, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(repo.Root(), runID); err == nil || !strings.Contains(err.Error(), "members the loop never writes") {
				t.Fatalf("%s is refused as members the loop never writes: %v", name, err)
			}
		})
	}
}

// TestIssuesAndCoversReadTheLeadAndItsMembers: Issues() is the lead and each
// member not handed back alone, the lead too once handed back alone; covers
// is true for an issue of Issues(), false for one handed back alone, and false
// for every issue of a run the drain has split.
func TestIssuesAndCoversReadTheLeadAndItsMembers(t *testing.T) {
	at := time.Now()
	hb := &HandBack{At: at, Kind: "decision"}
	run := func(lead *HandBack, members ...Member) State {
		return State{Key: bundleLead, LeadHandBack: lead, Bundle: members}
	}
	two, three := Member{Issue: bundleTwo, Footprint: []string{"docs/b.md"}}, Member{Issue: bundleThree, Footprint: []string{"docs/c.md"}}
	backed := three
	backed.HandBack = hb
	cases := []struct {
		name   string
		st     State
		issues []string
		covers map[string]bool
	}{
		{"a lead alone", run(nil), []string{bundleLead}, map[string]bool{bundleLead: true, bundleTwo: false}},
		{"a lead and members", run(nil, two, three), bundleIssues, map[string]bool{bundleLead: true, bundleTwo: true, bundleThree: true, eligibleIssue: false}},
		{"a member handed back", run(nil, two, backed), []string{bundleLead, bundleTwo}, map[string]bool{bundleLead: true, bundleTwo: true, bundleThree: false}},
		{"the lead handed back alone", run(hb, two, three), []string{bundleTwo, bundleThree}, map[string]bool{bundleLead: false, bundleTwo: true, bundleThree: true}},
	}
	split := run(nil, two, three)
	for i := range split.Bundle {
		split.Bundle[i].Split = true
	}
	cases = append(cases, struct {
		name   string
		st     State
		issues []string
		covers map[string]bool
	}{"a run split", split, bundleIssues, map[string]bool{bundleLead: false, bundleTwo: false, bundleThree: false}})
	for _, tc := range cases {
		if got := tc.st.Issues(); !slices.Equal(got, tc.issues) {
			t.Errorf("%s: Issues() = %v; want %v", tc.name, got, tc.issues)
		}
		for id, want := range tc.covers {
			if got := tc.st.covers(id); got != want {
				t.Errorf("%s: covers(%s) = %v; want %v", tc.name, id, got, want)
			}
		}
	}
}

// TestARunsMembersAreMaskedWhenStored: what version 30 added is stored masked
// as the rest of the state is, so a member's hand-back carrying a bidi
// override and a line's issue are written as plain text the next read takes.
func TestARunsMembersAreMaskedWhenStored(t *testing.T) {
	repo, _ := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, at)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Bundle[0].HandBack = &HandBack{At: at, Kind: "decision", Reason: "the page‮'s owner decides"}
		st.LeadHandBack = &HandBack{At: at, Kind: "decision", Reason: "the lead‮'s owner decides"}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	data := stateBytes(t, repo.Root(), runID)
	if strings.ContainsRune(string(data), '‮') || strings.Contains(string(data), `‮`) {
		t.Fatalf("a member's text is stored unmasked:\n%s", data)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatalf("the stored state reads: %v", err)
	}
	if st.Bundle[0].HandBack == nil || !strings.Contains(st.Bundle[0].HandBack.Reason, "owner decides") || st.LeadHandBack == nil {
		t.Fatalf("the masked hand-backs are read back: %+v %+v", st.Bundle[0].HandBack, st.LeadHandBack)
	}
}

// TestARunWithEveryIssueHandedBackAndNoLaneHandedBackIsRefused: a run with
// members whose lead and every member are handed back alone holds no issue, and
// the loop hands its lane back as it hands back the last, so such a run with no
// lane handed back is refused as members the loop never writes; one issue still
// held is not, and nor is every issue handed back beside a lane carrying its
// hand-back at handed-back or at discarded, the stage a discard moves such a
// lane on to.
func TestARunWithEveryIssueHandedBackAndNoLaneHandedBackIsRefused(t *testing.T) {
	repo, _ := bundleRepo(t)
	runID := bundleRun(t, repo, time.Now().UTC().Truncate(time.Second))
	good := stateBytes(t, repo.Root(), runID)
	handBack := func() map[string]any { return map[string]any{"at": "2026-11-20T09:00:00Z", "kind": "decision"} }
	for name, c := range map[string]struct {
		all     bool
		lane    Stage
		refused bool
	}{
		"every issue handed back":                       {all: true, refused: true},
		"one member still held":                         {all: false},
		"every issue handed back, lane handed back":     {all: true, lane: StageHandedBack},
		"every issue handed back, lane discarded after": {all: true, lane: StageDiscarded},
	} {
		t.Run(name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(good, &doc); err != nil {
				t.Fatal(err)
			}
			doc["lead_hand_back"] = handBack()
			members := doc["bundle"].([]any)
			for i, m := range members {
				if c.all || i < len(members)-1 {
					m.(map[string]any)["hand_back"] = handBack()
				}
			}
			if c.lane != "" {
				lane := doc["lanes"].([]any)[0].(map[string]any)
				lane["stage"] = string(c.lane)
				lane["hand_back"] = handBack()
			}
			out, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(runStatePath(repo.Root(), runID), out, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = ReadState(repo.Root(), runID)
			if refused := err != nil && strings.Contains(err.Error(), "members the loop never writes"); refused != c.refused || (!c.refused && err != nil) {
				t.Fatalf("refused %v, want %v: %v", refused, c.refused, err)
			}
		})
	}
}
