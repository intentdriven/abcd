---
name: embargo
description: "Keep private advisories off the public ledger, naming those ready to publish: Writes nothing bare, only the embargo store; refuses an unknown sub-verb."
argument-hint: "[intake | triage <iss-N|GHSA-id> <verdict> --by <who> [--reason <why>] [--exists-without <why> --cannot-help <why>] | fixed <iss-N|GHSA-id> <commit>... [--replace] | queue | close <iss-N|GHSA-id> [--approve <digest>] | check [--range <a>..<b>] [--file <path>|-]... [--allow <GHSA-id>]... | publish [--release <vX.Y.Z>] [--approve <digest>]]"
block: people
---

# `/abcd:embargo` private security advisories

<!-- generated: ask-the-binary -->
Before you fill in an input for this command, run `"${CLAUDE_PLUGIN_ROOT}/abcd" explain embargo`, and where its answer and this page disagree, follow the binary, since this page may be older than it.
<!-- /generated -->

When someone reports a security hole privately, `embargo` keeps it private
until its fix ships, then closes or publishes it with the person. Each
advisory becomes a private record in the **embargo store**, a worktree in the
machine's worktree store (`~/.abcd.noindex/worktrees/<root-sha>/embargo/`),
created detached and never committed. The record never enters this
repository's public ledger, and the link from an advisory to its record lives
in the store's state, not in the record. Every sub-verb checks the store
before it reads or writes it, and refuses, writing nothing, when the store is
on a branch, tracks a private record or any file in its local tier, sits inside the checkout, or is open to
other accounts on the machine: the store's folder is kept at `700` and each
private record at `600`, and a refusal names the `chmod` that restores them.
The store's local tier (`.abcd/.work.local/`) holds only files abcd writes: a
store whose index tracks anything there, under any spelling of its case, is
refused, and so is a state file under a link or under a folder spelled another
way, whose refusal names the folder to rename in place. Another project's store
under the same home is held to the same rules before the check reads it.

Every request to GitHub is the person's own `gh`, signed in to an account that
may read and edit the repository's security advisories; abcd holds no token.
A refusal for want of a scope names it and the person's own remedy
(`gh auth refresh -s repository_advisories:write`); one for want of a sign-in,
a `gh` signed in to no account or holding a token GitHub rejects,
names `gh auth login`.

Pick the sub-verb from the user's input. With none, run the bare listing
below and show the user what it reports.

## Bare: what is ready to publish

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo --json
```

Lists each advisory judged a vulnerability and neither closed nor published,
in id order, in one of three lists. `ready` holds each whose fix a published
release holds, every fix commit of it: `release` names the first such release
and `publish_with` the publish command that shows its plan. `waiting` holds
each whose fix is recorded and not yet in a published release, with the
`reason`. `no_fix` holds each with no fix recorded. The fix is the commits
`abcd embargo fixed` recorded, or, with none recorded, the fix commit its
private record, resolved in the store, names, as the publication plan takes
it. Which release tags hold a commit is read from the commits as git stores
them, so a `refs/replace/` entry or a graft does not make a release hold a fix
it lacks, and a tag counts only as a published release (see Publish below).
Origin's tags and GitHub's release are read only for a release tag that holds
a fix, so a fix in no tag asks nothing of the network; an answer that cannot be
had leaves that advisory waiting, with the reason, rather than refusing the
listing. A repository with no embargo store reports `no_store: true` and makes
none.

The listing writes nothing and publishes nothing, while the person is away as
while they are present. Ready means the person can publish now, never that
abcd will: show the user each ready advisory and its `publish_with`, and run
the publish only as Publish below says. While the person is away, queue the
publish steps for their return with Queue below.

## Queue

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo queue --json
```

While the person is away (`abcd mode autonomous`), queues one step for their
return for each advisory the bare listing names ready, naming its private
record, the release and the publish command that shows its plan, never the
advisory's id or title. A step already queued is not queued again: the check
and the append are one write under the away store's lock, so two runs at once
queue it once. `ready` lists the ready records, `queued` those this call
queued for, and `already_queued` those queued before; `abcd mode back` prints
the steps. It writes nothing in the embargo store and publishes nothing.

While the person is present, or when the away record cannot be read, it
refuses with exit 2 and queues nothing: tell the person instead. A step that
cannot be queued exits 1, naming each queued before it.

