package ahoy

import "slices"

// SummaryItem explains one thing an install reports, for the person who ran it
// rather than for abcd's implementers (iss-164): what it is, why it matters for
// their work, and what, if anything, they should do. Refs names the exact
// entries of the result it explains (write paths, category names, step ids), so
// the plain words and the precise record can always be matched up.
type SummaryItem struct {
	What   string   `json:"what"`
	Why    string   `json:"why"`
	Action string   `json:"action"`
	Refs   []string `json:"refs,omitempty"`
}

// writeKind says what one write of the install is, in the terms the summary
// explains it in. Every write carries one (applyCtx.note takes it), so no write
// reaches the person as a bare path.
type writeKind string

const (
	writeSettings                writeKind = "settings"
	writeGitignore               writeKind = "gitignore"
	writeLocalTier               writeKind = "local-tier"
	writeNameGuard               writeKind = "name-guard"
	writePrivateNames            writeKind = "private-names"
	writeDocsCheck               writeKind = "docs-check"
	writeAttributionHook         writeKind = "attribution-hook"
	writeSessionStore            writeKind = "session-store"
	writeConventionsBlock        writeKind = "conventions-block"
	writeConventionsBlockRemoved writeKind = "conventions-block-removed"
	writeCommandEntry            writeKind = "command-entry"
	writeStatusLine              writeKind = "status-line"
	writeRouting                 writeKind = "routing"
	writeDrainRule               writeKind = "drain-rule"
	writeToolFileRetired         writeKind = "conventions-file-retired"
	writeRules                   writeKind = "rules"
	writeIdentityPin             writeKind = "identity-pin"
	writeGitIdentity             writeKind = "git-identity"
	writeArtefactKind            writeKind = "artefact-kind"
)

// allWriteKinds is every kind, in the order the summary lists them: the
// repository's own files first, then this machine, then the optional extras.
var allWriteKinds = []writeKind{
	writeSettings, writeGitignore, writeLocalTier, writeNameGuard, writePrivateNames,
	writeDocsCheck, writeAttributionHook, writeRules, writeConventionsBlock,
	writeConventionsBlockRemoved, writeToolFileRetired, writeGitIdentity, writeIdentityPin, writeArtefactKind, writeCommandEntry, writeSessionStore,
	writeStatusLine, writeRouting, writeDrainRule,
}

