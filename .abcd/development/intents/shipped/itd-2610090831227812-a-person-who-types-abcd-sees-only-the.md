---
id: itd-2610090831227812
slug: a-person-who-types-abcd-sees-only-the
spec_id: spc-2610100613109045
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609212130136102, itd-2609212113220149]
related_adrs: [adr-40]
impact: breaking
---

# A person who types `/abcd:` sees the commands people use

## Press Release

> A person who types `/abcd:` sees only the commands people use. Commands only agents run, such as the guard that judges a shell command, are hidden from the menu and keep their names; typed in full, the host says to ask the agent, which still runs them. `abcd --help` lists the same commands.
>
> "I used to scroll past twenty entries I would never type to reach the three I use," said Iris, a product thinker. "Now the menu is mine, and the agent has not lost anything: when it needs the guard, it still runs the guard."

## Why This Matters

`/abcd:` lists 35 commands; a person reads them all to find the few they type. An autonomous run set most agent labels unasked, so ideate and the dashboard sat with agents. Now the menu holds the 15 people use and hides the other 15; four source commands become `library`; three fold into commands doing their job.


## Mechanism

A person finds their command faster and stops opening agent pages by mistake: the menu drops from 35 to 15, all typed by people, and an agent still reaches each hidden page by name. Shown wrong if people ask agents to run hidden commands for their own work, or an agent can't find a page the help names.


## Scope Conditions

- Holds for a host whose plugin loader registers every markdown file under `commands/` as a slash command and honours `user-invocable: false` on it, as the trial of 2026-10-10 showed on the host then current. A host without that key re-decides the mechanism. <!-- cond: cond-2610100613101033 -->
- Holds while itd-146's two-block classification stands and each binary-backed page's `block:` frontmatter is the one source of its class, gated by `TestCommandPagesDeclareTheirBlock`. <!-- cond: cond-2610100613106642 -->
- Holds for the pages with no block: their class is ruled by the audit iss-2610090831317531, since no tree placement can be checked for them. <!-- cond: cond-2610100613103552 -->
- Holds while abcd ships zero skills; the `/abcd:` namespace is commands only (`05-internals/08-skills.md`), so a page moved off the person's list stays a command. <!-- cond: cond-2610100613103858 -->

## What's In Scope

- The person's `/abcd:` menu lists the pages whose `block:` is `people`, set to the placements the product thinker ruled on 2026-10-09; every page whose `block:` is `agents` carries the host's `user-invocable: false`, so it is hidden from the person's menu and refused when a person types it.
- Every page whose `block:` is `agents` stays a command, keeps its name, and is reached by an agent from the agents-block help line that names it; `abcd --help --agent` stays the one place both lists render.
- The command-line help already sorts every verb into For people and For agents and hosts (itd-146); the plugin's command list keeps that same classification, and the page's `block:` stays the one source of it.
- A test holds the plugin menu to the classification, the way `TestCommandPagesDeclareTheirBlock` holds the pages to the tree.
- The merges and the retirement the product thinker ruled on 2026-10-09: the version page retires and the version shows on the board; `changelog` merges into `launch` as a preview; `prepare-this-repo` folds into `ahoy install`; `consult`, `ingest`, `memory` and `source` merge into `library`, which asks whether an item is confidential when the person did not say. Under adr-40 no old name survives as an alias.
- The verb audit's table (iss-2610090831317531) is the first section of this intent's spec. It starts from the recorded `block:` frontmatter and rules what that leaves open: the four pages with no block the record has not ruled (`abcd`, `consult`, `ingest`, `prepare-this-repo`; `version` is ruled blockless on 2026-09-25 as a deprecated stub, and `drain` sits with the agents until it runs end to end, ruling BX1 of 2026-09-29), the pages either side could claim (`report`, `ideate`, `dashboard`), and the keep/merge/rename/retire column no record holds.

## What's Out of Scope

- Any rename or retirement beyond the merges in scope. A further one is its own intent under adr-40's pre-1.0 clean-break rule (no alias) and the retire-the-name principle.
- Moving any page to `skills/`. abcd ships zero skills and the `/abcd:` namespace is commands only (`05-internals/08-skills.md`).
- Deleting a page outside the merges in scope. Every verb that remains stays reachable from both the CLI and the plugin surface (AGENTS.md, Boundaries); hiding a page keeps it on the plugin surface.
- Giving `/abcd:memory` its new job as memory notes: that is itd-2610091918433290.

## Prior Art

- itd-146 (shipped): the two-block help behind one flag, every verb running the same whichever block lists it; this intent carries the same classification to the plugin surface and adds nothing to the CLI.
- itd-2609212130136102 (shipped): the consolidation that brought the person's list to its ceiling; its AC5 count is the CLI's, and this intent needs the plugin-side counterpart.
- itd-2609212113220149 (shipped): one sentence per verb, byte-identical on the list, `--help` and the page; a page moved off the person's menu keeps its sentence.
- [dcn-2609250000000021](../../../work/decisions/dcn-2609250000000021-help-placement-of-the-verbs-itd-146-s.md), 2026-09-25 "Help placement of the verbs itd-146's decision 2 does not name" and 2026-09-29 ruling BX1 ([dcn-2609290000000007](../../../work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md)): the placements and the one provisional one this intent inherits.
- `.abcd/development/research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md`: hiding and a namespaced plugin tree were both tried against the record; the objections that transfer are named in Open Questions.
- `04-surfaces/README.md` "The command files": the flat directory is load-bearing (iss-161) and the index is gated by `index_drift`.

