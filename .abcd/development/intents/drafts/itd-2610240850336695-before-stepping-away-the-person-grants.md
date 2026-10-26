---
id: itd-2610240850336695
slug: before-stepping-away-the-person-grants
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610180912189125, itd-2610170638184486]
related_adrs: [adr-2610240850193281]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# The person grants their sessions what they may do alone, until a time

Typed links: `builds_on`
[itd-2610180912189125](../shipped/itd-2610180912189125-a-session-told-to-work-on-while-its.md)
(the shipped away mode, whose record and hooks this grant sits beside, and
whose "or asking a session to" route Decision 14 **reverses**) and
[itd-2610170638184486](itd-2610170638184486-every-abcd-session-keeps-a-checklist-on.md)
(the checklist, whose pre-run blocker sweep asks the person for every undecided
design decision before they leave, and whose parked person-only guard work this
intent takes over, Decision 5); `related_adrs`
[adr-2610240850193281](../../decisions/adrs/2610240850193281-a-human-set-autonomy-grant-permits.md)
(the decision this intent delivers). Merging and cloud sessions are split out
to their own draft (Decision 13).

## Press Release

> **Before stepping away, a person grants their sessions exactly what they may do alone, and abcd holds them to it instead of leaving it to a sentence in chat.** Alice, a solo founder who is also the product thinker, types `/abcd:mode autonomous --away --until 09:00` before the evening. That grants the default "ship work" preset: every session of this repository on Alice's computer may push the branches it made, open their pull requests and answer its own technical questions, without stopping to ask. None of them merges to `main` or cuts a release on the grant alone, and none ever force-pushes, because no preset grants those and abcd's own commands refuse them. The status line shows the grant and its end beside the badge that says whose answer is owed. At 09:00 the grant lapses by itself and sessions go back to asking, and Alice's morning list shows each push and pull request a session made under it. Alice can grant while still at the keyboard too, to stop being asked whether a session may push; and a session told in chat "I'm away, carry on" does not grant itself anything, it names the command.
>
> "I used to come back to sessions that had waited all night to ask whether they could push," said Alice, a solo founder. "Now I say once what they may do, and in the morning I see exactly what they did with it."

## Why This Matters

_Written by the agent on 2026-10-24 from the session in which the product
thinker found the gap; confirmed by the planning interview the same day._

On 2026-10-24 the product thinker asked a session to build an issue. Getting
it started meant committing the issue record on a branch, running the
preflight, pushing and opening a pull request, and the session stopped at the
push on its own rule that a push or pull request needs a standing permission.
The product thinker asked why it paused when "haven't we set the TF role to
autonomous?", and found that no such setting exists:

- The badge (`managed`, `facilitator`, `product-thinker`) is the only thing the
  mode page offers, and in practice the agent writes it: a question's chip sets
  it and the answer resets it.
- The away mode (`abcd mode autonomous`) is the human-set control, but it
  "grants no rights": nothing in `build`, `implement`, `drain` or `landing`
  reads it, and `--hold-pushes` changes only the stop hook's wording.
- The one setting meant to grant something, `facilitator.mode: auto|human`,
  lives in the repository's config and is not built.

So the person's permission has nowhere to live but a sentence in chat, which
one session hears and its peers do not, and which compaction can drop. The
shipped away mode also promises it can be set by "asking a session to", which
lets a sentence in chat stand for the person's act; this intent withdraws that
route (Decision 14), which is why its impact is `breaking`.

The away mode was designed with a narrow push grant and then reversed
(dcn-2610181003456444, then dcn-2610181016510153), because the guard cannot
fence the person's own sessions. That reasoning treats a grant as a fence; the
state of the art keeps them apart (see SOTA).

## Mechanism

_Confirmed by the product thinker in the planning interview, 2026-10-24._

