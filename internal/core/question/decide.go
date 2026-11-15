package question

import "slices"

// The one decision the question gate makes (spc-2610301917245442, step 1;
// itd-2610230756259215 decisions 2 and 3). The guard hook and the read-only
// `abcd guard question` both call Decide over the same field view and the same
// state, so a draft the verb passes cannot be refused by a hook of the same
// build: two callers of one function cannot drift apart as two copies of the
// same rules would. The stores that feed State are read in the surface, and
// the writes an admission makes (the open-question marker, the mode set from
// a chip) stay in the hook, after the decision, where the verb never reaches
// them.
//
// A question is abcd's by its header alone (spc-2610301917245442, step 3;
// itd-2610230756259215 decision 36): a header in abcd's grammar, the number
// form ("Q1", "Q1a", "Q2/5") or the older form opened by a role word
// ("Product Q2"). The mode no longer makes a question abcd's: a question with
// no header in the grammar is another tool's in every mode state.

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
	// OutcomeAdmitUnchecked is a question that is not abcd's, no header being
	// in abcd's grammar, whatever the mode: the hook lets it run unchecked and
	// unmarked.
	OutcomeAdmitUnchecked Outcome = "admit-unchecked"
	// OutcomeUnchecked is a question the hook admits on its loud,
	// non-blocking status because a store the decision needed could not be
	// read, and the verdict is partial. abcd's question has been checked
	// (Abcds is set), and only one the gate cannot tell is abcd's runs with
	// no rule applied.
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
	// whom a number-form header is for, and so the product thinker's register
	// rule on its tab, and an older-form chip sets it.
	StoreMode Store = "mode"
	// StoreLimits is the question limits a repository's .abcd/config.json
	// and the person's own ~/.abcd.noindex/config.json set
	// (itd-2610301914350541). It is read for abcd's question only; a layer
	// of it that could not be read leaves abcd's question checked at the
	// defaults, so the rules it decides are applied, at the wrong figures
	// for a repository that set its own.
	StoreLimits Store = "limits"
)

// LimitsRefusal is a question limit the resolver read and refused: a value
// outside its bounds, not a whole number, a key the question namespace does
// not claim, or a personal value looser than the limit in force
// (itd-2610301914350541 decision 7). While it holds, every question of
// abcd's is refused and none is checked: nothing falls through to the
// defaults. The fields are the resolver's; Key, Value, Bounds and File are
// empty where the refusal names no single key (an unclaimed key is named in
// Reason alone), and Limit and LimitFile are set only for a personal value
// that loosens.
type LimitsRefusal struct {
	Key       string
	Value     string
	Bounds    string
	File      string
	Limit     int
	LimitFile string
	// Reason is the resolver's own refusal, naming the file; the surface
	// scrubs it before it is shown.
	Reason string
}

// limitsRules is the rules whose figures the question limits set: the label
// words, the description sentences and the rows.
func limitsRules() []Rule { return []Rule{RuleLabelWords, RuleMeaningSentences, RuleRows} }

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
	// Limits is the question limits in force, as the resolver returned them
	// (itd-2610301914350541): the defaults with what the repository's file
	// set and the person's own file tightened. Where a layer was unread, and
	// where a value was refused, it is the defaults, read then only for the
	// header grammar and the rows split. The surface passes it to Decide.
	Limits Limits
	// LimitsSource is where each figure of Limits came from, one origin per
	// key in Bounds' order: "bundled", or the file that set it.
	LimitsSource []string
	// LimitsRefused is set when a question limit was read and refused.
	LimitsRefused *LimitsRefusal
	// LimitsErr is why a layer of the question limits could not be read, and
	// LimitsFile the file it names; abcd's question is then checked at the
	// defaults.
	LimitsErr  error
	LimitsFile string
}

// Unread is one store the decision needed and could not read, with why and
// the rules it therefore could not apply.
type Unread struct {
	Store Store
	Err   error
	Rules []Rule
	// File is the file that could not be read, where the store is one: set
	// for the question limits.
	File string
}

