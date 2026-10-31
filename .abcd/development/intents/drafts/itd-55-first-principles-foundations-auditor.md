---
id: itd-55
slug: first-principles-foundations-auditor
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
severity: minor
---

# abcd Can Tell Whether Its Own Reasoning Rests On Bedrock Or On An Unexamined Assumption

## Press Release

> **abcd gains a foundations auditor: a surface that takes a reasoned document — a brief claim, an ADR rationale, an intent's "Why This Matters" — and reports where its justification actually terminates, flagging the difference between a genuine first principle, an accepted convention, a justification that simply stops, and a circular dependency that smuggles the conclusion into a premise.** abcd already stress-tests acceptance criteria (the grill) and checks whether delivery matches intention (the fidelity reviewer). Neither asks the deeper question: is the *reasoning itself* epistemically honest about where it starts? This auditor adds that lens. It surfaces the claim architecture, excavates the unstated premises, audits the regress terminus, interrogates the causal account through the four causes, and checks that the mode of reasoning fits the kind of claim being made — turning "this feels well-argued" into "here is exactly what this argument is standing on, and whether that ground holds."

> "Our briefs read persuasively, which is exactly the danger," said Bob, a staff engineer who keeps abcd's reasoning honest. "Persuasive prose hides where the justification quietly ran out. I want a pass that says: this claim bottoms out in an unexamined assumption, and this other one is circular — it assumes the thing it's trying to prove. Then I can fix the foundation instead of polishing the surface."

## Why This Matters

abcd's entire proposition is that it forces product-thinking before engineering: the press-statement-shaped intent, the "Why This Matters" section, the brief's mental map all exist to make the *why* explicit and defensible before scope is set. But a why can be explicit and still be unfounded — resting on a premise nobody examined, or circular, or borrowing the certainty of one domain and misapplying it to another. abcd has no tool that catches this. The grill challenges whether acceptance criteria are testable; the fidelity reviewer checks whether what shipped matches what was promised; the lint enforces vocabulary. None of them audits the *foundation* a claim terminates on.

This is the missing discipline at the root of abcd's own value chain. A framework that disciplines product clarity should be able to turn its discipline on its own reasoning — to distinguish a claim grounded in a genuine first principle or an acknowledged convention from one that stops at an unexamined ideological commitment or quietly presupposes its own conclusion. The method is well-established (Aristotle's *archai*: the regress-terminus classification, the four causes, domain-appropriate reasoning) and maps cleanly onto the documents abcd already produces.

## What's In Scope