## Acceptance Criteria

- **Given** a product thinker in a host with abcd installed, **when** they type `/abcd:` and read the completion menu, **then** every entry shown is a page whose `block:` is `people` or a page the audit classed as a person's, and no page classed as an agent's is listed.
- **Given** a facilitator watching an agent in that same host, **when** the agent is told by `abcd --help --agent` to read `commands/guard.md` and run `/abcd:guard`, **then** the agent runs it and it behaves exactly as before; no invocation named on an agents-block help line is renamed or removed.
- **Given** `abcd --help --agent`, **when** it renders, **then** it is still the one place both blocks appear, and each agents-block line still names the page an agent reads next.
- **Given** a new hire who types the full name of an agent's page, **when** the host answers, **then** the host says the page is not a person's command and points them to ask the agent, as the trial of 2026-10-10 showed, so the behaviour is a decision, not an accident.
- **Given** a command page is added or changed with `block: agents` but without the host's `user-invocable: false`, or with `block: people` but carrying it, **when** the tests run, **then** a test fails naming the page; the plugin menu is gated like the CLI list (itd-146, criterion 3).
- **Given** the thirty-five pages at the tip, **when** this intent ships, **then** the verb audit's table (iss-2610090831317531) names every page with its class and keep/merge/retire decision, and the only pages that leave are the ones the merges below retire.
- **Given** a live host session, **when** this intent ships, **then** a capture of the `/abcd:` menu before and after is attached to the spec, since the host's menu rendering is not visible to a test.
- **Given** a person who types bare `abcd` in a terminal or `/abcd:abcd` in the host, **when** the board renders, **then** it shows the installed abcd version, and the retired `/abcd:version` page is gone; `abcd --version` still answers.
- **Given** a person about to cut a release, **when** they ask launch for a preview, **then** launch shows the changelog the cut would write, and the `changelog` command and its page are gone.
- **Given** a person setting abcd up in a repository, **when** they run `ahoy install`, **then** it does what `prepare-this-repo` did, and the `prepare-this-repo` page is gone.
- **Given** a person adding or recalling a source, **when** they use `library`, **then** it does what `consult`, `ingest`, `memory` and `source` did, and those four names are gone from the command line and the plugin alike, with no alias (adr-40).
- **Given** a person adding an item to the library without saying whether it is confidential, **when** the item is added, **then** abcd asks them first; an item added with `--private` is not asked about.

## Open Questions

- What gates the plugin menu's classification, the way `TestCommandPagesDeclareTheirBlock` and the surface snapshot gate the CLI's? The person's menu needs a test that fails when a page's `block:` and its menu placement disagree, and one that holds the plugin's person list equal to the CLI's (Decisions, 2026-10-09: no cap, the two kept in step); under the hiding ruled below, the first reads each page's `block:` against its `user-invocable` key.
- Would a later recommendation of which verb fits a situation belong to this intent, or to a separate one? The nearest record today is itd-2609212113220149, one actionable sentence per verb, and bare `abcd <record-id>` already names a record's next move.

## Decisions