We expect sessions to stop pausing on the acts a grant covers, because the
grant is a record each session reads at its start, after each compaction and
again at each push or pull request it makes, not a sentence one session heard
in chat; and we expect no rise in harm, because the acts a person cannot undo
are in no preset, abcd's own commands refuse them on the grant alone, and the
guard filters the shell routes to them where its hooks run. As one night: the
person runs `/abcd:mode autonomous --away --until 09:00` and leaves. A session
fixing an issue on its own branch pushes it and opens its pull request without
asking; a second session started an hour later, which never heard the person,
reads the grant and does the same; a third, compacted at 23:00, reads it again
and carries on. In the morning there are pull requests waiting, nothing on the
default branch has changed, and a force-push or release a session attempted
through abcd was refused.

The claim is wrong if a session under a grant stops to ask about an act the
grant covers (the record did not reach it), or if a session acting on the grant
alone does something the person cannot easily undo through abcd (the refusals
did not hold).

## Scope Conditions

_Confirmed by the product thinker in the planning interview, 2026-10-24._

- One person on one machine grants their own local sessions of one repository,
  keyed on its root commit. Cloud sessions are outside this intent (Decision
  13): here a cloud session reads no grant.
- The grant holds only where abcd can enforce it: a host whose hooks run abcd's
  guard. Where they do not, the grant does not apply and the session asks as it
  would with no grant (Decision 12).
- The grant is a permission, not a lock: it removes the stop for the acts it
  covers, and does not claim to stop a session that acts without asking, as a
  session can today. The guard is a mistake filter (adr-42); the refusals that
  hold are the ones abcd's own commands make.

## Acceptance Criteria

_Confirmed by the product thinker in the planning interview, 2026-10-24;
rewritten to the adversarial reviews' findings the same day._

- **Given** a grant carrying `push-branch` until 09:00, **when** a session
  pushes a branch this working copy created, before 09:00, **then** it pushes
  without asking for permission, and the morning list names the push, the
  session and the capability.
- **Given** a grant carrying `open-pr`, **when** a session opens a pull request
  from a branch this working copy created, **then** it opens without asking and
  the morning list names it.
- **Given** any grant, **when** a session acting on the grant alone asks abcd to
  merge to the default branch or cut a release, **then** abcd's command refuses
  it and names the act as outside every preset; **and given** the same act
  through a raw shell where abcd's hooks run, **then** the guard refuses it as a
  mistake filter.
- **Given** any grant or none, **when** a session force-pushes, even on the
  person's word in the session, **then** the push is refused (by the forge on
  the default branch, release tags and abcd's own branches; by the pre-push
  hook and the guard on every other branch) and the refusal names the person's
  own `gh` or the forge's web interface as the place to do it.
- **Given** a grant set or revoked while sessions are running, **when** a
  running session next pushes or opens a pull request, **then** it acts on the
  grant as it now stands, not as it stood when the session started.
- **Given** a grant whose end has passed, or a grant record that cannot be
  read, **when** a session reads it, **then** it reads as no grant and the
  session asks as it would without one.
- **Given** a grant set with no `--until`, **when** it is recorded, **then** it
  ends seven days later and the status line names that end.
- **Given** a host where abcd's hooks do not run, **when** a session there would
  act on a grant, **then** the grant reaches it as no grant and it asks.
- **Given** a grant naming `decide-as facilitator` only, **when** a session meets
  a product question, **then** it waits for the person and records no answer of
  its own; **and given** a technical question, **then** it answers, recording
  the answer as a decision.
- **Given** a grant naming the `nothing` preset, **when** a session pushes,
  **then** it asks; **given** `everything`, **then** it pushes, opens pull
  requests and answers both roles' questions; **given** a list of capabilities,
  **then** only those apply.
- **Given** the repository's committed `.abcd/config.json` withholds a
  capability, **when** the grant is read, **then** the effective grant omits it,
  and a capability the repository names that the person did not grant is
  ignored; an uncommitted change to that file widens nothing.
