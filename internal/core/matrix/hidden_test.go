package matrix

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A code task's hidden tests (decision 9) are written before any run and kept
// outside every builder's reach: never in a copy's clone, and never in the
// frozen commit a clone is made from (spc-2610161949559050 step 4).

var widgetHidden = []hiddenTest{
	{Path: "widget/hidden_test.go", Data: []byte("package widget\n\nfunc TestHidden(t *testing.T) {}\n")},
	{Path: "widget/testdata/hidden-case.txt", Data: []byte("a case no builder saw\n")},
}

func TestHiddenTestsSitOutsideTheClone(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden)
	if err != nil {
		t.Fatal(err)
	}
	dir := hiddenDir(store, cloneRun, "widget")
	if rel, err := filepath.Rel(clone, dir); err != nil || !strings.HasPrefix(rel, "..") {
		t.Fatalf("the hidden tests are at %s, inside the clone %s", dir, clone)
	}
	for _, other := range []TrialKey{key, {Task: "widget", Cell: "cheap", Repeat: 2}, {Task: "hidden", Cell: "x", Repeat: 1}} {
		if rel, _ := filepath.Rel(cloneDir(store, cloneRun, keyAlias(other)), dir); !strings.HasPrefix(rel, "..") {
			t.Fatalf("the hidden tests sit inside copy %s's clone", other)
		}
	}
	if len(refs) != len(widgetHidden) {
		t.Fatalf("the run keeps %d hidden tests by content; want %d", len(refs), len(widgetHidden))
	}
	for i, h := range widgetHidden {
		want := contentOf(h.Path, h.Data)
		if refs[i] != want {
			t.Errorf("hidden test %d is kept as %+v; want %+v", i, refs[i], want)
		}
		if err := refs[i].validate(); err != nil {
			t.Errorf("hidden test %s is not a reference a run record keeps: %v", h.Path, err)
		}
		p := filepath.Join(dir, filepath.FromSlash(h.Path))
		data, err := os.ReadFile(p)
		if err != nil || string(data) != string(h.Data) {
			t.Fatalf("hidden test %s: %q %v", h.Path, data, err)
		}
		if fi, _ := os.Stat(p); fi.Mode().Perm()&0o222 != 0 {
			t.Errorf("hidden test %s is writable (%v); it is kept read-only", h.Path, fi.Mode())
		}
		if _, err := os.Stat(filepath.Join(clone, filepath.FromSlash(h.Path))); !os.IsNotExist(err) {
			t.Errorf("the clone holds hidden test %s: %v", h.Path, err)
		}
	}
	_ = filepath.WalkDir(clone, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := os.ReadFile(p)
		for _, h := range widgetHidden {
			if string(data) == string(h.Data) {
				t.Errorf("the clone holds a hidden test's content at %s", p)
			}
		}
		return nil
	})
	if err := hiddenOutsideClone(clone, dir, refs); err != nil {
		t.Fatalf("hidden tests outside the clone are refused: %v", err)
	}
}

// A frozen commit that already holds a hidden test, at any path, would hand it
// to every builder, so the copy is refused before it starts; so is a store
// whose hidden tests would sit inside the clone.
func TestAHiddenTestTheCloneHoldsIsRefused(t *testing.T) {
	src, _ := cloneSource(t)
	src.Write("vendor/leaked.txt", string(widgetHidden[1].Data))
	src.Commit("leak a hidden case")
	frozen := src.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}, keyAlias(TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden)
	if err != nil {
		t.Fatal(err)
	}
	err = hiddenOutsideClone(clone, hiddenDir(store, cloneRun, "widget"), refs)
	if err == nil || !strings.Contains(err.Error(), "vendor/leaked.txt") || !strings.Contains(err.Error(), "widget/testdata/hidden-case.txt") {
		t.Fatalf("a clone holding a hidden test is admitted, or the refusal names neither: %v", err)
	}
	if strings.Contains(err.Error(), clone) {
		t.Errorf("the refusal names where the clone sits on this machine: %v", err)
	}
	if err := hiddenOutsideClone(clone, filepath.Join(clone, "hidden"), refs); err == nil || !strings.Contains(err.Error(), "inside") {
		t.Fatalf("hidden tests inside the clone are admitted: %v", err)
	}
	if err := hiddenOutsideClone(clone, filepath.Dir(clone), refs); err == nil || !strings.Contains(err.Error(), "inside") {
		t.Fatalf("a clone inside the hidden tests' directory is admitted: %v", err)
	}
}

