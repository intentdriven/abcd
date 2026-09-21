---
id: itd-200
slug: abcd-s-presence-should-be-visible-in-the-host-harness-s-stat
spec_id: spc-70
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-20]
severity: minor
impact: additive
promoted_from: iss-168
origin: extracted-from-record
production_mode: dictated-and-formatted
---

# A managed repository shows it is managed, and whose answer the loop is waiting on, in the host's status line

## Press Release

> **A repository under abcd management now looks managed, and the status
> line says whose answer the agent loop is waiting on.** The line leads with
> one badge in three states: abcd is here and nobody is waiting; the loop is
> parked on the facilitator; the loop is parked on the product thinker. The
> agent sets the state when it stops for a verdict, and the human sets it by
> hand to say which hat they are wearing. Where the host offers no status
> surface, one line at the stop names whose answer is owed, and the bare
> `/abcd` board answers the same question on demand. An unmanaged repository
> shows nothing. The install step offers the line, says why it is worth
> having, and lets the user configure or decline it.

> "I could see the loop was parked on someone who was not me before I read a
> single line of output," said a facilitator who runs the agents for a small
> product team. "The badge was there before the directory and the branch, so
> a narrow window never hid it." A product thinker on the same team never
> sees the line at all: "I was told an answer was owed to me. That is the
> whole point, and the terminal was never going to be where I heard it."

## Why This Matters

The status line is the visible face of abcd. Today a managed repository is
indistinguishable from any other until somebody runs a command, and a stop
that the agent loop parks on the product thinker is invisible to the person
sitting in front of the repository. Under the roles decisions the agents run
unattended and stop only to obtain a verdict, and the product thinker answers
on a surface of their own, so the terminal is exactly where that parked stop
goes unnoticed. One leading badge closes both gaps with one slot.

The record already holds the design, settled in [iss-168](../../../work/issues/resolved/iss-168-abcd-s-presence-should-be-visible-in-the-host-harness-s-stat.md)
by a live demonstration on 2026-08-29: the marker leads the line because the
harness hands the status command no terminal width and the right edge is what
truncation cuts first; the badge is inverse video with its word inside so it
reads without colour; there is no animation because the line re-renders on
events, not on a clock; the presence badge is a setting and the role badges
are fixed; the default is gold on dark grey at a measured 6.94 to 1.

## Decisions (grilled 2026-09-01)

The maintainer resolved these at the planning interview; they are commitments,
not options.

- **Three states, one slot, two writers.** The badge is a mode abcd stores:
  managed and nobody waiting, waiting on the facilitator, waiting on the
  product thinker. The agent writes it when it needs an answer, according to
  whom it is addressing; the human writes it to say which hat they wear. Both
  the status line and the agent read the same state.
- **Managed repositories get abcd's line; the rest keep the user's own.**
  The defaults ship with abcd and move into the user-level home under
  `~/.abcd/` at install time. In an abcd-managed repository they override the
  harness's own status setting; in any other repository the harness's setting
  stands untouched. The user-level setting can switch abcd's line off
  entirely.
- **The install step asks.** `ahoy install` offers the status line, says why
  it is worth having, and takes the basic configuration of its elements as
  part of onboarding.
- **Where the host has no status surface**, one line at the stop names whose
  answer is owed, once, and the bare `/abcd` board shows the same state on
  demand. The board is the fallback, never a separate status command
  ([itd-20](../superseded/itd-20-top-level-abcd-dispatcher.md) owns it).
- **First harness only.** This cut wires the harness abcd already ships
  lifecycle hooks for. Others follow one at a time in the roster order the
  record sets.
- **Word and inverse video carry the meaning.** Colour reinforces; it is never
  the only signal.
