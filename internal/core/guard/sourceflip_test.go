package guard

import "testing"

// TestSourceLedgerFlipIsLeftToThePerson — iss-2609252007448074. Flipping a
// source-ledger line to public citation is the person's act under adr-41 gate
// 2, and the plugin pages tell an agent never to run it, but nothing stopped
// one: the default registry did not name it. ledger.go still refuses a
// confidential or non-citable source mechanically, so what the entry guards is
// provenance — a line that says a human cited it when a model did. Recording a
// line, listing the ledger, and every other source verb stay allowed.
func TestSourceLedgerFlipIsLeftToThePerson(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`abcd library ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd library ledger --flip=3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd --json library ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd library --corpus /srv/corpus ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`"$CLAUDE_PLUGIN_ROOT"/abcd library ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`sh -c 'abcd library ledger --flip 3'`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd library ledger --list`, VerdictAllow, ""},
		{`abcd library ledger --decision adr-7 --claim c --source k --influence i`, VerdictAllow, ""},
		{`abcd library cite-check README.md`, VerdictAllow, ""},
		{`abcd capture "an agent ran abcd library ledger --flip 3"`, VerdictAllow, ""},
	})
}

// TestRetiredSourceLedgerFlipIsStillLeftToThePerson holds the retired spelling
// to the same refusal: a binary older than the library verb still answers
// `abcd source ledger --flip`, and such a binary is often reachable (the
// plugin root's copy, an older build), so the flip it writes is the same
// false record of who decided.
func TestRetiredSourceLedgerFlipIsStillLeftToThePerson(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`abcd source ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip-retired"},
		{`abcd source ledger --flip=3`, VerdictBlock, "abcd-source-ledger-flip-retired"},
		{`abcd source --corpus /srv/corpus ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip-retired"},
		{`"$CLAUDE_PLUGIN_ROOT"/abcd source ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip-retired"},
		{`abcd source ledger --list`, VerdictAllow, ""},
		{`abcd capture "an agent ran abcd source ledger --flip 3"`, VerdictAllow, ""},
	})
}
