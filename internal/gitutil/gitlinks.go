package gitutil

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// maxGitlinks bounds how many gitlinks Gitlinks returns: a repository holds a
// handful of submodules, and a listing past this is not one to pin whole.
const maxGitlinks = 4096

// Gitlinks is the path of every gitlink (mode 160000, a submodule) in root's
// index, relative to the working tree's top and slash-separated. The index
// listing is streamed, keeping only the gitlinks, so a large index costs time,
// not memory.
func Gitlinks(root string) ([]string, error) {
	cmd := isolatedGit(root, "ls-files", "--stage", "-z", "--full-name")
	e := &capWriter{remaining: 4096}
	cmd.Stderr = e
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var links []string
	var parseErr error
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, 0); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF && len(data) > 0 {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	for sc.Scan() {
		meta, p, ok := strings.Cut(sc.Text(), "\t")
		if !ok || !strings.HasPrefix(meta, "160000 ") {
			continue
		}
		if len(links) == maxGitlinks {
			parseErr = fmt.Errorf("the index holds more than %d gitlinks", maxGitlinks)
			break
		}
		links = append(links, p)
	}
	if parseErr == nil {
		parseErr = sc.Err()
	}
	if parseErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("git ls-files: %w", parseErr)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git ls-files: %w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	return links, nil
}
