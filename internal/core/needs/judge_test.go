package needs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// tree makes the directories rel names under root, each 0o755, and returns
// root joined with rel.
func tree(t *testing.T, root string, rel ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{root}, rel...)...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// program writes a stand-in program at dir/name, mode 0o755. The judge never
// runs what it judges, so its contents are not a program.
func program(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("not run\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func link(t *testing.T, target, at string) string {
	t.Helper()
	if err := os.Symlink(target, at); err != nil {
		t.Fatal(err)
	}
	return at
}

// chmod sets a mode and puts it back before the test's directories are
// removed, so a directory made unenterable does not outlive the test.
func chmod(t *testing.T, p string, mode os.FileMode) {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, fi.Mode().Perm()|fi.Mode()&os.ModeSticky) })
}

// refusedAs asserts err is a *Refusal of kind, naming a path that is the same
// file as at when at is not "".
func refusedAs(t *testing.T, err error, kind Kind, at string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) {
		t.Fatalf("err = %v, want a %v refusal", err, kind)
	}
	if r.Kind != kind {
		t.Fatalf("refused as %v (%v), want %v", r.Kind, err, kind)
	}
	if at == "" {
		return
	}
	a, errA := os.Stat(r.Path)
	b, errB := os.Stat(at)
	if errA != nil || errB != nil || !os.SameFile(a, b) {
		t.Fatalf("the refusal names %q, want %q", r.Path, at)
	}
}

// TestTheJudgeAdmitsAPlainCopy: an absolute path to a file of the tool's name,
// in directories only this account or root can write, is admitted, and the
// verdict names the path given and the file it resolves to.
func TestTheJudgeAdmitsAPlainCopy(t *testing.T) {
	bin := tree(t, t.TempDir(), "sdk", "bin")
	goBin := program(t, bin, "go")
	v, err := Rule{}.Judge("go", goBin)
	if err != nil {
		t.Fatalf("a plain copy: %v", err)
	}
	want, _ := filepath.EvalSymlinks(goBin)
	if v.Path != goBin || v.Target != want || len(v.Links) != 0 || !v.SameName() {
		t.Fatalf("verdict = %+v, want the path %q resolving to %q", v, goBin, want)
	}
	dir := tree(t, bin, "go-dir", "go")
	_, err = Rule{}.Judge("go", dir)
	refusedAs(t, err, NotRegular, dir)
}

// TestTheJudgeRefusesARelativePath: a relative path names whatever the
// working directory makes of it, so it is never registered.
func TestTheJudgeRefusesARelativePath(t *testing.T) {
	for _, p := range []string{"go", "bin/go", "./go", "../bin/go"} {
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Relative, "")
	}
}

// TestTheJudgeRefusesAPathOfAnotherName: the path's own last part is the
// tool's name. A link of that name pointing at a program of another name, as a
// version manager's launcher is, is admitted, and the verdict says so, since
// such a program is never asked its version.
func TestTheJudgeRefusesAPathOfAnotherName(t *testing.T) {
	root := t.TempDir()
	bin := tree(t, root, "bin")
	other := program(t, bin, "go1.26")
	_, err := Rule{}.Judge("go", other)
	refusedAs(t, err, WrongName, "")
	_, err = Rule{}.Judge("go", filepath.Join(bin, "Go"))
	refusedAs(t, err, WrongName, "")

	shims := tree(t, root, "shims")
	launcher := program(t, bin, "mise")
	v, err := Rule{}.Judge("go", link(t, launcher, filepath.Join(shims, "go")))
	if err != nil {
		t.Fatalf("a link named go: %v", err)
	}
	if v.SameName() {
		t.Fatalf("verdict = %+v; a link to a program of another name must say so", v)
	}
}

