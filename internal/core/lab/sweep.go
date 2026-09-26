package lab

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Correction is one retracted claim: the literal text the lab's documents must
// no longer carry.
type Correction struct {
	N         int        `json:"n"`
	Line      int        `json:"line"`
	Pattern   string     `json:"pattern"`
	Applied   bool       `json:"applied"`
	Invalid   string     `json:"invalid,omitempty"`
	Instances []Instance `json:"instances"`
}

// Instance is one place a retracted pattern still stands.
type Instance struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Swept is the retraction sweep's result.
type Swept struct {
	ID          string       `json:"id"`
	Passed      bool         `json:"passed"`
	Corrections []Correction `json:"corrections"`
	Files       int          `json:"files"`
	NotSwept    []string     `json:"not_swept"`
	Artefact    string       `json:"artefact"`
	Finding     string       `json:"finding,omitempty"`
	Lifted      string       `json:"lifted,omitempty"`
}

// retractRe is a correction line. It starts at the margin: an indented line is
// a markdown code block, which is how the scaffold shows the shape without
// becoming a correction itself.
var retractRe = regexp.MustCompile("^[-*] retract:\\s*(.*)$")

// minPatternRunes refuses a pattern short enough to match nearly anything, so a
// sweep cannot pass or fail on noise.
const minPatternRunes = 3

// parseCorrections reads the corrections log: every `- retract: <literal>` line.
// A literal in backticks ends at the closing backtick and the rest of the line
// is its reason; otherwise the whole remainder is the literal.
func parseCorrections(doc string) []Correction {
	var out []Correction
	for i, line := range strings.Split(doc, "\n") {
		m := retractRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		c := Correction{N: len(out) + 1, Line: i + 1, Instances: []Instance{}}
		rest := strings.TrimSpace(m[1])
		if strings.HasPrefix(rest, "`") {
			if end := strings.Index(rest[1:], "`"); end >= 0 {
				rest = rest[1 : end+1]
			} else {
				c.Invalid = "an unclosed backtick"
			}
		}
		c.Pattern = rest
		if c.Invalid == "" && utf8.RuneCountInString(strings.TrimSpace(rest)) < minPatternRunes {
			c.Invalid = fmt.Sprintf("a literal shorter than %d characters matches noise", minPatternRunes)
		}
		out = append(out, c)
	}
	return out
}

// sweepSkipDirs are the lab-home directories that hold no claims: the world
// under study, the lab's HOME and binaries, and raw transcripts.
var sweepSkipDirs = map[string]bool{
	snapshotDir: true, labHomeDir: true, binDir: true, "transcripts": true,
}

// sweepSkipFiles hold the patterns by design.
var sweepSkipFiles = map[string]bool{correctionsName: true, sweepMD: true}

// isCaptureFile reports whether p is one of the five capture files of a probe
// record — the probe's input, command line, exit status and output, which are
// instrument output rather than claims. Everything else in a probe record,
// record.md's observation first, is prose the harvest cites, and is swept.
func isCaptureFile(p string) bool {
	rest, ok := strings.CutPrefix(p, probesDir+"/")
	if !ok {
		return false
	}
	name, file, ok := strings.Cut(rest, "/")
	return ok && probeNameRe.MatchString(name) && slices.Contains(probeFiles, file)
}

// unswept is the check a sweep fails when a document it could not read might
// still carry a retracted claim.
const unswept = "unswept"

