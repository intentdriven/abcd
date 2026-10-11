# abcd ADRs

Architecture Decision Records — retrospective records of settled decisions, their context, alternatives rejected, and consequences.

---

## What's an ADR?

An **ADR** captures a *settled decision* — written after the decision is made, recording why it was made and what was rejected. ADRs exist to keep decisions intelligible to future readers (and future selves) who weren't in the room when the decision happened.

ADRs are used when **all three** are true:

1. **Hard to reverse** — the cost of changing the decision later is meaningful.
2. **Surprising without context** — a future reader will wonder "why this way?"
3. **Result of a real trade-off** — there were genuine alternatives and one was picked for specific reasons.

If any of the three is missing, skip the ADR. File-scoped rationale (why this section reads this way) lives inline in the brief; project-scoped framework decisions earn an ADR.

ADRs are *not* used for:

- Forward-looking discussion — those are RFCs (`../../roadmap/rfcs/`).
- User-facing capability — those are intents (`../../intents/`).
- Bug fixes, refactors, content edits — git log + commit messages cover deltas.

---

## ADR IDs

ADR IDs follow the pattern `adr-N` (unpadded, mirrors `itd-N` / `rfc-N`) as the prose handle, and the filename is `<N>-<slug>.md` — the number the handle carries, then the slug ([ADR-30](0030-record-information-architecture.md)).

