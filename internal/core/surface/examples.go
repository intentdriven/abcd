package surface

import "sort"

// examples.go — one worked example per visible command, leaf or parent
// (iss-2609100508565741, itd-2610301346085246 ac-1): the shortest legal
// invocation, with every required positional and flag filled in. A verb's
// required inputs were learned from its first refusal rather than its help,
// because a list of flags does not say which combination is a legal call; this
// is that combination, written once. A command that takes no required input
// gets its bare call, or the one-flag call it is most often run with.
//
// The front door (internal/surface/cli) sets each command's Example from here,
// so it renders under the verb's own --help, in the generated CLI reference and
// in the command facts `abcd explain` prints. A test holds every visible
// command to an entry, but for one whose own builder writes a longer Example
// (abcd drain), and each entry to the command it names: the path it starts
// with, flags the command registers, and every required flag its Use line
// declares. Ids and values are placeholders of the right shape, and names come
// from the reserved example namespace (examples-use-reserved-identifiers).
var examples = map[string]string{
	"abcd": "abcd",

	"abcd agents": "abcd agents --all",

	"abcd ahoy":                     "abcd ahoy",
	"abcd ahoy connect":             "abcd ahoy connect local --base-url http://127.0.0.1:8080/v1 --model example-model --home none",
	"abcd ahoy credential":          "abcd ahoy credential",
	"abcd ahoy doctor":              "abcd ahoy doctor",
	"abcd ahoy install":             "abcd ahoy install",
	"abcd ahoy remote":              "abcd ahoy remote",
	"abcd ahoy remote apply":        "abcd ahoy remote apply",
	"abcd ahoy shared-folder":       "abcd ahoy shared-folder /Users/Shared",
	"abcd ahoy uninstall":           "abcd ahoy uninstall",
	"abcd ahoy worktree":            "abcd ahoy worktree",
	"abcd ahoy worktree add":        "abcd ahoy worktree add example-lane",
	"abcd ahoy worktree list":       "abcd ahoy worktree list",
	"abcd ahoy worktree prune":      "abcd ahoy worktree prune --dry-run",
	"abcd ahoy worktree trim-cache": "abcd ahoy worktree trim-cache --dry-run",

	"abcd announce": "abcd announce merging iss-2609010000000001",

	"abcd banlist":         "abcd banlist",
	"abcd banlist add":     "abcd banlist add --private acme-internal 'acme-internal\\.example\\.com'",
	"abcd banlist list":    "abcd banlist list --private",
	"abcd banlist migrate": "abcd banlist migrate",
	"abcd banlist remove":  "abcd banlist remove --private acme-internal",

	"abcd build":      "abcd build itd-2609010000000001",
	"abcd build next": "abcd build next",

	"abcd capture":             `abcd capture "the board drops its last row on an empty ledger" --remedy "render the row before the count"`,
	"abcd capture admit":       `abcd capture admit rdi-2609010000000001 --grounds "the widened configuration is one the next release has to serve"`,
	"abcd capture defer":       `abcd capture defer iss-2609010000000001 --after v0.1.0 --reason "the fix needs the parser rewrite that lands next cycle"`,
	"abcd capture disposition": `abcd capture disposition rdi-2609010000000001 --state accepted --grounds "pursued: the tension is real and the next reading will show it again"`,
	"abcd capture link":        "abcd capture link iss-2609010000000001 --blocked-by iss-2609010000000002",
	"abcd capture list":        "abcd capture list --open",
	"abcd capture mentions":    "abcd capture mentions",
	"abcd capture migrate":     "abcd capture migrate",
	"abcd capture promote":     "abcd capture promote iss-2609010000000001",
	"abcd capture reframe":     `abcd capture reframe --occasioned-by rdi-2609010000000001 --grounds "the reading showed the construal assumed a single operator" --open`,
	"abcd capture remedy":      `abcd capture remedy iss-2609010000000001 "make the map before the first write"`,
	"abcd capture resolve":     `abcd capture resolve iss-2609010000000001 "fixed by the parser change" --impact fix`,
	"abcd capture surprise":    `abcd capture surprise --occasioned-by rdi-2609010000000001 "the proposal nobody expected ranked first"`,
	"abcd capture wontfix":     `abcd capture wontfix iss-2609010000000001 "the behaviour is the documented one"`,

	"abcd checklist":        "abcd checklist",
	"abcd checklist add":    `abcd checklist add "Regenerate the CLI reference"`,
	"abcd checklist render": "abcd checklist render",
	"abcd checklist set":    "abcd checklist set regenerate-the-cli-reference --state done",
	"abcd checklist show":   "abcd checklist show regenerate-the-cli-reference",

	"abcd dashboard":        "abcd dashboard",
	"abcd dashboard start":  "abcd dashboard start",
	"abcd dashboard status": "abcd dashboard status",
	"abcd dashboard stop":   "abcd dashboard stop",

	"abcd decide": `abcd decide "Decisions are one flat file each"`,

	"abcd disembark":               "abcd disembark",
	"abcd disembark coverage":      "abcd disembark coverage probe-report.json",
	"abcd disembark graveyard":     "abcd disembark graveyard ../lifeboat --lessons-json lessons.json",
	"abcd disembark pack":          "abcd disembark pack . ../lifeboat",
	"abcd disembark plan":          "abcd disembark plan .",
	"abcd disembark press-release": "abcd disembark press-release ../lifeboat",
	"abcd disembark principles":    "abcd disembark principles ../lifeboat",
	"abcd disembark probe":         "abcd disembark probe .",
	"abcd disembark review":        "abcd disembark review ../lifeboat .",

	"abcd docs":                 "abcd docs",
	"abcd docs cite":            "abcd docs cite",
	"abcd docs cite confirm":    "abcd docs cite confirm https://example.com/article.html",
	"abcd docs cite refresh":    "abcd docs cite refresh",
	"abcd docs fidelity":        "abcd docs fidelity",
	"abcd docs fidelity record": "abcd docs fidelity record --verdict-json verdict.json",

	"abcd embargo":         "abcd embargo",
	"abcd embargo check":   "abcd embargo check --range v0.11.0..HEAD",
	"abcd embargo close":   "abcd embargo close GHSA-xxxx-xxxx-xxxx",
	"abcd embargo fixed":   "abcd embargo fixed GHSA-xxxx-xxxx-xxxx 0123abc",
	"abcd embargo intake":  "abcd embargo intake",
	"abcd embargo publish": "abcd embargo publish",
	"abcd embargo triage":  `abcd embargo triage GHSA-xxxx-xxxx-xxxx not-a-vulnerability --by "<who decided>" --reason "the behaviour is the documented one"`,

	"abcd embark":         "abcd embark",
	"abcd embark from":    "abcd embark from ../lifeboat",
	"abcd embark lessons": "abcd embark lessons ../lifeboat",
	"abcd embark probe":   "abcd embark probe ../lifeboat",

	"abcd explain": "abcd explain ideate record --input verdict-json",

	"abcd guard":          "abcd guard",
	"abcd guard allow":    "abcd guard allow --once --command 'git branch -D feat/done'",
	"abcd guard approve":  "abcd guard approve --dir /work/example-repo --base 0123456789abcdef0123456789abcdef01234567",
	"abcd guard check":    "abcd guard check --command 'git push --force origin main'",
	"abcd guard hook":     "abcd guard hook < payload.json",
	"abcd guard question": "abcd guard question draft.json",

	"abcd history":             "abcd history",
	"abcd history capture":     "abcd history capture 0123abcd-session.jsonl",
	"abcd history discard":     "abcd history discard 0123abcd-session.raw --yes",
	"abcd history drain":       "abcd history drain",
	"abcd history ingest":      "abcd history ingest --into .",
	"abcd history list":        "abcd history list",
	"abcd history migrate":     "abcd history migrate",
	"abcd history reconstruct": "abcd history reconstruct 0123abcd-session",
	"abcd history separation":  "abcd history separation",
	"abcd history show":        "abcd history show 0123abcd-session",
	"abcd history staged":      "abcd history staged",

	"abcd ideate":        "abcd ideate",
	"abcd ideate record": "abcd ideate record widen-the-public-api --verdict-json verdict.json",

	"abcd identity":        "abcd identity",
	"abcd identity init":   `abcd identity init --title "Example Widgets" --tagline "Widgets that explain themselves"`,
	"abcd identity render": "abcd identity render",

	"abcd implement":               "abcd implement",
	"abcd implement check":         "abcd implement check lane --session s-example",
	"abcd implement claim":         "abcd implement claim iss-2609010000000001 --session s-example --lane cli",
	"abcd implement join":          "abcd implement join --session s-example --role first",
	"abcd implement leave":         "abcd implement leave --session s-example",
	"abcd implement load":          "abcd implement load --site preflight",
	"abcd implement log":           "abcd implement log lane_open --session s-example",
	"abcd implement mode":          "abcd implement mode single --session s-example",
	"abcd implement outage":        "abcd implement outage",
	"abcd implement outage ack":    "abcd implement outage ack --session s-example",
	"abcd implement outage clear":  "abcd implement outage clear --session s-example --reason \"the network is back\"",
	"abcd implement outage probe":  "abcd implement outage probe --session s-example",
	"abcd implement outage record": "abcd implement outage record --session s-example --service network --kind tool --lane cli --what \"git push\"",
	"abcd implement receipt":       "abcd implement receipt review-receipt.json --run run-2609010000000001",
	"abcd implement record":        "abcd implement record",
	"abcd implement release":       "abcd implement release iss-2609010000000001 --session s-example",
	"abcd implement report":        "abcd implement report",
	"abcd implement resume":        "abcd implement resume --run run-2609010000000001",
	"abcd implement status":        "abcd implement status",
	"abcd implement step":          "abcd implement step",
	"abcd implement stop":          "abcd implement stop",
	"abcd implement widen":         "abcd implement widen internal/widget/b.go --lane lane-1 --run run-2609010000000001",

	"abcd inbox":         "abcd inbox",
	"abcd inbox promote": "abcd inbox promote rpt-2609010000000001",
	"abcd inbox show":    "abcd inbox show rpt-2609010000000001",

	"abcd intent":                    `abcd intent "let a reviewer read the board from a phone"`,
	"abcd intent audit":              "abcd intent audit itd-2609010000000001",
	"abcd intent audit ingest":       "abcd intent audit ingest --verdict-json verdict.json",
	"abcd intent condition":          "abcd intent condition itd-2609010000000001",
	"abcd intent consistency":        "abcd intent consistency",
	"abcd intent consistency ingest": "abcd intent consistency ingest --findings-json findings.json",
	"abcd intent edge":               "abcd intent edge itd-2609010000000001 --blocked-by itd-2609010000000002",
	"abcd intent hold":               `abcd intent hold itd-2609010000000001 --reason "waiting on the product thinker's ruling on scope"`,
	"abcd intent interview":          "abcd intent interview itd-2609010000000001",
	"abcd intent link":               "abcd intent link itd-2609010000000001 spc-2609010000000002",
	"abcd intent plan":               "abcd intent plan itd-2609010000000001",
	"abcd intent prepass":            "abcd intent prepass itd-2609010000000001 --json",
	"abcd intent ready":              "abcd intent ready itd-2609010000000001",
	"abcd intent reclassify":         `abcd intent reclassify itd-2609010000000001 --kind superseded --by itd-2609010000000002 --reason "absorbed by the later intent"`,
	"abcd intent target":             "abcd intent target itd-2609010000000001 v0.11.0",
	"abcd intent unhold":             "abcd intent unhold itd-2609010000000001",

	"abcd lab":           "abcd lab",
	"abcd lab harvest":   "abcd lab harvest lab-260901000000-0123abc",
	"abcd lab matrix":    "abcd lab matrix lab-260901000000-0123abc matrix.json",
	"abcd lab mint":      `abcd lab mint "does the snapshot keep the checkout's hooks from firing?"`,
	"abcd lab preflight": "abcd lab preflight lab-260901000000-0123abc",
	"abcd lab record":    "abcd lab record lab-260901000000-0123abc bare-status",
	"abcd lab sweep":     "abcd lab sweep lab-260901000000-0123abc",

	"abcd land": "abcd land",

	"abcd landings": "abcd landings --since v0.11.0",

	"abcd launch":           "abcd launch --dry-run",
	"abcd launch archive":   "abcd launch archive --out dist",
	"abcd launch manifests": "abcd launch manifests --tree public",
	"abcd launch receipts":  "abcd launch receipts",
	"abcd launch scaffold":  "abcd launch scaffold",
	"abcd launch ship":      "abcd launch ship --dry-run",

	"abcd library":              "abcd library",
	"abcd library add":          `abcd library add article.md --public --key example-paper-2026 --title "Example Widgets in Practice"`,
	"abcd library ask":          `abcd library ask "why do record ids carry a timestamp?"`,
	"abcd library cite-check":   "abcd library cite-check draft.md",
	"abcd library declassify":   "abcd library declassify example-paper-2026",
	"abcd library ingest":       "abcd library ingest https://example.com/article.html",
	"abcd library init":         "abcd library init",
	"abcd library ledger":       "abcd library ledger --list",
	"abcd library lint":         "abcd library lint",
	"abcd library sync-banlist": "abcd library sync-banlist",

	"abcd lint":          "abcd lint",
	"abcd lint docs":     "abcd lint docs",
	"abcd lint identity": "abcd lint identity",
	"abcd lint outbound": "abcd lint outbound pr-body.md",
	"abcd lint site":     "abcd lint site",

	"abcd memory":        "abcd memory",
	"abcd memory add":    `abcd memory add --title "The tests need the race flag" --recall test --recall race --by Example:model-1 < note.txt`,
	"abcd memory list":   "abcd memory list",
	"abcd memory remove": "abcd memory remove mem-2609010000000001",

	"abcd mode":            "abcd mode",
	"abcd mode autonomous": "abcd mode autonomous --until 09:00",
	"abcd mode back":       "abcd mode back",
	"abcd mode queue":      `abcd mode queue "approve the embargo close for the example advisory"`,

	"abcd peers": "abcd peers",

	"abcd reading":          "abcd reading",
	"abcd reading assemble": "abcd reading assemble --position widening --target HEAD",
	"abcd reading ingest":   "abcd reading ingest --reading-json reading.json",

	"abcd reflect":           "abcd reflect v0.11.0",
	"abcd reflect interview": "abcd reflect interview v0.11.0",
	"abcd reflect write":     "abcd reflect write v0.11.0 --answers answers.json",

	"abcd report": "abcd report --template",

	"abcd reserve": "abcd reserve iss-2609010000000001 --footprint internal/core/example/example.go",

	"abcd rules": "abcd rules",

	"abcd scribe":          "abcd scribe",
	"abcd scribe assemble": "abcd scribe assemble --run rdg-2609010000000001 --dispositions dispositions.md",
	"abcd scribe ingest":   "abcd scribe ingest --scribe-json scribe.json --dispositions dispositions.md",

	"abcd session":           "abcd session",
	"abcd session blueprint": "abcd session blueprint",
	"abcd session page":      "abcd session page --out brief-page",
	"abcd session serve":     "abcd session serve",
	"abcd session setup":     "abcd session setup --where local",
	"abcd session start":     "abcd session start",

	"abcd site":       "abcd site",
	"abcd site build": "abcd site build",
	"abcd site setup": "abcd site setup",

	"abcd spec":       "abcd spec",
	"abcd spec close": "abcd spec close spc-2609010000000001",

	"abcd status": "abcd status",

	"abcd statusline": "abcd statusline < payload.json",

	"abcd sweep": "abcd sweep",

	"abcd update":      "abcd update --check",
	"abcd update news": "abcd update news",
}

// ExampleFor returns the worked example the manifest declares for the command
// at path ("abcd capture resolve"), and whether it declares one.
func ExampleFor(path string) (string, bool) {
	e, ok := examples[path]
	return e, ok
}

// ExamplePaths returns every command path the manifest declares an example
// for, sorted.
func ExamplePaths() []string {
	out := make([]string, 0, len(examples))
	for p := range examples {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