// Sweep verifies every correction the lab recorded is applied: each retracted
// literal is searched for across the lab's own documents — the pattern, not
// the instance — and every place it still stands is listed. An unapplied or
// unreadable correction fails the sweep, and so does any document the sweep
// could not read while a correction is recorded (fail-closed: it is listed by
// path, never passed over). A failed sweep halts the lab and records the
// refusal as a finding (ErrHalted); a passing sweep lifts a standing sweep halt.
// The artefact and the finding name a correction by its number, never by its
// text, so neither becomes a new instance.
func Sweep(repoRoot, id string) (Swept, error) {
	l, err := open(repoRoot, id)
	if err != nil {
		return Swept{}, err
	}
	defer l.close()
	doc, err := l.readDoc(correctionsName)
	if err != nil {
		return Swept{}, err
	}
	res := Swept{ID: id, Passed: true, Corrections: parseCorrections(doc), NotSwept: []string{}, Artefact: l.display(sweepMD)}
	if res.Corrections == nil {
		res.Corrections = []Correction{}
	}

	type file struct {
		rel   string
		lines [][]byte
	}
	var files []file
	walkErr := fs.WalkDir(l.root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if sweepSkipDirs[p] {
				return fs.SkipDir
			}
			return nil
		}
		if sweepSkipFiles[p] || strings.HasPrefix(p, "state/halt-") || isCaptureFile(p) {
			return nil
		}
		if !d.Type().IsRegular() {
			// A link or a device is a document the sweep never reads.
			res.NotSwept = append(res.NotSwept, p+" (not a regular file)")
			return nil
		}
		data, rerr := fsutil.ReadGuardedInRoot(l.root, p, maxDocBytes)
		if rerr != nil {
			res.NotSwept = append(res.NotSwept, p+" ("+notSweptReason(rerr)+")")
			return nil
		}
		if bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0 {
			res.NotSwept = append(res.NotSwept, p+" (binary: a NUL byte in its first 8000 bytes)")
			return nil
		}
		files = append(files, file{rel: p, lines: bytes.Split(data, []byte("\n"))})
		return nil
	})
	if walkErr != nil {
		return Swept{}, fmt.Errorf("cannot walk the lab home: %v", redact(walkErr, l.store.home))
	}
	res.Files = len(files)

	var failed []string
	for i := range res.Corrections {
		c := &res.Corrections[i]
		if c.Invalid != "" {
			failed = append(failed, fmt.Sprintf("correction-%d", c.N))
			continue
		}
		pat := []byte(c.Pattern)
		for _, f := range files {
			for n, line := range f.lines {
				if bytes.Contains(line, pat) {
					c.Instances = append(c.Instances, Instance{File: f.rel, Line: n + 1})
				}
			}
		}
		c.Applied = len(c.Instances) == 0
		if !c.Applied {
			failed = append(failed, fmt.Sprintf("correction-%d", c.N))
		}
	}
	// Fail-closed: a document the sweep could not read may carry any correction
	// the log records, so with a correction to check it is a refusal, never a
	// pass with a footnote.
	sort.Strings(res.NotSwept)
	if len(res.Corrections) > 0 && len(res.NotSwept) > 0 {
		failed = append(failed, unswept)
	}
	res.Passed = len(failed) == 0
	if err := l.writeDoc(sweepMD, sweepDoc(l.entry, res)); err != nil {
		return Swept{}, err
	}
	if res.Passed {
		lifted, err := l.lift("sweep")
		if err != nil {
			return Swept{}, err
		}
		res.Lifted = lifted
		return res, nil
	}
	var detail strings.Builder
	detail.WriteString("The retraction sweep refused (patterns are named by number, so this entry is\nnot itself an instance):\n\n")
	for _, c := range res.Corrections {
		switch {
		case c.Invalid != "":
			fmt.Fprintf(&detail, "- correction %d (corrections.md line %d) is unreadable: %s\n", c.N, c.Line, c.Invalid)
		case !c.Applied:
			fmt.Fprintf(&detail, "- correction %d (corrections.md line %d) still stands in %d place(s): %s\n", c.N, c.Line, len(c.Instances), instanceList(c.Instances, 5))
		}
	}
	title := "Halted: the retraction sweep found unapplied corrections"
	if failed[len(failed)-1] == unswept {
		fmt.Fprintf(&detail, "- %d document(s) could not be swept, so a retracted claim may stand in them unseen: %s\n", len(res.NotSwept), strings.Join(first(res.NotSwept, 5), ", "))
		if len(failed) == 1 {
			title = "Halted: the retraction sweep could not read every document"
		}
	}
	fid, err := l.haltAndRecord("sweep", failed,
		title,
		strings.TrimRight(detail.String(), "\n"), sweepMD)
	if err != nil {
		return Swept{}, err
	}
	res.Finding = fid
	return res, fmt.Errorf("%w: the retraction sweep refused %s; recorded as %s in %s", ErrHalted, strings.Join(failed, ", "), fid, l.display(findingsName))
}

func notSweptReason(err error) string {
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return "larger than the sweep reads"
	case errors.Is(err, fsutil.ErrNotRegular):
		return "not a regular file"
	}
	return "unreadable"
}

func instanceList(is []Instance, n int) string {
	var parts []string
	for _, in := range is {
		parts = append(parts, fmt.Sprintf("%s:%d", in.File, in.Line))
	}
	return strings.Join(first(parts, n), ", ")
}

// sweepDoc renders the sweep artefact. It is the one lab file that quotes the
// patterns, and the sweep never reads it.
func sweepDoc(e Entry, res Swept) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Retraction sweep — %s\n\n", e.ID)
	verdict := "PASSED"
	if !res.Passed {
		verdict = "HALTED"
	}
	fmt.Fprintf(&b, "Run %s over %d lab document(s): **%s**.\n\n", stamp(), res.Files, verdict)
	if len(res.Corrections) == 0 {
		b.WriteString("No corrections are recorded in corrections.md.\n")
	}
	for _, c := range res.Corrections {
		switch {
		case c.Invalid != "":
			fmt.Fprintf(&b, "- correction %d (line %d): UNREADABLE, %s\n", c.N, c.Line, c.Invalid)
		case c.Applied:
			fmt.Fprintf(&b, "- correction %d (line %d) `%s`: absent, applied\n", c.N, c.Line, c.Pattern)
		default:
			fmt.Fprintf(&b, "- correction %d (line %d) `%s`: UNAPPLIED, %d instance(s)\n", c.N, c.Line, c.Pattern, len(c.Instances))
			for _, in := range c.Instances {
				fmt.Fprintf(&b, "  - %s:%d\n", in.File, in.Line)
			}
		}
	}
	if len(res.NotSwept) > 0 {
		b.WriteString("\nNot swept:\n\n")
		for _, n := range res.NotSwept {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	return b.String()
}
