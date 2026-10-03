# Multi-agent coordination: Record inventory, run lessons and the state of the art

Dated 2026-10-03. A synthesis written the day a second person began working on
abcd and the repository was public, to ground the product thinker's rulings on
how agents reserve work across people (itd-2609091034175565,
itd-2609150819440345 decision 4, itd-2610031259176838). It was composed in
the local tier and promoted here, scrubbed of machine-private detail, so that
the second person can read the grounds the rulings rest on. The closing
section folds in the evidence of the two adversarial reviews of
itd-2609091034175565 held the same day.

Three scopes are distinguished throughout:

- **S1**: One account on one machine.
- **S2**: Several accounts on one machine.
- **S3**: Several contributors on several machines, meeting only at the remote.

## Record inventory

- **S1, shipped.** itd-2609221656373558 (two orchestrators share one run;
  exclusive-create file claims with a two-hour lease in the machine-scoped run
  store; other accounts out of scope), itd-2609091416295622 (`abcd peers`,
  records only, other accounts out of scope), itd-114 and adr-45
  (collision-free ids), and `abcd build` refusing a peer-claimed intent.
  Planned: itd-148 (a declared coordinator; no code yet) and
  itd-2609091014076309 (the worktree store's verbs).
- **S1, drafts and issues.** itd-2609091034175565 (then a `claimed_by` stamp),
  itd-2609150819440345 (the register; transport undecided), itd-33
  (`active-work.json`; in tension with adr-2609091248200336 and
  iss-2609240227595724), iss-2608220750029993 (presence),
  iss-2609091037191879 (a release cut invisible to peers),
  iss-2609100519122086 (major), iss-213 (major), iss-2608230847432285
  (major), iss-2609240646546286 (cluster claims).
- **S2.** Nothing shipped beyond defensive refusals of foreign-owned roots;
  drafts itd-2609151838312703 (a signed mailbox) and itd-2609151838327688 (a
  local broker); the decision log's 2026-09-22 ruling keeps cross-account
  communication out of the run.
- **S3.** Merge plumbing only (adr-43's queue, the RS001 to RS006 gates);
  iss-2609020716570699 (major, duplicate re-fixes), iss-344, iss-193 (major),
  iss-2609211105023379 (major). Nothing planned.
- **An abandoned branch.** A local, never-pushed branch held fourteen commits,
  thousands behind the default branch: a `capture claim` writing `claimed_by`
  (holder = account plus harness, no expiry) and an advisory duplicate guard
  whose rule id collides with the default branch's RS004. It delivered an
  intent that exists only on that branch and is superseded by the 2026-09-09
  split recorded in the decision log.
- **Contradictions.** Four claim designs coexist; the brief's build surface
  (`04-surfaces/34-build.md`) says a claim stops a "second clone", which is
  true only within one account; adr-43's required reviewer sits against a
  later ruling.

## Run lessons

Ranked. Sources: The autonomous runs' handover notes and run logs (local and
machine-scoped, never committed), the two accounts' exchange through a shared
note file, the run files, and
[`2026-09-24-autonomous-run-a.md`](2026-09-24-autonomous-run-a.md).

1. Records carry a run; session habits die at handover (measurement stopped
   at the first rotation; rulings were re-read from prose).
2. The exclusive-create lease claim works in one account (no collisions) but
   is per record, per session and per account; every rotation re-claimed by
   hand, and a peer outside the run re-fixed two issues.
3. Presence sees records, not sessions: Agents mistook each other for foreign
   peers, and a stuck orchestrator could double-orchestrate.
4. The merge is the structural serial point (the forge ignores
   `merge=union`, so a record conflict reads as dirty; a pull request behind
   its base disarms auto-merge; the preflight lock is not first-in,
   first-out). The fix was integration branches of at most four lanes, kept
   only as a handover rule.
5. Beyond about five agents, machine load, not slots, limits throughput (load
   averages of 134 to 535); the ceiling is counted per session, not per
   machine.
6. Actions on a shared machine cross session and account lines (a broad
   process kill, a panic over orphaned load, a shared stash).
7. The agent ceiling is blind to forks and relayed agents.
8. Rotation at about 60% of context works; notifications reach only the
   launching session; paths were tied to a session's scratch directory.
9. Crashes were noticed late (about 46 minutes) and recovered by hand; a
   usage limit stalled a run for about 33 hours; there is no watchdog.
10. Holds before a release cut are cooperative and lose races (a pull request
    was queued before the note asking for a hold was read).
11. Only the decision log carried authority; rulings sat in scratch, and the
    product thinker could not see which session ran the run.
12. The single slot for the review model is a machine-wide resource.
13. One intent per lane, and a fresh agent per fix round.
14. Isolation is a checkout in the machine store, never the harness's own
    isolation.
15. Home-directory and account-name collisions across accounts.
16. Attribution must read the model id at start, through hooks.
17. Autonomy gaps: Opening sessions, switching accounts, restarting after a
    crash, and rulings.

Not tested: S3 at all; S2 beyond one exchange of notes; contested leases;
three or more orchestrators.

## State of the art

Ranked.

1. **A claim is a create-only compare-and-swap on a git ref.** In S1,
   `git update-ref refs/abcd/claims/<id> <sha> ""` on the shared ref store,
   offline; in S2 and S3 with write access, a push of the claim ref that
   succeeds only if the ref is absent on the remote. Evidence: Anthropic's
   sixteen-agent C-compiler run, coordinated by lock files and git push
   conflicts (Carlini, February 2026).
2. **A lease with a heartbeat, and a fencing re-check at land** (the
   Kubernetes Lease fields; Kleppmann on fencing tokens). The Beads claim has
   no expiry.
3. **The orchestrator's role and run state are a claim plus files in the
   repository**; a successor acquires the expired run lease (Anthropic's
   long-running harness: a progress file, git and a feature list). Claude
   Code's agent teams cannot hand over (a fixed lead, no resume).
4. **A path footprint per claim, with overlaps serialised** (Xu et al.,
   arXiv 2607.04697: a 41.7% conflict rate between different agents'
   concurrent pull requests; AgenticFlict: 27.67%).
5. **Single-threaded writes, parallel reads and review** (Cognition, 2026; the
   MAST taxonomy); contested.
6. **For S3, a claim on the forge**: A bot's issue assignment (as the Rust
   project's rustbot does), or a draft pull request keyed by record id with a
   CI duplicate check; a vouched or pre-accepted-issue policy for outside
   agent contributions.
7. **Every peer channel is untrusted data** (the 2026 incidents named Comment
   and Control, Clinejection and GitLost).
8. **Same-account cross-session messaging** in Claude Code (v2.1.224) is a
   same-uid Unix socket: An accelerator only, unable to cross accounts.
9. **Attribution**: The Linux kernel's `coding-assistants.rst` `Assisted-by`
   trailer already matches abcd's.
10. **Review is the bottleneck** (AIDev: 69% of rejected agent pull requests
    received no feedback).

Rejected: A2A, Beads and Gas Town, shared group directories and ACLs (the
CVE-2022-24765 class), a cross-account socket daemon, a desktop text file,
SLSA, in-toto and Agent Trace, jj, and stacked-pull-request tooling.

Unverified at the time: Whether the forge's rulesets can protect a custom ref
namespace (so S2 claims are advisory among trusted writers).

The shared primitive: A claim {record id, holder, footprint, expiry} taken by
compare-and-swap and re-checked at land; backends a local `update-ref`, a
remote conditional push, or a forge assignment; git is the truth, and local
channels only accelerate.

## Review evidence

Two adversarial reviews of itd-2609091034175565 were held on 2026-10-03:
review 1 (design and feasibility) and review 2 (record discipline). Both
returned NEEDS-WORK. The evidence that moved the design:

- **A hosted cloud session pushes branches only.** The host's documentation
  of its git proxy
  ([cloud environments, "GitHub proxy"](https://code.claude.com/docs/en/cloud-environments))
  says "the proxy rejects branch deletions and pushes of anything other than
  a branch, such as a tag". A probe from a hosted session on 2026-10-03 got
  HTTP 403 pushing a ref under `refs/abcd/`, while branch pushes succeeded;
  the session had no valid forge token, so the REST API's ref creation
  ([Git references](https://docs.github.com/en/rest/git/refs)) was no
  fallback; its git identity was the person's forge no-reply address. A
  claim outside `refs/heads/` therefore cannot include a cloud session, and
  the product thinker ruled the claim a branch (itd-2609091034175565,
  decision 10).
- **A plain push is already the compare-and-swap.** In scratch bare
  repositories (git 2.52.0): A parentless commit pushed over an existing
  branch is rejected as non-fast-forward; of two children pushed on one
  parent, exactly one is admitted; the remote checks the advertised old value
  on every update; and a child of an unexpired claim is accepted, so takeover
  is the client's policy, not the transport's. A plain delete push removed
  another holder's branch, so a release is a tombstone child rather than a
  deletion.
- **Lease flags are unreachable.** abcd's hazard registry refuses every
  `--force-with-lease` push, a host shell guard refused the reviewer's own
  test of one, and the cloud proxy rejects deletions.
- **The committed pre-push hook refuses a claim push today.** Fed a
  parentless claim commit, `.githooks/pre-push` refused it for want of a
  preflight receipt; an exemption by branch name alone would let code ride a
  claim branch past the gate, so the exemption must require the empty tree
  on every pushed commit.
- **A claim branch costs no CI.** Every `on: push` trigger in
  `.github/workflows/` filters on the default branch or release tags, the
  rulesets target only the default branch, and the merge queue never sees a
  claim branch. The forge's limit of 5,000 branches
  ([repository limits](https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits))
  means tombstones need a clean-up verb in time.
- **The claimer sets every time.** `GIT_COMMITTER_DATE` is honoured, so the
  commit date and `expires_at` are both the claimer's: Readers must clamp
  expiry and lapse unreadable claims rather than hold them.
- **The record must stand where the second person can read it.** Review 2
  found the grounds cited only from the gitignored local tier, the decision
  log without the day's rulings, compound acceptance criteria, and "hidden"
  surviving in three records after the branch ruling; it listed the logged
  rulings the day's decisions reverse for the product thinker to confirm.
