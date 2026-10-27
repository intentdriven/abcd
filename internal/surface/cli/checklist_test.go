package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// checklist_test.go — the `abcd checklist` verbs end to end through the front
// door (spc-2610230805429429 step 2): each verb and its --json, a refused write
// exiting 2 without the value, and ids that stay put across add, set and the
// render.

// testSessionID is the host session id the tests hand the verbs.
const testSessionID = "6f1c2a9e-3b4d-4e5f-8a7b-0c1d2e3f4a5b"

// checklistCheckout is a managed checkout with the local tier, standing in it,
// with the host's session id pinned so updated_by is known.
func checklistCheckout(t *testing.T) string {
	t.Helper()
	root := managedCheckout(t)
	t.Setenv(hostSessionIDEnv, testSessionID)
	return root
}

// runChecklist runs the binary's front door and returns both streams and the
// exit code.
func runChecklist(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var so, se bytes.Buffer
	code = Run(append([]string{"checklist"}, args...), &so, &se)
	return so.String(), se.String(), code
}

// mustChecklist runs a verb that must succeed.
func mustChecklist(t *testing.T, args ...string) string {
	t.Helper()
	out, errOut, code := runChecklist(t, args...)
	if code != 0 {
		t.Fatalf("checklist %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
	}
	return out
}

func decodeJSON[T any](t *testing.T, raw string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	return v
}

// onDisk is the checklist the verbs wrote, read through the core's reader.
func onDisk(t *testing.T, root string) checklist.Checklist {
	t.Helper()
	res := checklist.Read(root)
	if res.Source != checklist.FromFile {
		t.Fatalf("the checklist on disk did not read clean: %s (%s)", res.Source, res.Reason)
	}
	return res.Checklist
}

func TestChecklistAddMintsAStableSlugAndWritesTheFile(t *testing.T) {
	root := checklistCheckout(t)

	got := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Fix the parser's error path", "--json"))
	if got.ID != "fix-the-parser-s-error-path" {
		t.Errorf("id = %q, want the title's slug", got.ID)
	}
	if got.Item.State != checklist.Working || got.Item.Title != "Fix the parser's error path" {
		t.Errorf("item = %+v, want a working item with the title as given", got.Item)
	}

	// The same title again is a second item under its own id.
	again := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Fix the parser's error path", "--json"))
	if again.ID != "fix-the-parser-s-error-path-2" {
		t.Errorf("second id = %q, want the slug with -2", again.ID)
	}

	// A wait and a promise: the item waits, and the promise splits at "=>".
	wait := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Wait for the review",
		"--waits-on", "pr:42", "--next-look", "2026-10-24T09:00:00Z",
		"--promise", "pr:42 merged => add the wrapper", "--json"))
	if wait.Item.State != checklist.Waiting || wait.Item.WaitsOn != "pr:42" || wait.Item.NextLook != "2026-10-24T09:00:00Z" {
		t.Errorf("waiting item = %+v", wait.Item)
	}
	if p := wait.Item.Promise; p == nil || p.Trigger != "pr:42 merged" || p.Act != "add the wrapper" {
		t.Errorf("promise = %+v, want trigger %q and act %q", p, "pr:42 merged", "add the wrapper")
	}

	c := onDisk(t, root)
	if len(c.Items) != 3 || c.Items[0].ID != got.ID || c.Items[1].ID != again.ID || c.Items[2].ID != wait.ID {
		t.Fatalf("items on disk = %+v", c.Items)
	}
	if c.UpdatedBy != testSessionID {
		t.Errorf("updated_by = %q, want the host's session id", c.UpdatedBy)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(checklist.LastGoodRelPath))); err != nil {
		t.Errorf("the write produced no last-good copy: %v", err)
	}

	// The text form names the id it minted.
	if out := mustChecklist(t, "add", "Write the command page"); !strings.Contains(out, "write-the-command-page") {
		t.Errorf("add printed %q, want the minted id", out)
	}
}

func TestChecklistSetChangesTheStateAndKeepsEveryID(t *testing.T) {
	root := checklistCheckout(t)
	first := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Build the verbs", "--json")).ID
	second := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Regenerate the reference", "--json")).ID

	set := decodeJSON[checklistItemOutput](t, mustChecklist(t, "set", first, "--state", "done", "--json"))
	if set.ID != first || set.Item.State != checklist.Done {
		t.Errorf("set = %+v, want %s done", set, first)
	}
	if out := mustChecklist(t, "set", second, "--state", "dropped"); !strings.Contains(out, second) || !strings.Contains(out, "dropped") {
		t.Errorf("set printed %q, want the id and the state", out)
	}

	c := onDisk(t, root)
	if len(c.Items) != 2 || c.Items[0].ID != first || c.Items[1].ID != second ||
		c.Items[0].State != checklist.Done || c.Items[1].State != checklist.Dropped {
		t.Fatalf("items on disk = %+v", c.Items)
	}

	// The list and one item's view carry the same ids.
	list := decodeJSON[checklistListOutput](t, mustChecklist(t, "--json"))
	if list.Checklist == nil || len(list.Checklist.Items) != 2 ||
		list.Checklist.Items[0].ID != first || list.Checklist.Items[1].ID != second {
		t.Fatalf("bare list = %+v", list)
	}
	one := decodeJSON[checklistItemOutput](t, mustChecklist(t, "show", second, "--json"))
	if one.ID != second || one.Item.Title != "Regenerate the reference" {
		t.Errorf("show = %+v", one)
	}
}

