---
id: itd-168
slug: the-product-thinker-sets-how-the-system
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# The product thinker sets how the system talks to them, and the defaults are plain language and options to choose from

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The roles page promises the product thinker a moment of reading the verdict on whether the why was delivered. The verdict that exists states a per-criterion acceptance judgement with a `file:line` evidence pointer behind every claim. It is the right artefact for a facilitator and unreadable for the role it is promised to.

The register is therefore a property of the artefact, and it belongs to the reader rather than the writer: a product thinker who cannot follow the answers they are given must be able to change how they are addressed without asking anyone. It is a configured profile with defaults that hold until its owner changes them.

The defaults are plain language, no filler, at least one concrete example drawn from the product being built rather than from the framework, trade offs stated concretely with their options named, and selection over composition. Parts of this are mechanically checkable: the presence of an example, the absence of location pointers and bare internal identifiers, and a bounded vocabulary. Because the profile is configured, the check reads the profile rather than a constant.

The addressee is shown, not merely stated. A question marks whose it is with a label, and reinforces it with colour where colour is available: one for the facilitator, another for the product thinker, so a reader can tell at a glance whether a question is theirs to answer without parsing the sentence.

Colour is never the only signal. This repository already honours the convention that asks a program to emit no colour, so the label carries the meaning on its own and colour only strengthens it. A design where the roles are distinguishable only by hue fails for anyone reading without colour, and fails silently, which is the worst way to fail.

Accessibility is a property of the defaults rather than a setting someone must discover. Fixed defaults only work if they work for everyone who gets them, and a person who cannot separate the two hues will not necessarily know that a setting exists to ask for. Three requirements follow.

The roles differ in more than hue. Each carries its own word and its own glyph, so the distinction survives a monochrome terminal, a screen reader, a log file, and a colour scheme that renders two hues alike. Red and green are never the pair, since red-green deficiency is the commonest form by a wide margin, and a terminal's sixteen colours are mapped by the reader's own theme rather than by abcd, so two hues chosen as distinct in one theme can arrive nearly identical in another. That is the argument for shape and word carrying the load and colour reinforcing it.

Where abcd owns the pixels rather than borrowing them, it owns the contrast too. On a surface abcd renders, the pair is drawn from a palette designed for colour vision deficiency rather than picked by eye, and each is held to a contrast ratio that meets the accessibility standard against its background.

An override exists for the person who still cannot tell them apart, and it is an accommodation rather than a style preference, which is why it is theirs to set even though the defaults are not.

The pair is specified by value and by measured contrast, not by colour name, because a name lets a later change reintroduce a failure nobody sees. The facilitator badge is white on a dark blue and reaches 6.45 to 1; the product thinker badge is black on an amber and reaches 11.38 to 1. Both clear the 4.5 to 1 bar for body text. A brighter blue was tried first and rejected at 3.55 to 1 with white, a failure invisible by eye and obvious once computed.

Inverse video is not how the badge is drawn, though it is the obvious way. Inverting hands the text colour to whatever the reader's theme uses as a background, which makes the contrast unknowable to the author and different for every reader. Setting both colours explicitly is what makes the ratio a property of the design rather than of the terminal.

The two badges differ in polarity as well as in hue: light text on a dark fill against dark text on a light fill. That is what carries the distinction into greyscale, a monochrome terminal, and every form of colour vision deficiency, and it satisfies the differ-in-more-than-hue requirement by construction rather than by adding a glyph on top of a colour.

At a conjectural question, one whose answer is a judgement the reader owns rather than a fact the record settles, the surface widens the space and stays out of it. At most three options, the null answer always present, and nothing recommended: no starred default, no recommended label, no ordering by the tool's own preference. That is the committed principle `widen-options-never-recommend`, and a product-thinker surface is where it bites hardest, because that reader has the least means to notice a thumb on the scale.

The absence of a recommendation is stated rather than merely practised. A reader handed three options and no steer will infer one from ordering or from length unless told there is none to infer, so the surface says plainly that it is not recommending and that the judgement is theirs. Silence about a preference is not the same as declaring there is none, and only the second defuses the anchoring.

The marker belongs only where the question is conjectural. Where the record settles the answer, a fact, an existing decision, a lint result, stating it plainly is reporting, and attaching a not-recommending disclaimer to it would manufacture false balance, which the principle's own bounds name as the failure to avoid. A surface that disclaims everywhere teaches its reader to skip the disclaimer.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- Mechanically checkable plain-language rules exist in two public standards:
  GOV.UK ("Try to split up sentences that are over 25 words long", no more than
  five sentences a paragraph, specialist terms explained on first use) and the US
  federal guidance (write for a specific audience, present tense, no hidden
  verbs). ISO 24495-1:2023 is the anchor for "plain language documents" and
  applies to technical writing too; its four principle names were not readable on
  any page opened.
- Readability formulas are not gates: four formulas rated the flawed survey
  question easier than its improved version with a success rate "below 50
  percent" (Lenzner, 2014), and of eight metrics "most correlate poorly with human
  judgments, including … FKGL" (Cachola et al., 2025). Check surface rules, not a
  score.
- Contrast: text needs "at least 4.5:1", values "should not be rounded (e.g.,
  4.499:1 would not meet the 4.5:1 threshold)", and "Color is not used as the only
  visual means of conveying information" (W3C). The badge figures stated here
  clear that where abcd owns the pixels; on a sixteen-colour terminal the slots
  carry no standard values, so the ratio is certified against a named palette and
  the role word is the primary cue.
- The palette authors differ on the pair: Okabe and Ito avoid red with green and
  say "Show difference BOTH in color and shape"; Tol says "pure red and green can
  be distinguished. Instead, yellow/green, blue/violet and … cyan/pink/grey
  combinations can be problematic". The chosen dark blue and amber sit close to
  Tol's high-contrast blue and yellow, which he rates colour-blind safe and
  monochrome safe; the polarity difference is his lightness-separation rule.
  Polarity on its own is a weak cue (the dark-mode evidence is about reading
  speed, not recognition).
- Stating that nothing is recommended is justified by the mechanism of the default
  effect: defaults work partly by "conveying what the choice architect thinks"
  (Jachimowicz et al., 2019), which an unstated absence of preference leaves the
  reader to infer.
- The host's screen-reader mode ("No color-only cues") is the nearest shipped
  precedent for a role label that survives without colour.

**Path 2**: a configured profile whose defaults are the GOV.UK and federal
surface rules (an example present, no record ids or `file:line` pointers in the
product thinker's layer, sentence length, present tense, bounded vocabulary),
checked mechanically with the profile as the seam a later standard replaces; the
badges by value where abcd draws, by role word plus polarity everywhere; the
non-recommendation statement at conjectural questions only. No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** The register belongs to the product thinker and is changeable by them at any time, without asking the facilitator.

- (SOTA 2026-10-23) Which named reference palette the badge ratios are certified
  against where the terminal, not abcd, maps the sixteen colours.
- (SOTA 2026-10-23) Whether the register check records which standard each rule
  came from, so a disputed rule can be traced to GOV.UK, the federal guidance or
  ISO 24495-1.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
