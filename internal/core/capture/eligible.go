package capture

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/launch"
)

// eligible.go — the drain's field-only eligibility rule (itd-82 decisions 4, 6
// and 7; spc-2609212015054359 scope 2, 6 and 9). It decides which open issues
// a machine may take alone, from the record's fields and nothing else, and
// gives every other open issue the disposition of the rule that excluded it.
//
// Which categories and severities a drain may take, and whether it takes
// security issues, is the drained repository's own decision, read from its own
// decision record by core/drainrule (rulings BX2 and H11); a repository without
// one is refused. Two hand-backs hold whatever that record says: a remedy that
// waits on a ruling, and a deferral that is live at the current anchor tag or
// whose liveness the checkout cannot read (no release tag, a tag it lacks, or
// a value that is not a release tag).
//
// Nothing here writes. The plan is what a drain WOULD do; the run that hands
// each eligible issue to an issue-keyed lane is not built yet, and DrainStart
// says so rather than pretending to run.

// DrainOutcome is the disposition a drain gives one open issue.
type DrainOutcome string

// The dispositions, one per open issue.
const (
	// DrainEligible: every field rule passes, so the issue would go to a lane.
	// The host judgement over the remedy (scope 3) may still hand it back; it
	// can never make an issue eligible.
	DrainEligible DrainOutcome = "eligible"
	// DrainHandBack: the issue is a person's, by severity or category.
	DrainHandBack DrainOutcome = "handback"
	// DrainIneligible: a field the rule reads is missing (the remedy), or holds
	// the machine value an automatic filer writes in place of one.
	DrainIneligible DrainOutcome = "ineligible"
	// DrainSkipped: an open record blocks it; the blocker is named.
	DrainSkipped DrainOutcome = "skipped"
	// DrainUnreadable: the reader refuses the record, so no field can be read.
	DrainUnreadable DrainOutcome = "unreadable"
)

// DrainRule names the rule that decided a disposition.
type DrainRule string

// The rules, in the order eligibility asks them.
const (
	RuleUnreadable DrainRule = "unreadable"
	RuleNotOpen    DrainRule = "not-open"
	RuleBlocked    DrainRule = "blocked"
	RuleSecurity   DrainRule = "security"
	RuleCategory   DrainRule = "category"
	RuleSeverity   DrainRule = "severity"
	// RuleDeferred: the record's deferral past the current anchor tag is live,
	// so a person carried it past this release.
	RuleDeferred DrainRule = "deferred"
	// RuleWaitsOnRuling: the remedy opens "Waits on", so the fix it proposes
	// waits on a ruling a person has not given.
	RuleWaitsOnRuling DrainRule = "waits-on-ruling"
	RuleRemedy        DrainRule = "remedy"
	RuleFields        DrainRule = "fields"
)

// DrainVerdict is one open issue's disposition, with the rule that decided it
// and the reason in words.
type DrainVerdict struct {
	ID       string       `json:"id"`
	Path     string       `json:"path"`
	Severity Severity     `json:"severity,omitempty"`
	Category Category     `json:"category,omitempty"`
	Outcome  DrainOutcome `json:"outcome"`
	Rule     DrainRule    `json:"rule"`
	Reason   string       `json:"reason"`
	// Blockers are the open records a skipped issue waits on.
	Blockers []string `json:"blockers,omitempty"`
}

