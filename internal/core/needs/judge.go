package needs

// judge.go is the one judge of a program's copy. Rule.Admit is the runner's
// admit rule, extracted from internal/core/runner/proc.go so the runner calls
// it rather than a copy: an absolute path whose file, whose directory and the
// directory it resolves into no one but its owner (or, on darwin, the admin
// group) can write, and that lies within none of the guarded trees. Rule.Judge
// is Admit with the checks a person's registration is held to besides
// (adr-2611041815445203, rule 3): the path's own last part is the tool's
// name, every link on the way and the file it ends at are held to the same
// rule, each at its real location as well as as written, every directory above any of them is one no one else can write (a
// sticky directory admitted over an entry this account or root owns), and
// each of those directories and the file is owned by this account or root.
//
// Like the rest of the package it is a filter against mistakes (adr-42): it
// reads the file system's metadata at the moment it is asked, runs nothing,
// and says nothing about what happens to a file after it has answered.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Kind is why the judge refused a copy.
type Kind int

const (
	// Relative: the path is not absolute.
	Relative Kind = iota + 1
	// Unresolved: the path, a link on its way or a directory above it does not
	// exist, or its links do not end.
	Unresolved
	// Unreachable: a link leads through a directory this account cannot enter.
	Unreachable
	// Unexaminable: something on the way could not be examined.
	Unexaminable
	// Writable: the file, or a directory it is reached through, is one group
	// or other can write.
	Writable
	// Inside: the path, a link on its way or its target lies within a guarded
	// tree: the repository a role runs in, or the checkout it belongs to.
	Inside
	// WrongName: the path's own last part is not the tool's name.
	WrongName
	// ForeignOwner: the file, or a directory it is reached through, is owned
	// by an account other than this one or root.
	ForeignOwner
	// NotRegular: what the path resolves to is not a regular file.
	NotRegular
)

var kindNames = map[Kind]string{
	Relative: "relative", Unresolved: "unresolved", Unreachable: "unreachable", Unexaminable: "unexaminable",
	Writable: "writable", Inside: "inside", WrongName: "wrong-name", ForeignOwner: "foreign-owner", NotRegular: "not-regular",
}

func (k Kind) String() string {
	if s, ok := kindNames[k]; ok {
		return s
	}
	return fmt.Sprintf("kind(%d)", int(k))
}

// Refusal is the judge's answer for a copy it refuses: why, and the path on
// the way that it refused, which may be the copy, a link, a directory above
// either, or the file a link ends at.
type Refusal struct {
	Kind Kind
	Path string
}

func (r *Refusal) Error() string {
	switch r.Kind {
	case Relative:
		return fmt.Sprintf("%q is a relative path; a copy is named by its absolute path", r.Path)
	case Unresolved:
		return fmt.Sprintf("%q does not resolve to a file", r.Path)
	case Unreachable:
		return fmt.Sprintf("%q leads through a directory this account cannot enter", r.Path)
	case Writable:
		return fmt.Sprintf("%q is one group or other can write, so whoever can write it chooses what runs; chmod go-w it", r.Path)
	case Inside:
		return fmt.Sprintf("%q lies inside a repository or checkout; a program there is repository content", r.Path)
	case WrongName:
		return fmt.Sprintf("%q is not named for the tool it would be", r.Path)
	case ForeignOwner:
		return fmt.Sprintf("%q is owned by an account other than this one or root, which chooses what runs", r.Path)
	case NotRegular:
		return fmt.Sprintf("%q is not a regular file", r.Path)
	}
	return fmt.Sprintf("%q could not be examined", r.Path)
}

// Rule is what a copy is judged against.
type Rule struct {
	// Guards are the trees a program must not lie within: the repository a
	// role runs in and the checkout it belongs to. "" is skipped.
	Guards []string
	// ReachedThrough is a further path the program was reached through, held
	// to the guards as the path is; "" for none.
	ReachedThrough string
	// Group reads the owning group of a stat result; nil reads it from the
	// result itself. The runner's tests narrow it.
	Group func(os.FileInfo) (uint32, bool)
}

// maxLinks bounds the links Judge follows, as the system's own limit does.
const maxLinks = 40

// ownerOf is the owner of the file fi describes at p. A variable because a
// second uid is not something a test can make.
var ownerOf = func(_ string, fi os.FileInfo) (uint32, error) { return fsutil.InfoOwnerUID(fi) }

