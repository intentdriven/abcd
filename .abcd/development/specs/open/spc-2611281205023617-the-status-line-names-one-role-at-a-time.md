---
id: spc-2611281205023617
slug: the-status-line-names-one-role-at-a-time
intent: itd-2611031342000215
origin: researcher-authored
production_mode: hand-written
---
# The status line shows each role what it needs, in a table a repository can tune

## Summary

This spec delivers itd-2611031342000215. abcd's row on the host's status line becomes one table: an entry per element, a column per view, with the thresholds beside it, shipped as bundled defaults and overridable per entry from the machine's and the repository's `config.json`. The first element names one role, the one whose item has waited longest, or `abcd-managed`; under a facilitator working by hand a red `waiting` and its age follow it where an answer blocks; under `auto` no mode badge renders and the existing `facilitator:` element says the mode. The view follows the blocking answer, giving three views. The row sheds to the width the host reports, detail first, whole elements next and the role word last. Every element has a deterministic explanation, `abcd statusline --explain`. The row gains text mode, a warning style, gauges, links, and the session's own figures.

The intent's decisions 1 to 19 are the product thinker's rulings of 2026-11-03. Decisions 20 to 30 are the session's technical calls of 2026-11-28 (dcn-2611281148461313 to dcn-2611281148519955), and decisions 31 to 35 are the session's defaults for the five product trade-offs the planning put to the person (dcn-2611281155245894, dcn-2611281201513663, dcn-2611281201512624, dcn-2611281201529151, dcn-2611281201531202), each for the person's review. This spec builds on all of them; a reversal of one changes the step that names it.

The frame ships first (decision 31). Steps 1 to 7 build the table, the badge and views, the fit, the explanation, the gauges, the links and the builds, none of which waits on another record. Steps 8 to 10 each add an element whose source belongs to another record: the review count (spc-2610301925429250), the spend (itd-2611011230226453, a draft) and the branch's pull request (spc-2611251806215442). Until its source ships each element is absent from the row and listed in the explanation as unavailable (criterion 21). A step whose source has not shipped when the rest have landed is carried by a `--remainder` close, which keeps the intent planned until it lands.

## Approach

### What is on the default branch today

- **The row.** `internal/core/statusline/render.go` holds a fixed `order` of elements (`presence`, `repo`, `branch`, `merging`, `model`, `context`, `five_hour`, `seven_day`, `intents`, `issues`), each with a `Rendered` and a `Plain` form; only the badge's differ. Every element after the badge has an on and off switch in the person's `~/.abcd.noindex/statusline.json` (`settings.go`, defaults in `defaults/statusline.json`). There is no width read, no view, no warning style, and no repository or machine layer.
- **The badge.** `compose.go` reads the mode through `mode.ReadAtContext`; `badge.go` renders one of three words. The store, `.abcd/.work.local/mode`, is one word with no time and no session; its one writer is `mode.SetAt` (`internal/core/mode/store.go`). The question marker (`internal/core/mode/question.go`) records a question an agent's question was admitted under, and `mode.ResetOnAnswer` resets the mode only while that marker matches the stored word, so a mode set by hand is never cleared by the next message.
- **The mode the row names.** The `merging` element reads `facilitator: <mode> (<whose>)` in both modes (itd-2611252117557299), from the kept reading `landing.KeptMerging` (`internal/core/landing/keptmode.go`), which follows the resolver of adr-2611262044127943: the project's committed value, else the person's global setting, else `human`.
- **The status path's limits.** `statuslineBudget` is 500 ms with a 300 ms git share (`internal/surface/cli/statusline.go`). The one admitted write is the fill file (`rules.WriteFill`). `TestStatusVerbReachesNoWritePrimitive` (`internal/core/statusline/readonly_guard_test.go`) and `TestStatuslineWritesOnlyTheFillFile` (`internal/surface/cli/statusline_readonly_test.go`) hold it. No network. `TestEveryPrefixOfTheRowBeginsWithTheBadge` holds the badge first.
- **Width.** `internal/textwidth.Columns` counts East Asian wide runes as two and every other rune, escapes and joiners included, as one. go.mod carries no grapheme library.
- **The install.** `internal/core/ahoy/statusline_apply.go` writes `{"type":"command","command":…}` into the harness's `statusLine` setting, keeps a backup and restores it on uninstall. It writes no refresh interval.
- **The sibling spec.** spc-2611251806215442 (itd-2611141507299828, open, not started) plans `internal/core/sessionwork`, the part frame `part.go` and three parts, `abcd hook pr-watch`, `switches.go` (the `statusline.elements.<key>.visible` claim, off winning from any layer) and the first `fit.go`, which fits only rows that carry one of its parts.

