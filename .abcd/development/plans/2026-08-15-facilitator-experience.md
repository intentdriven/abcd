# Facilitator experience — usability and docs plan and run queue (2026-08-15)

**Status:** the third of the three forward plans settled at the 2026-08-15
maintainer grill. Continuous rather than a discrete cycle: items here ride
along as the other two plans execute — a docs or wording improvement lands
with the feature it describes, not in a batch at the end. Consumed by the
generic protocol at
[`2026-07-12-abcd-run-protocol.md`](2026-07-12-abcd-run-protocol.md) when run
as bursts. Supersedes the pick-up role of
[`2026-07-29-explainability-follow-up.md`](2026-07-29-explainability-follow-up.md),
whose framing, STOP conditions and persona-lens review it carries forward
(iss-166/167/171 from that queue landed via the install-experience Cut A;
the remainder is absorbed below).

**Admission test.** An item qualifies only if a product thinker or
facilitator **directly invokes or reads it** — a prompt, a summary, a banner,
a status line, a page of documentation. That is the touch side of the grill's
touch-vs-machinery boundary; the machinery that makes behaviour predictable
without human action belongs in
[`2026-08-15-predictable-development.md`](2026-08-15-predictable-development.md).
The ledger stays the backlog of record.

**Framing (carried from the explainability plan, still true).** The tool
explains itself: every choice abcd asks, every result it reports, and its
very presence in a session must be understandable to the persona roster's
non-implementers — the product thinker and the facilitator are the bar, not
the staff engineer. The root cause is structural: core emits bare enum values
and result data, so every front door improvises its own words. The fix puts
canonical plain language in core, rendered identically by every surface.

## Run contract

As the plugin-user-safety plan's, plus the explainability plan's
**persona-lens review** on every diff touching user-facing strings: the
reviewer reads each prompt choice, summary and notice against the persona
registry's roles and must name which persona would fail to understand any
string it blocks on. Auto-merge never inherited; authorised per cycle.

## Workstream A — the explanation layer (absorbed from explainability)

Ordering and item specs unchanged from that plan:

1. **[iss-163](../../work/issues/open/iss-163-the-ahoy-install-config-prompts-are-unexplainable-to-a-first.md)**
   — canonical per-choice help text lives in core; the foundation item.
2. **[iss-164](../../work/issues/open/iss-164-the-ahoy-install-completion-summary-is-written-for-abcd-s-im.md)**
   (blocked by iss-163) — persona-readable result summaries.
3. **[itd-63](../intents/planned/itd-63-setup-wizard-explains-installs.md)**
   — the intent frame A1/A2 deliver into. Lifecycle first: planned but
   spec-less, so the first milestone is the spec and `intent ready`, never
   code.
4. **[itd-20](../intents/planned/itd-20-top-level-abcd-dispatcher.md)** —
   "`/abcd` tells you where you are". Same lifecycle-first rule; last in the
   queue, dropped first when trimming.

## Workstream B — presence and orientation

5. **[itd-112](../intents/planned/itd-112-bare-abcd-opens-with-a-generated-banner.md)**
   — a bare `abcd` opens with a generated object-style banner from the
   canonical identity block. Draft (quoted-text seed): grill before plan.
6. **[iss-168](../../work/issues/resolved/iss-168-abcd-s-presence-should-be-visible-in-the-host-harness-s-stat.md)**
   — abcd's presence in the host status line, under the
   basics-built-in/SOTA-delegated stance; committed prose stays
   host-agnostic per the docs-lint rules.
7. **[iss-216](../../work/issues/open/iss-216-install-instructions-need-a-getting-started-page-once-more-t.md)**
   — a getting-started page and a host version floor, due once more than one
   harness is supported; also the anchor for Workstream D's tutorial.

## Workstream C — interaction polish

8. **[itd-110](../intents/drafts/itd-110-the-grill-interview-renders-with-clear-structure-and-colour.md)**
   — the grill interview renders with structure and colour. Draft seed:
   needs expansion, then grill.
9. **[iss-230](../../work/issues/open/iss-230-one-tap-micro-prompt-channel.md)**
   — the one-tap micro-prompt channel (survey pattern), graduated out of
   itd-111's open questions at the 2026-08-15 grill. Seed: when it matures it
   becomes its own intent, likely absorbing part of A1's prompt seam.

## Workstream D — documentation (continuous)

Docs are part of every item above, not a separate batch; this workstream
names the standing docs work with no code twin:

- A facilitator-oriented **getting-started tutorial** under
  `docs/tutorials/`, anchored on iss-216 — learning-oriented, one Diátaxis
  type, British English.
- **How-to coverage for the day-to-day verbs** a facilitator actually runs
  (`abcd capture`, `abcd rules`, `/abcd:audit`, `/abcd:intent` …) under
  `docs/how-to/` — each page one task, written against the flattened
  `/abcd:<verb>` names.
- Every user-facing behaviour change in the other two plans lands with its
  docs edit in the same PR; a claim docs make that the code does not honour
  is a defect (`abcd docs lint` plus the docs-currency reviewer at release).

## Ordering and collisions

- A1 before A2 and before any prompt-wording work elsewhere: it defines the
  structure the others consume (carried rule).
- B5 (itd-112) reads the canonical identity block; it collides with nothing
  but must not invent a second identity home (`abcd identity` is the seam).
- D's how-to pages wait for the verbs they document to be stable in the
  current cycle — write against what ships, never ahead of it.

## STOP conditions (this plan)

Carried verbatim from the explainability plan: explanations live in core as
data (a front door hard-coding its own copy is a STOP); wording changes that
alter behaviour are a STOP; a persona-lens BLOCK stops the change; scope
creep into the grill-delegation family (iss-165, itd-27/itd-42) is a STOP;
missing or ambiguous record fails closed.

## Explicitly out

Everything machinery (predictable-development plan); everything
installer-safety (plugin-user-safety plan); iss-165 (grill delegation — its
own future cycle, carried exclusion).