## Fixed

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo fixed --json <iss-N|GHSA-id> <commit>... [--replace]
```

Records the commits that fix one advisory judged a vulnerability and neither
closed nor published, in the store's state and nowhere else: the public ledger
and this checkout are never written, so the link from the fix to the advisory
stays private. Name each commit by its id, abbreviated or full; an id that is
not 7 to 64 lower-case hex digits (a ref name, a tag, anything shaped like an
option) is refused, and so is one git does not resolve to a single commit of
this repository. Each is kept by its full id. A commit already recorded is not
recorded twice, and `--replace` swaps the recorded list for the commits named.
`fix_commits` is the list now recorded and `added` each commit this call
added. Every refusal writes nothing.

Run it once the fix lands, with the commit or commits that make it: the bare
listing then names the advisory ready to publish as soon as a published
release holds every one, and the publication plan takes them as its fix (see
Publish below).

## Intake

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo intake --json
```

Reads the advisories GitHub holds in triage and draft and mints one private
record for each one not seen before. `minted` lists each new record with its
advisory and severity (high is major, medium and low are minor, critical is
critical); `held` lists the advisories already held, which are never recorded
twice; `needs_severity` lists the advisories GitHub gives no severity, which
get no record yet but are held under embargo all the same, so the check
refuses their id, reserved CVE id and title: tell the user to set the severity
on GitHub and run intake again. Nothing is written to the public ledger.

An advisory the store holds that GitHub no longer lists in triage or draft was
published or closed there, by abcd or by hand, and intake reads it on its own.
`ended` lists each now recorded, with the `published_at` or `closed_at` taken
from GitHub's own timestamp: a published one is recorded whatever its verdict,
and leaves the check; a closed one is recorded only when its verdict already
ends the private track: `not-abcds`, `not-a-vulnerability` or
`public-track`. `unread` lists each left as the store
holds it, under embargo, with the reason, and read again by the next intake or
plan: a failed read, a 404 or 403 on its read (GitHub reports no such
advisory to this account: it may be deleted, or this account may not see
it), a state this abcd does not know, a timestamp that is missing or is not
one, or a close GitHub reports while the store holds the
advisory as a vulnerability or untriaged. That last is the person's to
settle: tell the user, who reopens it on GitHub to publish it or records a
verdict that ends the private track with triage, which the next read then
records. A missing `gh`, a `gh` signed in to no account or holding a token
GitHub rejects, or a refusal whose words name the advisory scope, refuses the
intake whole, recording nothing, and names the remedy. A 404 or 403 alone
never does: the listing intake has just read proves the scope. `restores`
lists each restore owed, as for publish below.

## Triage

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo triage --json <iss-N|GHSA-id> <verdict> --by "<who decided>" [...]
```

The verdict is the person's decision, recorded under their name, so never record one they have not given. Do not hand the choice back bare either: ask the product thinker to confirm, under a `Product` chip, with the question text itself carrying, one advisory per tab, the advisory's title, in outcome terms throughout, with no record ids, no code and no verb's verdict words: its evidence (what someone could do with it and who is exposed), the verdict the record supports, and one line on why each other verdict does not fit. Where two verdicts are each defensible, ask between them neutrally instead. Run triage with `--by` naming them only on their yes.

| Verdict | Needs | What happens |
| --- | --- | --- |
| `vulnerability` | `--by` | the record stays open in the store, to be fixed |
| `not-abcds` | `--by`, `--exists-without "<why>"`, `--cannot-help "<why>"` | the record is won't-fixed in the store and the closing text is drafted |
| `not-a-vulnerability` | `--by`, `--reason "<why>"` | the record is won't-fixed in the store and the closing text is drafted |
| `public-track` | `--by`, `--reason "<why>"` | the record is won't-fixed in the store and the closing text is drafted; `capture_command` is the `abcd capture` command to run in the public checkout |

The two halves of `not-abcds` are the two halves of the rule that abcd fixes
what it adds: why the problem would exist without abcd, and why abcd cannot
help without changing how the repository works. The time is stamped from the
clock. A second triage keeps the earlier verdict in its history. An unknown
verdict, or one missing its reason, is refused with nothing written.

## Close

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo close --json <iss-N|GHSA-id>
```

For an advisory whose verdict is `not-abcds`, `not-a-vulnerability` or
`public-track`. The public-track text says the report moved to the public
track and gives the triage reason; it names no public issue, since the store
keeps none. A public-track verdict recorded before triage drafted a closing
text has its text drafted at close time from the reason the store keeps, with
the same checks and the same approval. GitHub
offers no comment on a repository advisory, so the explanation is posted at the
head of the advisory's description, with the reporter's own text kept whole
below a separator line. Without `--approve` it sends nothing but a read: show
the user `text` exactly as returned, then `approve_with`. The reporter reads
that text, so it is refused before it is shown when it carries a secret, a
name the private-names layer bans, another advisory still under embargo
(its id, reserved CVE id or title, a closed advisory's title included), the
embargo store's location on this machine (absolute, home-relative or its
worktree-store form), or any private record id the store holds, this
advisory's own included; the refusal names the kind and the line, never the match, and the remedy is to
record the verdict again with a reason that does not carry it.

