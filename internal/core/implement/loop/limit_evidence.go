package loop

// limit_evidence.go is what a drive leaves behind about a limit (iss-2611050753557564).
// How a headless agent behaves at a limit has not been observed, so the first limit
// a run meets, and the first of each other kind of failure that may be one, each leave
// a file in the local tier: what the harness showed, its exit, the reset read, what the loop
// decided, each probe's answer and when the run resumed. The file is how the person
// learns what really happens. Nothing reads it back, and a failure to keep it never
// stops the run: it says so and goes on.
//
// The runner has already dropped every identifier and reduced free text to its
// length (runner.HarnessSeen). What is left passes through the scanner's redactor,
// an address masked on top, so the file identifies no account, and only then is it
// cut, so the redactor sees a whole token. A file stops taking that detail past its
// size and goes on taking the answers, and only the newest files of a kind are kept.
// All of it goes through one os.Root, as the launch ledger's does, so a link swapped
// in carries no write out of the folder.

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// evidenceDir is where the files go, beside the drive's own log.
const evidenceDir = ".abcd/.work.local/logs"

// What a file keeps, and how many are kept.
const (
	// evidenceMaxBytes is the size past which a file takes no more of what the
	// harness showed, and evidenceMaxLine and evidenceMaxStderr bound one line of
	// it.
	evidenceMaxBytes  = 64 << 10
	evidenceMaxLine   = 2000
	evidenceMaxStderr = 400
	// evidenceKeepFiles is how many files of a kind are kept across runs.
	evidenceKeepFiles = 20
)

// evidenceEntry is one line of an evidence file.
type evidenceEntry struct {
	Event    string `json:"event"`
	At       string `json:"at"`
	Role     string `json:"role,omitempty"`
	Runner   string `json:"runner,omitempty"`
	Response string `json:"response,omitempty"`
	// ExitCode and SilentForSeconds come from Seen, with Shape and Stderr.
	ExitCode         *int     `json:"exit_code,omitempty"`
	SilentForSeconds *int     `json:"silent_for_seconds,omitempty"`
	Shape            []string `json:"shape,omitempty"`
	Stderr           string   `json:"stderr,omitempty"`
	ResetRead        string   `json:"reset_read,omitempty"`
	Decided          string   `json:"decided,omitempty"`
	NextLook         string   `json:"next_look,omitempty"`
	Answer           string   `json:"answer,omitempty"`
	// Seen is what the harness showed, as yet unredacted.
	Seen *runner.HarnessSeen `json:"-"`
}

// driveEvidence keeps a run's evidence files: one of each kind, the latest begun
// open until the run moves again.
type driveEvidence struct {
	root string
	say  func(string)
	seen map[string]bool
	open string
	size int
}

// newDriveEvidence is a run's evidence, writing under root.
func newDriveEvidence(root string, say func(string)) *driveEvidence {
	return &driveEvidence{root: root, say: say, seen: map[string]bool{}}
}

// folder opens the folder the files live in, making it if it is missing.
func (e *driveEvidence) folder() (*os.Root, error) {
	if err := fsutil.EnsureRealDirAll(e.root, evidenceDir, 0o700); err != nil {
		return nil, err
	}
	return os.OpenRoot(filepath.Join(e.root, filepath.FromSlash(evidenceDir)))
}

// begin writes the first entry of kind this run has met, and leaves that file open
// for what follows. Any later one of the kind adds nothing.
func (e *driveEvidence) begin(kind string, entry evidenceEntry, now time.Time) {
	if e.seen[kind] {
		return
	}
	e.seen[kind] = true
	dir, err := e.folder()
	if err != nil {
		e.say("the evidence of this " + kind + " could not be kept under " + evidenceDir + ": " + fsutil.RedactHome(err.Error()))
		return
	}
	defer dir.Close()
	prefix := "drive-" + kind + "-"
	name := prefix + now.UTC().Format("20060102T150405Z") + ".jsonl"
	f, err := dir.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		e.say("the evidence of this " + kind + " could not be kept under " + evidenceDir + ": " + fsutil.RedactHome(err.Error()))
		return
	}
	_ = f.Close()
	pruneEvidence(dir, prefix)
	e.open, e.size = name, 0
	e.say("the evidence of this " + kind + " is kept at " + evidenceDir + "/" + name)
	e.add(entry, now)
}

// pruneEvidence removes the oldest files of a kind past those kept. Their names sort
// by when they were begun.
func pruneEvidence(dir *os.Root, prefix string) {
	entries, err := fs.ReadDir(dir.FS(), ".")
	if err != nil {
		return
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".jsonl") {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	for len(names) > evidenceKeepFiles {
		_ = dir.Remove(names[0])
		names = names[1:]
	}
}

// add appends an entry to the file open now, if one is.
func (e *driveEvidence) add(entry evidenceEntry, now time.Time) {
	if e.open == "" {
		return
	}
	entry.At = now.UTC().Format(time.RFC3339)
	if s := entry.Seen; s != nil && e.size < evidenceMaxBytes {
		exit, silent := s.ExitCode, int(s.SilentFor.Round(time.Second)/time.Second)
		entry.ExitCode, entry.SilentForSeconds = &exit, &silent
		entry.Stderr = cutEvidence(e.redact(s.Stderr), evidenceMaxStderr)
		for _, line := range s.Shape {
			entry.Shape = append(entry.Shape, cutEvidence(e.redact(line), evidenceMaxLine))
		}
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	dir, err := e.folder()
	if err != nil {
		return
	}
	defer dir.Close()
	f, err := dir.OpenFile(e.open, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	if n, err := f.Write(append(raw, '\n')); err == nil {
		e.size += n
	}
}

// resumed closes the open file, the run having moved again.
func (e *driveEvidence) resumed(now time.Time) {
	if e.open == "" {
		return
	}
	e.add(evidenceEntry{Event: "resumed"}, now)
	e.open = ""
}

// evidenceMail is an address, which names a person whatever else is around it.
var evidenceMail = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+`)

// redact is a harness's own words as the file keeps them.
func (e *driveEvidence) redact(s string) string {
	if s == "" {
		return ""
	}
	return evidenceMail.ReplaceAllString(fsutil.RedactHome(scanner.RedactRefusal(e.root, s)), "[address]")
}

// cutEvidence is s at most limit bytes long, cut on a character, and said to be cut.
func cutEvidence(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit] + "..."
}
