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
