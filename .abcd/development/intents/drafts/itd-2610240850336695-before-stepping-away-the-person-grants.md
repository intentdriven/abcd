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
- Given a session, When it runs `abcd mode autonomous` itself or writes the
  away store through a shell or a file tool, Then the guard refuses it, records
  the attempt, and the person sees it on their next prompt.
- Given the person runs `abcd mode back`, When it prints, Then it lists each
  act a session took under the grant, with the session and the capability.

## Decisions

1. **Grant and fence are two axes (product thinker, 2026-10-24).** The grant
   is adopted as a permission with no boundary claimed. Where abcd fences, the
   runner's agents under itd-2610161621364075, the grant is embedded in that
   fence; the person's own sessions get the grant alone. Recorded in
   adr-2610240850193281.
2. **The capability set is four (product thinker, 2026-10-24).**
   `push-branch` (push a branch the session created), `open-pr` (open a pull
   request from it), `queue-merge` (queue a pull request whose ruleset needs no
   approval) and `decide-as` (answer its own questions as a named role).
   Merging to the default branch outside the queue, cutting a release and
   force-pushing are outside every preset. Which presets bundle them is still
   open.
3. **`decide-as` takes either role or both (product thinker, 2026-10-24).**
   The person names `facilitator`, `product-thinker` or both on each grant; a
   session answers only the roles named, recording each answer as a decision,
   and a question for a role not named waits for the person.
4. **The default preset is "ship work", once outstanding design work is
   confirmed (product thinker, 2026-10-24).** A grant that names no preset
   carries `push-branch`, `open-pr` and `decide-as facilitator`. It takes
   effect only after the pre-run blocker sweep of itd-2610170638184486 has put
   the outstanding design work to the person, so a session is not left alone
   with a product decision nobody made.

Consequence for itd-2610170638184486: its parked work making the away mode the
person's act alone (the guard refusing an agent's own grant, and the refusal of
shell and file-tool writes to the away store) was parked "should a later design
give it rights worth guarding". This intent gives it those rights.

5. **That person-only work moves into this intent (product thinker,
   2026-10-24).** The grant ships with what protects it: the guard refuses an
   agent's own `abcd mode autonomous` (the binary on PATH, the plugin root's,
   and routes through `sh -c` and a script a shell runs), shell and file-tool
   writes to the away store are refused, the PreToolUse matcher gains `Edit`,
   `Write`, `MultiEdit` and `NotebookEdit`, and refused attempts are recorded
   and shown to the person. itd-2610170638184486 drops it from its remaining
   work when this intent is planned.
6. **The planned facilitator setting folds into the grant (product thinker,
   2026-10-24).** The grant is the one place that lets abcd merge on its own:
   `queue-merge` replaces `facilitator.mode: auto` (adr-2610150603484802,
   itd-97, itd-2610150602547271). A repository keeps only the power to
   narrow: it may withhold `queue-merge`, never grant it. Planning reconciles
   those records with this one.
7. **A grant covers one repository (product thinker, 2026-10-24).** It is
   keyed on the repository's root commit, as the away record is, so every
   worktree and clone of it on this machine shares it; another repository
   needs its own grant.
8. **A grant holds whether the person is present or away (product thinker,
   2026-10-24).** Being away stays its own record, with its question reroute
   and stop check; the grant is a separate record beside it, so a person at
   the keyboard can grant `push-branch` and stop being asked. Ending the away
   mode does not end the grant; the grant ends at its own time or when the
   person revokes it. The verb's shape (a flag on `abcd mode autonomous` or a
   verb of its own) is planning's to settle, and the press release's example
   follows it.

## Open Questions

- Which presets exist beyond the default? The interview offered three
  ("nothing", "ship work" and "everything"); planning confirms the set and
  their names.
- The verb's shape: a flag on `abcd mode autonomous`, or a verb of its own now
  that the grant holds while the person is present (Decision 8).
- The Mechanism, Scope Conditions and Acceptance Criteria above are the
  agent's drafts and are not yet confirmed.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
