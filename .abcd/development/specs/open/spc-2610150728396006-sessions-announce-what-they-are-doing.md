---
id: spc-2610150728396006
slug: sessions-announce-what-they-are-doing
intent: itd-2609151838312703
origin: researcher-authored
production_mode: hand-written
---
# sessions-announce-what-they-are-doing

## Summary

This spec delivers [itd-2609151838312703](../../intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md). Sessions under any account, on any machine or in a hosted cloud session post short fixed-form announcements on the shared remote. A person reads everyone's status with one command. A release cut holds every merge through a reservation that the land step reads.

**Precondition: the cloud probe.** Before step 1 starts, the product thinker runs the prompt the planning session wrote (decision 8). It records two things:

- whether a hosted session's push to a branch outside `claude/` is accepted;
- which forge identity the push arrives as.

A refused push sends the branch prefix back to the product thinker. An app identity means cloud announcements show as claimed (C17); nothing else changes.

**Approach.** One canonical primitive: announcements reuse the reservation code in `internal/core/claim` rather than copying it ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)). The design review of 2026-10-15 found that seven parts of that code must generalise:

- `fetch` takes a namespace instead of the fixed reservation refspec, and keeps a fetched-at stamp per namespace.
- `parseMessage` splits into one trailer collector plus a validator per family. The announcement item grammar is a record id or a pull-request number.
- `parseTip` takes the parser.
- `writeCommit` takes a message interface.
- `checkMessage` loses its reservation wording.
- `pushArgs` and `refName` take a full ref.
- A bounded history walk reads the commits added since this checkout last read a branch.

The reservation's `lostRace` refusal does not carry over: an announcement's lost race is rebuilt on the new tip and retried (C2). Its `terminalOnDefault` check is the wrong way round for C4. C4 instead asks whether the record is present on the default branch as last fetched, through `peers.FoldersAt`, and refuses when no default branch was fetched.

**Shape on the remote.** Each person has one branch, `abcd-status/<forge username in lower case>` (decisions 9, 10 and 20). Each announcement is one empty-tree commit carrying validated trailers:

- the kind, one of `merging`, `merged`, `asking`, `answered`, `handed-over`;
- the item;
- the sender's username;
- the working copy;
- the time sent;
- the expiry, at most 24 hours.

Nothing else is carried. The commit is authored as the person, as reservations are.

**When abcd touches the network.** Decision 14 and invariant 7 settle this:

- `abcd announce` fetches the default branch and every status branch, the whole `abcd-status/` namespace, then pushes.
  - **Amended 2026-10-16 in the step 2 review:** this line said the sender's own status branch. A fetch naming one branch fails while origin holds none, which is the case at a person's first announcement, so the whole namespace is fetched, as invariant 7 states.
- `abcd status` fetches every status branch, the reservations and the default branch, then reads the forge's activity record for each new update (C17).
- The land step fetches the reservations before it judges a release hold (C7).
- The prompt hook only renders the last completed fetch from disk, with its age (C12), and never touches the network.

**How each criterion is met:**

- **C1, C2:** `claim.Announce` writes the commit through the generalised writer and pushes with one plain push. A rejected push is rebuilt on the fetched tip and pushed again, up to three times; after that the sender sees "not sent" and the reason. Nothing is ever force-pushed.
- **C3:** the validator refuses an unknown kind, a malformed item, any text outside the trailers, and an expiry over 24 hours. Each refusal names the field and the limit, and nothing is pushed.
- **C4, C5:**
  - **C4:** a record absent from the default branch as last fetched is refused before the push.
  - **C5:** a record the embargo store holds is refused through `abcd embargo check`. A store that exists but cannot be checked also refuses.
  - **Dependency:** C5 waits for the embargo work to reach the default branch (scope condition).
- **C6, C9, C18:** the reader keeps per-checkout state in the local tier:
  - which announcements this checkout has shown, keyed by commit;
  - when each was first fetched.

  The prompt hook renders each unshown, unexpired announcement once. Rendering takes only the validated fields, through `termsafe` (invariant 13), so the commit's raw text is never shown. Each rendered announcement ends with the line that it informs and grants nothing. An announcement's life is counted from its first fetch, not from the sender's clock. The hook shows at most ten lines, the newest per item, then one more line with a count and the command that lists the rest.
