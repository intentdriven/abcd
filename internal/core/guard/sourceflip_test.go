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
		{`abcd source ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd source ledger --flip=3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd --json source ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd source --corpus /srv/corpus ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`"$CLAUDE_PLUGIN_ROOT"/abcd source ledger --flip 3`, VerdictBlock, "abcd-source-ledger-flip"},
		{`sh -c 'abcd source ledger --flip 3'`, VerdictBlock, "abcd-source-ledger-flip"},
		{`abcd source ledger --list`, VerdictAllow, ""},
		{`abcd source ledger --decision adr-7 --claim c --source k --influence i`, VerdictAllow, ""},
		{`abcd source cite-check README.md`, VerdictAllow, ""},
		{`abcd capture "an agent ran abcd source ledger --flip 3"`, VerdictAllow, ""},
	})
}
