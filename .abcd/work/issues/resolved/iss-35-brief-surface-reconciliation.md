---
schema_version: 1
id: "iss-35"
slug: "brief-surface-reconciliation"
severity: "critical"
category: "drift"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: ".abcd/development/brief/05-internals/08-skills.md"
resolution: "graduation gate shipped and armed: surface_coverage record-lint blocker, 16 surface chapters, skills-to-commands relabel, machine-checked staged Status column, crosscheck wired as release gate; maintainer grill decision 2026-07-24 (DECISIONS.md); residue tracked as iss-122"
impact: internal
---

brief-vs-shipped-surface reconciliation: 05-internals/08-skills.md claims abcd ships zero user-facing skills and six top-level commands while 04-surfaces/README.md itself tables nine and /abcd:consult and /abcd:ingest are shipped; the shipped skills violate the brief criterion that any artefact mutation is a command, not a skill; the skills/ layout described (abcd-ahoy, commit-attribution, secrets-and-pii) is fictional vs the real consult/ and ingest/; the implemented, user-reachable abcd docs lint and abcd history verbs have no home in 04-surfaces at all; the operator-internal paragraph contradicts the commands/ directory that exists. Detector (per spec-moves-with-the-surface): a record-lint cross-check that every entry under commands/ and skills/ resolves to a brief surface row, and every brief surface row resolves to a shipped or explicitly staged surface. Acceptance corpus: each falsified claim above — the check fails on all of them today. Fix amends the criterion or the surface in one change, never silently.

---

**Detector run 1 (2026-07-10, autonomous run, workflow MVP per
script-first):** the bidirectional cross-check ran as 22 checker agents
(11 brief docs verified against the shipped binary, 11 real surfaces
seeking their brief home). **150 unique discrepancies**: false-claim 77,
fictional-layout 29, undocumented-surface 24, criterion-violation 15,
stale-count 5. Every falsified claim enumerated above reproduced, plus:
no `abcd init`/`config`/`run` verbs exist though the brief cites them
unmarked; launch's brief row claims artifact-cutting while the binary is
read-only preview; the bare-command-as-help "universal convention" is met
by no shipped verb; `docs` and `history` verbs have no brief home. Worst
docs: 05-intent (17), 07-memory (13), 01-ahoy/04-launch/06-capture/
08-abcd/08-skills (10 each). Full corpus: the run's local log
(iss35-discrepancies.json) — re-derivable by re-running the workflow.
Next: reconcile per doc behind this detector (amend criterion or surface,
never silently), then graduate the check to a record-lint rule
(spec-moves-with-the-surface).

---

**Reconciliation batch 1 (2026-07-10, b32cf40):** 60/150 dispositioned
across the five worst surface chapters (01-ahoy, 04-launch, 05-intent,
06-capture, 07-memory). Every finding re-verified against the binary
before editing; 2 rows batched for maintainer adjudication.

**Reconciliation batch 2 (2026-07-10):** the six remaining direction-A
surface docs (08-abcd, 05-internals/08-skills, 04-surfaces/README,
09-reflect, 02-disembark, 03-embark) — **43 rows dispositioned: 20 fixed,
20 staged, 3 adjudication, 0 rejected**. Reconcilers ran on **Opus 4.8
high** (Fable 5 credit exhaustion this session; see NEXT.md model-gap note)
and emitted per-row binary evidence; every load-bearing claim independently
re-verified in the main loop (all held — no invented-home defects, unlike
the batch-1 Fable sweep that needed a 30-item review pass). record-lint and
docs lint both clean. 3 new adjudication items batched to NEXT.md's
ADJUDICATION QUEUE (docs/history surface-taxonomy ×2; the read-only-skill
boundary rule vs the three mutating shipped skills). Remaining: re-run the
cross-check detector to measure the direction-B ratchet, fix true
direction-B leftovers, then graduate the check to a record-lint rule.

**Detector re-run (2026-07-10, wf_ed4cf12c-231, same 22-checker arg set):
150 -> 24 discrepancies** (false-claim 10, undocumented-surface 5,
fictional-layout 3, criterion-violation 4, stale-count 2) — an 84% ratchet.
Finding: the remaining 24 were mostly **direction-A leftovers the per-doc
reconciliations had not sampled** — the LLM cross-check is a stochastic
discovery tool, not a convergent gate; each run surfaces a different subset —
plus one self-introduced slip and two adjudication clusters. 13 fixes applied
across 8 docs (6ab1432), each re-verified against the binary; 1 detector
finding disproved and skipped (repo-scope vs user-scope meta.json). The 2
adjudication clusters remain the only substantive criterion/direction-B items
(docs/history taxonomy; read-only-skill rule vs mutating skills) — batched for
the maintainer, NOT fixed. **Lesson: chasing the stochastic detector to zero
is the wrong loop; the fix is to graduate the cross-check to a deterministic
record-lint rule (spec-moves-with-the-surface) so the invariant is enforced
structurally.** That graduation is the next work item.

**Graduation is a design-STOP (2026-07-11, slice 10), held for maintainer
sign-off.** Building the record-lint rule surfaced that the detector is
bidirectional but only Direction B (surface→brief coverage) is deterministically
lintable; Direction A (brief→binary-behaviour) is irreducibly semantic and stays
an agent/periodic check. Even the structural half cannot be armed to green until
the docs/history surface-taxonomy adjudication is decided (the rule fires on the
three chapterless shipped verbs), and it needs a decided staged-vs-shipped marker
convention. Design options (3, recommend Option A) written to
`.abcd/development/plans/2026-07-11-iss35-record-lint-graduation.md`. iss-35 stays
open ONLY for this gate — the reconciliation itself is complete to the limit of
what is decidable without the maintainer (remaining discrepancies = the 2
adjudication clusters). Blockers: adjudication items 5 (docs/history taxonomy)
and 6 (skill classification) in NEXT.md.
