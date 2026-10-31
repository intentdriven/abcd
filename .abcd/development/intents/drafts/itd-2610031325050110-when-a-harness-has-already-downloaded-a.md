---
id: itd-2610031325050110
slug: when-a-harness-has-already-downloaded-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031026190632]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd notices a harness update downloaded but not yet running

## Press Release

> When a harness has already downloaded a newer version of itself that a restart would start, abcd says so, reading only this computer: the person learns a restart is pending without abcd asking the internet anything.

_Proposed by the facilitator on filing (2026-10-03): the product thinker asked, at itd-2610031026190632's interview (its decision 6), that this signal be filed as its own draft; to be confirmed at its own planning interview._

## Why This Matters

A harness can download an update while a session runs; the update starts only when the harness is restarted. On the facilitator's test machine the native installer keeps each downloaded version in a versions folder beside the one the launcher points at, so a newer folder than the running one would mean a restart is pending. No such case was seen on 2026-10-03: the running version was the newest download. Both reviews of itd-2610031026190632 found this a second user moment (the running session, not the installed harness) with no evidence yet that it occurs, so it waits here until a case is seen.

Typed links: builds on itd-2610031026190632 (the offline harness line in abcd's update check) and its installed-version reading.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).

- **The phenomenon is by design, and the harness's notice of it is partial**:
  the harness documents that updates "download and install in the background,
  then take effect the next time you start", and that it "shows a restart
  prompt on success" only for the opt-in package-manager auto-upgrade on
  Homebrew or WinGet; for a native install, the case this draft observed, the
  setup page documents no in-session notice of a successful update, only a
  footer for a failed one (the harness's setup page). The state this draft
  waits to see exists; the harness tells the person about it in the session
  only on the opt-in shape.
- **The versions folder the facilitator saw is documented; when the launcher
  moves is not**: the native installer manages the launcher as a symlink into
  the versions folder, keeps the two newest versions beside the one the
  launcher points at and any a session is running, and deletes the rest. The
  docs imply, without stating it, that the launcher is repointed when the
  update installs; on that reading a newer folder than the launcher's target
  is not a documented intermediate state, and after an update only the
  running session is older.
- **Other tools surface "restart pending" from inside the running program**:
  a browser shows a relaunch option in its About page when an update is
  pending and nothing otherwise.
- **The harness's doctor command records the last update attempt**, the
  vendor's own disk-readable signal of recent update activity.
- Reading the running session's own version is what decision 8 of the parent
  ruled out, and no documented hook input or environment carries it.

**Path 1 — leave the notice to the harness** until a session-version signal
is documented: the harness prompts only on the opt-in package-manager
upgrade and documents no notice of success for a native install, but abcd
has nothing better to say, since the disk comparison has no documented window
in which it differs and the session comparison needs a read the parent
deliberately excluded. The draft's condition, "waits here until a case is
seen", stands, with the addition that the case may never be visible from
disk. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Does any documented hook input or environment variable
  carry the running session's version? If one appears, the comparison against
  the installed version becomes readable and this draft can be planned.
- (SOTA 2026-10-23) The harness shows a restart prompt only on the opt-in
  package-manager upgrade and documents none for a native install; the parent
  accepted a second place only for a warning the harness does not give. Is a
  line of abcd's own worth its cost on the native shape, should a disk signal
  for it ever be documented?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
