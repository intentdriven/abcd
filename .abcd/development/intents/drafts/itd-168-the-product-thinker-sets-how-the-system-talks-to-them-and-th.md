---
id: itd-168
slug: the-product-thinker-sets-how-the-system-talks-to-them-and-th
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

## Open Questions

**Decided 2026-08-29 (product thinker).** The register belongs to the product thinker and is changeable by them at any time, without asking the facilitator.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
