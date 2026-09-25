package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The pinned plugin archive (the 2026-09-23 ruling E1, adr-2609231048308186):
// the ship renders the release's plugin zip from its own tree and commits its
// address and digest into the catalog; the release workflow renders the zip
// again from the tagged commit through `abcd launch archive` and refuses to
// publish unless the digest matches the committed pin.

const fixtureRepository = "https://github.com/example/abcd"

// shipArchiveRepo is a repository whose ship publishes a pinned archive:
// shipRenderableRepo, whose plugin manifest names the repository the release
// download address derives from, plus the version-location contract's positive
// declaration that its release publishes the archive.
func shipArchiveRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipRenderableRepo(t)
	r.Write(".abcd/config/version-location.json",
		`{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version", "publishes_plugin_archive": true}`+"\n")
	r.Commit("declare the published plugin archive")
	return r
}

// refreshSurface regenerates the committed surface snapshot after a fixture
// edits a manifest, so the ship's guardrail compares like with like.
func refreshSurface(t *testing.T, r *gittest.Repo) {
	t.Helper()
	data, err := GenerateSurface(r.Root())
	if err != nil {
		t.Fatalf("GenerateSurface: %v", err)
	}
	r.Write(SurfaceSnapshotPath, string(data))
	r.Commit("refresh the surface snapshot")
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestLaunchShipPinsTheReleaseArchive is the wiring detector for the ship half:
// a ship that lands the dated heading also renders the release archive and
// commits its address and digest into the catalog — and that pin is exactly
// what an independent re-render by the release gate reproduces.
func TestLaunchShipPinsTheReleaseArchive(t *testing.T) {
	r := shipArchiveRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	pluginBefore := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/plugin.json"))

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
	}
	if !strings.Contains(string(out), "archive:") || !strings.Contains(string(out), "abcd-plugin-v0.4.1.zip") {
		t.Errorf("the ship must report the archive it pinned:\n%s", out)
	}

	pin, ok, err := launch.ReadArchivePin(r.Root())
	if err != nil || !ok {
		t.Fatalf("the catalog must carry the archive pin after a ship: ok=%v err=%v", ok, err)
	}
	wantURL := fixtureRepository + "/releases/download/v0.4.1/abcd-plugin-v0.4.1.zip"
	if pin.URL != wantURL {
		t.Errorf("pin url = %q, want %q", pin.URL, wantURL)
	}
	if got := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/plugin.json")); got != pluginBefore {
		t.Error("adr-19: the ship mutated the working tree's plugin.json")
	}
	if res := launch.CheckLockstep(launch.TreeDev, r.Root(), filepath.Join(r.Root(), ".abcd/config/version-location.json")); !res.OK {
		t.Errorf("a pinned working tree must stay version-absent, got %+v", res)
	}

	// The surface snapshot follows the catalog's new shape, so the ship's own
	// commit carries a snapshot that matches its tree.
	live, err := GenerateSurface(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if committed := readFileString(t, filepath.Join(r.Root(), SurfaceSnapshotPath)); committed != string(live) {
		t.Error("the ship rewrote the catalog but left the surface snapshot stale")
	}

	// The release gate's independent re-render of the same tree reproduces the pin.
	r.Commit("the release cut")
	outDir := t.TempDir()
	out, err = shipIn(t, r, "launch", "archive", "--out", outDir, "--tag", "v0.4.1", "--verify", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("launch archive --verify exit = %d, want 0\n%s\n%v", code, out, err)
	}
	var rep struct {
		Archive launch.PluginArchive `json:"archive"`
		Pin     struct {
			Checked bool `json:"checked"`
			OK      bool `json:"ok"`
		} `json:"pin"`
	}
	if err := json.Unmarshal(out, &rep); err != nil {
		t.Fatalf("parse --json: %v\n%s", err, out)
	}
	if rep.Archive.SHA256 != pin.SHA256 || !rep.Pin.Checked || !rep.Pin.OK {
		t.Errorf("the re-render must reproduce the pinned digest %s, got %+v", pin.SHA256, rep)
	}
	if _, err := os.Stat(filepath.Join(outDir, "abcd-plugin-v0.4.1.zip")); err != nil {
		t.Errorf("the verified archive must be written to --out: %v", err)
	}
}

