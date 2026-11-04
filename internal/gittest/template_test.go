package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTemplatesBuildOnceAndCopy: a key is built once, every test holds its own
// working repository with the fixture's history, and what one test changes
// reaches neither the template nor another test's copy.
func TestTemplatesBuildOnceAndCopy(t *testing.T) {
	var ts Templates
	builds := 0
	build := func(t *testing.T) *Repo {
		builds++
		r := NewRepo(t)
		r.Write("a.txt", "a\n")
		r.Commit("seed")
		return r
	}
	first := ts.Repo(t, "k", build)
	first.Write("a.txt", "changed\n")
	first.Git("add", "-A")
	first.Commit("second")

	second := ts.Repo(t, "k", build)
	if builds != 1 {
		t.Errorf("built %d times, want once", builds)
	}
	if second.Root() == first.Root() {
		t.Fatalf("two tests share %s", second.Root())
	}
	if got := second.Git("rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("the second copy holds %s commits, want the template's one: the first test's commit reached it", got)
	}
	if got := second.Git("status", "--porcelain"); got != "" {
		t.Errorf("a fresh copy is not clean: %q", got)
	}
	if raw, err := os.ReadFile(filepath.Join(second.Root(), "a.txt")); err != nil || string(raw) != "a\n" {
		t.Errorf("the second copy's file = %q (%v)", raw, err)
	}
}

// TestTemplatesKeepEachKeyApart: two keys are two builds, whatever a key's text.
func TestTemplatesKeepEachKeyApart(t *testing.T) {
	var ts Templates
	build := func(body string) func(t *testing.T) *Repo {
		return func(t *testing.T) *Repo {
			r := NewRepo(t)
			r.Write("f", body)
			return r
		}
	}
	a, b := ts.Repo(t, "a/with a slash", build("a")), ts.Repo(t, "b", build("b"))
	for dir, want := range map[string]string{a.Root(): "a", b.Root(): "b"} {
		if raw, err := os.ReadFile(filepath.Join(dir, "f")); err != nil || string(raw) != want {
			t.Errorf("%s holds %q (%v), want %q", dir, raw, err, want)
		}
	}
}

// TestATemplateKeepsModesAndLinks: a hook stays runnable, a private file stays
// private, a read-only folder stays read-only and a link stays a link.
func TestATemplateKeepsModesAndLinks(t *testing.T) {
	src := t.TempDir()
	for name, mode := range map[string]os.FileMode{"hook": 0o755, "private": 0o600, "read-only": 0o444} {
		p := filepath.Join(src, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("hook", filepath.Join(src, "link")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	made := filepath.Join(src, "sealed")
	if err := os.Mkdir(made, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(made, "f"), []byte("f"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(made, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(made, 0o755) })
	got := filepath.Join(t.TempDir(), "copy")
	readTree(t, src).write(t, got)
	sealed := filepath.Join(got, "sealed")
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o755) })
	for name, want := range map[string]os.FileMode{"hook": 0o755, "private": 0o600, "read-only": 0o444, "sealed": 0o555} {
		fi, err := os.Stat(filepath.Join(got, name))
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s has mode %v, want %v", name, fi.Mode().Perm(), want)
		}
	}
	if target, err := os.Readlink(filepath.Join(got, "link")); err != nil || target != "hook" {
		t.Errorf("link = %q (%v), want a link to hook", target, err)
	}
}

// TestAnEmptyRepositoryIsWhatGitInitMakes: the kept repository holds exactly
// what the real commands leave, entry for entry, in each kind and format.
func TestAnEmptyRepositoryIsWhatGitInitMakes(t *testing.T) {
	realWork := func(t *testing.T, format string) string {
		dir, env := t.TempDir(), Env(t)
		args := []string{"-C", dir, "init", "--initial-branch=main"}
		if format != "" {
			args = append(args, "--object-format="+format)
		}
		cmd := exec.Command("git", args...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git init unavailable: %v (%s)", err, out)
		}
		disableMaintenance(t, env, dir)
		return dir
	}
	realBare := func(t *testing.T, format string) string {
		dir := filepath.Join(t.TempDir(), "bare.git")
		initBare(t, Env(t), dir, format)
		return dir
	}
	viaHelper := map[string]func(t *testing.T, format string) string{
		"work": func(t *testing.T, format string) string { return NewRepoFormat(t, format).Root() },
		"bare": func(t *testing.T, format string) string {
			dir := filepath.Join(t.TempDir(), "made", "bare.git")
			InitBareFormat(t, dir, format)
			return dir
		},
	}
	for kind, real := range map[string]func(t *testing.T, format string) string{"work": realWork, "bare": realBare} {
		for _, format := range []string{"", "sha256"} {
			t.Run(kind+" "+format, func(t *testing.T) {
				viaHelper[kind](t, format) // the first may be the one that keeps it
				got, want := readTree(t, viaHelper[kind](t, format)), readTree(t, real(t, format))
				if len(got) != len(want) {
					t.Errorf("%d entries, want %d", len(got), len(want))
				}
				byRel := map[string]treeEntry{}
				for _, e := range got {
					byRel[e.rel] = e
				}
				for _, w := range want {
					g, ok := byRel[w.rel]
					switch {
					case !ok:
						t.Errorf("%s is missing", w.rel)
					case g.mode != w.mode:
						t.Errorf("%s has mode %v, want %v", w.rel, g.mode, w.mode)
					case string(g.body) != string(w.body):
						t.Errorf("%s differs:\n%s\nwant:\n%s", w.rel, g.body, w.body)
					}
				}
			})
		}
	}
}

// TestAnotherGitGetsItsOwnEmptyRepository: a test that puts another git first on
// PATH sees that git run, and with no git at all NewRepo still skips.
func TestAnotherGitGetsItsOwnEmptyRepository(t *testing.T) {
	NewRepo(t) // kept for the ordinary git
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	t.Run("another git", func(t *testing.T) {
		bin := t.TempDir()
		marker := filepath.Join(bin, "ran")
		shim := "#!/bin/sh\necho x >> '" + marker + "'\nexec '" + real + "' \"$@\"\n"
		if err := os.WriteFile(filepath.Join(bin, "git"), []byte(shim), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		NewRepo(t)
		if _, err := os.Stat(marker); err != nil {
			t.Errorf("NewRepo did not run the git PATH names: %v", err)
		}
	})
	var skipped bool
	t.Run("no git", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		t.Setenv("PATH", t.TempDir())
		NewRepo(t)
		t.Error("NewRepo carried on with no git on PATH")
	})
	if !skipped {
		t.Error("NewRepo did not skip with no git on PATH")
	}
}

// TestInitBareOverWhatIsThereLeavesIt: a bare repository made over a directory
// holding files keeps them, as git init does.
func TestInitBareOverWhatIsThereLeavesIt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kept"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	InitBare(t, dir)
	if _, err := os.Stat(filepath.Join(dir, "kept")); err != nil {
		t.Errorf("the file already there went: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		t.Errorf("no bare repository: %v", err)
	}
}
