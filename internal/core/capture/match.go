package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/readingitem"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// match.go is the ledger's half of the filing-time match
// (itd-2609212137116617): the candidate set a new capture or draft is compared
// with — every open and resolved issue and every intent, through the fields the
// configuration names — and the typed links a capture writes. The score is
// internal/core/record/match's, the one term-overlap primitive. The ledger
// gathers the intents too, because it already reads the intent store and the
// intent store does not read the ledger; the intent create takes this set
// through its Matcher.

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
// ledger's own surfaces already report it. A status directory that exists and
// cannot be listed is different: the candidate set is then unknown rather than
// smaller, so it is an error, which the caller reports as an unread match.
func matchCandidates(repoRoot, issuesRoot string, cfg match.Config) ([]match.Candidate, error) {
	var out []match.Candidate
	if cfg.Compares(match.FieldIssueBody) {
		for _, st := range []State{StateOpen, StateResolved} {
			issues, _, err := scanStatusDir(issuesRoot, st)
			if err != nil {
				return nil, err // readStatusDir names the directory
			}
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
// back as an outcome saying why, with the content unlinked. A candidate named
// in except is not compared.
func matchAndLink(repoRoot, issuesRoot string, cfg match.Config, text string, except []string, content string, fm map[string]any) (string, *match.Outcome) {
	if match.Short(text) {
		o := match.Rank(text, nil, cfg.Threshold)
		return content, &o
	}
	cands, err := matchCandidates(repoRoot, issuesRoot, cfg)
	if err != nil {
		o := match.Unread(cfg.Threshold, err)
		return content, &o
	}
	return linkMatches(rankExcept(text, cands, except, cfg.Threshold), content, fm, validateStrict)
}

// rankExcept ranks text against every candidate not named in except: the
// records a filer has already filed in the same pass, which are never doubles
// of each other.
func rankExcept(text string, cands []match.Candidate, except []string, threshold float64) match.Outcome {
	if len(except) > 0 {
		cands = slices.DeleteFunc(slices.Clone(cands), func(c match.Candidate) bool { return slices.Contains(except, c.ID) })
	}
	return match.Rank(text, cands, threshold)
}

// linkMatches writes an outcome's links into a record's rendered content and
// validates the frontmatter they join with the record family's own validator.
// A link the validator refuses comes back as an outcome saying so, with the
// content unlinked: the match never refuses the write.
func linkMatches(o match.Outcome, content string, fm map[string]any, validate func(map[string]any) error) (string, *match.Outcome) {
	links := o.Links()
	if len(links) == 0 {
		return content, &o
	}
	var err error
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
		err = validate(withLinks)
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

// readingMatchText is a reading item's comparable text: the pattern it names
// and the body its position declares, in the declared order. That is the
// finding in the instrument's own words. The envelope (run, manifest,
// position, regime) is left out, because every item of a run carries the same
// envelope and two findings would otherwise match on it alone.
func readingMatchText(fm map[string]any) string {
	parts := []string{asString(fm["pattern"])}
	for _, f := range issueschema.ReadingBodyFields[asString(fm["position"])] {
		parts = append(parts, asString(fm[f]))
	}
	return strings.Join(parts, "\n")
}

// readingFilingCandidates is the candidate set a stored reading finding is
// matched against (ruling DQ2b, adr-2609300821558671): the capture set, and
// every reading item already in the ledger, so a finding a later reading
// returns again is linked to the item that first carried it. A reading record
// the family's validator refuses is not a candidate, as a skipped issue is not
// one; a readings directory that cannot be listed makes the set unknown, which
// is an error the caller reports as an unread match.
func readingFilingCandidates(repoRoot, issuesRoot string, cfg match.Config) ([]match.Candidate, error) {
	out, err := matchCandidates(repoRoot, issuesRoot, cfg)
	if err != nil {
		return nil, err
	}
	readingsRoot := filepath.Join(issuesRoot, issueschema.ReadingsDir)
	if err := readingitem.RefuseSymlinkedDir(readingsRoot); err != nil {
		return nil, wrapLocatorErr(err)
	}
	runs, err := os.ReadDir(readingsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for _, run := range runs {
		if !recordid.ValidReadingRunID(run.Name()) {
			continue
		}
		// The store's one guard refuses anything at a run's name that is not a
		// real directory, a stray regular file included, so past it every entry
		// is a run directory. The refusal is deliberate: the ingest's mint meets
		// the same guard and refuses the whole ingest, and a matcher that skipped
		// what the mint refuses would be a second walk disagreeing about what the
		// ledger holds.
		runDir := filepath.Join(readingsRoot, run.Name())
		if err := readingitem.RefuseSymlinkedDir(runDir); err != nil {
			return nil, wrapLocatorErr(err)
		}
		items, err := os.ReadDir(runDir)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			id, ok := strings.CutSuffix(it.Name(), ".md")
			if !ok || !recordid.ValidReadingItemID(id) || !it.Type().IsRegular() {
				continue
			}
			content, err := readRecordGuarded(filepath.Join(runDir, it.Name()))
			if err != nil {
				continue
			}
			fm, _, err := parseFrontmatterAndBody(content)
			if err != nil || validateReadingStrict(fm) != nil {
				continue
			}
			out = append(out, match.Candidate{ID: id, Text: readingMatchText(fm)})
		}
	}
	return out, nil
}
