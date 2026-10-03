package ahoy

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/statusline"
)

// ChoiceHelp explains one answer to an install question in plain language:
// what choosing it does to the person's repository or machine, and what it
// asks of them (keys, tools, cost) where it asks anything.
type ChoiceHelp struct {
	Value   string `json:"value"`
	Meaning string `json:"meaning"`
}

// PromptHelp is the canonical explanation of one value question the install
// asks through Prompter.Prompt: what is being decided, and what each answer
// means. It lives in core so every front door renders the same words and none
// has to invent them (iss-163) — a front door that describes an answer on its
// own can describe it wrongly, or circularly ("native — uses the native
// backend").
//
// The prompt itself stays key, choices and default, so a scripted answer
// stream lines up with the questions as before; the help is looked up by the
// same key and rendered beside it.
type PromptHelp struct {
	Key     string       `json:"key"`
	About   string       `json:"about"`
	Choices []ChoiceHelp `json:"choices"`
	// Flag is the install flag that answers this question without asking it,
	// or "" for a question no flag answers. A piped run cannot rely on its
	// answers lining up with the questions, so the flag is the reliable route
	// there, and the question and the gap both name it (iss-2609120447486547).
	Flag string `json:"flag,omitempty"`
	// ChangeLater says where the answer is changed later, for a question no
	// flag answers (a flag's hint takes that place where one exists). It is
	// the question's change-later line, so the material need not repeat it
	// (iss-2610031236155833).
	ChangeLater string `json:"change_later,omitempty"`
}

// YesStillAsksValues is said once, before the first value question, by a run
// that approved every kind of change with --yes and still has a value to ask:
// the approval chooses no value, so the questions below are asked all the same
// (iss-2609120447486547). The words are core's so every front door says the
// same thing.
const YesStillAsksValues = "--yes approves each kind of change but chooses no value, so the questions below are still asked; " +
	"each names the flag that answers it without asking."

// FlagHint is the sentence naming the flag that answers this question without
// asking it, or "" when no flag does.
func (h PromptHelp) FlagHint() string {
	if h.Flag == "" {
		return ""
	}
	values := make([]string, len(h.Choices))
	for i, c := range h.Choices {
		values[i] = c.Value
	}
	return "pass " + h.Flag + " " + strings.Join(values, "|") + " to answer without asking"
}

// ChangeLaterLine is the question's change-later line: the flag hint where a
// flag answers the question, else ChangeLater, else "".
func (h PromptHelp) ChangeLaterLine() string {
	if hint := h.FlagHint(); hint != "" {
		return hint
	}
	return h.ChangeLater
}

// Meaning returns what answering value means, or "" for a value the question
// does not offer.
func (h PromptHelp) Meaning(value string) string {
	for _, c := range h.Choices {
		if c.Value == value {
			return c.Meaning
		}
	}
	return ""
}

// noAdapterYet closes every oracle answer other than host-delegated: nothing in
// abcd reads the backend setting yet, so the answer is recorded and changes no
// review today. Saying so is the honest half of the explanation; an answer that
// promised a direct model call would describe a behaviour abcd does not have.
const noAdapterYet = " abcd does not ship this adapter yet, so the choice is recorded and reviews still go to the assistant you are working in."

// oracleAdapterShipped names the oracle answers abcd has an adapter for. Every
// other answer carries noAdapterYet in its meaning, and a test holds the two in
// step, so marking an adapter here and rewording its meaning is one change.
var oracleAdapterShipped = map[string]bool{"host-delegated": true}

// oracleBackendAsked reports whether the install asks which oracle to use. A
// question with one defensible answer is not asked: while only one answer has
// an adapter, the install records it and says so (oracleBackendRecordedNote).
// The question returns on its own once a second answer is marked as shipped
// (the 2026-10-03 ruling on iss-2610031236155833).
func oracleBackendAsked() bool {
	n := 0
	for _, v := range oracleBackendChoices {
		if oracleAdapterShipped[v] {
			n++
		}
	}
	return n >= 2
}

// oracleBackendRecordedNote is the one line an install that recorded the
// oracle without asking says in its report: what it recorded, why nothing was
// asked, and how to choose another reviewer once one arrives.
const oracleBackendRecordedNote = "the AI reviewer was not asked: host-delegated, the assistant you are working in, " +
	"is recorded because it is the only reviewer abcd ships; other reviewers arrive later, and " +
	"abcd ahoy install --oracle-backend <value> chooses one then."