// TestTheJudgeHoldsALinkAndItsTargetAlike: every link on the way and the file
// it ends at are judged by the same rule, so a link kept in a safe directory
// does not launder a target others can replace, nor the reverse.
func TestTheJudgeHoldsALinkAndItsTargetAlike(t *testing.T) {
	t.Run("a target others can write", func(t *testing.T) {
		root := t.TempDir()
		open := tree(t, root, "open")
		target := program(t, open, "go")
		chmod(t, open, 0o777)
		p := link(t, target, filepath.Join(tree(t, root, "bin"), "go"))
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, open)
	})
	t.Run("a link others can replace", func(t *testing.T) {
		root := t.TempDir()
		target := program(t, tree(t, root, "sdk"), "go")
		open := tree(t, root, "open")
		p := link(t, target, filepath.Join(open, "go"))
		chmod(t, open, 0o777)
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, open)
	})
	t.Run("a link in the middle of the chain", func(t *testing.T) {
		root := t.TempDir()
		target := program(t, tree(t, root, "sdk"), "go")
		middle := tree(t, root, "middle")
		mid := link(t, target, filepath.Join(middle, "go"))
		p := link(t, mid, filepath.Join(tree(t, root, "bin"), "go"))
		chmod(t, middle, 0o777)
		if _, err := (Rule{}).Admit(p); err != nil {
			t.Fatalf("the runner's rule judges the first and last steps only, and admits this: %v", err)
		}
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, middle)
	})
	t.Run("the chain is named", func(t *testing.T) {
		root := t.TempDir()
		target := program(t, tree(t, root, "sdk"), "go")
		mid := link(t, target, filepath.Join(tree(t, root, "middle"), "go"))
		p := link(t, "../middle/go", filepath.Join(tree(t, root, "bin"), "go"))
		v, err := Rule{}.Judge("go", p)
		if err != nil {
			t.Fatalf("a two-step chain: %v", err)
		}
		if len(v.Links) != 2 || !sameEntry(v.Links[0], mid) || !sameEntry(v.Links[1], target) {
			t.Fatalf("links = %q, want [%s %s]", v.Links, mid, target)
		}
	})
	t.Run("a link to somewhere this account cannot enter", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("root enters every directory")
		}
		root := t.TempDir()
		shut := tree(t, root, "home", "sdk")
		target := program(t, shut, "go")
		p := link(t, target, filepath.Join(tree(t, root, "bin"), "go"))
		chmod(t, filepath.Dir(shut), 0o000)
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Unreachable, "")
		if _, err := (Rule{}).Admit(p); err == nil {
			t.Fatal("the runner's rule admits a link it cannot follow")
		}
	})
	t.Run("a link that does not end", func(t *testing.T) {
		root := t.TempDir()
		bin := tree(t, root, "bin")
		a := filepath.Join(bin, "go")
		link(t, filepath.Join(root, "go"), a)
		link(t, a, filepath.Join(root, "go"))
		if _, err := (Rule{}).Judge("go", a); err == nil {
			t.Fatal("a link loop was admitted")
		}
	})
	t.Run("a link out of a guarded tree, and a link into one", func(t *testing.T) {
		root := t.TempDir()
		repo := tree(t, root, "repo")
		planted := program(t, tree(t, root, "repo", "tools"), "go")
		out := link(t, planted, filepath.Join(tree(t, root, "bin"), "go"))
		_, err := Rule{Guards: []string{repo}}.Judge("go", out)
		refusedAs(t, err, Inside, "")
		safe := program(t, tree(t, root, "sdk"), "go")
		in := link(t, safe, filepath.Join(repo, "go"))
		_, err = Rule{Guards: []string{repo}}.Judge("go", in)
		refusedAs(t, err, Inside, "")
	})
}

