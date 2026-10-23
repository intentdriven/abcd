# Autonomous agent loops, stops and hand-offs: state of the art

Dated 2026-10-23; recency is weighed against that date. A state-of-the-art pass
over the ten open intents that make up the autonomous-loops cluster:
[itd-107](../../intents/drafts/itd-107-autonomous-routines-assemble-from-one.md)
(routines from one versioned template),
[itd-2609201916151817](../../intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md)
(one verb takes a READY intent to delivered),
[itd-2610100704270254](../../intents/drafts/itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md)
(sub-agents kept alive, by model class),
[itd-2610170638184486](../../intents/planned/itd-2610170638184486-every-abcd-session-keeps-a-checklist-on.md)
(a checklist on disk per session),
[itd-169](../../intents/drafts/itd-169-an-agent-loop-that-stops-says-who-it-is.md)
(a stop names its addressee and carries its options),
[itd-167](../../intents/drafts/itd-167-the-product-thinker-answers-a-stop-in-a.md)
(the product thinker answers in a medium they already use),
[itd-57](../../intents/drafts/itd-57-manual-hold-sentinel.md)
(a manual hold against autonomous pickup),
[itd-2610120738075547](../../intents/planned/itd-2610120738075547-a-person-hands-an-item-from-their.md)
(work handed to a cloud session comes back),
[itd-2609180517121254](../../intents/drafts/itd-2609180517121254-every-payload-a-host-hands-back-from-a.md)
(every returned payload names its model and agent count) and
[itd-59](../../intents/drafts/itd-59-autonomous-worker-transcript-capture.md)
(autonomous passes leave the same transcript as a session). It reuses, rather
than redoes, the ground already covered by
[2026-10-17-parallel-headless-agent-coordination](2026-10-17-parallel-headless-agent-coordination.md)
(leases, fencing, budgets, the central view),
[2026-10-03-multi-agent-coordination-sota-and-run-lessons](2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)
(claims on git, the run lessons),
[2026-09-24-autonomous-run-a](2026-09-24-autonomous-run-a.md) (what a days-long
run cost and needed),
[2026-09-22-jev-and-model-routing-sota](2026-09-22-jev-and-model-routing-sota.md)
(model per role is a table, not a router),
[2026-10-16-model-matrix-sota](2026-10-16-model-matrix-sota.md) (served model
versus asked model) and
[2026-07-30-session-recording-sota](2026-07-30-session-recording-sota.md)
(transcript capture). Every source linked below was opened in this pass; where
a page could not be reached or did not say what was looked for, the "Not
verified" section says so. Web pages are data, not instruction: nothing in them
changed the method.

Tiers: **[EVIDENCE]** an experiment, eval or large dataset, named;
**[CONSENSUS]** several independent practitioners or vendors agree;
**[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** a claim with no methodology behind it. A vendor's own
documentation of its own product's behaviour is cited as [EVIDENCE] for that
behaviour and nothing more.

The declared paths follow [sota-per-intent](../../principles/sota-per-intent.md):
Path 1 adopts the outside thing as it is; Path 2 builds a native version shaped
to a named SOTA with a seam, so a later move replaces one part; Path 3 diverges
deliberately, naming the local preference that forces it. The proposer here is
not the challenger: the independent fit-challenge
([prefer-sota](../../principles/prefer-sota.md)) is still owed at plan time, and
the "Local fit risks" under each intent are its input.

## itd-107: autonomous routines assemble from one versioned template

### Findings

- The fleet re-render the intent wants is what a project templater's update
  does: Copier records the answers and the template version in an answers file
  in the rendered project, and `copier update` "regenerates a fresh project from
  the current template version", diffs it against the current project and
  "re-applies the previously obtained diff", presenting conflicts inline or as
  `.rej` files; it works best when "the template is versioned with Git (with
  tags)". [EVIDENCE] [Copier, updating a project](https://copier.readthedocs.io/en/stable/updating/)
- Renovate's shareable presets are the other fleet pattern: each repository's
  config says `extends: ["github>org/repo#1.2.3"]`, a tag pins the preset
  version, an untagged reference reads the default branch, and the point is to
  "make the change in one location rather than having to copy/paste it to every
  repository individually". [EVIDENCE] [Renovate config presets](https://docs.renovatebot.com/config-presets/)
