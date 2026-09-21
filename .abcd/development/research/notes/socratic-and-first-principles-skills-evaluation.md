# Evaluation: socratic-grill + first-principles-analysis skills

Two third-party `.skill` archives (each a single `SKILL.md`, MIT-spirit, dated
2026-05-17) were evaluated for usefulness to abcd. Both are the work of the
GitHub user [mglgit](https://github.com/mglgit) (author identification
supplied by the maintainer, 2026-08-28; the account lists no public
repositories, so the archives have no resolvable public source and the
MIT-spirit licence stays unverifiable). Source archives were on the
author's Desktop; their content is summarized here so the evaluation survives
independently of those files. This note is the durable record; the actionable
outcomes are itd-55 (new intent) and an abcd-intent-grill enhancement entry in
`.work/issues.md`.

## socratic-grill — OVERLAPS an existing abcd surface

A domain-agnostic Socratic cross-examiner. Five passes (map structure → surface
hidden assumptions → find contradictions → test robustness → demand
justification), three temperature modes (`explore` / `challenge` /
`stress-test`), and argument-vs-system-design vocabulary tables so the same moves
apply to prose or to architecture. Output: assumption inventory, contradictions,
robustness failures, unanswered questions, a forcing-function verdict.

**abcd already has this, designed further:** the grill is a two-phase Socratic
challenger (interactive one-question loop, 12-question cap, EARS rewrites, Toulmin
tagging, exit ramps, glossary/lite tiers, Phase-2 PRD synthesis) — itself adapted
from `mattpocock/skills` (`/grill-me`, `/grill-with-docs`). So socratic-grill is a
sibling of something abcd specified deeper. Importing it whole would collide with
the grill and violate the one-canonical-surface-per-concept rule (the
abstraction-layer boundary).

> **Pointer corrected 2026-07-13.** This note originally cited a
> `skills/abcd-intent-grill/` directory and its ACKNOWLEDGEMENTS as existing.
> They do not exist in the tree: the Go rebuild left the grill as *planned,
> unbuilt* work — [itd-27](../../intents/superseded/itd-27-grill-skill-and-glossary.md),
> [adr-7](../../decisions/adrs/0007-grill-skill-and-glossary.md), extended by
> [itd-42](../../intents/planned/itd-42-coherence-aware-grill.md). The harvest
> disposition below stands; its target is itd-27/itd-42, and the mattpocock
> attribution is owed when the grill is built, not before.

**What socratic-grill does better (the harvest):**
1. **Domain-agnostic targeting.** abcd-intent-grill only grills `itd-N` intents.
   socratic-grill's structure works on ANY artifact — a spec, an ADR, an
   architecture sketch. abcd has no way to grill a non-intent today (the spc-37
   grill this session was done by hand, with no skill behind it).
2. **Temperature dial.** `explore` / `challenge` / `stress-test` is a cleaner,
   more explicit model than abcd-intent-grill's gentle-refine-vs-adversarial-grill
   binary.

**Disposition:** do NOT import as a skill. Harvest (1) the domain-agnostic
vocabulary tables and (2) the three temperature modes into a future
abcd-intent-grill enhancement (generalize grill to target specs/ADRs/designs,
add an explicit temperature parameter). Logged in `.work/issues.md` as an
enhancement candidate.

## first-principles-analysis — fills a GENUINE GAP

An Aristotelian *archai* foundations auditor — distinct from argument critique:
it asks whether an argument is *epistemically honest about where it starts*. Five
operations: surface claim architecture → excavate implicit premises → **audit the
regress terminus** (classify where justification stops as genuine-first-principle
/ domain-conventional / unjustified-stop / **circular-dependency**) → interrogate
the causal model via the **four causes** (material/formal/efficient/final) →
assess domain-appropriateness (is the reasoning mode right for the claim type;
is a normative claim being smuggled as empirical). Three output formats, three
tones. Framework from *Posterior Analytics* / *Physics* II / *Nicomachean Ethics*
I, used as method not citation.

**abcd has NOTHING like this.** abcd's whole pitch is forcing product-thinking
before engineering (the press-release / "Why This Matters" discipline). But abcd
has no tool that audits whether a "Why This Matters" rests on a genuine first
principle vs an unexamined assumption or a circular justification. The
fidelity-reviewer (spc-12) checks intent-vs-delivery fidelity; the grill
(abcd-intent-grill) stress-tests acceptance criteria; neither audits the
*foundations* of the reasoning. The regress-terminus classifier (esp. the
circular-dependency flag) and the four-causes lens are a strong fit for auditing
the brief's claims, ADR rationales, and an intent's "Why This Matters".

**Disposition:** capture as a NEW abcd intent (itd-55) — a foundations/first-
principles auditor surface, drawing on this method. NOT a duplicate of grill: the
grill challenges claims interactively; this audits the *foundation* a claim
terminates on. Likely a non-interactive analytical pass over a target document.

## Cross-cutting note

Both skills are MIT-spirit adaptations; socratic-grill shares lineage with what
abcd already credited (mattpocock). Attribution for any harvested material is
straightforward — extend `skills/abcd-intent-grill/ACKNOWLEDGEMENTS.md` (for the
grill harvest) and add an ACKNOWLEDGEMENTS for the new foundations-auditor skill
when itd-55 is built.