**Approval, one per act.** Pass `--approve <digest>` only after the person, in
this session, has read that text and said yes to it. Never supply the digest on
your own, and never reuse an approval for another advisory or another text:
the digest binds the text and the advisory as read, so an edited draft or an
advisory changed on GitHub sends nothing, and the remedy is to show the text
again. With the approval abcd closes the advisory, reads it back to confirm it
is closed with that text, and records `closed_at`. A read-back that does not
confirm it exits 1 naming which half did not hold. An advisory GitHub already
reports closed (a close whose read-back or record failed, or one closed by
hand) is not sent anything: `closed_at` is recorded from GitHub's own
timestamp, `reconciled` is true, and `text_held` says whether its description
begins with the drafted text; one GitHub reports published is refused, and its
`published_at` recorded from GitHub's own timestamp. One GitHub reports closed
or published with no time for it, a time abcd cannot read, or one more than
five minutes later than this machine's clock, is refused with nothing sent or
recorded, never stamped with the clock's time.

## Check

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo check --json --range <a>..<b>
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo check --json --file <path> [--file -]
```

Run it before anything is made public: a push, a pull-request body, a
changelog section, a release page, a record. It reads each commit's message
and the lines it adds (a merge's, the lines it adds against every parent, so
what a merge of the default branch brings in is not read as the branch's
own), and each text, and
refuses every advisory not yet published by its id, a CVE id reserved for it,
its title (not once the advisory is closed, since its reporter chose it and
nothing will publish it, nor once its verdict is `public-track`, since its
issue goes into the public ledger; its id and CVE id stay refused), and the
store's location on this machine; any other advisory id is
refused too unless `--allow <GHSA-id>` names it as another project's, already
public, or GitHub lists it among this repository's own published advisories.
GitHub is asked for those once, through your `gh`, and only when the inputs
name an id no store holds; when it cannot be asked (no github.com origin, `gh`
signed out, GitHub unreachable) each such id stays refused, and `note` says
so. Each finding names its `kind`, `source`, `where` and `line` and never
repeats the match. Text is folded before it is matched: fullwidth and other
compatibility letters read as plain ones, every dash as a hyphen, invisible
format characters (Unicode category Cf, such as a zero-width space or a soft
hyphen) are dropped, and a token broken at a hyphen across a line end reads
whole. Exit 0 is clean (or no store exists on this machine), exit 2 is a
finding. On a finding, `--json` writes the refusal envelope after the result
as a second JSON document, which a host tells apart by its leading
`"abcd": "error"`. `intake_stale` says a store has never had an intake, or the oldest
intake among the stores is more than a day old: run intake first.

Limits the user must hear: the check cannot recognise a paraphrase or a count
("fixes several private reports"), which stays the author's to avoid; a
title shorter than 12 characters is not matched on its own, though its
advisory's id still is; and an advisory id of the placeholder form, each
four-character group one character repeated (`GHSA-xxxx-xxxx-xxxx`,
`GHSA-2222-cccc-xxxx`), is taken for a documentation or test placeholder and
not refused as unlisted, so a real id of that form would pass unremarked. An
advisory the store holds is still refused by its id whatever its form.

The fold catches less than a reader would. It does not catch look-alike
letters from other scripts (a Cyrillic or Greek letter standing in for a
Latin one); invisible characters outside category Cf (the combining grapheme
joiner U+034F, variation selectors, the Hangul filler U+3164); an id or title
written as HTML entities or with markdown backslash escapes; or a word whose
line wrap inserted a hyphen that was not in it (a title word split as
`over-` and `flow`). `--range` reads commit messages, the lines each commit
adds and the names of the files it adds, and nothing else: branch names and
annotated tag messages are not checked. Each commit is read as git stores it,
the form a push sends: a `refs/replace/` entry, a `.git/info/grafts` file or a
commit-graph does not change the message, the parents or the lines it reads.

The check reads every embargo store under this account's home, whichever
repository each belongs to, so a second clone, a shallow clone or a repository
with merged-in history is checked against the records held for it. A store
whose checkout was deleted is still read and its records still refused; the
`note` counts such stores, naming none. A store this repository registered
under another home is refused (run abcd with that home, or, if that home no
longer exists, run `git worktree prune`), and a store's place that cannot be
examined, or whose state file cannot be read, refuses the check. `no_store:
true` means no store exists under this home at all. Because every store is
read, `under_embargo` counts the ids held across all of them, so it can be
non-zero in an unrelated repository that holds nothing itself, and it stays
off public logs there too.

A consumer of the output must treat `no_store: true` as "not checked", never
as clean: nothing was compared against anything. Keep the output off public
logs, its `under_embargo` count included: how many advisories are held is
itself a disclosure.

## Publish

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" embargo publish --json [--release <vX.Y.Z>]
```

