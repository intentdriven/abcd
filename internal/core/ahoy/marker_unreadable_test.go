package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unreadableTargets is each way a conventions file can exist and still be one
// install can never read whole: a folder, a FIFO, and a file with no read
// permission.
var unreadableTargets = []struct {
	name string
	make func(t *testing.T, path string)
}{
	{"a directory", func(t *testing.T, path string) {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}},
	{"a FIFO", func(t *testing.T, path string) { mkfifoOrSkip(t, path) }},
	{"an unreadable file", func(t *testing.T, path string) {
		if err := os.WriteFile(path, []byte("# The owner's\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
		if f, err := os.Open(path); err == nil {
			f.Close()
			t.Skip("a mode-000 file is readable here (running as root)")
		}
	}},
}

// TestUnreadableMarkerTargetIsNotAResolvableGap (iss-2610032202256580): a
// conventions file that exists but cannot be read whole is never a resolvable
// marker.missing gap, which install would promise to close and never could,
// leaving the repository partial on every run. It is its own non-resolvable
// gap, as a symlinked file is.
func TestUnreadableMarkerTargetIsNotAResolvableGap(t *testing.T) {
	for _, tc := range unreadableTargets {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(repo, "AGENTS.md")
			tc.make(t, p)
			var st markerState
			withinDeadline(t, "classifyMarker", func() { st = classifyMarker(p) })
			if st != markerUnreadable {
				t.Errorf("classifyMarker = %q, want %q", st, markerUnreadable)
			}
			res, err := Install(repo, installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range res.Remaining {
				if strings.HasPrefix(id, "marker.") {
					t.Errorf("install leaves %s outstanding, a gap it can never close: %+v", id, res)
				}
			}
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			named := false
			for _, g := range det.Gaps {
				if !strings.HasPrefix(g.ID, "marker.") {
					continue
				}
				if g.Resolvable {
					t.Errorf("gap %s is resolvable, but install can never write the file", g.ID)
				}
				if g.ID == "marker.unreadable" && strings.Contains(g.Title, "AGENTS.md") {
					named = true
				}
			}
			if !named {
				t.Errorf("no marker.unreadable gap names AGENTS.md: %+v", det.Gaps)
			}
		})
	}
}

// pathFree fails when text carries dir, lexically or resolved.
func pathFree(t *testing.T, label, text, dir string) {
	t.Helper()
	forms := []string{dir}
	if r, err := filepath.EvalSymlinks(dir); err == nil && r != dir {
		forms = append(forms, r)
	}
	for _, f := range forms {
		if strings.Contains(text, f) {
			t.Errorf("%s carries the absolute path %s: %q", label, f, text)
		}
	}
}

// TestMarkerRefusalsCarryNoPath (iss-2610032202259011): a marker refusal names
// the file by its base name alone. The OS error it wraps names the absolute
// path the syscall was given, and a note printed or emitted as JSON would carry
// it, the account name included, wherever the file sits.
func TestMarkerRefusalsCarryNoPath(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(repo, "AGENTS.md")
	unreadableTargets[2].make(t, p)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	refused := false
	for _, n := range res.Notes {
		pathFree(t, "install note", n, repo)
		if strings.Contains(n, "AGENTS.md") {
			refused = true
		}
	}
	if !refused {
		t.Errorf("install gave no refusal naming AGENTS.md: %q", res.Notes)
	}
	for _, dry := range []bool{true, false} {
		_, err := EnsureMarker(p, dry)
		if err == nil {
			t.Fatalf("EnsureMarker(dryRun=%v) wrote an unreadable file", dry)
		}
		pathFree(t, "EnsureMarker", err.Error(), repo)
	}
	if _, err := removeMarkerFile(p); err == nil {
		t.Error("removeMarkerFile read an unreadable file")
	} else {
		pathFree(t, "removeMarkerFile", err.Error(), repo)
	}
}

// TestMarkerInAFolderThatCannotTakeItIsNotAResolvableGap: an absent
// conventions file, or one whose block is out of date, in a folder that does
// not let abcd create a file there is a gap install can never close, since
// the write creates its lock and its temporary file beside the target. It is
// raised non-resolvable, naming the file and not its folder's path, and
// install leaves no marker gap outstanding (iss-2610032303183254).
func TestMarkerInAFolderThatCannotTakeItIsNotAResolvableGap(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root creates files in every folder")
	}
	for _, tc := range []struct {
		name  string
		shape func(t *testing.T, p string)
	}{
		{"absent", func(t *testing.T, p string) {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		}},
		{"outdated", func(t *testing.T, p string) {
			if err := os.WriteFile(p, []byte("# Project\n\n<!-- BEGIN ABCD -->\nold\n<!-- END ABCD -->\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			tc.shape(t, filepath.Join(repo, "AGENTS.md"))
			if err := os.Chmod(repo, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(repo, 0o755) })

			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			named := false
			for _, g := range det.Gaps {
				if !strings.HasPrefix(g.ID, "marker.") {
					continue
				}
				if g.Resolvable {
					t.Errorf("gap %s is resolvable, but the folder cannot take the file: %+v", g.ID, g)
				}
				if g.ID == "marker.unwritable" && strings.Contains(g.Title, "AGENTS.md") {
					named = true
					pathFree(t, "the gap", g.Title+" "+g.Detail+" "+g.FixHint, repo)
				}
			}
			if !named {
				t.Errorf("no marker.unwritable gap names AGENTS.md: %+v", det.Gaps)
			}
			res, err := Install(repo, installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range res.Remaining {
				if strings.HasPrefix(id, "marker.") {
					t.Errorf("install leaves %s outstanding, a gap it can never close: %+v", id, res)
				}
			}
		})
	}
}
