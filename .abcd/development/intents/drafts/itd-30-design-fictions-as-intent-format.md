---
id: itd-30
slug: design-fictions-as-intent-format
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-05-07, from: bundle-member, to: standalone, reason: "Originally bundled with itd-27 (grill sub-verb) under `intent-capture-discipline`, but itd-27 and itd-30 are not co-scheduled — bundle members must belong to the same phase. Reclassified to standalone; when this lands, its epic depends on or extends spc-3 (the grill sub-verb's epic) for shared interview/lint/persona-registry plumbing." }
blocked_by: [itd-94]
builds_on: [itd-1, itd-34]
severity: minor
---

# Design Fictions As An Alternative Capture Format For Intents

## Press Release

> **abcd lets intent authors choose between a press release and a design fiction when capturing a user-facing change.** The default remains the Amazon working-backwards press release — terse, present-tense, persona-quoted. The new alternative is a *design fiction*: a short scenario-based artefact (a diary entry, support transcript, screenshot caption, news clipping, or "day in the life" vignette) that describes the shipped capability through lived experience rather than marketing prose. `/abcd:intent new --format=fiction` opens a fiction-shaped interview ("Whose day are we in? What just happened? What does it feel like five minutes after?"). The same acceptance-criteria gate applies — Given/When/Then bullets are extracted from the fiction, not bypassed by it. The intent file declares its format in frontmatter (`capture_format: press-release | design-fiction`), and `intent-fidelity-reviewer` reads either dialect against the same delivered reality. Domain experts who think in stories rather than headlines now have a first-class lane; the discipline (acceptance criteria, persona realism, scope boundaries) is preserved across both formats.
>
> "I kept staring at the press release template trying to write a punchy headline for something that is, honestly, just a quiet improvement to a Tuesday afternoon," said Iris, product manager. "When I switched to a design fiction — literally writing the support email that *won't* get sent because the friction is gone — the scope clarified in twenty minutes. Same acceptance criteria fell out, but I actually wanted to write it."

## Why This Matters

The press-release format is excellent for capabilities with a clear "moment" — a shippable headline, a feature a user would notice on Monday morning. It is **less natural** for:

1. **Quiet quality-of-life changes** that have no headline ("the loop no longer loses your place when the laptop sleeps") but materially change a workflow.
2. **Multi-touch capabilities** experienced over a session or week, not as a single moment.
3. **Domain-expert intents** authored by people who think narratively (designers, researchers, support leads, healthcare or legal SMEs) rather than in marketing-shaped prose. Forcing a press release on a narrative thinker produces stilted text that satisfies the template but loses the texture that justified the work.
4. **Capabilities defined by the *absence* of friction** — easier to describe what no longer happens than to advertise what now does.

Design fictions — short, plausible, near-future artefacts written from inside a scenario — are the working method already used in service design, speculative design research, and HCI. They are well-suited to capturing user-facing change as *lived experience* rather than *announceable feature*. abcd's intent layer should accommodate both because:

- The press-release format is an opinionated default, not a mandate. Codified abcd principle: discipline (acceptance + persona + scope) is preserved; the *vehicle* for that discipline can vary.
- Inviting design fictions broadens the pool of competent intent authors beyond product-marketing-shaped thinkers.
- A second format gives `intent-fidelity-reviewer` a richer signal to compare against — fictions tend to specify *texture* (timing, emotion, sequence) that press releases skip, exposing drift earlier.

The press-release format ships as the codified default; design fictions are introduced once there are ≥ 5 shipped intents under the press-release format and direct user feedback that the format excluded otherwise-good intent authors.

## What's In Scope

- **Format flag in capture command:** `/abcd:intent new --format=fiction "<free-text>"`. Default remains `--format=press-release` (no behaviour change for existing users).
- **Fiction-shaped interview** in `/abcd:intent new`: prompts shift from "headline + press-release body + customer quote" to:
  - "Whose day are we in? Pick a persona from `personas.json`."
  - "What kind of artefact: diary entry, support transcript that won't get sent, screenshot caption, internal-Slack message, news clipping, calendar invite, post-mortem-that-didn't-happen?"
  - "Set the scene: when, where, what just happened just before."
  - "Show — don't tell — the capability through what the persona does, says, or feels."
  - "What's the after-state five minutes / one hour / one week later?"
