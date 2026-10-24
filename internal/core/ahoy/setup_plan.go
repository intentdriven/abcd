package ahoy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// The plan screen (itd-2610240823055889, criteria 2 and 5): setup checks
// everything first, then shows one screen listing every change it would make
// and every default it would take, each default with where to change it, and
// asks once: proceed, customise or cancel. Proceed approves every kind of
// change and takes the defaults listed; customise asks each kind of change and
// each value in turn, as setup asked before the plan existed; cancel writes
// nothing. The steps only a person can approve are listed under "asks you"
// and asked as their own questions whatever the answer (person_only.go).

// PlanQuestionID is the plan screen's id: the id an answers file answers it
// by. It replaces the approve.<category> ids of the approvals the plan asks
// for at once; an answers file naming one of those is refused, naming this.
const PlanQuestionID = approvePrefix + "plan"

// PlanChoice is the answer to the plan screen.
type PlanChoice string

// The plan screen's answers.
const (
	PlanProceed   PlanChoice = "proceed"
	PlanCustomise PlanChoice = "customise"
	PlanCancel    PlanChoice = "cancel"
)

// planChoices are the plan's answers in the order it offers them.
var planChoices = []PlanChoice{PlanProceed, PlanCustomise, PlanCancel}

// SetupPlan is what setup would do in one repository, as the plan screen
// lists it: the changes grouped by where they land, the defaults taken, and
// the questions that follow whatever the answer.
type SetupPlan struct {
	Changes  []PlanGroup   `json:"changes"`
	Defaults []PlanDefault `json:"defaults"`
	AsksYou  []string      `json:"asks_you"`
}

// PlanGroup is the changes that land in one place.
type PlanGroup struct {
	Where string   `json:"where"`
	Items []string `json:"items"`
}

// PlanDefault is one default the plan takes: what it decides, the value, and
// where to change it.
type PlanDefault struct {
	Key        string `json:"key"`
	What       string `json:"what"`
	Value      string `json:"value"`
	ChangeWith string `json:"change_with"`
}

// PlanPrompter is a prompter that puts the plan screen. asked is false where
// it does not (a piped answer stream, whose answers line up with the
// approvals one by one), and the run then asks as it did before the plan.
type PlanPrompter interface {
	Prompter
	Plan(p SetupPlan) (choice PlanChoice, asked bool)
}

// planWhere names each scope's group on the screen.
var planWhere = map[string]string{
	"repo":    "In this repository",
	"machine": "On this machine",
	"user":    "In your home folder",
}

// planDefaultFor is each default the plan can take, keyed by the gap whose
// presence makes setup ask it.
var planDefaultFor = []struct {
	gap string
	def PlanDefault
}{
	{"config.visibility_missing", PlanDefault{Key: "visibility", What: "abcd's records", Value: "private: committed with your code",
		ChangeWith: "abcd ahoy install --visibility public"}},
	{"config.docs_target_missing", PlanDefault{Key: "docs_target", What: "the abcd block in AGENTS.md", Value: docsTargetDefault + ": no block",
		ChangeWith: "abcd ahoy install --docs-target agents_md"}},
	{"config.scan_deep_missing", PlanDefault{Key: "scan_deep", What: "the deep secret scan", Value: "off",
		ChangeWith: "abcd ahoy install --scan-deep true"}},
	{"banlist.public_family_missing", PlanDefault{Key: emDashPromptKey, What: "an em dash in a list item", Value: "a warning",
		ChangeWith: banlist.PublicConfigRelPath}},
	{ArtefactMissingGapID, PlanDefault{Key: artefactKindKey, What: "what this repository releases", Value: artefactKindDefault,
		ChangeWith: ".abcd/config/artefact.json"}},
	{OracleRoutingMachineGapID, PlanDefault{Key: OracleRoutingMachineGapID, What: "model routing for this machine", Value: "abcd's suggested table",
		ChangeWith: abcdhome.Display(layered.OracleRouting.MachineRel)}},
	{OracleRoutingRepoGapID, PlanDefault{Key: OracleRoutingRepoGapID, What: "model routing for this repository", Value: "abcd's suggested table",
		ChangeWith: layered.OracleRouting.RepoRel}},
	{DrainRuleOfferGapID, PlanDefault{Key: DrainRuleOfferGapID, What: "which issues abcd drain fixes alone", Value: "abcd's strict baseline",
		ChangeWith: drainrule.ADRsRelDir + "/"}},
	{ReadmeOfferGapID, PlanDefault{Key: ReadmePromptKey, What: "README parts abcd keeps current", Value: "the licence line and the links",
		ChangeWith: "abcd ahoy install --readme-parts <list>|none"}},
	{StatusLineOfferGapID, PlanDefault{Key: elementPromptPrefix + "*", What: "the status line's parts", Value: "all on",
		ChangeWith: statusline.SettingsDisplay}},
	{ConventionsRetireGapID, PlanDefault{Key: conventionsRetirePromptPrefix + "*", What: "a tool's file that only repeats AGENTS.md", Value: "kept",
		ChangeWith: "remove it yourself, or choose customise"}},
}

