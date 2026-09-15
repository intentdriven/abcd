---
id: itd-173
slug: verification-escalates-from-the-built-in-check-to-an-outside
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Verification escalates from the built-in check to an outside audit, and the product thinker picks the rung

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

How much verification a piece of work deserves is a property of the project, not of the framework. A personal tool where a fault inconveniences only its author needs less than software other people pay for. Fixing the level in the framework gets one of those two wrong.

So verification is a ladder and the product thinker chooses the rung, in the plain terms of what is at stake rather than in the terms of what the machinery does. The rungs run from the built-in check, to two models from the same provider, to models from different providers where more than one is available, to an outside service, to a paid human audit. Evidence supports the shape: a single automated review finds a minority of real problems, and reviews from different model families find different ones, so each rung buys a measurably different thing rather than more of the same.

**Security is the first rung to build out and the least served today.** It is the case where the product thinker most clearly cannot judge what they are being told, where the consequences reach furthest beyond them, and where the framework currently offers no way to escalate at all. A repository can already obtain private security review through its forge, so the early rungs of a security ladder need no new relationship with anyone.

The default stays small. Nothing here obliges a project to climb, and a project that never leaves the bottom rung should not feel broken.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

**Decided 2026-08-29 (product thinker).** The ladder is the answer, not a single fixed level: built-in check by default, two models from one provider where a subscription allows it, different providers where more than one is available, an outside dependency for higher-value work, and a paid audit where the stakes justify it. Security is where the ladder gets built first.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
