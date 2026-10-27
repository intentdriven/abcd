---
schema_version: 1
id: "iss-2610262041018201"
slug: "the-routing-intent-says-nothing-reads"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 971 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401]
finding_ref: cfk-4d7535cea77c9969
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's premise was true when written and the delivery is recorded by its closed spec and the matrix, so the two ends describe before and after rather than contradict."
---

The routing intent says nothing reads the oracle choice today; the verification matrix has close read the intent-auditor's own oracle role and send it to a provider.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:56` — "When `oracle.roles.intent-auditor` reaches a provider connection a hand-run close sends that request there"
- End B: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26` — "backend is one repo-wide value that nothing reads"

itd-2609170822093401 starts from the premise that the oracle backend is one repo-wide value nothing reads. The verification matrix describes shipped behaviour in which `oracle.roles.intent-auditor` is read per agent role and a hand-run close sends the audit request to that provider. Per-role routing that is read cannot also be a single value that nothing reads, and the matrix's `oracle.roles` key and the intent's routing table name the same per-agent choice two ways.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 971.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's premise was true when written and the delivery is recorded by its closed spec and the matrix, so the two ends describe before and after rather than contradict.
