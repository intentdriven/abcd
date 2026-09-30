---
schema_version: 1
id: "iss-141"
slug: "tamper-evident-receipts"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-100 terminology crosswalk"
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed G): Plan tamper-evident receipts through draft itd-16, and by signing or by a verifiable log?"
remedy: "Waits on planning ruling G (tamper-evident receipts through itd-16, by signing or by a verifiable log): if by signing, extend the semantic-release-gate attestation release.yml already makes (actions/attest over the admitted receipts, Rekor-logged) to every receipt kind a release relies on, and have record-lint's release gate verify it; if by a verifiable log, plan itd-16 as a hash chain in which each receipt carries the previous receipt's digest, with record-lint refusing a broken or rewritten link. Prove either with a test that edits one committed receipt and expects the refusal."
---

tamper-evident receipts: receipts and review directories are hash-anchored and manifests are verified, but nothing is cryptographically signed and there is no append-only chain over the receipt history — integrity evidence, not proof against a capable insider. itd-16 (hash-chain/Merkle audit umbrella, draft) is the design home; this capture tracks the narrower question of whether the existing receipt flow should become tamper-evident (RFC 9162-style verifiable log or signing) independent of the full audit umbrella. The itd-100 crosswalk's tamper-evidence row cites this id.

## Remedy grounds (2026-09-29)

- Release-gate receipts are already signed at release (release.yml, the semantic-gate attestation and its gh attestation verify step), so the body's 'nothing is signed' is stale for that class; the gap is the other receipts and the history. RFC 9162 (experimental) is the reference design for an append-only Merkle log with inclusion and consistency proofs: https://www.rfc-editor.org/rfc/rfc9162 (consulted 2026-09-29).
- Rejected: a self-hosted transparency log, a new service needing sign-off.
