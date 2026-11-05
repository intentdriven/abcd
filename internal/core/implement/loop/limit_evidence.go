package loop

// limit_evidence.go is what a drive leaves behind about a limit (iss-2611050753557564).
// How a headless agent behaves at a subscription limit has not been observed, so
// the first limit a run meets, and the first other failure that may be one, each
// leave a file in the local tier: what the harness showed, its exit, the reset
// read, what the loop decided, each probe's answer and when the run resumed. The
// file is how the person learns what really happens. Nothing reads it back, and a
// failure to keep it never stops the run: it says so and goes on.
//
// The runner has already dropped every identifier and reduced free text to its
// length (runner.HarnessSeen). What is left passes through the scanner's redactor,
// and an address is masked on top, so the file identifies no account.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// evidenceDir is where the files go, beside the drive's own log.
const evidenceDir = ".abcd/.work.local/logs"

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
}

// newDriveEvidence is a run's evidence, writing under root.
func newDriveEvidence(root string, say func(string)) *driveEvidence {
	return &driveEvidence{root: root, say: say, seen: map[string]bool{}}
}

// begin writes the first entry of kind this run has met, and leaves that file open
// for what follows. Any later one of the kind adds nothing.
func (e *driveEvidence) begin(kind string, entry evidenceEntry, now time.Time) {
	if e.seen[kind] {
		return
	}
	e.seen[kind] = true
	if err := fsutil.EnsureRealDirAll(e.root, evidenceDir, 0o700); err != nil {
		e.say("the evidence of this " + kind + " could not be kept under " + evidenceDir + ": " + fsutil.RedactHome(err.Error()))
		return
	}
	rel := evidenceDir + "/drive-" + kind + "-" + now.UTC().Format("20060102T150405Z") + ".jsonl"
	f, err := os.OpenFile(filepath.Join(e.root, filepath.FromSlash(rel)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		e.say("the evidence of this " + kind + " could not be kept under " + evidenceDir + ": " + fsutil.RedactHome(err.Error()))
		return
	}
	_ = f.Close()
	e.open = rel
	e.say("the evidence of this " + kind + " is kept at " + rel)
	e.add(entry, now)
}

// add appends an entry to the file open now, if one is.
func (e *driveEvidence) add(entry evidenceEntry, now time.Time) {
	if e.open == "" {
		return
	}
	entry.At = now.UTC().Format(time.RFC3339)
	if s := entry.Seen; s != nil {
		exit, silent := s.ExitCode, int(s.SilentFor.Round(time.Second)/time.Second)
		entry.ExitCode, entry.SilentForSeconds = &exit, &silent
		entry.Stderr = e.redact(s.Stderr)
		for _, line := range s.Shape {
			entry.Shape = append(entry.Shape, e.redact(line))
		}
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(e.root, filepath.FromSlash(e.open)), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
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
