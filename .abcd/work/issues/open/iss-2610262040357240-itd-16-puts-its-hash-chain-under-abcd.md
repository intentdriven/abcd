---
schema_version: 1
id: "iss-2610262040357240"
slug: "itd-16-puts-its-hash-chain-under-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 942 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-16]
finding_ref: cfk-25dcb0c005e49143
---

itd-16 puts its hash chain under .abcd/logbook/; the build sequence says that location is retired and a test fails the build on it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23` — "entries land in a Merkle chain at `.abcd/logbook/audit/chain-<ts>/`"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:178` — "`TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source"

itd-16 puts the Merkle chain segments and verification reports at .abcd/logbook/audit/chain-< ts>/. The build sequence says .abcd/logbook/ is retired and that TestNoRetiredLogbookLocationInSource fails the build on any non-test Go source naming it, so the chain as specified cannot be built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 942.
