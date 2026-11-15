package gitutil

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Gitlinks is the path of every gitlink (mode 160000, a submodule) in root's
// index, relative to the working tree's top and slash-separated. The index
// listing is streamed, keeping only the gitlinks, so a large index costs time,
// not memory beyond the gitlinks themselves; no count of them is refused.
func Gitlinks(root string) ([]string, error) {
	return gitlinks(isolatedGit(root, gitlinksArgs...))
}

// GitlinksWithIndex is Gitlinks of the scratch index at index (an absolute
// path, as RunWithIndex takes one).
func GitlinksWithIndex(root, index string) ([]string, error) {
	cmd, err := indexCommand(root, index, gitlinksArgs)
	if err != nil {
		return nil, err
	}
	return gitlinks(cmd)
}

// gitlinksArgs is the index listing Gitlinks reads.
var gitlinksArgs = []string{"ls-files", "--stage", "-z", "--full-name"}

// gitlinks is Gitlinks and GitlinksWithIndex, streaming cmd's listing.
func gitlinks(cmd *exec.Cmd) ([]string, error) {
	var links []string
	err := streamRecords(cmd, "ls-files", func(rec string) error {
		meta, p, ok := strings.Cut(rec, "\t")
		if ok && strings.HasPrefix(meta, "160000 ") {
			links = append(links, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return links, nil
}

// MovedGitlinks is the set of gitlinks of root's index (of the scratch index
// at index, an absolute path, when it is not "") whose working tree no
// longer holds the commit the index records: a submodule checked out at
// another commit, a path gone, or one that holds something else. It reads
// `git diff-files --ignore-submodules=dirty -z` over the whole tree,
// streamed, and keeps the rows whose index side is a gitlink, so neither the
// size of the tree nor the number of its changes bounds it.
//
// With --ignore-submodules=dirty git starts nothing inside a submodule: it
// compares the commit checked out there with the index's. A file whose stat
// cannot tell it apart from the index (racily clean) is hashed, which runs
// its content filter, so every filter the repository configures is blanked
// on the command line and its required switch turned off (filterBlanks):
// what such a file hashes to changes only the file's own row, which is not
// kept.
func MovedGitlinks(root, index string) (map[string]bool, error) {
	blanks, err := filterBlanks(root)
	if err != nil {
		return nil, err
	}
	args := append(blanks, "diff-files", "--ignore-submodules=dirty", "-z")
	var cmd *exec.Cmd
	if index == "" {
		cmd = isolatedGit(root, args...)
	} else if cmd, err = indexCommand(root, index, args); err != nil {
		return nil, err
	}
	moved := map[string]bool{}
	// Each row is a header (":<index mode> <tree mode> <ids> <status>") and
	// then its path, or its two paths for a rename or a copy.
	var paths int
	var keep bool
	err = streamRecords(cmd, "diff-files", func(rec string) error {
		if paths > 0 {
			paths--
			if keep {
				moved[rec] = true
				keep = false
			}
			return nil
		}
		f := strings.Fields(strings.TrimPrefix(rec, ":"))
		if !strings.HasPrefix(rec, ":") || len(f) < 5 {
			return fmt.Errorf("an unexpected row %q", rec)
		}
		paths = 1
		if s := f[len(f)-1]; s != "" && (s[0] == 'R' || s[0] == 'C') {
			paths = 2
		}
		keep = f[0] == "160000"
		return nil
	})
	if err != nil {
		return nil, err
	}
	return moved, nil
}

// filterBlanks is FilterOverrides with each filter's required switch turned
// off too, for a read that must neither run a filter nor fail on one the
// repository marks required: the bytes of the file are hashed as they stand.
func filterBlanks(root string) ([]string, error) {
	names, err := configNames(root, "filter")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range names {
		for _, v := range []string{"clean=", "smudge=", "process=", "required=false"} {
			out = append(out, "-c", "filter."+name+"."+v)
		}
	}
	return out, nil
}

// streamRecords runs cmd and gives each NUL-terminated record of its output
// to each as git writes it, keeping none, its stderr bounded into the error.
// An error from each stops git and is the call's.
func streamRecords(cmd *exec.Cmd, verb string, each func(rec string) error) error {
	e := &capWriter{remaining: 4096}
	cmd.Stderr = e
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
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
		if parseErr = each(sc.Text()); parseErr != nil {
			break
		}
	}
	if parseErr == nil {
		parseErr = sc.Err()
	}
	if parseErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("git %s: %w", verb, parseErr)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git %s: %w (stderr: %q)", verb, err, strings.TrimSpace(string(e.buf)))
	}
	return nil
}