// Admit is the runner's admit rule: it returns the file p resolves to, or a
// *Refusal when p is not absolute, does not resolve, when it, the directory p
// reaches it through or the directory it resolves into is one group or other
// can write (darwin's admin group aside, adminGroupWritableOnly), or when p,
// the file or ReachedThrough lies within a guard, lexically or after symlink
// resolution. Its checks and their order are the runner's; it adds two
// refusals after them: a chain of p's links that does not end at the file the
// system resolves p to (Unexaminable), and a step of p's links, or
// ReachedThrough, whose real location (its directory with every link
// resolved, and its own name) lies within a guard.
func (r Rule) Admit(p string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", &Refusal{Kind: Relative, Path: p}
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", &Refusal{Kind: followed(err), Path: p}
	}
	// Whoever can write the binary, the directory PATH reaches it through, or
	// the directory it resolves into chooses what runs.
	for _, c := range []string{resolved, filepath.Dir(resolved), filepath.Dir(filepath.Clean(p))} {
		fi, err := os.Stat(c)
		if errors.Is(err, fs.ErrNotExist) {
			return "", &Refusal{Kind: Unresolved, Path: c}
		}
		if err != nil {
			return "", &Refusal{Kind: Unexaminable, Path: c}
		}
		if r.writable(fi) {
			return "", &Refusal{Kind: Writable, Path: c}
		}
	}
	if c := r.within(filepath.Clean(p), resolved, r.ReachedThrough); c != "" {
		return "", &Refusal{Kind: Inside, Path: c}
	}
	// A step reached through a directory link lies where that link leads, so
	// each step is held to the guards at its real location too: a link the
	// repository holds is repository content wherever it points.
	chain, err := linkChain(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	// The chain is followed as the system follows it, or not at all: a
	// relative target such as d/../x, where d is a directory link, is read
	// lexically here and by the system through d, and the steps held to the
	// guards would not be the route that runs.
	if last, err := filepath.EvalSymlinks(chain[len(chain)-1]); err != nil || last != resolved {
		return "", &Refusal{Kind: Unexaminable, Path: filepath.Clean(p)}
	}
	reals, err := realSteps(chain)
	if err != nil {
		return "", err
	}
	if r.ReachedThrough != "" {
		if dir, err := filepath.EvalSymlinks(filepath.Dir(r.ReachedThrough)); err == nil {
			reals = append(reals, filepath.Join(dir, filepath.Base(r.ReachedThrough)))
		}
	}
	if c := r.within(reals...); c != "" {
		return "", &Refusal{Kind: Inside, Path: c}
	}
	return resolved, nil
}

// Verdict is what Judge admitted: the path as given, cleaned; the file it
// resolves to; and each step its links pass through after the path, the last
// being the file itself, empty when the path is no link.
type Verdict struct {
	Path   string
	Target string
	Links  []string
}

// SameName reports whether the file the path resolves to carries the path's
// own name. A link to a program of another name, as a version manager's
// launcher is, is admitted, and is never asked its version.
func (v Verdict) SameName() bool {
	return filepath.Base(v.Target) == filepath.Base(v.Path)
}

// Judge is Admit with the checks a registration of name is held to: p's own
// last part is name; every link on the way is followed and each step held to
// the guards, as written and at its real location; every directory above p, above each step and above the
// directory each step resolves into is one no one else can write, a sticky
// directory admitted over an entry this account or root owns; each of those
// directories and the file is owned by this account or root; and the file is
// a regular one.
func (r Rule) Judge(name, p string) (Verdict, error) {
	if !filepath.IsAbs(p) {
		return Verdict{}, &Refusal{Kind: Relative, Path: p}
	}
	clean := filepath.Clean(p)
	if name == "" || filepath.Base(clean) != name {
		return Verdict{}, &Refusal{Kind: WrongName, Path: clean}
	}
	resolved, err := r.Admit(clean)
	if err != nil {
		return Verdict{}, err
	}
	chain, err := linkChain(clean)
	if err != nil {
		return Verdict{}, err
	}
	reals, err := realSteps(chain)
	if err != nil {
		return Verdict{}, err
	}
	if c := r.within(append(slices.Clone(chain), reals...)...); c != "" {
		return Verdict{}, &Refusal{Kind: Inside, Path: c}
	}
	seen := map[string]bool{}
	for i, step := range chain {
		for _, q := range []string{step, reals[i]} {
			if err := r.ancestors(q, seen); err != nil {
				return Verdict{}, err
			}
		}
	}
	// The chain is followed as the system follows it, or not at all.
	if last, err := filepath.EvalSymlinks(chain[len(chain)-1]); err != nil || last != resolved {
		return Verdict{}, &Refusal{Kind: Unexaminable, Path: clean}
	}
	fi, err := os.Stat(resolved)
	if err != nil {
		return Verdict{}, &Refusal{Kind: Unexaminable, Path: resolved}
	}
	if !fi.Mode().IsRegular() {
		return Verdict{}, &Refusal{Kind: NotRegular, Path: resolved}
	}
	if !held(resolved, fi) {
		return Verdict{}, &Refusal{Kind: ForeignOwner, Path: resolved}
	}
	return Verdict{Path: clean, Target: resolved, Links: chain[1:]}, nil
}

// linkChain is p and each path its links lead to, in order, the last being no
// link. A relative target is read against the directory its link really lies
// in, as the system reads it.
func linkChain(p string) ([]string, error) {
	chain := []string{p}
	cur := p
	for range maxLinks {
		fi, err := os.Lstat(cur)
		if err != nil {
			return nil, &Refusal{Kind: followed(err), Path: cur}
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			return chain, nil
		}
		target, err := os.Readlink(cur)
		if err != nil {
			return nil, &Refusal{Kind: Unexaminable, Path: cur}
		}
		if !filepath.IsAbs(target) {
			dir, err := filepath.EvalSymlinks(filepath.Dir(cur))
			if err != nil {
				return nil, &Refusal{Kind: followed(err), Path: filepath.Dir(cur)}
			}
			target = filepath.Join(dir, target)
		}
		cur = filepath.Clean(target)
		chain = append(chain, cur)
	}
	return nil, &Refusal{Kind: Unresolved, Path: p}
}

