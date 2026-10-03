package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// specHostReachWhy is the fixed sentence every presence warning carries, word for
// word from spc-2610031156364295: why a check above the project is not a read
// of settings from above it.
const specHostReachWhy = "abcd reads no settings from folders above this project. " +
	"This check only asks whether a file of this name exists there; it reads nothing in it " +
	"and changes nothing abcd does, because the agent tool itself reads that folder."

// versionShaped is any release number a warning might carry.
var versionShaped = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

func placeFile(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Someone's own\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if mode != 0o644 {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	}
}

// TestHostReachWarningsArePresenceOnly is A5 (itd-2610030814013772): a
// CLAUDE.md in a folder above the project, a personal CLAUDE.local.md at its
// root, and the files of those names in the folders between each raise one
// warning naming the file by its path, found by its presence alone, so a file
// no one can read is named all the same. The user-level .claude/CLAUDE.md in
// the home folder is not named, since it does not switch AGENTS.md off; a
// CLAUDE.md directly in the home folder is. Install is not refused, each
// warning carries the fixed sentence on abcd's reading nothing above the
// project, and no warning names a version.
func TestHostReachWarningsArePresenceOnly(t *testing.T) {
	home, _ := setupHermetic(t)
	work := filepath.Join(home, "work")
	repo := filepath.Join(work, "proj")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	placeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), 0o644) // user-level: never named
	placeFile(t, filepath.Join(home, "CLAUDE.md"), 0o644)
	placeFile(t, filepath.Join(work, "CLAUDE.md"), 0o000)
	placeFile(t, filepath.Join(work, ".claude", "CLAUDE.md"), 0o644)
	placeFile(t, filepath.Join(work, "CLAUDE.local.md"), 0o644)
	placeFile(t, filepath.Join(repo, "CLAUDE.local.md"), 0o644)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "refused" || res.Status == "aborted" {
		t.Fatalf("install was %s over presence warnings: %+v", res.Status, res.Notes)
	}
	want := map[string]bool{
		"~/CLAUDE.md":                 false,
		"~/work/CLAUDE.md":            false,
		"~/work/.claude/CLAUDE.md":    false,
		"~/work/CLAUDE.local.md":      false,
		"~/work/proj/CLAUDE.local.md": false,
	}
	inHome := 0
	for _, w := range res.Warnings {
		if strings.Contains(w, "~/.claude/CLAUDE.md") {
			t.Errorf("the user-level file is named: %q", w)
		}
		if !strings.Contains(w, specHostReachWhy) {
			t.Errorf("a warning lacks the fixed sentence:\n%s", w)
		}
		if versionShaped.MatchString(w) {
			t.Errorf("a warning names a version: %q", w)
		}
		if strings.Contains(w, "\n") {
			t.Errorf("a warning is not one line: %q", w)
		}
		if !strings.Contains(w, "~/") {
			continue // a folder above the test's temporary tree
		}
		inHome++
		named := ""
		for p := range want {
			if strings.HasPrefix(w, p+" ") || strings.HasPrefix(w, p+":") || strings.HasPrefix(w, p+",") {
				named = p
			}
		}
		if named == "" {
			t.Errorf("a warning names no expected file: %q", w)
			continue
		}
		want[named] = true
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("no warning names %s: %q", p, res.Warnings)
		}
	}
	if inHome != len(want) {
		t.Errorf("%d warnings name a file in the home folder, want one per file (%d): %q", inHome, len(want), res.Warnings)
	}

	// Install checks only: detection, which the board and the hooks call,
	// raises none of them.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range det.Gaps {
		if strings.Contains(g.Detail, specHostReachWhy) {
			t.Errorf("detection raised a host-reach gap: %+v", g)
		}
	}
}

// TestHostReachWalkEndsAtAnUnsearchableFolder: a folder on the walk that
// cannot be searched ends it quietly, naming nothing beyond it and refusing
// nothing. Every folder above a project abcd can reach is searchable, so the
// folder met is a .claude one, whose CLAUDE.md cannot be looked for.
func TestHostReachWalkEndsAtAnUnsearchableFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches every folder")
	}
	home, _ := setupHermetic(t)
	work := filepath.Join(home, "work")
	repo := filepath.Join(work, "proj")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	placeFile(t, filepath.Join(home, "CLAUDE.md"), 0o644)
	shut := filepath.Join(work, ".claude")
	if err := os.Mkdir(shut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0o600); err != nil { // read and write, no search
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shut, 0o755) })

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "refused" || res.Status == "aborted" {
		t.Fatalf("install was %s at an unsearchable folder: %+v", res.Status, res.Notes)
	}
	for _, w := range res.Warnings {
		if strings.Contains(w, "~/CLAUDE.md") {
			t.Errorf("the walk went on past a folder it could not search: %q", w)
		}
	}
}

// fakeClaude puts a command named claude on PATH, alone in its folder, running
// script; it returns the folder.
func fakeClaude(t *testing.T, dir, script string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestHostVersionWarning: a claude command on PATH older than the release
// that reads AGENTS.md on its own raises one warning, naming no version; one
// at that release, none on PATH, one printing nothing parsable, one that does
// not answer in time, and one inside the project (repository content, never
// run) raise nothing.
func TestHostVersionWarning(t *testing.T) {
	real := func(project string) (hostVersion, error) { return readHostVersion(claudeCommand, project) }
	for _, tc := range []struct {
		name   string
		script string // "" for no claude on PATH
		inside bool   // the command sits inside the project
		warn   bool
	}{
		{"below the floor", `echo "2.1.280 (Claude Code)"`, false, true},
		{"well below the floor, two digits", `echo "1.10.9 (Claude Code)"`, false, true},
		{"at the floor", `echo "2.1.281 (Claude Code)"`, false, false},
		{"above the floor", `echo "2.10.0 (Claude Code)"`, false, false},
		{"absent", "", false, false},
		{"nothing parsable", `echo "unknown"`, false, false},
		{"an error exit", "echo \"2.1.200\"\nexit 3", false, false},
		{"no answer in time", "sleep 5\necho \"2.1.200\"", false, false},
		{"inside the project", `echo "2.1.200 (Claude Code)"`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := setupHermetic(t)
			readClaudeVersion = real
			t.Cleanup(func() { readClaudeVersion = noHostVersion })
			if tc.name == "no answer in time" {
				hostVersionTimeout = 300 * time.Millisecond
				t.Cleanup(func() { hostVersionTimeout = 3 * time.Second })
			}
			repo := filepath.Join(home, "proj")
			if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(home, "fakebin")
			if tc.inside {
				bin = filepath.Join(repo, "tools")
			}
			if tc.script != "" {
				fakeClaude(t, bin, tc.script)
			}
			// Only the fake: the system folders the code under test runs
			// beside it, and nothing that resolves a claude of the machine's.
			t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")

			start := time.Now()
			res, err := Install(repo, installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); tc.name == "no answer in time" && elapsed > 4*time.Second {
				t.Errorf("install waited %s on a command that never answers", elapsed)
			}
			var hits []string
			for _, w := range res.Warnings {
				if strings.Contains(w, "older than the release that reads AGENTS.md") {
					hits = append(hits, w)
				}
			}
			switch {
			case tc.warn && len(hits) != 1:
				t.Fatalf("warnings = %q, want exactly one host-version warning", res.Warnings)
			case !tc.warn && len(hits) != 0:
				t.Fatalf("warnings = %q, want no host-version warning", res.Warnings)
			}
			for _, w := range hits {
				if versionShaped.MatchString(w) {
					t.Errorf("the warning names a version: %q", w)
				}
			}
		})
	}
}