// TestLaunchShipArchiveRefusalsWriteNothing is the atomicity detector for the
// archive half: every reason it can refuse without a version is found BEFORE
// the dated heading is written, so a refused ship leaves the release record and
// the catalog exactly as they were.
func TestLaunchShipArchiveRefusalsWriteNothing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, r *gittest.Repo)
		want  string
	}{
		{
			name: "the plugin manifest names no repository",
			setup: func(t *testing.T, r *gittest.Repo) {
				r.Write(".claude-plugin/plugin.json", `{"name":"abcd","description":"fixture"}`+"\n")
				r.Commit("drop the repository")
				refreshSurface(t, r)
			},
			want: "repository",
		},
		{
			name: "a payload file carries an uncommitted change",
			setup: func(t *testing.T, r *gittest.Repo) {
				r.Write(".claude-plugin/extra.json", "{}\n")
			},
			want: "uncommitted",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := shipArchiveRepo(t)
			tc.setup(t, r)
			payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
			changelog := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md"))
			market := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/marketplace.json"))

			out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal should name %q, got %v", tc.want, err)
			}
			if got := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")); got != changelog {
				t.Error("a refused archive wrote the release record anyway")
			}
			if got := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/marketplace.json")); got != market {
				t.Error("a refused archive rewrote the catalog anyway")
			}
		})
	}
}

// TestLaunchArchiveVerifyRefusesADrift is the release gate's refusal: once the
// payload changes after the ship pinned it, the re-render no longer reproduces
// the committed digest, and the gate exits 1 naming both digests — and leaves
// no archive behind for a later upload step to pick up.
func TestLaunchArchiveVerifyRefusesADrift(t *testing.T) {
	r := shipArchiveRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	if out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload); exitCodeOf(err) != 0 {
		t.Fatalf("ship: %v\n%s", err, out)
	}
	pin, _, _ := launch.ReadArchivePin(r.Root())
	r.Write(".claude-plugin/extra.json", `{"late": true}`+"\n")
	r.Commit("the release cut, plus a payload change the pin never saw")

	outDir := t.TempDir()
	out, err := shipIn(t, r, "launch", "archive", "--out", outDir, "--verify")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s\n%v", code, out, err)
	}
	if !strings.Contains(string(out), pin.SHA256) || !strings.Contains(string(out), "MISMATCH") {
		t.Errorf("the refusal must name the committed digest:\n%s", out)
	}
	if entries, _ := os.ReadDir(outDir); len(entries) != 0 {
		t.Errorf("a refused archive must not be left in --out, found %v", entries)
	}
}

// TestLaunchArchiveRefusesAnotherReleasesTag pins the tag binding: the archive
// is rendered for the newest dated CHANGELOG version, and a caller that says it
// is releasing a different tag is refused before anything is rendered.
func TestLaunchArchiveRefusesAnotherReleasesTag(t *testing.T) {
	r := shipArchiveRepo(t)
	outDir := t.TempDir()
	out, err := shipIn(t, r, "launch", "archive", "--out", outDir, "--tag", "v9.9.9")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
	}
	if err == nil || !strings.Contains(err.Error(), "v0.4.0") {
		t.Errorf("the refusal must name the version the CHANGELOG dates, got %v", err)
	}
	if entries, _ := os.ReadDir(outDir); len(entries) != 0 {
		t.Errorf("nothing may be written on a refused tag, found %v", entries)
	}
}

// TestLaunchArchiveWithoutVerifyWritesTheArchive keeps the render usable on its
// own — to inspect the zip a release would publish — and says plainly that the
// pin was not checked.
func TestLaunchArchiveWithoutVerifyWritesTheArchive(t *testing.T) {
	r := shipArchiveRepo(t)
	outDir := t.TempDir()
	out, err := shipIn(t, r, "launch", "archive", "--out", outDir)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "abcd-plugin-v0.4.0.zip")); err != nil {
		t.Errorf("the archive must be written: %v", err)
	}
	if !strings.Contains(string(out), "not checked") {
		t.Errorf("an unverified render must say the pin was not checked:\n%s", out)
	}
}

// TestLaunchArchiveWithoutVerifyRefusesADirtyTree is iss-2609251827294854 at
// the archive verb: the render states its dirty-tree policy, and without
// --verify nothing else judges the tree, so an uncommitted change refuses and
// nothing is left in --out.
func TestLaunchArchiveWithoutVerifyRefusesADirtyTree(t *testing.T) {
	r := shipArchiveRepo(t)
	r.Write("notes.txt", "scratch\n")
	outDir := t.TempDir()
	out, err := shipIn(t, r, "launch", "archive", "--out", outDir)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
	}
	if err == nil || !strings.Contains(err.Error(), "uncommitted") || !strings.Contains(err.Error(), "notes.txt") {
		t.Errorf("the refusal must name the uncommitted file, got %v", err)
	}
	if entries, _ := os.ReadDir(outDir); len(entries) != 0 {
		t.Errorf("a refused render must leave nothing in --out, found %v", entries)
	}
}