- **Given** a grant in this repository, **when** a session in a sibling
  worktree of it reads the grant, **then** the grant applies there, and a
  session in another repository reads none.
- **Given** a grant set while the person is present, **when** they later end
  the absence with `abcd mode back`, **then** the grant still holds until its
  own end or until it is revoked.
- **Given** a grant is in force, **when** the status line renders, **then** it
  shows the grant's preset and end beside the waiting-on badge.
- **Given** a session, **when** it runs `abcd mode autonomous` itself by any
  route (the binary on PATH, the plugin root's, `go run ./cmd/abcd`, a built
  `bin/abcd-<goos>-<arch>`, through `sh -c` or a script), or writes the grant
  record or the away store through a shell or a file tool, **then** the guard
  refuses it, records the attempt, and the person sees it on their next prompt.
- **Given** the person writes "I'm away, carry on" in chat, **when** a session
  reads it, **then** it records no grant and no absence and names the command.
- **Given** the person reads the morning list, **when** it prints, **then** it
  lists each act a session took under the grant, with the session, the
  capability and the time.

## Prior Art

- itd-2610180912189125 (shipped): the away mode, record and hooks; this intent
  sits beside it and withdraws its "asking a session to" route.
- itd-2610170638184486 (draft): the checklist and its pre-run blocker sweep,
  which asks the person for every undecided design decision before they leave;
  its parked person-only guard work moves here.
- itd-2610161621364075 (planned): the operating-system fence for the agents
  abcd's runner launches; the grant is embedded in that fence where it exists.
- itd-148 (planned) and iss-2610240859590333: a session working in its own
  worktree, which the grant's "a branch this working copy created" relies on.
- adr-42: the guard is a mistake filter, not a boundary.

## SOTA

- Claude Code permission modes (default, acceptEdits, plan, auto,
  bypassPermissions) with allow, ask and deny rules: mature; the mode is kept
  apart from the session's "needs input" state, and a short list stays
  prompted in every mode.
- OpenAI Codex CLI: a sandbox axis and an approval axis, with named presets;
  mature.
- Cline auto-approve: one toggle per capability; mature.
- GitHub Copilot coding agent: may push only to its own branch and can never
  approve or merge; mature.
- None of them expires a grant on a clock.

**Path:** 3, bespoke, because the grant is abcd's own record read by abcd's own
hooks and commands across the person's sessions of one repository, which none
of the tools above holds; the shape follows them (capabilities with presets, a
status kept apart from the grant, irreversible acts outside every level).

## Decisions

1. **Grant and fence are two axes (product thinker, 2026-10-24).** The grant
   is adopted as a permission with no boundary claimed. Where abcd fences, the
   runner's agents under itd-2610161621364075, the grant is embedded in that
   fence; the person's own sessions get the grant alone. Recorded in
   adr-2610240850193281.
2. **The capability set (product thinker, 2026-10-24).** `push-branch` (push a
   branch this working copy created), `open-pr` (open a pull request from it)
   and `decide-as` (answer its own questions as a named role). A fourth,
   `queue-merge`, moved to the merging draft by Decision 13. Merging to the
   default branch, cutting a release and force-pushing are in no preset.
3. **`decide-as` takes either role or both (product thinker, 2026-10-24).**
   The person names `facilitator`, `product-thinker` or both on each grant; a
   session answers only the roles named, recording each answer as a decision,
   and a question for a role not named waits for the person. It never widens
   what invariant 19 lets a machine take alone.
4. **The default preset is "ship work" (product thinker, 2026-10-24).** A grant
   that names no preset carries `push-branch`, `open-pr` and `decide-as
   facilitator`. Its first wording made it wait on the checklist's blocker
   sweep; Decision 16 amends that.
