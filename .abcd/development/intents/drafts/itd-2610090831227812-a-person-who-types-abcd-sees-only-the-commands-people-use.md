---
id: itd-2610090831227812
slug: a-person-who-types-abcd-sees-only-the-commands-people-use
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609212130136102, itd-2609212113220149]
related_adrs: [adr-40]
---

# A person who types `/abcd:` sees the commands people use

## Press Release

> A person who types `/abcd:` sees the commands people use. The ones only agents run, such as the guard that judges a shell command or the loop that steps an autonomous run, are off the person's list, so the list a person scans reads as their own toolbox and an agent still finds every page it needs, by name, from the help line that names it.
>
> "I used to scroll past twenty entries I would never type to reach the three I use," said Iris, a product thinker. "Now the menu is mine, and the agent has not lost anything: when it needs the guard, it still runs the guard."

## Why This Matters

_Proposed by the drafting reviews; not yet confirmed by the product thinker._

`/abcd:` lists thirty-five commands. Thirteen carry `block: people`, seventeen carry `block: agents`, and five carry no block at all. A person scanning that menu reads every entry to find the handful they type, and seventeen of the entries are pages an agent runs from a hook, a loop or a reviewer's instruction, never from a person's keyboard. `abcd --help` already solved this for the terminal (itd-146): the default list is the person's, one line says `--agent` expands it, and every verb runs the same whichever block lists it. The slash menu is the surface a person actually meets inside the host, and it is the one surface that still shows the two audiences as one undifferentiated list.

## Mechanism

_Proposed by the drafting reviews; not yet confirmed by the product thinker._

We expect a person to find their command faster and to stop opening agent pages by mistake, because the person's menu drops from thirty-five entries to the people block's thirteen, the same sorting itd-146 made for `abcd --help`, and because an agent reaches a page by its name from the help line that names it (`(read commands/<page>.md)`), never by scanning the menu. Shown wrong if session transcripts after it ships show a person typing an agent page, or an agent failing to find a page the help names.

## Scope Conditions

_Proposed by the drafting reviews; not yet confirmed by the product thinker._

- Holds for a host whose plugin loader registers every markdown file under `commands/` as a slash command and maps a subdirectory to an extra namespace segment (`04-surfaces/README.md`, "The command files"). A host without that loader re-decides the mechanism.
- Holds while itd-146's two-block classification stands and each binary-backed page's `block:` frontmatter is the one source of its class, gated by `TestCommandPagesDeclareTheirBlock`.
- Holds for the pages with no block: their class is ruled by the audit iss-2610090831317531, since no tree placement can be checked for them.
- Holds while abcd ships zero skills; the `/abcd:` namespace is commands only (`05-internals/08-skills.md`), so a page moved off the person's list stays a command.

## What's In Scope

- The person's `/abcd:` menu lists the pages whose `block:` is `people`, plus whichever of the unblocked pages the audit classes as a person's.
- Every page whose `block:` is `agents` stays a command, keeps its page and its invocation, and is reached by an agent from the agents-block help line that names it; `abcd --help --agent` stays the one place both lists render.
- The command-line help already sorts every verb into For people and For agents and hosts (itd-146); the plugin's command list keeps that same classification, and the page's `block:` stays the one source of it.
- A test holds the plugin menu to the classification, the way `TestCommandPagesDeclareTheirBlock` holds the pages to the tree.
- The verb audit's table (iss-2610090831317531) is the first section of this intent's spec. It starts from the recorded `block:` frontmatter and rules what that leaves open: the four pages with no block the record has not ruled (`abcd`, `consult`, `ingest`, `prepare-this-repo`; `version` is ruled blockless on 2026-09-25 as a deprecated stub, and `drain` sits with the agents until it runs end to end, ruling BX1 of 2026-09-29), the pages either side could claim (`report`, `ideate`, `dashboard`), and the keep/merge/rename/retire column no record holds.

## What's Out of Scope

- Renaming or retiring any verb or page. The audit proposes a keep/merge/rename/retire column; each rename or retirement is its own intent under adr-40's pre-1.0 clean-break rule (no alias) and the retire-the-name principle.
- Moving any page to `skills/`. abcd ships zero skills and the `/abcd:` namespace is commands only (`05-internals/08-skills.md`).
- Deleting any page. Every verb stays reachable from both the CLI and the plugin surface (AGENTS.md, Boundaries); hiding or namespacing a page keeps it on the plugin surface.