// TestLaunchShipPinsOnlyOnTheDeclaration is the other side of the pin: a
// repository with the version-location contract but no declaration that its
// release publishes the archive — the managed repository whose scaffolded
// workflows upload none — keeps its catalog exactly as it was, and the ship
// says so in both renderings. A pin there would name an asset nothing uploads.
func TestLaunchShipPinsOnlyOnTheDeclaration(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		t.Run(map[bool]string{false: "human", true: "json"}[asJSON], func(t *testing.T) {
			r := shipRenderableRepo(t)
			payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
			market := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/marketplace.json"))
			snapshot := readFileString(t, filepath.Join(r.Root(), SurfaceSnapshotPath))

			args := []string{"launch", "ship", "--changelog-json", payload}
			if asJSON {
				args = append(args, "--json")
			}
			out, err := shipIn(t, r, args...)
			if code := exitCodeOf(err); code != 0 {
				t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
			}
			if got := readFileString(t, filepath.Join(r.Root(), ".claude-plugin/marketplace.json")); got != market {
				t.Errorf("an undeclared archive was pinned into the catalog:\n%s", got)
			}
			if got := readFileString(t, filepath.Join(r.Root(), SurfaceSnapshotPath)); got != snapshot {
				t.Error("the surface snapshot was rewritten for a catalog that did not change")
			}
			if !asJSON {
				if !strings.Contains(string(out), "not pinned") || !strings.Contains(string(out), "publishes_plugin_archive") {
					t.Errorf("the ship must say the catalog was left untouched, and why:\n%s", out)
				}
				return
			}
			var rep map[string]any
			if err := json.Unmarshal(out, &rep); err != nil {
				t.Fatalf("parse --json: %v\n%s", err, out)
			}
			if _, ok := rep["archive"]; ok {
				t.Errorf("--json reports an archive that was not pinned: %v", rep["archive"])
			}
			note, _ := rep["archive_unpinned"].(string)
			if !strings.Contains(note, "publishes_plugin_archive") {
				t.Errorf("--json must say why the catalog was left untouched, got archive_unpinned=%q", note)
			}
		})
	}
}

// TestLaunchShipRefusesAnUnreadableDeclaration keeps the declaration honest: a
// publishes_plugin_archive that is not a boolean is neither answer, so the ship
// refuses before the release record is written rather than guessing.
func TestLaunchShipRefusesAnUnreadableDeclaration(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write(".abcd/config/version-location.json",
		`{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version", "publishes_plugin_archive": "yes"}`+"\n")
	r.Commit("an unreadable declaration")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	changelog := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md"))

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
	}
	if err == nil || !strings.Contains(err.Error(), "publishes_plugin_archive") {
		t.Errorf("the refusal must name the declaration, got %v", err)
	}
	if got := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")); got != changelog {
		t.Error("a refused declaration wrote the release record anyway")
	}
}

