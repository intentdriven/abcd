# principles/

Distilled cross-cutting design principles — the rules that hold across the whole
system (e.g. "transport-agnostic core", "wired or it isn't done", "host-delegated by
default"). A first-class abcd artefact: the lifeboat packs *decisions, principles,
pitfalls, and the spine*, so principles live here — distinct from `../decisions/`
(ADRs: the ratified *why we chose*) and `../intents/` (the user-facing *why it
matters*).

One principle per file. Populated during the Phase 0.5 content reconciliation.

**Promotion path.** The full ladder has three rungs: a **principle** — the
normative statement (a value, a definition of proven/done/good, or a rule of
action) that survives any particular mechanism; beneath it an **enabling
convention, script, or file format** — the MVP, the smallest unenforced
enabler; and above it a **discipline-kind intent or core absorption** — the
tool, which makes the practice enforced or cheap at the price of becoming a
maintained artefact with its own lifecycle (false-positive budget, gaming
surface, saturation, kill criterion). An unenforced convention remains a
principle-layer artefact in this record; "MVP" names the enabling artefact
beneath a principle, not a third governance category. The moment a principle
gains a mechanical gate (a lint code, a hook, a CI check), it is promoted to
a discipline-kind intent — the lifecycle'd, spec-inherited form (see
[`../intents/disciplines/`](../intents/disciplines)); enforced principle ⇒
discipline, and this directory is the not-yet-enforced layer. The ladder also
runs downward: a tool demotes to advisory on stale calibration and archives
to regression-only on saturation.

**Intake.** A candidate's layer is its *entry rung* — repo-relative and
dated: the rung where the actionable delta sits given the current record,
with every rung marked exists/partial/absent. Evidence artefacts (acceptance
corpora, calibration fixtures, baselines) are attachments to a rung, never
rungs. Record-shaped work may declare a degenerate ladder (principle = MVP,
or topping out at MVP). Two rules: articulate the full ladder for every
candidate, and never fabricate an absent rung. Provenance:
[`../research/notes/2026-07-09-practice-mvp-tool-extraction.md`](../research/notes/2026-07-09-practice-mvp-tool-extraction.md).

**Typed claims.** The family is a declared record store
([adr-2609021016270132](../decisions/adrs/2609021016270132-the-principles-family-is-a-declared-record-store-whose-entri.md)):
an entry's handle is `prn-<filename stem>`, and an entry may open with a
frontmatter block declaring what kind of claim it makes and what it rests on.

```yaml
---
id: prn-<filename stem>
claim_type: causal        # criterion, causal or context (mechanism reads as causal)
reference: "abcd lint"    # a record handle, or a double-quoted surface name
comparison: "What was compared to produce it, in one sentence."
evidence: [itd-181, cond-2608311949582375]   # record handles and scope-condition identities
---
```

A key considered and declined is the literal `null`; an absent key is a claim
not carried. Population is forward-only: an entry carrying none of the four keys
is counted by the record lint as untyped (`principle_untyped`, a warning), and
nothing backfills one. An entry carrying any of them carries all four
(`principle_claims`, blocking), states its rule as a `**The rule.**`
paragraph rather than a `## The rule` heading, and carries no record handle and
no link of any shape (inline, reference-style, autolink or bare URL) in that
paragraph or its H1 title, because a reading receives the title and that
paragraph and nothing else. Evidence naming a scope condition is read against
the condition's standing disposition: falsified blocks (`principle_falsified`),
and narrowed, untested or unresolvable is reported (`principle_inheritance`).
