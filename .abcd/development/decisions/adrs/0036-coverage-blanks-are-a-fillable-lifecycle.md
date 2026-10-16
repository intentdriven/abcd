---
id: adr-36
slug: coverage-blanks-are-a-fillable-lifecycle
status: accepted
date: 2026-07-15
supersedes: null
superseded_by: null
related_intents: [itd-88, itd-90, itd-86]
related_rfcs: []
related_adrs: [adr-35]
---

# ADR-36: Coverage blanks are a fillable lifecycle — authored is not extracted, and the interview is its own step

## Context

[adr-35](0035-lifeboat-as-coverage-experiment.md) made a blank a first-class result: the coverage probe reports, for every brief section it cannot ground, what it searched and **the question a human must answer**. M2 built that — `abcd disembark probe` produces a per-section `{status, searched, question}` report, and the cross-repo aggregate settled the brief structure. The 2026-07-15 gate decision (recorded as [dcn-2607150000000001](../../../work/decisions/dcn-2607150000000001-the-lifeboat-coverage-experiment-s.md)) then made an operative call: `product/personas` is **manual** — not derivable from a repository, ever, and always a human's to write.

That decision exposed a hole the plan implied but never named: **the coverage report knows what to ask, but nothing says who answers, or when, or where.** Three facts force the issue:

1. **The person with the tool is not the person with the answer.** Disembark is a facilitator operation — someone technical, pointing abcd at a repository. The blanks that matter (personas, the mental model, why it was built this way) are owned by a *product thinker*, who may never touch the tooling. The facilitator has the questions and none of the answers; the product thinker has the answers and is not in the room.

2. **Neither end of the round-trip is reliably that room.** Disembarking a stranger's dead repo, the facilitator cannot answer the blanks — mining is not authoring. Embarking into a new repo, the product thinker *should* be present, but embark is still a facilitator-run operation and the guarantee is weak. Pinning "answer the blanks" to either disembark or embark assumes a presence that is often not there.

3. **A filled blank has no repo source.** The moment a human answers "who are the users," the brief gains a claim that cites no file. adr-35's whole guarantee — every grounded claim cites its source — is silent about a claim that was *authored*, not *extracted*. Left unhandled, an opinion launders into a fact, which is the exact fiction the experiment was built to prevent.

## Decision

**A coverage blank is a durable, addressable object that is progressively filled across the round-trip. Answering it is a distinct step, decoupled from disembark and embark, and a filled blank is marked as authored, never disguised as extracted.**

1. **The coverage report is a portable interview script, not a snapshot.** Each blank travels with the lifeboat (`coverage.{json,md}` is already a first-class output per adr-35) as an object that can be answered later, by a different person, in a different environment. The JSON *is* the interface between the facilitator's tool and whoever holds the answers — so the answer need not happen where or when the probe ran.

2. **Two classes of blank, declared in the mapping.** A blank carries a `kind`:
   - `extractable` — a source or a better adapter could ground it (e.g. `evidence/open-questions` once a `TODO`/`FIXME` scan exists; `constraints/dependencies` once the manifest set is complete). The blank is a **coverage failure**, and the fix is in abcd.
   - `human-owned` — never derivable from a repository by design (`product/personas`, `product/mental-model`). The blank is not a failure; it is a **standing prompt**. It must be framed to the product thinker as "this is yours to write," never as "abcd could not find it."

   `internal/core/lifeboat/mapping.go` declares each section's kind. The gate's "personas is manual" decision is exactly the assignment of `product/personas` to `human-owned`.

3. **Answering is decoupled from disembark and embark.**
   - **Disembark raises, it does not force.** It packs the questions precisely; it *offers* optional inline answering (you may be rescuing your own project and know the answers) but never blocks on it. Mining a stranger's repo, every blank is skipped and carried forward — that is correct.
   - **The interview is its own step.** The act of answering runs asynchronously, driven by the coverage JSON, wherever the product thinker actually works. This is a host-delegated, environment-agnostic capability: abcd defines the JSON in and the JSON out, not the environment it runs in. It is specified at the user level by [itd-90](../../intents/drafts/itd-90-brief-interview-for-the-blanks.md).
   - **Embark reconciles and re-surfaces.** Embark scaffolds the grounded brief, ingests any answers, and surfaces every still-open blank as the first thing the product thinker sees — the backstop for anything the interview did not reach.