- 2026-10-09: the product thinker ruled who a command belongs to. Asked whose command menu it is, given that some commands marked for agents are typed by people (the inbox, the drain, the dashboard), they answered: "who marked them for agents? If people have to use them to do their job, they're not agents-only". Most of the current agent labels were placed on 2026-09-25 by an autonomous run's orchestrator, recorded and not asked, with a tie-break that sent a command either side could claim to the agents' side to keep the person's list under its cap ([dcn-2609250000000021](../../../work/decisions/dcn-2609250000000021-help-placement-of-the-verbs-itd-146-s.md)). So a command a person needs to do their job is a person's command whatever else also runs it; the verb audit (iss-2610090831317531) re-labels every command by who actually types it, and only commands no person types leave the person's menu.
- 2026-10-09: the product thinker chose to try the host's hiding before choosing between hiding and grouping. Answer verbatim: "Try it first". One command no person types is hidden in a real session, with captures before and after of what a person sees on typing it and whether an agent still runs it; the choice of mechanism (the first open question) waits on that trial.
- 2026-10-09, settled by the ruling above without a further question (one defensible answer each): `ideate` and `dashboard` are a person's commands, since the product thinker types both; `consult` and `ingest` are classed by the audit on the same rule; and each command keeps its one-sentence description unchanged wherever it is listed (itd-2609212113220149).
- 2026-10-09: the product thinker ruled on the menu's length, for the command line and the plugin alike. Answer verbatim: "no cap for either plugin or cli -- both should expose the same and stay in sync; we need to research (SOTA) all available verbs first before making a decision. 14 is a good aspirational target but not one that must be met for now." So neither surface caps the person's list; the command line and the plugin list the same commands for a person and are kept in step; fourteen stays an aspiration, not a gate; and a state-of-the-art review of every verb comes before any keep, merge, rename or retire decision. This reverses the cap of itd-2609212130136102's fifth criterion (held by TestPersonsListHoldsAtMostFourteenVerbs), flagged here for the spec to carry out.
- 2026-10-09: the product thinker ruled how agent-only steps on a person's command are shown. Asked whether steps such as the intent page's pre-pass and audit ingest should get commands of their own, they answered: "verbs the human uses are listed, options that are agent specific are not listed in help or explained to where humans look. It's a simple distinction: An agent gets to see everything, a human only what they need to see". So no agent step is moved to a command of its own and no invocation is renamed; a person's view (the menu, the default help, the user-facing docs) lists the commands a person uses and leaves out agent-only steps and options, while the agent's view (`--help --agent` and the command pages an agent reads) shows everything.
- 2026-10-09: the product thinker placed commands one by one, after the verb inventory (typed counts from saved sessions, agent run counts, and who placed each command; scratch research for iss-2610090831317531). On the person's list: abcd (the board), ahoy, build, capture, decide, disembark, embark, intent, launch, update, ideate, dashboard, lab and reading. Agent-only: lint, reflect, inbox, implement, history, site, docs, scribe, identity, guard, hook and statusline. Deferred: source ("Decide later"). They asked why several were placed where they were, and on reading asked whether a cold reading is only triggered automatically and only for the brief: it is neither, but abcd offers a person no way to commission one on a document they choose (captured separately). With reading a person's command and scribe an agent's, merging the two is no longer asked.
- 2026-10-09: drain stays on the agents' list under ruling BX1 and joins the person's list when itd-82 ships; the product thinker grouped the drain's remaining work into itd-82's plan the same day.
- 2026-10-09: peers and mode are agent-only (answers "Agent-only" to each).
- 2026-10-09: the product thinker placed the remaining commands. report, spec and rules are agent-only (rules after an answer the product thinker corrected: the first "Keep on your list" was accidental); report must ask the person's permission before it files from an abcd-managed repository (iss-2610091903077581). The `/abcd:version` page retires, and the version number shows instead when a person types `/abcd:abcd` and on bare `abcd` in a terminal. prepare-this-repo folds into `ahoy install`. changelog merges into launch, as a preview. consult, ingest, memory and source merge into one person's command named `library`, which asks whether an item is confidential when the person did not say. With the old memory command merged away, `/abcd:memory` becomes an agent's command for memory notes (itd-2610091918433290). The person's list is then: the board, ahoy, build, capture, decide, disembark, embark, intent, launch, update, ideate, dashboard, lab, reading and library, with drain joining when itd-82 ships.
- 2026-10-10: the product thinker chose hiding over grouping, after the trial they asked for. In a live session with a trial plugin holding one visible command (capture) and one hidden with the host's `user-invocable: false` (scribe): typing the plugin's prefix listed capture alone; typing the hidden command's full name was refused by the host with "This skill can only be invoked by Claude, not directly by users"; and an agent asked in plain words ran it and printed its help. Answer: "Hide them". So an agent-only page keeps its name and its place in the flat `commands/` directory, carries `user-invocable: false`, and is run by asking the agent; its documentation stays published and gated, and the agents-block help lines keep naming it. Grouping under `/abcd:agent:` is not taken, so no invocation is renamed and the three gates that locate a page by its verb name stand. The open question on the mechanism is closed by this entry.
- 2026-10-10: the product thinker confirmed the press release. They took a rewritten opening paragraph that states the hiding, the host's answer to a hidden name typed in full, and the same list in `abcd --help` (answer: "Take it"). They kept the quote word for word, asking whether Iris was the right persona; the persona registry picks the name from the role, and asked whose voice the quote should carry they chose the product thinker (answer: "Iris, product thinker").
- 2026-10-10: the product thinker walked the acceptance criteria. They accepted the three a person sees (the menu listing only people's commands, the host's answer to a hidden name typed in full, the docs setting the library beside memory notes) and, as one list, the five technical ones carried from the draft, with the gate re-read for hiding. Asked where the merges and the retirement ship, they answered "In this intent", so the out-of-scope ban on renames and retirements narrows to any beyond these, and five criteria for the merges were added and accepted ("Accept all").
- 2026-10-10: the product thinker confirmed Why This Matters, the Mechanism with its falsifier, and the four scope conditions, each as put to them ("Take it", "Take it", "Accept all"), and the grounds recorded on the intent ("Record it"). Settled without a further question by the ruling that only commands no person types leave the person's menu: `banlist`, typed by no person in the saved sessions and run by an agent at a person's word, stays agent-only. The impact is breaking, because the merged commands' old names go with no alias (adr-40).

