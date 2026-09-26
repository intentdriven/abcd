package capture

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// match.go is the ledger's half of the filing-time match
// (itd-2609212137116617): the candidate set a new capture or draft is compared
// with — every open and resolved issue and every intent, through the fields the
// configuration names — and the typed links a capture writes. The score is
// internal/core/record/match's, the one term-overlap primitive. The ledger
// gathers the intents too, because it already reads the intent store and the
// intent store does not read the ledger; the intent create takes this set
// through its Matcher.

// reLinkID is the shape of a typed link's target: an issue or an intent, the
// two families the match compares with.
var reLinkID = regexp.MustCompile(`^(iss|itd)-[0-9]+$`)

// linkFields are the typed-link keys a record may carry, in the order a
// capture writes them.
var linkFields = []match.Relation{match.Duplicates, match.Refines}

// MatchCandidates is the candidate set for a filing in the checkout at
// repoRoot, for a caller outside this package: the quoted-text intent create.
func MatchCandidates(repoRoot string, cfg match.Config) ([]match.Candidate, error) {
	rr, issuesRoot, err := resolveRoots(repoRoot, "")
	if err != nil {
		return nil, err
	}
	return matchCandidates(rr, issuesRoot, cfg)
}

// matchCandidates reads every open and resolved issue's body and every
// intent's title and press release, as far as cfg compares them. A wontfix
// record is not a candidate: it names work nobody is doing. A ledger record
// the reader skips is not one either, since its text cannot be trusted; the
// ledger's own surfaces already report it.
func matchCandidates(repoRoot, issuesRoot string, cfg match.Config) ([]match.Candidate, error) {
	var out []match.Candidate
	if cfg.Compares(match.FieldIssueBody) {
		for _, st := range []State{StateOpen, StateResolved} {
			issues, _ := scanLedger(issuesRoot, st)
			for _, iss := range issues {
				out = append(out, match.Candidate{ID: iss.ID, Text: iss.Body})
			}
		}
	}
	if cfg.Compares(match.FieldIntentTitle) || cfg.Compares(match.FieldIntentPressRelease) {
		texts, err := intent.MatchTexts(repoRoot)
		if err != nil {
			return nil, err
		}
		for _, t := range texts {
			var parts []string
			if cfg.Compares(match.FieldIntentTitle) {
				parts = append(parts, t.Title)
			}
			if cfg.Compares(match.FieldIntentPressRelease) {
				parts = append(parts, t.PressRelease)
			}
			out = append(out, match.Candidate{ID: t.ID, Text: strings.Join(parts, "\n")})
		}
	}
	return out, nil
}

// matchAndLink runs the match for a capture and writes its links into the
// record's rendered content, validating the frontmatter they join. It runs
// under the ledger lock, before the write. It never fails the capture: a
// candidate set that cannot be read, or a link the schema would refuse, comes
// back as an outcome saying why, with the content unlinked.
func matchAndLink(repoRoot, issuesRoot string, cfg match.Config, text, content string, fm map[string]any) (string, *match.Outcome) {
	if match.Short(text) {
		o := match.Rank(text, nil, cfg.Threshold)
		return content, &o
	}
	cands, err := matchCandidates(repoRoot, issuesRoot, cfg)
	if err != nil {
		o := match.Unread(cfg.Threshold, err)
		return content, &o
	}
	o := match.Rank(text, cands, cfg.Threshold)
	links := o.Links()
	if len(links) == 0 {
		return content, &o
	}
	linked := content
	withLinks := make(map[string]any, len(fm)+len(links))
	for k, v := range fm {
		withLinks[k] = v
	}
	for _, rel := range linkFields {
		ids := links[rel]
		if len(ids) == 0 {
			continue
		}
		withLinks[string(rel)] = ids
		if linked, err = setListField(linked, string(rel), ids); err != nil {
			break
		}
	}
	if err == nil {
		err = validateStrict(withLinks)
	}
	if err != nil {
		o.Skipped = fmt.Sprintf("the links could not be written (%v), so the record is filed unlinked", err)
		for i := range o.Matches {
			o.Matches[i].Linked = false
		}
		return content, &o
	}
	return linked, &o
}
