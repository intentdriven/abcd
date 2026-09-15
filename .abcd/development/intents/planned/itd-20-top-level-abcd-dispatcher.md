---
id: itd-20
slug: top-level-abcd-dispatcher
spec_id: null
kind: bundle-member
bundle: spc-83-operator-surfaces
suggested_kind: null
reclassification_history: []
prd_path: null
prd_grandfathered: true
grandfathered: true
grandfathered_at_phase: phase-5-roundtrip
glossary_terms_used:
  - core/lifeboat
  - core/oracle
  - core/disembark
  - core/phase
  - core/intent
  - core/spec
  - core/brief
  - distribution/end-user
  - distribution/release
  - interview/embark
severity: minor
builds_on: [itd-4]
---

# `/abcd` Tells You Where You Are

## Press Release

> **abcd ships a top-level `/abcd` command that shows status and quick actions.** Type `/abcd` (no subcommand) and abcd reports: current voyage, visibility, lifeboat status, dev-sync staleness, last disembark, suggested next actions. It's the at-a-glance command for "what's the state of my abcd setup right now?" without having to remember which subcommand to invoke.
>
> "I'd open a voyage I hadn't touched in weeks and have no idea where I'd left off with abcd," said Iris, product manager. "The `/abcd` command is a one-line status board. Tells me dev-sync hasn't run in 3 days, no recent disembark, and a planned intent ready to pick up. Perfect re-orientation."

## Why This Matters

abcd ships six commands (ahoy, disembark, embark, launch, intent, capture). Every command's bare invocation is status+help only (per the universal abcd convention) — but each shows status scoped to its own command's surface (ahoy shows install state, disembark shows last lifeboat pack timestamp, etc.). None of them gives a *cross-command* "tell me what's happening across the whole project" answer. Users coming back to a project after time away have to either invoke every command individually or inspect filesystem state manually to figure out where they left off.

The pattern was already prototyped in `~/.claude/commands/abcd.md` (the user-level command that did `/abcd status` etc.). Lifting it to be a plugin command makes status-checking discoverable and uniform.

## What's In Scope

- `/abcd` (no subcommand) shows status: project name, visibility, lifeboat presence + age, dev-sync last run, recent commands, planned/active intents, suggested next actions
- `/abcd status` as explicit alias
- `/abcd help` summarising the four main verbs + this status command + `/abcd:intent` family
- Output is markdown with light formatting (tables for state, bullets for suggestions)

## What's Out of Scope

- `/abcd init`, `/abcd init-project` (those map to `/abcd:ahoy` and the itd-21 intent)
- Cross-project status (only current cwd's project)
- Interactive status dashboard (this is one-shot output, not a TUI)

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-managed repo, **when** the user runs `/abcd` (no subcommand), **then** the output includes (in this order): project name + visibility, lifeboat presence + age, dev-sync last-run timestamp, recent commands (last 5 from logbook), planned/active intents (from `intents/planned/` and any in-flight specs), and a "suggested next actions" bullet list — all in markdown with light table formatting.
- **Given** a project where `dev-sync` hasn't run in > N hours (configurable threshold), **when** `/abcd` runs, **then** the dev-sync status is highlighted (e.g., "stale: 3 days") AND "Run dev-sync" appears in suggested next actions.
- **Given** a project with no lifeboat ever produced, **when** `/abcd` runs, **then** the lifeboat status reads "never disembarked" AND "Run /abcd:disembark when ready" is among suggested next actions.
- **Given** a project with planned intents in `intents/planned/`, **when** `/abcd` runs, **then** each planned intent is listed with its title, its phase, and `spec_id` AND an "in flight on spec spc-N" annotation if the linked spec shows active status.
- **Given** the user runs `/abcd help`, **when** the output renders, **then** it summarises the four main verbs (ahoy, disembark, embark, launch) and the meta-development surfaces (`/abcd:intent`, `/abcd:capture`) plus the `/abcd` status command — single-page, scannable.
- **Given** the user is OUTSIDE an abcd-managed repo (no `.abcd/` directory), **when** they run `/abcd`, **then** the output gracefully reports "no abcd config in this directory" and suggests either `cd` to an abcd repo or `/abcd:ahoy` to install — does not error.

## Open Questions

- ~~How does this interact with `/abcd:intent list` — overlap or complementary?~~ Resolved 2026-05-08 by the bare-command-as-render discipline (see `02-constraints/04-naming.md`): `/abcd:intent list` is forbidden (collapses to bare `/abcd:intent`). The question becomes: does `/abcd` (this dispatcher's bare) overlap with bare `/abcd:intent`? Answer: complementary — `/abcd` is the cross-verb status (project / lifeboat / dev-sync / recent commands / planned intents); bare `/abcd:intent` is the intent-corpus render (drafts/planned/shipped/disciplines/superseded grouped). Dispatcher links to bare `/abcd:intent` for drilldown.
- Should "suggested next actions" be opinionated (specific recommendations) or open-ended ("you might want to disembark; you might want to start an intent")?
- How does the user discover this exists — README, ahoy summary, both?

## Audit Notes

_Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Implementation notes (spc-83.2)

- **Dev-sync staleness is a v1 terminal known-state stub.** The dev-sync
  implementation exposes migration logic (`abcd dev-sync work`), NOT a durable last-run
  timestamp; no config field or history-store record captures when dev-sync
  last ran. Section (3) of the `/abcd` board therefore renders `no dev-sync
  record` permanently in v1 — the staleness signal does not function until a
  state substrate exists. This is a recorded terminal state, not a shipped
  capability. Adding the substrate is out of scope. Full record in the
  surface doc: [`../../brief/04-surfaces/08-abcd.md`](../../brief/04-surfaces/08-abcd.md).
- **No spc-17 stub to replace.** spc-17 shipped bare/probe renders for the
  *sub-verb* surfaces only; the top-level `commands/abcd.md` never existed.
  This task creates it fresh — the "stub replacement" premise is
  not-applicable (verified against `git log`).
- **SD001 alias rationale:** `status` / `help` are positional aliases routing
  to the identical bare render (no distinct behaviour), which satisfies SD001
  rather than tripping it — recorded in the surface doc against the SD001
  clause.
- **`/abcd help` covered via the alias (v1 scope).** The draft AC had `/abcd
  help` render a distinct single-page summary of the four main verbs plus the
  meta-development surfaces. V1 makes `help` a byte-identical POSITIONAL ALIAS
  of the bare status board (bare-as-render IS the help), so the where-am-i board
  — which already surfaces the active surfaces and suggested next actions — is
  the help output. A separate verb-summary render would be a forbidden SD001
  sub-verb; folding help into the bare render is the deliberate v1 resolution,
  fully covering the `/abcd help` requirement without a divergent surface.
- **Bundle-member linkage (spc-83 four-intents-one-spec).** This intent is
  authored as a standalone press release but ships as one of FOUR intents
  sharing spec `spc-83-operator-surfaces-manifest-lockstep`. abcd's data model
  represents "N intents, one spec" as a bundle (`kind: bundle-member` + shared
  `bundle: spc-83-operator-surfaces`), which the doc_fidelity intent-resolution
  and the spec-close preflight require — so this intent carries that linkage. It
  is a bundle member by delivery relationship, not a change to its standalone
  press-release scope.