// TestTheJudgeHoldsEachStepAtItsRealLocation: a step reached through a
// directory link lies where that link leads, so a link the repository holds
// is repository content even when the path the person gives, and the file the
// link points at, both lie outside it. Both rules refuse it, the first step
// and a step in the middle of the chain alike.
func TestTheJudgeHoldsEachStepAtItsRealLocation(t *testing.T) {
	t.Run("the path's own step", func(t *testing.T) {
		root := t.TempDir()
		repo := tree(t, root, "repo")
		safe := program(t, tree(t, root, "safe", "sdk", "bin"), "go")
		link(t, safe, filepath.Join(tree(t, root, "repo", "bin"), "go"))
		alias := link(t, filepath.Join(repo, "bin"), filepath.Join(tree(t, root, "home"), "alias"))
		p := filepath.Join(alias, "go")
		rule := Rule{Guards: []string{repo}}
		_, err := rule.Judge("go", p)
		refusedAs(t, err, Inside, filepath.Join(repo, "bin", "go"))
		_, err = rule.Admit(p)
		refusedAs(t, err, Inside, filepath.Join(repo, "bin", "go"))
	})
	t.Run("a step in the middle of the chain", func(t *testing.T) {
		root := t.TempDir()
		repo := tree(t, root, "repo")
		safe := program(t, tree(t, root, "safe", "sdk", "bin"), "go")
		link(t, safe, filepath.Join(tree(t, root, "repo", "bin"), "go"))
		alias := link(t, filepath.Join(repo, "bin"), filepath.Join(tree(t, root, "home"), "alias"))
		p := link(t, filepath.Join(alias, "go"), filepath.Join(tree(t, root, "bin"), "go"))
		rule := Rule{Guards: []string{repo}}
		_, err := rule.Judge("go", p)
		refusedAs(t, err, Inside, filepath.Join(repo, "bin", "go"))
		_, err = rule.Admit(p)
		refusedAs(t, err, Inside, filepath.Join(repo, "bin", "go"))
	})
	t.Run("a directory link outside every guard", func(t *testing.T) {
		root := t.TempDir()
		repo := tree(t, root, "repo")
		safe := program(t, tree(t, root, "safe", "sdk", "bin"), "go")
		link(t, safe, filepath.Join(tree(t, root, "opt", "bin"), "go"))
		alias := link(t, filepath.Join(root, "opt", "bin"), filepath.Join(tree(t, root, "home"), "alias"))
		rule := Rule{Guards: []string{repo}}
		if _, err := rule.Judge("go", filepath.Join(alias, "go")); err != nil {
			t.Fatalf("a directory link that leads nowhere guarded: %v", err)
		}
	})
}

// TestTheJudgeHoldsEveryAncestor: whoever can write any directory above a
// copy can replace the directory beneath it, so the judge walks every
// ancestor of the path and of each step of its links, where the runner's rule
// stops at the immediate directories. A sticky directory such as /tmp is
// admitted where the entry beneath it is this account's or root's, since no
// one else may rename it.
func TestTheJudgeHoldsEveryAncestor(t *testing.T) {
	t.Run("an open ancestor of the path", func(t *testing.T) {
		root := t.TempDir()
		open := tree(t, root, "open")
		p := program(t, tree(t, root, "open", "sdk", "bin"), "go")
		chmod(t, open, 0o777)
		if _, err := (Rule{}).Admit(p); err != nil {
			t.Fatalf("the runner's rule is unchanged and admits this: %v", err)
		}
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, open)
	})
	t.Run("an open ancestor of the target", func(t *testing.T) {
		root := t.TempDir()
		open := tree(t, root, "open")
		target := program(t, tree(t, root, "open", "sdk", "bin"), "go")
		chmod(t, open, 0o777)
		p := link(t, target, filepath.Join(tree(t, root, "bin"), "go"))
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, open)
	})
	t.Run("an open ancestor reached through a linked directory", func(t *testing.T) {
		root := t.TempDir()
		open := tree(t, root, "open")
		program(t, tree(t, root, "open", "sdk", "bin"), "go")
		chmod(t, open, 0o777)
		mid := link(t, filepath.Join(open, "sdk"), filepath.Join(root, "sdk"))
		p := link(t, filepath.Join(mid, "bin", "go"), filepath.Join(tree(t, root, "bin"), "go"))
		_, err := Rule{}.Judge("go", p)
		refusedAs(t, err, Writable, open)
	})
	t.Run("a sticky ancestor over an entry of this account", func(t *testing.T) {
		root := t.TempDir()
		sticky := tree(t, root, "tmp")
		p := program(t, tree(t, root, "tmp", "mine", "bin"), "go")
		chmod(t, sticky, 0o777|os.ModeSticky)
		if _, err := (Rule{}).Judge("go", p); err != nil {
			t.Fatalf("a sticky ancestor over this account's own directory: %v", err)
		}
		mine := filepath.Join(sticky, "mine")
		withOwners(t, map[string]uint32{mine: 4242})
		_, err := Rule{}.Judge("go", p)
		var r *Refusal
		if !errors.As(err, &r) || (r.Kind != Writable && r.Kind != ForeignOwner) {
			t.Fatalf("err = %v; an entry another account owns under a sticky directory is that account's to replace", err)
		}
	})
}

