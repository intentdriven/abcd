# Naming Convention

A person who has typed one abcd verb should be able to guess the next. That is
what the naming rules below buy, and it is the only reason they exist: a
consistent namespace is a smaller thing to learn, and a bare verb that renders
state means nobody has to remember a sub-command to find out where they stand.

Commands and abcd-owned directories use ship/voyage metaphors where a maritime
word teaches something. Where none does, the surface is exempt and says so on the
record, so an exemption is a decision rather than an omission.

| Path / command | Meaning |
|---|---|
| `/abcd:ahoy` | hail a project → install abcd into it, or report what is installed. The bare form and the read-only sub-verbs report; `install`, `uninstall` and `remote apply` are the write paths. See [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md) |
| `/abcd:disembark` | leave the ship → pack a lifeboat for the journey |
| `/abcd:embark` | board a new ship → unpack the lifeboat |
| `/abcd:launch` | put the (cleaned) ship to sea publicly |
| `/abcd:dredge` | cross-corpus synthesis: surface latent patterns from accumulated captures (itd-25, a later phase). Maritime: dredging the seabed for what has settled. Pairs with `lifeboat` (per-project rescue) as the cross-corpus counterpart |
| `lifeboat` | the portable artefact (rescue from a sinking project). Written to an **operator-chosen destination**, never back into the source repo, per [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md). The in-tree `.abcd/lifeboat/` home is superseded |
| `~/.abcd/voyage/` | record of voyages: the operator-level, per-source-root operations namespace, keyed on the root-commit SHA and never committed. See the `voyage/` row in the reserved-vocabulary table below |

**Sense disambiguation:** `/abcd:launch` uses the *nautical* sense (a ship's first
entry to water, the public maiden voyage of a cleaned repo), not the generic
software sense of "run a program".

## What stays outside the metaphor

Reach for a metaphor only when it teaches. The criterion is a natural maritime
cognate that adds meaning — `dredge` literally raises settled material, `loot`
carries a licence-check reflex — and everything else stays exempt, because a
stretched metaphor obscures the verb it names.

The exemptions carrying a rationale of their own:

- `/abcd:intent`: product-framing surface. "Intent" does semantic work the brief
  depends on, and no maritime word carries that meaning.
- `/abcd:capture`: issue-capture surface (itd-4). "Capture" is deliberately
  neutral so the verb does not pre-commit to whether a finding is a bug, a
  nitpick, or a systemic pattern.