### The table and its layers (criterion 10; decisions 13, 28)

The bundled table is `internal/core/statusline/defaults/table.json`, embedded and validated at init as `defaults/statusline.json` is. Each element entry carries, per view (`pt`, `tf_human`, `tf_auto`), its `words`, its `visible` flag, its detail `stages[]` and its `drop` rank; beside the entries sit `detail_order[]` and `thresholds` (`reviews_count` 5, `reviews_age_hours` 24, `key_budget_pct` 80, `context_pct` 80, `key_list_max` 2, `margin` 6, `label_cut` 24). `render.go`'s fixed `order` is read from the table.

`table.go` loads the table through `internal/core/layered` under the one namespace `statusline`, from the machine's `~/.abcd.noindex/config.json` and then the repository's `.abcd/config.json`, the repository file read from the working tree as a file, with no git question. Each field of an entry is replaced whole, as `.abcd/rules.json` layers are. For every field but `visible` the repository's value wins the machine's; `visible: false` from any layer, or the element switched off in the person's `statusline.json`, hides the element, which is spc-2611251806215442's rule, extended here and not renamed. An unknown key under `statusline` is refused naming its file. A drop position is an integer rank, ties broken by bundled order, so one entry moves alone.

The person's `statusline.json` keeps the on and off switches, the colours and `previous_command`, and gains `text_mode`, `theme` (`dark` or `light`, `dark` by default) and `links` (true by default). A repository or machine `config.json` naming any of those, or `presence`, under `statusline` is refused by name.

The table's views leave out of the product thinker's view the dirty marker, the branch's pull-request state, the person's open pull requests, the peers, the model, the context and the agents (`visible: false` in the `pt` column; decision 33). The builds element's `words` differ by view: the `pt` column reads `in progress N`, and the two facilitator columns carry lane ids and stage names (decision 35). The drive's part, spc-2611251806215442's, keeps one product wording in every column (itd-2611141507299828 decision 9), and its entries are that spec's to write.

### One role, `waiting`, and the views (criteria 1, 2, 3, 5; decisions 1, 2, 5, 26, 32, 34)

The badge names one role, chosen oldest first:

1. The oldest block. The mode store's block is dated by the mode file's modification time (decision 26). `mode.SetAt` writes nothing when the word already stored is the one being set, so a re-set does not reset the age. A drive's block, once spc-2611251806215442 step 8 has given the drive state its `Needs`, is dated by the drive state's `Since`. A source that has not shipped or cannot be read takes no part.
2. With no block, the role of the oldest pending review, once step 8 here supplies the kept spot-check reading.
3. With neither, `abcd-managed`.