- **abcd owns the whole row in a managed repository.** Once installed, the
  line is abcd's own elements, not a prefix on the user's previous status
  command, in this order: the badge, the repository name, the branch, the
  model, context used as a percentage, the five-hour and seven-day usage
  percentages, and the record's counts of intents and issues. The badge is
  fixed and leads; every element after it is one the user can switch off at
  install time or later in the same user-level setting. Elements that come
  from the harness's status payload (model, context, usage) render only when
  the payload carries them, and an absent field drops the element rather than
  showing a placeholder. The branch comes from the repository; the counts
  come from the record, and what they count exactly (open issues; intents not
  yet shipped) is settled in the spec.
- **Out of scope, recorded for a later iteration:** a communication path that
  adapts to the role, where the product thinker answers through a web surface
  and the facilitator stays at the terminal. This intent only makes the parked
  stop visible; it does not carry the answer back.

## What's In Scope

- One canonical render of the line in core, consumed by the status surface
  and by the `/abcd` board, so no front door invents its own words.
- The stored mode with its two writers: a verb the agent calls when it stops
  for a verdict, and the same verb or a setting the human uses by hand.
- Install-time detection of the first harness, the wiring `ahoy install`
  writes, the offer and its explanation, and the basic element configuration.
- The user-level defaults under `~/.abcd/`, the managed-only override, and
  the off switch.
- The one-line notice at the stop where no status surface exists.
- Contrast checking of a configured presence badge against its own
  background, refusing a pair below the bar with the measured ratio.

## What's Out of Scope

- Guard health, peer-session counts, uncommitted ledger counts and any other
  candidate for the line. Each is its own decision once the badge has lived
  in real sessions.
- Update-available signalling: knowing the latest release needs the network,
  which implicit operations never touch.
- Any second harness.
- The role-adapted communication path named in the Decisions.

## Mechanism

We expect a visible mode to make both the agent and the human address the
right person, because the current hat is always on screen.

## Scope Conditions

- Holds only on a harness that renders a status line by invoking a command <!-- cond: cond-2609012158056931 -->
  and displaying its stdout, which is the first harness in the roster.
- Holds only in an abcd-managed repository; elsewhere abcd writes nothing to <!-- cond: cond-2609012158055017 -->
  the line and changes no setting.
- Holds where the badge's word and inverse video are legible; colour is <!-- cond: cond-2609012158057285 -->
  reinforcement and the design does not depend on it.
- Holds under the width the host chooses: the line is designed for <!-- cond: cond-2609012158056783 -->
  truncation, not layout, and only the leading badge is guaranteed to survive.
- Holds for the payload fields the first harness supplies today (model name, <!-- cond: cond-2609012158058166 -->
  context percentage, five-hour and seven-day usage); a harness that stops
  supplying one drops that element rather than breaking the line. The
  maintainer's own 2026-08-29 demonstration wiring is the reference rendering
  for every element except the two record counts, which are new.

## Acceptance Criteria

- **Given** an abcd-managed repository and no parked stop, **when** the host
  renders its status line, **then** the line begins with the presence badge
  in its configured or default colours, and nothing else abcd owns precedes
  it.
- **Given** the agent loop stops to obtain a verdict from the product
  thinker, **when** it records the stop, **then** the next render shows the
  product thinker badge, and the `/abcd` board reports the same state.
- **Given** the agent loop stops to obtain a verdict from the facilitator,
  **when** it records the stop, **then** the next render shows the
  facilitator badge.
- **Given** a facilitator sets the mode by hand, **when** the line next
  renders, **then** it shows the state they set, and the agent reads the same
  state.
- **Given** a repository that is not abcd-managed, **when** the host renders
  its status line, **then** abcd contributes nothing and the harness's own
  setting is untouched.
- **Given** the user has switched abcd's line off in their user-level
  settings, **when** a managed repository renders its status line, **then**
  abcd contributes nothing.
- **Given** a harness with no status surface, **when** the loop parks a stop,
  **then** exactly one line names whose answer is owed, and the `/abcd`
  board shows the state afterwards.
