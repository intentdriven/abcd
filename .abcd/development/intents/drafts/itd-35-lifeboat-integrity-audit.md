---
id: itd-35
slug: lifeboat-integrity-audit
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-16]
builds_on: [itd-9]
severity: minor
---

<!-- 2026-05-08: Captured as a sibling to itd-16. itd-16 owns the /abcd:audit umbrella substrate
     (chain mechanics over project conversation/edit history); this intent owns the lifeboat-specific
     application (`/abcd:audit lifeboat`). Both share the substrate's primitives (JCS RFC 8785,
     UUIDv7 RFC 9562, SHA-256 Merkle); they differ in what they audit. The original itd-16
     (2026-05-03 → 2026-05-08) was scoped to lifeboat-Merkle only; that scope was extracted here when
     the user clarified the umbrella/application split. -->

# Lifeboats Carry Their Own Integrity Proof

## Press Release

> **abcd produces tamper-evident lifeboats via `/abcd:audit lifeboat`.** Every disembark generates a Merkle hash chain over the lifeboat's content using the audit substrate from itd-16 (JCS RFC 8785 for canonical JSON, UUIDv7 RFC 9562 for time-ordered identifiers). Embark verifies the chain before unpacking. If anyone modified a single byte in the lifeboat between disembark and embark, verification fails loudly with the specific file flagged. abcd lifeboats can now ship to compliance-sensitive contexts where provenance matters — and `/abcd:audit lifeboat <path>` lets anyone re-verify a lifeboat's integrity outside the embark flow.
>
> "Before this feature, we couldn't ship abcd lifeboats to clients in regulated industries — they couldn't prove provenance," said Dave, compliance lead. "Now the lifeboat carries its own integrity proof; embark either honours it or refuses to unpack. And when an auditor wants to spot-check a lifeboat we received six months ago, `/abcd:audit lifeboat <path>` answers the question without re-running embark."

## Why This Matters

abcd's lifeboats are plain directories today. Anyone with filesystem access can edit them after disembark and before embark. For most uses this is fine. For compliance-sensitive contexts (regulated industries, audit trails, third-party handoffs), tamper-evidence becomes a hard requirement.

This intent is a **specific application** of the audit substrate provided by itd-16: artefact hashes → section hashes → lifeboat root hash, anchored under the same Merkle primitives that itd-16 ships. The application-specific concern is *what* gets hashed (lifeboat artefacts), *when* the hash is computed (at disembark pack time), and *who* verifies it (embark mandatory; `/abcd:audit lifeboat` on demand).