// promptHelp is the canonical help, one entry per value question. The choice
// order matches the order the question offers them.
var promptHelp = map[string]PromptHelp{
	"visibility": {
		Key:  "visibility",
		Flag: "--visibility",
		About: "Whether abcd's records (decisions, intents and issues, under .abcd/) are committed with your code " +
			"or kept out of git, by abcd's block in .gitignore.",
		Choices: []ChoiceHelp{
			{Value: "private", Meaning: "records committed, shared by all who see the repository; only per-machine scratch, " +
				".abcd/.work.local/, is ignored. Suits unpublished code."},
			{Value: "public", Meaning: "all of .abcd/ is ignored, so records stay on this machine, unpublished; " +
				"so is memory/ at the top, abcd's older memory home."},
		},
	},
	"docs_target": {
		Key:  "docs_target",
		Flag: "--docs-target",
		About: "Which conventions file, if any, gets a short block explaining how abcd works in this repository. " +
			"AI coding assistants read these files at the start of every session; the block names abcd, so it goes only where you choose.",
		Choices: []ChoiceHelp{
			{Value: "claude_md", Meaning: "writes the block into CLAUDE.md, and creates that file if it does not exist."},
			{Value: "agents_md", Meaning: "writes the block into AGENTS.md, the conventions file many AI coding assistants read, and creates it if it does not exist."},
			{Value: "both", Meaning: "writes the same block into both CLAUDE.md and AGENTS.md."},
			{Value: "skip", Meaning: "writes no block: abcd names itself in none of your conventions files. " +
				"You can choose a file later with abcd ahoy install --docs-target."},
		},
	},
	"oracle_backend": {
		Key:  "oracle_backend",
		Flag: "--oracle-backend",
		About: "Which AI reviewer abcd uses. abcd calls it an oracle: the AI model asked to review or check your work, " +
			"for example to read a change and say whether it is ready. The choice decides who runs that model, and so what it costs and which keys or tools it needs.",
		Choices: []ChoiceHelp{
			{Value: "host-delegated", Meaning: "the AI assistant you are already working in runs every review. " +
				"No API key, no extra tool and no cost beyond the assistant you already use."},
			{Value: "native", Meaning: "abcd would call a model itself, through an adapter built into abcd; that needs the provider's API key, and the use is billed by that provider." + noAdapterYet},
			{Value: "cli", Meaning: "abcd would run a model's command-line tool installed on this machine; that tool must be installed and signed in, and its use may be billed." + noAdapterYet},
			{Value: "api", Meaning: "abcd would call a model provider's web API directly; that needs an API key, and each call is billed by the provider." + noAdapterYet},
			{Value: "mcp", Meaning: "abcd would reach a model through an MCP server (a standard way to connect tools to AI models) that you run or connect to; " +
				"it needs that server set up, and whatever keys and cost the server brings." + noAdapterYet},
		},
	},
	"scan_deep": {
		Key:  "scan_deep",
		Flag: "--scan-deep",
		About: "Whether this private repository also wants a deep secret scan with trufflehog, a scanner found on this machine " +
			"that checks whether a leaked password or key still works. abcd's built-in secret scan runs either way.",
		Choices: []ChoiceHelp{
			{Value: "true", Meaning: "records that you want the trufflehog scan (scan.deep in .abcd/config.json); " +
				"no abcd check runs it yet, so nothing else changes."},
			{Value: "false", Meaning: "keeps to abcd's built-in secret scan and records that choice, so the question is not asked again."},
		},
	},
	artefactKindKey: {
		Key:         artefactKindKey,
		About:       "What this repository releases; abcd's release commands refuse to guess it.",
		ChangeLater: "edit .abcd/config/artefact.json, read by the release commands",
		Choices: []ChoiceHelp{
			{Value: "plugin", Meaning: "an agent plugin: the release preview scans the plugin payload listed in " +
				".abcd/config/launch-payload.json and checks that it would install."},
			{Value: "binary", Meaning: "a built program: the release preview scans what the release tag would archive; " +
				"the release set-up lays a gate whose empty build job you fill in."},
			{Value: "application", Meaning: "as binary today; the default, since it assumes least about how you build."},
		},
	},
	emDashPromptKey: {
		Key: emDashPromptKey,
		About: "How strict abcd's documentation check is about one house-style rule: an em dash (—) inside a list item. " +
			"It is a matter of style, not correctness, so this repository chooses. The answer is written into " +
			".abcd/docs-lint.json, where it can be changed later.",
		Choices: []ChoiceHelp{
			{Value: "blocking", Meaning: "an em dash in a list item fails the documentation check until it is fixed."},
			{Value: "warning", Meaning: "an em dash in a list item is reported, but the documentation check still passes."},
		},
	},
}

