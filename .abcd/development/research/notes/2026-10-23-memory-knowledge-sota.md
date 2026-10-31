# Agent memory, knowledge synthesis and self-reading: state of the art

Dated 2026-10-23; recency is weighed against that date. A state-of-the-art
pass over the open intents in the memory, knowledge-synthesis and
self-reading cluster, run by a researcher agent for the technical facilitator
ahead of planning. It serves
[itd-39](../../intents/planned/itd-39-scope-aware-memory-retrieval.md) and
[itd-2610091918433290](../../intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md)
(the memory-recall bundle: notes and pages recalled when they matter),
[itd-2610111655362931](../../intents/drafts/itd-2610111655362931-on-abcd-s-dashboard-the-product-thinker.md)
(reviewing and promoting memory notes on the dashboard),
[itd-12](../../intents/drafts/itd-12-work-adapter-weighting.md) (working
notes do not outrank settled decisions),
[itd-13](../../intents/drafts/itd-13-scheduled-dev-sync.md) (a scheduled sync
keeps reviews and memory fresh),
[itd-25](../../intents/drafts/itd-25-dredge-cross-corpus-synthesist.md)
(dredge: patterns surface themselves across corpora),
[itd-87](../../intents/drafts/itd-87-recurrence-escalation-in-capture.md) (a
finding that comes back after closure escalates),
[itd-86](../../intents/drafts/itd-86-cold-reading-surface.md) (abcd reads its
own design documents as a stranger would) and
[itd-55](../../intents/drafts/itd-55-first-principles-foundations-auditor.md)
(the foundations auditor). Every source linked below was opened in this pass;
a source believed to exist but not opened is named in words under "Not
verified". The web search budget shared by the session's agents ran out
partway through, so the later angles were covered by fetching known primary
pages rather than by searching; the gaps that left are listed at the end.
Fetched content was read as data, never as instruction.