// eligibility judges one issue by its fields alone, under the repository's
// rule r, with anchor the checkout's current release tag as far as it is known. iss.BlockedByOpen must be the derived projection List fills (the
// blockers still in open/). The rules are asked in a fixed order, and the
// first that excludes the issue decides: not open, blocked, security, a
// category outside the rule's set, a severity outside it, a remedy that waits
// on a ruling, a live deferral, no remedy, the automatic filers' remedy. An
// issue no rule excludes is eligible.
//
// Blocked comes first because a blocked issue is not considered at all until
// its blocker clears; the severity and category hand-backs come before the
// others because answering a ruling, waiting out a deferral or adding a remedy
// would not make such an issue eligible, so naming them would send a reader to
// the wrong fix. A remedy waiting on a ruling is named before a deferral
// because it says which decision is owed; a record carrying both waits on
// both. `capture defer` writes a deferral only onto a major or
// critical record, but a hand-written one on a lighter record is read alike.
func eligibility(iss Issue, r drainrule.Rule, anchor deferralAnchor) DrainVerdict {
	v := DrainVerdict{ID: iss.ID, Path: iss.Path, Severity: iss.Severity, Category: iss.Category}
	decide := func(o DrainOutcome, rule DrainRule, reason string) DrainVerdict {
		v.Outcome, v.Rule, v.Reason = o, rule, reason
		return v
	}
	deferred, deferredOK := parseReleaseTag(iss.deferredAfter)
	switch {
	case iss.Status != StateOpen:
		return decide(DrainSkipped, RuleNotOpen, fmt.Sprintf("the record is %s, and a drain takes only open issues", iss.Status))
	case len(iss.BlockedByOpen) > 0:
		v.Blockers = append([]string(nil), iss.BlockedByOpen...)
		return decide(DrainSkipped, RuleBlocked, "blocked by "+strings.Join(iss.BlockedByOpen, ", ")+", still open")
	case iss.Category == drainrule.SecurityCategory && r.Security != drainrule.SecurityTake:
		return decide(DrainHandBack, RuleSecurity, "category security is always a person's under this repository's rule")
	case !r.TakesCategory(string(iss.Category)):
		return decide(DrainHandBack, RuleCategory, fmt.Sprintf("category %s is outside the fixable set (%s)",
			iss.Category, strings.Join(r.Categories, ", ")))
	case !r.TakesSeverity(string(iss.Severity)):
		return decide(DrainHandBack, RuleSeverity, fmt.Sprintf("severity %s is above the drain's (%s)",
			iss.Severity, strings.Join(r.Severities, ", ")))
	case waitsOnRuling(iss.Remedy):
		return decide(DrainHandBack, RuleWaitsOnRuling,
			"the remedy opens \"Waits on\": the fix waits on a person's ruling, so it is a person's until the ruling is given and the remedy rewritten")
	case anchor.unknown && iss.deferredAfter != "":
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %s, anchor unknown: this checkout holds no release tag (a shallow clone fetches none), so whether the deferral is live cannot be read and it is a person's; `git fetch --tags` and drain again",
			iss.deferredAfter))
	case iss.deferredAfter != "" && !deferredOK:
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %q, which is not a release tag (want vMAJOR.MINOR.PATCH): whether the deferral is live cannot be read, so it is a person's; `abcd capture defer` writes one the drain reads",
			iss.deferredAfter))
	case deferredOK && launch.CoreGreater(deferred, anchor.local):
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %s, anchor stale: this checkout's newest release tag is %s and it lacks %s (a checkout not fetched since the last cut), so whether the deferral is live cannot be read and it is a person's; `git fetch --tags` and drain again",
			iss.deferredAfter, anchor.tag, iss.deferredAfter))
	case anchor.stale == "" && anchor.tag != "" && iss.deferredAfter == anchor.tag:
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %s, the current anchor: a person carried it past this release, so it is a person's until the deferral lapses", anchor.tag))
	case strings.TrimSpace(iss.Remedy) == "":
		return decide(DrainIneligible, RuleRemedy, fmt.Sprintf(
			"no remedy: field; ineligible until someone adds one with `abcd capture remedy %s \"<fix>\"`", iss.ID))
	case issueschema.IsMachineRemedy(iss.Remedy):
		// An automatic filer's record (ruling H12): filed, and skipped until a
		// person writes the fix it proposes.
		return decide(DrainIneligible, RuleRemedy, fmt.Sprintf(
			"remedy is %q, written by an automatic filer; ineligible until a person writes a real remedy with `abcd capture remedy %s \"<fix>\"`",
			issueschema.MachineRemedy, iss.ID))
	}
	return decide(DrainEligible, RuleFields,
		"every field rule passes on its remedy; the host judgement over the remedy may still hand it back")
}

// waitsOnPrefix opens a remedy whose fix waits on an unanswered ruling, the
// shape the ledger's remedies are written in ("Waits on <ruling>: ...").
const waitsOnPrefix = "waits on"

// waitsOnRuling reports whether a remedy opens "Waits on" as words: followed
// by a blank, a colon, or nothing, compared case-folded after leading blanks,
// so "Waits on: ruling H4." and a lower-case spelling are held back too, and
// "Waits onward" is not.
func waitsOnRuling(remedy string) bool {
	rest, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(remedy)), waitsOnPrefix)
	if !ok {
		return false
	}
	return rest == "" || rest[0] == ':' || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\n' || rest[0] == '\r'
}

// drainOrder states the ordering rule r takes eligible issues in.
func drainOrder(r drainrule.Rule) string {
	return "category " + strings.Join(r.Categories, ", ") + "; then " +
		strings.Join(r.Severities, " before ") + "; then oldest first"
}

