package surface

import "sort"

// next.go — what usually comes next after each visible command succeeds
// (itd-2610301428378830, spc-2610301431108175 approach 7, step 6). One entry
// per visible command, keyed by its full path, as the sentence and example
// manifests are. Each step is a command template and one line saying when it
// applies; a word in angle brackets is a value the caller fills in, as in a
// refusal's next command. A command after which nothing usually follows has
// the explicit empty list and the reason, so even it names what comes next.
//
// This table is the one source: the command facts carry it (Facts.Next), so
// `abcd explain`, the per-command JSON files and the command-facts block of
// every page show the steps a successful --json result carries under its
// top-level next_steps key. Tests in the front door hold every visible
// command to an entry, every entry to a visible command, and every template
// to a command path the tree has, with flags that command takes. A step that
// passes a flag giving the person's yes (--yes, --confirm, --take and the
// rest), or that the guard refuses an agent as the person's own act
// (internal/core/guard/personacts.go), says in its when that it waits on the
// person (only when the person says yes), even where the command cannot run
// without the flag.

// NextStep is one step that usually follows a command's success.
//
// The field order is the JSON key order.
type NextStep struct {
	// Command is the command template, "abcd capture resolve <iss-N> ...".
	Command string `json:"command"`
	// When is one line saying when the step applies.
	When string `json:"when"`
}

// Next is one command's entry: its steps, or the reason none usually
// follows, and whether the command streams documents.
type Next struct {
	// Steps is what usually comes next, in the order a caller would weigh
	// them. Empty only with None.
	Steps []NextStep
	// None is the reason nothing usually follows, for the explicit empty
	// list.
	None string
	// Streaming marks a command whose --json run can write a document before
	// its last one, such as a loop's refusal document ahead of the error
	// object. Only its closing summary carries next_steps.
	Streaming bool
}

// step is a NextStep, spelled short for the table.
func step(command, when string) NextStep { return NextStep{Command: command, When: when} }

// steps is a Next holding the given steps.
func steps(s ...NextStep) Next { return Next{Steps: s} }

// streaming is a Next holding the given steps, for a streaming command.
func streaming(s ...NextStep) Next { return Next{Steps: s, Streaming: true} }

// none is the explicit empty list, with the reason.
func none(reason string) Next { return Next{None: reason} }

// runsOn is the explicit empty list for a command that goes on running after
// it writes its document, such as a server's start line: the document is
// written at once, as a streaming command's are, and carries no next steps,
// since the run has not ended.
func runsOn(reason string) Next { return Next{None: reason, Streaming: true} }

