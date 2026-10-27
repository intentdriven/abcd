---
schema_version: 1
id: "iss-2610262044218994"
slug: "abcd-audit-names-a-repo-conformance"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1200 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-85, itd-4]
finding_ref: cfk-27779cd9eb387017
---

`abcd audit` names a repo-conformance check in one record and is reserved for the hash-chain merkle audit in another

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "a read-only check that tells you, in one command, whether a repository follows the working conventions"
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:45` — "`/abcd:audit` (reserved, see itd-16 hash-chain-merkle-audit)"

itd-85 uses `abcd audit` for a read-only repository-conventions check. itd-4 reserves `/abcd:audit` for itd-16's hash-chain merkle audit. One verb name stands for two unrelated concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1200.