// Hidden tests are placed once, by relative path, from the run's own kept
// copy: a path that climbs, is absolute, enters a git directory or is not a
// recorded name is refused with nothing left behind, and so is a second
// placement or a store inside a working tree.
func TestHiddenTestsArePlacedOnceByRelativePath(t *testing.T) {
	store := filepath.Join(t.TempDir(), "matrix-store")
	for name, tests := range map[string][]hiddenTest{
		"none":                       nil,
		"a climb":                    {{Path: "../escape_test.go", Data: []byte("x")}},
		"an inner climb":             {{Path: "a/../../b_test.go", Data: []byte("x")}},
		"absolute":                   {{Path: "/tmp/a_test.go", Data: []byte("x")}},
		"unclean":                    {{Path: "a//b_test.go", Data: []byte("x")}},
		"a git dir":                  {{Path: ".git/hooks/pre-commit", Data: []byte("x")}},
		"a nested git":               {{Path: "a/.git/config", Data: []byte("x")}},
		"a git dir in other case":    {{Path: ".GIT/hooks/pre-commit", Data: []byte("x")}},
		"a nested git in other case": {{Path: "a/.Git/config", Data: []byte("x")}},
		"a space":                    {{Path: "a b_test.go", Data: []byte("x")}},
		"empty path":                 {{Path: "", Data: []byte("x")}},
		"empty content":              {{Path: "a_test.go", Data: nil}},
		"twice":                      {{Path: "a_test.go", Data: []byte("x")}, {Path: "a_test.go", Data: []byte("y")}},
		"file and dir":               {{Path: "a", Data: []byte("x")}, {Path: "a/b_test.go", Data: []byte("y")}},
	} {
		if _, err := placeHiddenTests(store, cloneRun, "widget", tests); err == nil {
			t.Errorf("%s: placed", name)
		}
		if _, err := os.Stat(hiddenDir(store, cloneRun, "widget")); !os.IsNotExist(err) {
			t.Fatalf("%s: a refused placement left %v", name, err)
		}
	}
	if _, err := placeHiddenTests(store, "not-a-run", "widget", widgetHidden); err == nil {
		t.Error("a bad run id was admitted")
	}
	if _, err := placeHiddenTests(store, cloneRun, "Not_A_Task", widgetHidden); err == nil {
		t.Error("a bad task id was admitted")
	}
	if _, err := placeHiddenTests("relative/store", cloneRun, "widget", widgetHidden); err == nil {
		t.Error("a relative store was admitted")
	}
	if _, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden); err != nil {
		t.Fatal(err)
	}
	if _, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden); err == nil || !strings.Contains(err.Error(), "once") {
		t.Fatalf("a second placement is admitted: %v", err)
	}
	src, _ := cloneSource(t)
	inside := filepath.Join(src.Root(), "store")
	if _, err := placeHiddenTests(inside, cloneRun, "widget", widgetHidden); err == nil || !strings.Contains(err.Error(), "working tree") {
		t.Fatalf("hidden tests inside a working tree are placed: %v", err)
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("nothing is made inside a working tree: %v", err)
	}
}

// A refusal names where a path sits relative to the clone store or the clone,
// never where either sits on this machine.
func TestHiddenTestErrorsNameNoPlaceOnThisMachine(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes and reads whatever a mode says")
	}
	store := filepath.Join(t.TempDir(), "matrix-store")
	parent := filepath.Dir(hiddenDir(store, cloneRun, "widget"))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	_, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden)
	if err == nil || namesPlace(err, store) || !strings.Contains(err.Error(), "<clone store>/hidden/") {
		t.Fatalf("a placement that could not be made: %v; want it named relative to the clone store", err)
	}

	src, frozen := cloneSource(t)
	store = filepath.Join(t.TempDir(), "matrix-store")
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}, keyAlias(TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden)
	if err != nil {
		t.Fatal(err)
	}
	dir := hiddenDir(store, cloneRun, "widget")
	locked := filepath.Join(clone, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	for name, err := range map[string]error{
		"an unreadable folder": hiddenOutsideClone(clone, dir, refs),
		"no clone":             hiddenOutsideClone(filepath.Join(store, "gone"), dir, refs),
	} {
		if err == nil || namesPlace(err, store) {
			t.Errorf("%s: %v; want a refusal naming no place on this machine", name, err)
		}
	}
}

// namesPlace reports whether err names dir, or dir as its links resolve.
func namesPlace(err error, dir string) bool {
	real, _ := filepath.EvalSymlinks(dir)
	return strings.Contains(err.Error(), dir) || real != "" && strings.Contains(err.Error(), real)
}