5. **The person-only work moves into this intent (product thinker,
   2026-10-24).** The grant ships with what protects it: the guard refuses a
   session's own `abcd mode autonomous` by every route (the binary on PATH, the
   plugin root's, `go run ./cmd/abcd`, a built `bin/abcd-<goos>-<arch>`, and
   routes through `sh -c` and a script a shell runs); shell and file-tool writes
   to the grant record and the away store are refused; the PreToolUse matcher
   gains `Edit`, `Write`, `MultiEdit` and `NotebookEdit`; refused attempts are
   recorded and shown to the person. It also takes the two live checks
   dcn-2610181003456444 required: that the person's `!` command reaches no
   pre-tool hook, and that a message relayed from another session starting with
   `!` is not run as a shell command. itd-2610170638184486 drops this from its
   remaining work.
6. **The facilitator-setting fold (product thinker, 2026-10-24).** Moved to the
   merging draft by Decision 13.
7. **A grant covers one repository (product thinker, 2026-10-24).** It is
   keyed on the repository's root commit, as the away record is, so every
   worktree and clone of it on this machine shares it; another repository
   needs its own grant.
8. **A grant holds whether the person is present or away (product thinker,
   2026-10-24).** Being away stays its own record, with its question reroute
   and stop check; the grant is a separate record beside it. Ending the
   absence does not end the grant; the grant ends at its own time or when the
   person revokes it.
9. **Three presets ship (product thinker, 2026-10-24).** `nothing` (sessions
   keep asking), `ship-work` (the default) and `everything` (every capability
   this intent carries, `decide-as` both roles). A grant may instead list
   capabilities one by one.
10. **`abcd mode autonomous` becomes the grant; being away is a flag on it
    (technical facilitator, 2026-10-24).** `abcd mode autonomous` with no
    `--grant` grants `ship-work`; `--grant <preset|capability,...>` names
    another; `--until` ends it; `--away` also records the absence. `abcd mode
    back` ends the absence only.
11. **Cloud sessions are autonomous by default (product thinker,
    2026-10-24).** Moved to the merging draft by Decision 13, where it meets
    invariant 23 and Decision 12; here a cloud session reads no grant.
12. **No enforcement, no grant (product thinker, 2026-10-24).** Where abcd
    cannot enforce the grant (no hooks, so no guard), the grant does not apply:
    the session reads it as absent and asks. The grant reaches a session only
    through abcd's hook output, never as plain text a session reads for
    itself, so a host without hooks never sees it as permission.
13. **The split (product thinker, 2026-10-24, after the two adversarial
    reviews).** This intent keeps the grant, its protection and the refusals.
    `queue-merge`, the facilitator-setting fold and the cloud default move to
    a draft of their own, with the findings they carry: the inversion of
    ruling AM1 (abcd arms a merge only where the ruleset requires a person's
    approval), the reversal of adr-2610150603484802, and invariant 23.
14. **The grant is set only by the person's explicit command, never from a
    sentence in chat (product thinker, 2026-10-24).** The person runs
    `/abcd:mode autonomous` (or `abcd mode autonomous` at a terminal); a
    session asked in prose ("I'm away, carry on") refuses and names the
    command. This withdraws the shipped away mode's "or asking a session to"
    route, so the impact is `breaking`. How the guard tells the person's slash
    command from a session's own call is the spec's to settle, starting from
    the live checks of Decision 5.
15. **A grant with no end falls back to a cap (product thinker, 2026-10-24).**
    `--until` is optional; a grant that names no end lasts seven days at most,
    the away mode's existing cap, and the status line shows the end either way.
16. **The grant works without the blocker sweep (product thinker,
    2026-10-24).** The default preset takes effect when the person sets it; it
    does not wait for itd-2610170638184486's sweep. When the sweep ships, it
    runs before an away grant takes effect, so the person is asked every
    undecided design decision first; until then the person decides that for
    themselves.
