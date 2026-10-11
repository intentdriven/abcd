package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/memnote"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// memoryFixture is a repository with a root commit as the working directory
// and a fresh HOME.
func memoryFixture(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	repo, _ = filepath.EvalSymlinks(t.TempDir())
	gitInitAt(t, repo)
	gitCommitAt(t, repo, "root")
	t.Chdir(repo)
	return home, repo
}

// memoryAdd runs `memory add` with body on stdin and the usual flags, plus
// extra, and returns the output and the error.
func memoryAdd(t *testing.T, body string, extra ...string) (string, error) {
	t.Helper()
	args := append([]string{"memory", "add", "--title", "The tests need the race flag",
		"--recall", "test", "--recall", "race", "--by", "Example:model-1"}, extra...)
	out, err := runCLIStdinErr(t, body, args...)
	return string(out), err
}

const raceBody = "Run the race-enabled tests before saying they pass.\n"

func readMachineNotes(t *testing.T, home string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(abcdhome.Path(home, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Notes []map[string]any `json:"notes"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f.Notes
}

func TestMemoryAddWritesTheMachineLayer(t *testing.T) {
	home, repo := memoryFixture(t)
	out, err := memoryAdd(t, raceBody, "--session", "s-0001", "--json")
	if err != nil {
		t.Fatalf("memory add: %v\n%s", err, out)
	}
	var env struct {
		ID      string   `json:"id"`
		Notices []string `json:"notices"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("envelope: %v\n%s", err, out)
	}
	if !memnote.ValidID(env.ID) {
		t.Errorf("id = %q", env.ID)
	}
	notes := readMachineNotes(t, home)
	if len(notes) != 1 || notes[0]["id"] != env.ID || notes[0]["body"] != strings.TrimSpace(raceBody) || notes[0]["session"] != "s-0001" {
		t.Fatalf("machine layer = %v", notes)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "memory.json")); !os.IsNotExist(err) {
		t.Errorf("add touched the project's file (stat err %v)", err)
	}
}

