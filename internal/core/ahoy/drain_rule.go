package ahoy

// The drain eligibility record `ahoy install` offers (the product thinker's
// ruling BX2 of 2026-09-29, verbatim: "the PROJECT MUST HOLD the eligibility
// decision in its own record (e.g. added at setup); drain refuses there until
// it does"). `abcd drain` reads which open issues it may take alone from an
// accepted decision record in the repository's own store (core/drainrule), and
// refuses a repository without one. Setup is where one is added:
//
//   - detection raises an optional repository gap while the store holds no
//     accepted record carrying the drain fields; a record that states the rule
//     badly is not offered a second one, since the drain names what is wrong
//     with the record it has;
//   - the offer states abcd's strict baseline and, on the person's yes, mints
//     it through the decision store's own seam as an accepted record, which is
//     committed with the repository and decides for everyone who drains it.
//
// It is asked only of a person at a terminal (the itd-131 precedent the git
// identity step set): --yes approves the category but never writes the record,
// and off a terminal (a pipe, a routine, CI) neither the category nor the offer
// is asked, because the record decides what an unattended agent may do in this
// repository and a scripted yes is not a person's. Adding the question to a
// piped answer stream would also shift every later answer onto the wrong
// question. Either way it is reported under optional_skipped. A decline writes
// nothing and records nothing, so the next install offers again. The offer
// only ever writes the baseline; loosening a floor is an edit a person makes
// to the record.

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/drainrule"
)

// DrainRule covers adding the repository's drain eligibility record.
const DrainRule GapCategory = "drain-rule"

// DrainRuleOfferGapID is the offer of the drain eligibility record.
const DrainRuleOfferGapID = "drain_rule.offered"

// drainRuleQuestionTail ends the offer, so a scripted answer stream and the
// transcript can tell it apart.
const drainRuleQuestionTail = "Add this rule to this repository's decision records?"

// detectDrainRule raises the offer while the repository holds no record of the
// rule.
func detectDrainRule(cwd string) []Gap {
	if _, err := drainrule.Load(cwd); !errors.Is(err, drainrule.ErrUnrecorded) {
		return nil
	}
	return []Gap{{
		ID: DrainRuleOfferGapID, Category: DrainRule, Scope: "repo",
		Title: "no drain eligibility record in this repository",
		Detail: "abcd drain takes an open issue alone only under this repository's own recorded rule, and refuses " +
			"to run until an accepted decision record states it.",
		FixHint: "ahoy install offers abcd's strict baseline as an accepted decision record and writes it only on " +
			"consent; --yes never accepts it. Or add the four drain_ fields to an accepted record by hand.",
		Required: false, Resolvable: true,
	}}
}

// stepDrainRule makes the offer. It never runs under --yes, and never off a
// terminal.
func (a *applyCtx) stepDrainRule() {
	if a.autoYes || !atTerminal(a.prompter) || !a.approved[DrainRule] || !a.has(DrainRuleOfferGapID) {
		return
	}
	if !a.prompter.Confirm(drainRuleQuestion()) {
		return
	}
	// Re-read at the moment of writing: a record that appeared while the
	// question was open is the repository's rule, and a second is never added.
	if _, err := drainrule.Load(a.cwd); !errors.Is(err, drainrule.ErrUnrecorded) {
		a.refuse("the drain eligibility record was not written: the repository's decision store changed while the question was open, and is left as it is.")
		return
	}
	d, err := decide.CreateStated(a.cwd, decide.Stated{
		Title:       drainrule.ProposalTitle,
		Frontmatter: drainrule.ProposalFrontmatter(),
		Body:        drainrule.ProposalBody(),
	})
	if err != nil {
		a.refuse("could not write the drain eligibility record (" + errText(err) + "); nothing was written.")
		return
	}
	a.note(writeDrainRule, filepath.Join(a.cwd, filepath.FromSlash(d.Path)))
}

// drainRuleQuestion is the reason, the rule and the question: core never
// prints, so the rule travels as the text of the confirm.
func drainRuleQuestion() string {
	b := drainrule.Baseline()
	return "abcd drain works the open issue ledger unattended, and takes an issue alone only under a rule this " +
		"repository records for itself; until it does, the drain refuses to run here. abcd's strict baseline: " +
		"take an issue only when its category is one of " + strings.Join(b.Categories, ", ") +
		", its severity is " + strings.Join(b.Severities, " or ") +
		", it carries a remedy, and nothing open blocks it; every security issue, and every major or critical " +
		"one, is a person's. Accepting writes this rule as an accepted decision record under " +
		drainrule.ADRsRelDir + "/, committed with the repository, where it can be edited; loosening it is " +
		"named on every drain. Declining writes nothing. " + drainRuleQuestionTail
}
