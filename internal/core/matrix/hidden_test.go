package matrix

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, key)
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
		if rel, _ := filepath.Rel(cloneDir(store, cloneRun, other), dir); !strings.HasPrefix(rel, "..") {
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
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1})
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
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1})
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