// writeKindHelp is the plain-language explanation of each kind of write.
var writeKindHelp = map[writeKind]SummaryItem{
	writeSettings: {
		What:   "Saved this repository's abcd settings in .abcd/config.json.",
		Why:    "Later runs read your answers from there instead of asking again.",
		Action: "Nothing. To change a setting, run abcd ahoy install with its option, for example --visibility public.",
	},
	writeGitignore: {
		What:   "Told git which abcd files stay on this machine, in a fenced block in .gitignore.",
		Why:    "Your private notes and scratch files, and for a public repository abcd's records, are never committed by accident.",
		Action: "Nothing. Leave the fenced block as it is; abcd keeps it up to date.",
	},
	writeLocalTier: {
		What:   "Created .abcd/.work.local/, a private working folder for this machine only.",
		Why:    "abcd keeps handover notes, logs and scratch work there, and git ignores it.",
		Action: "Nothing to do.",
	},
	writeNameGuard: {
		What:   "Added a check that runs before every commit and merge and stops one that contains a name you have banned.",
		Why:    "Names you want kept private, such as people, clients or machines, never reach the repository's history.",
		Action: "Nothing now. To ban a name, use abcd banlist add.",
	},
	writePrivateNames: {
		What:   "Created an empty private list of banned names for this machine, which is never committed.",
		Why:    "It is where names too sensitive to write into the repository go, and the pre-commit check reads it.",
		Action: "Add names to it with abcd banlist add when you have any.",
	},
	writeDocsCheck: {
		What:   "Set up the documentation check's settings in .abcd/docs-lint.json.",
		Why:    "The check flags writing in your documentation that is out of date or breaks the house style, and this file says how strict it is.",
		Action: "Nothing. Edit the file if you want a rule stricter or looser.",
	},
	writeAttributionHook: {
		What:   "Added a commit-message prompt that asks every commit to say whether an AI tool helped write it.",
		Why:    "It keeps an honest record of AI assistance in the repository's history.",
		Action: "Nothing. Answer the prompt when you commit.",
	},
	writeRules: {
		What:   "Created .abcd/rules.json, which switches abcd's working-rule reminders on or off for this repository.",
		Why:    "abcd reminds your AI assistant of the relevant rules only when a request touches them; this file lets the repository turn a rule off or add its own.",
		Action: "Nothing, unless you want to change a rule.",
	},
	writeConventionsBlock: {
		What:   "Added a short block describing how abcd works here to the conventions file you chose.",
		Why:    "AI assistants read that file at the start of every session, so they follow this repository's abcd conventions.",
		Action: "Nothing. Do not edit inside the block; abcd rewrites it on each run.",
	},
	writeConventionsBlockRemoved: {
		What:   "Removed abcd's block from a conventions file you no longer chose.",
		Why:    "abcd names itself only in the files you pick.",
		Action: "Nothing to do.",
	},
	writeToolFileRetired: {
		What:   "Removed an agent tool's own conventions file that only repeated AGENTS.md, as you answered.",
		Why:    "That tool read the file in place of AGENTS.md; without it, the tool reads AGENTS.md, the one conventions file abcd writes.",
		Action: "Commit the removal; abcd removed the file from your working tree only.",
	},
	writeGitIdentity: {
		What:   "Set the git name and email this repository commits under, in its own .git/config, to the identity you confirmed.",
		Why:    "Commits here were about to be made under a different identity, such as a leftover test account or an agent's, and the human is the author of record.",
		Action: "Nothing. Your global git settings are unchanged; to undo it, run git config --local --unset user.name and git config --local --unset user.email.",
	},
	writeIdentityPin: {
		What:   "Recorded the git name and email that commit to this repository.",
		Why:    "abcd can then warn when a commit is about to be made under a different identity, such as an agent's.",
		Action: "Nothing, unless the recorded name or email is wrong; then correct it in .abcd/config/identity.json, because running abcd ahoy install again leaves a recorded name and email as they are.",
	},
	writeArtefactKind: {
		What:   "Recorded what this repository releases, a plugin, a binary or an application, in .abcd/config/artefact.json.",
		Why:    "abcd's release commands read it to choose what to preview, check and set up, and refuse to guess it.",
		Action: "Nothing, unless the kind is wrong; then correct it in .abcd/config/artefact.json, because running abcd ahoy install again leaves a recorded kind as it is.",
	},
	writeCommandEntry: {
		What:   "Made the abcd command available in your terminal.",
		Why:    "You can run abcd directly, and the plugin's automatic checks can find it.",
		Action: "Nothing, unless a note below says its folder is not on your command path; then follow that note.",
	},
	writeSessionStore: {
		What:   "Registered this repository on this machine, in abcd's folder in your home directory.",
		Why:    "abcd can keep a redacted record of your working sessions for this repository outside the repository itself.",
		Action: "Nothing to do.",
	},
	writeStatusLine: {
		What:   "Set up abcd's status line in your AI assistant.",
		Why:    "In abcd repositories the line shows whether abcd is active and whose answer the work is waiting on.",
		Action: "Nothing. Switch parts of it off in ~/.abcd/statusline.json, or remove it with abcd ahoy uninstall.",
	},
	writeRouting: {
		What:   "Saved which size of AI model each of abcd's review steps asks for.",
		Why:    "Larger models cost more; the table keeps the expensive ones for the steps that need them.",
		Action: "Nothing. Edit the saved table if you want a different split.",
	},
	writeDrainRule: {
		What:   "Recorded which open issues abcd drain may fix without asking you, as a decision record in this repository.",
		Why:    "abcd drain refuses to run in a repository until it records that rule; this one is abcd's strict default.",
		Action: "Commit the record. Edit its drain_ fields to change the rule; abcd drain names any loosening.",
	},
}

