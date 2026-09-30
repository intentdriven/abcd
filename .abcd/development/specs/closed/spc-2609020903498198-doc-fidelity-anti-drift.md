---
id: spc-2609020903498198
slug: doc-fidelity-anti-drift
intent: itd-60
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The brief describes every surface that ships, or the intent does not reach shipped

## Summary

The spec delivers itd-60: a doc-fidelity gate over the brief, standing at two
moments and running in two layers.

The two moments are `abcd spec close` — the verb that moves a planned intent to
`shipped/`, run in the change that lands the work — and the release cut, which
runs the same judgement over every intent shipped since the last cut. Belt and
braces: the first catches a change while whoever made it still has the surface in
their head; the second catches whatever reached `main` around it.

The two layers are a deterministic coverage floor and a semantic pass. The
deterministic layer needs no oracle: every verb, every sub-verb and every agent
the binary ships must have a brief chapter naming it, and a surface with no
chapter refuses the move on its own. The semantic layer is host-delegated: it
reads what the change delivered against the chapter that claims to describe it
and refuses on a confirmed false sentence, naming the sentence; where no verdict
can be obtained it fails closed.

The scope is **brief only**. A public-doc sentence that lags is reported and
refuses nothing. When the gate finds the brief lagging, the pass drafts the brief
edit **and applies it** in the shipping change, flagging it for the maintainer to
read afterwards.

## Scope

- **The gate in core.** `internal/core/docfidelity` (name provisional) composes
  one judgement from three inputs: the command tree and agent set, the brief's
  surface chapters, and the delivery of the change under judgement. It returns a
  structured verdict — per-surface coverage rows, per-sentence semantic findings,
  and a single refuse/allow — and writes nothing. The two front doors format it;
  neither invents a word.
- **Layer 1, the coverage floor.** Derived from the same command tree that
  produces `docs/reference/cli/commands.md` and
  `.abcd/development/release/surface.json`, plus the agent set the plugin ships.
  Every verb, sub-verb and agent must be named by a chapter under the brief's
  `04-surfaces/`. Missing coverage is a refusal on its own authority — no oracle
  call, no wait, no degraded mode.
- **Layer 2, the semantic pass.** A host-delegated reviewer receives the chapter
  and what the change delivered, and returns a verdict per sentence it judges
  false, each carrying a pointer at the divergence. Only a **confirmed** false
  sentence refuses; an inconclusive verdict is not a pass. The reviewer's
  transport is the delegated-review posture already used for the intent-fidelity
  reviewer (itd-80): a deterministic shell, a thin delegated core, fail-closed on
  absence.
- **Enforcement point one: `abcd spec close`.** The close hook that ships the
  linked intent gains the gate. A refusal names the surface and the lagging
  sentence, and neither the spec nor the intent moves.
- **Enforcement point two: the release cut.** `abcd launch ship` runs the same
  judgement over every intent that reached `shipped/` since the last tag, and
  refuses the cut naming the intent and the sentence.
- **Draft and apply.** On a refusal the pass composes the brief edit and writes
  it into the working tree of the shipping change, then records a review flag the
  maintainer reads. The change proceeds; the edit is visible in its diff. The
  flag is the honesty: the brief may carry a sentence the maintainer did not
  write until they read it.
- **The per-task report.** After a task the same core runs in report mode:
  findings, evidence, no refusal, no exit code of its own.
- **The legitimate lead.** A brief chapter describing a surface the binary ships
  is current even when the edit post-dates the last cut. The gate judges the
  brief against the **binary**, never against the last tag, so the lead cannot
  read as drift.

## Approach

The core is a pure judgement: inputs in, verdict out, no writes and no network of
its own. The delegated call is behind an interface with a fail-closed default, so
a test can substitute a reviewer that refuses, one that confirms, and one that is
absent, and assert all three outcomes without a host.

The two enforcement points share one entry point, called with a different
population: one intent for the close hook, every intent since the tag for the
cut. That is what makes "belt and braces" one mechanism rather than two
implementations that can disagree.

Draft-and-apply is deliberately the last step and deliberately separate from the
judgement: the judging code returns a verdict and a proposed edit; a distinct
writer applies it and records the flag. A judge that writes is a judge whose
output depends on who ran it.

