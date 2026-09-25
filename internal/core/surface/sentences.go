package surface

import "sort"

// sentences.go — the sentence field of the surface manifest
// (itd-2609212113220149). One entry per visible command, keyed by the full
// command path a caller types, holding the one sentence every place the verb
// is listed renders: the parent's command list, the root's groups and agents
// block, the verb's own --help, and — for a top-level verb with a plugin page
// — the page's description. The front door (internal/surface/cli) sets each
// command's Short from here, the snapshot records it, and the generator writes
// the pages from it, so this table is the one place a sentence is worded.
//
// Every entry is the form ParseSentence checks: what the verb does, a colon,
// what it writes ("Writes nothing" when it writes nothing), a semicolon, and
// when it refuses, under docs/reference/writing-style.md, at most SentenceCap
// characters. "Refuses" names the case a caller most needs to know before the
// call; a verb whose only refusal is the usage error every verb has says so.
//
// A hidden command carries no entry: nothing lists it, so there is nothing to
// render. cobra's generated `help` and `completion` carry none either, because
// they are the framework's and exist only once the tree executes.
var sentences = map[string]string{
	"abcd": "Render the status board, or say what one record id is and its next move: " +
		"Writes nothing; refuses any other positional argument.",

	"abcd ahoy": "Detect abcd's install state and list its gaps, or report one mode a flag names: " +
		"Writes nothing; refuses any argument or two modes at once.",
	"abcd ahoy doctor": "Report every install gap, user-scope state included: " +
		"Writes nothing; refuses any argument.",
	"abcd ahoy install": "Apply the install gaps the detection finds: " +
		"Writes the .abcd/ scaffolding, the name-guard hooks, and the PATH entry; refuses a stale binary before any write.",
	"abcd ahoy remote": "Enable GitHub secret scanning and push protection: " +
		"Writes nothing bare, only the settings and their mirror; refuses bare, naming `abcd ahoy --remote`.",
	"abcd ahoy remote apply": "Enable GitHub secret scanning and push protection on this repository: " +
		"Writes both settings and their mirror; refuses an unconfirmed run.",
	"abcd ahoy uninstall": "Remove abcd from this repository, leaving .abcd/ in place: " +
		"Writes the removal of the marker block, PATH copy, and provenance record; refuses any argument.",

	"abcd banlist": "Render both banned-names layers: " +
		"Writes nothing; refuses an unknown word without echoing it.",
	"abcd banlist add": "Add one banned-name entry to the layer a flag names: " +
		"Writes that layer's store; refuses without exactly one of --private or --public.",
	"abcd banlist list": "Render the banned-names layers, private entries by key only: " +
		"Writes nothing; refuses --private and --public together.",
	"abcd banlist remove": "Remove one banned-name entry from the layer a flag names: " +
		"Writes that layer's store; refuses a public entry curated by hand.",

	"abcd capture": "File an issue from quoted text, or render the ledger's status bare: " +
		"Writes one record under open/; refuses a lone word and any folder outside a checkout.",
	"abcd capture disposition": "Answer one reading item with a disposition record: " +
		"Writes the record keyed to the item; refuses a second answer without --supersedes.",
	"abcd capture link": "Add or remove blocked_by edges on an issue: " +
		"Writes the issue's blocked_by list; refuses an id the ledger does not hold.",
	"abcd capture list": "List the issues in one status folder or all three: " +
		"Writes nothing; refuses when no status flag is given.",
	"abcd capture mentions": "List open issues that default-branch history names with no resolution behind them: " +
		"Writes nothing; refuses outside a git checkout.",
	"abcd capture migrate": "Rewrite retired promote back-links as related_intents and related_issues: " +
		"Writes the records only with --apply; refuses outside a git checkout.",
	"abcd capture promote": "Graduate an issue or an accepted reading item into an intent draft: " +
		"Writes the draft and both back-links; refuses a promoted issue or an unaccepted item.",
	"abcd capture resolve": "Move an open issue to resolved/, naming what fixed it: " +
		"Writes the moved record; refuses without --impact or on an id this ledger does not hold.",
	"abcd capture wontfix": "Move an open issue to wontfix/ with the reason it is not acted on: " +
		"Writes the moved record; refuses an id this ledger does not hold.",

	"abcd changelog": "Preview the next release cut's version, records, and guardrail verdict: " +
		"Writes nothing; refuses outside a checkout, exiting 0 on a cut the gates would stop.",

	"abcd decide": "Mint an ADR id and lay the record's empty skeleton: " +
		"Writes one proposed record into the decisions store; refuses a missing or unusable title.",

	"abcd disembark": "Pack a repository into a lifeboat, probing and planning first: " +
		"Writes nothing in the source, only inside the lifeboat; refuses an unknown sub-verb.",
	"abcd disembark coverage": "Aggregate saved probe reports into the section-by-repository coverage table: " +
		"Writes nothing; refuses a file that is not a probe report.",
	"abcd disembark graveyard": "Validate host-produced lesson JSON against a packed lifeboat: " +
		"Writes the lessons that cite their evidence; refuses without --lessons-json.",
	"abcd disembark pack": "Pack a lifeboat from a repository into a destination directory: " +
		"Writes the destination only; refuses when the secret scanner is unavailable.",
	"abcd disembark plan": "Show the file set a pack would write: " +
		"Writes nothing; refuses a repository path that is not a directory.",
	"abcd disembark press-release": "Compose a lifeboat's press release, or validate the host's: " +
		"Writes the press-release files in the lifeboat; refuses a host draft citing nothing resolvable.",
	"abcd disembark principles": "Distil a lifeboat's principles from its ADRs, or validate the host's: " +
		"Writes the principles files in the lifeboat; refuses a directory that is not a lifeboat.",
	"abcd disembark probe": "Report which brief sections a lifeboat could ground from a repository: " +
		"Writes nothing; refuses a repository path that is not a directory.",
	"abcd disembark review": "Review a packed lifeboat against its source repository, or validate the host's verdict: " +
		"Writes the review in the lifeboat; refuses an unregistered verdict.",

	"abcd docs": "Keep the citation baseline that `abcd lint docs` enforces offline: " +
		"Writes nothing but that baseline; refuses an unknown sub-verb.",
	"abcd docs cite": "Keep the citation baseline the docs lint enforces offline: " +
		"Writes nothing bare, and only that baseline; refuses an unknown sub-verb.",
	"abcd docs cite confirm": "Record that a person verified a cited URL the fetcher could not read: " +
		"Writes a dated manual entry in the baseline; refuses a URL the docs do not cite.",
	"abcd docs cite refresh": "Fetch every cited URL once, the one documentation verb that reaches the network: " +
		"Writes the citation baseline; refuses an unreadable docs-lint configuration.",

	"abcd embark": "Unpack a verified lifeboat into a target repository, probing first: " +
		"Writes only its record families and marker block; refuses the whole write on any conflict.",
	"abcd embark from": "Unpack a lifeboat's record families into a target repository: " +
		"Writes those families and the marker block; refuses the whole write on any conflict.",
	"abcd embark probe": "Report what a lifeboat would write into a target, coverage blanks first: " +
		"Writes nothing; refuses a lifeboat whose manifest does not verify.",

	"abcd guard": "Judge a shell command against the hazard registry before it runs: " +
		"Writes nothing; refuses a hazard through check or hook, and an unknown sub-verb.",
	"abcd guard check": "Judge one shell command against the hazard registry: " +
		"Writes nothing; refuses a hazard with exit 1 and a command it cannot parse with exit 2.",
	"abcd guard hook": "Judge the shell command in a host's pre-tool-use payload: " +
		"Writes nothing; refuses a hazard with the host's blocking status.",

	"abcd history": "Keep session transcripts in the user-level store and read them back: " +
		"Writes nothing bare, and redacts each one it stores; refuses an unknown sub-verb.",
	"abcd history capture": "Redact and store one raw session transcript from a file or stdin: " +
		"Writes one record into the store; refuses stdin without --session.",
	"abcd history discard": "Delete one staged or quarantined raw transcript for good: " +
		"Writes the deletion; refuses without --yes.",
	"abcd history drain": "Redact and store every transcript staged for this repository: " +
		"Writes the records into the store; refuses outside a git checkout.",
	"abcd history ingest": "Redact and store transcripts already on disk into a named repository: " +
		"Writes that repository's store; refuses without --into.",
	"abcd history list": "List this repository's stored transcripts, newest first: " +
		"Writes nothing; refuses outside a git checkout.",
	"abcd history migrate": "Repair records filed under a composite session id: " +
		"Writes the repaired records only with --apply; refuses outside a git checkout.",
	"abcd history reconstruct": "Render one session and its sub-agents as one artefact plus telemetry: " +
		"Writes both files into --out; refuses an --out that is not an existing directory.",
	"abcd history show": "Show one stored transcript's metadata and redacted body: " +
		"Writes nothing; refuses an id the store does not hold.",
	"abcd history staged": "List the transcripts that ended but are not yet redacted into the store: " +
		"Writes nothing; refuses outside a git checkout.",

	"abcd ideate": "Judge an idea through the host-run admission gauntlet: " +
		"Writes nothing bare, and one research record and its decision-log line; refuses an unknown sub-verb.",
	"abcd ideate record": "Validate a host-composed gauntlet verdict: " +
		"Writes the dated research record; refuses without an idea slug or --verdict-json.",

	"abcd identity": "Record the identity block and propose drift corrections: " +
		"Writes nothing bare, only the block and its pointer; refuses bare, naming `abcd lint identity`.",
	"abcd identity init": "Record this repository's identity block and the pointer to it: " +
		"Writes the block and the pointer; refuses without --title and --tagline when no block exists.",
	"abcd identity render": "Print the correction for every drifted surface as a unified diff: " +
		"Writes nothing; refuses a repository that records no identity block.",

	"abcd implement": "Share one autonomous run between sessions, from joining to reporting: " +
		"Writes nothing bare, only the machine-scoped run state; refuses an unknown sub-verb.",
	"abcd implement check": "Ask whether this session may take a step before taking it: " +
		"Writes a run-log line only on a refusal; refuses a step the second session's bounds forbid.",
	"abcd implement claim": "Claim a record for this session before opening its lane: " +
		"Writes the claim and a run-log line; refuses a record another session holds.",
	"abcd implement join": "Join the run with a stated role: " +
		"Writes the session's record and a session_open line; refuses the other role on a resume.",
	"abcd implement leave": "Leave the run, releasing every claim this session holds: " +
		"Writes the releases and a session_close line; refuses without --session.",
	"abcd implement load": "Check the machine's load before abcd's own tests start: " +
		"Writes a load event to the run log inside a run; refuses an unknown --site, never a loaded machine.",
	"abcd implement log": "Append one of the run's events to today's run log: " +
		"Writes one line; refuses the claim, window, and session events their own verbs write.",
	"abcd implement mode": "Open a window by logging its division mode: " +
		"Writes a window_mode line; refuses any session but the first.",
	"abcd implement release": "Release this session's claim on a record: " +
		"Writes the release and a claim_released line; refuses a claim another session holds.",
	"abcd implement report": "Derive the comparison of the division modes from the run log: " +
		"Writes nothing; refuses --date and --log together.",

	"abcd inbox": "List the reports managed repositories filed back to abcd, newest first: " +
		"Writes nothing; refuses any argument.",
	"abcd inbox promote": "File one report as a capture in abcd's own ledger: " +
		"Writes the capture and marks the report promoted; refuses outside abcd's own checkout.",
	"abcd inbox show": "Render one report whole: " +
		"Writes nothing; refuses an id the inbox does not hold.",

	"abcd intent": "File a draft intent from quoted text, or render the intent store's status bare: " +
		"Writes the draft into drafts/; refuses a lone word.",
	"abcd intent audit": "Emit a shipped intent's audit request, or check the issue and intent joins with --issue-drift: " +
		"Writes nothing; refuses an intent not shipped.",
	"abcd intent audit ingest": "Ingest an intent-audit verdict into the shipped intent: " +
		"Writes its Audit Notes; refuses without --verdict-json.",
	"abcd intent condition": "Read or disposition a shipped intent's scope conditions: " +
		"Writes a dated condition block; refuses an unresolved occasion or thin grounds.",
	"abcd intent hold": "Hold a draft or planned intent so that planning refuses it: " +
		"Writes the held line with its reason; refuses without --reason.",
	"abcd intent link": "Link a planned intent to an existing spec: " +
		"Writes the intent's spec_id; refuses an intent that is not planned.",
	"abcd intent plan": "Plan a draft intent by minting and linking its spec, or stamp a planned one's scope conditions: " +
		"Writes both records; refuses an intent on hold.",
	"abcd intent ready": "Report whether an intent is ready to implement, exiting 1 when not: " +
		"Writes its grounds only with --grounds; refuses malformed grounds.",
	"abcd intent unhold": "Lift an intent's hold: " +
		"Writes the removal of its held line; refuses a record not held.",

	"abcd launch": "Preview the public launch bundle, its secret scan, and the release gates: " +
		"Writes nothing; refuses without --dry-run.",
	"abcd launch archive": "Render the release's plugin archive: " +
		"Writes the archive into --out; refuses with exit 1 when --verify finds the catalogue does not pin it.",
	"abcd launch scaffold": "Scaffold the changelog-driven release gate: " +
		"Writes the release workflows and runbook; refuses to overwrite a hand-edited one without --confirm.",
	"abcd launch ship": "Cut a release, deriving its version and records from what shipped: " +
		"Writes the CHANGELOG heading, RELEASE.md, and the archive pin; refuses a cut its gates stop.",

	"abcd lint": "Check this repository against the conventions, every target included: " +
		"Writes nothing; refuses with exit 2 on an error finding and exit 1 on warnings alone.",
	"abcd lint docs": "Lint the docs for change-narration, broken links, citations, and stray root markdown: " +
		"Writes nothing; refuses a tree with a blocker finding.",
	"abcd lint identity": "Show this repository's identity block and every surface held to it: " +
		"Writes nothing; refuses a repository that records no identity block.",
	"abcd lint outbound": "Judge one outbound text against the session-URL and tool-footer policy: " +
		"Writes nothing; refuses a text carrying either with exit 1.",
	"abcd lint site": "Gate the built website, rendering it first when absent: " +
		"Writes only inside the output directory; refuses a site failing any gate with exit 1.",

	"abcd memory": "Render the memory store's status: " +
		"Writes nothing; refuses outside a git checkout.",
	"abcd memory ask": "Query memory and synthesise a cited answer: " +
		"Writes a memory page only with --file-back; refuses outside a git checkout.",
	"abcd memory ingest": "Distil a local file or an https source into cited memory pages: " +
		"Writes the pages; refuses a URL that is not https.",
	"abcd memory lint": "Health-check the whole memory store: " +
		"Writes a lint report; refuses a store with a blocker finding.",

	"abcd mode": "Print or set whose answer the agent loop is waiting on: " +
		"Writes the state only when setting it; refuses an unknown state or a checkout with no local tier.",

	"abcd peers": "List the records sibling worktrees and local branches hold that this checkout does not: " +
		"Writes nothing; refuses outside a git checkout.",

	"abcd reading": "Render the cold-reading assembler's state: " +
		"Writes nothing; refuses any argument.",
	"abcd reading assemble": "Assemble one reading's input and its hashed manifest at one position: " +
		"Writes both artefacts; refuses a target that is not HEAD or a commit sha.",
	"abcd reading ingest": "Validate the JSON one cold reading returned: " +
		"Writes its reading records; refuses output the position's licence does not allow.",

	"abcd report": "File a defect report or an enhancement proposal about abcd: " +
		"Writes it into your account's inbox; refuses a malformed field or a filesystem path.",

	"abcd rules": "Render the active rule set, or the one domain named: " +
		"Writes nothing; refuses an unknown domain.",

	"abcd site": "Report what the website declares and what was built: " +
		"Writes nothing; refuses any argument.",
	"abcd site build": "Render the website into the output directory: " +
		"Writes only inside that directory; refuses a non-empty directory it did not write.",
	"abcd site setup": "Take the website from this checkout to a live address: " +
		"Writes its files, and the forge and host changes once confirmed; refuses a folder abcd does not manage.",

	"abcd spec": "Render the spec store's status: " +
		"Writes nothing; refuses outside a git checkout.",
	"abcd spec close": "Close a spec, and ship its intent when no open spec names it: " +
		"Writes the moves to closed/ and shipped/; refuses to ship an intent with no impact.",

	"abcd statusline": "Render abcd's status-line row from the host's payload on stdin: " +
		"Writes nothing; never refuses.",

	"abcd update": "Swap the PATH-installed binary for a verified release, or with --check only compare: " +
		"Writes the swapped binary; refuses a binary it cannot prove is abcd's.",
}

// SentenceFor returns the sentence the manifest declares for the command at
// path ("abcd capture list"), and whether it declares one.
func SentenceFor(path string) (string, bool) {
	s, ok := sentences[path]
	return s, ok
}

// SentencePaths returns every command path the manifest declares a sentence
// for, sorted.
func SentencePaths() []string {
	out := make([]string, 0, len(sentences))
	for p := range sentences {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
