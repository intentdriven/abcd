package gitutil

import (
	"fmt"
	"strconv"
	"strings"
)

// BatchObject is one object `git cat-file --batch` answered for: its type and
// content, or Missing when the repository holds no object of that name.
// Content is a slice of the batch's one answer, never a copy of it.
type BatchObject struct {
	Name    string
	Type    string
	Content string
	Missing bool
}

// CatFileBatch reads objects through ONE isolated `git cat-file --batch`, in
// the order asked, so a caller reading many blobs starts one process rather
// than one per blob. Every name must be a full object name, so nothing but
// names reaches git's input. maxBytes caps git's whole answer, headers
// included: an answer past it is an error, never a truncated one, so a caller
// bounds it by what it already knows of each object's size. An answer whose
// shape git does not document is an error too.
//
// The answer is read into one buffer sized to maxBytes up front, so it is
// never grown by copying, and each object's content is a slice of it: what
// the batch holds in memory is about maxBytes, however many objects it reads.
func CatFileBatch(root string, objects []string, maxBytes int) ([]BatchObject, error) {
	if len(objects) == 0 {
		return nil, nil
	}
	var in strings.Builder
	for _, o := range objects {
		if !IsFullSHA(o) {
			return nil, fmt.Errorf("git cat-file --batch: %q is not a full object name", o)
		}
		in.WriteString(o)
		in.WriteByte('\n')
	}
	cmd := isolatedGit(root, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(in.String())
	w := &capBuilder{remaining: maxBytes}
	w.b.Grow(maxBytes)
	e := &capWriter{remaining: 4096}
	cmd.Stdout, cmd.Stderr = w, e
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	if w.overflowed {
		return nil, fmt.Errorf("git cat-file --batch: output exceeded the %d-byte cap; the answer would be truncated", maxBytes)
	}
	out := w.b.String()
	res := make([]BatchObject, 0, len(objects))
	for _, name := range objects {
		nl := strings.IndexByte(out, '\n')
		if nl < 0 {
			return nil, fmt.Errorf("git cat-file --batch: no answer for %s", name)
		}
		header := strings.Fields(out[:nl])
		out = out[nl+1:]
		if len(header) == 2 && header[0] == name && header[1] == "missing" {
			res = append(res, BatchObject{Name: name, Missing: true})
			continue
		}
		if len(header) != 3 || header[0] != name {
			return nil, fmt.Errorf("git cat-file --batch: answered %q for %s, a header it does not document", header, name)
		}
		size, err := strconv.Atoi(header[2])
		if err != nil || size < 0 || size+1 > len(out) || out[size] != '\n' {
			return nil, fmt.Errorf("git cat-file --batch: the answer for %s does not hold the %s bytes its header names", name, header[2])
		}
		res = append(res, BatchObject{Name: name, Type: header[1], Content: out[:size]})
		out = out[size+1:]
	}
	if len(out) != 0 {
		return nil, fmt.Errorf("git cat-file --batch: %d byte(s) past the last answer", len(out))
	}
	return res, nil
}

// capBuilder is capWriter over a strings.Builder, whose String is the bytes
// written with no copy: what it drops past its cap marks it overflowed.
type capBuilder struct {
	b          strings.Builder
	remaining  int
	overflowed bool
}

func (w *capBuilder) Write(p []byte) (int, error) {
	n := len(p)
	if n > w.remaining {
		n, w.overflowed = w.remaining, true
	}
	if n > 0 {
		w.b.Write(p[:n])
		w.remaining -= n
	}
	return len(p), nil
}

// TreePathTypes asks ONE isolated `git cat-file --batch-check` for the type
// of what commit's tree holds at each path, one `<commit>:<path>` line on
// stdin per path, and answers in the order asked: "blob" (a file or a
// symbolic link, which git stores as a blob), "tree", "commit" for a
// submodule's gitlink, or "" for a path the tree does not hold, a path
// beneath a gitlink or through a symbolic link among them. Git answers a
// gitlink "<sha> commit" when this repository holds the commit it names, and
// "<sha> submodule", ignoring the format asked, when it does not (git 2.52);
// both are "commit" here, never an error. Each path is resolved on its own,
// so a directory asked beside a path beneath it is still answered as a tree,
// and the answer is one short line per path, whatever the trees beneath hold.
// commit must be a full object name, and a path holding a newline or a NUL,
// or none at all, never reaches git.
func TreePathTypes(root, commit string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if !IsFullSHA(commit) {
		return nil, fmt.Errorf("git cat-file --batch-check: %q is not a full object name", commit)
	}
	var in strings.Builder
	limit := 0
	specs := make([]string, len(paths))
	for i, p := range paths {
		if p == "" || strings.ContainsAny(p, "\n\x00") {
			return nil, fmt.Errorf("git cat-file --batch-check: path %q cannot be asked on one line", p)
		}
		specs[i] = commit + ":" + p
		in.WriteString(specs[i])
		in.WriteByte('\n')
		// A found object answers "<name> <type>", a missing one echoes the
		// line asked with " missing".
		limit += len(specs[i]) + 80
	}
	cmd := isolatedGit(root, "cat-file", "--batch-check=%(objectname) %(objecttype)")
	cmd.Stdin = strings.NewReader(in.String())
	out, overflowed, err := runBoundedCmd(cmd, limit)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git cat-file --batch-check: output exceeded the %d-byte cap", limit)
	}
	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(lines) != len(specs) {
		return nil, fmt.Errorf("git cat-file --batch-check: %d answer(s) for %d path(s)", len(lines), len(specs))
	}
	types := make([]string, len(specs))
	for i, line := range lines {
		if line == specs[i]+" missing" {
			continue
		}
		name, kind, ok := strings.Cut(line, " ")
		if kind == "submodule" {
			kind = "commit"
		}
		if !ok || !IsFullSHA(name) || (kind != "blob" && kind != "tree" && kind != "commit" && kind != "tag") {
			return nil, fmt.Errorf("git cat-file --batch-check: answered %q for %s, an answer it does not document", line, paths[i])
		}
		types[i] = kind
	}
	return types, nil
}