// TestLaunchArchiveRepositoryBindsTheAddress is the release workflow's
// repository gate: the archive address derives from plugin.json's repository,
// which a rename, a transfer or a fork leaves naming another repository —
// --verify passes on it and every install 404s. --repository refuses (exit 1,
// the archive removed) unless the address sits under that repository's
// download path for the release's tag.
func TestLaunchArchiveRepositoryBindsTheAddress(t *testing.T) {
	t.Run("this repository", func(t *testing.T) {
		r := shipArchiveRepo(t)
		outDir := t.TempDir()
		out, err := shipIn(t, r, "launch", "archive", "--out", outDir, "--repository", "Example/ABCD", "--json")
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
		}
		var rep struct {
			Repository struct {
				Checked bool `json:"checked"`
				OK      bool `json:"ok"`
			} `json:"repository"`
		}
		if err := json.Unmarshal(out, &rep); err != nil {
			t.Fatalf("parse --json: %v\n%s", err, out)
		}
		if !rep.Repository.Checked || !rep.Repository.OK {
			t.Errorf("the repository check must be reported as passed, got %+v", rep.Repository)
		}
		if _, err := os.Stat(filepath.Join(outDir, "abcd-plugin-v0.4.0.zip")); err != nil {
			t.Errorf("the archive must be written: %v", err)
		}
	})
	t.Run("another repository", func(t *testing.T) {
		r := shipArchiveRepo(t)
		outDir := t.TempDir()
		out, err := shipIn(t, r, "launch", "archive", "--out", outDir, "--repository", "example/fork")
		if code := exitCodeOf(err); code != 1 {
			t.Fatalf("exit = %d, want 1\n%s\n%v", code, out, err)
		}
		if !strings.Contains(string(out), "example/fork") || !strings.Contains(string(out), "MISMATCH") {
			t.Errorf("the refusal must name the repository it was bound to:\n%s", out)
		}
		if entries, _ := os.ReadDir(outDir); len(entries) != 0 {
			t.Errorf("a refused archive must not be left in --out, found %v", entries)
		}
	})
	t.Run("not owner/name", func(t *testing.T) {
		r := shipArchiveRepo(t)
		outDir := t.TempDir()
		out, err := shipIn(t, r, "launch", "archive", "--out", outDir, "--repository", "https://github.com/example/abcd")
		if code := exitCodeOf(err); code != 2 {
			t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
		}
		if entries, _ := os.ReadDir(outDir); len(entries) != 0 {
			t.Errorf("nothing may be written on a malformed operand, found %v", entries)
		}
	})
}

// TestRollbackCutAfterTheArchivePinRestoresEverything: a cut refused once the
// archive pin has landed — the last write a ship makes — is undone whole. The
// CHANGELOG heading, the release page, the archived outgoing page, the catalog's
// pin and the surface snapshot all return to their pre-ship bytes. The core's
// undo plan carries the first three and the ship holds the other two itself, so
// a rollback that applies only one of them leaves a half-cut release behind.
func TestRollbackCutAfterTheArchivePinRestoresEverything(t *testing.T) {
	r := shipArchiveRepo(t)
	r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n")
	r.Commit("the previous release page")

	watched := []string{
		"CHANGELOG.md",
		"RELEASE.md",
		".abcd/development/releases/0.4.0.md",
		".claude-plugin/marketplace.json",
		SurfaceSnapshotPath,
	}
	read := func(rel string) (string, bool) {
		data, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(rel)))
		if os.IsNotExist(err) {
			return "", false
		}
		if err != nil {
			t.Fatal(err)
		}
		return string(data), true
	}
	type state struct {
		data    string
		present bool
	}
	pre := map[string]state{}
	for _, rel := range watched {
		data, ok := read(rel)
		pre[rel] = state{data, ok}
	}
	before := cliTreeDigest(t, r.Root())

	// What the ship holds before its ingest, exactly as runShipIngest does.
	var saved []savedFile
	for _, rel := range []string{".claude-plugin/marketplace.json", SurfaceSnapshotPath} {
		f, err := saveFile(r.Root(), rel)
		if err != nil {
			t.Fatal(err)
		}
		saved = append(saved, f)
	}

	raw, err := os.ReadFile(composedPayload(t, t.TempDir(), "v0.4.1", "itd-73"))
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 21, 9, 30, 0, 0, time.UTC)
	ingested, err := ingestCut(r.Root(), raw, at)
	if err != nil || !ingested.Written {
		t.Fatalf("ingest: written=%v err=%v", ingested.Written, err)
	}
	scratch := t.TempDir()
	zipDir := filepath.Join(scratch, "archive")
	if err := os.Mkdir(zipDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := pinReleaseArchive(r.Root(), filepath.Join(scratch, "payload"), zipDir, ingested.Cut, at); err != nil {
		t.Fatalf("pin: %v", err)
	}
	for _, rel := range watched {
		data, ok := read(rel)
		if (state{data, ok}) == pre[rel] {
			t.Fatalf("the cut left %s untouched, so this detector would prove nothing about restoring it", rel)
		}
	}

	if got := rollbackCut(r.Root(), "", ingested.Undo, saved...); !strings.Contains(got, "the release record and page were rolled back") {
		t.Fatalf("rollback report = %q", got)
	}
	for _, rel := range watched {
		data, ok := read(rel)
		if (state{data, ok}) != pre[rel] {
			t.Errorf("the rollback did not restore %s (present=%v, want present=%v)", rel, ok, pre[rel].present)
		}
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("the rollback left the tree different from before the cut")
	}
}