// hiddenFolderWith is a folder of hidden tests outside every checkout, named
// by its real path, holding files by path.
func hiddenFolderWith(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(realPath(t, t.TempDir()), "hidden")
	for p, text := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A task's hidden tests are read from the folder its declaration names, and
// only from one that lies apart from everything a copy or its export reaches:
// a folder inside the checkout, a worktree of it or the clone store, one
// holding the checkout, one reached through a link or that is one, one
// naming the checkout in another letter case, and one holding what is not a
// plain test file are each refused, and a folder of its own is read whole,
// in path order.
func TestAHiddenTestsFolderIsReadOnlyWhereItLiesApart(t *testing.T) {
	src, _ := cloneSource(t)
	root := realPath(t, src.Root())
	store := filepath.Join(realPath(t, t.TempDir()), "matrix-store")
	mk := func(p string) string {
		t.Helper()
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "widget_hidden_test.go"), []byte("package widget\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := hiddenFolderWith(t, map[string]string{"widget_hidden_test.go": "package widget\n", "sub/b_test.go": "package sub\n"})
	got, err := readHiddenFolder(root, store, good)
	if err != nil {
		t.Fatalf("a folder of its own is refused: %v", err)
	}
	if len(got) != 2 || got[0].Path != "sub/b_test.go" || got[1].Path != "widget_hidden_test.go" || string(got[1].Data) != "package widget\n" {
		t.Fatalf("the folder reads as %+v; want both tests, in path order", got)
	}
	wt := filepath.Join(realPath(t, t.TempDir()), "wt")
	add := exec.Command("git", "-C", root, "worktree", "add", "-q", "--detach", wt)
	add.Env = gittest.Env(t)
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	link := filepath.Join(realPath(t, t.TempDir()), "link")
	if err := os.Symlink(filepath.Dir(good), link); err != nil {
		t.Fatal(err)
	}
	lastLink := filepath.Join(realPath(t, t.TempDir()), "last")
	if err := os.Symlink(good, lastLink); err != nil {
		t.Fatal(err)
	}
	refused := map[string]string{
		"inside the checkout":          mk(filepath.Join(root, "hidden")),
		"inside a worktree":            mk(filepath.Join(wt, "hidden")),
		"inside the clone store":       mk(filepath.Join(store, "elsewhere")),
		"holding the checkout":         filepath.Dir(root),
		"reached through a link":       filepath.Join(link, "hidden"),
		"a link itself":                lastLink,
		"holding a link":               hiddenFolderWith(t, nil),
		"holding a git directory":      hiddenFolderWith(t, map[string]string{".git/config": "x\n"}),
		"holding a test with no bytes": hiddenFolderWith(t, map[string]string{"a_test.go": ""}),
		"holding no file":              mk(filepath.Join(realPath(t, t.TempDir()), "empty")),
		"not a clean path":             good + "/../hidden",
	}
	if err := os.Remove(filepath.Join(refused["holding no file"], "widget_hidden_test.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(refused["holding a link"], 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(good, "widget_hidden_test.go"), filepath.Join(refused["holding a link"], "a_test.go")); err != nil {
		t.Fatal(err)
	}
	// A case-insensitive filesystem names the checkout in any letter case.
	if upper := filepath.Join(filepath.Dir(root), strings.ToUpper(filepath.Base(root)), "hidden"); upper != filepath.Join(root, "hidden") {
		if _, err := os.Lstat(upper); err == nil {
			refused["the checkout in another letter case"] = upper
		}
	}
	for what, folder := range refused {
		if _, err := readHiddenFolder(root, store, folder); err == nil {
			t.Errorf("a folder %s is read", what)
		}
	}
}

// The folder is opened once, as the directory its check judged: a folder
// swapped for another after the check is refused, nothing in it read.
func TestAHiddenTestsFolderSwappedAfterItsCheckIsRefused(t *testing.T) {
	src, _ := cloneSource(t)
	good := hiddenFolderWith(t, map[string]string{"a_test.go": "package a\n"})
	other := hiddenFolderWith(t, map[string]string{"b_test.go": "package b\n"})
	was := hiddenFolderChecked
	hiddenFolderChecked = func(folder string) {
		if err := os.Rename(folder, folder+".was"); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(other, folder); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { hiddenFolderChecked = was })
	got, err := readHiddenFolder(realPath(t, src.Root()), filepath.Join(realPath(t, t.TempDir()), "store"), good)
	if err == nil || !strings.Contains(err.Error(), "replaced") {
		t.Fatalf("a folder swapped after its check gave %+v, %v; want it refused", got, err)
	}
}
