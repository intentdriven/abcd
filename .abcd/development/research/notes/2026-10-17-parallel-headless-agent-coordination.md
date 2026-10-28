# Coordinating parallel headless agents on one repository: state of the art

Dated 2026-10-17. A state-of-the-art pass behind the draft intent
[itd-2610170634394993](../../intents/planned/itd-2610170634394993-several-headless-agents-work-one.md), asked for by the technical facilitator
after the evening of 2026-10-16. The question: how abcd, run from the
command line, should coordinate several headless agents working in
parallel on one repository, so they use the machine and the models well
and do not collide. Every source below was opened in the pass. Tiers: T1
is primary documentation or peer-reviewed work, T2 a credible engineering
write-up, T3 a blog or anecdote.

## What the evening of 2026-10-16 showed

- Sessions held ordering in conversation. One session asked a new session
  to hold its records pull request until another pull request landed, so
  that the first would not fall into conflict again, and promised to tell
  four waiting sessions when it merged. A headless agent has no
  conversation to receive that in, and the promise lives only as long as
  the session that made it.
- The machine's load gate deadlocked for hours across two builds of its
  own queue rule: a waiter on the newer rule and two on the older rule
  each saw the other as ahead. Nothing timed out for eight hours.
- Two preflights, each held behind the load gate for its full two hours,
  failed on the gate alone with every check before it passing.

## Findings

### Shared state, not a live coordinator