// unexplainedWriteHelp is what a write reaches the person as when it carries no
// kind, or a kind this table has no entry for. Every write site passes a kind
// today, so it is a fallback that should never be seen; if it is, it must say
// plainly that the summary cannot describe the write, rather than borrow another
// kind's explanation, and it must still list the path.
var unexplainedWriteHelp = SummaryItem{
	What:   "Wrote a file this summary has no plain description for.",
	Why:    "abcd lists every file it writes, so none is left out even when it cannot say what the file is for.",
	Action: "Look at the file listed; abcd ahoy doctor shows what abcd expects to find in this repository.",
}

// declinedCategoryHelp explains each kind of change the person declined.
var declinedCategoryHelp = map[GapCategory]SummaryItem{
	SafeAutocreate: {
		What:   "You declined creating abcd's folders and starter files in this repository.",
		Why:    "abcd has nowhere to keep its records here until they exist.",
		Action: "Run abcd ahoy install again and answer y to create them.",
	},
	ConfigChange: {
		What:   "You declined saving this repository's abcd settings.",
		Why:    "abcd asks the same questions again on every run until they are saved.",
		Action: "Run abcd ahoy install again and answer y, or pass the settings as options.",
	},
	PluginOwned: {
		What:   "You declined writing abcd's description block into your conventions file.",
		Why:    "Your AI assistant will not be told how abcd works in this repository.",
		Action: "Run abcd ahoy install again and answer y if you want the block.",
	},
	Dependency: {
		What:   "You declined being shown how to install the optional extra scanners.",
		Why:    "abcd's own checks still run; only the deeper extra scans are missing.",
		Action: "Nothing, unless you want them; then run abcd ahoy install again and answer y.",
	},
	UserState: {
		What:   "You declined registering this repository on this machine.",
		Why:    "abcd cannot keep a record of your working sessions for this repository until it is registered.",
		Action: "Run abcd ahoy install again and answer y to register it.",
	},
	StatusLine: {
		What:   "You declined abcd's status line.",
		Why:    "Your assistant's status line stays exactly as it was.",
		Action: "Nothing. Run abcd ahoy install again if you change your mind.",
	},
	OracleRouting: {
		What:   "You declined abcd's suggested table of which AI model size each review step uses.",
		Why:    "Review steps use whatever model your assistant picks by default.",
		Action: "Nothing. Run abcd ahoy install again if you want the table.",
	},
	DrainRule: {
		What:   "You declined recording which open issues abcd drain may fix without asking you.",
		Why:    "abcd drain refuses to run in this repository until the rule is recorded.",
		Action: "Nothing, unless you want to drain; then run abcd ahoy install again and answer y.",
	},
	ConventionsFile: {
		What:   "You declined being asked about removing an agent tool's own conventions file that only repeats AGENTS.md.",
		Why:    "The file stays, and that tool keeps reading it in place of AGENTS.md.",
		Action: "Nothing, unless you want it gone; then run abcd ahoy install again at a terminal and answer y, or remove the file yourself.",
	},
}