// orderEligible sorts eligible verdicts by the drain order: category, then
// severity, then oldest first (ascending id number; ids are minted in time
// order, the hand-numbered ordinals before the timestamp ids).
func orderEligible(vs []DrainVerdict, r drainrule.Rule) {
	sort.SliceStable(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if ca, cb := slices.Index(r.Categories, string(a.Category)), slices.Index(r.Categories, string(b.Category)); ca != cb {
			return ca < cb
		}
		if sa, sb := slices.Index(r.Severities, string(a.Severity)), slices.Index(r.Severities, string(b.Severity)); sa != sb {
			return sa < sb
		}
		return issNumber(a.ID) < issNumber(b.ID)
	})
}

// DrainPlanRequest is the input to PlanDrain.
type DrainPlanRequest struct {
	RepoRoot   string
	IssuesRoot string
}

// DrainPlan is what a drain would do over the open ledger, and writes nothing:
// every open issue's disposition, the eligible ones first in the order a drain
// takes them and the rest after them by id, with the repository's rule, the
// decision record it is stated in, and every floor it loosens.
type DrainPlan struct {
	// Record is the repository's own decision record the rule is read from.
	Record string `json:"record"`
	// Rule is the rule as that record states it.
	Rule drainrule.Rule `json:"rule"`
	// Loosened names every floor the record loosens against abcd's baseline
	// (ruling H11); an empty list when it loosens none.
	Loosened []string `json:"loosened"`
	// Anchor is the checkout's newest release tag, the one a live deferral
	// names, when any open record carries a deferral; empty otherwise.
	Anchor string `json:"anchor,omitempty"`
	// AnchorUnknown reports that an open record carries a deferral and the
	// checkout holds no release tag (a shallow clone fetches none), so every
	// record carrying a deferral is handed back rather than judged.
	AnchorUnknown bool `json:"anchor_unknown,omitempty"`
	// AnchorStale names the newest release tag an open record is deferred
	// past that is newer than Anchor, so the checkout lacks it (one not
	// fetched since the last cut); empty when no deferral names such a tag.
	// Every record deferred past a tag the checkout lacks is handed back
	// rather than judged.
	AnchorStale  string               `json:"anchor_stale,omitempty"`
	Order        string               `json:"order"`
	Dispositions []DrainVerdict       `json:"dispositions"`
	Counts       map[DrainOutcome]int `json:"counts"`
}

// PlanDrain classifies every open issue by field, under the repository's own
// rule. It refuses, before reading the ledger, a repository whose rule is
// unrecorded, ambiguous or malformed. Read-only: it takes no lock and writes
// nothing, as List does.
func PlanDrain(req DrainPlanRequest) (DrainPlan, error) {
	repoRoot, _, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return DrainPlan{}, err
	}
	rule, err := drainrule.Load(repoRoot)
	if err != nil {
		return DrainPlan{}, err
	}
	lr, err := List(ListRequest{RepoRoot: req.RepoRoot, IssuesRoot: req.IssuesRoot, State: StateOpen})
	if err != nil {
		return DrainPlan{}, err
	}
	anchor, err := liveDeferralAnchor(repoRoot, lr.Issues)
	if err != nil {
		return DrainPlan{}, err
	}
	var eligible, rest []DrainVerdict
	for _, iss := range lr.Issues {
		v := eligibility(iss, rule, anchor)
		if v.Outcome == DrainEligible {
			eligible = append(eligible, v)
		} else {
			rest = append(rest, v)
		}
	}
	// A record the reader refuses is still an open issue, and "every open issue
	// receives exactly one disposition" holds of it too: it is named with the
	// reader's refusal rather than silently absent.
	for _, sk := range lr.Skipped {
		id := skippedRecordID(sk.Path)
		if id == "" {
			continue
		}
		rest = append(rest, DrainVerdict{ID: id, Path: sk.Path, Outcome: DrainUnreadable, Rule: RuleUnreadable,
			Reason: fmt.Sprintf("the reader refuses the record at its %s stage: %s", sk.Layer, sk.Error)})
	}
	orderEligible(eligible, rule)
	sort.SliceStable(rest, func(i, j int) bool { return issNumber(rest[i].ID) < issNumber(rest[j].ID) })
	plan := DrainPlan{
		Record:        rule.Record,
		Rule:          rule,
		Loosened:      rule.Loosened,
		Anchor:        anchor.tag,
		AnchorUnknown: anchor.unknown,
		AnchorStale:   anchor.stale,
		Order:         drainOrder(rule),
		Dispositions:  append(append([]DrainVerdict{}, eligible...), rest...),
		Counts:        map[DrainOutcome]int{},
	}
	for _, v := range plan.Dispositions {
		plan.Counts[v.Outcome]++
	}
	return plan, nil
}

