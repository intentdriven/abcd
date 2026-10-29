---
schema_version: 1
id: "iss-2610262040192600"
slug: "the-lint-contract-calls-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 923 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:10"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-f6093e28e4061b4d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. git history shows the one-release stubs for the moved spellings were removed; the older intent's spelling was true when written."
---

The lint contract calls the docs-currency gate `abcd lint docs`; itd-60 calls the same gate `abcd docs lint`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:10` — "(`abcd lint docs`, config `.abcd/docs-lint.json`) lints `docs/` and the repo-root prose for change-narration"
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:151` — "`abcd docs lint` — change-narration, resolvable cross-links, no stray root documents"

06-lint names the docs-currency gate, which checks docs/ and the root prose for change-narration, broken links and stray root markdown, as `abcd lint docs`. itd-60 describes the same already-shipped gate, with the same three checks, as `abcd docs lint`. That is two invocations for one verb, and only the lint sub-verb is the gate.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 923.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. git history shows the one-release stubs for the moved spellings were removed; the older intent's spelling was true when written.