func TestChecklistBareListsEveryItemAndWritesNothing(t *testing.T) {
	root := checklistCheckout(t)

	// Nothing written yet: bare says so, and creates nothing.
	empty := decodeJSON[checklistListOutput](t, mustChecklist(t, "--json"))
	if empty.Source != checklist.FromNothing || empty.Checklist != nil || empty.Reason == "" {
		t.Errorf("bare on an empty tier = %+v", empty)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(checklist.FileRelPath))); !os.IsNotExist(err) {
		t.Fatalf("bare wrote the checklist: %v", err)
	}

	mustChecklist(t, "add", "Wait for the review", "--waits-on", "pr:42", "--promise", "pr:42 merged => add the wrapper")
	out := mustChecklist(t)
	for _, want := range []string{"wait-for-the-review", "waiting", "Wait for the review", "pr:42", "pr:42 merged => add the wrapper"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare list lacks %q:\n%s", want, out)
		}
	}
	list := decodeJSON[checklistListOutput](t, mustChecklist(t, "--json"))
	if list.Source != checklist.FromFile || list.Checklist == nil || list.Checklist.UpdatedBy != testSessionID {
		t.Errorf("bare --json = %+v", list)
	}
}

func TestChecklistRenderPrintsOpenItemsThenClosedOnes(t *testing.T) {
	checklistCheckout(t)
	done := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Land step one", "--json")).ID
	mustChecklist(t, "add", "Build step two")
	mustChecklist(t, "add", "Wait for the review", "--waits-on", "pr:42")
	mustChecklist(t, "set", done, "--state", "done")

	out := mustChecklist(t, "render")
	open, closed := strings.Index(out, "Build step two"), strings.Index(out, "Land step one")
	waiting := strings.Index(out, "Wait for the review")
	if open < 0 || closed < 0 || waiting < 0 {
		t.Fatalf("render lacks an item:\n%s", out)
	}
	if open > closed || waiting > closed {
		t.Errorf("render put a closed item before an open one:\n%s", out)
	}
	if !strings.Contains(out, "pr:42") {
		t.Errorf("render lacks the wait:\n%s", out)
	}

	r := decodeJSON[checklistRenderOutput](t, mustChecklist(t, "render", "--json"))
	if r.Source != checklist.FromFile || r.Text != out {
		t.Errorf("render --json = %+v, want the text form's text", r)
	}
}

// secretToken is a GitHub-PAT-shaped string built at run time, so no
// secret-shaped literal is committed.
func secretToken() string { return "ghp_" + testsecret.Synthetic(2610230806, 36) }

// assertNoSecretEcho fails when text carries the token's prefix or any run of
// four of its characters: a redaction fingerprint is a part of the secret too.
func assertNoSecretEcho(t *testing.T, what, text, tok string) {
	t.Helper()
	if strings.Contains(text, tok[:4]) {
		t.Errorf("%s repeats the secret's prefix: %s", what, text)
	}
	for i := 4; i+4 <= len(tok); i++ {
		if strings.Contains(text, tok[i:i+4]) {
			t.Errorf("%s repeats part of the secret: %s", what, text)
			return
		}
	}
}

func TestChecklistRefusesASecretWithoutEchoingIt(t *testing.T) {
	root := checklistCheckout(t)
	mustChecklist(t, "add", "Build the verbs")
	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checklist.FileRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	tok := secretToken()

	for _, tc := range []struct {
		name  string
		args  []string
		field string
	}{
		{"title", []string{"add", "Rotate " + tok + " today"}, ".title"},
		{"promise act", []string{"add", "Ship it", "--promise", "pr:7 merged => paste " + tok}, ".promise.act"},
	} {
		for _, asJSON := range []bool{false, true} {
			args := tc.args
			if asJSON {
				args = append(append([]string{}, args...), "--json")
			}
			out, errOut, code := runChecklist(t, args...)
			if code != 2 {
				t.Errorf("%s (json=%v): exit %d, want 2\n%s%s", tc.name, asJSON, code, out, errOut)
			}
			assertNoSecretEcho(t, tc.name, out+errOut, tok)
			if !strings.Contains(out+errOut, tc.field) || !strings.Contains(out+errOut, "github") {
				t.Errorf("%s (json=%v): the refusal does not name the field and the pattern kind:\n%s%s", tc.name, asJSON, out, errOut)
			}
		}
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checklist.FileRelPath)))
	if err != nil || !bytes.Equal(before, after) {
		t.Errorf("a refused write changed the file (%v)", err)
	}
}

