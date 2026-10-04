package abcdhome

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestHomeStopCheck covers each row of the stop's table
// (spc-2610031309233367, "The stop"): an old ~/.abcd stops whatever it is (a
// folder, a file or a link), alone or beside ~/.abcd.noindex, and an absent one
// never stops; an empty or relative HOME is no stop, because the refusals for
// that shape stand where they are. Check reads the two names with Lstat and
// writes nothing, so the home's tree is the same after every call.
func TestHomeStopCheck(t *testing.T) {
	type shape func(t *testing.T, path string)
	folder := func(t *testing.T, path string) {
		t.Helper()
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "trusted-roots"), []byte("/x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	file := func(t *testing.T, path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A link at the old name stops even when it leads into the new folder:
	// the old path is still the old home, and Check never follows it (open
	// question 2).
	linkToNew := func(t *testing.T, path string) {
		t.Helper()
		if err := os.Symlink(name, path); err != nil {
			t.Fatal(err)
		}
	}
	dangling := func(t *testing.T, path string) {
		t.Helper()
		if err := os.Symlink(filepath.Join(filepath.Dir(path), "nowhere"), path); err != nil {
			t.Fatal(err)
		}
	}

	rows := []struct {
		name     string
		old, new shape
		stop     bool
		both     bool
	}{
		{"neither stands", nil, nil, false, false},
		{"only the new folder stands", nil, folder, false, false},
		{"the old folder stands", folder, nil, true, false},
		{"the old name is a file", file, nil, true, false},
		{"the old name is a dangling link", dangling, nil, true, false},
		{"the old name links to the new folder", linkToNew, folder, true, true},
		{"both folders stand", folder, folder, true, true},
		{"the old folder beside a new file", folder, file, true, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			home := t.TempDir()
			if r.new != nil {
				r.new(t, filepath.Join(home, name))
			}
			if r.old != nil {
				r.old(t, filepath.Join(home, oldName))
			}
			before := listTree(t, home)
			got := Check(home)
			if after := listTree(t, home); after != before {
				t.Fatalf("Check changed the home:\nbefore:\n%s\nafter:\n%s", before, after)
			}
			if !r.stop {
				if got != nil {
					t.Fatalf("Check = %+v, want no stop", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("Check = nil, want a stop")
			}
			if got.Both != r.both {
				t.Errorf("Both = %v, want %v", got.Both, r.both)
			}
			want, short := oldStandsLine, oldStandsShort
			if r.both {
				want, short = bothStandLine, bothStandShort
			}
			if got.Line != want || got.Short != short {
				t.Errorf("stop = %+v, want line %q and short %q", *got, want, short)
			}
		})
	}

	for _, home := range []string{"", "relative/home", "."} {
		t.Run("HOME "+strings.TrimSpace(home)+" is no stop", func(t *testing.T) {
			if got := Check(home); got != nil {
				t.Fatalf("Check(%q) = %+v, want no stop: the refusals for that HOME stand", home, *got)
			}
		})
	}
}

// TestStopLinesNameTheRenameAndTheRepair pins the two lines the stop prints
// (the spec's text, as amended by open question 6): the old folder's line names
// the folder, the new one and the one rename command; the both line names both
// folders and never the rename, which would move the old folder INTO the new
// one; and each prints the repair loop that reconnects the worktrees the
// rename moved (iss-2610040147016103).
func TestStopLinesNameTheRenameAndTheRepair(t *testing.T) {
	if RenameCommand != "mv ~/.abcd ~/.abcd.noindex" {
		t.Errorf("RenameCommand = %q", RenameCommand)
	}
	if RepairCommand != `for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done` {
		t.Errorf("RepairCommand = %q", RepairCommand)
	}
	for _, c := range []struct {
		line   string
		want   []string
		refuse []string
	}{
		{oldStandsLine, []string{"~/.abcd.noindex", "~/.abcd still stands", "`" + RenameCommand + "`", "`" + RepairCommand + "`", "has written nothing"}, nil},
		{bothStandLine, []string{"Both ~/.abcd and ~/.abcd.noindex exist", "`" + RepairCommand + "`", "has written nothing"}, []string{RenameCommand}},
	} {
		for _, w := range c.want {
			if !strings.Contains(c.line, w) {
				t.Errorf("line %q does not name %q", c.line, w)
			}
		}
		for _, r := range c.refuse {
			if strings.Contains(c.line, r) {
				t.Errorf("line %q names %q", c.line, r)
			}
		}
		if strings.Contains(c.line, "\n") {
			t.Errorf("line %q is not one line", c.line)
		}
	}
	if oldStandsShort != "abcd stopped: rename ~/.abcd to ~/.abcd.noindex" {
		t.Errorf("short form = %q", oldStandsShort)
	}
}

// listTree renders every entry under root with its size and mode, so a test
// can prove a call wrote nothing.
func listTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		b.WriteString(rel + " " + info.Mode().String() + " " + info.ModTime().String())
		if !info.IsDir() {
			b.WriteString(" " + strconv.FormatInt(info.Size(), 10))
		}
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
