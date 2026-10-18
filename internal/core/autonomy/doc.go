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
// to change such a record refuses, naming the file.
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
//	stops           []Stop: the stops recorded, a stop identical to the last
//	                kept as that entry with a count and the last time
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
// its pull request and head where it has them). The kinds, in the order the
// next item is chosen (OwedKinds): a renewal due, a failing pull request, a
// behind one, a stacked branch, an unpushed branch, and a reservation whose
// work no pull request carries. Owed.Fingerprint and Owed.CappedFingerprint
// compare two owed sets item by item, ignoring order and reasons.
// Owed.NextRenewal is the earliest renewal due among the live reservations
// under this copy's key (zero when none), and Owed.HandedBack the items a
// queued step handed back, which the verdict accepts only under
// --hold-pushes and only for an unpushed branch's item with its head.
//
// # The stop verdict
//
// Judge(record, owed, input, session, now) is the stop check's verdict, pure:
// present, it allows (with a SystemMessage naming owed work); away, the seven
// rows of the plan decide, from the owed set, the host's payload (StopInput:
// stop_hook_active, the last message, session_crons, background_tasks, the
// host's cap) and the session's SessionState. A wake qualifies
// (QualifyingWake) when a session cron's latest fire time (LatestFire: its
// next match, by abcd's own five-field cron parser, ParseCron, plus the
// host's lateness) is no later than Deadline. The session's state lives in
// the copy's local tier (SessionsDir, LoadSession, SaveSession), keyed by a
// session id ValidSessionID accepts. The session-start hook notes each
// session's start copy in the store (Store.NoteSessionCopy); ResolveCopy
// reads it back.
package autonomy