`waiting` is an element after the role, red foreground text with no background, the only red on the row, and only where `facilitator.mode` resolves to `human` as the kept reading holds it (never the working tree's copy, criterion 3). It carries the block's age, `waiting 2h`, in the table's words. Under `auto` no mode badge and no `waiting` render, whatever is owed; the `merging` element already says `facilitator: auto (project)` (decision 32). The badge's `plain` text changes (a role, and `waiting` beside it, in place of `waiting on the technical facilitator`); the board relays it, and the JSON `state`, the element keys and the off switch are unchanged (decision 30).

The view is chosen in `view.go` from the blocking role and the merge mode: the product thinker's view while a block is owed to the product thinker, else the facilitator's working by hand or the facilitator's under auto. A pending review sets the badge, never the view, and no role setting exists (criterion 5).

**The session-start reset (decision 34; iss-2609291032194222).** The session-start hook (`hook session-start`, in `internal/surface/cli/cli.go`) runs the same reset the answer runs, `mode.ResetOnAnswer`, through a named entry `mode.ResetAtSessionStart` in `question.go`: it sets the mode back to managed only when the question marker is present and names the stored word, so an agent's question a crash left behind is cleared at the next start and a mode a person set by hand survives, keeping its age. The cost the decision states stands: such a park stays until a session starts; and a second session started in the same checkout while a first is mid-question clears that first's waiting mode, which its answer would have cleared in any case. The step that builds it resolves iss-2609291032194222 with its trailer.

### Fitting the width (criteria 7, 8, 9; decisions 6, 23)

The front door reads `COLUMNS` as a positive integer; absent, zero or not a number means no fit, and nothing is shed. Core reads no environment: the width is passed in as a value. The fit measures the `Plain` form plus the badge's two fill cells; a test holds that no `Plain` carries `\x1b` or an OSC 8 sequence.

`fit.go` is one engine: an ordered list of sheds, each a detail stage of one element or a whole element, applied in order and stopped at the first row that fits `COLUMNS` less the margin. The list is the table's `detail_order` (the same in every view), then the view's elements by drop rank, then the role word's short form (`PT` or `TF`). The badge is never shed, so every prefix begins with it. If spc-2611251806215442 step 3 has landed, this spec lifts its guard that fits only rows carrying one of its parts, so every row sheds, and re-baselines that spec's golden row, naming the change in the pull request; if this spec's step lands first, it builds `fit.go` in that spec's shape and that step only adds its sheds (decision 29).

`internal/textwidth` gains a cluster-aware measure, with no new dependency: a zero-width joiner joins a sequence that counts two; VS16 makes its base two; VS15, skin-tone modifiers and combining marks count nothing; a regional-indicator pair counts two; C0 and C1 controls count nothing. The emoji the table declares are a closed set, each tested against the measure.

### The explanation, warnings and text mode (criteria 11, 14, 15, 21; decisions 8, 9, 12)

`abcd statusline --explain` emits one JSON record: the view, and for every element of the view's table its `value`, `warns`, `why`, `threshold`, `source` and `available`. An element whose source has not shipped or cannot be read is absent from the row with no placeholder and no separator, and appears in the record with `available: false` and a `why` naming the record whose work supplies it (`not yet shipped: spc-2610301925429250`). Two runs over one input emit identical bytes, and the run writes nothing (the snapshot test).

A warning is yellow and, where it needs a symbol, `!`; never red, never emoji (`warn.go`). Text mode renders gauges as percentages (`18%/27%`), the collapsed counts as `PT 3`, `TF 1`, `ag 2/94`, the warning as `!` and the resets as words, so no rune outside ASCII renders but the separator `·`.

### Gauges and the session's own figures (criteria 16, 18; decisions 10, 14, 15, 16, 27)

`payload.go` parses the reset times, the session's API time and lines changed. `gauge.go` draws the context gauge as four cells on a greyish-blue ramp, distinct at every 5% step, and the usage gauge in the same four cells with the seven-day reading as the dot and the five-hour as an underline; a session without the five-hour reading keeps the dots alone. Resets read `resets 54m/3d19h`. The ramp follows `theme`; the light ramp is an untuned first value, and every ramp passes a contrast test in the style of `contrast.go`.

The transcript token total is kept incrementally in the fill record (`internal/core/rules/fill.go`): its byte offset and running sum by basis, so a refresh reads only the tail appended since the last. This extends the one admitted write and adds none. Only the session's own payload, transcript and fill are read.

### The refresh interval and links (criteria 11, 12, 13; decisions 11, 21, 22, 24)

`ahoy install` sets the host's refresh interval on the `statusLine` setting to five seconds unless it already carries one, which it keeps; uninstall restores the setting byte for byte. The step confirms the key's exact name against the host's current status-line documentation and pins it in `statusline_install_test.go` from a captured settings file.

OSC 8 links go in the `Rendered` form only, as `ESC ] 8 ; ; <uri> ESC \ text ESC ] 8 ; ; ESC \`. Only `https://` links to the forge host of the repository's `origin`, and `file://` links under `~/.abcd.noindex/session-work/`, are admitted, each through `termsafe`; a URI with any byte outside its safe set is dropped and its text kept. `links: false` emits none.

The explanation page and the lanes page are written off the status path by a detached `abcd hook statusline-pages`, which the prompt hook (`internal/surface/cli/status.go`) starts at most once per refresh interval per session, in the pattern of `spawnScheduledRefresh` (`hook_scheduled.go`), into `~/.abcd.noindex/session-work/<key>.explain.html` and `<key>.lanes.html`. Each page states when it was written. The row links to a page only once it exists; otherwise the symbol or count shows with no link.

### The builds and the worktree label (criteria 6, 20; decisions 7, 17, 35)

The run-state writer (`internal/core/implement/loop/state.go`) keeps a small runs index beside the run states, `runindex.go`, written whole by rename: per live run its intent's short id, its lanes and each lane's stage and worktree path. The row reads the index once. In the facilitator's views the builds element lists lanes and stages; in the product thinker's view it reads `in progress N`, the number of distinct pieces of work, with no record id, lane id or stage name. The builds count links to the lanes page. A session whose working directory is a lane's worktree reads `<repo> ‹<intent short id> L<lane>›` from the index; another worktree reads by its own name; a label wider than 24 columns is cut in the middle; the main checkout still reads `<repo>:<branch>`.

### The review count (criteria 1, 4; decisions 4, 25)

A kept reading per repository under `~/.abcd.noindex/`, keyed as `landing/keptmode.go` keys the facilitator's, holds the waiting count and the oldest drawn time for each spot-check role. The spot-check writers write it, and so does the prompt hook after its fetch, from the branch as last fetched; the row reads one file and asks git nothing. The product thinker's count is the product-thinker role's; the technical facilitator's under `human` is the facilitator role's; under `auto` it is the human facilitator's sample of abcd's own decisions. The count is grey (`manual reviews 3`), and yellow behind a yellow `!` at the count or age threshold, read from the repository over the machine over the bundled default. With no kept reading the count is absent and the badge's review rung is skipped. The count's source is spc-2610301925429250 step 3 (the sitting and judgements); this step lands after it.

### Agents and spend (criteria 17, 18; decisions 14, 16, 20)

`RecordLaunch` (`internal/core/runner/ledger.go`) rewrites a per-session rollup, `~/.abcd.noindex/session-work/<sha256(session)>.agents.json`, whole by rename at each launch's start and end: running, started and failed counts, spend per key, tokens by basis, the entry count, the newest entry's id and the time written. It lives in spc-2611251806215442's `sessionwork` leaf. Any writer, and `abcd agents`, rebuilds it when its count or newest id disagrees with the ledger; the explanation calls it a derived cache, and `abcd agents` stays the account. Past the 30-second interval the agents element reads "as of" its written time. The spend element shows each key's spend against its budget, yellow at 80%, collapsing past two keys to `3 keys $142/$450` plus any key past its share; a subscription session shows no cost. spc-2611050023402644 step 1 plans the same row read; whichever lands first builds the rollup in this shape, and the other adds nothing. The spend element's budgets come from the key route of itd-2611011230226453, a draft; until it ships the agents element renders and the spend element is absent.

### The forge's figures (criteria 12, 19; decision 18)

The branch's pull-request number and state and the person's open pull requests are added to the file spc-2611251806215442's `abcd hook pr-watch` keeps, `~/.abcd.noindex/session-work/<key>.prs.json`, through `internal/core/landing/prwatch.go`, so there is one watcher, not two. The row reads that file; data older than its interval renders "as of". The number links to the pull request's forge page. The status command makes no network call, starts no process and asks git nothing new. This step lands after spc-2611251806215442 step 2.

### Coordination

- **With spc-2611251806215442 (decision 29).** Both specs edit `render.go` and `compose.go`, so their steps land one at a time. That spec's steps 1 to 3 land first where they are in flight; this spec's step 1 then extends `switches.go`, and its step 3 generalises `fit.go`. Where this spec's step reaches either file first, it creates it in that spec's shape and that spec's step shrinks to its own rows. The step parser reads `needs:` as this spec's numbers alone, so each wait on the other spec is stated in the step's text, and the session that starts the step checks the default branch first.
- **Sources.** Step 8 waits on spc-2610301925429250 step 3; step 9's spend half waits on itd-2611011230226453; step 10 waits on spc-2611251806215442 step 2. Steps 1 to 7 wait on no other record.
- **Reservations.** Lapsed reservations of 2026-11-26 name `internal/core/statusline` as a whole directory (iss-2610302057380175, itd-2611252049031692). Each step runs `scripts/abcd.sh peers` first and reserves its own footprint's files.
- **The configuration page.** Step 1 edits `.abcd/development/brief/05-internals/03-configuration.md`, which another intent's reservation held on 2026-11-28; it is cut from the default branch as it then stands.

## How the Acceptance Criteria are met

The criteria are numbered in the order the intent lists them.

1. One role, oldest first, its own count, unshipped sources skipped: step 2 (the block rung, the mode file's age, a drive's `Since` once readable) and step 8 (the review rung). Tests: a table of blocks and reviews for both roles gives the oldest block's role, then the oldest review's, then `abcd-managed`; a source absent takes no part.
2. `waiting` red with its age under `human`; nothing under `auto`; the session-start reset: step 2. Tests: the mode × resolved mode table; `waiting` is the only red foreground SGR and has no background; under `auto` no badge word and no `waiting`; a session start over an agent-set mode resets it, over a hand-set one keeps it and its age.
3. The resolver's answer, never the working tree's: step 2. Test: a working-tree `facilitator.mode` that differs from the kept reading changes nothing.
4. The review count grey, yellow `!` at the thresholds, layered: step 8.
5. The view follows a blocking answer only: step 2.
6. The product thinker's view: outcomes, no ids, no left-out elements; the facilitator's views in their words, the drive's part in product words: step 2 (the left-out elements, from the table of step 1) and step 7 (the builds). Test: a regular expression over every `Plain` of the product thinker's view at every width finds no record id, lane id or stage name and none of the seven left-out keys.
7. Shedding, detail first, role word last, stopping at the first fit, badge first: step 3.
8. Width measure: step 3.
9. `COLUMNS` absent sheds nothing: step 3.
10. The table's layers: step 1.
11. The explanation record, and the warning link to a written page: step 4 (the record) and step 6 (the link).
12. Links to the pull request and the lanes page, text with no link otherwise, neither page written by the status command: step 6 (the mechanism and the lanes page), step 7 (the builds count's target) and step 10 (the pull request's target).
13. The refresh interval: step 6.
14. Text mode: step 4 for the elements then present; steps 5, 7, 8 and 9 add their elements to its test.
15. Warnings yellow and `!`, never red or emoji: step 4; each later step adds its warnings to the test.
16. Gauges: step 5.
17. Spend: step 9.
18. The session's own figures: step 5 (payload and transcript) and step 9 (the ledger).
19. The forge's figures from the one watcher's file: step 10.
20. The worktree label: step 7.
21. An element with no source is absent and explained: step 4, with each of steps 8 to 10 removing its element from the unavailable list as its source ships.

## Open Questions

_All resolved: no question is open; the notes below are for the reader._

- The intent's own open questions 1, 3, 4, 5 and 6 are defaults the build ships at the values the intent states, tuned in use; none holds a step.
- Decisions 31 to 35 are the session's defaults while the person is away, each for the person's review. A reversal of decision 32 changes step 2's badge rule; of 33 or 35, the table of step 1; of 34, step 2's session-start reset; of 31, whether steps 8 to 10 hold the close.

## Footprint

- packages: internal/core/statusline/table.go, internal/core/statusline/defaults/table.json, internal/core/statusline/defaults/statusline.json, internal/core/statusline/settings.go, internal/core/statusline/switches.go, internal/core/statusline/render.go, internal/core/statusline/compose.go, internal/core/statusline/badge.go, internal/core/statusline/view.go, internal/core/statusline/fit.go, internal/core/statusline/explain.go, internal/core/statusline/warn.go, internal/core/statusline/payload.go, internal/core/statusline/gauge.go, internal/core/statusline/contrast.go, internal/core/statusline/link.go, internal/core/statusline/part_builds.go, internal/core/statusline/part_reviews.go, internal/core/statusline/part_agents.go, internal/core/statusline/part_forge.go, internal/core/mode/store.go, internal/core/mode/question.go, internal/textwidth/textwidth.go, internal/core/rules/fill.go, internal/core/ahoy/statusline_apply.go, internal/core/implement/loop/state.go, internal/core/implement/loop/runindex.go, internal/core/spotcheck/kept.go, internal/core/sessionwork/agents.go, internal/core/runner/ledger.go, internal/core/landing/prwatch.go, internal/surface/cli/statusline.go, internal/surface/cli/status.go, internal/surface/cli/cli.go, internal/surface/cli/hook_statusline_pages.go, commands/abcd.md, commands/mode.md, commands/ahoy.md, docs/reference/cli/commands.md, docs/reference/cli/commands/statusline.json, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/26-mode.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/work/issues/open/iss-2609291032194222-the-status-line-s-waiting-role-badge.md, .abcd/work/issues/resolved/iss-2609291032194222-the-status-line-s-waiting-role-badge.md
- tests: internal/core/statusline/table_test.go, internal/core/statusline/settings_test.go, internal/core/statusline/view_test.go, internal/core/statusline/badge_role_test.go, internal/core/statusline/fit_test.go, internal/core/statusline/explain_test.go, internal/core/statusline/gauge_test.go, internal/core/statusline/payload_test.go, internal/core/statusline/contrast_test.go, internal/core/statusline/link_test.go, internal/core/statusline/part_builds_test.go, internal/core/statusline/part_reviews_test.go, internal/core/statusline/part_agents_test.go, internal/core/statusline/part_forge_test.go, internal/core/statusline/readonly_guard_test.go, internal/core/mode/store_test.go, internal/core/mode/question_test.go, internal/textwidth/textwidth_test.go, internal/core/rules/fill_test.go, internal/core/ahoy/statusline_install_test.go, internal/core/implement/loop/runindex_test.go, internal/core/spotcheck/kept_test.go, internal/core/sessionwork/agents_test.go, internal/core/runner/ledger_test.go, internal/core/landing/prwatch_test.go, internal/surface/cli/statusline_width_test.go, internal/surface/cli/statusline_explain_test.go, internal/surface/cli/statusline_readonly_test.go, internal/surface/cli/statusline_network_test.go, internal/surface/cli/board_presence_test.go, internal/surface/cli/hook_session_start_mode_test.go, internal/surface/cli/hook_statusline_pages_test.go, internal/surface/cli/hook_git_audit_test.go, internal/surface/cli/testdata/hook-git-baseline.txt, internal/surface/cli/ahoy_statusline_test.go

## Steps

1. The table and its layers
   - criteria: 10
   - needs: none
   - packages: internal/core/statusline/table.go, internal/core/statusline/defaults/table.json, internal/core/statusline/settings.go, internal/core/statusline/render.go, internal/core/statusline/switches.go, .abcd/development/brief/05-internals/03-configuration.md, docs/reference/cli/commands/statusline.json
   - tests: internal/core/statusline/table_test.go: one entry's field overridden in the machine layer replaces that field alone and leaves every other at its bundled default; the repository's value wins the machine's for the same field; `visible: false` from either layer hides the element, and `true` in one does not show an element the other layer or the person's `statusline.json` turned off; an unknown key under `statusline` is refused naming its file; `presence`, a colour, `text_mode`, `theme` or `links` in either `config.json` is refused by name; a drop rank moves one entry alone, ties by bundled order; the bundled table validates at init, and the product thinker's column leaves out the seven elements of decision 33. internal/core/statusline/settings_test.go: `text_mode`, `theme` and `links` read with their defaults. A golden row over the existing fixtures is byte for byte today's with no override. Each is written first and watched fail. `switches.go` is extended if spc-2611251806215442 step 4 has landed and created in its shape if not. Waits on no other step.
2. One role, `waiting` with its age, the views, and the session-start reset
   - criteria: 1 (the block rung), 2, 3, 5
   - needs: 1
   - packages: internal/core/statusline/badge.go, internal/core/statusline/view.go, internal/core/statusline/compose.go, internal/core/statusline/render.go, internal/core/mode/store.go, internal/core/mode/question.go, internal/surface/cli/cli.go, internal/surface/cli/statusline.go, commands/abcd.md, commands/mode.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/26-mode.md, .abcd/work/issues/open/iss-2609291032194222-the-status-line-s-waiting-role-badge.md, .abcd/work/issues/resolved/iss-2609291032194222-the-status-line-s-waiting-role-badge.md
   - tests: internal/core/statusline/badge_role_test.go: a table of stored mode × resolved `facilitator.mode` (human, auto, unread) × block ages gives the role, the presence of `waiting` and its age, and the view; the oldest of two blocks wins; a source that cannot be read takes no part; under `auto` no mode badge and no `waiting` render, whatever is owed; `waiting` is the only red foreground SGR on the row and carries no background; with escapes stripped every state still differs (extending `TestBadgeMeaningSurvivesColourRemoval`); every prefix begins with the badge. internal/core/statusline/view_test.go: a review owed to the product thinker turns the badge but not the view; a block owed to them turns the view; the product thinker's view carries none of the seven elements decision 33 leaves out; no role setting is read. internal/core/mode/store_test.go: setting the stored word again keeps the file's modification time. internal/core/mode/question_test.go and internal/surface/cli/hook_session_start_mode_test.go: a session start over a mode an agent's admitted question set (the marker naming the stored word) resets it to managed and clears the marker; over a mode set by hand (no marker, or a marker naming another word) it changes nothing and the age holds; a status-line run for a fresh session over that stale store shows no `waiting`. internal/surface/cli/board_presence_test.go: the board's presence line agrees with the row. A working-tree `facilitator.mode` that differs from the kept reading changes neither the badge nor `waiting`. Each is written first and watched fail. The change resolves iss-2609291032194222 with a `Resolves:` trailer, its record moved to `resolved/` in the same diff. `cli.go` was under a live reservation of itd-2611141610197753 on 2026-11-28; the step is cut from the default branch as it then stands.
3. Every row fits the width
   - criteria: 7, 8, 9
   - needs: 1, 2
   - packages: internal/textwidth/textwidth.go, internal/core/statusline/fit.go, internal/core/statusline/render.go, internal/surface/cli/statusline.go
   - tests: internal/core/statusline/fit_test.go: a row that fits is unchanged; sheds stop at the first fit; in each view every detail stage goes before any element; the role word shortens to `PT` or `TF` only after the last element; every prefix of every fitted row begins with the badge; no `Plain` form carries an escape or an OSC 8 sequence. internal/textwidth/textwidth_test.go: an OSC 8 target counts nothing, an emoji two, a joined family two, VS16 makes two, a skin tone nothing, an escape nothing, a regional-indicator pair two; every emoji the table declares measures as asserted. internal/surface/cli/statusline_width_test.go: `COLUMNS` at 120, 80 and 40 fits each view; `COLUMNS` absent, `0` and `abc` shed nothing. Where spc-2611251806215442 step 3 has landed, its guard is lifted and its golden row re-baselined, named in the pull request. Each is written first and watched fail.
4. The explanation, the warning style, text mode, and elements with no source
   - criteria: 11 (the record), 14 (for the elements then present), 15, 21
   - needs: 1
   - packages: internal/core/statusline/explain.go, internal/core/statusline/warn.go, internal/core/statusline/render.go, internal/surface/cli/statusline.go, docs/reference/cli/commands/statusline.json, docs/reference/cli/commands.md, commands/abcd.md, .abcd/development/brief/04-surfaces/08-abcd.md
   - tests: internal/core/statusline/explain_test.go: two runs over one input give identical bytes; every element of the view's table appears with its value, warns, why, threshold, source and availability; an element whose source has not shipped is absent from the row, leaves no separator, and appears with `available: false` and its record id. internal/surface/cli/statusline_explain_test.go: `--explain` writes no file (the snapshot test); each warning is yellow SGR and `!`, never red and never emoji; in text mode no rune outside ASCII but `·` renders. Each is written first and watched fail.
5. Gauges and the session's own figures
   - criteria: 16, 18 (payload and transcript)
   - needs: 4
   - packages: internal/core/statusline/payload.go, internal/core/statusline/gauge.go, internal/core/statusline/contrast.go, internal/core/statusline/render.go, internal/core/rules/fill.go, internal/surface/cli/statusline.go
   - tests: internal/core/statusline/gauge_test.go: the gauge's rendering differs at each 5% step from 0 to 100 for each window; a missing five-hour reading keeps the dots alone; resets render `54m/3d19h`; text mode gives `18%/27%`. internal/core/statusline/contrast_test.go: the dark and light ramps each pass. internal/core/statusline/payload_test.go: the reset times, API time and lines changed parse, and an absent field drops its element. internal/core/rules/fill_test.go: the token total over a transcript appended between two refreshes reads only the tail; a second session's transcript and fill are never read. internal/surface/cli/statusline_readonly_test.go: the fill file is still the only write. Each is written first and watched fail.
6. The refresh interval, the links and the pages
   - criteria: 11 (the link), 12 (the mechanism and the lanes page), 13
   - needs: 4
   - packages: internal/core/ahoy/statusline_apply.go, internal/core/statusline/link.go, internal/core/statusline/render.go, internal/surface/cli/hook_statusline_pages.go, internal/surface/cli/status.go, internal/surface/cli/cli.go, commands/ahoy.md, .abcd/development/brief/04-surfaces/01-ahoy.md
   - tests: internal/core/ahoy/statusline_install_test.go: the install writes the five-second interval under the host's key, pinned from a captured settings file; a person's own value survives a reinstall; uninstall restores the exact prior setting. internal/surface/cli/ahoy_statusline_test.go: the install's report names the interval. internal/core/statusline/link_test.go: a URI off the admitted set renders its text with no link; a byte outside the safe set drops the link; `links: false` emits no OSC 8; the `Plain` form never carries one; the `!` links to the explanation page only once it exists. internal/surface/cli/hook_statusline_pages_test.go: the pages writer writes both pages with their written time, runs at most once per interval per session, and is started only by the prompt hook; the status path contains no start (structural, with the network test). internal/surface/cli/hook_git_audit_test.go and internal/surface/cli/testdata/hook-git-baseline.txt gain the writer's one detached start. The status path's write snapshot is unchanged. Each is written first and watched fail.
7. The builds and the worktree label
   - criteria: 6, 12 (the builds count's target), 20
   - needs: 2, 3, 4
   - packages: internal/core/implement/loop/state.go, internal/core/implement/loop/runindex.go, internal/core/statusline/part_builds.go, internal/core/statusline/render.go, internal/core/statusline/compose.go, internal/surface/cli/statusline.go
   - tests: internal/core/implement/loop/runindex_test.go: the run-state writer adds, updates and removes a run's entry, whole by rename, within its cap. internal/core/statusline/part_builds_test.go: the facilitator's views list lanes and stages; the product thinker's view reads `in progress N` and a regular expression over its `Plain` at every width finds no record id, lane id or stage name; the count links to the lanes page; a lane's worktree reads `<repo> ‹<intent short id> L<lane>›`, another worktree its own name, a label over 24 columns is cut in the middle, the main checkout `<repo>:<branch>`. internal/surface/cli/statusline_network_test.go: the row asks git nothing new (the recording-git test). Each is written first and watched fail.
8. The review count
   - criteria: 1 (the review rung), 4
   - needs: 2, 3, 4
   - packages: internal/core/spotcheck/kept.go, internal/core/statusline/part_reviews.go, internal/core/statusline/badge.go, internal/core/statusline/render.go, internal/surface/cli/status.go
   - tests: internal/core/spotcheck/kept_test.go: the spot-check writers and the prompt hook write the kept reading, counts and oldest time per role. internal/core/statusline/part_reviews_test.go: with reviews for both roles and no block the badge names the oldest review's role and shows that role's count alone; grey below both thresholds, yellow `!` at 5 or 24 hours, never red; thresholds from repository over machine over bundled; under `auto` the facilitator's count is the human facilitator's sample; no kept reading leaves the count absent and the explanation marks it unavailable. The row asks git nothing new. Each is written first and watched fail. Waits on spc-2610301925429250 step 3 being on the default branch; the session that starts it checks first.
9. Agents and spend
   - criteria: 17, 18 (the ledger)
   - needs: 3, 4, 6
   - packages: internal/core/sessionwork/agents.go, internal/core/runner/ledger.go, internal/core/statusline/part_agents.go, internal/core/statusline/render.go
   - tests: internal/core/sessionwork/agents_test.go: the rollup is written whole at a launch's start and end, and rebuilt by the next writer when its count or newest id disagrees with the ledger. internal/core/runner/ledger_test.go: `RecordLaunch` rewrites the rollup for its own session alone. internal/core/statusline/part_agents_test.go: the figures equal a hand-laid ledger's for this session only; a key at 80% renders yellow; three keys collapse to `3 keys $142/$450` plus any key past its share; a subscription session shows no cost; no launch with a key route leaves the spend element absent and explained; past 30 seconds the element reads as of its written time. Each is written first and watched fail. Builds on spc-2611251806215442 step 1's `sessionwork` leaf, created in its shape if that step has not landed; the spend element's budgets wait on itd-2611011230226453, and until it ships the agents element lands alone and the spend element stays unavailable.
10. The forge's figures
   - criteria: 12 (the pull request's target), 19
   - needs: 3, 4, 6
   - packages: internal/core/landing/prwatch.go, internal/core/statusline/part_forge.go, internal/core/statusline/render.go
   - tests: internal/core/landing/prwatch_test.go: the watcher's file gains the branch's pull request and the person's open pull requests, on its existing rate. internal/core/statusline/part_forge_test.go: the number links to the forge's page; data older than its interval reads as of its time. internal/surface/cli/statusline_network_test.go: the row makes no network call, starts no process and asks git nothing new, and its ceiling holds. Each is written first and watched fail. Waits on spc-2611251806215442 step 2 being on the default branch; the session that starts it checks first.

## Size

Ten pull requests. Steps 1, 2 and 3 are the largest, and change what every session sees; steps 4 to 7 are moderate; steps 8 to 10 are small, each one element on a source another record supplies. Steps 1 to 7 can start once the pull request that files this spec has landed; steps 8 to 10 start when their sources are on the default branch, and any not landed when the rest have are carried by a `--remainder` close.
