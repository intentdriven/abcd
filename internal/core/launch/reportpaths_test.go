package launch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noHomeIn fails when raw names the home directory in either spelling — the
// one the environment gives and the symlink-resolved one the kernel reports —
// which is the developer-identity path machine output never carries (iss-81).
func noHomeIn(t *testing.T, what string, raw []byte, home string) {
	t.Helper()
	spellings := []string{home}
	if real, err := filepath.EvalSymlinks(home); err == nil && real != home {
		spellings = append(spellings, real)
	}
	for _, h := range spellings {
		if strings.Contains(string(raw), h) {
			t.Errorf("%s carries the home directory %q:\n%s", what, h, raw)
		}
	}
}

// homeFixture is renderFixture laid down under a fresh home directory, so every
// absolute path the render and the archive work with sits under $HOME — the
// shape that named the developer in `launch ship --json` and
// `launch archive --json`.
func homeFixture(t *testing.T) (home, root string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	root = filepath.Join(home, "src", "repo")
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md"]}`)
	writeFile(t, root, "README.md", "readme\n")
	writeLockstepTree(t, root, "", "", "")
	writeFile(t, root, ".claude-plugin/plugin.json",
		`{"name": "abcd", "repository": "https://github.com/example/abcd"}`)
	return home, root
}

// The payload's destination is reported without the home path: payload.dest
// names the staging directory with the home redacted to "~", and every bundle
// file's resolved_path is repository-relative — while the render still writes
// to, and the archive still packs from, the real directory
// (iss-2609261848338673, iss-2609261950077063).
func TestTheRenderAndTheArchiveReportTheirPathsWithoutTheHome(t *testing.T) {
	home, root := homeFixture(t)
	out := filepath.Join(home, "dist")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	a, res, err := RenderPluginArchive(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(home, "staging"), Version: "1.2.3", Entry: sampleEntry(), Dirty: DirtySkip,
	}, out)
	if err != nil {
		t.Fatalf("RenderPluginArchive: %v", err)
	}

	rawRes, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	noHomeIn(t, "the render's JSON", rawRes, home)
	var gotRes struct {
		Dest   string `json:"dest"`
		Bundle struct {
			Files []struct {
				LogicalPath  string `json:"logical_path"`
				ResolvedPath string `json:"resolved_path"`
			} `json:"files"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(rawRes, &gotRes); err != nil {
		t.Fatal(err)
	}
	if gotRes.Dest != "~/staging" {
		t.Errorf("payload.dest = %q, want ~/staging", gotRes.Dest)
	}
	if len(gotRes.Bundle.Files) == 0 {
		t.Fatal("the render reported no bundle files")
	}
	for _, f := range gotRes.Bundle.Files {
		if f.ResolvedPath != f.LogicalPath {
			t.Errorf("bundle file %q: resolved_path = %q, want it repository-relative", f.LogicalPath, f.ResolvedPath)
		}
	}

	rawArchive, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	noHomeIn(t, "the archive's JSON", rawArchive, home)
	var gotArchive struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rawArchive, &gotArchive); err != nil {
		t.Fatal(err)
	}
	if want := "~/dist/" + a.Name; gotArchive.Path != want {
		t.Errorf("archive.path = %q, want %q", gotArchive.Path, want)
	}

	// The working values still reach the directories: the payload is on disk
	// where the render put it, and the archive opens from where it was written.
	if _, err := os.Stat(filepath.Join(home, "staging", ".claude-plugin", "plugin.json")); err != nil {
		t.Errorf("the staged payload is not where the render was told to write it: %v", err)
	}
	if entries := zipEntries(t, filepath.Join(out, a.Name)); len(entries) == 0 {
		t.Error("the archive written to --out is empty")
	}
}

// An archive written inside the repository — the release workflow's
// `--out bin` — is reported relative to it.
func TestAnArchiveInsideTheRepositoryIsReportedRelativeToIt(t *testing.T) {
	home, root := homeFixture(t)
	out := filepath.Join(root, "bin")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	a, _, err := RenderPluginArchive(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(home, "staging"), Version: "1.2.3", Entry: sampleEntry(), Dirty: DirtySkip,
	}, out)
	if err != nil {
		t.Fatalf("RenderPluginArchive: %v", err)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if want := "bin/" + a.Name; got.Path != want {
		t.Errorf("archive.path = %q, want %q", got.Path, want)
	}
}