Once the release carrying the fixes is cut, this shows the publication plan
and sends nothing but reads. For each advisory judged a vulnerability (the
release defaults to the highest published release): its record resolved in
the release, its fix commit, the first published release holding the fix, the
patched version that gives, and the affected range in GitHub's form.
The record, its fix commit and the tags holding the fix are read from the
objects as git stores them, so a `refs/replace/` entry or a graft does not
change which release the plan names.

**What counts as a published release.** A `vMAJOR.MINOR.PATCH` tag counts only
when both hold: origin holds the tag at the same object as this checkout
(read once per plan with `git ls-remote --tags origin`), and GitHub reports a
release for it that is neither a draft nor a prerelease. A tag that only this
checkout holds, or one whose release run failed or is still drafting, gives no
patched version: the next published release holding the fix gives it, and a
fix in none up to the release is left out, saying so. The affected range then
reaches below that later version, so it also names the skipped tag, which
over-warns rather than under-warns. A `--release` naming a
tag that is not a published release is refused with what it lacks and the
remedy (push the tag with `git push origin <tag>`, or publish its release on
GitHub). When origin's tags cannot be listed, or GitHub cannot be read for a
release or answers for another tag, the plan refuses whole, recording
nothing; the checkout's own tags are never taken in their place. Origin's tags
vouch only for the repository the store was made for, so a plan, and so a
publish, whose origin names another repository (a fork, say) is refused, as
intake refuses it.

The security rule keeps a security record out of every committed tree until
its advisory is published, so the release usually carries no public record of
it. Then the fix itself is the evidence: the private record resolved in the
store's own ledger, naming its fix commit. Resolve it there once the fix has
landed, from the store's folder, naming the commit by its full id:

```bash
cd ~/.abcd.noindex/worktrees/<root-sha>/embargo
"${CLAUDE_PLUGIN_ROOT}/abcd" capture resolve <iss-N> "<what fixed it>" --impact fix --commit <full commit id>
```

The resolve keeps the record's mode, so it stays this account's alone. The
plan takes that commit only when it is a full object name of this repository's
hash that git resolves to exactly itself as a commit, so an abbreviation, a
tag's name or anything shaped like an option is left out with the reason; the
first published release holding the commit gives the patched version, as for a
public record. Such an entry carries `store_evidence: true`. A record resolved in the
release is always taken first, whatever the store holds.

Fix commits recorded with `abcd embargo fixed` come next: with no record
resolved in the release, the plan takes them before the store's resolved
record. Each is checked again to be a commit of this repository, since the
state is a file on disk, and the first published release holding every one
gives the patched version. Such an entry carries `store_evidence: true` and
lists them in `fix_commits`, `fix_commit` being the first.