**An ADR is minted by the binary, not numbered by hand:** `abcd decide "<title>"` draws `adr-<yymmddHHMMSS><rrrr>` through the same record-id seam captures, intents and specs mint through ([ADR-45](0045-record-ids-are-timestamp-numeric-and.md), and the ruling of 2026-09-01 in [`.abcd/work/DECISIONS.md`](../../../work/DECISIONS.md) that this is the family ADR-45's rollout note 3 deferred). The mint reads no maximum anywhere, which is the property being bought: a hand-allocated ordinal is read off the directory, so two branches deciding on the same day allocate the same number — `0055` and `0056` were each minted twice, for different decisions, on the day the ruling was taken.

**`0001`–`0058` keep their IDs and their filenames.** Nothing is renumbered. Those records carry the zero-padded four-digit ordinal (`adr-7` lives in `0007-<slug>.md`); a minted record carries the sixteen-digit stamp (`adr-2609021016286571` lives in `2609021016286571-<slug>.md`). Every reader of an ADR ID admits both vintages through one derivation — the citation resolver, the `abcd <record-id>` dispatch, the `record_schema` and citation-currency gates, the website's decisions index, and the lifeboat packer. Every ordinal is shorter and numerically smaller than every stamp, so the hand-numbered records sort first in both the directory listing and the derived index order.

IDs are capture-stable. Once assigned, an ADR's ID never changes — superseding ADRs use new IDs and link backwards.

---

## Lifecycle (Status Field)

| Status | Meaning |
|---|---|
| `proposed` | Draft; the decision is not yet locked. Rare — most ADRs are written after the fact. |
| `accepted` | The decision is in force. Default for retrospective ADRs. |
| `superseded` | Replaced by a newer record. The successor's `supersedes` and this ADR's `superseded_by` both name the pair, and one of two things follows: the ADR is pruned (its text is fully carried by the successor, and the `supersedes` declaration is the trace the record keeps), or it is retained because later records still cite it. |
| `deprecated` | The decision no longer applies but no successor replaces it (the surface itself was removed). |

Transitions are deliberate. Accepted ADRs are retained in the record. A superseded ADR is pruned once its successor lands and carries the transition rationale — git history preserves the original text — unless later records still cite it, in which case it is retained and both halves of the supersession stay in its frontmatter.

---

## Format

`abcd decide "<title>"` writes this shape — the frontmatter, the H1, and the four sections, each carrying the question it answers. Every ADR has frontmatter (machine-readable) plus a Markdown body following this structure:

```markdown
---
id: adr-N
slug: <kebab-case-slug>
status: accepted                 # proposed | accepted | superseded | deprecated
date: YYYY-MM-DD
supersedes: null                 # [adr-N, itd-N, ...] this record replaces
superseded_by: null              # the record that replaced this one (adr-N or itd-N)
related_intents: []              # [itd-N, ...] cross-references
related_rfcs: []                 # [rfc-N, ...] cross-references
related_adrs: []                 # [adr-N, ...] sibling decisions
---

# ADR-N: <Title — short noun phrase, the decision in one line>

## Context

What forced the decision? What was the world look like before? What constraints
were already locked?

## Decision

What did we decide? Stated as a positive declaration: "We will X."

## Alternatives Considered

2–4 options laid out fairly, including the chosen one. For each: what it would
have looked like, why it was rejected (or chosen).

## Consequences

What follows from the decision — both gains and costs. Honest about trade-offs.
What's now easier; what's now harder; what new obligations the decision creates
(lint rules, vocabulary terms, audit gates).
```

---

## Bidirectional Linking

| File | Frontmatter field |
|---|---|
| `adrs/<N>-<slug>.md` | `related_intents: [itd-N, ...]` (intents whose framework this ADR justifies) |
| `adrs/<N>-<slug>.md` | `related_rfcs: [rfc-N, ...]` (RFCs that informed this decision) |
| `adrs/<N>-<slug>.md` | `supersedes: <handle>` / `superseded_by: <handle>` (chain) — **both directions are required**: if A declares `superseded_by: B`, B declares `supersedes: A`. A supersession may cross stores (an ADR that redecides the question an intent rested on retires that intent), so a handle here is `adr-N` or `itd-N`. `record_schema` enforces the pair. |
| `intents/{drafts,planned,shipped,disciplines}/itd-N-<slug>.md` | `related_adrs: [adr-N, ...]` (when an intent references an ADR) |
| `rfcs/rfc-N-<slug>.md` | `related_adrs: [adr-N, ...]` (when an RFC references an ADR or its resolution becomes one) |

The intent lint (a Go implementation) extends to verify these reciprocally.

---

## Index

> **Index maintenance:** `abcd decide` mints the ID and materialises the ADR
> file, but appending the row to this index table is a manual edit; add the row
> by hand when an ADR is captured.

| ID | Title | Status | Date |
|---|---|---|---|
| [adr-1](0001-three-layer-mental-model.md) | Three-layer mental model (brief / intent / spec) | accepted | 2026-05-04 |
| [adr-2](0002-three-intent-kinds.md) | Three intent kinds (standalone / bundle-member / discipline) | accepted | 2026-05-07 |
| [adr-3](0003-directory-as-truth-for-lifecycle.md) | Directory location is the source of truth for lifecycle state | accepted | 2026-05-07 |
| [adr-5](0005-brief-is-current-state.md) | Brief is the current state; no version label, no archive directory | accepted | 2026-05-08 |
| [adr-7](0007-grill-skill-and-glossary.md) | `/abcd:intent grill` — one sub-verb with two inseparable phases; cite-or-fail lint; bounded-context glossary structure | accepted | 2026-05-11 |
| [adr-9](0009-phase-as-product-layer.md) | Phase as a product-reflection layer between brief and intent; replaces plugin-version language (superseded by adr-2609212115255771, which retires phases and milestones) | superseded | 2026-05-16 |
| [adr-10](0010-phase-negotiator-grounded-tradeoffs.md) | The phase negotiator — a Socratic agent that proposes phases and grounds every trade-off in the DAG / phase acceptance | accepted | 2026-05-16 |
| [adr-11](0011-spec-terminology-rename.md) | One canonical word for a specced block of work — spec | accepted | 2026-05-18 |
| [adr-12](0012-issue-ledger-live-vs-structured.md) | `.work/issues.md` (historical) stays the live operational ledger; structured `iss-*` store deferred until the native spec layer schedules the migration (superseded by adr-32, which redecides where the ledger lives) | superseded | 2026-06-06 |
| [adr-13](0013-fn38-memory-single-writer-and-write-lint.md) | Durable memory writes — single-writer, atomic-rename crash model | accepted | 2026-06-09 |
| [adr-19](0019-plugin-json-version-carve-out.md) | The plugin version lives only in the released artifact; the working tree stays unversioned, and the version location is chosen by a schema-validated decision artifact, not hard-coded | accepted | 2026-07-01 |
| [adr-20](0020-manifest-version-lockstep.md) | The two release manifests stay version-consistent via a read-only anti-drift checker over a pinned per-view path list; the source view stays unversioned; `--allow-dirty` must never bypass manifest consistency (wiring policy); the marketplace changelog entry gets a committed schema | accepted | 2026-07-03 |
| [adr-21](0021-rebuild-in-go.md) | Rebuild abcd as a Go binary | accepted | 2026-07-06 |
| [adr-22](0022-bundled-deps-as-pluggable-adapters.md) | Bundled dependencies become pluggable adapters over a native default (supersedes adr-14, adr-15, adr-17) | accepted | 2026-07-06 |
| [adr-23](0023-transport-agnostic-core.md) | A transport-agnostic Go core behind thin front doors | accepted | 2026-07-06 |
| [adr-24](0024-companion-harness-peer-via-conventions.md) | the companion harness is a peer integrated via conventions and MCP, not a code dependency | accepted | 2026-07-06 |
| [adr-25](0025-host-delegated-llm-default.md) | The LLM is host-delegated by default; oracles are opt-in adapters (supersedes adr-8) | accepted | 2026-07-06 |
| [adr-26](0026-native-spec-layer-ccpm-backend.md) | A native minimal spec layer with the companion harness `ccpm` as the primary deeper backend | accepted | 2026-07-06 |
| [adr-27](0027-autonomous-run-pluggable-seam.md) | The autonomous run is a pluggable seam, not a Ralph port (supersedes adr-16) | accepted | 2026-07-06 |
| [adr-28](0028-single-repo-curated-release.md) | One repository, a curated release artifact — no dev→public mirror (supersedes adr-18) | accepted | 2026-07-06 |
| [adr-29](0029-native-transcript-corpus.md) | A native local redacted transcript corpus (superseded by adr-2609090717039680, which relocates the store and makes it create itself) | superseded | 2026-07-06 |
| [adr-30](0030-record-information-architecture.md) | Design-record information architecture — flat artefact-type folders | accepted | 2026-07-06 |
| [adr-31](0031-derived-versioning-from-intents.md) | The release version is derived from the intents in it, never authored (extends adr-19, adr-20) | accepted | 2026-07-07 |
| [adr-32](0032-issue-ledger-is-working-tier-data.md) | The issue ledger is working-tier data, not authored record — move to `.abcd/work/issues/`, drop git-inferable timestamps, derive priority (supersedes adr-12) | accepted | 2026-07-08 |
| [adr-33](0033-launch-phase-ownership-tiered.md) | Launch phase ownership is tiered — Phase 1 owns the curated-release cut; deepenings are separately scheduled intents; the phase index is the sole ownership source | accepted | 2026-07-08 |
| [adr-34](0034-lifecycle-and-scheduling-orthogonal.md) | Intent lifecycle and phase scheduling are orthogonal axes — scheduled ⇒ committed (`planned/`), but planned intents may be unscheduled | accepted | 2026-07-08 |
| [adr-35](0035-lifeboat-as-coverage-experiment.md) | The lifeboat is a coverage experiment — read-only, out-of-tree, and proven before it is packed (supersedes adr-4) | accepted | 2026-07-14 |
| [adr-36](0036-coverage-blanks-are-a-fillable-lifecycle.md) | Coverage blanks are a fillable lifecycle — authored is not extracted, and the interview is its own step | accepted | 2026-07-15 |
| [adr-37](0037-changelog-driven-releases.md) | Releases are changelog-driven — rolling `[Unreleased]` is the release decision, and automation tags exactly that commit | accepted | 2026-07-17 |
| [adr-38](0038-implicit-checks-are-disk-only.md) | Implicit checks are disk-only — the network answers only an explicit ask | accepted | 2026-08-15 |
| [adr-39](0039-host-tier-policy.md) | Host-tier policy — MCP floor, reference host, open-source default | accepted | 2026-08-16 |
| [adr-40](0040-review-audit-lint-are-three-verbs.md) | Review, audit, lint, and gate are four buckets, separated by what each compares | accepted | 2026-08-16 |
| [adr-41](0041-corpus-trust-boundary.md) | Documents and ledgers never leave the user tier; a public citation requires both gates | proposed | 2026-08-16 |
| [adr-42](0042-guard-parse-layer-is-a-mistake-filter.md) | The guard's parse layer is a mistake filter, not a security boundary — matching is two-tier, and completeness is abandoned as a goal | accepted | 2026-08-18 |
| [adr-43](0043-inbound-equals-outbound-and-the-org-role.md) | Contributions are inbound = outbound MIT, and the trust boundary is the organisation's role ladder | accepted | 2026-08-19 |
| [adr-44](0044-remote-mutation-and-caller-identity.md) | abcd never mutates a remote uninvited, and identity derives from caller-local facts | accepted | 2026-08-19 |
| [adr-45](0045-record-ids-are-timestamp-numeric-and.md) | Record ids are timestamp-numeric, collision-proof by construction, and capture-stable | accepted | 2026-08-20 |
| [adr-46](0046-persistence-never-weakens-the.md) | Persisting the hook binary never weakens the verification posture — every promotion re-verifies, and SessionEnd performs no network work (superseded by adr-2609151706587280, which carries decisions 1–5 forward and binds the cache-to-PATH promotion to a home-scoped attestation; retained because later records cite its numbered decisions) | superseded | 2026-08-21 |
| [adr-47](0047-abcdev-app-rendered-from-this-repository.md) | abcdev.app is rendered from this repository alone | accepted | 2026-08-22 |
| [adr-48](0048-website-deploys-on-release-not-on-merge.md) | The website deploys on release, not on merge | accepted | 2026-08-22 |
| [adr-49](0049-terminal-emission-discipline.md) | Terminal emission discipline — decoration only on interactive TTYs, machine streams undecorated, untrusted text always sanitised | accepted | 2026-08-22 |
| [adr-50](0050-framing-traces-never-enter-the-record.md) | Framing traces never enter the record, and automated reviewers never read them | accepted | 2026-08-22 |
| [adr-51](0051-intents-declare-mechanism-and-scope.md) | An intent can declare its mechanism claim and its scope conditions — optional sections, enforcement deferred | accepted | 2026-08-22 |
| [adr-52](0052-the-semantic-gate-sits-on-the-wrong-side.md) | The semantic release gate runs after tagging, so a refusal consumes the version rather than blocking it — problem and options recorded, no decision | proposed | 2026-08-23 |
| [adr-53](0053-audience-by-placement.md) | Audience is expressed by placement, never by a prose register | accepted | 2026-08-28 |
| [adr-54](0054-punctuation-enforcement-stays-mechanical.md) | Punctuation enforcement stays mechanical-only (supersedes itd-141) | accepted | 2026-08-28 |
| [adr-55](0055-the-construal-stands-in-the-record-its.md) | The construal stands in the record; its history does not — refines adr-50 | accepted | 2026-08-28 |
| [adr-56](0056-an-exclusion-control-asserts-only-what.md) | An exclusion control asserts only what it can prove | accepted | 2026-08-30 |
| [adr-57](0057-grounds-accumulate-as-an-append-only.md) | Grounds accumulate as an append-only section, in one shared form | accepted | 2026-08-30 |
| [adr-58](0058-a-reading-is-commissioned-about.md) | A reading is commissioned about something, so the invocation takes a scope (superseded by adr-2609021016286571, which makes the invocation a position and a target state) | superseded | 2026-08-31 |
| [adr-2609021016270132](2609021016270132-the-principles-family-is-a-declared.md) | The principles family is a declared record store whose entries carry typed claims, and it is read cold by statement and never by citation | accepted | 2026-09-02 |
| [adr-2609021016272867](2609021016272867-the-comparative-reading-receives-one.md) | The comparative reading receives one widening run's candidates, a positional exception to the prior-run exhaust, with the run derived from the record and no operand added | accepted | 2026-09-02 |
| [adr-2609021016275803](2609021016275803-no-session-holds-both-a-reading-and-the.md) | No session holds both a reading and the ledger, and a per-run context stamp is how the record shows it | accepted | 2026-09-02 |
| [adr-2609021016286571](2609021016286571-the-invocation-is-a-position-and-a.md) | The invocation is a position and a target state, and the committed preset for the position supplies the rest (supersedes adr-58) | accepted | 2026-09-02 |
| [adr-2609021016288378](2609021016288378-a-reframe-occasioned-by-a-reading-is-a.md) | A reframe occasioned by a reading is a committed pointer to an event whose content stays local, and the frame it fingerprints is the framing as it presently stands, which adr-55 enumerates as three committed surfaces | accepted | 2026-09-02 |
| [adr-2609090636172016](2609090636172016-sub-agent-transcript-lineage-is-carried.md) | Sub-agent transcript lineage is carried by explicit record fields, not a composite session id | accepted | 2026-09-09 |
| [adr-2609090717039680](2609090717039680-the-transcript-corpus-is-a-sibling-store.md) | The transcript corpus is a sibling store that creates itself, and the per-repo location is an opt-in pull (supersedes adr-29; superseded by adr-2609091248201071, which names the canonical directory primitive the store calls) | superseded | 2026-09-09 |
| [adr-2609091014087993](2609091014087993-a-tool-never-creates-directories-in-user.md) | A tool never creates directories in user-owned project space; agent and session scratch is machine-scoped (superseded by adr-2609091248200336, which states the split: the store's location binds now, its verbs bind when the store ships) | superseded | 2026-09-09 |
| [adr-2609091248200336](2609091248200336-a-tool-never-creates-directories-in-user.md) | A tool never creates directories in user-owned project space; the store's location binds now and its verbs bind when the store ships (supersedes adr-2609091014087993; superseded by adr-2610031751065746, which spells the store under the renamed home) | superseded | 2026-09-09 |
| [adr-2609091248201071](2609091248201071-the-transcript-corpus-is-a-sibling-store.md) | The transcript corpus is a sibling store that creates itself through the canonical directory primitive (supersedes adr-2609090717039680; superseded by adr-2610031751066232, which spells the store under the renamed home) | superseded | 2026-09-09 |
| [adr-2609151138420062](2609151138420062-the-decisions-log-is-a-folder-of-minted.md) | The decisions log is a folder of minted records, not one appended file | accepted | 2026-09-15 |
| [adr-2609151513118583](2609151513118583-an-intent-owns-one-or-more-specs-and-it.md) | An intent owns one or more specs, and it ships when its last spec closes | accepted | 2026-09-15 |
| [adr-2609151528057131](2609151528057131-abcd-owns-the-product-thinker-s-surface.md) | abcd owns the product thinker's surface | accepted | 2026-08-29 |
| [adr-2609151528057260](2609151528057260-three-roles-who-each-artefact-addresses.md) | Three roles, who each artefact addresses, and when the loop stops | accepted | 2026-08-29 |
| [adr-2609151706587280](2609151706587280-the-cache-reaches-path-only-through-a.md) | The cache reaches PATH only through a home-scoped attestation the environment cannot write (supersedes adr-46; GHSA-4q78-ccfv-f374, option B as ruled 2026-09-15) | accepted | 2026-09-15 |
| [adr-2609212115255771](2609212115255771-phases-and-milestones-are-retired.md) | Phases and milestones are retired: sequencing is dependencies rendered as status, the checkpoint is the derived release (supersedes adr-9; superseded by adr-2609292012006845, which lists an intent in a lane under Now only and keeps phases retired) | superseded | 2026-09-21 |
| [adr-2609221009491186](2609221009491186-a-provider-adapter-serves-only-the.md) | A provider adapter serves only the models it lists, under a vendor denylist no listing overrides; everything else runs on the host (superseded by adr-2609300107513982, which retires the bundled vendor denylist and carries decisions 1, 3, 4 and 5 forward) | superseded | 2026-09-22 |
| [adr-2609221017021499](2609221017021499-every-external-credential-abcd-holds-is.md) | Every external credential abcd holds is named in configuration and kept in one store the person chose; never the harness, never the repository | accepted | 2026-09-22 |
| [adr-2609231028044006](2609231028044006-surface-chapter-shape-claims-are-derived.md) | Shape claims in the brief's surface chapters are derived, never hand-authored — a generated appendix per chapter, with the `## Sub-verbs` table the one hand-written exception | accepted | 2026-09-23 |
| [adr-2609231048308186](2609231048308186-the-catalog-pins-the-latest-release-s.md) | The catalog pins the latest release's plugin archive by address and digest, where the release declares it publishes one (amends adr-19 and adr-20; ruling E1 of 2026-09-23) | accepted | 2026-09-23 |
| [adr-2609291342092738](2609291342092738-a-drain-takes-an-issue-alone-only-when.md) | A drain takes an issue alone only when its fields say it needs no decision | accepted | 2026-09-29 |
| [adr-2609292012006845](2609292012006845-now-next-and-later-list-an-intent-in-a.md) | Now, Next and Later list an intent in a lane under Now only; phases stay retired (supersedes adr-2609212115255771) | accepted | 2026-09-29 |
| [adr-2609292116133348](2609292116133348-a-dependency-bump-inside-the-bound-is-re.md) | A dependency bump inside the bound is re-authored as the owner and only pushed by an App | accepted | 2026-09-29 |
| [adr-2609300107513982](2609300107513982-a-provider-adapter-s-allowlist-alone.md) | A provider adapter's allowlist alone decides which models it serves, and abcd bundles no vendor denylist (supersedes adr-2609221009491186; ruling H9 of 2026-09-29) | accepted | 2026-09-30 |
| [adr-2609300821558671](2609300821558671-a-reading-finding-is-matched-against-the.md) | A reading finding is matched against the record when it is stored, reversing itd-180's warm-work-only rule | accepted | 2026-09-30 |
| [adr-2609301720596683](2609301720596683-abcd-adds-a-missing-site-label-to-an.md) | abcd adds a missing site label to an existing ui.json and never rewrites one | accepted | 2026-09-30 |
| [adr-2610030720195401](2610030720195401-abcd-keeps-its-own-folders-out-of.md) | abcd keeps its own folders out of desktop indexing only by changing those folders, never the computer's search settings | accepted | 2026-10-03 |
| [adr-2610030814023326](2610030814023326-agents-md-is-the-one-conventions-file.md) | AGENTS.md is the one conventions file abcd writes; it never writes a host-specific copy | proposed | 2026-10-03 |
| [adr-2610031751065746](2610031751065746-the-worktree-store-lives-under-the.md) | The worktree store lives under the renamed home, `~/.abcd.noindex/worktrees/<root-sha>/<name>/`, and the rest of its rule is unchanged (supersedes adr-2609091248200336) | accepted | 2026-10-03 |
| [adr-2610031751066232](2610031751066232-the-transcript-store-is-spelled-under.md) | The transcript store is spelled under the renamed home, `~/.abcd.noindex/transcripts/<root-sha>/`, and the rest of its decision is unchanged (supersedes adr-2609091248201071) | accepted | 2026-10-03 |
| [adr-2610050711287503](2610050711287503-abcd-reaches-the-harness-s-user-settings.md) | abcd reaches the harness's user settings only for the status line, with consent; hooks live only in the plugin | accepted | 2026-10-05 |
