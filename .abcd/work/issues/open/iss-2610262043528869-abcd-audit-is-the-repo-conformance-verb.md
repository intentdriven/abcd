---
schema_version: 1
id: "iss-2610262043528869"
slug: "abcd-audit-is-the-repo-conformance-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1167 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-85, itd-35]
finding_ref: cfk-af5be1b9fc3a0cf1
---

`abcd audit` is the repo-conformance verb in itd-85 and the lifeboat-integrity verb in itd-35

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "A new read-only `abcd audit` verb, distinct from `ahoy doctor` (doctor = tool-setup health; audit = repo conformance)."
- End B: `.abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:24` — "abcd produces tamper-evident lifeboats via `/abcd:audit lifeboat`."

itd-85 defines `abcd audit` as the read-only repo-conformance check, separate from `ahoy doctor`. itd-35 uses `/abcd:audit lifeboat <path>` to re-verify a lifeboat's hash chain, which has nothing to do with repo conformance. Both drafts claim the same verb for unrelated jobs, and neither mentions the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1167.