The pattern is well-established (lifted from `~/.abcd/` v0's F-031 / F-032 work): segment hashes → task hashes → feature hashes → milestone Merkle root. Apply that to lifeboats: artefact hashes → section hashes → lifeboat root hash. VAP-compliant means our integrity proofs interop with other audit tooling.

## What's In Scope

- **Lifeboat-specific Merkle implementation** layered on top of itd-16's substrate. Reuses the native hash-chain mechanics (JCS, UUIDv7, SHA-256); adds lifeboat-specific tree shape (artefact → section → root).
- **`_provenance.json` `lifeboat_merkle_root`** field in every lifeboat produced by `/abcd:disembark`. Distinct from itd-16's `audit_chain_root` (conversation/edit history); the two roots cover different artefact sets and live as sibling fields in the same provenance file.
- **Embark verification step**: compute lifeboat hash, compare to manifest, fail-loud on mismatch. Aborts unpack.
- **`/abcd:audit lifeboat <path>`** sub-verb: verifies a specific lifeboat's integrity outside the embark flow. Useful for compliance spot-checks of received lifeboats.
- **Verification report output**: `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}` (sub-tier `lifeboat` per the audit/<sub-tier>-<ts>/ convention in `brief/05-internals/04-universal-patterns.md § 6`).
- **`audit.schema.json` extension** (the public format from itd-16) covers lifeboat-tree shape as one application case.

## What's Out of Scope

- **Cryptographic signing** — chain entries are hashed but not signed. Same scope-out as itd-16.
- **Cross-lifeboat hash chains** — each lifeboat is self-contained; no global root spanning multiple lifeboats.
- **Hash verification of the source repo at disembark time** — we trust filesystem state at pack. Source repo provenance is git's job.
- **Conversation/edit-chain coverage** — that's itd-16's concern. This intent only handles lifeboat artefacts.
- **Real-time verification on every embark step** — verification is a single pre-unpack pass, not interspersed.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a disembark with the audit substrate enabled (per itd-16's `oracle.audit.chain.enabled`), **when** the lifeboat is packed, **then** every artefact has a JCS-canonicalised SHA-256 entry in `_provenance.json` AND a Merkle root hash anchors the full set; the file conforms to `audit.schema.json` (extended for the lifeboat-tree shape).
- **Given** a lifeboat with a valid hash chain, **when** `/abcd:embark from <path>` runs, **then** embark verifies every artefact's hash matches the manifest BEFORE unpacking AND a hash mismatch on any single file aborts the unpack with a clear message naming the offending file and the expected/observed hashes.
- **Given** a received lifeboat whose origin abcd doesn't manage, **when** the user runs `/abcd:audit lifeboat <path>`, **then** the verification report is written to `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}` listing every artefact's pass/fail status — independent of embark.
- **Given** the user runs bare `/abcd:audit` (per itd-16's umbrella), **when** the dispatcher returns, **then** the output mentions `lifeboat` as one of the available sub-verbs alongside the conversation/edit-chain summary — discoverability of the application surfaces from the umbrella.
- **Given** a lifeboat artefact uses UUIDv7 (RFC 9562) IDs, **when** any consumer (embark, audit, downstream tooling) reads the manifest, **then** the IDs sort lexically in the order they were generated AND the timestamp embedded in each ID can be extracted for audit display.
- **Given** the audit substrate is opt-in via `.abcd/config.json` (per itd-16's resolved open question), **when** a user with `oracle.audit.chain.enabled = false` runs disembark, **then** no `_provenance.json` Merkle hashes are written AND embark of the resulting lifeboat skips the lifeboat-verification step without warning — opt-in means opt-in cleanly on both ends.
- **Given** itd-9 (schema migration) and itd-35 (lifeboat audit) both ship, **when** an embark applies a schema migration, **then** the migrated artefact's hash is RE-COMPUTED post-migration and recorded alongside the original hash in the migration log — schema upgrades are not silent over the chain.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **A directory hash has a canonical Go form**: `dirhash` H1 is the SHA-256 of a sorted one-line-per-file summary (`sha256sum $(find . -type f | sort) | sha256sum`), and `HashZip` ignores "per-file modification times, and other metadata" (x/mod/sumdb/dirhash).
- **The manifest other tools can read is a subject list inside a Statement**: in-toto Statement subjects "are matched purely by digest, regardless of content type", so a manifest shaped as `[{name, digest:{sha256}}]` can be wrapped in a Statement (with `_type` and a `predicateType`, both required) when interoperation is wanted; the bare list alone is not an in-toto artefact (in-toto Statement).
- **Archives verify on their contents**: Go hashes "the files within the module's `.zip` file", not the archive bytes; the `.tar.gz` question is answered by verify-on-extract (Go modules reference).
- **A root stored inside what it covers is recomputable by whoever edits it**; the witness rule from itd-16 applies, and "prove provenance" needs a signed manifest (`ssh-keygen -Y sign`, no dependency) or a root conveyed separately. Signing is out of scope by the record's own Out-of-Scope bullet, shared with itd-16, not by a ruling; the gap is recorded here (tlog-witness; ssh-keygen(1)).

**Path 2: a flat, sorted, digest-per-file manifest in the in-toto subject shape, rooted with itd-16's RFC 6962 primitive, verified on extracted files**; sections become a report grouping rather than a hash layer. That amends the tree shape In Scope names (artefact → section → root) and criterion 1's per-artefact entries under a sectioned root, a change to the record the fit-challenge takes up before the build. The independent fit-challenge runs at plan time.

## Open Questions

- **Default-on or opt-in?** Inherits from itd-16's substrate (recommend opt-in to avoid surprise verification failures during normal flow).
- **Interaction with itd-9 (schema migration)** — does migration regenerate hashes or carry them forward? Recommend re-anchor with original hash preserved for forensic comparison (same pattern as itd-16's chain re-anchoring).
- **`dev-sync` content** (which changes between disembarks) — hashed at sync time or only at disembark? Recommend disembark only for this intent (sync time hashing is itd-16's concern, since `dev-sync` writes to the conversation/edit chain).
- **Should `/abcd:audit lifeboat` work on a lifeboat directory only, or also on a packed `.tar.gz`?** Likely both, since lifeboats may travel as archives. Verify on extract.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- itd-16 (`/abcd:audit` umbrella substrate) — provides the chain primitives this intent layers on. Sibling intent.
- itd-9 (cross-version lifeboat schema migration) — coordinates with hash re-anchoring on migration.
- itd-32 (audit-role taxonomy, **superseded** by itd-31 on 2026-05-07) — the dissolved-bundle history that motivated the per-verb audit split.
- `~/.abcd/` v0 F-031 / F-032 hash-chain work — provides the JCS + UUIDv7 + Merkle pattern.
- README.md `/abcd:audit` row (canonical command shape).
- Original framing in itd-16 (2026-05-03 → 2026-05-08) was lifeboat-Merkle only; that scope was extracted here when the user split itd-16 into umbrella substrate (itd-16) and lifeboat application (this intent) on 2026-05-08.
