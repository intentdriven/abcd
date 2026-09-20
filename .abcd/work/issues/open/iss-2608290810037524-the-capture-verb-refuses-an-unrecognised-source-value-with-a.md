---
schema_version: 1
id: "iss-2608290810037524"
slug: "the-capture-verb-refuses-an-unrecognised-source-value-with-a"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "intent-implementation-run"
found_at: "internal/surface/cli"
---

The capture verb refuses an unrecognised source value with a message that names neither the offending flag nor the accepted set, and blames the wrong layer: an invalid source produces a malformed-frontmatter error quoting the value, when the value came from a command-line flag and never reached any frontmatter the caller wrote. The accepted values are discoverable only by grepping existing records. The flag's help text lists no enumeration either. The same shape likely applies to category and to any other closed-set flag on this path. A closed set should be named in the refusal and in the help text.

**Corroboration (2026-09-10, autonomous-run field experiment in a managed
repository).** The prediction in the last two sentences held. Two independent
sessions hit the `--category` half of the same defect within two days, and
neither could recover from the error text.

One session was refused `--category defect` with `{"error": "malformed
frontmatter: invalid category \"defect\""}` and, finding nothing in `--help`
beyond "issue category (default observation)", resorted to grepping the ledger
for `^category:`. That is unreliable in both directions: it cannot show a valid
value no record has yet used, and the hand-written inbound records that session
was working from carry `future-work-seed`, which appears in no committed record
of that repository at all — so a reader grepping the ledger concludes
`future-work-seed` is invalid and a reader grepping the neighbouring files
concludes it is valid, and neither can tell which. The other session was refused
`--category test-flake` AND `--source ci-signal` in succession and spent two
round trips filing one issue.

The cost is not only the round trip. `capture` is reached mid-task with the
finding already written out at length; a rejected enum value throws the whole
invocation away and the text has to be re-sent, so the agent's cheapest recovery
is to guess again. Note also that `--severity` DOES enumerate its set in its help
line, so the two flags are already inconsistent with each other and `--category`
is the one that is wrong.

Severity is left as filed; three independent hits argue for raising it, which is
the maintainer's call.

A third independent hit, 2026-09-12, from a second autonomous-run experiment in a managed repository: `capture --severity medium` is refused as malformed frontmatter without naming the vocabulary, and the accepted set of critical, major, minor and nitpick is discoverable only by reading existing records. That is the third of the verb's three closed enumerations to be hit this way, after source and category, by three different sessions, none of which had seen the others' reports. The pattern is now strong enough to state as a rule rather than a list of instances: every closed enumeration this verb validates against will be met blind by somebody, because the refusal names the value it rejected and never the set it was checking against, and the help names the flag and never its vocabulary. Fixing the three fields one at a time would leave the fourth to be discovered the same way.

A fourth independent hit, 2026-09-17, from the Gropius managed-repo session
gropiusllm-56 (relayed to abcd-17): `abcd capture --category test-flake` was
refused with "invalid category" and, in that session's words, "still no list of
valid values". That repository's own ledger carries the recurrence under its own id
(the one ending 0910000001). Two things this hit adds. First, the v0.9.0 source tip
already enumerates the category, source and severity vocabularies in
`capture --help` (the flag help now reads `issue category: bug | documentation |
…`), so the help half of this record is answered at the tip. The session
confirmed afterwards that the binary that ran was the pinned v0.9.0 PATH copy,
so this hit is the refusal half alone: the refusal still names only the
rejected value and never the set, and an agent that reaches it mid-task with
the finding already written out is no better off than before. The refusal half
stands. Note also that `test-flake` is
in no vocabulary at all, so the list alone would have sent that session to pick
a neighbour (`bug` or `observation`); whether the taxonomy wants a value for a
flaky test is a separate question this record does not decide.
