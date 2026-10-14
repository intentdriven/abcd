// Package web is the brief session's page (itd-2610091016262335,
// spc-2610141709430146 step 3): the static files `abcd session page` writes,
// embedded in the binary. The page is plain HTML, one ES module and one
// stylesheet, plus the vendored, provenance-pinned editor bundle
// (adr-2610141949110653). It connects only to the relay the person names, to
// api.github.com with the person's own token, and to github.com for sign-in.
//
// The HTML is written twice, as index.html and as 404.html, byte for byte the
// same: a static host such as GitHub Pages serves 404.html for a path it has
// no file for, which is how /<owner>/<repo>/brief/ reaches the page. Every
// asset the HTML loads is named relative to that depth (../../../), so the
// page works under a base path too.
package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/intentdriven/abcd/internal/fsutil"
)

//go:embed index.html session.js session.css vendor/collab.js vendor/LICENSES.txt
var assets embed.FS

// Generator is the line the page's HTML carries, which is how a later run
// knows a directory holding the page's files as one it wrote.
const Generator = `<meta name="generator" content="abcd session page">`

// File is one file the page is made of.
type File struct {
	Path string // slash-separated, relative to the output directory
	Data []byte
}

// sources maps each written path to the embedded asset it is a copy of.
var sources = []struct{ path, asset string }{
	{"index.html", "index.html"},
	{"404.html", "index.html"},
	{"session.js", "session.js"},
	{"session.css", "session.css"},
	{"vendor/collab.js", "vendor/collab.js"},
	{"vendor/LICENSES.txt", "vendor/LICENSES.txt"},
}

// Files is every file the page is made of, in the order they are written.
func Files() []File {
	out := make([]File, 0, len(sources))
	for _, s := range sources {
		data, err := assets.ReadFile(s.asset)
		if err != nil {
			// The embed directive names every asset; a missing one is a
			// build that cannot happen.
			panic(err)
		}
		out = append(out, File{Path: s.path, Data: data})
	}
	return out
}

// Result is what Write did.
type Result struct {
	Out   string   `json:"out"`
	Files []string `json:"files"`
	Bytes int64    `json:"bytes"`
	// Replaced is true when the directory held an earlier copy of the page.
	Replaced bool `json:"replaced"`
}

// Refusal is a directory Write will not write into; nothing was written.
type Refusal struct{ Reason string }

func (r *Refusal) Error() string { return r.Reason }

// Write writes the page's files into out and nothing else. out may be absent
// or empty; a directory holding anything but the page's own files, or holding
// them without the page's own index.html, is refused untouched. Nothing is
// ever removed: an earlier copy of the page is overwritten file by file.
func Write(out string) (Result, error) {
	if out == "" {
		return Result{}, &Refusal{Reason: "name the directory to write the page into with --out"}
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return Result{}, err
	}
	replaced, err := inspect(abs)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return Result{}, err
	}
	if !fsutil.IsRealDir(abs) {
		return Result{}, &Refusal{Reason: abs + " is not a real directory; the page is written only into one"}
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()
	// Asked again through the handle the writes go through, so a directory
	// filled between the first look and the first write is still refused.
	if _, err := inspectRoot(root, abs); err != nil {
		return Result{}, err
	}
	res := Result{Out: abs, Replaced: replaced}
	for _, f := range Files() {
		if err := fsutil.WriteFileAtomicInRoot(root, f.Path, f.Data, 0o644); err != nil {
			return res, err
		}
		res.Files = append(res.Files, f.Path)
		res.Bytes += int64(len(f.Data))
	}
	return res, nil
}

// inspect judges out before anything is created: absent or empty is fine, a
// symlink or a file is refused, and a directory is judged by inspectRoot.
func inspect(abs string) (bool, error) {
	fi, err := os.Lstat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, err
	case fi.Mode()&os.ModeSymlink != 0:
		return false, &Refusal{Reason: abs + " is a symlink; the page is written only into a real directory"}
	case !fi.IsDir():
		return false, &Refusal{Reason: abs + " is not a directory"}
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return false, err
	}
	defer root.Close()
	return inspectRoot(root, abs)
}

// inspectRoot reports whether the directory holds an earlier copy of the
// page, and refuses one holding anything else.
func inspectRoot(root *os.Root, abs string) (bool, error) {
	ours := map[string]bool{"vendor": true}
	for _, s := range sources {
		ours[s.path] = true
	}
	var found []string
	foreign := ""
	err := fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		want := ours[p]
		isDir := p == "vendor"
		switch {
		case !want, d.Type()&fs.ModeSymlink != 0, d.IsDir() != isDir, !d.IsDir() && !d.Type().IsRegular():
			foreign = p
			return fs.SkipAll
		}
		found = append(found, p)
		return nil
	})
	if err != nil {
		return false, err
	}
	if foreign != "" {
		return false, &Refusal{Reason: fmt.Sprintf("%s holds %s, which the session page does not write; point --out at an empty directory or one an earlier `abcd session page` wrote", abs, foreign)}
	}
	if len(found) == 0 {
		return false, nil
	}
	head, err := fsutil.ReadGuardedInRoot(root, "index.html", 64<<10)
	if err != nil || !bytes.Contains(head, []byte(Generator)) {
		sort.Strings(found)
		return false, &Refusal{Reason: fmt.Sprintf("%s holds %s but no index.html the session page wrote, so it cannot be told apart from a directory it did not write", abs, found[0])}
	}
	return true, nil
}
