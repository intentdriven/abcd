package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// statusRecord lays, in the managed checkout at root, a record with one READY
// planned intent, one the readiness gate refuses, and one draft.
func statusRecord(t *testing.T, root string) {
	t.Helper()
	w := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	body := func(id, title, spec string) string {
		return "---\nid: " + id + "\nslug: s\nspec_id: " + spec + "\nkind: standalone\n---\n# " + title + "\n\n" +
			"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
			"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n"
	}
	w(".abcd/development/intents/planned/itd-2609010000000001-ready.md", body("itd-2609010000000001", "The ready one", "spc-2609010000000011"))
	w(".abcd/development/specs/open/spc-2609010000000011-ready.md",
		"---\nid: spc-2609010000000011\nslug: s\nintent: itd-2609010000000001\n---\n# s\n\n## Summary\n\nA written design record.\n")
	w(".abcd/development/intents/planned/itd-7-unlinked.md", body("itd-7", "The unlinked one", "null"))
	w(".abcd/development/intents/drafts/itd-9-idea.md", body("itd-9", "An idea", "null"))
}

// writeRunState writes one run's state file with a lane in progress on intentID.
func writeRunState(t *testing.T, root, intentID string) {
	t.Helper()
	const runID = "run-2609290000000001"
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	st := loop.State{
		SchemaVersion: loop.SchemaVersion, RunID: runID, Key: intentID, Intent: intentID, Spec: "spc-2609010000000011",
		Driver: loop.DriverHost, CreatedAt: now, UpdatedAt: now,
		Lanes:   []loop.Lane{{ID: "lane-1", Key: intentID, SpecStep: 1, StepTitle: "the whole spec", Step: loop.StepImplement}},
		Pending: []loop.PendingStep{}, Record: []loop.Entry{},
	}
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.FromSlash(loop.RunRelDir), runID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, loop.StateFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestBoardCarriesTheStatusBlock is criteria 1 and 4 at the bare board: in a
// managed checkout the text carries Now, Next and Later with each row's id and
// title — the lane row with its lane state, the head marked next up, the
// refused intent with its failing check, the draft — and --json carries the
// three lists.
func TestBoardCarriesTheStatusBlock(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	writeRunState(t, root, "itd-7")

	text := string(runCLI(t))
	for _, want := range []string{
		"  status:     Now 2 · Next 1 · Later 2",
		"    Now:\n      itd-7  The unlinked one  [lane-1: implement (run-2609290000000001)]\n      itd-2609010000000001  The ready one  [next up]\n",
		"    Next:\n      itd-2609010000000001  The ready one\n",
		"    Later:\n      itd-7  The unlinked one  [fails: spec_link, spec_body]\n      itd-9  An idea  [draft]\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the board lacks\n%s\nin\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "roadmap") {
		t.Errorf("the board says roadmap:\n%s", text)
	}

	var got struct {
		Status *struct {
			Now []struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				NextUp bool   `json:"next_up"`
				Lane   *struct {
					Run, Lane, Step string
				} `json:"lane"`
			} `json:"now"`
			Next []struct {
				ID string `json:"id"`
			} `json:"next"`
			Later []struct {
				ID      string   `json:"id"`
				Title   string   `json:"title"`
				Failing []string `json:"failing_checks"`
			} `json:"later"`
		} `json:"status"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	s := got.Status
	if s == nil || len(s.Now) != 2 || len(s.Next) != 1 || len(s.Later) != 2 {
		t.Fatalf("--json status = %+v, want two Now rows, one Next, two Later", s)
	}
	if s.Now[0].Lane == nil || s.Now[0].Lane.Step != "implement" || s.Now[0].Lane.Run != "run-2609290000000001" || !s.Now[1].NextUp {
		t.Errorf("--json Now = %+v, want the lane state then the head", s.Now)
	}
	if s.Later[0].ID != "itd-7" || s.Later[0].Title != "The unlinked one" || len(s.Later[0].Failing) == 0 {
		t.Errorf("--json Later[0] = %+v, want itd-7 with its title and failing checks", s.Later[0])
	}
}

// TestBoardWithoutAStateFileKeepsOnlyTheHead is criterion 3 at the board: with
// the state file gone, Now holds only the head, and every other line of the
// board is what it was.
func TestBoardWithoutAStateFileKeepsOnlyTheHead(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	writeRunState(t, root, "itd-7")
	with := string(runCLI(t))
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(loop.RunRelDir))); err != nil {
		t.Fatal(err)
	}
	without := string(runCLI(t))

	if !strings.Contains(without, "    Now:\n      itd-2609010000000001  The ready one  [next up]\n    Next:") {
		t.Errorf("without a state file Now is not the head alone:\n%s", without)
	}
	// Every line but the lane row and Now's count is unchanged.
	var kept []string
	for _, l := range strings.Split(with, "\n") {
		if strings.Contains(l, "(run-2609290000000001)") {
			continue
		}
		kept = append(kept, strings.Replace(l, "Now 2 ·", "Now 1 ·", 1))
	}
	if got := strings.Join(kept, "\n"); got != without {
		t.Errorf("removing the state file changed more than Now's lane rows:\nwith (lane row dropped)\n%s\nwithout\n%s", got, without)
	}
}

// TestBoardOmitsTheStatusBlockWhereUnmanaged: a checkout abcd does not manage
// gets no block and no status field.
func TestBoardOmitsTheStatusBlockWhereUnmanaged(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	statusRecord(t, repo)
	t.Chdir(repo)
	if text := string(runCLI(t)); strings.Contains(text, "status:") {
		t.Errorf("an unmanaged checkout rendered the block:\n%s", text)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if raw, has := got["status"]; has {
		t.Errorf("--json carries status=%s in an unmanaged checkout", raw)
	}
}