// N3: add takes no path operand, so it can never be pointed at the host's
// memory files; the text arrives on stdin only.
func TestMemoryAddTakesNoPathOperand(t *testing.T) {
	home, _ := memoryFixture(t)
	hostFile := filepath.Join(home, "host-memory.md")
	if err := os.WriteFile(hostFile, []byte("a lesson\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := memoryAdd(t, raceBody, hostFile)
	if err == nil {
		t.Fatalf("memory add accepted a positional operand:\n%s", out)
	}
	if _, serr := os.Stat(abcdhome.Path(home, "memory.json")); !os.IsNotExist(serr) {
		t.Error("a refused add wrote the machine layer")
	}
	add, _, ferr := NewRootCommand().Find([]string{"memory", "add"})
	if ferr != nil || add.Name() != "add" {
		t.Fatalf("no memory add verb: %v", ferr)
	}
	for _, f := range []string{"file", "path", "from"} {
		if add.Flags().Lookup(f) != nil {
			t.Errorf("memory add registers a --%s flag", f)
		}
	}
}

func TestMovedFromHostStampsTheOrigin(t *testing.T) {
	home, _ := memoryFixture(t)
	if out, err := memoryAdd(t, raceBody, "--moved-from-host"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := memoryAdd(t, raceBody); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	notes := readMachineNotes(t, home)
	if notes[0]["origin"] != memnote.OriginHostMemoryMove || notes[1]["origin"] != memnote.OriginAgent {
		t.Errorf("origins = %v, %v", notes[0]["origin"], notes[1]["origin"])
	}
}

// N7: a write with no session id says, on that write, that the session is
// recorded as unknown.
func TestMemoryAddWithoutSessionSaysUnknown(t *testing.T) {
	home, _ := memoryFixture(t)
	out, err := memoryAdd(t, raceBody)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "session unknown") {
		t.Errorf("the write does not say the session is unknown:\n%s", out)
	}
	if v, ok := readMachineNotes(t, home)[0]["session"]; !ok || v != nil {
		t.Errorf("session = %v, want null", v)
	}
	out, err = memoryAdd(t, raceBody, "--session", "s-0002")
	if err != nil || strings.Contains(out, "session unknown") {
		t.Errorf("with a session the write still says unknown (%v):\n%s", err, out)
	}
}

func TestMemoryAddEverywhereSaysOnlyThisCheckoutsNamesWereChecked(t *testing.T) {
	home, _ := memoryFixture(t)
	out, err := memoryAdd(t, raceBody, "--everywhere", "--session", "s-1")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "only this checkout's private names") {
		t.Errorf("--everywhere does not say which names were checked:\n%s", out)
	}
	if v, ok := readMachineNotes(t, home)[0]["root"]; !ok || v != nil {
		t.Errorf("root = %v, want null", v)
	}
	out, _ = memoryAdd(t, raceBody, "--session", "s-1")
	if strings.Contains(out, "private names") {
		t.Errorf("a tied note says the everywhere notice:\n%s", out)
	}
}

func TestMemoryListShowsEachNotesBodyStampsAndReach(t *testing.T) {
	_, repo := memoryFixture(t)
	if out, err := memoryAdd(t, raceBody, "--session", "s-0001"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := memoryAdd(t, "A lesson for every project.\n", "--everywhere"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	project := `{"schema_version":1,"notes":[{"id":"mem-2610010000000001","title":"A promoted lesson","body":"Promoted by a person.","recall":["promoted"],"origin":"agent","written_by":"Example:model-1","written_at":"2026-10-01T00:00:00Z","session":null}]}`
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "memory.json"), []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}
	out := string(runCLI(t, "memory", "list"))
	for _, want := range []string{
		"The tests need the race flag", strings.TrimSpace(raceBody), "test, race",
		"Example:model-1", "2026-", "session s-0001", "this repository",
		"A lesson for every project.", "everywhere", "session unknown",
		"A promoted lesson", "Promoted by a person.", "project's file",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("list lacks %q:\n%s", want, out)
		}
	}
	var env struct {
		Notes []struct {
			Layer string `json:"layer"`
			Reach string `json:"reach"`
			Body  string `json:"body"`
		} `json:"notes"`
	}
	if err := json.Unmarshal(runCLI(t, "memory", "list", "--json"), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Notes) != 3 {
		t.Fatalf("--json lists %d notes, want 3", len(env.Notes))
	}
}

