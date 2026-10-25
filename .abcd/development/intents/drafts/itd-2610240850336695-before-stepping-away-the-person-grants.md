---
id: itd-2610240850336695
slug: before-stepping-away-the-person-grants
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The person grants their sessions what they may do alone, until a time

## Press Release

> Before stepping away, the person grants their sessions exactly what they may do alone (push their own branches, open pull requests, queue a merge that needs no approval, decide as the technical facilitator) until a time they name, and abcd enforces that grant instead of leaving it to a sentence in chat. Alice, a solo founder who is also the product thinker, types one command before the evening: `abcd mode autonomous --until 09:00` with the preset they want. Every session on the machine reads the same grant, so none of them stops to ask whether it may push the branch it just finished, and none of them merges to `main`, cuts a release or force-pushes, because no preset grants those. The status line shows the grant and the time left beside the waiting-on badge, which stays the agent's own report of whose answer is owed. At 09:00 the grant lapses by itself and sessions go back to asking. In the morning `abcd mode back` lists what each session did under the grant, so Alice can see what was pushed and opened in their name.

## Why This Matters

_Written by the agent on 2026-10-24 from the session that filed
iss-2610240827243654 and the product thinker's framing in it; the planning
interview confirms or rewrites it._

On 2026-10-24 the product thinker asked a session to build an issue. The build
was refused, and the steps that would get it started (commit the record on a
branch, run the preflight, push, open a pull request) stopped on the agent's
own rule that a push or pull request needs a standing permission. The product
thinker asked why the session paused when "haven't we set the TF role to
autonomous?", and found that no such setting exists:

- The badge (`managed`, `facilitator`, `product-thinker`) is the only thing
  `/abcd:mode` offers, and it is in practice agent-written: a question's chip
  sets it and the answer resets it.
- The away mode (`abcd mode autonomous`) is the human-set control, but it
  "grants no rights": nothing in `build`, `implement`, `drain` or `landing`
  reads it, and `--hold-pushes` only changes the stop hook's wording.
- The one setting meant to grant something, `facilitator.mode: auto|human`,
  lives in the repository's config and is not built.

So the person's permission has nowhere to live but a sentence in chat, which
one session hears and its peers do not, and which compaction can drop.

The away mode was designed with a narrow push grant and then reversed
(`dcn-2610181003456444`, superseded by `dcn-2610181016510153`), because the
guard cannot fence the person's own interactive sessions. That reasoning treats
a grant as a fence. The state of the art treats them separately: Claude Code's
permission modes, Codex's approval presets and Cline's auto-approve toggles are
grants enforced by the harness, each kept apart from the session's "needs
input" status, and each keeps a short list of irreversible acts gated at every
level. [adr-2610240850193281](../../decisions/adrs/2610240850193281-a-human-set-autonomy-grant-permits.md)
records the decision this intent delivers.

Related: the shipped away mode (itd-2610180912189125) and the checklist draft
itd-2610170638184486, whose press release already names "the technical
facilitator set to auto" as part of a session's checklist.

## Mechanism

_Draft for the planning interview._ We expect sessions to stop pausing on
reversible, in-scope acts because the grant becomes a record every session
reads at start and after compaction, rather than an instruction one session
heard; and we expect no rise in harm because the irreversible acts are outside
every preset and refused by code, not by the agent's judgement.

## Scope Conditions

_Draft for the planning interview._

- One person on one machine, setting the grant for their own sessions of one
  repository (keyed on its root commit).
- Hosts whose hooks run abcd's guard; a host without hooks reads the grant as
  context only.
- The grant is a permission, not a sandbox: it does not claim to stop a
  session that ignores it.

## Acceptance Criteria

_Draft for the planning interview._

- Given the person has set a grant that includes `push-branch` until 09:00,
  When a session pushes a branch it created before 09:00, Then no session asks
  for permission and the push is recorded against the grant.
- Given any grant, When a session tries to merge to the default branch, cut a
  release or force-push, Then abcd refuses it and names the act as outside
  every preset.
- Given a grant whose end has passed, or a grant record that cannot be read,
  When a session reads it, Then it reads as no grant and the session asks as it
  would without one.
- Given a repository's `.abcd/config.json` names a capability, When the grant
  is read, Then the repository can remove that capability from the grant but
  never add one the person did not grant.
- Given a grant is in force, When the status line renders, Then it shows the
  grant's preset and time left beside the waiting-on badge.
- Given the person runs `abcd mode back`, When it prints, Then it lists each
  act a session took under the grant, with the session and the capability.

## Open Questions

- Which capabilities exist, and which presets bundle them?
- Does `decide-as` take `facilitator`, `product-thinker` or both, and what does
  the badge do while the agent wears that hat?
- Where does the planned `facilitator.mode: auto|human` go: folded into the
  grant, or kept as the repository's own opt-in for merging?
- Is the grant per repository (root commit) or per machine?
- Does setting a grant require the person to be away, or can it hold while
  they are present?
- Does this intent absorb part of itd-2610170638184486, or sit beside it?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
