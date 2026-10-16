package gitutil

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// BatchObject is one object `git cat-file --batch` answered for: its type and
// content, or Missing when the repository holds no object of that name.
type BatchObject struct {
	Name    string
	Type    string
	Content []byte
	Missing bool
}

// CatFileBatch reads objects through ONE isolated `git cat-file --batch`, in
// the order asked, so a caller reading many blobs starts one process rather
// than one per blob. Every name must be a full object name, so nothing but
// names reaches git's input. maxBytes caps git's whole answer, headers
// included: an answer past it is an error, never a truncated one, so a caller
// bounds it by what it already knows of each object's size. An answer whose
// shape git does not document is an error too.
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
	out, overflowed, err := runBoundedCmd(cmd, maxBytes)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git cat-file --batch: output exceeded the %d-byte cap; the answer would be truncated", maxBytes)
	}
	res := make([]BatchObject, 0, len(objects))
	for _, name := range objects {
		nl := bytes.IndexByte(out, '\n')
		if nl < 0 {
			return nil, fmt.Errorf("git cat-file --batch: no answer for %s", name)
		}
		header := strings.Fields(string(out[:nl]))
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
		res = append(res, BatchObject{Name: name, Type: header[1], Content: out[:size:size]})
		out = out[size+1:]
	}
	if len(out) != 0 {
		return nil, fmt.Errorf("git cat-file --batch: %d byte(s) past the last answer", len(out))
	}
	return res, nil
}