17. **The refusals apply to a session acting on the grant alone (recorded
    without a question: one defensible answer).** A release the person asks
    for in the session, or a cut the release orchestrator runs on the person's
    instruction, is the person's act, not the grant's, and is not refused. The
    record settles it: AGENTS.md has sessions cut releases on the person's
    word, and refusing that would refuse the person. Decision 18 narrows this
    for force-pushes.
18. **A session never force-pushes, even on the person's word (product
    thinker, 2026-10-24).** Force-pushing is refused whatever the grant and
    whatever the person says in the session; the person runs it themselves in
    their own terminal. Merging to the default branch and cutting a release
    stay as Decision 17 says.
19. **The force-push stop is deterministic where the forge holds it, and the
    gap until then is recorded (product thinker, 2026-10-24).** A session and
    abcd must never force-push; the person may, through `gh` or the forge's
    web interface. The deterministic form is a forge rule: a ruleset refusing
    non-fast-forward pushes on every branch, with a bypass for the person's
    admin role. It holds only once sessions push under a credential of their
    own, without that role (itd-2610170634394993, its Decision 5), because
    while a session pushes under the person's own login the bypass lets it
    through and the login can lift the rule. Today the forge already refuses a
    force-push to the default branch, release tags and abcd's own reservation
    and status branches, for everyone. On every other branch this intent ships
    with the stop that runs on this machine (the committed pre-push hook
    refusing a non-fast-forward push, and the guard refusing `--no-verify` and
    a changed hooks path), which is a mistake filter, not a guarantee. The gap
    closes when the credential split lands; the all-branch ruleset is carried
    in itd-2610170634394993.
20. **The grant joins the autonomous-agents bundle (product thinker,
    2026-10-24).** It is planned and shipped as one bundle with
    itd-2610170634394993 (sessions' own weaker credential, and coordination
    through the remote) and itd-2610170638184486 (the checklist), the bundle
    the product thinker named on 2026-10-17 as the next big release. So the
    force-push gap of Decision 19 never ships: the all-branch ruleset lands
    with the credential split, in the same release as the grant. This
    supersedes Decision 19's "ships with the stop that runs on this machine".
    The checklist's 2026-10-17 ruling that the person says they are away "or
    set it in the app" is narrowed by Decision 14, the same person's later
    ruling: a sentence in chat sets nothing.
21. **A grant holds only in a fenced session (product thinker, 2026-10-24,
    after the bundle's design review).** The person's own sessions run as the
    person's operating-system user, so they can read the person's forge login
    and any key that mints a session's own credential; a weaker credential
    alone stops nothing there. A grant therefore takes effect only in a
    session that runs inside an operating-system fence keeping the person's
    login and the issuing key out of reach: the fence of itd-2610161621364075
    for the agents abcd's runner launches, and, for the person's own
    interactive sessions, the host's own sandbox, which abcd never switches on
    (it does not edit host settings) but detects and reports, as that intent's
    part B names. A session abcd cannot confirm is fenced reads no grant and
    asks, as Decision 12 has it for a host without hooks.
22. **The bundle ships as one release (product thinker, 2026-10-24).** The
    bundle review proposed two releases; the product thinker kept one, so
    every member, the dependency lines and the model budget included, lands
    in the same cut. The phone page and the shared and public options are not
    members: they were split out of the checklist on 2026-10-22 into
    itd-2610222029228127, which the question put to the product thinker
    wrongly counted in.

## Open Questions

- The spelling of revoking the grant without ending the absence (the spec's to
  name).
- How the guard tells the person's `/abcd:mode autonomous` from a session's own
  call (Decision 14), once the live checks of Decision 5 have run.
- Planning is by the bundle form, all three members at once (Decision 20), so
  it waits until itd-2610170634394993 and itd-2610170638184486 are plannable;
  both were reviewed on 2026-10-24 and their findings are not yet applied.
- Decision 21 depends on the fence for the person's own sessions, which
  itd-2610161621364075 names as its part B, a draft not yet filed.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
