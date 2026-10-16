package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The host keeps the shell's working directory from one call to the next,
// while the approver judges every command as run from the working tree. An
// approved command therefore runs from the working tree: the approval hands
// back `cd -- '<dir>' && <command>`, so the place it was judged for is the
// place it runs, and a change of directory inside the line still holds for
// the rest of that line.

// TestApprovedCommandRunsFromTheWorkingTree: an allow returns the command
// prefixed with a change into the working tree, the directory quoted as one
// literal word whatever it holds.
func TestApprovedCommandRunsFromTheWorkingTree(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "it's a tree")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	a, updated := ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(`{"command":"go vet ./...","description":"vet"}`))
	if !a.Allow {
		t.Fatalf("a clean command was refused: %+v", a)
	}
	var got map[string]string
	if err := json.Unmarshal(updated, &got); err != nil {
		t.Fatal(err)
	}
	want := `cd -- '` + strings.ReplaceAll(dir, `'`, `'\''`) + `' && go vet ./...`
	if got["command"] != want || got["description"] != "vet" || len(got) != 2 {
		t.Errorf("updated input = %s; want command %q and the description", updated, want)
	}
}

// TestApproveResolvesRelativeNamesAfterAFixedCD: after a change of directory
// the line spells out, a relative name is the file in that directory, so an
// ordinary removal there is approved, and one reaching a protected or
// critical path from there is still refused.
func TestApproveResolvesRelativeNamesAfterAFixedCD(t *testing.T) {
	r := plainRig(t)
	if err := os.MkdirAll(filepath.Join(r.dir, "sub", "build"), 0o700); err != nil {
		t.Fatal(err)
	}
	r.run(t, []approveCase{
		{`cd sub && rm x.o`, nil},
		{`cd sub && rm -f x.o y.o`, nil},
		{`cd sub && rm -r build`, nil},
		{`cd ` + filepath.Join(r.dir, "sub") + ` && rm x.o`, nil},
		{`cd sub && rm -f ../.mcp.json`, protectedOrRegistry},
		{`cd sub && rm -r ..`, refusedAs(criticalRemovalEntryID)},
		{`cd / && rm -rf usr`, refusedAs(criticalRemovalEntryID)},
	})
}

// TestApproveNamesAnUnknownChangeOfDirectory: a relative removal after a
// change of directory the line does not spell out is refused as a name the
// line does not place, with that reason, not as a removal of the root or
// home.
func TestApproveNamesAnUnknownChangeOfDirectory(t *testing.T) {
	r := plainRig(t)
	for _, c := range []string{`cd "$D" && rm x.o`, `cd $(mktemp -d) && rm -f x.o`, `pushd "$D" && rm x.o`, `cd sub; rm x.o`} {
		a := approveShell(clean(), r.files(), c)
		if a.Allow {
			t.Errorf("%q was approved", c)
			continue
		}
		if a.EntryID != unfixedPathEntryID {
			t.Errorf("%q: refused under %q, want %q (%s)", c, a.EntryID, unfixedPathEntryID, a.Reason)
		}
		if !strings.Contains(a.Reason, "change of directory") || strings.Contains(a.Reason, "home directory") {
			t.Errorf("%q: the refusal does not name the change of directory: %s", c, a.Reason)
		}
	}
}
