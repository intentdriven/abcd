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
- **Given** a new hire who types the full name of an agent's page, **when** the host answers, **then** the answer is the one the chosen mechanism promises (the page runs, or the host says it is not a person's command) and the record states which, so the behaviour is a decision, not an accident.
- **Given** a new command page is added with `block: agents` but placed where a person's menu lists it, or with `block: people` but placed off the person's menu, **when** the tests run, **then** a test fails naming the page; the plugin menu is gated like the CLI list (itd-146, criterion 3).
- **Given** the thirty-five pages at the tip, **when** this intent ships, **then** the verb audit's table (iss-2610090831317531) names every page with its class and keep/merge/rename/retire decision, and no page is deleted.
- **Given** a live host session, **when** this intent ships, **then** a capture of the `/abcd:` menu before and after is attached to the spec, since the host's menu rendering is not visible to a test.

## Open Questions

- Which host mechanism carries the split? Two are known. (a) Hiding a page from the person's menu (the host's `user-invocable: false`, from the host's plugin documentation, frontmatter reference) shortens the menu and renames nothing; a person can no longer type that page, only the agent can run it. The 2026-08-22 taxonomy research's objections to hiding transfer in part: a hidden page's documentation must stay published and gated, and the agents-block help lines that name pages must keep naming them. itd-146's "no verb hidden" stands for execution and is not breached: every verb still runs. (b) A subdirectory namespace (`/abcd:agent:<page>`) clusters the menu without shortening it and renames every agent invocation. The record already rules the directory flat and load-bearing (`04-surfaces/README.md`, "The command files", iss-161), the `index_drift` rule reads a flat `commands/`, and `TestCommandPagesDeclareTheirBlock` and `helpPlacements` locate a page by verb name, so (b) is a change to three gates before it is a change to the menu, and under adr-40 the old spelling cannot survive as an alias.
- Does the split act at page grain only, or are agent sub-verbs moved to pages of their own? A person's page can carry agent sub-verbs: `/abcd:intent` carries `audit ingest`, `prepass` and `consistency ingest`. Page grain is the only option that changes no invocation.
- What gates the plugin menu's classification, the way `TestCommandPagesDeclareTheirBlock` and the surface snapshot gate the CLI's? The person's menu needs a test that fails when a page's `block:` and its menu placement disagree, and a count the way itd-2609212130136102's AC5 counts the CLI list; what that test reads depends on the mechanism chosen above.
- Would a later recommendation of which verb fits a situation belong to this intent, or to a separate one? The nearest record today is itd-2609212113220149, one actionable sentence per verb, and bare `abcd <record-id>` already names a record's next move.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