// withOwners makes the judge read each named path as owned by its uid, and
// every other as owned by this account. A second uid is not something a test
// can make, so the owner lookup is the seam.
func withOwners(t *testing.T, owners map[string]uint32) {
	t.Helper()
	canon := map[string]uint32{}
	for p, uid := range owners {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		canon[p] = uid
	}
	prev := ownerOf
	ownerOf = func(p string, fi os.FileInfo) (uint32, error) {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		if uid, ok := canon[p]; ok {
			return uid, nil
		}
		return prev(p, fi)
	}
	t.Cleanup(func() { ownerOf = prev })
}

// TestTheJudgeHoldsOwners: a copy, or a directory above it or above a link to
// it, owned by an account other than this one or root is refused: that
// account chooses what runs.
func TestTheJudgeHoldsOwners(t *testing.T) {
	root := t.TempDir()
	sdk := tree(t, root, "sdk")
	bin := tree(t, root, "sdk", "bin")
	target := program(t, bin, "go")
	links := tree(t, root, "links")
	p := link(t, target, filepath.Join(links, "go"))

	for name, at := range map[string]string{"the file": target, "its directory": bin, "an ancestor": sdk, "the directory of a link": links} {
		t.Run(name, func(t *testing.T) {
			withOwners(t, map[string]uint32{at: 4242})
			_, err := Rule{}.Judge("go", p)
			refusedAs(t, err, ForeignOwner, at)
		})
	}
	t.Run("root", func(t *testing.T) {
		withOwners(t, map[string]uint32{target: 0, bin: 0, sdk: 0})
		if _, err := (Rule{}).Judge("go", p); err != nil {
			t.Fatalf("a copy root owns: %v", err)
		}
	})
	t.Run("the runner's rule asks no owner", func(t *testing.T) {
		withOwners(t, map[string]uint32{target: 4242})
		if _, err := (Rule{}).Admit(p); err != nil {
			t.Fatalf("Admit: %v; the runner's rule is unchanged and judges modes alone", err)
		}
	})
}

// TestTheJudgeUsesTheGroupItIsGiven: the administrator group's write on
// darwin is admitted through the group the rule reads, the seam the runner's
// own tests narrow.
func TestTheJudgeUsesTheGroupItIsGiven(t *testing.T) {
	root := t.TempDir()
	bin := tree(t, root, "bin")
	p := program(t, bin, "go")
	chmod(t, bin, 0o775)
	admin := Rule{Group: func(os.FileInfo) (uint32, bool) { return 80, true }}
	_, err := admin.Judge("go", p)
	if runtime.GOOS == "darwin" {
		if err != nil {
			t.Fatalf("darwin's admin group: %v", err)
		}
	} else {
		refusedAs(t, err, Writable, bin)
	}
	staff := Rule{Group: func(os.FileInfo) (uint32, bool) { return 20, true }}
	_, err = staff.Judge("go", p)
	refusedAs(t, err, Writable, bin)
}

// sameEntry reports whether a and b name one directory entry, a link judged as
// itself.
func sameEntry(a, b string) bool {
	fa, errA := os.Lstat(a)
	fb, errB := os.Lstat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}
