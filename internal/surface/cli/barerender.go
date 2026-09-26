package cli

// barerender.go — the bare-render discipline's exceptions, each a decision
// with its reason rather than a gap (iss-2609091642508271).
//
// A bare verb renders its namespace's state and writes nothing
// (.abcd/development/brief/02-constraints/04-naming.md): that is what makes a
// verb safe to type when you do not know what it does. The verbs below do not,
// and each says why. TestEveryTopLevelVerbRendersStateBareOrIsAnException runs
// every other visible top-level verb bare in a scratch repository and fails
// when one prints only its usage, or nothing; a verb added later fails the same
// way until it renders or is listed here. The brief's one enumeration of the
// exceptions (04-surfaces/README.md, "Bare invocation") is held to this table.
var bareRenderExceptions = map[string]string{
	"decide": "its one operand is the quoted title it mints a record from, so bare " +
		"refuses (exit 2) naming the form, and writes nothing",
	"disembark": "a parent of stage sub-verbs that each act on a named repository or " +
		"lifeboat; with no operand there is no state to render, so bare prints its sub-verbs",
	"docs": "a parent holding the citation-baseline writer alone; the documentation's " +
		"state is `abcd lint docs`, so bare prints its sub-verb",
	"embark": "a parent whose sub-verbs act on a named lifeboat; with no operand there " +
		"is no state to render, so bare prints its sub-verbs",
	"guard": "it judges one command handed to it and keeps no standing state to " +
		"render, so bare prints its usage",
	"history": "not yet conformant: its store's state renders through `history list` and " +
		"`history staged`, the shape the naming rule forbids, while bare prints its sub-verbs",
	"ideate": "the gauntlet runs in the host on a named idea, and the verb keeps no " +
		"standing state beyond the records it writes, so bare prints its usage and sub-verbs",
	"identity": "its report moved to `abcd lint identity`; for one release bare names " +
		"that invocation and exits non-zero, because its init and render sub-verbs stay",
	"launch": "its state is the release preview, asked for with --dry-run; bare refuses " +
		"(exit 1) naming the flag, because publishing is not wired",
	"report": "it files a report from a file or the editor, so bare opens the editor on " +
		"a terminal and refuses (exit 2) anywhere else",
	"statusline": "its row belongs to a managed repository: there bare renders it with or " +
		"without the payload (TestStatuslineEmptyStdinStillRendersTheBadge), and anywhere else, " +
		"this test's scratch repository included, abcd has no row and bare prints nothing of " +
		"its own, or runs the user's recorded previous status command",
	"update": "bare update is the explicit ask to swap the PATH-installed binary, so it " +
		"writes; `abcd update --check` is its read-only form",
}
