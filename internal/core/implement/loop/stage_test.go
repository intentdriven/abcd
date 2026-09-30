package loop

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The lane's stages and the spec's steps are two words for two things
// (BU1, iss-2609291313276243): a spec lists steps, each landed by one lane, and
// the loop performs a lane's stages — worktree, brief, implement, validate,
// land. The state file, the step and receipt results and the refusal say
// "stage" for the lane's; only the spec's keep "step" (spec_step, step_title,
// pending).

// TestTheStateAndTheResultNameTheLaneStage: the state file's lane and record
// lines, the result of a step and a refusal carry the lane's stage under
// "stage", and no key of theirs says "step" for it.
func TestTheStateAndTheResultNameTheLaneStage(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	res, err := Advance(repo.Root(), start.RunID, f.steps(), Options{})
	if err != nil {
		t.Fatal(err)
	}

	var state struct {
		SchemaVersion int                          `json:"schema_version"`
		Lanes         []map[string]json.RawMessage `json:"lanes"`
		Record        []map[string]json.RawMessage `json:"record"`
	}
	if err := json.Unmarshal(stateBytes(t, repo.Root(), start.RunID), &state); err != nil {
		t.Fatal(err)
	}
	if state.SchemaVersion != 4 {
		t.Fatalf("the renamed shape is schema version 4, got %d", state.SchemaVersion)
	}
	for _, l := range state.Lanes {
		if string(l["stage"]) != `"brief"` {
			t.Fatalf("a lane names its next stage under \"stage\": %v", l)
		}
		if _, ok := l["step"]; ok {
			t.Fatalf("a lane still says \"step\" for its stage: %v", l)
		}
		if _, ok := l["spec_step"]; !ok {
			t.Fatalf("the spec's step keeps its word (spec_step): %v", l)
		}
	}
	for _, e := range state.Record {
		if _, ok := e["stage"]; !ok {
			t.Fatalf("a record line names its stage under \"stage\": %v", e)
		}
		if _, ok := e["step"]; ok {
			t.Fatalf("a record line still says \"step\": %v", e)
		}
	}

	var result map[string]json.RawMessage
	b, _ := json.Marshal(res)
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}
	if string(result["performed_stage"]) != `"worktree"` || string(result["stage"]) != `"brief"` {
		t.Fatalf("a step's result names the stage it performed and the next: %s", b)
	}
	for _, old := range []string{"performed", "step"} {
		if _, ok := result[old]; ok {
			t.Fatalf("a step's result still carries %q: %s", old, b)
		}
	}

	rb, _ := json.Marshal(refuse("worktree", "", "lane-1", "why", "remedy"))
	if !bytes.Contains(rb, []byte(`"stage":"worktree"`)) || bytes.Contains(rb, []byte(`"step"`)) {
		t.Fatalf("a refusal names the stage it happened at under \"stage\": %s", rb)
	}
	if got := refuse("worktree", "", "lane-1", "why", "remedy").Error(); !strings.HasPrefix(got, "refused at worktree for lane-1: why") {
		t.Fatalf("the refusal's line: %q", got)
	}
}

// TestAVersionThreeStateIsMigratedOnReadAndNeverRewrittenByTheRead: a state
// file written before the rename (versions 1 to 3, "step" for the lane's stage)
// is read, its stages carried over, without the read touching the file; the
// next mutation writes it at the current version under the new name. A
// pre-rename file that already says "stage" is not one its version wrote, and
// is refused.
func TestAVersionThreeStateIsMigratedOnReadAndNeverRewrittenByTheRead(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
	v3 := `{
  "schema_version": 3,
  "run_id": "` + start.RunID + `",
  "key": "itd-10",
  "intent": "itd-10",
  "spec": "spc-1",
  "driver": "host",
  "created_at": "2026-09-20T09:00:00Z",
  "updated_at": "2026-09-20T09:00:00Z",
  "lanes": [{"id": "lane-1", "key": "itd-10", "spec_step": 1, "step_title": "the whole spec", "step": "brief", "branch": "build/lane-1"}],
  "pending": [],
  "record": [{"at": "2026-09-20T09:00:00Z", "lane": "lane-1", "step": "start", "note": "checks passed"},
    {"at": "2026-09-20T09:01:00Z", "lane": "lane-1", "step": "worktree", "note": "worktree done"}]
}
`
	if err := os.WriteFile(path, []byte(v3), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatalf("a version-3 state is read: %v", err)
	}
	if st.SchemaVersion != SchemaVersion || len(st.Lanes) != 1 || st.Lanes[0].Stage != StageBrief || st.Lanes[0].Branch != "build/lane-1" {
		t.Fatalf("the lane's stage is carried over: %+v", st)
	}
	if len(st.Record) != 2 || st.Record[0].Stage != "start" || st.Record[1].Stage != "worktree" {
		t.Fatalf("the record's stages are carried over: %+v", st.Record)
	}
	if _, err := Runs(repo.Root()); err != nil {
		t.Fatal(err)
	}
	if got := stateBytes(t, repo.Root(), start.RunID); string(got) != v3 {
		t.Fatalf("a read rewrote the file:\n%s", got)
	}

	f := &fakeSteps{calls: map[Stage]int{}}
	res, err := Advance(repo.Root(), start.RunID, f.steps(), Options{})
	if err != nil || res.PerformedStage != StageBrief {
		t.Fatalf("a migrated run steps on from where it stood: %+v %v", res, err)
	}
	after := stateBytes(t, repo.Root(), start.RunID)
	if !bytes.Contains(after, []byte(`"schema_version": `+strconv.Itoa(SchemaVersion))) ||
		!bytes.Contains(after, []byte(`"stage": "implement"`)) || bytes.Contains(after, []byte(`"step":`)) {
		t.Fatalf("the next write carries the current version and name:\n%s", after)
	}

	for name, bad := range map[string]string{
		"a lane":        strings.Replace(v3, `"step": "brief"`, `"stage": "brief"`, 1),
		"a record line": strings.Replace(v3, `"step": "start"`, `"stage": "start"`, 1),
	} {
		if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadState(repo.Root(), start.RunID); err == nil || !strings.Contains(err.Error(), "never wrote") {
			t.Fatalf("%s saying \"stage\" in a version-3 file is refused: %v", name, err)
		}
	}
}
