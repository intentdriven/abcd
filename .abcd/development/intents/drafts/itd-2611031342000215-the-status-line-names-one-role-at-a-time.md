---
id: itd-2611031342000215
slug: the-status-line-names-one-role-at-a-time
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611031342011261, itd-2610301750162657, itd-2610240850336695]
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
[itd-2610301750162657](../planned/itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
(the spot-check review queue, which supplies the review count and the age of
its oldest item) and
[itd-2610240850336695](itd-2610240850336695-before-stepping-away-the-person-grants.md)
(the autonomy grant, whose widening of the facilitator's automatic mode will
change the mode badge's word); `refines`
[itd-2610032150577708](../planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md)
(the product thinker's dashboard: the product-thinker view built here is an
interim surface, and what it shows is evidence of what that dashboard must
show).

## Press Release

> The status line names one person at a time: the role whose item has waited longest. An answer that blocks comes first, the oldest of them; with nothing blocking, the role with the oldest review to make; with nothing owed, the row says abcd is managing. A red `waiting` stands beside the role only where an answer really blocks, which is a repository that merges by hand; a repository that merges on its own shows an `auto-merge` badge instead, whose word will follow the facilitator's automatic mode as it widens beyond merging. The reviews that role has to make show as a grey count, never a block, and turn yellow once the batch is large or old. Everything after the badges follows the person reading: the product thinker sees outcomes in plain words; the technical facilitator sees the checkout, their session's own time and context, the usage windows, the agents, the builds and the API spend against its budget, in one order when they merge by hand and another when abcd merges for them. A narrow terminal trims detail inside each element first and drops whole elements last, by the width the host reports, and the row never loses its badge. Every element and every warning has a plain explanation behind it (the value, the threshold it was judged against and where it was read from), which a person opens from the row and an agent reads as data. abcd ships the whole table as sensible defaults and makes all of it configurable from the first version, so the defaults are tuned by use and a repository's maintainers adjust them for their own.

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

- A host harness that runs a status command on its refresh events with a JSON
  payload on stdin and the terminal's width in `COLUMNS`; without the width
  the row renders in full and the badge-first order is its only protection,
  as today.
- A managed checkout: the row renders only where `.abcd/` is present, and the
  previous status command runs everywhere else.
- A terminal that interprets 24-bit colour, and a dark theme for the shipped
  gauge palette; a light theme inverts the gauge's reading and the shipped
  defaults do not cover it.
- Links are hover and click only where the terminal passes OSC 8 through; a
  terminal that does not shows the same text and loses only the link.
- The review count and its age read the spot-check queue
  (itd-2610301750162657) and show only once it ships; the agents, spend,
  API-time and token figures read the launch ledger (itd-2611031342011261)
  and are dropped, never invented, until it ships.
- The terminal is the technical facilitator's surface; the product-thinker
  view is interim, kept until the product thinker's dashboard
  (itd-2610032150577708) carries what it shows.
- The mode badge's word, `auto-merge`, holds while `facilitator.mode` governs
  merging alone; it changes with the wider automatic mode the autonomy grant
  (itd-2610240850336695) describes, and that mode is not this intent's to
  define.

## Acceptance Criteria

Restated from the product thinker's rulings of 2026-11-03; each to be walked
at the planning interview.

- **Given** blocking answers and pending reviews owed to both roles, **when**
  the row renders, **then** the first badge names exactly one role: the role
  of the oldest blocking answer; with no block, the role of the oldest pending
  review; with neither, `abcd-managed`; and the review count beside it is
  that role's own count and no other's.
- **Given** a blocking answer owed to the named role and a repository whose
  `facilitator.mode` is `human`, **when** the row renders, **then** `waiting`
  follows the role badge as red text with no background, and it is the only
  badge state rendered that way; **given** `facilitator.mode` is `auto`,
  **then** an `auto-merge` badge follows the role badge and `waiting` never
  renders, whatever is owed.
- **Given** a checkout whose working tree's `facilitator.mode` differs from
  the default branch's committed one, **when** the row renders, **then** the
  mode badge follows the committed one, as the merge gate reads it.
- **Given** reviews pending for the named role below both thresholds,
  **when** the row renders, **then** the count renders as grey text
  (`manual reviews 3`) with no background and no warning symbol; **given**
  the count reaches the count threshold or its oldest item reaches the age
  threshold, **then** the number renders yellow behind a yellow `!`, never
  red; and the thresholds are read from the repository's setting over the
  machine's over the bundled default (5 reviews, 24 hours).
- **Given** reviews pending for the product thinker and no blocking answer
  owed to them, **when** the row renders, **then** the badge names the
  product thinker but the view is the facilitator's; **given** a blocking
  answer owed to the product thinker, **then** the view is the product
  thinker's; and the view is never read from a role setting.
- **Given** the product-thinker view and build lanes underway, **when** the
  row renders, **then** work in progress reads as a count of distinct pieces
  of work (`in progress 2`), and the whole row carries no record id, no lane
  id and no stage name; the two facilitator views keep the builds element
  with its lanes and stages.
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
  **then** the row renders with that entry replaced and every other entry at
  its bundled default; the repository's entry wins the machine's; an unknown
  key under the namespace is refused by name; and the badge colours and the
  text mode are never read from the repository layer.
- **Given** the same payload, checkout and settings, **when**
  `abcd statusline --explain` runs, **then** it emits one JSON record naming
  the view and, for every element, its value, whether it warns, why (the
  reading against its threshold) and the source it was read from; two runs
  over the same input emit the same record; and each warning symbol on the
  row links to a rendering of that record.
- **Given** a row showing a pull request number or a builds count, **when**
  the terminal passes OSC 8 links through, **then** the number links to the
  pull request and the count to a local page listing each lane and its
  stage, rewritten on each refresh; a terminal that does not pass links
  through shows the same text.
- **Given** the install step wiring the status line into the harness,
  **when** it writes the harness's setting, **then** it sets a refresh
  interval of five seconds by default, the person can change or remove it in
  the same setting, and uninstall restores what was there before.
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
- **Given** API-billed launches recorded against the session with their
  keys, **when** the row renders the spend element, **then** each key shows
  its spend against its budget, a key at or past 80% of its budget renders
  yellow, the list collapses to a summary (`3 keys $142/$450`) plus any key
  past its share once there are more keys than the configured limit (two),
  and a session billed to the subscription shows no cost figure.
- **Given** a session's own figures (its API time, lines changed, running
  token total, the agents it launched and their spend), **when** the row
  renders in any session, **then** every figure is that session's own, read
  from its payload, its transcript and the launches recorded against its
  session id, never another session's and never a sum across sessions; the
  usage windows, which the host reports for the account, and the badges,
  which are the repository's, are the only figures shared.
- **Given** the row refreshing, **when** a forge read is needed (the
  branch's pull request, the person's open pull requests), **then** the row
  reads a cache a detached refresh fills at most once a minute, and the row
  itself never waits on the network, staying inside its existing time
  ceiling.

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

## Open Questions

Each is shipped as a default the person tunes in use (decision 13), not a
question that holds the intent; they are listed so the planning interview can
set the first values.

1. Each view's dropping order. The prototype's orders are a first guess.
   Product thinker: peers, work, model, pull request state, open pull
   requests, context, agents, usage windows, spend. Facilitator merging by
   hand: peers, builds, agents, spend, usage windows, model, open pull
   requests, context, pull request state. Facilitator on auto-merge: peers,
   model, pull request state, open pull requests, context, usage windows,
   agents, spend, builds.
2. Which elements the product-thinker view leaves out entirely rather than
   drops late. The catalogue leaves out the dirty marker, the pull request
   state, the person's open pull requests, the peers, the model, the context
   and the agents; the prototype only sheds them.
3. The product-thinker view's own words for the elements it keeps (`in
   progress`, the spend), in that register: outcomes, no ids, no internals.
4. The thresholds: the review warning at 5 reviews or 24 hours; a key's
   budget warning at 80%; the context warning at 80%; the key list
   collapsing beyond two keys; the shedding margin of six columns; the
   worktree label cut at 24; the refresh interval of five seconds.
5. Whether the collapsed counts keep their emoji figures, or follow the
   warnings into text (decision 8).
6. The gauge palette on a light terminal, which the payload cannot tell the
   row about (decision 10).
7. The mode badge's word once the wider automatic mode ships (decision 3).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the row is the one surface every session shows, and it is about to carry agents, builds, spend and reviews it cannot fit; we expect naming one role at a time and shedding by the host's width to let a person read it at a glance on any pane, and we are wrong if people still open the board to learn who is waited on, or if the per-view defaults need rewriting before anyone can use them.