- **Frontmatter declares format:** `capture_format: press-release | design-fiction` added to intent template. Defaults to `press-release` for backward compatibility.
- **Same acceptance-criteria gate.** Fictions still require a `## Acceptance Criteria` section with Given/When/Then bullets. The interview extracts these *from* the fiction at capture time (asks "what would have to be true in the product for this scene to happen?") rather than letting the narrative substitute for verifiable criteria. Hard block at `/abcd:intent plan` time, identical to press-release intents (per itd-1).
- **Persona registry shared across formats.** Same `personas.json`; same role-hint biasing; same PII-avoidance rule.
- **`intent-fidelity-reviewer` reads both dialects.** Auditor compares the fiction's *texture* (timing, sequence, after-state) against delivered reality, not just the acceptance bullets. Three-bucket prose audit (honoured / diverged / missing) extends to fiction-specific signals (e.g., "the fiction implied a sub-second response; reality shows 4-second latency → diverged").
- **Format chooser at `new` time only.** No mid-flight format switch. If an author starts in press-release format and realises a fiction would fit better, they `/abcd:intent refine <itd-N> --switch-format=fiction` (or vice versa); the refine flow re-interviews under the new format and rewrites the body, preserving frontmatter, ID, scope, acceptance, and audit notes.
- **README and brief surface doc updated** to describe both formats with a "when to choose which" prose section.
- **Examples shipped:** at least one design-fiction example intent in `examples/` (not `drafts/`) so authors have a concrete reference. Likely one diary-entry fiction and one support-transcript fiction.

## What's Out of Scope

- **Replacing the press-release format.** Press release stays the default. This intent adds a *second* lane, not a *replacement*.
- **Free-form intents with no format constraint.** "Pick fiction or press release" is the choice; "write whatever you want" is not. The discipline is the point.
- **Generating press releases from fictions automatically (or vice versa).** Tempting, but the formats encode different ways of thinking; auto-conversion would lose the texture the format choice was made to capture.
- **Fiction sub-formats as separate templates.** Diary, transcript, screenshot caption, etc., are *prompts* in the interview, not separate file templates. One template per format keeps the surface small.
- **Multi-fiction intents** (e.g., a fiction *and* a press release in the same file). One format per intent.
- **Auto-detecting which format would fit a free-text idea.** The author chooses; abcd doesn't second-guess.
- **Fiction-aware lifecycle states.** drafts/planned/shipped lifecycle is unchanged; format does not affect transitions.
- **Localisation of personas or fiction artefacts.** English-only initially; multilingual personas and fiction conventions are a future question if abcd ships beyond English-speaking users.

## Acceptance Criteria

- **Given** a user runs `/abcd:intent new --format=fiction "<free-text>"`, **when** the interview completes, **then** an intent file is written to `drafts/itd-N-<slug>.md` with `capture_format: design-fiction` in frontmatter, a `## Design Fiction` section (instead of `## Press Release`), a persona attribution, and a populated `## Acceptance Criteria` section with at least one Given/When/Then bullet.
- **Given** a fiction-format intent is missing or has malformed acceptance criteria, **when** the user runs `/abcd:intent plan <itd-N>`, **then** the command refuses to promote with the same error path used for press-release intents (per itd-1) — format does not relax the gate.
- **Given** a press-release intent in `drafts/`, **when** the user runs `/abcd:intent refine <itd-N> --switch-format=fiction`, **then** the body is rewritten in fiction form via the fiction interview, frontmatter `capture_format` updates to `design-fiction`, and the intent's ID, scope, acceptance criteria, and any audit notes are preserved verbatim.
- **Given** a design-fiction intent has shipped and `intent-fidelity-reviewer` runs, **when** the audit completes, **then** the `## Audit Notes` section contains both the per-criterion verdicts (identical schema to press-release intents) and a three-bucket prose audit (honoured / diverged / missing) that addresses the fiction's textural claims (timing, sequence, after-state), not only the acceptance bullets.
- **Given** a domain expert who has never written a press release, **when** they capture an intent using `--format=fiction`, **then** they reach a complete, lint-passing intent file without being asked to write a marketing-shaped headline or a feature-launch sentence.
- **Given** the README and brief surface doc are updated, **when** a new abcd user reads them, **then** they can articulate (a) when to pick press release, (b) when to pick fiction, and (c) what's preserved across both — without consulting the source code.
- **Given** a discipline-kind intent (per itd-34), **when** the user attempts `/abcd:intent new --format=fiction --kind=discipline` (or attempts a refine with `--switch-format=fiction` on an existing discipline), **then** the command refuses with a clear error: disciplines use `## Rule` (not `## Press Release` or `## Design Fiction`); the capture-format dimension applies only to `kind: standalone | bundle-member`. The orthogonality of `kind` (standalone / bundle-member / discipline) and `capture_format` (press-release / design-fiction) is enforced at capture time, not deferred to lint.

