// Package interview holds what every abcd interview writes and reads
// (spc-2610030911534855, "The answers record and the answered in field"): the
// answers record, written through one writer whichever front door asked, and
// the answers file a run off a terminal is answered from.
//
// A record holds, per question, the question as asked (the sanitised
// question.Ask the front door drew or wrote), the value chosen, the note if
// any (sanitised by Write), and where the answer was given: Terminal or
// Claude Code, and nothing else (itd-2610030810370060 decision 4). The time
// is in the file's name and never in its content, so two runs given the same
// answers write the same bytes, whichever front door asked them.
//
// The package is a library: it returns values and writes files, and never
// prints.
package interview

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the answers record's and the answers file's schema.
const SchemaVersion = 1

// Where an answer was given: the only two values answered_in takes.
const (
	// Terminal is an answer given in a plain Terminal: drawn, read from the
	// numbered list, or replayed from an answers file a person or a script
	// wrote there.
	Terminal = "Terminal"
	// ClaudeCode is an answer given through the host's question tool and
	// relayed to the binary in an answers file.
	ClaudeCode = "Claude Code"
)

// ValidAnsweredIn reports whether s is one of the two places an answer is
// given.
func ValidAnsweredIn(s string) bool { return s == Terminal || s == ClaudeCode }

// The targets a setup record names: the repository the install ran in, or the
// machine, for the machine-wide part of setup.
const (
	TargetRepository = "repository"
	TargetMachine    = "machine"
)

// Record is one interview's answers record.
type Record struct {
	SchemaVersion int      `json:"schema_version"`
	Interview     string   `json:"interview"`
	Target        string   `json:"target"`
	Answers       []Answer `json:"answers"`
}

// Answer is one question's entry in the record: the question as asked, the
// value chosen, the note if any, and where it was answered.
type Answer struct {
	ID         string       `json:"id"`
	Ask        question.Ask `json:"ask"`
	Value      string       `json:"value"`
	Note       string       `json:"note,omitempty"`
	AnsweredIn string       `json:"answered_in"`
}

// Place is where a record is written: exactly one of a repository's root,
// whose local tier holds it, and the user's home, whose ~/.abcd holds it.
type Place struct {
	Repo string
	Home string
}

// RecordsRel is the records' directory below a repository's root.
const RecordsRel = ".abcd/.work.local/interviews"

// localTierRel is the repository's local tier, which must already stand: the
// writer creates the records' directory inside it, never the tier itself.
const localTierRel = ".abcd/.work.local"

// homeRecordsLeaf is the records' directory below abcd's user-level home,
// reached through abcdhome, the one place the home's name is written.
const homeRecordsLeaf = "interviews"

// ErrNoLocalTier is Write's refusal for a repository with no local tier: a
// record is never the thing that creates .abcd/ in a repository.
var ErrNoLocalTier = errors.New("this repository has no local tier (.abcd/.work.local/) to keep the answers record in")

// nameRe is what an interview's name may be, since it is part of a file name.
var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

// stampLayout is the stamp in a record's file name.
const stampLayout = "20060102T150405Z"