var nextSteps = map[string]Next{
	"abcd": steps(
		step("abcd <record-id>", "to see what one record on the board is and its next move"),
		step("abcd capture \"<text>\" --remedy \"<fix>\"", "when the board shows a defect not yet on the ledger"),
		step("abcd build next", "to start the readiest planned intent"),
	),

	"abcd agents": steps(
		step("abcd implement status", "to see the lanes those agents are working"),
	),

	"abcd ahoy": steps(
		step("abcd ahoy install", "when it lists an install gap"),
		step("abcd ahoy doctor", "to see every gap, user-scope state included"),
	),
	"abcd ahoy connect": steps(
		step("abcd ahoy credential", "to list the credentials abcd now reads"),
		step("abcd ahoy doctor", "to check nothing else is missing"),
	),
	"abcd ahoy credential": steps(
		step("abcd ahoy connect <provider>", "to configure the provider the credential is for"),
	),
	"abcd ahoy doctor": steps(
		step("abcd ahoy install", "when it reports a gap install can apply"),
	),
	"abcd ahoy install": steps(
		step("abcd ahoy doctor", "to confirm no gap is left"),
		step("abcd", "to read the board of the repository just set up"),
	),
	"abcd ahoy remote": steps(
		step("abcd ahoy remote apply", "when a GitHub setting differs from the baseline"),
	),
	"abcd ahoy remote apply": steps(
		step("abcd ahoy remote", "to confirm no setting differs any more"),
	),
	"abcd ahoy shared-folder": steps(
		step("abcd implement stop", "only when the person says yes, and run by them from their own account, to stop the unattended runs of every account that chose the folder"),
	),
	"abcd ahoy uninstall": none("abcd is gone from the repository, so no abcd command usually follows"),
	"abcd ahoy worktree": steps(
		step("abcd ahoy worktree add <name>", "to start a second session's work in a worktree of its own"),
		step("abcd ahoy worktree prune --dry-run", "to see which merged, clean worktrees can be reclaimed"),
	),
	"abcd ahoy worktree add": steps(
		step("abcd reserve <record-id> --footprint <path>", "before working a record in the new worktree"),
	),
	"abcd ahoy worktree list": steps(
		step("abcd ahoy worktree prune --dry-run", "to see which merged, clean worktrees can be reclaimed"),
	),
	"abcd ahoy worktree prune": steps(
		step("abcd ahoy worktree prune", "after a --dry-run, to reclaim what it listed"),
		step("abcd ahoy worktree list", "to see the worktrees left"),
	),
	"abcd ahoy worktree trim-cache": steps(
		step("abcd ahoy worktree trim-cache", "after a --dry-run, to delete what it listed"),
	),

	"abcd announce": steps(
		step("abcd status", "to see the announcement beside everyone else's"),
	),

	"abcd banlist": steps(
		step("abcd banlist add --private <key> <pattern>", "to ban a name that is missing"),
	),
	"abcd banlist add": steps(
		step("abcd banlist list --private", "to see the entry among the others, by key"),
	),
	"abcd banlist list": steps(
		step("abcd banlist remove --private <key>", "to drop an entry that no longer applies"),
	),
	"abcd banlist migrate": steps(
		step("abcd banlist list --private", "to see the keyed store"),
	),
	"abcd banlist remove": steps(
		step("abcd banlist list --private", "to see the entries left"),
	),

	"abcd build": streaming(
		step("abcd implement step", "to perform the run's next stage"),
		step("abcd implement status", "to see the run lane by lane"),
	),
	"abcd build next": streaming(
		step("abcd implement step", "to perform the started run's next stage"),
		step("abcd implement status", "to see the run lane by lane"),
	),

	"abcd capture": steps(
		step("abcd reserve <iss-N> --footprint <path>", "before starting work on the issue"),
		step("abcd capture resolve <iss-N> \"<what fixed it>\" --impact <impact>", "once the fix lands"),
	),
	"abcd capture admit": steps(
		step("abcd capture disposition <rdi-N> --state <state> --grounds \"<why>\"", "to answer the run's other reading items"),
	),
	"abcd capture defer": steps(
		step("abcd capture list --open", "to see the open issues the deferral leaves"),
	),
	"abcd capture disposition": steps(
		step("abcd capture promote <rdi-N>", "when the item was accepted and should become an intent draft"),
	),
	"abcd capture link": steps(
		step("abcd capture list --open", "to see the issue among the others, blocked or not"),
	),
	"abcd capture list": steps(
		step("abcd <iss-N>", "to see one issue whole and its next move"),
		step("abcd reserve <iss-N> --footprint <path>", "before starting work on one"),
	),
	"abcd capture mentions": steps(
		step("abcd capture resolve <iss-N> \"<what fixed it>\" --impact <impact>", "for an issue a landed change already fixed"),
	),
	"abcd capture migrate": steps(
		step("abcd capture migrate --apply", "after a practice run, to rewrite what it listed"),
	),
	"abcd capture owed": steps(
		step("abcd capture \"<text>\" --remedy \"<fix>\"", "for an owed finding not yet on the ledger"),
		step("abcd capture owed <handle> --captured <iss-N>", "once its issue is filed, to settle the entry"),
	),
	"abcd capture promote": steps(
		step("abcd intent plan <itd-N>", "to plan the draft the promotion filed"),
	),
	"abcd capture reframe": steps(
		step("abcd capture reframe --complete <rfm-N>", "once the reframe has been worked through"),
	),
	"abcd capture remedy": steps(
		step("abcd reserve <iss-N> --footprint <path>", "before starting the fix"),
	),
	"abcd capture resolve": steps(
		step("abcd reserve <iss-N> --release", "to release the issue's reservation"),
		step("abcd land", "to push the branch carrying the fix and queue its pull request"),
	),
	"abcd capture surprise": steps(
		step("abcd capture list --open", "to see the ledger the surprise sits beside"),
	),
	"abcd capture wontfix": steps(
		step("abcd reserve <iss-N> --release", "to release the issue's reservation, if one was held"),
	),

	"abcd checklist": steps(
		step("abcd checklist set <id> --state <state>", "to move an item on"),
		step("abcd checklist add \"<title>\"", "to add an item that is missing"),
	),
	"abcd checklist add": steps(
		step("abcd checklist set <id> --state <state>", "once the item's state changes"),
	),
	"abcd checklist render": steps(
		step("abcd checklist set <id> --state <state>", "to move an item on"),
	),
	"abcd checklist set": steps(
		step("abcd checklist", "to see the checklist with the item moved"),
	),
	"abcd checklist show": steps(
		step("abcd checklist set <id> --state <state>", "to change the item's state"),
	),

	"abcd dashboard": steps(
		step("abcd dashboard start", "when the dashboard is not running"),
		step("abcd dashboard stop", "when it runs and is no longer wanted"),
	),
	"abcd dashboard start": steps(
		step("abcd dashboard status", "to see its address and the devices that opened it"),
	),
	"abcd dashboard status": steps(
		step("abcd dashboard start", "when the dashboard is not running"),
		step("abcd dashboard stop", "when it runs and is no longer wanted"),
	),
	"abcd dashboard stop": none("the dashboard is stopped, and nothing usually follows"),

	"abcd decide": steps(
		step("abcd decide \"<text>\"", "to record the next decision"),
		step("abcd decide --adr \"<title>\"", "when a decision shapes the architecture and should be an ADR"),
	),

	"abcd disembark": steps(
		step("abcd disembark probe <repo>", "to see which brief sections a lifeboat could ground"),
	),
	"abcd disembark coverage": steps(
		step("abcd disembark pack <repo> <dest>", "to pack a repository the table shows is ready"),
	),
	"abcd disembark graveyard": steps(
		step("abcd disembark principles <lifeboat-dir>", "to distil the lifeboat's principles"),
	),
	"abcd disembark pack": steps(
		step("abcd disembark review <lifeboat-dir> <source-repo>", "to review the packed lifeboat against its source"),
	),
	"abcd disembark plan": steps(
		step("abcd disembark pack <repo> <dest>", "to write the file set the plan showed"),
	),
	"abcd disembark press-release": steps(
		step("abcd embark probe <lifeboat-dir> <target-dir>", "to see what the lifeboat would write into a target"),
	),
	"abcd disembark principles": steps(
		step("abcd disembark press-release <lifeboat-dir>", "to compose the lifeboat's press release"),
	),
	"abcd disembark probe": steps(
		step("abcd disembark plan <repo>", "to see the file set a pack would write"),
	),
	"abcd disembark review": steps(
		step("abcd disembark graveyard <lifeboat-dir> --lessons-json <file>", "to validate the lifeboat's lessons"),
	),

	"abcd docs": steps(
		step("abcd docs cite refresh", "to fetch every cited URL and refresh the baseline"),
		step("abcd docs fidelity", "to judge the brief against the shipped surfaces"),
	),
	"abcd docs cite": steps(
		step("abcd docs cite refresh", "to fetch every cited URL and refresh the baseline"),
	),
	"abcd docs cite confirm": steps(
		step("abcd lint docs", "to check the docs against the confirmed baseline"),
	),
	"abcd docs cite refresh": steps(
		step("abcd docs cite confirm <url>", "for a URL the fetcher could not read that a person verified"),
		step("abcd lint docs", "to check the docs against the refreshed baseline"),
	),
	"abcd docs fidelity": steps(
		step("abcd docs fidelity record --verdict-json <file>", "once a docs review has been run for HEAD"),
	),
	"abcd docs fidelity record": steps(
		step("abcd spec close <spc-N>", "to close the spec the review was owed for"),
		step("abcd launch ship --dry-run", "to preview the release cut the review clears"),
	),

	"abcd drain": steps(
		step("abcd implement status", "to see the drain's lanes"),
		step("abcd capture list --open", "to see the issues handed back"),
	),

	"abcd embargo": steps(
		step("abcd embargo intake", "to take new private advisories into the embargo store"),
		step("abcd embargo publish", "to see the publication plan for those ready"),
	),
	"abcd embargo check": steps(
		step("abcd land", "when nothing in the range discloses an advisory under embargo"),
	),
	"abcd embargo close": steps(
		step("abcd embargo", "to see the advisories still under embargo"),
	),
	"abcd embargo fixed": steps(
		step("abcd embargo publish", "to see the publication plan the fix makes ready"),
	),
	"abcd embargo intake": steps(
		step("abcd embargo triage <GHSA-id> <verdict>", "to record a verdict on an advisory taken in"),
	),
	"abcd embargo publish": steps(
		step("abcd embargo close <GHSA-id>", "to close an advisory once it is published"),
	),
	"abcd embargo queue": steps(
		step("abcd mode back", "only when the person says yes, once they are back, to print what was queued"),
	),
	"abcd embargo triage": steps(
		step("abcd embargo fixed <GHSA-id> <commit>", "once the commits that fix the advisory exist"),
	),

	"abcd embark": steps(
		step("abcd embark probe <lifeboat-dir> <target-dir>", "to see what a lifeboat would write into a target"),
	),
	"abcd embark from": steps(
		step("abcd embark lessons <lifeboat-dir> <target-dir>", "to rank the lifeboat's lessons against the new brief"),
		step("abcd", "to read the board of the target repository"),
	),
	"abcd embark lessons": none("the ranking is read and acted on in the new voyage's brief; no command usually follows"),
	"abcd embark probe": steps(
		step("abcd embark from <lifeboat-dir> <target-dir>", "to unpack the lifeboat once the probe shows no conflict"),
	),

	"abcd explain": none("the answer is what the next call is built from; no abcd command usually follows it"),

	"abcd guard": steps(
		step("abcd guard check --command '<line>'", "to judge one shell command before running it"),
	),
	"abcd guard allow":   none("the allowance is used by the next matching call through the hook; nothing usually follows"),
	"abcd guard approve": none("it answers prompts until its input ends; nothing usually follows"),
	"abcd guard check": steps(
		step("abcd guard allow --once --command '<line>'", "only when the person says yes, and run by them, to let through one unproven delete they have judged safe"),
	),
	"abcd guard hook":     none("the host calls it on each tool call and reads its answer; nothing follows it"),
	"abcd guard question": none("a clean draft is handed to the host's question tool; no abcd command follows it"),

	"abcd history": steps(
		step("abcd history drain", "to redact and store every transcript staged for this repository"),
		step("abcd history list", "to list the stored transcripts"),
	),
	"abcd history capture": steps(
		step("abcd history list", "to see the stored transcript among the others"),
	),
	"abcd history discard": steps(
		step("abcd history staged", "to see the transcripts still staged"),
	),
	"abcd history drain": streaming(
		step("abcd history list", "to see the transcripts it stored"),
	),
	"abcd history ingest": steps(
		step("abcd history list", "to see the transcripts it stored"),
	),
	"abcd history list": steps(
		step("abcd history show <session-id>", "to read one stored transcript"),
	),
	"abcd history migrate": steps(
		step("abcd history list", "to see the repaired records"),
	),
	"abcd history reconstruct": none("the artefact and its telemetry are written for reading; nothing usually follows"),
	"abcd history separation":  none("the report answers the question asked; nothing usually follows"),
	"abcd history show": steps(
		step("abcd history reconstruct <session-id>", "to render the session with its sub-agents as one artefact"),
	),
	"abcd history staged": steps(
		step("abcd history drain", "to redact and store what is staged"),
		step("abcd history discard <staged-filename> --yes", "only when the person says yes, to delete a staged transcript for good"),
	),

	"abcd ideate": steps(
		step("abcd ideate record <idea-slug> --verdict-json <file>", "once the host has run the gauntlet and composed its verdict"),
	),
	"abcd ideate record": steps(
		step("abcd intent \"<text>\"", "when the verdict admits the idea, to file it as a draft intent"),
	),

	"abcd identity": steps(
		step("abcd identity init", "when the repository records no identity block yet"),
		step("abcd identity render", "to see the corrections for every drifted surface"),
	),
	"abcd identity init": steps(
		step("abcd lint identity", "to see every surface held to the block"),
	),
	"abcd identity render": steps(
		step("abcd lint identity", "once the corrections are applied, to confirm nothing drifts"),
	),

	"abcd implement": steps(
		step("abcd implement status", "to see the loop's runs lane by lane"),
		step("abcd implement join --session <id> --role <role>", "to join the shared run"),
	),
	"abcd implement check": steps(
		step("abcd implement claim <record> --session <id> --lane <lane>", "when the stage is allowed, to claim its record"),
	),
	"abcd implement claim": steps(
		step("abcd implement release <record> --session <id>", "once the lane's work is done"),
	),
	"abcd implement heartbeat": steps(
		step("abcd implement status", "to see the run lane by lane"),
	),
	"abcd implement join": steps(
		step("abcd implement mode <mode> --session <id>", "to open a window by logging its division mode"),
		step("abcd implement check lane --session <id>", "before taking a lane"),
	),
	"abcd implement leave": none("the session has left the run and holds no claim; nothing usually follows"),
	"abcd implement load": steps(
		step("abcd implement load --wait --narrow --timeout <duration> -- <command>", "to run a test command once there is capacity"),
	),
	"abcd implement log": none("the event is on today's run log; nothing usually follows"),
	"abcd implement mode": steps(
		step("abcd implement check lane --session <id>", "before taking a lane in the window"),
	),
	"abcd implement outage": steps(
		step("abcd implement outage probe --session <id>", "when the next probe is due"),
	),
	"abcd implement outage ack": steps(
		step("abcd implement outage clear --session <id> --reason \"<why>\"", "once the connection is back"),
	),
	"abcd implement outage clear": steps(
		step("abcd implement status", "to see the run carry on"),
	),
	"abcd implement outage probe": steps(
		step("abcd implement outage", "to see whether the outage is still open"),
	),
	"abcd implement outage record": steps(
		step("abcd implement outage", "to see the shared outage and when its next probe is due"),
	),
	"abcd implement receipt": streaming(
		step("abcd implement step", "to perform the run's next stage"),
	),
	"abcd implement record": streaming(
		step("abcd implement report", "to compare the division modes from the run log"),
	),
	"abcd implement release": steps(
		step("abcd implement leave --session <id>", "when the session holds nothing else in the run"),
	),
	"abcd implement report": none("the comparison is the run's final reading; nothing usually follows"),
	"abcd implement resume": streaming(
		step("abcd implement resume --take <run-id>", "only when the person says yes, to take over a run it listed"),
		step("abcd implement step", "to perform a taken run's next stage"),
	),
	"abcd implement status": streaming(
		step("abcd implement step", "to perform a run's next stage"),
	),
	"abcd implement step": streaming(
		step("abcd implement step", "to perform the stage after it"),
		step("abcd implement status", "to see the run lane by lane"),
	),
	"abcd implement stop": steps(
		step("abcd implement stop --lift", "only when the person says yes, and run by them from their own account, once the runs may start again"),
	),
	"abcd implement widen": streaming(
		step("abcd implement step", "to carry on the lane with the wider hold"),
	),

	"abcd inbox": steps(
		step("abcd inbox show <id>", "to read one report whole"),
	),
	"abcd inbox promote": steps(
		step("abcd capture list --open", "to see the capture it filed"),
	),
	"abcd inbox show": steps(
		step("abcd inbox promote <id>", "to file the report as a capture"),
	),

	"abcd intent": steps(
		step("abcd intent plan <itd-N>", "to plan the draft"),
		step("abcd intent hold <itd-N> --reason \"<text>\"", "when the draft must wait on a ruling"),
	),
	"abcd intent audit": steps(
		step("abcd intent audit ingest --verdict-json <path>", "once the host has run the audit and composed its verdict"),
	),
	"abcd intent audit ingest": steps(
		step("abcd intent condition <itd-N>", "to disposition the shipped intent's scope conditions"),
	),
	"abcd intent condition": steps(
		step("abcd intent condition <itd-N>", "to disposition the next condition"),
	),
	"abcd intent consistency": steps(
		step("abcd intent consistency ingest --findings-json <path>", "once the host has returned its findings"),
	),
	"abcd intent consistency ingest": steps(
		step("abcd capture list --open", "to see the captures the findings filed"),
	),
	"abcd intent edge": steps(
		step("abcd intent ready <itd-N>", "to see whether the intent is ready with its new edges"),
	),
	"abcd intent hold": steps(
		step("abcd intent unhold <itd-N>", "once the ruling the hold waits on is made"),
	),
	"abcd intent interview": steps(
		step("abcd intent ready <itd-N>", "to see whether the intent is ready once the interview is answered"),
	),
	"abcd intent link": steps(
		step("abcd intent ready <itd-N>", "to see whether the intent is ready to implement"),
	),
	"abcd intent plan": steps(
		step("abcd intent interview <itd-N>", "to run the intent's planning interview"),
		step("abcd intent ready <itd-N>", "to see whether the planned intent is ready"),
	),
	"abcd intent prepass": steps(
		step("abcd intent plan <itd-N>", "once the planning brief is written"),
	),
	"abcd intent ready": steps(
		step("abcd build <itd-N>", "when it reports the intent ready"),
	),
	"abcd intent reclassify": steps(
		step("abcd intent", "to see the intent store with the change"),
	),
	"abcd intent target": steps(
		step("abcd intent ready <itd-N>", "to see whether the intent is ready for its release"),
	),
	"abcd intent unhold": steps(
		step("abcd intent plan <itd-N>", "to plan the intent the hold kept back"),
	),

	"abcd lab": steps(
		step("abcd lab mint \"<question>\"", "to open a lab for a new question"),
	),
	"abcd lab harvest": steps(
		step("abcd capture \"<text>\" --remedy \"<fix>\"", "to file a capture candidate the harvest names"),
	),
	"abcd lab matrix": steps(
		step("abcd lab matrix report <lab-id> <run-id>", "once the matrix run is recorded"),
	),
	"abcd lab matrix copy": steps(
		step("abcd lab matrix review <lab-id> <run-id> <task>", "once every copy of a code task is handed back"),
	),
	"abcd lab matrix judge": steps(
		step("abcd lab matrix report <lab-id> <run-id>", "once the verdicts are recorded"),
	),
	"abcd lab matrix report": steps(
		step("abcd lab matrix rerun <lab-id> <run-id>", "to plan a re-run of the recorded run"),
	),
	"abcd lab matrix rerun": steps(
		step("abcd lab matrix report <lab-id> <run-id>", "once the re-run is recorded"),
	),
	"abcd lab matrix review": steps(
		step("abcd lab matrix report <lab-id> <run-id>", "once the reviewers' answers are recorded"),
	),
	"abcd lab mint": steps(
		step("abcd lab preflight <lab-id>", "to run the lab's isolation checks before its first probe"),
	),
	"abcd lab preflight": steps(
		step("abcd lab record <lab-id> <probe>", "to scaffold the lab's first probe record"),
	),
	"abcd lab record": steps(
		step("abcd lab harvest <lab-id>", "once the probes are recorded"),
	),
	"abcd lab sweep": steps(
		step("abcd lab harvest <lab-id>", "once the sweep finds nothing retracted"),
	),

	"abcd land": steps(
		step("abcd landings", "to see how the change landed once the queue merges it"),
		step("abcd reserve <record-id> --release", "to release the record's reservation once it merges"),
	),

	"abcd landings": none("the listing answers how each change landed; nothing usually follows"),

	"abcd launch": steps(
		step("abcd launch ship --dry-run", "to preview the release cut alone"),
		step("abcd launch ship", "once the preview and the gates are clean, to cut the release"),
	),
	"abcd launch archive": none("the archive is written for the release job; nothing usually follows"),
	"abcd launch manifests": steps(
		step("abcd launch --dry-run", "to preview the rest of the release gates"),
	),
	"abcd launch receipts": steps(
		step("abcd launch --dry-run", "to preview the rest of the release gates"),
	),
	"abcd launch reland": steps(
		step("abcd land", "to push the relanded bump and queue its pull request"),
	),
	"abcd launch scaffold": steps(
		step("abcd launch scaffold --confirm", "only when the person says yes to what a practice run listed"),
		step("abcd launch --dry-run", "to preview the release gates it scaffolded"),
	),
	"abcd launch ship": steps(
		step("abcd launch ship", "after a --dry-run, to cut the release"),
		step("abcd reflect <release-tag>", "once the release is cut, to open its retrospective"),
	),

	"abcd library": steps(
		step("abcd library ask \"<question>\"", "to query the memory store"),
		step("abcd library add <document>", "to register a new source"),
	),
	"abcd library add": steps(
		step("abcd library ingest <path-or-https-url>", "to distil the source into cited memory pages"),
	),
	"abcd library ask": none("the cited answer is the result; nothing usually follows"),
	"abcd library cite-check": steps(
		step("abcd library declassify <key>", "for a hit on a confidential source that is now published"),
	),
	"abcd library declassify": steps(
		step("abcd library sync-banlist", "to project the remaining confidential titles into the banlist"),
	),
	"abcd library ingest": steps(
		step("abcd library lint", "to health-check the memory store with the new pages"),
	),
	"abcd library init": steps(
		step("abcd library add <document>", "to register the corpus's first source"),
	),
	"abcd library ledger": steps(
		step("abcd library ledger --list", "to see the ledger's lines"),
	),
	"abcd library lint": steps(
		step("abcd library ingest <path-or-https-url>", "to fill a gap the check names"),
	),
	"abcd library sync-banlist": steps(
		step("abcd banlist list --private", "to see the projected entries by key"),
	),

	"abcd lint": steps(
		step("abcd lint docs", "to read the docs findings alone"),
		step("abcd land", "once nothing is left to fix"),
	),
	"abcd lint docs": steps(
		step("abcd docs cite refresh", "when a citation finding names a stale baseline"),
	),
	"abcd lint identity": steps(
		step("abcd identity render", "to see the correction for a drifted surface"),
	),
	"abcd lint outbound": none("a clean text is posted where it was written for; no abcd command follows it"),
	"abcd lint site": steps(
		step("abcd site build", "to render the website again after a fix"),
	),

	"abcd memory": steps(
		step("abcd memory list", "to read every note with its recall words"),
	),
	"abcd memory add": steps(
		step("abcd memory list", "to see the note among the others"),
	),
	"abcd memory list": steps(
		step("abcd memory remove <mem-id>", "to remove a note that no longer holds"),
	),
	"abcd memory remove": steps(
		step("abcd memory list", "to see the notes left"),
	),

	"abcd mode": steps(
		step("abcd mode autonomous", "only when the person says yes, as they go away"),
		step("abcd mode back", "only when the person says yes, once they are back"),
	),
	"abcd mode autonomous": steps(
		step("abcd mode queue \"<step>\"", "to queue a step for the person's return"),
		step("abcd mode back", "only when the person says yes, once they are back"),
	),
	"abcd mode back": none("the queue is printed for the person to act on; nothing usually follows"),
	"abcd mode queue": steps(
		step("abcd mode back", "only when the person says yes, once they are back"),
	),

	"abcd peers": steps(
		step("abcd reserve <record-id> --footprint <path>", "to reserve a record no peer holds"),
	),

	"abcd reading": steps(
		step("abcd reading assemble --position <position> --target <HEAD|sha>", "to assemble a reading's input"),
	),
	"abcd reading assemble": steps(
		step("abcd reading ingest --reading-json <path>", "once the reading has been run and returned its JSON"),
	),
	"abcd reading ingest": steps(
		step("abcd capture disposition <rdi-N> --state <state> --grounds \"<why>\"", "to answer each reading item"),
	),

	"abcd reflect": steps(
		step("abcd reflect interview <release-tag>", "to run the retrospective interview"),
	),
	"abcd reflect interview": steps(
		step("abcd reflect write <release-tag> --answers <file>", "once the interview is answered"),
	),
	"abcd reflect write": none("the retrospective is written; nothing usually follows"),

	"abcd report": none("the report is filed in abcd's own inbox; nothing usually follows"),

	"abcd reserve": steps(
		step("abcd reserve <record-id> --release", "once the work on the record is done"),
		step("abcd peers", "to see the reservation beside the others"),
	),

	"abcd rules": none("the rules are read and followed; nothing usually follows"),

	"abcd scribe": steps(
		step("abcd scribe assemble --run <rdg-N> --dispositions <path>", "to build a scribe session's context"),
	),
	"abcd scribe assemble": steps(
		step("abcd scribe ingest --scribe-json <path> --dispositions <path>", "once the scribe has transcribed"),
	),
	"abcd scribe ingest": none("what the scribe transcribed is on the ledger; nothing usually follows"),

	"abcd session": steps(
		step("abcd session setup", "to set up live brief sessions"),
		step("abcd session start", "to start a live brief session on this computer"),
	),
	"abcd session blueprint": steps(
		step("abcd session page", "to write the brief session's static page beside the relay"),
	),
	"abcd session page":  none("the page is written for hosting; nothing usually follows"),
	"abcd session serve": runsOn("the relay runs in the foreground until stopped; nothing usually follows"),
	"abcd session setup": steps(
		step("abcd session start", "to start a live brief session"),
	),
	"abcd session start": runsOn("the session runs on this computer; its link is shared with the team"),

	"abcd site": steps(
		step("abcd site build", "to render the website"),
		step("abcd site setup", "when the website has no live address yet"),
	),
	"abcd site build": steps(
		step("abcd lint site", "to gate the built website"),
	),
	"abcd site setup": steps(
		step("abcd site", "to see what the website now declares"),
	),

	"abcd spec": steps(
		step("abcd spec close <spc-N>", "once the change that delivers a spec lands"),
	),
	"abcd spec close": steps(
		step("abcd land", "to push the branch that carries the close and queue its pull request"),
	),

	"abcd start": none("the session it starts is where the work goes on; nothing follows in this shell"),

	"abcd status": steps(
		step("abcd announce <kind> <item>", "to post an announcement of your own"),
		step("abcd peers", "to see what peers hold that this checkout lacks"),
	),

	"abcd statusline": none("the host draws the row and calls it again; nothing follows it"),

	"abcd sweep": steps(
		step("abcd sweep --apply", "to clear what it listed as spent"),
	),

	"abcd update": steps(
		step("abcd update news", "to see what changed since the update"),
	),
	"abcd update news": steps(
		step("abcd ahoy install", "when it offers setup that waits"),
	),
}

// NextFor returns the next steps the table declares for the command at path
// ("abcd capture resolve"), and whether it declares an entry.
func NextFor(path string) (Next, bool) {
	n, ok := nextSteps[path]
	return n, ok
}

// NextPathsForTest returns every command path the table declares an entry for,
// sorted.
func NextPathsForTest() []string {
	out := make([]string, 0, len(nextSteps))
	for p := range nextSteps {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