- **C7, C8:**
  - **The subject:** the reservation subject grammar gains a release subject (`rel-vX.Y.Z`). It needs no footprint, sits outside the live cap, and ends when the tag is on the remote or the holder releases it.
  - **C7:** the land fence (`internal/core/implement/loop/reservation.go`, `landFence`) refuses while any other holder's release reservation is live, naming the holder and the release.
  - **C8:** `abcd status` and the prompt hook show the hold line.
  - **The issue:** this resolves iss-2609091037191879.
- **C10:** a failed fetch writes its time and reason to the local tier. The hook then shows "status unread since <time>: <reason>". An empty list is shown only after a fetch that succeeded. A checkout with no `origin` shows nothing.
  - **Decided 2026-10-16 in step 3's build (build note 19):** every fetch keeps its failure per namespace, and clears it when that namespace's fetch next succeeds. The hook says "status unread since 09:12 UTC" and "reservations unread since 09:12 UTC" each on its own, the time being that namespace's last successful fetch, in UTC. A namespace never read here says "never read here", with the time the fetch failed. Refined in step 3's fix round 2: the hook gives the reason in fixed words, "the last fetch failed; `abcd status` says why", since git's text can carry a remote's words; only `abcd status` shows git's reason.
- **C11:** each of these is shown, and every other branch is still read:
  - a commit whose sender trailer is not the branch's owner, marked doubtful with the reason;
  - a non-fast-forward tip relative to the last read, marked doubtful with the reason;
  - a branch name that is not a lower-case username, counted and never named (amended in step 3's fix rounds: whoever pushed the branch chose the name, so no reader shows it or keeps it).
  - **Decided 2026-10-16 in step 3's build (build notes 12, 24 and 25):** a commit that does not read as an announcement, one whose tree is not the empty tree among them, is shown as doubtful by its branch and the reason, and none of its text is shown. A walk from a mark that returns More, Merged or Rewritten marks the branch "continuity not verified", and the announcements it added are still read; so does a first read, with no mark, that returns More (decided 2026-10-16 in step 3's fix round, dcn-2610161100416718), since a first read stopped at the walk's limit could bury a live announcement under ended ones, whatever times its commits carry: within the bounded walk no signal settles whether a line older than what the read took in can still stand (decided in step 3's fix round 3, dcn-2610161737571276, refining dcn-2610161320457650, whose narrowing to reads whose oldest commit was recent it replaces); the shown-set keeps a repeat from showing twice. `abcd status` lists the status branches on origin first and fetches only the names that are lower-case usernames, each by its own refspec, so a case-colliding twin never breaks the fetch; every other name is counted, never named and never kept: the listing keeps how many there were and a digest of them, and a reader that cannot read that count back says it is unknown, never zero (step 3's fix round 2).
- **C9, a first read's history:** an announcement whose own expiry passed more than an hour before this checkout first fetched it is history a first read brought in, not news, and is not counted anew from that fetch. A sender's clock up to an hour slow is still counted from the first fetch (decided 2026-10-16 in step 3's build).
- **C12:** the hook reads only the local tier. A test with a stalled remote proves it returns within one second. One intake takes in at most 100 new commits across every status branch, the newest of each branch first, and a branch with more is shown as "continuity not verified"; what an intake keeps (each commit read, each doubt, each branch's read mark) is written without a flush to stable storage, since its loss costs only a read made again. So the first intake of a flood fits the hook's deadline with flushing on (decided 2026-10-16 in step 3's fix round 3).
  - **Decided 2026-10-16 in step 3's build (build notes 21 to 23):** the status block is read under a 600 ms deadline inside the hook's one-second budget. The walk and the batch read of new commits run under that context, and the read runs beside a timer, so a slow read never holds the prompt. When the deadline passes first, or the read fails, the block is one line, "status unavailable", with the reason, never nothing. Refined in step 3's fix round 2: the reason is in fixed words, one for a read that ran out of time and one for a read that failed, and git's own text goes to stderr only; the hook sets its one-second deadline when it starts, and the status read takes what the drain, the rules and the memory notes leave of it, less 150 ms for writing and marking the block, at most 600 ms; the render, the read of what was shown, the prune and the marking all run inside that deadline. The block is appended to the injected text after the rules and the memory notes, before the session title the session-names change sets, so both emit paths carry it.
  - **Decided the same day: the block joins the rules loader's error path.** A `rules.json` that will not load returns early, and the status block is emitted there too: in plain form as the whole output, and in the `--json` envelope as `text` beside `error`. The two subsystems share the hook, and neither may switch the other off, as the transcript drain already holds. When session names lands, that path's title, when one is owed, carries the block as its additional context.
  - **Decided the same day: where the block is rendered.** The render lives with the reader in `internal/core/claim` (`PromptNotes`), not in `internal/core/rules`. The rules package knows nothing of the status branches, and the hook joins the two texts at its one emit point. The render and stalled-remote tests are in `internal/core/claim` and in the hook's surface test.
- **C13, C14:** `.githooks/pre-push` gains a second receipt-less row, `abcd-status/`. It reuses `reservation_shape`: a fast-forward of commits that change no file. Every other push under the prefix is gated as any branch is.
- **C15:**
  - **The ruleset:** `.abcd/work/rulesets/status-protection.json` mirrors a ruleset on `refs/heads/abcd-status/**` that blocks deletion and non-fast-forward pushes with no bypass. It is applied through the rulesets' own apply verb (invariant 10).
  - **The ADR:** [adr-2610150729333337](../../decisions/adrs/2610150729333337-a-status-branch-moves-without-a.md) is the trust rule.
  - **The brief:** invariants 7 and 10 gain clauses naming `abcd announce`, `abcd status` and the status branch, written in the change that builds them.
- **C16, C17:**
  - **C16:** `abcd status` lists announcements newest first, with sender, kind, item, time and time left. Beside them it lists who holds which reservation, and any release being cut. `--json` gives the same.
  - **C17:** each new update's pusher is read from the forge's activity record (`GET /repos/{owner}/{repo}/activity?ref=`). The sender shows as verified when the record names the branch's owner, doubtful when it names someone else, and claimed when the forge cannot say. Claimed is the native default; the forge lookup is the adapter (invariant 2).

**Surfaces.** `abcd announce` joins the agents' command set beside `abcd reserve`. `abcd status` joins the people's set. Each has a plugin command page, so it is reachable from both the CLI and the plugin markdown surface.

**Out of scope here.** These are captured, not built:

- `abcd peers` keeps its name and its behaviour (iss-2610150726331622);
- the status board is renamed the dashboard (iss-2610150726321422).

## Footprint

- packages: internal/core/claim, internal/core/implement/loop, internal/core/rules, internal/surface/cli, .githooks, .abcd/work/rulesets, commands
- tests: internal/core/claim (announce, read, expiry, flood, doubtful-branch and race tests), internal/core/implement/loop (release land-fence test), internal/core/rules (prompt-hook render and stalled-remote test), the pre-push hook's receipt-less shape test, smoke (announce and status end to end against a local bare remote)

## Steps

1. Generalise the reservation primitive (namespace fetch, trailer collector with per-family validators, message interface, full-ref push, bounded history walk), with no change in reservation behaviour
   - packages: internal/core/claim
   - tests: internal/core/claim (every existing reservation test unchanged and passing)
2. Announcements and `abcd announce` (C1 to C5, C13, C14), the receipt-less row, the ruleset mirror and the ADR, with the invariant 7 and 10 clauses
   - packages: internal/core/claim, internal/surface/cli, .githooks, .abcd/work/rulesets, commands
   - tests: internal/core/claim (announce, validation and race tests), the pre-push shape test
3. Reading: `abcd status` and the prompt hook (C6, C9 to C12, C16 to C18)
   - packages: internal/core/claim, internal/core/rules, internal/surface/cli, commands
   - tests: internal/core/claim (read, expiry, flood, doubtful-branch tests), internal/core/rules (render and stalled-remote tests), smoke
4. The release hold (C7, C8)
   - packages: internal/core/claim, internal/core/implement/loop
   - tests: internal/core/claim (release subject tests), internal/core/implement/loop (land-fence test)