Layer 1 stands in front of layer 2 in the same run, and its refusal short-circuits
the delegated call — an undocumented surface needs no reviewer to be judged, and
paying for one would make the cheap half hostage to the expensive half.

## How the Acceptance Criteria are satisfied

- **ac-1 (undocumented surface refuses deterministically).** Layer 1 walks the
  command tree and the agent set against the surfaces index; a surface no chapter
  names is a refusal composed before the delegated call is constructed. The test
  asserts a refusal with the reviewer interface set to a stub that panics if
  called.
- **ac-2 (false sentence refuses the shipped move).** The close hook calls the
  gate with the intent's delivery; a reviewer fixture returning one confirmed
  false sentence makes the hook refuse, and the test asserts the spec is still in
  `open/` and the intent still in `planned/`.
- **ac-3 (fail closed).** The same hook with a reviewer fixture that errors, times
  out, or returns an inconclusive verdict refuses. Three separate cases, because
  three different absences have historically been collapsed into one pass.
- **ac-4 (the cut refuses).** The release path calls the gate over the intents
  shipped since the tag; a fixture with one lagging chapter refuses the cut and
  names the intent and the sentence.
- **ac-5 (draft and apply, flag after).** On a refusal the writer applies the
  proposed edit to the brief in the working tree and records the review flag; the
  test asserts the chapter changed, the flag exists, and no path exists in which
  the change is completed with the brief lagging and no flag recorded.
- **ac-6 (public docs report, never refuse).** A fixture whose public-doc sentence
  lags and whose brief does not: the gate reports the finding and allows both the
  move and the cut. The test asserts the finding is present **and** the verdict is
  allow — the pair, so the check cannot pass by simply not looking.
- **ac-7 (the per-task pass reports).** Report mode over the same fixtures yields
  the same findings and never a refusal.
- **ac-8 (the legitimate lead).** A fixture whose brief edit post-dates the tag
  and whose chapter matches the binary: no finding, at either point. The test pins
  that the judgement reads the binary and never the tag.

- **ac-9 (autonomous run).** One flag on the gate's verb (the spec names it
  `--autonomous`) makes an unattended run execute both layers without waiting
  for a person: the deterministic layer refuses on its own; the host-run
  reviewer is invoked by the routine the way the changelog composer is; a
  lagging chapter is drafted and applied in the run's own change; every applied
  edit is listed in the run's output for review after. The refusals are the
  same as attended: an undocumented surface, a confirmed false sentence, or
  a reviewer that is unavailable (the flag never turns fail-closed into pass).

## Tests

- Layer 1 table over command-tree fixtures: a new verb, a new sub-verb, a new
  agent, each with and without a chapter; the covered cases pass and the uncovered
  refuse, with the reviewer stub asserting it was never constructed.
- Layer 2 reviewer fixtures: confirmed false sentence, no finding, inconclusive,
  transport error, timeout. The last three refuse.
- Close-hook tests: refusal leaves both records unmoved; allow moves both.
- Release-cut tests over a population of shipped intents, including the
  empty-population case.
- Draft-and-apply tests: the edit lands in the tree, the flag is recorded, and the
  writer is never reached on an allow.
- Report-mode tests: identical findings, no refusal, exit 0.
- Legitimate-lead test: a brief ahead of the tag is not a finding.
- A boundary test asserting the core writes nothing and makes no request; the
  writer is the only path that touches the tree.

## Out of scope

- **The public docs as a gated surface.** Reported here, never refusing. Grading
  the brief against the public docs, drafting audience-adapted deltas, and the
  end-user / developer-extending split are a later rung.
- **The reverse direction** (a human brief edit drawing out its implied intents):
  itd-61.
- **Re-grading delivery against intention:** the intent-fidelity reviewer's job;
  this consumes its notion of delivery.
- **The docs-currency lint that already shipped from this draft** (`abcd docs
  lint`): the floor this gate stands on, unchanged and not re-claimed.
- **Authoring the brief's prose voice or its information architecture.**
- **Whether this pass becomes a framework-provided discipline** — the one open
  question the interview left open, and one that changes where the pass is
  configured rather than what this spec builds.
