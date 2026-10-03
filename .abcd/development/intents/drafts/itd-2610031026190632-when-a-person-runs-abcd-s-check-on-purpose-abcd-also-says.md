---
id: itd-2610031026190632
slug: when-a-person-runs-abcd-s-check-on-purpose-abcd-also-says
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-111]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd's check says when the harness is behind its latest release

## Press Release

> When a person runs abcd's check on purpose, abcd also says whether the harness it runs in is behind that harness's latest release: it reads the installed version from this computer, asks the internet for the newest one only because the person asked, and names the update to run. Nothing leaves the computer unless the person runs the check.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request "Warn the user if the harness isn't on it's latest version (if possible to test)"; to be confirmed at the planning interview._

## Why This Matters

A harness that falls behind its releases misses fixes abcd may rely on, and nothing tells the person: with the harness's own automatic updates switched off, an old version runs until someone notices. abcd already says when its own binary is stale (itd-111, which this builds on); this says the same of the harness it runs in. The facilitator's feasibility test on 2026-10-03, on the product thinker's machine: Claude Code's installed version reads from the command itself (2.1.288), the native installer keeps every downloaded version in a versions folder beside the active one, and the harness's automatic updates were off there. The newest release is not on disk: knowing it means asking the internet.

Typed links: builds on itd-111 (its stance that implicit checks read only the disk and the network answers only an explicit check, adr-38 and brief invariant 7, which decision 2 keeps). The reading of the installed harness version is shared with itd-2610030814013772's version-floor warning (its decision 3), which builds on this draft's reading rather than writing its own.

## Decisions

1. 2026-10-03, the product thinker, confirming the routing (itd-84, hand-run): split. The warning is this draft; the network rule stays with adr-38 and invariant 7; the installed-version reading is this draft's plumbing, shared with the AGENTS.md draft's floor warning.
2. 2026-10-03, the product thinker, asked when abcd may look up the latest harness version (only when asked; once a day at session start, which reverses the standing rule; from this computer only; decide later): only when asked. The warning appears when the person runs abcd's check on purpose; nothing goes out on its own, and adr-38 and invariant 7 stand unchanged. The product thinker accepted the cost shown: a person who never runs the check never sees the warning.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_Facilitator-written, for the reviews and the planning interview._

- Which check carries it: abcd's update check (today it compares abcd's own binary), the setup checks' explicit mode, or both.
- Hosts other than Claude Code: which expose an installed version and a release feed; a host with neither is named as unchecked rather than guessed.
- Whether the disk-only signal the test found (a newer version already downloaded than the one running) is worth saying at session start without the network, as a separate, smaller warning.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
