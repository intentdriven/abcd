// Package drainrule reads the drained repository's own record of which open
// issues an unattended drain may take alone (itd-82 decision 4; the product
// thinker's rulings BX2 and H11 of 2026-09-29).
//
// BX2, verbatim: "the PROJECT MUST HOLD the eligibility decision in its own
// record (e.g. added at setup); drain refuses there until it does." H11,
// verbatim: "MAY LOOSEN abcd's floors (a project may let drain take
// major/critical and security issues). NOTE for the lane: make a loosened floor
// loud (drain --dry-run and the drain start name every floor the project
// loosened), and keep abcd's own repository at the stricter default."
//
// The record is an accepted decision record in the repository's own store
// (.abcd/development/decisions/adrs/) whose frontmatter carries four fields:
//
//	drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]
//	drain_severities: [nitpick, minor]
//	drain_security: handback
//	drain_remedy: required
//
// The baseline is abcd's strict rule, bundled here; a record is measured
// against it, and every floor it loosens is named in Rule.Loosened. A record
// may narrow the fixable set and the severities, may widen the severities to
// major and critical, and may take security issues. It may not widen the
// categories past the fixable set (the other categories are decisions by kind),
// and it may not drop the remedy: the remedy is the brief a lane works from.
// Anything else it says, or fails to say, refuses: a partial or malformed
// record never falls back to a looser rule, or to a stricter one in silence.
//
// The package is a leaf over the frontmatter scanner and the issue schema, so
// the capture package (which applies the rule) and the setup offer (which
// writes the baseline as a record) both read the one definition.
package drainrule

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ADRsRelDir is the decision store the rule is read from, repo-relative and
// slash-separated. A test pins it to core/decide's own constant.
const ADRsRelDir = ".abcd/development/decisions/adrs"

// The record's four fields.
const (
	FieldCategories = "drain_categories"
	FieldSeverities = "drain_severities"
	FieldSecurity   = "drain_security"
	FieldRemedy     = "drain_remedy"
)

// fieldPrefix marks a frontmatter key as the drain rule's: a record carrying
// any key with it is a drain rule record, and every such key must be one of
// the four, so a misspelt field refuses rather than being ignored.
const fieldPrefix = "drain_"

// The values drain_security and drain_remedy take.
const (
	// SecurityHandBack: a security issue is always a person's (the baseline).
	SecurityHandBack = "handback"
	// SecurityTake: a drain may take a security issue that passes every other
	// rule, a loosened floor (H11).
	SecurityTake = "take"
	// RemedyRequired: a drain takes no issue without a remedy. It is the only
	// value: the remedy is the brief the issue-keyed lane works from
	// (itd-2609201916151817 decision 10), so a project cannot drop it.
	RemedyRequired = "required"
)

// SecurityCategory is the issue category drain_security decides.
const SecurityCategory = "security"

// baselineCategories is the fixable set in the order a drain takes it (itd-82
// decision 7): clean-ups and text first, behaviour changes last. A security
// issue a project lets through comes after all of them.
var baselineCategories = []string{"tech-debt", "documentation", "inconsistency", "drift", "bug", "ux"}

// baselineSeverities is the severities abcd's baseline lets a drain take.
var baselineSeverities = []string{"nitpick", "minor"}

// Rule is one repository's drain eligibility rule, as its record states it.
type Rule struct {
	// Record is the decision record's id (adr-N); empty for the bundled
	// baseline, which is never applied as a repository's rule.
	Record string `json:"record"`
	// Path is the record's repo-relative, slash-separated path.
	Path string `json:"path"`
	// Categories are the categories a drain may take, in the order it takes
	// them; "security" is last when the record takes it.
	Categories []string `json:"categories"`
	// Severities are the severities a drain may take, in the order it takes
	// them (nitpick, minor, major, critical).
	Severities []string `json:"severities"`
	Security   string   `json:"security"`
	Remedy     string   `json:"remedy"`
	// Loosened names every floor the record loosens against abcd's baseline,
	// "severity major", "severity critical" and "security", in that order.
	// Always non-nil, so a JSON reader sees an empty list rather than none.
	Loosened []string `json:"loosened"`
}

// Baseline is abcd's strict rule: the fixable set, nitpick and minor, security
// handed back, a remedy required. abcd's own repository states exactly this in
// adr-2609291342092738, and the setup offer writes it.
func Baseline() Rule {
	return Rule{
		Categories: slices.Clone(baselineCategories),
		Severities: slices.Clone(baselineSeverities),
		Security:   SecurityHandBack,
		Remedy:     RemedyRequired,
		Loosened:   []string{},
	}
}

// TakesCategory reports whether the rule lets a drain take category c.
func (r Rule) TakesCategory(c string) bool { return slices.Contains(r.Categories, c) }