// buildSetupPlan composes the plan for the repository det describes, given
// what the run will ask of the person on its own (asksYou).
func buildSetupPlan(det DetectionResult, asksYou []string) SetupPlan {
	present := gapIDSet(det.Gaps)
	plan := SetupPlan{Changes: []PlanGroup{}, Defaults: []PlanDefault{}, AsksYou: asksYou}
	if plan.AsksYou == nil {
		plan.AsksYou = []string{}
	}
	if det.FolderKind == UnmanagedRepo {
		plan.Changes = append(plan.Changes, PlanGroup{Where: planWhere["repo"], Items: []string{"adopt this repository into abcd"}})
	}
	offers := map[string]bool{}
	for _, d := range planDefaultFor {
		offers[d.gap] = true
	}
	for _, scope := range []string{"repo", "machine", "user"} {
		var items []string
		for _, g := range det.Gaps {
			if g.Scope != scope || !g.Resolvable || personOnlyGap(g.ID) || offers[g.ID] || g.Category == Readme {
				continue
			}
			if t := strings.Join(strings.Fields(g.Title), " "); t != "" && !slices.Contains(items, t) {
				items = append(items, t)
			}
		}
		if len(items) == 0 {
			continue
		}
		if i := slices.IndexFunc(plan.Changes, func(pg PlanGroup) bool { return pg.Where == planWhere[scope] }); i >= 0 {
			plan.Changes[i].Items = append(plan.Changes[i].Items, items...)
			continue
		}
		plan.Changes = append(plan.Changes, PlanGroup{Where: planWhere[scope], Items: items})
	}
	for _, d := range planDefaultFor {
		if present[d.gap] {
			plan.Defaults = append(plan.Defaults, d.def)
		}
	}
	return plan
}

// planAsk is the plan screen's one line.
const planAsk = "Proceed with this plan?"

// SetupPlanQuestion is the plan screen as the shared question type: the plan
// as its material (the changes grouped by where they land, the defaults each
// with where to change it, and the questions that follow), the one line, and
// the three answers, deciding later cancelling.
func SetupPlanQuestion(n int, p SetupPlan) question.Question {
	var material []question.Block
	for _, g := range p.Changes {
		material = append(material, question.Block{Kind: question.KindParagraph, Text: g.Where + ", setup:"},
			question.Block{Kind: question.KindList, Items: append([]string(nil), g.Items...)})
	}
	if len(p.Defaults) > 0 {
		items := make([]string, len(p.Defaults))
		for i, d := range p.Defaults {
			items[i] = d.What + ": " + d.Value + " (change: " + d.ChangeWith + ")"
		}
		material = append(material, question.Block{Kind: question.KindParagraph, Text: "Defaults it takes:"},
			question.Block{Kind: question.KindList, Items: items})
	}
	if len(p.AsksYou) > 0 {
		material = append(material, question.Block{Kind: question.KindParagraph, Text: "Then it asks you, one question each:"},
			question.Block{Kind: question.KindList, Items: append([]string(nil), p.AsksYou...)})
	}
	if len(material) == 0 {
		material = []question.Block{{Kind: question.KindParagraph, Text: "Setup has nothing to change here but the questions it asks you."}}
	}
	return question.Question{
		ID:       PlanQuestionID,
		Chip:     setupChip(n),
		Material: material,
		Ask:      planAsk,
		Options: []question.Option{
			{Value: string(PlanProceed), Label: "Proceed", Meaning: "Makes every change and takes every default on the plan; the questions it lists are still asked, one each."},
			{Value: string(PlanCustomise), Label: "Customise", Meaning: "Asks about each kind of change and each value in turn before anything is written."},
			{Value: string(PlanCancel), Label: "Cancel", Meaning: "Writes nothing and ends setup."},
		},
		Later: question.Option{Value: SetupLaterValue, Label: question.Default.LaterLabels[0],
			Meaning: "Cancels for now and writes nothing, so the next setup run shows the plan again."},
		Now:         question.Default.NotApplicable,
		ChangeLater: "each default's own line on the plan says where to change it",
	}
}

// PlanChoiceOf is the plan answer a recorded value stands for: deciding later
// cancels, and anything the plan does not offer is no answer.
func PlanChoiceOf(value string) (PlanChoice, bool) {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == SetupLaterValue {
		return PlanCancel, true
	}
	for _, c := range planChoices {
		if v == string(c) || (c == PlanCustomise && v == "customize") {
			return c, true
		}
	}
	return "", false
}

