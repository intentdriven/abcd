---
schema_version: 1
id: "iss-2610262031298011"
slug: "itd-16-lands-its-hash-chain-under-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 137 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-16]
finding_ref: cfk-84c919f8a578d022
---

itd-16 lands its hash chain under .abcd/logbook/audit/, the retired logbook location

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:23` — "entries land in a Merkle chain at `.abcd/logbook/audit/chain-<ts>/`"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run logs, state and reports. It is retired and must not be re-minted"

itd-16 says the canonicalised entries land in a Merkle chain at .abcd/logbook/audit/chain-< ts>/, and its verification report goes under the same root. The naming chapter says .abcd/logbook/ is retired and must not be re-minted. The draft cannot be built to its own paths without re-minting the retired name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 137.
