package gitutil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// streamFields runs cmd and hands fn each field of its output, a field being
// what lies between two sep bytes, as git writes it: the output is read as
// it comes, so its total length is never capped, while one field longer than
// maxField is an error, never a field cut short. fn returning an error stops
// the read. It returns how many fields fn was handed, so a caller can tell a
// git that refused before answering from one that failed partway.
//
// A git that fails, and a field past the bound, are errors carrying git's
// bounded stderr; git is killed on any early stop, so it is never left
// blocked writing to a pipe no one reads.
func streamFields(cmd *exec.Cmd, sep byte, maxField int, fn func(string) error) (int, error) {
	stderr := &capWriter{remaining: 4096}
	cmd.Stderr = stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 4096), maxField)
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, sep); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF && len(data) > 0 {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	n := 0
	var stop error
	for sc.Scan() {
		n++
		if stop = fn(sc.Text()); stop != nil {
			break
		}
	}
	if stop == nil {
		stop = sc.Err()
	}
	if errors.Is(stop, bufio.ErrTooLong) {
		stop = fmt.Errorf("git %s: a field of its output is longer than the %d-byte bound", strings.Join(cmd.Args[1:], " "), maxField)
	}
	if stop != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return n, stop
	}
	if err := cmd.Wait(); err != nil {
		return n, fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(stderr.buf)))
	}
	return n, nil
}
