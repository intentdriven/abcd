// Package reflect is the core of the release retrospective (itd-24,
// spc-2609211751376504): the seed a retrospective interview opens from, the
// thin-answer floor the interview holds its answers to, the writer that turns
// the answers into `.abcd/development/retrospectives/<release-tag>/README.md`,
// the lessons reader and ranking a later voyage's embark shows, and the one-line
// nudge a release cut prints.
//
// The interview itself is host-run: a front door renders the Seed, asks the four
// asked sections one question at a time, and hands the answers to Write. The
// metrics section is computed from the seed, never asked. Nothing here writes to
// stdout or knows a transport.
//
// The unit is the release (adr-2609212115255771; itd-24 decisions 4 and 5): a
// retrospective starts from the intents a tag shipped, with the audit notes the
// intent auditor wrote on each and the changelog section the cut composed.
package reflect

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// RetrospectivesRelDir is the retrospective store, relative to the repository
// root: the durable record tier, a peer of the intent store (itd-24 decision 3).
const RetrospectivesRelDir = ".abcd/development/retrospectives"

// outputRel is the repo-relative path of the retrospective for tag.
func outputRel(tag string) string {
	return path.Join(RetrospectivesRelDir, tag, "README.md")
}

// validTag reports whether tag is a release tag in the shape this repository
// tags with: a leading v and a strict MAJOR.MINOR.PATCH core, no prerelease and
// no build suffix. The tag becomes a directory name, so a value outside that
// shape never reaches a path.
func validTag(tag string) error {
	if !strings.HasPrefix(tag, "v") {
		return fmt.Errorf("reflect: %q is not a release tag (want vMAJOR.MINOR.PATCH, such as v0.11.0)", tag)
	}
	v, err := launch.ParseSemver(strings.TrimPrefix(tag, "v"))
	if err != nil || v.Prerelease != "" || v.Build != "" {
		return fmt.Errorf("reflect: %q is not a release tag (want vMAJOR.MINOR.PATCH, such as v0.11.0)", tag)
	}
	return nil
}

// IsReleaseTag reports whether tag has the shape a retrospective's directory
// takes: a leading v and a strict MAJOR.MINOR.PATCH core, no prerelease and no
// build suffix. The lifeboat's packer and embarker hold the store's directory
// names to it, so the name a hostile lifeboat carries never reaches a path in
// any other shape.
func IsReleaseTag(tag string) bool { return validTag(tag) == nil }

// ErrNothingShipped is the refusal for a tag whose release shipped no intent.
// NothingShippedError wraps it, so a front door can test for it with errors.Is.
var ErrNothingShipped = errors.New("nothing shipped to reflect on")

// NothingShippedError refuses a release that shipped no intent (criterion 3).
// Its text is the criterion's own wording.
type NothingShippedError struct{ Tag string }

func (e *NothingShippedError) Error() string {
	return fmt.Sprintf("no intent shipped in `%s` — nothing shipped to reflect on", e.Tag)
}

func (e *NothingShippedError) Unwrap() error { return ErrNothingShipped }

// ErrExists is the refusal for a tag that already has a retrospective: a
// retrospective is written once and not edited after (the spec's out-of-scope
// list).
var ErrExists = errors.New("retrospective already written")

// ExistsError names the retrospective that already exists.
type ExistsError struct{ Tag, Path string }

func (e *ExistsError) Error() string {
	return fmt.Sprintf("a retrospective for %s already exists at %s; a retrospective is written once and not edited after (nothing written)", e.Tag, e.Path)
}

func (e *ExistsError) Unwrap() error { return ErrExists }

// ErrUnshippedTargets is the refusal Write returns when intents targeted at the
// release are still unshipped and the person has not said to proceed anyway
// (criterion 7).
var ErrUnshippedTargets = errors.New("intents targeted at this release are still unshipped")

// UnshippedError lists the intents whose target_release names the release and
// which have not shipped.
type UnshippedError struct {
	Tag     string
	Intents []TargetedIntent
}

func (e *UnshippedError) Error() string {
	ids := make([]string, 0, len(e.Intents))
	for _, it := range e.Intents {
		ids = append(ids, it.ID)
	}
	return fmt.Sprintf("%d intent(s) targeted at %s are still unshipped: %s; confirm to write the retrospective anyway (nothing written)",
		len(e.Intents), e.Tag, strings.Join(ids, ", "))
}

func (e *UnshippedError) Unwrap() error { return ErrUnshippedTargets }

// ErrThinAnswers is the refusal Write returns when an answer falls under the
// floor and carries no follow-up (criterion 4).
var ErrThinAnswers = errors.New("an answer is under the floor and has no follow-up")

// ThinAnswersError lists every thin answer, each with the follow-up question to
// ask before anything is written.
type ThinAnswersError struct{ Thin []ThinAnswer }

// ThinAnswer is one section whose answer is under the floor.
type ThinAnswer struct {
	Section  Section `json:"section"`
	Heading  string  `json:"heading"`
	Reason   string  `json:"reason"`
	Question string  `json:"question"`
}

func (e *ThinAnswersError) Error() string {
	parts := make([]string, 0, len(e.Thin))
	for _, t := range e.Thin {
		parts = append(parts, fmt.Sprintf("%s (%s)", t.Heading, t.Reason))
	}
	return fmt.Sprintf("thin answer(s) need one follow-up question before anything is written: %s (nothing written)",
		strings.Join(parts, "; "))
}

func (e *ThinAnswersError) Unwrap() error { return ErrThinAnswers }