// statusLineElementAbout says what each switchable element of the status line
// shows. The badge (element one) is not switchable and is never asked about.
var statusLineElementAbout = map[statusline.ElementKey]string{
	statusline.KeyRepo:     "the name of the repository you are working in",
	statusline.KeyBranch:   "the git branch you are on",
	statusline.KeyModel:    "which AI model the assistant is using",
	statusline.KeyContext:  "how full the assistant's working memory for this conversation (its context window) is, as a percentage",
	statusline.KeyFiveHour: "how much of your assistant plan's five-hour usage allowance is used, as a percentage",
	statusline.KeySevenDay: "how much of your assistant plan's seven-day usage allowance is used, as a percentage",
	statusline.KeyIntents:  "how many intents (the planned pieces of work abcd tracks) the repository holds",
	statusline.KeyIssues:   "how many issues abcd's issue ledger holds for the repository",
}

// visibilityTrackedCaveat ends public's meaning where .abcd/ already holds
// tracked files: git cannot hide a file it tracks, so there the public block
// keeps out only the per-machine scratch space (effectiveVisibilityEntries
// narrows it). Where nothing under .abcd/ is tracked it would describe a
// repository the person does not have, so HelpIn adds it only where it applies
// (iss-2610031236155833).
const visibilityTrackedCaveat = "Here .abcd/ holds records git tracks and cannot hide, so only its scratch is ignored."

// visibilityHelp is the visibility question's help, with public's caveat about
// records git already tracks when tracked is true.
func visibilityHelp(tracked bool) PromptHelp {
	h := promptHelp["visibility"]
	if !tracked {
		return h
	}
	h.Choices = append([]ChoiceHelp(nil), h.Choices...)
	for i := range h.Choices {
		if h.Choices[i].Value == "public" {
			h.Choices[i].Meaning += " " + visibilityTrackedCaveat
		}
	}
	return h
}

// HelpIn returns the help for the value question keyed key as the install asks
// it in the repository at cwd. It is helpFor, except where a fact holds only in
// some repositories: public visibility's caveat is shown only where .abcd/
// holds tracked files, the same evidence the install narrows the public block
// on. The words stay core's; a front door passes the repository and renders
// what comes back (iss-2610031236155833). An empty cwd gives the help that is
// the same in every repository (helpFor).
func HelpIn(cwd, key string) (PromptHelp, bool) {
	if key != "visibility" || cwd == "" {
		return helpFor(key)
	}
	_, narrowed := effectiveVisibilityEntries(cwd, "public")
	return visibilityHelp(narrowed), true
}

// helpFor returns the canonical help for the value question keyed key, and
// false for a key the install does not ask about, so a front door renders
// nothing rather than a guess. It is the same in every repository: the
// visibility help is the one a repository whose .abcd/ holds no tracked files
// sees, and HelpIn, the one lookup front doors call, is the help for one
// repository.
func helpFor(key string) (PromptHelp, bool) {
	if h, ok := promptHelp[key]; ok {
		return h, true
	}
	if el, ok := strings.CutPrefix(key, elementPromptPrefix); ok {
		shows, known := statusLineElementAbout[statusline.ElementKey(el)]
		if !known {
			return PromptHelp{}, false
		}
		return PromptHelp{
			Key:   key,
			About: "Whether abcd's status line shows " + shows + ".",
			Choices: []ChoiceHelp{
				{Value: "on", Meaning: "shows it on the status line, in abcd-managed repositories."},
				{Value: "off", Meaning: "leaves it off the status line; switch it back on any time in " + statusline.SettingsDisplay + "."},
			},
		}, true
	}
	return PromptHelp{}, false
}