- 2026-10-11: the product thinker answered the spec's open question 12 with "Wait for memory notes": the docs comparison of the library and memory notes moves to itd-2610091918433290, which already carries it as its own criterion, because memory notes are a draft with nothing built and the docs describe only what ships. That criterion leaves this intent, so the clean-up ships without it.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-07866e464a3a -->
Fidelity review — receipt rcp-07866e464a3a (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e7b75f4feac5bb758a075fb0b8408a17c6170c837c56d077bc5117bc35a1957b
Input attestations: diff:5f5592eefe0d49037f3aae067751baaf1fccf0a8..2c11d34c3f660291b7b7306d5f858bd25757771a (PR #907, spec step 1; the request named no range, so the delivery was read from the four merge commits into main that carry the spec's steps and their first parents; digest over git diff --no-color --no-ext-diff)@sha256:670b2b17324303e2a49206c1a9e9e0ccb01d0ac3c23387017075a6d5fc803fd6; diff:a09dd42b6de9a6556d5ddc49b8810f65b5064b61..fdb21c66bf7567ca84b3ce187953177748f0ffed (PR #913, spec step 2; digest over git diff --no-color --no-ext-diff)@sha256:06c95761f9342991a5398528e9994740c19c28329cf1be565360ae6199b696ff; diff:8cbb1ae450a17bb3d95fc9366a5229be82cef1b6..4700d60e928272af1c0979ff05b636f12b4f6df1 (PR #922, spec steps 3 to 7, stacked branches; digest over git diff --no-color --no-ext-diff)@sha256:44c90d9df01c3fa3ffcdcd23698b391c1e6b72bc3a9e0f7c4f0828fa85b6736e; diff:22492dbf493756ee4c3963533a15c08d2738b0b3..f104bab826399e826d2a05123f30b53602b5d98e (PR #936, spec step 8: the live captures and the close; digest over git diff --no-color --no-ext-diff)@sha256:b5067cd0d7d4ad6f2fbb9ecc124a79ec3b3adec6edd2a5014ee69aa1718798e6; repository:2e03c61e454bf09ecda8cee0ae2f377157b522f7 (origin/main, where every unprefixed file:line was read; the local audit-notes commits above it touch none of them; a ref prefixed 314e9c3ce:, 839983b8c: or f104bab82: was read at that commit, which is step 1's base, step 1 and the close)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 6 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: At the close the fifteen block: people pages carry no host key and all fifteen block: agents pages carry user-invocable: false, held page by page by TestCommandPagesMatchTheMenu, and the after capture found /abcd:scr offering capture and reading but not scribe; concerns: the after capture records three partial-match probes, not the list bare /abcd: shows (the fifteen step 8 planned to capture), and the host still offers a hidden page as the single suggestion once its full name is typed
  evidence: commands/scribe.md:6 — "user-invocable: false"
  evidence: internal/surface/cli/menu_test.go:77 — "func TestCommandPagesMatchTheMenu(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:597 — "`/abcd:scr` offered `/abcd:capture` and `/abcd:reading` and not `/abcd:scribe`. A hidden page is absent from what the person browses and from partial matches."
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:597 — "The host does show a hidden command once its full name is typed (`/abcd:guard` appeared as the single suggestion), and Enter is then refused, as A4 records."
- ac-2 — MET_WITH_CONCERNS: The after capture shows an agent running the hidden guard (allow, exit 0), TestPlacementChangesNoInvocation still runs `guard check`, and every agents-block line but one survives; concerns: `abcd changelog`, an agents-block line naming commands/launch.md before step 1, was removed and is now an unknown command, which ac-9 and the 2026-10-10 narrowing require but this criterion's letter does not allow, and the captured agent was asked in plain words rather than led by `abcd --help --agent`
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:596 — "A2: asked in plain words to run abcd's guard check on `git status`, the agent loaded the guard and ran it ("The guard returned allow for git status, exit code 0")."
  evidence: internal/surface/cli/helpgroups_test.go:236 — "out := runCLI(t, "guard", "check", "--command", "ls", "--json")"
  evidence: 314e9c3ce:internal/surface/cli/helpgroups.go:117 — ""changelog": {group: groupAgents, page: "commands/launch.md"},"
  evidence: internal/surface/cli/staleusage.go:101 — "`changelog` merged into launch as its preview: run `abcd launch --dry-run`"
  evidence: .abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:100 — "so the out-of-scope ban on renames and retirements narrows to any beyond these"
- ac-3 — MET: The root-help renderer is byte-identical across the delivery and still prints each agents line ending `(read <page>)` under the one agents title; TestRootHelpAgentRendersBothBlocks fails on an agents line whose page is missing or never mentions the entry, and the moved rules and spec lines name commands/ahoy.md and commands/intent.md, both existing pages that describe them
  evidence: internal/surface/cli/helpgroups.go:379 — "fmt.Fprintf(w, " %-*s %s (read %s)\n", agentWidth, e.name, e.short, e.page)"
  evidence: internal/surface/cli/helpgroups.go:65 — "helpAgentsTitle = "For agents and hosts (each line names the page to read next):""
  evidence: internal/surface/cli/helpgroups_test.go:117 — "func TestRootHelpAgentRendersBothBlocks(t *testing.T) {"
  evidence: internal/surface/cli/helpgroups_test.go:169 — "t.Errorf("%q names %s, which does not exist: %v", m[1], page, err)"
  evidence: internal/surface/cli/helpgroups.go:157 — ""rules": {group: groupAgents, page: "commands/ahoy.md"},"
- ac-4 — MET: The after capture records the host refusing `/abcd:guard` typed in full with words saying users cannot invoke it and to ask the agent, the answer the 2026-10-10 trial got and the product thinker then ruled on ("Hide them"), so the behaviour is a recorded decision
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:595 — "A4: typing `/abcd:guard` in full was refused by the host: "This skill can only be invoked by Claude, not directly by users. Ask Claude to use the "abcd:guard" skill for you.""
  evidence: .abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:98 — "typing the hidden command's full name was refused by the host with "This skill can only be invoked by Claude, not directly by users""
  evidence: commands/guard.md:6 — "user-invocable: false"
- ac-5 — MET: TestCommandPagesMatchTheMenu walks every page under commands/ and names each one whose block: and user-invocable key disagree in either direction, and its synthetic negative control proves each defect is named with its page
  evidence: internal/surface/cli/menu_test.go:77 — "func TestCommandPagesMatchTheMenu(t *testing.T) {"
  evidence: internal/surface/cli/menu_test.go:61 — "says `block: agents` but carries no `%s: false`, so the host lists it on the person's menu"
  evidence: internal/surface/cli/menu_test.go:58 — "says `block: people` but carries `%s: %s`; a person's page carries no such key, or the host hides it from the person's menu"
  evidence: internal/surface/cli/menu_test.go:94 — "func TestPageMenuDefectsNamesEachDefect(t *testing.T) {"
- ac-6 — MET: The spec's verb audit has one row for each of the thirty-five pages at 3b8700ef7 (the row names equal that tree's commands/ listing), each with its ruled class and action; the four delivery PRs delete exactly version, prepare-this-repo, consult, ingest, memory and source and add library; and the audit issue is resolved against the spec
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:50 — "| 1 | `abcd` (the board) | none | the bare root | people (P1) | keep; add `block: people`; show the version |"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:84 — "| 35 | `version` | none | no verb (the root's `--version`) | retire; the version shows on the board (P4); people until then (decision 14) |"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:93 — "At the end, `commands/` holds thirty pages: the thirty-five, less version,"
  evidence: 314e9c3ce..f104bab82 -- commands/ (git diff --name-status; the same changes as the four PR diffs together, with no page deleted and re-added between them) — "D commands/consult.md, D commands/ingest.md, A commands/library.md, D commands/memory.md, D commands/prepare-this-repo.md, D commands/source.md, D commands/version.md"
  evidence: .abcd/work/issues/resolved/iss-2610090831317531-abcd-s-verbs-have-never-been-reviewed-as.md:13 — "Every verb was audited with the product thinker (spc-2610100613109045's verb audit)"
- ac-7 — MET_WITH_CONCERNS: The spec carries a Live captures section recording a before session (published 0.13.4, where /abcd: listed /abcd:scribe) and an after session at 3d171421a; concerns: the record is prose, and the screenshots are kept in another checkout's local tier, not attached to the spec and not found in this machine's main checkout; the after record never states what bare /abcd: lists, only three partial-match probes; and the before capture came from the published 0.13.4 plugin, not the planned checkout of 314e9c3ce (equivalent in effect, since v0.13.4 is an ancestor of 314e9c3ce and no page there carries the key)
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:589 — "## Live captures"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:591 — "Taken 2026-10-11 in live host sessions (Claude Code v2.1.296); the screenshots are kept in the coordinating checkout's local tier."
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:593 — "typing `/abcd:` opened the host's command list, and the agents' command `/abcd:scribe` was listed to the person"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:597 — "A1: `/abcd:vers` matched no command (no retired page, and no installed 0.13.4 loaded alongside); `/abcd:libr` offered `/abcd:library` with its use-when text;"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:585 — "the AFTER capture at the merged tip: what `/abcd:` lists (the fifteen, A1)"
- ac-8 — MET: The board appends a version line built from the same core.VersionInfo `abcd --version` reports; TestBoardShowsTheVersion asserts it as the last line of both views in text and in the markdown form /abcd:abcd pastes, and as `version` in --json; commands/version.md is deleted in PR #913; and TestRootVersionFlagPrintsWhatVersionPrinted still holds `abcd --version`
  evidence: internal/core/board/board.go:125 — "if l := versionLine(in.Version, f); l != "" {"
  evidence: internal/surface/cli/board_view_test.go:257 — "func TestBoardShowsTheVersion(t *testing.T) {"
  evidence: internal/surface/cli/consolidate_test.go:329 — "func TestRootVersionFlagPrintsWhatVersionPrinted(t *testing.T) {"
  evidence: a09dd42b6de9a6556d5ddc49b8810f65b5064b61..fdb21c66bf7567ca84b3ce187953177748f0ffed -- commands/ (git diff --name-status) — "D commands/version.md"
  evidence: commands/abcd.md:23 — "The board's last line is the installed abcd version, in both views and both"
- ac-9 — MET_WITH_CONCERNS: `launch --dry-run` now renders the cut the next ship would make (derived tag, impact, guard, findings, and the added and removed records) in text and --json, and `abcd changelog` is an unknown command whose note names the preview; concerns: the preview is the deterministic cut with no prose (spec decision 5), not the changelog prose the ship composes and writes to CHANGELOG.md, and the changelog verb never had a page, so there was no page to remove
  evidence: internal/surface/cli/launch_preview_test.go:85 — "func TestLaunchDryRunRendersTheCut(t *testing.T) {"
  evidence: internal/surface/cli/launch_preview_test.go:154 — "func TestChangelogIsAnUnknownCommand(t *testing.T) {"
  evidence: internal/surface/cli/ship.go:732 — "renderEntries(w, "added", cut.Added)"
  evidence: commands/launch.md:451 — "below), with no prose, so no agent runs in a preview. Relay the derived"
  evidence: commands/launch.md:21 — "changelog prose and the release page, write them. It writes the dated section"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:85 — "| — | `changelog` | no page | agents (names `commands/launch.md`) | merge into launch as a preview (P4) | merge; verb goes | Step 3. |"
- ac-10 — MET_WITH_CONCERNS: Every phase of the retired page (the ownership refusal, orientation, audit, the adoption with its AGENTS.md merge and identity block, the conventions section and the definition of done) now sits in commands/ahoy.md's install section, held there by TestAhoyInstallCarriesThePreparation, and commands/prepare-this-repo.md is deleted; concern: the fold is page-only (spec decision 6), so the preparation runs when the host runs /abcd:ahoy install, while `abcd ahoy install` typed in a terminal is the unchanged binary install, which writes no AGENTS.md by default
  evidence: commands/ahoy.md:121 — "## `install` — prepare the repository and apply the outstanding gaps"
  evidence: commands/ahoy.md:150 — "### Phase 0 — Refuse unless the user owns this repo"
  evidence: internal/core/ahoy/preparation_page_test.go:56 — "func TestAhoyInstallCarriesThePreparation(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:632 — "**6. The prepare-this-repo fold.** Page-only, into `commands/ahoy.md`'s install"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:307 — "writes the conventions file, and a default binary `abcd ahoy install` writes no"
  evidence: 8cbb1ae450a17bb3d95fc9366a5229be82cef1b6..4700d60e928272af1c0979ff05b636f12b4f6df1 -- commands/ (git diff --name-status) — "D commands/prepare-this-repo.md"
- ac-11 — MET_WITH_CONCERNS: library carries every memory and source sub-verb under its old name, and its page folds in the consult and ingest workflows; the six pages are gone, and at the close `abcd memory` and `abcd source` refused as unknown commands, while consult, ingest and prepare-this-repo, never verbs, refuse with a note naming the successor and no alias; concerns: the name ingest survives as `library ingest`, the memory store's distiller, while the retired /abcd:ingest meant registering a document (iss-2610110233253365, still open), and after the ship itd-2610091918433290 gave the memory name back to an agents' command, as this intent anticipated
  evidence: internal/surface/cli/library_test.go:20 — "func TestLibraryCarriesEveryMemoryAndSourceSubVerb(t *testing.T) {"
  evidence: f104bab82:internal/surface/cli/library_test.go:52 — "func TestMemoryAndSourceAreUnknownCommands(t *testing.T) {"
  evidence: commands/library.md:76 — "### Search the corpus"
  evidence: commands/library.md:194 — "### Record an influence"
  evidence: internal/surface/cli/staleusage.go:99 — "`abcd library ingest` distils a source into memory pages and registers nothing"
  evidence: .abcd/work/issues/open/iss-2610110233253365-on-the-library-surface-ingest-now-names.md:15 — "On the library surface 'ingest' now names two different acts"
  evidence: internal/surface/cli/library_test.go:52 — "memory verb's name is not retired any more: the verb split gave it to the"
- ac-12 — MET: `library add` with no class asks the one private-or-public question at a terminal and files by the answer; off a terminal it refuses with exit 2 and writes nothing; it files --private and --public without asking; and the library page tells the agent to ask through the host's question tool unless the person already said, each held by a test
  evidence: internal/surface/cli/library_class_test.go:124 — "func TestLibraryAddAsksTheClassAtATerminal(t *testing.T) {"
  evidence: internal/surface/cli/library_class_test.go:160 — "func TestLibraryAddRefusesWithoutAClassOffATerminal(t *testing.T) {"
  evidence: internal/surface/cli/library_class_test.go:179 — "func TestLibraryAddPrivateIsNotAsked(t *testing.T) {"
  evidence: internal/surface/cli/library_class_test.go:224 — "func TestLibraryPageAsksTheClassThroughTheQuestionTool(t *testing.T) {"
  evidence: commands/library.md:155 — "through the host's question tool under the chip naming their role, unless the"

Gap audit:
- honoured:
  - A person who types /abcd: sees only the commands people use: at the close the fifteen people's pages carry no host key and the fifteen agents' pages carry user-invocable: false, gated page by page
    evidence: commands/scribe.md:6 — "user-invocable: false"
    evidence: internal/surface/cli/menu_test.go:77 — "func TestCommandPagesMatchTheMenu(t *testing.T) {"
  - Commands only agents run, the guard among them, keep their names, and an agent still runs them
    evidence: commands/guard.md:6 — "user-invocable: false"
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:596 — "the agent loaded the guard and ran it"
  - Typed in full, a hidden page is refused by the host, which says to ask the agent
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:595 — "Ask Claude to use the "abcd:guard" skill for you."
  - abcd --help lists the same commands as the plugin's person menu
    evidence: internal/surface/cli/menu_test.go:149 — "func TestPluginPersonListEqualsTheCLIs(t *testing.T) {"
  - The menu drops from 35 pages to 15 visible and 15 hidden, with only the merged and retired pages gone
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:93 — "At the end, `commands/` holds thirty pages: the thirty-five, less version,"
    evidence: 314e9c3ce..f104bab82 -- commands/ (git diff --name-status) — "D commands/consult.md, D commands/ingest.md, A commands/library.md, D commands/memory.md, D commands/prepare-this-repo.md, D commands/source.md, D commands/version.md"
  - The version shows on the board and the version page retires, with abcd --version unchanged
    evidence: internal/core/board/board.go:125 — "if l := versionLine(in.Version, f); l != "" {"
    evidence: internal/surface/cli/consolidate_test.go:329 — "func TestRootVersionFlagPrintsWhatVersionPrinted(t *testing.T) {"
  - Four source commands become library, which asks whether an item is confidential when the person did not say
    evidence: internal/surface/cli/library_test.go:20 — "func TestLibraryCarriesEveryMemoryAndSourceSubVerb(t *testing.T) {"
    evidence: internal/surface/cli/library_class_test.go:124 — "func TestLibraryAddAsksTheClassAtATerminal(t *testing.T) {"
  - An agent still reaches each hidden page by name, from the agents-block help line that names it
    evidence: internal/surface/cli/helpgroups.go:379 — "fmt.Fprintf(w, " %-*s %s (read %s)\n", agentWidth, e.name, e.short, e.page)"
    evidence: internal/surface/cli/helpgroups_test.go:169 — "t.Errorf("%q names %s, which does not exist: %v", m[1], page, err)"
- diverged:
  - changelog merges into launch as a preview of the changelog the cut would write; delivered as the deterministic cut (tag, impact, records, guard) with no composed changelog prose (spec decision 5)
    evidence: commands/launch.md:451 — "below), with no prose, so no agent runs in a preview. Relay the derived"
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:629 — "**5. The launch preview.** Spelling: `launch --dry-run` renders the cut."
  - prepare-this-repo folds into ahoy install; delivered as a page-only fold, so /abcd:ahoy install in the host runs the preparation while the terminal binary `abcd ahoy install` is unchanged and writes no AGENTS.md by default (spec decision 6)
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:632 — "**6. The prepare-this-repo fold.** Page-only, into `commands/ahoy.md`'s install"
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:307 — "writes the conventions file, and a default binary `abcd ahoy install` writes no"
  - No invocation named on an agents-block help line is renamed or removed, and the agent loses nothing; `abcd changelog`, an agents-block line naming commands/launch.md, is now an unknown command whose note points at `abcd launch --dry-run`
    evidence: 314e9c3ce:internal/surface/cli/helpgroups.go:117 — ""changelog": {group: groupAgents, page: "commands/launch.md"},"
    evidence: internal/surface/cli/staleusage.go:101 — "`changelog` merged into launch as its preview: run `abcd launch --dry-run`"
  - Agents' pages are hidden from the menu; the host still offers a hidden page as the single suggestion once its full name is typed, and refuses it on Enter
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:597 — "The host does show a hidden command once its full name is typed (`/abcd:guard` appeared as the single suggestion), and Enter is then refused, as A4 records."
  - The names consult, ingest, memory and source are gone; ingest lives on as `library ingest`, the memory store's distiller, while the retired /abcd:ingest meant registering a document (iss-2610110233253365, open)
    evidence: .abcd/work/issues/open/iss-2610110233253365-on-the-library-surface-ingest-now-names.md:15 — "On the library surface 'ingest' now names two different acts"
    evidence: internal/surface/cli/staleusage.go:99 — "`abcd library ingest` distils a source into memory pages and registers nothing"
  - The intent carries itd-146's classification to the plugin and adds nothing to the CLI; the delivery also re-placed eight verbs in the CLI help (lint, reflect, rules and spec to the agents' block; ideate, reading, lab and dashboard to the person's) and superseded itd-146's criterion keeping rules and spec on the person's help (spec decision 3)
    evidence: .abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:60 — "this intent carries the same classification to the plugin surface and adds nothing to the CLI."
    evidence: 314e9c3ce:internal/surface/cli/helpgroups.go:100 — ""rules": {group: groupSetUp},"
    evidence: internal/surface/cli/helpgroups.go:157 — ""rules": {group: groupAgents, page: "commands/ahoy.md"},"
    evidence: .abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:161 — "**Given** `abcd rules` and `abcd spec`, **when** `--help` renders, **then** they keep their places; nothing is renamed or nested."
  - The docs set the library beside memory notes (accepted on 2026-10-10); moved to itd-2610091918433290 by the product thinker's 2026-10-11 answer, so this intent ships without it
    evidence: .abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:103 — "That criterion leaves this intent, so the clean-up ships without it."
- missing:
  - An after capture of what bare /abcd: lists (the fifteen, as step 8 planned), and the screenshots themselves attached to the spec: the spec records three partial-match probes in prose and keeps the screenshots in another checkout's local tier
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:585 — "the AFTER capture at the merged tip: what `/abcd:` lists (the fifteen, A1)"
    evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:591 — "the screenshots are kept in the coordinating checkout's local tier."

Scope-condition dispositions:
- cond-2610100613101033 — narrowed: The after capture on Claude Code v2.1.296 shows the host registering every page under commands/ and honouring user-invocable: false for browsing, partial matches and a person's invocation, but still offering a hidden page as the single suggestion once its full name is typed
  narrowing: Holds for the host as captured on 2026-10-11 (Claude Code v2.1.296): the key keeps a hidden page out of what a person browses and partially matches, and refuses a person's invocation, but does not keep a hidden page's fully typed name out of the suggestions.
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:595 — "A4: typing `/abcd:guard` in full was refused by the host"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:597 — "The host does show a hidden command once its full name is typed (`/abcd:guard` appeared as the single suggestion), and Enter is then refused, as A4 records."
  evidence: commands/guard.md:6 — "user-invocable: false"
- cond-2610100613106642 — narrowed: TestCommandPagesDeclareTheirBlock is unchanged and the two blocks still render, but a verb's class is declared in three places (helpPlacements, the page's block: and the host key), each held to the next by a gate rather than read from one source, and itd-146's criterion that rules and spec keep their places on the person's help was superseded (spec decision 3)
  narrowing: Holds as a gated equality, not a single source: TestCommandPagesDeclareTheirBlock holds the page's block: to helpPlacements and TestCommandPagesMatchTheMenu holds the host key to block:, so a class changes only when all three move together; itd-146's two blocks stand, but not its criterion keeping rules and spec on the person's help.
  evidence: internal/surface/cli/helpgroups_test.go:316 — "func TestCommandPagesDeclareTheirBlock(t *testing.T) {"
  evidence: internal/surface/cli/helpgroups.go:255 — "func helpBlock(cmd *cobra.Command) string {"
  evidence: internal/surface/cli/menu_test.go:77 — "func TestCommandPagesMatchTheMenu(t *testing.T) {"
  evidence: .abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:161 — "**Given** `abcd rules` and `abcd spec`, **when** `--help` renders, **then** they keep their places; nothing is renamed or nested."
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:617 — "**3. rules and spec leave the person's help.** Type: supersession."
- cond-2610100613103552 — survived: The audit rules each page that had no block (the board people under P1; consult, ingest, prepare-this-repo and version people until their own step, under decision 14); step 1 wrote that class into each page's frontmatter; and after the ship the board is the only page backing no verb, with its class asserted by the menu test
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:50 — "| 1 | `abcd` (the board) | none | the bare root | people (P1) |"
  evidence: .abcd/development/specs/closed/spc-2610100613109045-a-person-who-types-abcd-sees-only-the.md:55 — "| 6 | `consult` | none | no verb (host-delegated) | merge into library (P4); people until then (decision 14) |"
  evidence: 839983b8c:commands/version.md:4 — "block: people"
  evidence: internal/surface/cli/menu_test.go:83 — "if fields, _, err := launch.PageFieldsForTest(board); err != nil || fields["block"] != blockPeople {"
- cond-2610100613103858 — survived: abcd still ships zero skills: the brief says so, the release payload names commands, agents and hooks and no skills directory, and each hidden page stays a command under commands/ with only the host key added
  evidence: .abcd/development/brief/05-internals/08-skills.md:8 — "**abcd ships zero skills.** The `/abcd:` namespace is commands only, and there is"
  evidence: .abcd/config/launch-payload.json:4 — ""commands","
  evidence: commands/guard.md:6 — "user-invocable: false"
<!-- abcd-review-end receipt=rcp-07866e464a3a -->

## Grounds

- pursued: a menu of only people's commands lets a person find theirs without reading agents'; shown wrong if people keep asking agents to run hidden commands for their own work.
