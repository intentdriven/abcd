---
id: itd-2611031342000215
slug: the-status-line-names-one-role-at-a-time
spec_id: spc-2611281205023617
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611031342011261, itd-2610301750162657, itd-2610240850336695, itd-2611252117557299, itd-2611252049031692]
refines: [itd-2610032150577708]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The status line shows each role what it needs, in a table a repository can tune

Typed links: `builds_on`
[itd-2611031342011261](itd-2611031342011261-every-agent-abcd-launches-for-a-session.md)
(the per-session launch ledger, which supplies the agents, spend, API-time and
token figures the row sums),
[itd-2610301750162657](itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
(the spot-check review queue, which supplies the review count and the age of
its oldest item),
[itd-2610240850336695](../drafts/itd-2610240850336695-before-stepping-away-the-person-grants.md)
(the autonomy grant, whose widening of the facilitator's automatic mode will
change the mode badge's word),
[itd-2611252117557299](../drafts/itd-2611252117557299-the-status-line-and-the-bare-board.md)
(the row's `facilitator:` element and the kept reading of the mode, both on
the default branch, which the badge and the `waiting` rule read; it flags a
reversal of this draft's decision 3 for a person to confirm) and
[itd-2611252049031692](itd-2611252049031692-a-person-sets-for-their-own-git-identity.md)
(the order that decides the facilitator's mode, adr-2611262044127943, which
criterion 3 follows); `refines`
[itd-2610032150577708](itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md)
(the product thinker's dashboard: the product-thinker view built here is an
interim surface, and what it shows is evidence of what that dashboard must
show).

Built on by: itd-2611141507299828 (the status line shows the progress of work that runs without the person: a drive's state, the session's pull requests and a running check). Its spec, spc-2611251806215442, shares the row's code with this intent, as decision 29 divides it; the edge stays one way, since a `builds_on` back would close a cycle.

## Press Release

> The status line names one person at a time: the role whose item has waited longest. An answer that blocks comes first, the oldest of them; with nothing blocking, the role with the oldest review to make; with nothing owed, the row says abcd is managing. A red `waiting` stands beside the role only where an answer really blocks, which is a repository whose facilitator works by hand, and it says how long the answer has waited; where abcd is the facilitator, nothing blocks, no `waiting` appears, and the row's facilitator element is the one place that says so. The reviews that role has to make show as a grey count, never a block, and turn yellow once the batch is large or old. Everything after the badges follows the person reading: the product thinker sees outcomes in plain words; the technical facilitator sees the checkout, their session's own time and context, the usage windows, the agents, the builds and the API spend against its budget, in one order when they merge by hand and another when abcd merges for them. A narrow terminal trims detail inside each element first and drops whole elements last, by the width the host reports, and the row never loses its badge. Every element and every warning has a plain explanation behind it (the value, the threshold it was judged against and where it was read from), which a person opens from the row and an agent reads as data. abcd ships the whole table as sensible defaults and makes all of it configurable from the first version, so the defaults are tuned by use and a repository's maintainers adjust them for their own.

_The product thinker's rulings of 2026-11-03, from two labs and a live trial on their own status line; the terminal is the technical facilitator's surface, and the product-thinker view is kept until their web view ships._

> "I glance at the row and know whether it is me holding something up; if it is not, I can leave it." — a product thinker (they/them)

> "On a narrow pane I lose the burn rate and the lane list, not the warning that a key is near its budget." — a facilitator (they/them)

## Why This Matters

_Facilitator-written._

Today the row carries one badge with three words (managed, waiting on the
facilitator, waiting on the product thinker), the repository, the branch, the
model, three percentages and two record counts, in one fixed order, with no
knowledge of the terminal's width: it leads with the badge so that a host
cutting the right edge keeps the one element that matters. That was right
while the row had nine elements and the host passed no width. The evidence of
2026-11-03 (lab `lab-261103114820-bcba06f`, findings F-1 to F-38 and the
element catalogue in its `state/views.md`; lab `lab-261103115111-bcba06f`, the
gauges and the default fonts) shows three things.

First, the row has more to say than fits. The session's own API time, lines
changed, context and running token total; the agents it launched, how many
run and how many failed; the build lanes and their stages; the API spend per
key against each key's budget; the five-hour and seven-day windows with their
resets. Rendered in full the row runs to 127 or 132 columns against today's
107 (F-6, F-7), and the host cuts a longer row at the edge with an ellipsis,
losing its end. The host does pass the terminal's width in `COLUMNS` (F-15),
so the row can shed to fit rather than be cut.

Second, the badge said whom abcd waits on but not what for, and a reviews
count beside `abcd-managed` named nobody (F-24), while a badge naming two
roles at once named nobody in particular. The rulings settle one badge, one
role, chosen by who has waited longest, with `waiting` reserved for an answer
that blocks and a review batch shown as a count that warns instead (F-20,
F-21, F-25, F-26, F-28, F-29).

Third, what the row leads with, and what it sheds first, differs by who reads
it and by whether the repository merges on its own (F-18, F-19). The product
thinker reads outcomes and the review count, never lanes, stages or record
ids (F-34); a facilitator merging by hand needs the pull request's state last
of all; a facilitator on auto-merge needs the builds last of all. abcd
already carries both axes: the mode store says whose answer is owed, and the
repository's `facilitator.mode` says who merges.

The live trial also found what the row cannot read from the host alone (F-2,
F-3, F-4, F-11, F-12, F-13): the harness's figures cover none of the agents
abcd's runner launches, the run state keeps no API time and no orchestrating
session, and a subscription session's cost is an estimate rather than money.
Those are the launch ledger's to fix (itd-2611031342011261); this intent reads
what it records.

## Mechanism

- We expect a person to act on the row sooner and to look at it less, because
  the badge names the one role that has waited longest and the word `waiting`
  appears only where an answer blocks, so a glance separates "me, now" from
  "a batch for later" from "nobody". Shown wrong if people keep opening the
  board or the forge to learn whether they are the one holding things up, or
  if a blocking answer goes unnoticed because a review count was read as the
  same thing.
- We expect a narrow terminal to keep the row readable with no tuning by the
  person, because the host reports its width and the shedding order drops
  what a reader of that view can most afford to lose. Shown wrong if the rows
  people see on narrow panes end in the host's ellipsis, or if the per-view
  defaults need per-person overrides before they are usable.
- We expect the shipped table to converge, because every word, stage, order
  and threshold is configuration from the first version, so a tuning made in
  use is a setting that can be read back into the defaults rather than a
  patch. Shown wrong if repositories still carry large overrides of the same
  entries a release later.

## Scope Conditions

- A host harness that runs a status command on its refresh events with a JSON <!-- cond: cond-2611281205024831 -->
  payload on stdin and the terminal's width in `COLUMNS`; without the width
  the row renders in full and the badge-first order is its only protection,
  as today.
- A managed checkout: the row renders only where `.abcd/` is present, and the <!-- cond: cond-2611281205026208 -->
  previous status command runs everywhere else.
- A terminal that interprets 24-bit colour, and a dark theme for the shipped <!-- cond: cond-2611281205025624 -->
  gauge palette; a light theme inverts the gauge's reading and the shipped
  defaults do not cover it.
- Links are hover and click only where the terminal passes OSC 8 through; a <!-- cond: cond-2611281205022914 -->
  terminal that does not shows the same text and loses only the link.
- The review count and its age read the spot-check queue <!-- cond: cond-2611281205024400 -->
  (itd-2610301750162657) and show only once it ships; the agents, spend,
  API-time and token figures read the launch ledger (itd-2611031342011261)
  and are dropped, never invented, until it ships. The frame is built first,
  and the spec closes with a stated remainder for each element whose source
  has not shipped, which keeps the intent planned until that source ships
  (decision 31).
- The terminal is the technical facilitator's surface; the product-thinker <!-- cond: cond-2611281205025455 -->
  view is interim, kept until the product thinker's dashboard
  (itd-2610032150577708) carries what it shows.
- Under the facilitator's automatic mode the row shows no mode badge, and the <!-- cond: cond-2611281205027423 -->
  `facilitator:` element of itd-2611252117557299 names the mode (decision 32),
  so the row needs no new word as that mode widens under the autonomy grant
  (itd-2610240850336695); the wider mode is not this intent's to define.

## Acceptance Criteria

Restated from the product thinker's rulings of 2026-11-03; each to be walked
at the planning interview. On 2026-11-28 the session rewrote criteria 3, 10,
11, 12, 13 and 19 against what has shipped since and the technical calls of
decisions 20 to 30, and added criterion 20 for decision 17; later the same
day it rewrote criteria 1, 2, 6 and 17 on the session's defaults for the
planning questions (decisions 31 to 35), and added criterion 21 for an
element whose source has not shipped. After the two adversarial reviews it
rewrote criteria 2, 5, 10, 13 and 20 and added criterion 22. The person
approved the move to planned on 2026-11-28, after it was made, and asked to
see a live version first (dcn-2611281230532771); the criteria and the
session's defaults stay open to amendment after that preview. The rewrites
of criteria 2, 5, 10, 13 and 20 and the addition of criterion 22 came after
that approval: the person has not seen them, and they are owed the person's
confirmation at the preview.

- **Given** blocking answers owed to both roles, from the checkout's mode
  store and from a drive that needs a person, and pending reviews owed to
  both roles, **when** the row renders, **then** the first element names
  exactly one role: the role of the oldest block (the mode store's block
  dated from when its word last changed, a drive's from its stored `since`);
  with no block, the role of the oldest pending review; with neither,
  `abcd-managed`; the review count beside it is that role's count alone; and
  a source that has not shipped or cannot be read takes no part in the
  choice.
- **Given** a blocking answer owed to the named role and a
  `facilitator.mode` that resolves to `human`, **when** the row renders,
  **then** `waiting` follows the role as red text with no background, with
  how long the answer has waited beside it (`waiting 2h`), and no other
  element renders in red; **given** `facilitator.mode` resolves to `auto`,
  **then** no mode badge renders, `waiting` never renders whatever is owed,
  and the mode is said by the `facilitator:` element alone, so with that
  element switched off nothing on the row says it and `abcd statusline
  --explain` names it (open questions 8 and 9 hold this default for the
  person); **given** a new session starts over a waiting mode an agent's
  question set, and the session that asked it is the one starting or is no
  longer live, **then** the mode is back to managed before the row renders;
  a question another live session still holds open stays, with its age;
  and a mode a person set by hand survives the start and keeps its age.
- **Given** a checkout whose working tree's `.abcd/config.json` declares a
  `facilitator.mode` that differs from what the merge gate's resolver gives
  (the default branch's committed value, else the person's global
  `~/.abcd.noindex/config.json`, else `human`), **when** the row renders,
  **then** the mode the row shows and the `waiting` rule follow the
  resolver's answer as the kept reading holds it, never the working tree's
  copy.
- **Given** reviews pending for the named role below both thresholds,
  **when** the row renders, **then** the count renders as grey text
  (`manual reviews 3`) with no background and no warning symbol; **given**
  the count reaches the count threshold or its oldest item reaches the age
  threshold, **then** the number renders yellow behind a yellow `!`, never
  red; and the thresholds are read from the repository's setting over the
  machine's over the bundled default (5 reviews, 24 hours).
- **Given** reviews pending for the product thinker and no blocking answer
  owed to anyone, **when** the row renders, **then** the badge names the
  product thinker but the view is the facilitator's; **given** blocking
  answers, **then** the view follows the badge: it is the view of the role
  the badge names, the role of the oldest block, so with blocks owed to both
  roles a facilitator whose block is older sees the facilitator's view and a
  product thinker whose block is older sees theirs; and the view is never
  read from a role setting.
- **Given** the product thinker's view and work underway, **when** the row
  renders at any width, **then** work in progress reads as a count of
  distinct pieces of work (`in progress 2`), the row carries no record id,
  lane id or stage name, and none of the elements the table marks absent for
  that view (the dirty marker, the branch's pull-request state, the person's
  open pull requests, the peers, the model, the context and the agents);
  **given** either facilitator view, **then** the builds element shows its
  lanes and stages in the facilitator's words, and the drive's part keeps
  the product wording itd-2611141507299828 rules for both views.
- **Given** `COLUMNS` in the environment and a row wider than `COLUMNS` less
  the margin, **when** the row renders, **then** detail stages are removed
  first in the fixed order the table declares (the same in every view), then
  whole elements in the current view's dropping order, and the role badge's
  word shortens to `PT` or `TF` only after every detail and element has gone;
  shedding stops at the first row that fits; a row that already fits is
  unchanged; and every prefix of the rendered row still begins with the
  badge.
- **Given** a rendered row carrying links and emoji, **when** its width is
  measured for shedding, **then** a link's hidden target counts no columns,
  an emoji counts two, a joiner sequence counts as one figure, and escapes
  count nothing.
- **Given** `COLUMNS` absent from the environment, **when** the row renders,
  **then** nothing is shed and the row renders in full.
- **Given** any element of the table in any view, **when** its words,
  visibility, detail stages or dropping position are set in the repository's
  `.abcd/config.json` or the machine's `~/.abcd.noindex/config.json`,
  **then** the row renders with that field replaced and every other field at
  its bundled default; for every field but `visible` the repository's value
  wins the machine's; `visible: false` from any layer, or the element
  switched off in the person's `statusline.json`, hides the element; an
  unknown key under `statusline` is refused naming its file; a `words` or
  `stages` string from either file that carries a control byte, an escape,
  a bidirectional control or a hidden rune, or runs past 40 runes, is
  refused naming its file and key, the field renders at its bundled value,
  and the coloured row carries no escape but abcd's own colours and the
  links it admits; and the badge colours, the text mode, the theme and the
  links switch are refused by name when the repository's file sets them.
- **Given** the same payload, checkout and settings, **when**
  `abcd statusline --explain` runs, **then** it emits one JSON record naming
  the view and, for every element of the view's table, its value, whether it
  warns, why, the threshold it was judged against, its source and whether
  that source is available; two runs emit identical bytes; and the run
  writes no file; **given** a warning on the row and an explanation page the
  detached pages refresh has written for this session, **when** the terminal
  passes OSC 8 links through, **then** the `!` links to that page, which
  states the time it was written, and with no page written the `!` carries
  no link.
- **Given** a row showing the branch's pull request number or a builds
  count, **when** the terminal passes OSC 8 links through, **then** the
  number links to that pull request's forge page and the count to the lanes
  page the detached pages refresh last wrote for this session; **given** a
  terminal that does not, or `links: false` in the person's
  `statusline.json`, **then** the same text shows with no link; and the
  status command writes neither page.
- **Given** `ahoy install` wiring the status line, **when** it writes the
  harness's `statusLine` setting, **then** it sets the host's refresh
  interval to five seconds unless the setting already carries one, which it
  keeps, and records the interval it found (or that there was none) and the
  one it wrote; **when** `ahoy uninstall` runs, **then** an interval still
  equal to the one abcd wrote is put back to what it found, or removed where
  there was none, while an interval the person changed since, one abcd never
  wrote, and every other key of the setting are left as they stand, save
  where no previous command was recorded and uninstall removes abcd's whole
  line, as it does today; and a line wired before this change gains the
  interval at the next `ahoy install`, through a gap it closes.
- **Given** the text mode set in the person's `statusline.json`, **when**
  the row renders, **then** the gauges render as percentages, the collapsed
  counts as `PT`, `TF` and `ag` before their numbers, the warning symbol as
  `!`, and the resets as words, so that no character outside ASCII renders
  but the row's own separator.
- **Given** any warning on the row (a review batch past its threshold, a
  failed agent, the context window at or past its threshold, a key at or past
  its budget share), **when** it renders in either mode, **then** it is
  yellow and, where it needs a symbol, uses `!`; no warning renders as an
  emoji or in red.
- **Given** a context reading and the two usage windows, **when** the row
  renders in the default mode, **then** the context gauge is four cells whose
  rendering differs at every 5% step from 0 to 100, the usage gauge carries
  both windows in the same four cells (the seven-day reading as the dot, the
  five-hour as its underline) with a session that lacks the five-hour reading
  keeping the dots alone, and a reset renders as words (`resets 54m/3d19h`).
- **Given** launches recorded against the session with a key route and a
  budget, **when** the row renders the spend element, **then** each key
  shows its spend against its budget, a key at or past 80% of its budget
  renders yellow, more keys than the configured limit (two) collapse to a
  summary (`3 keys $142/$450`) plus any key past its share, and a session
  billed to the subscription shows no cost figure; **given** no launch with
  a key route, **then** the spend element is absent and the explanation
  names it as unavailable.
- **Given** a session's own figures (its API time, lines changed, running
  token total, the agents it launched and their spend), **when** the row
  renders in any session, **then** every figure is that session's own, read
  from its payload, its transcript and the launches recorded against its
  session id, never another session's and never a sum across sessions; the
  usage windows, which the host reports for the account, and the badges,
  which are the repository's, are the only figures shared.
- **Given** the row needing the branch's pull request or the person's open
  pull requests, **when** it renders, **then** it reads them from the file
  the detached pull-request refresh of itd-2611141507299828 keeps, at most a
  minute old in normal running and marked "as of" when older; the status
  command makes no network call, starts no process and asks git nothing new;
  and its existing time ceiling holds.
- **Given** a session whose working directory is a build lane's worktree,
  **when** the row renders, **then** the checkout element reads
  `<repo> ‹<intent short id> L<lane>›` from the run state, another worktree
  reads by its own name, a label wider than 24 columns is cut in the middle,
  and the main checkout reads as it does today, the two elements `repo` and
  `branch` side by side (`abcd · main`), which is decision 17's
  `repo:branch`.
- **Given** an element whose source has not shipped or cannot be read (the
  review count before the spot-check queue's reading exists, the agents and
  spend before the launch ledger's rollup, the branch's pull request before
  the pull-request watcher's file), **when** the row renders, **then** the
  element is absent with no placeholder and no separator left in its place,
  and `abcd statusline --explain` lists it with `available: false` and the
  reason, naming the record whose work supplies it.
- **Given** a blocking answer owed to the named role and a kept reading of
  `facilitator.mode` that is missing, malformed or refused, so the mode is
  not known, **when** the row renders, **then** `waiting` and its age render
  as under `human`, since an answer that may block is never hidden; the
  `facilitator:` element, where shown, says the mode is not read; and the
  badge reads the mode whether or not that element is switched on.

## Decisions

Ruled by the product thinker on 2026-11-03, in two labs and a live trial on
their own status line (`lab-261103114820-bcba06f`, findings F-19 to F-38, and
`lab-261103115111-bcba06f`); each is restated here in the record's words.

1. **One role at a time, whoever has waited longest** (F-26, correcting
   F-24's two-role badge). A blocking answer first, the oldest of them; when
   it is resolved, the next blocking answer, whoever it is for; with nothing
   blocking, the role with the oldest pending review; with nothing owed,
   `abcd-managed`. The role shown carries its own review count, so a reader
   always sees what has waited longest and for whom. The mode store today
   holds one word and no time, so the oldest block needs a time recorded when
   the loop parks; the spec says where.
2. **`waiting` is an answer that blocks; reviews are a batch** (F-20, F-25).
   `waiting` renders only in manual mode, where an answer blocks, beside the
   role it blocks on, as red text with no background, so it stands apart
   from every other badge the way the host's own red warnings do (F-21).
   Pending reviews set the badge but never the view, which only a blocking
   answer switches, and the count shows in whichever view is up.
3. **The mode badge reads `auto-merge`, and its word follows the mode's
   scope** (F-22, F-23). `facilitator.mode` governs merging alone today, so
   the badge says so; it is read from the default branch's committed config
   as the merge gate reads it, never from the working tree's copy. It
   renders in auto mode only and is never paired with `waiting`, since
   nothing blocks there. When the facilitator's automatic mode widens to
   every decision a human facilitator would make but a genuine trade-off or
   one with wide consequences, the badge's word changes with it; the wider
   mode is the autonomy grant's to define (itd-2610240850336695), and the
   stance that manual mode is about who holds responsibility rather than a
   person in the loop for each decision is a principle for the person to
   confirm apart from this intent.
4. **The review count is grey, and warns in yellow past thresholds a
   repository sets** (F-28, F-29). Reviews never block, so a batch can grow
   unnoticed; the count warns, never in red, once it is large or its oldest
   item old. The shipped default is 5 reviews or 24 hours, a repository
   overrides it, and the count links to the review queue of
   itd-2610301750162657. Until that queue ships the count is absent, never
   invented.
5. **The view follows the badge, and the facilitator's view splits by merge
   mode** (F-19). The product thinker's view is up while abcd waits on the
   product thinker, the facilitator's otherwise, `abcd-managed` included;
   there is no role setting. The facilitator's view is one of two, merging
   by hand or on auto-merge. Three views in all.
6. **Shedding trims detail first, drops whole elements last, and shortens
   the role word last of all** (F-15, F-19, F-32). The host passes `COLUMNS`
   and cuts a longer row at the edge; the row sheds to `COLUMNS` less a
   margin (six) instead. Detail stages go first, in one fixed order for every
   view (lines changed, fast, cache, the seven-day reset, today's spend, the
   lane list collapsing to counts by stage, burn, effort, the five-hour
   reset, the projection, the token counts, the key list collapsing to a
   summary, the review label); whole elements go next, in a dropping order
   per view; the role badge reads `product thinker` or `technical
   facilitator` in full until every detail and element has gone, and only
   then `PT` or `TF`. The host does not re-run the command on a resize, so
   the install sets a refresh interval of a few seconds (five on the trial),
   which also covers the quiet while a coordinator waits on agents (F-16).
7. **The product thinker sees outcomes** (F-34, F-27). In their view the
   builds element becomes `in progress N`, the number of distinct pieces of
   work underway, with no lanes, stages or record ids, placed near the front
   of that view's dropping order after the peers; the review count and the
   spend hold that reader's attention. The terminal is the technical
   facilitator's surface; the product thinker will have a web view, and the
   view built here is an interim surface whose findings are evidence for the
   dashboard (itd-2610032150577708) to take up.
8. **Names stay words; counts carry a figure; warnings never do** (F-31,
   F-32, F-38). The collapsed review count is the role's figure and a
   number, the agents a worker figure and theirs, drawn from the terminal's
   emoji font at whatever size it draws them, and the width rule counts each
   two columns; emoji for the roles themselves were tried and dropped, since
   they drew larger than the text. Every warning is yellow and, where it
   needs a symbol, `!`, the one mark all six default fonts share; a warning
   emoji was tried and dropped, since a bare server may have no emoji font
   and colour needs none. Whether the counts keep their figures is a default
   to revisit.
9. **A text mode renders ASCII** (lab-261103115111, F-11 and F-12). The
   status command cannot tell which font the terminal draws, and the six
   default fonts disagree on even the filled and empty circle, so the choice
   is a setting, not detection: in text mode the gauges are percentages
   (`18%/27%`), the collapsed counts `PT 3`, `TF 1`, `ag 2/94`, the warning
   `!`, the resets words. The setting lives in the person's
   `statusline.json`, since it is about their terminal.
10. **Gauges are four grey cells, both usage windows in one**
    (lab-261103115111, F-1, F-2, F-4, F-8; F-10 of the first lab). The
    context gauge is four cells on a greyish-blue ramp, distinct at every 5%
    step; the usage gauge is the same four cells with the seven-day reading
    as the dot and the five-hour as a sand underline beneath the cells it
    has reached, so every 5% step of either changes the gauge and a session
    without the five-hour figure keeps the dots alone. The reset countdown
    reads as words (`resets 54m/3d19h`), the circular arrow having no glyph
    in four of the six default fonts (F-33). The palette assumes a dark
    terminal; a light one inverts the reading and is an open default.
11. **Links stand in for hover** (F-30). Terminals have no tooltips; OSC 8
    links, which the host passes through and a terminal shows on hover and
    opens on a click, are the nearest thing. The pull request number links
    to the pull request, the builds count to a local page listing each lane
    and its stage, and each warning symbol to the explain record; a link's
    hidden target counts no columns.
12. **Every element has a deterministic explanation** (F-37). A reader who
    sees a warning, a product thinker above all, must be able to learn what
    it is about, and an agent asked must be able to read it. The row's state
    is a structured record: the view, and for every element its value,
    whether it warns, why (the reading against its threshold) and its
    source. In the product it is a verb's machine-readable output,
    `abcd statusline --explain`, so a person, an agent and a test get the
    same answer; the row's warning links point at a rendering of the same
    record written under the local tier on each refresh, since a link needs
    a target.
13. **One table, configurable throughout from the first version** (F-35,
    F-36, the second revising the first's staging). Per-view words,
    visibility, detail stages and dropping order amount to one table, an
    entry per element and a column per view, with the thresholds beside it.
    abcd ships it as bundled defaults; the machine's and the repository's
    `config.json` override it per entry through the one layered resolver,
    the repository winning. The whole table is configuration from the first
    version, not a later rung, so the defaults are tuned by use and a
    repository's maintainers adjust them for their repository. The badge
    colours and the text mode stay in the person's `statusline.json`: they
    are about the terminal the row is drawn on, and a repository must not
    dictate the colours of a row on somebody else's machine.
14. **Spend is split by billing route** (F-12, F-13; the ledger's to record,
    this intent's to show). Spend and its budget sum launches billed to a
    key; a subscription session's cost is what the work would have cost, not
    money, and is not shown; subscription work shows in the usage gauges,
    which the host reports for the whole account. API time and tokens sum
    everything. The time beside the session's name is the session's own API
    time; each agent's time is recorded, not shown.
15. **The session's token total comes from its transcript and counts
    everything** (F-5, F-11, F-14). The payload's token fields are the
    current window, not a running total; the transcript gives the total in
    tens of milliseconds. Cache reads dominate, so the figure is large and
    its basis is stated in the explain record; the agents' tokens are counted
    apart, beside their spend.
16. **Each session's row shows that session's own figures** (the trial's
    rule). The harness's status setting is per account, so the trial ran on
    one session alone and every other session, a peer's included, kept its
    usual row and had nothing of its own saved. The product keeps the rule:
    every figure is the session's own, read from its payload, its transcript
    and the launches recorded against its session id (itd-2611031342011261),
    never another session's; the usage windows and the badges are the only
    figures shared, the first because the host reports them for the account
    and the second because they are the repository's.
17. **A worktree is named for what it builds** (F-17). A build lane's
    worktree renders as `abcd ‹itd-82 L2›`, read from the run state; another
    worktree by its own name; a label longer than 24 columns is cut in the
    middle. The main checkout still shows `repo:branch` until every change
    starts in its own worktree (itd-148). The subscription's burn is a
    projection of when the five-hour window fills at its climb over the last
    half hour, shown only when that comes before the reset, since tokens per
    hour would be almost all cache reads.
18. **The row never waits on the network** (F-30, and the row's existing
    time ceiling). Forge reads, the branch's pull request state and the
    person's open pull requests, come from a cache a detached refresh fills
    at most once a minute; the row reads the cache and nothing else.

Settled by the record, so no question was put:

19. **Impact: additive.** A row with more elements, views and settings; the
    badge-first order, the previous-command fallback and the off switch are
    unchanged.

Technical calls the session made at planning, none of which waits on the
person's answers; a person may reverse any of them:

20. **The launch writers keep a rollup the row reads.** The session's, while
    the product thinker is away (2026-11-28): the row reads the launch ledger
    through a per-session rollup,
    `~/.abcd.noindex/session-work/<sha256(session)>.agents.json`, which the
    launch writer rewrites whole at each launch's start and end (and on a
    heartbeat while it runs, decision 37); this proposes an answer to
    spc-2611050023402644's open question, which stays open in that spec for
    its own ruling. The rollup is a derived cache: the
    explanation says so, `abcd agents` stays the account, and any writer
    rebuilds it when it disagrees with the ledger. Grounds: the status path
    keeps its one admitted write and reads one bounded file, as
    spc-2611251806215442's state files already do. To reverse: a product
    ruling that admits a second write on the status path. (dcn-2611281148461313)
21. **The pages a link opens are written off the status path.** The
    session's, while the product thinker is away (2026-11-28): the explanation
    page and the lanes page are written by a detached
    `abcd hook statusline-pages`, which the prompt hook starts at most once per
    refresh interval per session, into `~/.abcd.noindex/session-work/`. The
    row links to a page only once it exists, and each page states when it was
    written. This replaces decision 12's page "written under the local tier on
    each refresh". Grounds: the status command writes nothing but the fill
    file, and itd-2611141507299828 holds it to no new write. To reverse: a
    product ruling that admits a write on the status path.
    (dcn-2611281148468557)
22. **The host's refresh interval, and nothing else.** The session's, while
    the product thinker is away (2026-11-28): `ahoy install` writes a
    five-second refresh interval into the harness's `statusLine` setting;
    there is no daemon and no file watch. A person's own interval survives a
    reinstall; the install records the interval it found and the one it
    wrote, and uninstall takes back only an interval still equal to abcd's,
    leaving one the person changed and every other key alone where a
    previous command was recorded (with none, it removes abcd's whole line,
    as today); a line wired earlier gains the interval through an advisory
    gap `ahoy install` closes (the security
    review of 2026-11-28 corrected an earlier "restores the setting as it
    was"); and the build pins the host's key name in a test. Grounds: the host does not re-run the
    command on a resize, and one interval covers both a resize and a quiet
    wait. To reverse: drop the key from the install and its test.
    (dcn-2611281148475934)
23. **One fit engine, measured on the plain form.** The session's, while the
    product thinker is away (2026-11-28): the front door reads `COLUMNS` as a
    positive integer (absent, zero or not a number means no fit) and passes it
    to core as a value; the fit measures the plain form, which carries no
    escapes and no links; one engine, `fit.go`, works through the table's
    detail order, then the view's element order, then the role word's short
    form, and stops at the first fit; and `internal/textwidth` learns to count
    a joined emoji sequence as one figure, with no new dependency, tested
    against every emoji the table declares. Grounds: core reads no
    environment; it is the shape spc-2611251806215442's step 3 specifies, so
    whichever lands first builds it; a grapheme library would need sign-off.
    To reverse: swap the measure for such a library once one is signed off.
    (dcn-2611281148479303)
24. **Links have bounded targets and a switch.** The session's, while the
    product thinker is away (2026-11-28): OSC 8 links go in the coloured form
    only; only `https://` links to the forge of the repository's `origin` and
    `file://` links under `~/.abcd.noindex/session-work/` are admitted, each
    passed through `termsafe`, and a link that fails keeps its text. A person
    turns links off with `links: false` in `statusline.json`; nothing is
    detected. Grounds: a link is bytes sent into the terminal, and a terminal
    that does not pass OSC 8 shows the text anyway. To reverse: widen the
    admitted set in the link code and its test. (dcn-2611281148480035)
25. **The review count reads a kept reading.** The session's, while the
    product thinker is away (2026-11-28): a reading per repository under
    `~/.abcd.noindex/`, keyed as the facilitator's kept reading is, holds the
    waiting count and the oldest drawn time for each spot-check role; the
    spot-check writers write it, and so does the prompt hook after its fetch;
    the row reads one file and asks git nothing. Under `auto` the
    facilitator's count is the person's sample of abcd's own decisions.
    Grounds: the queue lives on a git branch, and itd-2611141507299828 forbids
    a new git question on the status path. To reverse: read the branch on the
    status path, under a ruling that admits the git question.
    (dcn-2611281148485635)
26. **A block is dated by the mode file's modification time.** The session's,
    while the product thinker is away (2026-11-28): the mode file keeps its
    one-word format; setting the word already stored writes nothing, so its
    age holds; a drive's block takes the drive state's stored `since`.
    Grounds: hooks and the guard read the one-word format, and the time costs
    one stat of a file the row already opens. It does not settle
    iss-2609291032194222, which stays the person's. To reverse: add a time
    line to the mode file and change every reader. (dcn-2611281148499429)
27. **A theme setting, never detection.** The session's, while the product
    thinker is away (2026-11-28): the row always emits 24-bit colour; the
    person's `statusline.json` gains `theme: dark | light`, dark by default,
    with an untuned light ramp (open question 6 sets it); text mode is the
    fallback that always works; and every gauge ramp passes a contrast test.
    Grounds: the payload cannot say which theme the terminal uses, and
    `COLORFGBG` is unreliable in a process with no terminal. To reverse:
    remove the setting and its ramp. (dcn-2611281148490880)
28. **One `statusline` namespace, a rank per view.** The session's, while the
    product thinker is away (2026-11-28): the table's settings sit under
    `statusline` in the repository's `.abcd/config.json` and the machine's
    `~/.abcd.noindex/config.json`, through `internal/core/layered`: per
    element `visible`, `words`, `stages` and `drop` for each view, then
    `detail_order` and `thresholds`. A drop position is a rank, so one entry
    moves alone; each field is replaced whole. `visible: false` from any layer
    wins, as spc-2611251806215442 rules; for every other field the repository
    wins the machine. The bundled table is
    `internal/core/statusline/defaults/table.json`, embedded and checked at
    start as `internal/core/statusline/defaults/statusline.json` is. Colours,
    text mode, theme, links and the on and off switches stay in the person's
    `statusline.json`, and a repository file naming one is refused by name.
    Grounds: one resolver and one replace-whole rule, as the rules layers use;
    a rank, unlike a list, lets one entry change alone. To reverse: move a
    field between the layers in the table's schema. (dcn-2611281148505220)
29. **The row's code is split with itd-2611141507299828.** The session's,
    while the product thinker is away (2026-11-28): spc-2611251806215442 owns
    the session state files, the part frame and its three parts, the
    pull-request watcher, the `visible` switches and the first `fit.go`; this
    intent owns the table and its defaults, the three views, the badge rework,
    the explanation, gauges and links, the width measure, and lifting
    `fit.go`'s guard so every row sheds. That spec's steps 1 to 3 land first,
    and this intent re-baselines its golden row, saying so in the pull
    request; built first instead, this intent builds `fit.go` in that spec's
    shape. Grounds: both edit `render.go` and `compose.go`, so one owner per
    file keeps their steps landing one at a time. To reverse: reassign a file
    in both specs' footprints. (dcn-2611281148506801)
30. **The impact stays additive.** The session's, while the product thinker
    is away (2026-11-28): the badge's plain words change (a role, and
    `waiting` beside it, in place of `waiting on the technical facilitator`),
    and the board relays them; the JSON state, the element keys and the off
    switch stay as they are, and the pull request names the wording change.
    Grounds: nothing a reader keys on changes. To reverse: mark it breaking if
    a reader of the plain words turns up. (dcn-2611281148519955)

The session's defaults for the five product trade-offs the planning put to
the person, taken while the person is away (2026-11-28) on the planning
brief's recommendation; each is for the person's review, and a person may
reverse any of them:

31. **The frame ships first.** The table, the three views, the one-role
    badge, fitting the pane width, `--explain`, text mode, the gauges and the
    links are built now, and so are the build lanes, whose runs index step 7
    of the spec builds from the run state already on the default branch;
    each element whose source has not shipped (the review count, the agents
    and spend, the branch's pull-request state) is a stated remainder that
    appears when its source ships (dcn-2611281155245894 listed the build
    lanes among the remainders; step 7 waits on no other record, so they ship
    with the frame); the counts keep their present form, left to iss-2610111726481495.
    Grounds: the badge and narrow-pane rulings reach people now instead of
    waiting on four records not yet built, one of them a draft. To reverse:
    hold the whole row until every source ships, or add the ready and
    fixable counts to the first version. (dcn-2611281155245894)
32. **Under auto there is no mode badge.** Under `facilitator.mode: auto` the
    badge names a role, or `abcd-managed`, as in every mode, and the row's
    `facilitator: auto (project)` element is what says the mode, so it is
    said once; `waiting` appears only under `human`. This replaces decision
    3's `auto-merge` badge, and open question 7 has nothing left to settle.
    Grounds: every row already names the mode in that element
    (itd-2611252117557299), and `auto` now means the whole facilitator role
    (itd-2611252049031692), so a badge word about merging would already be
    out of date. The cost: auto mode loses its own coloured block at the left
    edge. To reverse: give auto a badge with a wider word beside the element,
    or let that badge replace the element under auto. (dcn-2611281201513663)
33. **The product thinker's view leaves the technical elements out.** The
    dirty marker, the branch's pull-request state, the person's open pull
    requests, the peers, the model, the context and the agents never appear
    in that view, whatever the width; this settles open question 2. Grounds:
    it matches decision 7 and the lab's element catalogue, and the view is
    interim until the dashboard (itd-2610032150577708). The cost: a person
    who holds both roles loses those elements on a wide pane whenever an
    answer owed to the product thinker turns the view. To reverse: show them
    in that view and shed them first, as the prototype did, by changing their
    visibility for that view in the bundled table. (dcn-2611281201512624)
34. **A waiting badge an agent set ends at the next session start, and shows
    its age.** The session-start hook sets the mode back to managed only
    when the question marker shows that an agent put the question and the
    session that asked it is the one starting or is no longer live
    (decision 36), so a mode a person set by hand survives and a question a
    live session still holds open is never cleared by another's start; and
    `waiting` carries how long the answer has waited, dated as decision 26
    says. This takes up the ruling
    iss-2609291032194222 has waited on since 2026-09-29, which decision 26
    left open, and the spec step that builds it resolves that issue.
    Grounds: it is the smallest change to the ruled rule that a mode set by
    hand is never cleared, and with its age shown a park a crash left behind
    reads as old, not as current. The cost: such a park stays until some
    session starts, and a question an idle but live session holds stays until
    that session answers or ends. Whether this decision stands is open
    question 10, deferred to the person. To reverse: show the badge only in the session that set
    it, or let it lapse after a set time and show it as aged.
    (dcn-2611281201529151)
35. **The facilitator's views keep the facilitator's words.** In both
    facilitator views the builds element names lane ids, stage names and
    record ids, while the drive's part keeps the product wording
    itd-2611141507299828 rules for both views; the product thinker's view
    keeps outcome words alone, so open question 3 concerns that view only.
    Grounds: the facilitator is the reader who acts on a lane id, and the mix
    is confined to one part whose words a repository can change (decision
    13). The cost: one facilitator row mixes two registers. To reverse: use
    product words in every view and keep the ids to the explanation and the
    lanes page, by changing the builds element's words in the bundled table.
    (dcn-2611281201531202)

The person approved the move to planned on 2026-11-28 at about 12:25Z,
after it was made, in their own words: "correct, I approve", and "I really
would like to see a live version ... we may have to make amendments,
depending on how it looks" (dcn-2611281230532771). Decisions 31 to 35 stay
open to amendment after that preview.

Technical calls the session made after the two adversarial reviews of
2026-11-28; a person may reverse either:

36. **The session-start reset never clears a question a live session holds.**
    A file beside the question marker names the session whose question the
    guard admitted, so the marker itself and every reader of it, an older
    installed release's included, are unchanged; the session-start reset
    acts only where that session is the one starting or is no longer live
    (`occupancy.Live`), leaves a marker of another live session alone, resets
    a marker with no session file beside it only with no other live session
    in the checkout, and resets nothing where the
    live sessions cannot be read. Grounds: the occupancy markers already tell
    a live session from a gone one by its host process; a reset at every
    start would make a second session's start say nobody is waited on while
    an answer is still owed. The cost: a question an idle but live session
    holds stays, with its age, until it is answered or the session ends. To
    reverse: reset at every start where the marker names the stored word, and
    state that cost. (dcn-2611281231066048)
37. **The agents element reads stale only past a heartbeat's bound.** While
    a launch runs, its launcher rewrites the session's rollup every five
    minutes, so the rollup's written time is its running launches' last
    heartbeat; the element reads "as of" only when the rollup counts a
    running launch and is more than twelve minutes old, and a rollup with
    none running never reads stale. Grounds: the row reads no process table
    and the ledger keeps no heartbeat, and the launcher lives exactly as long
    as its launch. To reverse: drop the heartbeat and the time-based "as of".
    (dcn-2611281231064739)

## Open Questions

Questions 2 and 7 are answered by decisions 33 and 32. Questions 1 and 3 to
6 are defaults the build ships at the value stated and the person tunes in
use (decision 13); none holds the intent, and each is listed so a later
tuning can set it. Questions 8 to 11 are the person's to answer, deferred to
them after the live preview they asked for (dcn-2611281230532771); the build
ships each one's stated default meanwhile, and an answer that differs
rewrites the step it names.

1. **Deferred: a default, tuned in use.** Each view's dropping order. The
   prototype's orders are a first guess. Product thinker: work, usage
   windows, spend (the elements decision 33 leaves out of that view never
   enter its order). Facilitator working by hand: peers, builds, agents,
   spend, usage windows, model, open pull requests, context, pull request
   state. Facilitator under auto: peers, model, pull request state, open
   pull requests, context, usage windows, agents, spend, builds.
2. resolved: decision 33 (dcn-2611281201512624). The product thinker's view
   leaves the dirty marker, the pull request state, the person's open pull
   requests, the peers, the model, the context and the agents out entirely.
3. **Deferred: a default, tuned in use.** The product thinker's view's own
   words for the elements it keeps (`in progress`, the spend), in that
   register: outcomes, no ids, no internals (decision 35 keeps the
   facilitator's words in the facilitator's views).
4. **Deferred: defaults, tuned in use.** The thresholds: the review warning
   at 5 reviews or 24 hours; a key's budget warning at 80%; the context
   warning at 80%; the key list collapsing beyond two keys; the shedding
   margin of six columns; the worktree label cut at 24; the refresh interval
   of five seconds; and the form of the waiting age (decision 34).
5. **Deferred: a default, tuned in use.** Whether the collapsed counts keep
   their emoji figures, or follow the warnings into text (decision 8).
6. **Deferred: a default, tuned in use.** The gauge palette on a light
   terminal, which the payload cannot tell the row about (decisions 10 and
   27); the build ships an untuned light ramp.
7. resolved: decision 32 (dcn-2611281201513663). Under auto the row shows no
   mode badge, so no badge word waits on the wider automatic mode.
8. **Deferred to the person after the live preview.** For the product thinker: under `facilitator.mode: auto`, does an answer
   that holds work still show `waiting`, both when it is a product question
   an agent asks and when it is a drive that needs a person? The default
   says no: under `auto` nothing blocks and no `waiting` renders, the badge
   naming the role alone (criterion 2, decision 32, the press release). But
   itd-2611252049031692 says product questions are still asked under `auto`,
   and itd-2611141507299828's criterion, from your decision 4 there, says the
   badge "turns to waiting on the role the drive needs", a drive stalled
   under `auto` included; under the default a product thinker whose answer
   holds a run, or a facilitator a stalled drive needs, sees their role on
   the badge with no `waiting`, the same as a review batch. Spec step 2's
   badge rule, and spc-2611251806215442 step 8 under `auto`.
9. **Deferred to the person after the live preview.** For the product thinker: decision 32 reverses your decision 3, which gave
   auto mode an `auto-merge` badge; does the reversal stand? The default
   says yes: under `auto` no mode badge renders and the `facilitator:`
   element alone says the mode, so with that element switched off nothing on
   the row says it and only `abcd statusline --explain` does. Spec step 2's
   badge rule.
10. **Deferred to the person after the live preview.** For the product thinker: decision 34 settles iss-2609291032194222, whose
    ruling is owed to you; does it stand? The default says yes: a waiting
    badge an agent's question set goes back to `abcd-managed` at the next
    session start, only where the session that asked is the one starting or
    has ended (decision 36), a mode set by hand is never cleared, and
    `waiting` shows its age; spec step 2 builds it and resolves the issue.
11. **Deferred to the person after the live preview.** For the product thinker: may the fill file carry the transcript's offset
    and running token sums, so a refresh reads only the transcript's new
    tail? Your ruling of 2026-10-11 (itd-2610091918433290, "Allow one fill
    file") admitted exactly the one figure, with no cache. The default says
    yes: the file, its path and its one writer are unchanged, and spec step 5
    loosens the two tests that pin its fields; a no keeps the file at its two
    fields and reads the transcript whole on each refresh.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the row is the one surface every session shows, and it is about to carry agents, builds, spend and reviews it cannot fit; we expect naming one role at a time and shedding by the host's width to let a person read it at a glance on any pane, and we are wrong if people still open the board to learn who is waited on, or if the per-view defaults need rewriting before anyone can use them.