// Write writes rec at p as <interview>-<stamp>.json, the stamp being at in
// UTC, and returns the path written. A record never replaces another: a
// second one in the same second takes the next free suffix. The repository's
// records' directory is made inside its existing local tier, each level a
// real directory; the home's through the guarded home-scope maker the other
// ~/.abcd stores use, so a symlinked level is refused rather than written
// through. The file is 0600.
func Write(p Place, rec Record, at time.Time) (string, error) {
	if (p.Repo == "") == (p.Home == "") {
		return "", errors.New("interview: a record is written to a repository or to the home, exactly one")
	}
	if !nameRe.MatchString(rec.Interview) {
		return "", fmt.Errorf("interview: %q is not an interview name", rec.Interview)
	}
	// The answers are copied before the note is sanitised, so the caller's
	// record is left as it was.
	rec.Answers = append([]Answer(nil), rec.Answers...)
	for i, a := range rec.Answers {
		if !ValidAnsweredIn(a.AnsweredIn) {
			return "", fmt.Errorf("interview: answered_in %q for %s is neither %q nor %q", a.AnsweredIn, a.ID, Terminal, ClaudeCode)
		}
		// A note is data (an answers file names it), so it is kept with
		// every terminal-control and hidden rune made visible, as the
		// question it answers is.
		rec.Answers[i].Note = termsafe.Sanitize(a.Note)
	}
	rec.SchemaVersion = SchemaVersion
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')

	var root *os.Root
	var dir string
	if p.Repo != "" {
		ok, err := fsutil.ProbeRealDirAll(p.Repo, localTierRel)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrNoLocalTier
		}
		if err := fsutil.EnsureRealDirAll(p.Repo, RecordsRel, 0o700); err != nil {
			return "", err
		}
		r, err := os.OpenRoot(p.Repo)
		if err != nil {
			return "", err
		}
		root, dir = r, filepath.Join(p.Repo, filepath.FromSlash(RecordsRel))
	} else {
		r, err := fsutil.EnsureHomeScope(p.Home, abcdhome.Rel(homeRecordsLeaf), 0o700)
		if err != nil {
			return "", err
		}
		root, dir = r, abcdhome.Path(p.Home, homeRecordsLeaf)
	}
	defer root.Close()

	stem := rec.Interview + "-" + at.UTC().Format(stampLayout)
	for n := 1; n <= 99; n++ {
		name := stem + ".json"
		if n > 1 {
			name = fmt.Sprintf("%s-%d.json", stem, n)
		}
		rel := name
		if p.Repo != "" {
			rel = path.Join(RecordsRel, name)
		}
		err := fsutil.CreateExclusiveIn(root, rel, body, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, name), nil
	}
	return "", fmt.Errorf("interview: 99 records named %s already stand", stem)
}

// Answers is an answers file: the answers a run off a terminal is given, one
// shape for every interview. A fixed interview matches them by question id.
type Answers struct {
	SchemaVersion int          `json:"schema_version"`
	Interview     string       `json:"interview"`
	Answers       []FileAnswer `json:"answers"`
}

// FileAnswer is one answer in an answers file. AnsweredIn is where it was
// given; empty, the verb's --answered-in flag stamps it.
type FileAnswer struct {
	ID         string `json:"id"`
	Value      string `json:"value"`
	Note       string `json:"note,omitempty"`
	AnsweredIn string `json:"answered_in,omitempty"`
}

// MaxAnswersBytes caps an answers file.
const MaxAnswersBytes = 1 << 20

// ParseAnswers reads an answers file strictly (jsonstrict): an unknown key or
// a repeated one refuses, so a mistyped key never drops an answer without a
// word. It refuses another schema, a file for another interview, an answer
// with no id or no value, an id answered twice, and an answered_in that is
// neither Terminal nor Claude Code.
func ParseAnswers(data []byte, interview string) (Answers, error) {
	if len(data) > MaxAnswersBytes {
		return Answers{}, fmt.Errorf("the answers file exceeds %d bytes", MaxAnswersBytes)
	}
	var a Answers
	if err := jsonstrict.Decode(data, &a); err != nil {
		return Answers{}, fmt.Errorf("the answers file: %w", err)
	}
	if a.SchemaVersion != SchemaVersion {
		return Answers{}, fmt.Errorf("the answers file has schema_version %d; this abcd reads %d", a.SchemaVersion, SchemaVersion)
	}
	if a.Interview != interview {
		return Answers{}, fmt.Errorf("the answers file is for the %q interview, not %q", a.Interview, interview)
	}
	seen := map[string]bool{}
	for i, e := range a.Answers {
		id := strings.TrimSpace(e.ID)
		switch {
		case id == "":
			return Answers{}, fmt.Errorf("answer %d has no id", i+1)
		case seen[id]:
			return Answers{}, fmt.Errorf("%q is answered twice", id)
		case strings.TrimSpace(e.Value) == "":
			return Answers{}, fmt.Errorf("%q has no value", id)
		case e.AnsweredIn != "" && !ValidAnsweredIn(e.AnsweredIn):
			return Answers{}, fmt.Errorf("%q: answered_in %q is neither %q nor %q", id, e.AnsweredIn, Terminal, ClaudeCode)
		}
		seen[id] = true
	}
	return a, nil
}

// Find returns the answer the file gives the question id.
func (a Answers) Find(id string) (FileAnswer, bool) {
	for _, e := range a.Answers {
		if strings.TrimSpace(e.ID) == id {
			return e, true
		}
	}
	return FileAnswer{}, false
}
