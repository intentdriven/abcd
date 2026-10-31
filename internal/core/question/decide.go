package question

// The one decision the question gate makes (spc-2610301917245442, step 1;
// itd-2610230756259215 decisions 2 and 3). The guard hook and the read-only
// `abcd guard question` both call Decide over the same field view and the same
// state, so a draft the verb passes cannot be refused by a hook of the same
// build: two callers of one function cannot drift apart as two copies of the
// same rules would. The stores that feed State are read in the surface, and
// the writes an admission makes (the open-question marker, the mode set from
// a chip) stay in the hook, after the decision, where the verb never reaches
// them.

// Outcome is what the guard hook does with a question-tool call.
type Outcome string

// The outcomes, decision 3's set.
const (
	// OutcomeAdmit is a question of abcd's with no finding: the hook lets it
	// run silently and, where the badge shows, marks it open.
	OutcomeAdmit Outcome = "admit"
	// OutcomeAdmitWithRowsNote is a question whose only findings are the rows
	// limit's: the hook shows it and tells the agent afterwards
	// (iss-2610070637562567).
	OutcomeAdmitWithRowsNote Outcome = "admit-with-rows-note"
	// OutcomeRefuse is a question of abcd's with a finding that refuses: the
	// hook denies the call naming the findings.
	OutcomeRefuse Outcome = "refuse"
	// OutcomeRefuseAway is any question while the person is away: the hook
	// denies it before any rule is read (spc-2610180914469489).
	OutcomeRefuseAway Outcome = "refuse-away"
	// OutcomeAdmitUnchecked is a question that is not abcd's: the hook lets it
	// run unchecked and unmarked.
	OutcomeAdmitUnchecked Outcome = "admit-unchecked"
	// OutcomeUnchecked is one of the hook's fail-open paths: a store the
	// decision needed could not be read, so the question runs on the hook's
	// loud, non-blocking status and the verdict is partial.
	OutcomeUnchecked Outcome = "unchecked"
)

// Refuses reports whether the hook denies a call with this outcome.
func (o Outcome) Refuses() bool { return o == OutcomeRefuse || o == OutcomeRefuseAway }

// Store names a state store the decision reads.
type Store string

// The stores.
const (
	// StoreAway is the away record: while it holds every question is
	// refused, so it decides whether any rule is read at all.
	StoreAway Store = "away"
	// StoreMode is the waiting-on mode, read where the badge shows: it decides
	// whether a question without abcd's chip is abcd's and, for one, the
	// product thinker's register rule.
	StoreMode Store = "mode"
)

// State is what the gate knows about the checkout before it decides: whether
// the person is away, whom the mode names, whether the badge shows, and the
// binary's verbs. An Err field set means that store could not be read; its
// text is the reader's and is the surface's to scrub.
type State struct {
	// Repo is true when the repository root was found, so the away record
	// was consulted and the verdict depends on it.
	Repo bool
	// Away is true while the person's away record holds.
	Away bool
	// AwayErr is why the away record could not be read safely; it then reads
	// as not away.
	AwayErr error
	// Badge is true in a checkout abcd manages with the local tier, the only
	// place a mode store exists.
	Badge bool
	// Mode is the person the mode names; Unnamed for managed or no store.
	Mode Person
	// ModeErr is why the mode store could not be read where the badge shows.
	ModeErr error
	// RootErr is why the repository root could not be found in a tree that
	// looks like a checkout, so the mode store was never read.
	RootErr error
	// Verbs is the binary's verb list, for the register rule.
	Verbs []string
}

// Unread is one store the decision needed and could not read, with why and
// the rules it therefore could not apply.
type Unread struct {
	Store Store
	Err   error
	Rules []Rule
}

// Verdict is the decision: the hook's outcome, every finding, each tab's
// addressee, the rows of every checked tab, and what the decision could not
// read.
type Verdict struct {
	Outcome Outcome
	// Abcds is true when the question is abcd's: a header in the chip
	// grammar, or a mode naming somebody.
	Abcds bool
	// Chipped is true when a header is in abcd's chip grammar.
	Chipped bool
	// Findings is every finding of the limits check, in CheckLimits's order;
	// empty when no rule was read (away, or not abcd's question).
	Findings []Finding
	// Addressees is the person each checked tab is for: the chip's role for
	// a chipped tab, the mode's person otherwise.
	Addressees []Person
	// ModeAfter is the mode an admitted question leaves where the badge
	// shows: the chip's person for a chipped question, the current mode
	// otherwise. ModeChanges is true when it differs from State.Mode
	// (decision 15).
	ModeAfter   Person
	ModeChanges bool
	// Rows is the row split of every checked tab, in tab order.
	Rows []RowSplit
	// Partial is true when a store the decision needed could not be read;
	// Unread names each one.
	Partial bool
	Unread  []Unread
	// DependsOnState is true when another mode or away state could change
	// the outcome: a mode set, or the person marked away, between this
	// verdict and the ask can turn it (decision 5).
	DependsOnState bool
}

// allRules is every rule of the limits check, in the spec's order.
func allRules() []Rule {
	return []Rule{RuleHeader, RuleQuestionsPerCall, RuleOptions, RuleLabelWords, RuleMeaningSentences,
		RuleDecideLater, RuleNoBold, RuleNeverRecommended, RuleRegister, RuleNowAndChangeLater,
		RuleThingFirst, RuleEmDashListItem, RuleRows, RuleNoPreview}
}

// RefusingFinding reports whether a finding refuses the call on its own:
// every rule but the rows limit (iss-2610070637562567).
func RefusingFinding(f Finding) bool { return f.Rule != RuleRows }