func TestMemoryRemoveRemovesAnyPersonalNote(t *testing.T) {
	home, _ := memoryFixture(t)
	out, err := memoryAdd(t, raceBody, "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	var env struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(out), &env)
	// A note another writer left, by hand, with another origin.
	if out, err := memoryAdd(t, "Another lesson.\n", "--moved-from-host", "--by", "Other:model-2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	other := readMachineNotes(t, home)[1]["id"].(string)
	for _, id := range []string{env.ID, other} {
		if out, err := runCLIErr(t, "memory", "remove", id); err != nil {
			t.Fatalf("remove %s: %v\n%s", id, err, out)
		}
	}
	if n := readMachineNotes(t, home); len(n) != 0 {
		t.Fatalf("notes left after remove: %v", n)
	}
	if out := string(runCLI(t, "memory", "list")); strings.Contains(out, "race flag") {
		t.Errorf("a removed note is still listed:\n%s", out)
	}
	if _, err := runCLIErr(t, "memory", "remove", env.ID); exitCode(err) != 2 {
		t.Errorf("removing an absent note: exit %d, want 2", exitCode(err))
	}
}

func TestMemoryRemoveRefusesANoteInTheProjectsFile(t *testing.T) {
	_, repo := memoryFixture(t)
	project := `{"schema_version":1,"notes":[{"id":"mem-2610010000000001","title":"A promoted lesson","body":"Promoted by a person.","recall":["promoted"],"origin":"agent","written_by":"Example:model-1","written_at":"2026-10-01T00:00:00Z","session":null}]}`
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(repo, ".abcd", "memory.json")
	if err := os.WriteFile(p, []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCLIErr(t, "memory", "remove", "mem-2610010000000001")
	if exitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2:\n%s", exitCode(err), out)
	}
	if !strings.Contains(err.Error(), ".abcd/memory.json") || !strings.Contains(err.Error(), "reviewed change") {
		t.Errorf("the refusal does not say where the note is and how it goes: %v", err)
	}
	if data, _ := os.ReadFile(p); string(data) != project {
		t.Error("remove changed the project's file")
	}
}

func TestBareMemoryNamesAnUnreadableLayer(t *testing.T) {
	home, repo := memoryFixture(t)
	if out, err := memoryAdd(t, raceBody); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := memoryAdd(t, raceBody, "--everywhere"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	out := string(runCLI(t, "memory"))
	for _, want := range []string{"1 personal note(s) for this repository", "1 recalled everywhere", "0 in the project's file"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare lacks %q:\n%s", want, out)
		}
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "memory.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcdhome.Path(home, "memory.json"), 0o666); err != nil {
		t.Fatal(err)
	}
	out = string(runCLI(t, "memory"))
	for _, want := range []string{".abcd/memory.json", "malformed", "~/.abcd.noindex/memory.json", "writable by others"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare does not name the unreadable layer (%q):\n%s", want, out)
		}
	}
	if strings.Contains(out, home) {
		t.Errorf("bare carries the expanded home:\n%s", out)
	}
}

// The flags the Use line names are refused by the core when missing, with
// exit 2 and nothing written.
func TestMemoryAddRefusesAMissingTitleRecallOrWriter(t *testing.T) {
	home, _ := memoryFixture(t)
	for name, args := range map[string][]string{
		"title":  {"memory", "add", "--recall", "test", "--by", "Example:model-1"},
		"recall": {"memory", "add", "--title", "T", "--by", "Example:model-1"},
		"by":     {"memory", "add", "--title", "T", "--recall", "test"},
	} {
		out, err := runCLIStdinErr(t, raceBody, args...)
		if exitCode(err) != 2 {
			t.Errorf("without --%s: exit %d, want 2:\n%s", name, exitCode(err), out)
		}
	}
	if _, err := os.Stat(abcdhome.Path(home, "memory.json")); !os.IsNotExist(err) {
		t.Error("a refused add wrote the machine layer")
	}
}

// A root the resolver refuses is named on stderr before the verb's own
// refusal: rules.Resolution's notes are a front door's to print, and a memory
// verb that dropped them left a no-root-commit refusal with no word of why the
// repository was not used.
func TestMemoryAddPrintsTheOwnershipRefusalBeforeItsOwn(t *testing.T) {
	home, repo := memoryFixture(t)
	sub := mustMkdirAll(t, filepath.Join(repo, "pkg"))
	t.Chdir(sub)
	gitRefusesOwnership(t, sub)
	t.Cleanup(rules.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil }))

	var so, se bytes.Buffer
	code := run([]string{"memory", "add", "--title", "The tests need the race flag",
		"--recall", "test", "--recall", "race", "--by", "Example:model-1"}, strings.NewReader(raceBody), &so, &se)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstdout:\n%s\nstderr:\n%s", code, so.String(), se.String())
	}
	stderr := se.String()
	refused, noRoot := strings.Index(stderr, "REFUSED"), strings.Index(stderr, "no root commit")
	if refused < 0 || noRoot < 0 || refused > noRoot {
		t.Errorf("stderr must name the refused root before the no-root-commit refusal:\n%s", stderr)
	}
	if _, err := os.Stat(abcdhome.Path(home, "memory.json")); !os.IsNotExist(err) {
		t.Error("a refused add wrote the machine layer")
	}
}