func TestChecklistRefusesASchemaViolationWithExit2(t *testing.T) {
	root := checklistCheckout(t)
	id := decodeJSON[checklistItemOutput](t, mustChecklist(t, "add", "Build the verbs", "--json")).ID
	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checklist.FileRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	// marker is a value no refusal may repeat.
	const marker = "zq-unrepeated-value"
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"long title", []string{"add", strings.Repeat("x", checklist.MaxTitleRunes+1)}, "title"},
		{"bad waits-on", []string{"add", "Wait", "--waits-on", marker}, "waits_on"},
		{"bad next-look", []string{"add", "Wait", "--next-look", marker}, "next_look"},
		{"promise without =>", []string{"add", "Ship it", "--promise", marker}, "--promise"},
		{"unknown state", []string{"set", id, "--state", marker}, "state"},
		{"set without --state", []string{"set", id}, "--state"},
		{"unknown id", []string{"set", marker, "--state", "done"}, "id"},
		{"show unknown id", []string{"show", marker}, "id"},
	} {
		for _, asJSON := range []bool{false, true} {
			args := tc.args
			if asJSON {
				args = append(append([]string{}, args...), "--json")
			}
			out, errOut, code := runChecklist(t, args...)
			if code != 2 {
				t.Errorf("%s (json=%v): exit %d, want 2\n%s%s", tc.name, asJSON, code, out, errOut)
			}
			if strings.Contains(out+errOut, marker) {
				t.Errorf("%s (json=%v): the refusal repeats the value:\n%s%s", tc.name, asJSON, out, errOut)
			}
			if !strings.Contains(out+errOut, tc.want) {
				t.Errorf("%s (json=%v): the refusal does not name %q:\n%s%s", tc.name, asJSON, tc.want, out, errOut)
			}
			if asJSON {
				env := decodeJSON[errorEnvelope](t, out)
				if env.Abcd != "error" || env.ExitCode != 2 {
					t.Errorf("%s: --json refusal = %+v", tc.name, env)
				}
			}
		}
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checklist.FileRelPath)))
	if err != nil || !bytes.Equal(before, after) {
		t.Errorf("a refused write changed the file (%v)", err)
	}
}

func TestChecklistWriteRefusesWithoutTheLocalTier(t *testing.T) {
	root := checklistCheckout(t)
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(checklist.TierRelPath))); err != nil {
		t.Fatal(err)
	}
	if out, errOut, code := runChecklist(t, "add", "Build the verbs"); code != 2 || !strings.Contains(out+errOut, checklist.TierRelPath) {
		t.Errorf("add with no local tier: exit %d, want 2 naming the tier\n%s%s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(checklist.TierRelPath))); !os.IsNotExist(err) {
		t.Errorf("the refused add created the tier: %v", err)
	}
}

func TestChecklistUpdatedByFallsBackWithoutAHostSessionID(t *testing.T) {
	root := checklistCheckout(t)
	t.Setenv(hostSessionIDEnv, "")
	mustChecklist(t, "add", "Build the verbs")
	if c := onDisk(t, root); c.UpdatedBy == "" || c.UpdatedBy == testSessionID {
		t.Errorf("updated_by = %q, want the fallback writer's name", c.UpdatedBy)
	}
	// A value that is not a session id's shape is not taken as one.
	t.Setenv(hostSessionIDEnv, "not a session id")
	mustChecklist(t, "add", "Regenerate the reference")
	if c := onDisk(t, root); c.UpdatedBy == "not a session id" {
		t.Errorf("updated_by took a malformed session id")
	}
}

// TestChecklistAddRefusesATrackedChecklist: once git tracks the checklist, a
// change would rewrite the committed file and the next read would refuse it,
// losing the item, so add refuses with exit 2, names how to untrack it, and
// leaves the file as it was.
func TestChecklistAddRefusesATrackedChecklist(t *testing.T) {
	root := checklistCheckout(t)
	mustChecklist(t, "add", "Build the verbs")
	p := filepath.Join(root, filepath.FromSlash(checklist.FileRelPath))
	gitCmd(t, root, "add", "-f", checklist.FileRelPath)
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if out, errOut, code := runChecklist(t, "add", "Lose this"); code != 2 || !strings.Contains(out+errOut, "git rm --cached") {
		t.Errorf("add over a tracked checklist: exit %d, want 2 naming git rm --cached\n%s%s", code, out, errOut)
	}
	if after, err := os.ReadFile(p); err != nil || !bytes.Equal(after, before) {
		t.Errorf("the refused add changed the tracked file: %v", err)
	}
}
