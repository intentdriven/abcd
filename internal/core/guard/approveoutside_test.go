package guard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// abcdLayerRig is a working tree kept where abcd keeps a run's worktree,
// under the home's abcd folder, with that folder's trust files beside it.
func abcdLayerRig(t *testing.T) (approveRig, string) {
	t.Helper()
	home := t.TempDir()
	layer := abcdhome.Path(home)
	r := approveRig{dir: filepath.Join(layer, "worktrees", "0123abcd", "lane"), home: home}
	for _, d := range []string{r.dir, filepath.Join(layer, "worktrees", "0123abcd", "other"), filepath.Join(r.dir, "build")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"trusted-roots", "rules.json", "config.json"} {
		if err := os.WriteFile(filepath.Join(layer, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return r, layer
}

// TestApproveProtectsAbcdsOwnUserLayer: the home's abcd folder holds the
// files that decide what abcd trusts on the machine (trusted-roots,
// rules.json, config.json and the rest of the layer), so a write into it or
// a removal of it is refused, while the run's own worktree kept inside it
// stays writable.
func TestApproveProtectsAbcdsOwnUserLayer(t *testing.T) {
	r, layer := abcdLayerRig(t)
	r.run(t, []approveCase{
		{`echo /x >> ` + abcdhome.Display("trusted-roots"), protectedOrRegistry},
		{`printf '{}' > ` + abcdhome.Display("rules.json"), protectedOrRegistry},
		{`echo '{}' > ` + filepath.Join(layer, "config.json"), protectedOrRegistry},
		{`cp x ` + abcdhome.Display("local-transcript-roots"), protectedOrRegistry},
		{`touch ../other/notes.txt`, protectedOrRegistry},
		{`rm -f ` + abcdhome.Display("trusted-roots"), protectedOrRegistry},
		// The run's own worktree.
		{`echo x > notes.txt`, nil},
		{`echo x > ` + filepath.Join(r.dir, "notes.txt"), nil},
		{`rm -rf build`, nil},
		{`cat ` + abcdhome.Display("trusted-roots") + ` > roots.copy`, nil},
	})
}

// TestApproveRefusesRemovalsOutsideTheWorkingTree: a removal whose target
// the line fixes outside the launched working tree, the run's own
// directories and the temporary directory is refused; a write outside the
// tree is judged as before, by the protected set alone.
func TestApproveRefusesRemovalsOutsideTheWorkingTree(t *testing.T) {
	root := t.TempDir()
	r := approveRig{dir: filepath.Join(root, "repo"), home: filepath.Join(root, "home")}
	run := filepath.Join(root, "run")
	tmp := filepath.Join(root, "tmp")
	for _, d := range []string{r.dir, r.home, run, tmp, filepath.Join(r.home, "Documents", "x"), filepath.Join(r.home, ".ssh")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	temps := approveTempDirs
	t.Cleanup(func() { approveTempDirs = temps })
	approveTempDirs = func() []string { return []string{tmp} }
	outside := refusedAs(outsideRemovalEntryID)
	cases := []approveCase{
		{`rm -rf ~/Documents/x`, outside},
		{`rm ~/.ssh/id_ed25519`, outside},
		{`rm -f ` + filepath.Join(r.home, ".ssh", "known_hosts"), outside},
		{`unlink ` + filepath.Join(root, "elsewhere.txt"), outside},
		{`mv ~/Documents/x/report.pdf .`, outside},
		{`find ~/Documents/x -name '*.tmp' -delete`, outside},
		{`rsync -a --delete src/ ~/Documents/x/`, outside},
		{`cd .. && rm -f elsewhere.txt`, outside},
		// Inside the tree, the run's own directories and the temporary one.
		{`rm -f build/x.o`, nil},
		{`rm -rf ` + filepath.Join(r.dir, "build"), nil},
		{`rm -f ` + filepath.Join(run, "receipt.json"), nil},
		{`rm -rf ` + filepath.Join(tmp, "scratch"), nil},
		{`find build -name '*.o' -delete`, nil},
		// A write outside the tree is the sandbox's to bound.
		{`echo x > ~/Documents/x/notes.txt`, nil},
		{`cp build/out.txt ` + filepath.Join(root, "elsewhere.txt"), nil},
	}
	f := func() *Files { return r.files().WithRunDirs(run) }
	for _, c := range cases {
		a := approveShell(clean(), f(), c.line)
		switch {
		case len(c.ids) == 0 && !a.Allow:
			t.Errorf("%q: refused (%s): %s", c.line, a.EntryID, a.Reason)
		case len(c.ids) > 0 && a.Allow:
			t.Errorf("%q was approved; want a refusal under %v", c.line, c.ids)
		case len(c.ids) > 0 && !containsString(c.ids, a.EntryID):
			t.Errorf("%q: refused under %q, want %v (%s)", c.line, a.EntryID, c.ids, a.Reason)
		}
	}
	if !containsString(reservedEntryIDs, outsideRemovalEntryID) {
		t.Errorf("reservedEntryIDs lacks %q", outsideRemovalEntryID)
	}
}