// chipped reports whether any tab's header is in abcd's chip grammar.
func chipped(f Fields, l Limits) bool {
	for _, t := range f.Tabs {
		if _, ok := chipRole(t.Header, l); ok {
			return true
		}
	}
	return false
}

// ChipPerson is the person one chip role word names: the product thinker for
// the Product role, the technical facilitator for every other role (Tech and
// Setup; the setup interview asks how abcd is installed and run, the
// facilitator's half).
func ChipPerson(role string) Person {
	if role == ProductRole {
		return ProductThinker
	}
	return Facilitator
}

// callPerson is the person a chipped call names as a whole: the product
// thinker when any tab's chip names them, else the technical facilitator, so
// the status line never says only the facilitator is owed an answer the
// product thinker owes too.
func callPerson(f Fields, l Limits) Person {
	p := Facilitator
	for _, t := range f.Tabs {
		if role, ok := chipRole(t.Header, l); ok && ChipPerson(role) == ProductThinker {
			p = ProductThinker
		}
	}
	return p
}

// checkedTabs is the tabs the limits check reads field by field.
func checkedTabs(f Fields, l Limits) []Tab {
	return f.Tabs[:min(len(f.Tabs), max(l.QuestionsPerCall[1], 0))]
}

// Decide judges one question-tool call as the guard hook judges it, in the
// hook's order: the away record first (refuse-away before any limit), then
// the mode store where the badge shows, then whether the question is abcd's,
// then the limits. It writes nothing and reads no store: s carries what the
// surface read.
func Decide(f Fields, s State, l Limits) Verdict {
	v := Verdict{DependsOnState: s.Repo, ModeAfter: s.Mode}
	for _, t := range checkedTabs(f, l) {
		v.Rows = append(v.Rows, tabRows(t, l))
	}
	v.Chipped = chipped(f, l)
	if s.Away {
		v.Outcome = OutcomeRefuseAway
		return v
	}
	if s.AwayErr != nil {
		// The away record reads as not away, so the question is checked as
		// if the person were present; whether any rule should have been read
		// at all is unknown.
		v.Partial = true
		v.Unread = append(v.Unread, Unread{Store: StoreAway, Err: s.AwayErr, Rules: allRules()})
	}
	if s.Badge && s.ModeErr != nil {
		// The hook fails open here, whatever the question. A chipped
		// question needs no mode for its rules, so its findings are still
		// reported; for one without a chip, whether it is abcd's at all is
		// unknown, so no rule was applied.
		v.Partial = true
		v.Outcome = OutcomeUnchecked
		if !v.Chipped {
			v.Unread = append(v.Unread, Unread{Store: StoreMode, Err: s.ModeErr, Rules: allRules()})
			return v
		}
		v.Unread = append(v.Unread, Unread{Store: StoreMode, Err: s.ModeErr})
		v.Abcds = true
		v.Addressees = addressees(f, l, Unnamed)
		v.Findings = CheckLimits(f, l, Addressee{Person: Unnamed, Verbs: s.Verbs})
		return v
	}
	if !v.Chipped && s.Mode == Unnamed {
		// A question without a chip is abcd's only through the mode. Where
		// the mode store was never read because the root could not be found,
		// whether the mode names somebody is unknown (iss-2610110432159652).
		if s.RootErr != nil {
			v.Partial = true
			v.Unread = append(v.Unread, Unread{Store: StoreMode, Err: s.RootErr, Rules: allRules()})
			v.Outcome = OutcomeUnchecked
			return v
		}
		v.Outcome = OutcomeAdmitUnchecked
		return failedOpen(v)
	}
	v.Abcds = true
	// A chip names the addressee of its own tab, and the mode the gate sets
	// follows the chip, so a chipped question is checked against its chips
	// and not against a mode it is about to replace.
	who := s.Mode
	if v.Chipped {
		who = Unnamed
		v.ModeAfter = callPerson(f, l)
		v.ModeChanges = s.Badge && v.ModeAfter != s.Mode
	}
	v.Addressees = addressees(f, l, s.Mode)
	v.Findings = CheckLimits(f, l, Addressee{Person: who, Verbs: s.Verbs})
	refusing := false
	for _, fd := range v.Findings {
		if RefusingFinding(fd) {
			refusing = true
			break
		}
	}
	switch {
	case refusing:
		v.Outcome = OutcomeRefuse
		v.ModeAfter, v.ModeChanges = s.Mode, false
		return v
	case len(v.Findings) > 0:
		v.Outcome = OutcomeAdmitWithRowsNote
	default:
		v.Outcome = OutcomeAdmit
	}
	return failedOpen(v)
}

// failedOpen is the away record's fail-open applied to an admission the hook
// would otherwise make silently: with the away record unread, the hook admits
// on its loud, non-blocking status, which is the unchecked outcome. An
// admission whose rows note the hook writes stays that outcome, since the note
// is the hook's answer.
func failedOpen(v Verdict) Verdict {
	if !v.Partial {
		return v
	}
	if v.Outcome == OutcomeAdmit || v.Outcome == OutcomeAdmitUnchecked {
		v.Outcome = OutcomeUnchecked
	}
	return v
}

// addressees is the person each checked tab is for: its chip's role where it
// carries one, else the person the mode names.
func addressees(f Fields, l Limits, mode Person) []Person {
	tabs := checkedTabs(f, l)
	out := make([]Person, len(tabs))
	for i, t := range tabs {
		out[i] = mode
		if role, ok := chipRole(t.Header, l); ok {
			out[i] = ChipPerson(role)
		}
	}
	return out
}