// TakesSeverity reports whether the rule lets a drain take severity s.
func (r Rule) TakesSeverity(s string) bool { return slices.Contains(r.Severities, s) }

// The refusals. Each is wrapped with the record and field it is about.
var (
	// ErrUnrecorded: the repository holds no accepted record of the rule.
	ErrUnrecorded = errors.New("this repository holds no drain eligibility record")
	// ErrMalformed: the record is partial, misspelt, or states a value the rule
	// does not take.
	ErrMalformed = errors.New("the drain eligibility record is malformed")
	// ErrAmbiguous: more than one accepted record states the rule.
	ErrAmbiguous = errors.New("more than one accepted record states the drain eligibility rule")
	// ErrUnreadable: the decision store, or a record in it, could not be read
	// safely: a link, a file past the size cap, or a read that failed.
	ErrUnreadable = errors.New("the decision store the drain eligibility rule is read from could not be read safely")
)

// HowToAdd is the remedy every ErrUnrecorded refusal names.
const HowToAdd = "add it: run `abcd ahoy install` at a terminal and accept the drain rule it offers, " +
	"which writes abcd's strict baseline as an accepted decision record; or give an accepted decision record " +
	"in " + ADRsRelDir + "/ the four fields " + FieldCategories + ", " + FieldSeverities + ", " +
	FieldSecurity + " and " + FieldRemedy + " (mint one with `abcd decide \"<title>\"`)"

// Load reads the repository's drain eligibility rule from its decision store.
// It refuses, with ErrUnrecorded, a repository whose store holds no accepted
// record carrying the drain fields; with ErrAmbiguous, one holding two; and
// with ErrMalformed, a record that is partial, states a value the rule does
// not take, states any key twice, or claims an id its file name does not give
// it; and with ErrUnreadable, a store or record that cannot be read safely.
//
// The store is read inside an os.Root at the checkout, and each record through
// the capped trust-boundary reader, so a store that is a symlink leaving the
// checkout, a record that is a symlink at all, and a record past the size cap
// are refused, never followed or read whole: the rule is the drained tree's
// committed history, and a rule from elsewhere is not it.
func Load(repoRoot string) (Rule, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Rule{}, fmt.Errorf("%w: opening the checkout: %w", ErrUnreadable, err)
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ADRsRelDir)
	if errors.Is(err, fs.ErrNotExist) {
		return Rule{}, fmt.Errorf("%w: it has no decision store at %s; %s", ErrUnrecorded, ADRsRelDir, HowToAdd)
	}
	if err != nil {
		return Rule{}, fmt.Errorf("%w: reading %s: %w", ErrUnreadable, ADRsRelDir, err)
	}
	type candidate struct {
		id, rel, status string
		fields          map[string]frontmatter.Field
	}
	var accepted, other []candidate
	for _, e := range entries {
		fileID := recordid.ADRFileID(e.Name())
		if e.IsDir() || fileID == "" {
			continue
		}
		rel := path.Join(ADRsRelDir, e.Name())
		raw, err := readRecord(root, rel)
		if err != nil {
			return Rule{}, err
		}
		lines := strings.Split(string(raw), "\n")
		fields := frontmatter.Fields(lines)
		if !carriesDrainFields(fields) {
			continue
		}
		c := candidate{id: fileID, rel: rel, fields: fields}
		// A record that states any top-level key twice says two things: the
		// line scanner keeps the first value and a YAML reader the last, so
		// `status: accepted` then `status: superseded` would be admitted here
		// and called superseded everywhere else. Refused whatever its status
		// reads as, so neither reading decides what an unattended drain takes.
		if dups := frontmatter.Duplicates(lines); len(dups) > 0 {
			keys := make([]string, 0, len(dups))
			for _, d := range dups {
				if !slices.Contains(keys, d.Key) {
					keys = append(keys, d.Key)
				}
			}
			return Rule{}, fmt.Errorf("%w: %s (%s) states %s more than once; state each key once",
				ErrMalformed, fileID, rel, strings.Join(keys, ", "))
		}
		// Every surface names the rule by its record's id, so a record whose
		// frontmatter claims another record's id would put that record's name
		// on its own rule. The file name is the id the store allocated.
		if fmID, _ := frontmatter.ScalarString(fields["id"].Value); fmID != "" {
			if recordid.CanonADRID(fmID) != fileID {
				return Rule{}, fmt.Errorf("%w: %s says its id is %s, but its file name makes it %s; a record states its own id",
					ErrMalformed, rel, fmID, fileID)
			}
			c.id = fmID
		}
		c.status, _ = frontmatter.ScalarString(fields["status"].Value)
		if c.status == "accepted" {
			accepted = append(accepted, c)
		} else {
			other = append(other, c)
		}
	}
	switch len(accepted) {
	case 0:
		var named []string
		for _, c := range other {
			named = append(named, fmt.Sprintf("%s carries the drain fields but is %s, not accepted", c.id, orNone(c.status)))
		}
		note := ""
		if len(named) > 0 {
			note = " (" + strings.Join(named, "; ") + ")"
		}
		return Rule{}, fmt.Errorf("%w: no accepted decision record in %s carries the drain fields%s; %s",
			ErrUnrecorded, ADRsRelDir, note, HowToAdd)
	case 1:
	default:
		ids := make([]string, len(accepted))
		for i, c := range accepted {
			ids[i] = c.id
		}
		return Rule{}, fmt.Errorf("%w: %s each carry the drain fields; supersede all but one, so the rule a drain applies is one record's",
			ErrAmbiguous, strings.Join(ids, " and "))
	}
	c := accepted[0]
	r, err := parse(c.fields)
	if err != nil {
		return Rule{}, fmt.Errorf("%w: %s (%s): %s", ErrMalformed, c.id, c.rel, err.Error())
	}
	r.Record, r.Path = c.id, c.rel
	return r, nil
}

