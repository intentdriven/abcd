---
id: itd-16
slug: hash-chain-merkle-audit
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-9]
severity: major
---

<!-- 2026-05-08: scope rewritten and split. /abcd:audit is positioned as the umbrella verb;
     this intent owns the substrate (chain mechanics + the conversation/edit-chain application).
     Lifeboat verification (the original itd-16 framing, 2026-05-03 → 2026-05-08) was extracted
     to itd-35 as a sibling sub-verb application. The umbrella/application split lets future
     audit applications (e.g., schema audit, prompt-quality audit) register as sub-verbs without
     re-litigating the substrate. -->

# `/abcd:audit` Is the Umbrella; Conversation/Edit Chain Is Its Default Application

## Press Release

> **abcd ships `/abcd:audit` — a compliance-grade audit umbrella with a tamper-evident hash chain over the project's conversation and edit history as its default application.** `/abcd:audit` is the *whole*; specific verifications register as sub-verbs that share the substrate. The default application — `/abcd:audit chain` — covers every Claude Code session transcript, every native spec state transition, every disembark/embark/launch/intent invocation, and every file edit committed under abcd's lifecycle hooks. JCS-canonicalised (RFC 8785) entries land in a Merkle chain at `.abcd/logbook/audit/chain-<ts>/`. Sibling sub-verbs (`/abcd:audit lifeboat <path>` per itd-35; future audit applications) reuse the substrate's primitives without re-implementing them. Bare `/abcd:audit` shows status, last-verified timestamp, chain head, and the registered sub-verbs (per the universal bare-command-as-help convention) — it never mutates state. The chain-specific verification report comes from `/abcd:audit chain`. Mismatches surface immediately with the offending entry's path, line, and expected vs observed hash.
>
> "When an auditor asked 'can you prove the project's history is intact?' we used to say 'check git'," said Dave, compliance lead. "git is great for code provenance but says nothing about the agent conversations and tool invocations that produced the code. `/abcd:audit` is the missing half: every conversation, every edit, hashed and chained. And because `/abcd:audit` is an umbrella, when we needed lifeboat integrity verification we got `/abcd:audit lifeboat` for free — same substrate, different target."

## Why This Matters

**`/abcd:audit` is the whole; specific verifications are applications.** This intent ships the *whole*: the umbrella verb, the substrate (JCS, UUIDv7, Merkle), the bare-verb dispatcher behaviour, and the default conversation/edit-chain application. Future audit needs (lifeboat integrity per itd-35, schema audit, prompt-quality audit) register as sub-verbs that share this substrate rather than re-implementing chain mechanics from scratch.

abcd captures a lot of project history — the native transcript store's transcripts, the native spec store's spec states, `.abcd/logbook/` per-command reports, the `.abcd/work/` shared working material and `.abcd/development/` curated durable record. That history is **legible** but not **tamper-evident**: anyone with filesystem access can edit a transcript or rewrite a logbook entry post-hoc, and a casual reader has no way to detect it. For most uses this is fine — abcd's audience is solo developers and small teams operating in good faith.

For compliance-sensitive contexts (regulated industries, third-party handoffs, post-incident audits, regulated AI governance), the **non-repudiability** of project history becomes a hard requirement. Auditors need to prove that what's in the repo today is what was written yesterday — that the conversations, the agent decisions, and the resulting edits haven't been silently rewritten.

