package capture

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// eligible.go — the drain's field-only eligibility rule (itd-82 decisions 4, 6
// and 7; spc-2609212015054359 scope 2, 6 and 9). It decides which open issues
// a machine may take alone, from the record's fields and nothing else, and
// gives every other open issue the disposition of the rule that excluded it.
//
// Which categories and severities a drain may take, and whether it takes
// security issues, is the drained repository's own decision, read from its own
// decision record by core/drainrule (rulings BX2 and H11); a repository without
// one is refused. Three hand-backs hold whatever that record says: an issue
// whose related_intents names an intent (promoted; spc-2610161044461448 step
// 4), which goes through that intent, a record the consistency pass files
// aside (ruling H12, dcn-2610161425496537); a remedy that waits on a ruling;
// and a deferral that is live at the current anchor tag or whose liveness the
// checkout cannot read (no release tag, a tag it lacks, or a value that is not
// a release tag).
//
// Nothing here writes. The plan is what a drain WOULD do; the run that hands
// each eligible issue to an issue-keyed lane is the implement loop's
// (loop.Drain), which reads this plan afresh at every move.

// DrainOutcome is the disposition a drain gives one open issue.
type DrainOutcome string

// The dispositions, one per open issue.
const (
	// DrainEligible: every field rule passes, so the issue would go to a lane.
	// The host judgement over the remedy (scope 3) may still hand it back; it
	// can never make an issue eligible.
	DrainEligible DrainOutcome = "eligible"
	// DrainHandBack: the issue is not a drain's to take alone: it goes through
	// the intent its related_intents names (promoted), or it is a person's, by
	// category, severity, a remedy waiting on a ruling or a live deferral.
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
	// RulePromoted: the issue's related_intents names an intent, so it goes
	// through that intent (`abcd build`, or `abcd intent audit` for an
	// audit's owed check), never through a drain alone.
	RulePromoted DrainRule = "promoted"
	RuleSecurity DrainRule = "security"
	RuleCategory DrainRule = "category"
	RuleSeverity DrainRule = "severity"
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
	ID   string `json:"id"`
	Path string `json:"path"`
	// Title is the record's one-line summary: its first non-blank body line.
	Title    string       `json:"title,omitempty"`
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
// first that excludes the issue decides: not open, blocked, promoted to an
// intent, security, a category outside the rule's set, a severity outside it,
// a remedy that waits on a ruling, a live deferral, no remedy, the automatic
// filers' remedy. An issue no rule excludes is eligible.
//
// Blocked comes first because a blocked issue is not considered at all until
// its blocker clears; promoted comes next because an issue an intent owns is
// worked through that intent whatever its other fields say, so naming any of
// them would send a reader to the wrong owner (spc-2610161044461448 step 4;
// the field travels with the record, so a drain that reads a promoted record
// never takes the issue again); the severity and category hand-backs come before the
// others because answering a ruling, waiting out a deferral or adding a remedy
// would not make such an issue eligible, so naming them would send a reader to
// the wrong fix. A remedy waiting on a ruling is named before a deferral
// because it says which decision is owed; a record carrying both waits on
// both. `capture defer` writes a deferral only onto a major or
// critical record, but a hand-written one on a lighter record is read alike.
//
// setAsidePromoted asks every rule but promoted, in the same order: the
// verdict the issue would have without its link to an intent. A person's own
// build reads it to take a promoted issue only when no other hand-back would
// hold it (refining dcn-2610161428244461); the drain never passes it.
func eligibility(iss Issue, r drainrule.Rule, anchor deferralAnchor, opts ...eligibilityOption) DrainVerdict {
	skipPromoted := slices.Contains(opts, setAsidePromoted)
	// The title is the record's own first line, so it is masked: it reaches
	// `drain --json` (iss-2610121542258778).
	v := DrainVerdict{ID: iss.ID, Path: iss.Path, Title: termsafe.Sanitize(issueTitleLine(iss.Body, iss.Slug)), Severity: iss.Severity, Category: iss.Category}
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
	case !skipPromoted && len(promotedIntents(iss)) > 0:
		return decide(DrainHandBack, RulePromoted, promotedReason(promotedIntents(iss)))
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
		// The tag is the record's own text, unjudged here, so it is masked: the
		// reason reaches `drain --json` and a judgement's stored note
		// (iss-2610121516552317).
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %s, anchor unknown: this checkout holds no release tag (a shallow clone fetches none), so whether the deferral is live cannot be read and it is a person's; `git fetch --tags` and drain again",
			termsafe.Sanitize(iss.deferredAfter)))
	case iss.deferredAfter != "" && !deferredOK:
		// Masked before it is quoted, as above: %q escapes only the runes Go
		// deems unprintable, and termsafe masks some Go prints (U+034F).
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %q, which is not a release tag (want vMAJOR.MINOR.PATCH): whether the deferral is live cannot be read, so it is a person's; `abcd capture defer` writes one the drain reads",
			termsafe.Sanitize(iss.deferredAfter)))
	case deferredOK && launch.CoreGreater(deferred, anchor.local):
		return decide(DrainHandBack, RuleDeferred, fmt.Sprintf(
			"deferred past %s, anchor stale: this checkout's newest release tag is %s and it lacks %s (a checkout not fetched since the last cut), so whether the deferral is live cannot be read and it is a person's; `git fetch --tags` and drain again",
			iss.deferredAfter, anchor.tag, iss.deferredAfter))
	case anchor.tag != "" && iss.deferredAfter == anchor.tag:
		// Handed back whether or not the anchor is stale: a newer tag named
		// only in the ledger is not a tag this checkout holds, so the
		// deferral at the local anchor lapses when that tag is fetched.
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

