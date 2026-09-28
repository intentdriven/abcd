---
id: itd-51
slug: harness-adoption-readiness-rubric
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-27]
routed_from: ["spc-33:I-D1-extdep", "spc-33:I-D2-extdep"]
prd_path: null
severity: major
---

# abcd Knows What "Safe Enough to Adopt" Means Before a New Harness Arrives

## Press Release

> **abcd carries a written rubric of what any autonomous harness must guarantee before abcd will drive it — so when a candidate harness for the run seam appears, adoption is a checklist run, not a fresh argument.** abcd is an abstraction layer: operators work the `/abcd:*` surface and abcd drives an autonomous harness underneath through a **pluggable run seam** ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)). A security review of an early bundled harness surfaced concrete failure modes — unbounded runaway when a session timeout never fires, a worker that can forge its own review verdict, protected paths bypassable through the shell. Those specific defects were assigned to the predecessor store's spc-33 (abcd-owned code) and spc-37 (abcd-orchestrated sessions); neither spec was carried into the Go rebuild, and the live `spc-33` and `spc-37` are unrelated records. But the deeper asset is the *generalization*: the same review implicitly defined a set of properties abcd cares about in ANY harness it drives. The pluggable seam makes that rubric load-bearing — a candidate engine is only as safe as it scores. This intent makes the set explicit and durable — a harness-adoption-readiness rubric — so abcd never re-derives "is this safe to drive?" from scratch, and so a future adoption is evaluated against stated requirements instead of vibes.

> "The first time we vetted a harness it took a full security review to notice the worker could run for a hundred hours unattended," said Carol, the platform tech lead. "I don't want to re-learn that the hard way on the next one. I want a page that says: here is what abcd refuses to drive blind — bounded runtime, verdict integrity it can't forge, a real boundary on what it can write and push. Score the candidate against it. Done."

## Why This Matters

abcd's whole proposition is that operators trust the `/abcd:*` surface and stay out of the dependency's complexity. That trust is only honest if abcd itself knows what it is willing to drive. Right now that knowledge is implicit — scattered across review findings, two specs of the predecessor store (spc-33, spc-37), and institutional memory. The moment a candidate harness for the run seam appears, that implicit knowledge has to be re-assembled under time pressure, which is exactly when important properties get dropped.

A written rubric converts a one-off security review into a reusable adoption gate. It also sharpens the predecessor store's spc-37, which enforced guarantees for the harness abcd drove then; this rubric states the guarantees abcd requires of *any* harness, so a harness-specific guarantee list reads as one instantiation of the rubric rather than an ad-hoc list. The rubric is forward-looking and has no implementation surface of its own — it is a decision artifact (a scored checklist) that downstream adoption work consumes.

Critically, this is **not** a plan to swap the configured harness. This intent only ensures abcd is *ready to evaluate* a candidate whenever one becomes available — readiness, not migration.

## What's In Scope

- A durable rubric document enumerating the properties abcd requires of any harness it will drive under the hood, grouped by the categories the 2026-06-02 review exposed: bounded runaway (wall-clock / loop-count / token budget that abcd can enforce externally), timeout-mechanism-as-precondition, in-flight pathology detection, review-verdict integrity from the reviewer's own output channel, write/push boundary enforceable in abcd's own layer, and no info-leak from orchestration artifacts.
- For each rubric item: a plain statement of the required guarantee, and how abcd would verify a candidate satisfies it (a check, not just a wish).
- A short scoring/decision shape so a candidate can be marked pass / conditional / fail against the rubric, with conditional-pass tied to what abcd's own wrapper would have to supply.
- A back-reference from whichever live spec enforces the guarantees for the harness abcd drives, noting it is the instantiation of this rubric (the predecessor store's spc-37 played that role; no live spec does).

## What's Out of Scope

- **Choosing or building a replacement harness.** No replacement is planned; this is the evaluation rubric only.
- **Fixing the 2026-06-02 findings.** Those were assigned to the predecessor store's spc-33 (abcd-owned code) and spc-37 (abcd-orchestration guarantees). This intent generalizes the *requirements*, it does not re-fix.
- **Editing the current harness's upstream files.** abcd does not fork upstream; the rubric is about what abcd requires and verifies, not about patching what exists.
- **A new `/abcd:*` surface.** The rubric is a decision artifact, not an operator-facing command.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** the rubric document, **when** a contributor reads it, **then** it lists the abcd-required harness guarantees grouped by category, each with a stated verification check, in plain decision-artifact form (no implementation surface).
- **Given** a hypothetical candidate harness, **when** it is scored against the rubric, **then** each item resolves to pass / conditional / fail, and any conditional pass names the wrapper-level compensation abcd would have to supply.
- **Given** the live spec that enforces the guarantees for the harness abcd drives, **when** a reader follows its references, **then** it is identified as the instantiation of this rubric, and the rubric does not duplicate its enforcement detail.
- **Given** the rubric, **when** a reader checks its framing, **then** it states explicitly that no replacement is planned and the rubric is readiness-only.

## Open Questions

- Where does the rubric live as a durable artifact — an ADR (it is a standing decision), a research-note record-type, or a dedicated `roadmap/` document? (Relates to the open research-notes record-type question logged in a working-log entry.)
- Should the rubric's verification checks be purely manual (a human scores a candidate) or should some be machine-runnable probes reusing the existing external-tool monitor surface (the doctor/probes layer of spc-34, predecessor store)?
- Does the rubric also cover non-safety adoption properties (cost, latency, observability, recovery), or stay scoped to the safety/integrity properties the 2026-06-02 review exposed?
- How does the rubric stay in sync when a live spec refines the guarantee set — is the rubric the source of truth such a spec cites, or a downstream summary?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Source: the security/architecture review block in a dated working-log entry (2026-06-02),
  generalized from specific findings into standing requirements.
- Instantiated-by: **spc-37, predecessor store** (abcd orchestration-layer safety
  when driving the harness of the time) — the rubric's first concrete
  application. The Go rebuild carried no successor spec.
- Routed-here: the `[external-dep]` harness halves of the predecessor store's
  spc-33 cluster-I
  deferrals — **I-D1** (abstraction-layer boundary) and **I-D2** (review-queue
  auto-drain) each split an abcd-owned half (routed to itd-53) from an
  `[external-dep]` harness half. Those external-dep halves are owned by the
  harness-readiness rubric (this intent) and were enforced by the predecessor
  store's spc-37, not built in its spc-33 sweep. Tracked by the `routed_from`
  tokens `spc-33:I-D1-extdep` and `spc-33:I-D2-extdep`, which name the
  predecessor store's spc-33, not the live one.
- Sibling cleanup: **spc-33, predecessor store** (Phase 3→4 cleanup) — the
  abcd-owned defects the same review found.
- Amended 2026-09-26 (iss-239): every `spc-N` in this draft names the retired
  predecessor spec store; the live `spc-33`, `spc-34` and `spc-37` are
  unrelated records (see the specs charter, "Two `spc-N` Namespaces").
- Principle: the abcd abstraction-layer boundary — `/abcd:*` is where abcd's
  guarantees live; abcd must know what it requires of anything it drives beneath
  that surface.
