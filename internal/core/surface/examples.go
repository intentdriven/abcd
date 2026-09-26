package surface

import "sort"

// examples.go — one worked example per verb that takes a required input
// (iss-2609100508565741): the shortest legal invocation, with every required
// positional and flag filled in. A verb's required inputs were learned from its
// first refusal rather than its help, because a list of flags does not say
// which combination is a legal call; this is that combination, written once.
//
// The front door (internal/surface/cli) sets each command's Example from here,
// so it renders under the verb's own --help and in the generated CLI reference.
// A test holds every visible command whose Use line declares a required input
// to an entry, and each entry to the command it names: the path it starts with,
// flags the command registers, and every required flag its Use line declares.
// Ids and values are placeholders of the right shape, and names come from the
// reserved example namespace (examples-use-reserved-identifiers).
var examples = map[string]string{
	"abcd banlist add":    "abcd banlist add --private acme-internal 'acme-internal\\.example\\.com'",
	"abcd banlist remove": "abcd banlist remove --private acme-internal",

	"abcd capture disposition": `abcd capture disposition rdi-2609010000000001 --state accepted --grounds "pursued: the tension is real and the next reading will show it again"`,
	"abcd capture link":        "abcd capture link iss-2609010000000001 --blocked-by iss-2609010000000002",
	"abcd capture promote":     "abcd capture promote iss-2609010000000001",
	"abcd capture resolve":     `abcd capture resolve iss-2609010000000001 "fixed by the parser change" --impact fix`,
	"abcd capture wontfix":     `abcd capture wontfix iss-2609010000000001 "the behaviour is the documented one"`,

	"abcd decide": `abcd decide "Record ids are minted from a timestamp"`,

	"abcd disembark coverage":      "abcd disembark coverage probe-report.json",
	"abcd disembark graveyard":     "abcd disembark graveyard ../lifeboat --lessons-json lessons.json",
	"abcd disembark pack":          "abcd disembark pack . ../lifeboat",
	"abcd disembark press-release": "abcd disembark press-release ../lifeboat",
	"abcd disembark principles":    "abcd disembark principles ../lifeboat",
	"abcd disembark review":        "abcd disembark review ../lifeboat .",

	"abcd embark from":  "abcd embark from ../lifeboat",
	"abcd embark probe": "abcd embark probe ../lifeboat",

	"abcd history discard":     "abcd history discard 0123abcd-session.raw --yes",
	"abcd history reconstruct": "abcd history reconstruct 0123abcd-session",
	"abcd history show":        "abcd history show 0123abcd-session",

	"abcd ideate record": "abcd ideate record widen-the-public-api --verdict-json verdict.json",

	"abcd implement check":   "abcd implement check lane --session s-example",
	"abcd implement claim":   "abcd implement claim iss-2609010000000001 --session s-example --lane cli",
	"abcd implement join":    "abcd implement join --session s-example --role first",
	"abcd implement leave":   "abcd implement leave --session s-example",
	"abcd implement load":    "abcd implement load --site preflight",
	"abcd implement log":     "abcd implement log lane_open --session s-example",
	"abcd implement mode":    "abcd implement mode single --session s-example",
	"abcd implement release": "abcd implement release iss-2609010000000001 --session s-example",

	"abcd inbox promote": "abcd inbox promote rpt-2609010000000001",
	"abcd inbox show":    "abcd inbox show rpt-2609010000000001",

	"abcd intent audit":        "abcd intent audit itd-2609010000000001",
	"abcd intent audit ingest": "abcd intent audit ingest --verdict-json verdict.json",
	"abcd intent condition":    "abcd intent condition itd-2609010000000001",
	"abcd intent hold":         `abcd intent hold itd-2609010000000001 --reason "waiting on the product thinker's ruling on scope"`,
	"abcd intent link":         "abcd intent link itd-2609010000000001 spc-2609010000000002",
	"abcd intent plan":         "abcd intent plan itd-2609010000000001",
	"abcd intent ready":        "abcd intent ready itd-2609010000000001",
	"abcd intent reclassify":   `abcd intent reclassify itd-2609010000000001 --kind superseded --by itd-2609010000000002 --reason "absorbed by the later intent"`,
	"abcd intent unhold":       "abcd intent unhold itd-2609010000000001",

	"abcd launch archive": "abcd launch archive --out dist",

	"abcd memory ask":    `abcd memory ask "why do record ids carry a timestamp?"`,
	"abcd memory ingest": "abcd memory ingest https://example.com/paper.pdf",

	"abcd reading assemble": "abcd reading assemble --position widening --target HEAD",
	"abcd reading ingest":   "abcd reading ingest --reading-json reading.json",

	"abcd spec close": "abcd spec close spc-2609010000000001",
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
