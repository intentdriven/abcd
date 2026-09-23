package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The pinned plugin archive (the 2026-09-23 ruling E1, adr-2609231200000000):
// the ship renders the release's plugin zip from its own tree and commits its
// address and digest into the catalog; the release workflow renders the zip
// again from the tagged commit through `abcd launch archive` and refuses to
// publish unless the digest matches the committed pin.

const fixtureRepository = "https://github.com/example/abcd"

// shipArchiveRepo is a repository whose ship publishes a pinned archive:
// shipRenderableRepo, whose plugin manifest names the repository the release
// download address derives from.
func shipArchiveRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	return shipRenderableRepo(t)
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
