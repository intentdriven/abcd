package history

// The session-separation check (adr-2609021016275803, spc-2609020626045177).
//
// Brief invariant 15 states that no session holds both a reading and the
// ledger. The per-run context stamp is how a transcript shows which contexts
// its session held, and Capture records every stamp a transcript carried as
// metadata. This file reads that metadata and nothing else: it is the consumer
// the invariant enumerates as session-separation evidence, metadata only, never
// bodies.
//
// It reports three things and never a fourth. A violation is named. The
// property held for what was seen is said with the runs it saw. And a store
// that holds no transcript, or none carrying any stamp, is UNOBSERVED with the
// reason — never clean, because a check that saw nothing and a check that could
// see nothing must not produce the same artefact (adr-56).

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/sessionkind"
)

// Violation is one retained transcript carrying both the reading stamp and the
// scribe stamp of one run, named by its session, its record file and the run.
// File is the record's basename, never a path.
type Violation struct {
	SessionID string `json:"session_id"`
	File      string `json:"file"`
	Run       string `json:"run"`
}

// SeparationReport is what the check saw.
type SeparationReport struct {
	// Transcripts is how many records the store holds, and Stamped how many of
	// them carry at least one stamp.
	Transcripts int `json:"transcripts"`
	Stamped     int `json:"stamped"`
	// Runs is every run any stamp named, sorted, so a clean report says what it
	// examined.
	Runs       []string    `json:"runs"`
	Violations []Violation `json:"violations"`
	// Unobserved is true when there was nothing to examine, and Reason says why.
	Unobserved bool   `json:"unobserved"`
	Reason     string `json:"reason,omitempty"`
}

// The two unobserved reasons.
const (
	reasonNoTranscript = "the store holds no retained transcript, so whether any session held both a reading " +
		"and the ledger cannot be observed here; the scribe definition's protocol remains the gate"
	reasonNoStamp = "no retained transcript carries a context stamp, so no session is known to have held a " +
		"reading bundle or a scribe context; a host that assembles context before anything is retained is " +
		"outside this check's reach, and there the scribe definition's protocol remains the gate"
)

// SessionSeparation runs the check over this repository's lane of the store.
// It reads record metadata through List and never opens a body for it.
func SessionSeparation(repoRoot, rootSHA string) (SeparationReport, error) {
	records, err := List(repoRoot, rootSHA)
	if err != nil {
		return SeparationReport{}, err
	}
	return separationOf(records), nil
}

// separationOf is the judgement, over metadata already read.
func separationOf(records []Record) SeparationReport {
	rep := SeparationReport{Transcripts: len(records), Runs: []string{}, Violations: []Violation{}}
	runs := map[string]bool{}
	for _, r := range records {
		kinds := map[string]map[sessionkind.Kind]bool{}
		for _, s := range r.ContextStamps {
			p, ok := sessionkind.Parse(s)
			if !ok {
				continue
			}
			if kinds[p.Run] == nil {
				kinds[p.Run] = map[sessionkind.Kind]bool{}
			}
			kinds[p.Run][p.Kind] = true
			runs[p.Run] = true
		}
		if len(kinds) > 0 {
			rep.Stamped++
		}
		held := make([]string, 0, len(kinds))
		for run, k := range kinds {
			if k[sessionkind.Reading] && k[sessionkind.Scribe] {
				held = append(held, run)
			}
		}
		sort.Strings(held)
		for _, run := range held {
			rep.Violations = append(rep.Violations, Violation{
				SessionID: r.SessionID, File: filepath.Base(r.Path), Run: run,
			})
		}
	}
	for run := range runs {
		rep.Runs = append(rep.Runs, run)
	}
	sort.Strings(rep.Runs)
	sort.SliceStable(rep.Violations, func(i, j int) bool {
		if rep.Violations[i].Run != rep.Violations[j].Run {
			return rep.Violations[i].Run < rep.Violations[j].Run
		}
		return rep.Violations[i].File < rep.Violations[j].File
	})
	switch {
	case rep.Transcripts == 0:
		rep.Unobserved, rep.Reason = true, reasonNoTranscript
	case rep.Stamped == 0:
		rep.Unobserved, rep.Reason = true, reasonNoStamp
	}
	return rep
}

// Summary is the report's one line, the same wherever it is rendered: the
// violations by name, or the property held with the runs seen, or the
// unobserved reason. It carries no path.
func (r SeparationReport) Summary() string {
	switch {
	case len(r.Violations) > 0:
		named := make([]string, 0, len(r.Violations))
		for _, v := range r.Violations {
			named = append(named, fmt.Sprintf("%s (session %s) holds both stamps of %s", v.File, v.SessionID, v.Run))
		}
		return fmt.Sprintf("session separation BREACHED: %d retained transcript(s) carry a reading stamp and a "+
			"scribe stamp of one run: %s", len(r.Violations), strings.Join(named, "; "))
	case r.Unobserved:
		return "session separation unobserved: " + r.Reason
	default:
		return fmt.Sprintf("session separation held for what was seen: no retained transcript carries two "+
			"stamps of one run (%d of %d transcript(s) stamped; runs seen: %s)",
			r.Stamped, r.Transcripts, strings.Join(r.Runs, ", "))
	}
}
