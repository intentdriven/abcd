# `/abcd:embargo` — Private Security Advisories, Kept Private Until Published

`/abcd:embargo` keeps a privately reported security advisory private until its
fix ships, then closes or publishes it with the person
(itd-2610112239022766, spc-2610120557432880). Its trust rule is
[adr-2610120558328615](../../decisions/adrs/2610120558328615-a-private-advisory-s-record-never-enters.md)
and brief invariant 22
([`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)): a
private advisory's record never enters the public ledger, its store is never
committed, nothing public carries an advisory id or a hint of one, and closing
or publishing an advisory takes one approval per act, bound to exactly what was
shown.

This chapter describes intake, triage, recording a fix, the bare listing of
what is ready to publish, queuing its publish step while the person is away,
closing, the outbound check and publication. `embargo` runs no fix lane, lands no fix, requests no CVE,
restores no redacted record and has no step for an embargo broken before
publication; the person handles each of those, and the section below lists the
ones the command page hands them.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `intake` | — | shipped |
| `triage` | — | shipped |
| `fixed` | — | shipped |
| `close` | — | shipped |
| `check` | gate | shipped |
| `publish` | — | shipped |

## The embargo store

The private records live in a worktree of the repository in the machine's
worktree store, `~/.abcd.noindex/worktrees/<root-sha>/embargo/`
(adr-2610031751065746), created detached at the default branch's tip so
nothing in it has a ref to push. Records sit in the store's own ledger, where
the capture and resolve code reads them, under a neutral slug; the link from an
advisory to its record lives in the store's state file in its local tier,
never in the record. Every sub-verb verifies the store first and refuses,
writing nothing, when its HEAD is on a branch, its index tracks a record the
state names, it resolves inside the checkout, or its state file is not this
account's alone. `abcd peers` skips the store, counting it without its path.

## What each sub-verb does

- **intake** lists the advisories GitHub holds in triage and draft through the
  person's own `gh` and mints one private record per advisory not seen before;
  severity maps critical to critical, high to major, medium and low to minor,
  and an advisory with no severity is listed for the person, not recorded.
  Each held advisory GitHub lists in neither state is read on its own. One
  GitHub reports published, by any hand, is recorded so, which releases it
  from the outbound check; one it reports closed is recorded so only when the
  store's verdict on it already ends the private track (a closing verdict, or
  the public track), since a close recorded on GitHub's
  word alone would drop an advisory judged a vulnerability, or never judged,
  from every plan for good. Every timestamp is GitHub's own: its answer is held
  to a known state and a well-formed time, present. An answer that fails that,
  or a close the store's verdict does not make, leaves the advisory under
  embargo, read again on the next run so a reopen is seen, and listed with
  the reason; the second is the person's to settle. So does an advisory
  GitHub reports no such advisory for, with a 404 or a 403: it may be deleted,
  or hidden from this account, and one such answer is that advisory's, not the
  run's. A missing `gh`, a `gh` signed in to no account or holding a token
  GitHub rejects, or a refusal whose words name the advisory scope, refuses the
  run whole with its remedy, recording nothing. Every publication plan reads
  the held advisories the same way, and lists what it recorded. A plan whose
  every read meets a 404 or a 403 cannot tell deleted advisories from a token
  without the scope, so it asks once for the repository's advisory listing,
  which answers only a token holding the scope: answered, the plan goes on
  with those advisories unread; refused, or failing otherwise, the plan
  refuses whole.
- **triage** records one of four verdicts (vulnerability, not abcd's, not a
  vulnerability, public track) with who decided it and when; the two closing
  verdicts draft the closing explanation, and "not abcd's" names both halves of
  [abcd-fixes-what-it-adds](../../principles/abcd-fixes-what-it-adds.md).
- **fixed** records the commits that fix an advisory judged a vulnerability
  and neither closed nor published, in the store's state and nowhere else, so
  the link from a fix to its advisory never reaches the public ledger or the
  checkout. Each commit is named by its id, abbreviated or full, and kept by
  its full id once git resolves it to a single commit of this repository; a
  ref name or anything shaped like an option is refused before git sees it.
  A commit already recorded is not recorded twice, and a replace switch swaps
  the list. Nothing else links a cut release to the advisories it fixes, so
  without it publishing each one after a release is a match a person makes by
  hand.
- **bare** `abcd embargo` lists each advisory judged a vulnerability and
  neither closed nor published as ready to publish, waiting, or with no fix
  recorded. It is ready when a published release holds every fix commit
  recorded for it (or, with none recorded, the fix commit its private record,
  resolved in the store, names), and the listing names the first such release
  and the publish command that shows its plan. Which tags hold a commit is
  read from the commits as stored, and a tag counts only as a published
  release, as publish holds; origin and GitHub are asked only about a tag that
  holds a fix, and an answer that cannot be had leaves the advisory waiting
  with the reason. The listing writes nothing and publishes nothing, while the
  person is away as while they are present: a bare verb renders, and the act
  that changes state has its own sub-verb.
- **queue** is that act. While the person is away it queues one step for
  their return for each ready advisory not queued already, naming the private
  record and the release and never the advisory, so the publish waits for
  them rather than for someone to remember it. The check for a step already
  queued and the append are one write under the away store's lock
  (`autonomy.QueueStepOnce`), so two runs at once queue it once; while the
  person is present it refuses and queues nothing.
- **close** shows the closing text and its digest and sends only a read; with
  the approval of that digest it closes the advisory with the text at the head
  of its description (GitHub offers no comment on an advisory), the reporter's
  text kept below a separator, and reads it back.
- **check** is the outbound check: over a commit range (messages and added
  lines, a merge's being those it adds against every parent) and texts, it
  refuses an unpublished advisory's id, its reserved CVE id, its title (12
  characters or more, unless the advisory is closed or its verdict is the
  public track) and the store's location on this machine, naming where and
  never repeating the match. It refuses as well any other advisory id that
  the allow switch (repeatable, another project's advisory id, already public)
  does not name and GitHub does not list among this repository's published
  advisories. GitHub is asked only when such an id is found, an answer that
  cannot be had leaves each refused, and an id of the placeholder form, each
  four-character group one character repeated (`GHSA-xxxx-xxxx-xxxx`), is not
  refused as unlisted. A range's commits, and the history publish reads, are
  read as git stores them, with no replace ref, graft file or commit-graph
  standing in. A paraphrase or a count is beyond it and stays the author's to
  avoid.
- **publish** shows the plan with each patched version taken from the first
  published release containing the fix, leaving out, with the reason, an
  advisory not resolved or released, one with no package, and one whose text would disclose
  a secret, a private name or another advisory under embargo; with the approval
  of that plan's digest it publishes one advisory as a canary, reads it back,
  publishes the rest, and confirms none is left in draft. A release tag counts
  as a published release only when origin holds it at the object this
  checkout holds and GitHub reports its release neither a draft nor a
  prerelease: the release workflow drafts a release before it publishes it,
  and a failed run leaves a tag with no release behind it, so a tag that only
  this checkout holds, or one still drafting, would name a patched version
  nobody can install. Such a tag is passed over for the next published
  release, the plan's own release must be one, and origin or GitHub that
  cannot be read refuses the plan rather than falling back on the checkout's
  tags. Origin's tags vouch only while origin names the repository the store
  was made for, so a repointed origin refuses the plan, as it refuses intake. The fix commit comes
  from the advisory's record resolved in the release, or, since the security
  rule keeps that record out of every committed tree until the advisory is
  published, from the fix commits recorded for it, or else from its private
  record resolved in the store's own ledger; recorded commits give the
  patched version of the first published release holding every one. The
  store is a trust boundary: the plan takes a commit from it only when it is
  the full object name of a commit of this repository, as git resolves it, and
  leaves the advisory out otherwise. The command that copies a private
  record, resolved, into the public ledger is read from the store's state,
  not from the run that published: every plan, intake and publication names
  it for each advisory recorded published whose record is resolved in the
  store, passes the same checks, and is not yet in the checkout's ledger, so
  a stopped run loses nothing. A record whose path the terminal would not
  show as written names no command, saying why. Before the record lands, the
  person compares its body, minted from the reporter's report, with the
  published advisory.

## What stays the person's

Every verdict, every approval, fixing and landing the fixes, recording each
fix or resolving each private record in the store with it, publishing an
advisory the bare listing names ready, requesting a CVE, restoring each
record into the public ledger after publication, and putting back any record
details redacted to land a fix, once the advisory is published. The command
page, [`commands/embargo.md`](../../../../commands/embargo.md), says so to the
host and to the person.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd embargo`

Sub-verbs: `abcd embargo check`, `abcd embargo close`, `abcd embargo fixed`, `abcd embargo intake`, `abcd embargo publish`, `abcd embargo queue`, `abcd embargo triage`.

Flags: none.

### `abcd embargo check`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--allow` | stringArray |
| `--file` | stringArray |
| `--range` | string |

### `abcd embargo close`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | string |

### `abcd embargo fixed`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--replace` | bool |

### `abcd embargo intake`

Sub-verbs: none.

Flags: none.

### `abcd embargo publish`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | string |
| `--release` | string |

### `abcd embargo queue`

Sub-verbs: none.

Flags: none.

### `abcd embargo triage`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--by` | string |
| `--cannot-help` | string |
| `--exists-without` | string |
| `--reason` | string |

<!-- surface-appendix:end -->
