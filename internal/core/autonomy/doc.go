// Package autonomy is the away mode: the person's word that they are away and
// sessions should work on, held once per repository on the machine, and the
// shapes the stop check reads (itd-2610180912189125, spc-2610180914469489).
//
// It is a library: it returns values and errors, and never prints. The front
// doors (`abcd mode autonomous|back|queue`, and later the hooks) format what it
// returns.
//
// # The away store
//
// One folder per repository, keyed on the repository's root commit as the
// worktree store is, below abcd's home: `away/<root-sha>/` (Store.Dir,
// Store.Rel). Every working copy of the repository reads the same folder. It
// holds:
//
//   - record.json (RecordFile, Store.RecordPath): the Record;
//   - lock (LockFile, Store.LockPath): the one lock every read-modify-write in
//     the folder holds, the record's and any sibling file's alike;
//   - sibling files other parts of the slice add (a working copy's created
//     branches, a session's start copy), read and written through
//     Store.ReadFile and Store.WriteFile inside Store.WithLock.
//
// Every file in it is used only when it is a regular file this account owns
// that no one else can write, within MaxFileBytes (64 KiB), and only through
// folders no one else can change: the checks a user-layer rules.json gets.
// A file failing them is ErrUnsafe, naming the file. Writes go through
// fsutil's atomic writer.
//
// # The lock
//
// Store.WithLock holds fsutil.WithFileLock on the lock file for at most
// LockTimeout (two seconds); past it the call returns ErrLockBusy. Store.Update
// re-reads the record inside the lock, hands it to the caller's function, and
// writes what that returns, so two writers never lose an update and a check
// that read the record before a slow forge listing never writes away back
// after `abcd mode back` (Store.RecordStop records only while the re-read
// record still holds).
//
// # Failing open
//
// Store.Away is the hooks' read: an unreadable or unsafe record (or lock)
// reads as not away, with the reason returned for a stderr line. A verb asked
// to change such a record refuses, naming the file, except Back: ending the
// mode never depends on the record (end.go). A record past its bound is
// printed whole and kept fitted (Record.Fit); one that does not parse, names
// another schema or is not safely this account's is replaced by a closed
// record, printing nothing from it; parts of the wrong shape are left out;
// and a lock not had within LockTimeout, or not safe to take, is gone
// around, the atomic write re-checked. BackResult.Notes names each.
//
// # Record
//
// Record is the away record. Its fields, as written to record.json:
//
//	schema_version  1
//	set_by          who set it (Who.Name): the holder name reservations
//	                resolve without a network read, else git's user.name
//	set_from        the working copy it was set from (Who.Copy, its root)
//	start, until    when it was set and when it ends
//	open_ended      set without --until (until is then start plus MaxWindow)
//	hold_pushes     the person said pushes wait (--hold-pushes); it narrows
//	                and grants nothing
//	queue           []QueuedStep: steps for the person (`abcd mode queue`)
//	handed_back     []HandBack: items the stop check handed back, by branch
//	                and head, once each
//	stops           []Stop: the stops recorded, a stop identical to one
//	                already recorded, anywhere, kept as that entry with a
//	                count and the last time
//	dropped_*       what was let go to stay within MaxFileBytes, counted:
//	                the oldest stops and hand-backs past their own bounds,
//	                and queued steps only when SetAway carries, or Back
//	                closes, a record past its bound; a queue past its bound
//	                refuses the next step (ErrQueueFull) instead
//	closed          *Closing: once ended by word, who ended it and from
//	                which copy
//
// A record holds (Record.Holds) while it is not closed and now is before
// Record.Ends, which is never more than MaxWindow (seven days) after its
// start, whatever is stored. Past its end and not yet closed it is lapsed
// (Record.Lapsed). A human prompt never ends it.
//
// # Owed
//
// Owed is the stop check's input: the owed items, each an OwedItem with its
// OwedKind, whether it is capped (an item the session can act on itself, which
// a wake releases at most twice in a row), and its branch or record id (plus
// its pull request and head where it has them). While away, a capped item on
// a branch carries Live, the live work standing for it, when the preflight's
// whole test run of this account (`make preflight` at the load gate or
// running past it, whose waiter records itself: implement.WholeRunMarker) has
// a marker that says its command is the preflight's stages and names the
// item's head as the commit it was written at, runs in the worktree the
// item's branch is checked out in, and no receipt names that head
// (livework.go); Owed.CappedAllLive reports that every capped item has it. A
// run started before a commit does not count for it. A checkout whose abcd
// predates the marker's present shape writes none that counts. A process of
// this account can write a marker and run under abcd's name, so the proof
// holds against other accounts and stale state, not against this account's
// own agent: that a counted marker is a real preflight is a must nothing
// enforces. A build run's lane agent and a host's background agent abcd did
// not start are not live work, and must not be counted as such until
// something proves them; nothing enforces that beyond the reader reading
// neither. The kinds, in the order the
// next item is chosen (owedKinds): a renewal due, a failing pull request, a
// behind one, a stacked branch, an unpushed branch, and a reservation whose
// work no pull request carries. Owed.Fingerprint and Owed.CappedFingerprint
// compare two owed sets item by item, ignoring order and reasons.
// Owed.NextRenewal is the earliest renewal due among the live reservations
// under this copy's key (zero when none), Owed.WakeBy the latest a wake may
// fire for them (a renewal not yet due, or the lapse of one already due, so
// a renewal only due keeps releasing on a wake), and Owed.HandedBack the items a
// queued step handed back, which the verdict accepts only under
// --hold-pushes and only for an unpushed branch's item with its head.
//
// # The stop verdict
//
// Judge(record, owed, input, session, now) is the stop check's verdict, pure:
// present, it allows (with a SystemMessage naming owed work); away, the seven
// rows of the plan decide, from the owed set, the host's payload (StopInput:
// stop_hook_active, the last message, session_crons, background_tasks, the
// host's cap) and the session's SessionState. A set whose every capped item
// has live work is waiting, not stuck: a qualifying wake releases it however
// many wakes have passed, and the wake count does not advance, for
// LiveWorkBound (one preflight's bound, 18 hours) from the first check that
// found it (SessionState.WaitingSince, reset with the capped set, so the
// bound does not hold against a set that keeps changing, as the wake count
// does not); past it the two-wake rule holds again, and its block says the
// wait on the named branch is past the bound. Without a wake, the block names
// the branch, the live work and since when. A wake qualifies
// (qualifyingWake) when a session cron's latest fire time (latestFire: its
// next match, by abcd's own five-field cron parser, parseCron, plus the
// host's lateness) is no later than wakeDeadline, which is never sooner than
// now plus MinWakeLead. The session's state lives in
// the copy's local tier (SessionsDir, LoadSession, SaveSession), keyed by a
// session id ValidSessionID accepts. The session-start hook notes each
// session's start copy in the store (Store.NoteSessionCopy); ResolveCopy
// reads it back.
//
// # The created-branch list
//
// The guard's pre-tool hook records the branches a session creates
// (spc-2610180914469489 step 3) through RecordCreated(copyRoot, runDir,
// changes, Options), whether or not the person is away: a BranchCreated name
// only when no branch of that name exists yet, a BranchCheckedOut one (`git
// worktree add <path> [<branch>]`) when it is a branch no worktree has
// checked out (git refuses that command, which runs after the recorder) or
// names nothing git resolves, and a BranchMoved rename carrying a record by
// whole name, its creator with it. Each name added keeps the session that
// created it (Options.SessionID), when the hook had one. The list
// is createdFile(copyKey) in the away store, keyed on the working copy's
// reservation copy key (claim.CopyKey), written under the store's lock
// (Store.RecordBranches) and capped at MaxCreatedBranches, the oldest
// dropped; Store.CreatedBranches reads it. A checkout abcd does not manage
// (Managed: no `.abcd/` at its root, nor at its main worktree's) records
// nothing, and every away hook returns at once there.
//
// # Reading the owed work
//
// ReadOwed is the stop check's one seam onto owed work:
//
//	func ReadOwed(ctx context.Context, o ReadOptions) (Owed, error)
//
// o.Root is the session's working copy and o.Record the away record as the
// check read it. While o.Record does not hold at o.Now the person is present
// and ReadOwed reads disk only: renewals due and unpushed own branches, with
// no forge call. While it holds, ReadOwed adds one `gh pr list` of the open
// pull requests (o.Forge, landing.GHContext by default) under ForgeDeadline
// (ten seconds, o.ForgeDeadline in a test), for failing, behind (the forge's
// merge state: BEHIND, and DIRTY as behind; UNKNOWN is a note), stacked and
// uncarried work. A listing that fails or is late is a note, and the check
// judges from disk alone: a pushed branch is compared with the last-fetched
// default branch, naming the fetch, and no reservation is reported uncarried
// (adr-2610181003541997).
//
// A pull request into the default branch that is BEHIND, and not a draft, is
// judged by the merge queue where one guards that branch
// (iss-2610311339274372, dcn-2611081501502395): the queue tests every entry
// against the latest base, so merging the base in would only move the head
// and void its receipts. The forge's live rules for the branch are read once
// (`gh api repos/{owner}/{repo}/rules/branches/<branch>`, never the in-tree
// mirror under .abcd/work/rulesets/), and where they declare a merge queue a
// pull request armed, or already queued (claim.ArmedPullRequestsQuery, read
// once and only for one the listing shows unarmed), is not owed; one nobody
// armed is owed with "arm it" as its next move. Each read runs under
// RulesetDeadline (three seconds, o.RulesetDeadline in a test); one that
// fails, is late or cannot be read whole is a note, and the pull request is
// owed its merge as before, unless the check's own deadline ended it: then
// it owes nothing on that account and the reading is marked cut short. A
// conflict (DIRTY), failing checks and a stacked base are owed as they were.
//
// A branch is the session's own when this copy's HEAD reflog has checked it
// out or the copy's created-branch list holds it, either only while it holds
// a commit of this checkout's author email made within StaleWindow (14 days)
// or a live reservation under this copy's key links it; when it is a lane of
// an unfinished run in this copy's local tier whose reservation is live under
// this copy's key; or when it heads an open pull request of the person's that
// a live reservation under this copy's key links. Never the default branch,
// nor a branch whose open pull request another person authored. A pull
// request carries a reservation's work when the record id is in its head
// branch's name or on a Resolves: or Delivers: line of its body, on word
// boundaries; a Refs: line carries it only for a head already own.
//
// Several sessions can work from one copy, or from the worktrees beside it.
// With o.SessionID set, what another live session holds is left out of the
// owed work and named once in a note, holder by holder (holders.go): the
// lanes of a run whose control record names that session as its driver, a
// branch the guard recorded it creating (CreatedBranch.Session, which
// RecordCreated keeps from Options.SessionID), a branch checked out in the
// worktree it started in (its occupancy marker), and a reservation only its
// runs are for. The session's own evidence wins over another's: a branch it
// recorded creating, or a lane of a run it drives, stays its own. An item
// whose holder cannot be told stays owed: a record with no session kept, a
// lane of a run no session drives (unless a live session started in the
// worktree it is checked out in), a reservation made by hand, a holder that
// has ended (its work returns to the sessions left), and every item when the
// session has no id or the live sessions (ReadOptions.LiveSessions,
// occupancy.Live by default) cannot be read. A branch created again under a
// name the guard recorded is its new creator's. A branch a peer holds still
// serves as the base of the session's own branch stacked on it. Whenever owed
// work remains, a copy another live session started in is named, once, as a
// breach of one session per checkout.
//
// A stacked branch (its pull request's base is not the default branch, or,
// with none, its nearest own ancestor branch is not in the default branch) is
// owed by the clearing rule alone: while it is behind its base, failing, has a
// review outstanding (CHANGES_REQUESTED, or a run lane with no passing review
// of its head) or is not verified (a preflight receipt naming its head where
// the repository mints receipts, else passing checks); an item whose only
// unmet condition is checks still running is not capped. With --hold-pushes,
// an unpushed own branch a queued step names as a whole name, whose head
// contains its base's last-fetched tip and has a receipt, is handed back:
// moved out of Owed.Items into Owed.HandedBack. ReadOwed records nothing:
// the verdict accepts the hand-back (Verdict.HandBacks) and the stop hook
// records it, once per branch and head (Store.RecordHandBack), so a
// hand-back has one owner. Owed.NextRenewal is filled present or away. The
// items come sorted by owedKinds' order. The
// error is a root that is not a working copy; anything else ReadOwed could
// not read is a note in Owed.Notes.
//
// These reads of ReadOwed are bound to ctx: finding the checkout, the
// author's identity and the reservations' holder, the default branch and
// the local branches, the fetch stamp, the receipt folder and the landing
// receipt, the runs' tracked check, the created-branch list's store key and
// this copy's key, the other worktrees, each branch's own history, reflog,
// ancestry and landing, the reservations as last fetched (claim.ViewContext),
// the forge reads and the live sessions' wait. The stack-base search reads
// each own branch's reflog once a check and walks each branch's own history
// once, so its cost grows with the branches, not their pairs. A context
// that ends before everything is read stops the reading where it is:
// Owed.Incomplete is set, a note says what was not read, and Judge allows
// the stop with a line naming it, recording a StopCheckIncomplete while
// away. These reads mark the reading cut short, which sets Owed.Incomplete,
// when the context ended them: the author's identity, the holder, the
// default branch, the receipt folder, the landing receipt, the runs' tracked
// check, this copy's key and the created-branch list's store, the other
// worktrees, the fetch stamp, the stack index, the reservations as last
// fetched, the live sessions' wait, the merge queue's rules and its armed
// set, and each call made through `r.git`. A reservations read the context
// ended uses none of what it read: no reservation reads as live from it, and
// none counts as this copy's. The pull-request listing and the landing
// check's merge-driver, merge-tree and cherry reads are marked when the
// reading runs out. Four of the reads owe nothing on their own account once
// the context has ended: no preflight for a receipt it could not read, no
// merge for an ancestry it could not ask, and no merge or arming for a merge
// queue's rules or armed set it could not read. A branch with no upstream
// and no commit in any remote branch is owed only when merging it into the
// last-fetched default branch would change it; one squash- or rebase-merged
// already is not, and one that cannot be told is a note. A listing at its
// limit of 200 may be truncated: a note says so, and no reservation is
// reported uncarried from it.
package autonomy