// realSteps is where each step of chain really lies: its directory with every
// link resolved, joined with its own name. A step reached through a directory
// link lies where that link leads, which its lexical form does not show.
func realSteps(chain []string) ([]string, error) {
	reals := make([]string, 0, len(chain))
	for _, step := range chain {
		dir, err := filepath.EvalSymlinks(filepath.Dir(step))
		if err != nil {
			return nil, &Refusal{Kind: followed(err), Path: filepath.Dir(step)}
		}
		reals = append(reals, filepath.Join(dir, filepath.Base(step)))
	}
	return reals, nil
}

// ancestors judges every directory above p, each over the entry beneath it.
func (r Rule) ancestors(p string, seen map[string]bool) error {
	for child := p; ; {
		dir := filepath.Dir(child)
		if dir == child {
			return nil
		}
		if key := dir + "\x00" + child; !seen[key] {
			seen[key] = true
			if err := r.judgeDir(dir, child); err != nil {
				return err
			}
		}
		child = dir
	}
}

// judgeDir refuses dir when someone other than this account or root can
// replace child within it, or owns it.
func (r Rule) judgeDir(dir, child string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return &Refusal{Kind: examined(err), Path: dir}
	}
	if r.writable(fi) && !stickyOver(fi, child) {
		return &Refusal{Kind: Writable, Path: dir}
	}
	if !held(dir, fi) {
		return &Refusal{Kind: ForeignOwner, Path: dir}
	}
	return nil
}

// stickyOver reports whether dir, described by fi, is sticky and child is an
// entry this account or root owns: then no one else may rename or remove it,
// whoever can write dir.
func stickyOver(fi os.FileInfo, child string) bool {
	if fi.Mode()&os.ModeSticky == 0 {
		return false
	}
	cfi, err := os.Lstat(child)
	return err == nil && held(child, cfi)
}

// held reports whether p, described by fi, is owned by this account or root.
// An owner that cannot be read is not held.
func held(p string, fi os.FileInfo) bool {
	uid, err := ownerOf(p, fi)
	return err == nil && (uid == uint32(os.Getuid()) || uid == 0)
}

// within is the first of paths that lies within a guard, lexically or after
// symlink resolution of the guard, "" for none.
func (r Rule) within(paths ...string) string {
	var guards []string
	for _, g := range r.Guards {
		if g == "" {
			continue
		}
		guards = append(guards, filepath.Clean(g))
		if e, err := filepath.EvalSymlinks(g); err == nil {
			guards = append(guards, e)
		}
	}
	fold := fsutil.CaseFoldingFS()
	for _, g := range guards {
		for _, c := range paths {
			if c != "" && fsutil.PathWithin(c, g, fold) {
				return c
			}
		}
	}
	return ""
}

// writable reports whether group or other can write what fi describes,
// darwin's admin group aside.
func (r Rule) writable(fi os.FileInfo) bool {
	group := r.Group
	if group == nil {
		group = statGroup
	}
	return fsutil.WritableByOthers(fi) && !adminGroupWritableOnly(fi, group)
}

// adminGroupWritableOnly reports whether fi's only write bit beyond its
// owner's is the group's, and that group is darwin's admin group (gid 80),
// on darwin only. Homebrew installs /opt/homebrew/bin as root- or
// user-owned, group admin, mode 0775, so a harness installed through it is
// reached through a group-writable directory. Members of darwin's admin
// group can sudo by default, so that write grants them nothing they do not
// hold, and the directory is admitted. gid 0 is never admitted: being in
// Linux's root group, or darwin's wheel, does not by itself let a member act
// as root. Other-writable is never admitted, whatever the group, and neither
// is any other group, gid 80 off darwin, nor a group that cannot be read.
func adminGroupWritableOnly(fi os.FileInfo, group func(os.FileInfo) (uint32, bool)) bool {
	if fi.Mode().Perm()&0o002 != 0 {
		return false
	}
	gid, ok := group(fi)
	if !ok {
		return false
	}
	return runtime.GOOS == "darwin" && gid == 80
}

// statGroup reads the owning group of a stat result; ok is false when the
// platform does not report one.
func statGroup(fi os.FileInfo) (uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Gid, true
}

// followed is the kind of a failure to follow a path: a directory this
// account cannot enter is Unreachable, anything else Unresolved.
func followed(err error) Kind {
	if errors.Is(err, fs.ErrPermission) {
		return Unreachable
	}
	return Unresolved
}

// examined is the kind of a failure to examine a directory on the way.
func examined(err error) Kind {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Unresolved
	case errors.Is(err, fs.ErrPermission):
		return Unreachable
	}
	return Unexaminable
}