4. **Authored is not extracted.** A filled blank carries provenance that is structurally distinct from a grounded section's citation:
   - a grounded claim cites `extracted-from: <file>`;
   - an answered blank cites `authored-by: <who>, <when>` (with an `agent-assisted` flag when the interview used a host-delegated helper).

   adr-35's guarantee is preserved, not weakened: an authored claim's "source" is a person and a date — an honest non-file provenance — never a file it did not come from. This is the same discipline as Pass B's declared exemption in `_provenance.json`.

5. **The coverage schema grows to v2 — a fillable object.** `SectionCoverage` (M2, schema v1) gains `kind ∈ {extractable, human-owned}`, `resolution ∈ {open, answered, deferred}`, and an optional `answer` block carrying the authored-by provenance above. `schema_version` becomes `2`; the M2 version guard already refuses a report newer than the tool knows, so a v2 report requires v2 tooling — an acceptable, loud break, not a silent misread.

6. **The interview is not the cold reading ([itd-86](../../intents/drafts/itd-86-cold-reading-surface.md)).** They point in opposite directions and must not be merged: the cold reading *reviews* a document for internal contradictions and is deliberately **denied** context; the interview *answers* the coverage report's open questions and is fed the product thinker's context. Different actor, different direction, different output.

## Alternatives Considered

1. **Answer blanks only at embark.** Rejected: the product thinker may not be present at embark either — it is a facilitator operation — and the person who holds the answers may never touch abcd. Decoupling via a portable interview script serves the asynchronous reality; pinning to embark assumes a presence that is often absent.

2. **Answer blanks at disembark, as the single point.** Rejected as the *sole* point: mining a stranger's dead repo, the facilitator cannot answer. Retained as an *optional* affordance, because disembarking your own dying project is the case where you do know.

3. **Treat all blanks identically.** Rejected: it conflates "abcd failed to extract this" with "this is inherently a human's to write." That misframes `product/personas` as a coverage failure — demoralising for the product thinker and dishonest about what abcd can promise — and it hides the `extractable` blanks that are genuinely adapter debt (iss-98, iss-99, iss-100).

4. **Let a filled blank be indistinguishable from an extracted claim.** Rejected: it launders an opinion into a fact and breaks adr-35's cite-your-source guarantee — the precise fiction the coverage experiment exists to prevent.

## Consequences

**Gains:**

- The round-trip survives the human's absence at any single step. The questions are raised by the tool and answered by whoever holds the answers, whenever they can, through one JSON interface.
- `human-owned` sections are framed as prompts, not failures. Personas stops reading as "abcd could not find your users" and starts reading as "these are yours to name."
- The brief stays honest under filling: authored and extracted are visibly different, so a lifeboat never presents an opinion as an extraction.

**Costs / obligations:**

- **The coverage schema evolves.** M2's `SectionCoverage` grows `kind`, `resolution`, and an authored `answer` block; `schema_version` → 2. The aggregate and the renderers must carry the new fields.
- **The mapping must declare each section's kind.** `mapping.go` gains a `Kind` per row; at minimum `product/personas` and `product/mental-model` are `human-owned`. This is the durable form of the gate decision.
- **Vocabulary must be registered** in `02-constraints/04-naming.md`: `extractable` / `human-owned` (blank kinds), `open` / `answered` / `deferred` (resolution), `authored-by` vs `extracted-from` (provenance), and "interview" as the name of the answering step.
- **The interview agent is a host-delegated oracle** and inherits itd-5's obligations — `prompt_version`, `reads_untrusted_input`, and a canary fixture — like every other agent in the M6 synthesis set.

**Downstream decisions enabled:**

- M3's packer writes coverage with `kind` and `resolution`, and honours the optional inline answer at disembark.
- M5's embark reconciles ingested answers and re-surfaces open blanks first.
- itd-90 builds the interview surface to this contract; the M6 synthesis agents can consume authored answers as legitimate, provenance-tagged brief content.
