package testreuse

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// outcome is what go test said of one package across a stage's lines.
type outcome struct {
	cached, ran, failed bool
}

// Tally reads go test's result lines.
type Tally struct {
	mu   sync.Mutex
	pkgs map[string]*outcome
}

var resultLine = regexp.MustCompile(`^(ok|FAIL)\s+(\S+)\s+(.*)$`)

// Line records one line of go test's output.
func (t *Tally) Line(line string) {
	m := resultLine.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
	if m == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pkgs == nil {
		t.pkgs = map[string]*outcome{}
	}
	notests := strings.Contains(m[3], "[no tests to run]")
	if notests && m[1] == "ok" {
		return
	}
	o := t.pkgs[m[2]]
	if o == nil {
		o = &outcome{}
		t.pkgs[m[2]] = o
	}
	switch {
	case m[1] == "FAIL":
		o.failed = true
	case strings.Contains(m[3], "(cached)"):
		o.cached = true
	default:
		o.ran = true
	}
}

// Counts is a stage's reuse: Reused of Packages had a result replayed, Whole of
// them with no test run at all, and Ran names the packages any test ran in.
type Counts struct {
	Packages, Reused, Whole int
	Ran                     []string
}

// Counts sums the lines read so far.
func (t *Tally) Counts() Counts {
	t.mu.Lock()
	defer t.mu.Unlock()
	var c Counts
	for name, o := range t.pkgs {
		c.Packages++
		if o.cached && !o.failed {
			c.Reused++
			if !o.ran {
				c.Whole++
			}
		}
		if o.ran || o.failed {
			c.Ran = append(c.Ran, name)
		}
	}
	slices.Sort(c.Ran)
	return c
}

// lineWriter hands each whole line to a Tally and writes it on.
type lineWriter struct {
	tally *Tally
	out   io.Writer
	buf   bytes.Buffer
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		i := bytes.IndexByte(w.buf.Bytes(), '\n')
		if i < 0 {
			return len(p), nil
		}
		line := w.buf.Next(i + 1)
		w.tally.Line(string(line))
		if _, err := w.out.Write(line); err != nil {
			return len(p), err
		}
	}
}

// flush passes on a last line with no newline.
func (w *lineWriter) flush() {
	if w.buf.Len() > 0 {
		w.tally.Line(w.buf.String())
		_, _ = w.out.Write(w.buf.Bytes())
		w.buf.Reset()
	}
}

// reportDir is where a checkout keeps its stages' reports.
const reportDir = ".abcd/.work.local/preflight-reuse"

// StageReport is one go test line's record for the receipt: which commit and
// tree it checked, how, and what it reused.
type StageReport struct {
	Commit, Tree, Stage, Mode string
	Counts                    Counts
	Passed                    bool
}

// write records the report under the checkout's local tier, named for the
// commit, the stage and what the command selects, never its -p or -timeout,
// which move with the load: a second pass of one stage has its own file and the
// same pass run again replaces its own.
func (r StageReport) write(worktree string, selects []string) error {
	dir := filepath.Join(worktree, filepath.FromSlash(reportDir), r.Commit)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(strings.Join(selects, "\x00")))
	result := "fail"
	if r.Passed {
		result = "pass"
	}
	body := fmt.Sprintf("commit %s\ntree %s\nstage %s\nmode %s\nreused %d\npackages %d\nresult %s\n",
		r.Commit, r.Tree, r.Stage, r.Mode, r.Counts.Reused, r.Counts.Packages, result)
	name := r.Stage + "-" + hex.EncodeToString(sum[:6])
	tmp := filepath.Join(dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name))
}