The mechanism is well-established (lifted from `~/.abcd/` v0's F-031 / F-032 hash-chain work): JCS-canonicalised JSON for deterministic serialisation, UUIDv7 IDs for time-ordered entries, SHA-256 segment hashes chained via a Merkle tree to a single root hash. The trail covers four artefact classes:

1. **Conversation transcripts** (native transcript store) — every session as a discrete entry
2. **Lifecycle hook events** (intent moves drafts→planned→shipped, capture state changes, ahoy install/upgrade) — every state transition
3. **Tool invocations** (`/abcd:disembark`, `/abcd:embark`, `/abcd:launch`, `/abcd:intent plan/ship/review/consistency/shape`, `/abcd:capture`, `/abcd:grill`) — every command run
4. **File edits** committed under abcd's pre-commit hook — the diffs that actually changed the repo

abcd doesn't need to host a separate audit service: the chain lives in `.abcd/logbook/audit/chain-<ts>/`, ships with disembark, and verifies on embark.

## What's In Scope

- **Native hash-chain mechanics** — port the chain mechanics from `~/.abcd/scripts/hash-chain.py` patterns. JCS RFC 8785 canonicalisation; UUIDv7 RFC 9562 IDs; SHA-256 segment hashes; Merkle root anchoring.
- **Hook integration**: ahoy installs a pre-commit hook + a Stop hook that append entries to the active chain segment as transcripts close, lifecycle hooks fire, and edits commit. Append-only; no in-place rewrites.
- **Chain segmentation**: each segment covers one calendar day (configurable). Segment-to-segment chaining anchors yesterday's segment in today's, so the chain head transitively covers all history.
- **`/abcd:audit` (bare)** — status+help only per the universal bare-command-as-help convention. Shows chain head, last-verified timestamp, registered sub-verbs (`chain`, `lifeboat`, future), and suggested next actions. Never mutates state.
- **`/abcd:audit chain`** — the verification action: verifies the chain end-to-end and writes a report to `.abcd/logbook/audit/chain-<ts>/report.{json,md}`. This is the default application of the umbrella; itd-16 owns this sub-verb's contract.
- **`audit.schema.json`** — the public format for chain segments and the verification report.
- **Disembark integration**: `/abcd:disembark` includes the latest chain segments in the lifeboat under `_provenance.json` `audit_chain_root`. The chain root hash anchors the lifeboat's authenticity.
- **Embark integration**: `/abcd:embark` verifies the lifeboat's chain root before unpacking; mismatch aborts the unpack with the offending segment named.
- **Launch integration** (per itd-66): launch payload manifest excludes the chain by default (sensitive); explicit opt-in via `.abcd/launch.allow` if a release should ship its own chain.
- **Opt-in via `.abcd/config.json`** — `oracle.audit.chain.enabled = true | false` (default `false` to avoid surprising users; flip to `true` for compliance-sensitive projects).

## What's Out of Scope

- **Cryptographic signing** — chain entries are hashed but not signed. Signing is a separate concern (could become itd-XX if a use case emerges; e.g., signed releases).
- **Cross-project chains** — each project's chain is self-contained. No global root hash spanning multiple repos.
- **Hashing source repo state at disembark** — abcd hashes its own conversation/edit history, not arbitrary repo contents. `git` already provides the latter.
- **Lifeboat-artefact-only Merkle** (the original itd-16 framing, 2026-05-03 → 2026-05-08) — replaced by this broader conversation/edit chain. If a narrower lifeboat-only integrity proof is needed independent of the full chain, that's a separate intent.
- **Real-time chain verification on every command** — verification is on-demand (`/abcd:audit chain`) plus mandatory at embark. Continuous verification would add latency without proportional value.
- **Chain compression / pruning** — entries grow unboundedly with project activity. Pruning policy is a future concern if chains become unwieldy.
- **Specific audit applications other than chain and lifeboat** — schema audit, prompt-quality audit, third-party-vendor audit, etc., are deliberately not in scope here. The umbrella is open; specific applications register as their own intents (precedent: itd-35 for lifeboat).

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** `oracle.audit.chain.enabled = true`, **when** any abcd command runs (`/abcd:intent plan`, `/abcd:capture`, `/abcd:disembark`, etc.), **then** an entry is appended to today's chain segment with a UUIDv7 ID, JCS-canonicalised payload, SHA-256 hash, and a `prev_hash` field linking to the previous entry; the entry conforms to `audit.schema.json`.
- **Given** a project with chain segments spanning multiple days, **when** the user runs `/abcd:audit chain`, **then** abcd recomputes every entry's hash, walks the chain, and writes a verification report to `.abcd/logbook/audit/chain-<ts>/report.{json,md}`. Bare `/abcd:audit` (no sub-verb) shows the chain head hash, last-verified timestamp, registered sub-verbs, and any pending entries — status+help only, no state mutation, per the universal bare-command-as-help convention.
- **Given** a chain entry has been tampered with (e.g., a transcript edited post-hoc), **when** `/abcd:audit chain` runs, **then** verification fails on the offending entry, names the file and entry ID, prints expected vs observed hashes, and exits non-zero. No further entries are verified after the first mismatch (preserves the failing-fast invariant).
- **Given** a disembark with chain enabled, **when** the lifeboat is packed, **then** `_provenance.json` includes `audit_chain_root: <hash>` and the latest chain segments are included in the lifeboat under `audit/`. The Merkle root anchors the lifeboat to a verifiable point in project history.
- **Given** an embark with a lifeboat that includes a chain root, **when** `/abcd:embark from <path>` runs, **then** embark verifies the chain root matches the embedded segments BEFORE unpacking; a mismatch aborts the unpack with a clear message naming the divergent segment.
- **Given** chain enablement is opt-in (default `false`), **when** a user with `oracle.audit.chain.enabled = false` runs disembark, **then** no chain entries are written, `_provenance.json` omits `audit_chain_root`, and embark of the resulting lifeboat skips chain verification without warning — opt-in means opt-in cleanly on both ends.
- **Given** itd-9 (schema migration) and itd-16 (hash chain) both ship, **when** embark applies a schema migration to a chain entry, **then** the migrated entry's hash is RE-COMPUTED post-migration and recorded alongside the original hash in the migration log — schema upgrades are not silent over the chain.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **The named primitives are standards and fit**: RFC 8785 (JCS, Informational) sorts properties recursively and serialises numbers as ECMAScript does, for data that must be "expressed in an invariant format"; RFC 9562 (Proposed Standard) gives UUIDv7 "a time-ordered value field" and leaves counter rollover to the application (RFC 8785; RFC 9562).
- **The tree should be RFC 6962's**: leaf `SHA-256(0x00 || d)`, node `SHA-256(0x01 || l || r)`, "required to give second preimage resistance"; Go's `x/mod/sumdb/tlog` is compatible with those proofs and would be a Go dependency needing sign-off (RFC 6962; x/mod/sumdb/tlog).
- **The chain head has a standard text form**: a C2SP checkpoint, three lines (origin, tree size, base64 root) plus signature lines, the form the Go checksum database serves (tlog-checkpoint; Go modules reference).
- **Tamper-evidence needs a reader who holds an earlier head**: a witness cosigns only a checkpoint consistent with the last one it signed; without someone outside the filesystem, whoever edits a transcript can recompute the chain, so a chain alone detects a partial edit, not a rewrite. The press release's "non-repudiability" is not reachable without that anchor; the ruling stands and this records the gap (tlog-witness).
- **Signing needs no new dependency**: `ssh-keygen -Y sign` under a namespace, verified against an allowed-signers file; git's `gpg.format = ssh` uses the same (ssh-keygen(1); git gpg config). Signing is out of scope by the record; the cheapest anchor is nonetheless a signed head, the next cheapest a head pushed to the forge as a tag, which anchors only where the forge refuses to move or delete it (a tag ruleset or an immutable release) or a third party keeps a copy; otherwise whoever can push can rewrite the tag along with the chain.
- **The attestation stack exists and is more than this needs**: in-toto Statements in DSSE envelopes, SLSA provenance, Rekor; "still under development", and a public log for private transcripts runs against the launch rule (in-toto Statement; DSSE; Rekor).
- **What is hashed is the redacted transcript**, since the history store redacts on write; the claim must say so.

**Path 2: native, shaped to RFC 6962 hashing and the C2SP checkpoint, with the head's signature and external anchor as the first seam**; JCS and UUIDv7 as drafted. The RFC 6962 tree is the segment's Merkle root; criterion 1's per-entry `prev_hash` stays as drafted, so the tree adds inclusion and consistency proofs over a chain it does not replace, and no criterion changes. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) **Where the head is anchored**: a signed checkpoint in the local tier, a tag pushed to the forge (an anchor only where protected or witnessed), or nowhere (and then the claim is "partial edits are detected", not "history is provable").
- **Default on or opt-in?** Recommend opt-in (`enabled: false`) to avoid surprising users with verification failures during normal flow. Compliance-sensitive projects flip to `true` deliberately.
- **Segment granularity** — daily segments fit the "one Merkle root per day" pattern from `~/.abcd/` v0. Hourly might be too granular; weekly might miss the audit point. Configurable in `.abcd/config.json`?
- **Interaction with itd-9 schema migration** — does a migration regenerate the chain or carry it forward? Likely re-anchors the migrated entry while preserving the pre-migration hash for forensic comparison.
- **`dev-sync` content** (which changes between disembarks) — hashed at sync time or only at disembark? Likely sync time, so the chain reflects every dev-sync step.
- **Public-repo launch behaviour** — should `/abcd:launch` strip chain segments from the public payload (default-deny) or include them as ship-with-provenance (opt-in)? Recommend default-deny; users with compliance ship-with-provenance needs opt in via `.abcd/launch.allow`.
- **Verification cost** — chains grow unboundedly; large projects may need partial verification (`/abcd:audit --since <date>`). Out of scope for the initial ship; revisit if chains become unwieldy in practice.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- README.md `/abcd:audit` row (canonical command shape: "Compliance-grade trail: tamper-evident hash-chain over the project's conversations and edits, so anyone can verify nothing was retroactively altered").
- `02-constraints/04-naming.md` — `/abcd:audit` reserved for formal verification surface; not metaphor-mapped, dignified register.
- itd-9 (cross-version lifeboat schema migration) — coordinates with chain re-anchoring on migration.
- `~/.abcd/` v0 F-031 / F-032 hash-chain work — provides the JCS + UUIDv7 + Merkle pattern.
- Prior framing: this intent originally scoped to lifeboat-artefact integrity (2026-05-03 → 2026-05-08). Rewritten 2026-05-08 to align with README's canonical conversation/edit framing. The narrower lifeboat-only integrity proof may return as a separate intent if the use case re-surfaces independently.
