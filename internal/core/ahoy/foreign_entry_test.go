package ahoy

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// TestForeignRegularFileGapDescribesTheFile is iss-2609120447482255: a regular
// file occupying the PATH entry was reported only as "resolve manually", so the
// user had to inspect it by hand before deciding. The gap now says what the
// file is: its size, when it was last modified, and whether it identifies as an
// abcd build.
func TestForeignRegularFileGapDescribesTheFile(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	target := os.Getenv("ABCD_BIN_TARGET")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("#!/bin/sh\necho someone else\n")
	if err := os.WriteFile(target, body, 0o755); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 9, 12, 4, 47, 0, 0, time.UTC)
	if err := os.Chtimes(target, when, when); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	var g *Gap
	for i := range det.Gaps {
		if det.Gaps[i].ID == "symlink.foreign" {
			g = &det.Gaps[i]
		}
	}
	if g == nil {
		t.Fatalf("no symlink.foreign gap for a regular file at the entry: %+v", det.Gaps)
	}
	for _, want := range []string{strconv.Itoa(len(body)) + " bytes", "2026-09-12 04:47 UTC", "not an abcd build"} {
		if !strings.Contains(g.Detail, want) {
			t.Errorf("the foreign gap's detail does not say %q: %q", want, g.Detail)
		}
	}
	if strings.Contains(g.Detail, target) {
		t.Errorf("the detail carries an unredacted absolute path: %q", g.Detail)
	}
}

// TestDescribeBuildNamesAnAbcdBuildAndItsVersion pins the identification the
// detail rests on, over the build metadata a Go binary carries: abcd's own
// main package with its version (and revision when stamped), another Go
// program by its package path, and a build that states no version.
func TestDescribeBuildNamesAnAbcdBuildAndItsVersion(t *testing.T) {
	rev := "4ae6f2210a327e549431170bd9b7eef774429c03"
	cases := []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{"release", &debug.BuildInfo{Path: abcdMainPackage, Main: debug.Module{Version: "v0.9.0"},
			Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: rev}}}, "an abcd build, version v0.9.0 (revision 4ae6f2210a32)"},
		{"devel", &debug.BuildInfo{Path: abcdMainPackage, Main: debug.Module{Version: "(devel)"}}, "an abcd build of no stated version"},
		{"other", &debug.BuildInfo{Path: "example.com/tool/cmd/tool"}, "not an abcd build: a Go program built from example.com/tool/cmd/tool"},
	}
	for _, tc := range cases {
		if got := describeBuild(tc.bi); got != tc.want {
			t.Errorf("%s: describeBuild = %q, want %q", tc.name, got, tc.want)
		}
	}
	// End to end over a real Go binary: this test's own, which is not abcd.
	exe, err := os.Executable()
	if err != nil {
		t.Skip("no executable path: ", err)
	}
	fi, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if got := describeForeignFile(exe, fi); !strings.Contains(got, "not an abcd build: a Go program built from github.com/intentdriven/abcd/internal/core/ahoy") {
		t.Errorf("a real Go binary is not identified by its package path: %q", got)
	}
}

// TestDescribeEntrySaysWhatAForeignFileIs is the sweep of the same pattern:
// the install refusal and the shadowed-entry message describe a foreign PATH
// entry through describeEntry, which said only "a file abcd does not own".
func TestDescribeEntrySaysWhatAForeignFileIs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "abcd")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := describeEntry(pathEntry{path: path, kind: binTargetForeign})
	for _, want := range []string{"a file abcd does not own", "10 bytes", "not an abcd build"} {
		if !strings.Contains(got, want) {
			t.Errorf("describeEntry does not say %q: %q", want, got)
		}
	}
}

// TestDescribeForeignFileDoesNotBlockOnAFIFOSwappedIn is the lstat-to-open
// window: the caller judged the entry a regular file, and by the time the build
// metadata is read a FIFO stands at the path. A plain open of a FIFO with no
// writer blocks forever, and detection with it. The read opens without
// following and without blocking, re-checks the descriptor, and falls back to
// the size and time it was handed — it cannot say what it never read.
func TestDescribeForeignFileDoesNotBlockOnAFIFOSwappedIn(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "was-regular")
	if err := os.WriteFile(regular, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(regular)
	if err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "abcd")
	mkfifoOrSkip(t, fifo)
	var got string
	withinDeadline(t, "describeForeignFile", func() { got = describeForeignFile(fifo, fi) })
	if !strings.Contains(got, "10 bytes") {
		t.Errorf("the fallback lost the size it was handed: %q", got)
	}
	if strings.Contains(got, "abcd build") {
		t.Errorf("a file it never read is described as a build or not: %q", got)
	}
}

// TestDescribeBuildCapsWhatItPrints: every value describeBuild prints comes from
// the file, so a hostile binary could stamp a 5000-rune version into the gap's
// detail and the JSON. Each value is cut to a bounded length, marked.
func TestDescribeBuildCapsWhatItPrints(t *testing.T) {
	long := strings.Repeat("é", 5000)
	for name, bi := range map[string]*debug.BuildInfo{
		"version":  {Path: abcdMainPackage, Main: debug.Module{Version: "v" + long}},
		"path":     {Path: "example.com/" + long},
		"revision": {Path: abcdMainPackage, Main: debug.Module{Version: "v1.0.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "a" + long}, {Key: "vcs.revision", Value: "a" + long}}},
	} {
		got := describeBuild(bi)
		if n := utf8.RuneCountInString(got); n > 160 {
			t.Errorf("%s: describeBuild printed %d runes; the value is not capped", name, n)
		}
		if !utf8.ValidString(got) {
			t.Errorf("%s: the cut split a rune", name)
		}
		if name != "revision" && !strings.Contains(got, "…") {
			t.Errorf("%s: the cut is not marked (%d runes)", name, utf8.RuneCountInString(got))
		}
	}
}

// TestForeignOccupantIsNamedForWhatItIs: the foreign gap said "A regular file
// occupies the PATH entry" whatever stood there. A directory or a named pipe is
// named as one, and describeEntry says the same.
func TestForeignOccupantIsNamedForWhatItIs(t *testing.T) {
	for _, tc := range []struct {
		kind, gapWant, entryWant string
		plant                    func(t *testing.T, path string)
	}{
		{"directory", "A directory occupies the PATH entry", "a directory abcd does not own", func(t *testing.T, p string) {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"fifo", "A named pipe occupies the PATH entry", "a named pipe abcd does not own", mkfifoOrSkip},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			setupHermetic(t)
			harnessFixture(t, "")
			target := os.Getenv("ABCD_BIN_TARGET")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			tc.plant(t, target)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			var det DetectionResult
			var err error
			withinDeadline(t, "Detect", func() { det, err = Detect(repo) })
			if err != nil {
				t.Fatal(err)
			}
			var g *Gap
			for i := range det.Gaps {
				if det.Gaps[i].ID == "symlink.foreign" {
					g = &det.Gaps[i]
				}
			}
			if g == nil {
				t.Fatalf("no symlink.foreign gap for a %s at the entry: %+v", tc.kind, det.Gaps)
			}
			if !strings.HasPrefix(g.Detail, tc.gapWant) {
				t.Errorf("the gap does not name the occupant: %q", g.Detail)
			}
			if got := describeEntry(pathEntry{path: target, kind: binTargetForeign}); !strings.HasPrefix(got, tc.entryWant) {
				t.Errorf("describeEntry does not name the occupant: %q", got)
			}
		})
	}
}
