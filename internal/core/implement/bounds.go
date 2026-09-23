package implement

import (
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Step is a point in a run where the second session's bounds are checked before
// the session acts. A lane is opened by a claim (Claim applies the lane bounds
// itself); Check is for the steps that are not claims, and for a lane whose
// files are only known once it has been built.
type Step string

// The steps.
const (
	StepLane    Step = "lane"
	StepRelease Step = "release"
	StepReview  Step = "review"
	StepAudit   Step = "audit"
	StepLand    Step = "land"
)

// Steps returns the closed step vocabulary.
func Steps() []Step { return []Step{StepLane, StepRelease, StepReview, StepAudit, StepLand} }

// ParseStep accepts exactly one of Steps.
func ParseStep(s string) (Step, error) {
	for _, st := range Steps() {
		if string(st) == s {
			return st, nil
		}
	}
	return "", refusal("unknown step %q (one of: lane, release, review, audit, land)", s)
}

// readingCorpus is what "a lane that touches the reading corpus" means: the
// preset file whose recalibration serialises such lanes (iss-2609211105023379),
// and the packages and command pages the run's routing names as growing the
// cold-reading corpus. A prefix ending in "/" is a directory; anything else is
// one file.
//
// It is a stated list rather than a derivation from the preset file, because the
// preset's object is most of the tree and every lane would match it; the list is
// the set the run has found to move the measured windows, and it moves when that
// finding does.
var readingCorpus = []string{
	".abcd/config/reading-presets.json",
	"internal/core/capture/",
	"internal/core/grounds/",
	"internal/core/intent/",
	"internal/core/issueschema/",
	"internal/core/lint/",
	"internal/core/provenance/",
	"commands/capture.md",
	"commands/intent.md",
	"commands/reading.md",
}

// ReadingCorpus returns the list TouchesReadingCorpus checks against.
func ReadingCorpus() []string { return append([]string(nil), readingCorpus...) }

// TouchesReadingCorpus returns the first path that falls in the reading corpus,
// or "" when none does.
func TouchesReadingCorpus(paths []string) string {
	for _, p := range paths {
		for _, c := range readingCorpus {
			if p == c || (strings.HasSuffix(c, "/") && strings.HasPrefix(p, c)) {
				return p
			}
		}
	}
	return ""
}

// Verdict is Check's answer.
type Verdict struct {
	Session string `json:"session"`
	Role    Role   `json:"role"`
	Step    Step   `json:"step"`
	Mode    Mode   `json:"mode,omitempty"`
	Allowed bool   `json:"allowed"`
}

// Check says whether a session may take a step, and logs the refusal when it may
// not. The first session may take every step. The second is refused:
//
//   - the release step, always — only the first session cuts a release;
//   - a lane in a split-roles window, where the second only reviews, audits and
//     lands;
//   - a lane whose paths reach the reading corpus.
//
// Review, audit and land are open to both. A refusal is an ErrRefused-classed
// error and a `refusal` line in the log; an allowed step writes nothing.
func (r *Run) Check(session string, step Step, paths []string) (Verdict, error) {
	if _, err := ParseStep(string(step)); err != nil {
		return Verdict{}, err
	}
	for _, p := range paths {
		if !fsutil.ValidRelPath(p) {
			return Verdict{}, refusal("path %q is not a repository-relative path", p)
		}
	}
	var out Verdict
	err := r.withLock(func() error {
		s, err := r.requireSession(session)
		if err != nil {
			return err
		}
		out = Verdict{Session: session, Role: s.Role, Step: step, Allowed: true}
		w, ok, err := r.CurrentMode()
		if err != nil {
			return err
		}
		if ok {
			out.Mode = w.Mode
		}
		if s.Role != RoleSecond {
			return nil
		}
		switch step {
		case StepRelease:
			out.Allowed = false
			return r.refuseLogged(session, "second_session_release", map[string]any{"step": string(step)},
				"only the first session cuts a release; leave the release step to it")
		case StepLane:
			if ok && w.Mode == ModeSplitRoles {
				out.Allowed = false
				return r.refuseLogged(session, "split_roles_second_builds_nothing", map[string]any{"step": string(step)},
					"in a split-roles window the second session reviews, audits and lands; it opens no lane")
			}
			if hit := TouchesReadingCorpus(paths); hit != "" {
				out.Allowed = false
				return r.refuseLogged(session, "reading_corpus_lane", map[string]any{"step": string(step), "path": hit},
					hit+" is in the reading corpus; a lane that touches it is the first session's")
			}
		}
		return nil
	})
	return out, err
}