## Prior Art

- itd-146 (shipped): the two-block help behind one flag, every verb running the same whichever block lists it; this intent carries the same classification to the plugin surface and adds nothing to the CLI.
- itd-2609212130136102 (shipped): the consolidation that brought the person's list to its ceiling; its AC5 count is the CLI's, and this intent needs the plugin-side counterpart.
- itd-2609212113220149 (shipped): one sentence per verb, byte-identical on the list, `--help` and the page; a page moved off the person's menu keeps its sentence.
- `.abcd/work/DECISIONS.md`, 2026-09-25 "Help placement of the verbs itd-146's decision 2 does not name" and 2026-09-29 ruling BX1: the placements and the one provisional one this intent inherits.
- `.abcd/development/research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md`: hiding and a namespaced plugin tree were both tried against the record; the objections that transfer are named in Open Questions.
- `04-surfaces/README.md` "The command files": the flat directory is load-bearing (iss-161) and the index is gated by `index_drift`.

## Acceptance Criteria

_Proposed by the drafting reviews; not yet confirmed by the product thinker._

- **Given** a product thinker in a host with abcd installed, **when** they type `/abcd:` and read the completion menu, **then** every entry shown is a page whose `block:` is `people` or a page the audit classed as a person's, and no page classed as an agent's is listed.
- **Given** a facilitator watching an agent in that same host, **when** the agent is told by `abcd --help --agent` to read `commands/guard.md` and run `/abcd:guard`, **then** the agent runs it and it behaves exactly as before; no invocation named on an agents-block help line is renamed or removed.
- **Given** `abcd --help --agent`, **when** it renders, **then** it is still the one place both blocks appear, and each agents-block line still names the page an agent reads next.
- **Given** a new hire who types the full name of an agent's page, **when** the host answers, **then** the host says the page is not a person's command and points them to ask the agent, as the trial of 2026-10-10 showed, so the behaviour is a decision, not an accident.
- **Given** a new command page is added with `block: agents` but placed where a person's menu lists it, or with `block: people` but placed off the person's menu, **when** the tests run, **then** a test fails naming the page; the plugin menu is gated like the CLI list (itd-146, criterion 3).
- **Given** the thirty-five pages at the tip, **when** this intent ships, **then** the verb audit's table (iss-2610090831317531) names every page with its class and keep/merge/rename/retire decision, and no page is deleted.
- **Given** a live host session, **when** this intent ships, **then** a capture of the `/abcd:` menu before and after is attached to the spec, since the host's menu rendering is not visible to a test.
- **Given** a person reading abcd's user-facing docs after this intent ships, **when** they look up the library, **then** the page explains the library and memory notes side by side in plain words (what each holds, who adds to it, how it comes back, where it can go), the same comparison the brief's surfaces overview carries. (Required by the product thinker, 2026-10-09: "explain both in that accessible way in the docs and in the brief".)

## Open Questions

- What gates the plugin menu's classification, the way `TestCommandPagesDeclareTheirBlock` and the surface snapshot gate the CLI's? The person's menu needs a test that fails when a page's `block:` and its menu placement disagree, and one that holds the plugin's person list equal to the CLI's (Decisions, 2026-10-09: no cap, the two kept in step); under the hiding ruled below, the first reads each page's `block:` against its `user-invocable` key.
- Would a later recommendation of which verb fits a situation belong to this intent, or to a separate one? The nearest record today is itd-2609212113220149, one actionable sentence per verb, and bare `abcd <record-id>` already names a record's next move.

## Decisions

