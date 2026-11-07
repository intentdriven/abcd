package cloudscripts

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// Most of this package's tests start from one of a few cases, a clone and its
// bare remote, which take dozens of processes to stand up before a test reaches
// the script it judges. Each is built once a test process and written out again
// for every later test; every test still gets its own files.
//
// A case's files name its directory: the clone's remote URL, the insteadOf that
// maps the forge's URL onto the bare repository, and the local abcd's wrapper. A
// byte copy would leave a clone pushing into the first test's remote, so each
// file naming the directory has it rewritten as it is written out
// (TestACopiedCaseIsItsOwn). git quotes a path holding some characters, which a
// rewrite would not, so a test whose directory holds one builds its case with
// git, and one never becomes the template.

// caseTemplates are the kept cases, by what was built. Tests here never run in
// parallel, so nothing guards it.
var caseTemplates = map[string]*caseTemplate{}

type caseTemplate struct {
	base  string
	files []caseFile
}

type caseFile struct {
	rel  string
	mode fs.FileMode
	body []byte
	// names says the body holds the template's directory.
	names bool
}

// gitQuotes reports whether a path can land in a git config value or subsection
// in a form other than as it is spelt.
func gitQuotes(path string) bool { return strings.ContainsAny(path, "#;\"\\ \t") }

// base is the directory the case stands in.
func (c *cloudCase) base() string { return filepath.Dir(c.dir) }

// keep records the case as it stands under key.
func (c *cloudCase) keep(key string) {
	c.t.Helper()
	if gitQuotes(c.base()) {
		return
	}
	tpl := &caseTemplate{base: c.base()}
	err := filepath.WalkDir(tpl.base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == tpl.base {
			return err
		}
		rel, err := filepath.Rel(tpl.base, p)
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		f := caseFile{rel: rel, mode: fi.Mode()}
		if fi.Mode().IsRegular() {
			if f.body, err = os.ReadFile(p); err != nil {
				return err
			}
			f.names = bytes.Contains(f.body, []byte(tpl.base))
		} else if !fi.IsDir() {
			c.t.Fatalf("a kept case cannot hold %s, of mode %v", rel, fi.Mode())
		}
		tpl.files = append(tpl.files, f)
		return nil
	})
	if err != nil {
		c.t.Fatal(err)
	}
	caseTemplates[key] = tpl
}

// restore writes the case kept under key into this case's directory and reports
// whether there was one. The index is refreshed, since a plumbing diff reads a
// new inode as an edit.
func (c *cloudCase) restore(key string) bool {
	c.t.Helper()
	tpl := caseTemplates[key]
	if tpl == nil || gitQuotes(c.base()) {
		return false
	}
	for _, f := range tpl.files {
		p := filepath.Join(c.base(), f.rel)
		var err error
		if f.mode.IsDir() {
			err = os.Mkdir(p, 0o700)
		} else {
			body := f.body
			if f.names {
				body = bytes.ReplaceAll(body, []byte(tpl.base), []byte(c.base()))
			}
			if err = os.WriteFile(p, body, 0o600); err == nil {
				err = os.Chmod(p, f.mode.Perm())
			}
		}
		if err != nil {
			c.t.Fatal(err)
		}
	}
	for i := len(tpl.files) - 1; i >= 0; i-- {
		if f := tpl.files[i]; f.mode.IsDir() {
			if err := os.Chmod(filepath.Join(c.base(), f.rel), f.mode.Perm()); err != nil {
				c.t.Fatal(err)
			}
		}
	}
	if out, err := c.run(c.dir, c.realGit, "update-index", "-q", "--refresh"); err != nil {
		c.t.Fatalf("refreshing a copied case's index: %v\n%s", err, out)
	}
	return true
}

// checkoutRoot is the root of the checkout under test, asked of git once a
// process; a test skips where there is none.
var checkoutRootOnce struct {
	root string
	err  error
	done bool
}

func checkoutRoot(t *testing.T) string {
	t.Helper()
	o := &checkoutRootOnce
	if !o.done {
		top := exec.Command("git", "rev-parse", "--show-toplevel")
		top.Env = gittest.Env(t)
		out, err := top.Output()
		o.root, o.err, o.done = strings.TrimSpace(string(out)), err, true
	}
	if o.err != nil {
		t.Skip("not in a git checkout")
	}
	return o.root
}

// TestACopiedCaseIsItsOwn: a case written out from a template pushes into its
// own remote, reaches the forge's URL through it, runs its own local abcd, and
// shows a clean tree to a plumbing diff, and nothing it does reaches another
// case's remote.
func TestACopiedCaseIsItsOwn(t *testing.T) {
	first := newForgeCase(t, "")
	second := newForgeCase(t, "")
	if first.base() == second.base() {
		t.Fatal("two cases stand in one directory")
	}
	for _, c := range []*cloudCase{first, second} {
		if got := c.git("config", "--get", "url."+c.origin+".insteadOf"); got != forgeURL {
			t.Errorf("the case's insteadOf for its own remote reads %q, want %q", got, forgeURL)
		}
		if body, err := os.ReadFile(filepath.Join(c.dir, filepath.FromSlash(localAbcdRel))); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(string(body), c.base()) {
			t.Errorf("the case's local abcd runs %q, outside its own directory %s", body, c.base())
		}
		c.git("diff-files", "--quiet")
		c.git("diff-index", "--quiet", "HEAD")
	}
	before := first.originRef("refs/heads/session")
	second.git("push", "-q", "--no-verify", "origin", "HEAD:refs/heads/session")
	if got := second.originRef("refs/heads/session"); got == "" {
		t.Error("the second case's push did not reach its own remote")
	}
	if got := first.originRef("refs/heads/session"); got != before {
		t.Errorf("the second case's push moved the first case's remote: %q, was %q", got, before)
	}
}