// optionalSkippedHelp explains each optional step an unattended run left alone.
var optionalSkippedHelp = map[string]SummaryItem{
	OptionalPinGapID: {
		What:   "Recording who commits to this repository was left for you to confirm.",
		Why:    "It would record whatever git name and email happen to be set, which in an unattended run may be an agent's.",
		Action: "Check that git config user.name and git config user.email give your own name and email, then run abcd ahoy install without --yes and answer y to the config-change question; abcd records those two values.",
	},
	StatusLineOfferGapID: {
		What:   "abcd's status line was not set up.",
		Why:    "It changes a setting of your AI assistant that applies everywhere, so it needs your own yes.",
		Action: "Run abcd ahoy install without --yes and answer the status-line question.",
	},
	OracleRoutingMachineGapID: {
		What:   "abcd's suggested table of which AI model size each review step uses was not saved for this machine.",
		Why:    "The table decides which model, and so what cost, each review step asks for, so it needs your own yes.",
		Action: "Run abcd ahoy install without --yes and answer the question about the table.",
	},
	OracleRoutingRepoGapID: {
		What:   "abcd's suggested table of which AI model size each review step uses was not saved for this repository.",
		Why:    "The table decides which model, and so what cost, each review step asks for, so it needs your own yes.",
		Action: "Run abcd ahoy install without --yes and answer the question about the table.",
	},
	DrainRuleOfferGapID: {
		What:   "The rule for which open issues abcd drain may fix without asking you was not recorded.",
		Why:    "The rule decides what an unattended agent may change in this repository, so it needs your own yes; until it is recorded, abcd drain refuses to run here.",
		Action: "Run abcd ahoy install at a terminal, without --yes, and answer the question about the drain rule.",
	},
	ConventionsRetireGapID: {
		What:   "An agent tool's own conventions file that only repeats AGENTS.md was left in place.",
		Why:    "Removing a file from your project needs your own answer; until it goes, that tool reads it in place of AGENTS.md.",
		Action: "Run abcd ahoy install at a terminal, without --yes, and answer the question about the file, or remove it yourself.",
	},
}

// remainingHelp explains the required work a run left outstanding.
var remainingHelp = SummaryItem{
	What:   "Some required set-up steps are still not done.",
	Why:    "abcd does not work fully in this repository until they are.",
	Action: "Run abcd ahoy install again and answer y to each question; abcd ahoy doctor lists the steps one by one.",
}

// statusHeadline is the one sentence that opens the summary for each status.
var statusHeadline = map[string]string{
	"already_up_to_date": "abcd was already set up in this repository, so nothing needed changing.",
	"clean":              "abcd is set up in this repository.",
	"partial":            "abcd is only partly set up in this repository; the items below say what is missing and how to finish.",
	"aborted":            "Nothing was set up: abcd was not added to this folder.",
	"refused":            "Nothing was set up: abcd stopped before changing anything, and the notes say why.",
}

// explain composes Headline and Summary from the exact record the rest of the
// result carries. It adds nothing the record does not say; it says it plainly.
//
// It is also where the result is sealed for rendering: every outcome, the early
// returns included, passes through here once, so the record's list fields are
// seeded non-nil here and an empty one renders [] rather than null
// (iss-2609120447487070).
func (r *InstallResult) explain() {
	r.Headline = statusHeadline[r.Status]
	r.Summary = []SummaryItem{}
	for _, list := range []*[]string{&r.Warnings, &r.Writes, &r.Remaining, &r.DeclinedCategories} {
		if *list == nil {
			*list = []string{}
		}
	}

	refs := map[writeKind][]string{}
	var unexplained []string
	for i, w := range r.Writes {
		var k writeKind
		if i < len(r.writeKinds) {
			k = r.writeKinds[i]
		}
		if _, known := writeKindHelp[k]; !known || !slices.Contains(allWriteKinds, k) {
			if !slices.Contains(unexplained, w) {
				unexplained = append(unexplained, w)
			}
			continue
		}
		if !slices.Contains(refs[k], w) {
			refs[k] = append(refs[k], w)
		}
	}
	for _, k := range allWriteKinds {
		if len(refs[k]) == 0 {
			continue
		}
		it := writeKindHelp[k]
		it.Refs = refs[k]
		r.Summary = append(r.Summary, it)
	}
	if len(unexplained) > 0 {
		it := unexplainedWriteHelp
		it.Refs = unexplained
		r.Summary = append(r.Summary, it)
	}
	for _, c := range r.DeclinedCategories {
		it, ok := declinedCategoryHelp[GapCategory(c)]
		if !ok {
			continue
		}
		it.Refs = []string{c}
		r.Summary = append(r.Summary, it)
	}
	if len(r.Remaining) > 0 {
		it := remainingHelp
		it.Refs = append([]string(nil), r.Remaining...)
		r.Summary = append(r.Summary, it)
	}
	for _, id := range r.OptionalSkipped {
		it, ok := optionalSkippedHelp[id]
		if !ok {
			continue
		}
		it.Refs = []string{id}
		r.Summary = append(r.Summary, it)
	}
}