- 2026-10-09: the product thinker ruled who a command belongs to. Asked whose command menu it is, given that some commands marked for agents are typed by people (the inbox, the drain, the dashboard), they answered: "who marked them for agents? If people have to use them to do their job, they're not agents-only". Most of the current agent labels were placed on 2026-09-25 by an autonomous run's orchestrator, recorded and not asked, with a tie-break that sent a command either side could claim to the agents' side to keep the person's list under its cap (.abcd/work/DECISIONS.md, that date). So a command a person needs to do their job is a person's command whatever else also runs it; the verb audit (iss-2610090831317531) re-labels every command by who actually types it, and only commands no person types leave the person's menu.
- 2026-10-09: the product thinker chose to try the host's hiding before choosing between hiding and grouping. Answer verbatim: "Try it first". One command no person types is hidden in a real session, with captures before and after of what a person sees on typing it and whether an agent still runs it; the choice of mechanism (the first open question) waits on that trial.
- 2026-10-09, settled by the ruling above without a further question (one defensible answer each): `ideate` and `dashboard` are a person's commands, since the product thinker types both; `consult` and `ingest` are classed by the audit on the same rule; and each command keeps its one-sentence description unchanged wherever it is listed (itd-2609212113220149).
- 2026-10-09: the product thinker ruled on the menu's length, for the command line and the plugin alike. Answer verbatim: "no cap for either plugin or cli -- both should expose the same and stay in sync; we need to research (SOTA) all available verbs first before making a decision. 14 is a good aspirational target but not one that must be met for now." So neither surface caps the person's list; the command line and the plugin list the same commands for a person and are kept in step; fourteen stays an aspiration, not a gate; and a state-of-the-art review of every verb comes before any keep, merge, rename or retire decision. This reverses the cap of itd-2609212130136102's fifth criterion (held by TestPersonsListHoldsAtMostFourteenVerbs), flagged here for the spec to carry out.
- 2026-10-09: the product thinker ruled how agent-only steps on a person's command are shown. Asked whether steps such as the intent page's pre-pass and audit ingest should get commands of their own, they answered: "verbs the human uses are listed, options that are agent specific are not listed in help or explained to where humans look. It's a simple distinction: An agent gets to see everything, a human only what they need to see". So no agent step is moved to a command of its own and no invocation is renamed; a person's view (the menu, the default help, the user-facing docs) lists the commands a person uses and leaves out agent-only steps and options, while the agent's view (`--help --agent` and the command pages an agent reads) shows everything.
- 2026-10-09: the product thinker placed commands one by one, after the verb inventory (typed counts from saved sessions, agent run counts, and who placed each command; scratch research for iss-2610090831317531). On the person's list: abcd (the board), ahoy, build, capture, decide, disembark, embark, intent, launch, update, ideate, dashboard, lab and reading. Agent-only: lint, reflect, inbox, implement, history, site, docs, scribe, identity, guard, hook and statusline. Deferred: source ("Decide later"). They asked why several were placed where they were, and on reading asked whether a cold reading is only triggered automatically and only for the brief: it is neither, but abcd offers a person no way to commission one on a document they choose (captured separately). With reading a person's command and scribe an agent's, merging the two is no longer asked.
- 2026-10-09: drain stays on the agents' list under ruling BX1 and joins the person's list when itd-82 ships; the product thinker grouped the drain's remaining work into itd-82's plan the same day.
- 2026-10-09: peers and mode are agent-only (answers "Agent-only" to each).
- 2026-10-09: the product thinker placed the remaining commands. report, spec and rules are agent-only (rules after an answer the product thinker corrected: the first "Keep on your list" was accidental); report must ask the person's permission before it files from an abcd-managed repository (iss-2610091903077581). The `/abcd:version` page retires, and the version number shows instead when a person types `/abcd:abcd` and on bare `abcd` in a terminal. prepare-this-repo folds into `ahoy install`. changelog merges into launch, as a preview. consult, ingest, memory and source merge into one person's command named `library`, which asks whether an item is confidential when the person did not say. With the old memory command merged away, `/abcd:memory` becomes an agent's command for memory notes (itd-2610091918433290). The person's list is then: the board, ahoy, build, capture, decide, disembark, embark, intent, launch, update, ideate, dashboard, lab, reading and library, with drain joining when itd-82 ships.
- 2026-10-10: the product thinker chose hiding over grouping, after the trial they asked for. In a live session with a trial plugin holding one visible command (capture) and one hidden with the host's `user-invocable: false` (scribe): typing the plugin's prefix listed capture alone; typing the hidden command's full name was refused by the host with "This skill can only be invoked by Claude, not directly by users"; and an agent asked in plain words ran it and printed its help. Answer: "Hide them". So an agent-only page keeps its name and its place in the flat `commands/` directory, carries `user-invocable: false`, and is run by asking the agent; its documentation stays published and gated, and the agents-block help lines keep naming it. Grouping under `/abcd:agent:` is not taken, so no invocation is renamed and the three gates that locate a page by its verb name stand. The open question on the mechanism is closed by this entry.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