- `grill`: Socratic-questioning register, borrowed from prior art
  ([Pocock skills](https://github.com/mattpocock/skills)), signalling adversarial
  interrogation directly. It ships as the second leg of `/abcd:ideate`'s
  admission gauntlet, not as a sub-verb of any command: `/abcd:intent grill` is
  **staged** (itd-27), and no `intent grill` sub-verb is registered.
- `/abcd:audit`: formal verification surface, **staged** (itd-16). Reserved, not
  metaphor-mapped, dignified register.
- `/abcd:reflect`: phase-retrospective surface, **staged** (itd-24). Not
  metaphor-mapped, soft register.
- `/abcd` (bare, top-level): where-am-i status board (itd-20). The namespace root
  refuses any positional that is not a record id, so `status` is not a registered
  command; `abcd help` prints the root help, a different render from the bare
  board.

The remaining surfaces are exempt for the plain reason that no maritime cognate
adds meaning: `banlist`, `changelog`, `consult`, `decide`, `docs`, `guard`,
`history`, `ideate`, `identity`, `ingest`, `lint`, `memory`,
`prepare-this-repo`, `reading`, `rules`, `site`, `spec`, `update`, and
`version`. They are registered here so the exemption is on the record.

**Reserved meta-development commands** (later phases; named now to prevent
collisions):

> **Note:** `/abcd:audit` appears both here and in the exemptions above. The two
> listings encode two distinct contracts: the exemptions say the verb is exempt
> from the maritime convention; this table says it is reserved for a later-phase
> intent. Both are true, so both are kept.

| Path / command | Meaning |
|---|---|
| `/abcd:dredge` | cross-corpus synthesis (itd-25, a later phase). Maritime: dredging the seabed. Pairs with `lifeboat` as the cross-corpus counterpart to per-project rescue |
| `/abcd:loot` | OSS-vendor-with-provenance: clone selected files from public repos, recording origin, licence, SHA and rationale (itd-26, a later phase). Maritime: raid the open ocean for outside cargo. The pirate connotation is feature rather than bug — the verb itself prompts a licence-check reflex |
| `/abcd:audit` | formal verification surface: hash-chain and Merkle audit trails, fidelity checks (itd-16, a later phase). Reserved, not metaphor-mapped, dignified register |

Technical files (`config.json`, `rules.json`, and `corpus.json`, which is reserved
here as a name and is not yet in the tree) are exempt — no metaphor needed.

**Retired maritime names.** `.abcd/logbook/` was the maritime name for per-run
logs, state and reports. It is retired and must not be re-minted: run output goes
to the local ephemeral tier, and the operator-level voyage record to
`~/.abcd/voyage/<source-root-sha>/`. `TestNoRetiredLogbookLocationInSource` in
`internal/adapter/scanner` fails the build if any Go source names the retired
location (iss-73).

## Bare invocation renders, sub-verbs earn their place

Every `/abcd:<verb>` treats the bare invocation as status plus help plus a render
of that namespace's current state. A sub-verb earns its existence by doing
something the bare invocation cannot: mutating state, taking a positional
argument, scoping to a different time-axis or granularity, or performing an
action distinct from rendering.

That is what gives abcd its discoverability — type the verb, see where you stand.
A sub-verb that just renames "show me the state" obscures it instead, so
`<verb> show`, `<verb> stats`, `<verb> view` and a plain unfiltered `<verb> list`
are **forbidden** at design time rather than argued about in review. Lint code
`SD001` is reserved for the check; no implementation exists, so the discipline is
held by review.

**Conformance is partial, and the gap is the shipped surface's rather than the
discipline's.** Several parents print usage with no state, two verbs refuse
instead of rendering, and `history` breaks the rule from both ends at once: bare
`abcd history` renders nothing while `history list` and `history show` exist,
which is exactly the shape the rule forbids. The one enumeration of where the
convention holds and where it does not lives in
[`../04-surfaces/README.md`](../04-surfaces/README.md#bare-invocation); the
per-verb inventory of registered sub-verbs is the machine-checked `## Sub-verbs`
table in each surface chapter, so neither is restated here.

Sub-verbs that are **staged**, named to hold the shape: `/abcd:audit chain` and
`/abcd:audit lifeboat` (itd-16); `/abcd:intent grill <itd-N>` (itd-27); and
`/abcd:oracle ask <prompt>` — `oracle` is the model-access seam per adr-25, and
no `oracle` verb is registered.

**Brief-is-current-state discipline** (per
[adr-5](../../decisions/adrs/0005-brief-is-current-state.md)): the brief reflects
the project's *current* state. No version label on the brief, no `archive/`
directory inside it, no version-changelog blobs in its README. History lives in
`git log`; inflection-point rationale lives in
[`../../decisions/adrs/`](../../decisions/adrs); forensic snapshots come from
`/abcd:disembark`.

## Vocabulary-registration requirement (HARD from the start)

Every term introduced in a spec's `## Modification Grammar > Ripple > Vocabulary
delta` sub-bullet (per itd-37) MUST be registered in the same spec, in whichever
of the two registries below fits it. Lint code `VR001` is reserved for the check;
no implementation exists, so registration is enforced by review.

**Why hard from the start, not soft.** A discipline that ships as "soft
initially, hard once stable" is structurally weaker than itd-1 and itd-5, both of
which ship hard from day one. Hard enforcement costs about thirty seconds per new
term. Soft enforcement costs compounding vocabulary drift, found post-hoc by the
cross-document fidelity reviewer instead of blocked at design time.

**Which registry.** The project keeps two, holding different kinds of thing.

- **The glossary** at [`../glossary/`](../glossary) is the one canonical
  glossary and the only place a glossary lives. It holds cross-cutting
  natural-language vocabulary, one file per term per bounded context, each
  declaring its aliases and forbidden synonyms, and it is what the `GL002`
  forbidden-synonym rule reads. A term naming a concept the record uses in prose
  is registered there.
- **This file** is the naming-convention and reserved-vocabulary register: the
  maritime table, the exemptions, and the reserved-vocabulary table below. These
  are closed vocabularies tied to a named spec rather than glossary terms; this
  file is not a glossary and holds no term files.

The Role 2 cross-document audit verifies registration on every plan-review.

**Reserved vocabulary** (controlled enums, PR-to-extend).

Two things to read the table with:

- **A row is a reserved name, not a delivery claim.** Where the machinery a row describes is a design target rather than shipped behaviour, the row says **(staged)** in its Type column, per the truth rule in [`../00-meta.md`](../00-meta.md#the-truth-rule). Current delivery state lives in the roadmap dashboard, never here.
- **A row the shipped design has superseded says so.** Some names entered this register from the retired predecessor implementation and describe machinery abcd neither ships nor plans; they stay registered so the name is not re-minted on a different meaning, marked **(predecessor vocabulary; superseded)** with a pointer to what replaced it.
- **Source ids carry their namespace.** A `spc-N` in the Source column that names the retired predecessor store is written *(predecessor store)*, per the two-namespace rule in [`../../specs/README.md`](../../specs/README.md#two-spc-n-namespaces--always-qualify-above-the-ceiling). The two id spaces overlap below the live ceiling, so an unqualified `spc-2` or `spc-3` resolves in the live store to a spec about something else entirely.

| Term | Type | Source |
|---|---|---|
| `phase retrospective` | **(staged)** The five-section README (`went well` / `could improve` / `lessons learned` / `decisions made` / `metrics`) written by `/abcd:reflect <phase-id>` to `.abcd/retrospectives/<phase-id>/README.md`. Phase-grained only (the intent form was dropped per the itd-24 grill). Composed by the `reflection-composer` agent from the spc-66 (predecessor store) phase-audit receipt; rendered/written by the deterministic reflect writer. Links to the phase doc + audit report + member specs only (SSOT — no body duplication). | spc-83 (predecessor store) + `spc-83-operator-surfaces-manifest-lockstep.3` (itd-24) |
| `reflection-composer` | **(staged)** The 16th catalog agent: composes phase-retrospective prose from a seeded single-pass interview grounded in the spc-66 (predecessor store) phase-audit receipt's per-bullet acceptance verdicts. Dispatched by `/abcd:reflect`. `capability_scope.task_classes: [surface_render]`. | spc-83 (predecessor store) + `spc-83-operator-surfaces-manifest-lockstep.3` (itd-24) |
| `setup-wizard` | **(staged)** The display-only surface (in the Go binary, `internal/core/...`) that explains a missing external dependency when the spc-76 (predecessor store) validation gate fails closed: four fixed-order elements (tool name + version floor / requiring capability / what fails without it / exact install step), sourced from the gate's typed `MissingToolPayload` (single source) with a curated blurb registry for prose only. NEVER weakens the gate — declining stays fail-closed and the decline is recorded to the local ephemeral run-output tier, never to the retired `logbook` name. NOT a top-level command in v1 (rendered through the gate CLI + a standalone `explain` entrypoint). | spc-83 (predecessor store) + `spc-83-operator-surfaces-manifest-lockstep.4` (itd-63) |
| `JSON sidecar` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** The canonical `review.json` file written into each per-review directory in the review store. Consumers MUST read the JSON sidecar; the rendered `.md` is derived. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.1` (the cited `docs/reference/review-schema.md` schema page does not exist in this repository) |
| `MD render` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** The derived `review.md` file rendered mechanically from the JSON sidecar (front-matter from metadata, prose from `body_markdown`, "## Findings" from `findings[]`). Not canonical; consumers read the JSON sidecar. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.1` |
| `write-time sanitiser` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** The Stage 1 sanitiser applied to review body text before writing the JSON sidecar and rendered MD. Strips absolute paths, API keys, and PII patterns. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.3` |
| `Stage 1` (write-time sanitiser) | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** Write-time mutation pass: applied before writing `body_markdown` and before hashing the raw artifact. Strips absolute paths and secret/PII patterns. The only component that mutates content. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.3` |
| `Stage 2` (detect-and-block) | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** Pre-commit/CI secret-scan gate: the verifier runs the configured secret scan over the staged content and **blocks the commit** if secrets survive Stage 1. Detection only — does not rewrite files. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.3` |
| `staleness` (review-freshness) | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** A review is stale when the files listed in `reviewed_files` have changed since `review_of_commit` (detectable when `pinning: "commit"`). Staleness signals that a re-review may be needed. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.4` |
| `body cap` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** The `body_max_bytes` field: maximum byte length of `body_markdown` in the JSON sidecar before truncation applies. Active cap value stored in `review.json` at generation time. Replaces the legacy `summary cap` term. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.1` |
| `render cap` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** The `render_max_bytes` field: maximum byte length of the rendered `review.md`. May be smaller than `body cap` since MD adds structure overhead. Active cap value stored in `review.json` at generation time. Replaces the legacy `summary cap` term. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.1` |
| `staging directory` | **(predecessor vocabulary; superseded by the review charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md))** A `.staging-<NNNN>/` sibling directory used during atomic per-review directory writes. The writer creates the staging directory, populates it, then renames it to the final `<NNNN>-<slug>-<ref>/` name — POSIX rename is atomic within a single filesystem. Staging directories are gitignored. | spc-2 (predecessor store) + `spc-2-move-repoprompt-review-artifacts-into.1` |
| `bucket` ∈ `{lint, review, audit, gate}` | Assessment-surface classifier, separated by *what is compared to what*: `lint` compares an artefact against a rule about its form; `review` compares a change against judgement (family-1 verdicts); `audit` compares reality against a recorded commitment (family-2 verdicts); `gate` consumes findings and decides one action. `—` in a sub-verb table marks a non-assessment verb. Recorded per sub-verb in each `04-surfaces/` file's `## Sub-verbs` table and machine-checked by `surface_coverage`'s sub-verb pass. Closed list, PR-to-extend — an autonomous run given a criterion interprets it; a table lookup either matches or fails. | adr-40 + spc-27 (itd-122) |
| `decision class` ∈ `{intent, RFC, ADR}` | Roadmap-record classifier — three decision-record surfaces with distinct lifecycles (forward-user-facing / forward-contested / retrospective-settled) | adr-1, adr-5; see [`../../decisions/README.md`](../../decisions/README.md) |
| `kind` ∈ `{standalone, bundle-member, discipline}` | Persisted intent kind classifier — stays three-valued; `decision` is NEVER a persisted `kind` (it has no lifecycle directory under thin adoption) | itd-34 |
| `capture verdict` ∈ `{standalone, bundle-member, discipline, decision}` | **(staged)** Capture-TIME classifier verdict, produced by the capture-kind classifier the intent surface is designed to gain; no classifier sub-verb is registered today. The first three mirror the persisted `kind`; `decision` is capture-only: it routes a confirmed standing infrastructure choice to the existing ADR store (`adr-N`), never to a persisted `kind` or the intent lifecycle. Admitted by `suggested_kind` (advisory hint) but REFUSED by `plan_single`/`reclassify`. | itd-44 (spc-56, predecessor store) |
| `source.class` ∈ `{session_memory, external_pdf, external_transcript, external_article, oracle_review, work_notes, issue_ledger, dredge_synthesis, spec_modification_grammar, modification_grammar}` | Memory page source class | itd-36 |
| Lifecycle classes ∈ `{regenerable, append-only, compounding-curated}` | Artefact lifecycle taxonomy | `05-internals/04-universal-patterns.md § 8` |
| Review verdicts ∈ `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` | Carmack-style review verdicts | `05-internals/01-agents.md § Verdict-tag protocol` |
| Criterion verdicts ∈ `{MET, MET_WITH_CONCERNS, NOT_MET, INCONCLUSIVE}` | Per-criterion intent acceptance | itd-1 |
| `task_classes` (capability_scope tokens) ∈ `{oracle_review, intent_audit, spec_planning, code_rescue, principle_distillation, lifeboat_packing, audit, lint, surface_render, cross_document_audit, cold_reading}` | Closed enum, agent frontmatter, PR-to-extend. `cold_reading` names the four blind reading positions (itd-184): reusing `cross_document_audit` would name them as audits, and an audit judges against a standard, which is precisely the licence a widening reading does not hold. `oracle_review` names review work that reaches a model through the adr-25 **oracle seam** — the seam keeps its name even where the verb that consumes it does not (`disembark review`, spc-30), so the token survived that rename unchanged. This table is the source of truth today: the binary carries no `task_classes` schema and no cross-check test reads the field (iss-265). | itd-5 extension (idea-4) — lean ~10 tokens drawn from current abcd surfaces |
| `frozen_content_hash` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** SHA-256 hex string written to PRD frontmatter by `/abcd:intent plan` at freeze time. Non-self-referential: provenance fields included, operational fields excluded. Recipe documented in `prd.schema.json`. | spc-3 (predecessor store), task .5 |
| `intent_source_hash` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** SHA-256 hex string computed over the parent intent's body + stable frontmatter. Written by grill skill as `grilled_intent_hash`; copied to PRD as `source_intent_hash`. Recipe documented in `prd.schema.json`. | spc-3 (predecessor store), task .5 |
| `planning_attempt_id` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** UUIDv4 written to PRD frontmatter and the durable attempt journal by `/abcd:intent plan`. Used by GR004 to detect stale planning attempts. | spc-3 (predecessor store), task .5 |
| `prd_grandfathered` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** Boolean frontmatter field on pre-spc-3 planned intents. When `true`, GR002 and GL005 are suppressed-as-info (not blocker). Cleared on regrill. | spc-3 (predecessor store), task .5 |
| Grandfather migration | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** One-shot sweep at spc-3 ship time: appends `prd_path: null` + `prd_grandfathered: true` to every intent in `planned/` that predates the PRD requirement. | spc-3 (predecessor store), task .5 |
| `promote-check mode` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** The intent lint's `--promote-check <intent.md>` mode evaluates the intent as if it were already in `planned/`, firing GR002 and GL005 at planned-state severity for pre-flight checks. | spc-3 (predecessor store), task .5 |
| `abandon-attempt mode` | **(predecessor vocabulary; superseded by the native spec store, [`../../specs/README.md`](../../specs/README.md))** The intent lint's `--abandon-attempt <itd-N>` mode (also `/abcd:intent plan --abandon-attempt <itd-N>`) clears a stale planning attempt: removes the attempt journal and sets `planning_attempt_id: null` in the PRD. Freeze fields (`frozen_at`, `frozen_content_hash`, `spec`) are preserved — the PRD itself is not un-frozen. Remediation for GR004. | spc-3 (predecessor store), task .5 |
| `failure_mode_tag` ∈ `{hallucination, scope_drift, stale_context, under_specification_blindness, format_violation}` | Closed enum, in the later-phase Frontier Awareness intent | idea-4, a later phase |
| `work_item.type` ∈ `{intent_promotion, spec_task, command_run}` | Coordination claim unit (a later phase) | itd-33, a later phase |
| Claim primitives ∈ `{take, yield, escalate}` | Coordination conflict-resolution verbs (a later phase; cooperative-checkpoint semantics, no mid-flight abort) | itd-33, a later phase |
| Escalation choice ∈ `{wait_then_swap, swap_now, sequence, keep_both}` | Human-resolved escalation outcomes (a later phase) | itd-33, a later phase |
| `release.outcome` ∈ `{completed, abandoned}` | Audit-log outcome on claim release (a later phase) | itd-33, a later phase |
| `claim.status` ∈ `{active, paused, released}` | Three-state claim lifecycle; `paused` interlocks with itd-29's pause/resume/rewind (a later phase) | itd-33, a later phase |
| `recall` | Keyword-list field on each rules.json domain — natural-language phrases matched word-boundary against user prompts to trigger rule injection. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `domain` | Uppercase grouping key in rules.json (e.g. `COMMITTING`, `DOCUMENTATION`). Domains carry state (`active` / `dormant`), recall keywords, and rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `dormant` | rules.json domain state value (opposite of `active`). Recall-match injection is skipped for dormant domains; star-command `*<DOMAIN>` still activates them. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `active` | rules.json domain state value (default). Recall-match injection fires when prompt matches recall keywords. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `*<DOMAIN>` | Leading-anchored uppercase prompt prefix that explicitly activates a domain regardless of recall match. Regex `(?:^|\s)\*([A-Z][A-Z0-9_]*)(?=$|\s)`. Hyphen/period/slash following the domain name fails the boundary (no activation). Multi-star activates multiple domains in left-to-right order. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.5` |
| `force_refresh_every_n` | `.abcd/config.json` field under the `rules.` namespace. Integer; default 15 (`rules.DefaultRefreshBackstop`), which a missing key or a non-positive value falls back to. Every N prompts, the prompt-router hook forces a full re-inject regardless of dedup signature match (compaction recovery). | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.4` |
| `.abcd/config.json` | Repo-scope config file. Named in the in-repo carve-out in `05-internals/03-configuration.md` § The two `.abcd/` scopes, which places it in the repo scope alongside the rules overrides. Reads include `rules.force_refresh_every_n` and `docs.target`. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.8` |
| `rules.json` | Repo-scope rule overrides file at `<repo>/.abcd/rules.json`. Named in the in-repo carve-out in `05-internals/03-configuration.md` § The two `.abcd/` scopes. Validated by the Go binary rather than against a published schema document. The loader checks the structural invariants (`schema_version` is 1, every domain name is uppercase-anchored, every state is `active` or `dormant`, and no domain is left with nothing to say) and names what it refuses. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `COMMITTING` | Default plugin-bundled domain. Recall keywords trigger injection of commit-discipline rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `DOCUMENTATION` | Default plugin-bundled domain. Recall keywords trigger injection of documentation-discipline rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `ROADMAP` | Default plugin-bundled domain. Recall keywords trigger injection of roadmap/intent rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `ISSUES` | Default plugin-bundled domain. Recall keywords trigger injection of issue-tracking rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `INTENTS` | Default plugin-bundled domain. Recall keywords trigger injection of intent-capture rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `LIFEBOAT` | Default plugin-bundled domain. Recall keywords trigger injection of lifeboat/disembark rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `PII` | Default plugin-bundled domain. Recall keywords trigger injection of PII-protection rules. | spc-14 (predecessor store) + `spc-14-modular-rules-loader-prompt-router-hook.1` |
| `OPINIONS` | Default plugin-bundled domain, the eighth. Recall keywords trigger injection of the repository's standing conventions, whose rules point at the canonical principle pages rather than copying them. | spc-14 (predecessor store) + itd-3 |
| `managed-repo` | Folder kind: a git repo abcd already manages — has an ABCD marker block (or an in-tree `.abcd/`, or an `index.json` entry) and a `.git/` directory. Per brief `04-surfaces/01-ahoy.md` § What abcd manages. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.3` |
| `unmanaged-repo` | Folder kind: a git repo without abcd management; bare `/abcd:ahoy` offers install to adopt. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.3` |
| `unmanaged-folder` | Folder kind: not a git repo and no abcd markers; nothing to act on. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.3` |
| `root_commit` | Immutable repo identity key in `index.json`, computed via `git rev-list --max-parents=0 HEAD`. Survives rename, remote move, and GitHub-handle change. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.2` |
| `index.json` | History-store registry at `~/.abcd/history/index.json` recording each repo's identity + lineage. The **sole user-scope registry** (abcd is single-repo, adr-28 — there is no `workspaces.json`). Keyed on immutable `root_commit`. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.5` |
| `aliases` | Array of prior names a repo has had (e.g., renamed on GitHub). Recorded in per-root-sha `meta.json`. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.5` |
| `supersedes` | Lineage cross-ref in `index.json` repo entry: this entry was re-founded from another root-sha. | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.5` |
| `superseded_by` | Lineage cross-ref in `index.json` repo entry: this entry was superseded by another root-sha (re-founding produces a new entry). | spc-15 (predecessor store) + `spc-15-folder-classification-workspacesjson.5` |
| `voyage/` | The **operations** namespace (verb-side: what we did), as against `lifeboat` (noun-side: what gets carried). Lives at the **operator level**, `~/.abcd/voyage/<source-root-sha>/`, keyed on the root-commit SHA like the history store — never inside the source repo, and therefore never committed. Split by operation: `disembark/history.jsonl`, `embark/provenance.json`, `embark/from/<timestamp>/`. (adr-4 placed it at `.abcd/development/voyage/`; that collided with the `privacy-hygiene` audit rule, since voyage records absolute source paths.) | adr-35 (supersedes adr-4) |
| `history.jsonl` | The append-only log at `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl` — one line per disembark run: `manifest_sha256`, file list, oracle backend used, verdict. Genuinely appended, never rewritten whole. | adr-35 |
| `_provenance.json` | The lifeboat's own record of how it was produced: `schema_version`, source, tiers present, declared exemptions, and `manifest_sha256`. Written **last** — it is both the commit marker for a completed pack and the key to the destination safety gate (abcd never overwrites a directory it did not produce). Excluded from its own hash. | adr-35 |
| `manifest_sha256` | SHA-256 over the concatenation of `"<sha256>  <path>\n"` for every manifest entry, sorted lexicographically by path, POSIX separators, LF only, with `_provenance.json` excluded. adr-4 asserted this chain without defining it; adr-35 pins it. | adr-35 |
| `coverage.json` / `coverage.md` | The lifeboat's **first-class** report of what could *not* be filled: per brief section, the status, the confidence, the evidence cited, what was searched, and the question a human must answer. Schema aggregates across repositories — that aggregate is the experiment's readout. | adr-35, itd-88 |
| `graveyard/` | The lifeboat section carrying what a project tried and abandoned, in three layers: `archaeology.json` (Tier 0 evidence, no interpretation), `abandoned.json` (what the project declared dead), `lessons.json` (host-delegated interpretation — **every entry cites layer-1/layer-2 ids or is dropped by the validator**). | adr-35, itd-11 |
| Coverage status ∈ `{grounded, partial, blank}` | Per-section coverage result. `grounded` = every claim cites a source; `partial` = startable, not completable; `blank` = nothing in the repo grounds it. **A blank is a first-class result, not a failure** — it carries the question a human must answer. Machine-readable source of truth: the Go `Status` enum (`internal/core/lifeboat`). | adr-35, itd-88 |
| Source tier ∈ `{git, conventions, abcd-native}` | The class of source material a section is grounded from. **Cumulative** — a repo with conventions still has git — so a richer tier can never ground a section worse than a poorer one, and a test enforces it. `git` is present in every repository; `abcd-native` only where abcd manages the repo. Machine-readable source of truth: the Go `Tier` enum (`internal/core/lifeboat`). | adr-35, itd-88 |
