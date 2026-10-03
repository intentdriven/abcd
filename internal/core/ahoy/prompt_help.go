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
	return "to answer without being asked, pass " + h.Flag + " " + strings.Join(values, "|")
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
		About: "Whether abcd's records for this repository (its decisions, intents and issues, kept under .abcd/) " +
			"are committed with your code or kept out of git. It decides what the block abcd writes into .gitignore contains.",
		Choices: []ChoiceHelp{
			{Value: "private", Meaning: "the records under .abcd/ are committed with your code, so everyone who can see the repository shares them; " +
				"only abcd's per-machine scratch space, .abcd/.work.local/, is kept out of git. Suits a repository whose code is not published."},
			{Value: "public", Meaning: "the whole .abcd/ folder is kept out of git, so the records stay on this machine and are not published with your code, " +
				"and so is a memory/ folder at the top of the repository, the older home of abcd's memory store. " +
				"If .abcd/ already holds committed records, only its per-machine scratch space is kept out, because git cannot hide a file it already tracks; the memory/ folder is still kept out."},
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
		Key: artefactKindKey,
		About: "What this repository releases. abcd's release commands read the answer from .abcd/config/artefact.json " +
			"to choose what the release preview scans and which release workflow they lay, and refuse to guess it. " +
			"The answer can be changed in that file later.",
		Choices: []ChoiceHelp{
			{Value: "plugin", Meaning: "the repository is released as an agent plugin: the release preview scans the plugin payload " +
				"listed in .abcd/config/launch-payload.json and checks that it would install."},
			{Value: "binary", Meaning: "the repository is released as a built program: the release preview scans the files the release tag " +
				"would archive, and the release set-up lays a release gate with an empty build job for you to fill in. " +
				"abcd handles binary and application the same way today."},
			{Value: "application", Meaning: "the repository is released as an application, which abcd handles exactly as a binary today: " +
				"the release preview scans the files the release tag would archive, and the release set-up lays a release gate " +
				"with an empty build job for you to fill in. It is the default because it assumes least about how you build."},
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

// HelpFor returns the canonical help for the value question keyed key, and
// false for a key the install does not ask about, so a front door renders
// nothing rather than a guess.
func HelpFor(key string) (PromptHelp, bool) {
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
