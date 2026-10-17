package guard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The host keeps the shell's working directory from one call to the next,
// while the approver judges every command as run from the working tree. An
// approved command therefore runs from the working tree: the approval hands
// back `cd -- '<dir>' && exec /bin/bash --noprofile --norc -c '<command>'`,
// so the place it was judged for is the place it runs, nothing in the
// command runs when the change of directory fails, and a change of directory
// inside the line still holds for the rest of that line.

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
	want := "cd -- '" + strings.ReplaceAll(dir, `'`, `'\''`) + "' && exec /bin/bash --noprofile --norc -c 'go vet ./...'"
	if got["command"] != want || got["description"] != "vet" || len(got) != 2 {
		t.Errorf("updated input = %s; want command %q and the description", updated, want)
	}
}

// TestApprovedCommandRunsOnlyFromTheWorkingTree: the whole command is the
// one word of the bash call the change of directory guards, so when the
// directory cannot be entered no part of it runs, a part after `||` or `;`
// included, and a command that ends in a here-document or carries a trailing
// comment runs as the same command. Each line is run by every shell the host
// may hand it to that is installed.
func TestApprovedCommandRunsOnlyFromTheWorkingTree(t *testing.T) {
	var shells []string
	for _, sh := range []string{"bash", "zsh", "sh"} {
		if p, err := exec.LookPath(sh); err == nil {
			shells = append(shells, p)
		}
	}
	if len(shells) == 0 {
		t.Skip("no shell installed")
	}
	for _, sh := range shells {
		t.Run(filepath.Base(sh), func(t *testing.T) {
			host, tree := t.TempDir(), t.TempDir()
			missing := filepath.Join(tree, "gone")
			run := func(dir, command string) (string, error) {
				cmd := exec.Command(sh, "-c", runFrom(dir, command))
				cmd.Dir = host
				out, err := cmd.CombinedOutput()
				return string(out), err
			}
			for _, command := range []string{
				`false || touch ran`,
				`true; touch ran`,
				"true\ntouch ran",
				`touch ran # a note`,
			} {
				if out, err := run(missing, command); err == nil {
					t.Errorf("%q from a missing directory exited 0: %s", command, out)
				}
				if _, err := os.Stat(filepath.Join(host, "ran")); err == nil {
					t.Fatalf("%q ran in the host's directory when the change of directory failed", command)
				}
			}
			for command, want := range map[string]string{
				"cat <<'EOF' > out\nfrom the tree\nEOF":         "from the tree\n",
				"cat <<EOF > out\n$(printf tree)\nEOF\n":        "tree\n",
				"printf '%s' kept > out # a trailing } comment": "kept",
				"printf a > out || printf b > out":              "a",
			} {
				if out, err := run(tree, command); err != nil {
					t.Errorf("%q: %v: %s", command, err, out)
					continue
				}
				got, err := os.ReadFile(filepath.Join(tree, "out"))
				if err != nil || string(got) != want {
					t.Errorf("%q wrote %q (%v), want %q", command, got, err, want)
				}
				if _, err := os.Stat(filepath.Join(host, "out")); err == nil {
					t.Errorf("%q wrote into the host's directory", command)
				}
				_ = os.Remove(filepath.Join(tree, "out"))
			}
		})
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