// Verdict is the decision: the hook's outcome, every finding, each tab's
// addressee, the rows of every checked tab, and what the decision could not
// read.
type Verdict struct {
	Outcome Outcome
	// Abcds is true when the question is abcd's: a header in abcd's grammar,
	// either form.
	Abcds bool
	// Chipped is true when a header is in the older form, a role word before
	// the number, which names its tab's person and sets the mode.
	Chipped bool
	// Findings is every finding of the limits check, in CheckLimits's order;
	// empty when no rule was read (away, or not abcd's question).
	Findings []Finding
	// Addressees is the person each checked tab is for: the chip's role for
	// an older-form tab, the mode's person otherwise.
	Addressees []Person
	// ModeAfter is the mode an admitted question leaves where the badge
	// shows: for a call with an older-form chip, the product thinker when any
	// checked tab's addressee is, else the facilitator (callPerson); the
	// current mode otherwise, so a number-form question leaves the mode the
	// agent set before asking (decision 37). ModeChanges is true when it
	// differs from State.Mode (decision 15).
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
	// Refused is the question limit that refused the call, set with the
	// refuse outcome and no finding: the configuration, not the draft, is
	// what must be fixed (itd-2610301914350541 decision 7).
	Refused *LimitsRefusal
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

// abcds reports whether any tab's header is in abcd's grammar, either form.
func abcds(f Fields, l Limits) bool {
	for _, t := range f.Tabs {
		if isAbcds(t.Header, l) {
			return true
		}
	}
	return false
}

// chipped reports whether any tab's header is in the older form.
func chipped(f Fields, l Limits) bool {
	for _, t := range f.Tabs {
		if _, ok := chipRole(t.Header, l); ok {
			return true
		}
	}
	return false
}

// numbered reports whether any tab's header is in the number form, whose
// person is the one the mode names.
func numbered(f Fields, l Limits) bool {
	for _, t := range f.Tabs {
		if _, ok := chipRole(t.Header, l); isAbcds(t.Header, l) && !ok {
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

// callPerson is the person a chipped call names as a whole, read from the
// addressee of every checked tab: the product thinker when any tab is theirs,
// whether by its own chip or, for a number-form tab, by the mode in force,
// else the technical facilitator. So the status line never names only the
// facilitator while the product thinker owes an answer too: under the
// product thinker's mode, "Tech Q1" beside "Q2" leaves the badge on the
// product thinker, whom the Q2 tab is for.
func callPerson(addressees []Person) Person {
	if slices.Contains(addressees, ProductThinker) {
		return ProductThinker
	}
	return Facilitator
}

// checkedTabs is the tabs the limits check reads field by field.
func checkedTabs(f Fields, l Limits) []Tab {
	return f.Tabs[:min(len(f.Tabs), max(l.QuestionsPerCall[1], 0))]
}

// Decide judges one question-tool call as the guard hook judges it, in the
// hook's order: the away record first (refuse-away before any limit), then
// whether the question is abcd's, by its header alone, then the mode store
// where a number header needs it, then the limits. It writes nothing and
// reads no store: s carries what the surface read, and l is the limits the
// call is judged at, s.Limits as the surface resolved them.
//
// The question limits' outcome is read only for abcd's question
// (itd-2610301914350541): another tool's runs unchecked whatever either file
// says. For abcd's, a refused limit is the refuse outcome with no finding and
// Refused set, and an unread layer leaves the call checked at l (the
// defaults, on that path), partial and naming the file, so a configuration
// that cannot be read never lets a question through unchecked. A mode store
// that cannot be read leaves abcd's question checked too, partial and naming
// the store: an older-form chip names whom its tab is for and needs no mode,
// and a number header loses only the register rule, which follows the mode
// (spc-2610301917245442, step 3).
func Decide(f Fields, s State, l Limits) Verdict {
	v := Verdict{DependsOnState: s.Repo, ModeAfter: s.Mode}
	for _, t := range checkedTabs(f, l) {
		v.Rows = append(v.Rows, tabRows(t, l))
	}
	v.Abcds = abcds(f, l)
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
	if !v.Abcds {
		// No header in abcd's grammar: another tool's question, whatever the
		// mode names (decision 36), so the mode store is not consulted.
		v.Outcome = OutcomeAdmitUnchecked
		return failedOpen(v)
	}
	numbered := numbered(f, l)
	// The mode store matters to abcd's question in two ways: an older-form
	// chip sets it on admission, and a number header's register rule follows
	// it. Where the badge shows and the store cannot be read, or where the
	// root could not be found for a number header, the verdict is partial,
	// naming the register rule only for a number header.
	var modeErr error
	switch {
	case s.Badge && s.ModeErr != nil:
		modeErr = s.ModeErr
	case numbered && s.RootErr != nil:
		modeErr = s.RootErr
	}
	if modeErr != nil {
		v.Partial = true
		u := Unread{Store: StoreMode, Err: modeErr}
		if numbered {
			u.Rules = []Rule{RuleRegister}
		}
		v.Unread = append(v.Unread, u)
	}
	// The register rule is decided per tab (forProductThinker): an
	// older-form tab by its own role word, never by a mode it is about to
	// replace, and every other tab by the mode's person, passed where the
	// badge shows and the store was read; elsewhere nobody is named.
	who := Unnamed
	if s.Badge && modeErr == nil {
		who = s.Mode
	}
	v.Addressees = addressees(f, l, who)
	if v.Chipped {
		v.ModeAfter = callPerson(v.Addressees)
		v.ModeChanges = s.Badge && v.ModeAfter != s.Mode
	}
	if modeErr != nil {
		// The mode cannot be read, so the hook sets none from the chip.
		v.ModeAfter, v.ModeChanges = s.Mode, false
	}
	if s.LimitsRefused != nil {
		v.Outcome, v.Refused = OutcomeRefuse, s.LimitsRefused
		v.ModeAfter, v.ModeChanges = s.Mode, false
		return v
	}
	v = limitsUnread(v, s)
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

// limitsUnread marks v partial where a layer of the question limits could not
// be read, naming the file and the rules whose figures it would have set.
// Those rules are still applied, at the defaults the surface passes.
func limitsUnread(v Verdict, s State) Verdict {
	if s.LimitsErr == nil {
		return v
	}
	v.Partial = true
	v.Unread = append(v.Unread, Unread{Store: StoreLimits, Err: s.LimitsErr, Rules: limitsRules(), File: s.LimitsFile})
	return v
}

// failedOpen applies an unread store to an admission the hook would otherwise
// make silently: with the away record, the mode store or a layer of the
// question limits unread, the hook admits on its loud, non-blocking status,
// which is the unchecked outcome. An
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

// addressees is the person each checked tab is for, as its register rule was
// judged: its older-form chip's role where it carries one, else the person
// the mode names (who, Unnamed where the mode was not read).
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