// shipWithUnwritableDir runs a real `abcd launch ship` ingest against a
// repository whose relDir cannot take a new file, so the write the cut makes
// there fails AFTER the dated heading, the release page and the archived page
// have landed. It returns the refusal and the tree digests either side.
func shipWithUnwritableDir(t *testing.T, relDir string) (err error, before, after string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("a directory's mode does not bind the superuser")
	}
	r := shipArchiveRepo(t)
	r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n")
	r.Commit("the previous release page")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	dir := filepath.Join(r.Root(), filepath.FromSlash(relDir))
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	before = cliTreeDigest(t, r.Root())
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2 (the post-write fault)\n%s\n%v", code, out, err)
	}
	return err, before, cliTreeDigest(t, r.Root())
}

// TestLaunchShipRollbackSkipsAFileTheCutNeverChanged: a fault that stops the
// archive pin's own write leaves the catalog as it was, so the rollback has
// nothing to put back there. Rewriting it anyway fails the same way the pin did
// and reports THE ROLLBACK FAILED for a file the cut never changed, sending an
// operator to recover by hand a tree that is already clean.
func TestLaunchShipRollbackSkipsAFileTheCutNeverChanged(t *testing.T) {
	err, before, after := shipWithUnwritableDir(t, ".claude-plugin")
	if msg := err.Error(); strings.Contains(msg, "THE ROLLBACK FAILED") ||
		!strings.Contains(msg, "the release record and page were rolled back") {
		t.Errorf("a rollback that restored everything the cut changed must report success, got:\n%s", msg)
	}
	if after != before {
		t.Error("the refused cut left the tree different from before it")
	}
}

// TestLaunchShipRollbackGetsTheCutsUndoAndSavedFiles drives the rollback
// through the verb rather than calling it directly: the surface snapshot's
// directory refuses the refresh, which runs after the ingest's three writes
// AND the catalog pin. Only a rollback handed the ingest's real undo plan
// (CHANGELOG, RELEASE.md, the archived page) and the ship's saved catalog
// returns the tree to its pre-ship digest, so a call site that passes an empty
// plan or drops the saved files fails here.
func TestLaunchShipRollbackGetsTheCutsUndoAndSavedFiles(t *testing.T) {
	err, before, after := shipWithUnwritableDir(t, filepath.Dir(filepath.FromSlash(SurfaceSnapshotPath)))
	if msg := err.Error(); strings.Contains(msg, "THE ROLLBACK FAILED") ||
		!strings.Contains(msg, "the release record and page were rolled back") {
		t.Errorf("the rollback must report success, got:\n%s", msg)
	}
	if after != before {
		t.Error("the refused cut left the tree different from before it: the rollback was not handed every write the cut made")
	}
}

// TestSavedFileRestoreCases pins every branch of the restore: an unchanged file
// is left alone, a changed or deleted one gets its saved bytes back, and one the
// cut created is removed again.
func TestSavedFileRestoreCases(t *testing.T) {
	cases := []struct {
		name   string
		before *string // nil: absent before the cut
		cut    func(path string) error
		want   *string // nil: absent after the restore
	}{
		{"unchanged", restoreText("a"), func(string) error { return nil }, restoreText("a")},
		{"rewritten", restoreText("a"), func(p string) error { return os.WriteFile(p, []byte("b"), 0o644) }, restoreText("a")},
		{"deleted", restoreText("a"), os.Remove, restoreText("a")},
		{"created", nil, func(p string) error { return os.WriteFile(p, []byte("b"), 0o644) }, nil},
		{"still absent", nil, func(string) error { return nil }, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "f.json")
			if c.before != nil {
				if err := os.WriteFile(path, []byte(*c.before), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			f, err := saveFile(root, "f.json")
			if err != nil {
				t.Fatal(err)
			}
			if err := c.cut(path); err != nil {
				t.Fatal(err)
			}
			if err := f.restore(root); err != nil {
				t.Fatalf("restore: %v", err)
			}
			data, err := os.ReadFile(path)
			switch {
			case c.want == nil && !os.IsNotExist(err):
				t.Errorf("want the file absent, got err=%v data=%q", err, data)
			case c.want != nil && (err != nil || string(data) != *c.want):
				t.Errorf("want %q, got %q (err=%v)", *c.want, data, err)
			}
		})
	}
}

func restoreText(s string) *string { return &s }