// readRecord reads one record of the store through the capped trust-boundary
// reader: a link (even one resolving inside the checkout), a FIFO or device, or
// a file past the ledger's record cap is refused rather than read, since the
// store is repository-authored and decides what an unattended drain takes.
func readRecord(root *os.Root, rel string) ([]byte, error) {
	raw, err := fsutil.ReadGuardedInRoot(root, rel, issueschema.RecordReadLimit)
	switch {
	case err == nil:
		return raw, nil
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, fmt.Errorf("%w: %s is larger than the %d-byte size cap and was left unread", ErrUnreadable, rel, issueschema.RecordReadLimit)
	case errors.Is(err, fsutil.ErrNotRegular):
		return nil, fmt.Errorf("%w: %s is not a regular file (a link, a FIFO or a device), and a record is never read through one", ErrUnreadable, rel)
	}
	return nil, fmt.Errorf("%w: reading %s: %w", ErrUnreadable, rel, err)
}

func orNone(s string) string {
	if s == "" {
		return "without a status"
	}
	return s
}

// carriesDrainFields reports whether a record's frontmatter states any part of
// the drain rule.
func carriesDrainFields(fields map[string]frontmatter.Field) bool {
	for k := range fields {
		if strings.HasPrefix(k, fieldPrefix) {
			return true
		}
	}
	return false
}

var knownFields = []string{FieldCategories, FieldSeverities, FieldSecurity, FieldRemedy}

// parse reads the four fields and measures them against the baseline.
func parse(fields map[string]frontmatter.Field) (Rule, error) {
	var unknown []string
	for k := range fields {
		if strings.HasPrefix(k, fieldPrefix) && !slices.Contains(knownFields, k) {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return Rule{}, fmt.Errorf("%s is not a drain rule field; the fields are %s",
			strings.Join(unknown, ", "), strings.Join(knownFields, ", "))
	}
	for _, k := range knownFields {
		if _, ok := fields[k]; !ok {
			return Rule{}, fmt.Errorf("%s is missing; a drain rule states all of %s", k, strings.Join(knownFields, ", "))
		}
	}

	cats, err := flowList(fields, FieldCategories)
	if err != nil {
		return Rule{}, err
	}
	for _, c := range cats {
		switch {
		case slices.Contains(baselineCategories, c):
		case c == SecurityCategory:
			return Rule{}, fmt.Errorf("%s lists security; whether a drain takes security issues is %s's to state (%s or %s)",
				FieldCategories, FieldSecurity, SecurityHandBack, SecurityTake)
		case slices.Contains(issueschema.Categories, c):
			return Rule{}, fmt.Errorf("%s lists %s, a category a person decides by kind; a project's rule may narrow the fixable set (%s), never widen it",
				FieldCategories, c, strings.Join(baselineCategories, ", "))
		default:
			return Rule{}, fmt.Errorf("%s lists %q, which is not an issue category; the fixable set is %s",
				FieldCategories, c, strings.Join(baselineCategories, ", "))
		}
	}
	sevs, err := flowList(fields, FieldSeverities)
	if err != nil {
		return Rule{}, err
	}
	for _, s := range sevs {
		if !slices.Contains(issueschema.Severities, s) {
			return Rule{}, fmt.Errorf("%s lists %q, which is not a severity; the severities are %s",
				FieldSeverities, s, strings.Join(issueschema.Severities, ", "))
		}
	}
	security, _ := frontmatter.ScalarString(fields[FieldSecurity].Value)
	if security != SecurityHandBack && security != SecurityTake {
		return Rule{}, fmt.Errorf("%s is %q; it is %s (abcd's baseline) or %s", FieldSecurity, security, SecurityHandBack, SecurityTake)
	}
	remedy, _ := frontmatter.ScalarString(fields[FieldRemedy].Value)
	if remedy != RemedyRequired {
		return Rule{}, fmt.Errorf("%s is %q; it is %s, the only value: the remedy is the brief a lane works from, so no rule takes an issue without one",
			FieldRemedy, remedy, RemedyRequired)
	}

	r := Rule{Security: security, Remedy: remedy, Loosened: []string{}}
	for _, c := range baselineCategories {
		if slices.Contains(cats, c) {
			r.Categories = append(r.Categories, c)
		}
	}
	for _, s := range issueschema.Severities {
		if slices.Contains(sevs, s) {
			r.Severities = append(r.Severities, s)
			if !slices.Contains(baselineSeverities, s) {
				r.Loosened = append(r.Loosened, "severity "+s)
			}
		}
	}
	if security == SecurityTake {
		r.Categories = append(r.Categories, SecurityCategory)
		r.Loosened = append(r.Loosened, SecurityCategory)
	}
	return r, nil
}

// flowList reads a field written as an inline list, `[a, b]`, refusing any
// other shape (a block sequence reads as an empty value to the line scanner,
// and would otherwise state nothing) and an empty list.
func flowList(fields map[string]frontmatter.Field, key string) ([]string, error) {
	v := strings.TrimSpace(fields[key].Value)
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, fmt.Errorf("%s is not an inline list; write it as %s: [a, b]", key, key)
	}
	items := frontmatter.StringList(v)
	if len(items) == 0 {
		return nil, fmt.Errorf("%s is empty; a rule that takes nothing is not stated this way (narrow it, or leave the drain unrun)", key)
	}
	return items, nil
}