// LegacyApprovalRefusal is the refusal of an answers file that answers an
// approval of one kind of change by its old id: the plan asks for them all at
// once now, under PlanQuestionID. "" for any other id.
func LegacyApprovalRefusal(id string) string {
	if !strings.HasPrefix(id, approvePrefix) || id == PlanQuestionID {
		return ""
	}
	return fmt.Sprintf("the answers file answers %s, an approval setup no longer asks: one plan screen approves every kind of change, "+
		`so answer it with {"id": %q, "value": "proceed|customise|cancel"} instead`, id, PlanQuestionID)
}

// DefaultRecorder is a prompter that records the answer the plan took for a
// question it did not put, as the question was and as if it had been asked
// and answered so: a run's record then says what every answer was, the
// defaults the plan took included, and a replay from that record answers
// each question the same way.
type DefaultRecorder interface {
	RecordDefaultConfirm(text string, yes bool)
	RecordDefaultPrompt(key string, choices []string, def, value string)
	RecordDefaultHelped(h PromptHelp, choices []string, def, value string)
}

// planDefaults is the prompter of a run whose plan was proceeded with (or,
// off a terminal, customised): every approval of a kind of change is answered
// yes, and, where takeDefaults is
// set (proceed at a terminal), every value and offer the plan lists takes its
// default without being asked. The steps only a person can approve, and any
// question the plan does not list, reach inner as before.
type planDefaults struct {
	inner        Prompter
	takeDefaults bool
}

// Confirm answers an approval yes, an offer the plan lists by its default,
// and puts everything else to inner.
func (p *planDefaults) Confirm(text string) bool {
	id := setupConfirmID(text)
	switch {
	case strings.HasPrefix(id, approvePrefix), id == "adopt":
		return true
	case p.takeDefaults && (id == OracleRoutingMachineGapID || id == OracleRoutingRepoGapID || id == DrainRuleOfferGapID):
		if r, ok := p.inner.(DefaultRecorder); ok {
			r.RecordDefaultConfirm(text, true)
		}
		return true
	}
	return p.inner.Confirm(text)
}

// Prompt takes the plan's default for a value it lists, else puts it to inner.
func (p *planDefaults) Prompt(key string, choices []string, def string) string {
	if v, ok := p.planValue(key, def); ok {
		if r, ok := p.inner.(DefaultRecorder); ok {
			r.RecordDefaultPrompt(key, choices, def, v)
		}
		return v
	}
	return p.inner.Prompt(key, choices, def)
}

// PromptHelped is Prompt for a question with help built for this run.
func (p *planDefaults) PromptHelped(h PromptHelp, choices []string, def string) string {
	if v, ok := p.planValue(h.Key, def); ok {
		if r, ok := p.inner.(DefaultRecorder); ok {
			r.RecordDefaultHelped(h, choices, def, v)
		}
		return v
	}
	return promptHelped(p.inner, h, choices, def)
}

// AtTerminal is inner's.
func (p *planDefaults) AtTerminal() bool { return atTerminal(p.inner) }

// planValue is the default the plan takes for the value question key, whose
// own default is def.
func (p *planDefaults) planValue(key, def string) (string, bool) {
	if !p.takeDefaults {
		return "", false
	}
	switch {
	case key == "visibility":
		return "private", true
	case key == "docs_target":
		return docsTargetDefault, true
	case key == "scan_deep":
		return scanDeepDefault, true
	case key == emDashPromptKey:
		return "warning", true
	case key == artefactKindKey:
		return artefactKindDefault, true
	case key == ReadmePromptKey:
		return "yes", true
	case strings.HasPrefix(key, readmePartKeyPrefix):
		return def, true
	case strings.HasPrefix(key, elementPromptPrefix):
		return "on", true
	case strings.HasPrefix(key, conventionsRetirePromptPrefix):
		return retireKeep, true
	}
	return "", false
}

// askSetupPlan puts the plan for det to p when p puts one and the run asks
// its questions (no --yes, no categories answered in advance, no
// --only-waiting). It returns the prompter the rest of the run asks through,
// the answer, and whether the plan was put at all.
func askSetupPlan(det DetectionResult, opts InstallOptions, asksYou []string, p Prompter) (Prompter, PlanChoice, bool) {
	if opts.Yes || opts.ApprovedCategories != nil || opts.OnlyWaiting {
		return p, "", false
	}
	pp, ok := p.(PlanPrompter)
	if !ok {
		return p, "", false
	}
	choice, asked := pp.Plan(buildSetupPlan(det, asksYou))
	if !asked {
		return p, "", false
	}
	switch {
	case choice == PlanProceed:
		// At a terminal the plan's defaults are taken; from an answers file
		// every value still comes from the file.
		return &planDefaults{inner: p, takeDefaults: atTerminal(p)}, choice, true
	case choice == PlanCustomise && atTerminal(p):
		// Each kind of change and each value is asked in turn, as setup
		// asked before the plan existed.
		return p, choice, true
	case choice == PlanCustomise:
		// Off a terminal there is nothing to customise but the file's own
		// answers: every kind of change is approved, each value the file's.
		return &planDefaults{inner: p}, choice, true
	}
	return p, PlanCancel, true
}