## Revisit Triggers (when this intent escalates from drafts/ to planned/)

This intent moves from `drafts/` to `planned/` when ANY of the following happens:

1. **Five or more shipped intents** under the press-release format, providing a baseline to compare against.
2. **First user feedback** that the press-release format excluded an otherwise-good intent author (designer, researcher, SME) who tried and abandoned the capture flow.
3. **First shipped intent** whose `intent-fidelity-reviewer` finding is "diverged on texture not on acceptance" — i.e., the acceptance bullets passed but the *feel* the press release implied did not match. This signals fictions might catch what press releases miss.
4. **A non-engineer collaborator** asks for a less marketing-shaped capture format directly.

## Open Questions

- **Naming of the format flag.** `--format=fiction` (terse) vs `--format=design-fiction` (precise but verbose). Recommend short form with `design-fiction` only in frontmatter and docs.
- **Where shipped fiction examples live.** `examples/intents/` (parallel to `drafts/planned/shipped`) vs `.abcd/development/intents/examples/`. Recommend the latter to keep the registry self-contained.
- **Fiction artefact-type taxonomy.** Should the interview enumerate a fixed set (diary / transcript / screenshot caption / news clipping / calendar invite / post-mortem / Slack thread) or accept free-form? Recommend a fixed-with-Other set, ranked by frequency of use, to give authors a starting point without forcing a category.
- **Acceptance-extraction trust.** When the interview asks "what would have to be true for this scene to happen?", does the LLM-driven extraction reliably produce verifiable Given/When/Then bullets, or does it tend to restate the fiction in pseudo-formal language? Probably the latter on first pass — the prompt for that step needs explicit examples of "good" extracted criteria vs "narrative-restated" pseudo-criteria. Item for spc-N-42 (or successor prompt-quality spec).
- **Auditor reading model.** Press releases are short and quote-shaped; fictions are longer and scene-shaped. Does `intent-fidelity-reviewer` need a separate prompt path for each format, or does one prompt with a "format-aware" preamble suffice? Probably one prompt with conditional preamble; revisit after first three shipped fictions.
- **Interaction with itd-15 (self-dogfooded SOTA audit).** If abcd self-disembarks under the fiction format, does the SOTA audit treat fiction-format intents differently? Recommend no — same audit, same criteria.
- **CLI surface for choosing format on `intent new` without args.** When the user types `/abcd:intent new` with no args, do we ask format-first or free-text-first? Recommend free-text-first (capture the spark), then ask format ("does this feel more like a press release or a scene from someone's day?") — gives the choice context.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Builds on: itd-1 (acceptance gates — gate applies identically across formats), itd-27 (grill sub-verb and glossary — fictions may need their own glossary entry), the brief's `04-surfaces/05-intent.md` (which currently codifies press-release-only and will need updating to describe both lanes).
- Coordinates with: itd-15 (self-dogfooded SOTA audit — must remain format-neutral), `intent-fidelity-reviewer` agent (extends to read fiction texture).

**Implementation note (post-2026-05-07 reclassification):** the spec for this intent MUST extend itd-27's grill machinery — the capture-while-grilling pattern, the persona registry pickup, the lint integration, the glossary surface — rather than duplicating any of it. The standalone reclassification (2026-05-07) does not weaken this dependency; it only changes its shape from bundle-shaped (one shared spec) to dependency-shaped (spec depends on or extends the grill sub-verb's spec — spc-3 of the predecessor store, not the live `spc-3`). itd-30's plan-review must verify this dependency is honoured. If the spec plan duplicates rather than extends, the plan-review fails.
- Methodological precedents: Amazon's working-backwards press release (the existing default); Anthony Dunne and Fiona Raby's *Speculative Everything* (design fiction as a research method); Bruce Sterling's "design fiction" coinage; Julian Bleecker's *Design Fiction: A Short Essay on Design, Science, Fact and Fiction* (Near Future Laboratory); the diegetic-prototype tradition in HCI research; service-design tools like personas + journey maps that already inform `personas.json`.