// ProposalTitle is the title the setup offer gives the record it writes.
const ProposalTitle = "A drain takes an issue alone only when its fields say it needs no decision"

// ProposalFrontmatter is the baseline as the four frontmatter lines a record
// carries, each ending in a newline.
func ProposalFrontmatter() string {
	b := Baseline()
	return FieldCategories + ": [" + strings.Join(b.Categories, ", ") + "]\n" +
		FieldSeverities + ": [" + strings.Join(b.Severities, ", ") + "]\n" +
		FieldSecurity + ": " + b.Security + "\n" +
		FieldRemedy + ": " + b.Remedy + "\n"
}

// ProposalBody is the record's body below its title: the four sections every
// decision record carries, stating the baseline in words.
func ProposalBody() string {
	return "## Context\n\n" +
		"`abcd drain` works the open issue ledger unattended: it fixes what needs no decision and hands the " +
		"rest back to a person. Which issue a machine may take alone is this repository's decision, and the " +
		"drain refuses to run here until an accepted record states it.\n\n" +
		"## Decision\n\n" +
		"We will let a drain take an open issue alone only when its fields say it needs no decision, and " +
		"hand every other open issue back by the rule that excluded it. The four `drain_` fields in this " +
		"record's frontmatter are the rule the drain reads:\n\n" +
		"- `" + FieldCategories + "`: the categories a drain may take, from the fixable set " +
		"(`" + strings.Join(baselineCategories, "`, `") + "`); every other category is a person's.\n" +
		"- `" + FieldSeverities + "`: the severities a drain may take; `major` and `critical` are a person's " +
		"unless listed here.\n" +
		"- `" + FieldSecurity + "`: `" + SecurityHandBack + "` keeps every security issue a person's; `" +
		SecurityTake + "` lets a drain take one that passes every other rule.\n" +
		"- `" + FieldRemedy + "`: `" + RemedyRequired + "`; an issue without a remedy, or with the value an " +
		"automatic filer writes, is never taken.\n\n" +
		"These values are abcd's strict baseline. Listing `major` or `critical`, or setting `" + FieldSecurity +
		": " + SecurityTake + "`, loosens a floor, and `abcd drain --dry-run` and the drain start name every " +
		"floor loosened.\n\n" +
		"## Alternatives Considered\n\n" +
		"- **A model classifies each issue.** Rejected: a model detects its own ambiguity badly, and a " +
		"classifier that can let an issue through is the failure this rule exists to prevent.\n" +
		"- **Every issue with a remedy.** Rejected: a remedy says what someone proposed, not that the " +
		"proposal needs no decision.\n" +
		"- **The fields, read in a fixed order (chosen).** Each exclusion names the rule a person reads to " +
		"act on it.\n\n" +
		"## Consequences\n\n" +
		"- A change to what a drain may take is a change to this record, reviewed like code; a pull request " +
		"that loosens it is one a reviewer reads as a trust change.\n" +
		"- An issue whose remedy waits on a ruling, or whose deferral past the current release is live, is " +
		"handed back whatever this record says.\n"
}
