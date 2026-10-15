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

- `abcd announce` fetches the default branch and the sender's own status branch, then pushes.
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

  The prompt hook renders each unshown, unexpired announcement once. Rendering takes only the validated fields, through `termsafe` (invariant 13), so the commit's raw text is never shown. Each rendered announcement ends with the line that it informs and grants nothing. An announcement's life is counted from its first fetch, not from the sender's clock. The hook shows at most ten lines, the newest per item, then a count and the command that lists the rest.
- **C7, C8:**
  - **The subject:** the reservation subject grammar gains a release subject (`rel-vX.Y.Z`). It needs no footprint, sits outside the live cap, and ends when the tag is on the remote or the holder releases it.
  - **C7:** the land fence (`internal/core/implement/loop/reservation.go`, `landFence`) refuses while any other holder's release reservation is live, naming the holder and the release.
  - **C8:** `abcd status` and the prompt hook show the hold line.
  - **The issue:** this resolves iss-2609091037191879.
- **C10:** a failed fetch writes its time and reason to the local tier. The hook then shows "status unread since <time>: <reason>". An empty list is shown only after a fetch that succeeded. A checkout with no `origin` shows nothing.
- **C11:** any of these is shown marked as doubtful, with the reason, and every other branch is still read:
  - a commit whose sender trailer is not the branch's owner;
  - a non-fast-forward tip relative to the last read;
  - a branch name that is not a lower-case username.
- **C12:** the hook reads only the local tier. A test with a stalled remote proves it returns within one second.
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
