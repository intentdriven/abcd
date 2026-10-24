package ahoy

import (
	"slices"
	"strings"
)

// The report setup ends on (itd-2610240823055889, criteria 7 and 8): three
// short parts. Done is one line per change, with where it landed; needs you
// is a numbered checklist, each item the exact step and why; next is one
// step. A repository already fully set up is reported in one line. The exact
// record (writes, notes, remaining gap ids) stays on the result beside it.

// SetupReport is the three parts of the report.
type SetupReport struct {
	Done     []string     `json:"done"`
	NeedsYou []ReportStep `json:"needs_you"`
	Next     string       `json:"next"`
	// OneLine is the whole report of a repository already fully set up, and
	// "" otherwise: a front door prints it alone.
	OneLine string `json:"one_line,omitempty"`
}

// ReportStep is one item of the needs-you checklist: the exact step and why.
type ReportStep struct {
	Step string `json:"step"`
	Why  string `json:"why"`
}

// UpToDateLine is the one line a repository already fully set up is reported
// in.
const UpToDateLine = "abcd is already set up here; nothing changed."

// doneLabel is each kind of write in the words of one done line, its path
// following.
var doneLabel = map[writeKind]string{
	writeSettings:                "saved this repository's abcd settings",
	writeSetupRecord:             "recorded abcd's set-up details",
	writeGitignore:               "kept abcd's private files out of git",
	writeLocalTier:               "created the private working folder",
	writeRecordTiers:             "created a shared record folder",
	writeDecisionsFolder:         "started the decisions folder",
	writeDevelopmentReadme:       "started the lasting record's README",
	writeNameGuard:               "added the banned-names check",
	writePrivateNames:            "created this machine's banned-names list",
	writeDocsCheck:               "set up the documentation check",
	writeAttributionHook:         "added the AI-assistance commit prompt",
	writeOutboundHook:            "added the outbound commit-message check",
	writeOutboundForge:           "switched on the outbound post check",
	writeOutboundBinary:          "named the verified abcd for the commit check",
	writeOutboundShim:            "switched on the commit-message check",
	writeOutboundChain:           "saved abcd's commit-message check",
	writeSessionStore:            "registered this repository on this machine",
	writeConventionsBlock:        "added abcd's block",
	writeConventionsBlockRemoved: "removed abcd's block",
	writeCommandEntry:            "put abcd on your PATH",
	writeStatusLine:              "set up abcd's status line",
	writeStatusLineBackup:        "kept a copy of your assistant's settings",
	writeRouting:                 "saved the model routing table",
	writeDrainRule:               "recorded the drain rule",
	writeToolFileRetired:         "removed a file that only repeated AGENTS.md",
	writeRules:                   "created the working-rules switches",
	writeIdentityRecord:          "recorded who you commit as, on this machine",
	writeIdentityPinRemoved:      "removed the identity the repository named",
	writeProductionMode:          "kept the repository's record-writing default",
	writeShellPath:               "added abcd's folder to your shell settings",
	writeShellPathBackup:         "kept a copy of your shell settings",
	writeGitIdentity:             "set who commits here",
	writeArtefactKind:            "recorded what this repository releases",
	writeReadmeParts:             "wrote the README parts you chose",
	writeReadmeCreated:           "created README.md with the parts you chose",
	writeReadmePartsRemoved:      "removed README parts no longer chosen",
}

// composeReport builds the report from the exact record and the steps the
// run found left for the person (r.needs), so it says nothing the record does
// not.
func (r *InstallResult) composeReport() {
	rep := SetupReport{Done: []string{}, NeedsYou: []ReportStep{}}
	for i, w := range r.Writes {
		label := "wrote"
		if i < len(r.writeKinds) {
			if l, ok := doneLabel[r.writeKinds[i]]; ok {
				label = l
			}
		}
		line := label + ": " + w
		if !slices.Contains(rep.Done, line) {
			rep.Done = append(rep.Done, line)
		}
	}
	for _, c := range r.Changes {
		rep.Done = append(rep.Done, "changed: "+c)
	}
	add := func(step, why string) {
		step, why = strings.TrimSpace(step), strings.TrimSpace(why)
		if step == "" {
			return
		}
		for _, s := range rep.NeedsYou {
			if s.Step == step {
				return
			}
		}
		rep.NeedsYou = append(rep.NeedsYou, ReportStep{Step: step, Why: why})
	}
	for _, s := range r.needs {
		add(s.Step, s.Why)
	}
	for _, id := range r.Remaining {
		if id == UnverifiedCopyGapID {
			it := unverifiedCopyHelp(r.unverifiedCheck, r.unverifiedWhy)
			add(it.Action, it.What)
		}
	}
	for _, c := range r.DeclinedCategories {
		if it, ok := declinedCategoryHelp[GapCategory(c)]; ok && !strings.HasPrefix(it.Action, "Nothing") {
			add(it.Action, it.Why)
		}
	}
	for _, id := range r.OptionalSkipped {
		it, ok := optionalSkippedHelp[id]
		if !ok {
			continue
		}
		if id == ReadmeOfferGapID && r.readmeAwaitsConfig {
			it = readmeAwaitsConfigHelp
		}
		add(it.Action, it.What)
	}
	if len(r.needs) == 0 && slices.ContainsFunc(r.Remaining, func(id string) bool { return id != UnverifiedCopyGapID }) {
		add(remainingHelp.Action, remainingHelp.Why)
	}
	switch {
	case r.Status == "refused":
		rep.Next = "Do what the message above names, then run abcd ahoy install again."
	case r.Status == "aborted":
		rep.Next = "Run abcd ahoy install again when you want abcd set up here."
	case len(rep.NeedsYou) > 0:
		rep.Next = "Do step 1 above, then run abcd ahoy install again."
	default:
		rep.Next = "Open this repository in your AI coding assistant; run abcd here to see the status board."
	}
	if r.Status == "already_up_to_date" && len(rep.NeedsYou) == 0 && len(r.Warnings) == 0 {
		rep.OneLine = UpToDateLine
	}
	r.Report = rep
}

// remainingSteps is the needs-you checklist for the gaps a run left: each
// required gap's own fix, a step only the person can take (a folder not on
// PATH, an entry abcd will not replace) included, and each required tool
// still missing, as the step that installs it by hand. Ordered as the gaps
// are.
func remainingSteps(cwd string, gaps []Gap) []ReportStep {
	var out []ReportStep
	for _, g := range gaps {
		switch {
		case g.Category == Dependency && g.Tool != nil:
			// An optional tool is offered at a terminal, one question each,
			// and is not a step the person owes.
			if !g.Required || onPath(cwd, g.Tool.Tool) {
				continue
			}
			out = append(out, ReportStep{Step: "Install " + g.Tool.Tool + " yourself: " + g.Tool.StepText(),
				Why: g.Tool.Tool + " is optional; " + strings.TrimSuffix(g.Title, ".") + "."})
		case g.Required && g.FixHint != "" && !optionalGapID(g.ID):
			out = append(out, ReportStep{Step: g.FixHint, Why: strings.TrimSuffix(reportFirstSentence(g.Detail, g.Title), ".") + "."})
		}
	}
	return out
}

// optionalGapID reports whether id is one of the offers the report lists
// from OptionalSkipped instead.
func optionalGapID(id string) bool { return slices.Contains(optionalGapIDs, id) }

// reportFirstSentence is the first sentence of s, or fallback when s is empty.
func reportFirstSentence(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}