- **Given** `ahoy install` on the first harness, **when** it reaches the
  status line step, **then** it offers the line, states why it is worth
  having, lets the user switch each element after the badge on or off, and
  writes no wiring if the user declines.
- **Given** the line is installed with every element on, **when** the host
  renders it with a payload carrying model, context and usage figures,
  **then** the row reads, in order, the badge, the repository name, the
  branch, the model, the context percentage, the five-hour and seven-day
  usage percentages, and the intent and issue counts, each separated the same
  way.
- **Given** the payload lacks a field an element needs, **when** the line
  renders, **then** that element is absent and no placeholder takes its
  place.
- **Given** a configured presence badge whose foreground and background fall
  below the contrast bar, **when** the setting is read, **then** it is
  refused with the measured ratio and the default renders instead.
- **Given** a terminal narrower than the line, **when** the host truncates
  it, **then** the badge is what survives.

## Open Questions

1. The name and shape of the verb the agent calls to record a stop, and
   whether the human's hand-set mode uses the same verb or a setting. Spec
   detail.
2. Where the stored mode lives so that both the status command and the agent
   read it without a harness variable: repository-local state or the
   user-level home. Spec detail, constrained by the managed-only rule.
3. The roles decisions this intent refines live on the design branch under
   ids that collide with main's; they take their final ids at merge, and this
   record's links are re-pointed then.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-636b39d5541a -->