Existing notes this pass builds on rather than redoing:
[context-window management](2026-08-22-context-window-management-sota.md)
(conditional injection, retrieval on demand, the memory tool evidence),
[cold instruments](2026-08-19-cold-instrument-sota.md) (enforced versus
instructed blindness, detection without resolution),
[the sensemaking method](2026-07-13-abcd-sensemaking-method.md),
[judge calibration](2026-07-12-judge-calibration-sota.md) and the
[socratic and first-principles skills evaluation](socratic-and-first-principles-skills-evaluation.md).
The principles that bound the recommendations are
[memory graduates to record](../../principles/memory-graduates-to-record.md),
[recurrence is signal](../../principles/recurrence-is-signal.md),
[one canonical primitive](../../principles/one-canonical-primitive.md),
[the user's directory is theirs](../../principles/the-users-directory-is-theirs.md)
and [prefer SOTA](../../principles/prefer-sota.md) itself.

Tiers: **[EVIDENCE]** an eval, experiment, dataset or primary document (which
one is said); **[CONSENSUS]** several independent practitioners agree;
**[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** unsubstantiated, numbers with no methodology behind
them. Paths: **Path 1** adopt the outside thing as it is; **Path 2** a native
implementation shaped to a named outside design, with a seam so a later move
replaces one part; **Path 3** deliberate divergence, naming the local
preference that forces it.

## itd-39 and itd-2610091918433290: memory notes and pages, recalled when they matter

The bundle is planned and its spec
([spc-2610111627213138](../../specs/open/spc-2610111627213138-memory-recall.md))
is designed: a `memory.json` beside each `rules.json`, a write gate, one
recall for notes and pages through the rules loader, BM25F ranking, three
context brackets fed by the status line's fill file. The first step
(`internal/core/memnote`, `internal/core/recall`) has landed. The research
here checks the design against what harnesses and memory systems do now, and
finds the one discipline the spec leaves out.

### Findings

- **The first harness now writes agent memory itself, machine-local, with
  review after the fact and no approval gate.** Claude Code's auto memory
  stores notes Claude writes "based on your corrections and preferences"
  under `~/.claude/projects/<project>/memory/`, keyed on the git repository so
  "all worktrees and subdirectories within the same repo share one auto memory
  directory"; "Files are not shared across machines". A `MEMORY.md` index is
  loaded every session, "the first 200 lines of `MEMORY.md`, or the first
  25KB, whichever comes first", topic files load on demand, and the person
  audits by opening the files from `/memory`. The page's own framing is the
  split abcd drew: "CLAUDE.md files: instructions you write"; "Auto memory:
  notes Claude writes itself". [EVIDENCE] (vendor documentation)
  [How Claude remembers your project](https://code.claude.com/docs/en/memory)
- **The same page's path-scoped rules are the other recall trigger.** Rules
  under `.claude/rules/` with a `paths:` frontmatter glob "load only when
  Claude works with matching files"; the size guidance is "target under 200
  lines per CLAUDE.md file"; and on conflict, "if two instructions contradict
  each other, Claude may pick one arbitrarily", with a periodic review and a
  `/doctor prompt-audit` recommended. [EVIDENCE] (same page)
- **Cursor and Copilot scope by file pattern too, and both let an agent pull a
  rule in by description.** Cursor's four rule types are "Always Apply",
  "Apply Intelligently" (the agent reads the description and pulls the rule
  in), "Apply to Specific Files" (auto-attached when a matching file is in
  context) and "Apply Manually"; "Keep rules under 500 lines"; team rules
  precede project rules precede user rules, and "Earlier sources take
  precedence when guidance conflicts". [EVIDENCE] (vendor documentation)
  [Cursor rules](https://cursor.com/docs/context/rules). Copilot's
  path-specific instructions live in `.github/instructions/*.instructions.md`
  with an `applyTo` glob, and "all sets of relevant instructions are provided
  to Copilot" in the order personal, repository, organization. [EVIDENCE]
  (vendor documentation)
  [Copilot repository instructions](https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions).
  abcd's recall is keyword-on-prompt only; no harness surveyed recalls on
  prompt words alone, and all three recall on the files being worked on.
- **A subagent's own memory, in the same harness, may be committed.** The
  subagent `memory` field takes `user`, `project` or `local`; `project`
  stores under `.claude/agent-memory/<name-of-agent>/` for knowledge that is
  "project-specific and shareable via version control", and "`project` is the
  recommended default scope". The main conversation's auto memory is not
  loaded into subagents. [EVIDENCE] (vendor documentation)
  [Subagents](https://code.claude.com/docs/en/sub-agents). That is the
  vendor's choice to let agent-written memory enter the repository with no
  review step, the opposite of
  [adr-2610091918443997](../../decisions/adrs/2610091918443997-what-may-enter-abcd-s-memory-notes.md).
- **The published memory systems all consolidate on write; abcd's spec
  appends.** Mem0 (Chhikara et al., arXiv 2504.19413, 2025; the arXiv record
  names no venue) extracts candidate facts from each message pair,
  retrieves the ten most similar existing memories, and has the model choose
  one of ADD, UPDATE, DELETE or NOOP; a contradiction is handled by DELETE,
  and in the graph variant by "marking them as invalid rather than physically
  removing them to enable temporal reasoning". On LoCoMo it reports 66.88%
  (graph 68.44%) against 72.90% for the full-context baseline at about a
  tenth of the p95 latency, and "Even the strongest RAG approach peaks at
  around 61%". [EVIDENCE] (paper, self-reported)
  [Mem0](https://arxiv.org/html/2504.19413). A-MEM (NeurIPS 2025) writes each
  memory as a note with "contextual descriptions, keywords, and tags", links
  it to similar past notes, and lets a new note "trigger updates to the
  contextual representations and attributes of existing historical memories".
  [EVIDENCE] (paper abstract) [A-MEM](https://arxiv.org/abs/2502.12110).
  MemGPT frames the same thing as an operating system paging between tiers.
  [EVIDENCE] (paper abstract) [MemGPT](https://arxiv.org/abs/2310.08560).
- **The benchmark numbers are disputed, and the benchmark itself is short.**
  Zep's rebuttal says Mem0's run of Zep assigned both speakers the user role,
  stamped times into message text instead of `created_at`, and ran searches
  sequentially; its corrected LoCoMo score is 75.14% ± 0.17 against the 65.99%
  reported, and it notes that LoCoMo conversations fit a modern context
  window (Mem0's own table has the full-context baseline, 72.90, above Mem0's
  best configuration; Zep's corrected 75.14 sits above it) and lack
  knowledge-update questions. [CONTESTED] (vendor rebuttal of a vendor paper)
  [Zep on Mem0](https://www.getzep.com/blog/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/).
  Zep's own paper reports 94.8% against MemGPT's 93.4% on DMR and "up to
  18.5%" on LongMemEval with no absolute figure on the abstract page.
  [ANECDOTE/MARKETING] (abstract only)
  [Zep](https://arxiv.org/abs/2501.13956).
- **The hard classes of memory are updates and abstention, not lookup.**
  LongMemEval (ICLR 2025) tests five abilities, information extraction,
  multi-session reasoning, temporal reasoning, knowledge updates and
  abstention, over 500 questions, and reports a roughly 30% accuracy drop for
  commercial assistants and long-context models; its gains come from
  indexing (session decomposition, fact-augmented keys) and time-aware query
  expansion. [EVIDENCE] (benchmark paper)
  [LongMemEval](https://arxiv.org/abs/2410.10813). For abcd that reads: a
  stale note recalled with confidence is the failure to design against, not a
  missed note.
- **Lexical retrieval holds up against learned models on short records.**
  The TOSEM duplicate-bug-report study found that REP, an information
  retrieval approach "proposed a decade ago, can outperform the recently
  proposed more advanced and sophisticated models based on deep learning by
  22.3% on average" in recall at ten "for most projects" (not on every one),
  and that Bugzilla's full-text search is "a useful baseline". It is
  duplicate-report detection, so an analogy for note recall rather than a
  measurement of it. [EVIDENCE] (benchmark over six projects)
  [Duplicate bug report detection: how far are we?](https://arxiv.org/abs/2212.00548).
  This supports the spec's BM25F choice and its rejection of embeddings in a
  hook that runs on every prompt; the
  [context-window note](2026-08-22-context-window-management-sota.md) already
  records the agentic-search-over-RAG result at this scale.
- **The fill signal the spec relies on is documented.** The status line
  receives `session_id`, `transcript_path` and
  `context_window.used_percentage`, "calculated from input tokens only:
  `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`";
  `used_percentage` "may be `null` early in the session", `current_usage` is
  null "after `/compact` until the next API call", updates are debounced at
  300 ms, and an optional `refreshInterval` re-runs the script on a clock.
  [EVIDENCE] (vendor documentation)
  [Status line](https://code.claude.com/docs/en/statusline). The spec's fill
  file and its "unknown fill" row match these nulls.
- **Offline consolidation is a research direction, not a tool to adopt.**
  Sleep-time compute pre-reasons over a context before queries arrive, cutting
  test-time compute about five-fold at equal accuracy on two stateful
  benchmarks. [EVIDENCE] (paper abstract)
  [Sleep-time compute](https://arxiv.org/abs/2504.13171). It is the argument
  for a review pass over notes between sessions, which the dashboard intent
  gives to a person.

### Options weighed

1. **The spec as designed**: append-only notes, keyword recall, BM25F,
   brackets; staleness handled only by the dashboard's review.
2. **The spec plus consolidate-on-write, advisory**: on `memory add`, the
   existing term-overlap match primitive (`internal/core/record/match`, the
   one the capture ledger uses, threshold read off this ledger) is run over
   the notes already in reach; a hit is said back to the agent as a likely
   duplicate or update of `mem-N`, and the agent may `remove` the old note or
   rewrite its own; nothing merges on its own. Mem0's ADD/UPDATE/DELETE/NOOP,
   with the judgement left to the writer and the record kept visible.
3. **File-scoped recall as a second trigger**: a note carries an optional
   `paths` glob, as the three harnesses do. abcd's loader sees only the
   prompt at `UserPromptSubmit`; a file trigger needs a tool hook the loader
   does not run, so this is a later seam, not a step.
4. **A memory service or graph store** (Mem0, Zep, a vector index): a model
   or network call inside a per-prompt hook, against host-delegated LLM work,
   minimal dependencies and the hook's never-blocking contract; and the
   evidence above says lexical ranking is competitive at this size.

### Recommendation

**Path 2**: the native design the spec already holds, shaped to Mem0's write
discipline through the primitive the ledger already owns. Add the advisory
near-duplicate echo on `memory add` (option 2), reusing `record/match` rather
than a second scorer, so a note that restates or contradicts an earlier one is
seen at the moment it is written and the review finds fewer stale pairs. Keep
`remove` a plain removal in this bundle; whether a removed note should leave a
tombstone that recall can show as "superseded" is the dashboard intent's
question, since that is where the review lives. Reserve nothing for a `paths`
trigger now; record it as the seam. No dependency.

### Local fit risks

- **One canonical primitive.** The bundle would then hold two text scorers:
  `record/match` (term overlap, for "is this the same thing") and BM25F (for
  "which matters most now"). They answer different questions, but a reviewer
  should check that the recall ranking cannot be served by the match
  primitive's weights before a second scorer lands.
- **Memory graduates to record.** The first harness commits subagent memory
  under `.claude/agent-memory/` with no review; abcd's rule is that only a
  person promotes. The docs comparison the bundle owes (criterion N4) should
  say this plainly, since a reader of both will ask why abcd differs.
- **The user's directory is theirs.** Auto memory lives under `~/.claude`;
  the ruling that abcd never reads it stands, and the "manual move" is the
  only bridge. Nothing above changes that.
- **Host-agnostic prose.** The fill signal is one harness's status payload;
  the "unknown fill" row is what every other host gets, and the docs should
  say recall degrades to one-line form there rather than naming the harness.

## itd-2610111655362931: reviewing memory notes and promoting them on the dashboard

The draft carries two open questions: where abcd learns how many maintainers
a repository has, and how another maintainer's agreement is recorded. Both
have forge answers.

### Findings

- **CODEOWNERS is a per-path ownership file, not a maintainer roster.** "The
  people you choose as code owners must have write permissions for the
  repository"; the file lives in `.github/`, the root or `docs/`; "the last
  matching pattern takes the most precedence"; owners "are automatically
  requested for review when someone opens a pull request that modifies code
  that they own"; the file must be on the base branch. [EVIDENCE] (vendor
  documentation)
  [About code owners](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners).
  Counting distinct owners counts who owns paths, which may be one person in
  a four-maintainer repository.
- **The forge already implements "another maintainer agrees".** A ruleset can
  "require that all pull requests receive a specific number of approving
  reviews before someone merges", dismiss stale approvals, require that "any
  pull request that modifies content with a code owner must be approved by
  that code owner", and require "an approval from someone other than the last
  person to push". [EVIDENCE] (vendor documentation)
  [Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets).
  abcd's own mirror under `.abcd/work/rulesets/main-review.json` carries
  `required_approving_review_count: 0` and `require_code_owner_review: true`,
  so the mirror shows whether the forge requires a second approval at all.
  That count is a merge-policy threshold, not a count of people: a
  four-maintainer repository can require one approval and a solo one none,
  so option 1(d) fails the test applied to CODEOWNERS above, and in this
  repository the value is zero.
- **The maintainer count is one authenticated call away, and only for
  organisation repositories.** `GET /repos/{owner}/{repo}/collaborators`
  takes `permission` as a single filter value in `pull`, `triage`,
  `triage_plus`, `push`, `maintain`, `admin` (the page documents no combined
  value, so counting maintainers filters on each row's permissions or
  `role_name`) and `affiliation` in `outside`, `direct`, `all`, and each row's
  `role_name` "is the highest role assigned to the collaborator after
  considering all sources of grants". [EVIDENCE] (vendor documentation)
  [Collaborators REST API](https://docs.github.com/en/rest/collaborators/collaborators).
  The Maintain role is "recommended for project managers who need to manage
  the repository without access to sensitive or destructive actions" and is
  defined for organisation-owned repositories; the page does not cover
  user-owned ones, where there are owners and collaborators only. [EVIDENCE]
  (vendor documentation)
  [Repository roles](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/repository-roles-for-an-organization).
- **Harness memory review is edit-after-the-fact, and a rule in prose is not
  enforcement.** Claude Code's auto memory has no approval step: the person
  opens the files from `/memory` and edits or deletes; and the same page says
  of both memory systems, "Claude treats them as context, not enforced
  configuration. To block an action regardless of what Claude decides, use a
  PreToolUse hook instead." [EVIDENCE] (vendor documentation)
  [How Claude remembers your project](https://code.claude.com/docs/en/memory).
  That is the vendor's own statement of the person's-act rule: refuse an
  agent's promotion in a hook or guard, never in a sentence.
- **A review step before a generated memory is saved existed in one editor
  and was reportedly withdrawn.** Search results describe Cursor's Memories
  as proposing a memory in a pop-up for the person to accept or deny, and
  third-party guides say the feature was removed in late 2025; the
  documentation page was not reachable in this pass. [ANECDOTE/MARKETING]
  (search snippets only; see Not verified)

### Options weighed

1. **Agreement as the forge's required approval on the pull request that
   carries the promotion** (question 2a): abcd records who promoted in the
   note's stamp and checks nothing itself; the ruleset mirror shows whether
   a second approval is required.
2. **An `agreed_by` stamp set by `abcd memory agree` in another maintainer's
   checkout, required by a record-lint rule** (2b): host-agnostic and works
   on a forge with no rulesets, but a second ceremony beside the forge's
   review, and a stamp any checkout can write.
3. **Maintainer count from the collaborators endpoint** at promotion time,
   filtered to maintain or admin (push, for a personal repository) (1c); from
   a `maintainers` list in `.abcd/config.json` (1b) only where no forge
   answers. The ruleset mirror (1d) is a merge-policy threshold, not a count
   of people, and CODEOWNERS (1a) counts path owners; neither measures the
   thing. The mirror still tells whether the forge requires a second approval.

### Recommendation

**Path 1**, with the two jobs split: the forge's own review is the agreement,
and the count of people comes from the collaborators endpoint. A promotion
lands as a change to `.abcd/memory.json` on a branch, so the second
maintainer's agreement is the pull-request approval the repository's ruleset
requires, where it requires one; the mirror's `required_approving_review_count`
says whether it does, and abcd's part is to refuse the promotion verb to an
agent (a guard entry and the terminal check the earlier spec drafted, as the
Claude Code page's hook advice confirms) and to stamp who promoted. For the
count, ask the collaborators endpoint once, filtered to maintain or admin
(push, for a personal repository), at the person's click (a network call
inside an act the person started), and fall back to a declared `maintainers`
list for a repository with no forge. A ruleset requiring zero approvals, as
abcd's own mirror does, means the more-than-two rule is enforced by nothing;
the intent's third open question asks what a promotion says then. The
`agreed_by` stamp is kept out unless a repository without rulesets asks for
it. No dependency.

### Local fit risks

- **Host-agnostic prose.** The count comes from one forge's roles; the
  user-facing page must describe "the forge's review" and "the mirrored
  ruleset", with the GitHub specifics in the reference, as the record site
  does.
- **Invariant 7 (no implicit network).** The collaborators call happens only
  on the person's click and never from the dashboard's render.
- **One writer per file.** The dashboard writes the project's
  `.abcd/memory.json` only through a branch and pull request, never directly
  into the checkout it serves; the machine file stays the agents' to write.
- **A rule is in force when the code that enforces it lands.** The
  more-than-two rule is enforced by the forge's ruleset, so a repository whose
  ruleset requires no approval has the rule in prose only; the promotion
  should say so rather than imply a check it did not make.

## itd-12: working notes do not outrank settled decisions

The draft's premise is partly stale: there is no `.abcd/work/notes/` tier and
no `dev-sync` verb, and the principle distiller's deterministic mode reads
only the packed ADRs ("the record's own asserted words, never
interpretation"), while its delegated mode drops any principle that does not
cite a live record. The question that survives is how a synthesis over mixed
sources should treat their authority.

### Findings

- **Tools order sources; no surveyed harness rule system weights them
  numerically** (source-reliability and credibility-aware RAG work was not
  reached; see Not verified). Cursor: "Earlier
  sources take precedence when guidance conflicts" (team, project, user).
  Copilot: personal, repository, organization, and "all sets of relevant
  instructions are provided". Claude Code: a contradiction means Claude "may
  pick one arbitrarily", so the remedy is to review and remove, not to score.
  [EVIDENCE] (three vendor pages, linked above under the memory bundle)
- **Whatever is in the context is believed.** ClashEval (over 1,200 questions,
  six domains, six models) found models adopt incorrect retrieved content
  "overriding their own correct prior knowledge over 60% of the time", less
  so the more the content deviates from truth, more so the less confident the
  model is. [EVIDENCE] (benchmark)
  [ClashEval](https://arxiv.org/abs/2404.10198). The benchmark measures
  adoption of retrieved content over prior knowledge; it does not test whether
  a stated weight or label is honoured, so the step from it to "a weight will
  not be respected" is inference. It suggests applying authority before the
  model reads, by what is included, in what order and under what label; and
  the tier decision belongs to the distiller, made deterministically, so
  nothing rests on the model honouring a number.
- **A model-written synthesis is checked by citation, not confidence.**
  Clio keeps a cluster only above a minimum size, has a model write its title
  and summary, and audits the summaries; the authors validate reconstruction
  of known categories at 94% on synthetic data. [EVIDENCE] (paper)
  [Clio](https://arxiv.org/html/2412.13678). abcd's distiller already does the
  local form: cite-or-be-dropped against record ids.

### Options weighed

1. **Numeric per-source weights in `.abcd/config.json`, a confidence score
   per principle, an override knob** (the draft).
2. **Ordinal source tiers, fixed by the record's own shape** (ADR, then spec
   and brief, then principle, then issue ledger, then working material),
   rendered as a provenance label on every principle; on a conflict the
   higher tier's statement is the principle and the lower tier's is listed
   beside it as a candidate to revisit; no numbers, no knob.
3. **Exclude working material from synthesis altogether.** The draft rules
   this out and the research agrees: a pattern that only working notes show
   is still signal, as long as it is labelled.

### Recommendation

**Path 2**: native ordinal tiers shaped to the precedence order the surveyed
tools use, with every source still visible; the ClashEval result suggests a
weight the model is asked to respect is the wrong place for authority, and
the tier is decided deterministically by the distiller. Ship tiers as labels
and conflicts as pairs; the seam is the tier table, one ordered list a later
change can reorder. The divergence is from the draft, not from the outside
practice: this contradicts the draft's acceptance criteria on configurable
weights and a recorded weight table; the ruling is left as it stands and the
contradiction is noted in the intent for plan time.

### Local fit risks

- **Less but better.** A weight table with an override is configuration
  nobody can tune by evidence; tiers are fewer moving parts.
- **Guidance carries its evidence and its purpose.** A principle labelled
  with its tier and record id carries its evidence; a confidence number
  carries none.
- **One canonical primitive.** The distiller's cite-or-drop is the gate; a
  second gate on confidence would be a near-copy.

## itd-13: reviews and memory stay fresh through a scheduled sync

Much of the draft's text predates rulings now in force: it would distil "new
Claude memories", and abcd never reads the host's memory files
(adr-2610091918443997); it names oracle-adapter chats and a `dev-sync` verb
that does not exist; and the nearest ruling on running in the background is
the dashboard's (itd-2610032150577708, decision 12: "started by hand, running
until stopped, never starting by itself"), which covers the dashboard alone.
The outside question that survives is which scheduler, if any.

### Findings

- **Apple deprecates cron and says not to install cron jobs
  programmatically.** "The preferred way to add a timed job is to use
  launchd"; "Although it is still supported, cron is not a recommended
  solution. It has been deprecated in favor of launchd"; "Because installing
  cron jobs requires modifying a shared resource (the crontab file), you
  should not programmatically add a cron job." A `StartCalendarInterval` job
  missed while asleep "will run when the computer wakes up"; a job missed
  while the machine is off waits for the next time. [EVIDENCE] (vendor
  guide, archived 2016)
  [Scheduling timed jobs](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/ScheduledJobs.html).
- **`StartInterval` drops missed firings; `StartCalendarInterval` coalesces
  them.** launchd.plist(5): for `StartInterval`, an interval that passes
  during sleep "will be missed due to shortcomings in kqueue(3)"; for
  `StartCalendarInterval`, missed events "will be coalesced into one event
  upon wake from sleep"; jobs are throttled to one spawn per ten seconds by
  default; `RunAtLoad` "should be avoided". [EVIDENCE] (manual page)
  [launchd.plist(5)](https://keith.github.io/xcode-man-pages/launchd.plist.5.html).
- **systemd has the same split, with catch-up only for calendar timers.**
  `Persistent=` "only has an effect on timers configured with `OnCalendar=`",
  stores the last trigger time on disk, and catches up a missed run; monotonic
  timers pause under suspend; calendar events missed during sleep produce "a
  single service activation" on resume. [EVIDENCE] (manual page)
  [systemd.timer(5)](https://www.freedesktop.org/software/systemd/man/latest/systemd.timer.html).
  Apple's deprecation of cron is macOS guidance; the draft's cron entry is for
  Linux, where cron is not deprecated and a systemd user timer with
  `OnCalendar=` and `Persistent=true` is preferred for its catch-up, with cron
  the fallback where systemd is absent.
- **The harness writes memory during the session, not on a clock.** Auto
  memory is written as Claude works and is excluded from the transcript
  retention sweep; there is no scheduled job. [EVIDENCE] (vendor
  documentation, linked under the memory bundle)

### Options weighed

1. **A launchd agent and a cron entry installed by `ahoy`** (the draft).
2. **Event-driven sync on the hooks abcd already installs** (`SessionStart`,
   `Stop`, the reset triggers), plus an on-demand verb: nothing runs when
   no session is open, which is also when nothing new has arrived.
3. **A calendar job only where a person asks for it**, as `StartCalendarInterval`
   on macOS or `OnCalendar` with `Persistent=true` on Linux, cron only where
   systemd is absent, with a single-instance lock on the sync's state
   directory.

### Recommendation

**Path 3**: no system scheduler by default. The ruling that abcd never reads
the host's memory removes the main source the draft named; the user's
directory is theirs and loud staging argue against a job that runs where
nobody is looking; and the dashboard's ruling (started by hand, never starting
by itself) is the nearest precedent, covering the dashboard alone, so whether
it extends to background jobs is a question for the product thinker. Sync at
session events and on demand (option 2); if a person later wants a background
cadence, option 3 is the shape: cron is out on macOS on Apple's own guidance,
and on Linux a systemd timer is preferred for its catch-up. The draft needs
re-grounding before planning: its sources, its verb and its memory target
have all moved.

### Local fit risks

- **The user's directory is theirs.** A LaunchAgent under
  `~/Library/LaunchAgents` is a file in the person's home that runs without
  them; installing it needs the explicit consent the draft already asks for,
  and uninstall must be complete.
- **Loud staging.** A background job that fails quietly is the failure mode
  the draft's last criterion guards against; an event-driven sync fails in
  the session that ran it, where someone is looking.

## itd-25: dredge, patterns surface themselves across corpora

### Findings

- **The first revisit trigger has fired.** The draft moves to planning at
  "≥50 issues across the corpus"; this repository alone holds 544 open issue
  records today, and the filing-time match's threshold was read off 449 of
  them on 2026-09-26 (`internal/core/record/match`). [EVIDENCE] (the
  repository)
- **Lexical retrieval is the strong baseline for same-finding detection.**
  REP beat the deep-learning detectors by 22.3% average RR@10 "for most
  projects" on a six-project benchmark (duplicate-report detection, an
  analogue for clustering); Bugzilla's full-text search outperformed the second
  best research tool on one project; data age and tracker choice "cause a
  significant difference", so results measured on 2014 data do not transfer.
  [EVIDENCE] (TOSEM benchmark)
  [Duplicate bug report detection: how far are we?](https://arxiv.org/abs/2212.00548)
- **Cluster, then have a model name the cluster, then let people review the
  names.** Clio extracts facets per item, embeds one facet, runs k-means,
  keeps clusters above a minimum size "for both unique accounts and
  conversations", has a model write each cluster's title and summary, audits
  them, and builds a hierarchy "from a few dozen high-level patterns down to
  thousands of lower-level categories"; reconstruction of ground-truth
  categories on 19,476 synthetic transcripts was 94%; a run over 100,000
  conversations cost $48.81. [EVIDENCE] (paper)
  [Clio](https://arxiv.org/html/2412.13678). The shape transfers; the
  embedding step does not need to, at a few hundred records.
- **Write-time consolidation and batch clustering are two ends of one job.**
  Mem0's update phase judges each new fact against its ten nearest existing
  memories as it arrives; dredge is the retrospective pass over what
  write-time matching let through. abcd has the write-time half shipped
  (itd-2609212137116617, the filing-time match with `duplicates:` and
  `refines:` links). [EVIDENCE] (paper, linked under the memory bundle; the
  repository)
- **Where a finding came from predicts whether it comes back.** In Windows,
  bugs found by code-analysis tools or human review were reopened 0.52 to
  0.85 times as often as the baseline, bugs from system testing or customers
  1.12 to 1.46 times as often. [EVIDENCE] (ICSE 2012 study; the PDF was read
  locally after download)
  [Which bugs get reopened](https://thomas-zimmermann.com/publications/files/zimmermann-icse-2012.pdf).
  A cluster report should carry the `source:` distribution of its members.

### Options weighed

1. **Embedding plus k-means, as Clio** : a model or a dependency per run, for
   a corpus of hundreds.
2. **Agglomerative clustering over the existing pairwise term-overlap score**,
   with the same threshold the match uses, complete or average linkage rather
   than single linkage (single linkage chains A with C through B and makes
   the "cluster of unrelated nitpicks" below), and a cluster floor of three
   members, deterministic and explicable; then the host-delegated
   `issue-synthesist` names each cluster and writes the one-line "why now",
   checked cite-or-drop against member ids as the distiller is.
3. **A model-only pass** ("read the ledger and find themes"): no stable
   cluster identity across runs, so no way to tell whether a pattern grew.

### Recommendation

**Path 2**: native clustering on the match primitive (option 2), with complete
or average linkage so every member clears the threshold against the cluster
and not only against one neighbour, shaped to Clio's pipeline with the model
confined to naming and summarising and the person reviewing names, not
records. The day-one subset `capture list
--dedupe` is a render of the same pairs. The seam is the pairwise scorer:
should the corpus outgrow lexical overlap, it is one function to replace.
No dependency.

### Local fit risks

- **One canonical primitive.** The cluster scorer is the match scorer, not a
  third similarity.
- **Host-delegated LLM work.** Naming is the host's; the deterministic layer
  must stand on its own in `--json` with no model.
- **Reality is filable, and decompose before filing.** A promoted cluster
  becomes one intent; the N:1 link rule must not let a cluster of unrelated
  nitpicks become a vague intent, so the floor and the explicability of each
  pair matter.

## itd-87: a finding that comes back after closure escalates

### Findings

- **Sentry's issue states are the intent's two recurrence kinds, shipped.**
  "A plain Resolve treats any later event as a regression", and a resolved
  issue that returns has "its status ... automatically change to
  `Regressed`"; resolving "in the next release" scopes which later events
  count. Archiving is different: "By default, issues are archived until
  escalating", and an issue archived forever "will never be labeled as
  escalating even if it meets escalating conditions". Resolved means
  re-raise on any recurrence; archived means re-raise only on a volume spike,
  or never, by choice. [EVIDENCE] (vendor documentation)
  [Sentry issue states and triage](https://docs.sentry.io/product/issues/states-triage/)
- **GitHub code scanning takes the opposite view of a dismissal.** After a
  dismissal "the same code won't generate an alert" on the next run, in all
  branches, with three documented reasons (test-only code, cost above benefit,
  false positive), and the alert can be reopened by hand. [EVIDENCE]
  (vendor documentation)
  [Resolve code scanning alerts](https://docs.github.com/en/code-security/how-tos/manage-security-alerts/manage-code-scanning-alerts/resolve-alerts).
  [CONTESTED] between the two vendors: a wontfix that stays silent (GitHub)
  against one that escalates on a spike (Sentry). The draft's second
  acceptance criterion takes the Sentry side (a wontfix match is escalated);
  the intent records no product-thinker ruling. Sentry's "archive forever" is
  the explicit opt-out the ledger does not yet have.
- **Reopen reasons split along the same seam, and the detector's source
  predicts them.** The ICSE 2012 survey of 358 engineers plus the Windows
  data groups reopens as (1) resolved as not fixed (BY DESIGN, NOT
  REPRODUCIBLE, WON'T FIX) reopened when new reproduction steps or new
  information arrives, (2) resolved as fixed, "mostly regression bugs", from
  an incomplete fix or a fix removed, and (3) process reopens; bugs found by
  code-analysis tools were reopened 0.52 to 0.73 times as often as the
  baseline, partly because "code analysis tools do not argue that a bug
  should be fixed, once it has been closed as WON'T FIX". [EVIDENCE] (study;
  PDF read locally)
  [Which bugs get reopened](https://thomas-zimmermann.com/publications/files/zimmermann-icse-2012.pdf).
  That last sentence is the intent's falsifier in the field: a detector that
  cannot see the closure keeps arguing, which is what recurrence-is-signal
  asks of it.
- **The matching key is a declared fingerprint, first rule wins.** Sentry's
  fingerprint rules are `matcher:expression -> values`, "the first matching
  rule in the list is applied", a custom fingerprint "overrides the default
  grouping entirely" unless `{{ default }}` is kept to subdivide it, and the
  page warns that grouping on a changing message "can produce really bad
  groups". [EVIDENCE] (vendor documentation)
  [Fingerprint rules](https://docs.sentry.io/concepts/data-management/event-grouping/fingerprint-rules/).
  The peer note on the consistency pass records SARIF's versioned
  `partialFingerprints` and its `baselineState` vocabulary (`new`,
  `unchanged`, `updated`, `absent`) for the same job
  ([verification note](2026-10-23-verification-assurance-sota.md)).
- **Locally, the mechanism is three-quarters built.** The filing-time match
  already compares a new capture against "every open and resolved issue",
  writes a `duplicates:` link, and never refuses; wontfix records are
  excluded as "work nobody is doing" (`internal/core/capture/match.go`). A hit
  on a resolved record is today labelled a duplicate, which is the sentence
  the intent calls wrong. [EVIDENCE] (the repository)

### Options weighed

1. **A new relation on the existing match**: when the best candidate is in
   `resolved/`, write `recurs:` instead of `duplicates:` and mark the record
   a recurrence; add `wontfix/` to the candidate set with a `recurs_wontfix:`
   (or a `kind:` on the link) so the two facts stay distinct; same threshold,
   same never-refuse contract.
2. **A fuzzier matcher for recurrences** : the intent rules it out, and the
   duplicate-detection evidence says lexical overlap is the strong baseline
   anyway.
3. **Sticky wontfix as GitHub does**: silence on a wontfix match. Contradicts
   the ruling.
4. **Sentry's opt-out**: a `wontfix` may declare itself `silent`
   (archive-forever), so a known accepted cost does not escalate on every
   restatement; the default stays "escalate".

### Recommendation

**Path 2**: option 1 on the shipped match primitive, with Sentry's state
semantics as the named model: resolved plus match is always a recurrence;
wontfix plus match is a recurrence of the second kind, surfaced at capture
time (the person is there) and counted in `abcd audit`; and, if the product
thinker adds it (the intent's open question), option 4's explicit silence as
the only suppression, written by a person with a reason, which is an exception
to the draft's second criterion rather than part of it.
The matching rule stays the ledger's 0.6 term-overlap over the finding's own
words, which is the conservative and explicable rule the intent asks for;
versions of the key follow the SARIF practice the peer note recommends. No
dependency.

### Local fit risks

- **Verifier selects, gates decide.** A recurrence is a link and a label,
  never a status change; `resolved/` is not touched.
- **The record lands with the act.** A new relation is a schema change;
  record-lint must accept the old `duplicates:` links already filed.
- **Evaluator outside the loop.** The capture surface may see the ledger (it
  is the ledger); the cold reading that feeds it may not. The escalation must
  work on a reading record that carries no knowledge of prior closure.

## itd-86: abcd reads its own design documents as a stranger would

Most of this ground is covered by the
[cold-instrument note](2026-08-19-cold-instrument-sota.md), and most of the
intent has since been built: the four positions of itd-184 share one
blindness core, the assembler of itd-185 makes blindness a property of the
input with a manifest, and the amnesia eval compares bundles across runs
(`internal/core/reading`,
[itd-184](../../intents/shipped/itd-184-four-cold-reading-definitions-one.md),
[itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold.md)).
The pass here checked only what changed since August.

### Findings

- **The inheritance list has changed, and there is now a switch.** A
  subagent "doesn't see your conversation history", but its initial context
  holds "every level of the CLAUDE.md hierarchy the main conversation loads",
  a git-status snapshot and preloaded skills; `omitClaudeMd: true` starts it
  "without the user, project, and local CLAUDE.md files", though "managed
  policy files still load", and "a subdirectory's `CLAUDE.md` and matching
  path-scoped rules still load when the subagent works with files there".
  [EVIDENCE] (vendor documentation)
  [Subagents](https://code.claude.com/docs/en/sub-agents). So host-side
  blindness is now one flag, and still leaks through path-scoped rules the
  moment the reader opens a file. The assembler's bundle, read as data and
  not as files, is what closes that leak; the finding confirms the design
  and gives the eval a new case.
- **Familiarity helps a reviewer be useful, which is the contrary evidence.**
  At Microsoft, "the proportion of useful comments made by a reviewer
  increases dramatically in the first year", and usefulness falls as the
  number of files in a change rises. [EVIDENCE] (1.5 million comments, five
  projects)
  [Characteristics of useful code reviews](https://www.microsoft.com/en-us/research/publication/characteristics-of-useful-code-reviews-an-empirical-study-at-microsoft/).
  [CONTESTED] with the cold reading's premise: the study measures usefulness
  of comments on code changes, the cold reading measures detection of
  constraint divergence in a design document, and the method keeps both by
  pairing a blind detector with a warm ledger. The one-document scope the
  intent keeps is also the small-change condition the study favours.
- **The premortem is the human-process cousin.** Klein's premortem has the
  team "assume that the project they are planning has just failed" and
  write the reasons, "making it safe for dissenters ... to speak up".
  [CONSENSUS] (practitioner literature; only the summary was opened)
  [Performing a project premortem](https://hbr.org/2007/09/performing-a-project-premortem).
  Its mechanism is safety, not blindness; the cold reading replaces the
  social cost with structural ignorance.

### Options weighed

1. **Keep the intent as the single-document detection reading**, now one
   instance of the shipped detection position, and close its remaining
   gaps: the settled-enough check, and recurrence through itd-87.
2. **Retire the intent as refined away by itd-184/185.** The typed link
   already says `refines`; what is left is small but real.

### Recommendation

**Path 2, already taken**: the shipped assembler is the native
implementation of enforced blindness, and `omitClaudeMd` is the host-side
seam it can now also set when a reader is dispatched on the host. Close the
intent by delivering the settled-enough line and the reading-record
recurrence (with itd-87) rather than re-planning it. No dependency.

### Local fit risks

- **Recurrence is signal.** Without itd-87 the detector re-raises into a
  ledger that calls it a duplicate; the two ship together or the first is
  noise.
- **Evaluator outside the loop.** A reader dispatched on the host with
  `omitClaudeMd` still has tools; the no-fix rule is enforced by a read-only
  tool list, as the August note said, not by the flag.

## itd-55: the foundations auditor

### Findings

- **Naming the theory in the prompt is a workable zero-shot method.** Gupta,
  Zuckerman and O'Connor (ACL 2024) define argument explication as producing
  claim, reason and warrant triples from Toulmin's model, prompt an LLM with
  the theory's name ("According to Toulmin model") to get the named
  components, and evaluate "coverage and validity" through a human study,
  automatic evaluation on prior datasets and robustness checks "across
  alternative LMs, prompts, and argumentation theories". [EVIDENCE] (ACL
  long paper; numbers are in the PDF, which was not opened)
  [Harnessing Toulmin's theory for zero-shot argument explication](https://aclanthology.org/2024.acl-long.552/).
  The `first-principles-analysis` skill does the same with Aristotle's
  framework; the evidence says the move is sound, and that validity must be
  measured by people, not assumed.
- **The field has moved to prompt-driven argument mining, and warns about
  judging itself.** The 2025 survey covers claim and evidence detection,
  relation prediction, stance, quality assessment and summarisation, flags
  "risks of bias and evaluation circularity" in LLM-assisted annotation, and
  consolidates "LLM-judge reliability" as one of the evaluation practices to
  assess, beside component-level accuracy and soft-label quality (its open
  problems are long-context reasoning, multimodal and multilingual robustness,
  interpretability and cost). [EVIDENCE/CONSENSUS]
  (survey, revised November 2025)
  [Large Language Models in Argument Mining: A Survey](https://arxiv.org/abs/2506.16383).
  For abcd: the auditor's own output needs a small human-labelled set before
  it is trusted, the
  [judge-calibration note](2026-07-12-judge-calibration-sota.md)'s "ground
  truth has to be manufactured".
- **The forcing-function synthesis has a human precedent.** The premortem
  asks each person for the reasons the plan failed and makes dissent safe
  ([Klein, HBR 2007](https://hbr.org/2007/09/performing-a-project-premortem));
  the auditor's "single most load-bearing unexamined assumption" is the
  same move made by a reader with no stake. [CONSENSUS] (practitioner
  literature)
- **Locally, the output contract exists in two neighbours.** The intent
  auditor emits a verdict JSON with "every claim carrying a cited file:line
  evidence pointer", and the cold reading emits three-element detections in
  document order; both are host-delegated agent definitions validated by the
  binary on ingest. [EVIDENCE] (the repository;
  [skills evaluation](socratic-and-first-principles-skills-evaluation.md))

### Options weighed

1. **Structured report as the canonical shape**: claim architecture, a
   load-ranked premise inventory, a per-premise terminus class
   (genuine-first-principle, domain-conventional, unjustified-stop,
   circular-dependency), a four-causes row, a domain-appropriateness row and
   one named synthesis, each item quoting the span it rests on; the table
   and the commentary are renderings of it.
2. **Flowing commentary** : readable, not checkable, and not countable
   across runs.
3. **An interactive pass** : duplicates the grill; the source method is
   single-pass.

### Recommendation

**Path 2**: a native, host-delegated agent definition shaped to Toulmin
explication for the structure (claim, reason, warrant made explicit) with
the Aristotelian terminus classes layered on each warrant, option 1's
structured report as the one canonical output, single-pass, advisory, and a
standalone surface that takes a document path so it reads ADRs and the brief
as well as intents. The seam is the output schema: a later method change
replaces the agent definition and leaves the ingest and its validation. A
first run over ten records with the facilitator labelling each circularity
flag true or false is the calibration the survey asks for. The
`ACKNOWLEDGEMENTS.md` entry lands with the build. No dependency.

### Local fit risks

- **Widen options, never recommend.** Naming the single deepest flaw is a
  ranking of findings, not a remedy; the auditor must still propose no fix.
- **Evaluator outside the loop.** The auditor reads the document and nothing
  else; it must not read the grill's or the fidelity reviewer's output on
  the same record.
- **Host-delegated.** No provider call; the definition is dispatched by the
  host as the other reviewers are.

## Not worth adopting

- **A vector index, embedding model or memory service for recall or dredge**
  (Mem0, Zep/Graphiti, a local embedding model): a dependency or a network
  call in a per-prompt hook, and the lexical baseline is competitive at this
  corpus size on the published evidence.
- **Graph memory with temporal edges** at hundreds of notes: the gain Mem0
  reports for its graph variant is two points of J at twice the retrieved
  tokens; the review the dashboard gives to a person does the invalidation
  job.
- **Numeric source weights with a configuration override** (itd-12): no
  tool surveyed does it, and the model will not honour the number.
- **cron on macOS** (itd-13): deprecated by the vendor, and a shared file
  abcd should not edit programmatically.
- **Sticky dismissal of a wontfix** (itd-87): GitHub's behaviour, and the
  one the record's principle exists to refuse; keep only the explicit
  per-record silence.
- **A model-only clustering pass for dredge**: no stable cluster identity,
  so growth cannot be measured between runs.
- **A second scorer for near-duplicate notes**: the ledger's match primitive
  already exists.

## Not verified

- Cursor's Memories feature, its approval pop-up and its reported removal in
  version 2.1: search snippets only; the documentation page redirected and
  was not opened.
- GitHub's own similar-issue or duplicate-issue suggestions on issue
  creation: the search budget ran out before this was checked.
- The credibility-aware and source-reliability RAG literature, and
  knowledge-conflict surveys beyond ClashEval: not reached for the same
  reason.
- Zep's `valid_at` / `invalid_at` edge model: described in a search snippet
  and a third-party summary; the arXiv abstract page did not state it. Mem0's
  graph variant's "mark invalid rather than delete" was read in the paper
  and is cited instead.
- WarrantScore's figures on warrant acceptability (GPT-4 61.7% against
  human-written 45.5%): seen in a search snippet, not opened.
- Klein's prospective-hindsight percentage: the HBR page served only the
  summary.
- Clio's exact minimum cluster sizes and human-rater agreement: in the
  appendices beyond the portion read.
- LongMemEval's numeric gains from its indexing and retrieval optimisations:
  the abstract gives none.
- The TOSEM benchmark's per-project tables: the PDF was read locally for the
  headline findings only.
- Microsoft's hve-core policy of re-filing dismissed-but-still-detected
  alerts weekly: search snippet only.
- Whether user-owned GitHub repositories expose any maintainer role beyond
  owner and collaborator: the roles page covers organisations only.
