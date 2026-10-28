---
schema_version: 1
id: "iss-2610262042244924"
slug: "the-audit-verb-is-the-hash-chain"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1067 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-16, itd-85]
finding_ref: cfk-bc4937c79eb895cb
---

The audit verb is the hash-chain umbrella in itd-16 and a repo-conformance check in itd-85

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23` — "`/abcd:audit` is the *whole*; specific verifications register as sub-verbs that share the substrate"
- End B: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "a read-only check that tells you, in one command, whether a repository follows the working conventions"

itd-16 makes /abcd:audit an umbrella whose bare form shows the chain head and whose sub-verbs are chain and lifeboat. itd-85 makes bare abcd audit a read-only check of whether a repository follows the working conventions. Both cannot own the same verb: built as written, the bare invocation means two unrelated things, and one draft's sub-verb registry collides with the other's rule set.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1067.