Fidelity review — receipt rcp-636b39d5541a (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:db617047ff021296dd6f1d0ef5c664a924b816e5f2213ec8e300de58c8d296e6 · prompt_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5
Input attestations: diff:worktree feat/statusline-badge at eda67ccc, uncommitted: git diff HEAD plus untracked files (sorted, concatenated)@sha256:4c90ae144ab9b04a6b43f5eeade18c67d8e03a79da439feb290dda78dee943c7; rubric:.abcd/.work.local/reviews/rcp-636b39d5541a.request.md (the host supplied no policy hashes; rubric_hash is this file's digest)@sha256:db617047ff021296dd6f1d0ef5c664a924b816e5f2213ec8e300de58c8d296e6; prompt:agents/intent-auditor.md (prompt_hash is this file's digest)@sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Render walks a fixed order whose first key is the badge, the badge is painted in the settings' presence pair (default #f0c052 on #444444), and the status verb prints Row.String() with nothing before it; a live managed render opened with the badge's SGR run
  evidence: internal/core/statusline/render.go:166 — "The badge leads, and nothing abcd owns precedes it (ac-1)"
  evidence: internal/core/statusline/badge.go:114 — "Element{Key: KeyPresence, Plain: word, Rendered: paint(pair, " "+word+" ")}"
  evidence: internal/core/statusline/render_test.go:280 — "TestRenderedBadgeCarriesItsColourPair"
  evidence: internal/surface/cli/statusline.go:107 — "fmt.Fprintln(w, res.Row.String())"
- ac-2 — MET_WITH_CONCERNS: `abcd mode product-thinker` writes .abcd/.work.local/mode, Compose reads the same file for the next render, and the bare board's presence line and --json statusline.state report it (live: badge `waiting: product thinker`, board `presence: waiting: product thinker`); concern: nothing makes the loop record the stop — the only instruction is commands/mode.md, and no hook, rule domain or agent prompt directs an agent that stops for a verdict to call the verb
  evidence: internal/surface/cli/mode.go:74 — "if err := mode.Set(cwd, st); err != nil {"
  evidence: internal/core/statusline/compose.go:68 — "state, err := mode.ReadAt(root)"
  evidence: internal/surface/cli/board_presence_test.go:16 — "TestBoardPresenceLineIsTheRendersPlainForm"
  evidence: commands/mode.md:29 — "**The agent, at a stop.** When you stop to obtain a verdict, record whom you are addressing"
  evidence: hooks/hooks.json:1 — "(no hook names `abcd mode`; grep count 0)"
- ac-3 — MET_WITH_CONCERNS: `abcd mode facilitator` stores the state and the next render shows the fixed facilitator badge (`waiting: facilitator`, #1c1f26 on #e8e8e8), verified live and by the verb test; same concern as ac-2 — recording the stop is an instruction on the command page, not a wired behaviour
  evidence: internal/core/statusline/badge.go:56 — "StateFacilitator: "waiting: facilitator","
  evidence: internal/core/statusline/badge.go:77 — "StateFacilitator: {Foreground: "#1c1f26", Background: "#e8e8e8"},"
  evidence: internal/surface/cli/statusline_test.go:26 — "TestStatuslineManagedRendersTheRowBadgeFirst"
  evidence: commands/mode.md:37 — "Use `facilitator` when the verdict is the facilitator's to give."
- ac-4 — MET: The human's hand-set state goes through the same verb and file; the test sets each state, reads it back with bare `abcd mode` and through Compose, and the live run showed `mode` printing `facilitator` after the set and the render carrying it
  evidence: internal/surface/cli/mode_test.go:97 — "TestModeSetsTheStateBothWritersRead"
  evidence: internal/surface/cli/mode.go:62 — "st, err := mode.Read(cwd)"
  evidence: commands/mode.md:44 — "**The human, by hand.** `/abcd:mode facilitator` or `/abcd:mode product-thinker`"
- ac-5 — MET_WITH_CONCERNS: Outside a managed checkout the verb runs the recorded previous command with the same stdin and passes its streams and exit code through, so abcd contributes nothing to the line (live: `PREV:<payload>`); concern: the harness's own statusLine setting is not literally untouched — install rewrites the single harness-wide setting to abcd's command, and 'untouched' is honoured behaviourally by replaying the recorded previous command, the divergence the spec signed off
  evidence: internal/surface/cli/statusline.go:95 — "if !managed || set.Disabled {"
  evidence: internal/surface/cli/statusline.go:157 — "sh := exec.Command("sh", "-c", previous)"
  evidence: internal/surface/cli/statusline_test.go:76 — "TestStatuslineUnmanagedRunsThePreviousCommand"
  evidence: internal/core/ahoy/statusline_apply.go:10 — "< harness-home>/settings.json, where `statusLine` is pointed at `'<entry>' statusline`"
  evidence: internal/surface/cli/board.go:47 — "if err != nil || !ahoy.Managed(root) {"
- ac-6 — MET: The user-level `disabled` switch makes a managed checkout take the same branch as an unmanaged one — the previous command runs and abcd's row does not (live: `PREV:` output in the managed repo with disabled=true)
  evidence: internal/core/statusline/settings.go:79 — "Disabled bool `json:"disabled"`"
  evidence: internal/surface/cli/statusline.go:95 — "if !managed || set.Disabled {"
  evidence: internal/surface/cli/statusline_test.go:102 — "TestStatuslineDisabledBehavesAsUnmanaged"
- ac-7 — MET_WITH_CONCERNS: With no ~/.abcd/statusline.json (or disabled set) the set form prints exactly one line naming the addressee and the board shows the state afterwards (live: `abcd: waiting on the product thinker — an answer is owed`, then `presence: waiting: product thinker`); concern: 'a harness with no status surface' is inferred from the user-level setting's absence or off switch, never from the harness itself, and the line reaches the human only if the agent relays the --json `notice` field as commands/mode.md instructs
  evidence: internal/surface/cli/mode.go:78 — "if !statusSurfaceInstalled() {"
  evidence: internal/surface/cli/mode.go:121 — "return err == nil && set.Installed && !set.Disabled"
  evidence: internal/surface/cli/mode.go:131 — "return "abcd: waiting on the " + who + " — an answer is owed""
  evidence: internal/surface/cli/mode_test.go:147 — "TestModeSetPrintsOneLineWhereNoSurfaceExists"
  evidence: internal/surface/cli/board.go:58 — "set.Disabled = false"
- ac-8 — MET_WITH_CONCERNS: On consent the install shows a one-paragraph reason ending in the question, then one on/off prompt per element after the badge (eight, default on), and writes ~/.abcd/statusline.json (0600, previous_command recorded) plus the harness statusLine; a decline writes nothing and is re-offered (live transcript and tests); concern: the offer sits behind a generic category gate `Apply status-line changes? [y/N]` asked BEFORE the reason, so a user who declines at that gate never hears why the line is worth having, and --yes skips the offer entirely (reported under optional_skipped)
  evidence: internal/core/ahoy/statusline_apply.go:36 — "const statusLineOfferQuestion = "In an abcd-managed repository the host's status line becomes abcd's own row"
  evidence: internal/core/ahoy/statusline_apply.go:99 — "if !a.prompter.Confirm(statusLineOfferQuestion) {"
  evidence: internal/core/ahoy/statusline_apply.go:107 — "ans := a.prompter.Prompt(elementPromptPrefix+string(k), []string{"on", "off"}, "on")"
  evidence: internal/core/ahoy/statusline_install_test.go:242 — "TestStatusLineOfferDeclinedWritesNothing"
  evidence: internal/core/ahoy/statusline_install_test.go:296 — "TestStatusLineConsentWiresBothFiles"
  evidence: internal/core/ahoy/apply.go:1604 — "StatusLine,"
- ac-9 — MET: The fixed order is badge, repo, branch, model, context, five_hour, seven_day, intents, issues joined by one separator; the full-row test pins nine elements and eight separators, and a live render against this worktree's record read `abcd · sl-badge · feat/statusline-badge · Opus · ctx 8% · 5h 24% · 7d 41% · itd 134 · iss 385`, the counts matching the intent board's drafts+planned (73+61) and the open ledger (385)
  evidence: internal/core/statusline/render.go:62 — "var order = []ElementKey{"
  evidence: internal/core/statusline/render.go:37 — "const Separator = " · ""
  evidence: internal/core/statusline/render_test.go:35 — "TestRenderFullRow"
  evidence: internal/core/statusline/compose.go:267 — "return v.Buckets[intent.BucketDrafts] + v.Buckets[intent.BucketPlanned], nil"
- ac-10 — MET: Payload percentages are pointers and the model a string, so absence survives the parse; Render skips an element whose text is empty and places separators only between survivors; the field-by-field test and a live payload lacking context and seven_day (row keys presence, repo, branch, model, five_hour) show no placeholder
  evidence: internal/core/statusline/render.go:178 — "if text == "" {"
  evidence: internal/core/statusline/payload.go:36 — "ContextPct *float64 `json:"context_pct,omitempty"`"
  evidence: internal/core/statusline/render_test.go:87 — "TestRenderDropsAnAbsentPayloadField"
  evidence: internal/core/statusline/payload_test.go:62 — "TestParsePayloadTreatsEveryFieldAsOptional"
- ac-11 — MET: LoadFrom measures the configured pair against the 4.5:1 bar at read time, refuses with the measured ratio and substitutes the default; live: `REFUSED the presence colours in ~/.abcd/statusline.json — it measures 2.82:1, below the 4.50:1 bar; the default #f0c052 on #444444 renders instead` on stderr and the default badge on stdout
  evidence: internal/core/statusline/settings.go:277 — "if ratio < ContrastBar {"
  evidence: internal/core/statusline/settings.go:279 — ""it measures "+FormatRatio(ratio)+", below the "+FormatRatio(ContrastBar)+" bar","
  evidence: internal/core/statusline/contrast.go:37 — "const ContrastBar = 4.5"
  evidence: internal/core/statusline/settings_test.go:139 — "TestLoadRefusesAPairBelowTheBar"
- ac-12 — MET: The package does no truncation of its own because the harness supplies no width; the badge is element one and every later element optional, and the test asserts over every rune prefix of the plain row (and the rendered row's opening) for all three states and four switch sets that what survives a cut is the badge or a prefix of it
  evidence: internal/core/statusline/render.go:16 — "TRUNCATION SAFETY COMES FROM ORDER ALONE."
  evidence: internal/core/statusline/render_test.go:342 — "TestEveryPrefixOfTheRowBeginsWithTheBadge"
  evidence: internal/core/statusline/badge.go:90 — "const reset = "\x1b[0m""

Gap audit:
- honoured:
  - Three states, one slot, two writers: a stored mode the agent and the human set through one verb, read by the render and the board
    evidence: internal/core/mode/mode.go:49 — "type State string"
    evidence: internal/surface/cli/mode.go:74 — "if err := mode.Set(cwd, st); err != nil {"
    evidence: commands/mode.md:27 — "## Set the state — two writers, one verb"
  - One canonical render in core consumed by the status verb and the /abcd board, so no front door invents its own words
    evidence: internal/core/statusline/render.go:5 — "consumed by two front doors — the status verb the harness invokes and the bare `/abcd` board"
    evidence: internal/surface/cli/board.go:59 — "res, err := statusline.Compose(root, statusline.Payload{}, set)"
    evidence: internal/surface/cli/statusline.go:99 — "res, err := statusline.Compose(root, payload, set)"
  - Defaults ship with the binary and are written under ~/.abcd/ at install; the user-level setting carries the off switch and per-element switches
    evidence: internal/core/statusline/settings.go:112 — "//go:embed defaults/statusline.json"
    evidence: internal/core/ahoy/statusline_apply.go:285 — "set := statusline.Defaults()"
    evidence: internal/core/statusline/defaults/statusline.json:3 — ""disabled": false,"
  - Managed repositories get abcd's line; every other repository keeps the user's previous line, replayed with the same payload
    evidence: internal/surface/cli/statusline.go:94 — "managed := rootErr == nil && ahoy.Managed(root)"
    evidence: internal/core/ahoy/managed.go:28 — "func Managed(root string) bool {"
  - The install step asks: offer, reason, element switches, wiring only on consent, nothing on decline; ahoy uninstall restores the previous command
    evidence: internal/core/ahoy/statusline_apply.go:70 — "offerStatusLine is ac-8: the reason and the question, the element switches, then the two writes — and nothing at all on a decline"
    evidence: internal/core/ahoy/statusline_install_test.go:667 — "TestUninstallRestoresTheStatusLine"
    evidence: internal/core/ahoy/statusline_apply.go:226 — "uninstallStatusLine is the uninstall half"
  - Where the host has no status surface, one line at the stop names whose answer is owed and the bare /abcd board shows the state on demand
    evidence: internal/surface/cli/mode.go:126 — "answerOwedNotice is the one line the set form prints where there is no status surface"
    evidence: internal/surface/cli/cli.go:249 — "fmt.Fprintf(w, " presence: %s\n", board.Statusline.Plain)"
  - First harness only: the wiring targets the harness whose settings live at $CLAUDE_CONFIG_DIR or ~/.claude/settings.json
    evidence: internal/core/ahoy/statusline_detect.go:34 — "harnessHomeEnv = "CLAUDE_CONFIG_DIR""
    evidence: internal/core/ahoy/statusline_detect.go:36 — "harnessHomeDir = ".claude""
  - Word and inverse video carry the meaning; colour reinforces and is never the only signal
    evidence: internal/core/statusline/render_test.go:253 — "TestBadgeMeaningSurvivesColourRemoval"
    evidence: internal/core/statusline/badge.go:11 — "WORD AND INVERSE VIDEO CARRY THE MEANING; COLOUR ONLY REINFORCES."
  - abcd owns the whole row in the fixed order; payload elements drop when absent; the counts are open issues and drafts plus planned intents read through the board's own readers
    evidence: internal/core/statusline/render.go:56 — "order is the fixed element order itd-200's commitments set"
    evidence: internal/core/statusline/compose.go:253 — "res, err := capture.List(capture.ListRequest{RepoRoot: root, State: capture.StateOpen})"
    evidence: internal/core/statusline/compose.go:267 — "return v.Buckets[intent.BucketDrafts] + v.Buckets[intent.BucketPlanned], nil"
  - The presence pair is a setting checked against the contrast bar with the measured ratio; the role pairs are fixed constants held to the same bar
    evidence: internal/core/statusline/settings.go:269 — "func mergePresence(out *Settings, over Pair) []string {"
    evidence: internal/core/statusline/contrast_test.go:131 — "TestRoleBadgePairsClearTheBar"
  - The status and mode verbs make no network request; the mode store is per worktree, gitignored, and refuses outside the local tier
    evidence: internal/surface/cli/statusline_test.go:249 — "TestModeAndStatuslineTouchNoNetwork"
    evidence: internal/core/mode/store.go:122 — "fi, err := root.Lstat(TierRelPath)"
    evidence: internal/core/ahoy/local_tier_test.go:103 — "TestLocalTierFenceCoversTheModeFile"
  - Both front doors are wired and documented: CLI verbs mode and statusline, plugin page commands/mode.md, the board and ahoy pages, the CLI reference, the install how-to, the brief's surface chapter and the release surface
    evidence: internal/surface/cli/cli.go:261 — "root.AddCommand(newModeCommand(&asJSON))"
    evidence: internal/surface/cli/surfaceparity_test.go:44 — ""statusline": "harness-invoked status-line render, wired by `ahoy install`"
    evidence: docs/reference/cli/commands.md:798 — "### `abcd mode`"
    evidence: docs/how-to/install.md:144 — "## The status line"
    evidence: .abcd/development/release/surface.json:1060 — ""path": "abcd mode","
- diverged:
  - In any other repository the harness's setting stands untouched — delivered as: the single harness-wide statusLine is rewritten once to abcd's command at install, and the previous command is recorded and replayed with the same stdin so the visible line, not the setting, is unchanged (the spec's signed-off shape)
    evidence: internal/core/ahoy/statusline_apply.go:10 — "< harness-home>/settings.json, where `statusLine` is pointed at `'<entry>' statusline`"
    evidence: internal/surface/cli/statusline.go:143 — "runPreviousStatusCommand is the ac-5/ac-6 fallback"
  - The agent sets the state when it stops for a verdict — delivered as an instruction on the /abcd:mode command page only; no hook, rule domain or agent prompt directs a loop that stops for a verdict to call `abcd mode`, so the parked stop is visible only when the agent has read that page
    evidence: commands/mode.md:29 — "**The agent, at a stop.** When you stop to obtain a verdict, record whom you are addressing"
    evidence: hooks/hooks.json:1 — "(no entry names `abcd mode`)"
  - The badge is inverse video with its word inside — delivered as an explicit truecolour foreground/background fill (SGR 38;2 / 48;2) rather than the inverse-video attribute; the word still carries the meaning without colour, but a terminal that ignores 24-bit colour shows an unpainted word rather than a reversed block
    evidence: internal/core/statusline/badge.go:92 — "the rendered form is the word inside a filled block of colour, which is what "inverse video with its word inside it" amounts to once the pair is explicit"
    evidence: internal/core/statusline/badge.go:132 — "return "\x1b[" + fg.sgr(38) + ";" + bg.sgr(48) + "m" + text + reset"
  - The install step states why the line is worth having before the user decides — delivered with a generic category gate (`Apply status-line changes? [y/N]`) asked before the reason paragraph, so a decline at that gate never sees the reason; --yes skips the offer and reports it
    evidence: internal/core/ahoy/apply.go:1604 — "StatusLine,"
    evidence: internal/core/ahoy/statusline_apply.go:64 — "if a.autoYes || !a.approved[StatusLine] || !a.has(StatusLineOfferGapID) {"
  - Where the host has no status surface — delivered as: the user-level setting is absent or disabled; the harness's own capability is never consulted, and a user who switched the line off (ac-6) is treated as having no surface and gets the one-line notice
    evidence: internal/surface/cli/mode.go:113 — "statusSurfaceInstalled reports whether this machine renders abcd's status line: the user-level setting is present and its off switch is not thrown"
    evidence: internal/core/statusline/settings.go:94 — ""the host has no status surface" is this being false, or Disabled being true"
- missing: (none)

Scope-condition dispositions:
- cond-2609012158056931 — survived: The wiring writes a statusLine of type command whose stdout the harness displays, and the verb reads the harness payload on stdin; the live run wired `'<entry>' statusline` into settings.json and rendered from a piped payload
  evidence: internal/core/ahoy/statusline_detect.go:7 — "The status line is the top-level key `statusLine`, an object {"type": "command","
  evidence: internal/core/ahoy/statusline_apply.go:232 — "return shSingleQuote(entry) + " " + statusVerb"
  evidence: internal/surface/cli/statusline.go:123 — "raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1))"
- cond-2609012158055017 — narrowed: Outside a managed checkout abcd writes nothing of its own to the line (the previous command's output passes through, or nothing) and the mode store refuses without the local tier; but the harness's one statusLine setting is changed for every repository at install, so 'changes no setting' holds only behaviourally
  narrowing: holds for the line's content — abcd contributes no output and creates no state outside a managed checkout — but not for the harness setting, which the install rewrites once, harness-wide, to abcd's command; the previous command is recorded and replayed with the same stdin so the visible line is unchanged
  evidence: internal/surface/cli/statusline.go:95 — "if !managed || set.Disabled {"
  evidence: internal/core/mode/store.go:124 — "only a repository abcd manages has that tier, and it is never created on the way to a write"
  evidence: internal/core/ahoy/statusline_apply.go:10 — "< harness-home>/settings.json, where `statusLine` is pointed at `'<entry>' statusline`"
- cond-2609012158057285 — survived: The badge word is the element's plain form, the three words are distinct prose naming the addressee, and a test strips every escape and requires the states still to be told apart; inverse video is realised as an explicit fill rather than the SGR attribute, which does not touch the word's legibility
  evidence: internal/core/statusline/render_test.go:253 — "TestBadgeMeaningSurvivesColourRemoval"
  evidence: internal/core/statusline/badge.go:55 — "var badgeWord = map[State]string{"
- cond-2609012158056783 — survived: The package reads no width and truncates nothing; the badge is element one and every later element optional, and the prefix test holds that any cut leaves the badge or a prefix of it
  evidence: internal/core/statusline/render.go:20 — "package therefore implements no truncation of its own"
  evidence: internal/core/statusline/render_test.go:342 — "TestEveryPrefixOfTheRowBeginsWithTheBadge"
- cond-2609012158058166 — survived: The parser lifts exactly model, context_window.used_percentage and rate_limits.five_hour/seven_day.used_percentage, every level a pointer, ignores the rest of the payload, and an absent field drops its element; a live payload missing context and seven_day rendered five elements and no placeholder
  evidence: internal/core/statusline/payload.go:66 — "UsedPercentage *float64 `json:"used_percentage"`"
  evidence: internal/core/statusline/payload_test.go:62 — "TestParsePayloadTreatsEveryFieldAsOptional"
  evidence: internal/core/statusline/render_test.go:87 — "TestRenderDropsAnAbsentPayloadField"
## Grounds

- pursued: the status bar is the visible face of abcd, so a managed repository should look managed; we expect facilitators to keep the line on after living with it, and if they switch it off in their own settings that shows this was the wrong call