- An analytical surface that takes a target reasoned document (an intent's "Why This Matters", an ADR rationale, a brief claim, or supplied prose) and produces a foundations audit.
- The audit operations: surface the claim architecture (conclusion / explicit premises / inferential structure); excavate the implicit premises as a load-ranked inventory; audit the regress terminus per premise, classified as genuine-first-principle / domain-conventional / unjustified-stop / circular-dependency; interrogate the causal account via the four causes (material / formal / efficient / final); assess domain-appropriateness (reasoning mode vs claim type; normative-smuggled-as-empirical; abstraction vs evidence level).
- A forcing-function synthesis naming the single most load-bearing unexamined assumption or the deepest foundational flaw — not an undifferentiated list.
- A clear delineation from the existing grill (this audits foundations, non-interactively; grill challenges criteria, interactively) and the fidelity reviewer (intent-vs-delivery), so the three surfaces compose rather than overlap.

## What's Out of Scope

- **Replacing or duplicating `/abcd:intent grill`.** Grill stress-tests acceptance criteria interactively; this audits a document's reasoning foundation. They are complementary; this intent must not re-implement grill's interactive loop.
- **Fact-checking or external-evidence gathering.** The auditor analyses the internal epistemic structure of the supplied reasoning, not whether its empirical claims are true in the world.
- **Auto-firing on a lifecycle transition.** Whether the auditor runs automatically (e.g. on intent promotion) is a later policy question, not part of establishing the surface.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a reasoned document, **when** the auditor runs, **then** it reports the claim architecture, a load-ranked implicit-premise inventory, and a per-premise regress-terminus classification using the four categories (including an explicit circular-dependency flag).
- **Given** an argument with a genuinely circular premise, **when** audited, **then** the circular dependency is flagged as a critical foundational flaw, not merely noted as a weakness.
- **Given** the audit completes, **when** a reader reaches the synthesis, **then** it names the single most load-bearing unexamined assumption or deepest foundational flaw (a forcing function, not a flat list).
- **Given** the auditor and the existing grill, **when** their scopes are compared, **then** the auditor audits reasoning foundations and does not re-implement grill's interactive criteria-challenge loop.
- **Given** a document that misapplies one domain's reasoning mode to another (e.g. demonstrative certainty applied to a normative claim), **when** audited, **then** the domain-appropriateness check flags it.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [memory-knowledge SOTA note](../../research/notes/2026-10-23-memory-knowledge-sota.md).

- **Naming the theory in the prompt is a workable zero-shot method.** Gupta, Zuckerman and O'Connor (ACL 2024) produce claim, reason and warrant triples from Toulmin's model by prompting with the theory's name, and evaluate "coverage and validity" through a human study, automatic evaluation on prior datasets and robustness checks across models, prompts and theories. The source skill's move with Aristotle's framework is the same one; validity is measured by people, not assumed.
- **The field is prompt-driven now and warns about judging itself.** The 2025 argument-mining survey flags "risks of bias and evaluation circularity" in LLM-assisted annotation and consolidates "LLM-judge reliability" as one of the evaluation practices to assess, beside component-level accuracy and soft-label quality (Li et al.); the judge-calibration note's "ground truth has to be manufactured" applies, so a first run over ten records with each circularity flag labelled true or false is part of the build.
- **The forcing-function synthesis has a human precedent.** The premortem asks for the reasons the plan failed and makes dissent safe (Klein, HBR 2007); the "single most load-bearing unexamined assumption" is that move made by a reader with no stake.
- **The output contract exists in two neighbours.** The intent auditor's verdict JSON carries a cited pointer on every claim and the cold reading emits three-element detections in document order; both are host-delegated definitions validated by the binary on ingest.
- **Path 2**: a native, host-delegated agent definition shaped to Toulmin explication for the structure with the regress-terminus classes on each warrant, a structured report as the one canonical output (the table and the commentary are renderings of it), single-pass, advisory, as a standalone surface taking a document path so it reads ADRs and the brief; the seam is the output schema. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

- Is the auditor a sub-verb of `/abcd:intent` (e.g. `/abcd:intent foundations <itd-N>`) or a standalone surface that takes any document path? The first keeps it in the intent flow; the second makes it reusable on ADRs and the brief.
- Interactive or single-pass? The source method is a single analytical pass; the grill is interactive. A single non-interactive pass keeps the surfaces distinct and composable.
- Which output shape is canonical for abcd — the structured report, the flowing commentary, or the compressed diagnostic table (the source skill offers all three)?
- Should a foundations audit ever be a gate (e.g. an intent cannot promote with an unresolved circular-dependency flag), or always advisory?
- Provenance: the method derives from a third-party `first-principles-analysis` skill (MIT-spirit) and Aristotle's framework — record the adaptation in an ACKNOWLEDGEMENTS file when built, mirroring `abcd-intent-grill`.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Source method + evaluation: `.abcd/development/research/notes/socratic-and-first-principles-skills-evaluation.md`
  (the `first-principles-analysis` skill — Aristotelian *archai*: regress
  terminus, four causes, domain-appropriateness).
- Complements: `skills/abcd-intent-grill/` (interactive criteria challenge) and
  the fidelity reviewer of spc-12, predecessor store (intent-vs-delivery) — three distinct lenses that
  compose.
- Sibling harvest: the `socratic-grill` skill's domain-agnostic vocabulary +
  temperature modes are routed separately as an abcd-intent-grill enhancement
  (logged in a working-log entry), not into this intent.
