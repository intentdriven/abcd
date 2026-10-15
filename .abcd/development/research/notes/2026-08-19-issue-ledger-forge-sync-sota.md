# Issue ledger ↔ forge sync — SOTA survey (2026-08-19)

Commissioned host-delegated research pass on reconciling an in-repo, file-based
issue ledger with forge-hosted issues (GitHub Issues), given two hard
constraints: (a) autonomous CI/cloud agents produce findings, (b) some users
have no forge access and must read/write the record with only a checkout.
Recency assessed against 2026-08-19. This note grounds the 2026-08-19 decision
[dcn-2608190000000001](../../../work/decisions/dcn-2608190000000001-the-abcd-work-issues-ledger-is-the.md) (ledger-canonical, one-way mirror).

## Ranked findings

1. **In-repo file ledger canonical; forge issues a derived view.** [CONSENSUS]
   The one team with a post-mortem-grade public rationale, CoreOS's
   GitHub↔JIRA syncer, deliberately went one-way: "rather than require people
   to keep up with both sources, we decided to make *one* the single source of
   truth" (coreos/issue-sync, archived 2023). Fossil (the longest-lived
   in-repo tracker) avoids reconciliation by having exactly one store. The
   no-forge-users constraint forces the choice regardless: anything that
   exists only on the forge is invisible to them. Plain committed markdown is
   where the 2025-26 agent-tooling wave landed (Backlog.md: the same .md files
   humans read are the agent API, "no translation layer or sync issues").

2. **One-way sync (canonical → mirror), external id written back into the
   canonical record.** [EVIDENCE] Bidirectional sync has three documented
   failure modes — echo/"vampire record" loops, origin ambiguity,
   timestamp-only conflict detection under clock skew — and every mature
   mitigation (origin tagging, idempotent upserts keyed on a stable external
   id, per-field ownership) exists only to contain them. Mechanisms proven in
   the survivors: git-bug bridges stamp exported operations with
   `github-id`/`github-url` so re-import recognises its own exports; git-issue
   (Spinellis) keeps an imports mapping directory plus a checkpoint SHA. The
   file-ledger equivalent is one frontmatter field per record: a record
   carrying a forge id is updated in place on the mirror, never re-created,
   and a mirror event matching a known id with unchanged content is dropped.

3. **Status truth lives in the canonical store; the mirror auto-closes on
   canonical resolution.** [EVIDENCE] ClusterFuzz/OSS-Fuzz is the production
   precedent: fully automatic filing, triage, and closing where the tracker
   issue is a projection closed when the canonical evidence resolves. Forge
   triage is never synced automatically — it generates an import proposal a
   human accepts by an explicit import that writes the canonical file with
   provenance (forge id, actor, timestamp). A forge-side close never imported
   is reopened by the next mirror pass — self-healing, and the property that
   makes the topology safe.

4. **Agents enter findings as files-via-PR, validated by repo lint/CI; forge
   issues are at most a fallback inbox.** [CONSENSUS, strong negative
   evidence for the alternative] GitHub Advisory Database: finding records as
   committed OSV files entering by reviewed PR at scale. GitHub Security Lab
   Taskflow (2026-01) and Google Big Sleep gate every AI-found report on human
   review; Taskflow feeds reviewer dismissal reasons back into later runs as
   its cross-run suppression mechanism. The negative case is curl's tracker:
   ungated agent-generated reports drove signal from ~1-in-6 to ~1-in-20/30
   and forced the bounty to halt (2025-07). Practical shape: the finding
   file's id is a **stable content-derived fingerprint** (ClusterFuzz crash
   signatures are the model), so a repeat run produces a no-op diff and lint
   enforces fingerprint uniqueness.

5. **Steal mechanisms from git-bug and git-issue; adopt neither as the
   store.** [EVIDENCE on liveness; CONTESTED on viability] git-bug is the
   healthiest survivor (releases through 2025-05) but stores issues as git
   objects, not files — unreadable from a plain checkout. git-issue has the
   best file-based external-id mapping but no maintenance signal since 2020.
   Fossil and Radicle solve the problem only by replacing the VCS/forge
   wholesale. BugsEverywhere and Simple Defects are dead; the post-mortem
   consensus on what kills in-repo trackers: no shared schema, invisibility to
   non-developers, and "90% of bug tracking is sending messages to other
   people".

## Not worth adopting

- **True bidirectional sync** — enterprise-scale conflict-rule maintenance,
  no payoff at this scale. [CONSENSUS]
- **git-bug as storage** — hidden git objects defeat checkout readability;
  its bridge code remains the best reference implementation. [CONTESTED]
- **Fossil/Radicle migration** — correct architecture, wrong cost. [CONSENSUS]
- **CRDT/event-log issue formats** — they solve concurrent-edit merging that
  one canonical repo with PR-mediated writes mostly doesn't have. [ANECDOTE]
- **Ungated autonomous filing** — curl is the controlled-ish experiment;
  even the frontier labs gate on human review. [EVIDENCE]

## Sources

- <https://github.com/coreos/issue-sync>
- <https://fossil-scm.org/home/doc/trunk/www/bugtheory.wiki>
- <https://truto.one/blog/the-architects-guide-to-bi-directional-api-sync-without-infinite-loops/>
- <https://github.com/git-bug/git-bug> and
  <https://github.com/git-bug/git-bug/blob/master/doc/usage/third-party.md>
- <https://github.com/dspinellis/git-issue/blob/master/README.md>
- <https://github.com/MrLesk/Backlog.md>
- <https://google.github.io/clusterfuzz/>
- <https://github.blog/security/ai-supported-vulnerability-triage-with-the-github-security-lab-taskflow-agent/>
- <https://daniel.haxx.se/blog/2025/07/14/death-by-a-thousand-slops/>
- <https://github.com/github/advisory-database/blob/main/README.md>
- <https://news.ycombinator.com/item?id=43971620> (git-bug, 2025-05)
- <https://news.ycombinator.com/item?id=48155570> (Epiq)
- <https://lwn.net/Articles/966869/> and <https://radicle.dev/>
