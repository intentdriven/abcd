package implement

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/reading"
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

// ReadingCorpus is what "a lane that touches the reading corpus" means, derived
// from the checkout's committed preset file rather than restated here: the union
// of every position's object.paths, plus the preset file itself, whose
// recalibration serialises such lanes (iss-2609211105023379). A lane that edits
// one of those paths moves what a cold reading is handed, and so the measured
// windows the first session recalibrates. An entry names a file or a directory;
// a directory covers everything beneath it.
//
// It is read through reading.LoadPresets, the loader the reading verb itself
// uses, so the two can never disagree about what the corpus is, and it refuses
// what that loader refuses: an untracked, symlinked or unparseable preset file.
// A corpus that cannot be derived is an error, never an empty list — the caller
// fails closed on it.
func ReadingCorpus(repoRoot string) ([]string, error) {
	if repoRoot == "" {
		return nil, errors.New("no checkout to read the reading presets from")
	}
	pf, err := reading.LoadPresets(repoRoot)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{reading.PresetConfigPath: true}
	for _, e := range pf.Positions {
		for _, p := range e.Object.Paths {
			set[path.Clean(p)] = true
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// TouchesReadingCorpus returns the first path that falls in corpus — equal to
// an entry, or beneath one — or "" when none does.
func TouchesReadingCorpus(corpus, paths []string) string {
	for _, p := range paths {
		p = path.Clean(p)
		for _, c := range corpus {
			if p == c || strings.HasPrefix(p, c+"/") {
				return p
			}
		}
	}
	return ""
}

// corpusBound applies the reading-corpus bound to a second session's declared
// paths: nil when none is in the corpus, and a logged refusal when one is — or
// when the corpus cannot be derived, since then nothing can say the lane stays
// clear of it. A lane that declares no paths asks no corpus question.
func (r *Run) corpusBound(session string, fields map[string]any, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	corpus, err := ReadingCorpus(r.RepoRoot)
	if err != nil {
		f := map[string]any{"error": err.Error()}
		for k, v := range fields {
			f[k] = v
		}
		return r.refuseLogged(session, "reading_corpus_unknown", f,
			fmt.Sprintf("the reading corpus cannot be derived (%v), so a second session's lane that declares paths is refused until it can", err))
	}
	if hit := TouchesReadingCorpus(corpus, paths); hit != "" {
		f := map[string]any{"path": hit}
		for k, v := range fields {
			f[k] = v
		}
		return r.refuseLogged(session, "reading_corpus_lane", f,
			hit+" is in the reading corpus; a lane that touches it is the first session's")
	}
	return nil
}

// Verdict is Check's answer.
type Verdict struct {
	Session string `json:"session"`
	Role    Role   `json:"role"`
	Step    Step   `json:"step"`
	Mode    Mode   `json:"mode,omitempty"`
	Allowed bool   `json:"allowed"`
	// Ceiling is the session's own agent ceiling as it joined with it, zero when
	// it stated none: reported with every verdict so the session about to act
	// sees the limit it keeps (see MaxCeiling).
	Ceiling int `json:"ceiling,omitempty"`
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
		out = Verdict{Session: session, Role: s.Role, Step: step, Allowed: true, Ceiling: s.Ceiling}
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
			if err := r.corpusBound(session, map[string]any{"step": string(step)}, paths); err != nil {
				out.Allowed = false
				return err
			}
		}
		return nil
	})
	return out, err
}