- Kubernetes controllers each compare the wanted state with the current
  state and share nothing but that state; when one fails, another takes
  over. T1, [Kubernetes controllers](https://kubernetes.io/docs/concepts/architecture/controller/).
- A lease is a compare-and-swap with an expiry: the holder renews at half
  the lease duration, and when candidates race for an expired lease only
  one update succeeds. T1, [Kubernetes coordinated leader election](https://kubernetes.io/docs/concepts/cluster-administration/coordinated-leader-election).
- Expiry alone does not make a lock safe, because a paused holder can
  still write after its lease lapsed. Chubby hands out sequencers (lock
  name, mode and a generation number) for the receiving server to check,
  and a lock-delay of up to a minute where it cannot. T1, Burrows, OSDI
  2006, §2.4 ([page](https://research.google/pubs/the-chubby-lock-service-for-loosely-coupled-distributed-systems/)).
  Kleppmann names the same idea fencing tokens. T2, [Kleppmann 2016](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html).
- Job queues recover work by timeout, never by noticing a death. Faktory
  re-queues a job not acknowledged within its reservation (30 minutes by
  default). T1, [Faktory](https://cdn.jsdelivr.net/gh/contribsys/faktory@main/README.md).
  Temporal does not detect a crashed worker; it relies on a wait-to-start
  timeout, a per-attempt timeout and a heartbeat that can carry progress.
  T1, [Temporal](https://docs.temporal.io/encyclopedia/detecting-activity-failures).

### Merging work that depends on other work

- GitHub's merge queue tests each pull request with every one ahead of
  it, removes a failure and rebuilds those behind it. T1, [GitHub](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue).
- Mergify reads a `Depends-On: #123` line in a pull request's body and
  holds the merge until #123 lands; a dependency closed unmerged, or one it
  cannot resolve, fails rather than waits, and cycles are not detected.
  T1, [Mergify](https://docs.mergify.com/merge-protections/builtin).
  git-spice merges a stack from the bottom and restacks what sits above
  each merged branch. T1, [git-spice](https://abhinav.github.io/git-spice/).
- Aviator splits the queue by declared affected targets: changes that share
  none merge independently. T1, [Aviator](https://docs.aviator.co/mergequeue/concepts/affected-targets).

### Multi-agent coding

- An orchestrator-worker system (Opus lead, Sonnet workers) beat a single
  agent by 90.2% on Anthropic's research evaluation, used about fifteen
  times the tokens, and its authors note that most coding tasks hold fewer
  truly parallel parts than research. T2, [Anthropic, June 2025](https://www.anthropic.com/engineering/multi-agent-research-system).
- The counter-view: actions carry implicit decisions, and parallel agents
  make conflicting ones. T2/T3, [Cognition, June 2025](https://cognition.com/blog/dont-build-multi-agents).
  A taxonomy of 1,600 multi-agent traces finds 14 failure modes and gains
  over baselines that are often minimal. T1, [Cemri et al.](https://arxiv.org/abs/2503.13657).
- Measured conflict: 27.67% of about 107,000 simulated agent pull requests
  conflicted ([AgenticFlict](https://arxiv.org/abs/2604.03551)); pull
  requests open at the same time conflicted 19.8% of the time from one
  agent and 41.7% across agents ([Xu et al. 2026](https://arxiv.org/abs/2607.04697));
  telling an agent what had already landed recovered 82% of constructed
  "pass alone, fail together" runs ([Xia et al. 2026](https://arxiv.org/abs/2609.25396)).
  T1, preprints.
- Claude Code agent teams coordinate through a shared task list with
  dependencies and claim tasks under a file lock, advise that no two
  teammates edit one file, and are bound to one session on one machine.
  T1, [agent teams](https://code.claude.com/docs/en/agent-teams).

### Budgets

- The make jobserver hands out one token per job slot, and a tool that
  fails to return its tokens loses those slots. T1, [GNU make](https://www.gnu.org/software/make/manual/html_node/Job-Slots.html).
  Airflow lets one task take several slots of a pool. T1, [Airflow pools](https://airflow.apache.org/docs/apache-airflow/stable/administration-and-deployment/pools.html).
- Kubernetes queues requests per flow, so one client cannot starve the
  rest. T1, [API priority and fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/).
- Model rate limits are a token bucket per organisation, reported in
  response headers, so they span every account and machine of one
  organisation. T1, [Anthropic rate limits](https://platform.claude.com/docs/en/api/rate-limits).

### Mixed versions and waits

- Kubernetes' coordinated election keeps lease renewal unchanged so old
  and new instances can share one lease, prefers the oldest version as
  leader, and admits it cannot pick the best leader while versions are
  mixed. T1, [KEP-4355](https://www.kubernetes.dev/resources/keps/4355/).
  Kubernetes allows one minor version of skew. T1, [skew policy](https://kubernetes.io/releases/version-skew-policy/).
- Temporal refuses an activity with no per-attempt or overall timeout.
  T1, Temporal, above.

## Recommendations, ranked

1. Put a generation number on every reservation, re-checked before any
   act that matters (push, queue, tag), with a delay before an expired
   claim can be retaken.
2. Make "waits on #N" a line abcd reads and enforces: no queueing while the
   dependency is open, a failure (not a wait) on one closed unmerged or
   unresolvable, cycle detection, and the dependent branch stacked on its
   dependency and restacked when it lands.
3. Route merges by footprint: disjoint changes merge independently,
   overlapping ones in order, and an agent re-submitting is told what
   landed since its base.
4. Give every wait a deadline: a wait-to-start limit, a per-attempt limit
   and a heartbeat carrying progress.
5. Turn the load gate into a weighted slot pool, with a slot reclaimed when
   its holder's process is gone.
6. Version the coordination records: a reader refuses a newer format, the
   oldest participant's rules decide, and one version of skew is supported.
7. Hold the release cut as a lease the merge path checks.
8. Size the sub-agent budget from the rate-limit headers, starting at three
   to five workers.

Not worth adopting at abcd's scale: a coordinator process that must stay
alive (an etcd or Temporal cluster), Uber-style speculative build trees,
and locks that rely on timing without a fencing check.

## Not verified

How Cursor, Devin, OpenHands, SWE-agent, MetaGPT, AutoGen and LangGraph
divide work; Codex cloud's parallelism; any measured comparison of
orchestrator-worker and peer arrangements on coding tasks. None was found
or opened.

## The central view and each session's detail

A second pass the same day, after the product thinker asked for a central
place to see at a glance what is taken and worked on, with each session's
details kept within that session so they survive compaction, and named the
link between the two as the hard part. Every source below was opened in the
pass.

### Findings

- Kubernetes separates what was asked for (spec) from what is observed
  (status). The owner writes status through its own subresource, and data
  that changes often belongs in separate objects. T1, [API conventions](https://raw.githubusercontent.com/kubernetes/community/master/contributors/devel/sig-architecture/api-conventions.md).
  A node sends a light lease every ten seconds and its full status only on
  change or every five minutes. It reads as not ready after about forty to
  fifty seconds without a lease, and its pods are evicted five minutes later.
  T1, [node status](https://kubernetes.io/docs/reference/node/node-status/).
- Presence systems show more than live and dead:
  - Phoenix Tracker has no single source of truth and moves an entry from
    live to down after 30 seconds and to gone after 20 minutes. T1, [Tracker](https://phoenix-pubsub.hexdocs.pm/Phoenix.Tracker.html).
  - Yjs awareness marks a peer offline after 30 seconds without an update.
    T1, [Yjs](https://docs.yjs.dev/api/about-awareness).
  - Consul treats a time-to-live as a lower bound and holds a lock for 15
    seconds after it expires before anyone may retake it. T1, [Consul sessions](https://developer.hashicorp.com/consul/docs/automate/session).
- Git can carry one record per writer, so writers never contend:
  - Gerrit keeps each change's metadata under its own ref. T1, [NoteDb](https://gerrit-review.googlesource.com/Documentation/note-db.html).
  - git-bug keeps one ref per entity and merges concurrent edits with
    Lamport clocks. T1, [git-bug data model](https://github.com/git-bug/git-bug/blob/master/doc/design/data-model.md).
  - A push can be made conditional on the ref's current value, and can
    update several refs atomically. T1, [git push](https://git-scm.com/docs/git-push).
  - GitHub recommends at most six pushes a minute per repository and warns
    that many branches add data to every fetch. T1, [repository limits](https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits).
- Current multi-agent tools keep their state on one machine:
  - Claude Code's agent view keeps each session's state in a local file,
    and its docs say plainly that sessions are local. T1, [agent view](https://code.claude.com/docs/en/agent-view).
  - Agent teams admit that task status can lag because teammates do not
    always mark work done. T1, [agent teams](https://code.claude.com/docs/en/agent-teams).
  - Claude Squad, uzi and Conductor also keep worktree state on one
    machine. T1, [Claude Squad](https://github.com/smtg-ai/claude-squad),
    [uzi](https://github.com/devflowinc/uzi),
    [Conductor](https://www.conductor.build/docs/concepts/workspaces-and-branches).
  - Hosted tools (Codex cloud, Cursor cloud agents, Copilot's agent, Devin)
    serve one vendor's account. None found coordinates across accounts or
    machines over plain git.
- Working state outside the context:
  - After compaction Claude Code re-reads its memory files and the output of
    session-start hooks. T1, [context window](https://code.claude.com/docs/en/context-window).
  - Anthropic's long-running harness keeps a progress file and a feature
    list in JSON, which the model is less likely to overwrite than Markdown,
    and says compaction alone is not enough. T2, [effective harnesses](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents).
- Event sourcing would make the view a replayable projection, but its
  maintainers call it complex and unnecessary for most systems. T1, [event sourcing](https://learn.microsoft.com/en-us/azure/architecture/patterns/event-sourcing).

### Options weighed

1. Each session's checklist is the source of its detail; the session
   publishes one short status record of its own when that record changes,
   after the checklist is written; the claim renewal shows liveness; the
   view is every record folded together by whoever reads it. Copies
   Kubernetes' node status and lease, and Gerrit's per-entity refs.
2. Each session also publishes its checklist history, so another computer
   can rebuild a dead session's state. Copies git-bug and event sourcing.
3. Option 1, plus an instant tier for sessions on the same computer read
   from the process table.
4. One shared notes record every session appends to. Copies git-appraise.
   Rejected: every session contends for one record, and dead entries need
   explicit removal.

The product thinker chose option 1 on 2026-10-17, leaving option 2 as
something to add later. Option 3's instant local tier stays open. A line's
state is derived from what the machine and the remote can see, with what
the agent writes about itself as a hint, because agents' own status lags.

### Not verified

Figma's presence model, Sculptor's internals, Conductor's crash handling,
the meaning of Devin's status values, git-dit (its repository did not
load), whether GitHub enforces or only recommends its push rate, and where
Codex and Vibe Kanban keep their state.