// eligibilityOption changes which rules eligibility asks.
type eligibilityOption int

// setAsidePromoted: eligibility asks every rule but promoted.
const setAsidePromoted eligibilityOption = iota + 1

// promotedIntents is every intent the issue's related_intents names, each
// once, in the record's order, and none for a record the consistency pass
// filed, whose related_intents names the intents its contradiction is between
// (consistencyFinding). The reader has already refused a record whose
// list holds an entry that is not an intent id (`itd-` and digits) or names
// one twice, so no entry is blank; the de-duplication serves a caller that
// builds an Issue by hand.
func promotedIntents(iss Issue) []string {
	if consistencyFinding(iss) {
		return nil
	}
	var ids []string
	for _, id := range iss.RelatedIntents {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

// maxPromotedNamed bounds the intents a promoted reason names, so a record
// linked to many stays one readable line.
const maxPromotedNamed = 3

// promotedReason is a promoted hand-back's reason: what the field says, that
// related_intents names the intents, and the way out, the verbs that work an
// issue through its intent. It claims no more than the link: a related intent
// need not have taken the issue on. ids are distinct and non-empty; at most
// maxPromotedNamed are named, then how many more. The ids are the record's own
// text, unjudged here, so they are masked: the reason reaches `drain --json`.
func promotedReason(ids []string) string {
	named := strings.Join(ids[:min(len(ids), maxPromotedNamed)], ", ")
	if more := len(ids) - maxPromotedNamed; more > 0 {
		named += fmt.Sprintf(" and %d more", more)
	}
	target := "<itd-N>"
	if len(ids) == 1 {
		target = ids[0]
	}
	return termsafe.Sanitize(fmt.Sprintf(
		"related_intents names %s; an issue linked to an intent goes through that intent: `abcd build %s`, or `abcd intent audit %s` for an audit's owed check",
		named, target, target))
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
	// SetAsidePromoted judges every issue by every rule but promoted: the
	// disposition each would have without its link to an intent. A person's
	// own `abcd build <iss-N>` asks it of an issue the drain hands back as
	// promoted, and takes the issue only when nothing else would hold it; a
	// drain never sets it, so it hands every promoted issue back.
	SetAsidePromoted bool
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
	var opts []eligibilityOption
	if req.SetAsidePromoted {
		opts = append(opts, setAsidePromoted)
	}
	var eligible, rest []DrainVerdict
	for _, iss := range lr.Issues {
		v := eligibility(iss, rule, anchor, opts...)
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
		// The reader's refusal can quote the record's own text unquoted (a
		// schema_version's value), so it is masked where the reason is built
		// (iss-2610121516552317). The path is the reader's well-formed name.
		rest = append(rest, DrainVerdict{ID: id, Path: sk.Path, Outcome: DrainUnreadable, Rule: RuleUnreadable,
			Reason: fmt.Sprintf("the reader refuses the record at its %s stage: %s", sk.Layer, termsafe.Sanitize(sk.Error))})
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
// record deferred past a tag it lacks is handed back, and so is one deferred
// past the local tag, which lapses only when the newer tag is fetched. The
// zero value is "no deferral to judge".
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
// asked: the record's own tag is the evidence the local anchor may be behind.
// That evidence is a ledger field, not a git tag, so it never lapses a
// deferral either: a record deferred past the local tag is handed back whether
// or not the anchor is stale, which keeps this invariant. A live deferral is a
// person's decision.
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