// deferralAnchor is the release tag a deferral is judged against: the tag, or
// unknown when an open record carries a deferral and the checkout holds no
// release tag. stale names the newest tag an open record is deferred past that
// is newer than the checkout's own, which the checkout therefore lacks; a
// deferral past the local tag has then lapsed, and one past a tag it lacks is
// handed back. The zero value is "no deferral to judge".
type deferralAnchor struct {
	tag     string
	local   launch.Semver
	unknown bool
	stale   string
}

// liveDeferralAnchor returns the checkout's current release tag when any open
// record carries a deferral, and the zero anchor when none does. The tags are
// read only when a deferral needs judging, and not knowing whether a deferral
// is live never lets its record through: a failure to read the tags refuses
// the plan, a checkout holding no release tag (a shallow clone fetches none)
// marks the anchor unknown, which hands back every record carrying a
// deferral, and a deferral past a tag newer than the checkout's own (a
// checkout not fetched since the last cut) marks the anchor stale, which hands
// back every record deferred past a tag the checkout lacks. No remote is
// asked: the record's own tag is the evidence the local anchor is behind. A
// live deferral is a person's decision.
func liveDeferralAnchor(repoRoot string, issues []Issue) (deferralAnchor, error) {
	if !slices.ContainsFunc(issues, func(iss Issue) bool { return iss.deferredAfter != "" }) {
		return deferralAnchor{}, nil
	}
	tag, found, err := changelog.LatestReleaseTag(repoRoot)
	if err != nil {
		return deferralAnchor{}, fmt.Errorf("drain: an open record carries a deferral, and the release tags that say whether it is live could not be read: %w", err)
	}
	if !found {
		return deferralAnchor{unknown: true}, nil
	}
	a := deferralAnchor{tag: tag.Tag(), local: tag}
	newest := tag
	for _, iss := range issues {
		if d, ok := parseReleaseTag(iss.deferredAfter); ok && launch.CoreGreater(d, newest) {
			newest, a.stale = d, iss.deferredAfter
		}
	}
	return a, nil
}

// parseReleaseTag reads a deferral's tag as a release version: the shape
// `capture defer` writes (vMAJOR.MINOR.PATCH, strict SemVer core). Anything
// else reports false, and the drain hands the record back rather than guess.
func parseReleaseTag(tag string) (launch.Semver, bool) {
	if !reShippedIn.MatchString(tag) {
		return launch.Semver{}, false
	}
	v, err := launch.ParseSemver(strings.TrimPrefix(tag, "v"))
	return v, err == nil
}

// ErrDrainRuleUnrecorded is the refusal when the drained repository holds no
// record of its eligibility rule (itd-82 decision 4, criterion 11; ruling BX2).
var ErrDrainRuleUnrecorded = drainrule.ErrUnrecorded

// ErrDrainRunUnbuilt is the refusal of the run itself: the issue-keyed lane a
// drain hands each eligible issue to (itd-2609201916151817 decision 10) is
// not built, so there is nothing safe to start.
var ErrDrainRunUnbuilt = errors.New("the drain run is not built")

// DrainStart is the check an unattended drain makes before it starts. It reads
// the repository's own rule and refuses without it, naming how to add it, or
// when it is ambiguous or malformed. With the rule it still refuses, because
// the run has no lane to hand an issue to yet, naming the rule's record and
// every floor the record loosens (ruling H11). Writes nothing.
func DrainStart(repoRoot string) error {
	rule, err := drainrule.Load(repoRoot)
	if err != nil {
		return err
	}
	loosened := ""
	if len(rule.Loosened) > 0 {
		loosened = fmt.Sprintf("; this repository's rule loosens abcd's floors, letting a drain take %s",
			strings.Join(rule.Loosened, ", "))
	}
	return fmt.Errorf("%w: the issue-keyed lane it hands each eligible issue to does not exist yet "+
		"(itd-2609201916151817 decision 10); `abcd drain --dry-run` shows what it would do under %s%s",
		ErrDrainRunUnbuilt, rule.Record, loosened)
}
