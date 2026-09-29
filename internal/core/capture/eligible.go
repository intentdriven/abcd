package capture

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// eligible.go — the drain's field-only eligibility rule (itd-82 decisions 4, 6
// and 7; spc-2609212015054359 scope 2, 6 and 9). It decides which open issues
// a machine may take alone, from the record's fields and nothing else, and
// gives every other open issue the disposition of the rule that excluded it.
// The rule is a recorded decision: EligibilityRecord names it.
//
// Nothing here writes. The plan is what a drain WOULD do; the run that hands
// each eligible issue to an issue-keyed lane is not built yet, and DrainStart
// says so rather than pretending to run.

// EligibilityRecord is the decision record that states the rule eligibility
// applies (itd-82 decision 4). It is the record the drain refuses to start
// without; TestTheEligibilityRuleIsRecordedAndAccepted holds it to an accepted
// record in this repository's decision store, cited by the brief's invariants.
const EligibilityRecord = "adr-2609291342092738"

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
	// DrainIneligible: a field the rule reads is missing (the remedy).
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
	RuleRemedy     DrainRule = "remedy"
	RuleFields     DrainRule = "fields"
)

// DrainCategories is the fixable set in the order a drain takes it (decision
// 7): clean-ups and text first, on the evidence that they merge most often;
// behaviour changes last.
var DrainCategories = []Category{"tech-debt", "documentation", "inconsistency", "drift", "bug", "ux"}

// DrainSeverities is the severities a drain may take, in the order it takes
// them. major and critical are handed back by default.
var DrainSeverities = []Severity{SeverityNitpick, SeverityMinor}

// DrainOrder is the ordering rule as the summary states it.
const DrainOrder = "category tech-debt, documentation, inconsistency, drift, bug, ux; " +
	"then nitpick before minor; then oldest first"

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

// eligibility judges one issue by its fields alone. iss.BlockedByOpen must be
// the derived projection List fills (the blockers still in open/). The rules
// are asked in a fixed order, and the first that excludes the issue decides:
// not open, blocked, security, a category outside the fixable set, a severity
// above minor, no remedy. An issue no rule excludes is eligible.
//
// Blocked comes first because a blocked issue is not considered at all until
// its blocker clears; the severity and category hand-backs come before the
// missing remedy because adding a remedy would not make such an issue
// eligible, so naming the remedy would send a reader to the wrong fix.
func eligibility(iss Issue) DrainVerdict {
	v := DrainVerdict{ID: iss.ID, Path: iss.Path, Severity: iss.Severity, Category: iss.Category}
	decide := func(o DrainOutcome, r DrainRule, reason string) DrainVerdict {
		v.Outcome, v.Rule, v.Reason = o, r, reason
		return v
	}
	switch {
	case iss.Status != StateOpen:
		return decide(DrainSkipped, RuleNotOpen, fmt.Sprintf("the record is %s, and a drain takes only open issues", iss.Status))
	case len(iss.BlockedByOpen) > 0:
		v.Blockers = append([]string(nil), iss.BlockedByOpen...)
		return decide(DrainSkipped, RuleBlocked, "blocked by "+strings.Join(iss.BlockedByOpen, ", ")+", still open")
	case iss.Category == "security":
		return decide(DrainHandBack, RuleSecurity, "category security is always a person's")
	case slices.Index(DrainCategories, iss.Category) < 0:
		return decide(DrainHandBack, RuleCategory, fmt.Sprintf("category %s is outside the fixable set (%s)",
			iss.Category, joinCategories(DrainCategories)))
	case slices.Index(DrainSeverities, iss.Severity) < 0:
		return decide(DrainHandBack, RuleSeverity, fmt.Sprintf("severity %s is above the drain's (nitpick, minor)", iss.Severity))
	case strings.TrimSpace(iss.Remedy) == "":
		return decide(DrainIneligible, RuleRemedy, "no remedy: field; ineligible until someone adds one")
	}
	return decide(DrainEligible, RuleFields,
		"every field rule passes on its remedy; the host judgement over the remedy may still hand it back")
}

func joinCategories(cs []Category) string {
	s := make([]string, len(cs))
	for i, c := range cs {
		s[i] = string(c)
	}
	return strings.Join(s, ", ")
}

// orderEligible sorts eligible verdicts by the drain order: category, then
// severity, then oldest first (ascending id number; ids are minted in time
// order, the hand-numbered ordinals before the timestamp ids).
func orderEligible(vs []DrainVerdict) {
	sort.SliceStable(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if ca, cb := slices.Index(DrainCategories, a.Category), slices.Index(DrainCategories, b.Category); ca != cb {
			return ca < cb
		}
		if sa, sb := slices.Index(DrainSeverities, a.Severity), slices.Index(DrainSeverities, b.Severity); sa != sb {
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
// takes them and the rest after them by id, with the ordering rule and the
// decision record the rule is stated in.
type DrainPlan struct {
	Record       string               `json:"record"`
	Order        string               `json:"order"`
	Dispositions []DrainVerdict       `json:"dispositions"`
	Counts       map[DrainOutcome]int `json:"counts"`
}

// PlanDrain classifies every open issue by field. Read-only: it takes no lock
// and writes nothing, as List does.
func PlanDrain(req DrainPlanRequest) (DrainPlan, error) {
	lr, err := List(ListRequest{RepoRoot: req.RepoRoot, IssuesRoot: req.IssuesRoot, State: StateOpen})
	if err != nil {
		return DrainPlan{}, err
	}
	var eligible, rest []DrainVerdict
	for _, iss := range lr.Issues {
		v := eligibility(iss)
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
	orderEligible(eligible)
	sort.SliceStable(rest, func(i, j int) bool { return issNumber(rest[i].ID) < issNumber(rest[j].ID) })
	plan := DrainPlan{
		Record:       EligibilityRecord,
		Order:        DrainOrder,
		Dispositions: append(append([]DrainVerdict{}, eligible...), rest...),
		Counts:       map[DrainOutcome]int{},
	}
	for _, v := range plan.Dispositions {
		plan.Counts[v.Outcome]++
	}
	return plan, nil
}

// ErrDrainRuleUnrecorded is the refusal when the eligibility rule has no
// decision record (itd-82 decision 4, criterion 11).
var ErrDrainRuleUnrecorded = errors.New("the drain eligibility rule has no decision record")

// ErrDrainRunUnbuilt is the refusal of the run itself: the issue-keyed lane a
// drain hands each eligible issue to (itd-2609201916151817 decision 10) is
// not built, so there is nothing safe to start.
var ErrDrainRunUnbuilt = errors.New("the drain run is not built")

// DrainStart is the check an unattended drain makes before it starts. It
// refuses without the eligibility rule's decision record, naming the record it
// needs, and otherwise refuses because the run has no lane to hand an issue
// to yet. Writes nothing.
func DrainStart() error { return drainStartCheck(EligibilityRecord) }

func drainStartCheck(record string) error {
	if record == "" {
		return fmt.Errorf("%w: an unattended drain needs the decision record itd-82 decision 4 owes "+
			"(the rule for which issues need no decision) before it starts", ErrDrainRuleUnrecorded)
	}
	return fmt.Errorf("%w: the issue-keyed lane it hands each eligible issue to does not exist yet "+
		"(itd-2609201916151817 decision 10); `abcd drain --dry-run` shows what it would do under %s",
		ErrDrainRunUnbuilt, record)
}