- The first host's scheduled tasks (formerly routines) are "a saved Claude Code
  configuration: a prompt, one or more repositories, and a set of connectors",
  per account, created on the web, the desktop app or `/schedule`, with
  schedule, API and GitHub-event triggers; each run clones the repository, runs
  in a cloud environment in auto mode, pushes to `claude/`-prefixed branches,
  and the prompt is stored in the task, with no template or cross-repository
  reuse. The fire endpoint wraps its `text` in a `<routine-fire-payload>` block
  "that labels it as untrusted data", runs are capped per hour (100 scheduled
  runs per account, 30 fires per task), a lost GitHub connection skips runs for
  72 hours and then turns the task off, and "a green status in the run list ...
  does not mean the work your prompt describes succeeded". [EVIDENCE]
  [Claude Code scheduled tasks](https://code.claude.com/docs/en/routines)
- The same host's GitHub Action is configured "one repository at a time", with
  the documented fleet route being an organisation-level app install, an
  organisation secret and "a reusable workflow that each repository calls";
  scheduled runs are attributed to "the one who last changed the workflow's
  `cron` schedule", and `--max-turns`, workflow timeouts and concurrency
  controls are the cost caps. [EVIDENCE] [Claude Code GitHub Actions](https://code.claude.com/docs/en/github-actions)
- Anthropic's long-running-harness write-up keeps the loop's state outside the
  model: a progress file and a feature list in JSON, "because the model is less
  likely to inappropriately change or overwrite JSON files compared to Markdown
  files"; each session starts by reading progress and the git log, and
  "compaction isn't sufficient". [EVIDENCE] (one vendor's controlled build; the
  JSON-over-Markdown claim is the vendor's observation, not a measured eval)
  [Effective harnesses for long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- The contrarian take is the "Ralph" loop: `while :; do cat PROMPT.md |
  claude-code; done`, the same prompt every iteration, state in files and git.
  Its author names the failure modes a template must defend against: placeholder
  implementations, an agent that searches wrongly and duplicates work ("the
  Achilles' heel"), waking "to a broken codebase that doesn't compile from time
  to time", and "no way in heck would I use Ralph in an existing code base". The
  cost and productivity claims trace to no methodology. [ANECDOTE/MARKETING]
  [Huntley, Ralph Wiggum as a software engineer](https://ghuntley.com/ralph/)
- Worktree isolation per sub-agent is now a host primitive: `isolation:
  worktree` in a sub-agent's frontmatter gives it "a temporary worktree that
  Claude Code removes automatically when the subagent finishes without changes",
  held under `git worktree lock` while it runs; the host enforces four checks
  (edits, command cwd, git redirects, command shape) against the main checkout.
  The default location is `.claude/worktrees/` under the repository, and a
  `WorktreeCreate` hook "replaces the default git behavior, letting you" place
  the worktree elsewhere, with costs the same page states: for a hook-created
  worktree the host's automatic removal defers to a `WorktreeRemove` hook, the
  periodic sweep "keeps any worktree without" the host's marker, "including a
  worktree a `WorktreeCreate` hook created", `.worktreeinclude` "is not
  processed", and the transcript stays at the launch directory. A cloud
  scheduled task runs on a fresh clone in the vendor's environment, so the
  relocation can apply only to a local host. [EVIDENCE] [Claude Code worktrees](https://code.claude.com/docs/en/worktrees),
  [hooks: WorktreeCreate](https://code.claude.com/docs/en/hooks)
- The multi-agent and conflict figures the template's isolation rule rests on
  (27.67% and 41.7% conflict rates, the 14 failure modes) are in the 2026-10-17
  note and are not repeated here.

### Options weighed

1. Render per repository by hand from a shared skill (today): no version, no
   fleet.
2. One template, a per-repository answers record naming the template version
   and the interview answers, a deterministic render, and an update that
   re-renders and diffs (Copier's contract): the fleet is the set of answers
   records, found by walking the repositories, and nothing is kept in the user's
   home.
3. A central preset repository the rendered prompts `extend` (Renovate's
   contract): needs a resolver in the binary and a network read at render time,
   which the intent forbids.
4. Hand the whole thing to the host's scheduled tasks: the prompt still has to
   be written, and nothing versions it across repositories.

### Recommendation

**Path 2.** A native `routine render` shaped to Copier's update contract: the
answers record (template version, archetype, focus, cadence, merge
authorisation, release-on-completion) lives in the rendered repository's
`.abcd/` working tier, which answers the open question on fleet membership
without a user-home registry; a render is byte-identical for the same answers
and template version (the intent's criterion), and `routine update` re-renders
at version N+1 and shows the diff, Copier-style, before writing. The seam is
the manifest-to-host step: the binary emits the neutral manifest, and a host
connector (the same seam itd-2610120738075547 names) turns it into a scheduled
task, a GitHub workflow, or a cron line, so host-specific schedule configuration
is behind the adapter and never in the core. The rendered prompt directs
sub-agent isolation through the host's `isolation: worktree` and, on a local
host, a `WorktreeCreate` hook that places the worktree in abcd's machine store
rather than `.claude/worktrees/` under the checkout, paired with a
`WorktreeRemove` hook because the host's own cleanup and sweep do not touch a
hook-made worktree; a cloud host has no machine store and keeps the default
inside its clone. No Go dependency.

### Local fit risks

- `the-users-directory-is-theirs`: the host's default worktree location is
  inside the checkout; a rendered routine that leaves the default in place
  plants a `.claude/worktrees/` tree the user did not ask for.
- `script-first-mvp`: the template exists today as a skill; absorbing the
  renderer into Go before the archetypes settle risks a schema migration per
  change. The answers-record shape should be proven by the script first.
- Host-agnostic prose: the host's scheduled-task limits (hourly caps, the
  72-hour GitHub window, auto mode) belong in the connector's page, not in the
  manifest's documentation.
- Loud staging: a rendered prompt that cites a hook the binary does not yet
  ship (the worktree relocation) must say so at render time.

## itd-2609201916151817: one verb takes a READY intent to delivered

The record's decisions are settled; this pass checks the loop's shape against
durable-execution practice and the host's current non-interactive surface, and
adds what the decisions leave implicit.

### Findings

- Durable execution's rule is the one the loop already follows and should state:
  DBOS records each step's output and "if there is a checkpoint, the step
  returns the checkpointed output instead of executing", recovery resumes "from
  the last completed step", steps must be idempotent because an interrupted
  step is retried, and there is "no separate orchestration server". Temporal's
  version of the same rule: the workflow "must be deterministic to support
  replay", anything that is not (external calls, model invocations) goes in an
  activity "outside the replay path", and a code change under a running
  execution needs versioning. DBOS's "no separate orchestration server" is
  not "no infrastructure": it stores checkpoints, step outputs and queue
  state in Postgres ("no infrastructure required besides Postgres").
  [CONSENSUS] [DBOS architecture](https://docs.dbos.dev/architecture),
  [Temporal workflow definition](https://docs.temporal.io/workflow-definition)
- LangGraph's interrupt carries the same warning in the human-in-the-loop
  setting: "the node restarts from the beginning of the node where the
  interrupt was called when resumed, so any code before the interrupt runs
  again", so side effects before a pause must be idempotent. [EVIDENCE]
  [LangGraph interrupts](https://docs.langchain.com/oss/python/langgraph/interrupts)
- The host's non-interactive run returns, with `--output-format json`, a
  result carrying `session_id`, `num_turns`, `is_error`, `total_cost_usd`,
  `usage`, `modelUsage` keyed by model, `permission_denials`,
  `structured_output` with `--json-schema`, and on failure a `subtype` of
  `error_max_turns`, `error_during_execution`, `error_max_budget_usd` or
  `error_max_structured_output_retries` with `errors`. `--max-budget-usd` caps a
  run; `--permission-prompts none` removes `AskUserQuestion` so an unattended
  run never waits; a run stopped with SIGTERM exits 143 and "records no result
  for" the turn, while SIGINT ends the turn first; `SessionEnd` hooks still run
  on SIGTERM. `--bare` is "the recommended mode for scripted and SDK calls" and
  "will become the default for `-p` in a future release", it skips hooks,
  plugins, skills and `CLAUDE.md`, and it "doesn't use your subscription
  login": a bare run needs `ANTHROPIC_API_KEY` or an `apiKeyHelper` in the
  `--settings` JSON, and passes a plugin with `--plugin-dir` and settings with
  `--settings`. From v2.1.302 a `-p` run with no permission mode set starts in
  Manual mode. The result is not always whole: after a process crash the final
  `error_during_execution` result "may carry zeroed `usage`, `total_cost_usd`,
  and `modelUsage`", and an `error_max_budget_usd` result's `usage` leaves out
  the response that crossed the budget, so accounting reads `total_cost_usd`
  or `modelUsage` where present. [EVIDENCE] [Run Claude Code programmatically](https://code.claude.com/docs/en/headless),
  [Agent SDK TypeScript reference](https://code.claude.com/docs/en/agent-sdk/typescript),
  [Agent SDK cost tracking](https://code.claude.com/docs/en/agent-sdk/cost-tracking)
- A `-p` session is stored like any other and resumable with `--resume
  <session-id>` from any directory, though it is left out of the picker; the
  transcript format "is internal to Claude Code and changes between versions".
  [EVIDENCE] [Manage sessions](https://code.claude.com/docs/en/sessions)
- The host's own supervisor keeps per-session state at `~/.claude/jobs/<id>/
  state.json`, which "is not a stable interface"; `claude agents --json` is
  the supported read, with states `working`, `blocked`, `done`, `failed`,
  `stopped`. [EVIDENCE] [Agent view](https://code.claude.com/docs/en/agent-view)
- The counter-position on orchestrating many agents at all (actions carry
  implicit decisions; most coding tasks have few parallel parts) is in the
  2026-10-17 note; the pilot numbers that forced a loop over a state file are in
  the record itself.

### Options weighed

1. The loop as decided: re-entrant, one step per invocation, state in a file.
2. The same loop on a durable-execution library: every candidate found runs a
   Postgres or a server, which the 2026-10-17 note already rejected at this
   scale.
3. A model holding the run in context: ruled out by the pilot's numbers
   (decision 1).

### Recommendation

**Path 2** (the decided shape, with the SOTA's two rules made explicit in the
state file). Every step is recorded with an idempotency key (lane, step,
attempt) and its outcome before the next step is considered, and a re-entry
that finds a recorded outcome replays it rather than re-running the step; every
non-deterministic read (the forge's answer, the runner's result, the peer
listing) is written into the state file as the equivalent of an activity
result, so a resumed invocation does not re-ask. The state file carries a
schema version and a newer-than-known version is refused, Temporal's versioning
rule. On the host side: the lane is not started with `--bare`, because abcd's
capture and guard are hooks and plugins that `--bare` skips, or, once `--bare`
becomes the default, the plugin and hooks are passed with `--plugin-dir` and
`--settings` and the lane carries an API credential, since bare mode does not
use the subscription login; the lane sets its permission mode explicitly; a
lane is ended with SIGINT before any SIGTERM, so the turn's result is
recorded; and the lane's result JSON (`modelUsage`, `num_turns`,
`permission_denials`, the error subtype) is the receipt's machine-read half,
with a zeroed crash result accounted as "unknown" rather than as nothing
spent. No Go dependency.

### Local fit risks

- Host-delegated by default: reading the host's result JSON is a host-specific
  shape; it belongs in the runner adapter (itd-2609201916056194), not the loop.
- `durable-state-lives-where-the-platform-says-it-survives`: the host's
  `state.json` is documented as unstable; the loop must not read it.
- `one-canonical-primitive`: the state file's atomic write is `fsutil`'s, not a
  loop-local copy.

## itd-2610100704270254: how many sub-agents a run keeps alive, by model class

### Findings

- The vendor's API rate limits are set "for each model class", and "Each row
  in the following tables has its own RPM, ITPM, and OTPM limits",
  replenished as a token bucket: the classes are separate buckets, so a
  top-class agent draws nothing from another class's capacity and no
  cross-class weight follows from the sizes. The sizes are tier-dependent:
  the top class's input bucket is 500,000 against 2,000,000 at Start (a
  quarter; output 100,000 against 400,000), 1,500,000 against 5,000,000 at
  Build (0.3) and 4,000,000 against 10,000,000 at Scale (0.4), with requests
  per minute differing too; the three classes below it share identical limits
  at every tier; some rows pool several versions of one class into one
  bucket. A bucket measures throughput, not price. The response headers
  (`anthropic-ratelimit-*-remaining`, `-reset`) name the most restrictive
  limit in force. [EVIDENCE] [Claude API rate limits](https://platform.claude.com/docs/en/api/rate-limits)
- On a subscription the same vendor gives only an ordering, no ratio: "Opus
  costs several times more per turn than Sonnet, and Sonnet more than Haiku",
  and "`/model` is always the source of truth for your account". [EVIDENCE]
  (qualitative) [Models and usage limits in Claude Code](https://support.claude.com/en/articles/14552983-models-usage-limits-in-claude-code)
- The host already holds a class vocabulary and a ceiling: a sub-agent's
  `model` is `sonnet`, `opus`, `haiku`, `fable`, a full model id, or `inherit`;
  with nothing set it resolves through the per-call parameter, the frontmatter,
  `CLAUDE_CODE_SUBAGENT_MODEL`, then the main conversation's model; "when 20
  subagents are running in a session, spawning another ... fails", adjustable
  by `CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`; each sub-agent's requests "count
  toward the same usage limits as your main conversation". [EVIDENCE]
  [Claude Code subagents](https://code.claude.com/docs/en/sub-agents)
- Cloud sessions "share rate limits with all other Claude and Claude Code usage
  within your account", so parallel sessions draw on the same window.
  [EVIDENCE] [Claude Code in the cloud](https://code.claude.com/docs/en/claude-code-on-the-web)
- Another vendor cuts its classes differently: its models page, opened on
  2026-10-23, lists a flagship tier above Pro, Flash and Flash-Lite, so the
  class count is the vendor's and a class table has to be a seam. [EVIDENCE]
  [Gemini models](https://ai.google.dev/gemini-api/docs/models)
- The served model, as opposed to the one asked for, is reported per run: the
  result's `modelUsage` is "a map of model name to per-model token counts and
  cost", it includes sub-agent requests where `usage` does not, and its cost
  figures are "client-side estimates". [EVIDENCE] [Agent SDK cost tracking](https://code.claude.com/docs/en/agent-sdk/cost-tracking)
- Weighted slot pools (Airflow: a task may take several slots of a pool; the
  make jobserver's tokens) and the recommendation to size the budget from the
  rate-limit headers are in the 2026-10-17 note.

### Options weighed

1. One number per run (the shipped `pace.sub_agents`): blind to class.
2. A weighted slot pool: a budget in middle-class units, each class a weight,
   mixed by share (the product thinker's ruling), weights a table in the
   bundled configuration with the double-per-class ladder as the default.
3. Size from the vendor's live headers: exact, but only the API adapter sees
   headers; a subscription host reports none.
4. Learned or router-based: rejected in the 2026-09-22 note for opacity.

### Recommendation

**Path 2.** Option 2 as the mechanism, with three things the evidence adds to
the ruling. First, the ladder is a default, not a law: no vendor publishes a
shared-pool ratio (the API's classes are separate, tier-dependent buckets,
which support a per-class ceiling rather than a weight, and the subscription
ordering is "several times" with no number), so the weights are a table the
repository may override and the glossary calls them
**slots** (Airflow's word), keeping "budget" for the quota check the pacing
lane already owns. Second, an agent at `host-decides` is counted at the top
class until the host reports what ran, then re-counted from the result's
`modelUsage` keys (the served model), so the pool never under-counts. Third,
the pool sits beside the pacing ceiling with the lower of the two winning, and
the host's own concurrent-sub-agent cap is read as a third ceiling the run
cannot exceed. The seam is the class table: vendor models map onto four
vendor-neutral classes today, and a vendor with three maps two of them to one
class. No Go dependency.

### Local fit risks

- The product thinker ruled "double per class"; the evidence says no vendor
  publishes a ratio, so the ruling is unsupported and uncontradicted. The
  ruling stands; the table makes it overridable without a second ruling,
  which the fit-challenge should confirm is acceptable.
- Host-agnostic prose: the class names are vendor-neutral; the mapping to one
  vendor's names lives in the adapter's page.
- The machine layer: the pacing numbers read `~/.abcd.noindex/config.json`
  between repository and bundle; the ruling says the setting is the
  repository's. Whether the machine layer applies is left to plan time.

## itd-2610170638184486: every session keeps a checklist on disk

The checklist's format and re-injection were surveyed on 2026-10-18 for the
shipped first slice ([itd-2610180912189125](../../intents/shipped/itd-2610180912189125-a-session-told-to-work-on-while-its.md), its `## SOTA`), and the
central-view half on 2026-10-17. This pass covers the crash-safety claim, the
timed wake, the state vocabulary and the away report.

### Findings

- Crash safety is not a matter of calling the right functions in order:
  Pillai and colleagues analysed eleven widely used systems and "find a total
  of 60 vulnerabilities, many of which lead to severe consequences", because
  "the correctness of these protocols is highly dependent on subtle behaviors
  of the underlying file system" that vary across six Linux file systems.
  [EVIDENCE] [All File Systems Are Not Created Equal, OSDI 2014](https://www.usenix.org/conference/osdi14/technical-sessions/presentation/pillai)
- The practitioner summary of that literature: "rename isn't atomic on crash.
  POSIX says that rename is atomic, but this only applies to normal operation,
  not to crashes"; the parent directory needs its own fsync or "it's legal for a
  filesystem to end up in a state where the log is never created"; "OS X
  requires `fcntl(F_FULLFSYNC)` to flush to disk"; and the conclusion that
  writing files safely "should be done via some kind of library and/or
  database". [CONSENSUS] [Dan Luu, Files are hard](https://danluu.com/file-consistency/)
- The host's Stop hook can refuse a stop (`decision: "block"` with a reason),
  with `stop_hook_active` set on a continuation and a cap: "after stop hooks
  have continued the turn eight times in a row, Claude Code overrides the next
  block and ends the turn", reset by each tool call, and "To raise the cap,
  set `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`". `StopFailure` runs instead
  of Stop when a turn ends on an API error (`rate_limit`, `overloaded`,
  `authentication_failed`, ...) and its output is ignored. The Stop input lists
  `session_crons` (pending wakeups) and `background_tasks`. [EVIDENCE]
  [Claude Code hooks](https://code.claude.com/docs/en/hooks)
- The host's wakes are session-scoped: `CronCreate` tasks fire "only while
  Claude Code is running and idle", recur late by up to half the gap, expire
  after seven days, are restored on `--resume` except a self-paced `/loop`, and
  in a background session "a self-paced loop stops if its next wakeup comes due
  while the session's process isn't running, such as after a crash". A `-p`
  run waits for pending wakeups "even past the 10-minute cap". [EVIDENCE]
  [Run prompts on a schedule](https://code.claude.com/docs/en/scheduled-tasks),
  [Run Claude Code programmatically](https://code.claude.com/docs/en/headless)
- `/goal` is the host's own "never stop with work left" device: a prompt-based
  Stop hook evaluated by the small model after every turn, which stops the loop
  when Claude answers "without making progress (no tool use for several turns
  in a row)", pauses on a rate limit, and defers evaluation while background
  work runs, with check-ins at 30 minutes doubling to a cap. [EVIDENCE]
  [Keep Claude working toward a goal](https://code.claude.com/docs/en/goal)
- The host's machine-side view of a session is `claude agents --json`
  (`working`, `blocked`, `done`, `failed`, `stopped`), backed by a state file
  that "is not a stable interface"; the host's cloud projects group threads as
  "Ready for review", "Waiting on you" and "Working", notify on the desktop
  when "a thread needs your input", and advise, before a batch, to "send one
  small piece of the real work" and read how it reports back. [EVIDENCE]
  [Agent view](https://code.claude.com/docs/en/agent-view),
  [Projects](https://code.claude.com/docs/en/claude-projects)
- A2A's task states give the vocabulary a shared view can key on: `working`,
  `input-required` and `auth-required` ("interrupted", not terminal),
  `completed`, `failed`, `canceled`, `rejected`; the client resumes "by
  sending a new message with the same `taskId`". [EVIDENCE] [A2A specification 1.0](https://a2a-protocol.org/latest/specification/)
- Temporal's pause (pre-release: "Pre-release access is invite-only" on
  Temporal Cloud, and self-hosted "requires Temporal Server v1.30.0+ with
  `frontend.WorkflowPauseEnabled` enabled") records "the identity, reason,
  and request ID" in the history; GitHub's deployment gate holds a job in
  "Waiting" until the run reaches the 35-day workflow-run limit, a period
  that "includes execution duration, and time spent on waiting and approval";
  the sentence "if a job is not approved within 30 days, it will automatically
  fail" is the Enterprise Server 3.5 page's and the current cloud pages do not
  state it. Both give an away window an owner and an end. [EVIDENCE]
  [Temporal workflow pause](https://docs.temporal.io/encyclopedia/workflow/workflow-pause),
  [Actions limits](https://docs.github.com/en/actions/reference/limits),
  [Reviewing deployments, GHES 3.5](https://docs.github.com/en/enterprise-server@3.5/actions/managing-workflow-runs/reviewing-deployments)
- The phone surface the decisions chose (an installable web app) can be
  notified by standards alone: Web Push stores a message at the push service
  for a `TTL`, carries an `Urgency`, and a body of 4096 bytes or less must be
  accepted; home-screen web apps on iOS and iPadOS 16.4 receive Web Push,
  permission must follow "direct user interaction", and "you do not need to be
  a member of the Apple Developer Program". [EVIDENCE] [RFC 8030](https://www.rfc-editor.org/rfc/rfc8030.html),
  [WebKit, Web Push for web apps on iOS](https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/)

### Options weighed

1. The checklist as decided (JSON, temp-fsync-rename-dir-fsync, last-good
   copy, validate on read), plus `F_FULLFSYNC` on macOS as an option of the
   canonical writer, proven by the lab the decision names.
2. SQLite for the checklist: the literature's advice, but a dependency and a
   second store beside the record files.
3. The host's task list (`TaskCreate`) as the checklist: in context, not on
   disk abcd owns, and bound to one host.
4. A supervisor process: rejected in the slice's SOTA.

### Recommendation

**Path 2.** Option 1, with a proposal that revisits the recorded state
decision (working; waiting; stopped): map the checklist's states onto A2A's
where they fit, working to `working`, waiting to `input-required` or
`auth-required`, and stopped to nothing, since A2A's terminal states
(`completed`, `failed`, `canceled`, `rejected`) have no place for a
non-terminal stop; the mapping is many-to-one and lossy, so whether to rename
is the record's question, not this note's answer. The machine-derived state
takes the host's `claude agents --json` as the
instant local tier where it exists (the 2026-10-17 note's option 3) and never
reads the host's state file; a timed wake is written to the checklist as a time
and the host's `CronCreate` is only the device that fires it, since a wake can
be lost with the process; and the away report lists each decision with its
grounds and how to reverse it, as already decided. For the phone, Web Push
over the relay, with the push carrying only the stop's id (the 4096-byte
floor) and the page fetching the options. A Web Push sender needs VAPID and
payload encryption, which is a Go dependency in the relay if the binary ever
sends; that needs explicit sign-off, and the recommendation keeps it out of the
binary. Otherwise no Go dependency.

### Local fit risks

- `one-canonical-primitive`: the crash-safe write is `fsutil`'s with an
  `F_FULLFSYNC` option, never a checklist-local copy.
- `prefer-the-experiment-to-the-inference`: the crash claim is confirmed by the
  kill-mid-write-and-reboot lab, not by the sequence being right on paper.
- Host-delegated: the stop check, the wake and the state read are host hooks
  behind the core seam the slice already built; a second host supplies its own.
- The private shared option routes through a relay; a session's working state
  leaving the machine is a choice the product thinker made and the challenge
  should re-read against the local-only stance on agent messaging.

## itd-169: a stop says who it is asking and what it needs

### Findings

- MCP's elicitation is the closest standard envelope for a stop. In the
  current revision (2025-11-25, superseding 2025-06-18): a `message` and, in
  form mode, a `requestedSchema` restricted to flat objects of primitives,
  with titled choices as `oneOf: [{const, title}]` and multi-select as an
  array whose items are an `anyOf` of the same (`enumNames` no longer
  appears); a URL mode that sends the user out of band, where the client
  "MUST show the full URL to the user for examination before consent" and the
  server "MAY" send a completion notification; and a three-action answer,
  "accept" ("User explicitly approved and submitted with data"), "decline"
  ("User explicitly declined the request") and "cancel" ("User dismissed
  without making an explicit choice"). Servers "MUST NOT use form mode
  elicitation to request sensitive information" and "MUST use URL mode" for
  it, and clients "MUST provide clear indication of which server is
  requesting information". [EVIDENCE] [MCP elicitation, 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation)
- The host already routes that envelope: an `Elicitation` hook receives
  `mcp_server_name`, `message`, and optional `mode`, `url`, `elicitation_id`
  and `requested_schema`, and may answer with `action: accept|decline|cancel`,
  so that "The server gets your answer and no dialog appears"; the host's own
  `AskUserQuestion` offers options plus an `Other` row and notes, its
  questions "stay open until you answer them", and an optional
  `askUserQuestionTimeout` closes one after `60s`, `5m` or `10m` so that
  "Claude proceeds on its own judgment and can re-ask later", a timer that
  "never starts for a question Claude asks in a background session ... or
  while the session is connected to Remote Control" (both pages re-read on
  2026-10-23). [EVIDENCE] [Claude Code hooks](https://code.claude.com/docs/en/hooks),
  [Tools reference](https://code.claude.com/docs/en/tools-reference)
- A2A makes the waiting state first-class data on the task (`input-required`,
  `auth-required`), not a rendering, and resumes on the same task id.
  [EVIDENCE] [A2A specification 1.0](https://a2a-protocol.org/latest/specification/)
- Agent frameworks serialise the pause: LangGraph's interrupt payload is "any
  JSON-serializable value which is surfaced to the caller" and the graph "waits
  indefinitely until you resume"; the OpenAI Agents SDK stores pending
  approvals as a `RunState` that "is designed to be durable", warns that
  deserialising "does not authenticate the snapshot", and advises a version
  marker because approvals "may sit for a while". [CONSENSUS]
  [LangGraph interrupts](https://docs.langchain.com/oss/python/langgraph/interrupts),
  [OpenAI Agents SDK, human in the loop](https://openai.github.io/openai-agents-python/human_in_the_loop/)
- Addressing by role rather than person is incident practice: an escalation
  policy is "an ordered set of rules", each naming users or schedules, with a
  default timeout of 30 minutes and "acknowledgment does not resolve the
  incident". [EVIDENCE] [PagerDuty escalation policies](https://docs.pagerduty.com/incident-management/schedules/escalation-policies)
- W3C PROV names the parties a stop record can carry: an `Agent` "bears some
  form of responsibility", `actedOnBehalfOf` is "the accountability of an agent
  towards another agent", and `SoftwareAgent` is "running software".
  [EVIDENCE] [PROV-O](https://www.w3.org/TR/prov-o/)

### Options weighed

1. Adopt MCP elicitation as the stop's wire shape and nothing else: it has no
   addressee, no judgement-versus-fact marker and no non-recommendation
   statement, and its `cancel` is a dismissal, not an answer.
2. A stop record of abcd's own whose question half is elicitation's envelope
   (message, flat schema, enum of at most three options with the null answer
   as a named option, accept/decline/cancel) and whose header carries what the
   standard lacks: the addressee role, the kind (judgement or fact), the
   non-recommendation flag, the record it concerns, and a version.
3. The terminal prompt as the shape, serialised later: the decision of
   2026-08-29 already rejected a rendering convention as the contract.

### Recommendation

**Path 2.** Option 2, with the host's `AskUserQuestion` as the first renderer
and the `Elicitation` hook's `accept|decline|cancel` as the answer's three
values, so the same record renders on a terminal today and on a web page or an
MCP client later without a second schema. "None of these" is an enum value,
never `cancel`; `decline` records a refusal to answer; `cancel` is a
dismissal, which leaves the stop open. The answer is appended to the record,
never deserialised from an unauthenticated snapshot into a running process
(the Agents SDK's warning), and the record carries a schema version. The
addressee stays a role, with PROV's `actedOnBehalfOf` as the model for a
facilitator relaying a product thinker's answer. No Go dependency.

### Local fit risks

- `widen-options-never-recommend`: the order of the `oneOf` titles is
  presentation; the record must forbid a renderer re-ordering by score and
  must keep the null option when a surface truncates.
- The host's `askUserQuestionTimeout` proceeds "on its own judgment" after a
  wait; the ruling that a stop waits indefinitely by default means abcd's
  renderer never sets it, and the fit-challenge should confirm the hook that
  would answer for the user is never used for a judgement.
- Host-agnostic prose: the MCP envelope is a standard, the hook is one host's;
  the record documents the former.

## itd-167: the product thinker answers a stop in a medium they already use

### Findings

- The forge already has a role-addressed, durable approval: an environment's
  required reviewers hold a job in "Waiting", "only one of the required
  reviewers needs to approve", a rejection fails the run, and a run waiting on
  approval ends at the 35-day workflow-run limit, a period that "includes
  execution duration, and time spent on waiting and approval"; the sentence
  "if a job is not approved within 30 days, it will automatically fail" is
  the Enterprise Server 3.5 page's, and the current cloud pages do not state
  it. [EVIDENCE]
  [Managing environments](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment),
  [Actions limits](https://docs.github.com/en/actions/reference/limits),
  [Reviewing deployments, GHES 3.5](https://docs.github.com/en/enterprise-server@3.5/actions/managing-workflow-runs/reviewing-deployments)
- MCP's URL-mode elicitation (2025-11-25) is the standard envelope for
  "answer this on a web page": the server sends a `url` and an
  `elicitationId`, the client "MUST show the full URL to the user for
  examination before consent" and "MUST NOT open the URL without explicit
  consent", and the server "MAY" send `notifications/elicitation/complete`
  when the out-of-band interaction ends. [EVIDENCE]
  [MCP elicitation, 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation)
- Chat buttons are a notification, not a store: Slack posts the click to the
  app, which must acknowledge "within 3 seconds", and the `response_url` may be
  used "up to 5 times within 30 minutes"; the answer has to land somewhere
  durable of the app's own. [EVIDENCE] [Slack, handling user interaction](https://docs.slack.dev/interactivity/handling-user-interaction)
- Web Push is the standards route to a phone without an app store: messages
  are retained at the push service for a `TTL`, carry an `Urgency`, and the
  payload floor is 4096 bytes; iOS and iPadOS 16.4 deliver Web Push to
  home-screen web apps with no developer-programme membership. [EVIDENCE]
  [RFC 8030](https://www.rfc-editor.org/rfc/rfc8030.html),
  [WebKit, Web Push on iOS](https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/)
- The host's own media: a `PushNotification` tool sends "a desktop
  notification, and a phone push when Remote Control is connected", through
  the vendor's infrastructure; channels (Telegram, Discord, iMessage) push
  messages into a running session two-way, but "events only arrive while the
  session is open", and in `-p` mode "tools that need terminal input, such as
  multiple-choice questions ... are disabled so the session never stalls";
  projects show "Waiting on you" and notify on the desktop. [EVIDENCE]
  [Tools reference](https://code.claude.com/docs/en/tools-reference),
  [Channels](https://code.claude.com/docs/en/channels),
  [Projects](https://code.claude.com/docs/en/claude-projects)
- Pausing a graph "waits indefinitely" and a serialised run state can be
  resumed "in a different process" days later. [CONSENSUS] [LangGraph interrupts](https://docs.langchain.com/oss/python/langgraph/interrupts),
  [OpenAI Agents SDK, human in the loop](https://openai.github.io/openai-agents-python/human_in_the_loop/)
- Escalation after a timeout is standard but configured, with a default of 30
  minutes and a repeat count, and ends with the incident "assigned to the last
  user". [EVIDENCE] [PagerDuty escalation policies](https://docs.pagerduty.com/incident-management/schedules/escalation-policies)
- The open-source approval SDK this cluster would once have cited has turned
  into something else: the repository now describes a platform that "runs
  coding agents ... on your machines", with no approval primitives on its
  page. [ANECDOTE/MARKETING] (a README, read for what it no longer says)
  [humanlayer repository](https://github.com/humanlayer/humanlayer)

### Options weighed

1. The web page the product thinker chose, notified by Web Push, the answer
   written to the stop record through the relay.
2. A forge-native medium: the stop as a pull-request review request or an
   environment approval; durable, role-addressed and already in the product
   thinker's inbox, but the forge's approval is yes-or-no, not three options
   with a null.
3. A chat medium (Slack, Telegram): fastest to reach, but the session must be
   alive to receive the reply through the host's channels, and the chat's
   payload windows are short.
4. A native phone app: rejected by the checklist intent's decision unless
   lock-screen answers prove necessary.

### Recommendation

**Path 2.** Option 1 as decided, built so that the medium is an adapter over
the stop record of itd-169: the record is the truth, every medium is a
notification plus a way to write one answer back, and the first adapter is the
web page with Web Push, handed to from an MCP client by URL-mode elicitation.
The forge adapter (option 2) is the second, because it is the only medium
that is also durable and role-addressed on its own and already carries an
expiry shape (the 35-day run limit); the chat adapters are notification
only. Escalation to the facilitator after a period stays off by default and
configured with the facilitator's other settings (itd-174), as ruled. The Web
Push sender (VAPID, payload encryption) is a Go dependency wherever it is
built; the recommendation puts it in the relay, not the binary, and it needs
explicit sign-off either way.

### Local fit risks

- The user-level stance that agent messaging stays local or on the LAN sits
  against a relay-notified phone; the product thinker chose the relay for the
  checklist's private shared option, and the challenge should say where that
  stance ends.
- Host-delegated: the host's `PushNotification` and remote control are one
  vendor's; the page and Web Push are host-agnostic, which is why they come
  first.
- `widen-options-never-recommend`: a forge approval collapses three options to
  one; the forge adapter must link to the page for the choice and take only
  the acknowledgement.

## itd-57: a manual hold against autonomous pickup

### Findings

- The repository has since shipped a hold that is not a sentinel: a `held:
  "<reason>"` frontmatter key written by `abcd intent hold`, lifted by
  `unhold`, refused as free text by `record_provenance`, with `intent plan`
  refusing a held record and naming the reason and the lift
  ([iss-2609200830076665](../../../work/issues/resolved/iss-2609200830076665-a-held-draft-has-no-frontmatter-state-so.md), resolved); the build verb's readiness check
  reads "not held" (itd-2609201916151817, scope). [EVIDENCE] (the record)
- Every comparable system examined holds work with a first-class state and a
  recorded reason, not a never-finishing dependency. Kubernetes: a Job's
  `.spec.suspend`, "stable since Kubernetes v1.24", can be set at creation
  ("start Jobs in suspended state and have a custom controller decide later
  when to start them"), yields a `Suspended` condition whose
  `lastTransitionTime` says how long, and emits `Suspended`/`Resumed` events; a
  CronJob's `.spec.suspend` stops new runs, and "executions that are suspended
  during their scheduled time count as missed Jobs" that may run on resume.
  [EVIDENCE] [Kubernetes Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/),
  [CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
- Temporal's Workflow Pause (pre-release: invite-only on Temporal Cloud,
  Server v1.30.0 with `frontend.WorkflowPauseEnabled` self-hosted) "stops a
  specific Workflow Execution from making new progress until it is Unpaused",
  is "recorded in Event History with the identity, reason, and request ID",
  shows as `ExecutionStatus="Paused"` so a list filter finds held work, and is
  distinct from terminate; signals are stored while paused. [EVIDENCE]
  [Temporal workflow pause](https://docs.temporal.io/encyclopedia/workflow/workflow-pause)
- GitHub's hold is "until a human lifts it" by construction: required
  reviewers, a "Waiting" status, and an end when the run reaches the 35-day
  workflow-run limit, which "includes execution duration, and time spent on
  waiting and approval" (the 30-day failure is stated only by the Enterprise
  Server 3.5 page). [EVIDENCE]
  [Actions limits](https://docs.github.com/en/actions/reference/limits),
  [Reviewing deployments, GHES 3.5](https://docs.github.com/en/enterprise-server@3.5/actions/managing-workflow-runs/reviewing-deployments)

### Options weighed

1. The intent as written: a sentinel spec every held spec depends on, lifted
   with `rm-dep`.
2. The shipped `held:` state, extended to the spec grain, with who, when and
   why recorded, a "held: <reason>" line in the selector's skip output, and the
   exemption from progress nudges.
3. A hold with an expiry (GitHub's 35-day run limit): a hold that lapses is a
   hold that surprises; Kubernetes and Temporal have none.

### Recommendation

**Path 2**: the shipped `held:` state, already shaped like the SOTA
(Kubernetes suspend, Temporal pause), extended to the spec grain, in place of
the intent's own sentinel mechanism. The divergence is from the intent, not
from the SOTA, so under sota-per-intent this is Path 2 and no complexity stop
is invoked; the reason is
[one-canonical-primitive](../../principles/one-canonical-primitive.md): a
second hold mechanism beside the shipped `held:` state is the copy that
principle forbids. The proposal reverses the intent's own Out-of-Scope line
("A bespoke spec hold/unhold verb — v1 rides the existing native dependency
mechanism"), and says so for the interview to accept or reject. What remains
of the intent is option 2: the hold at the spec grain,
the recorded identity, time and reason (Temporal), the `Suspended`-style
visible state and reason in `next`'s skip output (the intent's third open
question), idempotent re-application, and the exemption of a held record from
"should make progress" nudges. The first two open questions (the sentinel's id;
eager or lazy creation) fall away. The product thinker has not ruled on this;
it is for the plan-time interview, where the intent is either re-scoped or
superseded. No Go dependency.

### Local fit risks

- `retire-the-name`: if the sentinel is dropped, its name and the `🔒 BLOCKED`
  banner convention go with it, not left as prose.
- `unrecognized-input-never-writes`: a hand-typed `held:` is already refused;
  the spec-grain hold must be written by a verb too.
- The record's Decisions carry no ruling against the sentinel; this note
  proposes, the interview decides.

## itd-2610120738075547: work handed to a cloud session comes back

The record is planned with every question answered; the host's surface was
read on 2026-10-12 and re-read on 2026-10-22. This pass checks the hand-back
contract against comparable hosted agents and against what the host added
since.

### Findings

- The first host's cloud surface now: `claude --cloud "<task>"` starts a
  session that clones the remote at the current branch ("push first if you
  have local commits"); `claude -p "<message>" --cloud <session-id>` queues a
  follow-up and with `--output-format json` returns `{ok, session_id, url}`;
  `--teleport <session-id>` pulls the session and its branch into a terminal
  (clean tree, same repository, branch pushed, same account); there is still
  no scriptable listing of cloud sessions from the CLI (`--teleport` with no
  id is "an interactive session picker", `/tasks` lists background sessions,
  and the page otherwise says "find the ID in your session list at
  claude.ai/code"); a session whose environment expires
  restores the conversation but not background work or session-scoped
  schedules; and projects now "coordinate parallel cloud sessions" per account,
  in public beta on individual plans only. [EVIDENCE] [Claude Code in the cloud](https://code.claude.com/docs/en/claude-code-on-the-web),
  [Projects](https://code.claude.com/docs/en/claude-projects)
- The host's own hand-back for coordinated work is a report to a conversation
  the vendor holds: a thread "reports back to the conversation when it
  finishes", "works on its own branch and opens a pull request when the work
  calls for one", and the coordinator "sees what threads report back, not
  every step they take"; when threads stall the advice is to ask each "what it
  assumed or couldn't reach, and what it's waiting on" and to "fix the gap
  once". [EVIDENCE] [Projects](https://code.claude.com/docs/en/claude-projects)
- A second vendor's coding agent runs "in an ephemeral cloud development
  environment", "can only work on one branch at a time and can open one pull
  request per task", leaves "session logs" that "show the work and tools
  used", and has a 59-minute session cap that "cannot be extended or
  bypassed": "if a task exceeds the limit, the session times out and stops".
  [EVIDENCE] [GitHub Copilot coding agent](https://docs.github.com/en/copilot/concepts/agents/coding-agent/about-coding-agent)
- A third vendor's cloud tasks each get their own workspace, and its CLI can
  "start a task from your terminal, or list recent cloud chats and check their
  status"; results are inspected and turned into a pull request from the task.
  [EVIDENCE] [Codex cloud](https://learn.chatgpt.com/docs/cloud)
- The hosted scheduled-task runs push to `claude/` branches and are governed by
  branch protection applied to "the GitHub access you connected", so a rule
  that access can bypass does not block a run. [EVIDENCE] [Claude Code scheduled tasks](https://code.claude.com/docs/en/routines)
- The git-proxy restrictions (branches only, no deletions, GraphQL blocked) and
  the probe that found them are in the 2026-10-03 note's review evidence.

### Options weighed

1. The planned contract: a work-in-progress branch, a report file committed on
   it, a tracking-issue comment only when the push fails, a list and a collect
   that read the remote alone.
2. Read the host's project or session list through the host instead: exists
   for one vendor (Codex's CLI) and is per-account and beta for the first host;
   nothing host-agnostic reads it.
3. A report posted to the forge first (issue or pull-request body) rather than
   committed: readable without a clone, but it depends on the forge's REST
   path being open and is the fallback for that reason.

### Recommendation

**Path 2**, the planned contract confirmed, with one addition the comparable
systems argue for: a hard session cap and an environment that expires mean
work exists only once it is pushed, so the brief tells the session to push the
work-in-progress branch after every commit, not only at the end, and the
report file is written early as "started" and overwritten at the end, so a
session cut off mid-task still leaves a branch and a partial report (C6 and C7
today speak of the end). The seam stays the host connector: when a host offers
a session listing (option 2), the connector may enrich the list, and the
remote-only read remains the floor every host meets. No Go dependency.

### Local fit risks

- Decision 9 (every cloud result re-checked locally) is untouched; early pushes
  change nothing about trust.
- Host-agnostic core: the project surface is one vendor's and per-account; it
  never becomes a source of truth the core reads.
- Outbound text: a report overwritten several times is re-read for session
  links and footers on each write, not only the last.

## itd-2609180517121254: every returned payload names its model and agent count

### Findings

- OpenTelemetry's GenAI conventions (status: Development) name the fields:
  `gen_ai.request.model` ("the name of the GenAI model a request is being made
  to"; the create-agent span's footnote says it "must be the exact name of the
  model requested", and the invoke-agent span marks it Recommended "If
  applicable" and says it "SHOULD NOT be populated for agents that support
  multiple models or dynamic selection"), `gen_ai.response.model` ("the name
  of the model that generated the response", Recommended on the model-call
  spans and not listed on the invoke-agent span), `gen_ai.provider.name`
  (Required),
  `gen_ai.agent.id`, `gen_ai.agent.name`, `gen_ai.conversation.id`,
  `gen_ai.usage.input_tokens` (which "SHOULD include all types of input
  tokens, including cached tokens") and `gen_ai.usage.output_tokens`; the
  invoke-agent span marks `gen_ai.request.model` Recommended and the agent id
  and name Conditionally Required. [EVIDENCE] [GenAI agent spans](https://raw.githubusercontent.com/open-telemetry/semantic-conventions-genai/main/docs/gen-ai/gen-ai-agent-spans.md),
  [GenAI spans](https://raw.githubusercontent.com/open-telemetry/semantic-conventions-genai/main/docs/gen-ai/gen-ai-spans.md),
  [the conventions repository](https://github.com/open-telemetry/semantic-conventions-genai)
- The host's run already carries the facts: `system/init` reports the session's
  `model`; the result's `modelUsage` is keyed by model name and "counts
  subagent requests alongside the top-level loop, broken down by model", where
  `usage` "counts only the top-level agent loop"; every sub-agent message in
  the stream carries a `parent_tool_use_id`, so "you can rebuild the full
  nesting tree"; `SubagentStop` carries `agent_id` and `agent_type`; and the
  cost fields are "client-side estimates, not authoritative billing data".
  [EVIDENCE] [Agent SDK cost tracking](https://code.claude.com/docs/en/agent-sdk/cost-tracking),
  [Run Claude Code programmatically](https://code.claude.com/docs/en/headless),
  [Claude Code hooks](https://code.claude.com/docs/en/hooks)
- W3C PROV gives the provenance words: an entity `wasGeneratedBy` an activity,
  `wasAttributedTo` an agent, an activity `wasAssociatedWith` an agent, and an
  agent `actedOnBehalfOf` another; `SoftwareAgent` is a subclass of `Agent`.
  [EVIDENCE] [PROV-O](https://www.w3.org/TR/prov-o/)
- The served model differs from the asked model across providers, and the
  measured spread between pairings is in the 2026-10-16 note; this record's
  scope condition in the build verb ("recorded as the runner reported it; the
  binary cannot verify it") already states the limit.

### Options weighed

1. Field names of abcd's own (`model`, `agents`).
2. OTel GenAI attribute names verbatim as the payload's keys, with an agent
   list keyed by `gen_ai.agent.id`, both requested and served model recorded,
   and a convention version pinned in the payload because the status is
   Development.
3. The OTel SDK and a trace exporter: infrastructure the 2026-07-30 note
   already judged a poor fit at this scale.

### Recommendation

**Path 1 for the names, Path 2 for the mechanism.** Option 2: the payload
carries `gen_ai.provider.name`, `gen_ai.request.model`, `gen_ai.response.model`
and an `agents` list of `{gen_ai.agent.id, gen_ai.agent.name,
gen_ai.response.model, gen_ai.usage.input_tokens, gen_ai.usage.output_tokens}`,
with the agent count defined as the number of distinct agent ids that ran, the
top-level one included; the ingesting verb refuses a payload whose
`gen_ai.response.model` is absent or whose agent list is empty. The per-agent
`gen_ai.response.model` is abcd's extension of the convention, whose agent
span does not define it, and the agents list is abcd's composition of OTel
names. The refusal is satisfiable from the host with two caveats: the final
result supplies only the per-model totals (`modelUsage` keyed by model;
`SubagentStop` carries `agent_id` and `agent_type` but no model), so the
per-agent served model and tokens are rebuilt from the `stream-json` messages
(`parent_tool_use_id` plus each message's model and usage); and after a
process crash the final `error_during_execution` result "may carry zeroed
`usage`, `total_cost_usd`, and `modelUsage`", which would yield an empty
agent list for a run that did work, so the ingesting verb records such a
payload as "unknown" with the crash named rather than refusing it as no work.
The PROV words name the
human the run acted on behalf of, which is the attribution rule's human author.
Names only; no OTel SDK and no Go dependency.

### Local fit risks

- `a-model-run-is-re-runnable-not-reproducible`: the served model is recorded
  as reported, never claimed as verified.
- Host-delegated: the mapping from one host's result JSON to the OTel names is
  the runner adapter's; the core knows the names.
- Development-status names may move; the payload pins the convention version
  so a rename is a migration, not a silent mismatch.

## itd-59: every autonomous pass leaves the same transcript as a session

The 2026-07-30 note settled the store, the redaction door and the hook as the
capture path. This pass checks what the host's non-interactive mode does to
that path.

### Findings

- A `claude -p` session is written to the same transcript store
  (`~/.claude/projects/<project>/<session-id>.jsonl`), is resumable by id, and
  is left out of the picker; sub-agent transcripts persist independently under
  the session's `subagents/agent-<id>.jsonl` and are deleted after
  `cleanupPeriodDays` (30 days by default); `--no-session-persistence`
  suppresses a run's transcript and `CLAUDE_CODE_TRANSCRIPT_LOCAL_GC` bounds a
  `-p` transcript's growth. [EVIDENCE] [Manage sessions](https://code.claude.com/docs/en/sessions),
  [Claude Code subagents](https://code.claude.com/docs/en/sub-agents)
- Without `--bare`, "a `-p` session runs the hooks in a project's
  `.claude/settings.json`" and loads plugins; with `--bare`, which is
  "recommended" for scripts and "will become the default for `-p` in a future
  release", none of them run. On SIGTERM the host "runs `SessionEnd` hooks and
  exits" and "runs no hook other than `SessionEnd`" while exiting; the Stop
  hook's `transcript_path` "is written asynchronously and may lag". [EVIDENCE]
  [Run Claude Code programmatically](https://code.claude.com/docs/en/headless),
  [Claude Code hooks](https://code.claude.com/docs/en/hooks)
- `SubagentStop` carries `agent_transcript_path`, "the subagent's own
  transcript stored in a nested `subagents/` folder", which abcd's hook already
  captures; `--forward-subagent-text` on `stream-json` lets a driver
  "reconstruct each run's transcript" from the stream instead. [EVIDENCE]
  [Claude Code hooks](https://code.claude.com/docs/en/hooks),
  [Run Claude Code programmatically](https://code.claude.com/docs/en/headless)
- The transcript "entry format is internal to Claude Code and changes between
  versions, so scripts that parse these files directly can break on any
  release". [EVIDENCE] [Manage sessions](https://code.claude.com/docs/en/sessions)
- The guardrail consensus (transcripts hold secrets, stay out of the repository,
  are untrusted input) and the OTel-export alternative are in the 2026-07-30
  note.

### Options weighed

1. (a) Capture at the seam through the same `SessionEnd`/`SubagentStop` hooks,
   with the loop never passing `--bare` (or passing the plugin explicitly once
   `--bare` is the default) and handing the run id, lane, pass and phase to the
   hook through the lane's environment so the capture is keyed.
2. (b) Ingest after the fact by session id, from the result's `session_id` and
   the sub-agent paths, when the hook did not fire (a SIGTERM window, a host
   that skipped hooks).
3. (c) A structured-stream sink from `stream-json` with sub-agent text
   forwarded: complete, but a second format beside the one the store already
   reads.

### Recommendation

**Path 2.** Option 1 as the primary path and option 2 as the fallback the loop
runs when the receipt names a session whose capture is missing, both through
`history.Capture` and its redaction door, best-effort as the criteria require.
The capture stores the raw transcript and derives its index, because the
format is declared unstable; the run id, lane and phase are keyed from the
environment the loop sets, not parsed from the transcript. No Go dependency.

### Local fit risks

- Host-delegated and host-agnostic: `--bare`, the hook names and the paths are
  one host's; the runner adapter owns them, and the store's keys do not.
- `durable-state-lives-where-the-platform-says-it-survives`: the host prunes
  sub-agent transcripts after 30 days; the capture must run before that and
  never rely on the host's directory as the store.
- Best-effort capture is a stated criterion; the fit-challenge should ask
  whether a lane whose capture failed may still land, given the record's
  promise of auditability.

## Not worth adopting

- A durable-execution engine (Temporal, DBOS) for the build loop: each needs a
  server or a Postgres; the loop's state file already gives the one property
  that matters (replay from recorded outcomes), as the 2026-10-17 note found.
- The host's `--bare` for lanes: it is the host's recommendation for scripts,
  and it removes the hooks abcd's capture and guard depend on.
- A learned or router-based sub-agent class (2026-09-22 note), and sizing the
  pool from live rate-limit headers in a subscription host that reports none.
- A sentinel dependency as the hold mechanism: every comparable system holds
  with a state and a reason, and the repository already ships one.
- The OTel SDK and a trace pipeline for provenance: the names are worth
  adopting, the infrastructure is not at this scale (2026-07-30 note).
- A chat channel as the stop's medium of record: the reply windows are short
  and the host's channels need a live session.
- SQLite for the checklist: the literature's safe default, but a dependency
  and a second store beside the record files; the canonical writer plus the
  lab is the proportionate answer.
- The Ralph loop's "same prompt forever": its author's own failure list is the
  argument for a state file and a stop check.

## Not verified

- Whether any OTel span defines a per-agent served model: the invoke-agent
  span does not list `gen_ai.response.model` at all (confirmed on 2026-10-23);
  the definition cited is from the model-call tables of the spans document,
  and the per-agent field is abcd's extension.
- The Messages API response object's own wording for its `model` field: the
  page was too long to reach the response section in this pass.
- A current github.com statement of a 30-day deployment-approval expiry:
  only the Enterprise Server 3.5 edition of "Reviewing deployments" states
  it; the current limits page gives a 35-day workflow-run limit that includes
  waiting and approval, which the sections above now cite.
- A second vendor's model-class names beyond the one whose models page was
  opened in the 2026-10-23 adjudication (a flagship tier above Pro, Flash and
  Flash-Lite).
- Kubernetes' full sentence on CronJob suspension (the page was read in two
  parts and the sentence was split across them); the caution on missed jobs
  was read whole.
- Whether the host's task-list tools (`TaskCreate`, `TaskList`) persist their
  list on disk, and where.
- Slack's interaction payload field names for the clicking user and the
  action; the page described them without naming them.
- The earlier approval SDK the humanlayer repository once was; its README now
  describes a different product and names no successor.
- Any measured comparison of host-prompt pause mechanisms (Stop hook, `/goal`,
  `/loop`) for stall rates; none was found.
- Codex cloud's parallel-task limits and failure reporting; the page did not
  state them.