`left_out` names each advisory the plan does not publish and why: its record
is resolved neither in the release nor in the store with a fix commit the plan
accepts, and no fix commit it accepts is recorded for it, its fix is in no published release up to the release, its package is not set on GitHub, or its
summary, description, or any affected package's name or vulnerable functions
carry a secret, a private name, or another advisory still under embargo
(a closed one's title included: it stays private on GitHub). Show the user the
whole plan, then `approve_with`.

**Approval, one per act.** Pass `--approve <digest>` only after the person, in
this session, has read that plan and said yes to it, and never on your own. A
digest for any other plan sends nothing. With it abcd publishes one advisory
first as a canary and reads it back, then the rest the same way, stopping at
the first mismatch, and confirms none is left in draft. A stopped run exits 1
naming what is published and what is not; running publish again plans only
what is still unpublished.

Every plan first reads each advisory the store holds that is neither published
nor closed, judged or not, as intake does, and records what GitHub reports on
the same terms: `ended` lists each it recorded, so a publication whose record
failed, or one made by hand on GitHub, is completed by the next run, and an
advisory found so is left out of the plan; `unread` lists each left as the
store holds it, with the reason; a 404 or 403 on one advisory's read leaves
that one unread and the plan goes on. A candidate GitHub cannot say is also
left out, with that reason. A missing `gh`, a `gh` signed in to no account or
holding a token GitHub rejects, or a refusal whose words name the advisory
scope, refuses the plan whole, recording nothing. When every advisory's read
meets a 404 or 403, a token without the scope would look the same, so the
plan asks GitHub once for the repository's advisory listing: if it answers,
the scope is proven, those advisories are unread, and the plan goes on (a
store whose one advisory was deleted on GitHub still plans); if GitHub
refuses it, the plan refuses whole with the scope remedy; if it fails
otherwise, the plan refuses whole and asks for a re-run. Neither list is
bound into the digest.

`restores` names each restore the store's state owes: every advisory recorded
published whose private record is resolved in the store, and passes the same
checks as the plan's evidence, while this checkout's `.abcd/work/issues/resolved/`
holds no record of it. Its `restore_with` is the command, run from the
checkout's root, that copies the record into the public ledger's `resolved/`
folder. It is read from the store's state, not from the run that published, so
every plan, every intake and every publication names it, a stopped run's
failure included, until the record is in the checkout. The reminder stands
until then even when the person decides never to commit the record: a record
kept out of the ledger on purpose is named on every run. A record that fails
those checks, or whose path holds a character the terminal would not show as
written, names no command and gives the `reason`.

abcd has no verb that restores a record; show the user the command, and the
person runs it. The record was minted at intake from the reporter's original
report, so before it lands, tell the person to compare its body with the
advisory as published on GitHub and remove anything only the reporter should
see. The person then lands the record in a change of its own.

## What stays the person's act

`embargo` does not fix, land or announce anything. These stay with the person:

- fixing each vulnerability, landing the fixes so they are in the release
  that publishes the advisories, and recording each fix with `abcd embargo
  fixed` or resolving each private record in the store with its fix commit;
- publishing an advisory the bare listing names ready to publish: the
  listing, and the step `abcd embargo queue` queues while the person is away,
  only say so;
- restoring each record into the public ledger once its advisory is
  published, with the command a plan or an intake names, after comparing its
  body with the published advisory;
- requesting a CVE for an advisory, on GitHub;
- putting back any record details they redacted to land a fix, once the
  advisory is published;
- every verdict, and every yes that lets abcd close or publish an advisory.

abcd closes or publishes an advisory only on the person's own approval of what
it showed them.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

<!-- generated: command-facts -->
**Command facts, format version 3.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain embargo` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd embargo [flags]`
  - An example call is `abcd embargo`.
  - What usually follows:
    - `abcd embargo intake`, to take new private advisories into the embargo store.
    - `abcd embargo publish`, to see the publication plan for those ready.
- `abcd embargo check [flags]`
  - An example call is `abcd embargo check --range v0.11.0..HEAD`.
  - What usually follows:
    - `abcd land`, when nothing in the range discloses an advisory under embargo.
- `abcd embargo close <iss-N|GHSA-id> [flags]`
  - An example call is `abcd embargo close GHSA-xxxx-xxxx-xxxx`.
  - What usually follows:
    - `abcd embargo`, to see the advisories still under embargo.
- `abcd embargo fixed <iss-N|GHSA-id> <commit>... [flags]`
  - An example call is `abcd embargo fixed GHSA-xxxx-xxxx-xxxx 0123abc`.
  - What usually follows:
    - `abcd embargo publish`, to see the publication plan the fix makes ready.
- `abcd embargo intake [flags]`
  - An example call is `abcd embargo intake`.
  - What usually follows:
    - `abcd embargo triage <GHSA-id> <verdict>`, to record a verdict on an advisory taken in.
- `abcd embargo publish [flags]`
  - An example call is `abcd embargo publish`.
  - What usually follows:
    - `abcd embargo close <GHSA-id>`, to close an advisory once it is published.
- `abcd embargo queue [flags]`
  - An example call is `abcd embargo queue`.
  - What usually follows:
    - `abcd mode back`, on the person's return, to print what was queued.
- `abcd embargo triage <iss-N|GHSA-id> <verdict> [flags]`
  - An example call is `abcd embargo triage GHSA-xxxx-xxxx-xxxx not-a-vulnerability --by "<who decided>" --reason "the behaviour is the documented one"`.
  - What usually follows:
    - `abcd embargo fixed <GHSA-id> <commit>`, once the commits that fix the advisory exist.
<!-- /generated -->
