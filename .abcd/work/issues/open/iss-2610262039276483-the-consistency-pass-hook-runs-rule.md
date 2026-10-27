---
schema_version: 1
id: "iss-2610262039276483"
slug: "the-consistency-pass-hook-runs-rule"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 861 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:151"
remedy: "none (filed automatically)"
related_intents: [itd-2609291923559186]
finding_ref: cfk-2dd1ded1fda1c5f5
---

The consistency-pass hook runs rule checks over the whole corpus at commit time, but the lint contract's pre-commit tier lints only staged lines so that old findings never block

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:151` — "the rule checks cover the whole corpus; the model pass covers the staged records and their links"
- End B: `.abcd/development/brief/05-internals/06-lint.md:52` — "so a pre-existing finding on an untouched line never blocks a commit"

itd-2609291923559186 rules that its pre-commit hook runs the rule checks over the whole corpus and, under the 'stop' setting, refuses the commit and names the finding. The lint contract's design target for the pre-commit tier lints only the files a commit stages, filtered to the staged lines, so that a pre-existing finding on an untouched line never blocks a commit. The two documents draw the pre-commit lint tier with opposite scope: under 'stop', a finding in an untouched record blocks a commit that the lint contract says must not be blocked.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 861.
